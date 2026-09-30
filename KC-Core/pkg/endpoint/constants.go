package endpoint

// PropagateStatus constants for BSSCI attach/detach propagate lifecycle.
// These constants are defined in a shared location to avoid import cycles
// between the bssci and endpoint packages.
const (
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
	EndpointStatusAttached  = PropagateStatusAttached // Reuse "attached" value
	EndpointStatusDetached  = PropagateStatusDetached // Reuse "detached" value
	EndpointStatusAttaching = "attaching"
)

// IsAttachmentDecision reports whether status is one the service center
// decides for an endpoint: attached or detached.
func IsAttachmentDecision(status string) bool {
	return status == EndpointStatusAttached || status == EndpointStatusDetached
}
