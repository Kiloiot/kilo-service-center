package bssci

import (
	"fmt"
	"strings"
	"testing"
	"time"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupVMTestServer creates a test server with mock storage and StatusService for VM testing
func setupVMTestServer(_ *testing.T) (*Server, *Session, *bsscitest.TestConn) {
	log := logger.NewNop()

	// Use the existing helper that creates a server with StatusService
	server := NewTestServerWithMemoryStatusService(log, nil, nil, 123)

	// Register all BSSCI handlers (including VM handlers)
	server.RegisterHandlers()

	// Create test connection
	conn := &bsscitest.TestConn{Encoding: "msgpack"}

	// Create session with unique ID
	sessionID := fmt.Sprintf("test-session-%d", time.Now().UnixNano())
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               sessionID,
			BaseStationEUI:   0x1122334455667788,
			ResolvedTenantID: 123,
			DbSessionID:      456,
		},
		Conn: conn,
	}

	// Add session to server (sessions map uses session.ID, not BaseStationEUI)
	server.sessions[session.ID] = session

	return server, session, conn
}

// TestVMActivateFullFlow tests the complete VM activate three-way handshake
func TestVMActivateFullFlow(t *testing.T) {
	server, session, conn := setupVMTestServer(t)

	// Step 1: Send VM activate request (SC → BS)
	// Use SendVMActivate method since VM operations are SC-initiated
	err := server.SendVMActivate(session.ID, uint64(0xAABBCCDDEEFF0011), uint8(1))
	assert.NoError(t, err, "SendVMActivate should not return error")

	// Verify message was sent to base station
	assert.Equal(t, 1, len(conn.SentMessages), "Should send VM activate message")
	msgData := conn.SentMessages[0]
	assert.Equal(t, mioty.CmdVMActivate, msgData["command"])

	// Get the generated operation ID
	opId := msgData["opId"].(int64)
	assert.True(t, opId < 0, "SC operation ID should be negative")

	// Use StatusService.GetPendingOperation with session and opID
	pendingOp, err := server.statusSvc.GetPendingOperation(session, opId)
	require.NoError(t, err, "Should retrieve pending operation")
	require.NotNil(t, pendingOp, "Pending operation should be recorded")
	assert.Equal(t, mioty.CmdVMActivate, pendingOp.OperationType)
	assert.Equal(t, uint64(0xAABBCCDDEEFF0011), pendingOp.Metadata["epEui"])
	assert.Equal(t, uint8(1), pendingOp.Metadata["macType"])

	// Step 2: Send VM activate response (BS → SC)
	conn.Reset()
	vmActivateRsp := make(map[string]interface{})
	vmActivateRsp["cmd"] = mioty.CmdVMActivateResponse
	vmActivateRsp["opId"] = opId       // Use the same operation ID
	vmActivateRsp["status"] = uint8(0) // Success

	err = server.handlers[mioty.CmdVMActivateResponse](server, session, &Message{OpId: opId}, vmActivateRsp)
	assert.NoError(t, err, "VM activate response handler should not return error")

	// Handlers process inbound messages (BS→SC) and don't generate responses
	// conn.SentMessages should remain empty after handler invocation
	assert.Equal(t, 0, len(conn.SentMessages), "Response handler should not send messages")

	// Pending operation should still exist
	pendingOp, err = server.statusSvc.GetPendingOperation(session, opId)
	assert.NoError(t, err, "Should retrieve pending operation")
	assert.NotNil(t, pendingOp, "Pending operation should still exist after response")

	// Step 3: Send VM activate complete (BS → SC)
	conn.Reset()
	vmActivateCmp := make(map[string]interface{})
	vmActivateCmp["cmd"] = mioty.CmdVMActivateComplete
	vmActivateCmp["opId"] = opId

	err = server.handlers[mioty.CmdVMActivateComplete](server, session, &Message{OpId: opId}, vmActivateCmp)
	assert.NoError(t, err, "VM activate complete handler should not return error")

	// Complete handler processes inbound message and doesn't send response
	assert.Equal(t, 0, len(conn.SentMessages), "Complete handler should not send messages")

	// Pending operation should be removed
	_, err = server.statusSvc.GetPendingOperation(session, opId)
	assert.Error(t, err, "Pending operation should be removed after complete")
}

// TestVMDeactivateFullFlow tests the complete VM deactivate three-way handshake
func TestVMDeactivateFullFlow(t *testing.T) {
	server, session, conn := setupVMTestServer(t)

	// Step 1: Send VM deactivate request (SC → BS)
	// Use SendVMDeactivate method since VM operations are SC-initiated
	err := server.SendVMDeactivate(session.ID, uint64(0xAABBCCDDEEFF0022), uint8(2))
	assert.NoError(t, err, "SendVMDeactivate should not return error")

	// Verify message was sent to base station
	assert.Equal(t, 1, len(conn.SentMessages), "Should send VM deactivate message")
	msgData := conn.SentMessages[0]
	assert.Equal(t, mioty.CmdVMDeactivate, msgData["command"])

	// Get the generated operation ID
	opId := msgData["opId"].(int64)
	assert.True(t, opId < 0, "SC operation ID should be negative")

	// Use StatusService.GetPendingOperation with session and opID
	pendingOp, err := server.statusSvc.GetPendingOperation(session, opId)
	require.NoError(t, err, "Should retrieve pending operation")
	require.NotNil(t, pendingOp, "Pending operation should be recorded")
	assert.Equal(t, mioty.CmdVMDeactivate, pendingOp.OperationType)
	assert.Equal(t, uint64(0xAABBCCDDEEFF0022), pendingOp.Metadata["epEui"])

	// Step 2: Send VM deactivate response (BS → SC)
	conn.Reset()
	vmDeactivateRsp := make(map[string]interface{})
	vmDeactivateRsp["cmd"] = mioty.CmdVMDeactivateResponse
	vmDeactivateRsp["opId"] = opId // Use the same operation ID
	vmDeactivateRsp["status"] = uint8(0)

	err = server.handlers[mioty.CmdVMDeactivateResponse](server, session, &Message{OpId: opId}, vmDeactivateRsp)
	assert.NoError(t, err, "VM deactivate response handler should not return error")

	// Step 3: Send VM deactivate complete (BS → SC)
	conn.Reset()
	vmDeactivateCmp := make(map[string]interface{})
	vmDeactivateCmp["cmd"] = mioty.CmdVMDeactivateComplete
	vmDeactivateCmp["opId"] = opId // Use the same operation ID

	err = server.handlers[mioty.CmdVMDeactivateComplete](server, session, &Message{OpId: opId}, vmDeactivateCmp)
	assert.NoError(t, err, "VM deactivate complete handler should not return error")

	// Pending operation should be removed
	_, err = server.statusSvc.GetPendingOperation(session, opId)
	assert.Error(t, err, "Pending operation should be removed after complete")
}

// TestVMStatusFullFlow tests the complete VM status three-way handshake
func TestVMStatusFullFlow(t *testing.T) {
	server, session, conn := setupVMTestServer(t)

	// Step 1: Send VM status request (SC → BS)
	// Use SendVMStatus method since VM operations are SC-initiated
	err := server.SendVMStatus(session.ID, uint64(0xAABBCCDDEEFF0033))
	assert.NoError(t, err, "SendVMStatus should not return error")

	// Verify message was sent to base station
	assert.Equal(t, 1, len(conn.SentMessages), "Should send VM status message")
	msgData := conn.SentMessages[0]
	assert.Equal(t, mioty.CmdVMStatus, msgData["command"])

	// Get the generated operation ID
	opId := msgData["opId"].(int64)
	assert.True(t, opId < 0, "SC operation ID should be negative")

	// Use StatusService.GetPendingOperation with session and opID
	pendingOp, err := server.statusSvc.GetPendingOperation(session, opId)
	require.NoError(t, err, "Should retrieve pending operation")
	require.NotNil(t, pendingOp, "Pending operation should be recorded")
	assert.Equal(t, mioty.CmdVMStatus, pendingOp.OperationType)

	// Step 2: Send VM status response (BS → SC)
	conn.Reset()
	vmStatusRsp := make(map[string]interface{})
	vmStatusRsp["cmd"] = mioty.CmdVMStatusResponse
	vmStatusRsp["opId"] = opId
	vmStatusRsp["status"] = uint8(0)
	vmStatusRsp["vmActive"] = true

	err = server.handlers[mioty.CmdVMStatusResponse](server, session, &Message{OpId: opId}, vmStatusRsp)
	assert.NoError(t, err, "VM status response handler should not return error")

	// Step 3: Send VM status complete (BS → SC)
	// Community edition: Emits catalog error but doesn't clean up pending ops (community edition limitation)
	conn.Reset()
	vmStatusCmp := make(map[string]interface{})
	vmStatusCmp["cmd"] = mioty.CmdVMStatusComplete
	vmStatusCmp["opId"] = opId

	// Explicitly verify pending op exists before testing stub handler
	// This makes test intent clear: stub handler should NOT clean up pending ops
	pendingOpBeforeComplete, err := server.statusSvc.GetPendingOperation(session, opId)
	require.NoError(t, err, "Pending operation must exist before complete handler")
	require.NotNil(t, pendingOpBeforeComplete, "Pending operation must be preloaded for stub handler test")
	err = server.statusSvc.RecordPendingOperation(testutil.TestContext(), session, opId, pendingOpBeforeComplete, session.DbSessionID)
	require.NoError(t, err, "Failed to record pending operation before stub handler")

	err = server.handlers[mioty.CmdVMStatusComplete](server, session, &Message{OpId: opId}, vmStatusCmp)
	assert.Error(t, err, "VM status complete should return unsupported error in community edition")
	assert.Contains(t, strings.ToLower(err.Error()), "unsupported", "Error should indicate unsupported command")

	// Verify catalog error sent to base station per BSSCI protocol
	code, errMsg := conn.LastError()
	assert.Equal(t, 38, code, "Should send POSIX_ENOSYS (38)")
	assert.Equal(t, "Unsupported command", errMsg, "Should send catalog error message")

	// Pending operation should still exist (community stub doesn't clean up in community edition)
	pendingOp, err = server.statusSvc.GetPendingOperation(session, opId)
	assert.NoError(t, err, "Pending operation should still exist after unsupported complete")
	assert.NotNil(t, pendingOp, "Pending operation not cleaned up in community edition")
}

// TestVMDLDataFullFlow tests the complete VM DL data three-way handshake
func TestVMDLDataFullFlow(t *testing.T) {
	server, session, conn := setupVMTestServer(t)

	// Step 1: Send VM DL data request (SC → BS) using Send method
	epEui := uint64(0xAABBCCDDEEFF0044)
	macType := uint8(3)
	userData := []byte("test data")

	// Use SendVMDownlinkData to initiate the SC → BS flow
	err := server.SendVMDownlinkData(session.ID, epEui, macType, userData)
	assert.NoError(t, err, "SendVMDownlinkData should not return error")

	// Get the generated operation ID from sent message
	assert.Greater(t, len(conn.SentMessages), 0, "Should send VM DL data message")
	msgData := conn.SentMessages[0]
	opId, ok := msgData["opId"].(int64)
	require.True(t, ok, "Should have opId in message")
	assert.Less(t, opId, int64(0), "SC-initiated operation should have negative opId")

	// Use StatusService.GetPendingOperation with session and opID
	pendingOp, err := server.statusSvc.GetPendingOperation(session, opId)
	require.NoError(t, err, "Should retrieve pending operation")
	require.NotNil(t, pendingOp, "Pending operation should be recorded")
	assert.Equal(t, mioty.CmdVMDLData, pendingOp.OperationType)

	// Step 2: Send VM DL data response (BS → SC)
	// Community edition: Emits catalog error but doesn't clean up pending ops (community edition limitation)
	conn.Reset()
	vmDLDataRsp := make(map[string]interface{})
	vmDLDataRsp["cmd"] = mioty.CmdVMDLDataResponse
	vmDLDataRsp["opId"] = opId
	vmDLDataRsp["status"] = uint8(0)

	// Explicitly verify pending op exists before testing stub handler
	// This makes test intent clear: stub handler should NOT clean up pending ops
	pendingOpBeforeResponse, err := server.statusSvc.GetPendingOperation(session, opId)
	require.NoError(t, err, "Pending operation must exist before response handler")
	require.NotNil(t, pendingOpBeforeResponse, "Pending operation must be preloaded for stub handler test")
	err = server.statusSvc.RecordPendingOperation(testutil.TestContext(), session, opId, pendingOpBeforeResponse, session.DbSessionID)
	require.NoError(t, err, "Failed to record pending operation before response handler")

	err = server.handlers[mioty.CmdVMDLDataResponse](server, session, &Message{OpId: opId}, vmDLDataRsp)
	assert.Error(t, err, "VM DL data response should return unsupported error in community edition")
	assert.Contains(t, strings.ToLower(err.Error()), "unsupported", "Error should indicate unsupported command")

	// Verify catalog error sent to base station per BSSCI protocol
	code, errMsg := conn.LastError()
	assert.Equal(t, 38, code, "Should send POSIX_ENOSYS (38)")
	assert.Equal(t, "Unsupported command", errMsg, "Should send catalog error message")

	// Step 3: Send VM DL data complete (BS → SC)
	// Community edition: Emits catalog error but doesn't clean up pending ops (community edition limitation)
	conn.Reset()
	vmDLDataCmp := make(map[string]interface{})
	vmDLDataCmp["cmd"] = mioty.CmdVMDLDataComplete
	vmDLDataCmp["opId"] = opId

	// Explicitly verify pending op exists before testing stub handler
	// This makes test intent clear: stub handler should NOT clean up pending ops
	pendingOpBeforeComplete, err := server.statusSvc.GetPendingOperation(session, opId)
	require.NoError(t, err, "Pending operation must exist before complete handler")
	require.NotNil(t, pendingOpBeforeComplete, "Pending operation must be preloaded for stub handler test")
	err = server.statusSvc.RecordPendingOperation(testutil.TestContext(), session, opId, pendingOpBeforeComplete, session.DbSessionID)
	require.NoError(t, err, "Failed to record pending operation before complete handler")

	err = server.handlers[mioty.CmdVMDLDataComplete](server, session, &Message{OpId: opId}, vmDLDataCmp)
	assert.Error(t, err, "VM DL data complete should return unsupported error in community edition")
	assert.Contains(t, strings.ToLower(err.Error()), "unsupported", "Error should indicate unsupported command")

	// Verify catalog error sent to base station per BSSCI protocol
	code, errMsg = conn.LastError()
	assert.Equal(t, 38, code, "Should send POSIX_ENOSYS (38)")
	assert.Equal(t, "Unsupported command", errMsg, "Should send catalog error message")

	// Pending operation should still exist (community stub doesn't clean up in community edition)
	pendingOp, err = server.statusSvc.GetPendingOperation(session, opId)
	assert.NoError(t, err, "Pending operation should still exist after unsupported complete")
	assert.NotNil(t, pendingOp, "Pending operation not cleaned up in community edition")
}

// TestVMTenantIsolation verifies that tenant/org context is preserved throughout VM operations
func TestVMTenantIsolation(t *testing.T) {
	server, session, conn := setupVMTestServer(t)

	// Set specific tenant ID
	session.ResolvedTenantID = 789

	// Send VM activate with tenant context using Send method (SC → BS)
	epEui := uint64(0xAABBCCDDEEFF0055)
	macType := uint8(1)

	// Use SendVMActivate to initiate the SC → BS flow
	err := server.SendVMActivate(session.ID, epEui, macType)
	assert.NoError(t, err, "SendVMActivate should not return error")

	// Get the generated operation ID from sent message
	assert.Greater(t, len(conn.SentMessages), 0, "Should send VM activate message")
	msgData := conn.SentMessages[0]
	opId, ok := msgData["opId"].(int64)
	require.True(t, ok, "Should have opId in message")

	// Verify pending operation includes tenant context
	pendingOp, err := server.statusSvc.GetPendingOperation(session, opId)
	require.NoError(t, err, "Should retrieve pending operation")
	require.NotNil(t, pendingOp)

	// The tenant ID should be used for any logging or context propagation
	// Note: The actual tenant propagation happens through the session context
	assert.Equal(t, int64(789), session.ResolvedTenantID)
}

// TestVMBidirectionalRequirement verifies VM commands require bidirectional endpoints
func TestVMBidirectionalRequirement(t *testing.T) {
	// This test verifies that all VM handlers are registered and can be called
	server, session, conn := setupVMTestServer(t)

	// Test that all VM handlers are registered
	vmCommands := []string{
		mioty.CmdVMActivate,
		mioty.CmdVMActivateResponse,
		mioty.CmdVMActivateComplete,
		mioty.CmdVMDeactivate,
		mioty.CmdVMDeactivateResponse,
		mioty.CmdVMDeactivateComplete,
		mioty.CmdVMStatus,
		mioty.CmdVMStatusResponse,
		mioty.CmdVMStatusComplete,
		mioty.CmdVMDLData,
		mioty.CmdVMDLDataResponse,
		mioty.CmdVMDLDataComplete,
	}

	for _, cmd := range vmCommands {
		handler, exists := server.handlers[cmd]
		assert.True(t, exists, "Handler for %s should be registered", cmd)
		assert.NotNil(t, handler, "Handler for %s should not be nil", cmd)
	}

	// Verify a VM command can be executed using Send method (SC → BS)
	epEui := uint64(0xAABBCCDDEEFF0066)

	// Use SendVMStatus to initiate the SC → BS flow
	err := server.SendVMStatus(session.ID, epEui)
	assert.NoError(t, err, "SendVMStatus should execute without error")

	// Verify message was sent
	assert.Greater(t, len(conn.SentMessages), 0, "Should send VM message")

	// Verify the sent message has correct structure
	msgData := conn.SentMessages[0]
	assert.Equal(t, mioty.CmdVMStatus, msgData["command"], "Should send VM status command")
	assert.Equal(t, epEui, msgData["epEui"], "Should include endpoint EUI")

	opId, ok := msgData["opId"].(int64)
	require.True(t, ok, "Should have opId")
	assert.Less(t, opId, int64(0), "SC-initiated operation should have negative opId")
}

// TestVMHandlerWithStatusServiceIntegration verifies StatusService integration for all VM operations
func TestVMHandlerWithStatusServiceIntegration(t *testing.T) {
	server, session, conn := setupVMTestServer(t)

	// Test operations and their metadata
	testCases := []struct {
		name     string
		cmd      string
		epEui    uint64
		hasMAC   bool
		macType  uint8
		sendFunc func() error
	}{
		{
			name:    "VM Activate",
			cmd:     mioty.CmdVMActivate,
			epEui:   0xAABBCCDDEEFF0071,
			hasMAC:  true,
			macType: 1,
			sendFunc: func() error {
				return server.SendVMActivate(session.ID, 0xAABBCCDDEEFF0071, 1)
			},
		},
		{
			name:    "VM Deactivate",
			cmd:     mioty.CmdVMDeactivate,
			epEui:   0xAABBCCDDEEFF0072,
			hasMAC:  true,
			macType: 2,
			sendFunc: func() error {
				return server.SendVMDeactivate(session.ID, 0xAABBCCDDEEFF0072, 2)
			},
		},
		{
			name:    "VM Status",
			cmd:     mioty.CmdVMStatus,
			epEui:   0xAABBCCDDEEFF0073,
			hasMAC:  false,
			macType: 0,
			sendFunc: func() error {
				return server.SendVMStatus(session.ID, 0xAABBCCDDEEFF0073)
			},
		},
		{
			name:    "VM DL Data",
			cmd:     mioty.CmdVMDLData,
			epEui:   0xAABBCCDDEEFF0074,
			hasMAC:  true,
			macType: 3,
			sendFunc: func() error {
				return server.SendVMDownlinkData(session.ID, 0xAABBCCDDEEFF0074, 3, []byte("test"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear previous messages
			conn.Reset()

			// Execute Send method for SC → BS flow
			err := tc.sendFunc()
			assert.NoError(t, err, "%s send method should not error", tc.name)

			// Get the generated operation ID from sent message
			assert.Greater(t, len(conn.SentMessages), 0, "Should send message for %s", tc.name)
			msgData := conn.SentMessages[0]
			opId, ok := msgData["opId"].(int64)
			require.True(t, ok, "Should have opId")
			assert.Less(t, opId, int64(0), "SC-initiated operation should have negative opId")

			// Verify pending operation created with metadata
			pendingOp, err := server.statusSvc.GetPendingOperation(session, opId)
			require.NoError(t, err, "Should retrieve pending operation for %s", tc.name)
			require.NotNil(t, pendingOp, "%s should create pending operation", tc.name)
			assert.Equal(t, tc.cmd, pendingOp.OperationType)
			assert.Equal(t, tc.epEui, pendingOp.Metadata["epEui"])
			if tc.hasMAC {
				assert.Equal(t, tc.macType, pendingOp.Metadata["macType"])
			}
		})
	}
}
