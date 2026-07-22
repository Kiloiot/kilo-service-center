// Package scaci error_recorder_test.go tests SCACI §3.14 error handling.
package scaci

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// ============================================================================
// Mocks for error recorder tests
// ============================================================================

// mockOperationRepoForErrors implements SCACIOperationRepository for error recorder tests
type mockOperationRepoForErrors struct {
	mock.Mock
}

func (m *mockOperationRepoForErrors) RecordOperation(_ context.Context, _ *models.SCACIOperationRequest) (*models.SCACIOperation, error) {
	return nil, nil
}

func (m *mockOperationRepoForErrors) UpdateOperationState(_ context.Context, _, _ int64, _ models.OperationState, _ map[string]interface{}) error {
	return nil
}

func (m *mockOperationRepoForErrors) UpdateOperationStateWithError(ctx context.Context, sessionID int64, opId int64, state models.OperationState, errorCode int, errorToken string, errorMessage string, responseData map[string]interface{}) error {
	args := m.Called(ctx, sessionID, opId, state, errorCode, errorToken, errorMessage, responseData)
	return args.Error(0)
}

func (m *mockOperationRepoForErrors) GetOperationByOpID(_ context.Context, _, _ int64) (*models.SCACIOperation, error) {
	return nil, nil
}

func (m *mockOperationRepoForErrors) GetPendingOperations(_ context.Context, _ int64) ([]*models.SCACIOperation, error) {
	return nil, nil
}

func (m *mockOperationRepoForErrors) GetRecentOperations(_ context.Context, _ int64, _ int) ([]*models.SCACIOperation, error) {
	return nil, nil
}

func (m *mockOperationRepoForErrors) CleanupCompletedOperations(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

func (m *mockOperationRepoForErrors) GetTenantOperationSummary(_ context.Context, _ int64, _ int) (*models.SCACIOperationSummary, error) {
	return nil, nil
}

func (m *mockOperationRepoForErrors) CompleteFailedOperation(ctx context.Context, sessionID int64, opId int64, responseData map[string]interface{}) error {
	args := m.Called(ctx, sessionID, opId, responseData)
	return args.Error(0)
}

// mockEventStoreForErrors implements SystemEventStore for error recorder tests
type mockEventStoreForErrors struct {
	mock.Mock
}

func (m *mockEventStoreForErrors) RecordSCACIError(ctx context.Context, tenantID, sessionID int64, command string, opId int64, posixCode int, message string) error {
	args := m.Called(ctx, tenantID, sessionID, command, opId, posixCode, message)
	return args.Error(0)
}

func (m *mockEventStoreForErrors) CreateEvent(_ context.Context, _ *models.SystemEvent) error {
	return nil
}

func (m *mockEventStoreForErrors) GetEvents(_ context.Context, _ interfaces.SystemEventFilter) ([]*models.SystemEvent, error) {
	return nil, nil
}

func (m *mockEventStoreForErrors) GetActiveAlerts(_ context.Context, _ interfaces.AlertFilter) ([]*models.SystemEvent, error) {
	return nil, nil
}

func (m *mockEventStoreForErrors) GetEventStats(_ context.Context, _ string, _ time.Time) (*models.SystemEventStats, error) {
	return nil, nil
}

func (m *mockEventStoreForErrors) CountEvents(_ context.Context, _ interfaces.SystemEventFilter) (int64, error) {
	return 0, nil
}
func (m *mockEventStoreForErrors) CountActiveAlerts(_ context.Context, _ interfaces.AlertFilter) (int64, error) {
	return 0, nil
}

// ============================================================================
// RecordOutboundError Tests (Fix 1: defaultCode fallback)
// ============================================================================

// TestRecordOutboundError_UsesDefaultCodeWhenCatalogCodeIsZero validates Fix 1:
// When the error catalog returns POSIXCode=0 for an unknown token, the defaultCode
// parameter should be used instead to ensure wire never emits code=0.
func TestRecordOutboundError_UsesDefaultCodeWhenCatalogCodeIsZero(t *testing.T) {
	mockRepo := &mockOperationRepoForErrors{}
	mockEvents := &mockEventStoreForErrors{}
	log := testLogger()

	recorder := NewErrorRecorder(mockRepo, mockEvents, log)

	session := &Session{
		ID:       100,
		TenantID: 42,
	}

	// Use an unknown token that will return POSIXCode=0 from catalog
	unknownToken := "unknown.error.token.not.in.catalog"
	defaultCode := POSIX_EINVAL // 22

	// Expect the repo to be called with defaultCode (22), not 0
	mockRepo.On("UpdateOperationStateWithError",
		mock.Anything, // ctx
		int64(100),    // sessionID
		int64(1),      // opId
		models.OperationStateFailed,
		defaultCode,   // errorCode should be defaultCode, not 0
		unknownToken,  // errorToken
		mock.Anything, // message from catalog (will be empty for unknown)
		mock.Anything, // responseData
	).Return(nil)

	mockEvents.On("RecordSCACIError",
		mock.Anything,
		int64(42),
		int64(100),
		CmdError,
		int64(1),
		defaultCode,
		mock.Anything,
	).Return(nil)

	posixCode, _, err := recorder.RecordOutboundError(
		testutil.TestContext(),
		session,
		1,            // opId
		CmdError,     // command
		unknownToken, // errorToken
		defaultCode,  // defaultCode - should be used when catalog returns 0
		"test detail",
	)

	assert.NoError(t, err)
	assert.Equal(t, defaultCode, posixCode, "should use defaultCode when catalog POSIXCode=0")
	mockRepo.AssertExpectations(t)
}

// TestRecordOutboundError_UsesCatalogCodeWhenNonZero verifies that when the catalog
// has a non-zero POSIX code, it is used instead of defaultCode.
func TestRecordOutboundError_UsesCatalogCodeWhenNonZero(t *testing.T) {
	mockRepo := &mockOperationRepoForErrors{}
	mockEvents := &mockEventStoreForErrors{}
	log := testLogger()

	recorder := NewErrorRecorder(mockRepo, mockEvents, log)

	session := &Session{
		ID:       100,
		TenantID: 42,
	}

	// Use a known token with non-zero POSIX code (POSIX_ENOTSUP = 95)
	knownToken := ErrMajorVersionUnsupported

	// Get the expected code from catalog
	errDef := GetErrorDefinition(knownToken)
	expectedCode := errDef.POSIXCode
	assert.Equal(t, POSIX_ENOTSUP, expectedCode, "ErrMajorVersionUnsupported should have POSIX_ENOTSUP in catalog")

	// Expect the repo to be called with catalog code, not defaultCode
	mockRepo.On("UpdateOperationStateWithError",
		mock.Anything,
		int64(100),
		int64(1),
		models.OperationStateFailed,
		expectedCode, // Should use catalog code (95, not 22)
		knownToken,
		mock.Anything,
		mock.Anything,
	).Return(nil)

	mockEvents.On("RecordSCACIError",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		expectedCode,
		mock.Anything,
	).Return(nil)

	posixCode, _, err := recorder.RecordOutboundError(
		testutil.TestContext(),
		session,
		1,
		CmdError,
		knownToken,
		POSIX_EINVAL, // defaultCode - should NOT be used since catalog has 95
		"test detail",
	)

	assert.NoError(t, err)
	assert.Equal(t, expectedCode, posixCode, "should use catalog POSIXCode when non-zero")
	assert.NotEqual(t, POSIX_EINVAL, posixCode, "should NOT use defaultCode when catalog has non-zero code")
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// CompleteErrorHandshake Tests (Fix 2: preserve state='failed')
// ============================================================================

// TestCompleteErrorHandshake_CallsCompleteFailedOperation validates Fix 2:
// CompleteErrorHandshake should call CompleteFailedOperation (not UpdateOperationState)
// to preserve the 'failed' state while setting completed_at.
func TestCompleteErrorHandshake_CallsCompleteFailedOperation(t *testing.T) {
	mockRepo := &mockOperationRepoForErrors{}
	log := testLogger()

	recorder := NewErrorRecorder(mockRepo, nil, log)

	session := &Session{
		ID:       100,
		TenantID: 42,
	}

	// Expect CompleteFailedOperation to be called with errorAckReceived metadata
	mockRepo.On("CompleteFailedOperation",
		mock.Anything,
		int64(100), // sessionID
		int64(1),   // opId
		mock.MatchedBy(func(data map[string]interface{}) bool {
			val, ok := data["errorAckReceived"]
			return ok && val == true
		}),
	).Return(nil)

	err := recorder.CompleteErrorHandshake(testutil.TestContext(), session, 1)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestCompleteErrorHandshake_NilRepoReturnsNil validates graceful handling.
func TestCompleteErrorHandshake_NilRepoReturnsNil(t *testing.T) {
	log := testLogger()
	recorder := NewErrorRecorder(nil, nil, log)

	session := &Session{ID: 100, TenantID: 42}

	err := recorder.CompleteErrorHandshake(testutil.TestContext(), session, 1)
	assert.NoError(t, err)
}

// TestCompleteErrorHandshake_ZeroSessionIDReturnsNil validates edge case.
func TestCompleteErrorHandshake_ZeroSessionIDReturnsNil(t *testing.T) {
	mockRepo := &mockOperationRepoForErrors{}
	log := testLogger()
	recorder := NewErrorRecorder(mockRepo, nil, log)

	session := &Session{ID: 0, TenantID: 42}

	err := recorder.CompleteErrorHandshake(testutil.TestContext(), session, 1)
	assert.NoError(t, err)
	// CompleteFailedOperation should NOT be called
	mockRepo.AssertNotCalled(t, "CompleteFailedOperation")
}

// TestCompleteErrorHandshake_PropagatesRepoError validates error propagation.
func TestCompleteErrorHandshake_PropagatesRepoError(t *testing.T) {
	mockRepo := &mockOperationRepoForErrors{}
	log := testLogger()

	recorder := NewErrorRecorder(mockRepo, nil, log)

	session := &Session{ID: 100, TenantID: 42}
	expectedErr := errors.New("database connection failed")

	mockRepo.On("CompleteFailedOperation",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(expectedErr)

	err := recorder.CompleteErrorHandshake(testutil.TestContext(), session, 1)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// RecordInboundError Tests
// ============================================================================

// TestRecordInboundError_PersistsACError validates inbound error persistence.
func TestRecordInboundError_PersistsACError(t *testing.T) {
	mockRepo := &mockOperationRepoForErrors{}
	mockEvents := &mockEventStoreForErrors{}
	log := testLogger()

	recorder := NewErrorRecorder(mockRepo, mockEvents, log)

	session := &Session{ID: 100, TenantID: 42}
	posixCode := POSIX_EIO
	message := "AC processing error"

	mockRepo.On("UpdateOperationStateWithError",
		mock.Anything,
		int64(100),
		int64(1),
		models.OperationStateFailed,
		posixCode,
		"inbound_error", // Internal token for AC errors
		message,
		mock.MatchedBy(func(data map[string]interface{}) bool {
			return data["inboundError"] == true &&
				data["acPosixCode"] == posixCode &&
				data["acErrorMessage"] == message
		}),
	).Return(nil)

	mockEvents.On("RecordSCACIError",
		mock.Anything,
		int64(42),
		int64(100),
		"error",
		int64(1),
		posixCode,
		message,
	).Return(nil)

	err := recorder.RecordInboundError(testutil.TestContext(), session, 1, posixCode, message)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockEvents.AssertExpectations(t)
}
