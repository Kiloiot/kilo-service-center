package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/jmoiron/sqlx"
)

// BaseStationSessionRepository implements the BaseStationSessionRepository interface for PostgreSQL
type BaseStationSessionRepository struct {
	db *sqlx.DB
}

// Ensure BaseStationSessionRepository implements the interface
var _ interfaces.BaseStationSessionRepository = (*BaseStationSessionRepository)(nil)

// NewBaseStationSessionRepository creates a new PostgreSQL Base Station session repository
func NewBaseStationSessionRepository(db *sqlx.DB) *BaseStationSessionRepository {
	return &BaseStationSessionRepository{db: db}
}

// CreateSession creates a new Base Station session per MIOTY BSSCI 3.3
func (r *BaseStationSessionRepository) CreateSession(ctx context.Context, req *models.BaseStationSessionCreateRequest) (*models.BaseStationSession, error) {
	if req == nil {
		return nil, fmt.Errorf("create request cannot be nil")
	}

	// Default encoding to msgpack if not specified (BSSCI Section 1)
	encoding := req.Encoding
	if encoding == "" {
		encoding = bssci.EncodingMessagePack
	}

	query := `
		INSERT INTO basestation_sessions (
			basestation_id, tenant_id, sn_bs_uuid, sn_sc_uuid,
			sn_bs_op_id, sn_sc_op_id, status,
			connection_id, remote_addr, can_resume,
			organization_id, encoding, protocol_version, connect_info, started_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, COALESCE($14, '{}'::jsonb), NOW(), NOW(), NOW()
		)
		RETURNING id, started_at, created_at, updated_at`

	session := &models.BaseStationSession{
		BaseStationID:   req.BaseStationID,
		TenantID:        req.TenantID,
		SnBsUuid:        req.SnBsUuid,
		SnScUuid:        req.SnScUuid,
		SnBsOpId:        0,
		SnScOpId:        0,
		Status:          models.SessionStatusActive,
		ConnectionId:    req.ConnectionId,
		RemoteAddr:      req.RemoteAddr,
		CanResume:       req.CanResume,
		Encoding:        encoding,
		ProtocolVersion: req.ProtocolVersion,
		ConnectInfo:     req.ConnectInfo,
	}

	err := r.db.QueryRowContext(ctx, query,
		req.BaseStationID,
		req.TenantID,
		req.SnBsUuid[:],
		req.SnScUuid[:],
		0,
		0,
		models.SessionStatusActive,
		req.ConnectionId,
		req.RemoteAddr,
		req.CanResume,
		req.OrganizationID,
		encoding,
		req.ProtocolVersion,
		req.ConnectInfo,
	).Scan(&session.ID, &session.StartedAt, &session.CreatedAt, &session.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create Base Station session: %w", err)
	}

	return session, nil
}

// GetSessionByID retrieves a session by ID
func (r *BaseStationSessionRepository) GetSessionByID(ctx context.Context, tenantID, sessionID int64) (*models.BaseStationSession, error) {
	query := `
		SELECT
			id, basestation_id, tenant_id,
			sn_bs_uuid, sn_sc_uuid,
			sn_bs_op_id, sn_sc_op_id,
			status, connection_id, remote_addr,
			started_at, last_ping_at, ended_at,
			can_resume, organization_id,
			encoding, protocol_version, connect_info,
			created_at, updated_at
		FROM basestation_sessions
		WHERE id = $1 AND tenant_id = $2`

	return r.scanSession(r.db.QueryRowContext(ctx, query, sessionID, tenantID))
}

// GetActiveSessionByBaseStation retrieves the active session for a Base Station
func (r *BaseStationSessionRepository) GetActiveSessionByBaseStation(ctx context.Context, tenantID, baseStationID int64) (*models.BaseStationSession, error) {
	query := `
		SELECT
			id, basestation_id, tenant_id,
			sn_bs_uuid, sn_sc_uuid,
			sn_bs_op_id, sn_sc_op_id,
			status, connection_id, remote_addr,
			started_at, last_ping_at, ended_at,
			can_resume, organization_id,
			encoding, protocol_version, connect_info,
			created_at, updated_at
		FROM basestation_sessions
		WHERE basestation_id = $1 AND tenant_id = $2 AND status = 'active'
		ORDER BY started_at DESC
		LIMIT 1`

	return r.scanSession(r.db.QueryRowContext(ctx, query, baseStationID, tenantID))
}

// GetSessionByBsUUID retrieves a session by Base Station session UUID (for resumption)
func (r *BaseStationSessionRepository) GetSessionByBsUUID(ctx context.Context, tenantID int64, snBsUUID [16]byte) (*models.BaseStationSession, error) {
	query := `
		SELECT
			id, basestation_id, tenant_id,
			sn_bs_uuid, sn_sc_uuid,
			sn_bs_op_id, sn_sc_op_id,
			status, connection_id, remote_addr,
			started_at, last_ping_at, ended_at,
			can_resume, organization_id,
			encoding, protocol_version, connect_info,
			created_at, updated_at
		FROM basestation_sessions
		WHERE sn_bs_uuid = $1 AND tenant_id = $2
		ORDER BY started_at DESC
		LIMIT 1`

	return r.scanSession(r.db.QueryRowContext(ctx, query, snBsUUID[:], tenantID))
}

// GetSessionByScUUID retrieves a session by Service Center session UUID
func (r *BaseStationSessionRepository) GetSessionByScUUID(ctx context.Context, tenantID int64, snScUUID [16]byte) (*models.BaseStationSession, error) {
	query := `
		SELECT
			id, basestation_id, tenant_id,
			sn_bs_uuid, sn_sc_uuid,
			sn_bs_op_id, sn_sc_op_id,
			status, connection_id, remote_addr,
			started_at, last_ping_at, ended_at,
			can_resume, organization_id,
			encoding, protocol_version, connect_info,
			created_at, updated_at
		FROM basestation_sessions
		WHERE sn_sc_uuid = $1 AND tenant_id = $2
		ORDER BY started_at DESC
		LIMIT 1`

	return r.scanSession(r.db.QueryRowContext(ctx, query, snScUUID[:], tenantID))
}

// UpdateSession updates session fields (operation IDs, status, timing)
func (r *BaseStationSessionRepository) UpdateSession(ctx context.Context, tenantID, sessionID int64, req *models.BaseStationSessionUpdateRequest) error {
	if req == nil {
		return fmt.Errorf("update request cannot be nil")
	}

	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.SnBsOpId != nil {
		updates = append(updates, fmt.Sprintf("sn_bs_op_id = $%d", argPos))
		args = append(args, *req.SnBsOpId)
		argPos++
	}

	if req.SnScOpId != nil {
		updates = append(updates, fmt.Sprintf("sn_sc_op_id = $%d", argPos))
		args = append(args, *req.SnScOpId)
		argPos++
	}

	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *req.Status)
		argPos++
	}

	if req.LastPingAt != nil {
		updates = append(updates, fmt.Sprintf("last_ping_at = $%d", argPos))
		args = append(args, *req.LastPingAt)
		argPos++
	}

	if req.EndedAt != nil && req.ClearEndedAt {
		return fmt.Errorf("update request cannot set and clear ended_at at once")
	}

	if req.EndedAt != nil {
		updates = append(updates, fmt.Sprintf("ended_at = $%d", argPos))
		args = append(args, *req.EndedAt)
		argPos++
	}

	if req.ClearEndedAt {
		updates = append(updates, "ended_at = NULL")
	}

	if req.CanResume != nil {
		updates = append(updates, fmt.Sprintf("can_resume = $%d", argPos))
		args = append(args, *req.CanResume)
		argPos++
	}

	if req.ConnectionId != nil {
		updates = append(updates, fmt.Sprintf("connection_id = $%d", argPos))
		args = append(args, *req.ConnectionId)
		argPos++
	}

	if req.RemoteAddr != nil {
		updates = append(updates, fmt.Sprintf("remote_addr = $%d", argPos))
		args = append(args, *req.RemoteAddr)
		argPos++
	}

	if req.OrganizationID != nil {
		updates = append(updates, fmt.Sprintf("organization_id = $%d", argPos))
		args = append(args, *req.OrganizationID)
		argPos++
	}

	if req.Encoding != nil {
		updates = append(updates, fmt.Sprintf("encoding = $%d", argPos))
		args = append(args, *req.Encoding)
		argPos++
	}

	if req.ProtocolVersion != nil {
		updates = append(updates, fmt.Sprintf("protocol_version = $%d", argPos))
		args = append(args, *req.ProtocolVersion)
		argPos++
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argPos))
	args = append(args, time.Now())
	argPos++

	args = append(args, sessionID, tenantID)

	query := fmt.Sprintf(`
		UPDATE basestation_sessions
		SET %s
		WHERE id = $%d AND tenant_id = $%d`,
		strings.Join(updates, ", "),
		argPos, argPos+1)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update Base Station session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found: id=%d tenant=%d", sessionID, tenantID)
	}

	return nil
}

// UpdateOperationIDs updates both Base Station and Service Center operation IDs atomically
func (r *BaseStationSessionRepository) UpdateOperationIDs(ctx context.Context, tenantID, sessionID int64, bsOpId, scOpId int64) error {
	query := `
		UPDATE basestation_sessions
		SET sn_bs_op_id = $1,
		    sn_sc_op_id = $2,
		    updated_at = $3
		WHERE id = $4 AND tenant_id = $5`

	result, err := r.db.ExecContext(ctx, query, bsOpId, scOpId, time.Now(), sessionID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update operation IDs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found: id=%d tenant=%d", sessionID, tenantID)
	}

	return nil
}

// UpdatePing updates the last ping timestamp
func (r *BaseStationSessionRepository) UpdatePing(ctx context.Context, tenantID, sessionID int64) error {
	query := `
		UPDATE basestation_sessions
		SET last_ping_at = $1,
		    updated_at = $1
		WHERE id = $2 AND tenant_id = $3`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, sessionID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update ping: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found: id=%d tenant=%d", sessionID, tenantID)
	}

	return nil
}

// UpdateEncoding updates the message encoding for a session
// This is called when encoding is negotiated on first message per BSSCI Section 1
func (r *BaseStationSessionRepository) UpdateEncoding(ctx context.Context, tenantID, sessionID int64, encoding string) error {
	// Validate encoding value
	if encoding != bssci.EncodingJSON && encoding != bssci.EncodingMessagePack {
		return fmt.Errorf("invalid encoding: must be '%s' or '%s', got '%s'", bssci.EncodingJSON, bssci.EncodingMessagePack, encoding)
	}

	query := `
		UPDATE basestation_sessions
		SET encoding = $1,
		    updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3`

	result, err := r.db.ExecContext(ctx, query, encoding, sessionID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update encoding: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found: id=%d tenant=%d", sessionID, tenantID)
	}

	return nil
}

// TerminateSession marks a session as terminated
func (r *BaseStationSessionRepository) TerminateSession(ctx context.Context, tenantID, sessionID int64) error {
	query := `
		UPDATE basestation_sessions
		SET status = $1,
		    can_resume = false,
		    ended_at = $2,
		    updated_at = $2
		WHERE id = $3 AND tenant_id = $4`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, models.SessionStatusTerminated, now, sessionID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to terminate session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found: id=%d tenant=%d", sessionID, tenantID)
	}

	return nil
}

// TerminateAllSessions terminates all active sessions for a Base Station (for cleanup)
func (r *BaseStationSessionRepository) TerminateAllSessions(ctx context.Context, tenantID, baseStationID int64) error {
	query := `
		UPDATE basestation_sessions
		SET status = $1,
		    ended_at = $2,
		    updated_at = $2
		WHERE basestation_id = $3
		  AND tenant_id = $4
		  AND status = 'active'`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, models.SessionStatusTerminated, now, baseStationID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to terminate all sessions: %w", err)
	}

	return nil
}

// ListSessions retrieves sessions based on filter criteria
func (r *BaseStationSessionRepository) ListSessions(ctx context.Context, filter *models.BaseStationSessionFilter) ([]*models.BaseStationSession, int64, error) {
	if filter == nil {
		return nil, 0, fmt.Errorf("filter cannot be nil")
	}

	whereClauses := []string{"tenant_id = $1"}
	args := []interface{}{filter.TenantID}
	argPos := 2

	if filter.BaseStationID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("basestation_id = $%d", argPos))
		args = append(args, *filter.BaseStationID)
		argPos++
	}

	if len(filter.Status) > 0 {
		placeholders := []string{}
		for _, status := range filter.Status {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argPos))
			args = append(args, status)
			argPos++
		}
		whereClauses = append(whereClauses, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ",")))
	}

	if filter.SnBsUuid != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("sn_bs_uuid = $%d", argPos))
		args = append(args, (*filter.SnBsUuid)[:])
		argPos++
	}

	if filter.SnScUuid != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("sn_sc_uuid = $%d", argPos))
		args = append(args, (*filter.SnScUuid)[:])
		argPos++
	}

	if filter.CanResume != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("can_resume = $%d", argPos))
		args = append(args, *filter.CanResume)
		argPos++
	}

	if filter.ActiveOnly {
		whereClauses = append(whereClauses, "status = 'active'")
	}

	if filter.Since != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("started_at >= $%d", argPos))
		args = append(args, *filter.Since)
		argPos++
	}

	whereClause := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM basestation_sessions WHERE %s", whereClause)
	var totalCount int64
	err := r.db.GetContext(ctx, &totalCount, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT
			id, basestation_id, tenant_id,
			sn_bs_uuid, sn_sc_uuid,
			sn_bs_op_id, sn_sc_op_id,
			status, connection_id, remote_addr,
			started_at, last_ping_at, ended_at,
			can_resume, organization_id,
			encoding, protocol_version, connect_info,
			created_at, updated_at
		FROM basestation_sessions
		WHERE %s
		ORDER BY started_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argPos, argPos+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// TODO: Repository lacks logger field - add for proper error tracking
			log.Printf("failed to close rows in basestation session listing: %v", err)
		}
	}()

	sessions := []*models.BaseStationSession{}
	for rows.Next() {
		session, err := r.scanSessionFromRows(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, totalCount, nil
}

// GetSessionStatistics retrieves session statistics
func (r *BaseStationSessionRepository) GetSessionStatistics(ctx context.Context, tenantID int64) (*interfaces.SessionStatistics, error) {
	query := `
		SELECT
			COUNT(*) as total_sessions,
			COUNT(*) FILTER (WHERE status = 'active') as active_sessions,
			COUNT(*) FILTER (WHERE status = 'terminated') as terminated_sessions,
			COUNT(*) FILTER (WHERE can_resume = true AND status = 'disconnected') as resumable_sessions,
			COALESCE(
				AVG(EXTRACT(EPOCH FROM (COALESCE(ended_at, NOW()) - started_at)) / 3600),
				0
			) as average_session_duration,
			COALESCE(
				SUM(EXTRACT(EPOCH FROM (COALESCE(ended_at, NOW()) - started_at)) / 3600),
				0
			) as total_session_time
		FROM basestation_sessions
		WHERE tenant_id = $1`

	stats := &interfaces.SessionStatistics{}
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&stats.TotalSessions,
		&stats.ActiveSessions,
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

// CheckSessionResumable determines if a session can be resumed per MIOTY spec
func (r *BaseStationSessionRepository) CheckSessionResumable(ctx context.Context, tenantID int64, snBsUUID [16]byte, bsOpId int64) (*interfaces.SessionResumptionInfo, error) {
	query := `
		SELECT
			id,
			sn_bs_op_id,
			sn_sc_op_id,
			status,
			can_resume,
			EXTRACT(EPOCH FROM (NOW() - started_at)) / 3600 as session_age_hours
		FROM basestation_sessions
		WHERE sn_bs_uuid = $1 AND tenant_id = $2
		ORDER BY started_at DESC
		LIMIT 1`

	var (
		sessionID  int64
		lastBsOpId int64
		lastScOpId int64
		status     string
		canResume  bool
		sessionAge float64
	)

	err := r.db.QueryRowContext(ctx, query, snBsUUID[:], tenantID).Scan(
		&sessionID,
		&lastBsOpId,
		&lastScOpId,
		&status,
		&canResume,
		&sessionAge,
	)

	if err == sql.ErrNoRows {
		return &interfaces.SessionResumptionInfo{
			CanResume:            false,
			ReasonIfNotResumable: "No existing session found",
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to check session resumability: %w", err)
	}

	info := &interfaces.SessionResumptionInfo{
		SessionID:       sessionID,
		LastKnownBsOpId: lastBsOpId,
		LastKnownScOpId: lastScOpId,
		SessionAge:      int64(sessionAge),
	}

	if !canResume {
		info.CanResume = false
		info.ReasonIfNotResumable = "Session marked as non-resumable"
		return info, nil
	}

	if status == string(models.SessionStatusTerminated) {
		info.CanResume = false
		info.ReasonIfNotResumable = "Session already terminated"
		return info, nil
	}

	if bsOpId <= lastBsOpId {
		info.CanResume = false
		info.ReasonIfNotResumable = fmt.Sprintf("Operation ID out of sequence: provided=%d, last=%d", bsOpId, lastBsOpId)
		return info, nil
	}

	maxAgeHours := config.MaxSessionResumptionAge.Hours()
	if sessionAge > maxAgeHours {
		info.CanResume = false
		info.ReasonIfNotResumable = fmt.Sprintf("Session too old: %.1f hours (limit %.0f)", sessionAge, maxAgeHours)
		return info, nil
	}

	info.CanResume = true
	return info, nil
}

// CleanupExpiredSessions removes old terminated sessions (housekeeping)
func (r *BaseStationSessionRepository) CleanupExpiredSessions(ctx context.Context, olderThan int64) (int64, error) {
	query := `
		DELETE FROM basestation_sessions
		WHERE status = 'terminated'
		  AND ended_at < NOW() - ($1 || ' hours')::INTERVAL`

	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// scanSession scans a single session from a query row
func (r *BaseStationSessionRepository) scanSession(row *sql.Row) (*models.BaseStationSession, error) {
	session := &models.BaseStationSession{}

	var (
		snBsUUIDBytes   []byte
		snScUUIDBytes   []byte
		connectionID    sql.NullString
		remoteAddr      sql.NullString
		lastPingAt      sql.NullTime
		endedAt         sql.NullTime
		protocolVersion sql.NullString
	)

	err := row.Scan(
		&session.ID,
		&session.BaseStationID,
		&session.TenantID,
		&snBsUUIDBytes,
		&snScUUIDBytes,
		&session.SnBsOpId,
		&session.SnScOpId,
		&session.Status,
		&connectionID,
		&remoteAddr,
		&session.StartedAt,
		&lastPingAt,
		&endedAt,
		&session.CanResume,
		&session.OrganizationID,
		&session.Encoding,
		&protocolVersion,
		&session.ConnectInfo,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan session: %w", err)
	}

	copy(session.SnBsUuid[:], snBsUUIDBytes)
	copy(session.SnScUuid[:], snScUUIDBytes)

	if connectionID.Valid {
		session.ConnectionId = &connectionID.String
	}
	if remoteAddr.Valid {
		session.RemoteAddr = &remoteAddr.String
	}
	if lastPingAt.Valid {
		session.LastPingAt = &lastPingAt.Time
	}
	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}
	if protocolVersion.Valid {
		session.ProtocolVersion = &protocolVersion.String
	}

	return session, nil
}

// scanSessionFromRows scans a single session from query rows (used in ListSessions)
func (r *BaseStationSessionRepository) scanSessionFromRows(rows *sql.Rows) (*models.BaseStationSession, error) {
	session := &models.BaseStationSession{}

	var (
		snBsUUIDBytes   []byte
		snScUUIDBytes   []byte
		connectionID    sql.NullString
		remoteAddr      sql.NullString
		lastPingAt      sql.NullTime
		endedAt         sql.NullTime
		protocolVersion sql.NullString
	)

	err := rows.Scan(
		&session.ID,
		&session.BaseStationID,
		&session.TenantID,
		&snBsUUIDBytes,
		&snScUUIDBytes,
		&session.SnBsOpId,
		&session.SnScOpId,
		&session.Status,
		&connectionID,
		&remoteAddr,
		&session.StartedAt,
		&lastPingAt,
		&endedAt,
		&session.CanResume,
		&session.OrganizationID,
		&session.Encoding,
		&protocolVersion,
		&session.ConnectInfo,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan session: %w", err)
	}

	copy(session.SnBsUuid[:], snBsUUIDBytes)
	copy(session.SnScUuid[:], snScUUIDBytes)

	if connectionID.Valid {
		session.ConnectionId = &connectionID.String
	}
	if remoteAddr.Valid {
		session.RemoteAddr = &remoteAddr.String
	}
	if lastPingAt.Valid {
		session.LastPingAt = &lastPingAt.Time
	}
	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}
	if protocolVersion.Valid {
		session.ProtocolVersion = &protocolVersion.String
	}

	return session, nil
}

// MarkDisconnected marks a session disconnected and resumable, guarded by
// the stored connection ID: when a reconnect already replaced this
// connection, the update matches zero rows and the newer session is left
// untouched (not an error).
func (r *BaseStationSessionRepository) MarkDisconnected(ctx context.Context, tenantID, sessionID int64, connectionID string, endedAt time.Time) error {
	query := `
		UPDATE basestation_sessions
		SET status = $1,
		    can_resume = true,
		    ended_at = $2,
		    updated_at = $2
		WHERE id = $3 AND tenant_id = $4 AND connection_id = $5`

	if _, err := r.db.ExecContext(ctx, query, models.SessionStatusDisconnected, endedAt, sessionID, tenantID, connectionID); err != nil {
		return fmt.Errorf("failed to mark session disconnected: %w", err)
	}
	return nil
}

// FindResumableSession finds the resumable session for a base station,
// scoped by tenant, base station EUI, and snBsUuid, requiring
// status=disconnected and can_resume=true (BSSCI §5.3.1)
func (r *BaseStationSessionRepository) FindResumableSession(ctx context.Context, tenantID int64, bsEUI []byte, snBsUUID [16]byte) (*models.BaseStationSession, error) {
	query := `
		SELECT s.id, s.basestation_id, s.tenant_id, s.sn_bs_uuid, s.sn_sc_uuid,
		       s.sn_bs_op_id, s.sn_sc_op_id, s.status, s.connection_id, s.remote_addr,
		       s.started_at, s.last_ping_at, s.ended_at, s.can_resume, s.encoding,
		       s.protocol_version, s.connect_info, s.organization_id, s.created_at, s.updated_at
		FROM basestation_sessions s
		JOIN basestations b ON b.id = s.basestation_id
		WHERE s.tenant_id = $1
		  AND b.bs_eui = $2
		  AND s.sn_bs_uuid = $3
		  AND s.status = $4
		  AND s.can_resume = true
		ORDER BY s.started_at DESC
		LIMIT 1`

	session := &models.BaseStationSession{}
	var snBsUUIDBytes, snScUUIDBytes []byte
	err := r.db.QueryRowContext(ctx, query, tenantID, bsEUI, snBsUUID[:], models.SessionStatusDisconnected).Scan(
		&session.ID, &session.BaseStationID, &session.TenantID, &snBsUUIDBytes, &snScUUIDBytes,
		&session.SnBsOpId, &session.SnScOpId, &session.Status, &session.ConnectionId, &session.RemoteAddr,
		&session.StartedAt, &session.LastPingAt, &session.EndedAt, &session.CanResume, &session.Encoding,
		&session.ProtocolVersion, &session.ConnectInfo, &session.OrganizationID, &session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find resumable session: %w", err)
	}
	copy(session.SnBsUuid[:], snBsUUIDBytes)
	copy(session.SnScUuid[:], snScUUIDBytes)
	return session, nil
}
