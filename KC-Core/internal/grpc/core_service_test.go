package grpc

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	healthstatus "github.com/Kiloiot/kilo-service-center/KC-Core/internal/health"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	kcerrors "github.com/Kiloiot/kilo-service-center/KC-DB/common/errors"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// ============================================================================
// Mock Implementations
// ============================================================================

// mockBasestationSvc implements grpcservices.BaseStationService for testing
type mockBasestationSvc struct {
	getByEUIFunc func(ctx context.Context, eui []byte, tenantID int64) (*models.BaseStation, error)
	createFunc   func(ctx context.Context, bs *models.BaseStation) (*models.BaseStation, error)
	updateFunc   func(ctx context.Context, bs *models.BaseStation) (*models.BaseStation, error)
}

func (m *mockBasestationSvc) Create(ctx context.Context, bs *models.BaseStation) (*models.BaseStation, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, bs)
	}
	return nil, nil
}

func (m *mockBasestationSvc) GetByEUI(ctx context.Context, eui []byte, tenantID int64) (*models.BaseStation, error) {
	if m.getByEUIFunc != nil {
		return m.getByEUIFunc(ctx, eui, tenantID)
	}
	return nil, nil
}

func (m *mockBasestationSvc) Update(ctx context.Context, bs *models.BaseStation) (*models.BaseStation, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, bs)
	}
	return nil, nil
}

func (m *mockBasestationSvc) Delete(_ context.Context, _ []byte, _ int64) error {
	return nil
}

func (m *mockBasestationSvc) List(_ context.Context, _ int64, _ int, _ int) ([]*models.BaseStation, error) {
	return nil, nil
}

func (m *mockBasestationSvc) UpdateEUI(_ context.Context, _ int64, _, _ []byte) (*models.BaseStation, error) {
	return nil, nil
}

func (m *mockBasestationSvc) ListAllLocations(_ context.Context) ([]*models.BaseStation, error) {
	return nil, nil
}

// mockSessionDir implements bssci.SessionDirectory for testing
type mockSessionDir struct {
	selectBidiFunc  func(tenantID int64, targetBsEui *uint64) (sessionID string, actualBsEui uint64, err error)
	testSessions    map[string]*bssci.Session
	findSessionHook func() // Hook for observing FindSessionForEndpointAttachment calls
}

// AddTestSession registers a properly initialized session for testing
// IMPORTANT: Caller must set HandshakeComplete, Bidirectional, ResolvedTenantID, Connected, LastSeen
func (m *mockSessionDir) AddTestSession(session *bssci.Session) {
	if m.testSessions == nil {
		m.testSessions = make(map[string]*bssci.Session)
	}
	m.testSessions[session.ID] = session
}

// GetConnectedSessions returns sessions matching production schema EXACTLY
// Production: server.go:4695-4707 (does NOT include resolvedTenantID in map)
func (m *mockSessionDir) GetConnectedSessions() []map[string]interface{} {
	sessions := make([]map[string]interface{}, 0, len(m.testSessions))
	for id, session := range m.testSessions {
		sessions = append(sessions, map[string]interface{}{
			"id":                id,
			"baseStationEui":    session.BaseStationEUI,
			"connected":         session.Connected,
			"lastSeen":          session.LastSeen,
			"vendor":            session.Vendor,
			"model":             session.Model,
			"name":              session.Name,
			"clientVersion":     session.ClientVersion,
			"negotiatedVersion": session.NegotiatedVersion,
			"bidirectional":     session.Bidirectional,
			"handshakeComplete": session.HandshakeComplete,
			// NO resolvedTenantID - production doesn't export it in the map
		})
	}
	return sessions
}

func (m *mockSessionDir) GetSessionByEUI(_ uint64) interface{} {
	return nil
}

func (m *mockSessionDir) SelectBidirectionalSession(tenantID int64, targetBsEui *uint64) (sessionID string, actualBsEui uint64, err error) {
	if m.selectBidiFunc != nil {
		return m.selectBidiFunc(tenantID, targetBsEui)
	}
	return "", 0, nil
}

// FindSessionForEndpointAttachment implements SessionDirectory interface for testing
// Invokes findSessionHook if set to allow tests to observe calls
func (m *mockSessionDir) FindSessionForEndpointAttachment(bsEui uint64) (string, error) {
	// Invoke hook if set (for tenant isolation tests)
	if m.findSessionHook != nil {
		m.findSessionHook()
	}

	// Iterate through connected sessions
	for _, session := range m.GetConnectedSessions() {
		if bsEuiVal, ok := session["baseStationEui"].(uint64); ok && bsEuiVal == bsEui {
			if id, ok := session["id"].(string); ok {
				// Check bidirectional capability
				if bidi, ok := session["bidirectional"].(bool); ok && bidi {
					// Check handshake complete
					if handshake, ok := session["handshakeComplete"].(bool); ok && handshake {
						return id, nil
					}
					return "", bssci.ErrSessionNotReady
				}
				return "", bssci.ErrSessionNotBidirectional
			}
		}
	}
	return "", bssci.ErrSessionNotFound
}

// mockULTransmit implements bssci.ULTransmitter for testing
type mockULTransmit struct {
	sendFunc func(sessionID string, epEui uint64, nwkSnKey []byte, shAddr uint16,
		packetCnt uint32, userData []byte, profile string, format uint8) (int64, error)
}

func (m *mockULTransmit) SendULDataTransmit(sessionID string, epEui uint64, nwkSnKey []byte,
	shAddr uint16, packetCnt uint32, userData []byte, profile string, format uint8) (int64, error) {
	if m.sendFunc != nil {
		return m.sendFunc(sessionID, epEui, nwkSnKey, shAddr, packetCnt, userData, profile, format)
	}
	return -1, nil
}

// mockLogger implements logger.Logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(_ string, _ ...interface{})                           {}
func (m *mockLogger) Info(_ string, _ ...interface{})                            {}
func (m *mockLogger) Warn(_ string, _ ...interface{})                            {}
func (m *mockLogger) Error(_ string, _ ...interface{})                           {}
func (m *mockLogger) Fatal(_ string, _ ...interface{})                           {}
func (m *mockLogger) DebugContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLogger) InfoContext(_ context.Context, _ string, _ ...interface{})  {}
func (m *mockLogger) WarnContext(_ context.Context, _ string, _ ...interface{})  {}
func (m *mockLogger) ErrorContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLogger) FatalContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLogger) WithField(_ string, _ interface{}) logger.Logger            { return m }
func (m *mockLogger) WithFields(_ map[string]interface{}) logger.Logger          { return m }

// mockMessageSvc implements grpcservices.MessageService for testing
type mockMessageSvc struct {
	getEndpointBSFunc func(ctx context.Context, epEui, tenantID string) (string, error)
}

func (m *mockMessageSvc) GetDownlinkByQueueID(_ context.Context, _ uint64, _ string) (*storage.DownlinkMessage, error) {
	return nil, nil
}

func (m *mockMessageSvc) GetDownlinkQueue(_ context.Context, _, _ string) ([]*storage.DownlinkMessage, error) {
	return nil, nil
}

func (m *mockMessageSvc) GetDownlinkResults(_ context.Context, _, _ string, _ *uuid.UUID, _ string,
	_, _ *time.Time, _, _ int) ([]*storage.DownlinkMessage, int, error) {
	return nil, 0, nil
}

func (m *mockMessageSvc) GetDLRXStatusByEndpoint(_ context.Context, _ int64, _ []byte,
	_, _ int, _, _ *time.Time) ([]*mioty.DLRXStatus, int, error) {
	return nil, 0, nil
}

func (m *mockMessageSvc) GetAverageDLRXMetrics(_ context.Context, _ int64, _ []byte,
	_, _ *time.Time) (avgSnr, avgRssi float64, count int, err error) {
	return 0, 0, 0, nil
}

func (m *mockMessageSvc) GetEndpointBaseStation(ctx context.Context, epEui, tenantID string) (string, error) {
	if m.getEndpointBSFunc != nil {
		return m.getEndpointBSFunc(ctx, epEui, tenantID)
	}
	return "", storage.ErrNotFound
}

// mockDownlinkCmd implements bssci.DownlinkCommander for testing
type mockDownlinkCmd struct {
	sendDLRXStatusQueryFunc func(sessionID string, epEui uint64) error
	lastSessionID           string
	lastEpEui               uint64
	callCount               int
}

// SendDLDataQueue - EXACT signature: 13 parameters, returns error
func (m *mockDownlinkCmd) SendDLDataQueue(
	_ string,
	_ uint64,
	_ [][]byte, // NOT []byte - array of byte arrays!
	_ int64,
	_ float32,
	_ bool,
	_ []int64,
	_ uint8,
	_ bool,
	_ bool,
	_ bool,
	_ bool,
	_ int64,
	_ bool,
) error {
	return nil
}

// SendDLDataRevoke - EXACT signature
func (m *mockDownlinkCmd) SendDLDataRevoke(_ string, _ uint64, _ uint64) error {
	return nil
}

// SendDLRXStatusQuery - EXACT signature: returns error (NOT int64)
func (m *mockDownlinkCmd) SendDLRXStatusQuery(sessionID string, epEui uint64) error {
	m.callCount++
	m.lastSessionID = sessionID
	m.lastEpEui = epEui
	if m.sendDLRXStatusQueryFunc != nil {
		return m.sendDLRXStatusQueryFunc(sessionID, epEui)
	}
	return nil
}

// ============================================================================
// Test cases for SendULTransmit bsEui ownership guard.
// ============================================================================

// TestSendULTransmit_CrossTenantBaseStation verifies that requesting a base station
// owned by a different tenant returns codes.NotFound without leaking information
// about the base station's existence or offline status.
//
// Security requirement: tenant isolation prevents cross-tenant information disclosure.
func TestSendULTransmit_CrossTenantBaseStation(t *testing.T) {
	// Setup: Create service with mock that returns ErrNotFound for cross-tenant BS
	mockBsSvc := &mockBasestationSvc{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.BaseStation, error) {
			// Simulate base station not found (cross-tenant access attempt)
			return nil, storage.ErrNotFound
		},
	}

	svc := &CoreService{
		basestationSvc: mockBsSvc,
		log:            &mockLogger{},
	}

	// Execute: Request UL transmit with cross-tenant base station
	ctx := testutil.TestContextWithTenant(100) // Tenant 100
	req := &pb.SendULTransmitRequest{
		EpEui:     "0000000000000001",
		BsEui:     "AAAAAAAAAAAAAAAA", // Base station owned by different tenant
		NwkSnKey:  make([]byte, 16),
		ShAddr:    1,
		PacketCnt: 1,
		UserData:  []byte("test"),
		Profile:   "default",
		Format:    0,
	}

	resp, err := svc.SendULTransmit(ctx, req)

	// Verify: Returns NotFound without leaking information
	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	assert.Equal(t, codes.NotFound, st.Code(), "Should return NotFound for cross-tenant BS")
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound), st.Message())
}

// TestSendULTransmit_ValidTenantOfflineBaseStation verifies that when a valid tenant
// requests a base station they own, but the base station is offline, the ownership
// check passes and the error comes from session selection (not ownership validation).
//
// Security requirement: ownership check must happen before session selection.
func TestSendULTransmit_ValidTenantOfflineBaseStation(t *testing.T) {
	// Setup: Create service with mock that passes ownership check but fails session selection
	mockBsSvc := &mockBasestationSvc{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.BaseStation, error) {
			// Ownership check passes - return valid base station
			return &models.BaseStation{
				EUI:      models.EUIFromString("AAAAAAAAAAAAAAAA"),
				Name:     "Test BS",
				TenantID: 100,
			}, nil
		},
	}

	mockSessionDir := &mockSessionDir{
		selectBidiFunc: func(_ int64, _ *uint64) (string, uint64, error) {
			// Base station is offline - session selection fails
			return "", 0, bssci.ErrBaseStationUnavailable
		},
	}

	svc := &CoreService{
		basestationSvc: mockBsSvc,
		sessionDir:     mockSessionDir,
		log:            &mockLogger{},
	}

	// Execute: Request UL transmit with owned but offline base station
	ctx := testutil.TestContextWithTenant(100)
	req := &pb.SendULTransmitRequest{
		EpEui:     "0000000000000001",
		BsEui:     "AAAAAAAAAAAAAAAA", // Valid base station owned by tenant 100
		NwkSnKey:  make([]byte, 16),
		ShAddr:    1,
		PacketCnt: 1,
		UserData:  []byte("test"),
		Profile:   "default",
		Format:    0,
	}

	resp, err := svc.SendULTransmit(ctx, req)

	// Verify: Returns NotFound due to offline base station (not ownership issue)
	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	assert.Equal(t, codes.NotFound, st.Code(), "Should return NotFound for offline BS")
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound), st.Message())
}

// TestSendULTransmit_DatabaseErrorDuringOwnershipCheck verifies that database errors
// during the ownership check are properly handled and return codes.Internal.
//
// Requirement: graceful error handling for infrastructure failures.
func TestSendULTransmit_DatabaseErrorDuringOwnershipCheck(t *testing.T) {
	// Setup: Create service with mock that returns database error
	mockBsSvc := &mockBasestationSvc{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.BaseStation, error) {
			// Simulate database connection failure
			return nil, fmt.Errorf("database connection failed")
		},
	}

	svc := &CoreService{
		basestationSvc: mockBsSvc,
		log:            &mockLogger{},
	}

	// Execute: Request UL transmit when database is unavailable
	ctx := testutil.TestContextWithTenant(100)
	req := &pb.SendULTransmitRequest{
		EpEui:     "0000000000000001",
		BsEui:     "AAAAAAAAAAAAAAAA",
		NwkSnKey:  make([]byte, 16),
		ShAddr:    1,
		PacketCnt: 1,
		UserData:  []byte("test"),
		Profile:   "default",
		Format:    0,
	}

	resp, err := svc.SendULTransmit(ctx, req)

	// Verify: Returns Internal error for database failures
	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	assert.Equal(t, codes.Internal, st.Code(), "Should return Internal for database errors")
	assert.Contains(t, st.Message(), "failed to verify base station ownership",
		"Error message should indicate ownership verification failure")
}

// TestSendULTransmit_NoBsEuiSpecified verifies that when no base station EUI is
// specified in the request, the ownership check is skipped and the system proceeds
// directly to session selection (which will select any available bidirectional BS).
//
// Requirement: ownership guard only applies when bsEui is explicitly specified.
func TestSendULTransmit_NoBsEuiSpecified(t *testing.T) {
	// Setup: Create service where ownership check should be bypassed
	mockBsSvc := &mockBasestationSvc{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.BaseStation, error) {
			// This should NOT be called when bsEui is empty
			t.Fatal("GetByEUI should not be called when bsEui is not specified")
			return nil, nil
		},
	}

	mockSessionDir := &mockSessionDir{
		selectBidiFunc: func(_ int64, targetBsEui *uint64) (string, uint64, error) {
			// Verify targetBsEui is nil when not specified
			assert.Nil(t, targetBsEui, "targetBsEui should be nil when not specified in request")
			return "test-session", 0xBBBBBBBBBBBBBBBB, nil
		},
	}

	mockULTransmit := &mockULTransmit{
		sendFunc: func(_ string, _ uint64, _ []byte, _ uint16,
			_ uint32, _ []byte, _ string, _ uint8) (int64, error) {
			return -12345, nil // SC-initiated operation ID (negative)
		},
	}

	svc := &CoreService{
		basestationSvc: mockBsSvc,
		sessionDir:     mockSessionDir,
		ulTransmit:     mockULTransmit,
		log:            &mockLogger{},
	}

	// Execute: Request UL transmit WITHOUT specifying bsEui
	ctx := testutil.TestContextWithTenant(100)
	req := &pb.SendULTransmitRequest{
		EpEui:     "0000000000000001",
		BsEui:     "", // No base station specified - should skip ownership check
		NwkSnKey:  make([]byte, 16),
		ShAddr:    1,
		PacketCnt: 1,
		UserData:  []byte("test"),
		Profile:   "default",
		Format:    0,
	}

	resp, err := svc.SendULTransmit(ctx, req)

	// Verify: Success - ownership check was skipped, operation completed
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "-12345", resp.Id, "Should return operation ID from BSSCI layer")
	assert.Equal(t, grpcerrors.StatusQueued, resp.Status, "Should indicate operation is queued")
}

// TestSendULTransmit_SuccessfulRequest verifies the happy path where ownership
// check passes, session selection succeeds, and UL transmit is queued.
//
// Requirement: verify complete flow with valid ownership.
func TestSendULTransmit_SuccessfulRequest(t *testing.T) {
	// Setup: Create service with all mocks returning success
	mockBsSvc := &mockBasestationSvc{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.BaseStation, error) {
			// Ownership check passes
			return &models.BaseStation{
				EUI:      models.EUIFromString("AAAAAAAAAAAAAAAA"),
				Name:     "Test BS",
				TenantID: 100,
			}, nil
		},
	}

	mockSessionDir := &mockSessionDir{
		selectBidiFunc: func(_ int64, targetBsEui *uint64) (string, uint64, error) {
			// Session selection succeeds
			require.NotNil(t, targetBsEui, "targetBsEui should be set when bsEui is specified")
			assert.Equal(t, uint64(0xAAAAAAAAAAAAAAAA), *targetBsEui)
			return "test-session", 0xAAAAAAAAAAAAAAAA, nil
		},
	}

	mockULTransmit := &mockULTransmit{
		sendFunc: func(sessionID string, epEui uint64, _ []byte, shAddr uint16,
			_ uint32, _ []byte, _ string, _ uint8) (int64, error) {
			// UL transmit succeeds
			assert.Equal(t, "test-session", sessionID)
			assert.Equal(t, uint64(0x0000000000000001), epEui)
			assert.Equal(t, uint16(1), shAddr)
			return -12345, nil // SC-initiated operation ID
		},
	}

	svc := &CoreService{
		basestationSvc: mockBsSvc,
		sessionDir:     mockSessionDir,
		ulTransmit:     mockULTransmit,
		log:            &mockLogger{},
	}

	// Execute: Request UL transmit with valid parameters
	ctx := testutil.TestContextWithTenant(100)
	req := &pb.SendULTransmitRequest{
		EpEui:     "0000000000000001",
		BsEui:     "AAAAAAAAAAAAAAAA",
		NwkSnKey:  make([]byte, 16),
		ShAddr:    1,
		PacketCnt: 1,
		UserData:  []byte("test"),
		Profile:   "default",
		Format:    0,
	}

	resp, err := svc.SendULTransmit(ctx, req)

	// Verify: Success with proper operation ID
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "-12345", resp.Id, "Should return operation ID from BSSCI layer")
	assert.Equal(t, grpcerrors.StatusQueued, resp.Status, "Should indicate operation is queued")
	assert.Equal(t, grpcerrors.MsgULTransmitQueued, resp.Message)
}

// ============================================================================
// Test Cases for QueryDLRXStatus (DL RX Status Query)
// ============================================================================

// TestQueryDLRXStatus_Success verifies the happy path where endpoint is attached,
// session is ready, and the DL RX status query is sent successfully
func TestQueryDLRXStatus_Success(t *testing.T) {
	mockMsgSvc := &mockMessageSvc{
		getEndpointBSFunc: func(_ context.Context, epEui, tenantID string) (string, error) {
			assert.Equal(t, "0000000000000001", epEui)
			assert.Equal(t, "42", tenantID)
			return "1122334455667788", nil // Return BS EUI as hex string
		},
	}

	mockDir := &mockSessionDir{
		testSessions: make(map[string]*bssci.Session),
	}
	mockDir.AddTestSession(&bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "session-ready",
			BaseStationEUI:    0x1122334455667788,
			HandshakeComplete: true,
			ResolvedTenantID:  42,
			ClientVersion:     "1.0.0",
			NegotiatedVersion: "1.0.0",
		},
		Bidirectional: true,
		Connected:     time.Now(),
		LastSeen:      time.Now(),
	})

	mockCmd := &mockDownlinkCmd{
		sendDLRXStatusQueryFunc: func(sessionID string, epEui uint64) error {
			assert.Equal(t, "session-ready", sessionID)
			assert.Equal(t, uint64(0x0000000000000001), epEui)
			return nil
		},
	}

	svc := &CoreService{
		messageSvc:  mockMsgSvc,
		sessionDir:  mockDir,
		downlinkCmd: mockCmd,
		log:         &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(42)
	req := &pb.QueryDLRXStatusRequest{
		EpEui: "0000000000000001",
	}

	resp, err := svc.QueryDLRXStatus(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 1, mockCmd.callCount, "SendDLRXStatusQuery should be called once")
	assert.Equal(t, "session-ready", mockCmd.lastSessionID)
	assert.Equal(t, uint64(0x0000000000000001), mockCmd.lastEpEui)
}

// TestQueryDLRXStatus_TenantIsolation verifies that cross-tenant endpoint access
// is prevented and FindSessionForEndpointAttachment is never called
func TestQueryDLRXStatus_TenantIsolation(t *testing.T) {
	findSessionCalled := false

	mockDir := &mockSessionDir{
		testSessions: make(map[string]*bssci.Session),
		findSessionHook: func() {
			findSessionCalled = true
		},
	}

	mockMsgSvc := &mockMessageSvc{
		getEndpointBSFunc: func(_ context.Context, _, tenantID string) (string, error) {
			// Reject cross-tenant access
			if tenantID != "42" {
				return "", storage.ErrNotFound
			}
			return "1122334455667788", nil
		},
	}

	svc := &CoreService{
		messageSvc: mockMsgSvc,
		sessionDir: mockDir,
		log:        &mockLogger{},
	}

	// Tenant 99 tries to query tenant 42's endpoint
	ctx := testutil.TestContextWithTenant(99)
	req := &pb.QueryDLRXStatusRequest{
		EpEui: "0000000000000001",
	}

	resp, err := svc.QueryDLRXStatus(ctx, req)

	require.NoError(t, err, "QueryDLRXStatus should return success response, not error")
	require.NotNil(t, resp)
	assert.False(t, resp.QueryInitiated, "Query should not be initiated for endpoint not found")
	assert.Contains(t, resp.Message, "not currently attached", "Message should indicate endpoint not attached")

	// CRITICAL: Verify FindSessionForEndpointAttachment was NEVER called
	assert.False(t, findSessionCalled,
		"FindSessionForEndpointAttachment must not be called when endpoint ownership check fails")
}

// TestQueryDLRXStatus_SessionNotFound verifies error handling when base station
// session is not found (endpoint attached but BS offline)
func TestQueryDLRXStatus_SessionNotFound(t *testing.T) {
	mockMsgSvc := &mockMessageSvc{
		getEndpointBSFunc: func(_ context.Context, _, _ string) (string, error) {
			return "AAAAAAAAAAAAAAAA", nil // Endpoint attached to this BS
		},
	}

	mockDir := &mockSessionDir{
		testSessions: make(map[string]*bssci.Session),
		// No sessions registered - BS is offline
	}

	svc := &CoreService{
		messageSvc: mockMsgSvc,
		sessionDir: mockDir,
		log:        &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(42)
	req := &pb.QueryDLRXStatusRequest{
		EpEui: "0000000000000001",
	}

	resp, err := svc.QueryDLRXStatus(ctx, req)

	require.NoError(t, err, "QueryDLRXStatus should return success response, not error")
	require.NotNil(t, resp)
	assert.False(t, resp.QueryInitiated, "Query should not be initiated when BS session not found")
	assert.Contains(t, resp.Message, "not currently connected", "Message should indicate BS not connected")
}

// TestQueryDLRXStatus_SessionNotReady verifies error handling when base station
// session exists but handshake hasn't completed (BSSCI §3.3)
func TestQueryDLRXStatus_SessionNotReady(t *testing.T) {
	mockMsgSvc := &mockMessageSvc{
		getEndpointBSFunc: func(_ context.Context, _, _ string) (string, error) {
			return "1122334455667788", nil
		},
	}

	mockDir := &mockSessionDir{
		testSessions: make(map[string]*bssci.Session),
	}
	mockDir.AddTestSession(&bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "incomplete-handshake",
			BaseStationEUI:    0x1122334455667788,
			HandshakeComplete: false,
		},
		Bidirectional: true,
		// Handshake NOT complete
		Connected: time.Now(),
		LastSeen:  time.Now(),
	})

	svc := &CoreService{
		messageSvc: mockMsgSvc,
		sessionDir: mockDir,
		log:        &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(42)
	req := &pb.QueryDLRXStatusRequest{
		EpEui: "0000000000000001",
	}

	resp, err := svc.QueryDLRXStatus(ctx, req)

	require.NoError(t, err, "QueryDLRXStatus should return success response, not error")
	require.NotNil(t, resp)
	assert.False(t, resp.QueryInitiated, "Query should not be initiated when handshake incomplete")
	assert.Contains(t, resp.Message, "handshake not complete", "Message should indicate handshake incomplete")
}

// TestQueryDLRXStatus_SessionNotBidirectional verifies error handling when
// base station doesn't support bidirectional operations
func TestQueryDLRXStatus_SessionNotBidirectional(t *testing.T) {
	mockMsgSvc := &mockMessageSvc{
		getEndpointBSFunc: func(_ context.Context, _, _ string) (string, error) {
			return "1122334455667788", nil
		},
	}

	mockDir := &mockSessionDir{
		testSessions: make(map[string]*bssci.Session),
	}
	mockDir.AddTestSession(&bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "unidirectional",
			BaseStationEUI:    0x1122334455667788,
			HandshakeComplete: true,
		},
		// NOT bidirectional
		Bidirectional: false,
		Connected:     time.Now(),
		LastSeen:      time.Now(),
	})

	svc := &CoreService{
		messageSvc: mockMsgSvc,
		sessionDir: mockDir,
		log:        &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(42)
	req := &pb.QueryDLRXStatusRequest{
		EpEui: "0000000000000001",
	}

	resp, err := svc.QueryDLRXStatus(ctx, req)

	require.NoError(t, err, "QueryDLRXStatus should return success response, not error")
	require.NotNil(t, resp)
	assert.False(t, resp.QueryInitiated, "Query should not be initiated when BS not bidirectional")
	assert.Contains(t, resp.Message, "does not support bidirectional", "Message should indicate BS not bidirectional")
}

// TestGetReleaseInfo verifies that GetReleaseInfo returns embedded release manifest data.
func TestGetReleaseInfo(t *testing.T) {
	svc := &CoreService{
		log:     &mockLogger{},
		edition: "ce",
	}

	ctx := testutil.TestContext()
	resp, err := svc.GetReleaseInfo(ctx, nil)

	require.NoError(t, err, "GetReleaseInfo should succeed")
	require.NotNil(t, resp, "Response should not be nil")

	// Verify required fields are populated from embedded manifest
	assert.NotEmpty(t, resp.Version, "Version should be populated")
	assert.NotEmpty(t, resp.BuildTime, "BuildTime should be populated")
	assert.NotEmpty(t, resp.GitCommit, "GitCommit should be populated")
	assert.NotEmpty(t, resp.GitBranch, "GitBranch should be populated")
	assert.NotEmpty(t, resp.GoVersion, "GoVersion should be populated")
	assert.Greater(t, resp.SchemaVersion, int32(0), "SchemaVersion should be > 0")

	assert.Equal(t, "Community Edition", resp.Edition, "Edition should be the canonical label for the runtime edition code")
	assert.Equal(t, "ce", resp.EditionCode, "EditionCode should match runtime edition")
	assert.NotEmpty(t, resp.LicenseId, "LicenseId should be populated")
	assert.NotEmpty(t, resp.LicenseUrl, "LicenseUrl should be populated")
	assert.NotEmpty(t, resp.SourceUrl, "SourceUrl should be populated")
	assert.NotEmpty(t, resp.DocumentationUrl, "DocumentationUrl should be populated")
	assert.NotEmpty(t, resp.HomepageUrl, "HomepageUrl should be populated")
	assert.NotEmpty(t, resp.TrademarkNotice, "TrademarkNotice should be populated")

	// Log the actual values for manual inspection
	t.Logf("Release Info:")
	t.Logf("  Version: %s", resp.Version)
	t.Logf("  BuildTime: %s", resp.BuildTime)
	t.Logf("  GitCommit: %s", resp.GitCommit)
	t.Logf("  GitBranch: %s", resp.GitBranch)
	t.Logf("  GoVersion: %s", resp.GoVersion)
	t.Logf("  SchemaVersion: %d", resp.SchemaVersion)
}

// ============================================================================
// Test Cases for CreateEndPoint Global EUI Uniqueness (Migration 000080)
// ============================================================================

// mockEndpointSvcForDuplicate implements grpcservices.EndpointService for duplicate testing
type mockEndpointSvcForDuplicate struct {
	createCallCount int
	returnDuplicate bool
}

func (m *mockEndpointSvcForDuplicate) Create(_ context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	m.createCallCount++
	if m.createCallCount == 1 && !m.returnDuplicate {
		// First call succeeds
		ep.ID = 123
		return ep, nil
	}
	// Subsequent calls return duplicate error (global uniqueness)
	return nil, kcerrors.ErrDuplicate
}

func (m *mockEndpointSvcForDuplicate) GetByEUI(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
	return nil, nil
}

func (m *mockEndpointSvcForDuplicate) Update(_ context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}

func (m *mockEndpointSvcForDuplicate) Delete(_ context.Context, _ []byte, _ int64) error {
	return nil
}

func (m *mockEndpointSvcForDuplicate) ListByModelWithSnapshot(_ context.Context, _ int64, _ uuid.UUID) ([]*models.EndPoint, error) {
	return nil, nil
}

func (m *mockEndpointSvcForDuplicate) List(_ context.Context, _ int64, _, _ int) ([]*models.EndPoint, error) {
	return nil, nil
}

func (m *mockEndpointSvcForDuplicate) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}

func (m *mockEndpointSvcForDuplicate) CheckEUIGloballyUnique(_ context.Context, _ []byte) error {
	return nil
}

// TestGRPCCreateEndpoint_GlobalUniqueness verifies global EUI uniqueness returns AlreadyExists
// for duplicate endpoint registration across tenants.
//
// This test validates migration 000080 global EUI uniqueness constraint at the gRPC layer.
// Flow: Tenant 1 succeeds, Tenant 2 same EUI fails with AlreadyExists
func TestGRPCCreateEndpoint_GlobalUniqueness(t *testing.T) {
	// Create mock service
	mockSvc := &mockEndpointSvcForDuplicate{}

	// Create gRPC service with mock
	service := &CoreService{
		endpointSvc: mockSvc,
		log:         &mockLogger{},
	}

	// Set up tenant/org contexts using pkgcontext
	org1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	org2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	ctx1 := pkgcontext.WithTenantID(testutil.TestContext(), int64(1))
	ctx1 = pkgcontext.WithOrganizationID(ctx1, org1)

	ctx2 := pkgcontext.WithTenantID(testutil.TestContext(), int64(2))
	ctx2 = pkgcontext.WithOrganizationID(ctx2, org2)

	// Create request with test EUI
	req := &pb.CreateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:    "1122334455667788",
			Name:     "gRPC Test EP",
			NwkSnKey: make([]byte, 16), // Required field
		},
	}

	// =========================================================================
	// Test 1: First request with tenant 1 should succeed
	// =========================================================================
	resp, err := service.CreateEndPoint(ctx1, req)
	require.NoError(t, err, "First request should succeed")
	require.NotNil(t, resp, "Response should not be nil")

	// =========================================================================
	// Test 2: Second request with tenant 2 should fail with AlreadyExists
	// (global uniqueness via migration 000080)
	// =========================================================================
	_, err = service.CreateEndPoint(ctx2, req)
	require.Error(t, err, "Second request with different tenant should fail")

	// Verify error is AlreadyExists status
	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	require.Equal(t, codes.AlreadyExists, st.Code(), "Error code should be AlreadyExists")
	require.Contains(t, st.Message(), "already exists", "Error message should indicate duplicate")

	// Verify both calls were made to the service
	require.Equal(t, 2, mockSvc.createCallCount, "Create should have been called 2 times")
}

// TestGRPCCreateEndpoint_SameTenantDuplicate verifies duplicate within same tenant returns AlreadyExists
func TestGRPCCreateEndpoint_SameTenantDuplicate(t *testing.T) {
	mockSvc := &mockEndpointSvcForDuplicate{}
	service := &CoreService{
		endpointSvc: mockSvc,
		log:         &mockLogger{},
	}

	ctx := pkgcontext.WithTenantID(testutil.TestContext(), int64(1))
	ctx = pkgcontext.WithOrganizationID(ctx, uuid.New())

	req := &pb.CreateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:    "AABBCCDDEEFF0011",
			Name:     "Test Endpoint",
			NwkSnKey: make([]byte, 16),
		},
	}

	// First request succeeds
	resp, err := service.CreateEndPoint(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Second request with same tenant fails
	_, err = service.CreateEndPoint(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.AlreadyExists, st.Code())
}

// TestGRPCCreateEndpoint_MissingEndpoint verifies nil endpoint returns InvalidArgument
func TestGRPCCreateEndpoint_MissingEndpoint(t *testing.T) {
	mockSvc := &mockEndpointSvcForDuplicate{}
	service := &CoreService{
		endpointSvc: mockSvc,
		log:         &mockLogger{},
	}

	ctx := pkgcontext.WithTenantID(testutil.TestContext(), int64(1))
	ctx = pkgcontext.WithOrganizationID(ctx, uuid.New())

	req := &pb.CreateEndPointRequest{Endpoint: nil}

	_, err := service.CreateEndPoint(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
	require.Contains(t, st.Message(), "endpoint is required")
}

// TestGRPCCreateEndpoint_MissingEUI verifies empty EUI returns InvalidArgument
func TestGRPCCreateEndpoint_MissingEUI(t *testing.T) {
	mockSvc := &mockEndpointSvcForDuplicate{}
	service := &CoreService{
		endpointSvc: mockSvc,
		log:         &mockLogger{},
	}

	ctx := pkgcontext.WithTenantID(testutil.TestContext(), int64(1))
	ctx = pkgcontext.WithOrganizationID(ctx, uuid.New())

	req := &pb.CreateEndPointRequest{
		Endpoint: &pb.EndPoint{EpEui: "", Name: "Test"},
	}

	_, err := service.CreateEndPoint(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
	require.Contains(t, st.Message(), "eui is required")
}

// TestGRPCCreateEndpoint_InvalidNwkSnKeyLength verifies invalid NwkSnKey length returns InvalidArgument
func TestGRPCCreateEndpoint_InvalidNwkSnKeyLength(t *testing.T) {
	mockSvc := &mockEndpointSvcForDuplicate{}
	service := &CoreService{
		endpointSvc: mockSvc,
		log:         &mockLogger{},
	}

	ctx := pkgcontext.WithTenantID(testutil.TestContext(), int64(1))
	ctx = pkgcontext.WithOrganizationID(ctx, uuid.New())

	req := &pb.CreateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:    "1122334455667788",
			Name:     "Test",
			NwkSnKey: make([]byte, 15),
		},
	}

	_, err := service.CreateEndPoint(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
	require.Contains(t, st.Message(), "nwkSnKey must be exactly 16 bytes")
}

// TestGRPCCreateEndpoint_InvalidAppKeyLength verifies invalid AppKey length returns InvalidArgument
func TestGRPCCreateEndpoint_InvalidAppKeyLength(t *testing.T) {
	mockSvc := &mockEndpointSvcForDuplicate{}
	service := &CoreService{
		endpointSvc: mockSvc,
		log:         &mockLogger{},
	}

	ctx := pkgcontext.WithTenantID(testutil.TestContext(), int64(1))
	ctx = pkgcontext.WithOrganizationID(ctx, uuid.New())

	req := &pb.CreateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:    "1122334455667788",
			Name:     "Test",
			NwkSnKey: make([]byte, 16),
			AppKey:   make([]byte, 15),
		},
	}

	_, err := service.CreateEndPoint(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
	require.Contains(t, st.Message(), "appKey must be exactly 16 bytes")
}

// TestGRPCCreateEndpoint_MissingTenant verifies missing tenant context returns Unauthenticated
func TestGRPCCreateEndpoint_MissingTenant(t *testing.T) {
	mockSvc := &mockEndpointSvcForDuplicate{}
	service := &CoreService{
		endpointSvc: mockSvc,
		log:         &mockLogger{},
	}

	// Intentionally bare context — tests unauthenticated failure path
	ctx := testutil.TestContext()

	req := &pb.CreateEndPointRequest{
		Endpoint: &pb.EndPoint{EpEui: "0x1122334455667788", Name: "Test", NwkSnKey: make([]byte, 16)},
	}

	_, err := service.CreateEndPoint(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unauthenticated, st.Code())
}

func TestBuildMessageFilters_InvalidEndpointEUI(t *testing.T) {
	_, err := buildMessageFilters("invalid-eui", "", nil, nil)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointEUIFormat), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointEUIFormat), st.Message())
}

func TestParseEUI_InvalidLength(t *testing.T) {
	_, err := parseEUI("010203")
	require.Error(t, err)
}

// ============================================================================
// AttachEndPoint / DetachEndPoint Tests
// ============================================================================

// mockEndpointAttachmentSvc implements grpcservices.EndpointAttachmentService for testing
type mockEndpointAttachmentSvc struct {
	attachFunc func(ctx context.Context, epEui string, tenantID int64) (*grpcservices.EndpointOperationResult, error)
	detachFunc func(ctx context.Context, epEui string, tenantID int64) (*grpcservices.EndpointOperationResult, error)
}

func (m *mockEndpointAttachmentSvc) AttachEndPoint(ctx context.Context, epEui string, tenantID int64) (*grpcservices.EndpointOperationResult, error) {
	if m.attachFunc != nil {
		return m.attachFunc(ctx, epEui, tenantID)
	}
	return nil, nil
}

func (m *mockEndpointAttachmentSvc) DetachEndPoint(ctx context.Context, epEui string, tenantID int64) (*grpcservices.EndpointOperationResult, error) {
	if m.detachFunc != nil {
		return m.detachFunc(ctx, epEui, tenantID)
	}
	return nil, nil
}

// TestAttachEndPoint_MissingEpEui verifies missing ep_eui returns ErrTokenEndpointEUIRequired
func TestAttachEndPoint_MissingEpEui(t *testing.T) {
	svc := &CoreService{
		endpointAttachmentSvc: &mockEndpointAttachmentSvc{},
		log:                   &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(42)
	req := &pb.AttachEndPointRequest{
		EpEui: "", // Empty EUI
	}

	resp, err := svc.AttachEndPoint(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired), st.Message())
}

// TestAttachEndPoint_ServiceNotConfigured verifies nil service returns ErrTokenServiceNotConfigured
func TestAttachEndPoint_ServiceNotConfigured(t *testing.T) {
	svc := &CoreService{
		endpointAttachmentSvc: nil, // Service not configured
		log:                   &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(42)
	req := &pb.AttachEndPointRequest{
		EpEui: "0000000000000001",
	}

	resp, err := svc.AttachEndPoint(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured), st.Message())
}

// TestAttachEndPoint_EndpointNotFound verifies storage.ErrNotFound returns ErrTokenEndpointNotFound
func TestAttachEndPoint_EndpointNotFound(t *testing.T) {
	svc := &CoreService{
		endpointAttachmentSvc: &mockEndpointAttachmentSvc{
			attachFunc: func(_ context.Context, _ string, _ int64) (*grpcservices.EndpointOperationResult, error) {
				return nil, storage.ErrNotFound
			},
		},
		log: &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(42)
	req := &pb.AttachEndPointRequest{
		EpEui: "0000000000000001",
	}

	resp, err := svc.AttachEndPoint(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound), st.Message())
}

// TestAttachEndPoint_Success verifies successful attach returns operation_id and status
func TestAttachEndPoint_Success(t *testing.T) {
	svc := &CoreService{
		endpointAttachmentSvc: &mockEndpointAttachmentSvc{
			attachFunc: func(_ context.Context, epEui string, tenantID int64) (*grpcservices.EndpointOperationResult, error) {
				assert.Equal(t, "0000000000000001", epEui)
				assert.Equal(t, int64(42), tenantID)
				return &grpcservices.EndpointOperationResult{
					OperationID: "attach-123-456",
					Status:      "initiated",
				}, nil
			},
		},
		log: &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(42)
	req := &pb.AttachEndPointRequest{
		EpEui: "0000000000000001",
	}

	resp, err := svc.AttachEndPoint(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "attach-123-456", resp.OperationId)
	assert.Equal(t, "initiated", resp.Status)
}

// TestDetachEndPoint_MissingEpEui verifies missing ep_eui returns ErrTokenEndpointEUIRequired
func TestDetachEndPoint_MissingEpEui(t *testing.T) {
	svc := &CoreService{
		endpointAttachmentSvc: &mockEndpointAttachmentSvc{},
		log:                   &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(42)
	req := &pb.DetachEndPointRequest{
		EpEui: "", // Empty EUI
	}

	resp, err := svc.DetachEndPoint(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointEUIRequired), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointEUIRequired), st.Message())
}

// TestDetachEndPoint_ServiceNotConfigured verifies nil service returns ErrTokenServiceNotConfigured
func TestDetachEndPoint_ServiceNotConfigured(t *testing.T) {
	svc := &CoreService{
		endpointAttachmentSvc: nil, // Service not configured
		log:                   &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(42)
	req := &pb.DetachEndPointRequest{
		EpEui: "0000000000000001",
	}

	resp, err := svc.DetachEndPoint(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured), st.Message())
}

// TestDetachEndPoint_EndpointNotFound verifies storage.ErrNotFound returns ErrTokenEndpointNotFound
func TestDetachEndPoint_EndpointNotFound(t *testing.T) {
	svc := &CoreService{
		endpointAttachmentSvc: &mockEndpointAttachmentSvc{
			detachFunc: func(_ context.Context, _ string, _ int64) (*grpcservices.EndpointOperationResult, error) {
				return nil, storage.ErrNotFound
			},
		},
		log: &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(42)
	req := &pb.DetachEndPointRequest{
		EpEui: "0000000000000001",
	}

	resp, err := svc.DetachEndPoint(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEndpointNotFound), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointNotFound), st.Message())
}

// TestDetachEndPoint_Success verifies successful detach returns operation_id and status
func TestDetachEndPoint_Success(t *testing.T) {
	svc := &CoreService{
		endpointAttachmentSvc: &mockEndpointAttachmentSvc{
			detachFunc: func(_ context.Context, epEui string, tenantID int64) (*grpcservices.EndpointOperationResult, error) {
				assert.Equal(t, "0000000000000001", epEui)
				assert.Equal(t, int64(42), tenantID)
				return &grpcservices.EndpointOperationResult{
					OperationID: "detach-789-012",
					Status:      "initiated",
				}, nil
			},
		},
		log: &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(42)
	req := &pb.DetachEndPointRequest{
		EpEui: "0000000000000001",
	}

	resp, err := svc.DetachEndPoint(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "detach-789-012", resp.OperationId)
	assert.Equal(t, "initiated", resp.Status)
}

// ============================================================================
// Proto Mapping Tests
// ============================================================================

// TestEndpointToProto_MapsAllFields verifies endpointToProto maps all MIOTY configuration
// fields from models.EndPoint to proto EndPoint per BSSCI v1.0.0 §3.8.1
func TestEndpointToProto_MapsAllFields(t *testing.T) {
	svc := &CoreService{log: &mockLogger{}}

	// models.EndPoint uses EUI type for identifiers and []byte for keys
	shAddr := uint16(1234)
	attachCnt := uint32(42)
	typeEui := models.EUIFromString("1122334455667788")
	endpoint := &models.EndPoint{
		EUI:      models.EUIFromString("0123456789abcdef"),
		TenantID: 123,
		Name:     "Test Endpoint",
		EPClass:  "A",
		NwkSnKey: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		AppKey:   []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01},
		// MIOTY configuration fields
		EpStatus:      "attached",
		ShAddr:        &shAddr,
		DualChan:      true,
		Repetition:    true,
		WideCarrOff:   false,
		LongBlkDist:   true,
		AttachCnt:     &attachCnt,
		PreAttach:     true,
		TypeEUI:       &typeEui,
		CarrierOffset: 1200,
		LastPacketCnt: 77,
	}

	pbEndpoint := svc.endpointToProto(endpoint)

	require.NotNil(t, pbEndpoint)
	assert.Equal(t, "0123456789abcdef", pbEndpoint.EpEui)
	assert.Equal(t, "Test Endpoint", pbEndpoint.Name)
	// Keys are passed through as bytes
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, pbEndpoint.NwkSnKey)
	assert.Equal(t, []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01}, pbEndpoint.AppKey)
	// Assert new MIOTY fields
	// AttachStatus and Status both map from models.EndPoint.EpStatus for frontend compatibility
	assert.Equal(t, uint32(1234), pbEndpoint.ShAddr)
	assert.True(t, pbEndpoint.DualChan)
	assert.True(t, pbEndpoint.Repetition)
	assert.False(t, pbEndpoint.WideCarrOff)
	assert.True(t, pbEndpoint.LongBlkDist)
	assert.Equal(t, uint32(42), pbEndpoint.AttachCnt)
	assert.True(t, pbEndpoint.PreAttach)
	assert.Equal(t, uint32(77), pbEndpoint.LastPacketCnt)
	assert.Equal(t, []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}, pbEndpoint.TypeEui)
	assert.Equal(t, int32(1200), pbEndpoint.CarrierOffset)
}

// TestEndpointToProto_EchoesBlueprintID: blueprint_id on the wire echoes the snapshot source_blueprint_id (empty when snapshot missing/invalid).
func TestEndpointToProto_EchoesBlueprintID(t *testing.T) {
	const sourceID = "3f2504e0-4f89-41d3-9a0c-0305e82c3301"
	tests := []struct {
		name     string
		snapshot []byte
		want     string
	}{
		{name: "snapshot with source id echoes it", snapshot: []byte(`{"source_blueprint_id":"` + sourceID + `","is_system":true}`), want: sourceID},
		{name: "nil snapshot echoes empty", snapshot: nil, want: ""},
		{name: "snapshot without source id echoes empty", snapshot: []byte(`{"is_system":false}`), want: ""},
		{name: "malformed snapshot echoes empty", snapshot: []byte(`{not-json`), want: ""},
	}
	svc := &CoreService{log: &mockLogger{}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &models.EndPoint{
				EUI:               models.EUIFromString("0123456789abcdef"),
				TenantID:          123,
				BlueprintSnapshot: tt.snapshot,
			}
			assert.Equal(t, tt.want, svc.endpointToProto(endpoint).BlueprintId)
		})
	}
}

// TestApplyBlueprintSnapshot_Materializes: selected blueprint is copied into the snapshot; device_model + type_eui follow the blueprint.
func TestApplyBlueprintSnapshot_Materializes(t *testing.T) {
	bpModelID := uuid.New()
	typeEUIBytes := []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0x00, 0x00, 0x94}
	bp := &models.Blueprint{
		ID:            uuid.New(),
		DeviceModelID: bpModelID,
		Version:       "1.2.3",
		TypeEUI:       typeEUIBytes,
		SpecJSON:      []byte(`{"codec":"x"}`),
	}
	otherModel := uuid.New()
	endpoint := &models.EndPoint{EUI: models.EUIFromString("0123456789abcdef"), DeviceModelID: &otherModel}

	svc := &CoreService{log: &mockLogger{}, blueprintSvc: &mockBlueprintSvcForEndpoint{
		getBlueprintFn: func(_ uuid.UUID) (*models.Blueprint, error) { return bp, nil },
	}}

	err := svc.applyBlueprintSnapshot(testutil.TestContext(), endpoint, bp.ID.String())
	require.NoError(t, err)
	require.NotEmpty(t, endpoint.BlueprintSnapshot)
	require.NotNil(t, endpoint.DeviceModelID)
	assert.Equal(t, bpModelID, *endpoint.DeviceModelID, "device model follows the blueprint")
	require.NotNil(t, endpoint.TypeEUI)
	assert.Equal(t, typeEUIBytes, endpoint.TypeEUI[:], "type_eui follows the blueprint")
	assert.Equal(t, bp.ID.String(), snapshotSourceID(endpoint.BlueprintSnapshot), "snapshot records source blueprint id")
}

// TestApplyBlueprintSnapshot_Errors covers the failure branches; none must leave a partial snapshot.
func TestApplyBlueprintSnapshot_Errors(t *testing.T) {
	validID := uuid.NewString()
	tests := []struct {
		name        string
		svc         grpcservices.BlueprintService
		blueprintID string
		wantCode    codes.Code
	}{
		{"service not configured", nil, validID, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured)},
		{"invalid uuid", &mockBlueprintSvcForEndpoint{}, "not-a-uuid", grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBlueprintIDFormat)},
		{"blueprint not found", &mockBlueprintSvcForEndpoint{getBlueprintFn: func(_ uuid.UUID) (*models.Blueprint, error) { return nil, nil }}, validID, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNotFound)},
		{"fetch error", &mockBlueprintSvcForEndpoint{getBlueprintFn: func(_ uuid.UUID) (*models.Blueprint, error) { return nil, errors.New("db down") }}, validID, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintNotFound)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &CoreService{log: &mockLogger{}}
			if tt.svc != nil {
				svc.blueprintSvc = tt.svc
			}
			endpoint := &models.EndPoint{EUI: models.EUIFromString("0123456789abcdef")}
			err := svc.applyBlueprintSnapshot(testutil.TestContext(), endpoint, tt.blueprintID)
			require.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.Empty(t, endpoint.BlueprintSnapshot, "no snapshot on error")
		})
	}
}

// TestUpdateEndPoint_SnapshotTrigger: three-branch re-materialization state machine whose failure modes are silent.
func TestUpdateEndPoint_SnapshotTrigger(t *testing.T) {
	priorModel, newModel := uuid.New(), uuid.New()
	priorBp, newBp, newDefBp := uuid.New(), uuid.New(), uuid.New()
	tePrior := []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0x00, 0x00, 0x01}
	teNew := []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0x00, 0x00, 0x02}
	teDef := []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0x00, 0x00, 0x03}
	const eui = "1122334455667788"

	priorSnap, err := buildBlueprintSnapshot(&models.Blueprint{ID: priorBp, DeviceModelID: priorModel, TypeEUI: tePrior, Version: "1", SpecJSON: []byte(`{"a":1}`)})
	require.NoError(t, err)
	existing := func() *models.EndPoint {
		m := priorModel
		return &models.EndPoint{EUI: models.EUIFromString(eui), TenantID: 1, DeviceModelID: &m, BlueprintSnapshot: priorSnap}
	}
	run := func(ep *mockEndpointSvcForUpdate, bp *mockBlueprintSvcForEndpoint, req *pb.UpdateEndPointRequest) error {
		svc := &CoreService{endpointSvc: ep, blueprintSvc: bp, log: &mockLogger{}}
		_, e := svc.UpdateEndPoint(testutil.TestContextWithTenant(1), req)
		return e
	}
	selReq := &pb.UpdateEndPointRequest{Endpoint: &pb.EndPoint{EpEui: eui, BlueprintId: newBp.String()}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{fieldMaskBlueprintID}}}
	modelReq := &pb.UpdateEndPointRequest{Endpoint: &pb.EndPoint{EpEui: eui, DeviceModelId: newModel.String()}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{fieldMaskDeviceModelID}}}

	t.Run("branch1 explicit selection re-materializes", func(t *testing.T) {
		ep := &mockEndpointSvcForUpdate{getByEUIResult: existing()}
		bp := &mockBlueprintSvcForEndpoint{getBlueprintFn: func(_ uuid.UUID) (*models.Blueprint, error) {
			return &models.Blueprint{ID: newBp, DeviceModelID: newModel, TypeEUI: teNew, Version: "2", SpecJSON: []byte(`{"b":2}`)}, nil
		}}
		require.NoError(t, run(ep, bp, selReq))
		got := ep.updateCapture
		require.NotNil(t, got)
		assert.Equal(t, newBp.String(), snapshotSourceID(got.BlueprintSnapshot), "snapshot re-forked to selected blueprint")
		require.NotNil(t, got.DeviceModelID)
		assert.Equal(t, newModel, *got.DeviceModelID, "device model follows selected blueprint")
		require.NotNil(t, got.TypeEUI)
		assert.Equal(t, teNew, got.TypeEUI[:], "type_eui follows the snapshot")
	})

	t.Run("branch1 snapshot type_eui overrides explicit mask write", func(t *testing.T) {
		ep := &mockEndpointSvcForUpdate{getByEUIResult: existing()}
		bp := &mockBlueprintSvcForEndpoint{getBlueprintFn: func(_ uuid.UUID) (*models.Blueprint, error) {
			return &models.Blueprint{ID: newBp, DeviceModelID: newModel, TypeEUI: teNew, Version: "2", SpecJSON: []byte(`{}`)}, nil
		}}
		req := &pb.UpdateEndPointRequest{
			Endpoint:   &pb.EndPoint{EpEui: eui, BlueprintId: newBp.String(), TypeEui: []byte{0xde, 0xad, 0xbe, 0xef, 0x00, 0x00, 0x00, 0x00}},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{fieldMaskBlueprintID, fieldMaskTypeEUI}},
		}
		require.NoError(t, run(ep, bp, req))
		require.NotNil(t, ep.updateCapture.TypeEUI)
		assert.Equal(t, teNew, ep.updateCapture.TypeEUI[:], "snapshot type_eui wins over explicit mask write")
	})

	t.Run("branch1 same blueprint id is a no-op", func(t *testing.T) {
		ep := &mockEndpointSvcForUpdate{getByEUIResult: existing()}
		bp := &mockBlueprintSvcForEndpoint{getBlueprintFn: func(_ uuid.UUID) (*models.Blueprint, error) { return nil, errors.New("must not fetch") }}
		req := &pb.UpdateEndPointRequest{Endpoint: &pb.EndPoint{EpEui: eui, BlueprintId: priorBp.String()}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{fieldMaskBlueprintID}}}
		require.NoError(t, run(ep, bp, req))
		assert.Equal(t, priorBp.String(), snapshotSourceID(ep.updateCapture.BlueprintSnapshot), "snapshot unchanged on same-id selection")
	})

	t.Run("branch2 model change re-seeds from new model default", func(t *testing.T) {
		ep := &mockEndpointSvcForUpdate{getByEUIResult: existing()}
		bp := &mockBlueprintSvcForEndpoint{
			getDeviceModelResult: &models.DeviceModel{ID: newModel},
			getDefaultForModelFn: func() (*models.Blueprint, error) {
				return &models.Blueprint{ID: newDefBp, DeviceModelID: newModel, TypeEUI: teDef, Version: "9", SpecJSON: []byte(`{}`)}, nil
			},
			getBlueprintFn: func(id uuid.UUID) (*models.Blueprint, error) {
				if id == priorBp { // pinned blueprint is native to the OLD model → re-seed
					return &models.Blueprint{ID: priorBp, DeviceModelID: priorModel}, nil
				}
				return &models.Blueprint{ID: newDefBp, DeviceModelID: newModel, TypeEUI: teDef, Version: "9", SpecJSON: []byte(`{}`)}, nil
			},
		}
		require.NoError(t, run(ep, bp, modelReq))
		got := ep.updateCapture
		assert.Equal(t, newDefBp.String(), snapshotSourceID(got.BlueprintSnapshot), "re-seeded to new model default")
		require.NotNil(t, got.TypeEUI)
		assert.Equal(t, teDef, got.TypeEUI[:], "type_eui follows re-seeded snapshot")
	})

	t.Run("branch2 new model without default keeps prior snapshot", func(t *testing.T) {
		ep := &mockEndpointSvcForUpdate{getByEUIResult: existing()}
		bp := &mockBlueprintSvcForEndpoint{
			getDeviceModelResult: &models.DeviceModel{ID: newModel},
			getDefaultForModelFn: func() (*models.Blueprint, error) { return nil, nil },
			getBlueprintFn: func(_ uuid.UUID) (*models.Blueprint, error) {
				return &models.Blueprint{ID: priorBp, DeviceModelID: priorModel}, nil
			},
		}
		require.NoError(t, run(ep, bp, modelReq))
		assert.Equal(t, priorBp.String(), snapshotSourceID(ep.updateCapture.BlueprintSnapshot), "prior snapshot kept when new model has no default")
	})

	t.Run("branch2 default lookup error fails the update", func(t *testing.T) {
		ep := &mockEndpointSvcForUpdate{getByEUIResult: existing()}
		bp := &mockBlueprintSvcForEndpoint{
			getDeviceModelResult: &models.DeviceModel{ID: newModel},
			getDefaultForModelFn: func() (*models.Blueprint, error) { return nil, errors.New("db down") },
			getBlueprintFn: func(_ uuid.UUID) (*models.Blueprint, error) {
				return &models.Blueprint{ID: priorBp, DeviceModelID: priorModel}, nil
			},
		}
		require.Error(t, run(ep, bp, modelReq))
		assert.Nil(t, ep.updateCapture, "update must not persist on infra failure")
	})

	t.Run("branch2 skipped when pinned blueprint is native to new model", func(t *testing.T) {
		ep := &mockEndpointSvcForUpdate{getByEUIResult: existing()}
		bp := &mockBlueprintSvcForEndpoint{
			getDeviceModelResult: &models.DeviceModel{ID: newModel},
			getDefaultForModelFn: func() (*models.Blueprint, error) { return nil, errors.New("must not be called") },
			getBlueprintFn: func(_ uuid.UUID) (*models.Blueprint, error) {
				return &models.Blueprint{ID: priorBp, DeviceModelID: newModel}, nil
			},
		}
		require.NoError(t, run(ep, bp, modelReq))
		assert.Equal(t, priorBp.String(), snapshotSourceID(ep.updateCapture.BlueprintSnapshot), "snapshot preserved; no re-seed when pinned is native to new model")
	})
}

// TestULDataMessageToBaseStationMessageProto_SetsDirection verifies ulDataMessageToBaseStationMessageProto
// sets direction to "uplink" for all ULDataMessage conversions per MIOTY protocol semantics
func TestULDataMessageToBaseStationMessageProto_SetsDirection(t *testing.T) {
	msg := &mioty.ULDataMessage{
		ID:        "msg-123",
		BsEui:     0x0123456789abcdef,
		EpEui:     0xfedcba9876543210,
		UserData:  []byte{0x01, 0x02},
		RSSI:      -50.5,
		SNR:       10.2,
		PacketCnt: 100,
	}

	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, msg.BsEui)
	pbMsg := ulDataMessageToBaseStationMessageProto(msg, bsEuiBytes)

	require.NotNil(t, pbMsg)
	assert.Equal(t, mioty.DirectionUplink, pbMsg.Direction, "ULDataMessage must have uplink direction")
	assert.Equal(t, "0123456789abcdef", pbMsg.BsEui)
	assert.Equal(t, "fedcba9876543210", pbMsg.EpEui)
	assert.Equal(t, []byte{0x01, 0x02}, pbMsg.Payload)
	assert.Equal(t, -50.5, pbMsg.Rssi)
	assert.Equal(t, 10.2, pbMsg.Snr)
	assert.Equal(t, uint32(100), pbMsg.PacketCounter)
}

// TestULDataMessageToBaseStationMessageProto_BS2Projection verifies that when the mapper
// targets BS2, it projects BS2-specific signal values and does not leak primary row EqSnr
func TestULDataMessageToBaseStationMessageProto_BS2Projection(t *testing.T) {
	primaryEqSnr := 22.5
	bs1EqSnr := 20.0
	msg := &mioty.ULDataMessage{
		ID:        "msg-bs2-proj",
		BsEui:     0x0000000000000001, // Primary row = BS1
		EpEui:     0xfedcba9876543210,
		UserData:  []byte{0x01},
		RSSI:      -50.0,
		SNR:       10.0,
		EqSnr:     &primaryEqSnr,
		PacketCnt: 200,
		BaseStations: []mioty.BaseStationReception{
			{BsEui: 0x0000000000000001, Rssi: -50.0, Snr: 10.0, EqSnr: &bs1EqSnr},
			{BsEui: 0x0000000000000002, Rssi: -85.0, Snr: 8.5, EqSnr: nil},
		},
	}

	// Target BS2
	bs2Bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bs2Bytes, 0x0000000000000002)
	pbMsg := ulDataMessageToBaseStationMessageProto(msg, bs2Bytes)

	require.NotNil(t, pbMsg)
	assert.Equal(t, "0000000000000002", pbMsg.BsEui, "BsEui should be BS2, not primary")
	assert.Equal(t, -85.0, pbMsg.Rssi, "Rssi should be BS2 value")
	assert.Equal(t, 8.5, pbMsg.Snr, "Snr should be BS2 value")
	assert.Equal(t, float64(0), pbMsg.EqSnr, "EqSnr should be 0 (BS2 has nil), not leaked primary value")
}

// TestULDataMessageToBaseStationMessageProto_FallbackPath verifies that when the target BS
// is not present in BaseStations, the mapper falls back to row-level signal values
func TestULDataMessageToBaseStationMessageProto_FallbackPath(t *testing.T) {
	primaryEqSnr := 18.3
	bs1EqSnr := 17.0
	msg := &mioty.ULDataMessage{
		ID:        "msg-fallback",
		BsEui:     0x0000000000000001,
		EpEui:     0xfedcba9876543210,
		UserData:  []byte{0x02},
		RSSI:      -60.0,
		SNR:       11.0,
		EqSnr:     &primaryEqSnr,
		PacketCnt: 201,
		BaseStations: []mioty.BaseStationReception{
			{BsEui: 0x0000000000000001, Rssi: -60.0, Snr: 11.0, EqSnr: &bs1EqSnr},
		},
	}

	// Target BS2 — not present in BaseStations
	bs2Bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bs2Bytes, 0x0000000000000002)
	pbMsg := ulDataMessageToBaseStationMessageProto(msg, bs2Bytes)

	require.NotNil(t, pbMsg)
	assert.Equal(t, "0000000000000001", pbMsg.BsEui, "BsEui should remain row-level primary")
	assert.Equal(t, -60.0, pbMsg.Rssi, "Rssi should be row-level fallback")
	assert.Equal(t, 11.0, pbMsg.Snr, "Snr should be row-level fallback")
	assert.Equal(t, primaryEqSnr, pbMsg.EqSnr, "EqSnr should be row-level fallback (primary)")
}

// ============================================================================
// UpdateEndPoint FieldMask Tests
// ============================================================================

// mockEndpointSvcForUpdate captures the endpoint passed to Update/UpdateWithEUI for verification
type mockEndpointSvcForUpdate struct {
	getByEUIResult       *models.EndPoint
	getByEUIErr          error
	updateCapture        *models.EndPoint
	updateWithEUICapture *models.EndPoint
	updateErr            error
}

func (m *mockEndpointSvcForUpdate) Create(_ context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}

func (m *mockEndpointSvcForUpdate) GetByEUI(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
	return m.getByEUIResult, m.getByEUIErr
}

func (m *mockEndpointSvcForUpdate) Update(_ context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	m.updateCapture = ep
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return ep, nil
}

func (m *mockEndpointSvcForUpdate) Delete(_ context.Context, _ []byte, _ int64) error {
	return nil
}

func (m *mockEndpointSvcForUpdate) ListByModelWithSnapshot(_ context.Context, _ int64, _ uuid.UUID) ([]*models.EndPoint, error) {
	return nil, nil
}

func (m *mockEndpointSvcForUpdate) List(_ context.Context, _ int64, _, _ int) ([]*models.EndPoint, error) {
	return nil, nil
}

func (m *mockEndpointSvcForUpdate) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	m.updateWithEUICapture = ep
	return ep, nil
}

func (m *mockEndpointSvcForUpdate) CheckEUIGloballyUnique(_ context.Context, _ []byte) error {
	return nil
}

// TestUpdateEndPoint_FieldMask_BooleanPartialUpdate verifies that updating one boolean field
// preserves the values of other boolean fields when using a FieldMask.
func TestUpdateEndPoint_FieldMask_BooleanPartialUpdate(t *testing.T) {
	// Setup: existing endpoint with DualChan=true, Repetition=true
	existing := &models.EndPoint{
		EUI:         models.EUIFromString("1122334455667788"),
		TenantID:    1,
		DualChan:    true,
		Repetition:  true,
		WideCarrOff: false,
		LongBlkDist: false,
	}
	mockSvc := &mockEndpointSvcForUpdate{getByEUIResult: existing}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	// Request: only update DualChan to false, mask only includes fieldMaskDualChan
	mask := &fieldmaskpb.FieldMask{Paths: []string{fieldMaskDualChan}}
	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:    "1122334455667788",
			DualChan: false, // explicitly setting to false
		},
		UpdateMask: mask,
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.NoError(t, err)

	// Assert: DualChan changed to false, Repetition preserved as true
	require.NotNil(t, mockSvc.updateCapture)
	assert.False(t, mockSvc.updateCapture.DualChan, "DualChan should be updated to false")
	assert.True(t, mockSvc.updateCapture.Repetition, "Repetition should be preserved as true")
}

// TestUpdateEndPoint_FieldMask_ShAddrZero verifies that ShAddr can be set to 0 via FieldMask.
func TestUpdateEndPoint_FieldMask_ShAddrZero(t *testing.T) {
	shAddr := uint16(1234)
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
		ShAddr:   &shAddr, // non-zero initial value
	}
	mockSvc := &mockEndpointSvcForUpdate{getByEUIResult: existing}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	// Request: set ShAddr to 0 with mask
	mask := &fieldmaskpb.FieldMask{Paths: []string{fieldMaskShAddr}}
	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:  "1122334455667788",
			ShAddr: 0, // explicitly setting to 0
		},
		UpdateMask: mask,
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, mockSvc.updateCapture)
	require.NotNil(t, mockSvc.updateCapture.ShAddr, "ShAddr should not be nil")
	assert.Equal(t, uint16(0), *mockSvc.updateCapture.ShAddr, "ShAddr should be set to 0")
}

// TestUpdateEndPoint_FieldMask_AttachCntZero verifies that AttachCnt can be reset to 0 via FieldMask.
func TestUpdateEndPoint_FieldMask_AttachCntZero(t *testing.T) {
	attachCnt := uint32(42)
	existing := &models.EndPoint{
		EUI:       models.EUIFromString("1122334455667788"),
		TenantID:  1,
		AttachCnt: &attachCnt, // non-zero initial value
	}
	mockSvc := &mockEndpointSvcForUpdate{getByEUIResult: existing}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	// Request: reset AttachCnt to 0 with mask
	mask := &fieldmaskpb.FieldMask{Paths: []string{fieldMaskAttachCnt}}
	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:     "1122334455667788",
			AttachCnt: 0, // reset counter
		},
		UpdateMask: mask,
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, mockSvc.updateCapture)
	require.NotNil(t, mockSvc.updateCapture.AttachCnt, "AttachCnt should not be nil")
	assert.Equal(t, uint32(0), *mockSvc.updateCapture.AttachCnt, "AttachCnt should be reset to 0")
}

// TestUpdateEndPoint_EmptyMask_ReturnsInvalidArgument verifies that requests
// without a FieldMask are rejected with InvalidArgument.
func TestUpdateEndPoint_EmptyMask_ReturnsInvalidArgument(t *testing.T) {
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
	}
	mockSvc := &mockEndpointSvcForUpdate{getByEUIResult: existing}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui: "1122334455667788",
			Name:  "Updated Name",
		},
		// UpdateMask: nil (empty mask)
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.Error(t, err, "Empty mask should be rejected")
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestUpdateEndPoint_EUIOnlyChange_PreservesExistingBooleans verifies that changing only the EUI
// (no mask, no other fields) does not reset existing boolean fields to false.
func TestUpdateEndPoint_EUIOnlyChange_PreservesExistingBooleans(t *testing.T) {
	existing := &models.EndPoint{
		EUI:         models.EUIFromString("1122334455667788"),
		TenantID:    1,
		DualChan:    true,
		Repetition:  true,
		WideCarrOff: true,
		LongBlkDist: true,
		PreAttach:   true,
	}
	mockSvc := &mockEndpointSvcForUpdate{getByEUIResult: existing}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	// Request: only change EUI, with mask for name (minimal mask to pass validation)
	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui: "1122334455667788",
		},
		NewEpEui:   "AABBCCDDEEFF0011",
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.NoError(t, err)

	// All booleans must be preserved
	require.NotNil(t, mockSvc.updateWithEUICapture)
	assert.True(t, mockSvc.updateWithEUICapture.DualChan, "DualChan should be preserved")
	assert.True(t, mockSvc.updateWithEUICapture.Repetition, "Repetition should be preserved")
	assert.True(t, mockSvc.updateWithEUICapture.WideCarrOff, "WideCarrOff should be preserved")
	assert.True(t, mockSvc.updateWithEUICapture.LongBlkDist, "LongBlkDist should be preserved")
	assert.True(t, mockSvc.updateWithEUICapture.PreAttach, "PreAttach should be preserved")
}

// TestUpdateEndPoint_MaskedBooleanChange_StillApplies verifies that masked boolean changes
// are correctly applied while non-masked booleans are preserved.
func TestUpdateEndPoint_MaskedBooleanChange_StillApplies(t *testing.T) {
	existing := &models.EndPoint{
		EUI:       models.EUIFromString("1122334455667788"),
		TenantID:  1,
		DualChan:  true,
		PreAttach: true,
	}
	mockSvc := &mockEndpointSvcForUpdate{getByEUIResult: existing}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	// Request: mask includes dual_chan, sets DualChan=false
	mask := &fieldmaskpb.FieldMask{Paths: []string{fieldMaskDualChan}}
	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:    "1122334455667788",
			DualChan: false,
		},
		UpdateMask: mask,
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, mockSvc.updateCapture)
	assert.False(t, mockSvc.updateCapture.DualChan, "DualChan should be updated to false")
	assert.True(t, mockSvc.updateCapture.PreAttach, "PreAttach should be preserved (not in mask)")
}

func TestGetSystemStatus_Defaults(t *testing.T) {
	startTime := time.Unix(1700000000, 0).UTC()
	service := &CoreService{
		log:          &mockLogger{},
		scSwVersion:  "test-version",
		serviceStart: startTime,
	}

	ctx := testutil.TestContext()
	resp, err := service.GetSystemStatus(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.NotEmpty(t, resp.Version)
	assert.Equal(t, string(healthstatus.StatusHealthy), resp.Status)
	require.NotNil(t, resp.Uptime)
	assert.True(t, resp.Uptime.AsTime().Equal(startTime))
	assert.Equal(t, int32(0), resp.ActiveEndpoints)
	assert.Equal(t, int32(0), resp.ActiveBasestations)
	assert.Equal(t, int64(0), resp.MessagesProcessed)
}

// mockSystemStatusService implements grpcservices.SystemStatusService for testing
type mockSystemStatusService struct {
	metrics *grpcservices.SystemStatusMetrics
	err     error
}

func (m *mockSystemStatusService) GetStatus(_ context.Context, _ int64) (*grpcservices.SystemStatusMetrics, error) {
	return m.metrics, m.err
}

func (m *mockSystemStatusService) GetServiceStatuses(_ context.Context) ([]*grpcservices.ServiceStatusDTO, error) {
	return nil, nil
}

// TestGetSystemStatus_WithMetrics verifies tenant-scoped metrics are returned
// when SystemStatusService is wired.
func TestGetSystemStatus_WithMetrics(t *testing.T) {
	// Use testutil.TestContextWithTenant to include tenant context.
	ctx := testutil.TestContextWithTenant(123)

	mockSvc := &mockSystemStatusService{
		metrics: &grpcservices.SystemStatusMetrics{
			ActiveBasestations: 5,
			ActiveEndpoints:    100,
			MessagesProcessed:  50000,
		},
	}

	service := &CoreService{
		log:             &mockLogger{},
		scSwVersion:     "test-version",
		serviceStart:    time.Now(),
		systemStatusSvc: mockSvc,
	}

	resp, err := service.GetSystemStatus(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, int32(5), resp.ActiveBasestations)
	assert.Equal(t, int32(100), resp.ActiveEndpoints)
	assert.Equal(t, int64(50000), resp.MessagesProcessed)
}

// TestGetSystemStatus_ServiceError verifies handler returns defaults on service error.
func TestGetSystemStatus_ServiceError(t *testing.T) {
	ctx := testutil.TestContextWithTenant(123)

	mockSvc := &mockSystemStatusService{
		err: errors.New("database error"),
	}

	service := &CoreService{
		log:             &mockLogger{},
		scSwVersion:     "test-version",
		serviceStart:    time.Now(),
		systemStatusSvc: mockSvc,
	}

	resp, err := service.GetSystemStatus(ctx, &emptypb.Empty{})
	require.NoError(t, err) // Should not return error to client
	require.NotNil(t, resp)

	// Should return defaults on service error
	assert.Equal(t, int32(0), resp.ActiveBasestations)
	assert.Equal(t, int32(0), resp.ActiveEndpoints)
	assert.Equal(t, int64(0), resp.MessagesProcessed)
}

// ============================================================================
// CreateBaseStation tests.
// ============================================================================

// mockStorageForBaseStation implements storage.Storage for CreateBaseStation tests.
// Captures the basestation passed to CreateBaseStation for assertion.
type mockStorageForBaseStation struct {
	storage.Storage // Embed to satisfy interface (panics on unimplemented methods)
	captured        *models.BaseStation
}

func (m *mockStorageForBaseStation) CreateBaseStation(_ context.Context, bs *models.BaseStation) (*models.BaseStation, error) {
	m.captured = bs
	// Simulate ID assignment like real storage
	bs.ID = 1
	bs.CreatedAt = time.Now()
	bs.UpdatedAt = time.Now()
	return bs, nil
}

// TestGRPCCreateBaseStation_DefaultsBSSCI verifies BSSCI defaults are set by service layer.
// Uses real basestationService to exercise ConnectionType and ServiceCenterURL defaults.
func TestGRPCCreateBaseStation_DefaultsBSSCI(t *testing.T) {
	// Mock storage that captures the basestation
	mockStorage := &mockStorageForBaseStation{}

	// Minimal protocol config for test - service layer needs this to set ServiceCenterURL
	testProtocolCfg := &config.ProtocolConfig{
		BSCIHost: "localhost",
		BSCIPort: 5000,
	}

	// Use real basestationService with mock storage
	// This exercises the domain logic in basestation_service.go:32-45
	realBsSvc := grpcservices.NewBaseStationService(mockStorage, &mockLogger{}, testProtocolCfg)

	// Create gRPC service with real basestationService
	service := &CoreService{
		basestationSvc: realBsSvc,
		log:            &mockLogger{},
	}

	// Set up tenant context
	ctx := testutil.TestContextWithTenant(123)

	// Create request without ConnectionType - service should default to BSSCI
	req := &pb.CreateBaseStationRequest{
		Basestation: &pb.BaseStation{
			BsEui: "AABBCCDDEEFF0011",
			Name:  "Test Base Station",
		},
	}

	resp, err := service.CreateBaseStation(ctx, req)
	require.NoError(t, err, "CreateBaseStation should succeed")
	require.NotNil(t, resp, "Response should not be nil")

	// Verify BSSCI defaults were set by service layer
	require.NotNil(t, mockStorage.captured, "Storage should have received the basestation")
	assert.Equal(t, models.ConnectionTypeBSSCI, mockStorage.captured.ConnectionType,
		"ConnectionType should default to BSSCI")
	assert.NotNil(t, mockStorage.captured.ServiceCenterURL,
		"ServiceCenterURL should be set for BSSCI connections")
	assert.Equal(t, "tls://localhost:5000", *mockStorage.captured.ServiceCenterURL,
		"ServiceCenterURL should match expected format")
}

// TestGRPCCreateBaseStation_TagsPersisted verifies tags round-trip through create and response.
// Ensures baseStationToProto returns tags correctly.
func TestGRPCCreateBaseStation_TagsPersisted(t *testing.T) {
	// Mock storage that captures and returns the basestation with tags
	mockStorage := &mockStorageForBaseStation{}

	// Minimal protocol config for test
	testProtocolCfg := &config.ProtocolConfig{
		BSCIHost: "localhost",
		BSCIPort: 5000,
	}

	realBsSvc := grpcservices.NewBaseStationService(mockStorage, &mockLogger{}, testProtocolCfg)

	service := &CoreService{
		basestationSvc: realBsSvc,
		log:            &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(123)

	// Create request with tags
	req := &pb.CreateBaseStationRequest{
		Basestation: &pb.BaseStation{
			BsEui: "AABBCCDDEEFF0022",
			Name:  "Test Base Station With Tags",
			Tags: map[string]string{
				"env":    "test",
				"region": "us-east",
			},
		},
	}

	resp, err := service.CreateBaseStation(ctx, req)
	require.NoError(t, err, "CreateBaseStation should succeed")
	require.NotNil(t, resp, "Response should not be nil")

	// Verify tags are in the response
	require.NotNil(t, resp.Tags, "Response should have tags")
	assert.Equal(t, "test", resp.Tags["env"], "Tags should include env=test")
	assert.Equal(t, "us-east", resp.Tags["region"], "Tags should include region=us-east")
}

// ============================================================================
// CreateEndPoint tests.
// ============================================================================

// mockStorageForEndpoint implements storage.Storage for CreateEndPoint tests.
// Captures the endpoint passed to CreateEndPoint for assertion.
type mockStorageForEndpoint struct {
	storage.Storage // Embed to satisfy interface (panics on unimplemented methods)
	captured        *models.EndPoint
}

func (m *mockStorageForEndpoint) CreateEndPoint(_ context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	m.captured = ep
	// Simulate ID assignment like real storage
	ep.ID = 1
	ep.CreatedAt = time.Now()
	ep.UpdatedAt = time.Now()
	return ep, nil
}

func (m *mockStorageForEndpoint) UpdateEndPoint(_ context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	m.captured = ep
	ep.UpdatedAt = time.Now()
	return ep, nil
}

func (m *mockStorageForEndpoint) GetEndPoint(_ context.Context, eui []byte, tenantID int64) (*models.EndPoint, error) {
	// Return a pre-populated endpoint for update tests
	var epEui models.EUI
	copy(epEui[:], eui)
	return &models.EndPoint{
		ID:        1,
		EUI:       epEui,
		TenantID:  tenantID,
		Name:      "Existing Endpoint",
		EpStatus:  "inactive",
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}, nil
}

// TestGRPCCreateEndPoint_TagsPersisted verifies tags round-trip through create and response.
// Ensures endpointToProto returns tags correctly.
func TestGRPCCreateEndPoint_TagsPersisted(t *testing.T) {
	mockStorage := &mockStorageForEndpoint{}
	realEpSvc := grpcservices.NewEndpointService(mockStorage, &mockLogger{})

	service := &CoreService{
		endpointSvc: realEpSvc,
		log:         &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(123)
	req := &pb.CreateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:    "AABBCCDDEEFF0011",
			Name:     "Test Endpoint With Tags",
			EpClass:  "A",
			NwkSnKey: make([]byte, 16),
			AppKey:   make([]byte, 16),
			Tags: map[string]string{
				"env":    "test",
				"region": "us-east",
			},
		},
	}

	resp, err := service.CreateEndPoint(ctx, req)
	require.NoError(t, err, "CreateEndPoint should succeed")
	require.NotNil(t, resp, "Response should not be nil")

	// Assert tags passed to storage
	require.NotNil(t, mockStorage.captured, "Storage should have received the endpoint")
	require.NotNil(t, mockStorage.captured.Tags, "Captured endpoint should have tags")
	assert.Equal(t, "test", mockStorage.captured.Tags["env"], "Tags should include env=test")
	assert.Equal(t, "us-east", mockStorage.captured.Tags["region"], "Tags should include region=us-east")

	// Assert tags in response
	require.NotNil(t, resp.Tags, "Response should have tags")
	assert.Equal(t, "test", resp.Tags["env"], "Response tags should include env=test")
	assert.Equal(t, "us-east", resp.Tags["region"], "Response tags should include region=us-east")
}

// TestGRPCCreateEndPoint_DerivesBidiFromEpClass verifies bidi is derived from epClass.
func TestGRPCCreateEndPoint_DerivesBidiFromEpClass(t *testing.T) {
	mockStorage := &mockStorageForEndpoint{}
	realEpSvc := grpcservices.NewEndpointService(mockStorage, &mockLogger{})

	service := &CoreService{
		endpointSvc: realEpSvc,
		log:         &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(123)
	req := &pb.CreateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:    "AABBCCDDEEFF00AA",
			Name:     "Test Endpoint Bidi",
			EpClass:  "A",
			NwkSnKey: make([]byte, 16),
		},
	}

	_, err := service.CreateEndPoint(ctx, req)
	require.NoError(t, err, "CreateEndPoint should succeed")
	require.NotNil(t, mockStorage.captured, "Storage should have received the endpoint")
	assert.True(t, mockStorage.captured.Bidi, "Bidi should be true when EpClass is 'A'")

	req.Endpoint.EpClass = "Z"
	_, err = service.CreateEndPoint(ctx, req)
	require.NoError(t, err, "CreateEndPoint should succeed for EpClass 'Z'")
	require.NotNil(t, mockStorage.captured, "Storage should have received the endpoint")
	assert.False(t, mockStorage.captured.Bidi, "Bidi should be false when EpClass is 'Z'")
}

// Minimal stubs for NewCoreService constructor test.
// These satisfy nil-checks only — methods are never called.

type stubDownlinkSvc struct{}

func (s *stubDownlinkSvc) EnqueueDownlink(_ context.Context, _ uint64, _ []byte, _ float32, _ int64) (int64, error) {
	return 0, nil
}
func (s *stubDownlinkSvc) UpdateDownlinkStatus(_ context.Context, _ uint64, _ string, _ string) error {
	return nil
}
func (s *stubDownlinkSvc) ProcessDLDataResult(_ context.Context, _ *bssci.Session, _ *mioty.DLDataResult) (map[string]interface{}, error) {
	return nil, nil
}
func (s *stubDownlinkSvc) ProcessRevokeResponse(_ context.Context, _ *bssci.Session, _ int64, _ int64, _ uint64) (map[string]interface{}, error) {
	return nil, nil
}

type stubStatusSvc struct{}

func (s *stubStatusSvc) RecordPendingOperation(_ context.Context, _ *bssci.Session, _ int64, _ *bssci.PendingOperation, _ int64) error {
	return nil
}

func (s *stubStatusSvc) RecordPendingOperations(_ context.Context, _ *bssci.Session, _ []*bssci.PendingOperation, _ int64) error {
	return nil
}

func (s *stubStatusSvc) RestorePendingOperation(_ *bssci.Session, _ int64, _ *bssci.PendingOperation) {
}
func (s *stubStatusSvc) GetPendingOperation(_ *bssci.Session, _ int64) (*bssci.PendingOperation, error) {
	return nil, nil
}
func (s *stubStatusSvc) RemovePendingOperation(_ context.Context, _ *bssci.Session, _ int64) error {
	return nil
}
func (s *stubStatusSvc) ExtractQueueMetadata(_ *bssci.Session, _ int64) (uint64, int64, string) {
	return 0, 0, ""
}

type stubDownlinkScheduler struct{}

func (s *stubDownlinkScheduler) QueueDownlink(_ context.Context, _ *mioty.DLDataQueue, _ int64) (uint64, uint64, error) {
	return 0, 0, nil
}
func (s *stubDownlinkScheduler) RevokeDownlink(_ int64, _ uint64) (uint64, error) { return 0, nil }

type stubStatusReq struct{}

func (s *stubStatusReq) SendStatusRequest(_ interface{}) (int64, error) { return 0, nil }

type stubPingCmd struct{}

func (s *stubPingCmd) InitiatePing(_ context.Context, _ uint64, _ int64) (int64, error) {
	return 0, nil
}

type stubMessageStore struct{}

func (s *stubMessageStore) GetBaseStationMessageStats(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (*mioty.BaseStationMessageStats, error) {
	return nil, nil
}
func (s *stubMessageStore) GetBaseStationEndpointCounts(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (map[string]int64, error) {
	return nil, nil
}
func (s *stubMessageStore) GetBaseStationLastSeen(_ context.Context, _ int64, _ []byte) (*time.Time, error) {
	return nil, nil
}

type stubDownlinkStore struct{}

func (s *stubDownlinkStore) EnqueueDownlink(_ context.Context, _ *storage.DownlinkMessage) (*storage.DownlinkMessage, error) {
	return nil, nil
}
func (s *stubDownlinkStore) UpdateDownlinkStatus(_ context.Context, _ string, _ string, _ *uuid.UUID) error {
	return nil
}

type stubDLRXStorage struct{}

func (s *stubDLRXStorage) GetDLRXStatusQueryHistory(_ context.Context, _ int64, _ []byte, _, _ int, _, _ *time.Time) ([]*mioty.DLRXStatusQuery, int, error) {
	return nil, 0, nil
}
func (s *stubDLRXStorage) GetDLRXStatusQueryStats(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (int64, int64, int64, error) {
	return 0, 0, 0, nil
}

// Compile-time interface satisfaction checks for stubs.
var (
	_ bssci.DownlinkService       = (*stubDownlinkSvc)(nil)
	_ bssci.StatusService         = (*stubStatusSvc)(nil)
	_ scheduler.DownlinkScheduler = (*stubDownlinkScheduler)(nil)
	_ bssci.StatusRequester       = (*stubStatusReq)(nil)
	_ bssci.PingCommander         = (*stubPingCmd)(nil)
	_ MessageStore                = (*stubMessageStore)(nil)
	_ DownlinkStore               = (*stubDownlinkStore)(nil)
	_ DLRXStatusQueryStorage      = (*stubDLRXStorage)(nil)
)

// TestNewCoreService_DefaultEndpointActivityWindow verifies the constructor
// initializes endpointActivityWindow from config.DefaultEndpointActivityWindowHours,
// and that WithEndpointActivityWindow overrides it.
func TestNewCoreService_DefaultEndpointActivityWindow(t *testing.T) {
	mockStorage := &mockStorageForEndpoint{}
	epSvc := grpcservices.NewEndpointService(mockStorage, &mockLogger{})

	svc, err := NewCoreService(CoreServiceDeps{
		EndpointSvc:       epSvc,
		BasestationSvc:    &mockBasestationSvc{},
		MessageSvc:        &mockMessageSvc{},
		DownlinkSvc:       &stubDownlinkSvc{},
		StatusSvc:         &stubStatusSvc{},
		DownlinkCmd:       &mockDownlinkCmd{},
		DownlinkScheduler: &stubDownlinkScheduler{},
		SessionDir:        &mockSessionDir{},
		ULTransmit:        &mockULTransmit{},
		StatusReq:         &stubStatusReq{},
		PingCmd:           &stubPingCmd{},
		StatsStore:        &stubMessageStore{},
		DownlinkStore:     &stubDownlinkStore{},
		DLRXStorage:       &stubDLRXStorage{},
		SCEui:             0x0011223344556677,
		SCVendor:          "test",
		SCModel:           "test",
		SCName:            "test",
		SCSwVersion:       "1.0.0",
	})
	require.NoError(t, err, "NewCoreService must succeed")

	expected := time.Duration(config.DefaultEndpointActivityWindowHours) * time.Hour
	assert.Equal(t, expected, svc.endpointActivityWindow,
		"Default endpointActivityWindow must equal DefaultEndpointActivityWindowHours from config")
	assert.Equal(t, 24*time.Hour, svc.endpointActivityWindow,
		"Default endpointActivityWindow must be 24 hours")

	svc.WithEndpointActivityWindow(48 * time.Hour)
	assert.Equal(t, 48*time.Hour, svc.endpointActivityWindow,
		"WithEndpointActivityWindow must override the default")
}

// TestEndpointStatusMapping_UsesStatusOverAttachStatus verifies Status takes priority over AttachStatus.
// Tests frontend compatibility fix.
func TestEndpointStatusMapping_UsesStatusOverAttachStatus(t *testing.T) {
	mockStorage := &mockStorageForEndpoint{}
	realEpSvc := grpcservices.NewEndpointService(mockStorage, &mockLogger{})

	service := &CoreService{
		endpointSvc: realEpSvc,
		log:         &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(123)
	req := &pb.CreateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:        "AABBCCDDEEFF0022",
			Name:         "Test Endpoint Status Priority",
			EpClass:      "A",
			NwkSnKey:     make([]byte, 16),
			AppKey:       make([]byte, 16),
			Status:       "active",   // Should take priority
			AttachStatus: "attached", // Should be ignored
		},
	}

	resp, err := service.CreateEndPoint(ctx, req)
	require.NoError(t, err, "CreateEndPoint should succeed")
	require.NotNil(t, resp, "Response should not be nil")

	// Assert model got Status value (priority over AttachStatus)
	require.NotNil(t, mockStorage.captured, "Storage should have received the endpoint")
	assert.Equal(t, "active", mockStorage.captured.EpStatus,
		"EpStatus should be set from Status, not AttachStatus")

	// Status is derived from LastSeenAt (no LastSeenAt on newly created → "inactive")
	// AttachStatus reflects the stored lifecycle state (EpStatus set from request Status)
	assert.Equal(t, "inactive", resp.Status, "Status derives from LastSeenAt — newly created has no activity")
	assert.Equal(t, "active", resp.AttachStatus, "AttachStatus reflects stored EpStatus from request Status")
}

// TestEndpointStatusMapping_FallbackToAttachStatus verifies AttachStatus used when Status is empty.
// Tests backward compatibility with AttachStatus-only requests.
func TestEndpointStatusMapping_FallbackToAttachStatus(t *testing.T) {
	mockStorage := &mockStorageForEndpoint{}
	realEpSvc := grpcservices.NewEndpointService(mockStorage, &mockLogger{})

	service := &CoreService{
		endpointSvc: realEpSvc,
		log:         &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(123)
	req := &pb.CreateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:        "AABBCCDDEEFF0033",
			Name:         "Test Endpoint AttachStatus Fallback",
			EpClass:      "A",
			NwkSnKey:     make([]byte, 16),
			AppKey:       make([]byte, 16),
			Status:       "",         // Empty - should fall back
			AttachStatus: "attached", // Should be used as fallback
		},
	}

	resp, err := service.CreateEndPoint(ctx, req)
	require.NoError(t, err, "CreateEndPoint should succeed")
	require.NotNil(t, resp, "Response should not be nil")

	// Assert model got AttachStatus fallback value
	require.NotNil(t, mockStorage.captured, "Storage should have received the endpoint")
	assert.Equal(t, "attached", mockStorage.captured.EpStatus,
		"EpStatus should fall back to AttachStatus when Status is empty")

	// Status is derived from LastSeenAt (no LastSeenAt on newly created → "inactive")
	// AttachStatus reflects the stored lifecycle state (EpStatus set from AttachStatus fallback)
	assert.Equal(t, "inactive", resp.Status, "Status derives from LastSeenAt — newly created has no activity")
	assert.Equal(t, "attached", resp.AttachStatus, "AttachStatus reflects stored EpStatus from AttachStatus fallback")
}

// TestGRPCUpdateEndPoint_StatusPriority verifies Status takes priority over AttachStatus in updates.
// Tests frontend compatibility fix for UpdateEndPoint.
func TestGRPCUpdateEndPoint_StatusPriority(t *testing.T) {
	mockStorage := &mockStorageForEndpoint{}
	realEpSvc := grpcservices.NewEndpointService(mockStorage, &mockLogger{})

	service := &CoreService{
		endpointSvc: realEpSvc,
		log:         &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(123)
	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:        "AABBCCDDEEFF0044",
			Status:       "active",   // Should take priority
			AttachStatus: "attached", // Should be ignored
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}

	resp, err := service.UpdateEndPoint(ctx, req)
	require.NoError(t, err, "UpdateEndPoint should succeed")
	require.NotNil(t, resp, "Response should not be nil")

	// Assert model got Status value (priority over AttachStatus)
	require.NotNil(t, mockStorage.captured, "Storage should have received the endpoint")
	assert.Equal(t, "active", mockStorage.captured.EpStatus,
		"EpStatus should be set from Status, not AttachStatus")

	// Status is derived from LastSeenAt (mock has no LastSeenAt → "inactive")
	// AttachStatus reflects the stored lifecycle state (EpStatus set from request Status)
	assert.Equal(t, "inactive", resp.Status, "Status derives from LastSeenAt — mock has no activity")
	assert.Equal(t, "active", resp.AttachStatus, "AttachStatus reflects stored EpStatus from request Status")
}

// TestGRPCUpdateEndPoint_TagsPersisted verifies tags round-trip through update and response.
// Tests endpoint tags persistence in UpdateEndPoint.
func TestGRPCUpdateEndPoint_TagsPersisted(t *testing.T) {
	mockStorage := &mockStorageForEndpoint{}
	realEpSvc := grpcservices.NewEndpointService(mockStorage, &mockLogger{})

	service := &CoreService{
		endpointSvc: realEpSvc,
		log:         &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(123)
	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui: "AABBCCDDEEFF0055",
			Tags: map[string]string{
				"env":    "production",
				"region": "eu-west",
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}

	resp, err := service.UpdateEndPoint(ctx, req)
	require.NoError(t, err, "UpdateEndPoint should succeed")
	require.NotNil(t, resp, "Response should not be nil")

	// Assert tags passed to storage
	require.NotNil(t, mockStorage.captured, "Storage should have received the endpoint")
	require.NotNil(t, mockStorage.captured.Tags, "Captured endpoint should have tags")
	assert.Equal(t, "production", mockStorage.captured.Tags["env"], "Tags should include env=production")
	assert.Equal(t, "eu-west", mockStorage.captured.Tags["region"], "Tags should include region=eu-west")

	// Assert tags in response
	require.NotNil(t, resp.Tags, "Response should have tags")
	assert.Equal(t, "production", resp.Tags["env"], "Response tags should include env=production")
	assert.Equal(t, "eu-west", resp.Tags["region"], "Response tags should include region=eu-west")
}

// ============================================================================
// Certificate Download Tests
// ============================================================================

// mockCertSvc implements grpcservices.CertificateService for testing
type mockCertSvc struct {
	downloadByIDFunc func(ctx context.Context, certType, certID string) ([]byte, string, error)
	storedFunc       func(ctx context.Context, tenantID int64, bsEui []byte, certType string) ([]byte, string, error)
}

func (m *mockCertSvc) GenerateCertificate(_ context.Context, _ *grpcservices.CertificateRequest) (*grpcservices.CertificateResponse, error) {
	return nil, nil
}

func (m *mockCertSvc) DownloadCertificateByID(ctx context.Context, certType, certID string) ([]byte, string, error) {
	if m.downloadByIDFunc != nil {
		return m.downloadByIDFunc(ctx, certType, certID)
	}
	return nil, "", nil
}

func (m *mockCertSvc) GenerateServerCertificates(_ context.Context) error { return nil }

func (m *mockCertSvc) RenewServerCertificates(_ context.Context) error { return nil }

func (m *mockCertSvc) GetServerCertificateStatus(_ context.Context) (*grpcservices.CertificateStatus, error) {
	return nil, nil
}

func (m *mockCertSvc) GetStoredCertificate(ctx context.Context, tenantID int64, bsEui []byte, certType string) ([]byte, string, error) {
	if m.storedFunc != nil {
		return m.storedFunc(ctx, tenantID, bsEui, certType)
	}
	return nil, "", nil
}

// TestDownloadCertificate_IDBasedRouting verifies certificate download by ID returns correct data.
func TestDownloadCertificate_IDBasedRouting(t *testing.T) {
	ctx := testutil.TestContext()

	mock := &mockCertSvc{
		downloadByIDFunc: func(_ context.Context, certType, certID string) ([]byte, string, error) {
			assert.Equal(t, "client", certType)
			assert.Equal(t, "test-cert-id", certID)
			return []byte("-----BEGIN CERTIFICATE-----\n..."), "basestation-70-B3-D5-client-certificate.crt", nil
		},
	}

	svc := &CoreService{certSvc: mock, log: &mockLogger{}}

	resp, err := svc.DownloadCertificate(ctx, &pb.DownloadCertificateRequest{
		Id:       "test-cert-id",
		CertType: "client",
	})

	require.NoError(t, err)
	assert.Equal(t, "basestation-70-B3-D5-client-certificate.crt", resp.Filename)
	assert.Equal(t, grpcerrors.ContentTypePEM, resp.ContentType)
}

// TestDownloadCertificate_MissingID verifies missing ID returns ErrTokenIDRequired.
func TestDownloadCertificate_MissingID(t *testing.T) {
	ctx := testutil.TestContext()

	svc := &CoreService{certSvc: &mockCertSvc{}, log: &mockLogger{}}

	_, err := svc.DownloadCertificate(ctx, &pb.DownloadCertificateRequest{
		Id:       "", // Missing ID
		CertType: "client",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired), st.Message())
}

// TestDownloadCertificate_MissingCertType verifies missing cert_type returns ErrTokenCertTypeRequired.
func TestDownloadCertificate_MissingCertType(t *testing.T) {
	ctx := testutil.TestContext()

	svc := &CoreService{certSvc: &mockCertSvc{}, log: &mockLogger{}}

	_, err := svc.DownloadCertificate(ctx, &pb.DownloadCertificateRequest{
		Id:       "test-cert-id",
		CertType: "", // Missing cert_type
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertTypeRequired), st.Message())
}

// TestDownloadCertificate_ServiceNotConfigured verifies nil certSvc returns ErrTokenServiceNotConfigured.
func TestDownloadCertificate_ServiceNotConfigured(t *testing.T) {
	ctx := testutil.TestContext()

	svc := &CoreService{certSvc: nil, log: &mockLogger{}}

	_, err := svc.DownloadCertificate(ctx, &pb.DownloadCertificateRequest{
		Id:       "test-cert-id",
		CertType: "client",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured), st.Message())
}

// TestDownloadBaseStationCertificate_InvalidCertType verifies invalid cert_type returns ErrTokenCertTypeRequired.
func TestDownloadBaseStationCertificate_InvalidCertType(t *testing.T) {
	ctx := testutil.TestContextWithTenant(123)

	mock := &mockCertSvc{
		storedFunc: func(_ context.Context, _ int64, _ []byte, certType string) ([]byte, string, error) {
			// Service rejects invalid cert types
			if certType != grpcerrors.CertTypeCA && certType != grpcerrors.CertTypeClient && certType != grpcerrors.CertTypeKey {
				return nil, "", grpcerrors.NewTokenError(grpcerrors.ErrTokenCertTypeRequired, nil)
			}
			return nil, "", nil
		},
	}

	svc := &CoreService{certSvc: mock, log: &mockLogger{}}

	_, err := svc.DownloadBaseStationCertificate(ctx, &pb.DownloadBaseStationCertificateRequest{
		BsEui:    "0102030405060708",
		CertType: "bad",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertTypeRequired), st.Message())
}

// TestDownloadBaseStationCertificate_Valid verifies stored certificate download returns content and content type.
func TestDownloadBaseStationCertificate_Valid(t *testing.T) {
	ctx := testutil.TestContextWithTenant(123)

	mock := &mockCertSvc{
		storedFunc: func(_ context.Context, _ int64, bsEui []byte, certType string) ([]byte, string, error) {
			require.Len(t, bsEui, 8)
			assert.Equal(t, grpcerrors.CertTypeCA, certType)
			return []byte("cert-data"), "basestation-0102030405060708-ca-certificate.crt", nil
		},
	}

	svc := &CoreService{certSvc: mock, log: &mockLogger{}}

	resp, err := svc.DownloadBaseStationCertificate(ctx, &pb.DownloadBaseStationCertificateRequest{
		BsEui:    "0102030405060708",
		CertType: grpcerrors.CertTypeCA,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "basestation-0102030405060708-ca-certificate.crt", resp.Filename)
	assert.Equal(t, grpcerrors.ContentTypePEM, resp.ContentType)
	assert.Equal(t, []byte("cert-data"), resp.Content)
}

// TestDownloadBaseStationCertificate_KeyEncryptorNotConfigured verifies key download without keyEncryptor returns ErrTokenServiceNotConfigured.
func TestDownloadBaseStationCertificate_KeyEncryptorNotConfigured(t *testing.T) {
	ctx := testutil.TestContextWithTenant(123)

	mock := &mockCertSvc{
		storedFunc: func(_ context.Context, _ int64, _ []byte, certType string) ([]byte, string, error) {
			// Service returns ServiceNotConfigured when keyEncryptor is nil for key type
			if certType == grpcerrors.CertTypeKey {
				return nil, "", fmt.Errorf("%s", grpcerrors.ErrTokenServiceNotConfigured)
			}
			return nil, "", nil
		},
	}

	svc := &CoreService{certSvc: mock, log: &mockLogger{}}

	_, err := svc.DownloadBaseStationCertificate(ctx, &pb.DownloadBaseStationCertificateRequest{
		BsEui:    "0102030405060708",
		CertType: grpcerrors.CertTypeKey,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured), st.Message())
}

// =============================================================================
// Update endpoint error classification tests
// =============================================================================

// mockEndpointSvcForUpdateErrors extends the update mock with configurable error returns.
type mockEndpointSvcForUpdateErrors struct {
	getByEUIResult   *models.EndPoint
	updateErr        error
	updateWithEUIErr error
	checkEUIErr      error
}

func (m *mockEndpointSvcForUpdateErrors) Create(_ context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}

func (m *mockEndpointSvcForUpdateErrors) GetByEUI(_ context.Context, _ []byte, _ int64) (*models.EndPoint, error) {
	return m.getByEUIResult, nil
}

func (m *mockEndpointSvcForUpdateErrors) Update(_ context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return ep, nil
}

func (m *mockEndpointSvcForUpdateErrors) Delete(_ context.Context, _ []byte, _ int64) error {
	return nil
}

func (m *mockEndpointSvcForUpdateErrors) ListByModelWithSnapshot(_ context.Context, _ int64, _ uuid.UUID) ([]*models.EndPoint, error) {
	return nil, nil
}

func (m *mockEndpointSvcForUpdateErrors) List(_ context.Context, _ int64, _, _ int) ([]*models.EndPoint, error) {
	return nil, nil
}

func (m *mockEndpointSvcForUpdateErrors) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	if m.updateWithEUIErr != nil {
		return nil, m.updateWithEUIErr
	}
	return ep, nil
}

func (m *mockEndpointSvcForUpdateErrors) CheckEUIGloballyUnique(_ context.Context, _ []byte) error {
	return m.checkEUIErr
}

func TestUpdateEndPoint_NilKeys_BlueprintOnlySuccess(t *testing.T) {
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
		NwkSnKey: nil,
		AppKey:   nil,
	}
	mockSvc := &mockEndpointSvcForUpdateErrors{getByEUIResult: existing}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui: "1122334455667788",
			Name:  "Updated",
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}

	resp, err := service.UpdateEndPoint(ctx, req)
	require.NoError(t, err, "Update with nil keys should succeed")
	require.NotNil(t, resp)
}

func TestUpdateEndPoint_StorageErrAlreadyExists_ReturnsEndpointExists(t *testing.T) {
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
	}
	mockSvc := &mockEndpointSvcForUpdateErrors{
		getByEUIResult: existing,
		updateErr:      storage.ErrAlreadyExists,
	}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	req := &pb.UpdateEndPointRequest{
		Endpoint:   &pb.EndPoint{EpEui: "1122334455667788"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointExists), st.Message())
}

func TestUpdateEndPoint_StorageErrForeignKeyViolation_ReturnsDeviceModelNotFound(t *testing.T) {
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
	}
	mockSvc := &mockEndpointSvcForUpdateErrors{
		getByEUIResult: existing,
		updateErr:      storage.ErrForeignKeyViolation,
	}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	req := &pb.UpdateEndPointRequest{
		Endpoint:   &pb.EndPoint{EpEui: "1122334455667788"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound), st.Message())
}

func TestUpdateEndPoint_StorageErrNwkKeyLength_ReturnsNwkSnKeyLength(t *testing.T) {
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
	}
	mockSvc := &mockEndpointSvcForUpdateErrors{
		getByEUIResult: existing,
		updateErr:      storage.ErrNwkKeyLength,
	}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	req := &pb.UpdateEndPointRequest{
		Endpoint:   &pb.EndPoint{EpEui: "1122334455667788"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNwkSnKeyLength), st.Message())
}

func TestUpdateEndPoint_StorageErrAppKeyLength_ReturnsAppKeyLength(t *testing.T) {
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
	}
	mockSvc := &mockEndpointSvcForUpdateErrors{
		getByEUIResult: existing,
		updateErr:      storage.ErrAppKeyLength,
	}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	req := &pb.UpdateEndPointRequest{
		Endpoint:   &pb.EndPoint{EpEui: "1122334455667788"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAppKeyLength), st.Message())
}

// =============================================================================
// EUI-change path error classification tests
// =============================================================================

func TestUpdateEndPoint_EUIChange_CheckEUIQueryFails_ReturnsUpdateEUIFailed(t *testing.T) {
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
	}
	mockSvc := &mockEndpointSvcForUpdateErrors{
		getByEUIResult: existing,
		checkEUIErr:    fmt.Errorf("database connection lost"),
	}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	req := &pb.UpdateEndPointRequest{
		Endpoint:   &pb.EndPoint{EpEui: "1122334455667788"},
		NewEpEui:   "AABBCCDDEEFF0011",
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateEndpointEUIFailed), st.Message(),
		"Non-duplicate CheckEUI failures must NOT map to endpoint-exists")
}

func TestUpdateEndPoint_EUIChange_CheckEUIAlreadyExists_ReturnsEndpointExists(t *testing.T) {
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
	}
	mockSvc := &mockEndpointSvcForUpdateErrors{
		getByEUIResult: existing,
		checkEUIErr:    storage.ErrAlreadyExists,
	}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	req := &pb.UpdateEndPointRequest{
		Endpoint:   &pb.EndPoint{EpEui: "1122334455667788"},
		NewEpEui:   "AABBCCDDEEFF0011",
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointExists), st.Message())
}

func TestUpdateEndPoint_EUIChange_UpdateWithEUIErr_AlreadyExists(t *testing.T) {
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
	}
	mockSvc := &mockEndpointSvcForUpdateErrors{
		getByEUIResult:   existing,
		updateWithEUIErr: storage.ErrAlreadyExists,
	}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	req := &pb.UpdateEndPointRequest{
		Endpoint:   &pb.EndPoint{EpEui: "1122334455667788"},
		NewEpEui:   "AABBCCDDEEFF0011",
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEndpointExists), st.Message())
}

func TestUpdateEndPoint_EUIChange_UpdateWithEUIErr_ForeignKeyViolation(t *testing.T) {
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
	}
	mockSvc := &mockEndpointSvcForUpdateErrors{
		getByEUIResult:   existing,
		updateWithEUIErr: storage.ErrForeignKeyViolation,
	}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	req := &pb.UpdateEndPointRequest{
		Endpoint:   &pb.EndPoint{EpEui: "1122334455667788"},
		NewEpEui:   "AABBCCDDEEFF0011",
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeviceModelNotFound), st.Message())
}

func TestUpdateEndPoint_EUIChange_UpdateWithEUIErr_NwkKeyLength(t *testing.T) {
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
	}
	mockSvc := &mockEndpointSvcForUpdateErrors{
		getByEUIResult:   existing,
		updateWithEUIErr: storage.ErrNwkKeyLength,
	}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	req := &pb.UpdateEndPointRequest{
		Endpoint:   &pb.EndPoint{EpEui: "1122334455667788"},
		NewEpEui:   "AABBCCDDEEFF0011",
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNwkSnKeyLength), st.Message())
}

func TestUpdateEndPoint_EUIChange_UpdateWithEUIErr_AppKeyLength(t *testing.T) {
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
	}
	mockSvc := &mockEndpointSvcForUpdateErrors{
		getByEUIResult:   existing,
		updateWithEUIErr: storage.ErrAppKeyLength,
	}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	req := &pb.UpdateEndPointRequest{
		Endpoint:   &pb.EndPoint{EpEui: "1122334455667788"},
		NewEpEui:   "AABBCCDDEEFF0011",
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAppKeyLength), st.Message())
}

// =============================================================================
// TypeEUI Field Mask Update Tests (F1-Update Verification)
// =============================================================================

// TestUpdateEndPoint_FieldMask_SetTypeEUI verifies that type_eui can be set via FieldMask.
func TestUpdateEndPoint_FieldMask_SetTypeEUI(t *testing.T) {
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
	}
	mockSvc := &mockEndpointSvcForUpdate{getByEUIResult: existing}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	typeEuiBytes := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22}
	mask := &fieldmaskpb.FieldMask{Paths: []string{fieldMaskTypeEUI}}
	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:   "1122334455667788",
			TypeEui: typeEuiBytes,
		},
		UpdateMask: mask,
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, mockSvc.updateCapture)
	require.NotNil(t, mockSvc.updateCapture.TypeEUI, "TypeEUI should be set")
	assert.Equal(t, typeEuiBytes, mockSvc.updateCapture.TypeEUI[:], "TypeEUI should match request value")
}

// TestUpdateEndPoint_FieldMask_ClearTypeEUI verifies that type_eui can be cleared via FieldMask.
func TestUpdateEndPoint_FieldMask_ClearTypeEUI(t *testing.T) {
	var existingTypeEUI models.EUI
	copy(existingTypeEUI[:], []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22})

	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
		TypeEUI:  &existingTypeEUI,
	}
	mockSvc := &mockEndpointSvcForUpdate{getByEUIResult: existing}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	// Clear: mask includes type_eui but value is empty
	mask := &fieldmaskpb.FieldMask{Paths: []string{fieldMaskTypeEUI}}
	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:   "1122334455667788",
			TypeEui: nil, // empty = explicit clear
		},
		UpdateMask: mask,
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, mockSvc.updateCapture)
	assert.Nil(t, mockSvc.updateCapture.TypeEUI, "TypeEUI should be cleared to nil")
}

// TestUpdateEndPoint_FieldMask_InvalidTypeEUILength verifies that invalid type_eui length is rejected.
func TestUpdateEndPoint_FieldMask_InvalidTypeEUILength(t *testing.T) {
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
	}
	mockSvc := &mockEndpointSvcForUpdate{getByEUIResult: existing}
	service := &CoreService{endpointSvc: mockSvc, log: &mockLogger{}}
	ctx := testutil.TestContextWithTenant(1)

	// Invalid: 5 bytes instead of 8
	mask := &fieldmaskpb.FieldMask{Paths: []string{fieldMaskTypeEUI}}
	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:   "1122334455667788",
			TypeEui: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
		},
		UpdateMask: mask,
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidTypeEUIFormat), st.Code())
}

// ============================================================================
// Blueprint service mock for endpoint handler tests
// ============================================================================

// mockBlueprintSvcForEndpoint implements grpcservices.BlueprintService for endpoint tests.
// Only ResolveEffectiveTypeEUI and GetDeviceModelForTenant are used by endpoint handlers.
type mockBlueprintSvcForEndpoint struct {
	resolveTypeEUIFn     func(ctx context.Context, tenantID int64, modelID uuid.UUID) (*models.EUI, error)
	getDeviceModelResult *models.DeviceModel
	getDeviceModelErr    error
	getDefaultForModelFn func() (*models.Blueprint, error)
	getBlueprintFn       func(id uuid.UUID) (*models.Blueprint, error)
}

func (m *mockBlueprintSvcForEndpoint) ResolveEffectiveTypeEUI(ctx context.Context, tenantID int64, modelID uuid.UUID) (*models.EUI, error) {
	if m.resolveTypeEUIFn != nil {
		return m.resolveTypeEUIFn(ctx, tenantID, modelID)
	}
	return nil, nil
}

func (m *mockBlueprintSvcForEndpoint) GetDefaultForModel(_ context.Context, _ int64, _ uuid.UUID) (*models.Blueprint, error) {
	if m.getDefaultForModelFn != nil {
		return m.getDefaultForModelFn()
	}
	return nil, nil
}

func (m *mockBlueprintSvcForEndpoint) GetDeviceModelForTenant(_ context.Context, _ int64, _ uuid.UUID) (*models.DeviceModel, error) {
	if m.getDeviceModelErr != nil {
		return nil, m.getDeviceModelErr
	}
	if m.getDeviceModelResult != nil {
		return m.getDeviceModelResult, nil
	}
	return &models.DeviceModel{ID: uuid.New()}, nil
}

// Stub methods required by BlueprintService interface
func (m *mockBlueprintSvcForEndpoint) CreateManufacturer(_ context.Context, _ *grpcservices.ManufacturerCreateRequest) (*models.Manufacturer, error) {
	return nil, nil
}
func (m *mockBlueprintSvcForEndpoint) GetManufacturer(_ context.Context, _ uuid.UUID) (*models.Manufacturer, error) {
	return nil, nil
}
func (m *mockBlueprintSvcForEndpoint) UpdateManufacturer(_ context.Context, _ uuid.UUID, _ *grpcservices.ManufacturerUpdateRequest) (*models.Manufacturer, error) {
	return nil, nil
}
func (m *mockBlueprintSvcForEndpoint) DeleteManufacturer(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockBlueprintSvcForEndpoint) ListManufacturers(_ context.Context, _ bool, _, _ int) ([]*models.Manufacturer, int64, error) {
	return nil, 0, nil
}
func (m *mockBlueprintSvcForEndpoint) CreateDeviceModel(_ context.Context, _ *grpcservices.DeviceModelCreateRequest) (*models.DeviceModel, error) {
	return nil, nil
}
func (m *mockBlueprintSvcForEndpoint) GetDeviceModel(_ context.Context, _ uuid.UUID) (*models.DeviceModel, error) {
	return nil, nil
}
func (m *mockBlueprintSvcForEndpoint) UpdateDeviceModel(_ context.Context, _ uuid.UUID, _ *grpcservices.DeviceModelUpdateRequest) (*models.DeviceModel, error) {
	return nil, nil
}
func (m *mockBlueprintSvcForEndpoint) DeleteDeviceModel(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockBlueprintSvcForEndpoint) ListDeviceModels(_ context.Context, _ bool, _ *uuid.UUID, _, _ int) ([]*models.DeviceModel, int64, error) {
	return nil, 0, nil
}
func (m *mockBlueprintSvcForEndpoint) CreateBlueprint(_ context.Context, _ *grpcservices.BlueprintCreateRequest) (*models.Blueprint, error) {
	return nil, nil
}
func (m *mockBlueprintSvcForEndpoint) GetBlueprint(_ context.Context, id uuid.UUID) (*models.Blueprint, error) {
	if m.getBlueprintFn != nil {
		return m.getBlueprintFn(id)
	}
	return nil, nil
}
func (m *mockBlueprintSvcForEndpoint) UpdateBlueprint(_ context.Context, _ uuid.UUID, _ *grpcservices.BlueprintUpdateRequest) (*models.Blueprint, error) {
	return nil, nil
}
func (m *mockBlueprintSvcForEndpoint) DeleteBlueprint(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockBlueprintSvcForEndpoint) ListBlueprints(_ context.Context, _ bool, _ *uuid.UUID, _, _ int) ([]*models.Blueprint, int64, error) {
	return nil, 0, nil
}
func (m *mockBlueprintSvcForEndpoint) SetDefaultBlueprint(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockBlueprintSvcForEndpoint) SubmitToRegistry(_ context.Context, _ uuid.UUID, _ *grpcservices.RegistrySubmitRequest) (*grpcservices.RegistrySubmitResult, error) {
	return nil, nil
}
func (m *mockBlueprintSvcForEndpoint) CreateDeviceModelWithBlueprint(_ context.Context, _ *grpcservices.DeviceModelWithBlueprintRequest) (*models.DeviceModel, *models.Blueprint, error) {
	return nil, nil, nil
}
func (m *mockBlueprintSvcForEndpoint) DecodePreview(_ context.Context, _ uuid.UUID, _ []byte, _ uint8) (*grpcservices.DecodePreviewResult, error) {
	return nil, nil
}

func (m *mockBlueprintSvcForEndpoint) DecodePreviewInline(_ context.Context, _, _ []byte, _ uint8) (*grpcservices.DecodePreviewResult, error) {
	return nil, nil
}

// ============================================================================
// TypeEUI/DeviceModel Precedence Tests
// ============================================================================

// TestCreateEndPoint_TypeEUI_PrecedenceOverride verifies that when both user typeEui
// and deviceModelId are provided, the resolver's value wins.
func TestCreateEndPoint_TypeEUI_PrecedenceOverride(t *testing.T) {
	modelID := uuid.New()
	resolverEUI := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11}

	mockEpSvc := &mockEndpointSvcForDuplicate{}
	bpSvc := &mockBlueprintSvcForEndpoint{
		resolveTypeEUIFn: func(_ context.Context, _ int64, id uuid.UUID) (*models.EUI, error) {
			assert.Equal(t, modelID, id)
			return &resolverEUI, nil
		},
	}

	service := &CoreService{
		endpointSvc:  mockEpSvc,
		blueprintSvc: bpSvc,
		log:          &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(1)
	req := &pb.CreateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:         "0000000000000001",
			Name:          "Test EP",
			NwkSnKey:      make([]byte, 16),
			ShAddr:        1,
			LastPacketCnt: 0,
			DeviceModelId: modelID.String(),
			TypeEui:       []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}, // User value
		},
	}

	resp, err := service.CreateEndPoint(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// The stored TypeEUI should be the resolver's, not the user's
	assert.Equal(t, resolverEUI[:], resp.TypeEui, "TypeEUI should be resolver value, not user-provided")
}

// TestCreateEndPoint_TypeEUI_ResolverErrorPropagates verifies that resolver failures
// return the proper gRPC error token.
func TestCreateEndPoint_TypeEUI_ResolverErrorPropagates(t *testing.T) {
	modelID := uuid.New()

	mockEpSvc := &mockEndpointSvcForDuplicate{}
	bpSvc := &mockBlueprintSvcForEndpoint{
		resolveTypeEUIFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.EUI, error) {
			return nil, fmt.Errorf("db connection lost")
		},
	}

	service := &CoreService{
		endpointSvc:  mockEpSvc,
		blueprintSvc: bpSvc,
		log:          &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(1)
	req := &pb.CreateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:         "0000000000000001",
			Name:          "Test EP",
			NwkSnKey:      make([]byte, 16),
			ShAddr:        1,
			LastPacketCnt: 0,
			DeviceModelId: modelID.String(),
		},
	}

	_, err := service.CreateEndPoint(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResolveTypeEUIFailed), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResolveTypeEUIFailed), st.Message())
}

// TestUpdateEndPoint_NoMask_ReturnsInvalidArgument verifies that a request
// without a FieldMask is rejected with InvalidArgument.
func TestUpdateEndPoint_NoMask_ReturnsInvalidArgument(t *testing.T) {
	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
		Name:     "Existing EP",
	}

	mockSvc := &mockEndpointSvcForUpdate{getByEUIResult: existing}
	service := &CoreService{
		endpointSvc: mockSvc,
		log:         &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(1)
	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui: "1122334455667788",
			Name:  "Updated Name",
		},
		// No UpdateMask
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.Error(t, err, "No mask should be rejected")
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestUpdateEndPoint_TypeEuiMaskOnly_ExistingModelEnforcesPrecedence verifies that
// when an update mask contains only type_eui (not device_model_id), the post-field
// normalization enforces model→typeEui precedence from the existing DeviceModelID.
func TestUpdateEndPoint_TypeEuiMaskOnly_ExistingModelEnforcesPrecedence(t *testing.T) {
	existingModelID := uuid.New()
	resolverEUI := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11}

	existing := &models.EndPoint{
		EUI:           models.EUIFromString("1122334455667788"),
		TenantID:      1,
		Name:          "Existing EP",
		DeviceModelID: &existingModelID,
	}

	mockSvc := &mockEndpointSvcForUpdate{getByEUIResult: existing}
	bpSvc := &mockBlueprintSvcForEndpoint{
		resolveTypeEUIFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.EUI, error) {
			return &resolverEUI, nil
		},
	}

	service := &CoreService{
		endpointSvc:  mockSvc,
		blueprintSvc: bpSvc,
		log:          &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(1)
	userTypeEui := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	mask := &fieldmaskpb.FieldMask{Paths: []string{fieldMaskTypeEUI}}
	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:   "1122334455667788",
			TypeEui: userTypeEui, // User tries to override
		},
		UpdateMask: mask,
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.NoError(t, err)

	captured := mockSvc.updateCapture
	require.NotNil(t, captured)
	// TypeEUI should be the resolver's value, not user's
	require.NotNil(t, captured.TypeEUI, "TypeEUI should not be nil when model has default blueprint")
	assert.Equal(t, resolverEUI, *captured.TypeEUI, "TypeEUI should be resolver value, not user payload")
}

// TestUpdateEndPoint_MaskedDeviceModelID_ResolverError verifies that when a masked
// device_model_id update triggers the resolver and it fails, the proper gRPC error
// token is returned (covers line 791-795).
func TestUpdateEndPoint_MaskedDeviceModelID_ResolverError(t *testing.T) {
	modelID := uuid.New()

	existing := &models.EndPoint{
		EUI:      models.EUIFromString("1122334455667788"),
		TenantID: 1,
		Name:     "Existing EP",
	}

	mockSvc := &mockEndpointSvcForUpdate{getByEUIResult: existing}
	bpSvc := &mockBlueprintSvcForEndpoint{
		getDeviceModelResult: &models.DeviceModel{ID: modelID},
		resolveTypeEUIFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.EUI, error) {
			return nil, fmt.Errorf("db connection lost")
		},
	}

	service := &CoreService{
		endpointSvc:  mockSvc,
		blueprintSvc: bpSvc,
		log:          &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(1)
	mask := &fieldmaskpb.FieldMask{Paths: []string{fieldMaskDeviceModelID}}
	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:         "1122334455667788",
			DeviceModelId: modelID.String(),
		},
		UpdateMask: mask,
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResolveTypeEUIFailed), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResolveTypeEUIFailed), st.Message())
}

// TestUpdateEndPoint_PostNormalization_ResolverError verifies that when the post-field
// normalization re-resolves TypeEUI for an existing DeviceModelID and the resolver fails,
// the proper gRPC error token is returned (covers line 835-839).
func TestUpdateEndPoint_PostNormalization_ResolverError(t *testing.T) {
	existingModelID := uuid.New()

	existing := &models.EndPoint{
		EUI:           models.EUIFromString("1122334455667788"),
		TenantID:      1,
		Name:          "Existing EP",
		DeviceModelID: &existingModelID,
	}

	mockSvc := &mockEndpointSvcForUpdate{getByEUIResult: existing}
	bpSvc := &mockBlueprintSvcForEndpoint{
		resolveTypeEUIFn: func(_ context.Context, _ int64, _ uuid.UUID) (*models.EUI, error) {
			return nil, fmt.Errorf("db connection lost")
		},
	}

	service := &CoreService{
		endpointSvc:  mockSvc,
		blueprintSvc: bpSvc,
		log:          &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(1)
	// Mask contains only type_eui — device_model_id block is skipped,
	// but post-normalization fires because existing DeviceModelID is non-nil.
	mask := &fieldmaskpb.FieldMask{Paths: []string{fieldMaskTypeEUI}}
	req := &pb.UpdateEndPointRequest{
		Endpoint: &pb.EndPoint{
			EpEui:   "1122334455667788",
			TypeEui: []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
		},
		UpdateMask: mask,
	}

	_, err := service.UpdateEndPoint(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResolveTypeEUIFailed), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResolveTypeEUIFailed), st.Message())
}

// ============================================================================
// baseStationToProto – ServiceCenterUrl mapping tests.
// ============================================================================

// TestBaseStationToProto_ServiceCenterURL_Stored verifies the proto response
// includes the per-BS stored ServiceCenterURL when set on the model.
func TestBaseStationToProto_ServiceCenterURL_Stored(t *testing.T) {
	storedURL := "tls://bssci.example.com:5000"
	bs := &models.BaseStation{
		TenantID:         1,
		Name:             "Test BS",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		ServiceCenterURL: &storedURL,
	}

	service := &CoreService{
		protocolConfig: &config.ProtocolConfig{
			BSCIHost: "localhost",
			BSCIPort: 5000,
		},
		log: &mockLogger{},
	}

	result := service.baseStationToProto(bs)
	assert.Equal(t, storedURL, result.ServiceCenterUrl,
		"ServiceCenterUrl should use the per-BS stored value")
}

// TestBaseStationToProto_ServiceCenterURL_NoFallback verifies the proto response
// returns an empty ServiceCenterUrl when the model has no stored URL.
func TestBaseStationToProto_ServiceCenterURL_NoFallback(t *testing.T) {
	bs := &models.BaseStation{
		TenantID:  1,
		Name:      "Test BS",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	service := &CoreService{
		protocolConfig: &config.ProtocolConfig{
			BSCIHost: "fallback-host",
			BSCIPort: 5555,
		},
		log: &mockLogger{},
	}

	result := service.baseStationToProto(bs)
	assert.Equal(t, "", result.ServiceCenterUrl,
		"ServiceCenterUrl should be empty when no per-BS URL is stored")
}

// ============================================================================
// Geolocation tests for CreateBaseStation / UpdateBaseStation
// ============================================================================

// TestUpdateBaseStation_LocationLookupFix verifies that updating a base station
// with location data uses the real ID from the GetByEUI lookup to feed into the
// Update call, and that GetByEUI errors are distinguished (not-found vs internal).
func TestUpdateBaseStation_LocationLookupFix(t *testing.T) {
	const tenantID int64 = 1
	validEUI := "70b3d59cd00009e6"

	existingBS := &models.BaseStation{
		ID:       42,
		EUI:      models.EUIFromString(validEUI),
		TenantID: tenantID,
		Name:     "Existing BS",
	}

	var updatedBS *models.BaseStation
	mockBsSvc := &mockBasestationSvc{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.BaseStation, error) {
			return existingBS, nil
		},
		updateFunc: func(_ context.Context, bs *models.BaseStation) (*models.BaseStation, error) {
			updatedBS = bs
			// Return the same object with updated fields
			return bs, nil
		},
	}

	svc := &CoreService{
		basestationSvc: mockBsSvc,
		log:            &mockLogger{},
		protocolConfig: &config.ProtocolConfig{},
	}

	ctx := testutil.TestContextWithTenant(tenantID)
	lat := 48.137154
	lon := 11.576124
	req := &pb.UpdateBaseStationRequest{
		Basestation: &pb.BaseStation{
			BsEui:     validEUI,
			Latitude:  wrapperspb.Double(lat),
			Longitude: wrapperspb.Double(lon),
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"latitude", "longitude"}},
	}

	resp, err := svc.UpdateBaseStation(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify the update used the real DB ID from the lookup
	require.NotNil(t, updatedBS, "Update should have been called")
	assert.Equal(t, int64(42), updatedBS.ID, "Update should use the ID from the lookup result")
	require.NotNil(t, updatedBS.Latitude)
	assert.InDelta(t, lat, *updatedBS.Latitude, 0.0001)
	require.NotNil(t, updatedBS.Longitude)
	assert.InDelta(t, lon, *updatedBS.Longitude, 0.0001)

	// Verify location source was set to manual
	require.NotNil(t, updatedBS.LocationSource)
	assert.Equal(t, models.LocationSourceManual, *updatedBS.LocationSource)
}

// TestCreateBaseStation_WithCoordinates verifies that DoubleValue wrappers
// round-trip correctly through CreateBaseStation, including explicit 0.0 values.
func TestCreateBaseStation_WithCoordinates(t *testing.T) {
	const tenantID int64 = 1
	validEUI := "70b3d59cd00009e6"

	mockBsSvc := &mockBasestationSvc{
		createFunc: func(_ context.Context, bs *models.BaseStation) (*models.BaseStation, error) {
			// Verify coordinates were extracted from wrappers
			require.NotNil(t, bs.Latitude, "Latitude should be set from DoubleValue wrapper")
			require.NotNil(t, bs.Longitude, "Longitude should be set from DoubleValue wrapper")
			assert.InDelta(t, 0.0, *bs.Latitude, 0.0001, "Latitude 0.0 should round-trip correctly")
			assert.InDelta(t, 0.0, *bs.Longitude, 0.0001, "Longitude 0.0 should round-trip correctly")
			require.NotNil(t, bs.LocationSource)
			assert.Equal(t, models.LocationSourceManual, *bs.LocationSource)
			// Return with an ID assigned
			bs.ID = 99
			return bs, nil
		},
	}

	svc := &CoreService{
		basestationSvc: mockBsSvc,
		log:            &mockLogger{},
		protocolConfig: &config.ProtocolConfig{},
	}

	ctx := testutil.TestContextWithTenant(tenantID)
	req := &pb.CreateBaseStationRequest{
		Basestation: &pb.BaseStation{
			BsEui:     validEUI,
			Name:      "Zero Coord BS",
			Latitude:  wrapperspb.Double(0.0),
			Longitude: wrapperspb.Double(0.0),
		},
	}

	resp, err := svc.CreateBaseStation(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify proto response includes the 0.0 wrappers
	require.NotNil(t, resp.Latitude, "Proto response should include Latitude wrapper for 0.0")
	assert.InDelta(t, 0.0, resp.Latitude.GetValue(), 0.0001)
	require.NotNil(t, resp.Longitude, "Proto response should include Longitude wrapper for 0.0")
	assert.InDelta(t, 0.0, resp.Longitude.GetValue(), 0.0001)
}

// TestCreateBaseStation_LatOnly_RejectsPartialPair verifies that providing latitude
// without longitude returns a validation error requiring both coordinates.
func TestCreateBaseStation_LatOnly_RejectsPartialPair(t *testing.T) {
	const tenantID int64 = 1
	validEUI := "70b3d59cd00009e6"

	svc := &CoreService{
		basestationSvc: &mockBasestationSvc{},
		log:            &mockLogger{},
		protocolConfig: &config.ProtocolConfig{},
	}

	ctx := testutil.TestContextWithTenant(tenantID)
	req := &pb.CreateBaseStationRequest{
		Basestation: &pb.BaseStation{
			BsEui:    validEUI,
			Name:     "Partial Coord BS",
			Latitude: wrapperspb.Double(48.137154),
			// Longitude intentionally omitted
		},
	}

	resp, err := svc.CreateBaseStation(ctx, req)
	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLatLonPairRequired), st.Message())
}

// TestUpdateBaseStation_FieldMask_ClearCoordinates verifies that including lat/lon
// in the field mask but sending nil wrappers clears the stored coordinates.
func TestUpdateBaseStation_FieldMask_ClearCoordinates(t *testing.T) {
	const tenantID int64 = 1
	validEUI := "70b3d59cd00009e6"
	existingLat := 48.137154
	existingLon := 11.576124
	src := models.LocationSourceManual
	now := time.Now()

	existingBS := &models.BaseStation{
		ID:                42,
		EUI:               models.EUIFromString(validEUI),
		TenantID:          tenantID,
		Name:              "Existing BS",
		Latitude:          &existingLat,
		Longitude:         &existingLon,
		LocationSource:    &src,
		LocationUpdatedAt: &now,
	}

	var updatedBS *models.BaseStation
	mockBsSvc := &mockBasestationSvc{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.BaseStation, error) {
			return existingBS, nil
		},
		updateFunc: func(_ context.Context, bs *models.BaseStation) (*models.BaseStation, error) {
			updatedBS = bs
			return bs, nil
		},
	}

	svc := &CoreService{
		basestationSvc: mockBsSvc,
		log:            &mockLogger{},
		protocolConfig: &config.ProtocolConfig{},
	}

	ctx := testutil.TestContextWithTenant(tenantID)
	req := &pb.UpdateBaseStationRequest{
		Basestation: &pb.BaseStation{
			BsEui: validEUI,
			// Latitude and Longitude are nil (clear)
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"latitude", "longitude"},
		},
	}

	resp, err := svc.UpdateBaseStation(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.NotNil(t, updatedBS, "Update should have been called")
	assert.Nil(t, updatedBS.Latitude, "Latitude should be cleared when masked but nil")
	assert.Nil(t, updatedBS.Longitude, "Longitude should be cleared when masked but nil")
	assert.Nil(t, updatedBS.LocationSource, "LocationSource should be cleared with coordinates")
	assert.Nil(t, updatedBS.LocationUpdatedAt, "LocationUpdatedAt should be cleared with coordinates")
}

// TestUpdateBaseStation_FieldMask_NameOnly_LocationUnchanged verifies that updating
// only the name via field mask does not modify existing location data.
func TestUpdateBaseStation_FieldMask_NameOnly_LocationUnchanged(t *testing.T) {
	const tenantID int64 = 1
	validEUI := "70b3d59cd00009e6"
	existingLat := 48.137154
	existingLon := 11.576124
	src := models.LocationSourceManual
	locTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	existingBS := &models.BaseStation{
		ID:                42,
		EUI:               models.EUIFromString(validEUI),
		TenantID:          tenantID,
		Name:              "Old Name",
		Latitude:          &existingLat,
		Longitude:         &existingLon,
		LocationSource:    &src,
		LocationUpdatedAt: &locTime,
	}

	var updatedBS *models.BaseStation
	mockBsSvc := &mockBasestationSvc{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.BaseStation, error) {
			return existingBS, nil
		},
		updateFunc: func(_ context.Context, bs *models.BaseStation) (*models.BaseStation, error) {
			updatedBS = bs
			return bs, nil
		},
	}

	svc := &CoreService{
		basestationSvc: mockBsSvc,
		log:            &mockLogger{},
		protocolConfig: &config.ProtocolConfig{},
	}

	ctx := testutil.TestContextWithTenant(tenantID)
	req := &pb.UpdateBaseStationRequest{
		Basestation: &pb.BaseStation{
			BsEui: validEUI,
			Name:  "New Name",
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"name"},
		},
	}

	resp, err := svc.UpdateBaseStation(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.NotNil(t, updatedBS, "Update should have been called")
	assert.Equal(t, "New Name", updatedBS.Name)
	require.NotNil(t, updatedBS.Latitude, "Latitude should remain unchanged")
	assert.InDelta(t, existingLat, *updatedBS.Latitude, 0.0001)
	require.NotNil(t, updatedBS.Longitude, "Longitude should remain unchanged")
	assert.InDelta(t, existingLon, *updatedBS.Longitude, 0.0001)
	require.NotNil(t, updatedBS.LocationSource, "LocationSource should remain unchanged")
	assert.Equal(t, models.LocationSourceManual, *updatedBS.LocationSource)
	require.NotNil(t, updatedBS.LocationUpdatedAt, "LocationUpdatedAt should remain unchanged")
	assert.Equal(t, locTime, *updatedBS.LocationUpdatedAt)
}

// TestUpdateBaseStation_DBError_NotMaskedAsNotFound verifies that a non-storage
// error from the GetByEUI lookup returns codes.Internal, not codes.NotFound.
func TestUpdateBaseStation_DBError_NotMaskedAsNotFound(t *testing.T) {
	const tenantID int64 = 1
	validEUI := "70b3d59cd00009e6"

	mockBsSvc := &mockBasestationSvc{
		getByEUIFunc: func(_ context.Context, _ []byte, _ int64) (*models.BaseStation, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	svc := &CoreService{
		basestationSvc: mockBsSvc,
		log:            &mockLogger{},
		protocolConfig: &config.ProtocolConfig{},
	}

	ctx := testutil.TestContextWithTenant(tenantID)
	req := &pb.UpdateBaseStationRequest{
		Basestation: &pb.BaseStation{
			BsEui:     validEUI,
			Latitude:  wrapperspb.Double(48.0),
			Longitude: wrapperspb.Double(11.0),
		},
	}

	resp, err := svc.UpdateBaseStation(ctx, req)
	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	assert.Equal(t, codes.Internal, st.Code(),
		"Non-storage errors should return Internal, not NotFound")
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateBaseStationFailed), st.Message())
}

// ============================================================================
// Test Cases for ListAllBaseStationLocations
// ============================================================================

// locOrgMapper implements OrgMapper for location tests
type locOrgMapper struct {
	getDefaultOrgFunc func(ctx context.Context, tenantID int64) (string, error)
}

func (m *locOrgMapper) GetDefaultOrgForTenant(ctx context.Context, tenantID int64) (string, error) {
	if m.getDefaultOrgFunc != nil {
		return m.getDefaultOrgFunc(ctx, tenantID)
	}
	return "", nil
}

func TestListAllBaseStationLocations(t *testing.T) {
	lat1 := 48.1351
	lon1 := 11.5820
	lat2 := 52.5200
	lon2 := 13.4050
	alt2 := 34.5
	locSrc := "gps"

	sampleStations := []*models.BaseStation{
		{
			EUI:            models.EUI{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x09, 0xE6},
			TenantID:       1,
			Name:           "Munich BS",
			Latitude:       &lat1,
			Longitude:      &lon1,
			LocationSource: &locSrc,
			IsOnline:       true,
		},
		{
			EUI:            models.EUI{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x09, 0xE2},
			TenantID:       4,
			Name:           "Berlin BS",
			Latitude:       &lat2,
			Longitude:      &lon2,
			Altitude:       &alt2,
			LocationSource: &locSrc,
			IsOnline:       false,
		},
	}

	adminUserID := uuid.New().String()

	t.Run("non-admin user returns PERMISSION_DENIED", func(t *testing.T) {
		svc := &CoreService{
			log:          &mockLogger{},
			adminChecker: &mockAdminChecker{isAdmin: false},
			orgMapper:    &locOrgMapper{},
		}

		ctx := pkgcontext.WithUserID(testutil.TestContext(), adminUserID)

		resp, err := svc.ListAllBaseStationLocations(ctx, &pb.ListAllBaseStationLocationsRequest{})
		require.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, st.Code())
	})

	t.Run("admin user returns locations with org_id", func(t *testing.T) {
		svc := &CoreService{
			log: &mockLogger{},
			basestationSvc: &locationMockBsSvc{
				mockBasestationSvc: &mockBasestationSvc{},
				listAllLocationsFunc: func(_ context.Context) ([]*models.BaseStation, error) {
					return sampleStations, nil
				},
			},
			adminChecker: &mockAdminChecker{isAdmin: true},
			orgMapper: &locOrgMapper{
				getDefaultOrgFunc: func(_ context.Context, tenantID int64) (string, error) {
					switch tenantID {
					case 1:
						return "org-uuid-tenant-1", nil
					case 4:
						return "org-uuid-tenant-4", nil
					default:
						return "", fmt.Errorf("unknown tenant %d", tenantID)
					}
				},
			},
		}

		ctx := pkgcontext.WithUserID(testutil.TestContext(), adminUserID)

		resp, err := svc.ListAllBaseStationLocations(ctx, &pb.ListAllBaseStationLocationsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int32(2), resp.TotalCount)
		require.Len(t, resp.Locations, 2)

		assert.Equal(t, "70b3d59cd00009e6", resp.Locations[0].BsEui)
		assert.Equal(t, "org-uuid-tenant-1", resp.Locations[0].OrgId)
		assert.Equal(t, true, resp.Locations[0].IsOnline)

		assert.Equal(t, "70b3d59cd00009e2", resp.Locations[1].BsEui)
		assert.Equal(t, "org-uuid-tenant-4", resp.Locations[1].OrgId)
		assert.Equal(t, false, resp.Locations[1].IsOnline)
		assert.NotNil(t, resp.Locations[1].Altitude)
	})

	t.Run("missing user context returns error", func(t *testing.T) {
		svc := &CoreService{
			log:          &mockLogger{},
			adminChecker: &mockAdminChecker{isAdmin: true},
			orgMapper:    &locOrgMapper{},
		}

		ctx := testutil.TestContext()
		resp, err := svc.ListAllBaseStationLocations(ctx, &pb.ListAllBaseStationLocationsRequest{})
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("admin checker failure returns INTERNAL", func(t *testing.T) {
		svc := &CoreService{
			log:          &mockLogger{},
			adminChecker: &mockAdminChecker{err: fmt.Errorf("identity service unavailable")},
			orgMapper:    &locOrgMapper{},
		}

		ctx := pkgcontext.WithUserID(testutil.TestContext(), adminUserID)

		resp, err := svc.ListAllBaseStationLocations(ctx, &pb.ListAllBaseStationLocationsRequest{})
		require.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})

	t.Run("org mapping failure returns INTERNAL (fail-closed)", func(t *testing.T) {
		svc := &CoreService{
			log: &mockLogger{},
			basestationSvc: &locationMockBsSvc{
				mockBasestationSvc: &mockBasestationSvc{},
				listAllLocationsFunc: func(_ context.Context) ([]*models.BaseStation, error) {
					return sampleStations, nil
				},
			},
			adminChecker: &mockAdminChecker{isAdmin: true},
			orgMapper: &locOrgMapper{
				getDefaultOrgFunc: func(_ context.Context, tenantID int64) (string, error) {
					if tenantID == 4 {
						return "", fmt.Errorf("org lookup failed")
					}
					return "org-uuid-tenant-1", nil
				},
			},
		}

		ctx := pkgcontext.WithUserID(testutil.TestContext(), adminUserID)

		resp, err := svc.ListAllBaseStationLocations(ctx, &pb.ListAllBaseStationLocationsRequest{})
		require.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})

	t.Run("nil admin checker returns INTERNAL", func(t *testing.T) {
		svc := &CoreService{
			log:       &mockLogger{},
			orgMapper: &locOrgMapper{},
		}

		ctx := pkgcontext.WithUserID(testutil.TestContext(), adminUserID)

		resp, err := svc.ListAllBaseStationLocations(ctx, &pb.ListAllBaseStationLocationsRequest{})
		require.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})

	t.Run("nil org mapper returns INTERNAL", func(t *testing.T) {
		svc := &CoreService{
			log: &mockLogger{},
			basestationSvc: &locationMockBsSvc{
				mockBasestationSvc: &mockBasestationSvc{},
				listAllLocationsFunc: func(_ context.Context) ([]*models.BaseStation, error) {
					return sampleStations, nil
				},
			},
			adminChecker: &mockAdminChecker{isAdmin: true},
			// orgMapper intentionally nil
		}

		ctx := pkgcontext.WithUserID(testutil.TestContext(), adminUserID)

		resp, err := svc.ListAllBaseStationLocations(ctx, &pb.ListAllBaseStationLocationsRequest{})
		require.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})
}

// locationMockBsSvc wraps mockBasestationSvc to override ListAllLocations
type locationMockBsSvc struct {
	*mockBasestationSvc
	listAllLocationsFunc func(ctx context.Context) ([]*models.BaseStation, error)
}

func (m *locationMockBsSvc) ListAllLocations(ctx context.Context) ([]*models.BaseStation, error) {
	if m.listAllLocationsFunc != nil {
		return m.listAllLocationsFunc(ctx)
	}
	return nil, nil
}
