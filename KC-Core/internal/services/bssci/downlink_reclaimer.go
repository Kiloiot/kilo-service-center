package bssciservices

import (
	"context"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// ReservationReclaimer returns to pending the downlinks a base station can no
// longer complete: its reservations, on a fresh session its whole queue, and
// an endpoint's queue an attach propagate made it discard; it names each
// downlink it returned.
type ReservationReclaimer interface {
	ReleaseStationReservations(ctx context.Context, bsEUI uint64, keepQueIDs []int64) ([]storage.PendingDownlink, error)
	ReleaseStationQueue(ctx context.Context, bsEUI uint64) ([]storage.PendingDownlink, error)
	ReleaseEndpointAtStation(ctx context.Context, tenantID int64, epEUI, bsEUI uint64, sentBy time.Time) ([]storage.PendingDownlink, error)
}

// RequeueRecorder records a downlink returned to the queue because the base
// station holding it let it go.
type RequeueRecorder interface {
	RecordRequeued(ctx context.Context, downlink storage.PendingDownlink, bsEUI uint64) error
}

// downlinkReclaimer implements bssci.DownlinkReclaimer: it returns to pending
// the downlinks a base station let go and records each one's return, so every
// open view of the downlink queue shows it pending again.
type downlinkReclaimer struct {
	store  ReservationReclaimer
	events RequeueRecorder
	logger logger.Logger
}

var _ bssci.DownlinkReclaimer = (*downlinkReclaimer)(nil)

// NewDownlinkReclaimer creates the reclaimer; every collaborator is mandatory,
// so a wiring fault surfaces at startup instead of on the first reconnect.
func NewDownlinkReclaimer(store ReservationReclaimer, events RequeueRecorder, log logger.Logger) (bssci.DownlinkReclaimer, error) {
	switch {
	case store == nil:
		return nil, ErrNilReservationReclaimer
	case events == nil:
		return nil, ErrNilRequeueRecorder
	case log == nil:
		return nil, ErrNilReclaimerLogger
	}
	return &downlinkReclaimer{store: store, events: events, logger: log}, nil
}

// ReclaimReservations returns to pending every downlink the base station
// holds reserved except the rows behind the reissued dlDataQue operations:
// a session that is not resumed discards the previous session's state (BSSCI
// §1), so no other reservation of the station can still be confirmed.
func (d *downlinkReclaimer) ReclaimReservations(ctx context.Context, bsEUI uint64, reissuedQueIDs []int64) (int64, error) {
	released, err := d.store.ReleaseStationReservations(ctx, bsEUI, reissuedQueIDs)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errReclaimReservations, err)
	}
	return d.requeued(ctx, bsEUI, released, bssci.LogDispatcherReservationsReclaimed), nil
}

// ReclaimDiscardedQueue returns to pending every downlink queued at a base
// station whose new session is not resumed: the station discarded them with
// the previous session (BSSCI §1), so they are dispatched again at the
// endpoint's next downlink window.
func (d *downlinkReclaimer) ReclaimDiscardedQueue(ctx context.Context, bsEUI uint64) (int64, error) {
	released, err := d.store.ReleaseStationQueue(ctx, bsEUI)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errReclaimDiscardedQueue, err)
	}
	return d.requeued(ctx, bsEUI, released, bssci.LogDispatcherDiscardedQueueReclaimed), nil
}

// ReclaimEndpointQueue returns to pending the owner tenant's downlinks for the
// endpoint that were queued at the base station by the time an attach
// propagate for the endpoint was issued: the station discards them when it
// takes the attachment (BSSCI §3.8), so they are dispatched again at the
// endpoint's next downlink window. A row still reserved, or sent to the
// station after the propagate was issued, may reach it behind the propagate
// and stay queued there, so it keeps its state.
func (d *downlinkReclaimer) ReclaimEndpointQueue(ctx context.Context, ownerTenantID int64, epEUI, bsEUI uint64, propagatedAt time.Time) (int64, error) {
	released, err := d.store.ReleaseEndpointAtStation(ctx, ownerTenantID, epEUI, bsEUI, propagatedAt)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errReclaimEndpointQueue, err)
	}
	return d.requeued(ctx, bsEUI, released, bssci.LogDispatcherEndpointQueueReclaimed, logger.FieldEpEui, epEUI), nil
}

// requeued logs a release under msg, records each released downlink's return
// to the queue and counts them; a record that fails leaves the release in place.
func (d *downlinkReclaimer) requeued(ctx context.Context, bsEUI uint64, released []storage.PendingDownlink, msg string, fields ...interface{}) int64 {
	if len(released) == 0 {
		return 0
	}
	d.logger.InfoContext(ctx, msg, append([]interface{}{logger.FieldBsEui, bsEUI, logger.FieldCount, len(released)}, fields...)...)
	for _, downlink := range released {
		if err := d.events.RecordRequeued(ctx, downlink, bsEUI); err != nil {
			d.logger.ErrorContext(ctx, LogDownlinkRequeueEventFailed,
				logger.FieldQueID, downlink.QueID, logger.FieldBsEui, bsEUI, logger.FieldError, err)
		}
	}
	return int64(len(released))
}
