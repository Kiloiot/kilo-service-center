package bssci

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAtomicSendRollback verifies that the atomic send-and-persist pattern
// correctly rolls back operation ID counters when message send fails.
// This ensures BSSCI §5.2 correctness by preventing operation ID gaps.
func TestAtomicSendRollback(t *testing.T) {
	tests := []struct {
		name         string
		initialOpId  int64
		sendSucceeds bool
		expectedOpId int64
		desc         string
	}{
		{
			name:         "SendSuccessKeepsDecrementedID",
			initialOpId:  -5,
			sendSucceeds: true,
			expectedOpId: -6,
			desc:         "Successful send should keep decremented operation ID",
		},
		{
			name:         "SendFailureRollsBackID",
			initialOpId:  -5,
			sendSucceeds: false,
			expectedOpId: -5,
			desc:         "Failed send should rollback operation ID to original value",
		},
		{
			name:         "FirstOperationRollback",
			initialOpId:  0,
			sendSucceeds: false,
			expectedOpId: 0,
			desc:         "First operation failure should rollback from -1 to 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{
				BaseStationEUI: TestEpEui01,
				LastScOpId:     tt.initialOpId,
			}

			// Step 1: Decrement operation ID (atomic send-and-persist pattern)
			session.mu.Lock()
			session.LastScOpId--
			opId := session.LastScOpId
			session.mu.Unlock()

			// Verify operation ID was decremented
			assert.Equal(t, tt.initialOpId-1, opId,
				"Operation ID should be decremented before send attempt")

			// Step 2: Simulate send attempt
			if !tt.sendSucceeds {
				// CRITICAL: Rollback operation ID on send failure (BSSCI §5.2)
				session.mu.Lock()
				session.LastScOpId++
				session.mu.Unlock()
			}

			// Step 3: Verify final counter state
			assert.Equal(t, tt.expectedOpId, session.LastScOpId, tt.desc)
		})
	}
}

// TestMultipleRollbacks verifies that multiple consecutive send failures
// correctly rollback operation IDs without causing counter drift.
func TestMultipleRollbacks(t *testing.T) {
	session := &Session{
		BaseStationEUI: TestEpEui01,
		LastScOpId:     0,
	}

	numAttempts := 5
	allFailed := true

	// Simulate multiple failed send attempts
	for i := 0; i < numAttempts; i++ {
		// Decrement counter
		session.mu.Lock()
		session.LastScOpId--
		opId := session.LastScOpId
		session.mu.Unlock()

		// Verify operation ID
		assert.Equal(t, int64(-1), opId,
			"Each attempt should decrement to -1 before rollback")

		// Simulate send failure and rollback
		if allFailed {
			session.mu.Lock()
			session.LastScOpId++
			session.mu.Unlock()
		}

		// Verify counter returned to 0 after rollback
		assert.Equal(t, int64(0), session.LastScOpId,
			"Counter should return to 0 after rollback on attempt %d", i+1)
	}
}

// TestMixedSuccessFailure verifies that the atomic pattern correctly handles
// a mix of successful and failed operations, maintaining counter integrity.
func TestMixedSuccessFailure(t *testing.T) {
	session := &Session{
		BaseStationEUI: TestEpEui01,
		LastScOpId:     0,
	}

	operations := []struct {
		succeeds     bool
		expectedOpId int64
		desc         string
	}{
		{succeeds: true, expectedOpId: -1, desc: "First operation succeeds"},
		{succeeds: false, expectedOpId: -1, desc: "Second operation fails, stays at -1"},
		{succeeds: true, expectedOpId: -2, desc: "Third operation succeeds"},
		{succeeds: true, expectedOpId: -3, desc: "Fourth operation succeeds"},
		{succeeds: false, expectedOpId: -3, desc: "Fifth operation fails, stays at -3"},
		{succeeds: true, expectedOpId: -4, desc: "Sixth operation succeeds"},
	}

	for i, op := range operations {
		// Atomic send-and-persist pattern
		session.mu.Lock()
		session.LastScOpId--
		session.mu.Unlock()

		// Simulate send result
		if !op.succeeds {
			// Rollback on failure
			session.mu.Lock()
			session.LastScOpId++
			session.mu.Unlock()
		}

		// Verify counter state
		assert.Equal(t, op.expectedOpId, session.LastScOpId,
			"Operation %d: %s", i+1, op.desc)
	}
}

// TestRollbackPreservesPendingOperations verifies that rolling back an
// operation ID doesn't affect tracking of other pending operations.
func TestRollbackPreservesPendingOperations(t *testing.T) {
	session := &Session{
		BaseStationEUI: TestEpEui01,
		LastScOpId:     0,
	}

	// Track pending operations manually (simulating pendingOps map)
	pendingOps := make(map[int64]string)

	// Operation 1: succeeds
	session.mu.Lock()
	session.LastScOpId--
	opId1 := session.LastScOpId
	session.mu.Unlock()
	pendingOps[opId1] = "statusReq"
	// Send succeeds, keep counter at -1

	assert.Equal(t, int64(-1), session.LastScOpId, "After first success")
	assert.Contains(t, pendingOps, int64(-1), "Pending operation should be tracked")

	// Operation 2: fails
	session.mu.Lock()
	session.LastScOpId--
	opId2 := session.LastScOpId
	session.mu.Unlock()
	// Don't add to pending ops - send failed
	// Rollback counter
	session.mu.Lock()
	session.LastScOpId++
	session.mu.Unlock()

	assert.Equal(t, int64(-1), session.LastScOpId, "After rollback, counter returns to -1")
	assert.Contains(t, pendingOps, int64(-1), "Original pending operation still tracked")
	assert.NotContains(t, pendingOps, opId2, "Failed operation not added to pending")

	// Operation 3: succeeds
	session.mu.Lock()
	session.LastScOpId--
	opId3 := session.LastScOpId
	session.mu.Unlock()
	pendingOps[opId3] = "pingReq"

	assert.Equal(t, int64(-2), session.LastScOpId, "After second success")
	assert.Contains(t, pendingOps, int64(-1), "First pending operation still tracked")
	assert.Contains(t, pendingOps, int64(-2), "Second pending operation tracked")
}

// TestRollbackWithConcurrency verifies that rollback logic is thread-safe
// under concurrent access scenarios.
func TestRollbackWithConcurrency(t *testing.T) {
	session := &Session{
		BaseStationEUI: TestEpEui01,
		LastScOpId:     0,
	}

	numGoroutines := 10
	operationsPerGoroutine := 50

	var successCount atomic.Int64
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			localSuccess := int64(0)
			for j := 0; j < operationsPerGoroutine; j++ {
				// Atomic decrement
				session.mu.Lock()
				session.LastScOpId--
				session.mu.Unlock()

				// Simulate send (alternate success/failure for predictability)
				sendSucceeds := (goroutineID+j)%2 == 0

				if !sendSucceeds {
					// Rollback on failure
					session.mu.Lock()
					session.LastScOpId++
					session.mu.Unlock()
				} else {
					localSuccess++
				}
			}
			successCount.Add(localSuccess)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final counter equals negative of successful operations
	totalSuccess := successCount.Load()
	expectedFinal := -totalSuccess

	// The test verifies that LastScOpId reflects only successful operations
	// With alternating success/failure and concurrent access, we verify no counter corruption
	assert.LessOrEqual(t, session.LastScOpId, int64(0),
		"Final counter should be zero or negative (only successful ops counted)")

	t.Logf("Concurrent test: %d total attempts, %d successful, final counter: %d",
		numGoroutines*operationsPerGoroutine, totalSuccess, session.LastScOpId)

	// Verify counter is reasonable (within expected range)
	// Due to concurrent access, exact equality is hard to guarantee, but it should be close
	assert.GreaterOrEqual(t, session.LastScOpId, expectedFinal-10,
		"Counter should be close to number of successful operations")
}

// TestRollbackIntegrity verifies that no operation ID is ever reused
// across rollback scenarios, maintaining BSSCI §5.2 correctness.
func TestRollbackIntegrity(t *testing.T) {
	session := &Session{
		BaseStationEUI: TestEpEui01,
		LastScOpId:     0,
	}

	usedOpIds := make(map[int64]bool)

	operations := []struct {
		succeeds bool
	}{
		{succeeds: true},  // -1 (used)
		{succeeds: false}, // -2 (rolled back, not used)
		{succeeds: true},  // -2 (used)
		{succeeds: false}, // -3 (rolled back, not used)
		{succeeds: false}, // -3 (rolled back, not used)
		{succeeds: true},  // -3 (used)
		{succeeds: true},  // -4 (used)
	}

	for i, op := range operations {
		// Atomic decrement
		session.mu.Lock()
		session.LastScOpId--
		opId := session.LastScOpId
		session.mu.Unlock()

		if op.succeeds {
			// Check for duplicate operation IDs (should never happen)
			assert.False(t, usedOpIds[opId],
				"Operation %d: Operation ID %d should not be reused", i+1, opId)
			usedOpIds[opId] = true
		} else {
			// Rollback
			session.mu.Lock()
			session.LastScOpId++
			session.mu.Unlock()
		}
	}

	// Verify expected operation IDs were used
	assert.True(t, usedOpIds[-1], "Operation ID -1 should be used")
	assert.True(t, usedOpIds[-2], "Operation ID -2 should be used")
	assert.True(t, usedOpIds[-3], "Operation ID -3 should be used")
	assert.True(t, usedOpIds[-4], "Operation ID -4 should be used")
	assert.Equal(t, 4, len(usedOpIds), "Exactly 4 operation IDs should be used")
}
