package scaci

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// SessionRegistry owns which connection serves which SCACI session and, through
// the holder, the resumable sessions that have none (SCACI §1). A session moves
// between connected and held only under its write lock. The row writes of a
// session's resume and of the loss of its connection run under the lifecycle
// lock of its ID, in the order of those moves, so the stored session is never
// left active without a connection.
type SessionRegistry struct {
	mu       sync.RWMutex
	sessions map[net.Conn]*Session
	holder   ResumeHolder
	rows     SessionLifecycleStore
	logger   logger.Logger
	// lifecycles serializes the lifecycle moves of each session ID.
	lifecycles sessionLocks
}

// NewSessionRegistry builds the registry over the holder of the resumable
// sessions and the store of their lifecycle rows.
func NewSessionRegistry(holder ResumeHolder, rows SessionLifecycleStore, log logger.Logger) (*SessionRegistry, error) {
	if holder == nil || rows == nil || log == nil {
		return nil, errMissingRegistryDependency
	}
	return &SessionRegistry{sessions: make(map[net.Conn]*Session), holder: holder, rows: rows, logger: log}, nil
}

// sessionTarget is one connection whose session receives an SC operation.
type sessionTarget struct {
	conn    net.Conn
	session *Session
}

// adopt maps the connection to its session. One session per application
// center is active (SCACI §1), so every other connection holding this session
// or an earlier session of the same application center is evicted and closed,
// and what was held for a session it supersedes is discarded. A resume
// continues the session it resumes (§3.2). Every adopted session is held
// until its connect operation completes, so what the service center produces
// meanwhile is reissued to it (§1, §3.3); adopt reports false for a resume of
// a session the registry neither serves nor holds. It returns the other
// sessions it superseded, each with the connection it closed.
func (r *SessionRegistry) adopt(ctx context.Context, conn net.Conn, session *Session) (bool, []sessionTarget) {
	r.mu.Lock()
	prev, superseded := r.evictLocked(conn, session)
	if held, ok := r.holder.Release(ctx, session); ok {
		prev = held
	}
	continues := !session.Resumed || prev != nil
	if prev != nil {
		session.continueFrom(prev)
	}
	if continues {
		r.sessions[conn] = session
		r.holder.Hold(ctx, session)
	}
	r.mu.Unlock()

	var replaced []sessionTarget
	for _, other := range superseded {
		r.logger.InfoContext(ctx, LogSCACISupersededConnectionClosed, logger.FieldRemote, other.conn.RemoteAddr())
		if err := other.conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			r.logger.WarnContext(ctx, LogSCACICloseConnectionFailed, logger.FieldRemote, other.conn.RemoteAddr(), logger.FieldError, err)
		}
		if other.session.ID != session.ID {
			replaced = append(replaced, other)
		}
	}
	return continues, replaced
}

// evictLocked unmaps every other connection whose session next supersedes
// and returns them with the connected session next resumes, if any. Callers
// hold mu.
func (r *SessionRegistry) evictLocked(conn net.Conn, next *Session) (*Session, []sessionTarget) {
	var resumed *Session
	var superseded []sessionTarget
	for other, live := range r.sessions {
		if other == conn || !live.SupersededBy(next) {
			continue
		}
		delete(r.sessions, other)
		superseded = append(superseded, sessionTarget{conn: other, session: live})
		if live.ID > 0 && live.ID == next.ID {
			resumed = live
		}
	}
	return resumed, superseded
}

// complete makes the session live once its connect operation completes on
// conn (SCACI §3.3): it is no longer held, and it reissues what it held before
// it is active. A resumed session's row records the connection
// that resumed it, with its TLS evidence (§1). complete refuses a session a
// newer session took conn over from before its connect completed, and a
// resume whose session is no longer held for resumption.
func (r *SessionRegistry) complete(ctx context.Context, conn net.Conn, session *Session, now time.Time, tlsVersion, cipherSuite string) error {
	defer r.lifecycles.lock(session.ID)()
	if err := r.goLive(ctx, conn, session, now); err != nil {
		return err
	}
	if !session.Resumed {
		return nil
	}
	if err := r.rows.PersistResume(ctx, session, tlsVersion, cipherSuite); err != nil {
		r.logger.ErrorContext(ctx, LogSCACIUpdateSessionFailed, logger.FieldError, err)
	}
	return nil
}

// goLive moves the session from held to connected. It runs under mu, as the
// eviction of a superseded session does, so a session is either seen live by
// the session that supersedes it or never goes live.
func (r *SessionRegistry) goLive(ctx context.Context, conn net.Conn, session *Session, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sessions[conn] != session {
		return errSessionSuperseded
	}
	if _, held := r.holder.Release(ctx, session); session.Resumed && !held {
		return errResumedSessionNotHeld
	}
	session.setState(StateReissuing)
	session.UpdateLastSeen(now)
	return nil
}

// release drops the connection's session mapping. A session the connection
// still owns lost its connection and stays resumable (SCACI §1): it is held,
// so its operations are recorded until it is resumed, and its row records the
// loss. One that a newer connection took over is left to its new owner.
// release returns the session that lost its connection, nil for none.
func (r *SessionRegistry) release(ctx context.Context, conn net.Conn, timeout time.Duration) *Session {
	defer r.lifecycles.lock(r.sessionIDOf(conn))()
	session, resumable := r.loseConnection(ctx, conn)
	if !resumable {
		return nil
	}
	writeCtx, cancel := context.WithTimeout(withSessionValues(ctx, session), timeout)
	defer cancel()
	if err := r.rows.PersistDisconnect(writeCtx, session); err != nil {
		r.logger.ErrorContext(writeCtx, LogSCACIMarkSessionDisconnectedFailed,
			logger.FieldSessionID, session.ID, logger.FieldError, err)
	}
	return session
}

// sessionIDOf is the ID of the session the connection serves, 0 for none.
func (r *SessionRegistry) sessionIDOf(conn net.Conn) int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if session, ok := r.sessions[conn]; ok {
		return session.ID
	}
	return 0
}

// loseConnection unmaps the connection and holds the session it still owned,
// reporting whether that session stays resumable.
func (r *SessionRegistry) loseConnection(ctx context.Context, conn net.Conn) (*Session, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	session, owned := r.sessions[conn]
	delete(r.sessions, conn)
	resumable := owned && session.ID > 0
	if resumable {
		r.holder.Hold(withSessionValues(ctx, session), session)
	}
	return session, resumable
}

// targets returns the connected sessions of the tenant that reaches selects
// and has the holder record the operation for the held ones, under one read
// lock: a session moves between connected and held only under the write lock,
// so none misses the operation.
func (r *SessionRegistry) targets(ctx context.Context, tenantID int64, reaches func(*Session) bool, command string, record OperationRecord) ([]sessionTarget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	connected := r.connectedLocked(func(session *Session) bool {
		return session.TenantID == tenantID && reaches(session)
	})
	return connected, r.holder.Record(ctx, tenantID, reaches, command, record)
}

// connectedWhere returns the sessions that completed their connect operation
// (SCACI §3.3) and that keep selects.
func (r *SessionRegistry) connectedWhere(keep func(*Session) bool) []sessionTarget {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.connectedLocked(keep)
}

// connectedLocked is connectedWhere for callers that hold mu.
func (r *SessionRegistry) connectedLocked(keep func(*Session) bool) []sessionTarget {
	var connected []sessionTarget
	for conn, session := range r.sessions {
		if session.connected() && keep(session) {
			connected = append(connected, sessionTarget{conn: conn, session: session})
		}
	}
	return connected
}

// sessionLocks serializes the lifecycle moves of each session ID; an ID's lock
// exists only while someone uses it.
type sessionLocks struct {
	mu    sync.Mutex
	locks map[int64]*sessionLock
}

type sessionLock struct {
	sync.Mutex
	users int
}

// lock locks the lifecycle of the session ID and returns its unlock.
func (l *sessionLocks) lock(id int64) func() {
	entry := l.acquire(id)
	entry.Lock()
	return func() {
		entry.Unlock()
		l.releaseEntry(id, entry)
	}
}

func (l *sessionLocks) acquire(id int64) *sessionLock {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.locks == nil {
		l.locks = make(map[int64]*sessionLock)
	}
	entry, ok := l.locks[id]
	if !ok {
		entry = &sessionLock{}
		l.locks[id] = entry
	}
	entry.users++
	return entry
}

func (l *sessionLocks) releaseEntry(id int64, entry *sessionLock) {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry.users--
	if entry.users == 0 {
		delete(l.locks, id)
	}
}
