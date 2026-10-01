package postgres

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// DownlinkQueueWriter enqueues downlinks and changes them while no base
// station holds them yet.
type DownlinkQueueWriter struct {
	db    *sqlx.DB
	clock clock.Clock
}

// storedTimePrecision is the resolution of a PostgreSQL timestamptz; times
// are cut to it before they are compared, so the comparison sees what is stored.
const storedTimePrecision = time.Microsecond

// EnqueueDownlink adds a downlink to the queue; it waits for a downlink
// window for lifetime from now, or until the deadline of the MQTT command
// that queued it when that comes first, and then expires. A ref the
// organization already queued for the endpoint queues nothing:
// storage.ErrDownlinkRefTaken, whether or not the deadline passed since. A
// deadline that is not after the moment the downlink is queued:
// storage.ErrDownlinkDeadlineElapsed.
func (r *DownlinkQueueWriter) EnqueueDownlink(ctx context.Context, downlink *storage.DownlinkMessage, lifetime time.Duration) (*storage.DownlinkMessage, error) {
	// Every queue row must carry its owning organization: dispatch derives the
	// delivery organization from the row, so an ownerless row is undeliverable.
	if downlink.OrganizationID == nil || *downlink.OrganizationID == uuid.Nil {
		return nil, fmt.Errorf("%s: %w", errWrapEnqueueDownlinkMissingOrg, storage.ErrInvalidInput)
	}
	epEuiBytes, err := hex.DecodeString(downlink.EPEUI)
	if err != nil || len(epEuiBytes) != 8 {
		return nil, fmt.Errorf(errFmtInvalidEPEUI, downlink.EPEUI)
	}
	tenantID, err := strconv.ParseInt(downlink.TenantID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf(errFmtInvalidTenantID, downlink.TenantID)
	}
	userDataJSON, err := encodeDownlinkUserData(downlink.CntDepend, downlink.UserData)
	if err != nil {
		return nil, err
	}
	command := storage.DownlinkCommandRef{
		TenantID: tenantID, OrganizationID: *downlink.OrganizationID, EpEUI: mioty.EUI64FromBytes(epEuiBytes), Ref: downlink.Ref,
	}
	enqueuedAt := r.clock.Now().Truncate(storedTimePrecision)
	err = r.inCommandTransaction(ctx, command, func(tx *sqlx.Tx) error {
		if err := admitDeadline(ctx, tx, command, downlink.ExpiresAt, enqueuedAt); err != nil {
			return err
		}
		return insertQueuedDownlink(ctx, tx, downlink, epEuiBytes, tenantID, userDataJSON, enqueuedAt, lifetime)
	})
	if err != nil {
		return nil, err
	}
	downlink.UpdatedAt = downlink.CreatedAt
	return downlink, nil
}

// insertQueuedDownlink inserts the pending row, its deadline cut to the stored
// precision so it is never stored at or before enqueuedAt once admitted.
func insertQueuedDownlink(ctx context.Context, tx *sqlx.Tx, downlink *storage.DownlinkMessage, epEUI []byte, tenantID int64,
	userDataJSON []byte, enqueuedAt time.Time, lifetime time.Duration,
) error {
	var packetCntArray interface{}
	if len(downlink.PacketCntArray) > 0 {
		packetCntArray = pq.Array(downlink.PacketCntArray)
	}
	maxAttempts := downlink.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = defaultDownlinkMaxAttempts
	}
	var deadline *time.Time
	if downlink.ExpiresAt != nil {
		truncated := downlink.ExpiresAt.Truncate(storedTimePrecision)
		deadline = &truncated
	}
	err := tx.QueryRowxContext(ctx, sqlEnqueueDownlink,
		epEUI, tenantID, downlink.OrganizationID, queuePayload(downlink.Payload),
		downlink.Priority, downlink.Status, downlink.Attempts, maxAttempts,
		downlink.QueID, downlink.CntDepend, packetCntArray, int(downlink.Format),
		downlink.ResponseExp, downlink.ResponsePrio, downlink.DlWindReq, downlink.ExpOnly,
		downlink.DlRxStatQry, userDataJSON, applicationQueueIDParam(downlink.ACQueID),
		enqueuedAt, enqueuedAt.Add(lifetime), optionalEUIParam(downlink.ACEUI), downlink.Ref, deadline,
	).Scan(&downlink.ID, &downlink.CreatedAt)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapEnqueueDownlink, classifyEnqueueError(err))
	}
	return nil
}

// sqlEnqueueDownlink inserts a pending row whose window opens at $20 and
// closes at $21 or the command's deadline $24, whichever comes first (LEAST
// ignores a NULL deadline); an empty ref ($23) is stored as NULL.
const sqlEnqueueDownlink = `
	INSERT INTO downlink_queue (
		ep_eui, tenant_id, organization_id, payload,
		priority, status, attempts, max_attempts,
		que_id, cnt_depend, packet_cnt, format,
		response_exp, response_prio, dl_wind_req, exp_only,
		dl_rx_stat_qry, user_data, ac_que_id, created_at, earliest_at, latest_at,
		ac_eui, ref
	) VALUES (
		$1, $2, $3, $4,
		$5, $6, $7, $8,
		$9, $10, $11, $12,
		$13, $14, $15, $16,
		$17, $18, $19, $20, $20, LEAST($21::timestamptz, $24::timestamptz),
		$22, NULLIF($23::text, '')
	) RETURNING id, created_at`

// optionalEUIParam renders a nullable EUI column; nil is NULL.
func optionalEUIParam(eui *uint64) interface{} {
	if eui == nil {
		return nil
	}
	return mioty.EUI64Bytes(*eui)
}

// classifyEnqueueError separates the collisions an insert can hit: the
// service center's own que_id, which a fresh id resolves, the MQTT command's
// ref, which the organization already queued for the endpoint, and the
// Application Center's id, which its organization already has in flight.
func classifyEnqueueError(err error) error {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code != pqCodeUniqueViolation {
		return err
	}
	switch pqErr.Constraint {
	case constraintDownlinkQueueID:
		return storage.ErrDownlinkQueueIDTaken
	case constraintDownlinkCommandRef:
		return storage.ErrDownlinkRefTaken
	default:
		return storage.ErrDuplicateKey
	}
}

// queuePayload renders a downlink payload for the NOT NULL payload column: a
// pure acknowledgement downlink (SCACI §3.10) carries no user data and is
// stored as zero bytes.
func queuePayload(payload []byte) []byte {
	if payload == nil {
		return []byte{}
	}
	return payload
}

// UpdateDownlinkStatus updates the status of a downlink message named by its
// queue id or, at or below queueIDThreshold, its row id. orgID filters by
// organization with strict equality (a NULL organization row never matches);
// nil means no filter.
func (r *DownlinkQueueWriter) UpdateDownlinkStatus(ctx context.Context, id string, status mioty.DLQueueStatus, orgID *uuid.UUID) error {
	key, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return fmt.Errorf(errFmtInvalidDownlinkID, id)
	}
	keyColumn := colID
	if key > queueIDThreshold {
		keyColumn = colQueID
	}
	scope := newSQLScope(status, r.clock.Now())
	scope.equals(keyColumn, key)
	scope.organization(orgID)
	result, err := r.db.ExecContext(ctx, `
		UPDATE downlink_queue
		SET status = $1, updated_at = $2, attempts = attempts + 1
		WHERE `+scope.where(), scope.args...)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapUpdateDownlinkStatus, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapGetAffectedRows, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf(errFmtDownlinkMessageNotFound, id)
	}
	return nil
}

// UpdatePendingDownlink replaces the SCACI §3.10.1 fields of a pending
// downlink in one statement scoped to the owning tenant and endpoint, so a
// dispatcher that reserves the row first wins and a foreign endpoint's queue
// id never touches another endpoint's row.
func (r *DownlinkQueueWriter) UpdatePendingDownlink(ctx context.Context, tenantID int64, orgID *uuid.UUID, epEUI []byte, queID int64, patch storage.DownlinkPatch) (*storage.DownlinkMessage, error) {
	userDataJSON, err := encodeDownlinkUserData(patch.CntDepend, patch.Payloads)
	if err != nil {
		return nil, err
	}
	var packetCnt interface{}
	if len(patch.PacketCnt) > 0 {
		packetCnt = pq.Array(patch.PacketCnt)
	}
	scope := newSQLScope(
		queuePayload(patchPayload(patch)), patch.Priority, patch.CntDepend, packetCnt, int(patch.Format),
		patch.ResponseExp, patch.ResponsePrio, patch.DlWindReq, patch.ExpOnly, patch.DlRxStatQry, userDataJSON,
		r.clock.Now(),
	)
	scopePendingDownlinkRow(scope, tenantID, orgID, epEUI, queID)
	scope.equals(colStatus, mioty.DLQueueStatusPending)
	query := `
		UPDATE downlink_queue
		SET payload = $1, priority = $2, cnt_depend = $3, packet_cnt = $4, format = $5,
		    response_exp = $6, response_prio = $7, dl_wind_req = $8, exp_only = $9, dl_rx_stat_qry = $10,
		    user_data = $11, updated_at = $12
		WHERE ` + scope.where() + `
		RETURNING ` + downlinkQueueColumns
	dl, err := scanDownlinkQueueRow(r.db.QueryRowxContext(ctx, query, scope.args...))
	if err == nil {
		return dl, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%s: %w", errWrapUpdatePendingDownlink, err)
	}
	return nil, r.notPendingReason(ctx, tenantID, orgID, epEUI, queID)
}

// notPendingReason tells whether a downlink no pending row matched is gone or
// moved on, looking only inside the same tenant/endpoint/organization scope
// so a foreign queue id is indistinguishable from a missing one.
func (r *DownlinkQueueWriter) notPendingReason(ctx context.Context, tenantID int64, orgID *uuid.UUID, epEUI []byte, queID int64) error {
	scope := newSQLScope()
	scopePendingDownlinkRow(scope, tenantID, orgID, epEUI, queID)
	var existingStatus string
	err := r.db.QueryRowxContext(ctx, `SELECT status FROM downlink_queue WHERE `+scope.where(), scope.args...).Scan(&existingStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return storage.ErrDownlinkNotFound
	}
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapUpdatePendingDownlink, err)
	}
	return storage.ErrDownlinkNotPending
}

// scopePendingDownlinkRow narrows a scope to the row an edit names.
func scopePendingDownlinkRow(scope *sqlScope, tenantID int64, orgID *uuid.UUID, epEUI []byte, queID int64) {
	scope.equals(colQueID, queID)
	scope.equals(colTenantID, tenantID)
	scope.equals(colEpEUI, epEUI)
	scope.organization(orgID)
}

// patchPayload is the entry the payload column keeps: the first one, none
// for a pure acknowledgement.
func patchPayload(patch storage.DownlinkPatch) []byte {
	if len(patch.Payloads) == 0 {
		return nil
	}
	return patch.Payloads[0]
}
