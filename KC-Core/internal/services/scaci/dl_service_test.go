package scaciservices

import (
	"context"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mocks for DLService Tests
// ============================================================================

// mockDownlinkScheduler implements scheduler.DownlinkScheduler for testing
type mockDownlinkScheduler struct {
	mock.Mock
}

func (m *mockDownlinkScheduler) QueueDownlink(_ context.Context, req *mioty.DLDataQueue, tenantID int64) (uint64, uint64, error) {
	args := m.Called(req, tenantID)
	return args.Get(0).(uint64), args.Get(1).(uint64), args.Error(2)
}

func (m *mockDownlinkScheduler) RevokeDownlink(tenantID int64, queId uint64) (uint64, error) {
	args := m.Called(tenantID, queId)
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
	log := &mockDLLogger{}

	// Test data
	tenantID := int64(1)
	queId := uint64(42)
	expectedBsEui := uint64(0x70B3D59CD00009E6)

	// Setup: scheduler returns success
	mockScheduler.On("RevokeDownlink", tenantID, queId).
		Return(expectedBsEui, nil)

	// Create service and call
	svc := NewDLService(mockScheduler, nil, log)
	bsEui, errToken := svc.RevokeDownlink(context.Background(), queId, tenantID)

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
	log := &mockDLLogger{}

	// Test data
	tenantID := int64(1)
	queId := uint64(999) // Non-existent

	// Setup: scheduler returns queue not found
	mockScheduler.On("RevokeDownlink", tenantID, queId).
		Return(uint64(0), scheduler.ErrSchedulerQueueNotFound)

	// Create service and call
	svc := NewDLService(mockScheduler, nil, log)
	bsEui, errToken := svc.RevokeDownlink(context.Background(), queId, tenantID)

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
	log := &mockDLLogger{}

	// Test data
	tenantID := int64(1)
	queId := uint64(42)

	// Setup: scheduler returns no resources
	mockScheduler.On("RevokeDownlink", tenantID, queId).
		Return(uint64(0), scheduler.ErrSchedulerNoResources)

	// Create service and call
	svc := NewDLService(mockScheduler, nil, log)
	bsEui, errToken := svc.RevokeDownlink(context.Background(), queId, tenantID)

	// Assert results
	assert.Equal(t, uint64(0), bsEui)
	require.Equal(t, scaci.ErrSchedulerUnavailable, errToken)

	// Verify mock
	mockScheduler.AssertExpectations(t)
}
