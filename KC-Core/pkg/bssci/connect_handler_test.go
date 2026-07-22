package bssci_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	bssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// mockConnectionService implements ConnectionService for testing error catalog behavior
type mockConnectionService struct {
	shouldFail      bool
	globalCalled    bool
	registerCalled  bool
	tenantID        int64 // tenant ID returned by GetBaseStationGlobal (default 1)
	lastRegisterCtx context.Context
}

func (m *mockConnectionService) GetBaseStationGlobal(_ context.Context, _ [8]byte) (*basestation.BaseStation, error) {
	m.globalCalled = true
	if m.shouldFail {
		return nil, errors.New("base station not found in database")
	}
	tid := m.tenantID
	if tid == 0 {
		tid = 1
	}
	return &basestation.BaseStation{
		ID:       1,
		TenantID: tid,
		EUI:      [8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
		Name:     "Test Base Station",
	}, nil
}

func (m *mockConnectionService) SaveSessionEncoding(_ context.Context, _ int64, _ string) error {
	return nil
}

func (m *mockConnectionService) RegisterConnection(ctx context.Context, _ *bssci.Session, _ *basestation.BaseStation) error {
	m.registerCalled = true
	m.lastRegisterCtx = ctx
	return nil
}

// mockConnForConnect implements net.Conn to capture error messages sent during connect
type mockConnForConnect struct {
	errorSent        bool
	lastErrorCode    int
	lastErrorMessage string
	data             []byte
}

func (m *mockConnForConnect) Read(_ []byte) (n int, err error) { return 0, nil }
func (m *mockConnForConnect) Write(b []byte) (n int, err error) {
	m.data = b
	// Skip BSSCI headers by checking protocol identifier
	if len(b) == 12 && bytes.HasPrefix(b, mioty.MIOTYFrameIdentifier[:]) {
		return len(b), nil
	}
	// Decode msgpack payloads to detect error frames
	var msg map[string]interface{}
	if err := msgpack.Unmarshal(b, &msg); err == nil {
		if cmd, ok := msg["command"].(string); ok && cmd == "error" {
			m.errorSent = true
			// Capture error code
			if code, ok := msg["code"]; ok {
				switch v := code.(type) {
				case int:
					m.lastErrorCode = v
				case int8:
					m.lastErrorCode = int(v)
				case int16:
					m.lastErrorCode = int(v)
				case int32:
					m.lastErrorCode = int(v)
				case int64:
					m.lastErrorCode = int(v)
				case uint8:
					m.lastErrorCode = int(v)
				case uint16:
					m.lastErrorCode = int(v)
				case uint32:
					m.lastErrorCode = int(v)
				case uint64:
					m.lastErrorCode = int(v)
				case float64:
					m.lastErrorCode = int(v)
				case float32:
					m.lastErrorCode = int(v)
				default:
					// Debug: unknown type
					_ = v
				}
			}
			// Capture error message
			if message, ok := msg["message"].(string); ok {
				m.lastErrorMessage = message
			}
		}
	}
	return len(b), nil
}
func (m *mockConnForConnect) Close() error { return nil }
func (m *mockConnForConnect) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5000}
}
func (m *mockConnForConnect) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
}
func (m *mockConnForConnect) SetDeadline(_ time.Time) error      { return nil }
func (m *mockConnForConnect) SetReadDeadline(_ time.Time) error  { return nil }
func (m *mockConnForConnect) SetWriteDeadline(_ time.Time) error { return nil }

// TestConnectHandlerErrorCatalogCompliance verifies that the error catalog token is properly
// defined and returns the expected message for unregistered base stations
func TestConnectHandlerErrorCatalogCompliance(t *testing.T) {
	// Verify: Error catalog token exists and is not empty
	token := bssci.ErrBaseStationNotRegistered
	assert.NotEmpty(t, token, "Error token should be defined")

	// Verify: ResolveErrorMessage returns proper catalog message (not literal)
	message := bssci.ResolveErrorMessage(bssci.ErrBaseStationNotRegistered)
	assert.NotEmpty(t, message, "Error message should not be empty")
	assert.Contains(t, strings.ToLower(message), "base station", "Error message should mention base station")
	assert.NotContains(t, message, "hardcoded", "Error message should not be a hardcoded literal")

	// Verify: Message is consistent (calling multiple times returns same result)
	message2 := bssci.ResolveErrorMessage(bssci.ErrBaseStationNotRegistered)
	assert.Equal(t, message, message2, "Error catalog should return consistent messages")

	t.Logf("PASS: Error catalog token: %s", token)
	t.Logf("PASS: Error catalog message: %s", message)
}

// TestHandleConnectCompleteUnregisteredBaseStationUsesErrorCatalog verifies that when an unregistered
// base station completes connection, the handler uses the error catalog instead of literal strings
func TestHandleConnectCompleteUnregisteredBaseStationUsesErrorCatalog(t *testing.T) {
	// Setup: Create test logger and mock services
	testLogger := &NopLogger{}

	// Create mock ConnectionService that returns error (simulating unregistered base station)
	mockConnectionSvc := &mockConnectionService{shouldFail: true}

	// Create other required services (using defaults)
	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, nil)

	// Create test server with mock ConnectionService
	server := bssci.NewTestServer(
		testLogger,
		nil, // No storage needed for this test
		nil, // No event store needed
		1,   // tenant ID
		sessionSvc,
		downlinkSvc,
		statusSvc,
		mockConnectionSvc, // Inject mock that will fail
		broadcaster,
		queueSerializer,
		auditLogger,
		tenantResolver,
	)

	// Set minimal config to avoid nil pointer errors (test focuses on GetBaseStation failure)
	server.SetConfig(&bssci.Config{
		ServiceCenterEUI: bssci.TestScEui01,
		Vendor:           "Test Vendor",
		Model:            "Test Model",
		Name:             "Test SC",
		SoftwareVersion:  "1.0.0",
	})

	// Set connection manager to nil (not needed for this test since we're mocking GetBaseStation)
	server.SetConnectionManager(nil)

	// Create test session with mock connection
	bsEUI := bssci.TestBsEui01
	mockConn := &mockConnForConnect{}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:             "test-session-1",
			BaseStationEUI: bsEUI,
			SessionUUID:    make([]byte, 16), // Required for handleConnectComplete
		},
		UserProvidedName: "Unregistered BS",
		Conn:             mockConn,
	}

	// Registration is validated during the connect request, before conRsp
	connectData := map[string]interface{}{
		"version": mioty.MIOTYProtocolVersion,
		"bsEui":   int64(0x0123456789ABCDEF),
		"bidi":    true,
	}
	msg := &bssci.Message{OpId: 0, Command: "con", Data: connectData}
	require.NoError(t, server.CallHandleConnect(session, msg, connectData),
		"rejected connect awaits errorAck instead of closing")

	// Primary verification: Error was sent to base station using catalog
	// (This is the key requirement - using error catalog instead of literals)
	require.True(t, mockConn.errorSent, "Error should be sent to base station when lookup fails")
	assert.Equal(t, bssci.POSIX_EPERM, mockConn.lastErrorCode,
		"Error code should be POSIX_EPERM (operation not permitted)")

	// Verify: Error message from catalog is used (not hardcoded literal)
	expectedMessage := bssci.ResolveErrorMessage(bssci.ErrBaseStationNotRegistered)
	assert.Equal(t, expectedMessage, mockConn.lastErrorMessage,
		"Error message should come from error catalog, not be hardcoded")

	// Verify: GetBaseStationGlobal was called before any conRsp
	assert.True(t, mockConnectionSvc.globalCalled, "GetBaseStationGlobal should be called during connect")
}

// TestConnectAdoptsRegisteredTenantCommunityFallback verifies that with
// organization enforcement disabled, the connect request adopts the base
// station's registered tenant before the response is offered.
func TestConnectAdoptsRegisteredTenantCommunityFallback(t *testing.T) {
	testLogger := &NopLogger{}

	// Mock returns base station registered under tenant 4 (not server default 1)
	mockConnectionSvc := &mockConnectionService{tenantID: 4}

	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, nil)

	server := bssci.NewTestServer(
		testLogger,
		nil,
		nil,
		1, // server default tenant = 1
		sessionSvc,
		downlinkSvc,
		statusSvc,
		mockConnectionSvc,
		broadcaster,
		queueSerializer,
		auditLogger,
		tenantResolver,
	)

	server.SetConfig(&bssci.Config{
		ServiceCenterEUI: bssci.TestScEui01,
		Vendor:           "Test Vendor",
		Model:            "Test Model",
		Name:             "Test SC",
		SoftwareVersion:  "1.0.0",
	})
	server.SetConnectionManager(nil)

	mockConn := &mockConnForConnect{}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:               "test-cross-tenant",
			BaseStationEUI:   bssci.TestBsEui01,
			SessionUUID:      make([]byte, 16),
			ResolvedTenantID: 1,
			// Starts with server default
			IsResumed: true,
		},
		UserProvidedName: "External BS",
		Conn:             mockConn,
		// Skip PersistSession DbSessionID assignment (avoids nil storage in loadPendingOperations)
	}

	connectData := map[string]interface{}{
		"version": mioty.MIOTYProtocolVersion,
		"bsEui":   int64(0x0123456789ABCDEF),
		"bidi":    true,
	}
	conMsg := &bssci.Message{OpId: 0, Command: "con", Data: connectData}
	require.NoError(t, server.CallHandleConnect(session, conMsg, connectData))

	// GetBaseStationGlobal was called (not tenant-scoped GetBaseStation)
	assert.True(t, mockConnectionSvc.globalCalled,
		"connect must use GetBaseStationGlobal for tenant-agnostic lookup")

	// Community fallback adopts the registered tenant 1 → 4
	assert.Equal(t, int64(4), session.ResolvedTenantID,
		"Session ResolvedTenantID must adopt the base station's registered tenant")

	// No error sent (base station found)
	assert.False(t, mockConn.errorSent, "No error should be sent for registered base station")

	msg := &bssci.Message{OpId: 0, Command: "conCmp"}
	_ = server.CallHandleConnectComplete(session, msg, map[string]interface{}{})

	// RegisterConnection was called (connection status updated)
	assert.True(t, mockConnectionSvc.registerCalled,
		"RegisterConnection must be called after successful completion")
}

// TestConnectTenantMismatchRejected verifies that with organization
// enforcement enabled, a certificate tenant that does not match the base
// station's registered tenant is rejected without revealing that the EUI
// exists under another tenant.
func TestConnectTenantMismatchRejected(t *testing.T) {
	testLogger := &NopLogger{}

	mockConnectionSvc := &mockConnectionService{tenantID: 4}

	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, nil)

	server := bssci.NewTestServer(
		testLogger, nil, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, mockConnectionSvc,
		broadcaster, queueSerializer, auditLogger, tenantResolver,
	)
	server.SetConfig(&bssci.Config{
		ServiceCenterEUI:      bssci.TestScEui01,
		Vendor:                "Test Vendor",
		Model:                 "Test Model",
		Name:                  "Test SC",
		SoftwareVersion:       "1.0.0",
		OrgEnforcementEnabled: true,
	})
	server.SetConnectionManager(nil)

	mockConn := &mockConnForConnect{}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:               "test-tenant-mismatch",
			BaseStationEUI:   bssci.TestBsEui01,
			SessionUUID:      make([]byte, 16),
			ResolvedTenantID: 1,
		},
		Conn: mockConn,
		// Certificate resolved to tenant 1; BS registered under 4
	}

	connectData := map[string]interface{}{
		"version": mioty.MIOTYProtocolVersion,
		"bsEui":   int64(0x0123456789ABCDEF),
		"bidi":    true,
	}
	conMsg := &bssci.Message{OpId: 0, Command: "con", Data: connectData}
	require.NoError(t, server.CallHandleConnect(session, conMsg, connectData),
		"rejected connect awaits errorAck instead of closing")

	require.True(t, mockConn.errorSent, "Tenant mismatch must be rejected")
	assert.Equal(t, bssci.POSIX_EPERM, mockConn.lastErrorCode)
	assert.Equal(t, bssci.ResolveErrorMessage(bssci.ErrBaseStationNotRegistered), mockConn.lastErrorMessage,
		"Rejection must not reveal that the EUI exists under another tenant")
	assert.Equal(t, int64(1), session.ResolvedTenantID,
		"Certificate tenant must not be overwritten on rejection")
}

// TestConnectCompleteDefaultTenantUnchanged verifies that when a base station
// matches the server's default tenant, the session tenant is not disturbed.
func TestConnectCompleteDefaultTenantUnchanged(t *testing.T) {
	testLogger := &NopLogger{}

	// Mock returns base station under server's default tenant (1)
	mockConnectionSvc := &mockConnectionService{tenantID: 1}

	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, nil)

	server := bssci.NewTestServer(
		testLogger,
		nil,
		nil,
		1,
		sessionSvc,
		downlinkSvc,
		statusSvc,
		mockConnectionSvc,
		broadcaster,
		queueSerializer,
		auditLogger,
		tenantResolver,
	)

	server.SetConfig(&bssci.Config{
		ServiceCenterEUI: bssci.TestScEui01,
		Vendor:           "Test Vendor",
		Model:            "Test Model",
		Name:             "Test SC",
		SoftwareVersion:  "1.0.0",
	})
	server.SetConnectionManager(nil)

	mockConn := &mockConnForConnect{}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:               "test-default-tenant",
			BaseStationEUI:   bssci.TestBsEui01,
			SessionUUID:      make([]byte, 16),
			ResolvedTenantID: 1,
			IsResumed:        true,
		},
		UserProvidedName: "Local BS",
		Conn:             mockConn,
	}

	connectData := map[string]interface{}{
		"version": mioty.MIOTYProtocolVersion,
		"bsEui":   int64(0x0123456789ABCDEF),
		"bidi":    true,
	}
	conMsg := &bssci.Message{OpId: 0, Command: "con", Data: connectData}
	require.NoError(t, server.CallHandleConnect(session, conMsg, connectData))

	msg := &bssci.Message{OpId: 0, Command: "conCmp"}
	_ = server.CallHandleConnectComplete(session, msg, map[string]interface{}{})

	// Tenant stays at 1
	assert.Equal(t, int64(1), session.ResolvedTenantID,
		"Session tenant should remain at server default when base station matches")
	assert.False(t, mockConn.errorSent, "No error for registered base station")
}

// TestConnectHandler_ValidGeoLocation_PersistsToDB verifies BSSCI §3.3.1:
// When conRsp includes a valid geoLocation array with in-range lat/lon, the connect
// handshake must persist latitude, longitude, altitude, location_source, and location_updated_at.
func TestConnectHandler_ValidGeoLocation_PersistsToDB(t *testing.T) {
	testLogger := &NopLogger{}

	mockConnectionSvc := &mockConnectionService{tenantID: 1}
	trackingRepo := &geoLocationTrackingRepo{}

	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssci.CreateTestServices(testLogger, nil)

	server := bssci.NewTestServerWithBaseStationRepo(testLogger, mockStorage, trackingRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, mockConnectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	server.SetConfig(&bssci.Config{
		ServiceCenterEUI: bssci.TestScEui01,
		Vendor:           "Test Vendor",
		Model:            "Test Model",
		Name:             "Test SC",
		SoftwareVersion:  "1.0.0",
	})
	server.SetConnectionManager(nil)

	mockConn := &mockConnForConnect{}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:             "test-geo-connect-valid",
			BaseStationEUI: bssci.TestBsEui01,
			SessionUUID:    make([]byte, 16),
		},
		UserProvidedName: "GeoLocation BS",
		Conn:             mockConn,
	}

	// handleConnect parses geoLocation from conRsp data into session.GeoLocation
	connectData := map[string]interface{}{
		"version":     mioty.MIOTYProtocolVersion,
		"bsEui":       int64(0x0123456789ABCDEF),
		"bidi":        true,
		"geoLocation": []interface{}{float64(48.8566), float64(2.3522), float64(35.0)},
	}
	connectMsg := &bssci.Message{OpId: 0, Command: "conRsp", Data: connectData}
	err := server.CallHandleConnect(session, connectMsg, connectData)
	require.NoError(t, err, "handleConnect should parse geoLocation successfully")

	// Verify session.GeoLocation was populated
	require.Len(t, session.GeoLocation, 3, "session.GeoLocation should have 3 elements")
	assert.Equal(t, float64(48.8566), session.GeoLocation[0])
	assert.Equal(t, float64(2.3522), session.GeoLocation[1])
	assert.Equal(t, float64(35.0), session.GeoLocation[2])

	beforeComplete := time.Now()

	// handleConnectComplete persists session.GeoLocation via basestationRepo.Update
	session.IsResumed = true // Skip PersistSession (avoids nil storage)
	conCmpMsg := &bssci.Message{OpId: 0, Command: "conCmp"}
	_ = server.CallHandleConnectComplete(session, conCmpMsg, map[string]interface{}{})

	// Verify basestationRepo.Update was called with location fields
	assert.True(t, trackingRepo.updateCalled, "basestationRepo.Update should be called")
	assert.Contains(t, trackingRepo.updatesMap, "latitude")
	assert.Equal(t, float64(48.8566), trackingRepo.updatesMap["latitude"])
	assert.Contains(t, trackingRepo.updatesMap, "longitude")
	assert.Equal(t, float64(2.3522), trackingRepo.updatesMap["longitude"])
	assert.Contains(t, trackingRepo.updatesMap, "altitude")
	assert.Equal(t, float64(35.0), trackingRepo.updatesMap["altitude"])
	assert.Contains(t, trackingRepo.updatesMap, "location_source")
	assert.Equal(t, models.LocationSourceGPS, trackingRepo.updatesMap["location_source"])
	assert.Contains(t, trackingRepo.updatesMap, "location_updated_at")
	updatedAt, ok := trackingRepo.updatesMap["location_updated_at"].(time.Time)
	require.True(t, ok, "location_updated_at should be a time.Time")
	assert.False(t, updatedAt.Before(beforeComplete), "location_updated_at should be recent")
}

// TestConnectHandler_InvalidGeoLocation_NoPersistence verifies BSSCI §3.3.1:
// When geoLocation cannot be parsed (wrong type), session.GeoLocation stays nil
// and handleConnectComplete must not include location fields in the DB update.
func TestConnectHandler_InvalidGeoLocation_NoPersistence(t *testing.T) {
	testLogger := &NopLogger{}

	mockConnectionSvc := &mockConnectionService{tenantID: 1}
	trackingRepo := &geoLocationTrackingRepo{}

	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssci.CreateTestServices(testLogger, nil)

	server := bssci.NewTestServerWithBaseStationRepo(testLogger, mockStorage, trackingRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, mockConnectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	server.SetConfig(&bssci.Config{
		ServiceCenterEUI: bssci.TestScEui01,
		Vendor:           "Test Vendor",
		Model:            "Test Model",
		Name:             "Test SC",
		SoftwareVersion:  "1.0.0",
	})
	server.SetConnectionManager(nil)

	mockConn := &mockConnForConnect{}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:             "test-geo-connect-invalid",
			BaseStationEUI: bssci.TestBsEui01,
			SessionUUID:    make([]byte, 16),
		},
		UserProvidedName: "Invalid Geo BS",
		Conn:             mockConn,
	}

	// geoLocation as a string (wrong type — should be array)
	connectData := map[string]interface{}{
		"version":     mioty.MIOTYProtocolVersion,
		"bsEui":       int64(0x0123456789ABCDEF),
		"bidi":        true,
		"geoLocation": "48.8566,2.3522,35.0",
	}
	connectMsg := &bssci.Message{OpId: 0, Command: "conRsp", Data: connectData}
	err := server.CallHandleConnect(session, connectMsg, connectData)
	require.NoError(t, err, "handleConnect should succeed even with unparseable geoLocation")

	// session.GeoLocation should be nil (parse failure)
	assert.Nil(t, session.GeoLocation, "session.GeoLocation should be nil for invalid format")

	// handleConnectComplete should not persist location fields
	session.IsResumed = true
	conCmpMsg := &bssci.Message{OpId: 0, Command: "conCmp"}
	_ = server.CallHandleConnectComplete(session, conCmpMsg, map[string]interface{}{})

	assert.True(t, trackingRepo.updateCalled, "basestationRepo.Update should still be called (for bidi)")
	assert.NotContains(t, trackingRepo.updatesMap, "latitude", "Latitude should not be in updates")
	assert.NotContains(t, trackingRepo.updatesMap, "longitude", "Longitude should not be in updates")
	assert.NotContains(t, trackingRepo.updatesMap, "altitude", "Altitude should not be in updates")
	assert.NotContains(t, trackingRepo.updatesMap, "location_source", "location_source should not be in updates")
	assert.NotContains(t, trackingRepo.updatesMap, "location_updated_at", "location_updated_at should not be in updates")
}

// TestConnectHandler_OutOfRangeGeoLocation_NoPersistence verifies BSSCI §3.3.1:
// When geoLocation has out-of-range coordinates (e.g., lat=200), session.GeoLocation
// stays nil and no location fields are persisted.
func TestConnectHandler_OutOfRangeGeoLocation_NoPersistence(t *testing.T) {
	testLogger := &NopLogger{}

	mockConnectionSvc := &mockConnectionService{tenantID: 1}
	trackingRepo := &geoLocationTrackingRepo{}

	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssci.CreateTestServices(testLogger, nil)

	server := bssci.NewTestServerWithBaseStationRepo(testLogger, mockStorage, trackingRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, mockConnectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	server.SetConfig(&bssci.Config{
		ServiceCenterEUI: bssci.TestScEui01,
		Vendor:           "Test Vendor",
		Model:            "Test Model",
		Name:             "Test SC",
		SoftwareVersion:  "1.0.0",
	})
	server.SetConnectionManager(nil)

	mockConn := &mockConnForConnect{}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:             "test-geo-connect-outofrange",
			BaseStationEUI: bssci.TestBsEui01,
			SessionUUID:    make([]byte, 16),
		},
		UserProvidedName: "OutOfRange Geo BS",
		Conn:             mockConn,
	}

	// Latitude = 200.0 exceeds LatitudeMax (90.0)
	connectData := map[string]interface{}{
		"version":     mioty.MIOTYProtocolVersion,
		"bsEui":       int64(0x0123456789ABCDEF),
		"bidi":        true,
		"geoLocation": []interface{}{float64(200.0), float64(13.4050), float64(34.0)},
	}
	connectMsg := &bssci.Message{OpId: 0, Command: "conRsp", Data: connectData}
	err := server.CallHandleConnect(session, connectMsg, connectData)
	require.NoError(t, err, "handleConnect should succeed even with out-of-range geoLocation")

	// session.GeoLocation should be nil (out-of-range lat rejected)
	assert.Nil(t, session.GeoLocation, "session.GeoLocation should be nil for out-of-range lat")

	// handleConnectComplete should not persist location fields
	session.IsResumed = true
	conCmpMsg := &bssci.Message{OpId: 0, Command: "conCmp"}
	_ = server.CallHandleConnectComplete(session, conCmpMsg, map[string]interface{}{})

	assert.True(t, trackingRepo.updateCalled, "basestationRepo.Update should still be called (for bidi)")
	assert.NotContains(t, trackingRepo.updatesMap, "latitude", "Latitude should not be in updates for out-of-range")
	assert.NotContains(t, trackingRepo.updatesMap, "longitude", "Longitude should not be in updates for out-of-range")
	assert.NotContains(t, trackingRepo.updatesMap, "altitude", "Altitude should not be in updates for out-of-range")
	assert.NotContains(t, trackingRepo.updatesMap, "location_source", "location_source should not be set for out-of-range")
	assert.NotContains(t, trackingRepo.updatesMap, "location_updated_at", "location_updated_at should not be set for out-of-range")
}

func (m *geoLocationTrackingRepo) UpdateTLSFingerprintIfBlank(_ context.Context, _, _ int64, _ string) (bool, error) {
	return true, nil
}
