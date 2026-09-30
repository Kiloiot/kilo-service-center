package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	analyticsTenant        int64  = 610
	analyticsForeignTenant int64  = 611
	analyticsEndpoint      uint64 = 0x70B3D56770111505
	analyticsNearStation   uint64 = 0x70B3D59CD00009E6
	analyticsFarStation    uint64 = 0x70B3D59CD00009E2
	analyticsForeignBS     uint64 = 0x70B3D59CD00009BB
)

type analyticsFixture struct {
	t    *testing.T
	db   *sqlx.DB
	repo *MessageAnalyticsRepository
}

func newAnalyticsFixture(t *testing.T) *analyticsFixture {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	createTestTenant(t, db, analyticsTenant, "Analytics Tenant")
	createTestTenant(t, db, analyticsForeignTenant, "Analytics Foreign Tenant")
	return &analyticsFixture{t: t, db: db, repo: NewMessageAnalyticsRepository(db)}
}

// uplink stores one reception of the endpoint by the given station.
func (f *analyticsFixture) uplink(tenantID int64, bsEui uint64, packetCnt int, snr, rssi float64) {
	f.t.Helper()
	_, err := f.db.Exec(`
		INSERT INTO messages (id, tenant_id, owner_tenant_id, command_type, op_id, ep_eui, bs_eui,
			rx_time, packet_cnt, snr, rssi, dl_open, response_exp, dl_ack, received_at)
		VALUES ($1, $2, $2, $3, 1, $4, $5, $6, $7, $8, $9, false, false, false, NOW())`,
		uuid.NewString(), tenantID, mioty.CmdULData, mioty.EUI64Bytes(analyticsEndpoint), mioty.EUI64Bytes(bsEui),
		time.Now().UnixNano(), packetCnt, snr, rssi)
	require.NoError(f.t, err)
}

func analyticsWindow() (time.Time, time.Time) {
	now := time.Now()
	return now.Add(-time.Hour), now.Add(time.Hour)
}

func TestGetSignalQualityByBaseStation_GroupsTheTenantsReceptionsPerStation(t *testing.T) {
	f := newAnalyticsFixture(t)
	f.uplink(analyticsTenant, analyticsNearStation, 1, 10, -80)
	f.uplink(analyticsTenant, analyticsNearStation, 2, 6, -90)
	f.uplink(analyticsTenant, analyticsFarStation, 3, 2, -110)
	f.uplink(analyticsForeignTenant, analyticsForeignBS, 4, 20, -50)
	start, end := analyticsWindow()

	stations, err := f.repo.GetSignalQualityByBaseStation(testutil.TestContext(), analyticsTenant, start, end)

	require.NoError(t, err)
	require.Len(t, stations, 2)
	assert.Equal(t, analyticsNearStation, stations[0].BsEui)
	assert.Equal(t, int64(2), stations[0].MessageCount)
	assert.InDelta(t, -85, stations[0].AvgRSSI, 0.0001)
	assert.InDelta(t, 8, stations[0].AvgSNR, 0.0001)
	assert.Equal(t, analyticsFarStation, stations[1].BsEui)
	assert.Equal(t, int64(1), stations[1].MessageCount)
}

func TestGetSignalQualityByBaseStation_ForeignTenantSeesNothing(t *testing.T) {
	f := newAnalyticsFixture(t)
	f.uplink(analyticsTenant, analyticsNearStation, 1, 10, -80)
	start, end := analyticsWindow()

	stations, err := f.repo.GetSignalQualityByBaseStation(testutil.TestContext(), analyticsForeignTenant, start, end)

	require.NoError(t, err)
	assert.Empty(t, stations)
}

func TestGetMessageCountsByBaseStation_DecodesStoredEUIs(t *testing.T) {
	f := newAnalyticsFixture(t)
	f.uplink(analyticsTenant, analyticsNearStation, 1, 10, -80)
	f.uplink(analyticsTenant, analyticsNearStation, 2, 10, -80)
	f.uplink(analyticsTenant, analyticsFarStation, 3, 10, -80)
	f.uplink(analyticsForeignTenant, analyticsForeignBS, 4, 10, -80)
	start, end := analyticsWindow()

	counts, err := f.repo.GetMessageCountsByBaseStation(testutil.TestContext(), analyticsTenant, start, end)

	require.NoError(t, err)
	assert.Equal(t, map[string]int64{
		mioty.FormatEUI64(analyticsNearStation): 2,
		mioty.FormatEUI64(analyticsFarStation):  1,
	}, counts)
}

func TestGetMessageCountsByEndpoint_DecodesStoredEUIs(t *testing.T) {
	f := newAnalyticsFixture(t)
	f.uplink(analyticsTenant, analyticsNearStation, 1, 10, -80)
	f.uplink(analyticsTenant, analyticsFarStation, 2, 10, -80)
	f.uplink(analyticsForeignTenant, analyticsForeignBS, 3, 10, -80)
	start, end := analyticsWindow()

	counts, err := f.repo.GetMessageCountsByEndpoint(testutil.TestContext(), analyticsTenant, start, end)

	require.NoError(t, err)
	assert.Equal(t, map[string]int64{mioty.FormatEUI64(analyticsEndpoint): 2}, counts)
}

const analyticsTopEndpointsLimit = 10

// GetTopEndpointsByActivity is retained public module surface: it ranks the
// tenant's endpoints by message count and decodes their stored EUIs.
func TestGetTopEndpointsByActivity_RanksTheTenantsEndpoints(t *testing.T) {
	f := newAnalyticsFixture(t)
	f.uplink(analyticsTenant, analyticsNearStation, 1, 10, -80)
	f.uplink(analyticsTenant, analyticsFarStation, 2, 10, -80)
	f.uplink(analyticsForeignTenant, analyticsForeignBS, 3, 10, -80)
	start, end := analyticsWindow()

	endpoints, err := f.repo.GetTopEndpointsByActivity(testutil.TestContext(), analyticsTenant, start, end, analyticsTopEndpointsLimit)

	require.NoError(t, err)
	require.Len(t, endpoints, 1)
	assert.Equal(t, analyticsEndpoint, endpoints[0].EUI)
	assert.Equal(t, 2, endpoints[0].MessageCount)
	assert.False(t, endpoints[0].LastSeen.IsZero())
}
