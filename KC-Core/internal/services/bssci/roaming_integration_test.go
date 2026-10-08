package bssciservices

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/roaming"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// Fixture policies for the roaming integration tests.
const (
	testRoamingAuditTrailEnabled = true
	testRoamingEnabled           = true
	testRoamingCacheTTL          = 100 * time.Millisecond
	testRoamingCacheExpiryWait   = 150 * time.Millisecond
	testVisitingRoamingIn        = 10
	testOwnedRoamingOut          = 5
	testRoamingCacheEnabled      = true
	testRoamingCacheLifetime     = 5 * time.Minute
	testRoamingCacheMaxSize      = 100
)

// testRoamingDetectorConfig is a cached, audited detector configuration.
func testRoamingDetectorConfig() roaming.DetectorConfig {
	return roaming.DetectorConfig{
		CacheEnabled:     testRoamingCacheEnabled,
		CacheTTL:         testRoamingCacheLifetime,
		CacheMaxSize:     testRoamingCacheMaxSize,
		EnableAuditTrail: testRoamingAuditTrailEnabled,
	}
}

func newTestRoamingDetector(t *testing.T, config roaming.DetectorConfig, resolver roaming.EndpointOwnershipResolver,
	recorder roaming.EventRecorder, clk clock.Clock,
) *roaming.Detector {
	t.Helper()
	detector, err := roaming.NewDetector(config, resolver, recorder, clk)
	require.NoError(t, err)
	return detector
}

// NewRoamingDetector applies the configured settings: with the cache off,
// every reception resolves ownership, and with the audit trail off no
// roaming event is written.
func TestNewRoamingDetector_AppliesTheConfiguredSettings(t *testing.T) {
	ctx := testutil.TestContext()
	epEui := []byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x15, 0x07}
	var lookups int
	mockStorage := &MockStorageWithRoaming{
		endpoints: map[string]*models.EndPoint{
			"70B3D56770111507": {ID: 458, TenantID: 3, OwnerTenantID: 3},
		},
		lookupCallback: func() { lookups++ },
	}
	detector, err := NewRoamingDetector(roaming.DetectorConfig{}, mockStorage, mockStorage, clock.SystemClock{})
	require.NoError(t, err)

	for range 2 {
		_, _, err = detector.DetectRoaming(ctx, epEui, 3)
		require.NoError(t, err)
	}
	assert.Equal(t, 2, lookups, "a disabled cache resolves ownership on every reception")
	require.NoError(t, detector.RecordAttachEvent(ctx, epEui, 3, 4, []byte{0x01}))
	assert.Empty(t, mockStorage.roamingEvents, "a disabled audit trail records no roaming event")
}

// Integration test for complete roaming flow
func TestRoamingIntegration_AttachDetachFlow(t *testing.T) {
	// This test simulates a complete roaming scenario:
	// 1. Endpoint from tenant 1 attaches to base station owned by tenant 2
	// 2. System detects roaming and validates partnership
	// 3. Messages are routed to owner tenant storage
	// 4. Detach cleans up roaming state

	// Setup test environment
	ctx := testutil.TestContext()

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
		sessionRoaming: make(map[int64][]roamingEndpointInfo),
	}

	// Create roaming configuration using production defaults
	roamingConfig := testRoamingDetectorConfig()

	// Create roaming service
	detector := newTestRoamingDetector(t, roamingConfig, mockStorage, mockStorage, clock.SystemClock{})
	roamingSvc := NewRoamingService(detector, mockStorage, logger.NewNop())

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
		assert.Equal(t, mioty.FormatEUIBytes(epEui), endpoints[0].EUI)
		assert.Equal(t, ownerTenantID, endpoints[0].OwnerTenantID)
	})

	// Test 2: UL Data routing to owner tenant
	t.Run("ULDataRouting", func(t *testing.T) {
		// Resolve owner for UL data
		_, resolvedOwner, err := roamingSvc.DetectAndValidateRoaming(ctx, epEui, servingTenantID)
		require.NoError(t, err)
		assert.Equal(t, ownerTenantID, resolvedOwner)
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
	ctx := testutil.TestContext()

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

	roamingConfig := roaming.DetectorConfig{
		EnableAuditTrail: testRoamingAuditTrailEnabled,
	}

	detector := newTestRoamingDetector(t, roamingConfig, mockStorage, mockStorage, clock.SystemClock{})
	roamingSvc := NewRoamingService(detector, mockStorage, logger.NewNop())

	// Attempt to validate roaming without partnership
	isRoaming, _, err := roamingSvc.DetectAndValidateRoaming(ctx, epEui, servingTenantID)

	// Should fail due to missing partnership
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no roaming agreement between tenants")
	assert.False(t, isRoaming)

	// Verify no events were recorded
	assert.Len(t, mockStorage.roamingEvents, 0)
}

// A reception through the owner's station caches the owner; a later
// reception through a station without a roaming agreement is still refused.
func TestRoamingIntegration_NonPartnerStationRefusedAfterOwnerReception(t *testing.T) {
	ctx := testutil.TestContext()
	epEui := []byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x15, 0x06}
	const ownerTenantID, strangerTenantID = int64(5), int64(9)
	mockStorage := &MockStorageWithRoaming{
		endpoints: map[string]*models.EndPoint{
			"70B3D56770111506": {ID: 457, TenantID: ownerTenantID, OwnerTenantID: ownerTenantID},
		},
		tenantPartners: map[string]bool{},
	}
	detector := newTestRoamingDetector(t, testRoamingDetectorConfig(), mockStorage, mockStorage, clock.SystemClock{})
	roamingSvc := NewRoamingService(detector, mockStorage, logger.NewNop())

	isRoaming, owner, err := roamingSvc.DetectAndValidateRoaming(ctx, epEui, ownerTenantID)
	require.NoError(t, err)
	assert.False(t, isRoaming)
	assert.Equal(t, ownerTenantID, owner)

	_, _, err = roamingSvc.DetectAndValidateRoaming(ctx, epEui, strangerTenantID)
	require.Error(t, err, "a station without a roaming agreement must not take the owner's uplinks")
	assert.ErrorIs(t, err, errRoamingNotAllowed)
}

// Test cache expiration and refresh
func TestRoamingIntegration_CacheExpiration(t *testing.T) {
	ctx := testutil.TestContext()

	epEui := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	ownerTenantID := int64(3)
	servingTenantID := int64(4)

	var lookupCount int
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
	roamingConfig := testRoamingDetectorConfig()
	roamingConfig.CacheTTL = testRoamingCacheTTL

	detector := newTestRoamingDetector(t, roamingConfig, mockStorage, mockStorage, clock.SystemClock{})

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
	time.Sleep(testRoamingCacheExpiryWait)

	// Third lookup - cache expired, should hit database
	isRoaming3, owner3, err := detector.DetectRoaming(ctx, epEui, servingTenantID)
	require.NoError(t, err)
	assert.True(t, isRoaming3)
	assert.Equal(t, ownerTenantID, owner3)
	assert.Equal(t, 2, lookupCount) // Additional lookup after expiry
}

// Test metrics collection
func TestRoamingIntegration_Metrics(t *testing.T) {
	ctx := testutil.TestContext()

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
	roamingConfig := testRoamingDetectorConfig()

	detector := newTestRoamingDetector(t, roamingConfig, mockStorage, mockStorage, clock.SystemClock{})
	roamingSvc := NewRoamingService(detector, mockStorage, logger.NewNop())

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

// roamingEndpointInfo mirrors one entry of the basestation_sessions.roaming_endpoints JSON array.
type roamingEndpointInfo struct {
	EUI           string `json:"eui"`
	OwnerTenantID int64  `json:"ownerTenantId"`
	AttachedAt    string `json:"attachedAt"`
}

// MockStorageWithRoaming implements full roaming storage interface for testing
type MockStorageWithRoaming struct {
	endpoints      map[string]*models.EndPoint
	tenantPartners map[string]bool
	roamingEvents  []models.RoamingEvent
	sessionRoaming map[int64][]roamingEndpointInfo
	lookupCallback func()
}

func (m *MockStorageWithRoaming) GetEndpointOwner(_ context.Context, epEui []byte) (int64, error) {
	if m.lookupCallback != nil {
		m.lookupCallback()
	}

	// Normalize to uppercase hex for consistent map lookups
	epEuiHex := mioty.FormatEUIBytes(epEui)

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
	m.sessionRoaming[sessionID] = append(m.sessionRoaming[sessionID], roamingEndpointInfo{
		EUI:           epEuiUpper,
		OwnerTenantID: ownerTenantID,
		AttachedAt:    time.Now().Format(time.RFC3339),
	})
	return nil
}

func (m *MockStorageWithRoaming) Close() error {
	return nil
}

func (m *MockStorageWithRoaming) GetSessionRoamingEndpoints(_ context.Context, sessionID int64) ([]roamingEndpointInfo, error) {
	return m.sessionRoaming[sessionID], nil
}

// Additional stubs for complete interface
func (m *MockStorageWithRoaming) GetEndpointWithOwnership(_ context.Context, epEui []byte, _ int64) (*models.EndPoint, error) {
	// Normalize to uppercase hex for consistent map lookups
	epEuiHex := mioty.FormatEUIBytes(epEui)
	return m.endpoints[epEuiHex], nil
}

func (m *MockStorageWithRoaming) IsRoamingEnabled(_ context.Context, _ int64) (bool, error) {
	return true, nil
}

// Additional stubs to satisfy storage.Storage interface
func (m *MockStorageWithRoaming) EnqueueDownlink(_ context.Context, _ *storage.DownlinkMessage, _ time.Duration) (*storage.DownlinkMessage, error) {
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

func (m *MockStorageWithRoaming) ListEndPointsByModelWithSnapshot(_ context.Context, _ int64, _ uuid.UUID) ([]*models.EndPoint, error) {
	return nil, nil
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

func (m *MockStorageWithRoaming) UpdatePendingDownlink(_ context.Context, _ int64, _ *uuid.UUID, _ []byte, _ int64, _ storage.DownlinkPatch) (*storage.DownlinkMessage, error) {
	return nil, nil
}

func (m *MockStorageWithRoaming) GetDownlinkResults(_ context.Context, _ int64, _ *uuid.UUID, _ storage.DownlinkResultFilter, _, _ int) ([]*storage.DownlinkMessage, int, error) {
	return nil, 0, nil
}

func (m *MockStorageWithRoaming) UpdateDownlinkStatus(_ context.Context, _ string, _ mioty.DLQueueStatus, _ *uuid.UUID) error {
	return nil
}

func (m *MockStorageWithRoaming) GetDownlinkByQueueID(_ context.Context, _ uint64, _ string) (*storage.DownlinkMessage, error) {
	return nil, nil
}

func (m *MockStorageWithRoaming) RevokeDownlink(context.Context, storage.DownlinkRevocation) (bool, error) {
	return true, nil
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
	filtered := []roamingEndpointInfo{}
	for _, ep := range endpoints {
		if strings.ToUpper(ep.EUI) != epEuiHexUpper {
			filtered = append(filtered, ep)
		}
	}
	m.sessionRoaming[sessionID] = filtered
	return nil
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
