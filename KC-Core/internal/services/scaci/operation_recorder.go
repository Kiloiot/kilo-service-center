package scaciservices

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// operationRecorder implements scaci.OperationRecorder interface
//
// This service persists SCACI operations to the scaci_operation_log table for resume safety
// per SCACI §3.4. All operations are logged with direction, command, and request data to enable
// session resumption and compliance auditing.
//
// Spec References:
//   - §3.4: Session resumption requires operation history
//   - §3.9.2: Operation logging for audit trail
type operationRecorder struct {
	repo SCACIOperationStore
}

// NewOperationRecorder creates a new operation recorder service
//
// Parameters:
//   - repo: SCACI operation repository for persisting operation records
//
// The repo dependency should be obtained via storage.SCACIOperations() or similar factory.
func NewOperationRecorder(repo SCACIOperationStore) scaci.OperationRecorder {
	return &operationRecorder{
		repo: repo,
	}
}

// Record persists a SCACI operation to the operation log per SCACI §3.4
//
// Operation records enable session resumption by tracking the AC's operation counter (opId)
// and the Service Center's operation counter. All operations are logged regardless of success
// or failure to maintain a complete audit trail.
//
// Operation Direction:
//   - OperationDirectionInbound: AC → SC operations (reg, dereg, dlDataQue, ulDataTx)
//   - OperationDirectionOutbound: SC → AC operations (ulData, dlDataRes)
//
// Parameters:
//   - ctx: Request context for database operation
//   - session: Active SCACI session (provides SessionID and TenantID)
//   - opId: Operation ID (positive for AC operations, negative for SC operations)
//   - command: SCACI command name (e.g., "reg", "dereg", "ulData", "dlDataQue")
//   - direction: Operation direction (use models.OperationDirectionInbound or models.OperationDirectionOutbound)
//   - data: Request data to persist (command-specific fields for resume/audit)
//
// Returns error if database persistence fails (callers should handle gracefully).
func (r *operationRecorder) Record(ctx context.Context, session *scaci.Session, opId int64,
	command string, direction models.OperationDirection, data map[string]interface{}) error {

	_, err := r.repo.RecordOperation(ctx, operationRequest(session, opId, command, direction, data))
	return err
}

// EnsureUplinkOperation records the outbound ulData operation delivering the
// stored uplink sourceMessageID to the session, unless the session has one
// already, and reports whether it recorded it now (SCACI §3.2).
func (r *operationRecorder) EnsureUplinkOperation(ctx context.Context, session *scaci.Session, opId int64,
	sourceMessageID string, data map[string]interface{}) (*models.SCACIOperation, bool, error) {
	return r.repo.EnsureUplinkOperation(ctx,
		operationRequest(session, opId, scaci.CmdULData, models.OperationDirectionOutbound, data), sourceMessageID)
}

// operationRequest is the operation log row of one operation of the session.
func operationRequest(session *scaci.Session, opId int64, command string,
	direction models.OperationDirection, data map[string]interface{}) *models.SCACIOperationRequest {
	return &models.SCACIOperationRequest{
		SessionID:   session.ID,
		TenantID:    session.TenantID,
		OpId:        opId,
		Command:     command,
		Direction:   string(direction),
		RequestData: data,
	}
}
