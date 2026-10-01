package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const unknownURLTenant int64 = 7801

var (
	unknownURLStation = models.EUI{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x78, 0x01}
	emptyURLStation   = models.EUI{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x78, 0x02}
)

func newUnknownURLRepository(t *testing.T) *BaseStationRepository {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupBaseStationTestDB(t)
	createTestTenant(t, db, unknownURLTenant, "TestUnknownServiceCenterURL")
	return NewBaseStationRepository(db, clock.SystemClock{}, logger.Get())
}

// A BSSCI station registered while the server knows no reachable Service
// Center URL is stored with a NULL URL.
func TestBaseStationRepository_Create_StoresABSSCIStationWithoutServiceCenterURL(t *testing.T) {
	repo := newUnknownURLRepository(t)
	ctx := testutil.TestContext()

	station := &models.BaseStation{
		EUI: unknownURLStation, TenantID: unknownURLTenant,
		Name: "TestUnknownURL-Create", ConnectionType: models.ConnectionTypeBSSCI,
	}
	require.NoError(t, repo.Create(ctx, station))

	stored, err := repo.GetByEUI(ctx, unknownURLTenant, unknownURLStation[:])
	require.NoError(t, err)
	assert.Nil(t, stored.ServiceCenterURL)
}

// The read backfill clears a stored empty URL to NULL through Update.
func TestBaseStationRepository_Update_ClearsAnEmptyServiceCenterURL(t *testing.T) {
	repo := newUnknownURLRepository(t)
	ctx := testutil.TestContext()

	empty := ""
	station := &models.BaseStation{
		EUI: emptyURLStation, TenantID: unknownURLTenant, Name: "TestUnknownURL-Empty",
		ConnectionType: models.ConnectionTypeBSSCI, ServiceCenterURL: &empty,
	}
	require.NoError(t, repo.Create(ctx, station))

	var unknown *string
	require.NoError(t, repo.Update(ctx, unknownURLTenant, station.ID, map[string]interface{}{
		"service_center_url": unknown,
	}))

	stored, err := repo.GetByEUI(ctx, unknownURLTenant, emptyURLStation[:])
	require.NoError(t, err)
	assert.Nil(t, stored.ServiceCenterURL)
}
