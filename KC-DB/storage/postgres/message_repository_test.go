package postgres

import (
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetDetachMessage_Success verifies FIX-5: GetDetachMessage retrieves
// detach messages with org_uuid field populated correctly.
func TestGetDetachMessage_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	// Setup: Create test tenant and organization
	const tenantID = int64(100)
	createTestTenant(t, db, tenantID, "TestTenant100")

	repo := NewMessageRepository(db)
	ctx := testutil.TestContext()

	// Create test detach message with org_uuid
	orgUUID := uuid.New()
	orgUUIDStr := orgUUID.String()
	messageID := uuid.New().String()

	detachMsg := &mioty.DetachMessage{
		ID:             messageID,
		TenantID:       tenantID,
		OrgUUID:        &orgUUIDStr,
		CommandType:    mioty.CmdDetach,
		OpId:           int64(700),
		EpEui:          epEuiToBytes(0x1111222233334444),
		BasestationEui: epEuiToBytes(0x5555666677778888),
		RxTime:         time.Now().UnixNano(),
		PacketCnt:      42,
		SNR:            15.5,
		RSSI:           -85.2,
		Signature:      []byte{0x01, 0x02, 0x03, 0x04},
		MessageType:    "detach",
		Direction:      "uplink",
		InterfaceType:  "bssci",
		ReceivedAt:     time.Now(),
	}

	// Persist message
	structuredMsg := map[string]interface{}{
		"command":   mioty.CmdDetach,
		"opId":      detachMsg.OpId,
		"epEui":     uint64(0x1111222233334444),
		"bsEui":     uint64(0x5555666677778888),
		"rxTime":    detachMsg.RxTime,
		"packetCnt": detachMsg.PacketCnt,
		"snr":       detachMsg.SNR,
		"rssi":      detachMsg.RSSI,
		"sign":      []byte{0x01, 0x02, 0x03, 0x04},
	}

	err := repo.CreateDetachMessage(ctx, detachMsg, structuredMsg)
	require.NoError(t, err, "CreateDetachMessage must succeed")

	// Test: Retrieve the message
	retrieved, err := repo.GetDetachMessage(ctx, messageID, tenantID)
	require.NoError(t, err, "GetDetachMessage must succeed")
	require.NotNil(t, retrieved, "Retrieved message must not be nil")

	// Assert: Verify all fields including org_uuid
	assert.Equal(t, messageID, retrieved.ID, "message ID must match")
	assert.Equal(t, tenantID, retrieved.TenantID, "tenant ID must match")
	assert.Equal(t, detachMsg.OpId, retrieved.OpId, "operation ID must match")
	assert.Equal(t, detachMsg.SNR, retrieved.SNR, "SNR must match")
	assert.Equal(t, detachMsg.RSSI, retrieved.RSSI, "RSSI must match")
	assert.Equal(t, detachMsg.PacketCnt, retrieved.PacketCnt, "packet count must match")
	assert.Equal(t, detachMsg.Signature, retrieved.Signature, "signature must match")

	// FIX-5: Verify org_uuid round-tripped correctly
	require.NotNil(t, retrieved.OrgUUID, "org_uuid must be populated")
	assert.Equal(t, orgUUIDStr, *retrieved.OrgUUID, "org_uuid must match persisted value")

	// Cleanup
	CleanupTestData(t, db, "mioty_messages", "id", messageID)
}

// TestGetDetachMessage_NotFound verifies error handling for non-existent message ID.
func TestGetDetachMessage_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewMessageRepository(db)
	ctx := testutil.TestContext()

	// Test: Query non-existent message
	nonExistentID := uuid.New().String()
	msg, err := repo.GetDetachMessage(ctx, nonExistentID, 100)

	// Assert: Error returned and message is nil
	require.Error(t, err, "GetDetachMessage must fail for non-existent ID")
	assert.Nil(t, msg, "Message must be nil for non-existent ID")
}

// TestGetDetachMessage_TenantIsolation verifies tenant isolation at database level.
func TestGetDetachMessage_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	// Setup: Create two tenants
	const tenant1 = int64(100)
	const tenant2 = int64(200)
	createTestTenant(t, db, tenant1, "TestTenant100")
	createTestTenant(t, db, tenant2, "TestTenant200")

	repo := NewMessageRepository(db)
	ctx := testutil.TestContext()

	// Create message for tenant1
	messageID := uuid.New().String()
	detachMsg := &mioty.DetachMessage{
		ID:             messageID,
		TenantID:       tenant1,
		CommandType:    mioty.CmdDetach,
		OpId:           int64(800),
		EpEui:          epEuiToBytes(0x2222333344445555),
		BasestationEui: epEuiToBytes(0x6666777788889999),
		RxTime:         time.Now().UnixNano(),
		PacketCnt:      10,
		SNR:            12.0,
		RSSI:           -90.0,
		Signature:      []byte{0xAA, 0xBB, 0xCC, 0xDD},
		MessageType:    "detach",
		Direction:      "uplink",
		InterfaceType:  "bssci",
		ReceivedAt:     time.Now(),
	}

	structuredMsg := map[string]interface{}{
		"command":   mioty.CmdDetach,
		"opId":      detachMsg.OpId,
		"epEui":     uint64(0x2222333344445555),
		"bsEui":     uint64(0x6666777788889999),
		"rxTime":    detachMsg.RxTime,
		"packetCnt": detachMsg.PacketCnt,
		"snr":       detachMsg.SNR,
		"rssi":      detachMsg.RSSI,
		"sign":      detachMsg.Signature,
	}

	err := repo.CreateDetachMessage(ctx, detachMsg, structuredMsg)
	require.NoError(t, err)

	// Test: tenant1 can retrieve
	retrieved1, err := repo.GetDetachMessage(ctx, messageID, tenant1)
	require.NoError(t, err, "tenant1 must be able to retrieve its own message")
	require.NotNil(t, retrieved1)

	// Test: tenant2 cannot retrieve (tenant isolation)
	retrieved2, err := repo.GetDetachMessage(ctx, messageID, tenant2)
	require.Error(t, err, "GetDetachMessage must fail for wrong tenant")
	assert.Nil(t, retrieved2, "Message must not be returned for wrong tenant")

	// Cleanup
	CleanupTestData(t, db, "mioty_messages", "id", messageID)
}

// TestGetDetachMessage_WithOptionalFields verifies optional fields are
// correctly deserialized (subpackets, eqSnr, profile, rxDuration).
func TestGetDetachMessage_WithOptionalFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	// Setup: Create test tenant
	const tenantID = int64(100)
	createTestTenant(t, db, tenantID, "TestTenant100")

	repo := NewMessageRepository(db)
	ctx := testutil.TestContext()

	// Create message with optional fields
	messageID := uuid.New().String()
	eqSnr := 23.5
	profile := "eu1"
	rxDuration := int64(1500000)

	detachMsg := &mioty.DetachMessage{
		ID:             messageID,
		TenantID:       tenantID,
		CommandType:    mioty.CmdDetach,
		OpId:           int64(900),
		EpEui:          epEuiToBytes(0x3333444455556666),
		BasestationEui: epEuiToBytes(0x7777888899990000),
		RxTime:         time.Now().UnixNano(),
		PacketCnt:      25,
		SNR:            14.0,
		RSSI:           -88.5,
		EqSnr:          &eqSnr,
		Profile:        &profile,
		RxDuration:     &rxDuration,
		Signature:      []byte{0x11, 0x22, 0x33, 0x44},
		MessageType:    "detach",
		Direction:      "uplink",
		InterfaceType:  "bssci",
		ReceivedAt:     time.Now(),
	}

	// Subpackets structure per BSSCI spec
	subpackets := &mioty.Subpackets{
		SNR:       []float64{15.0, 16.0},
		RSSI:      []float64{-85.0, -84.0},
		Frequency: []int64{868100000, 868100000},
	}
	detachMsg.Subpackets = subpackets

	structuredMsg := map[string]interface{}{
		"command":    mioty.CmdDetach,
		"opId":       detachMsg.OpId,
		"epEui":      uint64(0x3333444455556666),
		"bsEui":      uint64(0x7777888899990000),
		"rxTime":     detachMsg.RxTime,
		"packetCnt":  detachMsg.PacketCnt,
		"snr":        detachMsg.SNR,
		"rssi":       detachMsg.RSSI,
		"eqSnr":      eqSnr,
		"profile":    profile,
		"rxDuration": rxDuration,
		"sign":       detachMsg.Signature,
		"subpackets": map[string]interface{}{
			"snr":       []float64{15.0, 16.0},
			"rssi":      []float64{-85.0, -84.0},
			"frequency": []int64{868100000, 868100000},
		},
	}

	err := repo.CreateDetachMessage(ctx, detachMsg, structuredMsg)
	require.NoError(t, err)

	// Test: Retrieve and verify optional fields
	retrieved, err := repo.GetDetachMessage(ctx, messageID, tenantID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Assert: Optional fields present and correct
	require.NotNil(t, retrieved.EqSnr, "eqSnr must be populated")
	assert.Equal(t, eqSnr, *retrieved.EqSnr, "eqSnr value must match")

	require.NotNil(t, retrieved.Profile, "profile must be populated")
	assert.Equal(t, profile, *retrieved.Profile, "profile value must match")

	require.NotNil(t, retrieved.RxDuration, "rxDuration must be populated")
	assert.Equal(t, rxDuration, *retrieved.RxDuration, "rxDuration value must match")

	require.NotNil(t, retrieved.Subpackets, "subpackets must be populated")
	assert.Len(t, retrieved.Subpackets.SNR, 2, "subpackets SNR must have 2 entries")
	assert.Len(t, retrieved.Subpackets.RSSI, 2, "subpackets RSSI must have 2 entries")

	// Cleanup
	CleanupTestData(t, db, "mioty_messages", "id", messageID)
}

// Helper function to convert uint64 EUI to byte array
func epEuiToBytes(eui uint64) []byte {
	b := make([]byte, 8)
	b[0] = byte(eui >> 56)
	b[1] = byte(eui >> 48)
	b[2] = byte(eui >> 40)
	b[3] = byte(eui >> 32)
	b[4] = byte(eui >> 24)
	b[5] = byte(eui >> 16)
	b[6] = byte(eui >> 8)
	b[7] = byte(eui)
	return b
}

// ============================================================================
// UpdateULDataBaseStations Sentinel Error Tests
// Tests for ErrNoMatchingMessage when RowsAffected=0
// ============================================================================

// TestUpdateULDataBaseStations_RowsAffectedZero verifies ErrNoMatchingMessage sentinel
// is returned when UPDATE matches no rows (dedup window edge case).
func TestUpdateULDataBaseStations_RowsAffectedZero(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	// Setup: Create test tenant (no message created - we want 0 rows affected)
	const tenantID = int64(100)
	createTestTenant(t, db, tenantID, "TestTenant100")

	repo := NewMessageRepository(db)
	ctx := testutil.TestContext()

	// Test: Call UpdateULDataBaseStations with non-existent criteria
	// This should return ErrNoMatchingMessage because no message exists
	// Note: Use EUI without high bit set (< 2^63) to avoid pq driver limitation
	nonExistentEpEui := uint64(0x0011223344556677)
	packetCnt := uint32(999)
	rxTimeMin := time.Now().Add(-time.Minute).UnixNano()
	baseStationsJSON := []byte(`[{"bsEui":123456,"rssi":-80.0,"snr":10.0}]`)

	err := repo.UpdateULDataBaseStations(ctx, tenantID, nonExistentEpEui, packetCnt, rxTimeMin, baseStationsJSON)

	// Assert: Returns ErrNoMatchingMessage sentinel
	require.Error(t, err, "UpdateULDataBaseStations must return error when no rows affected")
	assert.ErrorIs(t, err, ErrNoMatchingMessage, "Error must be ErrNoMatchingMessage sentinel")
}

// TestUpdateULDataBaseStations_RowsAffectedPositive verifies UPDATE succeeds
// when a matching message exists (RowsAffected > 0).
func TestUpdateULDataBaseStations_RowsAffectedPositive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	// Setup: Create test tenant
	const tenantID = int64(100)
	createTestTenant(t, db, tenantID, "TestTenant100")

	repo := NewMessageRepository(db)
	ctx := testutil.TestContext()

	// Create a UL data message directly via SQL to avoid CreateULDataMessage's dependencies.
	// ep_eui/bs_eui are BYTEA(8); high-bit EUI now stores fine (was the BIGINT overflow).
	msgID := uuid.New().String()
	epEui := uint64(0xA011223344556688)
	bsEui := uint64(0x0022334455667788)
	packetCnt := uint32(42)
	rxTime := time.Now().UnixNano()

	_, err := db.ExecContext(ctx, `
		INSERT INTO messages (
			id, tenant_id, ep_eui, bs_eui, op_id, packet_cnt, rx_time,
			rssi, snr, command_type, dl_open, response_exp, dl_ack
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		msgID, tenantID, epEuiToBytes(epEui), epEuiToBytes(bsEui), int64(12345), packetCnt, rxTime,
		-80.0, 10.0, "ulData", false, false, false,
	)
	require.NoError(t, err, "Direct SQL insert must succeed")

	// Test: Update with matching criteria (should succeed)
	rxTimeMin := rxTime - int64(time.Minute) // Window includes our message
	baseStationsJSON := []byte(`[{"bsEui":987654321,"rssi":-85.0,"snr":8.0}]`)

	err = repo.UpdateULDataBaseStations(ctx, tenantID, epEui, packetCnt, rxTimeMin, baseStationsJSON)

	// Assert: No error (row was updated)
	assert.NoError(t, err, "UpdateULDataBaseStations must succeed when row exists")

	// Cleanup
	CleanupTestData(t, db, "messages", "id", msgID)
}

// ============================================================================
// CreateULDataMessage Tenant Validation Tests (BSSCI §5.10.1 fix)
// Defense-in-depth: validates tenant_id at repository layer before DB write
// ============================================================================

// TestCreateULDataMessage_RejectsZeroTenant verifies storage.ErrInvalidTenantID sentinel
// is returned when tenant_id is 0 (defense against upstream bugs).
func TestCreateULDataMessage_RejectsZeroTenant(t *testing.T) {
	// Unit test - no DB needed since validation happens before any DB operation
	repo := &MessageRepository{db: nil} // nil DB is fine - we fail before using it
	ctx := testutil.TestContext()

	msg := &mioty.ULDataMessage{
		TenantID:  0, // Invalid - triggers validation
		EpEui:     uint64(0x0011223344556677),
		BsEui:     uint64(0x0022334455667788),
		RxTime:    time.Now().UnixNano(),
		PacketCnt: 1,
		SNR:       10.0,
		RSSI:      -50.0,
	}

	err := repo.CreateULDataMessage(ctx, msg)

	require.Error(t, err, "CreateULDataMessage must reject zero tenant_id")
	assert.ErrorIs(t, err, storage.ErrInvalidTenantID, "Error must be ErrInvalidTenantID sentinel")
	assert.Contains(t, err.Error(), "got 0", "Error message must include the invalid value")
}

// TestCreateULDataMessage_RejectsNegativeTenant verifies storage.ErrInvalidTenantID sentinel
// is returned when tenant_id is negative (defense against signed overflow bugs).
func TestCreateULDataMessage_RejectsNegativeTenant(t *testing.T) {
	// Unit test - no DB needed since validation happens before any DB operation
	repo := &MessageRepository{db: nil} // nil DB is fine - we fail before using it
	ctx := testutil.TestContext()

	msg := &mioty.ULDataMessage{
		TenantID:  -1, // Invalid - triggers validation
		EpEui:     uint64(0x0011223344556677),
		BsEui:     uint64(0x0022334455667788),
		RxTime:    time.Now().UnixNano(),
		PacketCnt: 1,
		SNR:       10.0,
		RSSI:      -50.0,
	}

	err := repo.CreateULDataMessage(ctx, msg)

	require.Error(t, err, "CreateULDataMessage must reject negative tenant_id")
	assert.ErrorIs(t, err, storage.ErrInvalidTenantID, "Error must be ErrInvalidTenantID sentinel")
	assert.Contains(t, err.Error(), "got -1", "Error message must include the invalid value")
}

// ============================================================================
// CreateULDataMessage Owner Tenant ID Lookup Tests
// Verifies that owner_tenant_id is resolved from the endpoint table (roaming support)
// ============================================================================

// TestCreateULDataMessage_OwnerTenantFromEndpoint verifies that owner_tenant_id
// is resolved from the endpoint's owner_tenant_id when the endpoint exists.
func TestCreateULDataMessage_OwnerTenantFromEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	// Setup: Create serving tenant (100) and owning tenant (200)
	const servingTenantID = int64(100)
	const ownerTenantID = int64(200)
	createTestTenant(t, db, servingTenantID, "ServingTenant")
	createTestTenant(t, db, ownerTenantID, "OwnerTenant")

	// Insert endpoint owned by tenant 200
	epEui := uint64(0x0011223344556699)
	insertEndpoint(t, db, EndpointInsertParams{
		EpEUI:         epEui,
		Name:          "RoamingEndpoint",
		TenantID:      ownerTenantID,
		OwnerTenantID: ownerTenantID,
	})

	repo := NewMessageRepository(db)
	ctx := testutil.TestContext()

	msg := &mioty.ULDataMessage{
		TenantID:    servingTenantID,
		EpEui:       epEui,
		BsEui:       uint64(0x0022334455667788),
		CommandType: "ulData",
		OpId:        int64(5001),
		RxTime:      time.Now().UnixNano(),
		PacketCnt:   1,
		SNR:         10.0,
		RSSI:        -80.0,
	}

	err := repo.CreateULDataMessage(ctx, msg)
	require.NoError(t, err, "CreateULDataMessage must succeed")

	// Verify owner_tenant_id was resolved from the endpoint
	var resolvedOwner int64
	err = db.Get(&resolvedOwner,
		`SELECT owner_tenant_id FROM messages WHERE id = $1`, msg.ID)
	require.NoError(t, err, "SELECT owner_tenant_id must succeed")
	assert.Equal(t, ownerTenantID, resolvedOwner,
		"owner_tenant_id must come from endpoint, not serving tenant")

	// Cleanup
	CleanupTestData(t, db, "messages", "id", msg.ID)
}

// TestCreateULDataMessage_OwnerTenantFallback verifies that owner_tenant_id
// falls back to msg.TenantID when no matching endpoint exists.
func TestCreateULDataMessage_OwnerTenantFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	// Setup: Create serving tenant only — no endpoint for this EUI
	const servingTenantID = int64(100)
	createTestTenant(t, db, servingTenantID, "ServingTenant")

	repo := NewMessageRepository(db)
	ctx := testutil.TestContext()

	msg := &mioty.ULDataMessage{
		TenantID:    servingTenantID,
		EpEui:       uint64(0x00AABBCCDDEEFF00), // No endpoint in DB
		BsEui:       uint64(0x0022334455667788),
		CommandType: "ulData",
		OpId:        int64(5002),
		RxTime:      time.Now().UnixNano(),
		PacketCnt:   1,
		SNR:         10.0,
		RSSI:        -80.0,
	}

	err := repo.CreateULDataMessage(ctx, msg)
	require.NoError(t, err, "CreateULDataMessage must succeed even without matching endpoint")

	// Verify owner_tenant_id fell back to serving tenant
	var resolvedOwner int64
	err = db.Get(&resolvedOwner,
		`SELECT owner_tenant_id FROM messages WHERE id = $1`, msg.ID)
	require.NoError(t, err, "SELECT owner_tenant_id must succeed")
	assert.Equal(t, servingTenantID, resolvedOwner,
		"owner_tenant_id must fall back to msg.TenantID when no endpoint found")

	// Cleanup
	CleanupTestData(t, db, "messages", "id", msg.ID)
}
