package bssci_test

import (
	"context"
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// The release half of the station queue: the rows a station let go return to
// pending and are named, as the store returns them.

func (q *stationQueue) ReleaseStationReservations(context.Context, uint64, []int64) ([]storage.PendingDownlink, error) {
	return nil, nil
}

func (q *stationQueue) ReleaseStationQueue(context.Context, uint64) ([]storage.PendingDownlink, error) {
	return nil, nil
}

func (q *stationQueue) ReleaseEndpointAtStation(_ context.Context, tenantID int64, _, bsEUI uint64, sentBy time.Time) ([]storage.PendingDownlink, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var released []storage.PendingDownlink
	for _, row := range q.rows {
		if row.status == mioty.DLQueueStatusQueued && row.holder == bsEUI && !row.sentAt.After(sentBy) {
			row.status, row.holder = mioty.DLQueueStatusPending, 0
			queID, _ := row.message.WireQueueID()
			released = append(released, storage.PendingDownlink{QueID: queID, TenantID: tenantID, OrganizationID: q.org, EpEUI: row.epEUI})
		}
	}
	return released, nil
}

// requeueLog records the downlinks announced as returned to the queue.
type requeueLog struct {
	mu       sync.Mutex
	requeued []uint64
}

func (r *requeueLog) RecordRequeued(_ context.Context, downlink storage.PendingDownlink, _ uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requeued = append(r.requeued, downlink.QueID)
	return nil
}

func (r *requeueLog) queueIDs() []uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]uint64(nil), r.requeued...)
}
