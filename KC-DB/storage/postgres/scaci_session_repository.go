package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
)

// SCACISessionRepository implements the SCACISessionRepository interface for PostgreSQL
type SCACISessionRepository struct {
	db *sqlx.DB
}

// Ensure SCACISessionRepository implements the interface
var _ interfaces.SCACISessionRepository = (*SCACISessionRepository)(nil)

// NewSCACISessionRepository creates a new PostgreSQL SCACI session repository
func NewSCACISessionRepository(db *sqlx.DB) *SCACISessionRepository {
	return &SCACISessionRepository{db: db}
}

// CreateSession creates a new SCACI session
func (r *SCACISessionRepository) CreateSession(ctx context.Context, req *models.SCACISessionCreateRequest) (*models.SCACISession, error) {
	if req == nil {
		return nil, fmt.Errorf("create request cannot be nil")
	}

	// Serialize metadata to JSONB
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO scaci_sessions (
			tenant_id, ac_eui, sn_ac_uuid, sn_sc_uuid,
			last_op_id_ac, last_op_id_sc, status,
			certificate_fingerprint, client_cert_subject, remote_addr,
			tls_version, cipher_suite, negotiated_version,
			can_resume, metadata, organization_id,
			connected_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW(), NOW(), NOW()
		)
		RETURNING id, connected_at, created_at, updated_at`

	session := &models.SCACISession{
		TenantID:               req.TenantID,
		AcEUI:                  req.AcEUI,
		SnAcUUID:               req.SnAcUUID,
		SnScUUID:               req.SnScUUID,
		LastOpIDAc:             0,
		LastOpIDSc:             0,
		Status:                 "active",
		CertificateFingerprint: req.CertificateFingerprint,
		ClientCertSubject:      req.ClientCertSubject,
		RemoteAddr:             req.RemoteAddr,
		TLSVersion:             req.TLSVersion,
		CipherSuite:            req.CipherSuite,
		NegotiatedVersion:      req.NegotiatedVersion,
		CanResume:              req.CanResume,
		Metadata:               req.Metadata,
	}

	err = r.db.QueryRowContext(ctx, query,
		req.TenantID,
		req.AcEUI[:],
		req.SnAcUUID[:],
		req.SnScUUID[:],
		0, // last_op_id_ac
		0, // last_op_id_sc
		"active",
		req.CertificateFingerprint,
		req.ClientCertSubject,
		req.RemoteAddr,
		req.TLSVersion,
		req.CipherSuite,
		req.NegotiatedVersion,
		req.CanResume,
		metadataJSON,
		req.OrganizationID,
	).Scan(&session.ID, &session.ConnectedAt, &session.CreatedAt, &session.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create SCACI session: %w", err)
	}

	return session, nil
}

// GetSessionByID retrieves a session by ID
func (r *SCACISessionRepository) GetSessionByID(ctx context.Context, tenantID, sessionID int64) (*models.SCACISession, error) {
	query := `
		SELECT
			id, tenant_id, ac_eui, sn_ac_uuid, sn_sc_uuid,
			last_op_id_ac, last_op_id_sc, status,
			certificate_fingerprint, client_cert_subject, remote_addr,
			tls_version, cipher_suite, negotiated_version,
			connected_at, last_heartbeat, disconnected_at,
			can_resume, metadata, organization_id,
			created_at, updated_at
		FROM scaci_sessions
		WHERE id = $1 AND tenant_id = $2`

	return r.scanSession(r.db.QueryRowContext(ctx, query, sessionID, tenantID))
}

// GetActiveSessionByAcEUI retrieves the active session for an Application Center
func (r *SCACISessionRepository) GetActiveSessionByAcEUI(ctx context.Context, tenantID int64, acEui [8]byte) (*models.SCACISession, error) {
	query := `
		SELECT
			id, tenant_id, ac_eui, sn_ac_uuid, sn_sc_uuid,
			last_op_id_ac, last_op_id_sc, status,
			certificate_fingerprint, client_cert_subject, remote_addr,
			tls_version, cipher_suite, negotiated_version,
			connected_at, last_heartbeat, disconnected_at,
			can_resume, metadata, organization_id,
			created_at, updated_at
		FROM scaci_sessions
		WHERE tenant_id = $1 AND ac_eui = $2 AND status = 'active'
		ORDER BY connected_at DESC
		LIMIT 1`

	return r.scanSession(r.db.QueryRowContext(ctx, query, tenantID, acEui[:]))
}

// GetSessionByAcUUID retrieves a session by Application Center session UUID
func (r *SCACISessionRepository) GetSessionByAcUUID(ctx context.Context, tenantID int64, snAcUUID [16]byte) (*models.SCACISession, error) {
	query := `
		SELECT
			id, tenant_id, ac_eui, sn_ac_uuid, sn_sc_uuid,
			last_op_id_ac, last_op_id_sc, status,
			certificate_fingerprint, client_cert_subject, remote_addr,
			tls_version, cipher_suite, negotiated_version,
			connected_at, last_heartbeat, disconnected_at,
			can_resume, metadata, organization_id,
			created_at, updated_at
		FROM scaci_sessions
		WHERE tenant_id = $1 AND sn_ac_uuid = $2
		ORDER BY connected_at DESC
		LIMIT 1`

	return r.scanSession(r.db.QueryRowContext(ctx, query, tenantID, snAcUUID[:]))
}

// GetSessionByScUUID retrieves a session by Service Center session UUID
func (r *SCACISessionRepository) GetSessionByScUUID(ctx context.Context, tenantID int64, snScUUID [16]byte) (*models.SCACISession, error) {
	query := `
		SELECT
			id, tenant_id, ac_eui, sn_ac_uuid, sn_sc_uuid,
			last_op_id_ac, last_op_id_sc, status,
			certificate_fingerprint, client_cert_subject, remote_addr,
			tls_version, cipher_suite, negotiated_version,
			connected_at, last_heartbeat, disconnected_at,
			can_resume, metadata, organization_id,
			created_at, updated_at
		FROM scaci_sessions
		WHERE tenant_id = $1 AND sn_sc_uuid = $2
		ORDER BY connected_at DESC
		LIMIT 1`

	return r.scanSession(r.db.QueryRowContext(ctx, query, tenantID, snScUUID[:]))
}

// UpdateSession updates session fields
func (r *SCACISessionRepository) UpdateSession(ctx context.Context, tenantID, sessionID int64, req *models.SCACISessionUpdateRequest) error {
	if req == nil {
		return fmt.Errorf("update request cannot be nil")
	}

	// Build dynamic update query with deterministic field order
	type updatePair struct {
		field string
		value interface{}
	}
	updates := []updatePair{}

	// Add fields in consistent order to ensure deterministic SQL generation
	if req.Status != nil {
		updates = append(updates, updatePair{"status", *req.Status})
	}
	if req.LastOpIDAc != nil {
		updates = append(updates, updatePair{"last_op_id_ac", *req.LastOpIDAc})
	}
	if req.LastOpIDSc != nil {
		updates = append(updates, updatePair{"last_op_id_sc", *req.LastOpIDSc})
	}
	if req.LastHeartbeat != nil {
		updates = append(updates, updatePair{"last_heartbeat", *req.LastHeartbeat})
	}
	if req.DisconnectedAt != nil {
		updates = append(updates, updatePair{"disconnected_at", *req.DisconnectedAt})
	}
	if req.CanResume != nil {
		updates = append(updates, updatePair{"can_resume", *req.CanResume})
	}
	// TLS evidence fields for session resume per SCACI §1
	if req.TLSVersion != nil {
		updates = append(updates, updatePair{"tls_version", *req.TLSVersion})
	}
	if req.CipherSuite != nil {
		updates = append(updates, updatePair{"cipher_suite", *req.CipherSuite})
	}
	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		updates = append(updates, updatePair{"metadata", metadataJSON})
	}
	if req.OrganizationID != nil {
		updates = append(updates, updatePair{"organization_id", *req.OrganizationID})
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Always update updated_at as the last field
	updates = append(updates, updatePair{"updated_at", time.Now()})

	query := `UPDATE scaci_sessions SET `
	args := []interface{}{}
	argPos := 1

	for i, pair := range updates {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("%s = $%d", pair.field, argPos)
		args = append(args, pair.value)
		argPos++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND tenant_id = $%d", argPos, argPos+1)
	args = append(args, sessionID, tenantID)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("session not found: id=%d, tenant=%d", sessionID, tenantID)
	}

	return nil
}

// UpdateOperationIDs updates both AC and SC operation IDs atomically
func (r *SCACISessionRepository) UpdateOperationIDs(ctx context.Context, tenantID, sessionID int64, acOpId, scOpId int64) error {
	query := `
		UPDATE scaci_sessions
		SET last_op_id_ac = $1, last_op_id_sc = $2, updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4`

	result, err := r.db.ExecContext(ctx, query, acOpId, scOpId, sessionID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update operation IDs: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("session not found: id=%d, tenant=%d", sessionID, tenantID)
	}

	return nil
}

// UpdateHeartbeat updates the last heartbeat timestamp
func (r *SCACISessionRepository) UpdateHeartbeat(ctx context.Context, tenantID, sessionID int64) error {
	query := `
		UPDATE scaci_sessions
		SET last_heartbeat = NOW(), updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.ExecContext(ctx, query, sessionID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("session not found: id=%d, tenant=%d", sessionID, tenantID)
	}

	return nil
}

// DisconnectSession marks a session as disconnected
func (r *SCACISessionRepository) DisconnectSession(ctx context.Context, tenantID, sessionID int64) error {
	query := `
		UPDATE scaci_sessions
		SET status = 'disconnected', disconnected_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.ExecContext(ctx, query, sessionID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to disconnect session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("session not found: id=%d, tenant=%d", sessionID, tenantID)
	}

	return nil
}

// TerminateSession marks a session as terminated
func (r *SCACISessionRepository) TerminateSession(ctx context.Context, tenantID, sessionID int64) error {
	query := `
		UPDATE scaci_sessions
		SET status = 'terminated', can_resume = false, disconnected_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.ExecContext(ctx, query, sessionID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to terminate session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("session not found: id=%d, tenant=%d", sessionID, tenantID)
	}

	return nil
}

// TerminateAllSessions terminates all active sessions for an Application Center
func (r *SCACISessionRepository) TerminateAllSessions(ctx context.Context, tenantID int64, acEui [8]byte) error {
	query := `
		UPDATE scaci_sessions
		SET status = 'terminated', can_resume = false, disconnected_at = NOW(), updated_at = NOW()
		WHERE tenant_id = $1 AND ac_eui = $2 AND status IN ('active', 'resumed')`

	_, err := r.db.ExecContext(ctx, query, tenantID, acEui[:])
	if err != nil {
		return fmt.Errorf("failed to terminate all sessions: %w", err)
	}

	return nil
}

// ListSessions retrieves sessions based on filter criteria
func (r *SCACISessionRepository) ListSessions(ctx context.Context, filter *models.SCACISessionFilter) ([]*models.SCACISession, int64, error) {
	if filter == nil {
		filter = &models.SCACISessionFilter{}
	}

	if filter.TenantID == nil {
		return nil, 0, fmt.Errorf("tenant ID required")
	}

	// Build WHERE clause dynamically
	where := "WHERE 1=1"
	args := []interface{}{}
	argPos := 1

	if filter.TenantID != nil {
		where += fmt.Sprintf(" AND tenant_id = $%d", argPos)
		args = append(args, *filter.TenantID)
		argPos++
	}
	if filter.OrganizationID != nil {
		where += fmt.Sprintf(" AND organization_id = $%d", argPos)
		args = append(args, *filter.OrganizationID)
		argPos++
	}
	if filter.AcEUI != nil {
		where += fmt.Sprintf(" AND ac_eui = $%d", argPos)
		args = append(args, (*filter.AcEUI)[:])
		argPos++
	}
	if filter.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *filter.Status)
		argPos++
	}
	if filter.CanResume != nil {
		where += fmt.Sprintf(" AND can_resume = $%d", argPos)
		args = append(args, *filter.CanResume)
		argPos++
	}
	if filter.ConnectedFrom != nil {
		where += fmt.Sprintf(" AND connected_at >= $%d", argPos)
		args = append(args, *filter.ConnectedFrom)
		argPos++
	}
	if filter.ConnectedTo != nil {
		where += fmt.Sprintf(" AND connected_at <= $%d", argPos)
		args = append(args, *filter.ConnectedTo)
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM scaci_sessions " + where
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	// Query sessions
	query := `
		SELECT
			id, tenant_id, ac_eui, sn_ac_uuid, sn_sc_uuid,
			last_op_id_ac, last_op_id_sc, status,
			certificate_fingerprint, client_cert_subject, remote_addr,
			tls_version, cipher_suite, negotiated_version,
			connected_at, last_heartbeat, disconnected_at,
			can_resume, metadata, organization_id,
			created_at, updated_at
		FROM scaci_sessions ` + where + `
		ORDER BY connected_at DESC`

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// TODO: Repository lacks logger field - add for proper error tracking
			log.Printf("failed to close rows in SCACI session query: %v", err)
		}
	}()

	sessions := []*models.SCACISession{}
	for rows.Next() {
		session, err := r.scanSessionFromRows(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	return sessions, total, nil
}

// GetSessionStatistics retrieves aggregated session statistics
func (r *SCACISessionRepository) GetSessionStatistics(ctx context.Context, tenantID int64) (*models.SCACISessionStatistics, error) {
	query := `
		SELECT
			COUNT(*) as total_sessions,
			COUNT(*) FILTER (WHERE status = 'active') as active_sessions,
			COUNT(*) FILTER (WHERE status = 'resumed') as resumed_sessions,
			COUNT(*) FILTER (WHERE status = 'disconnected') as disconnected_sessions,
			COUNT(*) FILTER (WHERE status = 'terminated') as terminated_sessions,
			COUNT(*) FILTER (WHERE can_resume = true AND status IN ('active', 'disconnected')) as resumable_sessions,
			COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(disconnected_at, NOW()) - connected_at)) / 3600), 0) as avg_duration_hours,
			COALESCE(SUM(EXTRACT(EPOCH FROM (COALESCE(disconnected_at, NOW()) - connected_at)) / 3600), 0) as total_time_hours
		FROM scaci_sessions
		WHERE tenant_id = $1`

	stats := &models.SCACISessionStatistics{TenantID: tenantID}
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&stats.TotalSessions,
		&stats.ActiveSessions,
		&stats.ResumedSessions,
		&stats.DisconnectedSessions,
		&stats.TerminatedSessions,
		&stats.ResumableSessions,
		&stats.AverageSessionDuration,
		&stats.TotalSessionTime,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get session statistics: %w", err)
	}

	return stats, nil
}

// CheckSessionResumable determines if a session can be resumed per SCACI §1 and §§2.1-2.3
// Validates both AC and SC operation ID progression per SCACI-S.1-05/3.2-03
// Returns NegotiatedVersion for version mismatch validation at service layer
func (r *SCACISessionRepository) CheckSessionResumable(ctx context.Context, tenantID int64, snAcUUID [16]byte, acOpId int64, scOpId int64) (*models.SCACISessionResumptionInfo, error) {
	query := `
		SELECT
			id, last_op_id_ac, last_op_id_sc, can_resume, status, negotiated_version,
			EXTRACT(EPOCH FROM (NOW() - connected_at)) / 3600 as session_age_hours
		FROM scaci_sessions
		WHERE tenant_id = $1 AND sn_ac_uuid = $2
		ORDER BY connected_at DESC
		LIMIT 1`

	var sessionID, lastOpIDAc, lastOpIDSc int64
	var canResume bool
	var status string
	var negotiatedVersion string
	var sessionAgeHours float64

	err := r.db.QueryRowContext(ctx, query, tenantID, snAcUUID[:]).Scan(
		&sessionID, &lastOpIDAc, &lastOpIDSc, &canResume, &status, &negotiatedVersion, &sessionAgeHours,
	)

	if err == sql.ErrNoRows {
		return &models.SCACISessionResumptionInfo{
			CanResume:            false,
			ReasonIfNotResumable: "session not found",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check session resumability: %w", err)
	}

	info := &models.SCACISessionResumptionInfo{
		SessionID:         sessionID,
		LastKnownAcOpId:   lastOpIDAc,
		LastKnownScOpId:   lastOpIDSc,
		NegotiatedVersion: negotiatedVersion,
		SessionAgeHours:   sessionAgeHours,
	}

	// Check resumability conditions per SCACI §1
	if !canResume {
		info.CanResume = false
		info.ReasonIfNotResumable = "session marked as not resumable"
		return info, nil
	}

	if status == "terminated" {
		info.CanResume = false
		info.ReasonIfNotResumable = "session already terminated"
		return info, nil
	}

	// Validate AC operation ID progression (AC opId should be >= last known)
	if acOpId < lastOpIDAc {
		info.CanResume = false
		info.ReasonIfNotResumable = fmt.Sprintf("AC opId regression: provided=%d, expected>=%d", acOpId, lastOpIDAc)
		return info, nil
	}

	// Validate SC operation ID progression per SCACI-S.1-05/3.2-03
	// SC opIds are negative and decrement, so provided scOpId should be <= last known
	if scOpId > lastOpIDSc {
		info.CanResume = false
		info.ReasonIfNotResumable = fmt.Sprintf("SC opId regression: provided=%d, expected<=%d", scOpId, lastOpIDSc)
		return info, nil
	}

	info.CanResume = true
	return info, nil
}

// CleanupExpiredSessions removes old terminated sessions
func (r *SCACISessionRepository) CleanupExpiredSessions(ctx context.Context, olderThan int64) (int64, error) {
	query := `
		DELETE FROM scaci_sessions
		WHERE status = 'terminated'
		  AND disconnected_at < NOW() - INTERVAL '1 hour' * $1`

	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return deleted, nil
}

// scanSession is a helper to scan a single session from a query row
func (r *SCACISessionRepository) scanSession(row *sql.Row) (*models.SCACISession, error) {
	var session models.SCACISession
	var acEuiBytes, snAcUUIDBytes, snScUUIDBytes []byte
	var metadataJSON []byte

	err := row.Scan(
		&session.ID,
		&session.TenantID,
		&acEuiBytes,
		&snAcUUIDBytes,
		&snScUUIDBytes,
		&session.LastOpIDAc,
		&session.LastOpIDSc,
		&session.Status,
		&session.CertificateFingerprint,
		&session.ClientCertSubject,
		&session.RemoteAddr,
		&session.TLSVersion,
		&session.CipherSuite,
		&session.NegotiatedVersion,
		&session.ConnectedAt,
		&session.LastHeartbeat,
		&session.DisconnectedAt,
		&session.CanResume,
		&metadataJSON,
		&session.OrganizationID,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan session: %w", err)
	}

	// Convert byte arrays to fixed-size arrays
	copy(session.AcEUI[:], acEuiBytes)
	copy(session.SnAcUUID[:], snAcUUIDBytes)
	copy(session.SnScUUID[:], snScUUIDBytes)

	// Unmarshal metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &session.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &session, nil
}

// scanSessionFromRows is a helper to scan a session from sql.Rows
func (r *SCACISessionRepository) scanSessionFromRows(rows *sql.Rows) (*models.SCACISession, error) {
	var session models.SCACISession
	var acEuiBytes, snAcUUIDBytes, snScUUIDBytes []byte
	var metadataJSON []byte

	err := rows.Scan(
		&session.ID,
		&session.TenantID,
		&acEuiBytes,
		&snAcUUIDBytes,
		&snScUUIDBytes,
		&session.LastOpIDAc,
		&session.LastOpIDSc,
		&session.Status,
		&session.CertificateFingerprint,
		&session.ClientCertSubject,
		&session.RemoteAddr,
		&session.TLSVersion,
		&session.CipherSuite,
		&session.NegotiatedVersion,
		&session.ConnectedAt,
		&session.LastHeartbeat,
		&session.DisconnectedAt,
		&session.CanResume,
		&metadataJSON,
		&session.OrganizationID,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan session: %w", err)
	}

	// Convert byte arrays to fixed-size arrays
	copy(session.AcEUI[:], acEuiBytes)
	copy(session.SnAcUUID[:], snAcUUIDBytes)
	copy(session.SnScUUID[:], snScUUIDBytes)

	// Unmarshal metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &session.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &session, nil
}
