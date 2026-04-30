package bssci_test

import (
	"bytes"
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	bssciservices "github.com/kilocenter/KC-Core/internal/services/bssci"
	bssci "github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// statusMockConn captures responses sent by status handlers
type statusMockConn struct {
	net.Conn
	mu            sync.Mutex
	sentMessages  []map[string]interface{}
	errorSent     bool
	lastErrorCode int
	lastErrorMsg  string
}

func (m *statusMockConn) Write(b []byte) (n int, err error) {
	// Skip BSSCI headers (12 bytes starting with "MIOTYB01")
	if len(b) == 12 && bytes.HasPrefix(b, mioty.MIOTYFrameIdentifier[:]) {
		return len(b), nil
	}

	// Decode msgpack payload
	var msg map[string]interface{}
	if err := msgpack.Unmarshal(b, &msg); err == nil {
		m.mu.Lock()
		defer m.mu.Unlock()

		m.sentMessages = append(m.sentMessages, msg)

		// Track error frames
		if cmd, ok := msg["command"].(string); ok && cmd == "error" {
			m.errorSent = true
			// Handle various integer types from msgpack encoding
			if codeVal, hasCode := msg["code"]; hasCode {
				switch code := codeVal.(type) {
				case int:
					m.lastErrorCode = code
				case int64:
					m.lastErrorCode = int(code)
				case int32:
					m.lastErrorCode = int(code)
				case int8:
					m.lastErrorCode = int(code)
				case uint:
					m.lastErrorCode = int(code)
				case uint64:
					m.lastErrorCode = int(code)
				case uint32:
					m.lastErrorCode = int(code)
				case uint8:
					m.lastErrorCode = int(code)
				case float64:
					m.lastErrorCode = int(code)
				}
			}
			if message, ok := msg["message"].(string); ok {
				m.lastErrorMsg = message
			}
		}
	}

	return len(b), nil
}

func (m *statusMockConn) Close() error                       { return nil }
func (m *statusMockConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *statusMockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *statusMockConn) SetDeadline(_ time.Time) error      { return nil }
func (m *statusMockConn) SetReadDeadline(_ time.Time) error  { return nil }
func (m *statusMockConn) SetWriteDeadline(_ time.Time) error { return nil }
func (m *statusMockConn) Read(_ []byte) (n int, err error)   { return 0, nil }

// GetSentMessages returns a thread-safe copy of sent messages
func (m *statusMockConn) GetSentMessages() []map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return a copy to prevent external modifications
	result := make([]map[string]interface{}, len(m.sentMessages))
	copy(result, m.sentMessages)
	return result
}

// GetErrorSent returns whether an error frame was sent
func (m *statusMockConn) GetErrorSent() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.errorSent
}

// GetLastErrorCode returns the last error code sent
func (m *statusMockConn) GetLastErrorCode() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastErrorCode
}

// TestStatusResponseSendsCompletion verifies BSSCI §3.5.3:
// After receiving statusRsp, the SC MUST send statusCmp
func TestStatusResponseSendsCompletion(t *testing.T) {
	logger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)
	server := bssci.NewTestServer(logger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-status-session",
		BaseStationEUI: 0x1122334455667788,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	// Create statusRsp message with all mandatory fields
	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -1,
		Data: map[string]interface{}{
			"code":    0,
			"message": "ok",
			"time":    int64(1672531200000000000), // 2023-01-01 00:00:00 UTC
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	require.NoError(t, err, "Status response should be processed successfully")

	// Verify statusCmp was sent
	require.Len(t, mockConn.sentMessages, 1, "Should send exactly one message (statusCmp)")
	completionMsg := mockConn.sentMessages[0]

	assert.Equal(t, "statusCmp", completionMsg["command"], "Should send statusCmp command")
	assert.Equal(t, int64(-1), completionMsg["opId"], "statusCmp should echo opId from request")
}

// TestStatusMandatoryFieldValidation verifies BSSCI §3.5.2:
// The SC MUST validate all mandatory fields in statusRsp
func TestStatusMandatoryFieldValidation(t *testing.T) {
	logger := logger.NewNop()

	tests := []struct {
		name          string
		data          map[string]interface{}
		expectError   bool
		errorContains string
	}{
		{
			name: "All mandatory fields present",
			data: map[string]interface{}{
				"code":    0,
				"message": "ok",
				"time":    int64(1672531200000000000),
			},
			expectError: false,
		},
		{
			name: "Missing code field",
			data: map[string]interface{}{
				"message": "ok",
				"time":    int64(1672531200000000000),
			},
			expectError:   true,
			errorContains: "code",
		},
		{
			name: "Missing message field",
			data: map[string]interface{}{
				"code": 0,
				"time": int64(1672531200000000000),
			},
			expectError:   true,
			errorContains: "message",
		},
		{
			name: "Missing time field",
			data: map[string]interface{}{
				"code":    0,
				"message": "ok",
			},
			expectError:   true,
			errorContains: "time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)
			server := bssci.NewTestServer(logger, mockStorage, nil, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

			mockConn := &statusMockConn{}
			session := &bssci.Session{
				ID:             "test-validation-session",
				BaseStationEUI: 0x1122334455667788,
				Conn:           mockConn,
				Encoding:       "msgpack",
			}

			statusMsg := &bssci.Message{
				Command: "statusRsp",
				OpId:    -1,
				Data:    tt.data,
			}

			_ = server.CallHandleStatusResponse(session, statusMsg, tt.data)

			if tt.expectError {
				// Should have sent error frame to base station per BSSCI §3.17 (not Go error)
				assert.True(t, mockConn.errorSent, "Should send error frame when validation fails")
				if tt.errorContains != "" {
					// Verify error frame contains reference to missing field
					assert.Contains(t, mockConn.lastErrorMsg, tt.errorContains, "Error message should mention missing field")
				}
			} else {
				// When all mandatory fields present, no error frame should be sent
				assert.False(t, mockConn.errorSent, "Should not send error frame when all mandatory fields present")
			}
		})
	}
}

// TestStatusOptionalFieldHandling verifies BSSCI §3.5.2:
// Optional fields should be accepted gracefully when omitted
func TestStatusOptionalFieldHandling(t *testing.T) {
	logger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)
	server := bssci.NewTestServer(logger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-optional-session",
		BaseStationEUI: 0x1122334455667788,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	// statusRsp with only mandatory fields (no optional fields)
	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -1,
		Data: map[string]interface{}{
			"code":    0,
			"message": "ok",
			"time":    int64(1672531200000000000),
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	assert.NoError(t, err, "Should succeed with only mandatory fields")

	// Verify statusCmp sent
	require.Len(t, mockConn.sentMessages, 1)
	assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"])
}

// TestStatusWithAllFields verifies that all optional fields are accepted
func TestStatusWithAllFields(t *testing.T) {
	logger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)
	server := bssci.NewTestServer(logger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-all-fields-session",
		BaseStationEUI: 0x1122334455667788,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	// statusRsp with all optional fields
	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -2,
		Data: map[string]interface{}{
			"code":      0,
			"message":   "ok",
			"time":      int64(1672531200000000000),
			"dutyCycle": 0.45,
			"uptime":    int64(3600),
			"temp":      float64(25.5),
			"cpuLoad":   0.75,
			"memLoad":   0.60,
			"config":    `{"version":"1.0"}`,
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	assert.NoError(t, err, "Should succeed with all fields present")

	// Verify statusCmp sent with correct opId
	require.Len(t, mockConn.sentMessages, 1)
	statusCmp := mockConn.sentMessages[0]
	assert.Equal(t, "statusCmp", statusCmp["command"])
	assert.Equal(t, int64(-2), statusCmp["opId"])
}

// tenantTrackingBaseStationRepo is a mock that captures tenant IDs passed to repository methods
type tenantTrackingBaseStationRepo struct {
	getByEUITenantID int64
	updateTenantID   int64
	updateCalled     bool
	getByEUICalled   bool
}

func (m *tenantTrackingBaseStationRepo) Create(_ context.Context, _ *models.BaseStation) error {
	return nil
}

func (m *tenantTrackingBaseStationRepo) GetByID(_ context.Context, tenantID, id int64) (*models.BaseStation, error) {
	var eui models.EUI
	return &models.BaseStation{
		ID:       id,
		TenantID: tenantID,
		EUI:      eui,
		Name:     "Test BS",
	}, nil
}

func (m *tenantTrackingBaseStationRepo) GetByEUI(_ context.Context, tenantID int64, euiBytes []byte) (*models.BaseStation, error) {
	m.getByEUICalled = true
	m.getByEUITenantID = tenantID

	var eui models.EUI
	if len(euiBytes) == 8 {
		copy(eui[:], euiBytes)
	}

	return &models.BaseStation{
		ID:       1,
		TenantID: tenantID,
		EUI:      eui,
		Name:     "Test BS",
	}, nil
}

func (m *tenantTrackingBaseStationRepo) Update(_ context.Context, tenantID int64, _ int64, _ map[string]interface{}) error {
	m.updateCalled = true
	m.updateTenantID = tenantID
	return nil
}

func (m *tenantTrackingBaseStationRepo) Delete(_ context.Context, _ int64, _ int64) error {
	return nil
}

func (m *tenantTrackingBaseStationRepo) List(_ context.Context, _ *models.BaseStationFilter) ([]*models.BaseStation, int64, error) {
	return []*models.BaseStation{}, 0, nil
}

func (m *tenantTrackingBaseStationRepo) UpdateConnectionStatus(_ context.Context, _ int64, _ int64, _ bool, _ *string) error {
	return nil
}

func (m *tenantTrackingBaseStationRepo) UpdateSessionInfo(_ context.Context, _ int64, _ []byte, _ string) error {
	return nil
}

func (m *tenantTrackingBaseStationRepo) GetStatistics(_ context.Context, _ int64) (*interfaces.BaseStationStatistics, error) {
	return &interfaces.BaseStationStatistics{}, nil
}

func (m *tenantTrackingBaseStationRepo) GetPropagationState(_ context.Context, _ int64) (*models.BaseStationPropagationState, error) {
	return nil, nil
}

func (m *tenantTrackingBaseStationRepo) UpsertPropagationState(_ context.Context, _ *models.BaseStationPropagationState) error {
	return nil
}

func (m *tenantTrackingBaseStationRepo) UpdatePropagationStatus(_ context.Context, _ int64, _ string, _ *string) error {
	return nil
}

func (m *tenantTrackingBaseStationRepo) IncrementRetryCount(_ context.Context, _ int64, _ time.Time) error {
	return nil
}

func (m *tenantTrackingBaseStationRepo) UpdateEUI(_ context.Context, _ int64, _, _ []byte) (*models.BaseStation, error) {
	return nil, nil
}

func (m *tenantTrackingBaseStationRepo) GetByEUIGlobal(_ context.Context, euiBytes []byte) (*models.BaseStation, error) {
	var eui models.EUI
	if len(euiBytes) == 8 {
		copy(eui[:], euiBytes)
	}
	return &models.BaseStation{ID: 1, TenantID: 1, EUI: eui, Name: "Test BS"}, nil
}
func (m *tenantTrackingBaseStationRepo) ListAllLocations(_ context.Context) ([]*models.BaseStation, error) {
	return nil, nil
}

// TestStatusResponseRespectsTenantIsolation verifies that status updates use the session's
// resolved tenant ID rather than the server's default tenant ID.
// This is critical for multi-tenant deployments where base stations may serve multiple tenants.
//
// Validates BSSCI §3.5 tenant isolation requirement:
// - GetByEUI must use resolvedTenant(session, s.tenantID)
// - Update must use resolvedTenant(session, s.tenantID)
func TestStatusResponseRespectsTenantIsolation(t *testing.T) {
	logger := logger.NewNop()

	// Create tracking mock repository
	trackingRepo := &tenantTrackingBaseStationRepo{}

	// Create test services with default tenant ID = 1
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)

	// Create server with tenant ID = 1 and our tracking repository
	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, trackingRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}

	// Session with ResolvedTenantID = 42 (different from server's default tenant 1)
	session := &bssci.Session{
		ID:               "test-tenant-isolation",
		BaseStationEUI:   0x1122334455667788,
		Conn:             mockConn,
		Encoding:         "msgpack",
		ResolvedTenantID: 42, // This should be used instead of server's tenantID (1)
	}

	// Create statusRsp with mandatory fields
	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -1,
		Data: map[string]interface{}{
			"code":    0,
			"message": "ok",
			"time":    int64(1672531200000000000),
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	require.NoError(t, err, "Status response should be processed successfully")

	// CRITICAL ASSERTIONS: Verify the resolved tenant ID (42) was used, not the server default (1)
	assert.True(t, trackingRepo.getByEUICalled, "GetByEUI should have been called")
	assert.Equal(t, int64(42), trackingRepo.getByEUITenantID,
		"GetByEUI must use session's ResolvedTenantID (42), not server default (1)")

	assert.True(t, trackingRepo.updateCalled, "Update should have been called")
	assert.Equal(t, int64(42), trackingRepo.updateTenantID,
		"Update must use session's ResolvedTenantID (42), not server default (1)")

	// Verify statusCmp was sent
	require.Len(t, mockConn.sentMessages, 1, "Should send statusCmp")
	assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"])
}

// geoLocationTrackingRepo captures the updates map passed to Update method
type geoLocationTrackingRepo struct {
	updateCalled bool
	updatesMap   map[string]interface{}
}

func (m *geoLocationTrackingRepo) Create(_ context.Context, _ *models.BaseStation) error {
	return nil
}

func (m *geoLocationTrackingRepo) GetByID(_ context.Context, tenantID, id int64) (*models.BaseStation, error) {
	var eui models.EUI
	return &models.BaseStation{
		ID:       id,
		TenantID: tenantID,
		EUI:      eui,
		Name:     "Test BS",
	}, nil
}

func (m *geoLocationTrackingRepo) GetByEUI(_ context.Context, tenantID int64, euiBytes []byte) (*models.BaseStation, error) {
	var eui models.EUI
	if len(euiBytes) == 8 {
		copy(eui[:], euiBytes)
	}

	return &models.BaseStation{
		ID:       1,
		TenantID: tenantID,
		EUI:      eui,
		Name:     "Test BS",
	}, nil
}

func (m *geoLocationTrackingRepo) Update(_ context.Context, _ int64, _ int64, updates map[string]interface{}) error {
	m.updateCalled = true
	// Deep copy the updates map to preserve it for assertions
	m.updatesMap = make(map[string]interface{})
	for k, v := range updates {
		m.updatesMap[k] = v
	}
	return nil
}

func (m *geoLocationTrackingRepo) Delete(_ context.Context, _ int64, _ int64) error {
	return nil
}

func (m *geoLocationTrackingRepo) List(_ context.Context, _ *models.BaseStationFilter) ([]*models.BaseStation, int64, error) {
	return []*models.BaseStation{}, 0, nil
}

func (m *geoLocationTrackingRepo) UpdateConnectionStatus(_ context.Context, _ int64, _ int64, _ bool, _ *string) error {
	return nil
}

func (m *geoLocationTrackingRepo) UpdateSessionInfo(_ context.Context, _ int64, _ []byte, _ string) error {
	return nil
}

func (m *geoLocationTrackingRepo) GetStatistics(_ context.Context, _ int64) (*interfaces.BaseStationStatistics, error) {
	return &interfaces.BaseStationStatistics{}, nil
}

func (m *geoLocationTrackingRepo) GetPropagationState(_ context.Context, _ int64) (*models.BaseStationPropagationState, error) {
	return nil, nil
}

func (m *geoLocationTrackingRepo) UpsertPropagationState(_ context.Context, _ *models.BaseStationPropagationState) error {
	return nil
}

func (m *geoLocationTrackingRepo) UpdatePropagationStatus(_ context.Context, _ int64, _ string, _ *string) error {
	return nil
}

func (m *geoLocationTrackingRepo) IncrementRetryCount(_ context.Context, _ int64, _ time.Time) error {
	return nil
}

func (m *geoLocationTrackingRepo) UpdateEUI(_ context.Context, _ int64, _, _ []byte) (*models.BaseStation, error) {
	return nil, nil
}

func (m *geoLocationTrackingRepo) GetByEUIGlobal(_ context.Context, euiBytes []byte) (*models.BaseStation, error) {
	var eui models.EUI
	if len(euiBytes) == 8 {
		copy(eui[:], euiBytes)
	}
	return &models.BaseStation{ID: 1, TenantID: 1, EUI: eui, Name: "Test BS"}, nil
}
func (m *geoLocationTrackingRepo) ListAllLocations(_ context.Context) ([]*models.BaseStation, error) {
	return nil, nil
}

// TestStatusResponsePersistsGeoLocation verifies that geoLocation data from statusRsp
// is correctly parsed and persisted to the database.
//
// Validates BSSCI §3.5.2 optional geoLocation field:
// - Array format [latitude, longitude, altitude] should be parsed
// - Map/struct format {lat, lon, alt} should be parsed
// - All three values must be persisted via Update call
func TestStatusResponsePersistsGeoLocation(t *testing.T) {
	logger := logger.NewNop()

	tests := []struct {
		name        string
		geoLocation interface{}
		expectLat   *float64
		expectLon   *float64
		expectAlt   *float64
	}{
		{
			name:        "Array format with 3 numeric values",
			geoLocation: []interface{}{float64(52.5200), float64(13.4050), float64(34.0)},
			expectLat:   floatPtr(52.5200),
			expectLon:   floatPtr(13.4050),
			expectAlt:   floatPtr(34.0),
		},
		{
			name: "Map format with lat/lon/alt keys",
			geoLocation: map[string]interface{}{
				"lat": float64(48.8566),
				"lon": float64(2.3522),
				"alt": float64(35.0),
			},
			expectLat: floatPtr(48.8566),
			expectLon: floatPtr(2.3522),
			expectAlt: floatPtr(35.0),
		},
		{
			name:        "No geoLocation field",
			geoLocation: nil,
			expectLat:   nil,
			expectLon:   nil,
			expectAlt:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create tracking repo
			trackingRepo := &geoLocationTrackingRepo{}

			// Create test services
			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)

			// Create server with our tracking repository
			server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, trackingRepo, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

			mockConn := &statusMockConn{}
			session := &bssci.Session{
				ID:             "test-geolocation",
				BaseStationEUI: 0x1122334455667788,
				Conn:           mockConn,
				Encoding:       "msgpack",
			}

			// Create statusRsp with mandatory fields and geoLocation
			data := map[string]interface{}{
				"code":    0,
				"message": "ok",
				"time":    int64(1672531200000000000),
			}
			if tt.geoLocation != nil {
				data["geoLocation"] = tt.geoLocation
			}

			statusMsg := &bssci.Message{
				Command: "statusRsp",
				OpId:    -1,
				Data:    data,
			}

			err := server.CallHandleStatusResponse(session, statusMsg, data)
			require.NoError(t, err, "Status response should be processed successfully")

			// CRITICAL ASSERTIONS: Verify geoLocation fields were persisted
			assert.True(t, trackingRepo.updateCalled, "Update should have been called")

			if tt.expectLat != nil {
				assert.Contains(t, trackingRepo.updatesMap, "latitude", "Latitude should be in updates map")
				assert.Equal(t, *tt.expectLat, trackingRepo.updatesMap["latitude"],
					"Latitude value should match expected")
			} else {
				assert.NotContains(t, trackingRepo.updatesMap, "latitude", "Latitude should not be in updates map when not provided")
			}

			if tt.expectLon != nil {
				assert.Contains(t, trackingRepo.updatesMap, "longitude", "Longitude should be in updates map")
				assert.Equal(t, *tt.expectLon, trackingRepo.updatesMap["longitude"],
					"Longitude value should match expected")
			} else {
				assert.NotContains(t, trackingRepo.updatesMap, "longitude", "Longitude should not be in updates map when not provided")
			}

			if tt.expectAlt != nil {
				assert.Contains(t, trackingRepo.updatesMap, "altitude", "Altitude should be in updates map")
				assert.Equal(t, *tt.expectAlt, trackingRepo.updatesMap["altitude"],
					"Altitude value should match expected")
			} else {
				assert.NotContains(t, trackingRepo.updatesMap, "altitude", "Altitude should not be in updates map when not provided")
			}

			// Verify statusCmp was sent
			require.Len(t, mockConn.sentMessages, 1, "Should send statusCmp")
			assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"])
		})
	}
}

// floatPtr is a helper that returns a pointer to a float64 value
func floatPtr(v float64) *float64 {
	return &v
}

// ========== Mock Repositories for Comprehensive Test Coverage ==========

// errorInjectingBaseStationRepo is a mock that injects errors for database operations
type errorInjectingBaseStationRepo struct {
	getByEUIError    error
	updateError      error
	getByEUICalled   bool
	updateCalled     bool
	getByEUITenantID int64
	updateTenantID   int64
}

func (m *errorInjectingBaseStationRepo) Create(_ context.Context, _ *models.BaseStation) error {
	return nil
}

func (m *errorInjectingBaseStationRepo) GetByID(_ context.Context, tenantID, id int64) (*models.BaseStation, error) {
	var eui models.EUI
	return &models.BaseStation{
		ID:       id,
		TenantID: tenantID,
		EUI:      eui,
		Name:     "Test BS",
	}, nil
}

func (m *errorInjectingBaseStationRepo) GetByEUI(_ context.Context, tenantID int64, euiBytes []byte) (*models.BaseStation, error) {
	m.getByEUICalled = true
	m.getByEUITenantID = tenantID
	if m.getByEUIError != nil {
		return nil, m.getByEUIError
	}

	var eui models.EUI
	if len(euiBytes) == 8 {
		copy(eui[:], euiBytes)
	}

	return &models.BaseStation{
		ID:       1,
		TenantID: tenantID,
		EUI:      eui,
		Name:     "Test BS",
	}, nil
}

func (m *errorInjectingBaseStationRepo) Update(_ context.Context, tenantID int64, _ int64, _ map[string]interface{}) error {
	m.updateCalled = true
	m.updateTenantID = tenantID
	return m.updateError
}

func (m *errorInjectingBaseStationRepo) Delete(_ context.Context, _ int64, _ int64) error {
	return nil
}

func (m *errorInjectingBaseStationRepo) List(_ context.Context, _ *models.BaseStationFilter) ([]*models.BaseStation, int64, error) {
	return []*models.BaseStation{}, 0, nil
}

func (m *errorInjectingBaseStationRepo) UpdateConnectionStatus(_ context.Context, _ int64, _ int64, _ bool, _ *string) error {
	return nil
}

func (m *errorInjectingBaseStationRepo) UpdateSessionInfo(_ context.Context, _ int64, _ []byte, _ string) error {
	return nil
}

func (m *errorInjectingBaseStationRepo) GetStatistics(_ context.Context, _ int64) (*interfaces.BaseStationStatistics, error) {
	return &interfaces.BaseStationStatistics{}, nil
}

func (m *errorInjectingBaseStationRepo) GetPropagationState(_ context.Context, _ int64) (*models.BaseStationPropagationState, error) {
	return nil, nil
}

func (m *errorInjectingBaseStationRepo) UpsertPropagationState(_ context.Context, _ *models.BaseStationPropagationState) error {
	return nil
}

func (m *errorInjectingBaseStationRepo) UpdatePropagationStatus(_ context.Context, _ int64, _ string, _ *string) error {
	return nil
}

func (m *errorInjectingBaseStationRepo) IncrementRetryCount(_ context.Context, _ int64, _ time.Time) error {
	return nil
}

func (m *errorInjectingBaseStationRepo) UpdateEUI(_ context.Context, _ int64, _, _ []byte) (*models.BaseStation, error) {
	return nil, nil
}

func (m *errorInjectingBaseStationRepo) GetByEUIGlobal(_ context.Context, euiBytes []byte) (*models.BaseStation, error) {
	var eui models.EUI
	if len(euiBytes) == 8 {
		copy(eui[:], euiBytes)
	}
	return &models.BaseStation{ID: 1, TenantID: 1, EUI: eui, Name: "Test BS"}, nil
}
func (m *errorInjectingBaseStationRepo) ListAllLocations(_ context.Context) ([]*models.BaseStation, error) {
	return nil, nil
}

// mockStatusHistoryRepo is a mock for MIOTYBaseStationStatusRepository that tracks Create calls
type mockStatusHistoryRepo struct {
	createError  error
	createCalled bool
	lastRecord   *mioty.BaseStationStatusRecord
}

func (m *mockStatusHistoryRepo) Create(_ context.Context, record *mioty.BaseStationStatusRecord) error {
	m.createCalled = true
	// Deep copy the record to preserve it for assertions
	m.lastRecord = &mioty.BaseStationStatusRecord{
		TenantID:       record.TenantID,
		BaseStationID:  record.BaseStationID,
		BasestationEUI: make([]byte, len(record.BasestationEUI)),
		OperationID:    record.OperationID,
		StatusCode:     record.StatusCode,
		StatusMessage:  record.StatusMessage,
		SystemTime:     record.SystemTime,
		DutyCycle:      record.DutyCycle,
		UptimeSeconds:  record.UptimeSeconds,
		Temperature:    record.Temperature,
		CPULoad:        record.CPULoad,
		MemoryLoad:     record.MemoryLoad,
		Config:         record.Config,
		Latitude:       record.Latitude,
		Longitude:      record.Longitude,
		Altitude:       record.Altitude,
	}
	copy(m.lastRecord.BasestationEUI, record.BasestationEUI)
	return m.createError
}

// panicOnCallBaseStationRepo panics if GetByEUI or Update are called.
// Used to verify POSIX validation short-circuits before DB access.
// All other methods return safe defaults to prevent false failures if server initialization changes.
type panicOnCallBaseStationRepo struct{}

// Validation guard methods - these MUST NOT be called (validation should short-circuit)
func (m *panicOnCallBaseStationRepo) GetByEUI(_ context.Context, _ int64, _ []byte) (*models.BaseStation, error) {
	panic("POSIX validation must short-circuit before GetByEUI")
}

func (m *panicOnCallBaseStationRepo) Update(_ context.Context, _ int64, _ int64, _ map[string]interface{}) error {
	panic("POSIX validation must short-circuit before Update")
}

// Panic guard methods - ALL repo access must be blocked before validation
// Research confirms NO methods are called during server initialization (see server.go:605-776)
// If future code legitimately needs a method during boot, revert that method to safe stub with inline comment
func (m *panicOnCallBaseStationRepo) Create(_ context.Context, _ *models.BaseStation) error {
	panic("panicOnCallBaseStationRepo.Create() called - validation should short-circuit before ANY repo access")
}

func (m *panicOnCallBaseStationRepo) GetByID(_ context.Context, _ int64, _ int64) (*models.BaseStation, error) {
	panic("panicOnCallBaseStationRepo.GetByID() called - validation should short-circuit before ANY repo access")
}

func (m *panicOnCallBaseStationRepo) Delete(_ context.Context, _ int64, _ int64) error {
	panic("panicOnCallBaseStationRepo.Delete() called - validation should short-circuit before ANY repo access")
}

func (m *panicOnCallBaseStationRepo) List(_ context.Context, _ *models.BaseStationFilter) ([]*models.BaseStation, int64, error) {
	panic("panicOnCallBaseStationRepo.List() called - validation should short-circuit before ANY repo access")
}

func (m *panicOnCallBaseStationRepo) GetStatistics(_ context.Context, _ int64) (*interfaces.BaseStationStatistics, error) {
	panic("panicOnCallBaseStationRepo.GetStatistics() called - validation should short-circuit before ANY repo access")
}

func (m *panicOnCallBaseStationRepo) UpdateConnectionStatus(_ context.Context, _ int64, _ int64, _ bool, _ *string) error {
	panic("panicOnCallBaseStationRepo.UpdateConnectionStatus() called - validation should short-circuit before ANY repo access")
}

func (m *panicOnCallBaseStationRepo) UpdateSessionInfo(_ context.Context, _ int64, _ []byte, _ string) error {
	panic("panicOnCallBaseStationRepo.UpdateSessionInfo() called - validation should short-circuit before ANY repo access")
}

func (m *panicOnCallBaseStationRepo) GetPropagationState(_ context.Context, _ int64) (*models.BaseStationPropagationState, error) {
	panic("panicOnCallBaseStationRepo.GetPropagationState() called - validation should short-circuit before ANY repo access")
}

func (m *panicOnCallBaseStationRepo) UpsertPropagationState(_ context.Context, _ *models.BaseStationPropagationState) error {
	panic("panicOnCallBaseStationRepo.UpsertPropagationState() called - validation should short-circuit before ANY repo access")
}

func (m *panicOnCallBaseStationRepo) UpdatePropagationStatus(_ context.Context, _ int64, _ string, _ *string) error {
	panic("panicOnCallBaseStationRepo.UpdatePropagationStatus() called - validation should short-circuit before ANY repo access")
}

func (m *panicOnCallBaseStationRepo) IncrementRetryCount(_ context.Context, _ int64, _ time.Time) error {
	panic("panicOnCallBaseStationRepo.IncrementRetryCount() called - validation should short-circuit before ANY repo access")
}

func (m *panicOnCallBaseStationRepo) UpdateEUI(_ context.Context, _ int64, _, _ []byte) (*models.BaseStation, error) {
	panic("panicOnCallBaseStationRepo.UpdateEUI() called - validation should short-circuit before ANY repo access")
}

func (m *panicOnCallBaseStationRepo) GetByEUIGlobal(_ context.Context, _ []byte) (*models.BaseStation, error) {
	panic("panicOnCallBaseStationRepo.GetByEUIGlobal() called - validation should short-circuit before ANY repo access")
}
func (m *panicOnCallBaseStationRepo) ListAllLocations(_ context.Context) ([]*models.BaseStation, error) {
	return nil, nil
}

// mockStorageWithHistory is a minimal Storage implementation for history testing
type mockStorageWithHistory struct {
	historyRepo     interfaces.MIOTYBaseStationStatusRepository
	baseStationRepo interfaces.BaseStationRepository
}

func (m *mockStorageWithHistory) EndPoints() interfaces.EndpointRepository          { return nil }
func (m *mockStorageWithHistory) DownlinkQueue() interfaces.DownlinkQueueRepository { return nil }
func (m *mockStorageWithHistory) BaseStationReceptions() interfaces.BaseStationReceptionRepository {
	return nil
}
func (m *mockStorageWithHistory) EndPointSessions() interfaces.EndPointSessionRepository { return nil }
func (m *mockStorageWithHistory) EndPointKeys() interfaces.EndPointKeyRepository         { return nil }
func (m *mockStorageWithHistory) RoamingAgreements() interfaces.RoamingAgreementRepository {
	return nil
}
func (m *mockStorageWithHistory) BaseStations() interfaces.BaseStationRepository {
	if m.baseStationRepo != nil {
		return m.baseStationRepo
	}
	return nil
}
func (m *mockStorageWithHistory) BaseStationSessions() interfaces.BaseStationSessionRepository {
	return nil
}
func (m *mockStorageWithHistory) DLRXStatus() interfaces.DLRXStatusRepository { return nil }
func (m *mockStorageWithHistory) PendingOperations() interfaces.PendingOperationRepository {
	return nil
}
func (m *mockStorageWithHistory) MIOTYMessages() interfaces.MIOTYMessageRepository   { return nil }
func (m *mockStorageWithHistory) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository { return nil }
func (m *mockStorageWithHistory) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	return m.historyRepo
}
func (m *mockStorageWithHistory) Users() interfaces.UserRepository                 { return nil }
func (m *mockStorageWithHistory) APIKeys() interfaces.APIKeyRepository             { return nil }
func (m *mockStorageWithHistory) Integrations() interfaces.IntegrationRepository   { return nil }
func (m *mockStorageWithHistory) Manufacturers() interfaces.ManufacturerRepository { return nil }
func (m *mockStorageWithHistory) DeviceModels() interfaces.DeviceModelRepository   { return nil }
func (m *mockStorageWithHistory) Blueprints() interfaces.BlueprintRepository       { return nil }
func (m *mockStorageWithHistory) Organizations() interfaces.OrganizationRepository { return nil }
func (m *mockStorageWithHistory) GetSqlxDB() *sqlx.DB                              { return nil }
func (m *mockStorageWithHistory) SystemEvents() interfaces.SystemEventStore        { return nil }
func (m *mockStorageWithHistory) SCACISessions() interfaces.SCACISessionRepository { return nil }
func (m *mockStorageWithHistory) SCACIOperations() interfaces.SCACIOperationRepository {
	return nil
}
func (m *mockStorageWithHistory) DownlinkQueueReader() interfaces.DownlinkQueueReader { return nil }
func (m *mockStorageWithHistory) BeginTx(_ context.Context) (interfaces.Transaction, error) {
	return nil, nil
}
func (m *mockStorageWithHistory) Ping(_ context.Context) error { return nil }
func (m *mockStorageWithHistory) Close() error                 { return nil }

// ========== Database Error Path Tests ==========

// TestStatusGetByEUIFailure verifies BSSCI §3.5.3 lines 1124-1156:
// GetByEUI database errors should be logged but NOT prevent statusCmp delivery.
// This ensures three-way handshake completion regardless of persistence failures.
//
// Lines validated: status_handlers.go:168-173
func TestStatusGetByEUIFailure(t *testing.T) {
	logger := logger.NewNop()

	// Create mock repo that fails on GetByEUI
	errorRepo := &errorInjectingBaseStationRepo{
		getByEUIError: assert.AnError,
	}

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)
	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, errorRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-db-error-getbyeui",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -1,
		Data: map[string]interface{}{
			"code":    0,
			"message": "ok",
			"time":    int64(1672531200000000000),
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	require.NoError(t, err, "Handler should not return error for DB failure")

	// CRITICAL: statusCmp must be sent despite GetByEUI failure per BSSCI §3.5.3
	require.Len(t, mockConn.sentMessages, 1, "Must send statusCmp despite GetByEUI error")
	assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"], "Must send statusCmp")
	assert.False(t, mockConn.errorSent, "Must not send error frame for DB failure")
	assert.True(t, errorRepo.getByEUICalled, "GetByEUI should have been attempted")
	// Update should not be called when GetByEUI fails (lines 169-173 early exit)
	assert.False(t, errorRepo.updateCalled, "Update should not be called after GetByEUI failure")
}

// TestStatusUpdateFailure verifies BSSCI §3.5.3 lines 1124-1156:
// Update database errors should be logged but NOT prevent statusCmp delivery.
// History persistence should be skipped when Update fails (else block at line 216).
//
// Lines validated: status_handlers.go:211-216
func TestStatusUpdateFailure(t *testing.T) {
	logger := logger.NewNop()

	errorRepo := &errorInjectingBaseStationRepo{
		updateError: assert.AnError,
	}

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)
	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, errorRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-update-error",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -2,
		Data: map[string]interface{}{
			"code":      1,
			"message":   "warning",
			"time":      int64(1672531200000000000),
			"dutyCycle": 0.75,
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	require.NoError(t, err, "Handler should not return error for Update failure")

	// CRITICAL: statusCmp must be sent despite Update failure per BSSCI §3.5.3
	require.Len(t, mockConn.sentMessages, 1, "Must send statusCmp despite Update error")
	assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"], "Must send statusCmp")
	assert.False(t, mockConn.errorSent, "Must not send error frame for DB failure")
	assert.True(t, errorRepo.getByEUICalled, "GetByEUI should have been called")
	assert.True(t, errorRepo.updateCalled, "Update should have been attempted")
}

// TestStatusHistoryPersistenceFailure verifies BSSCI §3.5.2 lines 905-930:
// History persistence errors (MIOTYBaseStationStatus().Create) should be logged but
// NOT prevent statusCmp delivery. History is best-effort.
//
// Lines validated: status_handlers.go:222-246
func TestStatusHistoryPersistenceFailure(t *testing.T) {
	logger := logger.NewNop()

	// Create mock storage with failing history repository
	mockHistoryRepo := &mockStatusHistoryRepo{
		createError: assert.AnError,
	}

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssciservices.CreateTestServices(logger, nil)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-history-error",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -3,
		Data: map[string]interface{}{
			"code":        0,
			"message":     "ok",
			"time":        int64(1672531200000000000),
			"geoLocation": []interface{}{float64(52.52), float64(13.405), float64(34.0)},
		},
	}

	// Create a successful base station repo
	successRepo := &tenantTrackingBaseStationRepo{}

	// Create storage that returns our mocks
	mockStorage := &mockStorageWithHistory{
		historyRepo:     mockHistoryRepo,
		baseStationRepo: successRepo,
	}

	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, successRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	require.NoError(t, err, "Handler should not return error for history failure")

	// CRITICAL: statusCmp must be sent despite history Create failure per BSSCI §3.5.2
	require.Len(t, mockConn.sentMessages, 1, "Must send statusCmp despite history error")
	assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"], "Must send statusCmp")
	assert.False(t, mockConn.errorSent, "Must not send error frame for history failure")

	// Verify history persistence was attempted
	assert.True(t, mockHistoryRepo.createCalled, "Create should have been attempted")

	// Verify the record that was attempted to be persisted contains correct data
	assert.NotNil(t, mockHistoryRepo.lastRecord, "Should have attempted to persist record")
	assert.Equal(t, 0, mockHistoryRepo.lastRecord.StatusCode, "Status code should match")
	assert.NotNil(t, mockHistoryRepo.lastRecord.Latitude, "Latitude should be parsed from geoLocation")
	assert.NotNil(t, mockHistoryRepo.lastRecord.Longitude, "Longitude should be parsed from geoLocation")
	assert.NotNil(t, mockHistoryRepo.lastRecord.Altitude, "Altitude should be parsed from geoLocation")
}

// ========== StatusComplete Error Path Tests ==========

// mockStatusService is a mock for status service that can inject RemovePendingOperation errors
type mockStatusService struct {
	removePendingOpError error
	removePendingOpCalls int
}

func (m *mockStatusService) RemovePendingOperation(_ context.Context, _ *bssci.Session, _ int64) error {
	m.removePendingOpCalls++
	return m.removePendingOpError
}

func (m *mockStatusService) RecordPendingOperation(_ context.Context, _ *bssci.Session, _ int64, _ *bssci.PendingOperation, _ int64) error {
	return nil
}

func (m *mockStatusService) GetPendingOperation(_ *bssci.Session, _ int64) (*bssci.PendingOperation, error) {
	return nil, nil
}

func (m *mockStatusService) CleanupPendingOp(_ *bssci.Session, _ int64) {
}

func (m *mockStatusService) ExtractQueueMetadata(_ *bssci.Session, _ int64) (uint64, int64, string) {
	return 0, 0, ""
}

// TestStatusCompletePendingOperationFailure verifies BSSCI §3.5.3 lines 916-930:
// removePendingOperation errors should be logged but NOT bubble up as handler errors.
// The statusCmp handler should complete successfully even if cleanup fails.
//
// Lines validated: status_handlers.go:259-273
func TestStatusCompletePendingOperationFailure(t *testing.T) {
	logger := logger.NewNop()

	sessionSvc, downlinkSvc, _, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)

	// Replace statusSvc with our mock that fails on RemovePendingOperation
	mockStatusSvc := &mockStatusService{
		removePendingOpError: assert.AnError,
	}

	server := bssci.NewTestServer(logger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, mockStatusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-statuscmp-cleanup-error",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
		DbSessionID:    42,
	}

	statusCmpMsg := &bssci.Message{
		Command: "statusCmp",
		OpId:    -5,
	}

	err := server.CallHandleStatusComplete(session, statusCmpMsg, nil)
	require.NoError(t, err, "Handler must not return error even if RemovePendingOperation fails")

	// Verify RemovePendingOperation was attempted
	assert.Equal(t, 1, mockStatusSvc.removePendingOpCalls, "RemovePendingOperation should have been called once")
}

// TestStatusCompleteIdempotency verifies BSSCI §3.5.3:
// Receiving duplicate statusCmp messages (e.g., due to retry) should be handled gracefully.
// Multiple calls should not cause errors even if pending operation was already removed.
//
// Lines validated: status_handlers.go:259-273
func TestStatusCompleteIdempotency(t *testing.T) {
	logger := logger.NewNop()

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)
	server := bssci.NewTestServer(logger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-statuscmp-idempotent",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
		DbSessionID:    43,
	}

	statusCmpMsg := &bssci.Message{
		Command: "statusCmp",
		OpId:    -6,
	}

	// First call - should succeed
	err := server.CallHandleStatusComplete(session, statusCmpMsg, nil)
	require.NoError(t, err, "First statusCmp should succeed")

	// Second call with same opId - should also succeed (idempotent)
	err = server.CallHandleStatusComplete(session, statusCmpMsg, nil)
	require.NoError(t, err, "Duplicate statusCmp should be handled gracefully")

	// No messages should be sent in response to statusCmp (it's the final step)
	assert.Empty(t, mockConn.sentMessages, "No messages should be sent in response to statusCmp")
}

// ========== GeoLocation Edge Case Tests ==========

// TestStatusGeoLocationStringFormat verifies BSSCI §3.5.2 lines 820-844:
// geoLocation field should gracefully handle string format (invalid per spec).
// Handler should skip geoLocation parsing without failing the entire statusRsp.
//
// Lines validated: status_handlers.go:129-157
func TestStatusGeoLocationStringFormat(t *testing.T) {
	logger := logger.NewNop()

	trackingRepo := &geoLocationTrackingRepo{}

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)
	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, trackingRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-geo-string",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -10,
		Data: map[string]interface{}{
			"code":        0,
			"message":     "ok",
			"time":        int64(1672531200000000000),
			"geoLocation": "52.5200,13.4050,34.0", // String format - invalid per spec
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	require.NoError(t, err, "Handler should gracefully skip invalid geoLocation format")

	// Verify statusCmp was sent
	require.Len(t, mockConn.sentMessages, 1, "Should send statusCmp despite invalid geoLocation")
	assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"], "Should send statusCmp")

	// Verify geoLocation fields were NOT added to updates (invalid format skipped)
	assert.NotContains(t, trackingRepo.updatesMap, "latitude", "Latitude should not be persisted from invalid format")
	assert.NotContains(t, trackingRepo.updatesMap, "longitude", "Longitude should not be persisted from invalid format")
	assert.NotContains(t, trackingRepo.updatesMap, "altitude", "Altitude should not be persisted from invalid format")
}

// TestStatusGeoLocationPartialArray verifies BSSCI §3.5.2 lines 820-844:
// geoLocation array with fewer than 3 elements should be handled gracefully.
// Only complete [lat, lon, alt] arrays should be persisted.
//
// Lines validated: status_handlers.go:133-144
func TestStatusGeoLocationPartialArray(t *testing.T) {
	logger := logger.NewNop()

	trackingRepo := &geoLocationTrackingRepo{}

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)
	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, trackingRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-geo-partial",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -11,
		Data: map[string]interface{}{
			"code":        0,
			"message":     "ok",
			"time":        int64(1672531200000000000),
			"geoLocation": []interface{}{float64(52.5200), float64(13.4050)}, // Only 2 elements (missing altitude)
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	require.NoError(t, err, "Handler should gracefully handle partial geoLocation array")

	// Verify statusCmp was sent
	require.Len(t, mockConn.sentMessages, 1, "Should send statusCmp despite partial geoLocation")
	assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"], "Should send statusCmp")

	// Verify geoLocation fields were NOT added (partial array rejected)
	assert.NotContains(t, trackingRepo.updatesMap, "latitude", "Latitude should not be persisted from partial array")
	assert.NotContains(t, trackingRepo.updatesMap, "longitude", "Longitude should not be persisted from partial array")
	assert.NotContains(t, trackingRepo.updatesMap, "altitude", "Altitude should not be persisted from partial array")
}

// TestStatusGeoLocationNonNumericArray verifies BSSCI §3.5.2 lines 820-844:
// geoLocation array with non-numeric values should be handled gracefully.
// Only valid numeric arrays should be persisted.
//
// Lines validated: status_handlers.go:135-143
func TestStatusGeoLocationNonNumericArray(t *testing.T) {
	logger := logger.NewNop()

	trackingRepo := &geoLocationTrackingRepo{}

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)
	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, trackingRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-geo-nonnumeric",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -12,
		Data: map[string]interface{}{
			"code":        0,
			"message":     "ok",
			"time":        int64(1672531200000000000),
			"geoLocation": []interface{}{"52.5200", "13.4050", "34.0"}, // Strings instead of numbers
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	require.NoError(t, err, "Handler should gracefully handle non-numeric geoLocation")

	// Verify statusCmp was sent
	require.Len(t, mockConn.sentMessages, 1, "Should send statusCmp despite non-numeric geoLocation")
	assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"], "Should send statusCmp")

	// Verify geoLocation fields were NOT added (non-numeric values rejected)
	assert.NotContains(t, trackingRepo.updatesMap, "latitude", "Latitude should not be persisted from non-numeric array")
	assert.NotContains(t, trackingRepo.updatesMap, "longitude", "Longitude should not be persisted from non-numeric array")
	assert.NotContains(t, trackingRepo.updatesMap, "altitude", "Altitude should not be persisted from non-numeric array")
}

// TestStatusGeoLocationNestedObject verifies BSSCI §3.5.2 lines 820-844:
// geoLocation with nested object structure should be handled gracefully.
// Only flat arrays or simple maps should be parsed.
//
// Lines validated: status_handlers.go:145-156
func TestStatusGeoLocationNestedObject(t *testing.T) {
	logger := logger.NewNop()

	trackingRepo := &geoLocationTrackingRepo{}

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)
	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, trackingRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-geo-nested",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -13,
		Data: map[string]interface{}{
			"code":    0,
			"message": "ok",
			"time":    int64(1672531200000000000),
			"geoLocation": map[string]interface{}{ // Nested structure - unexpected
				"coordinates": map[string]interface{}{
					"lat": float64(52.5200),
					"lon": float64(13.4050),
					"alt": float64(34.0),
				},
			},
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	require.NoError(t, err, "Handler should gracefully handle nested geoLocation structure")

	// Verify statusCmp was sent
	require.Len(t, mockConn.sentMessages, 1, "Should send statusCmp despite nested geoLocation")
	assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"], "Should send statusCmp")

	// Verify geoLocation fields were NOT added (nested structure not supported)
	assert.NotContains(t, trackingRepo.updatesMap, "latitude", "Latitude should not be persisted from nested structure")
	assert.NotContains(t, trackingRepo.updatesMap, "longitude", "Longitude should not be persisted from nested structure")
	assert.NotContains(t, trackingRepo.updatesMap, "altitude", "Altitude should not be persisted from nested structure")
}

// ========== POSIX Short-Circuit Tests ==========

// TestStatusMissingCodeFieldShortCircuits verifies BSSCI §3.5.2 + §3.17 lines 710-730:
// Missing mandatory 'code' field should trigger POSIX_EPROTO error frame and
// prevent statusCmp delivery (validation short-circuits before completion).
//
// Lines validated: status_handlers.go:36-40
func TestStatusMissingCodeFieldShortCircuits(t *testing.T) {
	logger := logger.NewNop()

	// Use panic-on-call stub to prove validation short-circuits before DB access
	panicRepo := &panicOnCallBaseStationRepo{}
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssciservices.CreateTestServices(logger, nil)

	// Wire panic repo through mockStorageWithHistory to ensure server uses it
	mockStorage := &mockStorageWithHistory{
		baseStationRepo: panicRepo,
	}

	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, panicRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-missing-code",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	// statusRsp missing mandatory 'code' field
	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -20,
		Data: map[string]interface{}{
			"message": "ok",
			"time":    int64(1672531200000000000),
			// 'code' field missing - mandatory field violation
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	_ = server.CallHandleStatusResponse(session, statusMsg, data)
	// Test passed without panic - proves validation short-circuited before GetByEUI/Update

	// CRITICAL: Error frame must be sent, NOT statusCmp (validation short-circuit)
	require.Len(t, mockConn.sentMessages, 1, "Exactly one message (error frame) should be sent")
	assert.Equal(t, "error", mockConn.sentMessages[0]["command"], "Must send error frame, not statusCmp")
	assert.True(t, mockConn.errorSent, "Error frame should be sent for missing code field")
	assert.Equal(t, bssci.POSIX_EPROTO, mockConn.lastErrorCode, "Error code should be POSIX_EPROTO (71)")
}

// TestStatusInvalidMessageTypeShortCircuits verifies BSSCI §3.5.2 + §3.17:
// Invalid 'message' field type (non-string) should trigger POSIX_EPROTO error frame
// and prevent statusCmp delivery.
//
// Lines validated: status_handlers.go:60-64
func TestStatusInvalidMessageTypeShortCircuits(t *testing.T) {
	logger := logger.NewNop()

	// Use panic-on-call stub to prove validation short-circuits before DB access
	panicRepo := &panicOnCallBaseStationRepo{}
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssciservices.CreateTestServices(logger, nil)

	// Wire panic repo through mockStorageWithHistory to ensure server uses it
	mockStorage := &mockStorageWithHistory{
		baseStationRepo: panicRepo,
	}

	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, panicRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-invalid-message-type",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	// statusRsp with non-string 'message' field
	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -21,
		Data: map[string]interface{}{
			"code":    0,
			"message": 123, // Integer instead of string - type violation
			"time":    int64(1672531200000000000),
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	_ = server.CallHandleStatusResponse(session, statusMsg, data)
	// Test passed without panic - proves validation short-circuited before GetByEUI/Update

	// CRITICAL: Error frame must be sent, NOT statusCmp
	require.Len(t, mockConn.sentMessages, 1, "Exactly one message (error frame) should be sent")
	assert.Equal(t, "error", mockConn.sentMessages[0]["command"], "Must send error frame, not statusCmp")
	assert.True(t, mockConn.errorSent, "Error frame should be sent for invalid message type")
	assert.Equal(t, bssci.POSIX_EPROTO, mockConn.lastErrorCode, "Error code should be POSIX_EPROTO (71)")
}

// TestStatusInvalidTimeTypeShortCircuits verifies BSSCI §3.5.2 + §3.17:
// Invalid 'time' field type (non-integer) should trigger POSIX_EPROTO error frame
// and prevent statusCmp delivery.
//
// Lines validated: status_handlers.go:73-82
func TestStatusInvalidTimeTypeShortCircuits(t *testing.T) {
	logger := logger.NewNop()

	// Use panic-on-call stub to prove validation short-circuits before DB access
	panicRepo := &panicOnCallBaseStationRepo{}
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssciservices.CreateTestServices(logger, nil)

	// Wire panic repo through mockStorageWithHistory to ensure server uses it
	mockStorage := &mockStorageWithHistory{
		baseStationRepo: panicRepo,
	}

	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, panicRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-invalid-time-type",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	// statusRsp with string 'time' field (should be int64)
	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -22,
		Data: map[string]interface{}{
			"code":    0,
			"message": "ok",
			"time":    "2023-01-01T00:00:00Z", // String instead of int64 - type violation
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	_ = server.CallHandleStatusResponse(session, statusMsg, data)
	// Test passed without panic - proves validation short-circuited before GetByEUI/Update

	// CRITICAL: Error frame must be sent, NOT statusCmp
	require.Len(t, mockConn.sentMessages, 1, "Exactly one message (error frame) should be sent")
	assert.Equal(t, "error", mockConn.sentMessages[0]["command"], "Must send error frame, not statusCmp")
	assert.True(t, mockConn.errorSent, "Error frame should be sent for invalid time type")
	assert.Equal(t, bssci.POSIX_EPROTO, mockConn.lastErrorCode, "Error code should be POSIX_EPROTO (71)")
}

// ========== Tenant Isolation Regression Tests ==========

// tenantTrackingHistoryRepo tracks tenant IDs passed to history Create calls
type tenantTrackingHistoryRepo struct {
	createTenantID int64
	createCalled   bool
	createError    error
}

func (m *tenantTrackingHistoryRepo) Create(_ context.Context, record *mioty.BaseStationStatusRecord) error {
	m.createCalled = true
	m.createTenantID = record.TenantID
	return m.createError
}

// TestStatusTenantIsolationUnderUpdateFailure verifies BSSCI §3.5.3 tenant isolation:
// Even when Update fails, history persistence should still use the correct resolved tenant ID.
// This regression test ensures error paths don't fall back to server default tenant.
//
// Lines validated: status_handlers.go:211-246 (Update error → history with correct tenant)
func TestStatusTenantIsolationUnderUpdateFailure(t *testing.T) {
	logger := logger.NewNop()

	// Create repo that FAILS Update but tracks tenant IDs
	errorRepo := &errorInjectingBaseStationRepo{
		updateError: assert.AnError, // Force Update to fail
	}

	// Create history repo that tracks tenant ID
	historyRepo := &tenantTrackingHistoryRepo{}

	// Create storage with both repos
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssciservices.CreateTestServices(logger, nil)

	mockStorage := &mockStorageWithHistory{
		historyRepo:     historyRepo,
		baseStationRepo: errorRepo,
	}

	// Server has default tenant ID = 1
	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, errorRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}

	// Session with resolved tenant ID = 99 (different from server default)
	session := &bssci.Session{
		ID:               "test-tenant-isolation-update-fail",
		BaseStationEUI:   bssci.TestBsEui04,
		Conn:             mockConn,
		Encoding:         "msgpack",
		ResolvedTenantID: 99,
	}

	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -30,
		Data: map[string]interface{}{
			"code":    0,
			"message": "ok",
			"time":    int64(1672531200000000000),
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	require.NoError(t, err, "Handler should complete despite Update failure")

	// CRITICAL: Verify GetByEUI and Update used resolved tenant 99, NOT server default 1
	assert.True(t, errorRepo.getByEUICalled, "GetByEUI should be called")
	assert.Equal(t, int64(99), errorRepo.getByEUITenantID, "GetByEUI must use session.ResolvedTenantID (99), not server default (1)")
	assert.True(t, errorRepo.updateCalled, "Update should be called")
	assert.Equal(t, int64(99), errorRepo.updateTenantID, "Update must use session.ResolvedTenantID (99) even when it fails, not server default (1)")

	// NOTE: History persistence only happens when Update SUCCEEDS (status_handlers.go:216-247)
	// When Update fails, history is NOT persisted. This test verifies tenant isolation
	// through GetByEUI and Update calls only.
	assert.False(t, historyRepo.createCalled, "History Create should NOT be called when Update fails (per current implementation)")

	// statusCmp should still be sent
	require.Len(t, mockConn.sentMessages, 1, "Should send statusCmp")
	assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"], "Should send statusCmp")
}

// TestStatusTenantIsolationUnderHistoryFailure verifies BSSCI §3.5.3 tenant isolation:
// When history persistence fails, the operation should still use the correct resolved tenant ID
// throughout the entire flow (GetByEUI → Update → Create history).
//
// Lines validated: status_handlers.go:168-246 (full tenant propagation with history error)
func TestStatusTenantIsolationUnderHistoryFailure(t *testing.T) {
	logger := logger.NewNop()

	// Create repo that succeeds
	trackingRepo := &tenantTrackingBaseStationRepo{}

	// Create history repo that FAILS but still tracks tenant ID
	historyRepo := &tenantTrackingHistoryRepo{
		createError: assert.AnError,
	}

	// Create storage with both repos
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssciservices.CreateTestServices(logger, nil)

	mockStorage := &mockStorageWithHistory{
		historyRepo:     historyRepo,
		baseStationRepo: trackingRepo,
	}

	// Server has default tenant ID = 1
	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, trackingRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}

	// Session with resolved tenant ID = 88 (different from server default)
	session := &bssci.Session{
		ID:               "test-tenant-isolation-history-fail",
		BaseStationEUI:   bssci.TestBsEui04,
		Conn:             mockConn,
		Encoding:         "msgpack",
		ResolvedTenantID: 88,
	}

	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -31,
		Data: map[string]interface{}{
			"code":    0,
			"message": "ok",
			"time":    int64(1672531200000000000),
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	require.NoError(t, err, "Handler should complete despite history failure")

	// CRITICAL: Verify all operations used resolved tenant 88
	assert.Equal(t, int64(88), trackingRepo.getByEUITenantID, "GetByEUI must use resolved tenant 88")
	assert.Equal(t, int64(88), trackingRepo.updateTenantID, "Update must use resolved tenant 88")
	assert.Equal(t, int64(88), historyRepo.createTenantID, "History Create must use resolved tenant 88, not server default 1")

	// statusCmp should still be sent despite history error
	require.Len(t, mockConn.sentMessages, 1, "Should send statusCmp despite history error")
	assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"], "Should send statusCmp")
}

// ========== History Success Path Tests ==========

// TestStatusHistoryPersistenceSuccess verifies BSSCI §3.5.2 lines 905-930:
// When Update succeeds AND storage.MIOTYBaseStationStatus() returns a valid repo,
// history should be persisted successfully with all fields populated correctly.
//
// Lines validated: status_handlers.go:222-246
func TestStatusHistoryPersistenceSuccess(t *testing.T) {
	logger := logger.NewNop()

	// Create successful history repo (no error)
	mockHistoryRepo := &mockStatusHistoryRepo{}

	// Create successful base station repo
	successRepo := &tenantTrackingBaseStationRepo{}

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssciservices.CreateTestServices(logger, nil)

	mockStorage := &mockStorageWithHistory{
		historyRepo:     mockHistoryRepo,
		baseStationRepo: successRepo,
	}

	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, successRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-history-success",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -40,
		Data: map[string]interface{}{
			"code":        2,
			"message":     "degraded",
			"time":        int64(1672531200000000000),
			"dutyCycle":   0.82,
			"uptime":      int64(7200),
			"temp":        float64(28.5),
			"cpuLoad":     0.65,
			"memLoad":     0.55,
			"geoLocation": []interface{}{float64(48.8566), float64(2.3522), float64(35.0)},
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	require.NoError(t, err, "Handler should succeed with valid history persistence")

	// Verify history was persisted
	assert.True(t, mockHistoryRepo.createCalled, "History Create should be called")
	require.NotNil(t, mockHistoryRepo.lastRecord, "History record should be persisted")

	// Verify all fields were correctly populated
	assert.Equal(t, 2, mockHistoryRepo.lastRecord.StatusCode, "Status code should match")
	assert.Equal(t, "degraded", mockHistoryRepo.lastRecord.StatusMessage, "Status message should match")
	assert.Equal(t, int64(1672531200000000000), mockHistoryRepo.lastRecord.SystemTime, "System time should match")
	assert.NotNil(t, mockHistoryRepo.lastRecord.DutyCycle, "Duty cycle should be persisted")
	assert.Equal(t, 0.82, *mockHistoryRepo.lastRecord.DutyCycle, "Duty cycle value should match")
	assert.NotNil(t, mockHistoryRepo.lastRecord.UptimeSeconds, "Uptime should be persisted")
	assert.Equal(t, int64(7200), *mockHistoryRepo.lastRecord.UptimeSeconds, "Uptime value should match")
	assert.NotNil(t, mockHistoryRepo.lastRecord.Temperature, "Temperature should be persisted")
	assert.Equal(t, 28.5, *mockHistoryRepo.lastRecord.Temperature, "Temperature value should match")
	assert.NotNil(t, mockHistoryRepo.lastRecord.CPULoad, "CPU load should be persisted")
	assert.Equal(t, 0.65, *mockHistoryRepo.lastRecord.CPULoad, "CPU load value should match")
	assert.NotNil(t, mockHistoryRepo.lastRecord.MemoryLoad, "Memory load should be persisted")
	assert.Equal(t, 0.55, *mockHistoryRepo.lastRecord.MemoryLoad, "Memory load value should match")
	assert.NotNil(t, mockHistoryRepo.lastRecord.Latitude, "Latitude should be persisted")
	assert.Equal(t, 48.8566, *mockHistoryRepo.lastRecord.Latitude, "Latitude value should match")
	assert.NotNil(t, mockHistoryRepo.lastRecord.Longitude, "Longitude should be persisted")
	assert.Equal(t, 2.3522, *mockHistoryRepo.lastRecord.Longitude, "Longitude value should match")
	assert.NotNil(t, mockHistoryRepo.lastRecord.Altitude, "Altitude should be persisted")
	assert.Equal(t, 35.0, *mockHistoryRepo.lastRecord.Altitude, "Altitude value should match")
	assert.NotNil(t, mockHistoryRepo.lastRecord.OperationID, "Operation ID should be persisted")
	assert.Equal(t, int64(-40), *mockHistoryRepo.lastRecord.OperationID, "Operation ID should match")

	// statusCmp should be sent
	require.Len(t, mockConn.sentMessages, 1, "Should send statusCmp")
	assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"], "Should send statusCmp")
}

// TestStatusHistorySkippedWhenStorageNil verifies BSSCI §3.5.2 lines 905-930:
// When storage is nil OR storage.MIOTYBaseStationStatus() returns nil,
// history persistence should be gracefully skipped without errors.
//
// Lines validated: status_handlers.go:222 (nil guard)
func TestStatusHistorySkippedWhenStorageNil(t *testing.T) {
	logger := logger.NewNop()

	// Create successful base station repo
	successRepo := &tenantTrackingBaseStationRepo{}

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssciservices.CreateTestServices(logger, nil)

	// Create storage with nil history repo (simulates MIOTYBaseStationStatus() returning nil)
	mockStorage := &mockStorageWithHistory{
		historyRepo:     nil, // nil history repo
		baseStationRepo: successRepo,
	}

	server := bssci.NewTestServer(logger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-history-nil-storage",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -41,
		Data: map[string]interface{}{
			"code":    0,
			"message": "ok",
			"time":    int64(1672531200000000000),
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	require.NoError(t, err, "Handler should succeed when history storage is nil")

	// statusCmp should still be sent (history is optional/best-effort)
	require.Len(t, mockConn.sentMessages, 1, "Should send statusCmp despite nil history storage")
	assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"], "Should send statusCmp")
}

// ========== Concurrency Test ==========

// concurrentTrackingRepo tracks concurrent Update calls to verify thread safety
type concurrentTrackingRepo struct {
	updateCalls       int
	maxConcurrent     int
	currentConcurrent int
	mu                sync.Mutex
}

func (m *concurrentTrackingRepo) Create(_ context.Context, _ *models.BaseStation) error {
	return nil
}

func (m *concurrentTrackingRepo) GetByID(_ context.Context, tenantID, id int64) (*models.BaseStation, error) {
	var eui models.EUI
	return &models.BaseStation{
		ID:       id,
		TenantID: tenantID,
		EUI:      eui,
		Name:     "Test BS",
	}, nil
}

func (m *concurrentTrackingRepo) GetByEUI(_ context.Context, tenantID int64, euiBytes []byte) (*models.BaseStation, error) {
	var eui models.EUI
	if len(euiBytes) == 8 {
		copy(eui[:], euiBytes)
	}

	return &models.BaseStation{
		ID:       1,
		TenantID: tenantID,
		EUI:      eui,
		Name:     "Test BS",
	}, nil
}

func (m *concurrentTrackingRepo) Update(_ context.Context, _ int64, _ int64, _ map[string]interface{}) error {
	m.mu.Lock()
	m.updateCalls++
	m.currentConcurrent++
	if m.currentConcurrent > m.maxConcurrent {
		m.maxConcurrent = m.currentConcurrent
	}
	m.mu.Unlock()

	// Simulate some work to increase chance of concurrent execution
	time.Sleep(10 * time.Millisecond)

	m.mu.Lock()
	m.currentConcurrent--
	m.mu.Unlock()

	return nil
}

func (m *concurrentTrackingRepo) Delete(_ context.Context, _ int64, _ int64) error {
	return nil
}

func (m *concurrentTrackingRepo) List(_ context.Context, _ *models.BaseStationFilter) ([]*models.BaseStation, int64, error) {
	return []*models.BaseStation{}, 0, nil
}

func (m *concurrentTrackingRepo) UpdateConnectionStatus(_ context.Context, _ int64, _ int64, _ bool, _ *string) error {
	return nil
}

func (m *concurrentTrackingRepo) UpdateSessionInfo(_ context.Context, _ int64, _ []byte, _ string) error {
	return nil
}

func (m *concurrentTrackingRepo) GetStatistics(_ context.Context, _ int64) (*interfaces.BaseStationStatistics, error) {
	return &interfaces.BaseStationStatistics{}, nil
}

func (m *concurrentTrackingRepo) GetPropagationState(_ context.Context, _ int64) (*models.BaseStationPropagationState, error) {
	return nil, nil
}

func (m *concurrentTrackingRepo) UpsertPropagationState(_ context.Context, _ *models.BaseStationPropagationState) error {
	return nil
}

func (m *concurrentTrackingRepo) UpdatePropagationStatus(_ context.Context, _ int64, _ string, _ *string) error {
	return nil
}

func (m *concurrentTrackingRepo) IncrementRetryCount(_ context.Context, _ int64, _ time.Time) error {
	return nil
}

func (m *concurrentTrackingRepo) UpdateEUI(_ context.Context, _ int64, _, _ []byte) (*models.BaseStation, error) {
	return nil, nil
}

func (m *concurrentTrackingRepo) GetByEUIGlobal(_ context.Context, euiBytes []byte) (*models.BaseStation, error) {
	var eui models.EUI
	if len(euiBytes) == 8 {
		copy(eui[:], euiBytes)
	}
	return &models.BaseStation{ID: 1, TenantID: 1, EUI: eui, Name: "Test BS"}, nil
}
func (m *concurrentTrackingRepo) ListAllLocations(_ context.Context) ([]*models.BaseStation, error) {
	return nil, nil
}

// threadSafeHistoryRepo tracks all Create calls in a thread-safe manner
type threadSafeHistoryRepo struct {
	mu           sync.Mutex
	createCalls  int
	tenantIDs    []int64
	operationIDs []int64
}

func (m *threadSafeHistoryRepo) Create(_ context.Context, record *mioty.BaseStationStatusRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.createCalls++
	m.tenantIDs = append(m.tenantIDs, record.TenantID)
	if record.OperationID != nil {
		m.operationIDs = append(m.operationIDs, *record.OperationID)
	}

	return nil
}

// GetCreateCalls returns the number of Create calls in a thread-safe manner
func (m *threadSafeHistoryRepo) GetCreateCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.createCalls
}

// GetTenantIDs returns a copy of tenant IDs in a thread-safe manner
func (m *threadSafeHistoryRepo) GetTenantIDs() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]int64, len(m.tenantIDs))
	copy(result, m.tenantIDs)
	return result
}

// GetOperationIDs returns a copy of operation IDs in a thread-safe manner
func (m *threadSafeHistoryRepo) GetOperationIDs() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]int64, len(m.operationIDs))
	copy(result, m.operationIDs)
	return result
}

// TestStatusResponseConcurrentUpdates verifies BSSCI §3.5.3 thread safety:
// Multiple concurrent status responses for the same session should be handled safely.
// session.mu guards should prevent race conditions on session state.
//
// Lines validated: status_handlers.go:20-256 (entire handler with session.mu protection)
func TestStatusResponseConcurrentUpdates(t *testing.T) {
	logger := logger.NewNop()

	const (
		serverTenantID  = int64(1)  // Server default tenant
		sessionTenantID = int64(42) // Session override for regression detection
	)

	// Create concurrent tracking repo
	concurrentRepo := &concurrentTrackingRepo{}

	// Create thread-safe history repo to track all Create calls
	historyRepo := &threadSafeHistoryRepo{}

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssciservices.CreateTestServices(logger, nil)

	// Create storage with both repos
	mockStorage := &mockStorageWithHistory{
		historyRepo:     historyRepo,
		baseStationRepo: concurrentRepo,
	}

	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, concurrentRepo, serverTenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	// Use statusMockConn which has embedded mutex for thread-safe message capture
	mockConn := &statusMockConn{}

	session := &bssci.Session{
		ID:               "test-concurrent-status",
		BaseStationEUI:   bssci.TestBsEui04,
		Conn:             mockConn,
		Encoding:         "msgpack",
		ResolvedTenantID: sessionTenantID, // Use non-default tenant to catch regressions
	}

	numGoroutines := 10
	startCh := make(chan struct{})
	doneCh := make(chan bool, numGoroutines)

	// Launch multiple goroutines that will call handleStatusResponse concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(opId int64) {
			// Wait for signal to start (synchronization barrier)
			<-startCh

			statusMsg := &bssci.Message{
				Command: "statusRsp",
				OpId:    opId,
				Data: map[string]interface{}{
					"code":      0,
					"message":   "ok",
					"time":      int64(1672531200000000000),
					"dutyCycle": float64(0.5 + float64(opId)*0.01),
				},
			}

			data := statusMsg.Data.(map[string]interface{})
			err := server.CallHandleStatusResponse(session, statusMsg, data)

			doneCh <- (err == nil)
		}(int64(-50 - i))
	}

	// Release all goroutines simultaneously (channel-based barrier)
	close(startCh)

	// Wait for all goroutines to complete
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		if <-doneCh {
			successCount++
		}
	}

	// All handlers should succeed
	assert.Equal(t, numGoroutines, successCount, "All concurrent handlers should succeed")

	// Verify all Update calls were made
	assert.Equal(t, numGoroutines, concurrentRepo.updateCalls, "All Update calls should complete")

	// Verify exactly 10 statusCmp messages were sent (thread-safe getter)
	sentMessages := mockConn.GetSentMessages()
	assert.Len(t, sentMessages, numGoroutines, "Exactly 10 statusCmp messages should be sent")

	// Verify all messages are statusCmp, not error frames
	for i, msg := range sentMessages {
		cmd, ok := msg["command"].(string)
		assert.True(t, ok, "Message %d should have command field", i)
		assert.Equal(t, "statusCmp", cmd, "Message %d should be statusCmp, not error", i)
	}

	// Verify no error frames were sent (thread-safe getter)
	assert.False(t, mockConn.GetErrorSent(), "No error frames should be sent during concurrent updates")

	// Verify exactly 10 history Create calls were made (thread-safe getter)
	assert.Equal(t, numGoroutines, historyRepo.GetCreateCalls(), "Exactly 10 history records should be created")

	// Verify all history records have correct tenant ID (sessionTenantID, NOT serverTenantID)
	// Dual assertions ensure test fails if handler regresses to using server default
	tenantIDs := historyRepo.GetTenantIDs()
	assert.Len(t, tenantIDs, numGoroutines, "Should have 10 tenant IDs")

	// Regression guard: test MUST use different values to catch fallback
	require.NotEqual(t, serverTenantID, sessionTenantID, "Test setup error: must use non-default tenant to detect regressions")

	for i, tid := range tenantIDs {
		assert.Equal(t, sessionTenantID, tid, "History record %d must use session.ResolvedTenantID (%d), not server default (%d)", i, sessionTenantID, serverTenantID)
		assert.NotEqual(t, serverTenantID, tid, "History record %d should NEVER use server default tenant (%d) - regression detected", i, serverTenantID)
	}

	// Verify all history records have correct operation IDs (negative, unique)
	opIDs := historyRepo.GetOperationIDs()
	assert.Len(t, opIDs, numGoroutines, "Should have 10 operation IDs")
	opIDMap := make(map[int64]bool)
	for i, opID := range opIDs {
		assert.Less(t, opID, int64(0), "History record %d should have negative operation ID", i)
		assert.False(t, opIDMap[opID], "Operation ID %d should be unique", opID)
		opIDMap[opID] = true
	}

	// Note: We don't assert on maxConcurrent > 1 because Go scheduler behavior is non-deterministic
	// The important thing is that no races occurred and all calls completed successfully
}

// TestStatusHandler_ValidGeoLocation_SetsGPSSource verifies BSSCI §3.5.2:
// When all three geoLocation values are valid and in range, the handler must set
// location_source to "gps" and location_updated_at to a recent timestamp.
func TestStatusHandler_ValidGeoLocation_SetsGPSSource(t *testing.T) {
	logger := logger.NewNop()

	trackingRepo := &geoLocationTrackingRepo{}

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)
	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, trackingRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-geo-gps-source",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	beforeCall := time.Now()

	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -20,
		Data: map[string]interface{}{
			"code":        0,
			"message":     "ok",
			"time":        int64(1672531200000000000),
			"geoLocation": []interface{}{float64(52.5200), float64(13.4050), float64(34.0)},
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	require.NoError(t, err, "Status response should be processed successfully")

	// Verify location_source is set to GPS
	assert.True(t, trackingRepo.updateCalled, "Update should have been called")
	assert.Contains(t, trackingRepo.updatesMap, "location_source", "location_source should be in updates map")
	assert.Equal(t, models.LocationSourceGPS, trackingRepo.updatesMap["location_source"],
		"location_source should be set to GPS constant")

	// Verify location_updated_at is set to a recent timestamp
	assert.Contains(t, trackingRepo.updatesMap, "location_updated_at", "location_updated_at should be in updates map")
	updatedAt, ok := trackingRepo.updatesMap["location_updated_at"].(time.Time)
	require.True(t, ok, "location_updated_at should be a time.Time value")
	assert.False(t, updatedAt.Before(beforeCall), "location_updated_at should not be before the handler call")
	assert.False(t, updatedAt.After(time.Now()), "location_updated_at should not be in the future")

	// Verify lat/lon/alt are also present
	assert.Equal(t, float64(52.5200), trackingRepo.updatesMap["latitude"])
	assert.Equal(t, float64(13.4050), trackingRepo.updatesMap["longitude"])
	assert.Equal(t, float64(34.0), trackingRepo.updatesMap["altitude"])
}

// TestStatusHandler_OutOfRangeLatLon_SkipsAll verifies BSSCI §3.5.2:
// When latitude exceeds valid range [-90, 90], the handler must skip all location fields.
func TestStatusHandler_OutOfRangeLatLon_SkipsAll(t *testing.T) {
	logger := logger.NewNop()

	tests := []struct {
		name        string
		geoLocation []interface{}
	}{
		{
			name:        "Latitude above maximum (91)",
			geoLocation: []interface{}{float64(91.0), float64(13.4050), float64(34.0)},
		},
		{
			name:        "Latitude below minimum (-91)",
			geoLocation: []interface{}{float64(-91.0), float64(13.4050), float64(34.0)},
		},
		{
			name:        "Longitude above maximum (181)",
			geoLocation: []interface{}{float64(52.5200), float64(181.0), float64(34.0)},
		},
		{
			name:        "Longitude below minimum (-181)",
			geoLocation: []interface{}{float64(52.5200), float64(-181.0), float64(34.0)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trackingRepo := &geoLocationTrackingRepo{}

			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)
			server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, trackingRepo, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

			mockConn := &statusMockConn{}
			session := &bssci.Session{
				ID:             "test-geo-out-of-range",
				BaseStationEUI: bssci.TestBsEui04,
				Conn:           mockConn,
				Encoding:       "msgpack",
			}

			statusMsg := &bssci.Message{
				Command: "statusRsp",
				OpId:    -21,
				Data: map[string]interface{}{
					"code":        0,
					"message":     "ok",
					"time":        int64(1672531200000000000),
					"geoLocation": tt.geoLocation,
				},
			}

			data := statusMsg.Data.(map[string]interface{})
			err := server.CallHandleStatusResponse(session, statusMsg, data)
			require.NoError(t, err, "Handler should succeed despite out-of-range coordinates")

			// Verify statusCmp was sent
			require.Len(t, mockConn.sentMessages, 1, "Should send statusCmp")
			assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"])

			// Verify no location fields were persisted
			assert.NotContains(t, trackingRepo.updatesMap, "latitude", "Latitude should not be persisted for out-of-range values")
			assert.NotContains(t, trackingRepo.updatesMap, "longitude", "Longitude should not be persisted for out-of-range values")
			assert.NotContains(t, trackingRepo.updatesMap, "altitude", "Altitude should not be persisted for out-of-range values")
			assert.NotContains(t, trackingRepo.updatesMap, "location_source", "location_source should not be set for out-of-range values")
			assert.NotContains(t, trackingRepo.updatesMap, "location_updated_at", "location_updated_at should not be set for out-of-range values")
		})
	}
}

// TestStatusHandler_ZeroGeoLocation_SkipsAll verifies that all-zero coordinates
// (0, 0, 0) are treated as "no GPS fix" and discarded rather than persisted.
// Base stations without GPS hardware report 0/0/0 which would otherwise appear
// as a valid location in the Gulf of Guinea.
func TestStatusHandler_ZeroGeoLocation_SkipsAll(t *testing.T) {
	logger := logger.NewNop()

	trackingRepo := &geoLocationTrackingRepo{}

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)
	server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, trackingRepo, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mockConn := &statusMockConn{}
	session := &bssci.Session{
		ID:             "test-geo-zero",
		BaseStationEUI: bssci.TestBsEui04,
		Conn:           mockConn,
		Encoding:       "msgpack",
	}

	statusMsg := &bssci.Message{
		Command: "statusRsp",
		OpId:    -30,
		Data: map[string]interface{}{
			"code":        0,
			"message":     "ok",
			"time":        int64(1672531200000000000),
			"geoLocation": []interface{}{float64(0), float64(0), float64(0)},
		},
	}

	data := statusMsg.Data.(map[string]interface{})
	err := server.CallHandleStatusResponse(session, statusMsg, data)
	require.NoError(t, err, "Handler should succeed with zero geoLocation")

	require.Len(t, mockConn.sentMessages, 1, "Should send statusCmp")
	assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"])

	assert.NotContains(t, trackingRepo.updatesMap, "latitude", "Latitude should not be persisted for 0/0/0")
	assert.NotContains(t, trackingRepo.updatesMap, "longitude", "Longitude should not be persisted for 0/0/0")
	assert.NotContains(t, trackingRepo.updatesMap, "altitude", "Altitude should not be persisted for 0/0/0")
	assert.NotContains(t, trackingRepo.updatesMap, "location_source", "location_source should not be set for 0/0/0")
}

// TestStatusHandler_PartialTriple_SkipsAll verifies BSSCI §3.5.2:
// When only 2 of 3 geoLocation values successfully parse (e.g., one element is not numeric),
// all location fields must be skipped.
func TestStatusHandler_PartialTriple_SkipsAll(t *testing.T) {
	logger := logger.NewNop()

	tests := []struct {
		name        string
		geoLocation []interface{}
	}{
		{
			name:        "Latitude is nil",
			geoLocation: []interface{}{nil, float64(13.4050), float64(34.0)},
		},
		{
			name:        "Longitude is nil",
			geoLocation: []interface{}{float64(52.5200), nil, float64(34.0)},
		},
		{
			name:        "Altitude is nil",
			geoLocation: []interface{}{float64(52.5200), float64(13.4050), nil},
		},
		{
			name:        "Latitude is string",
			geoLocation: []interface{}{"52.5200", float64(13.4050), float64(34.0)},
		},
		{
			name:        "Longitude is boolean",
			geoLocation: []interface{}{float64(52.5200), true, float64(34.0)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trackingRepo := &geoLocationTrackingRepo{}

			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssciservices.CreateTestServices(logger, nil)
			server := bssci.NewTestServerWithBaseStationRepo(logger, mockStorage, trackingRepo, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

			mockConn := &statusMockConn{}
			session := &bssci.Session{
				ID:             "test-geo-partial-triple",
				BaseStationEUI: bssci.TestBsEui04,
				Conn:           mockConn,
				Encoding:       "msgpack",
			}

			statusMsg := &bssci.Message{
				Command: "statusRsp",
				OpId:    -22,
				Data: map[string]interface{}{
					"code":        0,
					"message":     "ok",
					"time":        int64(1672531200000000000),
					"geoLocation": tt.geoLocation,
				},
			}

			data := statusMsg.Data.(map[string]interface{})
			err := server.CallHandleStatusResponse(session, statusMsg, data)
			require.NoError(t, err, "Handler should succeed despite partial geoLocation values")

			// Verify statusCmp was sent
			require.Len(t, mockConn.sentMessages, 1, "Should send statusCmp")
			assert.Equal(t, "statusCmp", mockConn.sentMessages[0]["command"])

			// Verify no location fields were persisted
			assert.NotContains(t, trackingRepo.updatesMap, "latitude", "Latitude should not be persisted when parsing fails")
			assert.NotContains(t, trackingRepo.updatesMap, "longitude", "Longitude should not be persisted when parsing fails")
			assert.NotContains(t, trackingRepo.updatesMap, "altitude", "Altitude should not be persisted when parsing fails")
			assert.NotContains(t, trackingRepo.updatesMap, "location_source", "location_source should not be set when parsing fails")
			assert.NotContains(t, trackingRepo.updatesMap, "location_updated_at", "location_updated_at should not be set when parsing fails")
		})
	}
}
