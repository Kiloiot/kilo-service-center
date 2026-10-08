package downlinks

import "errors"

// Content sentinels: the SCACI §3.10.1 rule a downlink's content breaks.
var (
	// ErrPayloadRequired refuses a counter-dependent downlink without payloads.
	ErrPayloadRequired = errors.New("counter-dependent downlink carries no payload")
	// ErrPacketCountersUnpaired refuses counter-dependent payloads that do not pair one to one with packet counters.
	ErrPacketCountersUnpaired = errors.New("payloads and packet counters do not pair")
	// ErrTooManyPayloads refuses a counter-independent downlink with more than one payload.
	ErrTooManyPayloads = errors.New("counter-independent downlink carries more than one payload")
	// ErrPayloadTooLarge refuses a payload over the radio limit.
	ErrPayloadTooLarge = errors.New("downlink payload exceeds the radio limit")
	// ErrPriorityInvalid refuses a priority that is not a finite single-precision number.
	ErrPriorityInvalid = errors.New("downlink priority is not finite")
	// ErrFormatInvalid refuses a format that does not fit 8 bits.
	ErrFormatInvalid = errors.New("downlink format exceeds 8 bits")
	// ErrPacketCounterInvalid refuses a packet counter that does not fit 32 bits.
	ErrPacketCounterInvalid = errors.New("packet counter exceeds 32 bits")
)

// Request sentinels.
var (
	// ErrMissingDependency refuses to build the service without one of its collaborators.
	ErrMissingDependency = errors.New("downlink service dependency is nil")
	// ErrEndpointNotFound refuses a revoke for an endpoint the tenant has not registered.
	ErrEndpointNotFound = errors.New("endpoint not found")
	// ErrEndpointLookup wraps a failure to look the endpoint up.
	ErrEndpointLookup = errors.New("look up endpoint")
	// ErrQueue wraps a failure of the SCACI handler core to queue a downlink.
	ErrQueue = errors.New("queue downlink")
	// ErrUpdate wraps a store failure while rewriting a pending downlink.
	ErrUpdate = errors.New("update pending downlink")
	// ErrRevoke wraps a failure of the revoke path.
	ErrRevoke = errors.New("revoke downlink")
)
