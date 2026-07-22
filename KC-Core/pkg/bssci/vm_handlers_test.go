package bssci

import (
	"testing"
	"time"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// TestVMHandlerSignatures validates all VM handler function signatures
// This ensures handlers conform to the expected HandlerFunc type
func TestVMHandlerSignatures(t *testing.T) {
	server := createTestServerWithSession()
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:             "test-session-vm",
			BaseStationEUI: TestEpEui01,
			DbSessionID:    1,
		},
		Name: "test-bs",
	}
	msg := &Message{
		Command: mioty.CmdVMActivate,
		OpId:    -1,
	}
	data := make(map[string]interface{})

	t.Run("handleVMActivate", func(t *testing.T) {
		err := server.handleVMActivate(server, session, msg, data)
		assert.Error(t, err, "VM activate should return error for BS-initiated request")
		assert.Contains(t, err.Error(), ResolveErrorMessage(errVMOperationSentByBS))
	})

	t.Run("handleVMActivateResponse", func(t *testing.T) {
		// Use StatusService to record pending operation
		pendingOp := &PendingOperation{
			OperationType: mioty.CmdVMActivate,
			SessionSlug:   session.ID,
			OperationID:   -1,
			Endpoint:      []byte{0, 0, 0, 0, 0, 0, 0, 1},
			MACType:       1,
			Timestamp:     time.Now(),
		}
		ctx := testutil.TestContext()
		err := server.statusSvc.RecordPendingOperation(ctx, session, -1, pendingOp, session.DbSessionID)
		require.NoError(t, err, "Failed to record pending operation")

		data["result"] = true
		err = server.handleVMActivateResponse(server, session, msg, data)
		assert.NoError(t, err, "Valid VM activate response should succeed")
	})

	t.Run("handleVMActivateComplete", func(t *testing.T) {
		err := server.handleVMActivateComplete(server, session, msg, data)
		assert.NoError(t, err, "VM activate complete should clean up operation")
	})

	t.Run("handleVMDeactivate", func(t *testing.T) {
		err := server.handleVMDeactivate(server, session, msg, data)
		assert.Error(t, err, "VM deactivate should return error for BS-initiated request")
		assert.Contains(t, err.Error(), ResolveErrorMessage(errVMOperationSentByBS))
	})

	t.Run("handleVMDeactivateResponse", func(t *testing.T) {
		// Use StatusService to record pending operation
		pendingOp := &PendingOperation{
			SessionSlug:   session.ID,
			OperationID:   -2,
			OperationType: mioty.CmdVMDeactivate,
			Endpoint:      []byte{0, 0, 0, 0, 0, 0, 0, 1},
			MACType:       1,
			Timestamp:     time.Now(),
		}
		ctx := testutil.TestContext()
		err := server.statusSvc.RecordPendingOperation(ctx, session, -2, pendingOp, session.DbSessionID)
		require.NoError(t, err, "Failed to record pending operation")

		msg.OpId = -2
		data["result"] = true
		err = server.handleVMDeactivateResponse(server, session, msg, data)
		assert.NoError(t, err, "Valid VM deactivate response should succeed")
	})

	t.Run("handleVMDeactivateComplete", func(t *testing.T) {
		err := server.handleVMDeactivateComplete(server, session, msg, data)
		assert.NoError(t, err, "VM deactivate complete should clean up operation")
	})

	t.Run("handleVMStatus", func(t *testing.T) {
		err := server.handleVMStatus(server, session, msg, data)
		assert.Error(t, err, "VM status should return error for BS-initiated request")
		assert.Contains(t, err.Error(), ResolveErrorMessage(errVMOperationSentByBS))
	})

	t.Run("handleVMStatusResponse", func(t *testing.T) {
		// Use StatusService to record pending operation
		pendingOp := &PendingOperation{
			SessionSlug:   session.ID,
			OperationID:   -3,
			OperationType: mioty.CmdVMStatus,
			Endpoint:      []byte{0, 0, 0, 0, 0, 0, 0, 1},
			Timestamp:     time.Now(),
		}
		ctx := testutil.TestContext()
		err := server.statusSvc.RecordPendingOperation(ctx, session, -3, pendingOp, session.DbSessionID)
		require.NoError(t, err, "Failed to record pending operation")

		msg.OpId = -3
		data["macTypes"] = []interface{}{float64(1), float64(2)}
		err = server.handleVMStatusResponse(server, session, msg, data)
		assert.NoError(t, err, "Valid VM status response should succeed")
	})

	t.Run("handlePingComplete", func(t *testing.T) {
		// Test that ping complete properly records timestamp via UpdatePingTimestamp
		// This verifies the mock's lastPing map is updated
		err := server.handlePingComplete(server, session, msg, data)
		assert.NoError(t, err, "Ping complete should succeed")

		// Verify timestamp was recorded in mock's lastPing map
		mockSvc, ok := server.sessionSvc.(*mockSessionService)
		require.True(t, ok, "session service should be mockSessionService")

		pingTime, exists := mockSvc.GetLastPing(session.DbSessionID)
		require.True(t, exists, "ping timestamp should be recorded")
		assert.WithinDuration(t, time.Now(), pingTime, 2*time.Second, "ping timestamp should be recent")
	})
}

// TestVMCommunityEditionStubs validates community edition behavior for VM handlers
// Note: Response/Complete handlers for SC-initiated operations (Status, DL Data) are supported
// Only unsolicited BS-initiated operations (like handleVMDLData) return unsupported errors
func TestVMCommunityEditionStubs(t *testing.T) {
	server := createTestServerWithSession()
	conn := &bsscitest.TestConn{Encoding: "msgpack"}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			BaseStationEUI: TestEpEui01,
			DbSessionID:    1,
		},
		Name: "test-bs",
		Conn: conn,
	}
	msg := &Message{
		Command: mioty.CmdVMDLData,
		OpId:    -1,
	}
	data := make(map[string]interface{})

	// handleVMDLData is unsupported because it represents unsolicited BS→SC VM requests
	// (VM operations are SC-initiated, so we don't expect to receive VM DL Data from BS)
	t.Run("handleVMDLData_unsupported", func(t *testing.T) {
		msg.Command = mioty.CmdVMDLData
		err := server.handleVMDLData(server, session, msg, data)
		assert.Error(t, err, "Community edition should return unsupported error")
		assert.Contains(t, err.Error(), ResolveErrorMessage(errUnsupportedCommand))

		// Verify catalog error was sent to base station
		code, errMsg := conn.LastError()
		assert.Equal(t, 38, code, "Should send POSIX_ENOSYS (38)")
		assert.Equal(t, "Unsupported command", errMsg)
	})
}

// TestVMNilSessionHandling validates handlers properly reject nil sessions
func TestVMNilSessionHandling(t *testing.T) {
	server := createTestServerWithSession()
	msg := &Message{
		Command: mioty.CmdVMActivateResponse,
		OpId:    -1,
	}
	data := make(map[string]interface{})

	t.Run("handleVMActivateResponse_nil_session", func(t *testing.T) {
		err := server.handleVMActivateResponse(server, nil, msg, data)
		assert.Error(t, err, "Nil session should return error")
		assert.Contains(t, err.Error(), ResolveErrorMessage(errSessionNil))
	})

	t.Run("handleVMActivateComplete_nil_session", func(t *testing.T) {
		err := server.handleVMActivateComplete(server, nil, msg, data)
		assert.Error(t, err, "Nil session should return error")
		assert.Contains(t, err.Error(), ResolveErrorMessage(errSessionNil))
	})

	t.Run("handleVMDeactivateResponse_nil_session", func(t *testing.T) {
		err := server.handleVMDeactivateResponse(server, nil, msg, data)
		assert.Error(t, err, "Nil session should return error")
		assert.Contains(t, err.Error(), ResolveErrorMessage(errSessionNil))
	})

	t.Run("handleVMDeactivateComplete_nil_session", func(t *testing.T) {
		err := server.handleVMDeactivateComplete(server, nil, msg, data)
		assert.Error(t, err, "Nil session should return error")
		assert.Contains(t, err.Error(), ResolveErrorMessage(errSessionNil))
	})

	t.Run("handleVMStatusResponse_nil_session", func(t *testing.T) {
		err := server.handleVMStatusResponse(server, nil, msg, data)
		assert.Error(t, err, "Nil session should return error")
		assert.Contains(t, err.Error(), ResolveErrorMessage(errSessionNil))
	})

	t.Run("handleVMStatusComplete_nil_session", func(t *testing.T) {
		err := server.handleVMStatusComplete(server, nil, msg, data)
		assert.Error(t, err, "Nil session should return error")
	})

	t.Run("handleVMDLData_nil_session", func(t *testing.T) {
		err := server.handleVMDLData(server, nil, msg, data)
		assert.Error(t, err, "Nil session should return error")
	})

	t.Run("handleVMDLDataResponse_nil_session", func(t *testing.T) {
		err := server.handleVMDLDataResponse(server, nil, msg, data)
		assert.Error(t, err, "Nil session should return error")
	})

	t.Run("handleVMDLDataComplete_nil_session", func(t *testing.T) {
		err := server.handleVMDLDataComplete(server, nil, msg, data)
		assert.Error(t, err, "Nil session should return error")
	})
}

// TestVMSendMethods validates Send* public methods
func TestVMSendMethods(t *testing.T) {
	server := createTestServerWithSession()

	// Register test session
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:             "test-session-vm",
			BaseStationEUI: TestEpEui01,
			DbSessionID:    1,
			LastScOpId:     -1,
		},
		Name: "test-bs-vm",
	}
	server.RegisterSession(session)

	t.Run("SendVMActivate_session_not_found", func(t *testing.T) {
		err := server.SendVMActivate("nonexistent-session", TestEpEui01, 1)
		assert.Error(t, err, "Nonexistent session should return error")
		assert.Contains(t, err.Error(), ResolveErrorMessage(errSessionNotFound))
	})

	t.Run("SendVMDeactivate_session_not_found", func(t *testing.T) {
		err := server.SendVMDeactivate("nonexistent-session", TestEpEui01, 1)
		assert.Error(t, err, "Nonexistent session should return error")
		assert.Contains(t, err.Error(), ResolveErrorMessage(errSessionNotFound))
	})

	t.Run("SendVMStatus_session_not_found", func(t *testing.T) {
		err := server.SendVMStatus("nonexistent-session", TestEpEui01)
		assert.Error(t, err, "Nonexistent session should return error")
		assert.Contains(t, err.Error(), ResolveErrorMessage(errSessionNotFound))
	})

	t.Run("SendVMDownlinkData_session_not_found", func(t *testing.T) {
		err := server.SendVMDownlinkData("nonexistent-session", TestEpEui01, 1, []byte("test"))
		assert.Error(t, err, "Nonexistent session should return error")
		assert.Contains(t, err.Error(), ResolveErrorMessage(errSessionNotFound))
	})
}

// TestVMActiveTypesTracking validates session ActiveVMTypes map management
func TestVMActiveTypesTracking(t *testing.T) {
	server := createTestServerWithSession()
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			BaseStationEUI: TestEpEui01,
			DbSessionID:    1,
		},
		Name:          "test-bs",
		ActiveVMTypes: make(map[uint64][]uint8),
	}
	msg := &Message{
		Command: mioty.CmdVMActivateResponse,
		OpId:    -1,
	}

	t.Run("activate_adds_mac_type", func(t *testing.T) {
		// Use StatusService to record pending operation
		pendingOp := &PendingOperation{
			SessionSlug:   session.ID,
			OperationID:   -1,
			OperationType: mioty.CmdVMActivate,
			Endpoint:      []byte{0, 0, 0, 0, 0, 0, 0, 1},
			MACType:       1,
			Timestamp:     time.Now(),
		}
		ctx := testutil.TestContext()
		err := server.statusSvc.RecordPendingOperation(ctx, session, -1, pendingOp, session.DbSessionID)
		require.NoError(t, err, "Failed to record pending operation")

		data := map[string]interface{}{"result": true}
		err = server.handleVMActivateResponse(server, session, msg, data)
		require.NoError(t, err)

		// Verify MAC type was added
		server.mu.RLock()
		types := session.ActiveVMTypes[1]
		server.mu.RUnlock()
		assert.Contains(t, types, uint8(1), "MAC type 1 should be active")
	})

	t.Run("deactivate_removes_mac_type", func(t *testing.T) {
		// Setup session with active MAC type
		session.ActiveVMTypes[1] = []uint8{1, 2}

		// Use StatusService to record pending operation
		pendingOp := &PendingOperation{
			SessionSlug:   session.ID,
			OperationID:   -2,
			OperationType: mioty.CmdVMDeactivate,
			Endpoint:      []byte{0, 0, 0, 0, 0, 0, 0, 1},
			MACType:       1,
			Timestamp:     time.Now(),
		}
		ctx := testutil.TestContext()
		err := server.statusSvc.RecordPendingOperation(ctx, session, -2, pendingOp, session.DbSessionID)
		require.NoError(t, err, "Failed to record pending operation")

		msg.OpId = -2
		data := map[string]interface{}{"result": true}
		err = server.handleVMDeactivateResponse(server, session, msg, data)
		require.NoError(t, err)

		// Verify MAC type 1 was removed but 2 remains
		server.mu.RLock()
		types := session.ActiveVMTypes[1]
		server.mu.RUnlock()
		assert.NotContains(t, types, uint8(1), "MAC type 1 should be deactivated")
		assert.Contains(t, types, uint8(2), "MAC type 2 should remain active")
	})

	t.Run("status_response_replaces_mac_types", func(t *testing.T) {
		// Use StatusService to record pending operation
		pendingOp := &PendingOperation{
			OperationType: mioty.CmdVMStatus,
			SessionSlug:   session.ID,
			OperationID:   -3,
			Endpoint:      []byte{0, 0, 0, 0, 0, 0, 0, 1},
			Timestamp:     time.Now(),
		}
		ctx := testutil.TestContext()
		err := server.statusSvc.RecordPendingOperation(ctx, session, -3, pendingOp, session.DbSessionID)
		require.NoError(t, err, "Failed to record pending operation")

		msg.OpId = -3
		data := map[string]interface{}{
			"macTypes": []interface{}{float64(3), float64(4)},
		}
		err = server.handleVMStatusResponse(server, session, msg, data)
		require.NoError(t, err)

		// Verify MAC types were replaced
		server.mu.RLock()
		types := session.ActiveVMTypes[1]
		server.mu.RUnlock()
		assert.Equal(t, []uint8{3, 4}, types, "Status should replace active MAC types")
	})
}

// createTestServerWithSession creates a minimal test server with dependencies
// Uses CreateTestServices to wire all required service dependencies
func createTestServerWithSession() *Server {
	testLogger := logger.NewNop()

	// Create all required services via test helper
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)

	s := NewTestServer(
		testLogger,
		mockStorage,
		nil, // eventStore
		1,   // tenantID
		sessionSvc, downlinkSvc, statusSvc, connectionSvc,
		broadcaster, queueSerializer, auditLogger, tenantResolver,
	)

	s.deduplicator = NewMessageDeduplicator(1000)
	return s
}

// NewMockEventStore creates a minimal mock event store for testing
func NewMockEventStore() *MockEventStore {
	return &MockEventStore{}
}

// MockEventStore provides minimal event storage for tests
type MockEventStore struct{}

// CreateEvent stores an event (no-op for tests)
func (m *MockEventStore) CreateEvent(_ interface{}, _ *models.SystemEvent) error {
	return nil
}
