package bssci_test

import (
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	bssciutil "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"

	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	repodoubles "github.com/Kiloiot/kilo-service-center/KC-Core/internal/testsupport/repodoubles"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport/teststore"

	bssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/google/uuid"
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
// newDownlinkIntegrationServer builds a server whose DownlinkService runs over
// the real per-test database, with an event store double for the audit trail.
func newDownlinkIntegrationServer(t *testing.T, db *postgres.DB) (*bssci.Server, bssci.TenantResolver) {
	t.Helper()
	testLogger := &NopLogger{}
	sessionSvc, _, statusSvc, connectionSvc, broadcaster, queueSerializer, _, tenantResolver, _ := bssci.CreateTestServices(testLogger, nil)
	downlinks := postgres.NewRepositories(db).Downlinks
	auditLogger, err := bssciservices.NewAuditLogger(bssciservices.AuditLogDeps{
		Events: &repodoubles.SystemEventStore{}, Downlinks: downlinks, Stations: &repodoubles.BaseStationRepo{},
		Clock: clock.SystemClock{}, Logger: logger.NewNop(),
	})
	require.NoError(t, err)
	reporter, err := bssciservices.NewDownlinkResultReporter(bssciservices.NewSCACIForwarder(logger.NewNop()),
		bssciservices.DownlinkResultsWithoutMQTT{}, auditLogger, bssciservices.NewBackgroundWork(), logger.NewNop())
	require.NoError(t, err)
	downlinkSvc, err := bssciservices.NewDownlinkService(bssciservices.DownlinkServiceDeps{
		Logger: logger.NewNop(), Tenants: tenantResolver, Outcomes: downlinks, Holders: downlinks,
		Results: reporter, Serializer: bssciservices.NewQueueSerializer(), Clock: clock.SystemClock{},
	})
	require.NoError(t, err)
	server := bssci.NewTestServer(testLogger, bssci.RepositoryTestStore(db), nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	return server, tenantResolver
}

func TestHandleDLDataResultIntegration(t *testing.T) {
	// Skip if no database is available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	db := setupTestDatabase(t)

	server, tenantResolver := newDownlinkIntegrationServer(t, db)

	// Create test endpoint
	epEui := bssci.TestEpEui01
	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, epEui)

	// Create a downlink message in the database
	ctx := testutil.TestContext()
	queId := int64(42)

	// Insert directly into downlink_queue table
	_, err := db.Query(ctx, `
		INSERT INTO downlink_queue (que_id, ep_eui, tenant_id, payload, status, priority, organization_id, created_at, earliest_at, bs_eui)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL, $9)
	`, queId, epEuiBytes, 1, []byte("test payload"), "queued", 5, uuid.New(), time.Now(), mioty.EUI64Bytes(bssci.TestBsEui01))
	require.NoError(t, err, "Failed to create downlink message")

	tenantResolver.RegisterQueueTenant(queId, "1")
	tenantResolver.RegisterQueueTenant(queId+1, "1")

	// Create test session
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			BaseStationEUI: bssci.TestBsEui01,
		},
		Conn: &mockConn{},
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
				results, _, err := postgres.NewRepositories(db).Downlinks.GetDownlinkResults(ctx, 1, nil, storage.DownlinkResultFilter{Status: "transmitted"}, 10, 0)
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

	// Create server with service dependencies
	testLogger := &NopLogger{}
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, nil)
	server := bssci.NewTestServer(testLogger, bssci.RepositoryTestStore(db), nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			BaseStationEUI: bssci.TestBsEui01,
		},
		Conn: &mockConn{},
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
	payload := bssciutil.FramePayload(b)
	// Decode msgpack payloads to detect error frames
	var msg map[string]interface{}
	if err := msgpack.Unmarshal(payload, &msg); err == nil {
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
func (m *mockConn) Close() error { return nil }
func (m *mockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: mockConnLocalPort}
}
func (m *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: mockConnRemotePort}
}
func (m *mockConn) SetDeadline(_ time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(_ time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(_ time.Time) error { return nil }

// setupTestDatabase provides a migrated per-test database from the shared
// container harness; any setup failure fails the test.
func setupTestDatabase(t *testing.T) *postgres.DB {
	t.Helper()
	db, cleanup := teststore.Setup(t)
	t.Cleanup(cleanup)
	return db
}

// TestDLDataResultEpEuiMismatch: a dlDataRes whose epEui does not match the
// queue row's endpoint updates nothing - the WHERE clause pins que_id, ep_eui
// and tenant_id together.
func TestDLDataResultEpEuiMismatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDatabase(t)
	server, tenantResolver := newDownlinkIntegrationServer(t, db)

	ctx := testutil.TestContext()
	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, bssci.TestEpEui01)
	const queId = int64(300)
	tenantResolver.RegisterQueueTenant(queId, "1")
	_, err := db.Query(ctx, `
		INSERT INTO downlink_queue (que_id, ep_eui, tenant_id, payload, status, priority, organization_id, created_at, earliest_at, bs_eui)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL, $9)
	`, queId, epEuiBytes, 1, []byte("mismatch payload"), "queued", 5, uuid.New(), time.Now(), mioty.EUI64Bytes(bssci.TestBsEui01))
	require.NoError(t, err)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{BaseStationEUI: bssci.TestBsEui01},
		Conn:                 &mockConn{},
	}
	msg := &bssci.Message{OpId: 400, Command: "dlDataRes"}
	mismatchedEpEui := uint64(bssci.TestEpEui02)
	err = server.CallHandleDLDataResult(session, msg, map[string]interface{}{
		"command":   "dlDataRes",
		"epEui":     mismatchedEpEui, // does not match the row
		"queId":     queId,
		"result":    "sent",
		"txTime":    int64(1234567890),
		"packetCnt": uint32(7),
	})
	require.NoError(t, err)

	rows, err := db.Query(ctx, `SELECT status, result FROM downlink_queue WHERE que_id = $1`, queId)
	require.NoError(t, err)
	defer rows.Close() //nolint:errcheck // test cleanup
	require.True(t, rows.Next(), "the row must still exist")
	var status string
	var result *string
	require.NoError(t, rows.Scan(&status, &result))
	assert.Equal(t, "queued", status, "a mismatched epEui must not change the row")
	assert.Nil(t, result, "a mismatched epEui must not record a result")
}

// TestTenantIsolation: a dlDataRes resolved to tenant 1 must not touch an
// identical queue row owned by tenant 2.
func TestTenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDatabase(t)
	server, tenantResolver := newDownlinkIntegrationServer(t, db)

	ctx := testutil.TestContext()
	_, err := db.Query(ctx, `
		INSERT INTO tenants (id, name, description, status, created_at, updated_at)
		VALUES (2, 'Isolation Tenant', 'tenant isolation test', 'active', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING`)
	require.NoError(t, err)

	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, bssci.TestEpEui01)
	const foreignQueID = int64(311)
	tenantResolver.RegisterQueueTenant(foreignQueID, "1")
	_, err = db.Query(ctx, `
		INSERT INTO downlink_queue (que_id, ep_eui, tenant_id, payload, status, priority, organization_id, created_at, earliest_at, bs_eui)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL, $9)
	`, foreignQueID, epEuiBytes, 2, []byte("tenant two payload"), "queued", 5, uuid.New(), time.Now(), mioty.EUI64Bytes(bssci.TestBsEui01))
	require.NoError(t, err)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{BaseStationEUI: bssci.TestBsEui01},
		Conn:                 &mockConn{},
	}
	msg := &bssci.Message{OpId: 401, Command: "dlDataRes"}
	isolationEpEui := uint64(bssci.TestEpEui01)
	err = server.CallHandleDLDataResult(session, msg, map[string]interface{}{
		"command":   "dlDataRes",
		"epEui":     isolationEpEui,
		"queId":     foreignQueID, // owned by tenant 2; server resolves tenant 1
		"result":    "sent",
		"txTime":    int64(1234567890),
		"packetCnt": uint32(9),
	})
	require.NoError(t, err)

	rows, err := db.Query(ctx, `SELECT status, result FROM downlink_queue WHERE que_id = $1 AND tenant_id = 2`, foreignQueID)
	require.NoError(t, err)
	defer rows.Close() //nolint:errcheck // test cleanup
	require.True(t, rows.Next(), "tenant 2's row must still exist")
	var status string
	var result *string
	require.NoError(t, rows.Scan(&status, &result))
	assert.Equal(t, "queued", status, "tenant 2's row must be untouched by tenant 1's result")
	assert.Nil(t, result)
}

// TestDLRXStatusPersistence: a valid dlRxStat persists a status row with the
// converted EUIs and the reported metrics.
func TestDLRXStatusPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDatabase(t)
	server, _ := newDownlinkIntegrationServer(t, db)

	// The handler resolves the endpoint owner tenant before persisting, so
	// the endpoint must exist.
	var seedEpEui models.EUI
	binary.BigEndian.PutUint64(seedEpEui[:], uint64(bssci.TestEpEui01))
	err := postgres.NewRepositories(db).Endpoints.Create(testutil.TestContext(), &models.EndPoint{
		Name:          "dlrx-persistence-endpoint",
		EUI:           seedEpEui,
		TenantID:      1,
		OwnerTenantID: 1,
		NwkSnKey:      make([]byte, 16),
		EPClass:       "A",
	})
	require.NoError(t, err)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{BaseStationEUI: bssci.TestBsEui01},
		Conn:                 &mockConn{},
	}
	msg := &bssci.Message{OpId: 402, Command: "dlRxStat"}
	statusEpEui := uint64(bssci.TestEpEui01)
	err = server.CallHandleDLRXStatus(session, msg, map[string]interface{}{
		"command":   "dlRxStat",
		"epEui":     statusEpEui,
		"rxTime":    int64(1234567890),
		"packetCnt": uint32(42),
		"dlRxSnr":   15.5,
		"dlRxRssi":  -85.0,
	})
	require.NoError(t, err)

	ctx := testutil.TestContext()
	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, bssci.TestEpEui01)
	statuses, _, err := postgres.NewRepositories(db).DLRXStatus.GetDLRXStatusByEndpoint(ctx, 1, epEuiBytes, 10, 0, nil, nil)
	require.NoError(t, err)
	require.Len(t, statuses, 1, "the dlRxStat must be persisted")
	assert.Equal(t, 15.5, statuses[0].DlRxSnr)
	assert.Equal(t, -85.0, statuses[0].DlRxRssi)
	assert.Equal(t, uint32(42), statuses[0].PacketCnt)
	assert.Equal(t, epEuiBytes, []byte(statuses[0].EpEui))
}
