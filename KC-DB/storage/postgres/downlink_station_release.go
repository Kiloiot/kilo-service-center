package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// sqlReturningReleased names each downlink a release returned to pending.
const sqlReturningReleased = ` RETURNING que_id, tenant_id, organization_id, ep_eui`

// ReleaseStationReservations returns to pending every downlink the base
// station holds reserved, except the rows whose service center queue ids are
// kept, clearing the station as their holder, and returns the rows it
// released. que_id is unique across the installation, so the kept ids
// identify rows of every tenant the station carried.
func (r *DownlinkReservations) ReleaseStationReservations(ctx context.Context, bsEUI uint64, keepQueIDs []int64) ([]storage.PendingDownlink, error) {
	keep := keepQueIDs
	if keep == nil {
		keep = []int64{}
	}
	return r.release(ctx, errWrapReleaseStationReservations, `
		UPDATE downlink_queue
		SET status = $1, bs_eui = NULL, acknowledged_at = NULL, attempts = attempts + 1, updated_at = $5
		WHERE status = $2 AND bs_eui = $3 AND NOT (que_id = ANY($4))`+sqlReturningReleased,
		mioty.DLQueueStatusPending, mioty.DLQueueStatusReserved, mioty.EUI64Bytes(bsEUI), pq.Array(keep), r.clock.Now())
}

// ReleaseStationQueue returns to pending every downlink queued at the base
// station, whichever tenant owns it, clearing the station as its holder, and
// returns the rows it released.
func (r *DownlinkReservations) ReleaseStationQueue(ctx context.Context, bsEUI uint64) ([]storage.PendingDownlink, error) {
	return r.release(ctx, errWrapReleaseStationQueue, `
		UPDATE downlink_queue
		SET status = $1, bs_eui = NULL, acknowledged_at = NULL, attempts = attempts + 1, updated_at = $4
		WHERE status = $2 AND bs_eui = $3`+sqlReturningReleased,
		mioty.DLQueueStatusPending, mioty.DLQueueStatusQueued, mioty.EUI64Bytes(bsEUI), r.clock.Now())
}

// ReleaseEndpointAtStation returns to pending the tenant's downlinks for the
// endpoint that are queued at the base station and were sent to it no later
// than sentBy, clearing the station as their holder, and returns the rows it
// released. A queued row without a recorded send time predates it.
func (r *DownlinkReservations) ReleaseEndpointAtStation(ctx context.Context, tenantID int64, epEUI, bsEUI uint64, sentBy time.Time) ([]storage.PendingDownlink, error) {
	return r.release(ctx, errWrapReleaseEndpointAtStation, `
		UPDATE downlink_queue
		SET status = $1, bs_eui = NULL, acknowledged_at = NULL, attempts = attempts + 1, updated_at = $7
		WHERE tenant_id = $2 AND ep_eui = $3 AND bs_eui = $4 AND status = $5
		  AND (tx_time IS NULL OR tx_time <= $6)`+sqlReturningReleased,
		mioty.DLQueueStatusPending, tenantID, mioty.EUI64Bytes(epEUI), mioty.EUI64Bytes(bsEUI),
		mioty.DLQueueStatusQueued, sentBy.UnixNano(), r.clock.Now())
}

// release runs a release statement and reads the rows it returned to pending.
func (r *DownlinkReservations) release(ctx context.Context, wrap, query string, args ...interface{}) (released []storage.PendingDownlink, err error) {
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", wrap, err)
	}
	defer sqlcleanup.CloseRows(rows, wrap, &err)
	return scanPendingDownlinks(rows, wrap)
}
