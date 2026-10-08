package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/jmoiron/sqlx"
)

// SCACISessionRepository implements the SCACISessionRepository interface for
// PostgreSQL, over the database or over one transaction
// (Transaction.SCACISessions). Its writes take their time from the injected
// clock, except updated_at, which the table's update trigger stamps.
type SCACISessionRepository struct {
	log   logger.Logger
	clock clock.Clock
	db    sqlx.ExtContext
}

// Ensure SCACISessionRepository implements the interface
var _ interfaces.SCACISessionRepository = (*SCACISessionRepository)(nil)

// Error-wrap prefix of the retirement of earlier sessions.
const errWrapRetirePriorSessions = "failed to retire prior SCACI sessions"

// Reasons a SCACI session cannot be resumed.
const (
	scaciReasonSessionNotFound = "session not found"
	scaciReasonNotResumable    = "session marked as not resumable"
	scaciReasonTerminated      = "session already terminated"
)

// NewSCACISessionRepository creates a new PostgreSQL SCACI session repository.
func NewSCACISessionRepository(db *sqlx.DB, clk clock.Clock, log logger.Logger) *SCACISessionRepository {
	return &SCACISessionRepository{
		log: log, clock: clk, db: db}
}

// Retirement is serialized per Application Center - tenant, organization and
// acEui, the key of the one-active-session index - for the rest of the
// transaction it runs in, so two fresh connects of one Application Center
// cannot both pass it and collide on that index.
const (
	sqlLockApplicationCenterSessions = `SELECT pg_advisory_xact_lock(hashtextextended(
		$1::text || '/' || COALESCE($2::uuid::text, '') || '/' || encode($3::bytea, 'hex'), 0))`
	sqlRetirePriorSessions = `
		UPDATE scaci_sessions
		SET status = $1, can_resume = false, disconnected_at = COALESCE(disconnected_at, $2)
		WHERE tenant_id = $3 AND organization_id IS NOT DISTINCT FROM $4 AND ac_eui = $5 AND status <> $1`
)

// CreateSession inserts a new SCACI session. It retires nothing: the earlier
// sessions of the application center are retired by RetirePriorSessions, in
// the transaction that creates the new one.
func (r *SCACISessionRepository) CreateSession(ctx context.Context, req *models.SCACISessionCreateRequest) (*models.SCACISession, error) {
	if req == nil {
		return nil, errTextCreateRequestCannotBeNil
	}

	// Serialize metadata to JSONB
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapMarshalMetadata, err)
	}

	query := `
		INSERT INTO scaci_sessions (
			tenant_id, ac_eui, sn_ac_uuid, sn_sc_uuid,
			last_op_id_ac, last_op_id_sc, status,
			certificate_fingerprint, client_cert_subject, remote_addr,
			tls_version, cipher_suite, negotiated_version,
			can_resume, metadata, organization_id, sc_eui,
			connected_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $18, $18
		)
		RETURNING id, connected_at, created_at, updated_at`

	session := newSessionFromRequest(req)
	err = r.db.QueryRowxContext(
		ctx, query,
		req.TenantID,
		req.AcEUI[:],
		req.SnAcUUID[:],
		req.SnScUUID[:],
		0, // last_op_id_ac
		0, // last_op_id_sc
		models.SCACISessionStatusActive,
		req.CertificateFingerprint,
		req.ClientCertSubject,
		req.RemoteAddr,
		req.TLSVersion,
		req.CipherSuite,
		req.NegotiatedVersion,
		req.CanResume,
		metadataJSON,
		req.OrganizationID,
		req.ScEui[:],
		r.clock.Now(),
	).Scan(&session.ID, &session.ConnectedAt, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapCreateSCACISession, err)
	}
	return session, nil
}

// newSessionFromRequest is the session row a create request inserts, before
// the database assigns its ID and timestamps.
func newSessionFromRequest(req *models.SCACISessionCreateRequest) *models.SCACISession {
	return &models.SCACISession{
		TenantID:               req.TenantID,
		AcEUI:                  req.AcEUI,
		SnAcUUID:               req.SnAcUUID,
		SnScUUID:               req.SnScUUID,
		Status:                 models.SCACISessionStatusActive,
		CertificateFingerprint: req.CertificateFingerprint,
		ClientCertSubject:      req.ClientCertSubject,
		RemoteAddr:             req.RemoteAddr,
		TLSVersion:             req.TLSVersion,
		CipherSuite:            req.CipherSuite,
		NegotiatedVersion:      req.NegotiatedVersion,
		CanResume:              req.CanResume,
		Metadata:               req.Metadata,
	}
}

// RetirePriorSessions terminates every session of the application center, so
// the session created next in the same transaction starts from discarded
// state (SCACI §1: a new session discards the state of the previous one, also
// of a connection that was never torn down). Another organization's sessions
// of the same acEui are left alone. It holds the application center's lock
// until the transaction ends.
func (r *SCACISessionRepository) RetirePriorSessions(ctx context.Context, ac models.SCACIApplicationCenter) error {
	if _, err := r.db.ExecContext(ctx, sqlLockApplicationCenterSessions, ac.TenantID, ac.OrganizationID, ac.AcEUI[:]); err != nil {
		return fmt.Errorf("%s: %w", errWrapRetirePriorSessions, err)
	}
	if _, err := r.db.ExecContext(ctx, sqlRetirePriorSessions, models.SCACISessionStatusTerminated, r.clock.Now(),
		ac.TenantID, ac.OrganizationID, ac.AcEUI[:]); err != nil {
		return fmt.Errorf("%s: %w", errWrapRetirePriorSessions, err)
	}
	return nil
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

	return r.scanSession(r.db.QueryRowxContext(ctx, query, sessionID, tenantID))
}

// sqlApplicationCenterSession is the predicate of the application center's
// session of an snAcUuid: another application center's session of the same
// snAcUuid is never it.
const sqlApplicationCenterSession = `tenant_id = $1 AND organization_id IS NOT DISTINCT FROM $2 AND ac_eui = $3 AND sn_ac_uuid = $4`

// GetSessionByAcUUID retrieves the application center's latest session of
// its session UUID snAcUUID.
func (r *SCACISessionRepository) GetSessionByAcUUID(ctx context.Context, ac models.SCACIApplicationCenter, snAcUUID [16]byte) (*models.SCACISession, error) {
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
		WHERE ` + sqlApplicationCenterSession + `
		ORDER BY connected_at DESC
		LIMIT 1`

	return r.scanSession(r.db.QueryRowxContext(ctx, query, ac.TenantID, ac.OrganizationID, ac.AcEUI[:], snAcUUID[:]))
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

	return r.scanSession(r.db.QueryRowxContext(ctx, query, tenantID, snScUUID[:]))
}

// ResumeSession records the connection a resume moved the session to (SCACI
// §1): active again and owned by the resuming service center, with its
// heartbeat, TLS evidence and metadata. A session
// that is no longer resumable is left as it is and reported as
// storage.ErrNotFound.
func (r *SCACISessionRepository) ResumeSession(ctx context.Context, tenantID, sessionID int64, req *models.SCACISessionResume) error {
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapMarshalMetadata, err)
	}
	query := `
		UPDATE scaci_sessions
		SET status = $1, last_heartbeat = $2,
			tls_version = COALESCE($3, tls_version), cipher_suite = COALESCE($4, cipher_suite), metadata = $5, sc_eui = $9
		WHERE id = $6 AND tenant_id = $7 AND can_resume AND status <> $8`

	result, err := r.db.ExecContext(ctx, query, models.SCACISessionStatusActive, r.clock.Now(),
		req.TLSVersion, req.CipherSuite, metadataJSON, sessionID, tenantID, models.SCACISessionStatusTerminated, req.ScEui[:])
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapUpdateSession, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapFailedToGetRowsAffected, err)
	}
	if rows == 0 {
		return fmt.Errorf(errFmtSCACISessionNotResumable, storage.ErrNotFound, sessionID, tenantID)
	}
	return nil
}

// UpdateOperationIDs advances both operation ID counters atomically. Writers
// race, so a stale pair never moves a counter back: the AC counter only grows
// and the SC counter only shrinks (SCACI §3.2).
func (r *SCACISessionRepository) UpdateOperationIDs(ctx context.Context, tenantID, sessionID int64, acOpId, scOpId int64) error {
	query := `
		UPDATE scaci_sessions
		SET last_op_id_ac = GREATEST(last_op_id_ac, $1), last_op_id_sc = LEAST(last_op_id_sc, $2)
		WHERE id = $3 AND tenant_id = $4`

	result, err := r.db.ExecContext(ctx, query, acOpId, scOpId, sessionID, tenantID)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapUpdateOperationIDs, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapFailedToGetRowsAffected, err)
	}
	if rows == 0 {
		return fmt.Errorf(errFmtSCACISessionNotFoundIDTenant, sessionID, tenantID)
	}

	return nil
}

// UpdateHeartbeat updates the last heartbeat timestamp
func (r *SCACISessionRepository) UpdateHeartbeat(ctx context.Context, tenantID, sessionID int64) error {
	query := `
		UPDATE scaci_sessions
		SET last_heartbeat = $3
		WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.ExecContext(ctx, query, sessionID, tenantID, r.clock.Now())
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapUpdateHeartbeat, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapFailedToGetRowsAffected, err)
	}
	if rows == 0 {
		return fmt.Errorf(errFmtSCACISessionNotFoundIDTenant, sessionID, tenantID)
	}

	return nil
}

// MarkSessionDisconnected records the loss of a session's connection. The
// session stays resumable (SCACI §1); a session a newer one already replaced
// stays terminated.
func (r *SCACISessionRepository) MarkSessionDisconnected(ctx context.Context, tenantID, sessionID int64) error {
	query := `
		UPDATE scaci_sessions
		SET status = $1, disconnected_at = $6
		WHERE id = $2 AND tenant_id = $3 AND status IN ($4, $5)`

	if _, err := r.db.ExecContext(ctx, query,
		models.SCACISessionStatusDisconnected, sessionID, tenantID,
		models.SCACISessionStatusActive, models.SCACISessionStatusResumed, r.clock.Now(),
	); err != nil {
		return fmt.Errorf("%s: %w", errWrapMarkSessionDisconnected, err)
	}
	return nil
}

// TerminateSession marks a session as terminated
func (r *SCACISessionRepository) TerminateSession(ctx context.Context, tenantID, sessionID int64) error {
	query := `
		UPDATE scaci_sessions
		SET status = $3, can_resume = false, disconnected_at = $4
		WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.ExecContext(ctx, query, sessionID, tenantID, models.SCACISessionStatusTerminated, r.clock.Now())
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapTerminateSession, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapFailedToGetRowsAffected, err)
	}
	if rows == 0 {
		return fmt.Errorf(errFmtSCACISessionNotFoundIDTenant, sessionID, tenantID)
	}

	return nil
}

// ListSessions retrieves sessions based on filter criteria
func (r *SCACISessionRepository) ListSessions(ctx context.Context, filter *models.SCACISessionFilter) ([]*models.SCACISession, int64, error) {
	if filter == nil {
		filter = &models.SCACISessionFilter{}
	}

	if filter.TenantID == nil {
		return nil, 0, errTextTenantIDRequired
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
	err := r.db.QueryRowxContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapCountSessions, err)
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
		return nil, 0, fmt.Errorf("%s: %w", errWrapQuerySessions, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgCloseRowsSCACISessions, logger.FieldError, err)
		}
	}()

	sessions := []*models.SCACISession{}
	for rows.Next() {
		session, err := r.scanSessionFromRows(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("%s: %w", errWrapScanSession, err)
		}
		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapRowsIterationError, err)
	}

	return sessions, total, nil
}

// ListResumableSessions returns the sessions of every tenant an Application
// Center may still resume (SCACI §1), under the conditions
// CheckSessionResumable applies.
func (r *SCACISessionRepository) ListResumableSessions(ctx context.Context) ([]*models.SCACISession, error) {
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
		WHERE can_resume AND status <> $1
		ORDER BY id`

	rows, err := r.db.QueryContext(ctx, query, models.SCACISessionStatusTerminated)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapQuerySessions, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgCloseRowsSCACISessions, logger.FieldError, err)
		}
	}()

	var sessions []*models.SCACISession
	for rows.Next() {
		session, err := r.scanSessionFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanSession, err)
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapRowsIterationError, err)
	}
	return sessions, nil
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
			COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(disconnected_at, $2::timestamptz) - connected_at)) / 3600), 0) as avg_duration_hours,
			COALESCE(SUM(EXTRACT(EPOCH FROM (COALESCE(disconnected_at, $2::timestamptz) - connected_at)) / 3600), 0) as total_time_hours
		FROM scaci_sessions
		WHERE tenant_id = $1`

	stats := &models.SCACISessionStatistics{TenantID: tenantID}
	err := r.db.QueryRowxContext(ctx, query, tenantID, r.clock.Now()).Scan(
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
		return nil, fmt.Errorf("%s: %w", errWrapGetSessionStatistics, err)
	}

	return stats, nil
}

// CheckSessionResumable reports whether the application center's latest
// session of snAcUUID can still be resumed (SCACI §1) and returns the
// operation ID counters and the negotiated version (§§2.1-2.3) it stored.
// Whether the application center's view of the counters agrees with them is
// the SCACI rule's to decide (scaci.ResumeOpIDConflict, §3.3.1).
func (r *SCACISessionRepository) CheckSessionResumable(ctx context.Context, ac models.SCACIApplicationCenter, snAcUUID [16]byte) (*models.SCACISessionResumptionInfo, error) {
	query := `
		SELECT
			id, last_op_id_ac, last_op_id_sc, can_resume, status, negotiated_version,
			EXTRACT(EPOCH FROM ($5::timestamptz - connected_at)) / 3600 as session_age_hours
		FROM scaci_sessions
		WHERE ` + sqlApplicationCenterSession + `
		ORDER BY connected_at DESC
		LIMIT 1`

	var sessionID, lastOpIDAc, lastOpIDSc int64
	var canResume bool
	var status string
	var negotiatedVersion string
	var sessionAgeHours float64

	err := r.db.QueryRowxContext(ctx, query, ac.TenantID, ac.OrganizationID, ac.AcEUI[:], snAcUUID[:], r.clock.Now()).Scan(
		&sessionID, &lastOpIDAc, &lastOpIDSc, &canResume, &status, &negotiatedVersion, &sessionAgeHours,
	)

	if err == sql.ErrNoRows {
		return &models.SCACISessionResumptionInfo{
			CanResume:            false,
			ReasonIfNotResumable: scaciReasonSessionNotFound,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapCheckSessionResumability, err)
	}

	info := &models.SCACISessionResumptionInfo{
		SessionID:         sessionID,
		LastKnownAcOpId:   lastOpIDAc,
		LastKnownScOpId:   lastOpIDSc,
		NegotiatedVersion: negotiatedVersion,
		SessionAgeHours:   sessionAgeHours,
	}

	info.ReasonIfNotResumable = resumptionRefusal(canResume, status)
	info.CanResume = info.ReasonIfNotResumable == ""
	return info, nil
}

// resumptionRefusal is why a stored session can no longer be resumed (SCACI
// §1), or "" when it can.
func resumptionRefusal(canResume bool, status string) string {
	if !canResume {
		return scaciReasonNotResumable
	}
	if status == models.SCACISessionStatusTerminated {
		return scaciReasonTerminated
	}
	return ""
}

// scanSession is a helper to scan a single session from a query row
func (r *SCACISessionRepository) scanSession(row rowScanner) (*models.SCACISession, error) {
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

	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapScanSession, err)
	}

	// Convert byte arrays to fixed-size arrays
	copy(session.AcEUI[:], acEuiBytes)
	copy(session.SnAcUUID[:], snAcUUIDBytes)
	copy(session.SnScUUID[:], snScUUIDBytes)

	// Unmarshal metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &session.Metadata); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapUnmarshalMetadata, err)
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
		return nil, fmt.Errorf("%s: %w", errWrapScanSession, err)
	}

	// Convert byte arrays to fixed-size arrays
	copy(session.AcEUI[:], acEuiBytes)
	copy(session.SnAcUUID[:], snAcUUIDBytes)
	copy(session.SnScUUID[:], snScUUIDBytes)

	// Unmarshal metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &session.Metadata); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapUnmarshalMetadata, err)
		}
	}

	return &session, nil
}
