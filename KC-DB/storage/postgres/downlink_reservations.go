package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// Positions of the time parameter the reservation statements bind for the
// lifetime check.
const (
	reserveNextNowParam      = 4
	reserveByQueueIDNowParam = 8
	listPendingNowParam      = 2
)

// DownlinkReservations moves pending downlinks to the base station that
// transmits them and back when the station lets them go.
type DownlinkReservations struct {
	db    sqlx.ExtContext
	clock clock.Clock
}

// ListPendingDownlinks returns the unexpired pending downlinks of every
// tenant, grouped by endpoint in the order a downlink window takes them.
func (r *DownlinkReservations) ListPendingDownlinks(ctx context.Context) (pending []storage.PendingDownlink, err error) {
	query := `
		SELECT que_id, tenant_id, organization_id, ep_eui
		FROM downlink_queue
		WHERE status = $1 AND ` + downlinkNotExpired(listPendingNowParam) + `
		ORDER BY tenant_id, ep_eui, priority DESC, created_at ASC`
	rows, err := r.db.QueryxContext(ctx, query, mioty.DLQueueStatusPending, r.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapListPendingDownlinks, err)
	}
	defer sqlcleanup.CloseRows(rows, errWrapListPendingDownlinks, &err)
	return scanPendingDownlinks(rows, errWrapListPendingDownlinks)
}

// scanPendingDownlinks reads rows of que_id, tenant_id, organization_id and
// ep_eui, wrapping a failure with wrap.
func scanPendingDownlinks(rows *sqlx.Rows, wrap string) ([]storage.PendingDownlink, error) {
	var pending []storage.PendingDownlink
	for rows.Next() {
		var downlink storage.PendingDownlink
		var epEUI []byte
		if err := rows.Scan(&downlink.QueID, &downlink.TenantID, &downlink.OrganizationID, &epEUI); err != nil {
			return nil, fmt.Errorf("%s: %w", wrap, err)
		}
		downlink.EpEUI = mioty.EUI64FromBytes(epEUI)
		pending = append(pending, downlink)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", wrap, err)
	}
	return pending, nil
}

// ReservePendingDownlinkByQueueID reserves one exact pending row scoped to the
// enqueuing organization.
func (r *DownlinkReservations) ReservePendingDownlinkByQueueID(ctx context.Context, tenantID int64, organizationID uuid.UUID, queueID uint64, epEUI []byte, bsEUI uint64) (*storage.DownlinkMessage, error) {
	return reservePendingDownlinkByQueueID(ctx, r.db, r.clock.Now(), tenantID, organizationID, queueID, epEUI, bsEUI)
}

// MarkReservedAsQueued confirms a row the base station holds reserved as
// queued after the wire send; a station the row is not reserved for confirms nothing.
func (r *DownlinkReservations) MarkReservedAsQueued(ctx context.Context, queID uint64, tenantID int64, bsEUI uint64, txTime int64, packetCnt *uint32, orgID *uuid.UUID) error {
	return markReservedAsQueued(ctx, r.db, r.clock.Now(), queID, tenantID, bsEUI, txTime, packetCnt, orgID)
}

// ReserveNextPendingDownlink atomically selects+reserves highest-priority pending downlink
// Uses FOR UPDATE SKIP LOCKED to avoid blocking concurrent dispatchers
// Returns storage.ErrNotFound if no pending downlinks available
// Returns ErrDownlinkAlreadyReserved ONLY when UPDATE affects 0 rows due to race
// The selected row's organization_id is authoritative for the dispatch: the
// caller reads it from the returned message rather than filtering by one.
func (r *DownlinkReservations) ReserveNextPendingDownlink(
	ctx context.Context,
	tenantID int64,
	epEUI []byte,
	bsEUI uint64,
) (*storage.DownlinkMessage, error) {
	// Step 1: Select pending with lock (SKIP LOCKED prevents race)
	// Use column names matching downlink_queue schema
	selectQuery := `
		SELECT id, que_id, ep_eui, tenant_id, organization_id, payload, priority, status,
		       cnt_depend, packet_cnt, format, response_exp, response_prio,
		       dl_wind_req, exp_only, dl_rx_stat_qry, user_data, created_at
		FROM downlink_queue
		WHERE tenant_id = $1
		  AND ep_eui = $2
		  AND status = $3
		  AND ` + downlinkNotExpired(reserveNextNowParam) + `
		ORDER BY priority DESC, created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`

	now := r.clock.Now()
	dl, err := scanDownlinkQueueRow(r.db.QueryRowxContext(ctx, selectQuery, tenantID, epEUI, mioty.DLQueueStatusPending, now))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapSelectPendingDownlink, err)
	}

	if err := r.reserveSelected(ctx, dl.QueID, tenantID, bsEUI, now); err != nil {
		return nil, err
	}
	dl.Status = mioty.DLQueueStatusReserved
	dl.BsEui = bsEUI
	return dl, nil
}

// reserveSelected moves the pending row the reservation selected, still under
// its row lock, to reserved for the base station; ErrDownlinkAlreadyReserved
// only when a concurrent transaction took it first.
func (r *DownlinkReservations) reserveSelected(ctx context.Context, queID int64, tenantID int64, bsEUI uint64, now time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE downlink_queue
		SET status = $1, bs_eui = $2, updated_at = $6
		WHERE que_id = $3 AND tenant_id = $4 AND status = $5`,
		mioty.DLQueueStatusReserved, mioty.EUI64Bytes(bsEUI), queID, tenantID, mioty.DLQueueStatusPending, now)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapReserveDownlink, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapReserveDownlink, err)
	}
	if rows == 0 {
		return ErrDownlinkAlreadyReserved
	}
	return nil
}

// reservePendingDownlinkByQueueID performs the exact-match pending → reserved
// transition as one atomic UPDATE ... RETURNING statement. The row must match
// que_id, tenant_id, ep_eui, and organization_id and be in 'pending' state; a
// NULL organization row never satisfies the request.
func reservePendingDownlinkByQueueID(
	ctx context.Context,
	q sqlExecQuerier,
	now time.Time,
	tenantID int64,
	organizationID uuid.UUID,
	queueID uint64,
	epEUI []byte,
	bsEUI uint64,
) (*storage.DownlinkMessage, error) {
	query := `
		UPDATE downlink_queue
		SET status = $1, bs_eui = $2, updated_at = $8
		WHERE que_id = $3 AND tenant_id = $4 AND ep_eui = $5 AND status = $6
		  AND organization_id = $7 AND ` + downlinkNotExpired(reserveByQueueIDNowParam) + `
		RETURNING ` + downlinkQueueColumns

	dl, err := scanDownlinkQueueRow(q.QueryRowxContext(
		ctx, query,
		mioty.DLQueueStatusReserved,
		mioty.EUI64Bytes(bsEUI),
		queueID,
		tenantID,
		epEUI,
		mioty.DLQueueStatusPending,
		organizationID,
		now,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapReserveDownlinkByQueueID, err)
	}
	dl.BsEui = bsEUI
	return dl, nil
}

// markReservedAsQueued transitions reserved → queued with transmission
// metadata. Idempotent: a row already 'queued', or 'revoking' which only a
// queued row becomes, succeeds unchanged; any other state (pending, failed,
// completed, revoked) is an error because the caller's reservation no longer holds.
func markReservedAsQueued(
	ctx context.Context,
	q sqlExecQuerier,
	now time.Time,
	queID uint64,
	tenantID int64,
	bsEUI uint64,
	txTime int64,
	packetCnt *uint32,
	orgID *uuid.UUID,
) error {
	var transmissionPacketCnt *int64
	if packetCnt != nil {
		counter := int64(*packetCnt)
		transmissionPacketCnt = &counter
	}
	// The status is queued, sent to the station and awaiting transmission;
	// transmission_result stays NULL until dlDataRes reports it (BSSCI §5.14).
	scope := newSQLScope(mioty.DLQueueStatusQueued, txTime, transmissionPacketCnt, now)
	scope.equals(colQueID, queID)
	scope.equals(colTenantID, tenantID)
	scope.equals(colStatus, mioty.DLQueueStatusReserved)
	scope.equals(colBsEUI, mioty.EUI64Bytes(bsEUI))
	scope.organization(orgID)
	result, err := q.ExecContext(ctx, `
		UPDATE downlink_queue
		SET status = $1, transmission_time = $2, transmission_packet_cnt = $3, tx_time = $2, updated_at = $4
		WHERE `+scope.where(), scope.args...)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapMarkDownlinkQueued, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapMarkDownlinkQueued, err)
	}
	if rows > 0 {
		return nil
	}
	return queuedAlready(ctx, q, queID, tenantID, orgID)
}

// confirmedQueued are the states of a downlink whose send was confirmed queued.
var confirmedQueued = []mioty.DLQueueStatus{mioty.DLQueueStatusQueued, mioty.DLQueueStatusRevoking}

// queuedAlready is the idempotent success of a reserved-to-queued move that
// changed no row because a repair path or crash-recovery retry already
// confirmed the send; any other state means the reservation no longer holds.
func queuedAlready(ctx context.Context, q sqlExecQuerier, queID uint64, tenantID int64, orgID *uuid.UUID) error {
	scope := newSQLScope()
	scope.equals(colQueID, queID)
	scope.equals(colTenantID, tenantID)
	scope.organization(orgID)
	var status mioty.DLQueueStatus
	err := q.QueryRowxContext(ctx, `SELECT status FROM downlink_queue WHERE `+scope.where(), scope.args...).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrDownlinkAlreadyReserved
	}
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapMarkDownlinkQueuedStatusCheck, err)
	}
	if slices.Contains(confirmedQueued, status) {
		return nil
	}
	return ErrDownlinkAlreadyReserved
}
