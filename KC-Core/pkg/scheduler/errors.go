// Package scheduler provides neutral contracts for SCACI<->BSSCI scheduling operations.
//
// This package exists to prevent circular dependencies between SCACI and BSSCI protocol
// layers while maintaining proper Dependency Inversion. Both SCACI handlers
// and BSSCI adapters import this shared package for:
//   - Scheduler interface definitions (ULTransmitScheduler, DownlinkScheduler)
//   - Sentinel errors for error classification
//
// By placing these contracts in a neutral package, we ensure that neither protocol
// layer depends on the other's internals, preserving modularity and testability.
package scheduler

import "errors"

// Sentinel errors for scheduler operations
//
// These errors are returned by scheduler implementations (BSSCI) to signal specific
// failure conditions to callers (SCACI). The errors enable proper classification
// without creating dependencies between protocol layers.
//
// SCACI handlers use errors.Is() to detect these conditions and map them to
// appropriate POSIX codes for the wire protocol.
var (
	// ErrSchedulerNoResources indicates no bidirectional base stations are available
	// for the requested operation. This is a temporary condition - the caller
	// should retry later when resources become available.
	//
	// SCACI mapping: POSIX_EAGAIN (11) per §3.9, §3.10
	// Retry semantics: Temporary, AC should retry
	ErrSchedulerNoResources = errors.New("no bidirectional sessions available")

	// ErrSchedulerResourceMissing indicates a specific requested resource (base station
	// or endpoint) does not exist or is unavailable. This is a permanent condition for
	// the current request parameters.
	//
	// SCACI mapping: POSIX_ENOENT (2) per §3.9, §3.10
	// Retry semantics: Permanent for current parameters
	ErrSchedulerResourceMissing = errors.New("requested resource unavailable")

	// ErrSchedulerQueueNotFound indicates the requested downlink queue entry does not
	// exist or was already removed. Used during DL Data Revoke operations (SCACI §3.11).
	//
	// SCACI mapping: POSIX_ENOENT (2) per §3.11
	// Retry semantics: Permanent, queue already processed or never existed
	ErrSchedulerQueueNotFound = errors.New("queue entry not found")
)
