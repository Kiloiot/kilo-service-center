package bssciservices

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPendingOperationRepositoryWithCallTracking extends mockPendingOperationRepository with call count tracking
type mockPendingOperationRepositoryWithCallTracking struct {
	createCallCount int
	mu              sync.Mutex
}

func (m *mockPendingOperationRepositoryWithCallTracking) Create(_ context.Context, _ *interfaces.PendingOperationRequest) error {
	m.mu.Lock()
	m.createCallCount++
	m.mu.Unlock()
	return nil
}

func (m *mockPendingOperationRepositoryWithCallTracking) UpdateMetadata(_ context.Context, _ int64, _ int64, _ json.RawMessage) error {
	return nil
}

func (m *mockPendingOperationRepositoryWithCallTracking) DeleteBySessionAndOperation(_ context.Context, _ int64, _ int64) error {
	return nil
}

func (m *mockPendingOperationRepositoryWithCallTracking) DeleteByOperation(_ context.Context, _ int64) error {
	return nil
}

func (m *mockPendingOperationRepositoryWithCallTracking) DeleteBySession(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

func (m *mockPendingOperationRepositoryWithCallTracking) GetBySession(_ context.Context, _ int64) ([]*interfaces.PendingOperation, error) {
	return []*interfaces.PendingOperation{}, nil
}

// TestStatusServiceMultiSessionIsolation verifies that StatusService correctly isolates
// pending operations across multiple concurrent sessions using SessionOpKey composite key,
// preventing opId collisions and tenant leakage.
//
// BSSCI §5.11-5.12.3: Multi-session roaming scenario where same opId
// is used by different base stations without collision.
func TestStatusServiceMultiSessionIsolation(t *testing.T) {
	// Create shared pendingOps map with SessionOpKey composite key
	pendingOps := make(map[bssci.SessionOpKey]*bssci.PendingOperation)
	var mu sync.RWMutex

	// Create StatusService with mock repository
	log := logger.NewNop()
	mockRepo := &mockPendingOperationRepositoryWithCallTracking{}
	statusSvc := NewStatusService(&pendingOps, &mu, mockRepo, log)

	// Create three sessions from different base stations
	session1 := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:          "session-bs1",
			DbSessionID: 1001,
		},
	}
	session2 := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:          "session-bs2",
			DbSessionID: 1002,
		},
	}
	session3 := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:          "session-bs3",
			DbSessionID: 1003,
		},
	}

	ctx := context.Background()

	// Same opId used across all sessions (simulates concurrent downlink operations)
	opId := int64(-100)

	// Record pending operations with same opId but different sessions
	// Each has unique queue ID and tenant for isolation verification
	op1 := &bssci.PendingOperation{
		SessionSlug:   session1.ID,
		OperationID:   opId,
		OperationType: "dlDataQue",
		Endpoint:      []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07},
		Metadata: map[string]interface{}{
			"queId":    int64(5001),
			"tenantID": "tenant-1",
		},
	}
	err := statusSvc.RecordPendingOperation(ctx, session1, opId, op1, session1.DbSessionID)
	require.NoError(t, err, "Session 1 operation should be recorded")

	op2 := &bssci.PendingOperation{
		SessionSlug:   session2.ID,
		OperationID:   opId,
		OperationType: "dlDataQue",
		Endpoint:      []byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17},
		Metadata: map[string]interface{}{
			"queId":    int64(5002),
			"tenantID": "tenant-2",
		},
	}
	err = statusSvc.RecordPendingOperation(ctx, session2, opId, op2, session2.DbSessionID)
	require.NoError(t, err, "Session 2 operation should be recorded")

	op3 := &bssci.PendingOperation{
		SessionSlug:   session3.ID,
		OperationID:   opId,
		OperationType: "dlDataQue",
		Endpoint:      []byte{0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27},
		Metadata: map[string]interface{}{
			"queId":    int64(5003),
			"tenantID": "tenant-3",
		},
	}
	err = statusSvc.RecordPendingOperation(ctx, session3, opId, op3, session3.DbSessionID)
	require.NoError(t, err, "Session 3 operation should be recorded")

	// Verify map contains exactly 3 entries (no collisions)
	mu.RLock()
	mapSize := len(pendingOps)
	mu.RUnlock()
	assert.Equal(t, 3, mapSize, "Map should contain 3 distinct operations despite same opId")

	// Verify ExtractQueueMetadata returns correct data for each session
	// This is the critical Gap 1 fix - metadata must not leak across sessions
	eui1, qid1, tenant1 := statusSvc.ExtractQueueMetadata(session1, opId)
	assert.Equal(t, uint64(0x0001020304050607), eui1, "Session 1 should return correct EUI")
	assert.Equal(t, int64(5001), qid1, "Session 1 should return correct queue ID")
	assert.Equal(t, "tenant-1", tenant1, "Session 1 should return correct tenant")

	eui2, qid2, tenant2 := statusSvc.ExtractQueueMetadata(session2, opId)
	assert.Equal(t, uint64(0x1011121314151617), eui2, "Session 2 should return correct EUI")
	assert.Equal(t, int64(5002), qid2, "Session 2 should return correct queue ID")
	assert.Equal(t, "tenant-2", tenant2, "Session 2 should return correct tenant")

	eui3, qid3, tenant3 := statusSvc.ExtractQueueMetadata(session3, opId)
	assert.Equal(t, uint64(0x2021222324252627), eui3, "Session 3 should return correct EUI")
	assert.Equal(t, int64(5003), qid3, "Session 3 should return correct queue ID")
	assert.Equal(t, "tenant-3", tenant3, "Session 3 should return correct tenant")

	// Verify GetPendingOperation returns correct operation for each session
	retrieved1, err := statusSvc.GetPendingOperation(session1, opId)
	require.NoError(t, err)
	assert.Equal(t, session1.ID, retrieved1.SessionSlug)
	assert.Equal(t, "tenant-1", retrieved1.Metadata["tenantID"])

	retrieved2, err := statusSvc.GetPendingOperation(session2, opId)
	require.NoError(t, err)
	assert.Equal(t, session2.ID, retrieved2.SessionSlug)
	assert.Equal(t, "tenant-2", retrieved2.Metadata["tenantID"])

	retrieved3, err := statusSvc.GetPendingOperation(session3, opId)
	require.NoError(t, err)
	assert.Equal(t, session3.ID, retrieved3.SessionSlug)
	assert.Equal(t, "tenant-3", retrieved3.Metadata["tenantID"])

	// Verify RemovePendingOperation only removes the correct session's operation
	err = statusSvc.RemovePendingOperation(ctx, session2, opId)
	require.NoError(t, err, "Session 2 operation should be removed")

	mu.RLock()
	mapSize = len(pendingOps)
	mu.RUnlock()
	assert.Equal(t, 2, mapSize, "Map should have 2 operations after removing session 2")

	// Session 1 and 3 should still exist
	_, err = statusSvc.GetPendingOperation(session1, opId)
	assert.NoError(t, err, "Session 1 operation should still exist")

	_, err = statusSvc.GetPendingOperation(session3, opId)
	assert.NoError(t, err, "Session 3 operation should still exist")

	// Session 2 should be gone
	_, err = statusSvc.GetPendingOperation(session2, opId)
	assert.Error(t, err, "Session 2 operation should not exist")

	t.Logf("PASS: StatusService correctly isolates %d sessions with same opId=%d", 3, opId)
}

// TestStatusServiceCachePopulationForNewOperations verifies that StatusService cache
// is populated for NEW operations (not just session resume).
//
// Before fix: Cache only populated during session resume, causing cache misses
// After fix: Cache populated via RecordPendingOperation for all new operations
func TestStatusServiceCachePopulationForNewOperations(t *testing.T) {
	// Create shared pendingOps map
	pendingOps := make(map[bssci.SessionOpKey]*bssci.PendingOperation)
	var mu sync.RWMutex

	log := logger.NewNop()
	mockRepo := &mockPendingOperationRepositoryWithCallTracking{}
	statusSvc := NewStatusService(&pendingOps, &mu, mockRepo, log)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:          "test-session",
			DbSessionID: 2001,
		},
	}

	ctx := context.Background()

	// Record a NEW operation (simulates initDLDataQue calling persistPendingOperation)
	opId := int64(-200)
	op := &bssci.PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   opId,
		OperationType: "dlDataQue",
		Endpoint:      []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11},
		Metadata: map[string]interface{}{
			"queId":    int64(9001),
			"tenantID": "test-tenant",
		},
	}

	// CRITICAL: This should populate BOTH the DB and the in-memory cache
	err := statusSvc.RecordPendingOperation(ctx, session, opId, op, session.DbSessionID)
	require.NoError(t, err, "New operation should be recorded")

	// Verify the cache was populated (this is the Gap 1 fix)
	mu.RLock()
	key := bssci.SessionOpKey{
		SessionID:   session.ID,
		OperationID: opId,
	}
	cachedOp, exists := pendingOps[key]
	mu.RUnlock()

	assert.True(t, exists, "Operation MUST be in cache for new operations (Gap 1 fix)")
	assert.NotNil(t, cachedOp, "Cached operation must not be nil")

	// Verify ExtractQueueMetadata can retrieve data from cache
	// This simulates handleDLDataQueueResponse looking up queue metadata
	eui, qid, tenant := statusSvc.ExtractQueueMetadata(session, opId)
	assert.Equal(t, uint64(0xAABBCCDDEEFF0011), eui, "Should extract correct EUI from cache")
	assert.Equal(t, int64(9001), qid, "Should extract correct queue ID from cache")
	assert.Equal(t, "test-tenant", tenant, "Should extract correct tenant from cache")

	// Verify mock repository was called (DB persistence)
	assert.Equal(t, 1, mockRepo.createCallCount, "DB should be updated via repository")

	t.Logf("PASS: Cache populated for NEW operation opId=%d", opId)
}

// TestStatusServiceSingleWriterPattern verifies that StatusService
// is the single writer for both DB and in-memory map.
// No duplicate manual writes to s.pendingOps should occur.
func TestStatusServiceSingleWriterPattern(t *testing.T) {
	pendingOps := make(map[bssci.SessionOpKey]*bssci.PendingOperation)
	var mu sync.RWMutex

	log := logger.NewNop()
	mockRepo := &mockPendingOperationRepositoryWithCallTracking{}
	statusSvc := NewStatusService(&pendingOps, &mu, mockRepo, log)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:          "single-writer-test",
			DbSessionID: 3001,
		},
	}

	ctx := context.Background()
	opId := int64(-300)

	op := &bssci.PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   opId,
		OperationType: "vmActivate",
		Endpoint:      []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0},
		Metadata:      map[string]interface{}{"macType": 1},
	}

	// Record operation - StatusService writes to BOTH map and DB
	err := statusSvc.RecordPendingOperation(ctx, session, opId, op, session.DbSessionID)
	require.NoError(t, err)

	// Verify single entry in map (no duplicate writes)
	mu.RLock()
	mapSize := len(pendingOps)
	mu.RUnlock()
	assert.Equal(t, 1, mapSize, "Should have exactly 1 entry from single write")

	// Verify single DB write
	assert.Equal(t, 1, mockRepo.createCallCount, "Should have exactly 1 DB write")

	// Calling RecordPendingOperation again should UPDATE, not duplicate
	op.Metadata["updated"] = true
	err = statusSvc.RecordPendingOperation(ctx, session, opId, op, session.DbSessionID)
	require.NoError(t, err)

	mu.RLock()
	mapSize = len(pendingOps)
	mu.RUnlock()
	assert.Equal(t, 1, mapSize, "Should still have 1 entry after update")
	assert.Equal(t, 2, mockRepo.createCallCount, "Should have 2 DB writes (initial + update)")

	t.Logf("PASS: Single writer pattern enforced")
}
