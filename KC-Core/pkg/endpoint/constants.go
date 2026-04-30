package endpoint

// PropagateStatus constants for BSSCI attach/detach propagate lifecycle.
// These constants are defined in a shared location to avoid import cycles
// between the bssci and endpoint packages.
const (
	// PropagateStatusAttachReceived indicates a BS-originated attach frame was received
	PropagateStatusAttachReceived = "attach_received"

	// PropagateStatusDetachReceived indicates a BS-originated detach frame was received
	PropagateStatusDetachReceived = "detach_received"

	// PropagateStatusAttached indicates attach propagate completed successfully
	PropagateStatusAttached = "attached"

	// PropagateStatusDetaching indicates detach propagate is in-flight (awaiting BS response)
	PropagateStatusDetaching = "detaching"

	// PropagateStatusDetached indicates detach propagate completed successfully
	PropagateStatusDetached = "detached"
)

// EndpointStatus constants for ep_status database column
// CHECK constraint (migration 003): 'attached', 'detached', 'attaching'
// Simplified 3-state model vs PropagateStatus 5-state flow
const (
	EndpointStatusAttaching = "attaching"             // Attach handshake complete, propagation pending
	EndpointStatusAttached  = PropagateStatusAttached // Reuse "attached" value
	EndpointStatusDetached  = PropagateStatusDetached // Reuse "detached" value
)

// Event Type Constants
// System event types for BSSCI operations
const (
	// EventBSSCIPropagationCompleted is emitted when automatic propagation succeeds
	EventBSSCIPropagationCompleted = "bssci.propagation.completed"
)
