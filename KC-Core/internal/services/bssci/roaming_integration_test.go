package bssciservices

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/roaming"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Integration test for complete roaming flow
func TestRoamingIntegration_AttachDetachFlow(t *testing.T) {
	// This test simulates a complete roaming scenario:
	// 1. Endpoint from tenant 1 attaches to base station owned by tenant 2
	// 2. System detects roaming and validates partnership
	// 3. Messages are routed to owner tenant storage
	// 4. Detach cleans up roaming state

	// Setup test environment
	ctx := context.Background()

	// Create test data
	epEui := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	bsEui := []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01}
	ownerTenantID := int64(1)
	servingTenantID := int64(2)
	sessionID := int64(12345)

	// Create mock storage with roaming data
	mockStorage := &MockStorageWithRoaming{
		endpoints: map[string]*models.EndPoint{
			"0102030405060708": {
				ID:            123,
				TenantID:      servingTenantID, // Currently served by tenant 2
				OwnerTenantID: ownerTenantID,   // Owned by tenant 1
				Name:          "Roaming Sensor",
				Bidi:          true,
			},
		},
		tenantPartners: map[string]bool{
			"1-2": true, // Tenants 1 and 2 are partners
			"2-1": true,
		},
		roamingEvents:  []models.RoamingEvent{},
		sessionRoaming: make(map[int64][]models.RoamingEndpointInfo),
	}

	// Create roaming configuration using production defaults
	roamingConfig := roaming.DefaultDetectorConfig()

	// Create roaming service
	adapter := &RoamingAdapter{
		storage: mockStorage,
	}
	detector := roaming.NewDetector(roamingConfig, adapter, adapter)
	roamingSvc := &RoamingService{
		detector: detector,
		storage:  mockStorage,
		enabled:  true,
	}

	// Test 1: Attach operation with roaming detection
	t.Run("AttachWithRoaming", func(t *testing.T) {
		// Detect and validate roaming
		isRoaming, detectedOwner, err := roamingSvc.DetectAndValidateRoaming(ctx, epEui, servingTenantID)
		require.NoError(t, err)
		assert.True(t, isRoaming)
		assert.Equal(t, ownerTenantID, detectedOwner)

		// Record attach event
		err = roamingSvc.RecordAttach(ctx, epEui, bsEui, servingTenantID)
		require.NoError(t, err)

		// Update session roaming
		err = roamingSvc.UpdateSessionRoaming(ctx, sessionID, epEui, true, servingTenantID)
		require.NoError(t, err)

		// Verify events were recorded
		assert.Len(t, mockStorage.roamingEvents, 1)
		assert.Equal(t, "attach", mockStorage.roamingEvents[0].EventType)
		assert.Equal(t, epEui, mockStorage.roamingEvents[0].EpEUI)
		assert.Equal(t, ownerTenantID, mockStorage.roamingEvents[0].OwnerTenantID)
		assert.Equal(t, servingTenantID, mockStorage.roamingEvents[0].ServingTenantID)

		// Verify session tracking
		endpoints := mockStorage.sessionRoaming[sessionID]
		assert.Len(t, endpoints, 1)
		assert.Equal(t, hex.EncodeToString(epEui), endpoints[0].EUI)
		assert.Equal(t, ownerTenantID, endpoints[0].OwnerTenantID)
	})

	// Test 2: UL Data routing to owner tenant
	t.Run("ULDataRouting", func(t *testing.T) {
		// Resolve owner for UL data
		_, resolvedOwner, err := roamingSvc.DetectAndValidateRoaming(ctx, epEui, servingTenantID)
		require.NoError(t, err)
		assert.Equal(t, ownerTenantID, resolvedOwner)

		// Verify cache hit (should be cached from attach)
		metrics := detector.GetMetrics()
		assert.Greater(t, metrics.GetCacheHits(), uint64(0))
	})

	// Test 3: Detach operation with cleanup
	t.Run("DetachWithCleanup", func(t *testing.T) {
		// Record detach event
		err := roamingSvc.RecordDetach(ctx, epEui, bsEui, servingTenantID)
		require.NoError(t, err)

		// Remove from session roaming
		err = roamingSvc.UpdateSessionRoaming(ctx, sessionID, epEui, false, servingTenantID)
		require.NoError(t, err)

		// Verify detach event recorded
		assert.Len(t, mockStorage.roamingEvents, 2)
		assert.Equal(t, "detach", mockStorage.roamingEvents[1].EventType)

		// Verify session cleanup
		endpoints := mockStorage.sessionRoaming[sessionID]
		assert.Len(t, endpoints, 0)
	})
}

// Test roaming validation with invalid partnerships
func TestRoamingIntegration_InvalidPartnership(t *testing.T) {
	ctx := context.Background()

	epEui := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22}
	ownerTenantID := int64(5)
	servingTenantID := int64(9)

	mockStorage := &MockStorageWithRoaming{
		endpoints: map[string]*models.EndPoint{
			"AABBCCDDEEFF1122": {
				ID:            456,
				TenantID:      servingTenantID,
				OwnerTenantID: ownerTenantID,
			},
		},
		tenantPartners: map[string]bool{
			// No partnership between 5 and 9
		},
		roamingEvents: []models.RoamingEvent{},
	}

	roamingConfig := &roaming.DetectorConfig{
		EnableAuditTrail: true,
	}

	adapter := &RoamingAdapter{
		storage: mockStorage,
	}
	detector := roaming.NewDetector(roamingConfig, adapter, adapter)
	roamingSvc := &RoamingService{
		detector: detector,
		storage:  mockStorage,
		enabled:  true,
	}

	// Attempt to validate roaming without partnership
	isRoaming, _, err := roamingSvc.DetectAndValidateRoaming(ctx, epEui, servingTenantID)

	// Should fail due to missing partnership
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no roaming agreement between tenants")
	assert.False(t, isRoaming)

	// Verify no events were recorded
	assert.Len(t, mockStorage.roamingEvents, 0)
}

// Test cache expiration and refresh
func TestRoamingIntegration_CacheExpiration(t *testing.T) {
	ctx := context.Background()

	epEui := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	ownerTenantID := int64(3)
	servingTenantID := int64(4)

	lookupCount := 0
	mockStorage := &MockStorageWithRoaming{
		endpoints: map[string]*models.EndPoint{
			"1122334455667788": {
				ID:            789,
				TenantID:      servingTenantID,
				OwnerTenantID: ownerTenantID,
			},
		},
		lookupCallback: func() {
			lookupCount++
		},
	}

	// Short TTL for testing cache expiration
	// Use production defaults with custom TTL for expiration test
	roamingConfig := roaming.DefaultDetectorConfig()
	roamingConfig.CacheTTL = 100 * time.Millisecond // Short TTL for testing expiration

	adapter := &RoamingAdapter{
		storage: mockStorage,
	}
	detector := roaming.NewDetector(roamingConfig, adapter, adapter)

	// First lookup - should hit database
	isRoaming1, owner1, err := detector.DetectRoaming(ctx, epEui, servingTenantID)
	require.NoError(t, err)
	assert.True(t, isRoaming1)
	assert.Equal(t, ownerTenantID, owner1)
	assert.Equal(t, 1, lookupCount)

	// Second lookup - should hit cache
	isRoaming2, owner2, err := detector.DetectRoaming(ctx, epEui, servingTenantID)
	require.NoError(t, err)
	assert.True(t, isRoaming2)
	assert.Equal(t, ownerTenantID, owner2)
	assert.Equal(t, 1, lookupCount) // No additional lookup

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third lookup - cache expired, should hit database
	isRoaming3, owner3, err := detector.DetectRoaming(ctx, epEui, servingTenantID)
	require.NoError(t, err)
	assert.True(t, isRoaming3)
	assert.Equal(t, ownerTenantID, owner3)
	assert.Equal(t, 2, lookupCount) // Additional lookup after expiry
}

// Test metrics collection
func TestRoamingIntegration_Metrics(t *testing.T) {
	ctx := context.Background()

	mockStorage := &MockStorageWithRoaming{
		endpoints: map[string]*models.EndPoint{
			"1111111111111111": {
				ID:            1,
				TenantID:      1,
				OwnerTenantID: 1,
			},
			"2222222222222222": {
				ID:            2,
				TenantID:      2,
				OwnerTenantID: 1,
			},
		},
		tenantPartners: map[string]bool{"1-2": true, "2-1": true},
		roamingEvents:  []models.RoamingEvent{},
	}

	// Use production defaults for metrics test
	roamingConfig := roaming.DefaultDetectorConfig()

	adapter := &RoamingAdapter{
		storage: mockStorage,
	}
	detector := roaming.NewDetector(roamingConfig, adapter, adapter)
	roamingSvc := &RoamingService{
		detector: detector,
		storage:  mockStorage,
		enabled:  true,
	}

	// Perform multiple detections
	testCases := []struct {
		epEui           string
		servingTenantID int64
		expectRoaming   bool
	}{
		{"1111111111111111", 1, false}, // Not roaming
		{"2222222222222222", 2, true},  // Roaming
		{"1111111111111111", 1, false}, // Cache hit
		{"2222222222222222", 2, true},  // Cache hit
	}

	for _, tc := range testCases {
		epEui := hexToBytes(tc.epEui)
		isRoaming, _, err := roamingSvc.DetectAndValidateRoaming(ctx, epEui, tc.servingTenantID)
		require.NoError(t, err)
		assert.Equal(t, tc.expectRoaming, isRoaming)
	}

	// Check metrics
	metrics := detector.GetMetrics()
	// Only cache misses increment roaming/local counters
	totalDetections := metrics.GetRoamingDetected() + metrics.GetLocalDetected()
	assert.Equal(t, uint64(2), totalDetections)              // Only first 2 calls (cache misses)
	assert.Equal(t, uint64(1), metrics.GetRoamingDetected()) // Only line 290
	assert.Equal(t, uint64(2), metrics.GetCacheHits())
	assert.Equal(t, uint64(2), metrics.GetCacheMisses())
}

// Helper function to convert hex string to bytes
func hexToBytes(hex string) []byte {
	bytes := make([]byte, len(hex)/2)
	for i := 0; i < len(hex); i += 2 {
		var b byte
		for j := 0; j < 2; j++ {
			c := hex[i+j]
			if c >= '0' && c <= '9' {
				b = (b << 4) | (c - '0')
			} else if c >= 'A' && c <= 'F' {
				b = (b << 4) | (c - 'A' + 10)
			} else if c >= 'a' && c <= 'f' {
				b = (b << 4) | (c - 'a' + 10)
			}
		}
		bytes[i/2] = b
	}
	return bytes
}

// MockStorageWithRoaming implements full roaming storage interface for testing
type MockStorageWithRoaming struct {
	endpoints      map[string]*models.EndPoint
	tenantPartners map[string]bool
	roamingEvents  []models.RoamingEvent
	sessionRoaming map[int64][]models.RoamingEndpointInfo
	lookupCallback func()
}

func (m *MockStorageWithRoaming) GetEndpointOwner(_ context.Context, epEui []byte) (int64, error) {
	if m.lookupCallback != nil {
		m.lookupCallback()
	}

	// Normalize to uppercase hex for consistent map lookups
	epEuiHex := strings.ToUpper(hex.EncodeToString(epEui))

	if ep, ok := m.endpoints[epEuiHex]; ok {
		return ep.OwnerTenantID, nil
	}
	return 0, roaming.ErrEndpointNotFound
}

func (m *MockStorageWithRoaming) AreTenantsPartners(_ context.Context, tenant1, tenant2 int64) (bool, error) {
	key1 := fmt.Sprintf("%d-%d", tenant1, tenant2)
	key2 := fmt.Sprintf("%d-%d", tenant2, tenant1)
	return m.tenantPartners[key1] || m.tenantPartners[key2], nil
}

func (m *MockStorageWithRoaming) RecordRoamingEvent(_ context.Context, event *models.RoamingEvent) error {
	m.roamingEvents = append(m.roamingEvents, *event)
	return nil
}

func (m *MockStorageWithRoaming) AddRoamingEndpointToSession(_ context.Context, sessionID int64, epEui string, ownerTenantID int64) error {
	// Normalize to uppercase hex for consistent storage
	epEuiUpper := strings.ToUpper(epEui)
	m.sessionRoaming[sessionID] = append(m.sessionRoaming[sessionID], models.RoamingEndpointInfo{
		EUI:           epEuiUpper,
		OwnerTenantID: ownerTenantID,
		AttachedAt:    time.Now().Format(time.RFC3339),
	})
	return nil
}

func (m *MockStorageWithRoaming) Close() error {
	return nil
}

func (m *MockStorageWithRoaming) GetSessionRoamingEndpoints(_ context.Context, sessionID int64) ([]models.RoamingEndpointInfo, error) {
	return m.sessionRoaming[sessionID], nil
}

func (m *MockStorageWithRoaming) UpdateEndpointRoamingStatus(_ context.Context, epEui []byte, servingTenantID int64) error {
	// Normalize to uppercase hex for consistent map lookups
	epEuiHex := strings.ToUpper(hex.EncodeToString(epEui))

	if ep, ok := m.endpoints[epEuiHex]; ok {
		ep.TenantID = servingTenantID
		// In real implementation, would update serving_bs_eui field
	}
	return nil
}

// Additional stubs for complete interface
func (m *MockStorageWithRoaming) GetEndpointWithOwnership(_ context.Context, epEui []byte, _ int64) (*models.EndPoint, error) {
	// Normalize to uppercase hex for consistent map lookups
	epEuiHex := strings.ToUpper(hex.EncodeToString(epEui))
	return m.endpoints[epEuiHex], nil
}

func (m *MockStorageWithRoaming) IsRoamingEnabled(_ context.Context, _ int64) (bool, error) {
	return true, nil
}

func (m *MockStorageWithRoaming) GetRoamingStatistics(_ context.Context, _ int64) (*models.RoamingStatistics, error) {
	return &models.RoamingStatistics{
		VisitingRoamingIn: 10,
		OwnedRoamingOut:   5,
	}, nil
}

func (m *MockStorageWithRoaming) GetTenantRoamingConfig(_ context.Context, _ int64) (*models.TenantRoamingConfig, error) {
	return &models.TenantRoamingConfig{
		RoamingEnabled: true,
	}, nil
}

func (m *MockStorageWithRoaming) UpdateTenantRoamingConfig(_ context.Context, _ int64, _ *models.TenantRoamingConfig) error {
	return nil
}

// Additional stubs to satisfy storage.Storage interface
func (m *MockStorageWithRoaming) EnqueueDownlink(_ context.Context, _ *storage.DownlinkMessage) (*storage.DownlinkMessage, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) CreateEndPoint(_ context.Context, _ *models.EndPoint) (*models.EndPoint, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) GetEndPoint(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) UpdateEndPoint(_ context.Context, _ *models.EndPoint) (*models.EndPoint, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) DeleteEndPoint(_ context.Context, _ []byte, _ int64) error {
	return nil
}
func (m *MockStorageWithRoaming) ListEndPoints(_ context.Context, _ int64, _, _ int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) CreateBaseStation(_ context.Context, _ *models.BaseStation) (*models.BaseStation, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) GetBaseStation(_ context.Context, _ []byte, _ int64) (*models.BaseStation, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) UpdateBaseStation(_ context.Context, _ *models.BaseStation) (*models.BaseStation, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) DeleteBaseStation(_ context.Context, _ []byte, _ int64) error {
	return nil
}
func (m *MockStorageWithRoaming) ListBaseStations(_ context.Context, _ int64, _, _ int) ([]*models.BaseStation, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) GetDownlinkQueue(_ context.Context, _, _ string) ([]*storage.DownlinkMessage, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) GetDownlinkResults(_ context.Context, _, _ string, _ *uuid.UUID, _ string, _, _ *time.Time, _, _ int) ([]*storage.DownlinkMessage, int, error) {
	return nil, 0, nil
}
func (m *MockStorageWithRoaming) UpdateDownlinkStatus(_ context.Context, _, _ string, _ *uuid.UUID) error {
	return nil
}
func (m *MockStorageWithRoaming) UpdateDownlinkBaseStation(_ context.Context, _ uint64, _ string, _ uint64) error {
	return nil
}
func (m *MockStorageWithRoaming) GetDownlinkByQueueID(_ context.Context, _ uint64, _ string) (*storage.DownlinkMessage, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) GetDownlinkByPacketCnt(_ context.Context, _, _ string, _ uint32) (*storage.DownlinkMessage, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) RevokeDownlink(_ context.Context, _ int64, _ string) error {
	return nil
}
func (m *MockStorageWithRoaming) UpdateDownlinkResult(_ context.Context, _ int64, _ string, _ *int64, _ *uint32, _, _ []byte, _ string, _ *uuid.UUID) error {
	return nil
}
func (m *MockStorageWithRoaming) CreateDLRXStatus(_ context.Context, _ *mioty.DLRXStatus) error {
	return nil
}
func (m *MockStorageWithRoaming) GetDLRXStatusByEndpoint(_ context.Context, _ int64, _ []byte, _, _ int, _, _ *time.Time) ([]*mioty.DLRXStatus, int, error) {
	return nil, 0, nil
}
func (m *MockStorageWithRoaming) GetAverageDLRXMetrics(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (float64, float64, int, error) {
	return 0, 0, 0, nil
}
func (m *MockStorageWithRoaming) CreateDLRXStatusQuery(_ context.Context, _ int64, _ *uuid.UUID, _, _ []byte, _ int64) error {
	return nil
}
func (m *MockStorageWithRoaming) MarkDLRXStatusReceived(_ context.Context, _ int64, _ []byte, _ []byte, _ int64) (bool, error) {
	return false, nil
}
func (m *MockStorageWithRoaming) ExpireDLRXStatusQuery(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}
func (m *MockStorageWithRoaming) GetDLRXStatusQueryHistory(_ context.Context, _ int64, _ []byte, _, _ int, _, _ *time.Time) ([]*mioty.DLRXStatusQuery, int, error) {
	return nil, 0, nil
}
func (m *MockStorageWithRoaming) GetDLRXStatusQueryStats(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (int64, int64, int64, error) {
	return 0, 0, 0, nil
}
func (m *MockStorageWithRoaming) GetEndpointBaseStation(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (m *MockStorageWithRoaming) GetBaseStationMessageStats(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (*mioty.BaseStationMessageStats, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) GetBaseStationEndpointCounts(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (map[string]int64, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) GetBaseStationLastSeen(_ context.Context, _ int64, _ []byte) (*time.Time, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) Ping(_ context.Context) error { return nil }
func (m *MockStorageWithRoaming) RemoveRoamingEndpointFromSession(_ context.Context, sessionID int64, epEuiHex string) error {
	// Normalize to uppercase hex for consistent comparison
	epEuiHexUpper := strings.ToUpper(epEuiHex)

	endpoints := m.sessionRoaming[sessionID]
	filtered := []models.RoamingEndpointInfo{}
	for _, ep := range endpoints {
		if strings.ToUpper(ep.EUI) != epEuiHexUpper {
			filtered = append(filtered, ep)
		}
	}
	m.sessionRoaming[sessionID] = filtered
	return nil
}
func (m *MockStorageWithRoaming) GetRoamingEndpointsInSession(_ context.Context, sessionID int64) ([]models.RoamingEndpointInfo, error) {
	// Return actual data from sessionRoaming map
	if endpoints, ok := m.sessionRoaming[sessionID]; ok {
		return endpoints, nil
	}
	return []models.RoamingEndpointInfo{}, nil
}

func (m *MockStorageWithRoaming) UpdateBaseStationEUI(_ context.Context, _ int64, _, _ []byte) (*models.BaseStation, error) {
	return nil, nil
}
func (m *MockStorageWithRoaming) UpdateEndPointWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}
func (m *MockStorageWithRoaming) CheckEndPointEUIUnique(_ context.Context, _ []byte) error {
	return nil
}
func (m *MockStorageWithRoaming) ListAllBaseStationLocations(_ context.Context) ([]*models.BaseStation, error) {
	return nil, nil
}
