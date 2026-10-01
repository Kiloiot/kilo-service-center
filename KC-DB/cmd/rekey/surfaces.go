package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
)

// wireKeyLen is the MIOTY key length every binary key surface stores.
const wireKeyLen = 16

// Legacy raw-GCM ciphertext layout: nonce||ciphertext||tag of a 16-byte key.
const (
	legacyNonceLen = 12
	legacyGCMTag   = 16
)

// exportFilePerm restricts the encrypted endpoint_keys export to its owner.
const exportFilePerm = 0o600

// rekeyer walks every key surface.
type rekeyer struct {
	db      *sql.DB
	cipher  keycrypto.Cipher
	legacy  *legacyCipher
	hmacKey []byte
	apply   bool
}

// surfaceScan classifies every stored value of one surface into c and returns
// the conversions for the convertible ones.
type surfaceScan func(ctx context.Context, c *counts) ([]rowConversion, error)

// keySurface is one location of stored key material and its scanner.
type keySurface struct {
	name string
	scan surfaceScan
}

// surfaces lists every key column and every key field postgres declares, each
// with its scanner; a declared field without a scanner, or a scanner for an
// undeclared field, is an error.
func (r *rekeyer) surfaces() ([]keySurface, error) {
	columns := postgres.KeyMaterialColumns()
	fields, err := bindFieldScanners(postgres.KeyMaterialFields(), r.fieldScanners())
	if err != nil {
		return nil, err
	}
	list := make([]keySurface, 0, len(columns)+len(fields))
	for _, col := range columns {
		column := byteaColumn{col}
		list = append(list, keySurface{
			name: column.name(),
			scan: func(ctx context.Context, c *counts) ([]rowConversion, error) {
				return r.scanByteaColumn(ctx, column, c)
			},
		})
	}
	return append(list, fields...), nil
}

// fieldScanners maps each key field surface to the scanner that converts it.
func (r *rekeyer) fieldScanners() map[string]surfaceScan {
	return map[string]surfaceScan{
		postgres.KeyMaterialSurfaceTLSKey:             r.scanTLSKeys,
		postgres.KeyMaterialSurfacePendingMetadataKey: r.scanPendingMetadataKeys,
		postgres.KeyMaterialSurfacePendingRecordKey:   r.scanPendingRecordKeys,
	}
}

// bindFieldScanners pairs every declared key field with its scanner.
func bindFieldScanners(fields []postgres.KeyMaterialField, scanners map[string]surfaceScan) ([]keySurface, error) {
	if len(scanners) != len(fields) {
		return nil, fmt.Errorf(errFmtScannerCount, len(scanners), len(fields))
	}
	bound := make([]keySurface, 0, len(fields))
	for _, field := range fields {
		scan, ok := scanners[field.Surface]
		if !ok {
			return nil, fmt.Errorf(errFmtNoSurfaceScanner, field.Surface)
		}
		bound = append(bound, keySurface{name: field.Surface, scan: scan})
	}
	return bound, nil
}

func (r *rekeyer) run(ctx context.Context) (*report, error) {
	surfaces, err := r.surfaces()
	if err != nil {
		return nil, err
	}
	rep := newReport()
	for _, s := range surfaces {
		c := rep.surface(s.name)
		conversions, err := s.scan(ctx, c)
		if err != nil {
			return nil, fmt.Errorf(errFmtSurfaceConvert, s.name, err)
		}
		if r.apply {
			r.convertAll(ctx, s.name, c, conversions)
		}
	}
	return rep, nil
}

// storedEnvelope is the stored form of an envelope: bytes in BYTEA columns,
// text in TEXT columns and JSON fields.
type storedEnvelope interface{ []byte | string }

// envelopeCodec seals plaintext into the stored form S of a surface and opens
// that form again.
type envelopeCodec[S storedEnvelope] interface {
	seal(plaintext []byte) (S, error)
	open(stored S) ([]byte, error)
}

// binaryCodec stores the binary envelope used by BYTEA columns.
type binaryCodec struct{ cipher keycrypto.Cipher }

func (b binaryCodec) seal(plaintext []byte) ([]byte, error) { return b.cipher.Encrypt(plaintext) }

func (b binaryCodec) open(stored []byte) ([]byte, error) { return b.cipher.Decrypt(stored) }

// textCodec stores the text envelope used by TEXT columns and JSON fields.
type textCodec struct{ cipher keycrypto.Cipher }

func (t textCodec) seal(plaintext []byte) (string, error) { return t.cipher.EncryptString(plaintext) }

func (t textCodec) open(stored string) ([]byte, error) { return t.cipher.DecryptString(stored) }

// rowStatements rewrite one stored value as an envelope. swap binds the
// sealed value to $1 followed by swapArgs and must match the row only while
// it still holds the scanned value; readBack selects the stored envelope.
type rowStatements struct {
	swap         string
	swapArgs     []interface{}
	readBack     string
	readBackArgs []interface{}
}

// rowConversion is one stored value to rewrite; apply seals plaintext with
// the surface's codec and runs the statements.
type rowConversion struct {
	id        string
	plaintext []byte
	apply     func(ctx context.Context, db *sql.DB) error
}

// newRowConversion binds a row's statements to the codec of its surface.
func newRowConversion[S storedEnvelope](id string, plaintext []byte, codec envelopeCodec[S], statements rowStatements) rowConversion {
	return rowConversion{
		id:        id,
		plaintext: plaintext,
		apply: func(ctx context.Context, db *sql.DB) error {
			return convertRow(ctx, db, codec, plaintext, statements)
		},
	}
}

func (r *rekeyer) convertAll(ctx context.Context, surface string, c *counts, conversions []rowConversion) {
	for _, conversion := range conversions {
		if err := conversion.apply(ctx, r.db); err != nil {
			c.Failed++
			fmt.Printf(reportFmtConversionFailed,
				surface, conversion.id, keyedDigest(r.hmacKey, conversion.plaintext), err)
			continue
		}
		c.Converted++
	}
}

// convertRow rewrites one value in its own transaction and commits only
// after the stored value reads back and opens to plaintext.
func convertRow[S storedEnvelope](ctx context.Context, db *sql.DB, codec envelopeCodec[S], plaintext []byte, statements rowStatements) (err error) {
	sealed, err := codec.seal(plaintext)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer sqlcleanup.RollbackUncommitted(tx, errWrapRollback, &err)

	result, err := tx.ExecContext(ctx, statements.swap, append([]interface{}{sealed}, statements.swapArgs...)...)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf(errFmtRowsMatchedConcurrent, affected)
	}

	var stored S
	if err := tx.QueryRowContext(ctx, statements.readBack, statements.readBackArgs...).Scan(&stored); err != nil {
		return err
	}
	roundTrip, err := codec.open(stored)
	if err != nil {
		return fmt.Errorf(errFmtReadBackDecrypt, err)
	}
	if !bytes.Equal(roundTrip, plaintext) {
		return errors.New(errMsgReadBackMismatch)
	}
	return tx.Commit()
}
