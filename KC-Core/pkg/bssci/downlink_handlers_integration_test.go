package bssci_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"

	bssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// NopLogger implements logger.Logger interface with no-op methods for testing
type NopLogger struct{}

func (n *NopLogger) Debug(_ string, _ ...interface{})                           {}
func (n *NopLogger) Info(_ string, _ ...interface{})                            {}
func (n *NopLogger) Warn(_ string, _ ...interface{})                            {}
func (n *NopLogger) Error(_ string, _ ...interface{})                           {}
func (n *NopLogger) Fatal(_ string, _ ...interface{})                           {}
func (n *NopLogger) DebugContext(_ context.Context, _ string, _ ...interface{}) {}
func (n *NopLogger) InfoContext(_ context.Context, _ string, _ ...interface{})  {}
func (n *NopLogger) WarnContext(_ context.Context, _ string, _ ...interface{})  {}
func (n *NopLogger) ErrorContext(_ context.Context, _ string, _ ...interface{}) {}
func (n *NopLogger) FatalContext(_ context.Context, _ string, _ ...interface{}) {}
func (n *NopLogger) WithField(_ string, _ interface{}) logger.Logger            { return n }
func (n *NopLogger) WithFields(_ map[string]interface{}) logger.Logger          { return n }

// TestHandleDLDataResultIntegration tests the full dlDataRes handler with real database
func TestHandleDLDataResultIntegration(t *testing.T) {
	// Skip if no database is available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	// Create server with service dependencies
	testLogger := &NopLogger{}
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, nil)
	server := bssci.NewTestServer(testLogger, db, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	// Create test endpoint
	epEui := bssci.TestEpEui01
	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, epEui)

	// Create a downlink message in the database
	ctx := testutil.TestContext()
	queId := int64(42)

	// Insert directly into downlink_queue table
	_, err := db.Query(ctx, `
		INSERT INTO downlink_queue (que_id, ep_eui, tenant_id, payload, status, priority, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, queId, epEuiBytes, 1, []byte("test payload"), "pending", 5, time.Now())
	require.NoError(t, err, "Failed to create downlink message")

	// Create test session
	session := &bssci.Session{
		BaseStationEUI:   bssci.TestBsEui01,
		UserProvidedName: "Test BS",
		Conn:             &mockConn{},
	}

	tests := []struct {
		name        string
		data        map[string]interface{}
		expectError bool
		errorCode   int
		checkResult func(t *testing.T)
	}{
		{
			name: "valid result - sent with required fields",
			data: map[string]interface{}{
				"command":   "dlDataRes",
				"epEui":     epEui,
				"queId":     queId,
				"result":    "sent",
				"txTime":    int64(1234567890),
				"packetCnt": uint32(100),
			},
			expectError: false,
			checkResult: func(t *testing.T) {
				// Verify the downlink was updated in database
				results, _, err := db.GetDownlinkResults(ctx, "", "1", nil, "transmitted", nil, nil, 10, 0)
				require.NoError(t, err)
				require.Len(t, results, 1)
				assert.Equal(t, "sent", results[0].Result)
				assert.Equal(t, int64(1234567890), results[0].TxTime)
			},
		},
		{
			name: "valid result - expired without conditional fields",
			data: map[string]interface{}{
				"command": "dlDataRes",
				"epEui":   epEui,
				"queId":   queId + 1, // Different queue ID
				"result":  "expired",
			},
			expectError: false,
			checkResult: func(_ *testing.T) {
				// Should handle expired status correctly
			},
		},
		{
			name: "invalid enum value",
			data: map[string]interface{}{
				"command": "dlDataRes",
				"epEui":   epEui,
				"queId":   queId + 2,
				"result":  "unknown_status",
			},
			expectError: true,
			errorCode:   bssci.POSIX_EPROTO,
		},
		{
			name: "missing mandatory field epEui",
			data: map[string]interface{}{
				"command": "dlDataRes",
				"queId":   queId + 3,
				"result":  "sent",
			},
			expectError: true,
			errorCode:   bssci.POSIX_EPROTO,
		},
		{
			name: "sent without txTime",
			data: map[string]interface{}{
				"command":   "dlDataRes",
				"epEui":     epEui,
				"queId":     queId + 4,
				"result":    "sent",
				"packetCnt": uint32(100),
			},
			expectError: true,
			errorCode:   bssci.POSIX_EPROTO,
		},
		{
			name: "expired with txTime (should be rejected)",
			data: map[string]interface{}{
				"command": "dlDataRes",
				"epEui":   epEui,
				"queId":   queId + 5,
				"result":  "expired",
				"txTime":  int64(1234567890),
			},
			expectError: true,
			errorCode:   bssci.POSIX_EPROTO,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock connection
			session.Conn = &mockConn{errorToSend: tt.errorCode}

			msg := &bssci.Message{
				OpId:    int64(100 + len(tt.name)), // Unique OpId for each test
				Command: "dlDataRes",
			}

			err := server.CallHandleDLDataResult(session, msg, tt.data)

			if tt.expectError {
				// Handler returns nil even on validation failure
				assert.NoError(t, err, "Handler returns nil after sending protocol error")
				// Check that an error was sent to the base station
				mockConn := session.Conn.(*mockConn)
				assert.True(t, mockConn.errorSent, "Expected error to be sent")
				assert.Equal(t, tt.errorCode, mockConn.lastErrorCode, "Error code mismatch")
			} else {
				assert.NoError(t, err, "Handler should not return error")
				if tt.checkResult != nil {
					tt.checkResult(t)
				}
			}
		})
	}
}

// TestHandleDLDataResultCanonicalUnmarshalling tests the new canonical unmarshalling
func TestHandleDLDataResultCanonicalUnmarshalling(t *testing.T) {
	_ = &NopLogger{} // Logger for future use

	// Create properly formatted msgpack data
	dlResult := mioty.DLDataResult{
		BaseMessage: mioty.BaseMessage{
			CommandType: mioty.CmdDLDataResult,
			OpId:        123,
		},
		EpEui:     bssci.TestEpEui01,
		QueId:     42,
		Result:    "sent",
		TxTime:    bssci.Int64Ptr(1234567890),
		PacketCnt: bssci.Uint32Ptr(100),
	}

	// Marshal to msgpack
	msgpackData, err := msgpack.Marshal(dlResult)
	require.NoError(t, err)

	// Unmarshal to map (simulating what comes from the wire)
	var data map[string]interface{}
	err = msgpack.Unmarshal(msgpackData, &data)
	require.NoError(t, err)

	// Test that we can unmarshal back to the canonical type
	msgpackData2, err := msgpack.Marshal(data)
	require.NoError(t, err)

	var dlResult2 mioty.DLDataResult
	err = msgpack.Unmarshal(msgpackData2, &dlResult2)
	require.NoError(t, err)

	// Verify all fields are preserved
	assert.Equal(t, dlResult.EpEui, dlResult2.EpEui)
	assert.Equal(t, dlResult.QueId, dlResult2.QueId)
	assert.Equal(t, dlResult.Result, dlResult2.Result)
	assert.NotNil(t, dlResult2.TxTime)
	assert.Equal(t, *dlResult.TxTime, *dlResult2.TxTime)
	assert.NotNil(t, dlResult2.PacketCnt)
	assert.Equal(t, *dlResult.PacketCnt, *dlResult2.PacketCnt)
}

// TestHandleDLRXStatusIntegration tests the full dlRxStat handler with database
func TestHandleDLRXStatusIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	// Create server with service dependencies
	testLogger := &NopLogger{}
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, nil)
	server := bssci.NewTestServer(testLogger, db, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	session := &bssci.Session{
		BaseStationEUI:   bssci.TestBsEui01,
		UserProvidedName: "Test BS",
		Conn:             &mockConn{},
	}

	tests := []struct {
		name        string
		data        map[string]interface{}
		expectError bool
		errorCode   int
	}{
		{
			name: "valid DL RX status",
			data: map[string]interface{}{
				"command":   "dlRxStat",
				"epEui":     bssci.TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"dlRxSnr":   15.5,
				"dlRxRssi":  -85.0,
			},
			expectError: false,
		},
		{
			name: "missing mandatory field dlRxSnr",
			data: map[string]interface{}{
				"command":   "dlRxStat",
				"epEui":     bssci.TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"dlRxRssi":  -85.0,
			},
			expectError: true,
			errorCode:   bssci.POSIX_EPROTO,
		},
		{
			name: "missing mandatory field dlRxRssi",
			data: map[string]interface{}{
				"command":   "dlRxStat",
				"epEui":     bssci.TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"dlRxSnr":   15.5,
			},
			expectError: true,
			errorCode:   bssci.POSIX_EPROTO,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session.Conn = &mockConn{errorToSend: tt.errorCode}

			msg := &bssci.Message{
				OpId:    int64(200 + len(tt.name)),
				Command: "dlRxStat",
			}

			err := server.CallHandleDLRXStatus(session, msg, tt.data)

			if tt.expectError {
				// Handler returns nil even on validation failure
				assert.NoError(t, err, "Handler returns nil after sending protocol error")
				mockConn := session.Conn.(*mockConn)
				assert.True(t, mockConn.errorSent, "Expected error to be sent")
				assert.Equal(t, tt.errorCode, mockConn.lastErrorCode, "Error code mismatch")
			} else {
				assert.NoError(t, err, "Handler should not return error")
			}
		})
	}
}

// Mock connection for testing
type mockConn struct {
	errorSent     bool
	lastErrorCode int
	errorToSend   int
	data          []byte
}

func (m *mockConn) Read(_ []byte) (n int, err error) { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error) {
	m.data = b
	// Skip BSSCI headers by checking protocol identifier
	// Headers are exactly 12 bytes starting with "MIOTYB01"
	if len(b) == 12 && bytes.HasPrefix(b, mioty.MIOTYFrameIdentifier[:]) {
		return len(b), nil
	}
	// Decode msgpack payloads to detect error frames
	var msg map[string]interface{}
	if err := msgpack.Unmarshal(b, &msg); err == nil {
		if cmd, ok := msg["command"].(string); ok && cmd == "error" {
			m.errorSent = true
			// Msgpack may encode integers as int8, int16, int32, int64, uint8, etc.
			switch code := msg["code"].(type) {
			case int:
				m.lastErrorCode = code
			case int8:
				m.lastErrorCode = int(code)
			case int16:
				m.lastErrorCode = int(code)
			case int32:
				m.lastErrorCode = int(code)
			case int64:
				m.lastErrorCode = int(code)
			case uint8:
				m.lastErrorCode = int(code)
			case uint16:
				m.lastErrorCode = int(code)
			case uint32:
				m.lastErrorCode = int(code)
			case uint64:
				m.lastErrorCode = int(code)
			}
		}
	}
	return len(b), nil
}
func (m *mockConn) Close() error        { return nil }
func (m *mockConn) LocalAddr() net.Addr { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5000} }
func (m *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
}
func (m *mockConn) SetDeadline(_ time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(_ time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(_ time.Time) error { return nil }

// Helper to setup test database
func setupTestDatabase(t *testing.T) *postgres.DB {
	// Connect to test database
	cfg := config.StorageConfig{
		Host:     "localhost",
		Port:     5433,
		Database: "kilocenter_test",
		Username: "kilocenter",
		Password: "changeme",
		SSLMode:  "disable",
	}

	db, err := postgres.New(cfg)
	if err != nil {
		t.Skipf("Cannot connect to test database: %v", err)
	}

	// Clean existing test data
	ctx := testutil.TestContext()
	_, _ = db.Query(ctx, "DELETE FROM downlink_queue WHERE tenant_id IN (1, 2, 3)")
	_, _ = db.Query(ctx, "DELETE FROM bssci_pending_operations WHERE tenant_id IN (1, 2, 3)")

	return db
}

func cleanupTestDatabase(t *testing.T, db *postgres.DB) {
	if db != nil {
		ctx := testutil.TestContext()
		// Clean up test data
		_, _ = db.Query(ctx, "DELETE FROM downlink_queue WHERE tenant_id IN (1, 2, 3)")
		_, _ = db.Query(ctx, "DELETE FROM bssci_pending_operations WHERE tenant_id IN (1, 2, 3)")
		if err := db.Close(); err != nil {
			t.Logf("Failed to close database: %v", err)
		}
	}
}
