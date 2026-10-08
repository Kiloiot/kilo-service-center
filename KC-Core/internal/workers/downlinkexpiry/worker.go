// Package downlinkexpiry ends the wait of the downlinks that outlived their
// lifetime before a base station transmitted them: one no station holds is
// expired and reported so, and the station holding one is asked to drop it,
// which reports it expired once that station answers. A connected station
// that has not answered is asked again once per sweep interval; nothing here
// ends a downlink a connected station holds just because time passed. A
// downlink held by a station that no longer exists, whose deletion could not
// end it, is expired and reported here.
package downlinkexpiry

import (
	"context"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// Queue ends the wait of the in-flight downlinks whose lifetime has elapsed:
// it expires those no base station holds, moves those a station holds queued
// to revoking, naming that station, and claims the revoking ones a connected
// station left unanswered for at least an interval. askedAt, recorded as the
// ask, is the start of the sweep that asks.
type Queue interface {
	ExpireOverdueUnheld(ctx context.Context, limit int) ([]*storage.DownlinkMessage, error)
	RevokeOverdueHeld(ctx context.Context, askedAt time.Time, limit int) ([]*storage.DownlinkMessage, error)
	ClaimUnansweredRevocations(ctx context.Context, connected []uint64, askedAt time.Time, unansweredFor time.Duration, limit int) ([]*storage.DownlinkMessage, error)
}

// RemovedStations expires the downlinks held by base stations that no longer
// exist and returns each with the station that held it.
type RemovedStations interface {
	ExpireHeldAtRemovedStations(ctx context.Context, limit int) ([]*storage.DownlinkMessage, error)
}

// ExpiryReporter tells an expired downlink's originators it expired: the
// Application Center that queued it (SCACI §3.12), the MQTT downlink_result
// topic of its organization, and the owner tenant's events, which name the
// station that held it when one did.
type ExpiryReporter interface {
	ReportExpiredInQueue(ctx context.Context, downlink *storage.DownlinkMessage) error
	ReportExpiredAtStation(ctx context.Context, downlink *storage.DownlinkMessage)
}

// StationRevoker asks the base station holding a downlink whose lifetime
// ended to drop it (BSSCI §3.13), and names the stations connected now.
type StationRevoker interface {
	RevokeHeldDownlink(ctx context.Context, downlink *storage.DownlinkMessage) error
	ConnectedStations() []uint64
}

// QueueTenants forgets the owner cached for a queue id once its downlink is final.
type QueueTenants interface {
	UnregisterQueueTenant(queueID int64)
}

// Dependencies are the worker's collaborators.
type Dependencies struct {
	Queue        Queue
	Removed      RemovedStations
	Reporter     ExpiryReporter
	Revoker      StationRevoker
	QueueTenants QueueTenants
	Logger       logger.Logger
}

// Config bounds how often the queue is swept and how many rows one statement moves.
type Config struct {
	Interval  time.Duration
	BatchSize int
}

// Worker is the expiry sweep loop.
type Worker struct {
	deps Dependencies
	cfg  Config
}

// NewWorker wires the sweep and rejects a missing collaborator or an unusable configuration.
func NewWorker(deps Dependencies, cfg Config) (*Worker, error) {
	if deps.Queue == nil || deps.Removed == nil || deps.Reporter == nil || deps.Revoker == nil ||
		deps.QueueTenants == nil || deps.Logger == nil {
		return nil, errMissingDependency
	}
	if cfg.Interval <= 0 || cfg.BatchSize <= 0 {
		return nil, fmt.Errorf(errFmtInvalidConfig, errInvalidConfig, cfg.Interval, cfg.BatchSize)
	}
	return &Worker{deps: deps, cfg: cfg}, nil
}

// Start runs the sweep in the background; the returned stop waits for the sweep in flight.
func (w *Worker) Start(parent context.Context) (stop func()) {
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.run(ctx)
	}()
	return func() {
		cancel()
		<-done
	}
}

func (w *Worker) run(ctx context.Context) {
	started := time.Now()
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case tick := <-ticker.C:
			// A started sweep handles every row it moved, so a stop never leaves one unreported.
			w.SweepOnce(context.WithoutCancel(ctx), w.scheduledStart(started, tick))
		}
	}
}

// scheduledStart places a tick on the grid of whole intervals from started:
// a tick fires slightly late, and an ask recorded at a late start would not
// be due a whole interval later, at the next sweep.
func (w *Worker) scheduledStart(started, tick time.Time) time.Time {
	return started.Add(tick.Sub(started).Round(w.cfg.Interval))
}

// SweepOnce runs the sweep that started at startedAt: it expires the overdue
// downlinks no station holds and those held by a station that no longer
// exists and reports each, asks the station holding each overdue queued
// downlink to drop it, and asks again each connected station that left a
// revoke unanswered for an interval: a send that failed or an answer that
// proved nothing.
func (w *Worker) SweepOnce(ctx context.Context, startedAt time.Time) {
	w.sweep(ctx, w.deps.Queue.ExpireOverdueUnheld, w.report, LogDownlinksExpired)
	w.sweep(ctx, w.deps.Removed.ExpireHeldAtRemovedStations, w.reportAtStation, LogRemovedStationDownlinksExpired)
	w.sweep(ctx, w.overdueHeld(startedAt), w.revoke, LogDownlinksRevoking)
	w.sweep(ctx, w.unansweredRevocations(startedAt), w.revoke, LogRevocationsAskedAgain)
}

// overdueHeld moves the overdue queued downlinks to revoking, asked at startedAt.
func (w *Worker) overdueHeld(startedAt time.Time) func(context.Context, int) ([]*storage.DownlinkMessage, error) {
	return func(ctx context.Context, limit int) ([]*storage.DownlinkMessage, error) {
		return w.deps.Queue.RevokeOverdueHeld(ctx, startedAt, limit)
	}
}

// unansweredRevocations claims, as asked at startedAt, the revoking downlinks
// whose connected holder was last asked at least a sweep interval before.
func (w *Worker) unansweredRevocations(startedAt time.Time) func(context.Context, int) ([]*storage.DownlinkMessage, error) {
	return func(ctx context.Context, limit int) ([]*storage.DownlinkMessage, error) {
		connected := w.deps.Revoker.ConnectedStations()
		if len(connected) == 0 {
			return nil, nil
		}
		return w.deps.Queue.ClaimUnansweredRevocations(ctx, connected, startedAt, w.cfg.Interval, limit)
	}
}

// sweep moves overdue downlinks batch by batch and hands each to handle.
func (w *Worker) sweep(ctx context.Context, move func(context.Context, int) ([]*storage.DownlinkMessage, error),
	handle func(context.Context, *storage.DownlinkMessage), moved string,
) {
	for {
		batch, err := move(ctx, w.cfg.BatchSize)
		if err != nil {
			w.deps.Logger.ErrorContext(ctx, LogDownlinkExpirySweepFailed, logger.FieldError, err)
			return
		}
		for _, downlink := range batch {
			handle(ctx, downlink)
		}
		if len(batch) > 0 {
			w.deps.Logger.InfoContext(ctx, moved, logger.FieldCount, len(batch))
		}
		if len(batch) < w.cfg.BatchSize {
			return
		}
	}
}

// report forgets the queue owner of the expired downlink and has it reported.
func (w *Worker) report(ctx context.Context, downlink *storage.DownlinkMessage) {
	w.deps.QueueTenants.UnregisterQueueTenant(downlink.QueID)
	if err := w.deps.Reporter.ReportExpiredInQueue(ctx, downlink); err != nil {
		w.deps.Logger.ErrorContext(ctx, LogExpiredDownlinkUnidentified,
			logger.FieldQueID, downlink.QueID, logger.FieldError, err)
	}
}

// reportAtStation forgets the queue owner of a downlink that ended expired at
// the station holding it and has it reported.
func (w *Worker) reportAtStation(ctx context.Context, downlink *storage.DownlinkMessage) {
	w.deps.QueueTenants.UnregisterQueueTenant(downlink.QueID)
	w.deps.Reporter.ReportExpiredAtStation(ctx, downlink)
}

// revoke asks the station holding the downlink to drop it (BSSCI §3.13). A
// station that cannot be reached is asked again when it connects; the
// downlink stays revoking meanwhile.
func (w *Worker) revoke(ctx context.Context, downlink *storage.DownlinkMessage) {
	if err := w.deps.Revoker.RevokeHeldDownlink(ctx, downlink); err != nil {
		w.deps.Logger.WarnContext(ctx, LogOverdueDownlinkNotRevoked,
			logger.FieldQueID, downlink.QueID, logger.FieldBsEui, downlink.BsEui, logger.FieldError, err)
	}
}
