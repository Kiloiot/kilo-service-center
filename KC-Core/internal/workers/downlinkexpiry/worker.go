// Package downlinkexpiry ends the wait of the downlinks that outlived their
// lifetime before a base station transmitted them: one no station holds is
// expired and reported so, and the station holding one is asked to drop it,
// which reports it expired once that station answers. A connected station
// that has not answered is asked again once per sweep interval; nothing here
// ends a downlink a station holds just because time passed.
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
// station left unanswered for at least an interval.
type Queue interface {
	ExpireOverdueUnheld(ctx context.Context, limit int) ([]*storage.DownlinkMessage, error)
	RevokeOverdueHeld(ctx context.Context, limit int) ([]*storage.DownlinkMessage, error)
	ClaimUnansweredRevocations(ctx context.Context, connected []uint64, unansweredFor time.Duration, limit int) ([]*storage.DownlinkMessage, error)
}

// ExpiryReporter tells an expired downlink's originators it expired: the
// Application Center that queued it (SCACI §3.12), the MQTT downlink_result
// topic of its organization, and the owner tenant's events.
type ExpiryReporter interface {
	ReportExpiredInQueue(ctx context.Context, downlink *storage.DownlinkMessage) error
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
	if deps.Queue == nil || deps.Reporter == nil || deps.Revoker == nil ||
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
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// A started sweep handles every row it moved, so a stop never leaves one unreported.
			w.SweepOnce(context.WithoutCancel(ctx))
		}
	}
}

// SweepOnce expires the overdue downlinks no station holds and reports each,
// asks the station holding each overdue queued downlink to drop it, and asks
// again each connected station that left a revoke unanswered for an interval:
// a send that failed or an answer that proved nothing.
func (w *Worker) SweepOnce(ctx context.Context) {
	w.sweep(ctx, w.deps.Queue.ExpireOverdueUnheld, w.report, LogDownlinksExpired)
	w.sweep(ctx, w.deps.Queue.RevokeOverdueHeld, w.revoke, LogDownlinksRevoking)
	w.sweep(ctx, w.unansweredRevocations, w.revoke, LogRevocationsAskedAgain)
}

// unansweredRevocations claims the revoking downlinks whose connected holder
// was last asked at least a sweep interval ago.
func (w *Worker) unansweredRevocations(ctx context.Context, limit int) ([]*storage.DownlinkMessage, error) {
	connected := w.deps.Revoker.ConnectedStations()
	if len(connected) == 0 {
		return nil, nil
	}
	return w.deps.Queue.ClaimUnansweredRevocations(ctx, connected, w.cfg.Interval, limit)
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

// revoke asks the station holding the downlink to drop it (BSSCI §3.13). A
// station that cannot be reached is asked again when it connects; the
// downlink stays revoking meanwhile.
func (w *Worker) revoke(ctx context.Context, downlink *storage.DownlinkMessage) {
	if err := w.deps.Revoker.RevokeHeldDownlink(ctx, downlink); err != nil {
		w.deps.Logger.WarnContext(ctx, LogOverdueDownlinkNotRevoked,
			logger.FieldQueID, downlink.QueID, logger.FieldBsEui, downlink.BsEui, logger.FieldError, err)
	}
}
