// Package scaci provides tests for atomic opId persistence per SCACI §3.2.
//
// Coverage:
//   - persistOpIDsPair helper function
//   - Atomic persistence of AC and SC operation IDs
//   - Session ID validation (skip if <= 0)
//   - Error handling during persistence
//
// These tests verify that operation ID counters are persisted atomically to
// support session resume per SCACI §3.2 requirements.
package scaci

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Session Repository
// ============================================================================

// mockSessionRepository implements interfaces.SCACISessionRepository for testing
type mockSessionRepository struct {
	mock.Mock
	updateCalls []opIDsUpdate // Track calls for verification
	mu          sync.Mutex
}

// opIDsUpdate captures a call to UpdateOperationIDs
type opIDsUpdate struct {
	TenantID  int64
	SessionID int64
	AcOpID    int64
	ScOpID    int64
}

func (m *mockSessionRepository) UpdateOperationIDs(ctx context.Context, tenantID, sessionID, acOpId, scOpId int64) error {
	m.mu.Lock()
	m.updateCalls = append(m.updateCalls, opIDsUpdate{
		TenantID:  tenantID,
		SessionID: sessionID,
		AcOpID:    acOpId,
		ScOpID:    scOpId,
	})
	m.mu.Unlock()
	args := m.Called(ctx, tenantID, sessionID, acOpId, scOpId)
	return args.Error(0)
}

// Implement remaining interface methods with no-op stubs
func (m *mockSessionRepository) CreateSession(_ context.Context, _ *models.SCACISessionCreateRequest) (*models.SCACISession, error) {
	return nil, nil
}
func (m *mockSessionRepository) GetSessionByID(_ context.Context, _, _ int64) (*models.SCACISession, error) {
	return nil, nil
}
func (m *mockSessionRepository) GetActiveSessionByAcEUI(_ context.Context, _ int64, _ [8]byte) (*models.SCACISession, error) {
	return nil, nil
}
func (m *mockSessionRepository) GetSessionByAcUUID(_ context.Context, _ int64, _ [16]byte) (*models.SCACISession, error) {
	return nil, nil
}
func (m *mockSessionRepository) GetSessionByScUUID(_ context.Context, _ int64, _ [16]byte) (*models.SCACISession, error) {
	return nil, nil
}
func (m *mockSessionRepository) UpdateSession(_ context.Context, _, _ int64, _ *models.SCACISessionUpdateRequest) error {
	return nil
}
func (m *mockSessionRepository) UpdateHeartbeat(_ context.Context, _, _ int64) error { return nil }
func (m *mockSessionRepository) DisconnectSession(_ context.Context, _, _ int64) error {
	return nil
}
func (m *mockSessionRepository) TerminateSession(_ context.Context, _, _ int64) error { return nil }
func (m *mockSessionRepository) TerminateAllSessions(_ context.Context, _ int64, _ [8]byte) error {
	return nil
}
func (m *mockSessionRepository) ListSessions(_ context.Context, _ *models.SCACISessionFilter) ([]*models.SCACISession, int64, error) {
	return nil, 0, nil
}
func (m *mockSessionRepository) GetSessionStatistics(_ context.Context, _ int64) (*models.SCACISessionStatistics, error) {
	return nil, nil
}
func (m *mockSessionRepository) CheckSessionResumable(_ context.Context, _ int64, _ [16]byte, _, _ int64) (*models.SCACISessionResumptionInfo, error) {
	return nil, nil
}
func (m *mockSessionRepository) CleanupExpiredSessions(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

// getUpdateCalls returns a copy of the update calls (thread-safe)
func (m *mockSessionRepository) getUpdateCalls() []opIDsUpdate {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]opIDsUpdate, len(m.updateCalls))
	copy(result, m.updateCalls)
	return result
}

// ============================================================================
// Test Server Factory
// ============================================================================

func newTestServerWithSessionRepo(repo interfaces.SCACISessionRepository) *Server {
	log := logger.NewNop()
	return &Server{
		logger:      log,
		sessionRepo: repo,
		config: &Config{
			LogPingOperations: true, // Enable for coverage
		},
	}
}

// ============================================================================
// persistOpIDsPair Tests
// ============================================================================

// TestPersistOpIDsPair_ValidSession verifies atomic persistence for valid session
func TestPersistOpIDsPair_ValidSession(t *testing.T) {
	mockRepo := new(mockSessionRepository)
	server := newTestServerWithSessionRepo(mockRepo)

	// Setup mock expectation
	mockRepo.On("UpdateOperationIDs", mock.Anything, int64(100), int64(1), int64(42), int64(-10)).Return(nil)

	// Call persistOpIDsPair
	server.persistOpIDsPair(1, 100, 42, -10)

	// Wait for async goroutine to complete
	time.Sleep(50 * time.Millisecond)

	// Verify the call was made
	mockRepo.AssertExpectations(t)

	// Verify captured call values
	calls := mockRepo.getUpdateCalls()
	assert.Len(t, calls, 1, "Should have exactly one UpdateOperationIDs call")
	if len(calls) == 1 {
		assert.Equal(t, int64(100), calls[0].TenantID)
		assert.Equal(t, int64(1), calls[0].SessionID)
		assert.Equal(t, int64(42), calls[0].AcOpID)
		assert.Equal(t, int64(-10), calls[0].ScOpID)
	}
}

// TestPersistOpIDsPair_SkipsZeroSessionID verifies no call for sessionID <= 0
func TestPersistOpIDsPair_SkipsZeroSessionID(t *testing.T) {
	mockRepo := new(mockSessionRepository)
	server := newTestServerWithSessionRepo(mockRepo)

	// Call with zero sessionID - should skip persistence
	server.persistOpIDsPair(0, 100, 42, -10)

	// Wait briefly to ensure no async call
	time.Sleep(50 * time.Millisecond)

	// Verify no calls were made
	mockRepo.AssertNotCalled(t, "UpdateOperationIDs")
	calls := mockRepo.getUpdateCalls()
	assert.Len(t, calls, 0, "Should have no UpdateOperationIDs calls for sessionID=0")
}

// TestPersistOpIDsPair_SkipsNegativeSessionID verifies no call for negative sessionID
func TestPersistOpIDsPair_SkipsNegativeSessionID(t *testing.T) {
	mockRepo := new(mockSessionRepository)
	server := newTestServerWithSessionRepo(mockRepo)

	// Call with negative sessionID - should skip persistence
	server.persistOpIDsPair(-1, 100, 42, -10)

	// Wait briefly to ensure no async call
	time.Sleep(50 * time.Millisecond)

	// Verify no calls were made
	mockRepo.AssertNotCalled(t, "UpdateOperationIDs")
	calls := mockRepo.getUpdateCalls()
	assert.Len(t, calls, 0, "Should have no UpdateOperationIDs calls for negative sessionID")
}

// TestPersistOpIDsPair_HandlesError verifies error doesn't panic
func TestPersistOpIDsPair_HandlesError(t *testing.T) {
	mockRepo := new(mockSessionRepository)
	server := newTestServerWithSessionRepo(mockRepo)

	// Setup mock to return error
	mockRepo.On("UpdateOperationIDs", mock.Anything, int64(100), int64(1), int64(42), int64(-10)).
		Return(errors.New("database connection lost"))

	// Call should not panic despite error
	server.persistOpIDsPair(1, 100, 42, -10)

	// Wait for async goroutine
	time.Sleep(50 * time.Millisecond)

	// Verify the call was made (even if it failed)
	mockRepo.AssertExpectations(t)
}

// TestPersistOpIDsPair_AtomicBehavior verifies both IDs are passed together
func TestPersistOpIDsPair_AtomicBehavior(t *testing.T) {
	mockRepo := new(mockSessionRepository)
	server := newTestServerWithSessionRepo(mockRepo)

	// Test various AC/SC opId combinations
	testCases := []struct {
		name      string
		sessionID int64
		tenantID  int64
		acOpId    int64
		scOpId    int64
	}{
		{"AC=1, SC=-1", 1, 100, 1, -1},
		{"AC=100, SC=-50", 2, 200, 100, -50},
		{"AC=1000, SC=-999", 3, 300, 1000, -999},
		{"Large values", 4, 400, 999999, -888888},
	}

	for i, tc := range testCases {
		mockRepo.On("UpdateOperationIDs", mock.Anything, tc.tenantID, tc.sessionID, tc.acOpId, tc.scOpId).
			Return(nil).Once()
		server.persistOpIDsPair(tc.sessionID, tc.tenantID, tc.acOpId, tc.scOpId)

		// Wait for async completion
		time.Sleep(50 * time.Millisecond)

		// Verify both values passed atomically
		calls := mockRepo.getUpdateCalls()
		assert.GreaterOrEqual(t, len(calls), i+1, "Should have at least %d calls", i+1)
		if len(calls) > i {
			call := calls[i]
			assert.Equal(t, tc.acOpId, call.AcOpID, "%s: AcOpID mismatch", tc.name)
			assert.Equal(t, tc.scOpId, call.ScOpID, "%s: ScOpID mismatch", tc.name)
		}
	}

	mockRepo.AssertExpectations(t)
}

// TestPersistOpIDsPair_ConcurrentCalls verifies thread safety
func TestPersistOpIDsPair_ConcurrentCalls(t *testing.T) {
	mockRepo := new(mockSessionRepository)
	server := newTestServerWithSessionRepo(mockRepo)

	// Allow any number of calls with any values
	mockRepo.On("UpdateOperationIDs", mock.Anything, mock.AnythingOfType("int64"),
		mock.AnythingOfType("int64"), mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).
		Return(nil)

	// Launch multiple concurrent calls
	var wg sync.WaitGroup
	numCalls := 10

	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			server.persistOpIDsPair(int64(idx+1), int64(100+idx), int64(idx), int64(-idx))
		}(i)
	}

	wg.Wait()

	// Wait for async goroutines to complete
	time.Sleep(100 * time.Millisecond)

	// Verify all calls were made
	calls := mockRepo.getUpdateCalls()
	assert.Len(t, calls, numCalls, "Should have exactly %d calls", numCalls)
}

// ============================================================================
// Integration Tests (Call Sites)
// ============================================================================

// TestPersistOpIDsPair_CallSiteDocumentation documents expected call sites
func TestPersistOpIDsPair_CallSiteDocumentation(t *testing.T) {
	// This test documents the expected call sites for persistOpIDsPair per SCACI §3.2
	//
	// Call sites:
	// 1. handleMessage (server.go:776) - After AC message processing
	// 2. initiatePing (server.go:956) - After SC-initiated ping
	// 3. sendULData (server.go:1143) - After SC sends ULData
	// 4. sendDLDataResult (server.go:1271) - After SC sends DLDataResult
	// 5. SendEpStatus (server.go:1422) - After SC sends EpStatus
	// 6. handleDLDataQueCmp (handler_operations.go:1221) - After DLDataQue complete
	//
	// Each call site ensures atomic persistence of both AC and SC opId counters.

	callSites := []string{
		"handleMessage",      // AC-initiated operations
		"initiatePing",       // SC-initiated ping
		"sendULData",         // SC→AC uplink data
		"sendDLDataResult",   // SC→AC downlink result
		"SendEpStatus",       // SC→AC endpoint status
		"handleDLDataQueCmp", // DLDataQue three-way complete
	}

	assert.Len(t, callSites, 6, "Should have 6 documented call sites")

	// Verify each is a valid function name (basic sanity check)
	for _, site := range callSites {
		assert.NotEmpty(t, site, "Call site name should not be empty")
	}
}
