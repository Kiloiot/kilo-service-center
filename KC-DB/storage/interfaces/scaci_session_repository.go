package interfaces

import (
	"context"

	"github.com/kilocenter/KC-DB/storage/models"
)

// SCACISessionRepository defines the interface for SCACI session management
// Implements MIOTY SCACI v1.0.0 Section 1 session persistence requirements
type SCACISessionRepository interface {
	// CreateSession creates a new SCACI session per SCACI §1
	CreateSession(ctx context.Context, req *models.SCACISessionCreateRequest) (*models.SCACISession, error)

	// GetSessionByID retrieves a session by ID
	GetSessionByID(ctx context.Context, tenantID, sessionID int64) (*models.SCACISession, error)

	// GetActiveSessionByAcEUI retrieves the active session for an Application Center
	GetActiveSessionByAcEUI(ctx context.Context, tenantID int64, acEui [8]byte) (*models.SCACISession, error)

	// GetSessionByAcUUID retrieves a session by Application Center session UUID (for resumption)
	GetSessionByAcUUID(ctx context.Context, tenantID int64, snAcUuid [16]byte) (*models.SCACISession, error) //nolint:revive // snAcUuid matches SCACI spec naming

	// GetSessionByScUUID retrieves a session by Service Center session UUID
	GetSessionByScUUID(ctx context.Context, tenantID int64, snScUuid [16]byte) (*models.SCACISession, error) //nolint:revive // snScUuid matches SCACI spec naming

	// UpdateSession updates session fields (status, timing, metadata, etc.)
	UpdateSession(ctx context.Context, tenantID, sessionID int64, req *models.SCACISessionUpdateRequest) error

	// UpdateOperationIDs updates both AC and SC operation IDs atomically
	UpdateOperationIDs(ctx context.Context, tenantID, sessionID int64, acOpId, scOpId int64) error

	// UpdateHeartbeat updates the last heartbeat timestamp
	UpdateHeartbeat(ctx context.Context, tenantID, sessionID int64) error

	// DisconnectSession marks a session as disconnected (but potentially resumable)
	DisconnectSession(ctx context.Context, tenantID, sessionID int64) error

	// TerminateSession marks a session as terminated (not resumable)
	TerminateSession(ctx context.Context, tenantID, sessionID int64) error

	// TerminateAllSessions terminates all active sessions for an Application Center (cleanup)
	TerminateAllSessions(ctx context.Context, tenantID int64, acEui [8]byte) error

	// ListSessions retrieves sessions based on filter criteria
	ListSessions(ctx context.Context, filter *models.SCACISessionFilter) ([]*models.SCACISession, int64, error)

	// GetSessionStatistics retrieves aggregated session statistics
	GetSessionStatistics(ctx context.Context, tenantID int64) (*models.SCACISessionStatistics, error)

	// CheckSessionResumable determines if a session can be resumed per SCACI §1 spec
	// Validates UUID consistency and both AC/SC operation ID progression per SCACI-S.1-05/3.2-03
	CheckSessionResumable(ctx context.Context, tenantID int64, snAcUuid [16]byte, acOpId int64, scOpId int64) (*models.SCACISessionResumptionInfo, error) //nolint:revive // snAcUuid, acOpId, scOpId match SCACI spec naming

	// CleanupExpiredSessions removes old terminated sessions (housekeeping)
	// olderThan specifies hours since termination
	CleanupExpiredSessions(ctx context.Context, olderThan int64) (int64, error)
}

// SCACIOperationRepository defines the interface for SCACI operation tracking
// Implements operation state management per MIOTY SCACI requirements
type SCACIOperationRepository interface {
	// RecordOperation records a new SCACI operation
	RecordOperation(ctx context.Context, req *models.SCACIOperationRequest) (*models.SCACIOperation, error)

	// UpdateOperationState updates operation state (acknowledged, completed, failed)
	UpdateOperationState(ctx context.Context, sessionID int64, opId int64, state models.OperationState, responseData map[string]interface{}) error

	// UpdateOperationStateWithError updates state with explicit error columns per SCACI §3.14
	// Used when failing an operation with structured error metadata per SCACI §3.14
	UpdateOperationStateWithError(ctx context.Context, sessionID int64, opId int64,
		state models.OperationState, errorCode int, errorToken string, errorMessage string,
		responseData map[string]interface{}) error

	// GetOperationByOpID retrieves an operation by session ID and operation ID
	GetOperationByOpID(ctx context.Context, sessionID int64, opId int64) (*models.SCACIOperation, error)

	// GetPendingOperations retrieves all pending operations for a session
	GetPendingOperations(ctx context.Context, sessionID int64) ([]*models.SCACIOperation, error)

	// GetRecentOperations retrieves recent operations for a session (for monitoring)
	GetRecentOperations(ctx context.Context, sessionID int64, limit int) ([]*models.SCACIOperation, error)

	// CleanupCompletedOperations removes old completed operations (housekeeping)
	// olderThan specifies hours since completion
	CleanupCompletedOperations(ctx context.Context, olderThan int64) (int64, error)

	// GetTenantOperationSummary retrieves aggregated operation statistics for a tenant
	// windowHours specifies the lookback period for aggregation
	// Returns summary with command, direction, and state counts per SCACI §§3.8–3.12
	GetTenantOperationSummary(ctx context.Context, tenantID int64, windowHours int) (*models.SCACIOperationSummary, error)

	// CompleteFailedOperation sets completed_at on a failed operation without changing state
	// Used by error handshake (errorAck) to mark failed ops as "completed" while preserving
	// error evidence (state='failed', error_code, error_token remain intact)
	// Merges responseData with existing response_data JSONB column
	CompleteFailedOperation(ctx context.Context, sessionID int64, opId int64, responseData map[string]interface{}) error
}
