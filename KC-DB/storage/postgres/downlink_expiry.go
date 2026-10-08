package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// DownlinkExpirySweep ends the wait of the downlinks whose lifetime elapsed.
type DownlinkExpirySweep struct {
	db    sqlx.ExtContext
	clock clock.Clock
}

// downlinkNotExpired keeps a downlink whose lifetime has elapsed by the time
// bound to the numbered parameter out of every reservation; the expiry sweep
// ends it instead.
func downlinkNotExpired(nowParam int) string {
	return fmt.Sprintf("(latest_at IS NULL OR latest_at > $%d)", nowParam)
}

// unheldStatuses are the in-flight queue states in which no base station holds a downlink.
var unheldStatuses = []mioty.DLQueueStatus{mioty.DLQueueStatusPending, mioty.DLQueueStatusScheduled}

// heldStatuses are the queue states in which a base station holds a
// downlink: reserved for it, queued at it, or asked to drop it.
var heldStatuses = []mioty.DLQueueStatus{mioty.DLQueueStatusReserved, mioty.DLQueueStatusQueued, mioty.DLQueueStatusRevoking}

// revocableStatuses are the held states a dlDataRev applies to: the station
// confirmed it holds the downlink, and no revoke is under way yet. A reserved
// downlink is revoked once its dlDataQue is confirmed queued, so the
// dlDataRev never overtakes it on the wire.
var revocableStatuses = []mioty.DLQueueStatus{mioty.DLQueueStatusQueued}

// ExpireOverdueUnheld ends expired up to limit downlinks of any tenant whose
// lifetime elapsed while no base station held them, and returns them for
// their originators: no station can transmit them any more.
func (r *DownlinkExpirySweep) ExpireOverdueUnheld(ctx context.Context, limit int) ([]*storage.DownlinkMessage, error) {
	expired, _ := mioty.ResultForQueueStatus(mioty.DLQueueStatusExpired)
	move := overdueMove{status: mioty.DLQueueStatusExpired, result: sql.NullString{String: expired, Valid: true}}
	return r.sweep(ctx, errWrapExpireOverdueDownlinks, limit, unheldStatuses, move, expiredInQueue)
}

// RevokeOverdueHeld moves to revoking up to limit downlinks of any tenant
// whose lifetime elapsed while a base station held them queued, keeping that
// station as their holder and recording askedAt, the start of the sweep that
// asks it, and returns them so it is asked to drop them (BSSCI §3.13); their
// outcome waits for its answer.
func (r *DownlinkExpirySweep) RevokeOverdueHeld(ctx context.Context, askedAt time.Time, limit int) ([]*storage.DownlinkMessage, error) {
	move := overdueMove{status: mioty.DLQueueStatusRevoking, askedAt: sql.NullTime{Time: askedAt.Truncate(storedTimePrecision), Valid: true}}
	return r.sweep(ctx, errWrapRevokeOverdueDownlinks, limit, revocableStatuses, move, revokingOutcome)
}

// overdueMove is what a sweep records on the rows it moves: their status, the
// result that status ends them with and when their holder is asked to drop
// them; a NULL result or ask time keeps the column.
type overdueMove struct {
	status  mioty.DLQueueStatus
	result  sql.NullString
	askedAt sql.NullTime
}

// sweep moves the oldest overdue rows in one of from as move records,
// skipping rows a dispatcher holds locked, and returns each with its holder,
// marked by mark.
func (r *DownlinkExpirySweep) sweep(ctx context.Context, wrap string, limit int, from []mioty.DLQueueStatus,
	move overdueMove, mark func(*storage.DownlinkMessage) *storage.DownlinkMessage,
) (moved []*storage.DownlinkMessage, err error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%s: %w", wrap, storage.ErrInvalidInput)
	}
	rows, err := r.db.QueryxContext(ctx, sqlSweepOverdueDownlinks, move.status, move.result, limit, r.clock.Now(), statusArray(from), move.askedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", wrap, err)
	}
	defer sqlcleanup.CloseRows(rows, wrap, &err)
	return scanOutcomesWithHolder(rows, wrap, mark)
}

// sqlSweepOverdueDownlinks moves the oldest overdue rows in one of the states
// $5, skipping rows a dispatcher holds locked; a NULL result ($2) or ask time
// ($6) keeps the column.
const sqlSweepOverdueDownlinks = `
	UPDATE downlink_queue AS d
	SET status = $1, result = COALESCE($2::text, d.result), transmission_result = COALESCE($2::text, d.transmission_result),
	    revoke_asked_at = COALESCE($6::timestamptz, d.revoke_asked_at), updated_at = $4
	FROM (
		SELECT id FROM downlink_queue
		WHERE status = ANY($5::text[]) AND latest_at <= $4
		ORDER BY latest_at
		LIMIT $3
		FOR UPDATE SKIP LOCKED
	) AS due
	WHERE d.id = due.id` + sqlReturningHeldOutcome

// sqlReturningHeldOutcome returns the outcome columns of a moved row aliased
// d, followed by its holder.
const sqlReturningHeldOutcome = `
	RETURNING d.id, d.que_id, d.ac_que_id, d.ep_eui, d.tenant_id, d.organization_id, d.ac_eui, d.ref, d.bs_eui`

// expiredInQueue marks a downlink ended expired while no station held it; a
// pending row may still name the station of a released reservation.
func expiredInQueue(downlink *storage.DownlinkMessage) *storage.DownlinkMessage {
	downlink.BsEui = 0
	return expiredOutcome(downlink)
}
