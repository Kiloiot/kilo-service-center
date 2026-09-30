// Package downlinkexpiry expires the downlinks that outlived their lifetime
// before a base station transmitted them, has each one reported as expired to
// its originators, and asks the base station still holding one to drop it.
package downlinkexpiry

import (
	"context"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// Queue expires the in-flight downlinks whose lifetime has elapsed, naming
// the base station that held each one, if any.
type Queue interface {
	ExpireOverdueDownlinks(ctx context.Context, limit int) ([]*storage.DownlinkMessage, error)
}

// ExpiryReporter tells an expired downlink's originators it expired: the
// Application Center that queued it (SCACI §3.12), the MQTT downlink_result
// topic of its organization, and the owner tenant's events.
type ExpiryReporter interface {
	ReportExpiredInQueue(ctx context.Context, downlink *storage.DownlinkMessage) error
}

// StationRevoker asks the base station holding an expired downlink to drop it
// (BSSCI §3.13).
type StationRevoker interface {
	RevokeHeldDownlink(ctx context.Context, downlink *storage.DownlinkMessage) error
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

// Config bounds how often the queue is swept and how many rows one statement expires.
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
			// A started sweep reports every row it expired, so a stop never leaves one unreported.
			w.SweepOnce(context.WithoutCancel(ctx))
		}
	}
}

// SweepOnce expires overdue downlinks batch by batch and reports each of them.
func (w *Worker) SweepOnce(ctx context.Context) {
	for {
		expired, err := w.deps.Queue.ExpireOverdueDownlinks(ctx, w.cfg.BatchSize)
		if err != nil {
			w.deps.Logger.ErrorContext(ctx, LogDownlinkExpirySweepFailed, logger.FieldError, err)
			return
		}
		for _, downlink := range expired {
			w.report(ctx, downlink)
		}
		if len(expired) > 0 {
			w.deps.Logger.InfoContext(ctx, LogDownlinksExpired, logger.FieldCount, len(expired))
		}
		if len(expired) < w.cfg.BatchSize {
			return
		}
	}
}

// report forgets the queue owner of the expired downlink, has it reported
// and asks the base station still holding it to drop it (BSSCI §3.13).
func (w *Worker) report(ctx context.Context, downlink *storage.DownlinkMessage) {
	w.deps.QueueTenants.UnregisterQueueTenant(downlink.QueID)
	if err := w.deps.Reporter.ReportExpiredInQueue(ctx, downlink); err != nil {
		w.deps.Logger.ErrorContext(ctx, LogExpiredDownlinkUnidentified,
			logger.FieldQueID, downlink.QueID, logger.FieldError, err)
	}
	if downlink.BsEui == 0 {
		return
	}
	if err := w.deps.Revoker.RevokeHeldDownlink(ctx, downlink); err != nil {
		w.deps.Logger.WarnContext(ctx, LogExpiredDownlinkNotRevoked,
			logger.FieldQueID, downlink.QueID, logger.FieldBsEui, downlink.BsEui, logger.FieldError, err)
	}
}
