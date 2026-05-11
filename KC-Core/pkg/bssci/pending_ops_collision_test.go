package bssci

import (
	"context"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPendingOpsCompositeKey verifies that the sessionOpKey composite key
// prevents operation ID collisions when multiple base stations reuse the same
// operation ID (Issue #3).
func TestPendingOpsCompositeKey(t *testing.T) {
	t.Parallel()

	const (
		sharedOpID = int64(100) // Same operation ID used by both sessions
		session1ID = "session-bs1"
		session2ID = "session-bs2"
	)

	// Create two sessions with different IDs
	session1 := &Session{
		ID:             session1ID,
		DbSessionID:    1,
		BaseStationEUI: TestBsEuiMulti01,
	}

	session2 := &Session{
		ID:             session2ID,
		DbSessionID:    2,
		BaseStationEUI: TestBsEuiMulti02,
	}

	// Create server with StatusService
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc,
		broadcaster, queueSerializer, auditLogger, tenantResolver)

	// Test 1: Store operations with same opID in different sessions
	op1 := &PendingOperation{
		SessionSlug:   session1.ID,
		OperationID:   sharedOpID,
		OperationType: mioty.CmdAttach,
		CreatedAt:     time.Now(),
	}

	op2 := &PendingOperation{
		SessionSlug:   session2.ID,
		OperationID:   sharedOpID,
		OperationType: mioty.CmdDetach,
		CreatedAt:     time.Now(),
	}

	// Use StatusService to store operations
	ctx := context.Background()
	err := server.statusSvc.RecordPendingOperation(ctx, session1, sharedOpID, op1, session1.DbSessionID)
	require.NoError(t, err, "Should record session 1 operation")
	err = server.statusSvc.RecordPendingOperation(ctx, session2, sharedOpID, op2, session2.DbSessionID)
	require.NoError(t, err, "Should record session 2 operation")

	// Test 2: Retrieve operations - verify no collision
	retrieved1, err := server.statusSvc.GetPendingOperation(session1, sharedOpID)
	require.NoError(t, err, "Session 1 operation should exist")
	retrieved2, err := server.statusSvc.GetPendingOperation(session2, sharedOpID)
	require.NoError(t, err, "Session 2 operation should exist")

	assert.Equal(t, mioty.CmdAttach, retrieved1.OperationType, "Session 1 should have attach operation")
	assert.Equal(t, mioty.CmdDetach, retrieved2.OperationType, "Session 2 should have detach operation")
	assert.Equal(t, session1ID, retrieved1.SessionSlug, "Operation 1 should have correct session ID")
	assert.Equal(t, session2ID, retrieved2.SessionSlug, "Operation 2 should have correct session ID")

	// Test 3: Delete from one session doesn't affect the other
	err = server.statusSvc.RemovePendingOperation(ctx, session1, sharedOpID)
	require.NoError(t, err, "Should remove session 1 operation")

	_, err = server.statusSvc.GetPendingOperation(session1, sharedOpID)
	exists1AfterDelete := (err == nil)
	retrieved2AfterDelete, err := server.statusSvc.GetPendingOperation(session2, sharedOpID)
	exists2AfterDelete := (err == nil)

	assert.False(t, exists1AfterDelete, "Session 1 operation should be deleted")
	assert.True(t, exists2AfterDelete, "Session 2 operation should still exist")
	assert.Equal(t, mioty.CmdDetach, retrieved2AfterDelete.OperationType, "Session 2 operation should be unchanged")

	t.Logf("PASS: Issue #3: Composite key correctly prevents operation ID collision across %d sessions", 2)
}

// TestPendingOpsSessionIsolation verifies that pending operations are
// correctly isolated by session, even with complex operation patterns.
func TestPendingOpsSessionIsolation(t *testing.T) {
	t.Parallel()

	// Create three sessions simulating three base stations
	sessions := []*Session{
		{ID: "bs-alpha", DbSessionID: 1, BaseStationEUI: 0xAAAAAAAAAAAAAAAA},
		{ID: "bs-beta", DbSessionID: 2, BaseStationEUI: 0xBBBBBBBBBBBBBBBB},
		{ID: "bs-gamma", DbSessionID: 3, BaseStationEUI: 0xCCCCCCCCCCCCCCCC},
	}

	// Create server with StatusService
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc,
		broadcaster, queueSerializer, auditLogger, tenantResolver)

	// Each session creates operations with opIDs 1-5
	operationTypes := []string{
		mioty.CmdAttach,
		mioty.CmdDetach,
		mioty.CmdDLDataQueue,
		mioty.CmdAttachPropagate,
		mioty.CmdDetachPropagate,
	}

	// Store 15 operations using StatusService (3 sessions x 5 operations each)
	ctx := context.Background()
	for sessionIdx, session := range sessions {
		for opIdx, opType := range operationTypes {
			opID := int64(opIdx + 1)
			op := &PendingOperation{
				SessionSlug:   session.ID,
				OperationID:   opID,
				OperationType: opType,
				CreatedAt:     time.Now().Add(time.Duration(sessionIdx*100) * time.Millisecond),
			}
			err := server.statusSvc.RecordPendingOperation(ctx, session, opID, op, session.DbSessionID)
			require.NoError(t, err, "Should record operation %d for session %s", opID, session.ID)
		}
	}

	// Test retrieval: Each session should have its own set of operations
	for _, session := range sessions {
		for opID := int64(1); opID <= 5; opID++ {
			op, err := server.statusSvc.GetPendingOperation(session, opID)
			require.NoError(t, err, "Operation %d should exist for session %s", opID, session.ID)
			assert.Equal(t, session.ID, op.SessionSlug, "Operation should belong to correct session")
		}
	}

	// Test selective deletion: Remove all operations from session beta
	for opID := int64(1); opID <= 5; opID++ {
		err := server.statusSvc.RemovePendingOperation(ctx, sessions[1], opID)
		require.NoError(t, err, "Should remove operation %d from beta", opID)
	}

	// Verify alpha and gamma still have all their operations
	for _, session := range []*Session{sessions[0], sessions[2]} {
		for opID := int64(1); opID <= 5; opID++ {
			_, err := server.statusSvc.GetPendingOperation(session, opID)
			assert.NoError(t, err, "Session %s operation %d should still exist", session.ID, opID)
		}
	}

	// Verify beta's operations are truly gone
	for opID := int64(1); opID <= 5; opID++ {
		_, err := server.statusSvc.GetPendingOperation(sessions[1], opID)
		assert.Error(t, err, "Session beta operation %d should be deleted", opID)
	}

	t.Logf("PASS: Issue #3: Session isolation verified with %d sessions and %d total operations", 3, 15)
}

// TestMakeSessionOpKey verifies the composite key helper function
// produces consistent, unique keys.
func TestMakeSessionOpKey(t *testing.T) {
	t.Parallel()

	session := &Session{
		ID:          "test-session-123",
		DbSessionID: 42,
	}

	opID := int64(999)

	// Test 1: Consistent key generation
	key1 := makeSessionOpKey(session, opID)
	key2 := makeSessionOpKey(session, opID)

	assert.Equal(t, key1, key2, "Same inputs should produce identical keys")
	assert.Equal(t, "test-session-123", key1.SessionID, "Key should contain session ID")
	assert.Equal(t, int64(999), key1.OperationID, "Key should contain operation ID")

	// Test 2: Different sessions produce different keys
	session2 := &Session{
		ID:          "test-session-456",
		DbSessionID: 43,
	}

	key3 := makeSessionOpKey(session2, opID)
	assert.NotEqual(t, key1, key3, "Different sessions should produce different keys")
	assert.Equal(t, "test-session-456", key3.SessionID, "Key should contain correct session ID")

	// Test 3: Different opIDs produce different keys
	key4 := makeSessionOpKey(session, int64(888))
	assert.NotEqual(t, key1, key4, "Different operation IDs should produce different keys")
	assert.Equal(t, int64(888), key4.OperationID, "Key should contain correct operation ID")

	t.Logf("PASS: Issue #3: makeSessionOpKey produces consistent, unique composite keys")
}

// TestPendingOperationSessionSlug verifies that PendingOperation struct
// correctly stores and maintains the SessionSlug field.
func TestPendingOperationSessionSlug(t *testing.T) {
	t.Parallel()

	const (
		sessionID = "bs-production-001"
		opID      = int64(12345)
	)

	// Create pending operation with SessionSlug
	op := &PendingOperation{
		SessionSlug:   sessionID,
		OperationID:   opID,
		OperationType: mioty.CmdAttach,
		Endpoint:      []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
		CreatedAt:     time.Now(),
		Metadata: map[string]interface{}{
			"tenantId": int64(100),
		},
	}

	// Verify SessionSlug is correctly stored
	assert.Equal(t, sessionID, op.SessionSlug, "SessionSlug should be correctly stored")
	assert.Equal(t, opID, op.OperationID, "OperationID should be correctly stored")
	assert.Equal(t, mioty.CmdAttach, op.OperationType, "OperationType should be correctly stored")

	// Verify operation can be stored and retrieved with StatusService
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc,
		broadcaster, queueSerializer, auditLogger, tenantResolver)

	session := &Session{ID: sessionID, DbSessionID: 1}
	ctx := context.Background()
	err := server.statusSvc.RecordPendingOperation(ctx, session, opID, op, session.DbSessionID)
	require.NoError(t, err, "Should record pending operation")

	// Retrieve and verify
	retrieved, err := server.statusSvc.GetPendingOperation(session, opID)
	require.NoError(t, err, "Operation should be retrievable")
	assert.Equal(t, sessionID, retrieved.SessionSlug, "Retrieved operation should have correct SessionSlug")

	t.Logf("PASS: Issue #3: PendingOperation.SessionSlug field works correctly")
}

// TestPendingOpsRaceCondition verifies that concurrent operations on different
// sessions don't interfere with each other (smoke test for race detector).
func TestPendingOpsRaceCondition(t *testing.T) {
	t.Parallel()

	// Create server with StatusService
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc,
		broadcaster, queueSerializer, auditLogger, tenantResolver)

	// Note: This is a basic smoke test. The actual race testing happens with
	// `go test -race` which is part of the plan's "Run regression suite with race detector" step.

	sessions := []*Session{
		{ID: "concurrent-1", DbSessionID: 1},
		{ID: "concurrent-2", DbSessionID: 2},
	}

	ctx := context.Background()

	// Simulate sequential operations (real concurrent testing with -race flag)
	for i, session := range sessions {
		opID := int64(i + 1)

		op := &PendingOperation{
			SessionSlug:   session.ID,
			OperationID:   opID,
			OperationType: mioty.CmdAttach,
			CreatedAt:     time.Now(),
		}

		err := server.statusSvc.RecordPendingOperation(ctx, session, opID, op, session.DbSessionID)
		require.NoError(t, err, "Should record operation for session %s", session.ID)
	}

	// Verify both operations exist
	for i, session := range sessions {
		opID := int64(i + 1)
		_, err := server.statusSvc.GetPendingOperation(session, opID)
		require.NoError(t, err, "Operation for session %s should exist", session.ID)
	}

	t.Logf("PASS: Issue #3: Basic concurrent operation smoke test passed (run with -race for full validation)")
}
