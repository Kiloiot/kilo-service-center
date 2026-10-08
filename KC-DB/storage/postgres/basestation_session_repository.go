package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/jmoiron/sqlx"
)

// BaseStationSessionRepository implements the BaseStationSessionRepository interface for PostgreSQL
type BaseStationSessionRepository struct {
	log   logger.Logger
	clock clock.Clock
	db    *sqlx.DB
}

// Ensure BaseStationSessionRepository implements the interface
var _ interfaces.BaseStationSessionRepository = (*BaseStationSessionRepository)(nil)

const setNullClauseFmt = "%s = NULL"

// Reasons a BSSCI session cannot be resumed.
const (
	reasonNoExistingSession    = "No existing session found"
	reasonSessionNonResumable  = "Session marked as non-resumable"
	reasonSessionTerminated    = "Session already terminated"
	reasonFmtOpIDOutOfSequence = "Operation ID out of sequence: provided=%d, last=%d"
	reasonFmtSessionTooOld     = "Session too old: %.1f hours (limit %.0f)"
)

// NewBaseStationSessionRepository creates a new PostgreSQL Base Station session repository
// setNullClauseFmt renders a column-to-NULL assignment in the update
// builder.
func NewBaseStationSessionRepository(db *sqlx.DB, clk clock.Clock, log logger.Logger) *BaseStationSessionRepository {
	return &BaseStationSessionRepository{
		log: log, clock: clk, db: db}
}

// CreateSession creates a new Base Station session per MIOTY BSSCI 3.3
func (r *BaseStationSessionRepository) CreateSession(ctx context.Context, req *models.BaseStationSessionCreateRequest) (*models.BaseStationSession, error) {
	if req == nil {
		return nil, errTextCreateRequestCannotBeNil
	}

	// Default encoding to msgpack if not specified (BSSCI Section 1)
	encoding := req.Encoding
	if encoding == "" {
		encoding = mioty.EncodingMessagePack
	}

	query := `
		INSERT INTO basestation_sessions (
			basestation_id, tenant_id, sn_bs_uuid, sn_sc_uuid,
			sn_bs_op_id, sn_sc_op_id, status,
			connection_id, remote_addr, can_resume,
			organization_id, encoding, protocol_version, connect_info, sc_eui, started_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, COALESCE($14, '{}'::jsonb), $15, NOW(), NOW(), NOW()
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

	err := r.db.QueryRowContext(
		ctx, query,
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
		req.ScEui,
	).Scan(&session.ID, &session.StartedAt, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapCreateBaseStationSession, err)
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
		return errTextUpdateRequestCannotBeNil
	}

	builder, err := buildSessionUpdateClauses(req)
	if err != nil {
		return err
	}
	if len(builder.clauses) == 0 {
		return errTextNoFieldsUpdate
	}

	builder.set("updated_at", r.clock.Now())
	builder.args = append(builder.args, sessionID, tenantID)

	query := fmt.Sprintf(`
		UPDATE basestation_sessions
		SET %s
		WHERE id = $%d AND tenant_id = $%d`,
		strings.Join(builder.clauses, ", "),
		len(builder.args)-1, len(builder.args))

	result, err := r.db.ExecContext(ctx, query, builder.args...)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapUpdateBaseStationSession, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapFailedToGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf(errFmtSessionNotFoundIDTenant, sessionID, tenantID, storage.ErrNotFound)
	}

	return nil
}

// updateSetBuilder accumulates SET clauses with positional placeholders for a
// dynamic partial UPDATE.
type updateSetBuilder struct {
	clauses []string
	args    []interface{}
}

// set appends a column assignment bound to the next positional argument.
func (b *updateSetBuilder) set(column string, value interface{}) {
	b.args = append(b.args, value)
	b.clauses = append(b.clauses, fmt.Sprintf("%s = $%d", column, len(b.args)))
}

// setNull appends a literal NULL assignment for a column.
func (b *updateSetBuilder) setNull(column string) {
	b.clauses = append(b.clauses, fmt.Sprintf(setNullClauseFmt, column))
}

// setIfPresent appends a column assignment when the optional value is set.
func setIfPresent[T any](b *updateSetBuilder, column string, value *T) {
	if value != nil {
		b.set(column, *value)
	}
}

// buildSessionUpdateClauses maps the optional request fields onto SET clauses,
// rejecting a request that both sets and clears ended_at.
func buildSessionUpdateClauses(req *models.BaseStationSessionUpdateRequest) (*updateSetBuilder, error) {
	if req.EndedAt != nil && req.ClearEndedAt {
		return nil, errTextUpdateRequestCannotSetAndClearEndedAt
	}

	b := &updateSetBuilder{}
	setIfPresent(b, "sn_bs_op_id", req.SnBsOpId)
	setIfPresent(b, "sn_sc_op_id", req.SnScOpId)
	setIfPresent(b, "status", req.Status)
	setIfPresent(b, "last_ping_at", req.LastPingAt)
	setIfPresent(b, "ended_at", req.EndedAt)
	if req.ClearEndedAt {
		b.setNull("ended_at")
	}
	setIfPresent(b, "can_resume", req.CanResume)
	setIfPresent(b, "connection_id", req.ConnectionId)
	setIfPresent(b, "remote_addr", req.RemoteAddr)
	setIfPresent(b, "organization_id", req.OrganizationID)
	setIfPresent(b, "encoding", req.Encoding)
	setIfPresent(b, "protocol_version", req.ProtocolVersion)
	setIfPresent(b, "sc_eui", req.ScEui)
	return b, nil
}

// UpdateOperationIDs advances both Base Station and Service Center operation
// IDs atomically. Counters only move forward (BSSCI §3.2: base station IDs
// rise, service center IDs fall), so a snapshot written out of order never
// lowers what a later one stored.
func (r *BaseStationSessionRepository) UpdateOperationIDs(ctx context.Context, tenantID, sessionID int64, bsOpId, scOpId int64) error {
	query := `
		UPDATE basestation_sessions
		SET sn_bs_op_id = GREATEST(sn_bs_op_id, $1),
		    sn_sc_op_id = LEAST(sn_sc_op_id, $2),
		    updated_at = $3
		WHERE id = $4 AND tenant_id = $5`

	result, err := r.db.ExecContext(ctx, query, bsOpId, scOpId, r.clock.Now(), sessionID, tenantID)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapUpdateOperationIDs, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapFailedToGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf(errFmtSessionNotFoundIDTenant, sessionID, tenantID, storage.ErrNotFound)
	}

	return nil
}

// UpdateEncoding updates the message encoding for a session
// This is called when encoding is negotiated on first message per BSSCI Section 1
func (r *BaseStationSessionRepository) UpdateEncoding(ctx context.Context, tenantID, sessionID int64, encoding string) error {
	// Validate encoding value
	if encoding != mioty.EncodingJSON && encoding != mioty.EncodingMessagePack {
		return fmt.Errorf(errFmtInvalidEncodingMustBeOrGot, mioty.EncodingJSON, mioty.EncodingMessagePack, encoding)
	}

	query := `
		UPDATE basestation_sessions
		SET encoding = $1,
		    updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3`

	result, err := r.db.ExecContext(ctx, query, encoding, sessionID, tenantID)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapUpdateEncoding, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapFailedToGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf(errFmtSessionNotFoundIDTenant, sessionID, tenantID, storage.ErrNotFound)
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

	now := r.clock.Now()
	result, err := r.db.ExecContext(ctx, query, models.SessionStatusTerminated, now, sessionID, tenantID)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapTerminateSession, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapFailedToGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf(errFmtSessionNotFoundIDTenant, sessionID, tenantID, storage.ErrNotFound)
	}

	return nil
}

// ListSessions retrieves sessions based on filter criteria
func (r *BaseStationSessionRepository) ListSessions(ctx context.Context, filter *models.BaseStationSessionFilter) ([]*models.BaseStationSession, int64, error) {
	if filter == nil {
		return nil, 0, errTextFilterCannotBeNil
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
		return nil, 0, fmt.Errorf("%s: %w", errWrapCountSessions, err)
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
		return nil, 0, fmt.Errorf("%s: %w", errWrapListSessions, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgCloseRowsBasestationSessions, logger.FieldError, err)
		}
	}()

	sessions := []*models.BaseStationSession{}
	for rows.Next() {
		session, err := r.scanSessionFromRows(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("%s: %w", errWrapScanSession, err)
		}
		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapErrorIteratingSessions, err)
	}

	return sessions, totalCount, nil
}

// GetSessionStatistics retrieves session statistics
func (r *BaseStationSessionRepository) GetSessionStatistics(ctx context.Context, tenantID int64) (*models.SessionStatistics, error) {
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

	stats := &models.SessionStatistics{}
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&stats.TotalSessions,
		&stats.ActiveSessions,
		&stats.TerminatedSessions,
		&stats.ResumableSessions,
		&stats.AverageSessionDuration,
		&stats.TotalSessionTime,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetSessionStatistics, err)
	}

	return stats, nil
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
		return nil, errTextSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapScanSession, err)
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
		return nil, fmt.Errorf("%s: %w", errWrapScanSession, err)
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
