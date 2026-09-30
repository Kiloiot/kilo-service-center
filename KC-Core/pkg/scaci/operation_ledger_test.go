package scaci

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// operationLedger is the operation log of the unit tests. Like the unique
// index of migration 000186 it keeps one ulData per session and stored
// uplink; failFor makes recording fail for a session.
type operationLedger struct {
	mockOperationRepoStub
	mu         sync.Mutex
	operations []*models.SCACIOperation
	failFor    map[int64]error
}

func newOperationLedger() *operationLedger {
	return &operationLedger{failFor: make(map[int64]error)}
}

func (l *operationLedger) Record(_ context.Context, session *Session, opId int64, command string, direction models.OperationDirection, data map[string]interface{}) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.failFor[session.ID]; err != nil {
		return err
	}
	l.operations = append(l.operations, ledgerOperation(session, opId, command, direction, data))
	return nil
}

func (l *operationLedger) EnsureUplinkOperation(_ context.Context, session *Session, opId int64, sourceMessageID string, data map[string]interface{}) (*models.SCACIOperation, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.failFor[session.ID]; err != nil {
		return nil, false, err
	}
	for _, op := range l.operations {
		if op.SessionID == session.ID && op.RequestData[models.OperationRequestKeySourceMessageID] == sourceMessageID {
			return op, false, nil
		}
	}
	stored := make(map[string]interface{}, len(data)+1)
	for key, value := range data {
		stored[key] = value
	}
	stored[models.OperationRequestKeySourceMessageID] = sourceMessageID
	op := ledgerOperation(session, opId, CmdULData, models.OperationDirectionOutbound, stored)
	l.operations = append(l.operations, op)
	return op, true, nil
}

func (l *operationLedger) fail(sessionID int64, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failFor[sessionID] = err
}

func (l *operationLedger) heal(sessionID int64) { l.fail(sessionID, nil) }

// uplinkOpIDs are the opIds of the session's ulData operations for the uplink.
func (l *operationLedger) uplinkOpIDs(sessionID int64, messageID string) []int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	var opIDs []int64
	for _, op := range l.operations {
		if op.SessionID == sessionID && op.Command == CmdULData && op.RequestData[models.OperationRequestKeySourceMessageID] == messageID {
			opIDs = append(opIDs, op.OpId)
		}
	}
	return opIDs
}

func (l *operationLedger) UpdateOperationState(context.Context, int64, int64, models.OperationState, map[string]interface{}) error {
	return nil
}

// GetPendingOperations serves the session's operations back, as the log does
// for a resume.
func (l *operationLedger) GetPendingOperations(_ context.Context, sessionID int64) ([]*models.SCACIOperation, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var pending []*models.SCACIOperation
	for _, op := range l.operations {
		if op.SessionID == sessionID && op.State == string(models.OperationStatePending) {
			pending = append(pending, op)
		}
	}
	return pending, nil
}

// ledgerOperation stores the operation the way the JSONB operation log reads
// it back.
func ledgerOperation(session *Session, opId int64, command string, direction models.OperationDirection, data map[string]interface{}) *models.SCACIOperation {
	encoded, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	var stored map[string]interface{}
	if err := json.Unmarshal(encoded, &stored); err != nil {
		panic(err)
	}
	return &models.SCACIOperation{
		SessionID: session.ID, TenantID: session.TenantID, OpId: opId, Command: command,
		Direction: string(direction), State: string(models.OperationStatePending), RequestData: stored,
	}
}
