package scacimonitoring

import "errors"

// Domain sentinels for SCACI monitoring failures.
var (
	// ErrSessionNotFound reports a session ID with no matching SCACI session.
	ErrSessionNotFound = errors.New("session not found")
	// ErrInvalidSessionID reports a session ID that failed parsing.
	ErrInvalidSessionID = errors.New("invalid session ID")
	// ErrInvalidSessionStatus reports a status filter outside the SCACI session states.
	ErrInvalidSessionStatus = errors.New("invalid session status")
	// ErrInvalidTimeRange reports a window whose start lies after its end.
	ErrInvalidTimeRange = errors.New("invalid time range")
)

// Error-wrapping context prefixes describing the failed repository operation.
const (
	errCtxListSessions         = "list sessions"
	errCtxCountOperations      = "count session operations"
	errCtxGetSession           = "get session"
	errCtxGetSessionStatistics = "get session statistics"
	errCtxGetOperationSummary  = "get operation summary"
	errCtxListErrors           = "list failed operations"
	errCtxListQueue            = "list queue"
	errCtxCountQueue           = "count queue"
	errCtxPingSummary          = "get ping summary"
	errCtxCountReconnects      = "count reconnects"
	errCtxLatestConnect        = "get latest connect"
)
