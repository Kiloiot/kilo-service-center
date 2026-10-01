package scaciservices

import "errors"

// ErrNilDownlinkScheduler rejects a DL service built without the BSSCI scheduler.
var ErrNilDownlinkScheduler = errors.New("dl service: downlink scheduler is nil")

// ErrNilDownlinkStore rejects a DL service built without the downlink queue.
var ErrNilDownlinkStore = errors.New("dl service: downlink queue is nil")

// ErrNilEnqueueRecorder rejects a DL service built without the recorder of queued downlinks.
var ErrNilEnqueueRecorder = errors.New("dl service: enqueue recorder is nil")

// ErrNilDLServiceLogger rejects a DL service built without a logger.
var ErrNilDLServiceLogger = errors.New("dl service: logger is nil")

// ErrNilQueueIDAllocator rejects a DL service built without the allocator of
// the service center queue ids.
var ErrNilQueueIDAllocator = errors.New("downlink queue id allocator is nil")

// ErrNonPositiveDownlinkLifetime rejects a DL service whose downlinks would expire on arrival.
var ErrNonPositiveDownlinkLifetime = errors.New("downlink lifetime must be positive")

// Transaction lifecycle sentinels of a session creation. The composition
// root maps the storage adapter's lifecycle failures onto these.
var (
	ErrTxBegin  = errors.New("session creation transaction begin")
	ErrTxCommit = errors.New("session creation transaction commit")
)

// errMissingSessionPersistenceDependency refuses a session row owner built
// without one of its stores.
var errMissingSessionPersistenceDependency = errors.New("session rows: missing dependency")

// Collaborators an endpoint service is refused without.
var (
	errNilEndpointStore         = errors.New("endpoint service: endpoint store is nil")
	errNilDetachPropagator      = errors.New("endpoint service: detach propagator is nil")
	errNilAttachmentDecider     = errors.New("endpoint service: attachment decider is nil")
	errNilEndpointServiceLogger = errors.New("endpoint service: logger is nil")
)

// errDetachPropagationEndpointLookup refuses a detach propagation for an EUI
// the deregistering tenant's endpoint lookup did not return.
var errDetachPropagationEndpointLookup = errors.New("detach propagation: the tenant's endpoint lookup failed")

// Failures of the holder of resumable sessions without a connection.
var (
	errMissingResumeHolderDependency = errors.New("resume holder: missing dependency")
	errInvalidResumeLimit            = errors.New("resume holder: invalid limit")
	errLoadHeldSessions              = errors.New("load the SCACI sessions held for resumption")
	errCountHeldOperations           = errors.New("count the operations held for a SCACI session")
	errPersistHeldOpIDs              = errors.New("persist the operation IDs of a held SCACI session")
	errRecordHeldOperation           = errors.New("record an operation for a held SCACI session")
	errEndResumability               = errors.New("end the resumability of a held SCACI session")
)

// Error formats of the resume holder.
const (
	errFmtResumeLimitNotPositive = "%w: the operations a session holds must be positive, got %d"
	errFmtHoldOperation          = "hold %s for session %d: %w"
)
