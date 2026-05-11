package bssci

import (
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
)

// Message Encoding Constants
// BSSCI Section 1 - Dual encoding support (JSON and MessagePack)
const (
	// EncodingMessagePack is the default MIOTY message encoding (binary)
	EncodingMessagePack = "msgpack"

	// EncodingJSON is the alternative MIOTY message encoding (text-based)
	EncodingJSON = "json"
)

// Propagate Status Constants
// BSSCI Sections 5.6-5.7, 5.9 - Endpoint telemetry propagation status values
// These are re-exported from the endpoint package to avoid import cycles
const (
	// PropagateStatusDetachReceived indicates endpoint has received detach message
	PropagateStatusDetachReceived = endpoint.PropagateStatusDetachReceived

	// PropagateStatusAttachReceived indicates endpoint has received attach propagate
	PropagateStatusAttachReceived = endpoint.PropagateStatusAttachReceived

	// PropagateStatusAttached indicates attach propagate completed successfully
	PropagateStatusAttached = endpoint.PropagateStatusAttached

	// PropagateStatusDetaching indicates detach propagate is in-flight (awaiting BS response)
	PropagateStatusDetaching = endpoint.PropagateStatusDetaching

	// PropagateStatusDetached indicates detach propagate completed successfully
	PropagateStatusDetached = endpoint.PropagateStatusDetached

	// EndpointStatusAttached is the ep_status value for attached endpoints
	EndpointStatusAttached = endpoint.EndpointStatusAttached
)

// Event Severity Constants
// System event severity levels for audit logging and event categorization
const (
	// SeverityInfo indicates an informational event
	SeverityInfo = "info"

	// SeverityWarning indicates a warning event
	SeverityWarning = "warning"

	// SeverityError indicates an error event
	SeverityError = "error"
)

// Event Status Constants
// System event lifecycle status values
const (
	// EventStatusNew indicates a newly created event not yet processed
	EventStatusNew = "new"
)

// Operation Constants
// Endpoint attachment operation tracking
const (
	// OperationStatusInitiated indicates an async operation has been started
	OperationStatusInitiated = "initiated"

	// OperationIDPrefixAttach is the prefix for attach operation IDs
	OperationIDPrefixAttach = "attach"

	// OperationIDPrefixDetach is the prefix for detach operation IDs
	OperationIDPrefixDetach = "detach"

	// OperationIDFormat is the format string for operation IDs
	// Use with fmt.Sprintf(OperationIDFormat, prefix, endpointID, timestamp)
	OperationIDFormat = "%s-%d-%d"

	// OperationTypeAttach identifies attach operations
	OperationTypeAttach = "attach"

	// OperationTypeDetach identifies detach operations
	OperationTypeDetach = "detach"

	// OperationLookbackHours is the default lookback window for endpoint operations
	OperationLookbackHours = 24
)

// Event Data Keys
// JSON field names for system event details
const (
	EventKeyOperationID    = "operation_id"
	EventKeyOperationType  = "operation_type"
	EventKeyEndpointID     = "endpoint_id"
	EventKeyEpEui          = "epEui"
	EventKeyTargetBS       = "target_bs"
	EventKeyTargetBSList   = "target_bs_list"
	EventKeyTargetBSCount  = "target_bs_count"
	EventKeyError          = "error"
	EventKeyBsEui          = "bsEui"          // UI compatibility
	EventKeyBaseStationEui = "baseStationEui" // UI compatibility
)

// Event Field Values
// Centralized values for system event fields
const (
	// EventCategoryEndpoint is the category for endpoint-related events
	EventCategoryEndpoint = "endpoint"

	// EventSourceTypeEndpoint is the source type for endpoint-originated events
	EventSourceTypeEndpoint = "endpoint"

	// BaseStationNone indicates no base station is associated with an event
	BaseStationNone = "none"
)

// Event Type Constants
// System event types for BSSCI error handling per §5.17-5.17.2 and propagation per §5.8-5.8.3
const (
	// EventTypeBSSCIErrorSent indicates service center sent error message to base station
	EventTypeBSSCIErrorSent = "bssci_error_sent"

	// EventTypeBSSCIErrorReceived indicates service center received error message from base station
	EventTypeBSSCIErrorReceived = "bssci_error_received"

	// EventTypeBSSCIErrorAck indicates service center received errorAck from base station
	EventTypeBSSCIErrorAck = "bssci_error_ack"

	// EventTypeAttachPropagateInitiated indicates service center initiated attach propagate to base station
	// BSSCI §5.8-5.8.3 Attach Propagate operation
	EventTypeAttachPropagateInitiated = "attach_propagate_initiated"

	// EventTypeAttachPropagateCompleted indicates attach propagate three-way handshake completed
	// BSSCI §3.8.3 Attach Propagate Complete
	EventTypeAttachPropagateCompleted = "attach_propagate_completed"

	// EventTypeDetachPropagateInitiated indicates service center initiated detach propagate to base station
	// BSSCI §5.9 Detach Propagate operation
	EventTypeDetachPropagateInitiated = "detach_propagate_initiated"

	// EventTypeAttachPropagateFailed indicates attach propagate operation failed
	EventTypeAttachPropagateFailed = "attach_propagate_failed"

	// EventTypeDetachPropagateFailed indicates detach propagate operation failed
	EventTypeDetachPropagateFailed = "detach_propagate_failed"

	// EventTypeVMActivateSuccess indicates VM MAC type activation succeeded
	EventTypeVMActivateSuccess = "vm_activate_success"

	// EventTypeVMActivateFailed indicates VM MAC type activation failed
	EventTypeVMActivateFailed = "vm_activate_failed"

	// EventTypeVMDeactivateSuccess indicates VM MAC type deactivation succeeded
	EventTypeVMDeactivateSuccess = "vm_deactivate_success"

	// EventTypeVMStatusReceived indicates VM status response received from base station
	EventTypeVMStatusReceived = "vm_status_received"

	// EventTypeVMUplinkData indicates VM uplink data received from endpoint
	EventTypeVMUplinkData = "vm_uplink_data"

	// EventTypeDLRxStatusReceived indicates downlink RX status received from endpoint
	EventTypeDLRxStatusReceived = "dl_rx_status_received"

	// EventTypeAttachOperationFailed indicates attach propagate operation failed
	EventTypeAttachOperationFailed = "attach_operation_failed"

	// EventTypeDetachOperationFailed indicates detach propagate operation failed
	EventTypeDetachOperationFailed = "detach_operation_failed"
)

// Event Title Constants
// Human-readable titles for BSSCI error events (BSSCI §5.17) and propagation events (BSSCI §5.8)
const (
	// TitleBSSCIErrorSent is the title for outbound error events
	TitleBSSCIErrorSent = "BSSCI Error Sent to Base Station"

	// TitleBSSCIErrorReceived is the title for inbound error events
	TitleBSSCIErrorReceived = "BSSCI Error Received from Base Station"

	// TitleBSSCIErrorAck is the title for errorAck events
	TitleBSSCIErrorAck = "BSSCI Error Acknowledged"

	// TitleAttachPropagateInitiated is the title for attach propagate initiated events
	TitleAttachPropagateInitiated = "Attach Propagate Initiated"

	// TitleAttachPropagateCompleted is the title for attach propagate completion events
	TitleAttachPropagateCompleted = "Attach Propagate Completed"

	// TitleDetachPropagateInitiated is the title for detach propagate initiated events
	TitleDetachPropagateInitiated = "Detach Propagate Initiated"

	// TitleOperationFailed is the title for failed operation events
	TitleOperationFailed = "Operation Failed"
)

// Event Description Constants
// Human-readable descriptions for BSSCI propagation events (BSSCI §5.8-5.8.3)
const (
	// DescriptionAttachPropagateEndpoint is the description for endpoint-side attach propagate events
	DescriptionAttachPropagateEndpoint = "Attach propagate initiated"

	// DescriptionAttachPropagateBaseStation is the description for base station-side attach propagate events
	DescriptionAttachPropagateBaseStation = "Service center initiated attach propagate"

	// DescriptionDetachPropagateEndpoint is the description for detach propagate initiated events
	DescriptionDetachPropagateEndpoint = "Detach propagate initiated"

	// DescriptionOperationFailedFormat is the format string for operation failure descriptions
	// Use with fmt.Sprintf(DescriptionOperationFailedFormat, targetBS, err)
	DescriptionOperationFailedFormat = "Failed to propagate to %s: %v"
)

// DL Data Revoke Operation Constants (BSSCI §5.13)
const (
	OperationDLDataRevoke = "dlDataRev"
)

// DL Data Revoke Event Constants (BSSCI §5.13)
const (
	EventDLDataRevokeInitiated = "dl_data_revoke_initiated"
	EventDLDataRevoked         = "dl_data_revoked"
)

// DL Data Result Status Constants (BSSCI §5.14)
// These are stored in downlink queue status field after transmission/revocation
const (
	DLDataResultRevoked      = "revoked"       // Successfully revoked via dlDataRevCmp
	DLDataResultRevokeFailed = "revoke_failed" // Revoke operation failed
)

// Downlink Queue Status Constants (BSSCI §5.11-5.14)
// Queue lifecycle states for tracking downlink message progression
const (
	DLQueueStatusPending     = "pending"     // Queued awaiting scheduler processing
	DLQueueStatusScheduled   = "scheduled"   // Scheduler selected for transmission
	DLQueueStatusReserved    = "reserved"    // Reserved for auto-dispatch (transient, within transaction)
	DLQueueStatusQueued      = "queued"      // Sent to BS via dlDataQue, awaiting transmission
	DLQueueStatusTransmitted = "transmitted" // BS transmitted downlink (dlDataQueCmp success)
	DLQueueStatusDelivered   = "delivered"   // Endpoint acknowledged receipt (if ack requested)
	DLQueueStatusFailed      = "failed"      // Transmission failed (dlDataQueCmp error)
	DLQueueStatusExpired     = "expired"     // Validity period elapsed before transmission
	DLQueueStatusRevoked     = "revoked"     // Revoked via dlDataRev before transmission
	DLQueueStatusAcked       = "acked"       // Endpoint acknowledgment received (dlDataRes)
)

// DL RX Query Status Constants (BSSCI §5.15)
// Query tracking lifecycle for dlRxStatQry correlation
const (
	DLRXQueryStatusPending  = "pending"  // Query sent, awaiting dlRxStat response
	DLRXQueryStatusReceived = "received" // dlRxStat correlated with query
	DLRXQueryStatusTimeout  = "timeout"  // Query expired (cleanup after 15 minutes)
)

// Propagation Reconciliation Default Values
// These control automatic endpoint propagation to base stations
const (
	DefaultReconcileBatchSize       = 50                     // Endpoints per batch
	DefaultReconcileInterBatchDelay = 100 * time.Millisecond // Delay between batches
	DefaultReconcileMaxRetries      = 3                      // Max retry attempts before pause
	DefaultReconcileRetryBackoff    = 30 * time.Second       // Base backoff for exponential retry
)

// Propagation Status Constants
// State machine for automatic endpoint propagation reconciliation
const (
	PropagationStatusIdle      = "idle"      // Initial state, not started
	PropagationStatusRunning   = "running"   // Actively processing endpoints
	PropagationStatusCompleted = "completed" // Successfully finished
	PropagationStatusError     = "error"     // Failed, may retry
	PropagationStatusPaused    = "paused"    // Max retries exceeded, needs manual reset
)

// Validation Status Constants
// Detach signature validation tracking (BSSCI §5.7)
// These values indicate whether cryptographic validation was performed for a detach operation.
// Used in detachMetadata.ValidationStatus to track validation state for unknown endpoints.
const (
	// ValidationStatusValidated indicates signature was cryptographically validated against endpoint.Sign
	ValidationStatusValidated = "validated"

	// ValidationStatusUnknownEndpoint indicates endpoint not found in database, signature cannot be validated
	ValidationStatusUnknownEndpoint = "unknown_endpoint"

	// ValidationStatusInvalidSignature indicates signature cryptographically invalid (CMAC/equality check failed)
	ValidationStatusInvalidSignature = "invalid_signature"

	// ValidationStatusDisabled indicates signature validation is disabled in server config
	ValidationStatusDisabled = "disabled"
)
