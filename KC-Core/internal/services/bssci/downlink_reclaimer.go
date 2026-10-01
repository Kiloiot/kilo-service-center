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

// DiscardedRevocations ends expired the downlinks a base station was asked
// to drop, returning them for their originators, once its session that is
// not resumed discarded them.
type DiscardedRevocations interface {
	ExpireStationRevocations(ctx context.Context, bsEUI uint64) ([]*storage.DownlinkMessage, error)
}

// RemovedStationHolds ends expired every downlink a deleted base station
// held, of every tenant, and returns them for their originators.
type RemovedStationHolds interface {
	ExpireRemovedStationDownlinks(ctx context.Context, bsEUI uint64) ([]*storage.DownlinkMessage, error)
}

// StationExpiryReporter tells a downlink's originators it expired at the
// station that dropped it.
type StationExpiryReporter interface {
	ReportExpiredAtStation(ctx context.Context, downlink *storage.DownlinkMessage)
}

// QueueTenantCache forgets the owner cached for a queue id once its downlink ended.
type QueueTenantCache interface {
	UnregisterQueueTenant(queueID int64)
}

// RequeueRecorder records a downlink returned to the queue because the base
// station holding it let it go.
type RequeueRecorder interface {
	RecordRequeued(ctx context.Context, downlink storage.PendingDownlink, bsEUI uint64) error
}

// DownlinkReclaimer implements bssci.DownlinkReclaimer: it settles the
// downlinks a base station let go. It returns to pending the ones the station
// can no longer transmit and records each one's return, so every open view of
// the downlink queue shows it pending again, and reports expired the overdue
// downlinks the station discarded and every downlink a deleted station held.
type DownlinkReclaimer struct {
	store       ReservationReclaimer
	revocations DiscardedRevocations
	removed     RemovedStationHolds
	events      RequeueRecorder
	expiries    StationExpiryReporter
	tenants     QueueTenantCache
	logger      logger.Logger
}

// DownlinkReclaimerDeps are the reclaimer's collaborators, all required.
type DownlinkReclaimerDeps struct {
	Store       ReservationReclaimer
	Revocations DiscardedRevocations
	Removed     RemovedStationHolds
	Events      RequeueRecorder
	Expiries    StationExpiryReporter
	Tenants     QueueTenantCache
	Logger      logger.Logger
}

var _ bssci.DownlinkReclaimer = (*DownlinkReclaimer)(nil)

// NewDownlinkReclaimer creates the reclaimer; every collaborator is mandatory,
// so a wiring fault surfaces at startup instead of on the first reconnect.
func NewDownlinkReclaimer(deps DownlinkReclaimerDeps) (*DownlinkReclaimer, error) {
	switch {
	case deps.Store == nil:
		return nil, ErrNilReservationReclaimer
	case deps.Revocations == nil:
		return nil, ErrNilDiscardedRevocations
	case deps.Removed == nil:
		return nil, ErrNilRemovedStationHolds
	case deps.Events == nil:
		return nil, ErrNilRequeueRecorder
	case deps.Expiries == nil:
		return nil, ErrNilDiscardedExpiryReporter
	case deps.Tenants == nil:
		return nil, ErrNilReclaimerQueueTenants
	case deps.Logger == nil:
		return nil, ErrNilReclaimerLogger
	}
	return &DownlinkReclaimer{
		store: deps.Store, revocations: deps.Revocations, removed: deps.Removed, events: deps.Events,
		expiries: deps.Expiries, tenants: deps.Tenants, logger: deps.Logger,
	}, nil
}

// ReclaimReservations returns to pending every downlink the base station
// holds reserved except the rows behind the reissued dlDataQue operations:
// a session that is not resumed discards the previous session's state (BSSCI
// §1), so no other reservation of the station can still be confirmed.
func (d *DownlinkReclaimer) ReclaimReservations(ctx context.Context, bsEUI uint64, reissuedQueIDs []int64) (int64, error) {
	released, err := d.store.ReleaseStationReservations(ctx, bsEUI, reissuedQueIDs)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errReclaimReservations, err)
	}
	return d.requeued(ctx, bsEUI, released, bssci.LogDispatcherReservationsReclaimed), nil
}

// ReclaimDiscardedQueue settles the downlinks a base station whose new
// session is not resumed discarded with the previous session (BSSCI §1): the
// queued ones return to pending and are dispatched again at the endpoint's
// next downlink window, and the ones it was asked to drop because their
// lifetime ended will never be transmitted, so they end expired and are
// reported so. It counts the downlinks returned to pending.
func (d *DownlinkReclaimer) ReclaimDiscardedQueue(ctx context.Context, bsEUI uint64) (int64, error) {
	released, err := d.store.ReleaseStationQueue(ctx, bsEUI)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errReclaimDiscardedQueue, err)
	}
	requeued := d.requeued(ctx, bsEUI, released, bssci.LogDispatcherDiscardedQueueReclaimed)
	expired, err := d.revocations.ExpireStationRevocations(ctx, bsEUI)
	if err != nil {
		return requeued, fmt.Errorf("%w: %w", errExpireDiscardedRevocations, err)
	}
	d.endedAtStation(ctx, bsEUI, expired, bssci.LogDispatcherDiscardedRevocationsExpired)
	return requeued, nil
}

// EndDeletedStationDownlinks ends expired every downlink a deleted base
// station held, reserved, queued or asked to drop, once its session is closed,
// and reports each to its originators. The station never reports again, and
// one still powered may transmit what it held, so none returns to the queue
// where the endpoint could receive it twice. The work outlives the caller's
// cancellation; a failure is logged and the expiry sweep ends what is left.
func (d *DownlinkReclaimer) EndDeletedStationDownlinks(ctx context.Context, bsEUI uint64) {
	ctx = context.WithoutCancel(ctx)
	expired, err := d.removed.ExpireRemovedStationDownlinks(ctx, bsEUI)
	if err != nil {
		d.logger.ErrorContext(ctx, LogDeletedStationDownlinksUnsettled, logger.FieldBsEui, bsEUI, logger.FieldError, err)
		return
	}
	d.endedAtStation(ctx, bsEUI, expired, LogDeletedStationDownlinksExpired)
}

// endedAtStation forgets the cached owner of each downlink that ended expired
// at the station, reports it to its originators and logs the count under msg.
func (d *DownlinkReclaimer) endedAtStation(ctx context.Context, bsEUI uint64, expired []*storage.DownlinkMessage, msg string) {
	for _, downlink := range expired {
		d.tenants.UnregisterQueueTenant(downlink.QueID)
		d.expiries.ReportExpiredAtStation(ctx, downlink)
	}
	if len(expired) > 0 {
		d.logger.InfoContext(ctx, msg, logger.FieldBsEui, bsEUI, logger.FieldCount, len(expired))
	}
}

// ReclaimEndpointQueue returns to pending the owner tenant's downlinks for the
// endpoint that were queued at the base station by the time an attach
// propagate for the endpoint was issued: the station discards them when it
// takes the attachment (BSSCI §3.8), so they are dispatched again at the
// endpoint's next downlink window. A row still reserved, or sent to the
// station after the propagate was issued, may reach it behind the propagate
// and stay queued there, so it keeps its state.
func (d *DownlinkReclaimer) ReclaimEndpointQueue(ctx context.Context, ownerTenantID int64, epEUI, bsEUI uint64, propagatedAt time.Time) (int64, error) {
	released, err := d.store.ReleaseEndpointAtStation(ctx, ownerTenantID, epEUI, bsEUI, propagatedAt)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errReclaimEndpointQueue, err)
	}
	return d.requeued(ctx, bsEUI, released, bssci.LogDispatcherEndpointQueueReclaimed, logger.FieldEpEui, epEUI), nil
}

// requeued logs a release under msg, records each released downlink's return
// to the queue and counts them; a record that fails leaves the release in place.
func (d *DownlinkReclaimer) requeued(ctx context.Context, bsEUI uint64, released []storage.PendingDownlink, msg string, fields ...interface{}) int64 {
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
