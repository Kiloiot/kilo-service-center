package scaciservices

import (
	"context"

	"github.com/kilocenter/KC-Core/pkg/scaci"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
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
	repo interfaces.SCACIOperationRepository
}

// NewOperationRecorder creates a new operation recorder service
//
// Parameters:
//   - repo: SCACI operation repository for persisting operation records
//
// The repo dependency should be obtained via storage.SCACIOperations() or similar factory.
func NewOperationRecorder(repo interfaces.SCACIOperationRepository) scaci.OperationRecorder {
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

	req := &models.SCACIOperationRequest{
		SessionID:   session.ID,
		TenantID:    session.TenantID,
		OpId:        opId,
		Command:     command,
		Direction:   string(direction),
		RequestData: data,
	}

	_, err := r.repo.RecordOperation(ctx, req)
	return err
}
