package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	deleteTestOwner   = int64(461)
	deleteTestForeign = int64(462)
	deleteTestBsEUI   = uint64(0x70B3D59CD0000461)
	deleteTestBsName  = "delete-returning"
)

// TestBaseStationDeleteByEUI_ReturnsTheRemovedRowOfTheTenant removes the
// station in one statement: another tenant cannot remove it, the owner gets
// the removed row back, and a second delete finds nothing.
func TestBaseStationDeleteByEUI_ReturnsTheRemovedRowOfTheTenant(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	createTestTenant(t, db, deleteTestOwner, "DeleteTestOwner")
	createTestTenant(t, db, deleteTestForeign, "DeleteTestForeign")
	repo := NewBaseStationRepository(db, clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()

	var eui models.EUI
	copy(eui[:], eui64Bytes(deleteTestBsEUI))
	station := &models.BaseStation{
		EUI:              eui,
		TenantID:         deleteTestOwner,
		Name:             deleteTestBsName,
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
	}
	require.NoError(t, repo.Create(ctx, station))
	stored, err := repo.GetByEUI(ctx, deleteTestOwner, eui[:])
	require.NoError(t, err)

	_, err = repo.DeleteByEUI(ctx, deleteTestForeign, eui[:])
	require.ErrorIs(t, err, storage.ErrNotFound, "another tenant's delete matches no row")

	removed, err := repo.DeleteByEUI(ctx, deleteTestOwner, eui[:])
	require.NoError(t, err)
	assert.Equal(t, stored.ID, removed.ID)
	assert.Equal(t, deleteTestBsName, removed.Name)
	assert.Equal(t, eui, removed.EUI)

	_, err = repo.DeleteByEUI(ctx, deleteTestOwner, eui[:])
	require.ErrorIs(t, err, storage.ErrNotFound, "the station is gone")
}
