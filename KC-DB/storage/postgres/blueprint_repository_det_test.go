package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// TestGetByTypeEUI_Determinism validates the tie-break when duplicate type_eui rows exist: default wins, tenant beats System.
func TestGetByTypeEUI_Determinism(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := SetupTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup
	createTestTenant(t, db, 600, "TestTenant600")

	ctx := testutil.TestContext()
	typeEUI := []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0x00, 0x00, 0x94}

	var mfrID, modelID string
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO manufacturers (tenant_id, name) VALUES ($1, $2) RETURNING id`,
		600, "TestDetMfr").Scan(&mfrID))
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO device_models (manufacturer_id, tenant_id, name, code) VALUES ($1, $2, $3, $4) RETURNING id`,
		mfrID, 600, "TestDetModel", "det-model").Scan(&modelID))

	_, err := db.ExecContext(ctx,
		`INSERT INTO blueprints (device_model_id, tenant_id, version, type_eui, spec_json, is_default) VALUES ($1, $2, '1.0.0', $3, '{}', false)`,
		modelID, 600, typeEUI)
	require.NoError(t, err)
	var defaultID string
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO blueprints (device_model_id, tenant_id, version, type_eui, spec_json, is_default) VALUES ($1, $2, '2.0.0', $3, '{}', true) RETURNING id`,
		modelID, 600, typeEUI).Scan(&defaultID))

	repo := NewBlueprintRepository(db)

	bp, err := repo.GetByTypeEUI(ctx, 600, typeEUI)
	require.NoError(t, err)
	require.NotNil(t, bp)
	assert.Equal(t, defaultID, bp.ID.String(), "must deterministically return the default among same-type_eui rows")
	assert.True(t, bp.IsDefault)

	// System blueprint with same type_eui must not shadow the tenant's; it needs its own System model (unique-default-per-model).
	var sysMfrID, sysModelID string
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO manufacturers (tenant_id, is_system, name) VALUES (NULL, true, $1) RETURNING id`,
		"SysDetMfr").Scan(&sysMfrID))
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO device_models (manufacturer_id, tenant_id, is_system, name, code) VALUES ($1, NULL, true, $2, $3) RETURNING id`,
		sysMfrID, "SysDetModel", "sys-det-model").Scan(&sysModelID))
	_, err = db.ExecContext(ctx,
		`INSERT INTO blueprints (device_model_id, tenant_id, is_system, version, type_eui, spec_json, is_default) VALUES ($1, NULL, true, 'sys-1.0.0', $2, '{}', true)`,
		sysModelID, typeEUI)
	require.NoError(t, err)

	bp2, err := repo.GetByTypeEUI(ctx, 600, typeEUI)
	require.NoError(t, err)
	require.NotNil(t, bp2)
	require.NotNil(t, bp2.TenantID)
	assert.Equal(t, int64(600), *bp2.TenantID, "tenant blueprint must win over System (tenant>system tie-break)")
}
