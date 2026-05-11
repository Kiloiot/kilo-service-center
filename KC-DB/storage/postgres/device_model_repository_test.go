package postgres

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceModelRepository_ListByManufacturer_CaseInsensitiveOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	defer func() { _ = db.Close() }()

	const tenantID int64 = 501

	createTestTenant(t, db, tenantID, "TestDeviceModelOrderTenant")

	// Create a manufacturer to attach device models to
	manufacturerID := uuid.New()
	_, err := db.Exec(`
		INSERT INTO manufacturers (id, tenant_id, name, is_verified, created_at, updated_at)
		VALUES ($1, $2, 'TestMfr', false, NOW(), NOW())
	`, manufacturerID, tenantID)
	require.NoError(t, err, "Failed to insert test manufacturer")
	defer func() {
		_, _ = db.Exec("DELETE FROM device_models WHERE tenant_id = $1", tenantID)
		_, _ = db.Exec("DELETE FROM manufacturers WHERE tenant_id = $1", tenantID)
	}()

	// Insert device models in non-alphabetical order with mixed case
	type modelEntry struct {
		name string
		code string
	}
	deviceModels := []modelEntry{
		{"Zeta-Sensor", "zeta-sensor"},
		{"alpha-device", "alpha-device"},
		{"Mega-Node", "mega-node"},
		{"beta-unit", "beta-unit"},
	}
	for _, m := range deviceModels {
		_, err := db.Exec(`
			INSERT INTO device_models (id, manufacturer_id, tenant_id, name, code, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW())
		`, manufacturerID, tenantID, m.name, m.code)
		require.NoError(t, err, "Failed to insert device model %q", m.name)
	}

	repo := NewDeviceModelRepository(db)
	ctx := testutil.TestContext()

	result, err := repo.ListByManufacturer(ctx, tenantID, manufacturerID, 0, 0)
	require.NoError(t, err)
	require.Len(t, result, 4)

	expected := []string{"alpha-device", "beta-unit", "Mega-Node", "Zeta-Sensor"}
	actual := make([]string, len(result))
	for i, m := range result {
		actual[i] = m.Name
	}
	assert.Equal(t, expected, actual, "Device models should be sorted case-insensitively")
}

func TestDeviceModelRepository_ListWithManufacturer_CaseInsensitiveOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	defer func() { _ = db.Close() }()

	const tenantID int64 = 501

	createTestTenant(t, db, tenantID, "TestDeviceModelListWithMfrTenant")

	manufacturerID := uuid.New()
	_, err := db.Exec(`
		INSERT INTO manufacturers (id, tenant_id, name, is_verified, created_at, updated_at)
		VALUES ($1, $2, 'TestMfr', false, NOW(), NOW())
	`, manufacturerID, tenantID)
	require.NoError(t, err, "Failed to insert test manufacturer")
	defer func() {
		_, _ = db.Exec("DELETE FROM device_models WHERE tenant_id = $1", tenantID)
		_, _ = db.Exec("DELETE FROM manufacturers WHERE tenant_id = $1", tenantID)
	}()

	type modelEntry struct {
		name string
		code string
	}
	deviceModels := []modelEntry{
		{"Zeta-Sensor", "zeta-sensor"},
		{"alpha-device", "alpha-device"},
		{"Mega-Node", "mega-node"},
		{"beta-unit", "beta-unit"},
	}
	for _, m := range deviceModels {
		_, err := db.Exec(`
			INSERT INTO device_models (id, manufacturer_id, tenant_id, name, code, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW())
		`, manufacturerID, tenantID, m.name, m.code)
		require.NoError(t, err, "Failed to insert device model %q", m.name)
	}

	repo := NewDeviceModelRepository(db)
	ctx := testutil.TestContext()

	result, err := repo.ListWithManufacturer(ctx, &models.DeviceModelListParams{TenantID: tenantID})
	require.NoError(t, err)
	require.Len(t, result, 4)

	expected := []string{"alpha-device", "beta-unit", "Mega-Node", "Zeta-Sensor"}
	actual := make([]string, len(result))
	for i, m := range result {
		actual[i] = m.Name
	}
	assert.Equal(t, expected, actual, "Device models should be sorted case-insensitively")

	// Verify the JOIN populates manufacturer name
	for _, m := range result {
		assert.Equal(t, "TestMfr", m.ManufacturerName, "ManufacturerName should be populated from JOIN")
	}
}

func TestDeviceModelRepository_Transactional_ListByManufacturer_CaseInsensitiveOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	defer func() { _ = db.Close() }()

	const tenantID int64 = 501

	createTestTenant(t, db, tenantID, "TestDeviceModelTxListByMfrTenant")

	manufacturerID := uuid.New()
	_, err := db.Exec(`
		INSERT INTO manufacturers (id, tenant_id, name, is_verified, created_at, updated_at)
		VALUES ($1, $2, 'TestMfr', false, NOW(), NOW())
	`, manufacturerID, tenantID)
	require.NoError(t, err, "Failed to insert test manufacturer")
	defer func() {
		_, _ = db.Exec("DELETE FROM device_models WHERE tenant_id = $1", tenantID)
		_, _ = db.Exec("DELETE FROM manufacturers WHERE tenant_id = $1", tenantID)
	}()

	type modelEntry struct {
		name string
		code string
	}
	deviceModels := []modelEntry{
		{"Zeta-Sensor", "zeta-sensor"},
		{"alpha-device", "alpha-device"},
		{"Mega-Node", "mega-node"},
		{"beta-unit", "beta-unit"},
	}
	for _, m := range deviceModels {
		_, err := db.Exec(`
			INSERT INTO device_models (id, manufacturer_id, tenant_id, name, code, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW())
		`, manufacturerID, tenantID, m.name, m.code)
		require.NoError(t, err, "Failed to insert device model %q", m.name)
	}

	logger.Initialize("error", "json")
	pgDB := &DB{conn: db.DB, log: logger.Get()}

	ctx := testutil.TestContext()
	tx, err := pgDB.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	result, err := tx.DeviceModels().ListByManufacturer(ctx, tenantID, manufacturerID, 0, 0)
	require.NoError(t, err)
	require.Len(t, result, 4)

	expected := []string{"alpha-device", "beta-unit", "Mega-Node", "Zeta-Sensor"}
	actual := make([]string, len(result))
	for i, m := range result {
		actual[i] = m.Name
	}
	assert.Equal(t, expected, actual, "Transactional device models should be sorted case-insensitively")
}

func TestDeviceModelRepository_Transactional_ListWithManufacturer_CaseInsensitiveOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	defer func() { _ = db.Close() }()

	const tenantID int64 = 501

	createTestTenant(t, db, tenantID, "TestDeviceModelTxListWithMfrTenant")

	manufacturerID := uuid.New()
	_, err := db.Exec(`
		INSERT INTO manufacturers (id, tenant_id, name, is_verified, created_at, updated_at)
		VALUES ($1, $2, 'TestMfr', false, NOW(), NOW())
	`, manufacturerID, tenantID)
	require.NoError(t, err, "Failed to insert test manufacturer")
	defer func() {
		_, _ = db.Exec("DELETE FROM device_models WHERE tenant_id = $1", tenantID)
		_, _ = db.Exec("DELETE FROM manufacturers WHERE tenant_id = $1", tenantID)
	}()

	type modelEntry struct {
		name string
		code string
	}
	deviceModels := []modelEntry{
		{"Zeta-Sensor", "zeta-sensor"},
		{"alpha-device", "alpha-device"},
		{"Mega-Node", "mega-node"},
		{"beta-unit", "beta-unit"},
	}
	for _, m := range deviceModels {
		_, err := db.Exec(`
			INSERT INTO device_models (id, manufacturer_id, tenant_id, name, code, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW())
		`, manufacturerID, tenantID, m.name, m.code)
		require.NoError(t, err, "Failed to insert device model %q", m.name)
	}

	logger.Initialize("error", "json")
	pgDB := &DB{conn: db.DB, log: logger.Get()}

	ctx := testutil.TestContext()
	tx, err := pgDB.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	result, err := tx.DeviceModels().ListWithManufacturer(ctx, &models.DeviceModelListParams{TenantID: tenantID})
	require.NoError(t, err)
	require.Len(t, result, 4)

	expected := []string{"alpha-device", "beta-unit", "Mega-Node", "Zeta-Sensor"}
	actual := make([]string, len(result))
	for i, m := range result {
		actual[i] = m.Name
	}
	assert.Equal(t, expected, actual, "Transactional device models should be sorted case-insensitively")

	// Verify the JOIN populates manufacturer name
	for _, m := range result {
		assert.Equal(t, "TestMfr", m.ManufacturerName, "ManufacturerName should be populated from JOIN")
	}
}
