package postgres

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupBaseStationTestDB connects to the test database using testcontainers
func setupBaseStationTestDB(t *testing.T) *sqlx.DB {
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	return db
}

// cleanupBaseStationTestData removes test basestations by name pattern
func cleanupBaseStationTestData(t *testing.T, db *sqlx.DB, namePattern string) {
	_, err := db.Exec("DELETE FROM basestations WHERE name LIKE $1", namePattern)
	if err != nil {
		t.Logf("Warning: cleanup failed: %v", err)
	}
}

// =============================================================================
// Global EUI Uniqueness Enforcement (Migration 000080)
// =============================================================================

// TestBaseStationRepository_Create_RejectsCrossTenantDuplicate verifies that
// migration 000080 enforces global EUI uniqueness across all tenants for base stations
// **CRITICAL**: Requires testcontainer with migration 000080 applied (constraint verified in testcontainer.go)
// testServiceCenterURLPtr satisfies the check_bssci_config constraint for
// bssci-type fixtures (service_center_url must be present).
func testServiceCenterURLPtr() *string {
	url := "tls://kilocenter.local:5000"
	return &url
}

func TestBaseStationRepository_Create_RejectsCrossTenantDuplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupBaseStationTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	// Create two test tenants
	createTestTenant(t, db, 100, "TestTenant100")
	createTestTenant(t, db, 200, "TestTenant200")

	defer cleanupBaseStationTestData(t, db, "TestCrossTenant%")

	repo := NewBaseStationRepository(db)
	ctx := testutil.TestContext()

	// Shared EUI for both tenants (hardware reality: globally unique)
	eui := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22}

	// Create base station in tenant 100 - should succeed
	bs1 := &models.BaseStation{
		EUI:              eui,
		TenantID:         100,
		Name:             "TestCrossTenant-T100-BS1",
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
	}

	err := repo.Create(ctx, bs1)
	require.NoError(t, err, "First creation in tenant 100 should succeed")
	assert.NotZero(t, bs1.ID, "ID should be assigned")

	// Attempt to create same EUI in tenant 200 - should fail with global uniqueness violation
	bs2 := &models.BaseStation{
		EUI:              eui, // Same EUI as tenant 100
		TenantID:         200, // Different tenant
		Name:             "TestCrossTenant-T200-BS2",
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
	}

	err = repo.Create(ctx, bs2)

	// Assert global uniqueness enforcement (migration 000080)
	require.Error(t, err, "Cross-tenant duplicate base station EUI should be rejected")

	// Verify error is wrapped as unique violation
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		assert.Equal(t, "23505", string(pqErr.Code), "Should be PostgreSQL unique violation")
		// Two unique constraints guard bs_eui: the original column constraint
		// (basestations_eui_key, renamed with the column in migration 000060)
		// and unique_bs_eui from migration 000080; either proves global
		// uniqueness enforcement
		assert.Contains(t, []string{"unique_bs_eui", "basestations_eui_key"}, pqErr.Constraint,
			"Should reference a global EUI uniqueness constraint")
	}

	// Alternative check: Error should mention "already exists" or "unique"
	assert.Contains(t, strings.ToLower(err.Error()), "unique", "Error message should indicate uniqueness violation")
}

// TestBaseStationRepository_Create_AllowsSameTenantDifferentEUI verifies that
// the same tenant can create multiple base stations with different EUIs
func TestBaseStationRepository_Create_AllowsSameTenantDifferentEUI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupBaseStationTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	createTestTenant(t, db, 100, "TestTenant100")
	defer cleanupBaseStationTestData(t, db, "TestSameTenant%")

	repo := NewBaseStationRepository(db)
	ctx := testutil.TestContext()

	// Create first base station
	bs1 := &models.BaseStation{
		EUI:              models.EUI{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
		TenantID:         100,
		Name:             "TestSameTenant-BS1",
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
	}

	err := repo.Create(ctx, bs1)
	require.NoError(t, err, "First base station creation should succeed")
	assert.NotZero(t, bs1.ID)

	// Create second base station with different EUI - should succeed
	bs2 := &models.BaseStation{
		EUI:              models.EUI{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22}, // Different EUI
		TenantID:         100,                                                        // Same tenant
		Name:             "TestSameTenant-BS2",
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
	}

	err = repo.Create(ctx, bs2)
	require.NoError(t, err, "Second base station with different EUI should succeed")
	assert.NotZero(t, bs2.ID)
	assert.NotEqual(t, bs1.ID, bs2.ID, "IDs should be different")
}

// =============================================================================
// UpdateEUI Tests
// =============================================================================

// TestBaseStationRepository_UpdateEUI_Success verifies successful EUI update
func TestBaseStationRepository_UpdateEUI_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupBaseStationTestDB(t)
	defer func() { _ = db.Close() }()

	createTestTenant(t, db, 100, "TestTenant100")
	defer cleanupBaseStationTestData(t, db, "TestUpdateEUI%")

	repo := NewBaseStationRepository(db)
	ctx := testutil.TestContext()

	oldEui := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22}
	newEui := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x33}

	// Create base station with old EUI
	bs := &models.BaseStation{
		EUI:              oldEui,
		TenantID:         100,
		Name:             "TestUpdateEUI-BS1",
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
	}

	err := repo.Create(ctx, bs)
	require.NoError(t, err, "Base station creation should succeed")
	originalID := bs.ID

	// Update EUI
	updatedBS, err := repo.UpdateEUI(ctx, 100, oldEui[:], newEui[:])
	require.NoError(t, err, "EUI update should succeed")
	require.NotNil(t, updatedBS)

	// Verify the update
	assert.Equal(t, originalID, updatedBS.ID, "ID should remain the same")
	assert.Equal(t, newEui, updatedBS.EUI, "EUI should be updated")
	assert.Equal(t, "TestUpdateEUI-BS1", updatedBS.Name, "Name should be unchanged")
}

// TestBaseStationRepository_UpdateEUI_ErrAlreadyExists verifies uniqueness enforcement
func TestBaseStationRepository_UpdateEUI_ErrAlreadyExists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupBaseStationTestDB(t)
	defer func() { _ = db.Close() }()

	createTestTenant(t, db, 100, "TestTenant100")
	defer cleanupBaseStationTestData(t, db, "TestUpdateEUIExists%")

	repo := NewBaseStationRepository(db)
	ctx := testutil.TestContext()

	eui1 := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x44}
	eui2 := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x55}

	// Create two base stations with different EUIs
	bs1 := &models.BaseStation{
		EUI:              eui1,
		TenantID:         100,
		Name:             "TestUpdateEUIExists-BS1",
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
	}
	err := repo.Create(ctx, bs1)
	require.NoError(t, err)

	bs2 := &models.BaseStation{
		EUI:              eui2,
		TenantID:         100,
		Name:             "TestUpdateEUIExists-BS2",
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
	}
	err = repo.Create(ctx, bs2)
	require.NoError(t, err)

	// Try to update bs1's EUI to bs2's EUI - should fail with ErrAlreadyExists
	_, err = repo.UpdateEUI(ctx, 100, eui1[:], eui2[:])
	require.Error(t, err, "Updating to existing EUI should fail")
	assert.ErrorIs(t, err, storage.ErrAlreadyExists, "Should return ErrAlreadyExists")
}

// TestBaseStationRepository_UpdateEUI_ErrNotFound verifies tenant isolation
func TestBaseStationRepository_UpdateEUI_ErrNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupBaseStationTestDB(t)
	defer func() { _ = db.Close() }()

	createTestTenant(t, db, 100, "TestTenant100")
	createTestTenant(t, db, 200, "TestTenant200")
	defer cleanupBaseStationTestData(t, db, "TestUpdateEUINotFound%")

	repo := NewBaseStationRepository(db)
	ctx := testutil.TestContext()

	oldEui := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x66}
	newEui := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x77}

	// Create base station in tenant 100
	bs := &models.BaseStation{
		EUI:              oldEui,
		TenantID:         100,
		Name:             "TestUpdateEUINotFound-BS1",
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
	}
	err := repo.Create(ctx, bs)
	require.NoError(t, err)

	// Try to update from tenant 200 - should fail with ErrNotFound (tenant isolation)
	_, err = repo.UpdateEUI(ctx, 200, oldEui[:], newEui[:])
	require.Error(t, err, "Updating from wrong tenant should fail")
	assert.ErrorIs(t, err, storage.ErrNotFound, "Should return ErrNotFound for tenant isolation")
}

// TestBaseStationRepository_UpdateEUI_NonexistentEUI verifies ErrNotFound for missing EUI
func TestBaseStationRepository_UpdateEUI_NonexistentEUI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupBaseStationTestDB(t)
	defer func() { _ = db.Close() }()

	createTestTenant(t, db, 100, "TestTenant100")

	repo := NewBaseStationRepository(db)
	ctx := testutil.TestContext()

	nonexistentEui := models.EUI{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	newEui := models.EUI{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x88}

	// Try to update nonexistent EUI - should fail with ErrNotFound
	_, err := repo.UpdateEUI(ctx, 100, nonexistentEui[:], newEui[:])
	require.Error(t, err, "Updating nonexistent EUI should fail")
	assert.ErrorIs(t, err, storage.ErrNotFound, "Should return ErrNotFound")
}

// =============================================================================
// Location Field Persistence Tests
// =============================================================================

// TestBaseStation_CreateWithLocation verifies location fields round-trip through Create
func TestBaseStation_CreateWithLocation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupBaseStationTestDB(t)
	defer func() { _ = db.Close() }()

	createTestTenant(t, db, 100, "TestTenant100")
	defer cleanupBaseStationTestData(t, db, "TestCreateWithLoc%")

	repo := NewBaseStationRepository(db)
	ctx := testutil.TestContext()

	lat := 48.856600
	lon := 2.352200
	alt := 330.5
	locSource := models.LocationSourceManual
	locUpdated := time.Now().UTC().Truncate(time.Microsecond)

	bs := &models.BaseStation{
		EUI:               models.EUI{0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22, 0x33, 0x44},
		TenantID:          100,
		Name:              "TestCreateWithLoc-BS1",
		ConnectionType:    models.ConnectionTypeBSSCI,
		ServiceCenterURL:  testServiceCenterURLPtr(),
		Latitude:          &lat,
		Longitude:         &lon,
		Altitude:          &alt,
		LocationSource:    &locSource,
		LocationUpdatedAt: &locUpdated,
	}

	err := repo.Create(ctx, bs)
	require.NoError(t, err, "Create with location fields should succeed")
	require.NotZero(t, bs.ID)

	// Read back and verify round-trip
	got, err := repo.GetByID(ctx, 100, bs.ID)
	require.NoError(t, err)

	require.NotNil(t, got.Latitude, "Latitude should be persisted")
	require.NotNil(t, got.Longitude, "Longitude should be persisted")
	require.NotNil(t, got.Altitude, "Altitude should be persisted")
	require.NotNil(t, got.LocationSource, "LocationSource should be persisted")
	require.NotNil(t, got.LocationUpdatedAt, "LocationUpdatedAt should be persisted")

	assert.InDelta(t, lat, *got.Latitude, 0.000001)
	assert.InDelta(t, lon, *got.Longitude, 0.000001)
	assert.InDelta(t, alt, *got.Altitude, 0.01)
	assert.Equal(t, locSource, *got.LocationSource)
	assert.True(t, locUpdated.Equal(*got.LocationUpdatedAt),
		"LocationUpdatedAt should round-trip: want %v, got %v", locUpdated, *got.LocationUpdatedAt)
}

// TestBaseStation_UpdatePreservesUnmodifiedLocation verifies that updating non-location
// fields via DB.UpdateBaseStation does not clobber existing location data
func TestBaseStation_UpdatePreservesUnmodifiedLocation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	sqlxDB := setupBaseStationTestDB(t)
	defer func() { _ = sqlxDB.Close() }()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")
	defer cleanupBaseStationTestData(t, sqlxDB, "TestUpdatePreserve%")

	repo := NewBaseStationRepository(sqlxDB)
	ctx := testutil.TestContext()

	lat := 52.520008
	lon := 13.404954
	alt := 34.0
	locSource := models.LocationSourceGPS
	locUpdated := time.Now().UTC().Truncate(time.Microsecond)

	bs := &models.BaseStation{
		EUI:               models.EUI{0xDD, 0xEE, 0xFF, 0x11, 0x22, 0x33, 0x44, 0x55},
		TenantID:          100,
		Name:              "TestUpdatePreserve-BS1",
		ConnectionType:    models.ConnectionTypeBSSCI,
		ServiceCenterURL:  testServiceCenterURLPtr(),
		Latitude:          &lat,
		Longitude:         &lon,
		Altitude:          &alt,
		LocationSource:    &locSource,
		LocationUpdatedAt: &locUpdated,
	}

	err := repo.Create(ctx, bs)
	require.NoError(t, err)
	require.NotZero(t, bs.ID)

	// Fetch the stored row
	stored, err := repo.GetByID(ctx, 100, bs.ID)
	require.NoError(t, err)

	// Build update model with only name changed (all other fields from stored)
	stored.Name = "TestUpdatePreserve-Renamed"

	// Use the DB wrapper's UpdateBaseStation which builds the update map
	wrapper := &DB{sqlxDB: sqlxDB}
	_, err = wrapper.UpdateBaseStation(ctx, stored)
	require.NoError(t, err, "UpdateBaseStation should succeed")

	// Re-read and assert location fields are unchanged
	updated, err := repo.GetByID(ctx, 100, bs.ID)
	require.NoError(t, err)

	assert.Equal(t, "TestUpdatePreserve-Renamed", updated.Name)

	require.NotNil(t, updated.Latitude)
	require.NotNil(t, updated.Longitude)
	require.NotNil(t, updated.Altitude)
	require.NotNil(t, updated.LocationSource)
	require.NotNil(t, updated.LocationUpdatedAt)

	assert.InDelta(t, lat, *updated.Latitude, 0.000001)
	assert.InDelta(t, lon, *updated.Longitude, 0.000001)
	assert.InDelta(t, alt, *updated.Altitude, 0.01)
	assert.Equal(t, locSource, *updated.LocationSource)
	assert.True(t, locUpdated.Equal(*updated.LocationUpdatedAt))
}

// =============================================================================
// ListAllLocations Tests
// =============================================================================

// TestBaseStationRepository_ListAllLocations verifies that ListAllLocations returns
// base stations with coordinates from all tenants, excluding those without coordinates.
func TestBaseStationRepository_ListAllLocations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupBaseStationTestDB(t)
	defer func() { _ = db.Close() }()

	createTestTenant(t, db, 100, "TestTenant100")
	createTestTenant(t, db, 200, "TestTenant200")
	defer cleanupBaseStationTestData(t, db, "TestListAllLoc%")

	repo := NewBaseStationRepository(db)
	ctx := testutil.TestContext()

	lat1 := 48.1351
	lon1 := 11.5820
	lat2 := 52.5200
	lon2 := 13.4050

	// Tenant 100: base station WITH coordinates
	bs1 := &models.BaseStation{
		EUI:              models.EUI{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		TenantID:         100,
		Name:             "TestListAllLoc-A-WithCoords",
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
		Latitude:         &lat1,
		Longitude:        &lon1,
	}
	err := repo.Create(ctx, bs1)
	require.NoError(t, err)

	// Tenant 200: base station WITH coordinates
	bs2 := &models.BaseStation{
		EUI:              models.EUI{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18},
		TenantID:         200,
		Name:             "TestListAllLoc-B-WithCoords",
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
		Latitude:         &lat2,
		Longitude:        &lon2,
	}
	err = repo.Create(ctx, bs2)
	require.NoError(t, err)

	// Tenant 100: base station WITHOUT coordinates
	bs3 := &models.BaseStation{
		EUI:              models.EUI{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28},
		TenantID:         100,
		Name:             "TestListAllLoc-C-NoCoords",
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
	}
	err = repo.Create(ctx, bs3)
	require.NoError(t, err)

	// Execute
	stations, err := repo.ListAllLocations(ctx)
	require.NoError(t, err)

	// Filter to our test data (other tests may have created base stations with coords)
	var testStations []*models.BaseStation
	for _, s := range stations {
		if strings.HasPrefix(s.Name, "TestListAllLoc-") {
			testStations = append(testStations, s)
		}
	}

	require.Len(t, testStations, 2, "Should return exactly 2 base stations with coordinates")

	// Verify ordering (by tenant_id, name) — tenant 100 first, then 200
	assert.Equal(t, int64(100), testStations[0].TenantID)
	assert.Equal(t, "TestListAllLoc-A-WithCoords", testStations[0].Name)
	assert.Equal(t, int64(200), testStations[1].TenantID)
	assert.Equal(t, "TestListAllLoc-B-WithCoords", testStations[1].Name)

	// Verify coordinate values
	require.NotNil(t, testStations[0].Latitude)
	require.NotNil(t, testStations[0].Longitude)
	assert.InDelta(t, lat1, *testStations[0].Latitude, 0.0001)
	assert.InDelta(t, lon1, *testStations[0].Longitude, 0.0001)
}
