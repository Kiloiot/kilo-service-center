package sessionreconcile

import "errors"

var (
	// ErrNilAbandonedSessionStore rejects a session reconciler built without a session store.
	ErrNilAbandonedSessionStore = errors.New("session reconciler: session store is nil")
	// ErrNilReconcilerLogger rejects a session reconciler built without a logger.
	ErrNilReconcilerLogger = errors.New("session reconciler: logger is nil")
	// errReconcileAbandonedSessions reports sessions left live by a previous process that could not be made resumable.
	errReconcileAbandonedSessions = errors.New("reconcile abandoned sessions")
)
