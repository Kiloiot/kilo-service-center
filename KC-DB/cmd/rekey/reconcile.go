package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
)

// endpoint_keys key_type vocabulary (migration 015) and the endpoints columns
// the reconcilable types map onto. join/session keys have no destination
// column and are exported rather than dropped.
const (
	keyTypeNetwork     = "network"
	keyTypeApplication = "application"
)

// epKeyRow is one endpoint_keys or endpoint_keys_archive row. The id is kept
// as text: endpoint_keys has BIGINT ids, while endpoint_keys_archive kept the
// UUID ids of the table it was copied from before migration 015.
type epKeyRow struct {
	ID         string          `json:"id"`
	EndpointID int64           `json:"endpointId"`
	TenantID   int64           `json:"tenantId"`
	KeyType    string          `json:"keyType"`
	KeyVersion int64           `json:"keyVersion"`
	KeyValue   []byte          `json:"keyValue"`
	IsActive   bool            `json:"isActive"`
	Table      string          `json:"table"`
	Extra      json.RawMessage `json:"-"`
}

// countEndpointKeys records the row counts without touching anything
// (dry-run and verify modes).
func (r *rekeyer) countEndpointKeys(ctx context.Context, rep *report) error {
	exists, err := r.tableExists(ctx, "endpoint_keys")
	if err != nil {
		return err
	}
	if !exists {
		rep.epKeysTablesGone = true
		return nil
	}
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM endpoint_keys`).Scan(&rep.epKeysRows); err != nil {
		return err
	}
	archiveExists, err := r.tableExists(ctx, "endpoint_keys_archive")
	if err != nil {
		return err
	}
	if archiveExists {
		if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM endpoint_keys_archive`).Scan(&rep.epKeysArchiveRows); err != nil {
			return err
		}
	}
	return nil
}

// reconcileEndpointKeys resolves every endpoint_keys row so migration 000145
// can require both tables empty:
//   - network/application rows whose endpoint's envelope column decrypts to
//     the same bytes are redundant and deleted;
//   - network/application rows whose endpoint has no key yet are written into
//     the endpoint (envelope-encrypted, read back and compared) and deleted;
//   - rows that conflict with a different stored key, reference a missing
//     endpoint, or carry a type with no destination (join/session) are
//     exported to the encrypted archive file and reported as conflicts for
//     operator resolution - they are never silently dropped;
//   - archive rows are exported to the encrypted file and deleted.
func (r *rekeyer) reconcileEndpointKeys(ctx context.Context, rep *report, exportPath string) error {
	exists, err := r.tableExists(ctx, "endpoint_keys")
	if err != nil {
		return err
	}
	if !exists {
		rep.epKeysTablesGone = true
		return nil
	}

	rows, err := r.loadEndpointKeyRows(ctx, "endpoint_keys")
	if err != nil {
		return err
	}
	rep.epKeysRows = len(rows)

	var archiveRows []epKeyRow
	archiveExists, err := r.tableExists(ctx, "endpoint_keys_archive")
	if err != nil {
		return err
	}
	if archiveExists {
		archiveRows, err = r.loadEndpointKeyRows(ctx, "endpoint_keys_archive")
		if err != nil {
			return err
		}
		rep.epKeysArchiveRows = len(archiveRows)
	}

	if len(rows) == 0 && len(archiveRows) == 0 {
		return nil
	}

	var toExport []epKeyRow
	for _, row := range rows {
		resolved, conflict, err := r.reconcileRow(ctx, row)
		if err != nil {
			return fmt.Errorf(errFmtEndpointKeysRow, row.ID, err)
		}
		if resolved {
			rep.epKeysResolved++
			rep.epKeysRows--
			continue
		}
		rep.epKeysConflicts = append(rep.epKeysConflicts, conflict)
		toExport = append(toExport, row)
	}
	toExport = append(toExport, archiveRows...)

	if len(toExport) > 0 {
		if exportPath == "" {
			return errors.New(errMsgUnresolvedArchiveRows)
		}
		if err := r.exportRows(exportPath, toExport); err != nil {
			return err
		}
		rep.epKeysExported = len(toExport)
	}

	// Archive rows are safe in the export; delete them so migration 000145's
	// empty-table guard can pass. Conflicted live rows stay for the operator.
	if len(archiveRows) > 0 {
		if _, err := r.db.ExecContext(ctx, `DELETE FROM endpoint_keys_archive`); err != nil {
			return fmt.Errorf(errFmtDeleteExportedArchive, err)
		}
		rep.epKeysArchiveRows = 0
	}
	return nil
}

func (r *rekeyer) loadEndpointKeyRows(ctx context.Context, table string) (result []epKeyRow, err error) {
	// #nosec G201 -- table is one of two fixed identifiers.
	query := fmt.Sprintf(`SELECT id::text, endpoint_id, tenant_id, key_type, key_version, key_value, is_active FROM %s`, table)
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer sqlcleanup.CloseRows(rows, errWrapCloseRows, &err)

	for rows.Next() {
		row := epKeyRow{Table: table}
		if err := rows.Scan(&row.ID, &row.EndpointID, &row.TenantID, &row.KeyType, &row.KeyVersion, &row.KeyValue, &row.IsActive); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// reconcileRow resolves one live endpoint_keys row. It returns resolved=true
// when the row was deleted (redundant or migrated), otherwise a conflict
// description.
func (r *rekeyer) reconcileRow(ctx context.Context, row epKeyRow) (resolved bool, conflict string, err error) {
	var column string
	switch row.KeyType {
	case keyTypeNetwork:
		column = "nwk_key"
	case keyTypeApplication:
		column = "app_key"
	default:
		return false, fmt.Sprintf(errFmtNoDestinationColumn, row.ID, row.KeyType), nil
	}
	if len(row.KeyValue) != wireKeyLen {
		return false, fmt.Sprintf(errFmtKeyValueWrongLength, row.ID, wireKeyLen), nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, "", err
	}
	defer sqlcleanup.RollbackUncommitted(tx, errWrapRollback, &err)

	// #nosec G201 -- column is one of two fixed identifiers.
	var stored []byte
	err = tx.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT %s FROM endpoints WHERE id = $1 AND tenant_id = $2 FOR UPDATE`, column),
		row.EndpointID, row.TenantID).Scan(&stored)
	if errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Sprintf(errFmtEndpointMissing,
			row.ID, row.EndpointID, row.TenantID), nil
	}
	if err != nil {
		return false, "", err
	}

	switch {
	case len(stored) == 0:
		// No destination key: migrate the row's bytes into the endpoint.
		envelope, encErr := r.cipher.Encrypt(row.KeyValue)
		if encErr != nil {
			return false, "", encErr
		}
		// #nosec G201 -- column is one of two fixed identifiers.
		if _, err = tx.ExecContext(ctx,
			fmt.Sprintf(`UPDATE endpoints SET %s = $1 WHERE id = $2 AND tenant_id = $3`, column),
			envelope, row.EndpointID, row.TenantID); err != nil {
			return false, "", err
		}
	case keycrypto.IsEnvelope(stored):
		decrypted, decErr := r.cipher.Decrypt(stored)
		if decErr != nil {
			return false, fmt.Sprintf(errFmtEnvelopeNoDecrypt,
				row.ID, row.EndpointID, column), nil
		}
		if !bytes.Equal(decrypted, row.KeyValue) {
			return false, fmt.Sprintf(errFmtDifferentKeyDigest,
				row.ID, row.EndpointID, column,
				keyedDigest(r.hmacKey, decrypted), keyedDigest(r.hmacKey, row.KeyValue)), nil
		}
	default:
		if !bytes.Equal(stored, row.KeyValue) {
			return false, fmt.Sprintf(errFmtDifferentPreEnvelope,
				row.ID, row.EndpointID, column), nil
		}
	}

	// Read back and compare before the source row is deleted.
	// #nosec G201 -- column is one of two fixed identifiers.
	var readBack []byte
	if err = tx.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT %s FROM endpoints WHERE id = $1 AND tenant_id = $2`, column),
		row.EndpointID, row.TenantID).Scan(&readBack); err != nil {
		return false, "", err
	}
	if keycrypto.IsEnvelope(readBack) {
		decrypted, decErr := r.cipher.Decrypt(readBack)
		if decErr != nil || !bytes.Equal(decrypted, row.KeyValue) {
			return false, "", fmt.Errorf(errFmtReadBackMismatchColumn, row.EndpointID, column)
		}
	} else if !bytes.Equal(readBack, row.KeyValue) {
		return false, "", fmt.Errorf(errFmtReadBackMismatchColumn, row.EndpointID, column)
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM endpoint_keys WHERE id = $1`, row.ID); err != nil {
		return false, "", err
	}
	if err = tx.Commit(); err != nil {
		return false, "", err
	}
	return true, "", nil
}

// exportRows writes the rows as a master-key envelope so the backup is
// unreadable without the deployment's key.
func (r *rekeyer) exportRows(path string, rows []epKeyRow) error {
	payload, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	envelope, err := r.cipher.Encrypt(payload)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, envelope, exportFilePerm); err != nil {
		return fmt.Errorf(errFmtWriteExport, err)
	}
	fmt.Printf(reportFmtEndpointKeysExport, path, len(rows))
	return nil
}

func (r *rekeyer) tableExists(ctx context.Context, table string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1)`,
		table).Scan(&exists)
	return exists, err
}
