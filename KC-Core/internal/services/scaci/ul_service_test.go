package scaciservices

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/scaci"
	"github.com/kilocenter/KC-Core/pkg/scheduler"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mocks for ULService Tests
// ============================================================================

// mockULTransmitScheduler implements scheduler.ULTransmitScheduler for testing
type mockULTransmitScheduler struct {
	mock.Mock
}

func (m *mockULTransmitScheduler) ScheduleULDataTransmit(
	tenantID int64,
	req *mioty.ULDataTransmit,
	targetBsEui *uint64,
) (int64, uint64, error) {
	args := m.Called(tenantID, req, targetBsEui)
	return args.Get(0).(int64), args.Get(1).(uint64), args.Error(2)
}

// mockStatusService implements scaci.StatusService for testing preference lookup
type mockStatusService struct {
	mock.Mock
}

func (m *mockStatusService) GetUptime() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

func (m *mockStatusService) GetBaseStations(ctx context.Context, tenantID int64) ([]*models.BaseStation, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.BaseStation), args.Error(1)
}

func (m *mockStatusService) GetBaseStation(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error) {
	args := m.Called(ctx, tenantID, eui)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BaseStation), args.Error(1)
}

func (m *mockStatusService) GetPreferredBaseStation(ctx context.Context, tenantID int64, epEui []byte) (*uint64, bool, error) {
	args := m.Called(ctx, tenantID, epEui)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*uint64), args.Bool(1), args.Error(2)
}

// mockLogger implements logger.Logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(_ string, _ ...interface{})                   {}
func (m *mockLogger) Info(_ string, _ ...interface{})                    {}
func (m *mockLogger) Warn(_ string, _ ...interface{})                    {}
func (m *mockLogger) Error(_ string, _ ...interface{})                   {}
func (m *mockLogger) Fatal(_ string, _ ...interface{})                   {}
func (m *mockLogger) DebugContext(_ context.Context, _ string, _ ...any) {}
func (m *mockLogger) InfoContext(_ context.Context, _ string, _ ...any)  {}
func (m *mockLogger) WarnContext(_ context.Context, _ string, _ ...any)  {}
func (m *mockLogger) ErrorContext(_ context.Context, _ string, _ ...any) {}
func (m *mockLogger) FatalContext(_ context.Context, _ string, _ ...any) {}
func (m *mockLogger) WithField(_ string, _ interface{}) logger.Logger    { return m }
func (m *mockLogger) WithFields(_ map[string]interface{}) logger.Logger  { return m }

// ============================================================================
// ULService Preference Tests (SCACI §3.9.1)
// ============================================================================

// TestULService_PreferenceFound_UsesPreferedBS verifies that when
// StatusService.GetPreferredBaseStation returns a valid preference, it is used.
func TestULService_PreferenceFound_UsesPreferedBS(t *testing.T) {
	// Setup mocks
	mockScheduler := new(mockULTransmitScheduler)
	mockStatus := new(mockStatusService)
	log := &mockLogger{}

	// Test data
	tenantID := int64(1)
	epEui := uint64(0x0011223344556677)
	preferredBsEui := uint64(0xAABBCCDDEEFF0011)
	expectedOpID := int64(42)
	expectedActualBsEui := preferredBsEui

	// Create request with nil bsEui (triggers preference lookup)
	req := &mioty.ULDataTransmit{
		EpEui:     epEui,
		ShAddr:    0x1234,
		PacketCnt: 10,
		BsEui:     nil, // nil triggers preference lookup
	}

	// Convert epEui to bytes for mock matching
	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, epEui)

	// Setup expectations: preference found
	mockStatus.On("GetPreferredBaseStation", mock.Anything, tenantID, epEuiBytes).
		Return(&preferredBsEui, true, nil)

	// Scheduler should receive the preferred BS EUI
	mockScheduler.On("ScheduleULDataTransmit", tenantID, req, &preferredBsEui).
		Return(expectedOpID, expectedActualBsEui, nil)

	// Create service and call
	svc := NewULService(mockScheduler, mockStatus, log)
	opID, actualBsEui, errToken := svc.ScheduleULTransmit(context.Background(), req, tenantID)

	// Assert results
	assert.Equal(t, expectedOpID, opID)
	assert.Equal(t, expectedActualBsEui, actualBsEui)
	assert.Empty(t, errToken)

	// Verify mocks
	mockStatus.AssertExpectations(t)
	mockScheduler.AssertExpectations(t)
}

// TestULService_NoPreference_FallsBackToScheduler verifies that when
// StatusService.GetPreferredBaseStation returns (nil, false, nil), scheduler
// receives nil targetBsEui and uses deterministic fallback.
func TestULService_NoPreference_FallsBackToScheduler(t *testing.T) {
	// Setup mocks
	mockScheduler := new(mockULTransmitScheduler)
	mockStatus := new(mockStatusService)
	log := &mockLogger{}

	// Test data
	tenantID := int64(1)
	epEui := uint64(0x0011223344556677)
	fallbackBsEui := uint64(0x1111222233334444) // Scheduler picks lowest EUI
	expectedOpID := int64(100)

	// Create request with nil bsEui
	req := &mioty.ULDataTransmit{
		EpEui:     epEui,
		ShAddr:    0x1234,
		PacketCnt: 10,
		BsEui:     nil,
	}

	// Convert epEui to bytes for mock matching
	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, epEui)

	// Setup expectations: no preference found (nil, false, nil)
	mockStatus.On("GetPreferredBaseStation", mock.Anything, tenantID, epEuiBytes).
		Return(nil, false, nil)

	// Scheduler should receive nil (uses deterministic fallback)
	var nilBsEui *uint64
	mockScheduler.On("ScheduleULDataTransmit", tenantID, req, nilBsEui).
		Return(expectedOpID, fallbackBsEui, nil)

	// Create service and call
	svc := NewULService(mockScheduler, mockStatus, log)
	opID, actualBsEui, errToken := svc.ScheduleULTransmit(context.Background(), req, tenantID)

	// Assert results
	assert.Equal(t, expectedOpID, opID)
	assert.Equal(t, fallbackBsEui, actualBsEui)
	assert.Empty(t, errToken)

	// Verify mocks
	mockStatus.AssertExpectations(t)
	mockScheduler.AssertExpectations(t)
}

// TestULService_PreferenceLookupError_FallsBack verifies that
// preference lookup errors are non-fatal and result in fallback selection.
func TestULService_PreferenceLookupError_FallsBack(t *testing.T) {
	// Setup mocks
	mockScheduler := new(mockULTransmitScheduler)
	mockStatus := new(mockStatusService)
	log := &mockLogger{}

	// Test data
	tenantID := int64(1)
	epEui := uint64(0x0011223344556677)
	fallbackBsEui := uint64(0x1111222233334444)
	expectedOpID := int64(200)

	// Create request with nil bsEui
	req := &mioty.ULDataTransmit{
		EpEui:     epEui,
		ShAddr:    0x1234,
		PacketCnt: 10,
		BsEui:     nil,
	}

	// Convert epEui to bytes for mock matching
	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, epEui)

	// Setup expectations: lookup returns error
	lookupErr := errors.New("database connection failed")
	mockStatus.On("GetPreferredBaseStation", mock.Anything, tenantID, epEuiBytes).
		Return(nil, false, lookupErr)

	// Scheduler should receive nil (fallback due to error)
	var nilBsEui *uint64
	mockScheduler.On("ScheduleULDataTransmit", tenantID, req, nilBsEui).
		Return(expectedOpID, fallbackBsEui, nil)

	// Create service and call
	svc := NewULService(mockScheduler, mockStatus, log)
	opID, actualBsEui, errToken := svc.ScheduleULTransmit(context.Background(), req, tenantID)

	// Assert: operation should succeed despite lookup error
	assert.Equal(t, expectedOpID, opID)
	assert.Equal(t, fallbackBsEui, actualBsEui)
	assert.Empty(t, errToken, "lookup error should not cause operation failure")

	// Verify mocks
	mockStatus.AssertExpectations(t)
	mockScheduler.AssertExpectations(t)
}

// TestULService_ExplicitBsEui_SkipsPreferenceLookup verifies that when
// req.BsEui is non-nil, preference lookup is skipped entirely.
func TestULService_ExplicitBsEui_SkipsPreferenceLookup(t *testing.T) {
	// Setup mocks
	mockScheduler := new(mockULTransmitScheduler)
	mockStatus := new(mockStatusService)
	log := &mockLogger{}

	// Test data
	tenantID := int64(1)
	epEui := uint64(0x0011223344556677)
	explicitBsEui := uint64(0x9999888877776666)
	expectedOpID := int64(300)

	// Create request with explicit bsEui (should skip preference lookup)
	req := &mioty.ULDataTransmit{
		EpEui:     epEui,
		ShAddr:    0x1234,
		PacketCnt: 10,
		BsEui:     &explicitBsEui,
	}

	// StatusService.GetPreferredBaseStation should NOT be called
	// (no expectation set, will fail if called)

	// Scheduler should receive the explicit BS EUI
	mockScheduler.On("ScheduleULDataTransmit", tenantID, req, &explicitBsEui).
		Return(expectedOpID, explicitBsEui, nil)

	// Create service and call
	svc := NewULService(mockScheduler, mockStatus, log)
	opID, actualBsEui, errToken := svc.ScheduleULTransmit(context.Background(), req, tenantID)

	// Assert results
	assert.Equal(t, expectedOpID, opID)
	assert.Equal(t, explicitBsEui, actualBsEui)
	assert.Empty(t, errToken)

	// Verify: StatusService should NOT have been called
	mockStatus.AssertNotCalled(t, "GetPreferredBaseStation", mock.Anything, mock.Anything, mock.Anything)
	mockScheduler.AssertExpectations(t)
}

// TestULService_NilStatusSvc_FallsBackDirectly verifies that when
// statusSvc is nil, preference lookup is skipped and scheduler uses fallback.
func TestULService_NilStatusSvc_FallsBackDirectly(t *testing.T) {
	// Setup mocks - statusSvc is nil
	mockScheduler := new(mockULTransmitScheduler)
	log := &mockLogger{}

	// Test data
	tenantID := int64(1)
	epEui := uint64(0x0011223344556677)
	fallbackBsEui := uint64(0x1111222233334444)
	expectedOpID := int64(400)

	// Create request with nil bsEui
	req := &mioty.ULDataTransmit{
		EpEui:     epEui,
		ShAddr:    0x1234,
		PacketCnt: 10,
		BsEui:     nil,
	}

	// Scheduler should receive nil (no preference lookup attempted)
	var nilBsEui *uint64
	mockScheduler.On("ScheduleULDataTransmit", tenantID, req, nilBsEui).
		Return(expectedOpID, fallbackBsEui, nil)

	// Create service with nil statusSvc
	svc := NewULService(mockScheduler, nil, log)

	// Should not panic
	opID, actualBsEui, errToken := svc.ScheduleULTransmit(context.Background(), req, tenantID)

	// Assert results
	assert.Equal(t, expectedOpID, opID)
	assert.Equal(t, fallbackBsEui, actualBsEui)
	assert.Empty(t, errToken)

	// Verify mock
	mockScheduler.AssertExpectations(t)
}

// TestULService_NilScheduler_ReturnsNotSupported verifies that when
// scheduler is nil, the appropriate error token is returned.
func TestULService_NilScheduler_ReturnsNotSupported(t *testing.T) {
	// Setup: nil scheduler
	mockStatus := new(mockStatusService)
	log := &mockLogger{}

	req := &mioty.ULDataTransmit{
		EpEui:     0x0011223344556677,
		ShAddr:    0x1234,
		PacketCnt: 10,
	}

	// Create service with nil scheduler
	svc := NewULService(nil, mockStatus, log)
	opID, actualBsEui, errToken := svc.ScheduleULTransmit(context.Background(), req, 1)

	// Assert: should return error token
	assert.Equal(t, int64(0), opID)
	assert.Equal(t, uint64(0), actualBsEui)
	assert.Equal(t, scaci.ErrULTransmitNotSupported, errToken)
}

// TestULService_SchedulerError_MapsToToken verifies that scheduler errors
// are correctly mapped to SCACI error tokens.
func TestULService_SchedulerError_MapsToToken(t *testing.T) {
	testCases := []struct {
		name          string
		schedulerErr  error
		expectedToken string
	}{
		{
			name:          "NoResources",
			schedulerErr:  scheduler.ErrSchedulerNoResources,
			expectedToken: scaci.ErrBaseStationUnavailable,
		},
		{
			name:          "ResourceMissing",
			schedulerErr:  scheduler.ErrSchedulerResourceMissing,
			expectedToken: scaci.ErrBaseStationUnavailable,
		},
		{
			name:          "TenantMismatch",
			schedulerErr:  bssci.ErrBaseStationTenantMismatch,
			expectedToken: scaci.ErrBaseStationNotFound,
		},
		{
			name:          "SessionNotReady",
			schedulerErr:  bssci.ErrSessionNotReady,
			expectedToken: scaci.ErrBaseStationUnavailable,
		},
		{
			name:          "SessionNotBidirectional",
			schedulerErr:  bssci.ErrSessionNotBidirectional,
			expectedToken: scaci.ErrBaseStationUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockScheduler := new(mockULTransmitScheduler)
			log := &mockLogger{}

			req := &mioty.ULDataTransmit{
				EpEui:     0x0011223344556677,
				ShAddr:    0x1234,
				PacketCnt: 10,
			}

			// Scheduler returns error
			mockScheduler.On("ScheduleULDataTransmit", mock.Anything, req, mock.Anything).
				Return(int64(0), uint64(0), tc.schedulerErr)

			svc := NewULService(mockScheduler, nil, log)
			_, _, errToken := svc.ScheduleULTransmit(context.Background(), req, 1)

			require.Equal(t, tc.expectedToken, errToken)
		})
	}
}
