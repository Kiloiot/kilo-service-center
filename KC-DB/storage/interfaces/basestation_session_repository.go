package interfaces

import (
	"context"
	"time"

	"github.com/kilocenter/KC-DB/storage/models"
)

// BaseStationSessionRepository defines the interface for Base Station session management
// This implements MIOTY BSSCI v1.0.0 Section 3.3 session management requirements
type BaseStationSessionRepository interface {
	// CreateSession creates a new Base Station session per MIOTY BSSCI 3.3
	CreateSession(ctx context.Context, req *models.BaseStationSessionCreateRequest) (*models.BaseStationSession, error)

	// GetSessionByID retrieves a session by ID
	GetSessionByID(ctx context.Context, tenantID, sessionID int64) (*models.BaseStationSession, error)

	// GetActiveSessionByBaseStation retrieves the active session for a Base Station
	GetActiveSessionByBaseStation(ctx context.Context, tenantID, baseStationID int64) (*models.BaseStationSession, error)

	// GetSessionByBsUUID retrieves a session by Base Station session UUID (for resumption)
	GetSessionByBsUUID(ctx context.Context, tenantID int64, snBsUUID [16]byte) (*models.BaseStationSession, error)

	// GetSessionByScUUID retrieves a session by Service Center session UUID
	GetSessionByScUUID(ctx context.Context, tenantID int64, snScUUID [16]byte) (*models.BaseStationSession, error)

	// UpdateSession updates session fields (operation IDs, status, timing)
	UpdateSession(ctx context.Context, tenantID, sessionID int64, req *models.BaseStationSessionUpdateRequest) error

	// UpdateOperationIDs updates both Base Station and Service Center operation IDs atomically
	UpdateOperationIDs(ctx context.Context, tenantID, sessionID int64, bsOpId, scOpId int64) error

	// UpdatePing updates the last ping timestamp
	UpdatePing(ctx context.Context, tenantID, sessionID int64) error

	// UpdateEncoding updates the message encoding for a session (json or msgpack)
	// This is called when encoding is negotiated on first message per BSSCI Section 1
	UpdateEncoding(ctx context.Context, sessionID int64, encoding string) error

	// TerminateSession marks a session as terminated
	TerminateSession(ctx context.Context, tenantID, sessionID int64) error

	// TerminateAllSessions terminates all active sessions for a Base Station (for cleanup)
	TerminateAllSessions(ctx context.Context, tenantID, baseStationID int64) error

	// ListSessions retrieves sessions based on filter criteria
	ListSessions(ctx context.Context, filter *models.BaseStationSessionFilter) ([]*models.BaseStationSession, int64, error)

	// GetSessionStatistics retrieves session statistics
	GetSessionStatistics(ctx context.Context, tenantID int64) (*SessionStatistics, error)

	// CheckSessionResumable determines if a session can be resumed per MIOTY spec
	CheckSessionResumable(ctx context.Context, tenantID int64, snBsUuid [16]byte, bsOpId int64) (*SessionResumptionInfo, error) //nolint:revive // snBsUuid, bsOpId match BSSCI spec naming

	// CleanupExpiredSessions removes old terminated sessions (housekeeping)
	CleanupExpiredSessions(ctx context.Context, olderThan int64) (int64, error) // olderThan in hours

	// UpdateCountersAndTimestamp updates operation counters by session UUID
	// NOTE: sessionUUID is [16]byte to match database CHECK constraint (length = 16)
	UpdateCountersAndTimestamp(ctx context.Context, sessionUUID [16]byte, bsOpId, scOpId int64) error
}

// SessionStatistics represents aggregated session statistics
type SessionStatistics struct {
	TotalSessions          int64   `json:"total_sessions"`
	ActiveSessions         int64   `json:"active_sessions"`
	TerminatedSessions     int64   `json:"terminated_sessions"`
	ResumableSessions      int64   `json:"resumable_sessions"`
	AverageSessionDuration float64 `json:"average_session_duration_hours"`
	TotalSessionTime       float64 `json:"total_session_time_hours"`
}

// SessionResumptionInfo provides information for session resumption decision
type SessionResumptionInfo struct {
	CanResume            bool   `json:"can_resume"`
	SessionID            int64  `json:"session_id"`
	LastKnownBsOpId      int64  `json:"last_known_bs_op_id"`
	LastKnownScOpId      int64  `json:"last_known_sc_op_id"`
	SessionAge           int64  `json:"session_age_hours"`
	ReasonIfNotResumable string `json:"reason_if_not_resumable,omitempty"`
}

// OperationTrackingRepository defines the interface for BSSCI operation tracking
// This implements operation state management per MIOTY BSSCI requirements
type OperationTrackingRepository interface {
	// RecordOperation records a new BSSCI operation
	RecordOperation(ctx context.Context, req *OperationTrackingRequest) error

	// UpdateOperationState updates operation state (acknowledged, completed, failed)
	UpdateOperationState(ctx context.Context, sessionID int64, opId int64, state OperationState, responseData map[string]interface{}) error

	// GetPendingOperations retrieves pending operations for a session
	GetPendingOperations(ctx context.Context, sessionID int64) ([]*OperationTracking, error)

	// CleanupCompletedOperations removes old completed operations
	CleanupCompletedOperations(ctx context.Context, olderThan int64) (int64, error) // olderThan in hours
}

// OperationState represents the state of a BSSCI operation
type OperationState string

// OperationState constants define the lifecycle states for BSSCI operations.
const (
	OperationStatePending      OperationState = "pending"
	OperationStateAcknowledged OperationState = "acknowledged"
	OperationStateCompleted    OperationState = "completed"
	OperationStateFailed       OperationState = "failed"
)

// OperationDirection represents the direction of a BSSCI operation
type OperationDirection string

// OperationDirection constants define the message direction for BSSCI operations.
const (
	OperationDirectionInbound  OperationDirection = "inbound"  // From Base Station to Service Center
	OperationDirectionOutbound OperationDirection = "outbound" // From Service Center to Base Station
)

// OperationTrackingRequest represents a request to track a BSSCI operation
type OperationTrackingRequest struct {
	SessionID   int64                  `json:"session_id" validate:"required"`
	OpId        int64                  `json:"op_id" validate:"required"`
	Command     string                 `json:"command" validate:"required"`
	Direction   OperationDirection     `json:"direction" validate:"required"`
	RequestData map[string]interface{} `json:"request_data,omitempty"`
}

// OperationTracking represents a tracked BSSCI operation
type OperationTracking struct {
	ID             int64                  `json:"id"`
	SessionID      int64                  `json:"session_id"`
	OpId           int64                  `json:"op_id"`
	Command        string                 `json:"command"`
	Direction      OperationDirection     `json:"direction"`
	State          OperationState         `json:"state"`
	InitiatedAt    *time.Time             `json:"initiated_at"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	RequestData    map[string]interface{} `json:"request_data,omitempty"`
	ResponseData   map[string]interface{} `json:"response_data,omitempty"`
	ErrorMessage   *string                `json:"error_message,omitempty"`
}
