package bssciservices

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// mockPendingOpsStore is the pending-operation store double with
// in-memory rows so tests can assert which session's operations were removed
type mockPendingOpsStore struct {
	rows      map[int64][]int64 // basestation session ID -> operation IDs
	deleteErr error             // when set, DeleteBySession returns it
	mu        sync.Mutex
}

func newMockPendingOpsStore() *mockPendingOpsStore {
	return &mockPendingOpsStore{rows: make(map[int64][]int64)}
}

func (m *mockPendingOpsStore) seed(sessionID int64, operationIDs ...int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rows[sessionID] = append(m.rows[sessionID], operationIDs...)
}

func (m *mockPendingOpsStore) operationIDs(sessionID int64) []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]int64(nil), m.rows[sessionID]...)
}

func (m *mockPendingOpsStore) Create(_ context.Context, req *models.PendingOperationRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rows[req.SessionID] = append(m.rows[req.SessionID], req.OperationID)
	return nil
}

func (m *mockPendingOpsStore) CreateBatch(ctx context.Context, reqs []*models.PendingOperationRequest) error {
	for _, req := range reqs {
		if err := m.Create(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockPendingOpsStore) UpdateMetadata(_ context.Context, _ int64, _ int64, _ json.RawMessage) error {
	return nil
}

func (m *mockPendingOpsStore) DeleteBySessionAndOperation(_ context.Context, sessionID int64, operationID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	remaining := make([]int64, 0, len(m.rows[sessionID]))
	for _, opID := range m.rows[sessionID] {
		if opID != operationID {
			remaining = append(remaining, opID)
		}
	}
	m.rows[sessionID] = remaining
	return nil
}

func (m *mockPendingOpsStore) DeleteBySession(_ context.Context, sessionID int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.deleteErr != nil {
		return 0, m.deleteErr
	}
	deleted := int64(len(m.rows[sessionID]))
	delete(m.rows, sessionID)
	return deleted, nil
}

func (m *mockPendingOpsStore) GetBySession(_ context.Context, sessionID int64) ([]*models.PendingOperation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ops := make([]*models.PendingOperation, 0, len(m.rows[sessionID]))
	for _, opID := range m.rows[sessionID] {
		ops = append(ops, &models.PendingOperation{SessionID: sessionID, OperationID: opID})
	}
	return ops, nil
}
