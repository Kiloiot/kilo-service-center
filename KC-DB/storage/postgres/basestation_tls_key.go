package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// sqlLockBaseStationTLSKey reads a station's stored private key under a row
// lock that a concurrent take is refused rather than queued behind, since the
// holder may need another pooled connection before it can release the lock.
// NO KEY UPDATE, because that connection records an event referencing the row.
const sqlLockBaseStationTLSKey = `
	SELECT id, tls_key FROM basestations
	WHERE tenant_id = $1 AND bs_eui = $2 AND tls_key IS NOT NULL AND tls_key <> ''
	FOR NO KEY UPDATE NOWAIT`

const sqlClearBaseStationTLSKey = `UPDATE basestations SET tls_key = NULL, updated_at = NOW() WHERE id = $1`

// TakeTLSKey hands a tenant's base station's stored private key to open and
// removes it once open accepts it, all in one transaction: the key leaves the
// service center once, and a key open refuses (one that fails to decrypt, or
// does not match) stays stored. A station without a stored key, or of another
// tenant, is storage.ErrNotFound; a take while another holds the key fails at once.
func (r *BaseStationRepository) TakeTLSKey(ctx context.Context, tenantID int64, eui []byte, open func(sealed string) error) (err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapTakeTLSKey, err)
	}
	defer sqlcleanup.RollbackUncommitted(tx, errWrapRollbackTransaction, &err)

	var row struct {
		ID  int64  `db:"id"`
		Key string `db:"tls_key"`
	}
	err = tx.GetContext(ctx, &row, sqlLockBaseStationTLSKey, tenantID, eui)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s: %w", errWrapTLSKeyNotStored, storage.ErrNotFound)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapTakeTLSKey, err)
	}
	if err := open(row.Key); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, sqlClearBaseStationTLSKey, row.ID); err != nil {
		return fmt.Errorf("%s: %w", errWrapTakeTLSKey, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", errWrapTakeTLSKey, err)
	}
	return nil
}
