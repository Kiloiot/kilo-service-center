package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// ActivateSessionIfResumable applies the resume activation only while the row
// is still disconnected and resumable. Two connections that both found the same
// row through FindResumableSession reach this update, and only the first one
// matches: the loser gets false and must abandon the resume.
func (r *BaseStationSessionRepository) ActivateSessionIfResumable(ctx context.Context, tenantID, sessionID int64, req *models.BaseStationSessionUpdateRequest) (bool, error) {
	if req == nil {
		return false, errTextUpdateRequestCannotBeNil
	}

	builder, err := buildSessionUpdateClauses(req)
	if err != nil {
		return false, err
	}
	if len(builder.clauses) == 0 {
		return false, errTextNoFieldsUpdate
	}

	builder.set("updated_at", r.clock.Now())
	builder.args = append(builder.args, sessionID, tenantID, models.SessionStatusDisconnected)

	query := fmt.Sprintf(`
		UPDATE basestation_sessions
		SET %s
		WHERE id = $%d AND tenant_id = $%d
		  AND status = $%d AND can_resume = true`,
		strings.Join(builder.clauses, ", "),
		len(builder.args)-2, len(builder.args)-1, len(builder.args))

	result, err := r.db.ExecContext(ctx, query, builder.args...)
	if err != nil {
		return false, fmt.Errorf("%s: %w", errWrapActivateResumableSession, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("%s: %w", errWrapFailedToGetRowsAffected, err)
	}

	return rowsAffected > 0, nil
}

// CheckSessionResumable determines if a session can be resumed per MIOTY spec
func (r *BaseStationSessionRepository) CheckSessionResumable(ctx context.Context, tenantID int64, snBsUUID [16]byte, bsOpId int64) (*models.SessionResumptionInfo, error) {
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
		return &models.SessionResumptionInfo{
			CanResume:            false,
			ReasonIfNotResumable: reasonNoExistingSession,
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapCheckSessionResumability, err)
	}

	info := &models.SessionResumptionInfo{
		SessionID:       sessionID,
		LastKnownBsOpId: lastBsOpId,
		LastKnownScOpId: lastScOpId,
		SessionAge:      int64(sessionAge),
	}

	if !canResume {
		info.CanResume = false
		info.ReasonIfNotResumable = reasonSessionNonResumable
		return info, nil
	}

	if status == string(models.SessionStatusTerminated) {
		info.CanResume = false
		info.ReasonIfNotResumable = reasonSessionTerminated
		return info, nil
	}

	if bsOpId <= lastBsOpId {
		info.CanResume = false
		info.ReasonIfNotResumable = fmt.Sprintf(reasonFmtOpIDOutOfSequence, bsOpId, lastBsOpId)
		return info, nil
	}

	maxAgeHours := config.MaxSessionResumptionAge.Hours()
	if sessionAge > maxAgeHours {
		info.CanResume = false
		info.ReasonIfNotResumable = fmt.Sprintf(reasonFmtSessionTooOld, sessionAge, maxAgeHours)
		return info, nil
	}

	info.CanResume = true
	return info, nil
}

// MarkDisconnected marks an active session disconnected and resumable, guarded
// by the stored connection ID and by the active status: a reconnect that
// already replaced this connection, or a session already retired, matches zero
// rows and stays untouched (not an error).
func (r *BaseStationSessionRepository) MarkDisconnected(ctx context.Context, tenantID, sessionID int64, connectionID string, endedAt time.Time) error {
	query := `
		UPDATE basestation_sessions
		SET status = $1,
		    can_resume = true,
		    ended_at = $2,
		    updated_at = $2
		WHERE id = $3 AND tenant_id = $4 AND connection_id = $5 AND status = $6`

	if _, err := r.db.ExecContext(ctx, query, models.SessionStatusDisconnected, endedAt, sessionID, tenantID, connectionID, models.SessionStatusActive); err != nil {
		return fmt.Errorf("%s: %w", errWrapMarkSessionDisconnected, err)
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
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", errWrapFindResumableSession, err)
	}
	copy(session.SnBsUuid[:], snBsUUIDBytes)
	copy(session.SnScUuid[:], snScUUIDBytes)
	return session, nil
}

// TerminateResumableSessions retires the leftover resumable sessions of a base
// station so a fresh session starts from discarded state (BSSCI §3), returning
// the retired session ids for pending-operation cleanup. Zero matches is not an
// error.
func (r *BaseStationSessionRepository) TerminateResumableSessions(ctx context.Context, tenantID, baseStationID int64) ([]int64, error) {
	query := `
		UPDATE basestation_sessions
		SET status = $1,
		    can_resume = false,
		    ended_at = COALESCE(ended_at, $2),
		    updated_at = $2
		WHERE basestation_id = $3
		  AND tenant_id = $4
		  AND status = $5
		  AND can_resume = true
		RETURNING id`

	now := r.clock.Now()
	rows, err := r.db.QueryContext(ctx, query, models.SessionStatusTerminated, now, baseStationID, tenantID, models.SessionStatusDisconnected)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapTerminateResumableSessions, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgCloseRowsResumableRetirement, logger.FieldError, err)
		}
	}()
	return sessionIDsOf(rows)
}

// DisconnectAbandonedSessions hands the sessions a previous process of the
// service center scEUI left active back resumable, as a lost connection's
// teardown would have, and returns their ids. A row without an owner predates
// ownership tracking and is claimed for scEUI; rows of other service centers
// are not touched. The last write to a row is the closest known end time.
func (r *BaseStationSessionRepository) DisconnectAbandonedSessions(ctx context.Context, scEUI models.EUI) ([]int64, error) {
	// sc_eui IS NULL adopts the rows written before migration 000159 once:
	// the migration cannot fill the column because the service center EUI is
	// configuration, and the first start after the upgrade claims them.
	query := `
		UPDATE basestation_sessions
		SET status = $1,
		    can_resume = true,
		    sc_eui = $2,
		    ended_at = updated_at,
		    updated_at = $3
		WHERE status = $4
		  AND (sc_eui = $2 OR sc_eui IS NULL)
		RETURNING id`

	rows, err := r.db.QueryContext(ctx, query, models.SessionStatusDisconnected, scEUI, r.clock.Now(), models.SessionStatusActive)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapDisconnectAbandonedSessions, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgCloseRowsAbandonedSessions, logger.FieldError, err)
		}
	}()
	return sessionIDsOf(rows)
}
