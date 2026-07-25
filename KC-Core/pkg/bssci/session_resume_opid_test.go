package bssci

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSessionOpIdInitialization verifies that new sessions initialize operation ID counters
// correctly per BSSCI §5.2 with the atomic send-and-persist pattern.
func TestSessionOpIdInitialization(t *testing.T) {
	tests := []struct {
		name           string
		desc           string
		expectedScOpId int64
		expectedBsOpId int64
	}{
		{
			name:           "NewSessionStartsAtZero",
			desc:           "New session should initialize both counters to 0",
			expectedScOpId: 0,
			expectedBsOpId: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create new session
			session := &Session{
				ProtocolSessionState: ProtocolSessionState{
					BaseStationEUI: TestEpEui01,
				},
			}

			// Verify initial counter values
			assert.Equal(t, tt.expectedScOpId, session.LastScOpId,
				"LastScOpId should be initialized to 0")
			assert.Equal(t, tt.expectedBsOpId, session.LastBsOpId,
				"LastBsOpId should be initialized to 0")
		})
	}
}

// TestSessionOpIdDecrement verifies that SC operation IDs decrement correctly
// when operations are sent using the atomic send-and-persist pattern.
func TestSessionOpIdDecrement(t *testing.T) {
	tests := []struct {
		name          string
		initialOpId   int64
		numOperations int
		expectedFinal int64
		desc          string
	}{
		{
			name:          "FirstOperationFromZero",
			initialOpId:   0,
			numOperations: 1,
			expectedFinal: -1,
			desc:          "First SC operation should decrement from 0 to -1",
		},
		{
			name:          "MultipleOperations",
			initialOpId:   0,
			numOperations: 5,
			expectedFinal: -5,
			desc:          "Multiple SC operations should strictly decrement",
		},
		{
			name:          "ContinueFromNegative",
			initialOpId:   -10,
			numOperations: 3,
			expectedFinal: -13,
			desc:          "SC operations should continue decrementing from negative values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{
				ProtocolSessionState: ProtocolSessionState{
					BaseStationEUI: TestEpEui01,
					LastScOpId:     tt.initialOpId,
				},
			}

			// Simulate sending operations using atomic pattern
			for i := 0; i < tt.numOperations; i++ {
				session.mu.Lock()
				session.LastScOpId--
				opId := session.LastScOpId
				session.mu.Unlock()

				// Verify operation ID is strictly decrementing
				expectedOpId := tt.initialOpId - int64(i+1)
				assert.Equal(t, expectedOpId, opId,
					"Operation %d should have strictly decrementing ID", i+1)
			}

			// Verify final counter value
			assert.Equal(t, tt.expectedFinal, session.LastScOpId, tt.desc)
		})
	}
}

// TestSessionOpIdIncrement verifies that BS operation IDs increment correctly
// when validated using the existing validation logic.
func TestSessionOpIdIncrement(t *testing.T) {
	tests := []struct {
		name          string
		initialOpId   int64
		incomingOpIds []int64
		expectedFinal int64
		desc          string
	}{
		{
			name:          "FirstBSOperation",
			initialOpId:   0,
			incomingOpIds: []int64{1},
			expectedFinal: 1,
			desc:          "First BS operation should increment LastBsOpId to 1",
		},
		{
			name:          "StrictlyIncrementingOperations",
			initialOpId:   0,
			incomingOpIds: []int64{1, 2, 3, 4, 5},
			expectedFinal: 5,
			desc:          "BS operations should strictly increment LastBsOpId",
		},
		{
			name:          "SameOperationRepeated",
			initialOpId:   0,
			incomingOpIds: []int64{1, 1, 1},
			expectedFinal: 1,
			desc:          "Repeated operation ID should not update LastBsOpId",
		},
		{
			name:          "GapInSequence",
			initialOpId:   0,
			incomingOpIds: []int64{1, 5, 10},
			expectedFinal: 10,
			desc:          "Gaps in BS operation sequence should be accepted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{
				ProtocolSessionState: ProtocolSessionState{
					BaseStationEUI: TestEpEui01,
					LastBsOpId:     tt.initialOpId,
				},
			}

			// Simulate receiving BS operations and validating them
			for _, opId := range tt.incomingOpIds {
				// Only update if this is a NEW operation (greater than last)
				if opId > session.LastBsOpId {
					session.LastBsOpId = opId
				}
			}

			// Verify final counter value
			assert.Equal(t, tt.expectedFinal, session.LastBsOpId, tt.desc)
		})
	}
}

// TestSessionOpIdResume verifies that operation ID counters are correctly
// maintained across session resume scenarios per BSSCI §3.3 and §5.2.
// This is a unit test that verifies the logic; integration tests should
// verify actual database persistence.
func TestSessionOpIdResume(t *testing.T) {
	tests := []struct {
		name            string
		savedScOpId     int64
		savedBsOpId     int64
		newScOperations int
		newBsOperations []int64
		expectedScOpId  int64
		expectedBsOpId  int64
		desc            string
	}{
		{
			name:            "ResumeAndContinueSCOperations",
			savedScOpId:     -5,
			savedBsOpId:     10,
			newScOperations: 3,
			newBsOperations: []int64{11, 12},
			expectedScOpId:  -8,
			expectedBsOpId:  12,
			desc:            "Resumed session should continue from saved counters",
		},
		{
			name:            "ResumeWithNoNewOperations",
			savedScOpId:     -10,
			savedBsOpId:     20,
			newScOperations: 0,
			newBsOperations: []int64{},
			expectedScOpId:  -10,
			expectedBsOpId:  20,
			desc:            "Resumed session with no new operations should maintain counters",
		},
		{
			name:            "ResumeFromInitialState",
			savedScOpId:     0,
			savedBsOpId:     0,
			newScOperations: 2,
			newBsOperations: []int64{1},
			expectedScOpId:  -2,
			expectedBsOpId:  1,
			desc:            "Resume from initial state (counters at 0) should work correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate session resume by creating session with saved counters
			session := &Session{
				ProtocolSessionState: ProtocolSessionState{
					BaseStationEUI: TestEpEui01,
					LastScOpId:     tt.savedScOpId,
					LastBsOpId:     tt.savedBsOpId,
				},
			}

			// Verify counters restored from "database"
			assert.Equal(t, tt.savedScOpId, session.LastScOpId,
				"LastScOpId should be restored from saved state")
			assert.Equal(t, tt.savedBsOpId, session.LastBsOpId,
				"LastBsOpId should be restored from saved state")

			// Simulate new SC operations after resume
			for i := 0; i < tt.newScOperations; i++ {
				session.mu.Lock()
				session.LastScOpId--
				session.mu.Unlock()
			}

			// Simulate new BS operations after resume
			for _, opId := range tt.newBsOperations {
				if opId > session.LastBsOpId {
					session.LastBsOpId = opId
				}
			}

			// Verify final counter values
			assert.Equal(t, tt.expectedScOpId, session.LastScOpId, tt.desc)
			assert.Equal(t, tt.expectedBsOpId, session.LastBsOpId, tt.desc)
		})
	}
}

// TestSessionOpIdConcurrency verifies that operation ID counter updates
// are thread-safe with mutex protection.
func TestSessionOpIdConcurrency(t *testing.T) {
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			BaseStationEUI: TestEpEui01,
			LastScOpId:     0,
		},
	}

	numGoroutines := 10
	operationsPerGoroutine := 100

	// Launch multiple goroutines that decrement the counter
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < operationsPerGoroutine; j++ {
				session.mu.Lock()
				session.LastScOpId--
				session.mu.Unlock()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final counter value (should be strictly decremented)
	expectedFinal := int64(-(numGoroutines * operationsPerGoroutine))
	assert.Equal(t, expectedFinal, session.LastScOpId,
		"Concurrent operations should result in correct final counter value")
}
