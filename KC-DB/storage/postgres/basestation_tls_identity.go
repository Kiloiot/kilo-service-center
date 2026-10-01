package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// sqlBackfillTLSFingerprint sets the fingerprint only while none is stored.
const sqlBackfillTLSFingerprint = `
	UPDATE basestations
	SET tls_cert_fingerprint = $1, updated_at = NOW()
	WHERE tenant_id = $2 AND id = $3
	  AND (tls_cert_fingerprint IS NULL OR tls_cert_fingerprint = '')`

// sqlBackfillTLSCertExpiry sets the certificate expiry only while none is stored.
const sqlBackfillTLSCertExpiry = `
	UPDATE basestations
	SET tls_cert_expires_at = $1, updated_at = NOW()
	WHERE tenant_id = $2 AND id = $3 AND tls_cert_expires_at IS NULL`

// UpdateTLSFingerprintIfBlank persists the certificate fingerprint only while
// the stored tls_cert_fingerprint is still NULL or empty (see interface
// contract). The conditional WHERE makes the backfill race-safe: a concurrent
// writer wins and this call reports false so the caller reloads and compares.
func (r *BaseStationRepository) UpdateTLSFingerprintIfBlank(ctx context.Context, tenantID, id int64, fingerprint string) (bool, error) {
	result, err := r.db.ExecContext(ctx, sqlBackfillTLSFingerprint, fingerprint, tenantID, id)
	return updatedRow(result, err, errWrapBackfillTLSFingerprint, errWrapBackfillTLSFingerprintRows)
}

// UpdateTLSCertExpiryIfBlank persists the certificate expiry only while the
// stored tls_cert_expires_at is still NULL, so an existing expiry is never
// overwritten; reports whether a row was updated.
func (r *BaseStationRepository) UpdateTLSCertExpiryIfBlank(ctx context.Context, tenantID, id int64, expiresAt time.Time) (bool, error) {
	result, err := r.db.ExecContext(ctx, sqlBackfillTLSCertExpiry, expiresAt.UTC(), tenantID, id)
	return updatedRow(result, err, errWrapBackfillTLSCertExpiry, errWrapBackfillTLSCertExpiryRows)
}

// updatedRow reports whether a conditional update changed a row.
func updatedRow(result sql.Result, err error, execFailed, rowsFailed string) (bool, error) {
	if err != nil {
		return false, fmt.Errorf("%s: %w", execFailed, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("%s: %w", rowsFailed, err)
	}
	return rows > 0, nil
}
