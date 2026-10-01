package scaciservices

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// ResumeHolder holds the resumable SCACI sessions that have no connection and
// records, for each, the service center operations produced meanwhile, so its
// resume reissues them (SCACI §1). A session holds at most limit of them: the
// operation that exceeds it ends the session's resumability, so nothing it
// holds is reissued.
type ResumeHolder struct {
	rows       HeldSessionRows
	operations PendingOperationCounter
	limit      int
	logger     logger.Logger

	mu   sync.Mutex
	held map[int64]*heldSession
}

// heldSession is one held session; mu serializes what is recorded for it.
type heldSession struct {
	mu      sync.Mutex
	session *scaci.Session
	// pending counts the operations the session holds once counted is set.
	pending int
	counted bool
	// ended marks a session released or no longer resumable.
	ended bool
}

// NewResumeHolder builds the holder of resumable sessions; limit bounds the
// operations one session holds (protocol.scaci_resume_max_pending_operations).
func NewResumeHolder(rows HeldSessionRows, operations PendingOperationCounter, limit int, log logger.Logger) (*ResumeHolder, error) {
	if rows == nil || operations == nil || log == nil {
		return nil, errMissingResumeHolderDependency
	}
	if limit <= 0 {
		return nil, fmt.Errorf(errFmtResumeLimitNotPositive, errInvalidResumeLimit, limit)
	}
	return &ResumeHolder{
		rows:       rows,
		operations: operations,
		limit:      limit,
		logger:     log,
		held:       make(map[int64]*heldSession),
	}, nil
}

// Load holds every session an Application Center may still resume, so a
// restarted service center keeps recording for them (SCACI §1).
func (h *ResumeHolder) Load(ctx context.Context, sessions ResumableSessionLister) error {
	rows, err := sessions.ListResumableSessions(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", errLoadHeldSessions, err)
	}
	for _, row := range rows {
		h.Hold(ctx, scaci.RestoreSession(row))
	}
	return nil
}

// Hold records for a persisted session until it is released; a session
// already held keeps its entry.
func (h *ResumeHolder) Hold(ctx context.Context, session *scaci.Session) {
	if session.ID <= 0 || !h.hold(session) {
		return
	}
	h.logger.DebugContext(ctx, LogSessionHeldForResume,
		logger.FieldSessionID, session.ID, logger.FieldAcEui, mioty.FormatEUI64(session.AcEui))
}

// hold adds the session's entry and reports whether it was not held yet.
func (h *ResumeHolder) hold(session *scaci.Session) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, held := h.held[session.ID]; held {
		return false
	}
	h.held[session.ID] = &heldSession{session: session}
	return true
}

// Release stops holding every session next supersedes and returns the held
// session with next's ID, if any. It waits for what is being recorded for
// them, and nothing more is recorded for them afterwards.
func (h *ResumeHolder) Release(ctx context.Context, next *scaci.Session) (*scaci.Session, bool) {
	var resumed *scaci.Session
	for _, entry := range h.take(next) {
		entry.mu.Lock()
		entry.ended = true
		entry.mu.Unlock()
		if entry.session.ID == next.ID {
			resumed = entry.session
			continue
		}
		h.logger.InfoContext(ctx, LogHeldSessionDiscarded,
			logger.FieldSessionID, entry.session.ID, logger.FieldAcEui, mioty.FormatEUI64(entry.session.AcEui))
	}
	return resumed, resumed != nil
}

// take removes the entries of the sessions next supersedes.
func (h *ResumeHolder) take(next *scaci.Session) []*heldSession {
	h.mu.Lock()
	defer h.mu.Unlock()
	var taken []*heldSession
	for id, entry := range h.held {
		if entry.session.SupersededBy(next) {
			delete(h.held, id)
			taken = append(taken, entry)
		}
	}
	return taken
}

// Record records the service center operation command with record for every
// held session of the tenant that reaches selects, each under its next SC opId
// (SCACI §3.2).
func (h *ResumeHolder) Record(ctx context.Context, tenantID int64, reaches func(*scaci.Session) bool, command string, record scaci.OperationRecord) error {
	var failures []error
	for _, entry := range h.tenantEntries(tenantID, reaches) {
		if err := h.record(ctx, entry, record); err != nil {
			failures = append(failures, fmt.Errorf(errFmtHoldOperation, command, entry.session.ID, err))
		}
	}
	return errors.Join(failures...)
}

// tenantEntries returns the held sessions of the tenant that reaches selects.
func (h *ResumeHolder) tenantEntries(tenantID int64, reaches func(*scaci.Session) bool) []*heldSession {
	h.mu.Lock()
	defer h.mu.Unlock()
	var entries []*heldSession
	for _, entry := range h.held {
		if entry.session.TenantID == tenantID && reaches(entry.session) {
			entries = append(entries, entry)
		}
	}
	return entries
}

// record reserves the session's next SC opId, persists the counters, then
// records the operation. An operation the session already holds counts once;
// a new one past the limit ends the session's resumability.
func (h *ResumeHolder) record(ctx context.Context, entry *heldSession, record scaci.OperationRecord) error {
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.ended {
		return nil
	}
	recCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), dbconfig.DefaultQueryTimeout)
	defer cancel()
	held, err := h.heldOperations(recCtx, entry)
	if err != nil {
		return err
	}
	session := entry.session
	opId := session.NextScOpId()
	if err := h.rows.PersistOpIDs(recCtx, session, session.OpIDs()); err != nil {
		return fmt.Errorf("%w: %w", errPersistHeldOpIDs, err)
	}
	recording, err := record(session, opId)
	if err != nil {
		return fmt.Errorf("%w: %w", errRecordHeldOperation, err)
	}
	if recording != scaci.RecordedNew {
		return nil
	}
	if held >= h.limit {
		return h.end(recCtx, entry)
	}
	entry.pending++
	return nil
}

// heldOperations counts what the session holds, reading the operation log
// the first time.
func (h *ResumeHolder) heldOperations(ctx context.Context, entry *heldSession) (int, error) {
	if entry.counted {
		return entry.pending, nil
	}
	pending, err := h.operations.CountPendingSCOperations(ctx, entry.session.ID)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errCountHeldOperations, err)
	}
	entry.pending, entry.counted = pending, true
	return pending, nil
}

// end ends the resumability of a session that holds the limit: the
// Application Center starts a new session, and nothing it holds is reissued
// (SCACI §1).
func (h *ResumeHolder) end(ctx context.Context, entry *heldSession) error {
	session := entry.session
	if err := h.rows.EndResumability(ctx, session); err != nil {
		return fmt.Errorf("%w: %w", errEndResumability, err)
	}
	entry.ended = true
	h.mu.Lock()
	if h.held[session.ID] == entry {
		delete(h.held, session.ID)
	}
	h.mu.Unlock()
	h.logger.WarnContext(ctx, LogHeldSessionLimitReached,
		logger.FieldTenantID, session.TenantID,
		logger.FieldSessionID, session.ID,
		logger.FieldAcEui, mioty.FormatEUI64(session.AcEui),
		logger.FieldLimit, h.limit)
	return nil
}
