package bssci

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// dlrxExpiryWorker expires DL RX status queries that never received a
// response (BSSCI §5.16) on a fixed interval.
type dlrxExpiryWorker struct {
	store    DLRXStatusStore
	timeout  time.Duration
	interval time.Duration
	clock    clock.Clock
	log      logger.Logger
}

func (s *Server) newDLRXExpiryWorker() *dlrxExpiryWorker {
	return &dlrxExpiryWorker{
		store:    s.dlrxStore,
		timeout:  s.config.dlrxQueryTimeout(),
		interval: s.config.dlrxCleanupInterval(),
		clock:    s.clock,
		log:      s.logger,
	}
}

// startDLRXQueryExpiryWorker runs the sweep until the server context ends.
func (s *Server) startDLRXQueryExpiryWorker() {
	if s.dlrxStore == nil {
		return
	}
	worker := s.newDLRXExpiryWorker()
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		worker.run(s.ctx)
	}()
}

func (w *dlrxExpiryWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.sweep(ctx, w.clock.Now())
		}
	}
}

func (w *dlrxExpiryWorker) sweep(ctx context.Context, now time.Time) {
	cutoff := now.Add(-w.timeout)
	expired, err := w.store.ExpireDLRXStatusQuery(ctx, cutoff)
	if err != nil {
		w.log.WarnContext(ctx, LogBSSCIDLRXQueryExpirySweepFailed, logger.FieldError, err)
		return
	}
	if expired > 0 {
		w.log.InfoContext(ctx, LogBSSCIDLRXQueriesExpired, logger.FieldCount, expired)
	}
}
