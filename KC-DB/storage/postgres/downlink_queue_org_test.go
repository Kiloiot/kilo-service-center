package postgres

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

const queueOrgTestTenant = int64(351)

// TestListTenantQueue_ScopesToTheOrganization pins the organization scope of
// the queue listing: a tenant's other organization's downlinks are neither
// listed nor counted.
func TestListTenantQueue_ScopesToTheOrganization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	createTestTenant(t, sqlxDB, queueOrgTestTenant, "QueueOrgTenant")
	orgA, orgB := uuid.New(), uuid.New()
	for _, org := range []uuid.UUID{orgA, orgB} {
		_, err := sqlxDB.Exec(`INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, $2, $3)`, org, queueOrgTestTenant, org.String())
		require.NoError(t, err)
	}
	for queID, org := range map[int64]uuid.UUID{850001: orgA, 850002: orgB} {
		_, err := sqlxDB.Exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, earliest_at)
			VALUES (decode('70b3d59cd0000351', 'hex'), $1, $2, '\x01', 'pending', $3, NULL)`, queueOrgTestTenant, org, queID)
		require.NoError(t, err)
	}
	logger.Initialize("error", "json")
	reader := NewDownlinkQueueReader(sqlxDB, logger.Get())
	filter := storage.DownlinkQueueFilter{OrganizationID: &orgA}

	listed, err := reader.ListTenantQueue(t.Context(), queueOrgTestTenant, filter, 10, 0)
	require.NoError(t, err)
	count, err := reader.CountTenantQueue(t.Context(), queueOrgTestTenant, filter)
	require.NoError(t, err)

	require.Len(t, listed, 1)
	assert.Equal(t, int64(850001), listed[0].QueID)
	assert.Equal(t, int64(1), count)
}

// TestDownlinkListings_ReportTheStationOfTheRow: the queue and results
// listings carry the base station the row names, and none for a downlink no
// station holds yet.
func TestDownlinkListings_ReportTheStationOfTheRow(t *testing.T) {
	repos, db, orgs := downlinkRepositoriesFixture(t)
	const station = uint64(0x70B3D59CD00009E6)
	seed := func(queID int64, status mioty.DLQueueStatus, bsEUI []byte) {
		_, err := db.Exec(`INSERT INTO downlink_queue (que_id, ep_eui, tenant_id, organization_id, payload, status, bs_eui, earliest_at)
			VALUES ($1, $2, 321, $3, '\x01', $4, $5, NULL)`, queID, mioty.EUI64Bytes(ackEndpointEUI), orgs[321], status, bsEUI)
		require.NoError(t, err)
	}
	seed(850101, mioty.DLQueueStatusQueued, mioty.EUI64Bytes(station))
	seed(850102, mioty.DLQueueStatusPending, nil)
	seed(850103, mioty.DLQueueStatusTransmitted, mioty.EUI64Bytes(station))

	queued, err := repos.DownlinkQueueReader.ListTenantQueue(t.Context(), 321, storage.DownlinkQueueFilter{}, 10, 0)
	require.NoError(t, err)
	results, _, err := repos.Downlinks.GetDownlinkResults(t.Context(), 321, nil, storage.DownlinkResultFilter{}, 10, 0)
	require.NoError(t, err)

	stations := map[int64]uint64{}
	for _, dl := range append(queued, results...) {
		stations[dl.QueID] = dl.BsEui
	}
	assert.Equal(t, map[int64]uint64{850101: station, 850102: 0, 850103: station}, stations)
}

// TestListTenantQueue_FiltersByTheHoldingStation: the queue narrows to the
// downlinks a base station holds, and filtering by another tenant's station
// never reaches that tenant's downlinks.
func TestListTenantQueue_FiltersByTheHoldingStation(t *testing.T) {
	repos, db, orgs := downlinkRepositoriesFixture(t)
	const stationA = uint64(0x70B3D59CD00009E6)
	const stationB = uint64(0x70B3D59CD00009E2)
	seed := func(tenantID, queID int64, bsEUI []byte) {
		_, err := db.Exec(`INSERT INTO downlink_queue (que_id, ep_eui, tenant_id, organization_id, payload, status, bs_eui, earliest_at)
			VALUES ($1, $2, $3, $4, '\x01', 'queued', $5, NULL)`, queID, mioty.EUI64Bytes(ackEndpointEUI), tenantID, orgs[tenantID], bsEUI)
		require.NoError(t, err)
	}
	seed(321, 850201, mioty.EUI64Bytes(stationA))
	seed(321, 850202, nil)
	seed(322, 850203, mioty.EUI64Bytes(stationB))

	byStation := func(tenantID int64, station uint64) ([]int64, int64) {
		t.Helper()
		var eui [8]byte
		copy(eui[:], mioty.EUI64Bytes(station))
		filter := storage.DownlinkQueueFilter{BsEUI: &eui}
		listed, err := repos.DownlinkQueueReader.ListTenantQueue(t.Context(), tenantID, filter, 10, 0)
		require.NoError(t, err)
		count, err := repos.DownlinkQueueReader.CountTenantQueue(t.Context(), tenantID, filter)
		require.NoError(t, err)
		ids := make([]int64, 0, len(listed))
		for _, dl := range listed {
			ids = append(ids, dl.QueID)
		}
		return ids, count
	}

	ids, count := byStation(321, stationA)
	assert.Equal(t, []int64{850201}, ids, "only the downlink the station holds")
	assert.Equal(t, int64(1), count)

	ids, count = byStation(321, stationB)
	assert.Empty(t, ids, "another tenant's station reveals none of its downlinks")
	assert.Zero(t, count)
}
