package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// transactionalEndPointSessionRepository implements interfaces.EndPointSessionRepository within a transaction
type transactionalEndPointSessionRepository struct {
	tx           *sql.Tx
	db           *DB
	keyEncryptor *crypto.KeyEncryptor // Optional - nil if not configured
	logger       logger.Logger        // For migration logging
}

// Create creates a new endpoint session within the transaction
func (r *transactionalEndPointSessionRepository) Create(ctx context.Context, session *models.EndPointSession) error {
	query := `
		INSERT INTO endpoint_sessions (
			endpoint_id, tenant_id, session_id, session_key, attach_cnt,
			status, started_at, last_activity_at,
			sh_addr, last_packet_cnt, uplink_mode,
			dl_open, res_exp, dl_ack, repetition,
			primary_basestation_id, uplink_count, downlink_count, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id, created_at, updated_at`

	// Default metadata if nil to avoid invalid JSON syntax error
	if session.Metadata == nil {
		session.Metadata = json.RawMessage("{}")
	}

	err := r.tx.QueryRowContext(ctx, query,
		session.EndPointID,
		session.TenantID,
		session.SessionID,
		session.SessionKey,
		session.AttachCnt,
		session.Status,
		session.StartedAt,
		session.LastActivityAt,
		session.ShAddr,
		session.PacketCnt, // Note: model field is PacketCnt, DB column is last_packet_cnt
		session.UplinkMode,
		session.DlOpen,
		session.ResExp,
		session.DlAck,
		session.Repetition,
		session.PrimaryBaseStationID,
		session.UplinkCount,
		session.DownlinkCount,
		session.Metadata,
	).Scan(&session.ID, &session.CreatedAt, &session.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create endpoint session: %w", err)
	}

	return nil
}

// GetActive retrieves the active session for an endpoint within the transaction
func (r *transactionalEndPointSessionRepository) GetActive(ctx context.Context, endpointID string) (*models.EndPointSession, error) {
	query := `
		SELECT
			id, endpoint_id, tenant_id, session_id, session_key, attach_cnt,
			status, started_at, last_activity_at, ended_at,
			sh_addr, last_packet_cnt, uplink_mode,
			dl_open, res_exp, dl_ack, repetition,
			primary_basestation_id, uplink_count, downlink_count,
			metadata, created_at, updated_at
		FROM endpoint_sessions
		WHERE endpoint_id = $1 AND status = 'active'
		ORDER BY started_at DESC
		LIMIT 1`

	session := &models.EndPointSession{}
	err := r.tx.QueryRowContext(ctx, query, endpointID).Scan(
		&session.ID,
		&session.EndPointID,
		&session.TenantID,
		&session.SessionID,
		&session.SessionKey,
		&session.AttachCnt,
		&session.Status,
		&session.StartedAt,
		&session.LastActivityAt,
		&session.EndedAt,
		&session.ShAddr,
		&session.PacketCnt,
		&session.UplinkMode,
		&session.DlOpen,
		&session.ResExp,
		&session.DlAck,
		&session.Repetition,
		&session.PrimaryBaseStationID,
		&session.UplinkCount,
		&session.DownlinkCount,
		&session.Metadata,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint session: %w", err)
	}

	// Lazy migration: convert legacy base64 session keys to raw GCM format
	if session != nil && len(session.SessionKey) > 0 && r.keyEncryptor != nil {
		clearKey, wasLegacy, err := r.keyEncryptor.DecryptKeyWithMigration(session.SessionKey)
		if err != nil {
			if r.logger != nil {
				r.logger.WarnContext(ctx, "Failed to decrypt session key during migration",
					"error", err, "sessionID", session.SessionID, "tenantID", session.TenantID)
			}
		} else if wasLegacy {
			if r.logger != nil {
				r.logger.InfoContext(ctx, "Migrating session key from base64 to raw GCM",
					"sessionID", session.SessionID, "tenantID", session.TenantID)
			}
			newEncrypted, err := r.keyEncryptor.EncryptKeyRaw(clearKey)
			if err != nil {
				if r.logger != nil {
					r.logger.WarnContext(ctx, "Failed to re-encrypt session key", "error", err)
				}
			} else {
				if err := r.UpdateSessionKey(ctx, session.TenantID, session.SessionID, newEncrypted); err != nil {
					if r.logger != nil {
						r.logger.WarnContext(ctx, "Failed to persist migrated session key", "error", err)
					}
				} else {
					session.SessionKey = newEncrypted
				}
			}
		}
	}

	return session, nil
}

// GetByID retrieves a session by ID within the transaction
func (r *transactionalEndPointSessionRepository) GetByID(ctx context.Context, id string) (*models.EndPointSession, error) {
	query := `
		SELECT
			id, endpoint_id, tenant_id, session_id, session_key, attach_cnt,
			status, started_at, last_activity_at, ended_at,
			sh_addr, last_packet_cnt, uplink_mode,
			dl_open, res_exp, dl_ack, repetition,
			primary_basestation_id, uplink_count, downlink_count,
			metadata, created_at, updated_at
		FROM endpoint_sessions
		WHERE id = $1`

	session := &models.EndPointSession{}
	err := r.tx.QueryRowContext(ctx, query, id).Scan(
		&session.ID,
		&session.EndPointID,
		&session.TenantID,
		&session.SessionID,
		&session.SessionKey,
		&session.AttachCnt,
		&session.Status,
		&session.StartedAt,
		&session.LastActivityAt,
		&session.EndedAt,
		&session.ShAddr,
		&session.PacketCnt,
		&session.UplinkMode,
		&session.DlOpen,
		&session.ResExp,
		&session.DlAck,
		&session.Repetition,
		&session.PrimaryBaseStationID,
		&session.UplinkCount,
		&session.DownlinkCount,
		&session.Metadata,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Lazy migration: convert legacy base64 session keys to raw GCM format
	if session != nil && len(session.SessionKey) > 0 && r.keyEncryptor != nil {
		clearKey, wasLegacy, err := r.keyEncryptor.DecryptKeyWithMigration(session.SessionKey)
		if err != nil {
			if r.logger != nil {
				r.logger.WarnContext(ctx, "Failed to decrypt session key during migration",
					"error", err, "sessionID", session.SessionID, "tenantID", session.TenantID)
			}
		} else if wasLegacy {
			if r.logger != nil {
				r.logger.InfoContext(ctx, "Migrating session key from base64 to raw GCM",
					"sessionID", session.SessionID, "tenantID", session.TenantID)
			}
			newEncrypted, err := r.keyEncryptor.EncryptKeyRaw(clearKey)
			if err != nil {
				if r.logger != nil {
					r.logger.WarnContext(ctx, "Failed to re-encrypt session key", "error", err)
				}
			} else {
				if err := r.UpdateSessionKey(ctx, session.TenantID, session.SessionID, newEncrypted); err != nil {
					if r.logger != nil {
						r.logger.WarnContext(ctx, "Failed to persist migrated session key", "error", err)
					}
				} else {
					session.SessionKey = newEncrypted
				}
			}
		}
	}

	return session, nil
}

// Update updates an endpoint session within the transaction
func (r *transactionalEndPointSessionRepository) Update(ctx context.Context, session *models.EndPointSession) error {
	query := `
		UPDATE endpoint_sessions SET
			session_key = $2,
			attach_cnt = $3,
			sh_addr = $4,
			last_packet_cnt = $5,
			repetition = $6,
			status = $7,
			last_activity_at = $8,
			ended_at = $9,
			uplink_count = $10,
			downlink_count = $11,
			metadata = $12,
			uplink_mode = $13,
			dl_open = $14,
			res_exp = $15,
			dl_ack = $16,
			primary_basestation_id = $17,
			updated_at = NOW()
		WHERE id = $1 AND tenant_id = $18
		RETURNING updated_at`

	err := r.tx.QueryRowContext(ctx, query,
		session.ID,
		session.SessionKey,
		session.AttachCnt,
		session.ShAddr,
		session.PacketCnt, // Note: model field is PacketCnt, DB column is last_packet_cnt
		session.Repetition,
		session.Status,
		session.LastActivityAt,
		session.EndedAt,
		session.UplinkCount,
		session.DownlinkCount,
		session.Metadata,
		session.UplinkMode,
		session.DlOpen,
		session.ResExp,
		session.DlAck,
		session.PrimaryBaseStationID,
		session.TenantID, // WHERE clause for tenant isolation
	).Scan(&session.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update endpoint session: %w", err)
	}

	return nil
}

// UpdateActivity updates the last activity timestamp and counters within the transaction
func (r *transactionalEndPointSessionRepository) UpdateActivity(ctx context.Context, sessionID string, isUplink bool) error {
	var query string
	if isUplink {
		query = `
			UPDATE endpoint_sessions SET
				last_activity_at = NOW(),
				uplink_count = uplink_count + 1,
				updated_at = NOW()
			WHERE id = $1`
	} else {
		query = `
			UPDATE endpoint_sessions SET
				last_activity_at = NOW(),
				downlink_count = downlink_count + 1,
				updated_at = NOW()
			WHERE id = $1`
	}

	result, err := r.tx.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update activity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Terminate terminates a session within the transaction
func (r *transactionalEndPointSessionRepository) Terminate(ctx context.Context, sessionID string) error {
	query := `
		UPDATE endpoint_sessions SET
			ended_at = NOW(),
			status = 'terminated',
			updated_at = NOW()
		WHERE id = $1 AND status = 'active'`

	result, err := r.tx.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to terminate session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found or already terminated")
	}

	return nil
}

// ExpireOldSessions marks old sessions as expired within the transaction
func (r *transactionalEndPointSessionRepository) ExpireOldSessions(ctx context.Context, maxAge time.Duration) (int64, error) {
	query := `
		UPDATE endpoint_sessions SET
			status = 'expired',
			ended_at = NOW(),
			updated_at = NOW()
		WHERE status = 'active' 
		AND last_activity_at < NOW() - INTERVAL '%d seconds'`

	query = fmt.Sprintf(query, int(maxAge.Seconds()))

	result, err := r.tx.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to expire sessions: %w", err)
	}

	return result.RowsAffected()
}

// GetByEndPoint retrieves all sessions for an endpoint within the transaction
func (r *transactionalEndPointSessionRepository) GetByEndPoint(ctx context.Context, endpointID string, limit int, offset int) ([]*models.EndPointSession, error) {
	query := `
		SELECT 
			id, endpoint_id, session_key, started_at, ended_at,
			last_activity_at, uplink_count, downlink_count,
			status, metadata, created_at, updated_at
		FROM endpoint_sessions
		WHERE endpoint_id = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.tx.QueryContext(ctx, query, endpointID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.logger.Warn("rows close failed", "error", err, "operation", "GetActiveSessionsByEndpoint")
		}
	}()

	var endpointSessions []*models.EndPointSession
	for rows.Next() {
		session := &models.EndPointSession{}
		err := rows.Scan(
			&session.ID,
			&session.EndPointID,
			&session.SessionKey,
			&session.StartedAt,
			&session.EndedAt,
			&session.LastActivityAt,
			&session.UplinkCount,
			&session.DownlinkCount,
			&session.Status,
			&session.Metadata,
			&session.CreatedAt,
			&session.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		endpointSessions = append(endpointSessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return endpointSessions, nil
}

// GetStats retrieves session statistics for an endpoint within the transaction
func (r *transactionalEndPointSessionRepository) GetStats(ctx context.Context, endpointID string) (*models.EndPointSessionStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_sessions,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active_sessions,
			SUM(uplink_count) as total_uplinks,
			SUM(downlink_count) as total_downlinks,
			AVG(EXTRACT(EPOCH FROM (COALESCE(ended_at, NOW()) - started_at))) as avg_duration_seconds
		FROM endpoint_sessions
		WHERE endpoint_id = $1`

	stats := &models.EndPointSessionStats{
		EndPointID: endpointID,
	}

	var avgDurationSeconds sql.NullFloat64

	err := r.tx.QueryRowContext(ctx, query, endpointID).Scan(
		&stats.TotalSessions,
		&stats.ActiveSessions,
		&stats.TotalUplinks,
		&stats.TotalDownlinks,
		&avgDurationSeconds,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get session stats: %w", err)
	}

	if avgDurationSeconds.Valid {
		stats.AvgSessionDuration = time.Duration(avgDurationSeconds.Float64) * time.Second
	}

	return stats, nil
}

// UpdateSessionKey updates only the session_key field for a session within the transaction
// This is used for lazy migration from base64-encoded to raw GCM format
// Includes tenant scoping to prevent cross-tenant session_id collisions
func (r *transactionalEndPointSessionRepository) UpdateSessionKey(ctx context.Context, tenantID int64, sessionID string, encryptedKey []byte) error {
	query := `
		UPDATE endpoint_sessions
		SET session_key = $1,
			last_activity_at = NOW(),
			updated_at = NOW()
		WHERE session_id = $2 AND tenant_id = $3
	`

	result, err := r.tx.ExecContext(ctx, query, encryptedKey, sessionID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update session key: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("session not found or tenant mismatch: sessionID=%s, tenantID=%d", sessionID, tenantID)
	}

	return nil
}
