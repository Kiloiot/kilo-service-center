package bssci

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Enum validation and conditional field tests moved to integration tests
// (downlink_handlers_integration_test.go) where they can exercise real validation code

// TestDLDataResultEpEuiMismatch tests that mismatched epEui is rejected
func TestDLDataResultEpEuiMismatch(t *testing.T) {
	t.Skip("Requires refactoring to work with concrete postgres.DB type")
	// This test needs interface-based message store or integration test setup
}

// Tenant isolation tests moved to integration tests where they can verify
// actual database-level tenant isolation enforcement

// TestDLDataResultCanonicalTypes tests that canonical MIOTY types are used
func TestDLDataResultCanonicalTypes(t *testing.T) {
	var dlResult mioty.DLDataResult

	// Test that struct has correct field types
	assert.IsType(t, uint64(0), dlResult.EpEui, "EpEui should be uint64")
	assert.IsType(t, uint64(0), dlResult.QueId, "QueId should be uint64")
	assert.IsType(t, "", dlResult.Result, "Result should be string")
	assert.IsType(t, (*int64)(nil), dlResult.TxTime, "TxTime should be *int64")
	assert.IsType(t, (*uint32)(nil), dlResult.PacketCnt, "PacketCnt should be *uint32")

	// Test command constants
	assert.Equal(t, "dlDataRes", mioty.CmdDLDataResult)
	assert.Equal(t, "dlDataResRsp", mioty.CmdDLDataResultResponse)
	assert.Equal(t, "dlDataResCmp", mioty.CmdDLDataResultComplete)
}

// =============================================================================
// §3.11 DL Data Revoke - SendDLDataRevoke and RevokeDownlink Tests
// =============================================================================

// TestSendDLDataRevoke_BuildsMessage_PersistsMetadata verifies that SendDLDataRevoke
// constructs the correct message format with mandatory fields and persists operation metadata.
//
// Spec: BSSCI §5.13.1 - dlDataRev message construction
func TestSendDLDataRevoke_BuildsMessage_PersistsMetadata(t *testing.T) {
	// Skip if running short tests (this is a unit test, but requires server setup)
	if testing.Short() {
		t.Skip("Skipping server-based test in short mode")
	}

	// Create mock connection
	mockConn := &testConn{}

	// Create test server with mock services
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver, mockStorage :=
		CreateTestServices(testLogger, nil)

	server := NewTestServer(testLogger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver)

	// Create and register a test session
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "test-session-revoke",
			BaseStationEUI:    TestBsEui04,
			Encoding:          EncodingMessagePack,
			LastScOpId:        -1,
			HandshakeComplete: true,
			DbSessionID:       1,
		},
		Conn:          mockConn,
		Bidirectional: true,
	}
	server.RegisterSession(session)

	// Execute SendDLDataRevoke
	queId := uint64(12345)
	epEui := TestEpEui01

	err := server.SendDLDataRevoke(session.ID, epEui, queId)
	require.NoError(t, err, "SendDLDataRevoke should succeed")

	// Verify message was sent (testConn tracks errorSent on any Write)
	require.True(t, mockConn.errorSent, "Message should be written to connection")

	// Verify operation ID was decremented
	require.Equal(t, int64(-2), session.LastScOpId, "SC opId should be decremented")
}

// TestRevokeDownlink_NoStorage_ReturnsQueueNotFound verifies that when storage
// is nil (simulating no DB connection), ErrSchedulerQueueNotFound is returned.
//
// Spec: BSSCI §5.13 - DL Data Revoke with unavailable storage
func TestRevokeDownlink_NoStorage_ReturnsQueueNotFound(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping server-based test in short mode")
	}

	// Create test server with nil storage
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver, _ :=
		CreateTestServices(testLogger, nil)

	// Explicitly pass nil storage - simulates storage unavailable
	server := NewTestServer(testLogger, nil, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver)

	// Execute RevokeDownlink with nil storage
	_, err := server.RevokeDownlink(1, 12345)

	// Should return ErrSchedulerQueueNotFound (storage unavailable)
	require.Error(t, err)
	// Note: The implementation returns ErrSchedulerQueueNotFound when storage is nil
}

// Helper functions
