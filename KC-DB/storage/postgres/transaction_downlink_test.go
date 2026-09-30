package postgres

import (
	"encoding/binary"
	"encoding/json"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
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
	Status   mioty.DLQueueStatus
	UserData [][]byte // For multi-packet tests (stored as JSON in user_data column)

	// Optional fields
	QueID            int64
	BsEUI            uint64
	TxTime           *int64
	Format           uint8
	NoDefaultPayload bool // If true, skip default payload (for backward-compat tests)

	// SCACI §3.10 fields (migration 000089, 000090)
	DlRxStatQry    *bool   // DL RX status query requested; default false when nil (column is NOT NULL per migration 000089)
	OrganizationID *string // Organization UUID; a random one is generated when nil (column is NOT NULL per migration 000146)
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
		p.Status = mioty.DLQueueStatusPending
	}
	if p.Payload == nil {
		if p.NoDefaultPayload {
			// Caller wants "empty payload, UserData carries the bytes" — the
			// payload column is NOT NULL though, so substitute an empty bytea
			// instead of leaving nil (which lib/pq sends as SQL NULL).
			p.Payload = []byte{}
		} else {
			p.Payload = []byte{0x01, 0x02, 0x03}
		}
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
	//
	// dl_rx_stat_qry is NOT NULL per migration 000089. Default to false when the
	// caller didn't specify a value, so tests written before that migration (and
	// tests that don't care about the field) still insert successfully.
	dlRxStatQry := false
	if p.DlRxStatQry != nil {
		dlRxStatQry = *p.DlRxStatQry
	}

	// organization_id is NOT NULL per migration 000146: every queue row carries
	// the organization it was enqueued under. Tests that don't exercise
	// organization behavior get a generated one.
	if p.OrganizationID == nil {
		generated := uuid.New().String()
		p.OrganizationID = &generated
	}
	var holder interface{}
	if p.BsEUI != 0 {
		holder = mioty.EUI64Bytes(p.BsEUI)
	}
	var queID int64
	query := `
		INSERT INTO downlink_queue (
			ep_eui, tenant_id, payload, priority, status, user_data, format,
			dl_rx_stat_qry, organization_id, bs_eui,
			earliest_at, latest_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			NULL, NULL, NOW(), NOW())
		RETURNING que_id
	`
	err := db.conn.QueryRow(
		query,
		epEUIBytes, p.TenantID, p.Payload, p.Priority, p.Status, userDataJSON, p.Format,
		dlRxStatQry, p.OrganizationID, holder,
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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

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
	time.Sleep(testCreatedAtSpacing) // Ensure different created_at

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
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI)
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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

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
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI)

	assert.ErrorIs(t, err, storage.ErrNotFound, "nothing pending is reported as not found")
	assert.Nil(t, dl)
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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

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
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 200, epEUIBytes, bsEUI)

	assert.ErrorIs(t, err, storage.ErrNotFound, "wrong tenant must not see downlinks")
	assert.Nil(t, dl)
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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

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
		Status:   mioty.DLQueueStatusReserved,
	})
	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Priority: 9.0,
		Status:   mioty.DLQueueStatusQueued,
	})
	_ = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Priority: 8.0,
		Status:   mioty.DLQueueStatusTransmitted,
	})

	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	bsEUI := uint64(0xAABBCCDDEEFF0011)
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI)

	assert.ErrorIs(t, err, storage.ErrNotFound, "non-pending downlinks must not be selected")
	assert.Nil(t, dl)
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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

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
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI)

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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

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
		Status:   mioty.DLQueueStatusReserved,
	})

	assert.Greater(t, queID, int64(0), "Reserved status should be accepted by schema")

	// Verify via direct query
	var status mioty.DLQueueStatus
	err := sqlxDB.Get(&status, `SELECT status FROM downlink_queue WHERE que_id = $1`, queID)
	require.NoError(t, err)
	assert.Equal(t, mioty.DLQueueStatusReserved, status)
}

// TestMarkReservedAsQueued_TransitionsStatus verifies reserved→queued transition.
// Should update status and transmission_time, keeping the holding station.
func TestMarkReservedAsQueued_TransitionsStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	epEUI := uint64(0x0102030405060714)
	bsEUI := uint64(0xAABBCCDDEEFF0022)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-Transition",
		TenantID: 100,
	})

	// Insert as reserved for the station (simulating after ReserveNextPendingDownlink)
	queID := insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:    epEUI,
		TenantID: 100,
		Priority: 5.0,
		Status:   mioty.DLQueueStatusReserved,
		BsEUI:    bsEUI,
	})

	txTime := time.Now().UnixNano()
	err := NewRepositories(db).Downlinks.MarkReservedAsQueued(t.Context(), uint64(queID), 100, bsEUI, txTime, nil, nil)
	require.NoError(t, err)

	// Verify transition
	var status mioty.DLQueueStatus
	var storedTxTime int64
	var storedBsEUI []byte
	err = sqlxDB.QueryRow(`
		SELECT status, tx_time, bs_eui FROM downlink_queue WHERE que_id = $1
	`, queID).Scan(&status, &storedTxTime, &storedBsEUI)
	require.NoError(t, err)

	assert.Equal(t, mioty.DLQueueStatusQueued, status)
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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

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
		Status:   mioty.DLQueueStatusReserved,
		BsEUI:    bsEUI,
	})

	txTime := time.Now().UnixNano()
	// Pass nil for packetCnt
	err := NewRepositories(db).Downlinks.MarkReservedAsQueued(t.Context(), uint64(queID), 100, bsEUI, txTime, nil, nil)
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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

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
		Status:   mioty.DLQueueStatusPending,
	})

	txTime := time.Now().UnixNano()
	err := NewRepositories(db).Downlinks.MarkReservedAsQueued(t.Context(), uint64(queID), 100, bsEUI, txTime, nil, nil)

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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

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
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI)

	require.NoError(t, err)
	require.NotNil(t, dl)

	expectedHex := mioty.FormatEUIBytes(epEUIBytes)
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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

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
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI)

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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

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
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI)

	require.NoError(t, err)
	require.NotNil(t, dl)

	// DlRxStatQry should be false when column is FALSE
	// Note: DownlinkMessage.DlRxStatQry is bool (not pointer)
	assert.False(t, dl.DlRxStatQry, "DlRxStatQry should be false when column is FALSE")
}

// TestReserveNextPendingDownlink_DlRxStatQryDefaultFalse verifies that when
// the caller omits DlRxStatQry from the insert helper (nil), the row lands
// with dl_rx_stat_qry=false. The column is BOOLEAN NOT NULL DEFAULT false
// per migration 000089 — there is no valid SQL NULL path. The helper
// translates nil to false so legacy tests that pre-date migration 000089
// continue to insert successfully.
func TestReserveNextPendingDownlink_DlRxStatQryDefaultFalse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	epEUI := uint64(0x0102030405060722)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-DlRxStatQryDefaultFalse",
		TenantID: 100,
	})

	// DlRxStatQry omitted (nil) — helper must default to false at insert time
	// because the column is NOT NULL.
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
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI)

	require.NoError(t, err)
	require.NotNil(t, dl)

	// Helper default → false. (DownlinkMessage.DlRxStatQry is bool, not pointer.)
	assert.False(t, dl.DlRxStatQry, "nil input to helper must default the column to false")
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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

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
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI)

	require.NoError(t, err)
	require.NotNil(t, dl)

	// CRITICAL: Verify OrganizationID is populated
	require.NotNil(t, dl.OrganizationID, "OrganizationID should not be nil when column has UUID")
	assert.Equal(t, testOrgID, dl.OrganizationID.String(), "OrganizationID should match inserted value")
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
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

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
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 100, epEUIBytes, bsEUI)

	require.NoError(t, err)
	require.NotNil(t, dl)

	// CRITICAL: Verify both fields are populated correctly
	// Note: DownlinkMessage.DlRxStatQry is bool (not pointer)
	assert.True(t, dl.DlRxStatQry, "DlRxStatQry should be true")

	require.NotNil(t, dl.OrganizationID, "OrganizationID should not be nil")
	assert.Equal(t, testOrgID, dl.OrganizationID.String(), "OrganizationID should match")
}

// =============================================================================
// §3.11 DL Data Revoke - GetDownlinksByPacketCnt Tests
// =============================================================================

// DownlinkWithPacketCntParams extends DownlinkInsertParams for packet counter tests
type DownlinkWithPacketCntParams struct {
	DownlinkInsertParams
	CntDepend bool    // Counter-dependent flag
	PacketCnt []int64 // Packet counter array
}

// insertDownlinkWithPacketCnt inserts a test downlink with packet counter fields.
// Required for testing SCACI §3.11 GetDownlinksByPacketCnt functionality.
func insertDownlinkWithPacketCnt(t *testing.T, db *DB, p DownlinkWithPacketCntParams) int64 {
	t.Helper()

	// Convert epEUI to bytea
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, p.EpEUI)

	// Set defaults
	if p.Status == "" {
		p.Status = mioty.DLQueueStatusPending
	}
	if p.Payload == nil && !p.NoDefaultPayload {
		p.Payload = []byte{0x01, 0x02, 0x03}
	}

	// organization_id is NOT NULL per migration 000146; these tests don't
	// exercise organization behavior, so a generated one is used.
	if p.OrganizationID == nil {
		generated := uuid.New().String()
		p.OrganizationID = &generated
	}

	var queID int64
	query := `
		INSERT INTO downlink_queue (
			ep_eui, tenant_id, payload, priority, status, cnt_depend, packet_cnt,
			organization_id, earliest_at, latest_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8,
			NULL, NULL, NOW(), NOW())
		RETURNING que_id
	`

	var packetCntArg interface{}
	if len(p.PacketCnt) > 0 {
		packetCntArg = pq.Int64Array(p.PacketCnt)
	} else {
		packetCntArg = nil
	}

	err := db.conn.QueryRow(
		query,
		epEUIBytes, p.TenantID, p.Payload, p.Priority, p.Status, p.CntDepend, packetCntArg, p.OrganizationID,
	).Scan(&queID)
	require.NoError(t, err, "Failed to insert test downlink with packet counter")

	return queID
}

// TestGetDownlinksByPacketCnt_ReturnsEveryDownlinkScheduledForTheCounter
// pins SCACI §3.11.1: dlDataRev names the scheduled data by packet counter,
// so every counter-dependent downlink of the endpoint whose counters include
// it is addressed, newest first, and nothing else.
func TestGetDownlinksByPacketCnt_ReturnsEveryDownlinkScheduledForTheCounter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")
	createTestTenant(t, sqlxDB, 101, "TestTenant101")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	epEUI := uint64(0x0102030405060708)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-CntDepend",
		TenantID: 100,
	})
	scheduled := func(tenantID int64, cntDepend bool, packetCnt []int64) int64 {
		return insertDownlinkWithPacketCnt(t, db, DownlinkWithPacketCntParams{
			DownlinkInsertParams: DownlinkInsertParams{EpEUI: epEUI, TenantID: tenantID, Payload: []byte{0xAA}},
			CntDepend:            cntDepend,
			PacketCnt:            packetCnt,
		})
	}
	older := scheduled(100, true, []int64{100, 101, 102})
	newer := scheduled(100, true, []int64{101})
	scheduled(100, true, []int64{103})
	scheduled(100, false, nil)
	scheduled(101, true, []int64{101})

	result, err := NewRepositories(db).Downlinks.GetDownlinksByPacketCnt(t.Context(), "100", mioty.FormatEUIBytes(mioty.EUI64Bytes(epEUI)), 101)

	require.NoError(t, err)
	queIDs := make([]int64, 0, len(result))
	for _, dl := range result {
		assert.True(t, dl.CntDepend)
		assert.Contains(t, dl.PacketCntArray, int64(101))
		assert.Equal(t, "100", dl.TenantID, "another tenant's downlink is never addressed")
		queIDs = append(queIDs, dl.QueID)
	}
	assert.ElementsMatch(t, []int64{older, newer}, queIDs)
}

// TestGetDownlinksByPacketCnt_CounterIndependentIsNotAddressed: a downlink
// queued without counters has no packet counter to name it by (SCACI §3.11.1).
func TestGetDownlinksByPacketCnt_CounterIndependentIsNotAddressed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	epEUI := uint64(0x0102030405060709)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-CntIndepend",
		TenantID: 100,
	})
	insertDownlinkWithPacketCnt(t, db, DownlinkWithPacketCntParams{
		DownlinkInsertParams: DownlinkInsertParams{
			EpEUI:    epEUI,
			TenantID: 100,
			Payload:  []byte{0xCC, 0xDD},
		},
		CntDepend: false,
		PacketCnt: nil,
	})

	result, err := NewRepositories(db).Downlinks.GetDownlinksByPacketCnt(t.Context(), "100", mioty.FormatEUIBytes(mioty.EUI64Bytes(epEUI)), 999)

	require.NoError(t, err, "nothing scheduled for the counter is an empty list, not a failure")
	assert.Empty(t, result)
}

// TestGetDownlinksByPacketCnt_ExcludesTerminalStatuses verifies that
// terminal status downlinks (transmitted, expired, failed, revoked) are excluded.
//
// Spec: SCACI §3.11.1 - Only non-terminal downlinks can be revoked
func TestGetDownlinksByPacketCnt_ExcludesTerminalStatuses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	// Insert endpoint
	epEUI := uint64(0x0102030405060710)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-Terminal",
		TenantID: 100,
	})

	terminalStatuses := []mioty.DLQueueStatus{
		mioty.DLQueueStatusTransmitted,
		mioty.DLQueueStatusExpired,
		mioty.DLQueueStatusFailed,
		mioty.DLQueueStatusRevoked,
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

	result, err := NewRepositories(db).Downlinks.GetDownlinksByPacketCnt(t.Context(), "100", mioty.FormatEUIBytes(mioty.EUI64Bytes(epEUI)), 200)

	require.NoError(t, err)
	assert.Empty(t, result, "every entry is terminal")
}

// TestUpdateDownlinkResult_NilTxTimeAndPacketCnt reproduces the failure path
// fixed in cf7a5e70. When the BS reports dlDataRes with result="sent" but
// without txTime/packetCnt, lib/pq's prepared-statement type inference used
// to give up because parameter $1 was reused across column assignments and
// CASE comparisons while $2 was NULL — resulting in:
//
//	pq: inconsistent types deduced for parameter $1 at position 3:16 (42P08)
//
// The OTA delivery succeeded but the queue row stayed stuck in "queued"
// forever. The fix adds explicit ::text / ::bigint casts to every $N
// reference. This test pins that behaviour: with nil txTime and nil
// packetCnt, the UPDATE must succeed and move the row to status=transmitted.
func TestUpdateDownlinkResult_NilTxTimeAndPacketCnt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}

	epEUI := uint64(0x70B3D56770111505)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-UpdateResult-NilTxTime",
		TenantID: 100,
	})

	rxStat := false
	bsEUI := uint64(0x70B3D59CD00009E6)
	queID := insertDownlink(t, db, DownlinkInsertParams{
		EpEUI:       epEUI,
		TenantID:    100,
		Status:      mioty.DLQueueStatusQueued,
		Payload:     []byte{0x02},
		DlRxStatQry: &rxStat,
		BsEUI:       bsEUI,
	})

	// Both txTime and packetCnt are nil — the exact shape the real bug saw.
	_, err := NewRepositories(db).Downlinks.UpdateDownlinkResult(t.Context(), 100, bsEUI,
		&mioty.DLDataResult{EpEui: epEUI, QueId: uint64(queID), Result: mioty.DLDataResultSent})
	require.NoError(t, err, "UpdateDownlinkResult must succeed even when txTime and packetCnt are NULL")

	// Confirm row transitioned to transmitted.
	var status string
	var result *string
	var transmittedAt *time.Time
	err = sqlxDB.QueryRow(
		`SELECT status, result, transmitted_at FROM downlink_queue WHERE que_id = $1`,
		queID,
	).Scan(&status, &result, &transmittedAt)
	require.NoError(t, err)
	assert.Equal(t, "transmitted", status, "status must advance to transmitted on result=sent")
	require.NotNil(t, result, "result column must be populated")
	assert.Equal(t, "sent", *result)
	require.NotNil(t, transmittedAt, "transmitted_at must be populated on result=sent")
}

// TestListTenantQueue_StatusFilter pins the filter expansion from 1ac20fea.
// Before that fix, ListTenantQueue only returned rows with status in
// (pending, scheduled), so the moment a downlink advanced to "queued"
// (sent to BS via dlDataQue, awaiting OTA transmission) it disappeared from
// the Downlink Queue panel even though it was still in flight. The fix adds
// "reserved" and "queued" to the IN clause while still excluding all
// terminal statuses (transmitted, delivered, acked, failed, expired,
// revoked).
func TestListTenantQueue_StatusFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")

	logger.Initialize("error", "json")
	log := logger.Get()
	_ = log
	queueReader := NewDownlinkQueueReader(sqlxDB, logger.Get())

	epEUI := uint64(0x0102030405060708)
	insertEndpoint(t, sqlxDB, EndpointInsertParams{
		EpEUI:    epEUI,
		Name:     "TestEndpoint-QueueFilter",
		TenantID: 100,
	})

	// Seed one row per known status. The four in-flight statuses must be
	// returned; the six terminal/error statuses must be filtered out.
	inFlight := []mioty.DLQueueStatus{
		mioty.DLQueueStatusPending,
		mioty.DLQueueStatusScheduled,
		mioty.DLQueueStatusReserved,
		mioty.DLQueueStatusQueued,
	}
	terminal := []mioty.DLQueueStatus{
		mioty.DLQueueStatusTransmitted,
		mioty.DLQueueStatusDelivered,
		mioty.DLQueueStatusAcked,
		mioty.DLQueueStatusFailed,
		mioty.DLQueueStatusExpired,
		mioty.DLQueueStatusRevoked,
	}
	dbWrap := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: log}
	rxStat := false
	for _, s := range append(append([]mioty.DLQueueStatus{}, inFlight...), terminal...) {
		_ = insertDownlink(t, dbWrap, DownlinkInsertParams{
			EpEUI:       epEUI,
			TenantID:    100,
			Status:      s,
			Payload:     []byte{0x02},
			DlRxStatQry: &rxStat,
		})
	}

	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)
	var epArr [8]byte
	copy(epArr[:], epEUIBytes)

	rows, err := queueReader.ListTenantQueue(t.Context(), 100, storage.DownlinkQueueFilter{EpEUI: &epArr}, 100, 0)
	require.NoError(t, err)

	gotStatuses := make(map[mioty.DLQueueStatus]int, len(rows))
	for _, r := range rows {
		gotStatuses[r.Status]++
	}

	for _, want := range inFlight {
		assert.Equalf(t, 1, gotStatuses[want],
			"in-flight status %q must appear exactly once in queue listing, got %d", want, gotStatuses[want])
	}
	for _, term := range terminal {
		assert.Equalf(t, 0, gotStatuses[term],
			"terminal status %q must be excluded from queue listing, got %d", term, gotStatuses[term])
	}
	assert.Len(t, rows, len(inFlight),
		"queue listing must contain exactly the in-flight rows (got %d total)", len(rows))
}
