package postgres

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DownlinkInsertParams contains parameters for inserting test downlinks.
// Use with insertDownlink() for consistent test data.
type DownlinkInsertParams struct {
	EpEUI    uint64
	TenantID int64
	Payload  []byte
	Priority float32
	Status   string
	UserData [][]byte // For multi-packet tests (stored as JSON in user_data column)

	// Optional fields
	QueID            int64
	BsEUI            uint64
	TxTime           *int64
	Format           uint8
	NoDefaultPayload bool // If true, skip default payload (for backward-compat tests)

	// SCACI §3.10 fields (migration 000089, 000090)
	DlRxStatQry    *bool   // DL RX status query requested (nullable)
	OrganizationID *string // Organization UUID for multi-tenant audit (nullable)
}

// insertDownlink inserts a test downlink into downlink_queue.
// Returns the que_id (auto-generated if not provided).
func insertDownlink(t *testing.T, db *DB, p DownlinkInsertParams) int64 {
	t.Helper()

	// Convert epEUI to bytea
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, p.EpEUI)

	// Set defaults
	if p.Status == "" {
		p.Status = bssci.DLQueueStatusPending
	}
	if p.Payload == nil && !p.NoDefaultPayload {
		p.Payload = []byte{0x01, 0x02, 0x03} // Default payload
	}

	// Prepare user_data JSON if provided, or use SQL NULL for empty
	var userDataJSON interface{}
	if len(p.UserData) > 0 {
		dlQueue := mioty.DLDataQueue{
			UserData: p.UserData,
		}
		jsonBytes, err := json.Marshal(dlQueue)
		require.NoError(t, err, "Failed to marshal user_data JSON")
		userDataJSON = jsonBytes
	} else {
		// Pass nil interface{} for SQL NULL instead of nil []byte
		userDataJSON = nil
	}

	// Insert with RETURNING que_id
	// que_id now has DEFAULT nextval('downlink_queue_que_id_seq') from migration 085.
	// earliest_at and latest_at MUST both be NULL or both NOT NULL per downlink_queue_schedule_order constraint.
	// For test inserts, we set both to NULL to satisfy the constraint.
	var queID int64
	query := `
		INSERT INTO downlink_queue (
			ep_eui, tenant_id, payload, priority, status, user_data, format,
			dl_rx_stat_qry, organization_id,
			earliest_at, latest_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,
			NULL, NULL, NOW(), NOW())
		RETURNING que_id
	`
	err := db.conn.QueryRow(query,
		epEUIBytes, p.TenantID, p.Payload, p.Priority, p.Status, userDataJSON, p.Format,
		p.DlRxStatQry, p.OrganizationID,
	).Scan(&queID)
	require.NoError(t, err, "Failed to insert test downlink")

	return queID
}

// TestReserveNextPendingDownlink_SelectsHighestPriority verifies priority ordering.
// Downlinks should be returned ORDER BY priority DESC, created_at ASC.
func TestReserveNextPendingDownlink_SelectsHighestPriority(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	// Create test tenant
	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	// Initialize logger and DB wrapper
	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	// Insert endpoints (required for FK if present)
	epEUI := uint64(0x0102030405060708)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	// Insert endpoint
	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-Priority",
		TenantID: 100,
	})

	// Insert downlinks with different priorities (low first, high second)
	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Priority: 1.0, // Low priority
		Payload:  []byte{0x01},
	})
	time.Sleep(10 * time.Millisecond) // Ensure different created_at

	queIDHigh := insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Priority: 10.0, // High priority
		Payload:  []byte{0x02},
	})

	// Begin transaction and reserve
	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	bsEUI := uint64(0xAABBCCDDEEFF0011)
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI, nil)
	require.NoError(t, err)
	require.NotNil(t, dl, "Expected downlink to be returned")

	// Should return the high-priority one
	assert.Equal(t, queIDHigh, dl.QueID, "Should select highest priority downlink")
	assert.Equal(t, float32(10.0), dl.Priority, "Priority should be 10.0")
}

// TestReserveNextPendingDownlink_ReturnsNilWhenNoPending verifies nil return.
// When no pending downlinks exist, returns (nil, nil).
func TestReserveNextPendingDownlink_ReturnsNilWhenNoPending(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	// Insert endpoint but NO downlinks
	epEUI := uint64(0x0102030405060709)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-NoPending",
		TenantID: 100,
	})

	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	bsEUI := uint64(0xAABBCCDDEEFF0011)
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI, nil)

	assert.NoError(t, err, "Should not return error when no pending")
	assert.Nil(t, dl, "Should return nil when no pending downlinks")
}

// TestReserveNextPendingDownlink_TenantIsolation verifies tenant isolation.
// Downlinks for different tenant should not be visible.
func TestReserveNextPendingDownlink_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")
	createTestTenant(t, sqlxDB, 200, "TestTenant200")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	epEUI := uint64(0x0102030405060710)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	// Insert endpoint for tenant 100 only
	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-TenantIso",
		TenantID: 100,
	})

	// Insert downlink for tenant 100
	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Priority: 5.0,
	})

	// Try to reserve as tenant 200 - should return nil
	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	bsEUI := uint64(0xAABBCCDDEEFF0011)
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 200, epEUIBytes, bsEUI, nil)

	assert.NoError(t, err, "Should not return error")
	assert.Nil(t, dl, "Wrong tenant should not see downlinks")
}

// TestReserveNextPendingDownlink_OnlyPendingStatus verifies status filtering.
// Only 'pending' status should be selected; reserved/queued/sent are ignored.
func TestReserveNextPendingDownlink_OnlyPendingStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	epEUI := uint64(0x0102030405060711)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-StatusFilter",
		TenantID: 100,
	})

	// Insert downlinks with various statuses (non-pending)
	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Priority: 10.0,
		Status:   bssci.DLQueueStatusReserved,
	})
	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Priority: 9.0,
		Status:   bssci.DLQueueStatusQueued,
	})
	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Priority: 8.0,
		Status:   bssci.DLQueueStatusTransmitted,
	})

	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	bsEUI := uint64(0xAABBCCDDEEFF0011)
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI, nil)

	assert.NoError(t, err)
	assert.Nil(t, dl, "Should not select non-pending downlinks")
}

// TestReserveNextPendingDownlink_UserDataUnmarshaled is a regression test for Finding 1.
// Verifies that user_data JSON is properly unmarshaled into dl.UserData.
func TestReserveNextPendingDownlink_UserDataUnmarshaled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	epEUI := uint64(0x0102030405060712)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-UserData",
		TenantID: 100,
	})

	// Insert downlink with multi-packet user_data (no explicit Payload)
	// This tests backward-compat logic: when payload column is empty,
	// ReserveNextPendingDownlink should set Payload = UserData[0]
	expectedUserData := [][]byte{
		{0x11, 0x22, 0x33},
		{0x44, 0x55, 0x66},
		{0x77, 0x88, 0x99},
	}
	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:            epEUI,
		TenantID:         100,
		Priority:         5.0,
		NoDefaultPayload: true, // Explicitly skip default payload for backward-compat test
		UserData:         expectedUserData,
	})

	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	bsEUI := uint64(0xAABBCCDDEEFF0011)
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI, nil)

	require.NoError(t, err)
	require.NotNil(t, dl, "Expected downlink to be returned")

	// CRITICAL: Verify UserData is populated (this was the bug)
	require.Len(t, dl.UserData, 3, "UserData should have 3 payloads")
	assert.Equal(t, expectedUserData[0], dl.UserData[0], "First payload should match")
	assert.Equal(t, expectedUserData[1], dl.UserData[1], "Second payload should match")
	assert.Equal(t, expectedUserData[2], dl.UserData[2], "Third payload should match")

	// Verify backward-compat Payload is set to first UserData entry
	assert.Equal(t, expectedUserData[0], dl.Payload, "Payload should equal first UserData entry")
}

// TestReserveNextPendingDownlink_SchemaAcceptsReserved verifies schema accepts 'reserved' status.
// INSERT with status='reserved' should succeed after migration 000081.
func TestReserveNextPendingDownlink_SchemaAcceptsReserved(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	epEUI := uint64(0x0102030405060713)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-SchemaReserved",
		TenantID: 100,
	})

	// Direct insert with 'reserved' status should succeed
	queID := insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Priority: 5.0,
		Status:   bssci.DLQueueStatusReserved,
	})

	assert.Greater(t, queID, int64(0), "Reserved status should be accepted by schema")

	// Verify via direct query
	var status string
	err := sqlxDB.Get(&status, `SELECT status FROM downlink_queue WHERE que_id = $1`, queID)
	require.NoError(t, err)
	assert.Equal(t, bssci.DLQueueStatusReserved, status)
}

// TestMarkReservedAsQueued_TransitionsStatus verifies reserved→queued transition.
// Should update status, transmission_time, and bs_eui.
func TestMarkReservedAsQueued_TransitionsStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	epEUI := uint64(0x0102030405060714)
	bsEUI := uint64(0xAABBCCDDEEFF0022)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-Transition",
		TenantID: 100,
	})

	// Insert as reserved (simulating after ReserveNextPendingDownlink)
	queID := insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Priority: 5.0,
		Status:   bssci.DLQueueStatusReserved,
	})

	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)

	txTime := time.Now().UnixNano()
	err = tx.MIOTYDownlinks().MarkReservedAsQueued(t.Context(), uint64(queID), 100, bsEUI, txTime, nil, nil)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify transition
	var status string
	var storedTxTime int64
	var storedBsEUI []byte
	err = sqlxDB.QueryRow(`
		SELECT status, tx_time, bs_eui FROM downlink_queue WHERE que_id = $1
	`, queID).Scan(&status, &storedTxTime, &storedBsEUI)
	require.NoError(t, err)

	assert.Equal(t, bssci.DLQueueStatusQueued, status)
	assert.Equal(t, txTime, storedTxTime)

	// Verify bs_eui bytea matches
	expectedBsEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(expectedBsEUIBytes, bsEUI)
	assert.Equal(t, expectedBsEUIBytes, storedBsEUI)
}

// TestMarkReservedAsQueued_NullPacketCnt verifies NULL packet_cnt handling.
// packetCnt=nil should result in NULL in database.
func TestMarkReservedAsQueued_NullPacketCnt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	epEUI := uint64(0x0102030405060715)
	bsEUI := uint64(0xAABBCCDDEEFF0033)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-NullPktCnt",
		TenantID: 100,
	})

	queID := insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Status:   bssci.DLQueueStatusReserved,
	})

	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)

	txTime := time.Now().UnixNano()
	// Pass nil for packetCnt
	err = tx.MIOTYDownlinks().MarkReservedAsQueued(t.Context(), uint64(queID), 100, bsEUI, txTime, nil, nil)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify transmission_packet_cnt is NULL
	var packetCnt *int64
	err = sqlxDB.QueryRow(`
		SELECT transmission_packet_cnt FROM downlink_queue WHERE que_id = $1
	`, queID).Scan(&packetCnt)
	require.NoError(t, err)

	assert.Nil(t, packetCnt, "transmission_packet_cnt should be NULL when nil passed")
}

// TestMarkReservedAsQueued_WrongStatus verifies error when status is not 'reserved'.
// Should return ErrDownlinkAlreadyReserved if row not in reserved state.
func TestMarkReservedAsQueued_WrongStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	epEUI := uint64(0x0102030405060716)
	bsEUI := uint64(0xAABBCCDDEEFF0044)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-WrongStatus",
		TenantID: 100,
	})

	// Insert as 'pending' (not reserved)
	queID := insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Status:   bssci.DLQueueStatusPending,
	})

	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	txTime := time.Now().UnixNano()
	err = tx.MIOTYDownlinks().MarkReservedAsQueued(t.Context(), uint64(queID), 100, bsEUI, txTime, nil, nil)

	// Should return error because status is not 'reserved'
	assert.ErrorIs(t, err, ErrDownlinkAlreadyReserved, "Should error when status is not 'reserved'")
}

// TestReserveNextPendingDownlink_ConvertsEPEUIToHex verifies EPEUI hex conversion.
// The storage.DownlinkMessage.EPEUI should be hex-encoded string.
func TestReserveNextPendingDownlink_ConvertsEPEUIToHex(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	epEUI := uint64(0xDEADBEEFCAFEBABE)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-HexConvert",
		TenantID: 100,
	})

	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Priority: 5.0,
	})

	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	bsEUI := uint64(0xAABBCCDDEEFF0055)
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI, nil)

	require.NoError(t, err)
	require.NotNil(t, dl)

	expectedHex := hex.EncodeToString(epEUIBytes)
	assert.Equal(t, expectedHex, dl.EPEUI, "EPEUI should be hex-encoded")
}

// NOTE: SKIP LOCKED concurrency tests are intentionally not included.
// Testing concurrent FOR UPDATE SKIP LOCKED behavior requires complex
// multi-goroutine setup with precise timing and is better suited for
// integration tests with actual concurrent connections.
// See: https://www.postgresql.org/docs/current/sql-select.html#SQL-FOR-UPDATE-SHARE

// =============================================================================
// SCACI §3.10 dlDataQue Backend Tests
// Spec: SCACI §3.10.1 - dlRxStatQry optional field
// Spec: SCACI §3.10 - organization_id for multi-tenant audit trail
// =============================================================================

// TestReserveNextPendingDownlink_DlRxStatQryTrue verifies dl_rx_stat_qry=true is returned.
// When the column is TRUE, ReserveNextPendingDownlink should return DlRxStatQry=true.
//
// Spec: SCACI §3.10.1 - dlRxStatQry is optional field for querying DL RX status
func TestReserveNextPendingDownlink_DlRxStatQryTrue(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	epEUI := uint64(0x0102030405060720)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-DlRxStatQryTrue",
		TenantID: 100,
	})

	dlRxStatQryTrue := true
	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:       epEUI,
		TenantID:    100,
		Priority:    5.0,
		DlRxStatQry: &dlRxStatQryTrue,
	})

	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	bsEUI := uint64(0xAABBCCDDEEFF0077)
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI, nil)

	require.NoError(t, err)
	require.NotNil(t, dl)

	// CRITICAL: Verify DlRxStatQry is populated as true
	// Note: DownlinkMessage.DlRxStatQry is bool (not pointer) - DB NULL → false
	assert.True(t, dl.DlRxStatQry, "DlRxStatQry should be true when column is TRUE")
}

// TestReserveNextPendingDownlink_DlRxStatQryFalse verifies dl_rx_stat_qry=false is returned.
// When the column is FALSE, ReserveNextPendingDownlink should return DlRxStatQry=false.
//
// Spec: SCACI §3.10.1 - dlRxStatQry optional field defaults to false/nil
func TestReserveNextPendingDownlink_DlRxStatQryFalse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	epEUI := uint64(0x0102030405060721)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-DlRxStatQryFalse",
		TenantID: 100,
	})

	dlRxStatQryFalse := false
	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:       epEUI,
		TenantID:    100,
		Priority:    5.0,
		DlRxStatQry: &dlRxStatQryFalse,
	})

	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	bsEUI := uint64(0xAABBCCDDEEFF0078)
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI, nil)

	require.NoError(t, err)
	require.NotNil(t, dl)

	// DlRxStatQry should be false when column is FALSE
	// Note: DownlinkMessage.DlRxStatQry is bool (not pointer)
	assert.False(t, dl.DlRxStatQry, "DlRxStatQry should be false when column is FALSE")
}

// TestReserveNextPendingDownlink_DlRxStatQryNull verifies dl_rx_stat_qry=NULL is returned as nil.
// When the column is NULL, ReserveNextPendingDownlink should return DlRxStatQry=nil.
//
// Spec: SCACI §3.10.1 - dlRxStatQry is optional; NULL means not requested
func TestReserveNextPendingDownlink_DlRxStatQryNull(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	epEUI := uint64(0x0102030405060722)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-DlRxStatQryNull",
		TenantID: 100,
	})

	// DlRxStatQry = nil means column will be NULL
	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:       epEUI,
		TenantID:    100,
		Priority:    5.0,
		DlRxStatQry: nil,
	})

	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	bsEUI := uint64(0xAABBCCDDEEFF0079)
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI, nil)

	require.NoError(t, err)
	require.NotNil(t, dl)

	// DlRxStatQry should be false when column is NULL (bool default)
	// Note: DownlinkMessage.DlRxStatQry is bool (not pointer) - DB NULL → false
	assert.False(t, dl.DlRxStatQry, "DlRxStatQry should be false when column is NULL")
}

// TestReserveNextPendingDownlink_OrganizationID verifies organization_id is returned.
// When the column has a UUID, ReserveNextPendingDownlink should return OrganizationID.
//
// Spec: SCACI §3.10 - organization_id for multi-tenant audit trail
func TestReserveNextPendingDownlink_OrganizationID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	epEUI := uint64(0x0102030405060723)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-OrgID",
		TenantID: 100,
	})

	testOrgID := "550e8400-e29b-41d4-a716-446655440000"
	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:          epEUI,
		TenantID:       100,
		Priority:       5.0,
		OrganizationID: &testOrgID,
	})

	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	bsEUI := uint64(0xAABBCCDDEEFF0080)
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI, nil)

	require.NoError(t, err)
	require.NotNil(t, dl)

	// CRITICAL: Verify OrganizationID is populated
	require.NotNil(t, dl.OrganizationID, "OrganizationID should not be nil when column has UUID")
	assert.Equal(t, testOrgID, dl.OrganizationID.String(), "OrganizationID should match inserted value")
}

// TestReserveNextPendingDownlink_OrganizationIDNull verifies organization_id=NULL is handled.
// When the column is NULL, ReserveNextPendingDownlink should return OrganizationID=nil.
//
// Spec: SCACI §3.10 - organization_id is nullable for backward compatibility
func TestReserveNextPendingDownlink_OrganizationIDNull(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	epEUI := uint64(0x0102030405060724)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-OrgIDNull",
		TenantID: 100,
	})

	// OrganizationID = nil means column will be NULL
	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:          epEUI,
		TenantID:       100,
		Priority:       5.0,
		OrganizationID: nil,
	})

	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	bsEUI := uint64(0xAABBCCDDEEFF0081)
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI, nil)

	require.NoError(t, err)
	require.NotNil(t, dl)

	// OrganizationID should be nil when column is NULL
	assert.Nil(t, dl.OrganizationID, "OrganizationID should be nil when column is NULL")
}

// TestReserveNextPendingDownlink_BothDlRxStatQryAndOrgID verifies both fields are returned together.
// When both dl_rx_stat_qry and organization_id are set, both should be returned correctly.
//
// Spec: SCACI §3.10 - Complete field plumbing verification
func TestReserveNextPendingDownlink_BothDlRxStatQryAndOrgID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	epEUI := uint64(0x0102030405060725)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-BothFields",
		TenantID: 100,
	})

	dlRxStatQryTrue := true
	testOrgID := "660e8400-e29b-41d4-a716-446655440001"
	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:          epEUI,
		TenantID:       100,
		Priority:       5.0,
		DlRxStatQry:    &dlRxStatQryTrue,
		OrganizationID: &testOrgID,
	})

	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	bsEUI := uint64(0xAABBCCDDEEFF0082)
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI, nil)

	require.NoError(t, err)
	require.NotNil(t, dl)

	// CRITICAL: Verify both fields are populated correctly
	// Note: DownlinkMessage.DlRxStatQry is bool (not pointer)
	assert.True(t, dl.DlRxStatQry, "DlRxStatQry should be true")

	require.NotNil(t, dl.OrganizationID, "OrganizationID should not be nil")
	assert.Equal(t, testOrgID, dl.OrganizationID.String(), "OrganizationID should match")
}

// TestReserveNextPendingDownlink_OrgFilterIsolation verifies organization filter isolation.
// When orgID filter is provided, downlinks from other organizations are not returned.
// When orgID filter is nil, downlinks from any organization are returned.
//
// Spec: SCACI §3.10 R2 - Organization filter isolation
func TestReserveNextPendingDownlink_OrgFilterIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, log: log}

	epEUI := uint64(0x0102030405060730)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-OrgFilter",
		TenantID: 100,
	})

	// Insert downlinks for two different organizations
	org1ID := "550e8400-e29b-41d4-a716-446655440001"
	org2ID := "550e8400-e29b-41d4-a716-446655440002"

	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:          epEUI,
		TenantID:       100,
		Priority:       5.0, // Higher priority
		OrganizationID: &org1ID,
	})
	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:          epEUI,
		TenantID:       100,
		Priority:       10.0, // Even higher priority
		OrganizationID: &org2ID,
	})

	// Test 1: With org1 filter, should only get org1's downlink
	t.Run("FilteredByOrg1", func(t *testing.T) {
		tx, err := db.BeginTx(t.Context())
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()

		org1UUID, _ := parseUUID(org1ID)
		bsEUI := uint64(0xAABBCCDDEEFF0090)
		dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI, &org1UUID)

		require.NoError(t, err)
		require.NotNil(t, dl, "Should find org1 downlink")
		require.NotNil(t, dl.OrganizationID)
		assert.Equal(t, org1ID, dl.OrganizationID.String(), "Should be org1's downlink")
	})

	// Test 2: With org2 filter, should only get org2's downlink
	t.Run("FilteredByOrg2", func(t *testing.T) {
		tx, err := db.BeginTx(t.Context())
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()

		org2UUID, _ := parseUUID(org2ID)
		bsEUI := uint64(0xAABBCCDDEEFF0091)
		dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI, &org2UUID)

		require.NoError(t, err)
		require.NotNil(t, dl, "Should find org2 downlink")
		require.NotNil(t, dl.OrganizationID)
		assert.Equal(t, org2ID, dl.OrganizationID.String(), "Should be org2's downlink")
	})

	// Test 3: With nil org filter, should get highest priority (org2's downlink)
	t.Run("NoFilterReturnsHighestPriority", func(t *testing.T) {
		tx, err := db.BeginTx(t.Context())
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()

		bsEUI := uint64(0xAABBCCDDEEFF0092)
		dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI, nil)

		require.NoError(t, err)
		require.NotNil(t, dl, "Should find downlink")
		require.NotNil(t, dl.OrganizationID)
		assert.Equal(t, org2ID, dl.OrganizationID.String(), "Should be highest priority (org2)")
	})

	// Test 4: With unknown org filter, should return nil (no match)
	t.Run("UnknownOrgReturnsNil", func(t *testing.T) {
		tx, err := db.BeginTx(t.Context())
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()

		unknownOrg, _ := parseUUID("550e8400-e29b-41d4-a716-446655440999")
		bsEUI := uint64(0xAABBCCDDEEFF0093)
		dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI, &unknownOrg)

		require.NoError(t, err)
		assert.Nil(t, dl, "Should not find downlink for unknown org")
	})
}

// parseUUID is a test helper that parses a UUID string
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// =============================================================================
// §3.11 DL Data Revoke - GetDownlinkByPacketCnt Tests
// =============================================================================

// DownlinkWithPacketCntParams extends DownlinkInsertParams for packet counter tests
type DownlinkWithPacketCntParams struct {
	DownlinkInsertParams
	CntDepend bool    // Counter-dependent flag
	PacketCnt []int64 // Packet counter array
}

// insertDownlinkWithPacketCnt inserts a test downlink with packet counter fields.
// Required for testing SCACI §3.11 GetDownlinkByPacketCnt functionality.
func insertDownlinkWithPacketCnt(t *testing.T, db *DB, p DownlinkWithPacketCntParams) int64 {
	t.Helper()

	// Convert epEUI to bytea
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, p.EpEUI)

	// Set defaults
	if p.Status == "" {
		p.Status = bssci.DLQueueStatusPending
	}
	if p.Payload == nil && !p.NoDefaultPayload {
		p.Payload = []byte{0x01, 0x02, 0x03}
	}

	var queID int64
	query := `
		INSERT INTO downlink_queue (
			ep_eui, tenant_id, payload, priority, status, cnt_depend, packet_cnt,
			earliest_at, latest_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7,
			NULL, NULL, NOW(), NOW())
		RETURNING que_id
	`

	var packetCntArg interface{}
	if len(p.PacketCnt) > 0 {
		packetCntArg = pq.Int64Array(p.PacketCnt)
	} else {
		packetCntArg = nil
	}

	err := db.conn.QueryRow(query,
		epEUIBytes, p.TenantID, p.Payload, p.Priority, p.Status, p.CntDepend, packetCntArg,
	).Scan(&queID)
	require.NoError(t, err, "Failed to insert test downlink with packet counter")

	return queID
}

// TestGetDownlinkByPacketCnt_CounterDependent_ReturnsMatch verifies that
// counter-dependent downlinks (cnt_depend=true) are found when packetCnt matches.
//
// Spec: SCACI §3.11.1 - packetCnt resolution for counter-dependent queues
func TestGetDownlinkByPacketCnt_CounterDependent_ReturnsMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	// Insert endpoint
	epEUI := uint64(0x0102030405060708)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-CntDepend",
		TenantID: 100,
	})

	// Insert counter-dependent downlink with packetCnt = [100, 101, 102]
	insertDownlinkWithPacketCnt(t, db, DownlinkWithPacketCntParams{
		DownlinkInsertParams: DownlinkInsertParams{
			EpEUI:    epEUI,
			TenantID: 100,
			Payload:  []byte{0xAA, 0xBB},
		},
		CntDepend: true,
		PacketCnt: []int64{100, 101, 102},
	})

	// Query by packetCnt = 101 (in array)
	result, err := db.GetDownlinkByPacketCnt(t.Context(), "100", hex.EncodeToString(euiToBytes(epEUI)), 101)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.CntDepend, "Should be counter-dependent")
	assert.Contains(t, result.PacketCntArray, int64(101), "PacketCntArray should contain 101")
}

// TestGetDownlinkByPacketCnt_CounterIndependent_ReturnsEntry verifies that
// counter-independent downlinks (cnt_depend=false with null packet_cnt) are found.
//
// Spec: SCACI §3.11.1 - packetCnt resolution for counter-independent queues
func TestGetDownlinkByPacketCnt_CounterIndependent_ReturnsEntry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	// Insert endpoint
	epEUI := uint64(0x0102030405060709)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-CntIndepend",
		TenantID: 100,
	})

	// Insert counter-independent downlink (cnt_depend=false, packet_cnt=null)
	insertDownlinkWithPacketCnt(t, db, DownlinkWithPacketCntParams{
		DownlinkInsertParams: DownlinkInsertParams{
			EpEUI:    epEUI,
			TenantID: 100,
			Payload:  []byte{0xCC, 0xDD},
		},
		CntDepend: false,
		PacketCnt: nil, // NULL array
	})

	// Query by any packetCnt - should find counter-independent entry
	result, err := db.GetDownlinkByPacketCnt(t.Context(), "100", hex.EncodeToString(euiToBytes(epEUI)), 999)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.CntDepend, "Should be counter-independent")
}

// TestGetDownlinkByPacketCnt_ExcludesTerminalStatuses verifies that
// terminal status downlinks (transmitted, expired, failed, revoked) are excluded.
//
// Spec: SCACI §3.11.1 - Only non-terminal downlinks can be revoked
func TestGetDownlinkByPacketCnt_ExcludesTerminalStatuses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	// Insert endpoint
	epEUI := uint64(0x0102030405060710)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-Terminal",
		TenantID: 100,
	})

	terminalStatuses := []string{
		bssci.DLQueueStatusTransmitted,
		bssci.DLQueueStatusExpired,
		bssci.DLQueueStatusFailed,
		bssci.DLQueueStatusRevoked,
	}

	for _, status := range terminalStatuses {
		// Insert downlink with terminal status
		insertDownlinkWithPacketCnt(t, db, DownlinkWithPacketCntParams{
			DownlinkInsertParams: DownlinkInsertParams{
				EpEUI:    epEUI,
				TenantID: 100,
				Status:   status,
				Payload:  []byte{0xEE, 0xFF},
			},
			CntDepend: true,
			PacketCnt: []int64{200},
		})
	}

	// Query should return not found (all entries have terminal status)
	result, err := db.GetDownlinkByPacketCnt(t.Context(), "100", hex.EncodeToString(euiToBytes(epEUI)), 200)

	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
	assert.Nil(t, result)
}
