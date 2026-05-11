package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// setupRoamingTestDB creates a test database with minimal schema for roaming tests
// This avoids migration ordering bugs (018 vs 071) by creating only what's needed
func setupRoamingTestDB(t *testing.T) *sqlx.DB {
	db, _, cleanup := SetupPostgresContainerWithoutMigrations(t)
	t.Cleanup(cleanup)

	// Create minimal schema for roaming tests
	schema := `
		-- Tenants table (required for FK constraints)
		CREATE TABLE tenants (
			id BIGSERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			status VARCHAR(50) DEFAULT 'active',
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		-- Basestations table (required for FK constraints)
		CREATE TABLE basestations (
			id BIGSERIAL PRIMARY KEY,
			bs_eui BYTEA NOT NULL CHECK (length(bs_eui) = 8),
			name VARCHAR(255) NOT NULL,
			tenant_id BIGINT NOT NULL REFERENCES tenants(id),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		-- Basestation sessions table (the actual table under test)
		CREATE TABLE basestation_sessions (
			id BIGSERIAL PRIMARY KEY,
			basestation_id BIGINT NOT NULL REFERENCES basestations(id),
			sn_bs_uuid BYTEA CHECK (length(sn_bs_uuid) = 16),
			sn_sc_uuid BYTEA CHECK (length(sn_sc_uuid) = 16),
			tenant_id BIGINT NOT NULL REFERENCES tenants(id),
			started_at TIMESTAMP NOT NULL DEFAULT NOW(),
			roaming_endpoints JSONB,
			roaming_endpoint_count INTEGER,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
	`

	_, err := db.Exec(schema)
	require.NoError(t, err, "failed to create test schema")

	return db
}

// Test helper to create a test session with NULL roaming fields
func createTestSessionWithNullRoaming(t *testing.T, db *sqlx.DB, tenantID int64, bsEui []byte) int64 {
	// First create a basestation row (required FK)
	var basestationID int64
	err := db.QueryRow(`
		INSERT INTO basestations (bs_eui, name, tenant_id, created_at, updated_at)
		VALUES ($1, 'test-bs-null-roaming', $2, NOW(), NOW())
		RETURNING id
	`, bsEui, tenantID).Scan(&basestationID)
	require.NoError(t, err, "failed to create test basestation")

	// Then create session with correct columns
	query := `
		INSERT INTO basestation_sessions (
			basestation_id, sn_bs_uuid, sn_sc_uuid, tenant_id,
			started_at, roaming_endpoints, roaming_endpoint_count
		) VALUES ($1, $2, $3, $4, NOW(), NULL, NULL)
		RETURNING id`

	snBsUUID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	snScUUID := uuid.MustParse("20000000-0000-0000-0000-000000000001")

	var sessionID int64
	err = db.QueryRow(query,
		basestationID,
		snBsUUID[:], // sn_bs_uuid (16 bytes)
		snScUUID[:], // sn_sc_uuid (16 bytes)
		tenantID,
	).Scan(&sessionID)
	require.NoError(t, err, "failed to create test session")
	return sessionID
}

// Test helper to create a test session with existing roaming endpoints
func createTestSessionWithRoaming(t *testing.T, db *sqlx.DB, tenantID int64, bsEui []byte, endpoints []models.RoamingEndpointInfo) int64 {
	// First create a basestation row (required FK)
	var basestationID int64
	err := db.QueryRow(`
		INSERT INTO basestations (bs_eui, name, tenant_id, created_at, updated_at)
		VALUES ($1, 'test-bs-with-roaming', $2, NOW(), NOW())
		RETURNING id
	`, bsEui, tenantID).Scan(&basestationID)
	require.NoError(t, err, "failed to create test basestation")

	roamingJSON, err := json.Marshal(endpoints)
	require.NoError(t, err, "failed to marshal roaming endpoints")

	// Then create session with correct columns
	query := `
		INSERT INTO basestation_sessions (
			basestation_id, sn_bs_uuid, sn_sc_uuid, tenant_id,
			started_at, roaming_endpoints, roaming_endpoint_count
		) VALUES ($1, $2, $3, $4, NOW(), $5::jsonb, $6)
		RETURNING id`

	snBsUUID := uuid.MustParse("10000000-0000-0000-0000-000000000002")
	snScUUID := uuid.MustParse("20000000-0000-0000-0000-000000000002")

	var sessionID int64
	err = db.QueryRow(query,
		basestationID,
		snBsUUID[:], // sn_bs_uuid (16 bytes)
		snScUUID[:], // sn_sc_uuid (16 bytes)
		tenantID,
		roamingJSON,
		len(endpoints),
	).Scan(&sessionID)
	require.NoError(t, err, "failed to create test session with roaming")
	return sessionID
}

// Test helper to get roaming count from session
func getRoamingCount(t *testing.T, db *sqlx.DB, sessionID int64) int {
	var count int
	err := db.QueryRow(`SELECT COALESCE(roaming_endpoint_count, 0) FROM basestation_sessions WHERE id = $1`, sessionID).Scan(&count)
	require.NoError(t, err, "failed to get roaming count")
	return count
}

// Test helper to get roaming endpoints array length
func getRoamingArrayLength(t *testing.T, db *sqlx.DB, sessionID int64) int {
	var length int
	err := db.QueryRow(`SELECT COALESCE(jsonb_array_length(roaming_endpoints), 0) FROM basestation_sessions WHERE id = $1`, sessionID).Scan(&length)
	require.NoError(t, err, "failed to get roaming array length")
	return length
}

// TestAddRoamingEndpointToSession_NullArray verifies adding to NULL array works
func TestAddRoamingEndpointToSession_NullArray(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping database integration test in short mode")
	}

	db := setupRoamingTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	// Create tenant required by FK constraints
	createTestTenant(t, db, 1, "TestTenant1")

	ctx := context.Background()
	repo := NewRoamingRepository(db)

	// Create session with NULL roaming_endpoints
	sessionID := createTestSessionWithNullRoaming(t, db, 1, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})

	// Add first endpoint to NULL array
	err := repo.AddRoamingEndpointToSession(ctx, sessionID, "AABBCCDDEEFF1122", 99)
	require.NoError(t, err)

	// Verify counter is 1
	count := getRoamingCount(t, db, sessionID)
	assert.Equal(t, 1, count, "counter should be 1 after adding to NULL array")

	// Verify array length is 1
	length := getRoamingArrayLength(t, db, sessionID)
	assert.Equal(t, 1, length, "array should have 1 element")

	// Clean up
	_, _ = db.Exec(`DELETE FROM basestation_sessions WHERE id = $1`, sessionID)
}

// TestAddRoamingEndpointToSession_NullCounter verifies NULL counter initializes correctly
func TestAddRoamingEndpointToSession_NullCounter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping database integration test in short mode")
	}

	db := setupRoamingTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	// Create tenant required by FK constraints
	createTestTenant(t, db, 1, "TestTenant1")

	ctx := context.Background()
	repo := NewRoamingRepository(db)

	// Create basestation first (required FK)
	var basestationID int64
	err := db.QueryRow(`
		INSERT INTO basestations (bs_eui, name, tenant_id, created_at, updated_at)
		VALUES ($1, 'test-bs-null-counter', $2, NOW(), NOW())
		RETURNING id
	`, []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}, int64(1)).Scan(&basestationID)
	require.NoError(t, err)

	// Create session with NULL counter but empty array
	snBsUUID := uuid.MustParse("30000000-0000-0000-0000-000000000003")
	snScUUID := uuid.MustParse("40000000-0000-0000-0000-000000000003")

	query := `
		INSERT INTO basestation_sessions (
			basestation_id, sn_bs_uuid, sn_sc_uuid, tenant_id,
			started_at, created_at, updated_at,
			roaming_endpoints, roaming_endpoint_count
		) VALUES ($1, $2, $3, $4, NOW(), NOW(), NOW(), '[]'::jsonb, NULL)
		RETURNING id`

	var sessionID int64
	err = db.QueryRow(query, basestationID, snBsUUID[:], snScUUID[:], int64(1)).Scan(&sessionID)
	require.NoError(t, err)

	// Add endpoint with NULL counter
	err = repo.AddRoamingEndpointToSession(ctx, sessionID, "FFEEDDCCBBAA9988", 88)
	require.NoError(t, err)

	// Verify counter is 1 (NULL + 1 = 1 via COALESCE)
	count := getRoamingCount(t, db, sessionID)
	assert.Equal(t, 1, count, "NULL counter should initialize to 1")

	// Clean up
	_, _ = db.Exec(`DELETE FROM basestation_sessions WHERE id = $1`, sessionID)
}

// TestRemoveRoamingEndpointFromSession_ElementExists verifies counter decrements when element exists
func TestRemoveRoamingEndpointFromSession_ElementExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping database integration test in short mode")
	}

	db := setupRoamingTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	// Create tenant required by FK constraints
	createTestTenant(t, db, 1, "TestTenant1")

	ctx := context.Background()
	repo := NewRoamingRepository(db)

	// Create session with 2 roaming endpoints
	endpoints := []models.RoamingEndpointInfo{
		{EUI: "1111111111111111", OwnerTenantID: 10, AttachedAt: time.Now().Format(time.RFC3339)},
		{EUI: "2222222222222222", OwnerTenantID: 20, AttachedAt: time.Now().Format(time.RFC3339)},
	}
	sessionID := createTestSessionWithRoaming(t, db, 1, []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22}, endpoints)

	// Verify initial state
	count := getRoamingCount(t, db, sessionID)
	assert.Equal(t, 2, count, "initial count should be 2")

	// Remove first endpoint
	err := repo.RemoveRoamingEndpointFromSession(ctx, sessionID, "1111111111111111")
	require.NoError(t, err)

	// Verify counter decremented to 1
	count = getRoamingCount(t, db, sessionID)
	assert.Equal(t, 1, count, "counter should decrement to 1 after removal")

	// Verify array length is 1
	length := getRoamingArrayLength(t, db, sessionID)
	assert.Equal(t, 1, length, "array should have 1 element after removal")

	// Clean up
	_, _ = db.Exec(`DELETE FROM basestation_sessions WHERE id = $1`, sessionID)
}

// TestRemoveRoamingEndpointFromSession_ElementNotExists verifies counter DOESN'T decrement when element doesn't exist
func TestRemoveRoamingEndpointFromSession_ElementNotExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping database integration test in short mode")
	}

	db := setupRoamingTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	// Create tenant required by FK constraints
	createTestTenant(t, db, 1, "TestTenant1")

	ctx := context.Background()
	repo := NewRoamingRepository(db)

	// Create session with 2 roaming endpoints
	endpoints := []models.RoamingEndpointInfo{
		{EUI: "AAAAAAAAAAAAAAAA", OwnerTenantID: 30, AttachedAt: time.Now().Format(time.RFC3339)},
		{EUI: "BBBBBBBBBBBBBBBB", OwnerTenantID: 40, AttachedAt: time.Now().Format(time.RFC3339)},
	}
	sessionID := createTestSessionWithRoaming(t, db, 1, []byte{0x99, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22}, endpoints)

	// Verify initial state
	count := getRoamingCount(t, db, sessionID)
	assert.Equal(t, 2, count, "initial count should be 2")

	// Try to remove non-existent endpoint
	err := repo.RemoveRoamingEndpointFromSession(ctx, sessionID, "CCCCCCCCCCCCCCCC")
	require.NoError(t, err)

	// Verify counter STAYS at 2 (element wasn't present, so no decrement)
	count = getRoamingCount(t, db, sessionID)
	assert.Equal(t, 2, count, "counter should NOT decrement when element doesn't exist")

	// Verify array length is still 2
	length := getRoamingArrayLength(t, db, sessionID)
	assert.Equal(t, 2, length, "array should still have 2 elements")

	// Clean up
	_, _ = db.Exec(`DELETE FROM basestation_sessions WHERE id = $1`, sessionID)
}

// TestRemoveRoamingEndpointFromSession_EmptyArray verifies counter stays at 0 with empty array
func TestRemoveRoamingEndpointFromSession_EmptyArray(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping database integration test in short mode")
	}

	db := setupRoamingTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	// Create tenant required by FK constraints
	createTestTenant(t, db, 1, "TestTenant1")

	ctx := context.Background()
	repo := NewRoamingRepository(db)

	// Create session with empty roaming array
	sessionID := createTestSessionWithRoaming(t, db, 1, []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE}, []models.RoamingEndpointInfo{})

	// Verify initial state
	count := getRoamingCount(t, db, sessionID)
	assert.Equal(t, 0, count, "initial count should be 0")

	// Try to remove from empty array
	err := repo.RemoveRoamingEndpointFromSession(ctx, sessionID, "DDDDDDDDDDDDDDDD")
	require.NoError(t, err)

	// Verify counter stays at 0 (GREATEST prevents negative)
	count = getRoamingCount(t, db, sessionID)
	assert.Equal(t, 0, count, "counter should stay at 0 with empty array")

	// Verify array is still empty
	length := getRoamingArrayLength(t, db, sessionID)
	assert.Equal(t, 0, length, "array should still be empty")

	// Clean up
	_, _ = db.Exec(`DELETE FROM basestation_sessions WHERE id = $1`, sessionID)
}
