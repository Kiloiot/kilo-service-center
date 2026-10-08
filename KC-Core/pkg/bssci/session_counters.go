package bssci

import (
	"sync"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// counterFlush coalesces the persistence of a session's operation counters:
// every frame marks them dirty, and at most one flusher per session writes
// them, reading the newest values each round until none are pending.
type counterFlush struct {
	mu      sync.Mutex
	dirty   bool
	running bool
}

// request marks the counters dirty and reports whether the caller must start
// the flusher because none is running.
func (f *counterFlush) request() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dirty = true
	if f.running {
		return false
	}
	f.running = true
	return true
}

// next takes the dirty mark and reports whether a write is due; when none is,
// the flusher stops.
func (f *counterFlush) next() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.dirty {
		f.running = false
		return false
	}
	f.dirty = false
	return true
}

// persistSessionCounters persists the session's operation counters after a
// frame advanced them: a running flusher picks up the newest values, so a
// burst of frames keeps one write in flight instead of one per frame.
func (s *Server) persistSessionCounters(session *Session) {
	if !session.counterFlush.request() {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for session.counterFlush.next() {
			s.updateSessionCounters(session)
		}
	}()
}

// updateSessionCounters writes the session's operation counters; the frame
// path keeps its debug-only error handling, the teardown persists the final
// counters itself.
func (s *Server) updateSessionCounters(session *Session) {
	if err := s.sessionSvc.UpdateSessionCounters(s.safeCtx(), session); err != nil {
		lastBsOpID, lastScOpID := session.OperationCounters()
		s.logger.DebugContext(s.safeCtx(), LogBSSCIFailedToUpdateSessionCounters,
			logger.FieldError, err,
			logger.FieldBsOpID, lastBsOpID,
			logger.FieldScOpID, lastScOpID)
	}
}
