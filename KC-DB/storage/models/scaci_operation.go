package models

import (
	"time"
)

// SCACIOperation represents a tracked SCACI protocol operation
// Implements three-way handshake tracking: Request → Response → Complete
type SCACIOperation struct {
	ID        int64 `db:"id" json:"id"`
	SessionID int64 `db:"session_id" json:"session_id"`
	TenantID  int64 `db:"tenant_id" json:"tenant_id"`

	// Operation identification
	OpId      int64  `db:"op_id" json:"op_id"`         // Signed operation ID (SCACI §2.5)
	Command   string `db:"command" json:"command"`     // SCACI command name
	Direction string `db:"direction" json:"direction"` // inbound or outbound

	// Operation state
	State string `db:"state" json:"state"` // pending, acknowledged, completed, failed

	// Operation data
	RequestData  map[string]interface{} `db:"request_data" json:"request_data,omitempty"`
	ResponseData map[string]interface{} `db:"response_data" json:"response_data,omitempty"`
	ErrorMessage *string                `db:"error_message" json:"error_message,omitempty"`
	ErrorCode    *int                   `db:"error_code" json:"error_code,omitempty"`   // POSIX error code per SCACI §3.14.1
	ErrorToken   *string                `db:"error_token" json:"error_token,omitempty"` // Internal catalog token (not on wire)

	// Timing
	InitiatedAt    time.Time  `db:"initiated_at" json:"initiated_at"`
	AcknowledgedAt *time.Time `db:"acknowledged_at" json:"acknowledged_at,omitempty"`
	CompletedAt    *time.Time `db:"completed_at" json:"completed_at,omitempty"`

	// Audit
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// SCACIOperationRequest represents a request to track a new SCACI operation
type SCACIOperationRequest struct {
	SessionID   int64                  `json:"session_id" validate:"required"`
	TenantID    int64                  `json:"tenant_id" validate:"required"`
	OpId        int64                  `json:"op_id" validate:"required"`
	Command     string                 `json:"command" validate:"required"`
	Direction   string                 `json:"direction" validate:"required,oneof=inbound outbound"`
	RequestData map[string]interface{} `json:"request_data,omitempty"`
}

// SCACIOperationStateUpdate represents a request to update operation state
type SCACIOperationStateUpdate struct {
	State        string                 `json:"state" validate:"required,oneof=pending acknowledged completed completed_with_warnings failed"`
	ResponseData map[string]interface{} `json:"response_data,omitempty"`
	ErrorMessage *string                `json:"error_message,omitempty"`
}

// OperationState represents the state of a SCACI operation
type OperationState string

const (
	// OperationStatePending indicates the operation request has been sent
	OperationStatePending OperationState = "pending"

	// OperationStateAcknowledged indicates a response has been received
	OperationStateAcknowledged OperationState = "acknowledged"

	// OperationStateCompleted indicates the complete message has been received
	OperationStateCompleted OperationState = "completed"

	// OperationStateCompletedWithWarnings indicates the handshake succeeded but
	// post-handshake cleanup (e.g., downlink revoke, detach propagation) had issues.
	// Distinguishes protocol success from cleanup hygiene problems per §3.7.3.
	OperationStateCompletedWithWarnings OperationState = "completed_with_warnings"

	// OperationStateFailed indicates the operation failed
	OperationStateFailed OperationState = "failed"
)

// OperationDirection represents the direction of a SCACI operation
type OperationDirection string

const (
	// OperationDirectionInbound represents AC → SC operations
	OperationDirectionInbound OperationDirection = "inbound"

	// OperationDirectionOutbound represents SC → AC operations
	OperationDirectionOutbound OperationDirection = "outbound"
)

// IsPending returns true if the operation is still pending
func (o *SCACIOperation) IsPending() bool {
	return o.State == string(OperationStatePending)
}

// IsAcknowledged returns true if the operation has been acknowledged
func (o *SCACIOperation) IsAcknowledged() bool {
	return o.State == string(OperationStateAcknowledged)
}

// IsCompleted returns true if the operation has completed successfully
func (o *SCACIOperation) IsCompleted() bool {
	return o.State == string(OperationStateCompleted)
}

// IsCompletedWithWarnings returns true if the operation completed with cleanup warnings
func (o *SCACIOperation) IsCompletedWithWarnings() bool {
	return o.State == string(OperationStateCompletedWithWarnings)
}

// IsFailed returns true if the operation failed
func (o *SCACIOperation) IsFailed() bool {
	return o.State == string(OperationStateFailed)
}

// GetDuration returns the duration from initiation to completion (or current time if not completed)
func (o *SCACIOperation) GetDuration() time.Duration {
	endTime := time.Now()
	if o.CompletedAt != nil {
		endTime = *o.CompletedAt
	} else if o.AcknowledgedAt != nil {
		endTime = *o.AcknowledgedAt
	}
	return endTime.Sub(o.InitiatedAt)
}

// IsInbound returns true if this is an AC → SC operation
func (o *SCACIOperation) IsInbound() bool {
	return o.Direction == string(OperationDirectionInbound)
}

// IsOutbound returns true if this is an SC → AC operation
func (o *SCACIOperation) IsOutbound() bool {
	return o.Direction == string(OperationDirectionOutbound)
}

// SCACIOperationSummary represents aggregated operation statistics for a tenant
// Used for monitoring and reporting per SCACI §§3.8-3.12
type SCACIOperationSummary struct {
	TenantID        int64            `json:"tenant_id"`
	WindowHours     int              `json:"window_hours"`     // Time window for aggregation in hours
	TotalOperations int64            `json:"total_operations"` // Total number of operations in window
	CommandCounts   map[string]int64 `json:"command_counts"`   // Count by command (ulData, dlDataQue, etc.)
	DirectionCounts map[string]int64 `json:"direction_counts"` // Count by direction (inbound, outbound)
	StateCounts     map[string]int64 `json:"state_counts"`     // Count by state (pending, completed, failed)
	ErrorCount      int64            `json:"error_count"`      // Count of failed operations
}
