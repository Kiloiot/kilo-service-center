package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSCACISessionTestDB connects via testcontainers
func setupSCACISessionTestDB(t *testing.T) *sqlx.DB {
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	return db
}

// createSCACITestTenant inserts a tenant for SCACI session tests (mirrors endpoint_repository_test.go)
func createSCACITestTenant(t *testing.T, db *sqlx.DB, id int64, name string) {
	_, err := db.Exec(`INSERT INTO tenants (id, name, created_at, updated_at)
                       VALUES ($1, $2, NOW(), NOW()) ON CONFLICT (id) DO NOTHING`, id, name)
	require.NoError(t, err)
}

// TestSCACISessionRepository_CreateSession_WithTLS verifies TLS fields persist on create
func TestSCACISessionRepository_CreateSession_WithTLS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSCACISessionTestDB(t)
	defer func() { _ = db.Close() }()

	createSCACITestTenant(t, db, 100, "TestTenant100")

	repo := NewSCACISessionRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tlsVer := "TLS 1.3"
	cipher := "TLS_AES_256_GCM_SHA384"

	createReq := &models.SCACISessionCreateRequest{
		TenantID:    100,
		AcEUI:       [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		SnAcUUID:    [16]byte{0xA1, 0xA2, 0xA3, 0xA4, 0xA5, 0xA6, 0xA7, 0xA8, 0xA9, 0xAA, 0xAB, 0xAC, 0xAD, 0xAE, 0xAF, 0xB0},
		SnScUUID:    [16]byte{0xB1, 0xB2, 0xB3, 0xB4, 0xB5, 0xB6, 0xB7, 0xB8, 0xB9, 0xBA, 0xBB, 0xBC, 0xBD, 0xBE, 0xBF, 0xC0},
		TLSVersion:  &tlsVer,
		CipherSuite: &cipher,
		CanResume:   true,
	}

	session, err := repo.CreateSession(ctx, createReq)
	require.NoError(t, err)
	require.NotNil(t, session)

	// Fetch and verify TLS fields persisted
	fetched, err := repo.GetSessionByID(ctx, 100, session.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched.TLSVersion)
	require.NotNil(t, fetched.CipherSuite)
	assert.Equal(t, tlsVer, *fetched.TLSVersion)
	assert.Equal(t, cipher, *fetched.CipherSuite)
}

// TestSCACISessionRepository_UpdateSession_TLSFields verifies both TLS fields update on resume
func TestSCACISessionRepository_UpdateSession_TLSFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSCACISessionTestDB(t)
	defer func() { _ = db.Close() }()

	createSCACITestTenant(t, db, 101, "TestTenant101")

	repo := NewSCACISessionRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	initialTLS := "TLS 1.2"
	initialCipher := "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"

	createReq := &models.SCACISessionCreateRequest{
		TenantID:    101,
		AcEUI:       [8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18},
		SnAcUUID:    [16]byte{0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7, 0xC8, 0xC9, 0xCA, 0xCB, 0xCC, 0xCD, 0xCE, 0xCF, 0xD0},
		SnScUUID:    [16]byte{0xD1, 0xD2, 0xD3, 0xD4, 0xD5, 0xD6, 0xD7, 0xD8, 0xD9, 0xDA, 0xDB, 0xDC, 0xDD, 0xDE, 0xDF, 0xE0},
		TLSVersion:  &initialTLS,
		CipherSuite: &initialCipher,
		CanResume:   true,
	}

	session, err := repo.CreateSession(ctx, createReq)
	require.NoError(t, err)

	// Update with new TLS values (simulating resume)
	newTLS := "TLS 1.3"
	newCipher := "TLS_AES_256_GCM_SHA384"

	updateReq := &models.SCACISessionUpdateRequest{
		TLSVersion:  &newTLS,
		CipherSuite: &newCipher,
	}

	err = repo.UpdateSession(ctx, 101, session.ID, updateReq)
	require.NoError(t, err)

	// Verify updated values
	updated, err := repo.GetSessionByID(ctx, 101, session.ID)
	require.NoError(t, err)
	assert.Equal(t, newTLS, *updated.TLSVersion)
	assert.Equal(t, newCipher, *updated.CipherSuite)
}

// TestSCACISessionRepository_UpdateSession_PartialTLS verifies partial TLS update preserves other field
func TestSCACISessionRepository_UpdateSession_PartialTLS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSCACISessionTestDB(t)
	defer func() { _ = db.Close() }()

	createSCACITestTenant(t, db, 102, "TestTenant102")

	repo := NewSCACISessionRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	initialTLS := "TLS 1.2"
	initialCipher := "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"

	createReq := &models.SCACISessionCreateRequest{
		TenantID:    102,
		AcEUI:       [8]byte{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28},
		SnAcUUID:    [16]byte{0xE1, 0xE2, 0xE3, 0xE4, 0xE5, 0xE6, 0xE7, 0xE8, 0xE9, 0xEA, 0xEB, 0xEC, 0xED, 0xEE, 0xEF, 0xF0},
		SnScUUID:    [16]byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x00},
		TLSVersion:  &initialTLS,
		CipherSuite: &initialCipher,
		CanResume:   true,
	}

	session, err := repo.CreateSession(ctx, createReq)
	require.NoError(t, err)

	// Update ONLY TLSVersion
	newTLS := "TLS 1.3"
	updateReq := &models.SCACISessionUpdateRequest{
		TLSVersion: &newTLS,
		// CipherSuite intentionally nil
	}

	err = repo.UpdateSession(ctx, 102, session.ID, updateReq)
	require.NoError(t, err)

	// Verify TLSVersion updated, CipherSuite unchanged
	updated, err := repo.GetSessionByID(ctx, 102, session.ID)
	require.NoError(t, err)
	assert.Equal(t, newTLS, *updated.TLSVersion, "TLSVersion should be updated")
	assert.Equal(t, initialCipher, *updated.CipherSuite, "CipherSuite should remain unchanged")
}

// ============================================================================
// Cross-Tenant Protection Tests (Defense-in-Depth at Repository Layer)
// ============================================================================

// TestSCACISessionRepository_CheckSessionResumable_TenantFilter validates that
// CheckSessionResumable filters by tenant_id at SQL level, providing defense-in-depth
// against cross-tenant session hijacking.
// Ref: scaci_session_repository.go:507-575 (WHERE tenant_id = $1)
func TestSCACISessionRepository_CheckSessionResumable_TenantFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSCACISessionTestDB(t)
	defer func() { _ = db.Close() }()

	// Create two tenants
	createSCACITestTenant(t, db, 201, "TenantA_Victim")
	createSCACITestTenant(t, db, 202, "TenantB_Attacker")

	repo := NewSCACISessionRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create session owned by tenant 201 (victim)
	snAcUUID := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	createReq := &models.SCACISessionCreateRequest{
		TenantID:          201, // Session belongs to tenant 201
		AcEUI:             [8]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22},
		SnAcUUID:          snAcUUID,
		SnScUUID:          [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20},
		CanResume:         true,
		NegotiatedVersion: "1.0.0",
	}

	session, err := repo.CreateSession(ctx, createReq)
	require.NoError(t, err)
	require.NotNil(t, session)

	// Test 1: Same tenant (201) can check resumability - should succeed
	info, err := repo.CheckSessionResumable(ctx, 201, snAcUUID, 0, 0)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.True(t, info.CanResume, "Same tenant should be able to resume session")
	assert.Equal(t, session.ID, info.SessionID, "SessionID should match")

	// Test 2: Different tenant (202) tries to check resumability - should NOT find session
	// This is defense-in-depth: even if attacker knows the snAcUUID, they cannot resume
	infoAttacker, err := repo.CheckSessionResumable(ctx, 202, snAcUUID, 0, 0)
	require.NoError(t, err) // Query succeeds, but returns "not found"
	require.NotNil(t, infoAttacker)
	assert.False(t, infoAttacker.CanResume, "Different tenant should not be able to resume session")
	assert.Equal(t, "session not found", infoAttacker.ReasonIfNotResumable,
		"Cross-tenant lookup should return 'session not found' to prevent tenant enumeration")
}

// TestSCACISessionRepository_GetSessionByAcUUID_TenantScoped validates that
// GetSessionByAcUUID filters by tenant_id, preventing cross-tenant data access.
func TestSCACISessionRepository_GetSessionByAcUUID_TenantScoped(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSCACISessionTestDB(t)
	defer func() { _ = db.Close() }()

	// Create two tenants
	createSCACITestTenant(t, db, 203, "TenantC_Owner")
	createSCACITestTenant(t, db, 204, "TenantD_Other")

	repo := NewSCACISessionRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create session owned by tenant 203
	snAcUUID := [16]byte{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2A, 0x2B, 0x2C, 0x2D, 0x2E, 0x2F, 0x30}
	createReq := &models.SCACISessionCreateRequest{
		TenantID:          203,
		AcEUI:             [8]byte{0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22, 0x33, 0x44},
		SnAcUUID:          snAcUUID,
		SnScUUID:          [16]byte{0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3A, 0x3B, 0x3C, 0x3D, 0x3E, 0x3F, 0x40},
		CanResume:         true,
		NegotiatedVersion: "1.0.0",
	}

	session, err := repo.CreateSession(ctx, createReq)
	require.NoError(t, err)
	require.NotNil(t, session)

	// Test 1: Owner tenant (203) can retrieve session
	retrieved, err := repo.GetSessionByAcUUID(ctx, 203, snAcUUID)
	require.NoError(t, err)
	require.NotNil(t, retrieved, "Owner tenant should retrieve session")
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, int64(203), retrieved.TenantID)

	// Test 2: Different tenant (204) cannot retrieve the session
	crossTenant, err := repo.GetSessionByAcUUID(ctx, 204, snAcUUID)
	require.NoError(t, err) // Query succeeds
	assert.Nil(t, crossTenant, "Different tenant should not retrieve session (tenant filtering at SQL level)")
}

// TestSCACISessionRepository_GetSessionByID_TenantScoped validates that
// GetSessionByID also filters by tenant_id for defense-in-depth.
func TestSCACISessionRepository_GetSessionByID_TenantScoped(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSCACISessionTestDB(t)
	defer func() { _ = db.Close() }()

	// Create two tenants
	createSCACITestTenant(t, db, 205, "TenantE_Owner")
	createSCACITestTenant(t, db, 206, "TenantF_Other")

	repo := NewSCACISessionRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create session owned by tenant 205
	createReq := &models.SCACISessionCreateRequest{
		TenantID:          205,
		AcEUI:             [8]byte{0xEE, 0xFF, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
		SnAcUUID:          [16]byte{0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4A, 0x4B, 0x4C, 0x4D, 0x4E, 0x4F, 0x50},
		SnScUUID:          [16]byte{0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5A, 0x5B, 0x5C, 0x5D, 0x5E, 0x5F, 0x60},
		CanResume:         true,
		NegotiatedVersion: "1.0.0",
	}

	session, err := repo.CreateSession(ctx, createReq)
	require.NoError(t, err)
	require.NotNil(t, session)

	// Test 1: Owner tenant (205) can retrieve by ID
	retrieved, err := repo.GetSessionByID(ctx, 205, session.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved, "Owner tenant should retrieve session by ID")
	assert.Equal(t, session.ID, retrieved.ID)

	// Test 2: Different tenant (206) cannot retrieve by ID even if they know the session ID
	crossTenant, err := repo.GetSessionByID(ctx, 206, session.ID)
	require.NoError(t, err) // Query succeeds
	assert.Nil(t, crossTenant, "Different tenant should not retrieve session by ID (tenant filtering at SQL level)")
}
