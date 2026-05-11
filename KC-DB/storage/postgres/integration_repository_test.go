// Package postgres provides integration repository tests.
package postgres

import (
	"encoding/json"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupIntegrationTestDB connects to the test database using testcontainers
func setupIntegrationTestDB(t *testing.T) *sqlx.DB {
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	return db
}

// setupIntegrationTestData sets up prerequisite data for integration tests
func setupIntegrationTestData(t *testing.T, db *sqlx.DB) (tenantID int64, orgID uuid.UUID) {
	orgID = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	tenantID = 200

	// Create test tenant
	createTestTenant(t, db, tenantID, "Integration Test Tenant")

	// Clean up any existing test data
	_, _ = db.Exec("DELETE FROM integrations WHERE org_id = $1", orgID)
	_, _ = db.Exec("DELETE FROM organizations WHERE org_id = $1", orgID)

	// Insert test organization
	_, err := db.Exec(`
		INSERT INTO organizations (org_id, tenant_id, name, state, created_at, updated_at)
		VALUES ($1, $2, 'Integration Test Org', 'active', NOW(), NOW())
	`, orgID, tenantID)
	require.NoError(t, err)

	return tenantID, orgID
}

// TestIntegrationRepository_Create_Success tests successful integration creation
func TestIntegrationRepository_Create_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupIntegrationTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	tenantID, orgID := setupIntegrationTestData(t, db)
	defer func() {
		_, _ = db.Exec("DELETE FROM integrations WHERE org_id = $1", orgID)
		_, _ = db.Exec("DELETE FROM organizations WHERE org_id = $1", orgID)
	}()

	repo := NewIntegrationRepository(db)
	ctx := testutil.TestContext()

	// Create integration
	desc := "Test integration description"
	config := json.RawMessage(`{"url": "https://example.com/webhook"}`)
	integration := &models.Integration{
		OrgID:          orgID,
		TenantID:       tenantID,
		Name:           "Test HTTP Integration",
		Description:    &desc,
		Type:           models.IntegrationTypeHTTP,
		Config:         config,
		DeliveryFormat: models.IntegrationDeliveryFormatJSON,
		Status:         models.IntegrationStatusActive,
	}

	err := repo.Create(ctx, integration)
	require.NoError(t, err)
	assert.NotZero(t, integration.ID)
	assert.NotZero(t, integration.CreatedAt)
	assert.NotZero(t, integration.UpdatedAt)
}

// TestIntegrationRepository_GetByID_Success tests successful retrieval by ID
func TestIntegrationRepository_GetByID_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupIntegrationTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	tenantID, orgID := setupIntegrationTestData(t, db)
	defer func() {
		_, _ = db.Exec("DELETE FROM integrations WHERE org_id = $1", orgID)
		_, _ = db.Exec("DELETE FROM organizations WHERE org_id = $1", orgID)
	}()

	repo := NewIntegrationRepository(db)
	ctx := testutil.TestContext()

	// Create integration first
	desc := "Test description"
	integration := &models.Integration{
		OrgID:          orgID,
		TenantID:       tenantID,
		Name:           "Test Get Integration",
		Description:    &desc,
		Type:           models.IntegrationTypeMQTT,
		Config:         json.RawMessage(`{}`),
		DeliveryFormat: models.IntegrationDeliveryFormatJSON,
		Status:         models.IntegrationStatusActive,
	}

	err := repo.Create(ctx, integration)
	require.NoError(t, err)

	// Test retrieval
	retrieved, err := repo.GetByID(ctx, integration.ID, tenantID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, integration.ID, retrieved.ID)
	assert.Equal(t, "Test Get Integration", retrieved.Name)
	assert.Equal(t, models.IntegrationTypeMQTT, retrieved.Type)
}

// TestIntegrationRepository_GetByID_NotFound tests retrieval of non-existent integration
func TestIntegrationRepository_GetByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupIntegrationTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	_, _ = setupIntegrationTestData(t, db)

	repo := NewIntegrationRepository(db)
	ctx := testutil.TestContext()

	// Test retrieval of non-existent ID
	_, err := repo.GetByID(ctx, 99999, 200)
	require.Error(t, err)
}

// TestIntegrationRepository_ListByTenant_Success tests listing integrations by tenant
func TestIntegrationRepository_ListByTenant_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupIntegrationTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	tenantID, orgID := setupIntegrationTestData(t, db)
	defer func() {
		_, _ = db.Exec("DELETE FROM integrations WHERE org_id = $1", orgID)
		_, _ = db.Exec("DELETE FROM organizations WHERE org_id = $1", orgID)
	}()

	repo := NewIntegrationRepository(db)
	ctx := testutil.TestContext()

	// Create multiple integrations
	for i := 0; i < 3; i++ {
		integration := &models.Integration{
			OrgID:          orgID,
			TenantID:       tenantID,
			Name:           "List Test " + string(rune('A'+i)),
			Type:           models.IntegrationTypeHTTP,
			Config:         json.RawMessage(`{}`),
			DeliveryFormat: models.IntegrationDeliveryFormatJSON,
			Status:         models.IntegrationStatusActive,
		}
		err := repo.Create(ctx, integration)
		require.NoError(t, err)
	}

	// Test listing
	integrations, count, err := repo.ListByTenant(ctx, tenantID, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(3))
	assert.GreaterOrEqual(t, len(integrations), 3)
}

// TestIntegrationRepository_Update_Success tests successful integration update
func TestIntegrationRepository_Update_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupIntegrationTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	tenantID, orgID := setupIntegrationTestData(t, db)
	defer func() {
		_, _ = db.Exec("DELETE FROM integrations WHERE org_id = $1", orgID)
		_, _ = db.Exec("DELETE FROM organizations WHERE org_id = $1", orgID)
	}()

	repo := NewIntegrationRepository(db)
	ctx := testutil.TestContext()

	// Create integration first
	integration := &models.Integration{
		OrgID:          orgID,
		TenantID:       tenantID,
		Name:           "Update Test",
		Type:           models.IntegrationTypeHTTP,
		Config:         json.RawMessage(`{}`),
		DeliveryFormat: models.IntegrationDeliveryFormatJSON,
		Status:         models.IntegrationStatusActive,
	}
	err := repo.Create(ctx, integration)
	require.NoError(t, err)

	// Update integration
	integration.Name = "Updated Name"
	integration.Status = models.IntegrationStatusPaused
	err = repo.Update(ctx, integration)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.GetByID(ctx, integration.ID, tenantID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, models.IntegrationStatusPaused, updated.Status)
}

// TestIntegrationRepository_Delete_Success tests successful integration deletion
func TestIntegrationRepository_Delete_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupIntegrationTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	tenantID, orgID := setupIntegrationTestData(t, db)
	defer func() {
		_, _ = db.Exec("DELETE FROM organizations WHERE org_id = $1", orgID)
	}()

	repo := NewIntegrationRepository(db)
	ctx := testutil.TestContext()

	// Create integration first
	integration := &models.Integration{
		OrgID:          orgID,
		TenantID:       tenantID,
		Name:           "Delete Test",
		Type:           models.IntegrationTypeHTTP,
		Config:         json.RawMessage(`{}`),
		DeliveryFormat: models.IntegrationDeliveryFormatJSON,
		Status:         models.IntegrationStatusActive,
	}
	err := repo.Create(ctx, integration)
	require.NoError(t, err)

	// Delete integration
	err = repo.Delete(ctx, integration.ID, tenantID)
	require.NoError(t, err)

	// Verify deletion
	_, err = repo.GetByID(ctx, integration.ID, tenantID)
	require.Error(t, err)
}
