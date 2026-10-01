package scaciservices

import (
	"context"
	"errors"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// ============================================================================
// Mocks for DLService Tests
// ============================================================================

// mockDownlinkScheduler implements scheduler.DownlinkScheduler for testing
type mockDownlinkScheduler struct {
	mock.Mock
}

func (m *mockDownlinkScheduler) QueueDownlink(_ context.Context, req *mioty.DLDataQueue, tenantID int64, _ uuid.UUID) (uint64, uint64, error) {
	args := m.Called(req, tenantID)
	return args.Get(0).(uint64), args.Get(1).(uint64), args.Error(2)
}

func (m *mockDownlinkScheduler) RevokeDownlink(_ context.Context, ref scheduler.DownlinkRef) (uint64, error) {
	args := m.Called(ref)
	return args.Get(0).(uint64), args.Error(1)
}

// mockDLLogger implements logger.Logger for testing
type mockDLLogger struct{}

func (m *mockDLLogger) Debug(_ string, _ ...interface{})                   {}
func (m *mockDLLogger) Info(_ string, _ ...interface{})                    {}
func (m *mockDLLogger) Warn(_ string, _ ...interface{})                    {}
func (m *mockDLLogger) Error(_ string, _ ...interface{})                   {}
func (m *mockDLLogger) Fatal(_ string, _ ...interface{})                   {}
func (m *mockDLLogger) DebugContext(_ context.Context, _ string, _ ...any) {}
func (m *mockDLLogger) InfoContext(_ context.Context, _ string, _ ...any)  {}
func (m *mockDLLogger) WarnContext(_ context.Context, _ string, _ ...any)  {}
func (m *mockDLLogger) ErrorContext(_ context.Context, _ string, _ ...any) {}
func (m *mockDLLogger) FatalContext(_ context.Context, _ string, _ ...any) {}
func (m *mockDLLogger) WithField(_ string, _ interface{}) logger.Logger    { return m }
func (m *mockDLLogger) WithFields(_ map[string]interface{}) logger.Logger  { return m }

// ============================================================================
// §3.11 DL Data Revoke Service Tests
// ============================================================================

// TestDLService_RevokeDownlink_Success verifies that RevokeDownlink returns
// bsEui and empty error token on successful scheduler delegation.
//
// Spec: SCACI §3.11 - DL Data Revoke success path
func TestDLService_RevokeDownlink_Success(t *testing.T) {
	mockScheduler := new(mockDownlinkScheduler)

	orgID := uuid.New()
	epEUI := uint64(0x70B3D59CD0000042)
	ref := scheduler.DownlinkRef{TenantID: 1, QueID: 42, OrganizationID: &orgID, EpEUI: &epEUI}
	expectedBsEui := uint64(0x70B3D59CD00009E6)

	// The owner scope reaches the scheduler unchanged.
	mockScheduler.On("RevokeDownlink", ref).Return(expectedBsEui, nil)

	svc := newTestDLService(t, mockScheduler, &enqueueRecordingStore{})
	bsEui, errToken := svc.RevokeDownlink(testutil.TestContext(), ref)

	// Assert results
	assert.Equal(t, expectedBsEui, bsEui)
	assert.Empty(t, errToken, "Success path should return empty error token")

	// Verify mock
	mockScheduler.AssertExpectations(t)
}

// TestDLService_RevokeDownlink_QueueNotFound_MapsToDownlinkNotFound verifies that
// scheduler.ErrSchedulerQueueNotFound is mapped to errDownlinkNotFound token.
//
// Spec: SCACI §3.11 - Queue entry not found error mapping
func TestDLService_RevokeDownlink_QueueNotFound_MapsToDownlinkNotFound(t *testing.T) {
	mockScheduler := new(mockDownlinkScheduler)

	ref := scheduler.DownlinkRef{TenantID: 1, QueID: 999}

	mockScheduler.On("RevokeDownlink", ref).Return(uint64(0), scheduler.ErrSchedulerQueueNotFound)

	svc := newTestDLService(t, mockScheduler, &enqueueRecordingStore{})
	bsEui, errToken := svc.RevokeDownlink(testutil.TestContext(), ref)

	// Assert results
	assert.Equal(t, uint64(0), bsEui)
	require.Equal(t, scaci.ErrDownlinkNotFound, errToken)

	// Verify mock
	mockScheduler.AssertExpectations(t)
}

// TestDLService_RevokeDownlink_NoResources_MapsToSchedulerUnavailable verifies that
// scheduler.ErrSchedulerNoResources is mapped to errSchedulerUnavailable token.
//
// Spec: SCACI §3.11 - Scheduler unavailable error mapping
func TestDLService_RevokeDownlink_NoResources_MapsToSchedulerUnavailable(t *testing.T) {
	mockScheduler := new(mockDownlinkScheduler)

	ref := scheduler.DownlinkRef{TenantID: 1, QueID: 42}

	mockScheduler.On("RevokeDownlink", ref).Return(uint64(0), scheduler.ErrSchedulerNoResources)

	svc := newTestDLService(t, mockScheduler, &enqueueRecordingStore{})
	bsEui, errToken := svc.RevokeDownlink(testutil.TestContext(), ref)

	// Assert results
	assert.Equal(t, uint64(0), bsEui)
	require.Equal(t, scaci.ErrSchedulerUnavailable, errToken)

	// Verify mock
	mockScheduler.AssertExpectations(t)
}

// ============================================================================
// §3.10 DL Data Queue Service Tests
// ============================================================================

const (
	dlQueueTestTenant int64  = 1
	dlQueueTestQueID  uint64 = 42
	dlQueueTestBsEui  uint64 = 0x70B3D59CD00009E6
	dlQueueTestEpEui  uint64 = 0x70B3D56770111505
)

var errDLQueueTestDispatch = errors.New("dispatch failed")

func queueDownlinkThroughScheduler(t *testing.T, schedQueID, schedBsEui uint64, schedErr error) (scaci.DownlinkQueueOutcome, string) {
	t.Helper()
	mockScheduler := new(mockDownlinkScheduler)
	req := &mioty.DLDataQueue{EpEui: dlQueueTestEpEui, QueId: dlQueueTestQueID}
	mockScheduler.On("QueueDownlink", req, dlQueueTestTenant).Return(schedQueID, schedBsEui, schedErr)

	svc := newTestDLService(t, mockScheduler, &enqueueRecordingStore{})
	outcome, errToken := svc.QueueDownlink(testutil.TestContext(), req, dlQueueTestTenant, uuid.New())
	mockScheduler.AssertExpectations(t)
	return outcome, errToken
}

func TestDLService_QueueDownlink_DispatchedNow(t *testing.T) {
	outcome, errToken := queueDownlinkThroughScheduler(t, dlQueueTestQueID, dlQueueTestBsEui, nil)

	require.Equal(t, "", errToken)
	assert.Equal(t, scaci.DownlinkQueueOutcome{QueID: dlQueueTestQueID, BsEui: dlQueueTestBsEui}, outcome)
}

// SCACI §3.10 allows queueing downlink data a priori: without a connected
// bidirectional serving station the persisted row waits for the endpoint's
// next downlink window instead of being reported to the caller as a failure.
func TestDLService_QueueDownlink_NoBaseStation_DefersDelivery(t *testing.T) {
	outcome, errToken := queueDownlinkThroughScheduler(t, 0, 0, scheduler.ErrSchedulerNoResources)

	require.Equal(t, "", errToken)
	assert.Equal(t, scaci.DownlinkQueueOutcome{QueID: dlQueueTestQueID, Deferred: true}, outcome)
}

func TestDLService_QueueDownlink_QueueNotFound_MapsToDownlinkNotFound(t *testing.T) {
	outcome, errToken := queueDownlinkThroughScheduler(t, 0, 0, scheduler.ErrSchedulerQueueNotFound)

	require.Equal(t, scaci.ErrDownlinkNotFound, errToken)
	assert.Equal(t, scaci.DownlinkQueueOutcome{}, outcome)
}

func TestDLService_QueueDownlink_DispatchFailure_MapsToFailedRecordOperation(t *testing.T) {
	outcome, errToken := queueDownlinkThroughScheduler(t, 0, 0, errDLQueueTestDispatch)

	require.Equal(t, scaci.ErrFailedRecordOperation, errToken)
	assert.Equal(t, scaci.DownlinkQueueOutcome{}, outcome)
}
