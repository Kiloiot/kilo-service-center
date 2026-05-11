package adapters

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupAdapterTestDB connects to the test database
func setupAdapterTestDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Connect("postgres", "postgres://kilocenter:changeme@localhost:5433/kilocenter_test?sslmode=disable")
	if err != nil {
		t.Skipf("Cannot connect to test database: %v", err)
	}
	return db
}

// cleanupAdapterTestData removes test tenants by name pattern
func cleanupAdapterTestData(t *testing.T, db *sqlx.DB, namePattern string) {
	_, err := db.Exec("DELETE FROM tenants WHERE name LIKE $1", namePattern)
	if err != nil {
		t.Logf("Warning: cleanup failed: %v", err)
	}
}

// TestTenantStoreAdapter_ListTenants_AllStatuses verifies adapter list without filter
func TestTenantStoreAdapter_ListTenants_AllStatuses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAdapterTestDB(t)
	defer db.Close() //nolint:errcheck // test cleanup //nolint:errcheck // test cleanup

	cleanupAdapterTestData(t, db, "AdapterTestList%")
	defer cleanupAdapterTestData(t, db, "AdapterTestList%")

	// Insert test tenants
	_, err := db.Exec(`
		INSERT INTO tenants (name, status, created_at, updated_at)
		VALUES
			('AdapterTestList-Active', 'active', NOW(), NOW()),
			('AdapterTestList-Inactive', 'inactive', NOW(), NOW())
	`)
	require.NoError(t, err)

	adapter := NewTenantStoreAdapter(db)
	ctx := testutil.TestContext()

	// Test: List all tenants (empty filter)
	tenants, err := adapter.ListTenants(ctx, "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tenants), 2)

	// Verify both statuses included
	statuses := make(map[string]bool)
	for _, tenant := range tenants {
		if tenant.Name == "AdapterTestList-Active" || tenant.Name == "AdapterTestList-Inactive" {
			statuses[tenant.Status] = true
		}
	}
	assert.True(t, statuses[models.TenantStatusActive], "Should find active tenant")
	assert.True(t, statuses[models.TenantStatusInactive], "Should find inactive tenant")
}

// TestTenantStoreAdapter_ListTenants_StatusFilter verifies filtering
func TestTenantStoreAdapter_ListTenants_StatusFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAdapterTestDB(t)
	defer db.Close() //nolint:errcheck // test cleanup //nolint:errcheck // test cleanup

	cleanupAdapterTestData(t, db, "AdapterTestFilter%")
	defer cleanupAdapterTestData(t, db, "AdapterTestFilter%")

	// Insert test tenants
	_, err := db.Exec(`
		INSERT INTO tenants (name, status, created_at, updated_at)
		VALUES
			('AdapterTestFilter-1', 'active', NOW(), NOW()),
			('AdapterTestFilter-2', 'inactive', NOW(), NOW())
	`)
	require.NoError(t, err)

	adapter := NewTenantStoreAdapter(db)
	ctx := testutil.TestContext()

	// Test: Filter by active
	tenants, err := adapter.ListTenants(ctx, "active")
	require.NoError(t, err)

	for _, tenant := range tenants {
		assert.Equal(t, "active", tenant.Status)
	}
}

// TestTenantStoreAdapter_CreateTenant_Success verifies creation via adapter
func TestTenantStoreAdapter_CreateTenant_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAdapterTestDB(t)
	defer db.Close() //nolint:errcheck // test cleanup //nolint:errcheck // test cleanup

	cleanupAdapterTestData(t, db, "AdapterTestCreate%")
	defer cleanupAdapterTestData(t, db, "AdapterTestCreate%")

	adapter := NewTenantStoreAdapter(db)
	ctx := testutil.TestContext()

	description := "Adapter test tenant"

	// Test: Create tenant via adapter
	tenant, err := adapter.CreateTenant(ctx, "AdapterTestCreate-Tenant", &description)
	require.NoError(t, err)
	assert.NotZero(t, tenant.ID)
	assert.Equal(t, "AdapterTestCreate-Tenant", tenant.Name)
	assert.NotNil(t, tenant.Description)
	assert.Equal(t, description, *tenant.Description)
	assert.Equal(t, "active", tenant.Status)
}

// TestTenantStoreAdapter_GetTenantByID_Success verifies retrieval via adapter
func TestTenantStoreAdapter_GetTenantByID_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAdapterTestDB(t)
	defer db.Close() //nolint:errcheck // test cleanup

	cleanupAdapterTestData(t, db, "AdapterTestGet%")
	defer cleanupAdapterTestData(t, db, "AdapterTestGet%")

	adapter := NewTenantStoreAdapter(db)
	ctx := testutil.TestContext()

	// Create test tenant
	created, err := adapter.CreateTenant(ctx, "AdapterTestGet-Tenant", nil)
	require.NoError(t, err)

	// Test: Get by ID via adapter
	tenant, err := adapter.GetTenantByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, tenant.ID)
	assert.Equal(t, "AdapterTestGet-Tenant", tenant.Name)
}

// TestTenantStoreAdapter_GetTenantByID_NotFound verifies error handling
func TestTenantStoreAdapter_GetTenantByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAdapterTestDB(t)
	defer db.Close() //nolint:errcheck // test cleanup

	adapter := NewTenantStoreAdapter(db)
	ctx := testutil.TestContext()

	// Test: Non-existent ID should return error
	_, err := adapter.GetTenantByID(ctx, 999999999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestTenantStoreAdapter_UpdateTenant_Success verifies update via adapter
func TestTenantStoreAdapter_UpdateTenant_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAdapterTestDB(t)
	defer db.Close() //nolint:errcheck // test cleanup

	cleanupAdapterTestData(t, db, "AdapterTestUpdate%")
	defer cleanupAdapterTestData(t, db, "AdapterTestUpdate%")

	adapter := NewTenantStoreAdapter(db)
	ctx := testutil.TestContext()

	// Create test tenant
	tenant, err := adapter.CreateTenant(ctx, "AdapterTestUpdate-Original", nil)
	require.NoError(t, err)

	// Test: Update via adapter
	newName := "AdapterTestUpdate-Changed"
	newDesc := "Updated description"
	updated, err := adapter.UpdateTenant(ctx, tenant.ID, &newName, &newDesc)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
	assert.NotNil(t, updated.Description)
	assert.Equal(t, newDesc, *updated.Description)
}

// TestTenantStoreAdapter_UpdateTenant_NoFields verifies error on empty update
func TestTenantStoreAdapter_UpdateTenant_NoFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAdapterTestDB(t)
	defer db.Close() //nolint:errcheck // test cleanup

	cleanupAdapterTestData(t, db, "AdapterTestEmpty%")
	defer cleanupAdapterTestData(t, db, "AdapterTestEmpty%")

	adapter := NewTenantStoreAdapter(db)
	ctx := testutil.TestContext()

	// Create test tenant
	tenant, err := adapter.CreateTenant(ctx, "AdapterTestEmpty-Tenant", nil)
	require.NoError(t, err)

	// Test: Update with no fields should fail
	_, err = adapter.UpdateTenant(ctx, tenant.ID, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields to update")
}

// TestTenantStoreAdapter_SetTenantStatus_Activate verifies activation
func TestTenantStoreAdapter_SetTenantStatus_Activate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAdapterTestDB(t)
	defer db.Close() //nolint:errcheck // test cleanup

	cleanupAdapterTestData(t, db, "AdapterTestActivate%")
	defer cleanupAdapterTestData(t, db, "AdapterTestActivate%")

	adapter := NewTenantStoreAdapter(db)
	ctx := testutil.TestContext()

	// Create and deactivate tenant
	tenant, err := adapter.CreateTenant(ctx, "AdapterTestActivate-Tenant", nil)
	require.NoError(t, err)
	err = adapter.SetTenantStatus(ctx, tenant.ID, false)
	require.NoError(t, err)

	// Test: Activate via adapter
	err = adapter.SetTenantStatus(ctx, tenant.ID, true)
	require.NoError(t, err)

	// Verify status changed
	updated, err := adapter.GetTenantByID(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "active", updated.Status)
}

// TestTenantStoreAdapter_SetTenantStatus_Deactivate verifies deactivation
func TestTenantStoreAdapter_SetTenantStatus_Deactivate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAdapterTestDB(t)
	defer db.Close() //nolint:errcheck // test cleanup

	cleanupAdapterTestData(t, db, "AdapterTestDeactivate%")
	defer cleanupAdapterTestData(t, db, "AdapterTestDeactivate%")

	adapter := NewTenantStoreAdapter(db)
	ctx := testutil.TestContext()

	// Create tenant (defaults to active)
	tenant, err := adapter.CreateTenant(ctx, "AdapterTestDeactivate-Tenant", nil)
	require.NoError(t, err)

	// Test: Deactivate via adapter
	err = adapter.SetTenantStatus(ctx, tenant.ID, false)
	require.NoError(t, err)

	// Verify status changed
	updated, err := adapter.GetTenantByID(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, models.TenantStatusInactive, updated.Status)
}

// TestTenantStoreAdapter_ErrorWrapping verifies error messages are wrapped
func TestTenantStoreAdapter_ErrorWrapping(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAdapterTestDB(t)
	defer db.Close() //nolint:errcheck // test cleanup

	adapter := NewTenantStoreAdapter(db)
	ctx := testutil.TestContext()

	// Test: Error should contain adapter prefix
	_, err := adapter.GetTenantByID(ctx, 999999999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant adapter")
}
