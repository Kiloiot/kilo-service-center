package basestation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const adapterPostgresTenant int64 = 7802

var adapterPostgresStation = [8]byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x78, 0x03}

// Reading a BSSCI station stored with an empty URL, while the server knows no
// reachable URL, leaves the stored URL NULL in the database.
func TestGetBaseStation_ClearsAStoredEmptyServiceCenterURLInTheDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := testsupport.SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	_, err := db.Exec(`INSERT INTO tenants (id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', NOW(), NOW())`, adapterPostgresTenant, t.Name(), t.Name())
	require.NoError(t, err)

	repo := postgres.NewBaseStationRepository(db, clock.SystemClock{}, logger.NewNop())
	ctx := testutil.TestContext()
	empty := ""
	require.NoError(t, repo.Create(ctx, &models.BaseStation{
		EUI: adapterPostgresStation, TenantID: adapterPostgresTenant, Name: t.Name(),
		ConnectionType: models.ConnectionTypeBSSCI, ServiceCenterURL: &empty,
	}))

	station, err := NewRepositoryAdapter(repo, adapterPostgresTenant, "", logger.NewNop()).GetBaseStation(ctx, adapterPostgresStation)
	require.NoError(t, err)
	assert.Empty(t, station.ServiceCenterURL)

	stored, err := repo.GetByEUI(ctx, adapterPostgresTenant, adapterPostgresStation[:])
	require.NoError(t, err)
	assert.Nil(t, stored.ServiceCenterURL, "the read backfill stores an unknown URL as NULL")
}
