package postgres

import (
	"testing"

	"github.com/kilocenter/KC-Core/pkg/testutil"
	"github.com/kilocenter/KC-DB/storage/models"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTenantTestDB connects to the test database using testcontainers
func setupTenantTestDB(t *testing.T) *sqlx.DB {
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	return db
}

// cleanupTenantTestData removes test tenants by name pattern
func cleanupTenantTestData(t *testing.T, db *sqlx.DB, namePattern string) {
	_, err := db.Exec("DELETE FROM tenants WHERE name LIKE $1", namePattern)
	if err != nil {
		t.Logf("Warning: cleanup failed: %v", err)
	}
}

// TestListTenants_AllStatuses verifies listing all tenants without filter
func TestListTenants_AllStatuses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTenantTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	// Clean up test data
	cleanupTenantTestData(t, db, "TestListAll%")
	defer cleanupTenantTestData(t, db, "TestListAll%")

	// Insert test tenants with different statuses
	_, err := db.Exec(`
		INSERT INTO tenants (name, description, status, created_at, updated_at)
		VALUES
			('TestListAll-Active', 'Active tenant', 'active', NOW(), NOW()),
			('TestListAll-Inactive', 'Inactive tenant', 'inactive', NOW(), NOW())
	`)
	require.NoError(t, err)

	repo := NewTenantRepository(db)
	ctx := testutil.TestContext()

	// Test: List all tenants (no status filter)
	tenants, err := repo.ListTenants(ctx, "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tenants), 2, "Should have at least 2 test tenants")

	// Verify test tenants are included
	var foundActive, foundInactive bool
	for _, tenant := range tenants {
		if tenant.Name == "TestListAll-Active" {
			foundActive = true
			assert.Equal(t, models.TenantStatusActive, tenant.Status)
		}
		if tenant.Name == "TestListAll-Inactive" {
			foundInactive = true
			assert.Equal(t, models.TenantStatusInactive, tenant.Status)
		}
	}
	assert.True(t, foundActive, "Should find active test tenant")
	assert.True(t, foundInactive, "Should find inactive test tenant")
}

// TestListTenants_FilterActive verifies filtering by active status
func TestListTenants_FilterActive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTenantTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	cleanupTenantTestData(t, db, "TestFilterActive%")
	defer cleanupTenantTestData(t, db, "TestFilterActive%")

	// Insert test tenants
	_, err := db.Exec(`
		INSERT INTO tenants (name, status, created_at, updated_at)
		VALUES
			('TestFilterActive-1', 'active', NOW(), NOW()),
			('TestFilterActive-2', 'inactive', NOW(), NOW())
	`)
	require.NoError(t, err)

	repo := NewTenantRepository(db)
	ctx := testutil.TestContext()

	// Test: Filter by active status
	tenants, err := repo.ListTenants(ctx, "active")
	require.NoError(t, err)

	// Verify only active tenants returned
	for _, tenant := range tenants {
		assert.Equal(t, "active", tenant.Status, "All returned tenants should be active")
	}

	// Verify test tenant is included
	found := false
	for _, tenant := range tenants {
		if tenant.Name == "TestFilterActive-1" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find active test tenant")
}

// TestListTenants_FilterInactive verifies filtering by inactive status
func TestListTenants_FilterInactive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTenantTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	cleanupTenantTestData(t, db, "TestFilterInactive%")
	defer cleanupTenantTestData(t, db, "TestFilterInactive%")

	// Insert test tenants
	_, err := db.Exec(`
		INSERT INTO tenants (name, status, created_at, updated_at)
		VALUES
			('TestFilterInactive-1', 'active', NOW(), NOW()),
			('TestFilterInactive-2', 'inactive', NOW(), NOW())
	`)
	require.NoError(t, err)

	repo := NewTenantRepository(db)
	ctx := testutil.TestContext()

	// Test: Filter by inactive status
	tenants, err := repo.ListTenants(ctx, models.TenantStatusInactive)
	require.NoError(t, err)

	// Verify only inactive tenants returned
	for _, tenant := range tenants {
		assert.Equal(t, models.TenantStatusInactive, tenant.Status, "All returned tenants should be inactive")
	}

	// Verify test tenant is included
	found := false
	for _, tenant := range tenants {
		if tenant.Name == "TestFilterInactive-2" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find inactive test tenant")
}

// TestCreateTenant_Success verifies successful tenant creation
func TestCreateTenant_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTenantTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	cleanupTenantTestData(t, db, "TestCreateSuccess%")
	defer cleanupTenantTestData(t, db, "TestCreateSuccess%")

	repo := NewTenantRepository(db)
	ctx := testutil.TestContext()

	description := "Test tenant description"

	// Test: Create tenant with description
	tenant, err := repo.CreateTenant(ctx, "TestCreateSuccess-WithDesc", &description)
	require.NoError(t, err)
	assert.NotZero(t, tenant.ID, "Tenant ID should be assigned")
	assert.Equal(t, "TestCreateSuccess-WithDesc", tenant.Name)
	assert.NotNil(t, tenant.Description)
	assert.Equal(t, description, *tenant.Description)
	assert.Equal(t, "active", tenant.Status, "New tenant should default to active")
	assert.False(t, tenant.CreatedAt.IsZero())
	assert.False(t, tenant.UpdatedAt.IsZero())

	// Test: Create tenant without description
	tenant2, err := repo.CreateTenant(ctx, "TestCreateSuccess-NoDesc", nil)
	require.NoError(t, err)
	assert.NotZero(t, tenant2.ID)
	assert.Equal(t, "TestCreateSuccess-NoDesc", tenant2.Name)
	assert.Nil(t, tenant2.Description, "Description should be nil when not provided")
}

// TestCreateTenant_DuplicateName verifies unique name constraint
func TestCreateTenant_DuplicateName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTenantTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	cleanupTenantTestData(t, db, "TestCreateDuplicate%")
	defer cleanupTenantTestData(t, db, "TestCreateDuplicate%")

	repo := NewTenantRepository(db)
	ctx := testutil.TestContext()

	// Create first tenant
	_, err := repo.CreateTenant(ctx, "TestCreateDuplicate-Name", nil)
	require.NoError(t, err)

	// Test: Attempt to create tenant with duplicate name
	_, err = repo.CreateTenant(ctx, "TestCreateDuplicate-Name", nil)
	require.Error(t, err, "Should fail on duplicate name")
	assert.Contains(t, err.Error(), "unique")
}

// TestGetTenantByID_Success verifies retrieval by ID
func TestGetTenantByID_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTenantTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	cleanupTenantTestData(t, db, "TestGetByID%")
	defer cleanupTenantTestData(t, db, "TestGetByID%")

	repo := NewTenantRepository(db)
	ctx := testutil.TestContext()

	// Create test tenant
	created, err := repo.CreateTenant(ctx, "TestGetByID-Tenant", nil)
	require.NoError(t, err)

	// Test: Get tenant by ID
	tenant, err := repo.GetTenantByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, tenant.ID)
	assert.Equal(t, "TestGetByID-Tenant", tenant.Name)
	assert.Equal(t, "active", tenant.Status)
}

// TestGetTenantByID_NotFound verifies error for non-existent ID
func TestGetTenantByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTenantTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	repo := NewTenantRepository(db)
	ctx := testutil.TestContext()

	// Test: Non-existent tenant ID should return error
	_, err := repo.GetTenantByID(ctx, 999999999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestUpdateTenant_NameOnly verifies updating name field
func TestUpdateTenant_NameOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTenantTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	cleanupTenantTestData(t, db, "TestUpdateName%")
	defer cleanupTenantTestData(t, db, "TestUpdateName%")

	repo := NewTenantRepository(db)
	ctx := testutil.TestContext()

	// Create test tenant
	tenant, err := repo.CreateTenant(ctx, "TestUpdateName-Original", nil)
	require.NoError(t, err)

	// Test: Update name only
	updates := map[string]interface{}{
		"name": "TestUpdateName-Changed",
	}
	updated, err := repo.UpdateTenant(ctx, tenant.ID, updates)
	require.NoError(t, err)
	assert.Equal(t, "TestUpdateName-Changed", updated.Name)
	assert.Nil(t, updated.Description, "Description should remain nil")
	assert.True(t, updated.UpdatedAt.After(updated.CreatedAt), "UpdatedAt should be newer")
}

// TestUpdateTenant_DescriptionOnly verifies updating description field
func TestUpdateTenant_DescriptionOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTenantTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	cleanupTenantTestData(t, db, "TestUpdateDesc%")
	defer cleanupTenantTestData(t, db, "TestUpdateDesc%")

	repo := NewTenantRepository(db)
	ctx := testutil.TestContext()

	// Create test tenant
	tenant, err := repo.CreateTenant(ctx, "TestUpdateDesc-Tenant", nil)
	require.NoError(t, err)
	originalName := tenant.Name

	// Test: Update description only
	newDesc := "Updated description"
	updates := map[string]interface{}{
		"description": newDesc,
	}
	updated, err := repo.UpdateTenant(ctx, tenant.ID, updates)
	require.NoError(t, err)
	assert.Equal(t, originalName, updated.Name, "Name should remain unchanged")
	assert.NotNil(t, updated.Description)
	assert.Equal(t, newDesc, *updated.Description)
}

// TestUpdateTenant_BothFields verifies updating both name and description
func TestUpdateTenant_BothFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTenantTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	cleanupTenantTestData(t, db, "TestUpdateBoth%")
	defer cleanupTenantTestData(t, db, "TestUpdateBoth%")

	repo := NewTenantRepository(db)
	ctx := testutil.TestContext()

	// Create test tenant
	tenant, err := repo.CreateTenant(ctx, "TestUpdateBoth-Original", nil)
	require.NoError(t, err)

	// Test: Update both name and description
	newDesc := "New description"
	updates := map[string]interface{}{
		"name":        "TestUpdateBoth-Changed",
		"description": newDesc,
	}
	updated, err := repo.UpdateTenant(ctx, tenant.ID, updates)
	require.NoError(t, err)
	assert.Equal(t, "TestUpdateBoth-Changed", updated.Name)
	assert.NotNil(t, updated.Description)
	assert.Equal(t, newDesc, *updated.Description)
}

// TestUpdateTenant_EmptyMap verifies error when no fields provided
func TestUpdateTenant_EmptyMap(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTenantTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	cleanupTenantTestData(t, db, "TestUpdateEmpty%")
	defer cleanupTenantTestData(t, db, "TestUpdateEmpty%")

	repo := NewTenantRepository(db)
	ctx := testutil.TestContext()

	// Create test tenant
	tenant, err := repo.CreateTenant(ctx, "TestUpdateEmpty-Tenant", nil)
	require.NoError(t, err)

	// Test: Update with empty map should fail
	updates := map[string]interface{}{}
	_, err = repo.UpdateTenant(ctx, tenant.ID, updates)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields to update")
}

// TestSetStatus_Activate verifies activating a tenant
func TestSetStatus_Activate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTenantTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	cleanupTenantTestData(t, db, "TestSetActivate%")
	defer cleanupTenantTestData(t, db, "TestSetActivate%")

	repo := NewTenantRepository(db)
	ctx := testutil.TestContext()

	// Create tenant and deactivate it
	tenant, err := repo.CreateTenant(ctx, "TestSetActivate-Tenant", nil)
	require.NoError(t, err)
	err = repo.SetStatus(ctx, tenant.ID, false)
	require.NoError(t, err)

	// Test: Activate tenant
	err = repo.SetStatus(ctx, tenant.ID, true)
	require.NoError(t, err)

	// Verify status changed
	updated, err := repo.GetTenantByID(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "active", updated.Status)
}

// TestSetStatus_Deactivate verifies deactivating a tenant
func TestSetStatus_Deactivate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTenantTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	cleanupTenantTestData(t, db, "TestSetDeactivate%")
	defer cleanupTenantTestData(t, db, "TestSetDeactivate%")

	repo := NewTenantRepository(db)
	ctx := testutil.TestContext()

	// Create tenant (defaults to active)
	tenant, err := repo.CreateTenant(ctx, "TestSetDeactivate-Tenant", nil)
	require.NoError(t, err)

	// Test: Deactivate tenant
	err = repo.SetStatus(ctx, tenant.ID, false)
	require.NoError(t, err)

	// Verify status changed
	updated, err := repo.GetTenantByID(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, models.TenantStatusInactive, updated.Status)
}

// TestSetStatus_NotFound verifies error for non-existent tenant
func TestSetStatus_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTenantTestDB(t)
	defer func() {
		_ = db.Close() // Best effort close
	}()

	repo := NewTenantRepository(db)
	ctx := testutil.TestContext()

	// Test: Setting status on non-existent tenant should fail
	err := repo.SetStatus(ctx, 999999999, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
