// Package certcleanup removes the generated certificate bundles whose
// download window has passed, on the configured cleanup interval.
package certcleanup

import (
	"context"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// BundleSweeper removes the certificate bundles whose download window had
// passed at now.
type BundleSweeper interface {
	CleanupExpiredCertificates(ctx context.Context, now time.Time)
}

// Worker is the bundle cleanup loop.
type Worker struct {
	sweeper  BundleSweeper
	clock    clock.Clock
	interval time.Duration
}

// NewWorker wires the cleanup loop and rejects a missing collaborator or a
// non-positive interval.
func NewWorker(sweeper BundleSweeper, clk clock.Clock, interval time.Duration) (*Worker, error) {
	if sweeper == nil || clk == nil {
		return nil, errMissingDependency
	}
	if interval <= 0 {
		return nil, fmt.Errorf(errFmtInvalidInterval, errInvalidInterval, interval)
	}
	return &Worker{sweeper: sweeper, clock: clk, interval: interval}, nil
}

// Start sweeps every interval in the background until parent ends or the
// returned stop runs; stop waits for the loop to exit.
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
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.sweeper.CleanupExpiredCertificates(ctx, w.clock.Now())
		}
	}
}
