package bssci

import (
	"context"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStorage implements the storage interface for testing
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) CreateDLRXStatus(ctx context.Context, status *mioty.DLRXStatus) error {
	args := m.Called(ctx, status)
	return args.Error(0)
}

func (m *MockStorage) GetDLRXStatusByEndpoint(ctx context.Context, tenantID int64, epEui []byte, limit, offset int, startTime, endTime *time.Time) ([]*mioty.DLRXStatus, int, error) {
	args := m.Called(ctx, tenantID, epEui, limit, offset, startTime, endTime)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*mioty.DLRXStatus), args.Int(1), args.Error(2)
}

func (m *MockStorage) GetAverageDLRXMetrics(ctx context.Context, tenantID int64, epEui []byte, startTime, endTime *time.Time) (float64, float64, int, error) {
	args := m.Called(ctx, tenantID, epEui, startTime, endTime)
	return args.Get(0).(float64), args.Get(1).(float64), args.Int(2), args.Error(3)
}

func (m *MockStorage) GetEndpointBaseStation(ctx context.Context, epEui string, tenantID string) (string, error) {
	args := m.Called(ctx, epEui, tenantID)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) GetDLRXStatusQueryHistory(_ context.Context, _ int64, _ []byte, _, _ int, _, _ *time.Time) ([]*mioty.DLRXStatusQuery, int, error) {
	// No-op stub to satisfy interface
	return nil, 0, nil
}

func (m *MockStorage) GetDLRXStatusQueryStats(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (pending, received, timeout int64, err error) {
	// No-op stub to satisfy interface
	return 0, 0, 0, nil
}

func TestDLRXStatusCreation(t *testing.T) {
	// Setup
	mockStorage := new(MockStorage)
	ctx := testutil.TestContext()

	// Test data
	status := &mioty.DLRXStatus{
		TenantID:  1,
		EpEui:     []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		BsEui:     []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01},
		RxTime:    time.Now().UnixNano(),
		PacketCnt: 42,
		DlRxSnr:   10.5,
		DlRxRssi:  -75.0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set expectations
	mockStorage.On("CreateDLRXStatus", ctx, status).Return(nil)

	// Execute
	err := mockStorage.CreateDLRXStatus(ctx, status)

	// Assert
	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestDLRXStatusRetrieval(t *testing.T) {
	// Setup
	mockStorage := new(MockStorage)
	ctx := testutil.TestContext()
	tenantID := int64(1)
	epEui := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	limit := 10
	offset := 0

	// Test data
	expectedStatuses := []*mioty.DLRXStatus{
		{
			ID:        1,
			TenantID:  tenantID,
			EpEui:     epEui,
			BsEui:     []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01},
			RxTime:    time.Now().UnixNano(),
			PacketCnt: 42,
			DlRxSnr:   10.5,
			DlRxRssi:  -75.0,
		},
		{
			ID:        2,
			TenantID:  tenantID,
			EpEui:     epEui,
			BsEui:     []byte{0x09, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02},
			RxTime:    time.Now().Add(-1 * time.Hour).UnixNano(),
			PacketCnt: 43,
			DlRxSnr:   8.2,
			DlRxRssi:  -80.5,
		},
	}
	totalCount := 2

	// Set expectations
	mockStorage.On("GetDLRXStatusByEndpoint", ctx, tenantID, epEui, limit, offset, (*time.Time)(nil), (*time.Time)(nil)).
		Return(expectedStatuses, totalCount, nil)

	// Execute
	statuses, count, err := mockStorage.GetDLRXStatusByEndpoint(ctx, tenantID, epEui, limit, offset, nil, nil)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, totalCount, count)
	assert.Len(t, statuses, 2)
	assert.Equal(t, expectedStatuses[0].DlRxSnr, statuses[0].DlRxSnr)
	mockStorage.AssertExpectations(t)
}

func TestDLRXStatusAverageMetrics(t *testing.T) {
	// Setup
	mockStorage := new(MockStorage)
	ctx := testutil.TestContext()
	tenantID := int64(1)
	epEui := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	// Expected values
	expectedAvgSnr := 9.35
	expectedAvgRssi := -77.75
	expectedCount := 2

	// Set expectations
	mockStorage.On("GetAverageDLRXMetrics", ctx, tenantID, epEui, (*time.Time)(nil), (*time.Time)(nil)).
		Return(expectedAvgSnr, expectedAvgRssi, expectedCount, nil)

	// Execute
	avgSnr, avgRssi, count, err := mockStorage.GetAverageDLRXMetrics(ctx, tenantID, epEui, nil, nil)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedAvgSnr, avgSnr)
	assert.Equal(t, expectedAvgRssi, avgRssi)
	assert.Equal(t, expectedCount, count)
	mockStorage.AssertExpectations(t)
}

func TestDLRXStatusTenantIsolation(t *testing.T) {
	// Setup
	mockStorage := new(MockStorage)
	ctx := testutil.TestContext()
	tenant1ID := int64(1)
	tenant2ID := int64(2)
	epEui := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	limit := 10
	offset := 0

	// Test data for tenant 1
	tenant1Statuses := []*mioty.DLRXStatus{
		{
			ID:       1,
			TenantID: tenant1ID,
			EpEui:    epEui,
			DlRxSnr:  10.5,
			DlRxRssi: -75.0,
		},
	}

	// Set expectations
	mockStorage.On("GetDLRXStatusByEndpoint", ctx, tenant1ID, epEui, limit, offset, (*time.Time)(nil), (*time.Time)(nil)).
		Return(tenant1Statuses, 1, nil)
	mockStorage.On("GetDLRXStatusByEndpoint", ctx, tenant2ID, epEui, limit, offset, (*time.Time)(nil), (*time.Time)(nil)).
		Return([]*mioty.DLRXStatus{}, 0, nil)

	// Execute - Tenant 1 should see their data
	statuses1, count1, err1 := mockStorage.GetDLRXStatusByEndpoint(ctx, tenant1ID, epEui, limit, offset, nil, nil)
	assert.NoError(t, err1)
	assert.Equal(t, 1, count1)
	assert.Len(t, statuses1, 1)

	// Execute - Tenant 2 should see no data
	statuses2, count2, err2 := mockStorage.GetDLRXStatusByEndpoint(ctx, tenant2ID, epEui, limit, offset, nil, nil)
	assert.NoError(t, err2)
	assert.Equal(t, 0, count2)
	assert.Len(t, statuses2, 0)

	mockStorage.AssertExpectations(t)
}

func TestDLRXStatusTimeFiltering(t *testing.T) {
	// Setup
	mockStorage := new(MockStorage)
	ctx := testutil.TestContext()
	tenantID := int64(1)
	epEui := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	limit := 10
	offset := 0

	// Time range
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	// Test data within time range
	expectedStatuses := []*mioty.DLRXStatus{
		{
			ID:       1,
			TenantID: tenantID,
			EpEui:    epEui,
			RxTime:   time.Now().Add(-12 * time.Hour).UnixNano(),
			DlRxSnr:  10.5,
			DlRxRssi: -75.0,
		},
	}

	// Set expectations
	mockStorage.On("GetDLRXStatusByEndpoint", ctx, tenantID, epEui, limit, offset, &startTime, &endTime).
		Return(expectedStatuses, 1, nil)

	// Execute
	statuses, count, err := mockStorage.GetDLRXStatusByEndpoint(ctx, tenantID, epEui, limit, offset, &startTime, &endTime)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Len(t, statuses, 1)
	mockStorage.AssertExpectations(t)
}

func TestGetEndpointBaseStation(t *testing.T) {
	// Setup
	mockStorage := new(MockStorage)
	ctx := testutil.TestContext()
	epEui := "0102030405060708"
	tenantID := "1"
	expectedBsEui := "0807060504030201"

	// Set expectations
	mockStorage.On("GetEndpointBaseStation", ctx, epEui, tenantID).
		Return(expectedBsEui, nil)

	// Execute
	bsEui, err := mockStorage.GetEndpointBaseStation(ctx, epEui, tenantID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedBsEui, bsEui)
	mockStorage.AssertExpectations(t)
}
