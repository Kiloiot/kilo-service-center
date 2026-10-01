package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// DownlinkStationRemoval ends the downlinks a base station held once the
// station no longer exists: it never gets a session again, so it cannot
// report them, and a station still powered may transmit what it held, so none
// is returned to the queue where an endpoint could receive it twice.
type DownlinkStationRemoval struct {
	db    sqlx.ExtContext
	clock clock.Clock
}

// ExpireRemovedStationDownlinks ends expired every downlink of every tenant
// the deleted base station held, reserved, queued or asked to drop, and
// returns them for their originators.
func (r *DownlinkStationRemoval) ExpireRemovedStationDownlinks(ctx context.Context, bsEUI uint64) (expired []*storage.DownlinkMessage, err error) {
	scope := newSQLScope(mioty.DLQueueStatusExpired, mioty.ResultExpired, r.clock.Now())
	scope.anyOf(colStatus, statusArray(heldStatuses))
	scope.equals(colBsEUI, mioty.EUI64Bytes(bsEUI))
	rows, err := r.db.QueryxContext(ctx, sqlExpireRevoking+scope.where()+sqlReturningRevoked, scope.args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapExpireRemovedStationDownlinks, err)
	}
	defer sqlcleanup.CloseRows(rows, errWrapExpireRemovedStationDownlinks, &err)
	return scanHeldOutcomes(rows, bsEUI, errWrapExpireRemovedStationDownlinks, expiredOutcome)
}

// ExpireHeldAtRemovedStations ends expired up to limit downlinks of any
// tenant held by a base station no registration names any more, one whose
// deletion could not settle them, and returns each with that station. Rows
// another statement holds locked are skipped.
func (r *DownlinkStationRemoval) ExpireHeldAtRemovedStations(ctx context.Context, limit int) (expired []*storage.DownlinkMessage, err error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%s: %w", errWrapExpireHeldAtRemovedStations, storage.ErrInvalidInput)
	}
	rows, err := r.db.QueryxContext(ctx, sqlExpireHeldAtRemovedStations,
		mioty.DLQueueStatusExpired, mioty.ResultExpired, r.clock.Now(), statusArray(heldStatuses), limit)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapExpireHeldAtRemovedStations, err)
	}
	defer sqlcleanup.CloseRows(rows, errWrapExpireHeldAtRemovedStations, &err)
	return scanOutcomesWithHolder(rows, errWrapExpireHeldAtRemovedStations, expiredOutcome)
}

// sqlExpireHeldAtRemovedStations ends expired ($1, result $2, at $3) the
// oldest rows in one of the held states $4 whose holder has no base station
// row, at most $5.
const sqlExpireHeldAtRemovedStations = `
	UPDATE downlink_queue AS d
	SET status = $1, result = $2::text, transmission_result = $2::text, updated_at = $3
	FROM (
		SELECT q.id FROM downlink_queue AS q
		WHERE q.status = ANY($4::text[]) AND q.bs_eui IS NOT NULL
		  AND NOT EXISTS (SELECT 1 FROM basestations AS b WHERE b.bs_eui = q.bs_eui)
		ORDER BY q.id
		LIMIT $5
		FOR UPDATE OF q SKIP LOCKED
	) AS orphaned
	WHERE d.id = orphaned.id` + sqlReturningHeldOutcome
