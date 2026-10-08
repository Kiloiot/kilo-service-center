package postgres

import (
	"errors"
	"testing"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// TestWithSavepoint_RecoversFromDuplicate proves the collision-retry flow the
// blueprint slug generator depends on: PostgreSQL aborts the whole
// transaction on a failed statement, so a duplicate-code insert must run
// inside a savepoint for the retry with a suffixed slug to succeed in the
// same transaction.
func TestWithSavepoint_RecoversFromDuplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	sqlxDB := SetupTestDB(t)
	defer func() { _ = sqlxDB.Close() }() // #nosec G307 -- Test cleanup
	createTestTenant(t, sqlxDB, 620, "TestTenant620")

	ctx := testutil.TestContext()

	var mfrID uuid.UUID
	require.NoError(t, sqlxDB.QueryRowContext(ctx,
		`INSERT INTO manufacturers (tenant_id, name) VALUES ($1, $2) RETURNING id`,
		620, "TestSavepointMfr").Scan(&mfrID))

	logger.Initialize("error", "json")
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: logger.Get()}

	tx, err := db.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	newParams := func(code string) *models.DeviceModelCreateParams {
		return &models.DeviceModelCreateParams{
			TenantID:       620,
			ManufacturerID: mfrID,
			Name:           "TestSavepointModel",
			Code:           code,
		}
	}

	// First insert takes the slug.
	require.NoError(t, tx.WithSavepoint(ctx, func() error {
		_, createErr := tx.DeviceModels().Create(ctx, newParams("sp-model"))
		return createErr
	}))

	// The colliding insert fails inside its savepoint without aborting the
	// transaction.
	err = tx.WithSavepoint(ctx, func() error {
		_, createErr := tx.DeviceModels().Create(ctx, newParams("sp-model"))
		return createErr
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, storage.ErrDuplicateKey),
		"the duplicate must surface as ErrDuplicateKey for the retry loop, got %v", err)

	// The retry with the suffixed slug succeeds in the SAME transaction - the
	// behavior an aborted transaction (25P02) would make impossible.
	require.NoError(t, tx.WithSavepoint(ctx, func() error {
		_, createErr := tx.DeviceModels().Create(ctx, newParams("sp-model-2"))
		return createErr
	}))
	require.NoError(t, tx.Commit())

	var count int
	require.NoError(t, sqlxDB.QueryRowContext(ctx,
		`SELECT count(*) FROM device_models WHERE tenant_id = 620 AND name = 'TestSavepointModel'`).Scan(&count))
	assert.Equal(t, 2, count, "both successful inserts must be committed")
}
