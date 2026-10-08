package postgres

import (
	"errors"
	"testing"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// TestBlueprintFamilyNotFoundSentinel pins the interface contract of the
// blueprint, manufacturer, and device-model repositories: a missing row is
// reported as storage.ErrRecordNotFound (the sentinel the blueprint service
// maps to its NotFound errors), never as storage.ErrNotFound. A repository
// returning the wrong sentinel silently converts every missing-row result
// into a gRPC Internal error.
func TestBlueprintFamilyNotFoundSentinel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := SetupTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup
	createTestTenant(t, db, 610, "TestTenant610")

	ctx := testutil.TestContext()
	missing := uuid.New()

	_, err := NewManufacturerRepository(db, logger.Get()).GetByID(ctx, 610, missing)
	require.Error(t, err)
	assert.True(t, errors.Is(err, storage.ErrRecordNotFound),
		"manufacturer GetByID must report ErrRecordNotFound per its contract, got %v", err)

	_, err = NewDeviceModelRepository(db, logger.Get()).GetByID(ctx, 610, missing)
	require.Error(t, err)
	assert.True(t, errors.Is(err, storage.ErrRecordNotFound),
		"device model GetByID must report ErrRecordNotFound per its contract, got %v", err)

	_, err = NewBlueprintRepository(db).GetByID(ctx, 610, missing)
	require.Error(t, err)
	assert.True(t, errors.Is(err, storage.ErrRecordNotFound),
		"blueprint GetByID must report ErrRecordNotFound per its contract, got %v", err)
}
