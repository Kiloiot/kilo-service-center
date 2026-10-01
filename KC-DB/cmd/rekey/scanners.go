package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
)

// pemPrefix identifies cleartext PEM material on the tls_key surface.
const pemPrefix = "-----BEGIN"

// classifyKeyBytes classifies one binary key value and returns the recovered
// plaintext for convertible classes.
func (r *rekeyer) classifyKeyBytes(value []byte) (class string, plaintext []byte) {
	if keycrypto.IsEnvelope(value) {
		if _, err := r.cipher.Decrypt(value); err != nil {
			return classMalformed, nil
		}
		return classEnvelope, nil
	}
	if len(value) == wireKeyLen {
		return classPlaintext, value
	}
	// Legacy raw GCM: nonce||ciphertext of a 16-byte key (44 bytes), or its
	// base64 text form stored in the BYTEA column (60 bytes of base64).
	if r.legacy != nil {
		if decrypted, err := r.legacy.decryptRaw(value); err == nil && len(decrypted) == wireKeyLen {
			return classLegacy, decrypted
		}
		if decrypted, err := r.legacy.decryptBase64(string(value)); err == nil && len(decrypted) == wireKeyLen {
			return classLegacy, decrypted
		}
		return classMalformed, nil
	}
	// Without the legacy key the format cannot be authenticated; the shape
	// still identifies probable legacy ciphertext for reporting.
	if looksLikeLegacy(value) {
		return classLegacyLocked, nil
	}
	return classMalformed, nil
}

// looksLikeLegacy matches the two shapes the retired scheme produced for
// 16-byte keys: 44 raw bytes or 60 base64 characters.
func looksLikeLegacy(value []byte) bool {
	const rawLen = legacyNonceLen + wireKeyLen + legacyGCMTag
	if len(value) == rawLen {
		return true
	}
	decoded, err := base64.StdEncoding.DecodeString(string(value))
	return err == nil && len(decoded) == rawLen
}

// classifyTextEnvelope reports whether a text envelope opens under the
// master key.
func (r *rekeyer) classifyTextEnvelope(value string) string {
	if _, err := r.cipher.DecryptString(value); err != nil {
		return classMalformed
	}
	return classEnvelope
}

// byteaColumn builds the statements for one binary key column. Each row's
// primary key travels as text and binds to a parameter the server types from
// the key column, so every per-row statement can use the primary-key index.
type byteaColumn struct{ postgres.KeyMaterialColumn }

func (b byteaColumn) name() string { return b.Table + "." + b.Column }

func (b byteaColumn) selectQuery() string {
	// #nosec G201 -- identifiers come from the fixed key column list.
	return fmt.Sprintf(`SELECT %s::text, %s FROM %s WHERE length(%s) > 0`,
		b.PrimaryKey, b.Column, b.Table, b.Column)
}

func (b byteaColumn) swapQuery() string {
	// #nosec G201 -- identifiers come from the fixed key column list.
	return fmt.Sprintf(`UPDATE %s SET %s = $1 WHERE %s = $2 AND %s = $3`,
		b.Table, b.Column, b.PrimaryKey, b.Column)
}

func (b byteaColumn) readBackQuery() string {
	// #nosec G201 -- identifiers come from the fixed key column list.
	return fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`, b.Column, b.Table, b.PrimaryKey)
}

func (r *rekeyer) scanByteaColumn(ctx context.Context, column byteaColumn, c *counts) (conversions []rowConversion, err error) {
	rows, err := r.db.QueryContext(ctx, column.selectQuery())
	if err != nil {
		return nil, err
	}
	defer sqlcleanup.CloseRows(rows, errWrapCloseRows, &err)

	for rows.Next() {
		var id string
		var value []byte
		if err := rows.Scan(&id, &value); err != nil {
			return nil, err
		}
		class, plaintext := r.classifyKeyBytes(value)
		if !c.record(class) {
			continue
		}
		conversions = append(conversions, newRowConversion[[]byte](id, plaintext, binaryCodec{r.cipher}, rowStatements{
			swap:         column.swapQuery(),
			swapArgs:     []interface{}{id, value},
			readBack:     column.readBackQuery(),
			readBackArgs: []interface{}{id},
		}))
	}
	return conversions, rows.Err()
}

// classifyTLSKey classifies one basestations.tls_key value.
func (r *rekeyer) classifyTLSKey(value string) (class string, plaintext []byte) {
	switch {
	case keycrypto.IsTextEnvelope(value):
		return r.classifyTextEnvelope(value), nil
	case strings.HasPrefix(value, pemPrefix):
		return classPlaintext, []byte(value)
	case r.legacy != nil:
		decrypted, err := r.legacy.decryptBase64(value)
		if err == nil && strings.HasPrefix(string(decrypted), pemPrefix) {
			return classLegacy, decrypted
		}
		return classMalformed, nil
	}
	// Base64 of variable-length ciphertext cannot be authenticated without
	// the legacy key; a decodable value is reported as locked legacy.
	if _, err := base64.StdEncoding.DecodeString(value); err == nil {
		return classLegacyLocked, nil
	}
	return classMalformed, nil
}

func (r *rekeyer) scanTLSKeys(ctx context.Context, c *counts) (conversions []rowConversion, err error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id::text, tls_key FROM basestations WHERE tls_key <> ''`)
	if err != nil {
		return nil, err
	}
	defer sqlcleanup.CloseRows(rows, errWrapCloseRows, &err)

	for rows.Next() {
		var id, value string
		if err := rows.Scan(&id, &value); err != nil {
			return nil, err
		}
		class, plaintext := r.classifyTLSKey(value)
		if !c.record(class) {
			continue
		}
		conversions = append(conversions, newRowConversion[string](id, plaintext, textCodec{r.cipher}, rowStatements{
			swap:         `UPDATE basestations SET tls_key = $1 WHERE id = $2 AND tls_key = $3`,
			swapArgs:     []interface{}{id, value},
			readBack:     `SELECT tls_key FROM basestations WHERE id = $1`,
			readBackArgs: []interface{}{id},
		}))
	}
	return conversions, rows.Err()
}

// classifyPendingMetadataKey classifies the encryptedKey of a pending
// operation's metadata. storedUnencrypted is the record's isEncrypted=false
// marker, written when the key encryptor was unavailable.
func (r *rekeyer) classifyPendingMetadataKey(value string, storedUnencrypted bool) (class string, plaintext []byte) {
	if keycrypto.IsTextEnvelope(value) {
		return r.classifyTextEnvelope(value), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return classMalformed, nil
	}
	switch {
	case keycrypto.IsEnvelope(decoded):
		// Base64 of the binary envelope, the attach propagate form written
		// before the text form was shared, opens with the master key.
		decrypted, decryptErr := r.cipher.Decrypt(decoded)
		if decryptErr != nil {
			return classMalformed, nil
		}
		return classLegacy, decrypted
	case storedUnencrypted && len(decoded) == wireKeyLen:
		return classPlaintext, decoded
	case r.legacy != nil:
		decrypted, decryptErr := r.legacy.decryptRaw(decoded)
		if decryptErr == nil && len(decrypted) == wireKeyLen {
			return classLegacy, decrypted
		}
		return classMalformed, nil
	}
	return classLegacyLocked, nil
}

func (r *rekeyer) scanPendingMetadataKeys(ctx context.Context, c *counts) (conversions []rowConversion, err error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, metadata ->> 'encryptedKey', COALESCE(metadata -> 'isEncrypted' = 'false'::jsonb, false)
		FROM bssci_pending_operations
		WHERE metadata ? 'encryptedKey'`)
	if err != nil {
		return nil, err
	}
	defer sqlcleanup.CloseRows(rows, errWrapCloseRows, &err)

	for rows.Next() {
		var id, value string
		var storedUnencrypted bool
		if err := rows.Scan(&id, &value, &storedUnencrypted); err != nil {
			return nil, err
		}
		class, plaintext := r.classifyPendingMetadataKey(value, storedUnencrypted)
		if !c.record(class) {
			continue
		}
		conversions = append(conversions, newRowConversion[string](id, plaintext, textCodec{r.cipher}, rowStatements{
			swap: `
				UPDATE bssci_pending_operations
				SET metadata = jsonb_set(metadata - 'isEncrypted', '{encryptedKey}', to_jsonb($1::text))
				WHERE id = $2 AND metadata ->> 'encryptedKey' = $3`,
			swapArgs:     []interface{}{id, value},
			readBack:     `SELECT metadata ->> 'encryptedKey' FROM bssci_pending_operations WHERE id = $1`,
			readBackArgs: []interface{}{id},
		}))
	}
	return conversions, rows.Err()
}

// classifyPendingRecordKey classifies the cleartext Numeric[16] nwkSnKey an
// operation record carries when it was persisted before key sanitization. A
// record that also carries an encrypted metadata key holds two copies that
// cannot be reconciled automatically.
func classifyPendingRecordKey(raw string, hasMetadataKey bool) (class string, plaintext []byte) {
	if hasMetadataKey {
		return classMalformed, nil
	}
	var numeric []int
	if err := json.Unmarshal([]byte(raw), &numeric); err != nil || len(numeric) != wireKeyLen {
		return classMalformed, nil
	}
	key := make([]byte, wireKeyLen)
	for i, value := range numeric {
		if value < 0 || value > math.MaxUint8 {
			return classMalformed, nil
		}
		key[i] = byte(value)
	}
	return classPlaintext, key
}

func (r *rekeyer) scanPendingRecordKeys(ctx context.Context, c *counts) (conversions []rowConversion, err error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, operation_data -> 'nwkSnKey', COALESCE(metadata ? 'encryptedKey', false)
		FROM bssci_pending_operations
		WHERE operation_data ? 'nwkSnKey'`)
	if err != nil {
		return nil, err
	}
	defer sqlcleanup.CloseRows(rows, errWrapCloseRows, &err)

	for rows.Next() {
		var id, raw string
		var hasMetadataKey bool
		if err := rows.Scan(&id, &raw, &hasMetadataKey); err != nil {
			return nil, err
		}
		class, plaintext := classifyPendingRecordKey(raw, hasMetadataKey)
		if !c.record(class) {
			continue
		}
		conversions = append(conversions, newRowConversion[string](id, plaintext, textCodec{r.cipher}, rowStatements{
			swap: `
				UPDATE bssci_pending_operations
				SET operation_data = operation_data - 'nwkSnKey',
				    metadata = jsonb_set(COALESCE(metadata, '{}'::jsonb), '{encryptedKey}', to_jsonb($1::text))
				WHERE id = $2 AND operation_data -> 'nwkSnKey' = $3::jsonb
				  AND NOT COALESCE(metadata ? 'encryptedKey', false)`,
			swapArgs: []interface{}{id, raw},
			readBack: `
				SELECT metadata ->> 'encryptedKey' FROM bssci_pending_operations
				WHERE id = $1 AND NOT operation_data ? 'nwkSnKey'`,
			readBackArgs: []interface{}{id},
		}))
	}
	return conversions, rows.Err()
}
