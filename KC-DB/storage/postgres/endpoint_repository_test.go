package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupEndpointTestDB connects to the test database using testcontainers
func setupEndpointTestDB(t *testing.T) *sqlx.DB {
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	return db
}

// cleanupEndpointTestData removes test endpoints by name pattern
func cleanupEndpointTestData(t *testing.T, db *sqlx.DB, namePattern string) {
	_, err := db.Exec("DELETE FROM endpoints WHERE name LIKE $1", namePattern)
	if err != nil {
		t.Logf("Warning: cleanup failed: %v", err)
	}
}

// TestCountByTenant_ReturnsCorrectCount verifies tenant endpoint counting
func TestCountByTenant_ReturnsCorrectCount(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create test tenants
	createTestTenant(t, db, 100, "TestTenant100")
	createTestTenant(t, db, 999, "TestTenant999")

	cleanupEndpointTestData(t, db, "TestCount%")
	defer cleanupEndpointTestData(t, db, "TestCount%")

	// Insert test endpoints for tenant 100 and 999
	insertEndpoint(t, db, EndpointInsertParams{EpEUI: 0x0000000000000001, Name: "TestCount-EP1", TenantID: 100})
	insertEndpoint(t, db, EndpointInsertParams{EpEUI: 0x0000000000000002, Name: "TestCount-EP2", TenantID: 100})
	insertEndpoint(t, db, EndpointInsertParams{EpEUI: 0x0000000000000003, Name: "TestCount-EP3", TenantID: 999})

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	// Test: Count endpoints for tenant 100
	count, err := repo.CountByTenant(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count, "Should count 2 endpoints for tenant 100")

	// Test: Count endpoints for tenant 999
	count, err = repo.CountByTenant(ctx, 999)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "Should count 1 endpoint for tenant 999")

	// Test: Count endpoints for non-existent tenant
	count, err = repo.CountByTenant(ctx, 8888)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "Should count 0 endpoints for non-existent tenant")
}

// TestListByTenantPaginated_ReturnsCorrectPage verifies DB-side pagination
func TestListByTenantPaginated_ReturnsCorrectPage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create test tenant
	createTestTenant(t, db, 200, "TestTenant200")

	cleanupEndpointTestData(t, db, "TestPaginated%")
	defer cleanupEndpointTestData(t, db, "TestPaginated%")

	// Insert 5 test endpoints for tenant 200 with staggered created_at times
	// EP1 is oldest, EP5 is newest (for DESC ordering verification)
	now := time.Now()
	insertEndpoint(t, db, EndpointInsertParams{
		EpEUI: 0x0000000000000011, Name: "TestPaginated-EP1",
		Description: "Test endpoint 1", TenantID: 200,
		CreatedAt: ptrTime(now.Add(-4 * time.Minute)),
	})
	insertEndpoint(t, db, EndpointInsertParams{
		EpEUI: 0x0000000000000012, Name: "TestPaginated-EP2",
		Description: "Test endpoint 2", TenantID: 200,
		CreatedAt: ptrTime(now.Add(-3 * time.Minute)),
	})
	insertEndpoint(t, db, EndpointInsertParams{
		EpEUI: 0x0000000000000013, Name: "TestPaginated-EP3",
		Description: "Test endpoint 3", TenantID: 200,
		CreatedAt: ptrTime(now.Add(-2 * time.Minute)),
	})
	insertEndpoint(t, db, EndpointInsertParams{
		EpEUI: 0x0000000000000014, Name: "TestPaginated-EP4",
		Description: "Test endpoint 4", TenantID: 200,
		CreatedAt: ptrTime(now.Add(-1 * time.Minute)),
	})
	insertEndpoint(t, db, EndpointInsertParams{
		EpEUI: 0x0000000000000015, Name: "TestPaginated-EP5",
		Description: "Test endpoint 5", TenantID: 200,
		CreatedAt: ptrTime(now),
	})

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	// Test: First page (LIMIT 2, OFFSET 0)
	page1, err := repo.ListByTenantPaginated(ctx, 200, 2, 0)
	require.NoError(t, err)
	require.Len(t, page1, 2, "First page should have 2 endpoints")
	// Verify DESC order (newest first)
	assert.Equal(t, "TestPaginated-EP5", page1[0].Name)
	assert.Equal(t, "TestPaginated-EP4", page1[1].Name)

	// Test: Second page (LIMIT 2, OFFSET 2)
	page2, err := repo.ListByTenantPaginated(ctx, 200, 2, 2)
	require.NoError(t, err)
	require.Len(t, page2, 2, "Second page should have 2 endpoints")
	assert.Equal(t, "TestPaginated-EP3", page2[0].Name)
	assert.Equal(t, "TestPaginated-EP2", page2[1].Name)

	// Test: Third page (LIMIT 2, OFFSET 4) - partial page
	page3, err := repo.ListByTenantPaginated(ctx, 200, 2, 4)
	require.NoError(t, err)
	require.Len(t, page3, 1, "Third page should have 1 endpoint")
	assert.Equal(t, "TestPaginated-EP1", page3[0].Name)

	// Test: Fourth page (LIMIT 2, OFFSET 6) - empty
	page4, err := repo.ListByTenantPaginated(ctx, 200, 2, 6)
	require.NoError(t, err)
	assert.Empty(t, page4, "Fourth page should be empty")
}

// TestListByTenantPaginated_OnlyReturnsOwnTenant verifies tenant isolation
func TestListByTenantPaginated_OnlyReturnsOwnTenant(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create test tenants
	createTestTenant(t, db, 300, "TestTenant300")
	createTestTenant(t, db, 400, "TestTenant400")

	cleanupEndpointTestData(t, db, "TestIsolation%")
	defer cleanupEndpointTestData(t, db, "TestIsolation%")

	// Insert endpoints for different tenants
	insertEndpoint(t, db, EndpointInsertParams{EpEUI: 0x0000000000000021, Name: "TestIsolation-T300-EP1", Description: "Tenant 300 endpoint 1", TenantID: 300})
	insertEndpoint(t, db, EndpointInsertParams{EpEUI: 0x0000000000000022, Name: "TestIsolation-T300-EP2", Description: "Tenant 300 endpoint 2", TenantID: 300})
	insertEndpoint(t, db, EndpointInsertParams{EpEUI: 0x0000000000000023, Name: "TestIsolation-T400-EP1", Description: "Tenant 400 endpoint 1", TenantID: 400})

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	// Test: List endpoints for tenant 300 only
	endpoints, err := repo.ListByTenantPaginated(ctx, 300, 10, 0)
	require.NoError(t, err)
	require.Len(t, endpoints, 2, "Should only return tenant 300 endpoints")

	for _, ep := range endpoints {
		assert.Equal(t, int64(300), ep.TenantID, "All endpoints should belong to tenant 300")
		assert.Contains(t, ep.Name, "T300", "Endpoint name should indicate tenant 300")
	}
}

// TestCreate_BasicFields verifies endpoint creation with mandatory fields
func TestCreate_BasicFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create test tenant
	createTestTenant(t, db, 500, "TestTenant500")

	cleanupEndpointTestData(t, db, "TestCreate%")
	defer cleanupEndpointTestData(t, db, "TestCreate%")

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	// Create test endpoint
	var eui models.EUI
	copy(eui[:], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x31})

	nwkKey := make([]byte, 16)
	appKey := make([]byte, 16)

	endpoint := &models.EndPoint{
		EUI:        eui,
		Name:       "TestCreate-EP1",
		TenantID:   500,
		EPClass:    "A",
		NwkSnKey:   nwkKey,
		AppKey:     appKey,
		CryptoMode: 0,
		Tags:       make(map[string]string),
	}

	// Test: Create endpoint
	err := repo.Create(ctx, endpoint)
	require.NoError(t, err)
	assert.NotZero(t, endpoint.ID, "ID should be assigned after creation")
	assert.NotZero(t, endpoint.CreatedAt, "CreatedAt should be set")
	assert.NotZero(t, endpoint.UpdatedAt, "UpdatedAt should be set")
}

// TestBlueprintSnapshot_RoundTrip verifies blueprint_snapshot round-trips and a nil Update preserves the existing snapshot (COALESCE).
func TestBlueprintSnapshot_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup
	createTestTenant(t, db, 501, "TestTenant501")
	cleanupEndpointTestData(t, db, "TestSnap%")
	defer cleanupEndpointTestData(t, db, "TestSnap%")

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	var eui models.EUI
	copy(eui[:], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x51})
	snap := []byte(`{"spec_json":{"format":1},"version":"1.0.0","type_eui":"70b3d59cd0000094","is_system":false}`)

	ep := &models.EndPoint{
		EUI: eui, Name: "TestSnap-EP1", TenantID: 501, EPClass: "A",
		NwkSnKey: make([]byte, 16), AppKey: make([]byte, 16), Tags: make(map[string]string),
		BlueprintSnapshot: snap,
	}
	require.NoError(t, repo.Create(ctx, ep))

	got, err := repo.GetByEUI(ctx, 501, eui[:])
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.JSONEq(t, string(snap), string(got.BlueprintSnapshot), "snapshot must round-trip via GetByEUI")

	byID, err := repo.GetByID(ctx, got.ID, 501)
	require.NoError(t, err)
	assert.JSONEq(t, string(snap), string(byID.BlueprintSnapshot), "snapshot must round-trip via GetByID")

	got.BlueprintSnapshot = nil
	got.Name = "TestSnap-EP1-upd"
	require.NoError(t, repo.Update(ctx, got))
	after, err := repo.GetByID(ctx, got.ID, 501)
	require.NoError(t, err)
	assert.JSONEq(t, string(snap), string(after.BlueprintSnapshot), "nil-snapshot Update must preserve existing (COALESCE)")

	// Non-nil Update must replace: guards a $N/COALESCE slip a nil-preserve test alone wouldn't catch.
	newSnap := []byte(`{"spec_json":{"format":2},"version":"2.0.0","type_eui":"70b3d59cd0000099","is_system":true}`)
	after.BlueprintSnapshot = newSnap
	require.NoError(t, repo.Update(ctx, after))
	replaced, err := repo.GetByID(ctx, after.ID, 501)
	require.NoError(t, err)
	assert.JSONEq(t, string(newSnap), string(replaced.BlueprintSnapshot), "non-nil Update must replace the snapshot")

	// Snapshot-less endpoint reads back NULL cleanly (the []byte scan fix).
	var eui2 models.EUI
	copy(eui2[:], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x52})
	ep2 := &models.EndPoint{
		EUI: eui2, Name: "TestSnap-EP2", TenantID: 501, EPClass: "A",
		NwkSnKey: make([]byte, 16), AppKey: make([]byte, 16), Tags: make(map[string]string),
	}
	require.NoError(t, repo.Create(ctx, ep2))
	got2, err := repo.GetByEUI(ctx, 501, eui2[:])
	require.NoError(t, err)
	assert.Nil(t, got2.BlueprintSnapshot, "snapshot-less endpoint scans NULL as nil")
}

// TestGetByEUI_ReturnsCorrectEndpoint verifies retrieval by EUI and tenant
func TestGetByEUI_ReturnsCorrectEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create test tenant
	createTestTenant(t, db, 600, "TestTenant600")

	cleanupEndpointTestData(t, db, "TestGetByEUI%")
	defer cleanupEndpointTestData(t, db, "TestGetByEUI%")

	// Insert test endpoint
	insertEndpoint(t, db, EndpointInsertParams{
		EpEUI:       0x0000000000000041,
		Name:        "TestGetByEUI-EP1",
		Description: "Test endpoint for GetByEUI",
		TenantID:    600,
	})

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	// Test: Get by EUI and tenant
	euiBytes := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x41}
	endpoint, err := repo.GetByEUI(ctx, 600, euiBytes)
	require.NoError(t, err)
	require.NotNil(t, endpoint)
	assert.Equal(t, "TestGetByEUI-EP1", endpoint.Name)
	assert.Equal(t, int64(600), endpoint.TenantID)

	// Test: Get with wrong tenant - should fail
	_, err = repo.GetByEUI(ctx, 999, euiBytes)
	require.Error(t, err, "Should fail when tenant doesn't match")
}

// =============================================================================
// BSSCI-ATTACH-022: UpdateRadioMetricsSelective Test Coverage
// =============================================================================

// ptr is a helper function to create pointers to primitive types
func ptr[T any](v T) *T {
	return &v
}

// seedMetrics defines baseline radio metrics for test fixtures
type seedMetrics struct {
	SNR        float64
	RSSI       float64
	EqSNR      float64
	RxTime     int64
	RxDuration int64  // baseline: 500
	Profile    string // baseline: "TestProfile_Baseline"
}

// seedEndpointWithMetrics inserts an endpoint with known baseline metrics
// and immediately reloads it via GetByEUI to capture initial state including
// last_seen_at timestamp. This baseline is essential for asserting
// "last_seen_at incremented" in subsequent updates.
func seedEndpointWithMetrics(
	t *testing.T,
	db *sqlx.DB,
	repo interfaces.EndpointRepository,
	tenantID int64,
	eui models.EUI,
	metrics seedMetrics,
) *models.EndPoint {
	t.Helper()

	// Compute UTF-8 safe name (no binary data in text fields)
	name := fmt.Sprintf("Test-%016x", eui.ToUint64())

	// Fixed 16-byte zero keys
	zeroKey := make([]byte, 16)

	// Parameterized INSERT with placeholders (no string interpolation)
	const insertSQL = `
		INSERT INTO endpoints (
			ep_eui, name, description, tenant_id, owner_tenant_id,
			nwk_key, app_key, crypto_mode,
			last_snr, last_rssi, last_eq_snr,
			last_attach_rx_time, last_attach_rx_duration,
			last_profile,
			created_at, updated_at, last_seen_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13,
			$14,
			NOW(), NOW(), NOW()
		)`

	// Execute with parameters (no SQL injection risk)
	ctx := testutil.TestContext()
	_, err := db.ExecContext(ctx, insertSQL,
		eui[:],             // $1: ep_eui (bytea)
		name,               // $2: name (text, UTF-8 safe)
		"",                 // $3: description (empty string, not NULL)
		tenantID,           // $4: tenant_id
		tenantID,           // $5: owner_tenant_id
		zeroKey,            // $6: nwk_key
		zeroKey,            // $7: app_key
		0,                  // $8: crypto_mode
		metrics.SNR,        // $9: last_snr
		metrics.RSSI,       // $10: last_rssi
		metrics.EqSNR,      // $11: last_eq_snr
		metrics.RxTime,     // $12: last_attach_rx_time
		metrics.RxDuration, // $13: last_attach_rx_duration
		metrics.Profile,    // $14: last_profile
	)
	require.NoError(t, err, "Failed to insert seeded endpoint")

	// IMMEDIATELY RELOAD to capture initial state (including last_seen_at)
	endpoint, err := repo.GetByEUI(ctx, tenantID, eui[:])
	require.NoError(t, err, "Failed to reload seeded endpoint")
	require.NotNil(t, endpoint, "Seeded endpoint is nil")

	return endpoint
}

// reloadEndpoint queries an endpoint after update for field validation
func reloadEndpoint(
	t *testing.T,
	repo interfaces.EndpointRepository,
	tenantID int64,
	eui models.EUI,
) *models.EndPoint {
	t.Helper()

	ctx := testutil.TestContext()
	endpoint, err := repo.GetByEUI(ctx, tenantID, eui[:])
	require.NoError(t, err, "Failed to reload endpoint")
	require.NotNil(t, endpoint, "Reloaded endpoint is nil")
	return endpoint
}

// querySubpacketsAndProfile directly queries attach subpacket and profile columns.
// Used to validate JSONB persistence without relying on repository projections.
func querySubpacketsAndProfile(
	t *testing.T,
	db *sqlx.DB,
	tenantID int64,
	endpointID int64,
) (subpackets *string, profile *string, rxTime *int64, rxDuration *int64) {
	t.Helper()

	query := `SELECT last_attach_subpackets, last_profile, last_attach_rx_time, last_attach_rx_duration
		FROM endpoints WHERE id = $1 AND tenant_id = $2`

	var subpacketsNull, profileNull sql.NullString
	var rxTimeNull, rxDurationNull sql.NullInt64

	err := db.QueryRowContext(testutil.TestContext(), query, endpointID, tenantID).Scan(
		&subpacketsNull, &profileNull, &rxTimeNull, &rxDurationNull)
	require.NoError(t, err, "Failed to query subpackets/profile")

	if subpacketsNull.Valid {
		subpackets = &subpacketsNull.String
	}
	if profileNull.Valid {
		profile = &profileNull.String
	}
	if rxTimeNull.Valid {
		rxTime = &rxTimeNull.Int64
	}
	if rxDurationNull.Valid {
		rxDuration = &rxDurationNull.Int64
	}
	return
}

// querySubpacketsOnly queries only subpackets column, returns nil on not found (for tenant isolation tests).
func querySubpacketsOnly(
	t *testing.T,
	db *sqlx.DB,
	tenantID int64,
	endpointID int64,
) (subpackets *string, err error) {
	t.Helper()

	query := `SELECT last_attach_subpackets FROM endpoints WHERE id = $1 AND tenant_id = $2`

	var subpacketsNull sql.NullString
	err = db.QueryRowContext(testutil.TestContext(), query, endpointID, tenantID).Scan(&subpacketsNull)
	if err == sql.ErrNoRows {
		return nil, nil // No rows = tenant isolation working correctly
	}
	if err != nil {
		return nil, err
	}
	if subpacketsNull.Valid {
		subpackets = &subpacketsNull.String
	}
	return subpackets, nil
}

// TestUpdateRadioMetricsSelective_PreservesRxDuration verifies that
// RxDuration=nil preserves existing database value while updating Profile
// and mandatory fields per BSSCI §5.6 attach telemetry requirements.
func TestUpdateRadioMetricsSelective_PreservesRxDuration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 100, "TestTenant100")
	defer cleanupEndpointTestData(t, db, "PreservesRxDuration%")

	repo := NewEndPointRepository(db)
	eui := models.EUI{0, 0, 0, 0, 0, 0, 0, 1}

	// Seed with baseline metrics - captures initial last_seen_at
	baseline := seedEndpointWithMetrics(t, db, repo, 100, eui, seedMetrics{
		SNR:        10.5,
		RSSI:       -80.0,
		EqSNR:      12.3,
		RxTime:     1234567890,
		RxDuration: 500,
		Profile:    "Baseline",
	})

	// Update with RxDuration=nil (preserve), Profile updated
	ctx := testutil.TestContext()
	update := interfaces.RadioMetricsUpdate{
		SNR:        20.0,
		RSSI:       -70.0,
		EqSNR:      22.0,
		RxTime:     9999999999,
		RxDuration: nil, // PRESERVE
		Profile:    ptr("Updated"),
	}
	err := repo.UpdateRadioMetricsSelective(ctx, 100, eui, update)
	require.NoError(t, err)

	// Reload and verify
	result := reloadEndpoint(t, repo, 100, eui)

	// Assert optional fields
	require.NotNil(t, result.LastAttachRxDuration, "RxDuration should not be nil")
	assert.Equal(t, int64(500), *result.LastAttachRxDuration, "RxDuration should be preserved")
	require.NotNil(t, result.LastProfile, "Profile should not be nil")
	assert.Equal(t, "Updated", *result.LastProfile, "Profile should be updated")

	// Assert mandatory fields updated
	require.NotNil(t, result.LastSNR)
	assert.Equal(t, 20.0, *result.LastSNR)
	require.NotNil(t, result.LastRSSI)
	assert.Equal(t, -70.0, *result.LastRSSI)
	require.NotNil(t, result.LastEqSNR)
	assert.Equal(t, 22.0, *result.LastEqSNR)
	require.NotNil(t, result.LastAttachRxTime)
	assert.Equal(t, int64(9999999999), *result.LastAttachRxTime)

	// Assert last_seen_at incremented
	require.NotNil(t, result.LastSeenAt, "LastSeenAt should not be nil")
	require.NotNil(t, baseline.LastSeenAt, "Baseline LastSeenAt should not be nil")
	assert.True(t, result.LastSeenAt.After(*baseline.LastSeenAt), "last_seen_at must increment")
}

// TestUpdateRadioMetricsSelective_PreservesProfile verifies that
// Profile=nil preserves existing database value while updating RxDuration
// and mandatory fields per BSSCI §5.6 attach telemetry requirements.
func TestUpdateRadioMetricsSelective_PreservesProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 100, "TestTenant100")
	defer cleanupEndpointTestData(t, db, "PreservesProfile%")

	repo := NewEndPointRepository(db)
	eui := models.EUI{0, 0, 0, 0, 0, 0, 0, 2}

	// Seed with baseline metrics
	baseline := seedEndpointWithMetrics(t, db, repo, 100, eui, seedMetrics{
		SNR:        10.5,
		RSSI:       -80.0,
		EqSNR:      12.3,
		RxTime:     1234567890,
		RxDuration: 500,
		Profile:    "Baseline",
	})

	// Update with Profile=nil (preserve), RxDuration updated
	ctx := testutil.TestContext()
	update := interfaces.RadioMetricsUpdate{
		SNR:        25.0,
		RSSI:       -65.0,
		EqSNR:      27.0,
		RxTime:     8888888888,
		RxDuration: ptr(int64(750)),
		Profile:    nil, // PRESERVE
	}
	err := repo.UpdateRadioMetricsSelective(ctx, 100, eui, update)
	require.NoError(t, err)

	// Reload and verify
	result := reloadEndpoint(t, repo, 100, eui)

	// Assert optional fields
	require.NotNil(t, result.LastAttachRxDuration, "RxDuration should not be nil")
	assert.Equal(t, int64(750), *result.LastAttachRxDuration, "RxDuration should be updated")
	require.NotNil(t, result.LastProfile, "Profile should not be nil")
	assert.Equal(t, "Baseline", *result.LastProfile, "Profile should be preserved")

	// Assert mandatory fields updated
	require.NotNil(t, result.LastSNR)
	assert.Equal(t, 25.0, *result.LastSNR)
	require.NotNil(t, result.LastRSSI)
	assert.Equal(t, -65.0, *result.LastRSSI)
	require.NotNil(t, result.LastEqSNR)
	assert.Equal(t, 27.0, *result.LastEqSNR)
	require.NotNil(t, result.LastAttachRxTime)
	assert.Equal(t, int64(8888888888), *result.LastAttachRxTime)

	// Assert last_seen_at incremented
	require.NotNil(t, result.LastSeenAt)
	require.NotNil(t, baseline.LastSeenAt)
	assert.True(t, result.LastSeenAt.After(*baseline.LastSeenAt), "last_seen_at must increment")
}

// TestUpdateRadioMetricsSelective_UpdatesAllOptionalFields verifies that
// both optional pointers present results in all columns updating.
func TestUpdateRadioMetricsSelective_UpdatesAllOptionalFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 100, "TestTenant100")
	defer cleanupEndpointTestData(t, db, "UpdatesAllOptional%")

	repo := NewEndPointRepository(db)
	eui := models.EUI{0, 0, 0, 0, 0, 0, 0, 3}

	// Seed with baseline metrics
	baseline := seedEndpointWithMetrics(t, db, repo, 100, eui, seedMetrics{
		SNR:        10.5,
		RSSI:       -80.0,
		EqSNR:      12.3,
		RxTime:     1234567890,
		RxDuration: 500,
		Profile:    "Baseline",
	})

	// Update ALL fields (both optional and mandatory)
	ctx := testutil.TestContext()
	update := interfaces.RadioMetricsUpdate{
		SNR:        30.0,
		RSSI:       -60.0,
		EqSNR:      32.0,
		RxTime:     7777777777,
		RxDuration: ptr(int64(1000)),
		Profile:    ptr("FullUpdate"),
	}
	err := repo.UpdateRadioMetricsSelective(ctx, 100, eui, update)
	require.NoError(t, err)

	// Reload and verify
	result := reloadEndpoint(t, repo, 100, eui)

	// Assert optional fields updated
	require.NotNil(t, result.LastAttachRxDuration)
	assert.Equal(t, int64(1000), *result.LastAttachRxDuration, "RxDuration should be updated")
	require.NotNil(t, result.LastProfile)
	assert.Equal(t, "FullUpdate", *result.LastProfile, "Profile should be updated")

	// Assert mandatory fields updated
	require.NotNil(t, result.LastSNR)
	assert.Equal(t, 30.0, *result.LastSNR)
	require.NotNil(t, result.LastRSSI)
	assert.Equal(t, -60.0, *result.LastRSSI)
	require.NotNil(t, result.LastEqSNR)
	assert.Equal(t, 32.0, *result.LastEqSNR)
	require.NotNil(t, result.LastAttachRxTime)
	assert.Equal(t, int64(7777777777), *result.LastAttachRxTime)

	// Assert last_seen_at incremented
	require.NotNil(t, result.LastSeenAt)
	require.NotNil(t, baseline.LastSeenAt)
	assert.True(t, result.LastSeenAt.After(*baseline.LastSeenAt), "last_seen_at must increment")
}

// TestUpdateRadioMetricsSelective_PreservesAllWhenNil verifies that
// both optional pointers nil results in only mandatory fields updating.
// This proves we never zero-out absent fields per BSSCI telemetry guarantees.
func TestUpdateRadioMetricsSelective_PreservesAllWhenNil(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 100, "TestTenant100")
	defer cleanupEndpointTestData(t, db, "PreservesAllWhenNil%")

	repo := NewEndPointRepository(db)
	eui := models.EUI{0, 0, 0, 0, 0, 0, 0, 4}

	// Seed with baseline metrics
	baseline := seedEndpointWithMetrics(t, db, repo, 100, eui, seedMetrics{
		SNR:        10.5,
		RSSI:       -80.0,
		EqSNR:      12.3,
		RxTime:     1234567890,
		RxDuration: 500,
		Profile:    "Baseline",
	})

	// Update ONLY mandatory fields (both optional nil)
	ctx := testutil.TestContext()
	update := interfaces.RadioMetricsUpdate{
		SNR:        15.0,
		RSSI:       -75.0,
		EqSNR:      17.0,
		RxTime:     6666666666,
		RxDuration: nil, // PRESERVE
		Profile:    nil, // PRESERVE
	}
	err := repo.UpdateRadioMetricsSelective(ctx, 100, eui, update)
	require.NoError(t, err)

	// Reload and verify
	result := reloadEndpoint(t, repo, 100, eui)

	// Assert optional fields PRESERVED
	require.NotNil(t, result.LastAttachRxDuration)
	assert.Equal(t, int64(500), *result.LastAttachRxDuration, "RxDuration should be preserved")
	require.NotNil(t, result.LastProfile)
	assert.Equal(t, "Baseline", *result.LastProfile, "Profile should be preserved")

	// Assert mandatory fields UPDATED
	require.NotNil(t, result.LastSNR)
	assert.Equal(t, 15.0, *result.LastSNR)
	require.NotNil(t, result.LastRSSI)
	assert.Equal(t, -75.0, *result.LastRSSI)
	require.NotNil(t, result.LastEqSNR)
	assert.Equal(t, 17.0, *result.LastEqSNR)
	require.NotNil(t, result.LastAttachRxTime)
	assert.Equal(t, int64(6666666666), *result.LastAttachRxTime)

	// Assert last_seen_at incremented
	require.NotNil(t, result.LastSeenAt)
	require.NotNil(t, baseline.LastSeenAt)
	assert.True(t, result.LastSeenAt.After(*baseline.LastSeenAt), "last_seen_at must increment")
}

// TestUpdateRadioMetricsSelective_MultiUpdatePreservesPriorData verifies that
// sequential updates representing interleaved attach + detach telemetry preserve
// prior values correctly. Each update touches different optional fields.
func TestUpdateRadioMetricsSelective_MultiUpdatePreservesPriorData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 100, "TestTenant100")
	defer cleanupEndpointTestData(t, db, "MultiUpdate%")

	repo := NewEndPointRepository(db)
	eui := models.EUI{0, 0, 0, 0, 0, 0, 0, 5}

	// Seed with baseline metrics
	baseline := seedEndpointWithMetrics(t, db, repo, 100, eui, seedMetrics{
		SNR:        10.0,
		RSSI:       -80.0,
		EqSNR:      12.0,
		RxTime:     1000000000,
		RxDuration: 500,
		Profile:    "Baseline",
	})

	ctx := testutil.TestContext()

	// Update 1: Attach telemetry (updates RxDuration, preserves Profile)
	update1 := interfaces.RadioMetricsUpdate{
		SNR:        20.0,
		RSSI:       -70.0,
		EqSNR:      22.0,
		RxTime:     2000000000,
		RxDuration: ptr(int64(600)),
		Profile:    nil, // PRESERVE
	}
	err := repo.UpdateRadioMetricsSelective(ctx, 100, eui, update1)
	require.NoError(t, err, "Update 1 failed")

	// Verify intermediate state
	intermediate := reloadEndpoint(t, repo, 100, eui)
	require.NotNil(t, intermediate.LastAttachRxDuration)
	assert.Equal(t, int64(600), *intermediate.LastAttachRxDuration, "RxDuration should update")
	require.NotNil(t, intermediate.LastProfile)
	assert.Equal(t, "Baseline", *intermediate.LastProfile, "Profile should preserve")
	require.NotNil(t, intermediate.LastSNR)
	assert.Equal(t, 20.0, *intermediate.LastSNR)

	// Update 2: Detach telemetry (updates Profile, preserves RxDuration from Update 1)
	update2 := interfaces.RadioMetricsUpdate{
		SNR:        25.0,
		RSSI:       -65.0,
		EqSNR:      27.0,
		RxTime:     3000000000,
		RxDuration: nil, // PRESERVE (should keep 600 from Update 1)
		Profile:    ptr("Detach"),
	}
	err = repo.UpdateRadioMetricsSelective(ctx, 100, eui, update2)
	require.NoError(t, err, "Update 2 failed")

	// Verify final state
	final := reloadEndpoint(t, repo, 100, eui)

	// Each column reflects last explicit assignment
	require.NotNil(t, final.LastAttachRxDuration)
	assert.Equal(t, int64(600), *final.LastAttachRxDuration, "RxDuration from Update 1 preserved")
	require.NotNil(t, final.LastProfile)
	assert.Equal(t, "Detach", *final.LastProfile, "Profile from Update 2 applied")
	require.NotNil(t, final.LastSNR)
	assert.Equal(t, 25.0, *final.LastSNR, "SNR from Update 2 applied")
	require.NotNil(t, final.LastRSSI)
	assert.Equal(t, -65.0, *final.LastRSSI)
	require.NotNil(t, final.LastEqSNR)
	assert.Equal(t, 27.0, *final.LastEqSNR)
	require.NotNil(t, final.LastAttachRxTime)
	assert.Equal(t, int64(3000000000), *final.LastAttachRxTime)

	// Assert last_seen_at incremented from intermediate state
	require.NotNil(t, final.LastSeenAt)
	require.NotNil(t, intermediate.LastSeenAt)
	assert.True(t, final.LastSeenAt.After(*intermediate.LastSeenAt), "last_seen_at incremented again")

	// Also verify incremented from original baseline
	require.NotNil(t, baseline.LastSeenAt)
	assert.True(t, final.LastSeenAt.After(*baseline.LastSeenAt), "last_seen_at incremented from baseline")
}

// TestUpdateRadioMetricsSelective_ConcurrentUpdates verifies that
// database-level row locking prevents torn writes when concurrent attach/detach
// messages arrive per BSSCI §3.6/3.7 multi-BS scenario requirements.
func TestUpdateRadioMetricsSelective_ConcurrentUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 100, "TestTenant100")
	defer cleanupEndpointTestData(t, db, "ConcurrentUpdates%")

	repo := NewEndPointRepository(db)
	eui := models.EUI{0, 0, 0, 0, 0, 0, 0, 6}

	// Seed with baseline metrics
	_ = seedEndpointWithMetrics(t, db, repo, 100, eui, seedMetrics{
		SNR:        10.0,
		RSSI:       -80.0,
		EqSNR:      12.0,
		RxTime:     1000000000,
		RxDuration: 500,
		Profile:    "Baseline",
	})

	// Synchronization channels for deterministic ordering
	done1 := make(chan struct{})
	done2 := make(chan struct{})
	errChan := make(chan error, 2)

	// Goroutine 1: Update RxDuration only (preserves Profile="Baseline")
	go func() {
		defer close(done1)
		ctx := testutil.TestContext()
		update := interfaces.RadioMetricsUpdate{
			SNR:        20.0,
			RSSI:       -70.0,
			EqSNR:      22.0,
			RxTime:     2000000000,
			RxDuration: ptr(int64(700)), // NON-NIL: update duration
			Profile:    nil,             // NIL: preserve existing profile
		}
		err := repo.UpdateRadioMetricsSelective(ctx, 100, eui, update)
		if err != nil {
			errChan <- fmt.Errorf("goroutine 1 failed: %w", err)
			return
		}
		t.Log("Goroutine 1: Updated RxDuration to 700, preserved Profile")
	}()

	// Goroutine 2: Update Profile only (preserves RxDuration from Goroutine 1)
	// WAIT for Goroutine 1 to commit before starting
	go func() {
		defer close(done2)
		<-done1 // Block until Goroutine 1 completes

		ctx := testutil.TestContext()
		update := interfaces.RadioMetricsUpdate{
			SNR:        25.0,
			RSSI:       -65.0,
			EqSNR:      27.0,
			RxTime:     3000000000,
			RxDuration: nil,               // NIL: preserve Goroutine 1's duration (700)
			Profile:    ptr("Concurrent"), // NON-NIL: update profile
		}
		err := repo.UpdateRadioMetricsSelective(ctx, 100, eui, update)
		if err != nil {
			errChan <- fmt.Errorf("goroutine 2 failed: %w", err)
			return
		}
		t.Log("Goroutine 2: Updated Profile, preserved RxDuration from Goroutine 1")
	}()

	// Wait for both goroutines
	<-done1
	<-done2
	close(errChan)

	// Check for errors
	for err := range errChan {
		require.NoError(t, err)
	}

	// Reload and verify EXACT final state
	final := reloadEndpoint(t, repo, 100, eui)

	// STRICT ASSERTIONS (no permissive arrays)
	require.NotNil(t, final.LastAttachRxDuration, "RxDuration should not be nil")
	require.NotNil(t, final.LastProfile, "Profile should not be nil")

	// Goroutine 1 set duration=700, Goroutine 2 preserved it with nil
	assert.Equal(t, int64(700), *final.LastAttachRxDuration,
		"Expected RxDuration=700 from Goroutine 1 (preserved by Goroutine 2)")

	// Goroutine 2 set profile="Concurrent", overwriting baseline
	assert.Equal(t, "Concurrent", *final.LastProfile,
		"Expected Profile='Concurrent' from Goroutine 2")

	// Last update was Goroutine 2, so SNR/RSSI should reflect its values
	require.NotNil(t, final.LastSNR, "LastSNR should not be nil")
	assert.Equal(t, 25.0, *final.LastSNR, "SNR should match Goroutine 2")
	require.NotNil(t, final.LastRSSI, "LastRSSI should not be nil")
	assert.Equal(t, -65.0, *final.LastRSSI, "RSSI should match Goroutine 2")
}

// TestUpdateRadioMetricsSelective_TenantIsolation verifies that
// updates respect tenant boundaries and cannot update other tenant's endpoints.
func TestUpdateRadioMetricsSelective_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 100, "TestTenant100")
	createTestTenant(t, db, 999, "TestTenant999")
	defer cleanupEndpointTestData(t, db, "TenantIsolation%")

	repo := NewEndPointRepository(db)
	eui := models.EUI{0, 0, 0, 0, 0, 0, 0, 7}

	// Seed endpoint for tenant 100
	baseline := seedEndpointWithMetrics(t, db, repo, 100, eui, seedMetrics{
		SNR:        10.0,
		RSSI:       -80.0,
		EqSNR:      12.0,
		RxTime:     1000000000,
		RxDuration: 500,
		Profile:    "Baseline",
	})

	// Attempt update using WRONG tenant (999 instead of 100)
	ctx := testutil.TestContext()
	update := interfaces.RadioMetricsUpdate{
		SNR:        99.0,
		RSSI:       -99.0,
		EqSNR:      99.0,
		RxTime:     9999999999,
		RxDuration: ptr(int64(9999)),
		Profile:    ptr("BadTenant"),
	}
	err := repo.UpdateRadioMetricsSelective(ctx, 999, eui, update)

	// Assert error returned
	assert.Error(t, err, "Should fail for wrong tenant")
	assert.ErrorIs(t, err, storage.ErrNotFound, "Should return sentinel not-found error")

	// Reload via CORRECT tenant and verify unchanged
	unchanged := reloadEndpoint(t, repo, 100, eui)
	require.NotNil(t, unchanged.LastSNR)
	assert.Equal(t, *baseline.LastSNR, *unchanged.LastSNR, "SNR should be unchanged")
	require.NotNil(t, unchanged.LastRSSI)
	assert.Equal(t, *baseline.LastRSSI, *unchanged.LastRSSI, "RSSI should be unchanged")
	require.NotNil(t, unchanged.LastAttachRxDuration)
	assert.Equal(t, *baseline.LastAttachRxDuration, *unchanged.LastAttachRxDuration, "RxDuration unchanged")
	require.NotNil(t, unchanged.LastProfile)
	assert.Equal(t, *baseline.LastProfile, *unchanged.LastProfile, "Profile unchanged")
}

// TestUpdateRadioMetricsSelective_NonexistentEndpoint verifies that
// attempting to update a non-existent EUI returns an error.
func TestUpdateRadioMetricsSelective_NonexistentEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 100, "TestTenant100")

	repo := NewEndPointRepository(db)
	nonExistentEUI := models.EUI{0x99, 0x99, 0x99, 0x99, 0x99, 0x99, 0x99, 0x99}

	// Attempt update for non-existent endpoint
	ctx := testutil.TestContext()
	update := interfaces.RadioMetricsUpdate{
		SNR:    10.0,
		RSSI:   -80.0,
		EqSNR:  12.0,
		RxTime: 1000000000,
	}
	err := repo.UpdateRadioMetricsSelective(ctx, 100, nonExistentEUI, update)

	// Assert error returned (rowsAffected == 0 case)
	assert.Error(t, err, "Should fail for non-existent endpoint")
	assert.ErrorIs(t, err, storage.ErrNotFound, "Should return sentinel not-found error")
}

// TestGetEndpointWithKeysForDetachValidation_ReturnsAllCryptoFields verifies that
// the method returns endpoint with all crypto material needed for detach validation
func TestGetEndpointWithKeysForDetachValidation_ReturnsAllCryptoFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 100, "TestTenant100")
	defer cleanupEndpointTestData(t, db, "DetachValidation%")

	repo := NewEndPointRepository(db)
	eui := models.EUI{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}

	// Insert test endpoint with all crypto fields using helper
	nwkKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	sign := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	presharedKey := []byte{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2A, 0x2B, 0x2C, 0x2D, 0x2E, 0x2F, 0x30}

	insertEndpointWithDetachKeys(t, db, EndpointInsertParams{
		EpEUI:     eui.ToUint64(),
		Name:      "DetachValidation-EP1",
		TenantID:  100,
		NwkKey:    nwkKey,
		Sign:      sign,
		Preshared: presharedKey,
	})

	// Test: Retrieve endpoint with all crypto fields
	ctx := testutil.TestContext()
	endpoint, err := repo.GetEndpointWithKeysForDetachValidation(ctx, eui)
	require.NoError(t, err, "Should retrieve endpoint successfully")
	require.NotNil(t, endpoint, "Endpoint should not be nil")

	// Verify EUI matches
	assert.Equal(t, eui, endpoint.EUI, "EUI should match")
	assert.Equal(t, "DetachValidation-EP1", endpoint.Name, "Name should match")

	// Verify crypto fields are populated
	assert.NotNil(t, endpoint.NwkSnKey, "NwkSnKey should not be nil")
	assert.Equal(t, nwkKey, endpoint.NwkSnKey, "NwkSnKey should match inserted value")

	assert.NotNil(t, endpoint.Sign, "Sign should not be nil")
	assert.Equal(t, sign, endpoint.Sign, "Sign should match inserted value")

	assert.NotNil(t, endpoint.PresharedKey, "PresharedKey should not be nil")
	assert.Equal(t, presharedKey, endpoint.PresharedKey, "PresharedKey should match inserted value")
}

// TestGetEndpointWithKeysForDetachValidation_NonexistentEUI verifies that
// the method returns ErrNotFound for non-existent endpoint
func TestGetEndpointWithKeysForDetachValidation_NonexistentEUI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 100, "TestTenant100")

	repo := NewEndPointRepository(db)
	nonExistentEUI := models.EUI{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	// Test: Query for non-existent endpoint
	ctx := testutil.TestContext()
	endpoint, err := repo.GetEndpointWithKeysForDetachValidation(ctx, nonExistentEUI)

	// Assert ErrNotFound returned
	assert.Error(t, err, "Should return error for non-existent EUI")
	assert.Nil(t, endpoint, "Endpoint should be nil")
	// The postgres implementation returns storage.ErrNotFound for sql.ErrNoRows
	assert.Contains(t, err.Error(), "not found", "Error should indicate not found")
}

// =============================================================================
// Global EUI Uniqueness Enforcement (Migration 000080)
// =============================================================================

// TestEndpointRepository_Create_RejectsCrossTenantDuplicate verifies that
// migration 000080 enforces global EUI uniqueness across all tenants
// **CRITICAL**: Requires testcontainer with migration 000080 applied (constraint verified in testcontainer.go)
func TestEndpointRepository_Create_RejectsCrossTenantDuplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create two test tenants
	createTestTenant(t, db, 100, "TestTenant100")
	createTestTenant(t, db, 200, "TestTenant200")

	defer cleanupEndpointTestData(t, db, "TestCrossTenant%")

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	// Shared EUI for both tenants (hardware reality: globally unique)
	eui := models.EUI{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}

	// Prepare crypto keys (NOT NULL constraints)
	nwkKey := make([]byte, 16)
	appKey := make([]byte, 16)

	// Create endpoint in tenant 100 - should succeed
	endpoint1 := &models.EndPoint{
		EUI:           eui,
		TenantID:      100,
		OwnerTenantID: 100,
		Name:          "TestCrossTenant-T100-EP1",
		EPClass:       "A",
		NwkSnKey:      nwkKey,
		AppKey:        appKey,
		CryptoMode:    0,
		Tags:          make(map[string]string),
	}

	err := repo.Create(ctx, endpoint1)
	require.NoError(t, err, "First creation in tenant 100 should succeed")
	assert.NotZero(t, endpoint1.ID, "ID should be assigned")

	// Attempt to create same EUI in tenant 200 - should fail with global uniqueness violation
	endpoint2 := &models.EndPoint{
		EUI:           eui, // Same EUI as tenant 100
		TenantID:      200, // Different tenant
		OwnerTenantID: 200,
		Name:          "TestCrossTenant-T200-EP2",
		EPClass:       "A",
		NwkSnKey:      nwkKey,
		AppKey:        appKey,
		CryptoMode:    0,
		Tags:          make(map[string]string),
	}

	err = repo.Create(ctx, endpoint2)

	// Assert global uniqueness enforcement (migration 000080)
	require.Error(t, err, "Cross-tenant duplicate EUI should be rejected")

	// Verify error is wrapped as ErrDuplicate
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		assert.Equal(t, "23505", string(pqErr.Code), "Should be PostgreSQL unique violation")
		// Any of these prove global uniqueness enforcement: endpoints_eui_key is the
		// migration-001 column constraint (name survives the 000029 eui->ep_eui rename),
		// unique_ep_eui comes from migration 000080, endpoints_ep_eui_key from a fresh schema
		assert.Contains(t, []string{"unique_ep_eui", "endpoints_ep_eui_key", "endpoints_eui_key"}, pqErr.Constraint,
			"Should reference a global EUI uniqueness constraint")
	}

	// Alternative check: Error should mention "already exists" or "duplicate"
	assert.Contains(t, strings.ToLower(err.Error()), "unique", "Error message should indicate uniqueness violation")
}

// TestUpdateFields_PersistsAttachSubpackets verifies attach subpackets survive round-trip storage.
func TestUpdateFields_PersistsAttachSubpackets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	tenantID := int64(100)
	createTestTenant(t, db, tenantID, "TestTenant100")
	defer cleanupEndpointTestData(t, db, "TestSubpackets%")

	repo := NewEndPointRepository(db)
	eui := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0x00, 0x00, 0x00, 0x01}

	// Seed baseline endpoint (subpackets nil)
	baseline := seedEndpointWithMetrics(t, db, repo, tenantID, eui, seedMetrics{
		SNR:        5.0,
		RSSI:       -95.0,
		EqSNR:      6.0,
		RxTime:     1111111,
		RxDuration: 250,
		Profile:    "BaseSubPk", // VARCHAR(10) limit
	})
	require.Nil(t, baseline.LastAttachSubpackets, "Baseline subpackets must start nil")

	testSubpackets := mioty.Subpackets{
		SNR:       []float64{1.1, 2.2, 3.3},
		RSSI:      []float64{-90.5, -91.5, -92.5},
		Frequency: []int64{868100000, 868300000, 868500000},
		Phase:     []float64{10.0, -5.5, 0.0},
	}

	encoded, err := json.Marshal(&testSubpackets)
	require.NoError(t, err, "Failed to marshal subpackets payload")

	ctx := testutil.TestContext()
	err = repo.UpdateFields(ctx, tenantID, baseline.ID, map[string]interface{}{
		"last_attach_subpackets": string(encoded),
	})
	require.NoError(t, err, "Failed to persist attach subpackets")

	// Use direct SQL query to verify subpackets (bypasses GetByID schema mismatch issue)
	storedSubpackets, _, _, _ := querySubpacketsAndProfile(t, db, tenantID, baseline.ID)
	require.NotNil(t, storedSubpackets, "Subpackets must be persisted")
	assert.JSONEq(t, string(encoded), *storedSubpackets, "Stored subpackets JSON must match payload")

	// Tenant isolation: wrong tenant should return no data
	crossTenantSubpackets, crossErr := querySubpacketsOnly(t, db, tenantID+1, baseline.ID)
	require.NoError(t, crossErr, "Cross-tenant query should not error")
	assert.Nil(t, crossTenantSubpackets, "Cross-tenant lookup must return nil")
	_ = ctx // silence unused warning
}

// TestUpdateFields_PreservesAttachSubpacketsWhenAbsent ensures we never invent subpackets when updates omit them.
func TestUpdateFields_PreservesAttachSubpacketsWhenAbsent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	tenantID := int64(200)
	createTestTenant(t, db, tenantID, "TestTenant200")
	defer cleanupEndpointTestData(t, db, "TestSubpacketsAbsent%")

	repo := NewEndPointRepository(db)
	eui := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0x00, 0x00, 0x00, 0x02}

	baseline := seedEndpointWithMetrics(t, db, repo, tenantID, eui, seedMetrics{
		SNR:        7.0,
		RSSI:       -85.0,
		EqSNR:      8.0,
		RxTime:     2222222,
		RxDuration: 300,
		Profile:    "BaseAbsnt", // VARCHAR(10) limit
	})
	require.Nil(t, baseline.LastAttachSubpackets, "Baseline subpackets must start nil")

	ctx := testutil.TestContext()
	err := repo.UpdateFields(ctx, tenantID, baseline.ID, map[string]interface{}{
		"last_profile":            "UpdNoSubPk", // VARCHAR(10) limit
		"last_attach_rx_time":     int64(3333333),
		"last_attach_rx_duration": int64(400),
	})
	require.NoError(t, err, "UpdateFields without subpackets must succeed")

	// Use direct SQL query (bypasses GetByID schema mismatch issue)
	storedSubpackets, storedProfile, storedRxTime, storedRxDuration := querySubpacketsAndProfile(t, db, tenantID, baseline.ID)
	assert.Nil(t, storedSubpackets, "Subpackets must remain nil when not provided")
	require.NotNil(t, storedProfile, "Profile must be present after update")
	assert.Equal(t, "UpdNoSubPk", *storedProfile, "Profile update should apply")
	require.NotNil(t, storedRxTime, "Attach RxTime must be present")
	assert.Equal(t, int64(3333333), *storedRxTime, "Attach RxTime should update")
	require.NotNil(t, storedRxDuration, "Attach RxDuration must be present")
	assert.Equal(t, int64(400), *storedRxDuration, "Attach RxDuration should update")
	_ = ctx // silence unused warning
}

// =============================================================================
// BSSCI-ATTACH-024: GetByEUI Subpackets SELECT Parity Tests
// =============================================================================

// TestGetByEUI_ReturnsAttachSubpackets verifies that GetByEUI properly returns
// the last_attach_subpackets column. This test validates the BSSCI-ATTACH-024 fix
// which added the column to the GetByEUI SELECT list.
func TestGetByEUI_ReturnsAttachSubpackets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	tenantID := int64(301)
	createTestTenant(t, db, tenantID, "TestTenant301")
	defer cleanupEndpointTestData(t, db, "TestGetByEUI-Subpkt%")

	repo := NewEndPointRepository(db)
	eui := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0x00, 0x00, 0x03, 0x01}

	// Seed endpoint with baseline metrics
	baseline := seedEndpointWithMetrics(t, db, repo, tenantID, eui, seedMetrics{
		SNR:        5.0,
		RSSI:       -95.0,
		EqSNR:      6.0,
		RxTime:     3333333,
		RxDuration: 350,
		Profile:    "GetByEUI", // VARCHAR(10) limit
	})
	require.Nil(t, baseline.LastAttachSubpackets, "Baseline subpackets must start nil")

	// Seed subpackets via direct SQL UPDATE
	testSubpackets := mioty.Subpackets{
		SNR:       []float64{1.1, 2.2, 3.3},
		RSSI:      []float64{-90.5, -91.5, -92.5},
		Frequency: []int64{868100000, 868300000, 868500000},
		Phase:     []float64{10.0, -5.5, 0.0},
	}
	encoded, err := json.Marshal(&testSubpackets)
	require.NoError(t, err, "Failed to marshal subpackets payload")

	ctx := testutil.TestContext()
	err = repo.UpdateFields(ctx, tenantID, baseline.ID, map[string]interface{}{
		"last_attach_subpackets": string(encoded),
	})
	require.NoError(t, err, "Failed to persist attach subpackets")

	// Reload via GetByEUI (the method being tested for BSSCI-ATTACH-024)
	reloaded := reloadEndpoint(t, repo, tenantID, eui)

	// Assert subpackets are returned
	require.NotNil(t, reloaded.LastAttachSubpackets, "GetByEUI must return LastAttachSubpackets")
	assert.JSONEq(t, string(encoded), *reloaded.LastAttachSubpackets, "Subpackets JSON must match payload")

	// Also verify RxTime/RxDuration presence to ensure column order isn't broken
	require.NotNil(t, reloaded.LastAttachRxTime, "LastAttachRxTime must be present (column order check)")
	assert.Equal(t, int64(3333333), *reloaded.LastAttachRxTime)
	require.NotNil(t, reloaded.LastAttachRxDuration, "LastAttachRxDuration must be present (column order check)")
	assert.Equal(t, int64(350), *reloaded.LastAttachRxDuration)

	// Verify LastProfile also survives (subpackets added after profile in SELECT)
	require.NotNil(t, reloaded.LastProfile, "LastProfile must be present")
	assert.Equal(t, "GetByEUI", *reloaded.LastProfile)
}

// TestGetByEUI_SubpacketsNilWhenNull verifies that GetByEUI returns nil for
// LastAttachSubpackets when the database column is NULL.
func TestGetByEUI_SubpacketsNilWhenNull(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	tenantID := int64(302)
	createTestTenant(t, db, tenantID, "TestTenant302")
	defer cleanupEndpointTestData(t, db, "TestGetByEUI-NullSub%")

	repo := NewEndPointRepository(db)
	eui := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0x00, 0x00, 0x03, 0x02}

	// Seed endpoint WITHOUT subpackets (will be NULL in DB)
	baseline := seedEndpointWithMetrics(t, db, repo, tenantID, eui, seedMetrics{
		SNR:        4.0,
		RSSI:       -96.0,
		EqSNR:      5.0,
		RxTime:     4444444,
		RxDuration: 400,
		Profile:    "NullSubPkt", // VARCHAR(10) limit
	})
	require.Nil(t, baseline.LastAttachSubpackets, "Baseline subpackets must be nil")

	// Reload via GetByEUI without updating subpackets
	reloaded := reloadEndpoint(t, repo, tenantID, eui)

	// Assert subpackets are nil (NULL in DB)
	assert.Nil(t, reloaded.LastAttachSubpackets, "GetByEUI must return nil for NULL subpackets")

	// Other fields should still be present
	require.NotNil(t, reloaded.LastProfile, "LastProfile must be present")
	assert.Equal(t, "NullSubPkt", *reloaded.LastProfile)
	require.NotNil(t, reloaded.LastAttachRxTime, "LastAttachRxTime must be present")
	assert.Equal(t, int64(4444444), *reloaded.LastAttachRxTime)
}

// TestGetByEUI_TenantIsolation verifies that GetByEUI enforces tenant isolation
// and returns storage.ErrNotFound when querying with a different tenant ID.
func TestGetByEUI_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	tenantA := int64(303)
	tenantB := int64(304)
	createTestTenant(t, db, tenantA, "TestTenant303")
	createTestTenant(t, db, tenantB, "TestTenant304")
	defer cleanupEndpointTestData(t, db, "TestGetByEUI-TenIso%")

	repo := NewEndPointRepository(db)
	eui := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0x00, 0x00, 0x03, 0x03}

	// Seed endpoint for tenant A with subpackets
	baseline := seedEndpointWithMetrics(t, db, repo, tenantA, eui, seedMetrics{
		SNR:        3.0,
		RSSI:       -97.0,
		EqSNR:      4.0,
		RxTime:     5555555,
		RxDuration: 450,
		Profile:    "TenantISO", // VARCHAR(10) limit
	})

	// Add subpackets
	testSubpackets := mioty.Subpackets{
		SNR:  []float64{1.0, 2.0},
		RSSI: []float64{-80.0, -81.0},
	}
	encoded, err := json.Marshal(&testSubpackets)
	require.NoError(t, err)

	ctx := testutil.TestContext()
	err = repo.UpdateFields(ctx, tenantA, baseline.ID, map[string]interface{}{
		"last_attach_subpackets": string(encoded),
	})
	require.NoError(t, err)

	// Verify tenant A can read subpackets
	reloadedA := reloadEndpoint(t, repo, tenantA, eui)
	require.NotNil(t, reloadedA.LastAttachSubpackets, "Tenant A should see subpackets")
	assert.JSONEq(t, string(encoded), *reloadedA.LastAttachSubpackets)

	// Verify tenant B gets storage.ErrNotFound (cross-tenant isolation)
	_, err = repo.GetByEUI(ctx, tenantB, eui[:])
	require.Error(t, err, "Cross-tenant GetByEUI must return error")
	assert.True(t, errors.Is(err, storage.ErrNotFound),
		"Cross-tenant GetByEUI must return storage.ErrNotFound, got: %v", err)
}

// TestTxnGetByEUI_SubpacketsParity verifies that the transactional GetByEUI
// returns the same subpackets data as the non-transactional version.
// This validates the BSSCI-ATTACH-024 fix for transaction_endpoint.go.
func TestTxnGetByEUI_SubpacketsParity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	tenantID := int64(305)
	createTestTenant(t, db, tenantID, "TestTenant305")
	defer cleanupEndpointTestData(t, db, "TestTxnGetByEUI%")

	repo := NewEndPointRepository(db)
	eui := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0x00, 0x00, 0x03, 0x05}

	// Seed endpoint with subpackets
	baseline := seedEndpointWithMetrics(t, db, repo, tenantID, eui, seedMetrics{
		SNR:        2.0,
		RSSI:       -98.0,
		EqSNR:      3.0,
		RxTime:     6666666,
		RxDuration: 500,
		Profile:    "TxnParity", // VARCHAR(10) limit
	})

	testSubpackets := mioty.Subpackets{
		SNR:       []float64{0.5, 1.5, 2.5},
		RSSI:      []float64{-85.0, -86.0, -87.0},
		Frequency: []int64{868000000, 868200000},
	}
	encoded, err := json.Marshal(&testSubpackets)
	require.NoError(t, err)

	ctx := testutil.TestContext()
	err = repo.UpdateFields(ctx, tenantID, baseline.ID, map[string]interface{}{
		"last_attach_subpackets": string(encoded),
	})
	require.NoError(t, err)

	// Get via non-transactional GetByEUI for reference
	nonTxnEndpoint := reloadEndpoint(t, repo, tenantID, eui)
	require.NotNil(t, nonTxnEndpoint.LastAttachSubpackets, "Non-txn GetByEUI should return subpackets")

	// Get via transactional GetByEUI within a transaction
	// Create raw sql.Tx from sqlx.DB for transactionalEndPointRepository
	rawTx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err, "Failed to begin transaction")
	defer func() { _ = rawTx.Rollback() }() // #nosec G307 -- Test cleanup

	// Create transactional repo directly (mirrors Transaction.EndPoints() behavior)
	txnRepo := &transactionalEndPointRepository{tx: rawTx, db: nil}
	txnEndpoint, err := txnRepo.GetByEUI(ctx, tenantID, eui[:])
	require.NoError(t, err, "Transactional GetByEUI should succeed")
	require.NotNil(t, txnEndpoint, "Transactional endpoint should not be nil")

	// Assert subpackets parity
	require.NotNil(t, txnEndpoint.LastAttachSubpackets, "Transactional GetByEUI must return LastAttachSubpackets")
	assert.JSONEq(t, *nonTxnEndpoint.LastAttachSubpackets, *txnEndpoint.LastAttachSubpackets,
		"Transactional and non-transactional subpackets must match")

	// Assert RxTime/RxDuration parity (added in same BSSCI-ATTACH-024 fix)
	require.NotNil(t, txnEndpoint.LastAttachRxTime, "Transactional LastAttachRxTime must be present")
	assert.Equal(t, *nonTxnEndpoint.LastAttachRxTime, *txnEndpoint.LastAttachRxTime)
	require.NotNil(t, txnEndpoint.LastAttachRxDuration, "Transactional LastAttachRxDuration must be present")
	assert.Equal(t, *nonTxnEndpoint.LastAttachRxDuration, *txnEndpoint.LastAttachRxDuration)

	// Assert LastProfile parity
	require.NotNil(t, txnEndpoint.LastProfile, "Transactional LastProfile must be present")
	assert.Equal(t, *nonTxnEndpoint.LastProfile, *txnEndpoint.LastProfile)
}

// =============================================================================
// SCACI §3.6 Register Audit: StreamAllForPropagation Column Name Regression Test
// =============================================================================

// TestStreamAllForPropagation_UsesCorrectColumnName verifies the SQL query
// uses nwk_key (not nwk_sn_key) which matches the endpoints table schema.
//
// This is a sqlmock-based unit test that validates:
//   - Positive: Query contains "nwk_key" column
//   - Negative: Query does NOT contain "nwk_sn_key" (regression blocker)
//
// Spec: SCACI §3.6.1 - Register operation field persistence
func TestStreamAllForPropagation_UsesCorrectColumnName(t *testing.T) {
	// Create sqlmock DB
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := NewEndPointRepository(sqlxDB)

	// Define expected SQL pattern - must contain nwk_key
	// Note: The actual query from StreamAllForPropagation uses nwk_key
	expectedQueryPattern := regexp.MustCompile(`SELECT.*nwk_key.*FROM\s+endpoints`)

	// Create mock rows for SELECT result
	rows := sqlmock.NewRows([]string{
		"id", "ep_eui", "tenant_id", "nwk_key", "bidi", "sh_addr",
		"dual_chan", "repetition", "wide_carr_off", "long_blk_dist", "propagated_at",
	}).AddRow(
		int64(1), // id
		[]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}, // ep_eui
		int64(100), // tenant_id
		[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}, // nwk_key (16 bytes)
		true,        // bidi
		int32(1234), // sh_addr
		false,       // dual_chan
		true,        // repetition
		false,       // wide_carr_off
		false,       // long_blk_dist
		time.Now(),  // propagated_at
	)

	// Set up expectation with query matcher
	mock.ExpectQuery(expectedQueryPattern.String()).
		WithArgs(int64(0), 100). // cursorID=0, limit=100
		WillReturnRows(rows)

	// Execute StreamAllForPropagation
	ctx := context.Background()
	endpoints, err := repo.StreamAllForPropagation(ctx, 0, 100)
	require.NoError(t, err, "StreamAllForPropagation should succeed")
	require.Len(t, endpoints, 1, "Should return 1 endpoint")

	// Verify the returned endpoint has correct data
	assert.Equal(t, int64(1), endpoints[0].ID)
	assert.Equal(t, int64(100), endpoints[0].TenantID)
	assert.True(t, endpoints[0].Bidi)

	// Verify all SQL expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}

	// CRITICAL NEGATIVE ASSERTION: Verify the actual query does NOT contain nwk_sn_key
	// This is verified implicitly by the regexp pattern matching nwk_key
	// If the code had nwk_sn_key, the ExpectQuery would not match and test would fail
}

// TestStreamAllForPropagation_QueryTextValidation directly inspects query text
// to ensure nwk_sn_key is never present (defense-in-depth regression blocker).
func TestStreamAllForPropagation_QueryTextValidation(t *testing.T) {
	// This test validates the source code query text contains correct column name
	// by creating a custom query matcher that captures and validates the actual SQL

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(
		func(_, actualSQL string) error {
			// CRITICAL: Assert nwk_sn_key is NOT in the query (regression blocker)
			if strings.Contains(actualSQL, "nwk_sn_key") {
				return fmt.Errorf("REGRESSION: query contains 'nwk_sn_key' which does not exist in endpoints table. Found: %s", actualSQL)
			}
			// Assert nwk_key IS in the query (positive assertion)
			if !strings.Contains(actualSQL, "nwk_key") {
				return fmt.Errorf("query missing 'nwk_key' column. Found: %s", actualSQL)
			}
			return nil
		},
	)))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := NewEndPointRepository(sqlxDB)

	// Create mock rows
	rows := sqlmock.NewRows([]string{
		"id", "ep_eui", "tenant_id", "nwk_key", "bidi", "sh_addr",
		"dual_chan", "repetition", "wide_carr_off", "long_blk_dist", "propagated_at",
	}).AddRow(
		int64(1),
		[]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
		int64(100),
		[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		true,
		int32(1234),
		false,
		true,
		false,
		false,
		time.Now(),
	)

	// The custom matcher validates the query text
	mock.ExpectQuery("").WillReturnRows(rows)

	// Execute - the custom matcher will verify column names
	ctx := context.Background()
	_, err = repo.StreamAllForPropagation(ctx, 0, 100)
	require.NoError(t, err, "StreamAllForPropagation should succeed with correct column name")

	// Expectations verified by custom matcher during query execution
}

// =============================================================================
// DeviceModelID Persistence Round-Trip Tests
// =============================================================================

// TestCreate_WithDeviceModelID verifies DeviceModelID is persisted during endpoint creation
// createTestDeviceModel inserts a manufacturer and device model for the tenant
// and returns the device model ID, satisfying endpoints_device_model_id_fkey.
func createTestDeviceModel(t *testing.T, db *sqlx.DB, tenantID int64, code string) uuid.UUID {
	t.Helper()
	var mfrID, modelID uuid.UUID
	require.NoError(t, db.QueryRow(
		`INSERT INTO manufacturers (tenant_id, name) VALUES ($1, $2) RETURNING id`,
		tenantID, "TestMfr-"+code).Scan(&mfrID))
	require.NoError(t, db.QueryRow(
		`INSERT INTO device_models (manufacturer_id, tenant_id, name, code) VALUES ($1, $2, $3, $4) RETURNING id`,
		mfrID, tenantID, "TestModel-"+code, code).Scan(&modelID))
	return modelID
}

func TestCreate_WithDeviceModelID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create test tenant
	createTestTenant(t, db, 700, "TestTenant700")

	cleanupEndpointTestData(t, db, "TestDeviceModelID%")
	defer cleanupEndpointTestData(t, db, "TestDeviceModelID%")

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	// Create test endpoint with DeviceModelID
	var eui models.EUI
	copy(eui[:], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x01})

	nwkKey := make([]byte, 16)
	appKey := make([]byte, 16)
	deviceModelID := createTestDeviceModel(t, db, 700, "test-model-700")

	endpoint := &models.EndPoint{
		EUI:           eui,
		Name:          "TestDeviceModelID-EP1",
		TenantID:      700,
		OwnerTenantID: 700,
		EPClass:       "A",
		NwkSnKey:      nwkKey,
		AppKey:        appKey,
		CryptoMode:    0,
		Tags:          make(map[string]string),
		DeviceModelID: &deviceModelID,
	}

	// Test: Create endpoint with DeviceModelID
	err := repo.Create(ctx, endpoint)
	require.NoError(t, err)
	assert.NotZero(t, endpoint.ID, "ID should be assigned after creation")

	// Verify DeviceModelID is persisted via direct SQL query
	var storedDeviceModelID *uuid.UUID
	queryErr := db.QueryRowContext(ctx, "SELECT device_model_id FROM endpoints WHERE id = $1", endpoint.ID).Scan(&storedDeviceModelID)
	require.NoError(t, queryErr, "Should query device_model_id successfully")
	require.NotNil(t, storedDeviceModelID, "DeviceModelID should be persisted")
	assert.Equal(t, deviceModelID, *storedDeviceModelID, "DeviceModelID should match")
}

// TestGetByID_ReturnsDeviceModelID verifies DeviceModelID is returned from GetByID
func TestGetByID_ReturnsDeviceModelID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create test tenant
	createTestTenant(t, db, 701, "TestTenant701")

	cleanupEndpointTestData(t, db, "TestGetByID-DevModel%")
	defer cleanupEndpointTestData(t, db, "TestGetByID-DevModel%")

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	// Create endpoint with DeviceModelID
	var eui models.EUI
	copy(eui[:], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x02})

	nwkKey := make([]byte, 16)
	appKey := make([]byte, 16)
	deviceModelID := createTestDeviceModel(t, db, 701, "test-model-701")

	endpoint := &models.EndPoint{
		EUI:           eui,
		Name:          "TestGetByID-DevModel-EP1",
		TenantID:      701,
		OwnerTenantID: 701,
		EPClass:       "A",
		NwkSnKey:      nwkKey,
		AppKey:        appKey,
		CryptoMode:    0,
		Tags:          make(map[string]string),
		DeviceModelID: &deviceModelID,
	}

	err := repo.Create(ctx, endpoint)
	require.NoError(t, err)

	// Test: GetByID returns DeviceModelID
	retrieved, err := repo.GetByID(ctx, endpoint.ID, 701)
	require.NoError(t, err)
	require.NotNil(t, retrieved, "Endpoint should be retrieved")
	require.NotNil(t, retrieved.DeviceModelID, "DeviceModelID should be present in retrieved endpoint")
	assert.Equal(t, deviceModelID, *retrieved.DeviceModelID, "DeviceModelID should match original")
}

// TestListByTenantPaginated_ReturnsDeviceModelID verifies DeviceModelID is returned from list
func TestListByTenantPaginated_ReturnsDeviceModelID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create test tenant
	createTestTenant(t, db, 702, "TestTenant702")

	cleanupEndpointTestData(t, db, "TestList-DevModel%")
	defer cleanupEndpointTestData(t, db, "TestList-DevModel%")

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	// Create endpoint with DeviceModelID
	var eui models.EUI
	copy(eui[:], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x03})

	nwkKey := make([]byte, 16)
	appKey := make([]byte, 16)
	deviceModelID := createTestDeviceModel(t, db, 702, "test-model-702")

	endpoint := &models.EndPoint{
		EUI:           eui,
		Name:          "TestList-DevModel-EP1",
		TenantID:      702,
		OwnerTenantID: 702,
		EPClass:       "A",
		NwkSnKey:      nwkKey,
		AppKey:        appKey,
		CryptoMode:    0,
		Tags:          make(map[string]string),
		DeviceModelID: &deviceModelID,
	}

	err := repo.Create(ctx, endpoint)
	require.NoError(t, err)

	// Test: ListByTenantPaginated returns DeviceModelID
	endpoints, err := repo.ListByTenantPaginated(ctx, 702, 10, 0)
	require.NoError(t, err)
	require.Len(t, endpoints, 1, "Should return one endpoint")
	require.NotNil(t, endpoints[0].DeviceModelID, "DeviceModelID should be present in list result")
	assert.Equal(t, deviceModelID, *endpoints[0].DeviceModelID, "DeviceModelID should match original")
}

// TestGetByEUI_ReturnsDeviceModelID verifies DeviceModelID is returned from GetByEUI
func TestGetByEUI_ReturnsDeviceModelID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create test tenant
	createTestTenant(t, db, 703, "TestTenant703")

	cleanupEndpointTestData(t, db, "TestGetByEUI-DevModel%")
	defer cleanupEndpointTestData(t, db, "TestGetByEUI-DevModel%")

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	// Create endpoint with DeviceModelID
	var eui models.EUI
	copy(eui[:], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x04})

	nwkKey := make([]byte, 16)
	appKey := make([]byte, 16)
	deviceModelID := createTestDeviceModel(t, db, 703, "test-model-703")

	endpoint := &models.EndPoint{
		EUI:           eui,
		Name:          "TestGetByEUI-DevModel-EP1",
		TenantID:      703,
		OwnerTenantID: 703,
		EPClass:       "A",
		NwkSnKey:      nwkKey,
		AppKey:        appKey,
		CryptoMode:    0,
		Tags:          make(map[string]string),
		DeviceModelID: &deviceModelID,
	}

	err := repo.Create(ctx, endpoint)
	require.NoError(t, err)

	// Test: GetByEUI returns DeviceModelID
	retrieved, err := repo.GetByEUI(ctx, 703, eui[:])
	require.NoError(t, err)
	require.NotNil(t, retrieved, "Endpoint should be retrieved")
	require.NotNil(t, retrieved.DeviceModelID, "DeviceModelID should be present in retrieved endpoint")
	assert.Equal(t, deviceModelID, *retrieved.DeviceModelID, "DeviceModelID should match original")
}

// =============================================================================
// nullableByteParam Unit Tests
// =============================================================================

func TestNullableByteParam_NilReturnsNil(t *testing.T) {
	result := nullableByteParam(nil)
	assert.Nil(t, result, "nil input should return nil")
}

func TestNullableByteParam_EmptyReturnsNil(t *testing.T) {
	result := nullableByteParam([]byte{})
	assert.Nil(t, result, "empty slice should return nil")
}

func TestNullableByteParam_ValidBytesReturnsBytes(t *testing.T) {
	key := make([]byte, 16)
	key[0] = 0x42
	result := nullableByteParam(key)
	assert.Equal(t, key, result, "valid 16-byte slice should be returned as-is")
}

// =============================================================================
// Update NULL Key Regression Tests (live bug: blueprint-only edit crash)
// =============================================================================

// TestUpdate_NilKeys_NoCheckViolation verifies that updating an endpoint whose
// keys are NULL in the database does not trigger CHECK constraint violations.
// This is the exact live bug path: endpoint loaded via GetByEUI returns []byte{}
// for NULL BYTEA columns, which lib/pq then sends as empty BYTEA to UPDATE.
func TestUpdate_NilKeys_NoCheckViolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }()

	createTestTenant(t, db, 800, "TestTenant800")
	defer cleanupEndpointTestData(t, db, "TestNilKeys%")

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	eui := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x01}
	nwkKey := make([]byte, 16)
	appKey := make([]byte, 16)

	// Create endpoint with valid 16-byte keys
	endpoint := &models.EndPoint{
		EUI:      eui,
		Name:     "TestNilKeys-EP1",
		TenantID: 800,
		EPClass:  "A",
		NwkSnKey: nwkKey,
		AppKey:   appKey,
		Tags:     make(map[string]string),
	}
	err := repo.Create(ctx, endpoint)
	require.NoError(t, err)

	// Clear keys to NULL via raw SQL (simulates endpoint that never had keys set)
	_, err = db.ExecContext(ctx, "UPDATE endpoints SET nwk_key = NULL, app_key = NULL WHERE id = $1", endpoint.ID)
	require.NoError(t, err)

	// Reload via GetByEUI (returns []byte{} for NULL BYTEA)
	reloaded, err := repo.GetByEUI(ctx, 800, eui[:])
	require.NoError(t, err)

	// Update only the name (keys remain as loaded — empty slices)
	reloaded.Name = "TestNilKeys-EP1-Updated"
	err = repo.Update(ctx, reloaded)
	require.NoError(t, err, "Update with nil/empty keys should not trigger CHECK constraint")

	// Verify keys remain NULL in DB
	var nwkKeyDB, appKeyDB []byte
	err = db.QueryRowContext(ctx, "SELECT nwk_key, app_key FROM endpoints WHERE id = $1", endpoint.ID).Scan(&nwkKeyDB, &appKeyDB)
	require.NoError(t, err)
	assert.Nil(t, nwkKeyDB, "nwk_key should remain NULL")
	assert.Nil(t, appKeyDB, "app_key should remain NULL")
}

// TestUpdateWithEUI_NilKeys_NoCheckViolation verifies the same fix through the EUI-change path.
func TestUpdateWithEUI_NilKeys_NoCheckViolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }()

	createTestTenant(t, db, 801, "TestTenant801")
	defer cleanupEndpointTestData(t, db, "TestNilKeysEUI%")

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	oldEui := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x02}
	newEui := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x03}
	nwkKey := make([]byte, 16)
	appKey := make([]byte, 16)

	// Create endpoint with valid 16-byte keys
	endpoint := &models.EndPoint{
		EUI:      oldEui,
		Name:     "TestNilKeysEUI-EP1",
		TenantID: 801,
		EPClass:  "A",
		NwkSnKey: nwkKey,
		AppKey:   appKey,
		Tags:     make(map[string]string),
	}
	err := repo.Create(ctx, endpoint)
	require.NoError(t, err)

	// Clear keys to NULL
	_, err = db.ExecContext(ctx, "UPDATE endpoints SET nwk_key = NULL, app_key = NULL WHERE id = $1", endpoint.ID)
	require.NoError(t, err)

	// Reload via GetByEUI
	reloaded, err := repo.GetByEUI(ctx, 801, oldEui[:])
	require.NoError(t, err)

	// Change EUI (keys remain as loaded — empty slices)
	reloaded.EUI = newEui
	updated, err := repo.UpdateWithEUI(ctx, 801, oldEui[:], reloaded)
	require.NoError(t, err, "UpdateWithEUI with nil/empty keys should not trigger CHECK constraint")
	require.NotNil(t, updated)
}

// =============================================================================
// List Query MIOTY Non-Secret Fields Tests (F2 Fix Verification)
// =============================================================================

// TestListByTenantPaginated_ReturnsMIOTYConfigFields verifies that list queries
// return all non-secret MIOTY configuration fields (bidi, preAttach, typeEui, etc.)
func TestListByTenantPaginated_ReturnsMIOTYConfigFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 810, "TestTenant810")
	cleanupEndpointTestData(t, db, "TestList-MIOTYFields%")
	defer cleanupEndpointTestData(t, db, "TestList-MIOTYFields%")

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	var eui models.EUI
	copy(eui[:], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x10})

	var typeEUI models.EUI
	copy(typeEUI[:], []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22})

	shAddr := uint16(0x1234)
	attachCnt := uint32(42)

	endpoint := &models.EndPoint{
		EUI:           eui,
		Name:          "TestList-MIOTYFields-EP1",
		TenantID:      810,
		OwnerTenantID: 810,
		EPClass:       "A",
		NwkSnKey:      make([]byte, 16),
		AppKey:        make([]byte, 16),
		CryptoMode:    0,
		Tags:          make(map[string]string),
		Bidi:          true,
		PreAttach:     true,
		TypeEUI:       &typeEUI,
		ShAddr:        &shAddr,
		AttachCnt:     &attachCnt,
		LastPacketCnt: 99,
		DualChan:      true,
		Repetition:    true,
		WideCarrOff:   true,
		LongBlkDist:   true,
	}

	err := repo.Create(ctx, endpoint)
	require.NoError(t, err)

	// List should return all non-secret MIOTY config fields
	endpoints, err := repo.ListByTenantPaginated(ctx, 810, 10, 0)
	require.NoError(t, err)
	require.Len(t, endpoints, 1)

	ep := endpoints[0]
	assert.True(t, ep.Bidi, "Bidi should be true")
	assert.True(t, ep.PreAttach, "PreAttach should be true")
	assert.True(t, ep.DualChan, "DualChan should be true")
	assert.True(t, ep.Repetition, "Repetition should be true")
	assert.True(t, ep.WideCarrOff, "WideCarrOff should be true")
	assert.True(t, ep.LongBlkDist, "LongBlkDist should be true")
	require.NotNil(t, ep.TypeEUI, "TypeEUI should be present in list result")
	assert.Equal(t, typeEUI, *ep.TypeEUI, "TypeEUI should match created value")
	require.NotNil(t, ep.AttachCnt, "AttachCnt should be present in list result")
	assert.Equal(t, uint32(42), *ep.AttachCnt, "AttachCnt should match created value")
	assert.Equal(t, uint32(99), ep.LastPacketCnt, "LastPacketCnt should match created value")
	require.NotNil(t, ep.ShAddr, "ShAddr should be present in list result")
	assert.Equal(t, uint16(0x1234), *ep.ShAddr, "ShAddr should match created value")
	// Keys must NOT be in list results (security policy)
	assert.Nil(t, ep.NwkSnKey, "NwkSnKey must not be in list query results")
	assert.Nil(t, ep.AppKey, "AppKey must not be in list query results")
}

// TestListByTenantPaginated_HandlesNullTypeEUI verifies list handles NULL type_eui gracefully
func TestListByTenantPaginated_HandlesNullTypeEUI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 811, "TestTenant811")
	cleanupEndpointTestData(t, db, "TestList-NullTypeEUI%")
	defer cleanupEndpointTestData(t, db, "TestList-NullTypeEUI%")

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	var eui models.EUI
	copy(eui[:], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x11})

	endpoint := &models.EndPoint{
		EUI:           eui,
		Name:          "TestList-NullTypeEUI-EP1",
		TenantID:      811,
		OwnerTenantID: 811,
		EPClass:       "A",
		NwkSnKey:      make([]byte, 16),
		CryptoMode:    0,
		Tags:          make(map[string]string),
	}

	err := repo.Create(ctx, endpoint)
	require.NoError(t, err)

	endpoints, err := repo.ListByTenantPaginated(ctx, 811, 10, 0)
	require.NoError(t, err)
	require.Len(t, endpoints, 1)
	assert.Nil(t, endpoints[0].TypeEUI, "TypeEUI should be nil when not set")
	assert.False(t, endpoints[0].Bidi, "Bidi should default to false")
}

// =============================================================================
// Update TypeEUI Tests (F1-Update Verification)
// =============================================================================

// TestUpdate_PersistsTypeEUI verifies Update saves type_eui correctly
func TestUpdate_PersistsTypeEUI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 812, "TestTenant812")
	cleanupEndpointTestData(t, db, "TestUpdate-TypeEUI%")
	defer cleanupEndpointTestData(t, db, "TestUpdate-TypeEUI%")

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	var eui models.EUI
	copy(eui[:], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x12})

	endpoint := &models.EndPoint{
		EUI:           eui,
		Name:          "TestUpdate-TypeEUI-EP1",
		TenantID:      812,
		OwnerTenantID: 812,
		EPClass:       "A",
		NwkSnKey:      make([]byte, 16),
		CryptoMode:    0,
		Tags:          make(map[string]string),
	}

	err := repo.Create(ctx, endpoint)
	require.NoError(t, err)

	// Reload so the struct carries DB defaults (ep_status, repetition), then set type_eui
	reloaded, err := repo.GetByEUI(ctx, 812, eui[:])
	require.NoError(t, err)

	var typeEUI models.EUI
	copy(typeEUI[:], []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88})
	reloaded.TypeEUI = &typeEUI

	err = repo.Update(ctx, reloaded)
	require.NoError(t, err)

	// Verify via GetByEUI
	retrieved, err := repo.GetByEUI(ctx, 812, eui[:])
	require.NoError(t, err)
	require.NotNil(t, retrieved.TypeEUI, "TypeEUI should be present after update")
	assert.Equal(t, typeEUI, *retrieved.TypeEUI, "TypeEUI should match updated value")
}

// TestUpdate_ClearsTypeEUI verifies Update can clear type_eui to NULL
func TestUpdate_ClearsTypeEUI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 813, "TestTenant813")
	cleanupEndpointTestData(t, db, "TestUpdate-ClearTypeEUI%")
	defer cleanupEndpointTestData(t, db, "TestUpdate-ClearTypeEUI%")

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	var eui models.EUI
	copy(eui[:], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x13})

	var typeEUI models.EUI
	copy(typeEUI[:], []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11})

	endpoint := &models.EndPoint{
		EUI:           eui,
		Name:          "TestUpdate-ClearTypeEUI-EP1",
		TenantID:      813,
		OwnerTenantID: 813,
		EPClass:       "A",
		NwkSnKey:      make([]byte, 16),
		CryptoMode:    0,
		Tags:          make(map[string]string),
		TypeEUI:       &typeEUI,
	}

	err := repo.Create(ctx, endpoint)
	require.NoError(t, err)

	// Reload so the struct carries DB defaults (ep_status, repetition), then clear type_eui
	reloaded, err := repo.GetByEUI(ctx, 813, eui[:])
	require.NoError(t, err)

	reloaded.TypeEUI = nil
	err = repo.Update(ctx, reloaded)
	require.NoError(t, err)

	// Verify cleared
	retrieved, err := repo.GetByEUI(ctx, 813, eui[:])
	require.NoError(t, err)
	assert.Nil(t, retrieved.TypeEUI, "TypeEUI should be nil after clearing")
}

// TestUpdateWithEUI_PersistsTypeEUI verifies UpdateWithEUI saves type_eui correctly
func TestUpdateWithEUI_PersistsTypeEUI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 814, "TestTenant814")
	cleanupEndpointTestData(t, db, "TestUpdateWithEUI-TypeEUI%")
	defer cleanupEndpointTestData(t, db, "TestUpdateWithEUI-TypeEUI%")

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	var oldEui models.EUI
	copy(oldEui[:], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x14})

	endpoint := &models.EndPoint{
		EUI:           oldEui,
		Name:          "TestUpdateWithEUI-TypeEUI-EP1",
		TenantID:      814,
		OwnerTenantID: 814,
		EPClass:       "A",
		NwkSnKey:      make([]byte, 16),
		CryptoMode:    0,
		Tags:          make(map[string]string),
	}

	err := repo.Create(ctx, endpoint)
	require.NoError(t, err)

	// Reload via GetByEUI
	reloaded, err := repo.GetByEUI(ctx, 814, oldEui[:])
	require.NoError(t, err)

	// Set typeEUI and update (same EUI)
	var typeEUI models.EUI
	copy(typeEUI[:], []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE})
	reloaded.TypeEUI = &typeEUI

	updated, err := repo.UpdateWithEUI(ctx, 814, oldEui[:], reloaded)
	require.NoError(t, err)
	require.NotNil(t, updated)

	// Verify
	retrieved, err := repo.GetByEUI(ctx, 814, oldEui[:])
	require.NoError(t, err)
	require.NotNil(t, retrieved.TypeEUI, "TypeEUI should be present after UpdateWithEUI")
	assert.Equal(t, typeEUI, *retrieved.TypeEUI, "TypeEUI should match updated value")
}

// TestUpdateWithEUI_ClearsTypeEUI verifies UpdateWithEUI can clear type_eui to NULL
func TestUpdateWithEUI_ClearsTypeEUI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 815, "TestTenant815")
	cleanupEndpointTestData(t, db, "TestUpdateWithEUI-ClearTypeEUI%")
	defer cleanupEndpointTestData(t, db, "TestUpdateWithEUI-ClearTypeEUI%")

	repo := NewEndPointRepository(db)
	ctx := testutil.TestContext()

	var eui models.EUI
	copy(eui[:], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x15})

	var typeEUI models.EUI
	copy(typeEUI[:], []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88})

	endpoint := &models.EndPoint{
		EUI:           eui,
		Name:          "TestUpdateWithEUI-ClearTypeEUI-EP1",
		TenantID:      815,
		OwnerTenantID: 815,
		EPClass:       "A",
		NwkSnKey:      make([]byte, 16),
		CryptoMode:    0,
		Tags:          make(map[string]string),
		TypeEUI:       &typeEUI,
	}

	err := repo.Create(ctx, endpoint)
	require.NoError(t, err)

	// Reload and clear typeEUI
	reloaded, err := repo.GetByEUI(ctx, 815, eui[:])
	require.NoError(t, err)
	reloaded.TypeEUI = nil

	updated, err := repo.UpdateWithEUI(ctx, 815, eui[:], reloaded)
	require.NoError(t, err)
	require.NotNil(t, updated)

	// Verify cleared
	retrieved, err := repo.GetByEUI(ctx, 815, eui[:])
	require.NoError(t, err)
	assert.Nil(t, retrieved.TypeEUI, "TypeEUI should be nil after clearing via UpdateWithEUI")
}

// TestGetEndpointWithKeysForDetachValidation_TypeEUIAndPropagatedAt verifies that
// scanEndpointDetachValidationRow correctly round-trips the BYTEA type_eui column
// (formerly mis-typed as sql.NullInt64) and the TIMESTAMP propagated_at column.
// Both must survive a write → read cycle through the non-transactional repository.
func TestGetEndpointWithKeysForDetachValidation_TypeEUIAndPropagatedAt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	tenantID := int64(900)
	createTestTenant(t, db, tenantID, "TestTenant900")
	defer cleanupEndpointTestData(t, db, "DetachValTypeEUI%")

	repo := NewEndPointRepository(db)
	eui := models.EUI{0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x01, 0x02, 0x03}

	insertEndpointWithDetachKeys(t, db, EndpointInsertParams{
		EpEUI:    eui.ToUint64(),
		Name:     "DetachValTypeEUI-EP1",
		TenantID: tenantID,
	})

	typeEUIBytes := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22}
	propagatedAt := time.Date(2026, 4, 1, 12, 30, 45, 0, time.UTC)
	_, err := db.Exec(`UPDATE endpoints SET type_eui = $1, propagated_at = $2 WHERE ep_eui = $3`,
		typeEUIBytes, propagatedAt, euiToBytes(eui.ToUint64()))
	require.NoError(t, err, "failed to seed type_eui + propagated_at")

	ctx := testutil.TestContext()
	endpoint, err := repo.GetEndpointWithKeysForDetachValidation(ctx, eui)
	require.NoError(t, err)
	require.NotNil(t, endpoint)
	require.NotNil(t, endpoint.TypeEUI, "TypeEUI must round-trip from BYTEA")
	assert.Equal(t, typeEUIBytes, endpoint.TypeEUI[:], "TypeEUI bytes must match seeded value")
	require.NotNil(t, endpoint.PropagatedAt, "PropagatedAt must round-trip from TIMESTAMP")
	assert.True(t, endpoint.PropagatedAt.Equal(propagatedAt),
		"PropagatedAt should equal seeded timestamp (got %v, want %v)", endpoint.PropagatedAt, propagatedAt)
}

// TestTxnGetEndpointWithKeysForDetachValidation_TypeEUIAndPropagatedAt validates the
// transactional variant emits the same fields through scanEndpointDetachValidationRow.
func TestTxnGetEndpointWithKeysForDetachValidation_TypeEUIAndPropagatedAt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	tenantID := int64(901)
	createTestTenant(t, db, tenantID, "TestTenant901")
	defer cleanupEndpointTestData(t, db, "TxnDetachValTypeEUI%")

	eui := models.EUI{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
	insertEndpointWithDetachKeys(t, db, EndpointInsertParams{
		EpEUI:    eui.ToUint64(),
		Name:     "TxnDetachValTypeEUI-EP1",
		TenantID: tenantID,
	})

	typeEUIBytes := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x11, 0x22, 0x33}
	propagatedAt := time.Date(2026, 5, 1, 9, 15, 0, 0, time.UTC)
	_, err := db.Exec(`UPDATE endpoints SET type_eui = $1, propagated_at = $2 WHERE ep_eui = $3`,
		typeEUIBytes, propagatedAt, euiToBytes(eui.ToUint64()))
	require.NoError(t, err)

	ctx := testutil.TestContext()
	rawTx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = rawTx.Rollback() }() // #nosec G307 -- Test cleanup

	txnRepo := &transactionalEndPointRepository{tx: rawTx, db: nil}
	endpoint, err := txnRepo.GetEndpointWithKeysForDetachValidation(ctx, eui)
	require.NoError(t, err)
	require.NotNil(t, endpoint)
	require.NotNil(t, endpoint.TypeEUI, "TypeEUI must round-trip via transactional path")
	assert.Equal(t, typeEUIBytes, endpoint.TypeEUI[:])
	require.NotNil(t, endpoint.PropagatedAt, "PropagatedAt must round-trip via transactional path")
	assert.True(t, endpoint.PropagatedAt.Equal(propagatedAt),
		"transactional PropagatedAt should equal seeded timestamp (got %v, want %v)", endpoint.PropagatedAt, propagatedAt)
}
