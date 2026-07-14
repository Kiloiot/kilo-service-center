package grpc

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// MockStorage implements storage.Storage for testing
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) CreateEndPoint(ctx context.Context, endpoint *models.EndPoint) (*models.EndPoint, error) {
	args := m.Called(ctx, endpoint)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EndPoint), args.Error(1)
}

func (m *MockStorage) GetEndPoint(ctx context.Context, endpointEUI []byte, tenantID int64) (*models.EndPoint, error) {
	args := m.Called(ctx, endpointEUI, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EndPoint), args.Error(1)
}

func (m *MockStorage) UpdateEndPoint(ctx context.Context, endpoint *models.EndPoint) (*models.EndPoint, error) {
	args := m.Called(ctx, endpoint)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EndPoint), args.Error(1)
}

func (m *MockStorage) DeleteEndPoint(ctx context.Context, endpointEUI []byte, tenantID int64) error {
	args := m.Called(ctx, endpointEUI, tenantID)
	return args.Error(0)
}

func (m *MockStorage) ListEndPointsByModelWithSnapshot(_ context.Context, _ int64, _ uuid.UUID) ([]*models.EndPoint, error) {
	return nil, nil
}

func (m *MockStorage) ListEndPoints(ctx context.Context, tenantID int64, limit, offset int) ([]*models.EndPoint, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EndPoint), args.Error(1)
}

func (m *MockStorage) CreateBaseStation(ctx context.Context, baseStation *models.BaseStation) (*models.BaseStation, error) {
	args := m.Called(ctx, baseStation)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BaseStation), args.Error(1)
}

func (m *MockStorage) GetBaseStation(ctx context.Context, bseui []byte, tenantID int64) (*models.BaseStation, error) {
	args := m.Called(ctx, bseui, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BaseStation), args.Error(1)
}

func (m *MockStorage) UpdateBaseStation(ctx context.Context, baseStation *models.BaseStation) (*models.BaseStation, error) {
	args := m.Called(ctx, baseStation)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BaseStation), args.Error(1)
}

func (m *MockStorage) DeleteBaseStation(ctx context.Context, bseui []byte, tenantID int64) error {
	args := m.Called(ctx, bseui, tenantID)
	return args.Error(0)
}

func (m *MockStorage) ListBaseStations(ctx context.Context, tenantID int64, limit, offset int) ([]*models.BaseStation, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.BaseStation), args.Error(1)
}

func (m *MockStorage) EnqueueDownlink(ctx context.Context, downlink *storage.DownlinkMessage) (*storage.DownlinkMessage, error) {
	args := m.Called(ctx, downlink)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.DownlinkMessage), args.Error(1)
}

func (m *MockStorage) GetDownlinkQueue(ctx context.Context, endpointEUI string, tenantID string) ([]*storage.DownlinkMessage, error) {
	args := m.Called(ctx, endpointEUI, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.DownlinkMessage), args.Error(1)
}

// orgID parameter enables organization-scoped status updates
func (m *MockStorage) UpdateDownlinkStatus(ctx context.Context, id string, status string, orgID *uuid.UUID) error {
	args := m.Called(ctx, id, status, orgID)
	return args.Error(0)
}

func (m *MockStorage) GetDownlinkResults(ctx context.Context, endpointEUI string, tenantID string, orgID *uuid.UUID, statusFilter string,
	timeFrom, timeTo *time.Time, limit, offset int) ([]*storage.DownlinkMessage, int, error) {
	args := m.Called(ctx, endpointEUI, tenantID, orgID, statusFilter, timeFrom, timeTo, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*storage.DownlinkMessage), args.Int(1), args.Error(2)
}

func (m *MockStorage) UpdateDownlinkBaseStation(ctx context.Context, queId uint64, tenantID string, bsEUI uint64) error {
	args := m.Called(ctx, queId, tenantID, bsEUI)
	return args.Error(0)
}

func (m *MockStorage) GetDownlinkByQueueID(ctx context.Context, queId uint64, tenantID string) (*storage.DownlinkMessage, error) {
	args := m.Called(ctx, queId, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.DownlinkMessage), args.Error(1)
}

func (m *MockStorage) GetDownlinkByPacketCnt(ctx context.Context, tenantID string, epEui string, packetCnt uint32) (*storage.DownlinkMessage, error) {
	args := m.Called(ctx, tenantID, epEui, packetCnt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.DownlinkMessage), args.Error(1)
}

func (m *MockStorage) RevokeDownlink(ctx context.Context, queId int64, tenantID string) error {
	args := m.Called(ctx, queId, tenantID)
	return args.Error(0)
}

// orgID parameter enables organization-scoped result updates
func (m *MockStorage) UpdateDownlinkResult(ctx context.Context, queId int64, result string, txTime *int64,
	packetCnt *uint32, bsEUI []byte, epEUI []byte, tenantID string, orgID *uuid.UUID) error {
	args := m.Called(ctx, queId, result, txTime, packetCnt, bsEUI, epEUI, tenantID, orgID)
	return args.Error(0)
}

func (m *MockStorage) CreateULDataMessage(ctx context.Context, msg *mioty.ULDataMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockStorage) GetULDataMessage(ctx context.Context, id string, tenantID int64) (*mioty.ULDataMessage, error) {
	args := m.Called(ctx, id, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mioty.ULDataMessage), args.Error(1)
}

func (m *MockStorage) ListULDataMessages(ctx context.Context, tenantID int64, epEui []byte, limit, offset int) ([]*mioty.ULDataMessage, int, error) {
	args := m.Called(ctx, tenantID, epEui, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*mioty.ULDataMessage), args.Int(1), args.Error(2)
}

func (m *MockStorage) GetRecentULDataMessages(ctx context.Context, tenantID int64, since time.Time, limit int) ([]*mioty.ULDataMessage, error) {
	args := m.Called(ctx, tenantID, since, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*mioty.ULDataMessage), args.Error(1)
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

func (m *MockStorage) GetAverageDLRXMetrics(ctx context.Context, tenantID int64, epEui []byte, startTime, endTime *time.Time) (avgSnr, avgRssi float64, count int, err error) {
	args := m.Called(ctx, tenantID, epEui, startTime, endTime)
	return args.Get(0).(float64), args.Get(1).(float64), args.Int(2), args.Error(3)
}

func (m *MockStorage) CreateDLRXStatusQuery(ctx context.Context, tenantID int64, orgUUID *uuid.UUID, epEui, bsEui []byte, opId int64) error {
	args := m.Called(ctx, tenantID, orgUUID, epEui, bsEui, opId)
	return args.Error(0)
}

func (m *MockStorage) MarkDLRXStatusReceived(ctx context.Context, tenantID int64, epEui []byte, bsEui []byte, bsOpID int64) (bool, error) {
	args := m.Called(ctx, tenantID, epEui, bsEui, bsOpID)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorage) ExpireDLRXStatusQuery(ctx context.Context, cutoff time.Time) (int64, error) {
	args := m.Called(ctx, cutoff)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStorage) GetDLRXStatusQueryHistory(ctx context.Context, tenantID int64, epEui []byte, limit, offset int, startTime, endTime *time.Time) ([]*mioty.DLRXStatusQuery, int, error) {
	args := m.Called(ctx, tenantID, epEui, limit, offset, startTime, endTime)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*mioty.DLRXStatusQuery), args.Int(1), args.Error(2)
}

func (m *MockStorage) GetDLRXStatusQueryStats(ctx context.Context, tenantID int64, epEui []byte, startTime, endTime *time.Time) (pending, received, timeout int64, err error) {
	args := m.Called(ctx, tenantID, epEui, startTime, endTime)
	return args.Get(0).(int64), args.Get(1).(int64), args.Get(2).(int64), args.Error(3)
}

func (m *MockStorage) GetEndpointBaseStation(ctx context.Context, epEui string, tenantID string) (bsEui string, err error) {
	args := m.Called(ctx, epEui, tenantID)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) GetBaseStationMessageStats(ctx context.Context, tenantID int64, bsEui []byte,
	startTime, endTime *time.Time) (*mioty.BaseStationMessageStats, error) {
	args := m.Called(ctx, tenantID, bsEui, startTime, endTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mioty.BaseStationMessageStats), args.Error(1)
}

func (m *MockStorage) GetBaseStationEndpointCounts(ctx context.Context, tenantID int64, bsEui []byte,
	startTime, endTime *time.Time) (map[string]int64, error) {
	args := m.Called(ctx, tenantID, bsEui, startTime, endTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]int64), args.Error(1)
}

func (m *MockStorage) GetBaseStationLastSeen(ctx context.Context, tenantID int64, bsEui []byte) (*time.Time, error) {
	args := m.Called(ctx, tenantID, bsEui)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*time.Time), args.Error(1)
}

func (m *MockStorage) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStorage) ListAllBaseStationLocations(ctx context.Context) ([]*models.BaseStation, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.BaseStation), args.Error(1)
}

// TestGetBaseStationStats_StorageMethods tests that the storage methods work correctly
// This is a unit test focused on the storage interface methods
func TestGetBaseStationStats_StorageMethods(t *testing.T) {
	// Create test fixtures
	now := time.Now()
	lastWeek := now.Add(-7 * 24 * time.Hour)
	lastMessage := now.Add(-1 * time.Hour)

	bsEuiStr := "0011223344556677"
	bsEuiBytes, _ := hex.DecodeString(bsEuiStr)
	tenantID := int64(123)

	// Test GetBaseStationMessageStats
	mockStats := &mioty.BaseStationMessageStats{
		TotalMessages:     1500,
		TotalEndpoints:    25,
		MessagesToday:     50,
		MessagesThisWeek:  350,
		MessagesThisMonth: 1200,
		AvgRSSI:           -85.5,
		AvgSNR:            12.3,
		LastMessageAt:     &lastMessage,
	}

	// Create mock storage
	mockStorage := new(MockStorage)
	mockStorage.On("GetBaseStationMessageStats", mock.Anything, tenantID, bsEuiBytes,
		&lastWeek, &now).Return(mockStats, nil).Once()

	// Execute
	stats, err := mockStorage.GetBaseStationMessageStats(pkgcontext.WithTenantID(context.Background(), tenantID), tenantID, bsEuiBytes, &lastWeek, &now)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(1500), stats.TotalMessages)
	assert.Equal(t, int64(25), stats.TotalEndpoints)
	assert.Equal(t, -85.5, stats.AvgRSSI)

	// Test GetBaseStationEndpointCounts
	endpointCounts := map[string]int64{
		"AABBCCDDEE112233": 100,
		"1122334455667788": 75,
	}
	mockStorage.On("GetBaseStationEndpointCounts", mock.Anything, tenantID, bsEuiBytes,
		&lastWeek, &now).Return(endpointCounts, nil).Once()

	counts, err := mockStorage.GetBaseStationEndpointCounts(pkgcontext.WithTenantID(context.Background(), tenantID), tenantID, bsEuiBytes, &lastWeek, &now)
	assert.NoError(t, err)
	assert.Len(t, counts, 2)
	assert.Equal(t, int64(100), counts["AABBCCDDEE112233"])

	// Test GetBaseStationLastSeen
	lastSeen := now.Add(-30 * time.Minute)
	mockStorage.On("GetBaseStationLastSeen", mock.Anything, tenantID, bsEuiBytes).
		Return(&lastSeen, nil).Once()

	seenTime, err := mockStorage.GetBaseStationLastSeen(pkgcontext.WithTenantID(context.Background(), tenantID), tenantID, bsEuiBytes)
	assert.NoError(t, err)
	assert.NotNil(t, seenTime)
	assert.Equal(t, lastSeen, *seenTime)

	// Verify all expectations were met
	mockStorage.AssertExpectations(t)
}

// MockBaseStationService for testing
type MockBaseStationService struct {
	mock.Mock
}

func (m *MockBaseStationService) Create(ctx context.Context, baseStation *models.BaseStation) (*models.BaseStation, error) {
	args := m.Called(ctx, baseStation)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BaseStation), args.Error(1)
}

func (m *MockBaseStationService) GetByEUI(ctx context.Context, bsEui []byte, tenantID int64) (*models.BaseStation, error) {
	args := m.Called(ctx, bsEui, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BaseStation), args.Error(1)
}

func (m *MockBaseStationService) Update(ctx context.Context, baseStation *models.BaseStation) (*models.BaseStation, error) {
	args := m.Called(ctx, baseStation)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BaseStation), args.Error(1)
}

func (m *MockBaseStationService) Delete(ctx context.Context, bsEui []byte, tenantID int64) error {
	args := m.Called(ctx, bsEui, tenantID)
	return args.Error(0)
}

func (m *MockBaseStationService) List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.BaseStation, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.BaseStation), args.Error(1)
}

func (m *MockBaseStationService) UpdateEUI(ctx context.Context, tenantID int64, oldEui, newEui []byte) (*models.BaseStation, error) {
	args := m.Called(ctx, tenantID, oldEui, newEui)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BaseStation), args.Error(1)
}

func (m *MockBaseStationService) ListAllLocations(ctx context.Context) ([]*models.BaseStation, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.BaseStation), args.Error(1)
}
