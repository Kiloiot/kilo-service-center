package interfaces

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// SCACISessionRepository defines the interface for SCACI session management
// Implements MIOTY SCACI v1.0.0 Section 1 session persistence requirements
type SCACISessionRepository interface {
	// CreateSession creates a new SCACI session per SCACI §1
	CreateSession(ctx context.Context, req *models.SCACISessionCreateRequest) (*models.SCACISession, error)

	// RetirePriorSessions terminates every session of an application center,
	// in the transaction that creates its next one (SCACI §1)
	RetirePriorSessions(ctx context.Context, ac models.SCACIApplicationCenter) error

	// GetSessionByID retrieves a session by ID
	GetSessionByID(ctx context.Context, tenantID, sessionID int64) (*models.SCACISession, error)

	// GetSessionByAcUUID retrieves the application center's latest session of
	// its session UUID (for resumption)
	GetSessionByAcUUID(ctx context.Context, ac models.SCACIApplicationCenter, snAcUuid [16]byte) (*models.SCACISession, error) //nolint:revive // snAcUuid matches SCACI spec naming

	// GetSessionByScUUID retrieves a session by Service Center session UUID
	GetSessionByScUUID(ctx context.Context, tenantID int64, snScUuid [16]byte) (*models.SCACISession, error) //nolint:revive // snScUuid matches SCACI spec naming

	// UpdateOperationIDs updates both AC and SC operation IDs atomically
	UpdateOperationIDs(ctx context.Context, tenantID, sessionID int64, acOpId, scOpId int64) error

	// UpdateHeartbeat updates the last heartbeat timestamp
	UpdateHeartbeat(ctx context.Context, tenantID, sessionID int64) error

	// TerminateSession marks a session as terminated (not resumable)
	TerminateSession(ctx context.Context, tenantID, sessionID int64) error

	// ListSessions retrieves sessions based on filter criteria
	ListSessions(ctx context.Context, filter *models.SCACISessionFilter) ([]*models.SCACISession, int64, error)

	// GetSessionStatistics retrieves aggregated session statistics
	GetSessionStatistics(ctx context.Context, tenantID int64) (*models.SCACISessionStatistics, error)

	// CheckSessionResumable reports whether the application center's latest
	// session of snAcUuid can still be resumed (SCACI §1) and the operation ID
	// counters it stored
	CheckSessionResumable(ctx context.Context, ac models.SCACIApplicationCenter, snAcUuid [16]byte) (*models.SCACISessionResumptionInfo, error) //nolint:revive // snAcUuid matches SCACI spec naming
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

	// CompleteFailedOperation sets completed_at on a failed operation without changing state
	// Used by error handshake (errorAck) to mark failed ops as "completed" while preserving
	// error evidence (state='failed', error_code, error_token remain intact)
	// Merges responseData with existing response_data JSONB column
	CompleteFailedOperation(ctx context.Context, sessionID int64, opId int64, responseData map[string]interface{}) error
}
