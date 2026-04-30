package postgres

import (
	"testing"

	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/testutil"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManufacturerRepository_List_CaseInsensitiveOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	defer func() { _ = db.Close() }()

	const tenantID int64 = 500

	createTestTenant(t, db, tenantID, "TestManufacturerOrderTenant")

	// Insert manufacturers in non-alphabetical order with mixed case
	names := []string{"Zebra", "alpha", "Weptech", "test1"}
	for _, name := range names {
		_, err := db.Exec(`
			INSERT INTO manufacturers (id, tenant_id, name, is_verified, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, false, NOW(), NOW())
		`, tenantID, name)
		require.NoError(t, err, "Failed to insert manufacturer %q", name)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM manufacturers WHERE tenant_id = $1", tenantID)
	}()

	repo := NewManufacturerRepository(db)
	ctx := testutil.TestContext()

	result, err := repo.List(ctx, &models.ManufacturerListParams{
		TenantID: tenantID,
	})
	require.NoError(t, err)
	require.Len(t, result, 4)

	expected := []string{"alpha", "test1", "Weptech", "Zebra"}
	actual := make([]string, len(result))
	for i, m := range result {
		actual[i] = m.Name
	}
	assert.Equal(t, expected, actual, "Manufacturers should be sorted case-insensitively")
}

func TestManufacturerRepository_Transactional_List_CaseInsensitiveOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	defer func() { _ = db.Close() }()

	const tenantID int64 = 500

	createTestTenant(t, db, tenantID, "TestManufacturerTxOrderTenant")

	names := []string{"Zebra", "alpha", "Weptech", "test1"}
	for _, name := range names {
		_, err := db.Exec(`
			INSERT INTO manufacturers (id, tenant_id, name, is_verified, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, false, NOW(), NOW())
		`, tenantID, name)
		require.NoError(t, err, "Failed to insert manufacturer %q", name)
	}
	defer func() {
		_, _ = db.Exec("DELETE FROM manufacturers WHERE tenant_id = $1", tenantID)
	}()

	logger.Initialize("error", "json")
	pgDB := &DB{conn: db.DB, log: logger.Get()}

	ctx := testutil.TestContext()
	tx, err := pgDB.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	result, err := tx.Manufacturers().List(ctx, &models.ManufacturerListParams{
		TenantID: tenantID,
	})
	require.NoError(t, err)
	require.Len(t, result, 4)

	expected := []string{"alpha", "test1", "Weptech", "Zebra"}
	actual := make([]string, len(result))
	for i, m := range result {
		actual[i] = m.Name
	}
	assert.Equal(t, expected, actual, "Transactional manufacturers should be sorted case-insensitively")
}
