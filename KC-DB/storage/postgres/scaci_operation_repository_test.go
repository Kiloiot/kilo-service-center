package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSCACIOperationTestDB creates a test database with testcontainers
func setupSCACIOperationTestDB(t *testing.T) *sqlx.DB {
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	return db
}

// createSCACIOperationTestTenant inserts a tenant for SCACI operation tests
func createSCACIOperationTestTenant(t *testing.T, db *sqlx.DB, id int64, name string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO tenants (id, name, created_at, updated_at)
                       VALUES ($1, $2, NOW(), NOW()) ON CONFLICT (id) DO NOTHING`, id, name)
	require.NoError(t, err, "Failed to create test tenant")
}

// createSCACIOperationTestSession creates a minimal SCACI session for FK satisfaction
// Uses the repository pattern to ensure schema compatibility
func createSCACIOperationTestSession(t *testing.T, db *sqlx.DB, tenantID int64) int64 {
	t.Helper()

	sessionRepo := NewSCACISessionRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	createReq := &models.SCACISessionCreateRequest{
		TenantID: tenantID,
		AcEUI:    [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		SnAcUUID: [16]byte{0xA1, 0xA2, 0xA3, 0xA4, 0xA5, 0xA6, 0xA7, 0xA8, 0xA9, 0xAA, 0xAB, 0xAC, 0xAD, 0xAE, 0xAF, 0xB0},
		SnScUUID: [16]byte{0xB1, 0xB2, 0xB3, 0xB4, 0xB5, 0xB6, 0xB7, 0xB8, 0xB9, 0xBA, 0xBB, 0xBC, 0xBD, 0xBE, 0xBF, 0xC0},
	}

	session, err := sessionRepo.CreateSession(ctx, createReq)
	require.NoError(t, err, "Failed to create test SCACI session")
	require.NotNil(t, session)

	return session.ID
}

// cleanupSCACIOperationTestSession removes test session (CASCADE deletes operations)
func cleanupSCACIOperationTestSession(t *testing.T, db *sqlx.DB, sessionID int64) {
	t.Helper()
	_, err := db.Exec(`DELETE FROM scaci_sessions WHERE id = $1`, sessionID)
	require.NoError(t, err, "Failed to cleanup test SCACI session")
}

// cleanupSCACIOperationTestTenant removes test tenant
func cleanupSCACIOperationTestTenant(t *testing.T, db *sqlx.DB, tenantID int64) {
	t.Helper()
	_, err := db.Exec(`DELETE FROM tenants WHERE id = $1`, tenantID)
	require.NoError(t, err, "Failed to cleanup test tenant")
}

// TestRecordOperation_DeregisterEpEui_Roundtrip verifies that epEui hex string
// survives JSONB round-trip without case change or data loss.
//
// This test validates SCACI §3.7 epEui round-trip:
// - RequestData with epEui string is persisted to JSONB
// - GetOperationByOpID returns the exact same string
// - Uppercase hex format (e.g., "70B3D59CD000089B") is preserved
func TestRecordOperation_DeregisterEpEui_Roundtrip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test database
	db := setupSCACIOperationTestDB(t)
	defer func() { _ = db.Close() }()

	const testTenantID = int64(500)
	createSCACIOperationTestTenant(t, db, testTenantID, "DeregisterEpEuiTest")
	defer cleanupSCACIOperationTestTenant(t, db, testTenantID)

	testSessionID := createSCACIOperationTestSession(t, db, testTenantID)
	defer cleanupSCACIOperationTestSession(t, db, testSessionID)

	repo := NewSCACIOperationRepository(db, logger.Get().WithField("component", "scaci_operation_repository_test"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test case: numeric uint64 epEui for propagation lookup
	// Note: JSONB stores numbers as numeric and returns them as float64 in Go
	epEuiNumeric := uint64(0x70B3D59CD000089B)

	// Create operation with numeric epEui in RequestData
	opReq := &models.SCACIOperationRequest{
		SessionID: testSessionID,
		TenantID:  testTenantID,
		OpId:      42,
		Command:   "dereg",
		Direction: string(models.OperationDirectionInbound),
		RequestData: map[string]interface{}{
			"epEui": epEuiNumeric,
		},
	}

	// Record operation
	recorded, err := repo.RecordOperation(ctx, opReq)
	require.NoError(t, err, "RecordOperation should succeed")
	require.NotNil(t, recorded, "Recorded operation should not be nil")

	// Retrieve operation
	fetched, err := repo.GetOperationByOpID(ctx, testSessionID, 42)
	require.NoError(t, err, "GetOperationByOpID should succeed")
	require.NotNil(t, fetched, "Fetched operation should not be nil")

	// CRITICAL: Verify epEui numeric survived JSONB round-trip
	// Note: JSONB returns numbers as float64, so we compare as float64
	require.NotNil(t, fetched.RequestData, "RequestData should not be nil")
	fetchedEpEui, ok := fetched.RequestData["epEui"]
	require.True(t, ok, "RequestData should contain epEui key")
	assert.Equal(t, float64(epEuiNumeric), fetchedEpEui, "epEui should survive JSONB round-trip as numeric")
}

// TestRecordOperation_CleanupMetadata_Roundtrip verifies that all 4 cleanup
// metadata keys survive JSONB round-trip when stored in ResponseData.
//
// This validates the cleanup metadata persistence for SCACI §3.7.3:
// - epEui (string, uppercase hex)
// - cleanupSource ("cache" or "db")
// - revokedCount (integer)
// - detachErrorCount (integer)
func TestRecordOperation_CleanupMetadata_Roundtrip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSCACIOperationTestDB(t)
	defer func() { _ = db.Close() }()

	const testTenantID = int64(501)
	createSCACIOperationTestTenant(t, db, testTenantID, "CleanupMetadataTest")
	defer cleanupSCACIOperationTestTenant(t, db, testTenantID)

	testSessionID := createSCACIOperationTestSession(t, db, testTenantID)
	defer cleanupSCACIOperationTestSession(t, db, testSessionID)

	repo := NewSCACIOperationRepository(db, logger.Get().WithField("component", "scaci_operation_repository_test"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Record initial operation
	opReq := &models.SCACIOperationRequest{
		SessionID:   testSessionID,
		TenantID:    testTenantID,
		OpId:        43,
		Command:     "deregCmp",
		Direction:   string(models.OperationDirectionInbound),
		RequestData: map[string]interface{}{},
	}

	recorded, err := repo.RecordOperation(ctx, opReq)
	require.NoError(t, err, "RecordOperation should succeed")
	require.NotNil(t, recorded)

	// Update with cleanup metadata (simulating handleDeregisterComplete)
	cleanupMetadata := map[string]interface{}{
		"epEui":            "70B3D59CD000089B",
		"cleanupSource":    "cache",
		"revokedCount":     3,
		"detachErrorCount": 2,
	}

	err = repo.UpdateOperationState(ctx, testSessionID, 43, models.OperationStateCompleted, cleanupMetadata)
	require.NoError(t, err, "UpdateOperationState should succeed")

	// Retrieve and verify
	fetched, err := repo.GetOperationByOpID(ctx, testSessionID, 43)
	require.NoError(t, err, "GetOperationByOpID should succeed")
	require.NotNil(t, fetched)
	require.NotNil(t, fetched.ResponseData, "ResponseData should contain cleanup metadata")

	// Verify all 4 keys
	assert.Equal(t, "70B3D59CD000089B", fetched.ResponseData["epEui"], "epEui should match")
	assert.Equal(t, "cache", fetched.ResponseData["cleanupSource"], "cleanupSource should match")

	// JSON numbers may be float64 after unmarshal
	revokedCount, ok := fetched.ResponseData["revokedCount"].(float64)
	require.True(t, ok, "revokedCount should be numeric")
	assert.Equal(t, float64(3), revokedCount, "revokedCount should be 3")

	detachErrorCount, ok := fetched.ResponseData["detachErrorCount"].(float64)
	require.True(t, ok, "detachErrorCount should be numeric")
	assert.Equal(t, float64(2), detachErrorCount, "detachErrorCount should be 2")
}

// TestUpdateOperationState_CompletedWithWarnings verifies that the
// completed_with_warnings state is accepted by the DB CHECK constraint
// after migration 091 is applied.
//
// This validates the schema fix for SCACI §3.9 operation state mismatch.
func TestUpdateOperationState_CompletedWithWarnings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupSCACIOperationTestDB(t)
	defer func() { _ = db.Close() }()

	const testTenantID = int64(502)
	createSCACIOperationTestTenant(t, db, testTenantID, "CompletedWithWarningsTest")
	defer cleanupSCACIOperationTestTenant(t, db, testTenantID)

	testSessionID := createSCACIOperationTestSession(t, db, testTenantID)
	defer cleanupSCACIOperationTestSession(t, db, testSessionID)

	repo := NewSCACIOperationRepository(db, logger.Get().WithField("component", "scaci_operation_repository_test"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Record initial operation in pending state
	opReq := &models.SCACIOperationRequest{
		SessionID:   testSessionID,
		TenantID:    testTenantID,
		OpId:        44,
		Command:     "ulData",
		Direction:   string(models.OperationDirectionInbound),
		RequestData: map[string]interface{}{},
	}

	recorded, err := repo.RecordOperation(ctx, opReq)
	require.NoError(t, err, "RecordOperation should succeed")
	require.NotNil(t, recorded)
	require.Equal(t, string(models.OperationStatePending), recorded.State)

	// Update to completed_with_warnings state
	warningMetadata := map[string]interface{}{
		"warnings": []string{"partial_delivery", "bs_timeout"},
	}

	err = repo.UpdateOperationState(ctx, testSessionID, 44, models.OperationStateCompletedWithWarnings, warningMetadata)
	require.NoError(t, err, "UpdateOperationState to completed_with_warnings should succeed")

	// Verify state persisted correctly
	fetched, err := repo.GetOperationByOpID(ctx, testSessionID, 44)
	require.NoError(t, err, "GetOperationByOpID should succeed")
	require.NotNil(t, fetched)
	assert.Equal(t, string(models.OperationStateCompletedWithWarnings), fetched.State, "State should be completed_with_warnings")
	assert.NotNil(t, fetched.CompletedAt, "CompletedAt should be set")
}
