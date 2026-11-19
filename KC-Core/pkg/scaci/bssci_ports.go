package scaci

import "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"

// DetachPropagator allows SCACI to trigger BSSCI detach propagation without import cycles
//
// This interface is satisfied by the BSSCI Server and passed to SCACI during initialization.
// When an Application Center deregisters an endpoint, SCACI uses this interface to notify
// all connected base stations so they can clear their local endpoint state.
type DetachPropagator interface {
	// SendDetachPropagateToAll sends detach propagate to all connected base stations
	// Returns a slice of errors (one per failed base station), empty if all succeeded
	SendDetachPropagateToAll(epEUI uint64) []error
}

// Scheduler interface aliases for backward compatibility
// The actual interfaces are defined in pkg/scheduler to prevent circular dependencies
type (
	// ULTransmitScheduler is imported from pkg/scheduler
	ULTransmitScheduler = scheduler.ULTransmitScheduler

	// DownlinkScheduler is imported from pkg/scheduler
	DownlinkScheduler = scheduler.DownlinkScheduler
)
