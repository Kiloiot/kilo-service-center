package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// TestReleaseStationReservations pins the reclaim of a base station's
// orphaned reservations: every row the station holds reserved returns to
// pending across the tenants it carried, except the kept queue ids, and no
// other station's reservation or any queued row is touched.
func TestReleaseStationReservations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)

	const (
		tenantA = int64(331)
		tenantB = int64(332)
	)
	station := []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x03, 0x31}
	otherStation := []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x03, 0x32}
	createTestTenant(t, sqlxDB, tenantA, "ReleaseTenantA")
	createTestTenant(t, sqlxDB, tenantB, "ReleaseTenantB")
	orgs := map[int64]uuid.UUID{tenantA: uuid.New(), tenantB: uuid.New()}
	for tenantID, orgID := range orgs {
		_, err := sqlxDB.Exec(`INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, $2, $3)`, orgID, tenantID, "release-org-"+orgID.String())
		require.NoError(t, err)
	}
	seed := func(tenantID, queID int64, status string, bsEUI []byte) {
		_, err := sqlxDB.Exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, bs_eui, earliest_at)
			VALUES (decode('70b3d59cd0000331', 'hex'), $1, $2, '\x01', $3, $4, $5, NULL)`,
			tenantID, orgs[tenantID], status, queID, bsEUI)
		require.NoError(t, err)
	}
	seed(tenantA, 830001, "reserved", station)      // orphaned
	seed(tenantB, 830002, "reserved", station)      // orphaned, roaming tenant
	seed(tenantA, 830003, "reserved", station)      // kept: reissued on resume
	seed(tenantA, 830004, "reserved", otherStation) // another station's reservation
	seed(tenantA, 830005, "queued", station)        // acknowledged by the station

	logger.Initialize("error", "json")
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: logger.Get()}
	released, err := NewRepositories(db).Downlinks.ReleaseStationReservations(t.Context(),
		0x70B3D59CD0000331, []int64{830003})
	require.NoError(t, err)
	assert.ElementsMatch(t, []storage.PendingDownlink{
		{QueID: 830001, TenantID: tenantA, OrganizationID: orgs[tenantA], EpEUI: 0x70B3D59CD0000331},
		{QueID: 830002, TenantID: tenantB, OrganizationID: orgs[tenantB], EpEUI: 0x70B3D59CD0000331},
	}, released, "every released row names its tenant, organization and endpoint")

	status := func(queID int64) string {
		var s string
		require.NoError(t, sqlxDB.Get(&s, `SELECT status FROM downlink_queue WHERE que_id = $1`, queID))
		return s
	}
	assert.Equal(t, "pending", status(830001))
	assert.Equal(t, "pending", status(830002))
	assert.Equal(t, "reserved", status(830003))
	assert.Equal(t, "reserved", status(830004))
	assert.Equal(t, "queued", status(830005))
	holder := func(queID int64) []byte {
		var bsEUI []byte
		require.NoError(t, sqlxDB.Get(&bsEUI, `SELECT bs_eui FROM downlink_queue WHERE que_id = $1`, queID))
		return bsEUI
	}
	assert.Nil(t, holder(830001), "a released reservation has no holder")
	assert.Nil(t, holder(830002), "a released reservation has no holder")
	assert.Equal(t, station, holder(830003), "a kept reservation keeps its holder")

	released, err = NewRepositories(db).Downlinks.ReleaseStationReservations(t.Context(), 0x70B3D59CD0000331, nil)
	require.NoError(t, err)
	assert.Equal(t, []uint64{830003}, releasedQueueIDs(released), "without kept ids every reservation of the station is released")
	assert.Equal(t, "pending", status(830003))
}

// TestReleaseEndpointAtStation pins the release of the downlinks a base
// station discards when it takes an attach propagate for an endpoint: the
// tenant's rows for the endpoint queued at the station and sent to it by the
// cut, or with no recorded send time, return to pending with no holder. A row
// sent after the cut, one still reserved, another endpoint's, another
// station's, another tenant's and a pending row keep their state.
func TestReleaseEndpointAtStation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)

	const (
		tenant      = int64(341)
		otherTenant = int64(342)
	)
	station := []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x03, 0x41}
	otherStation := []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x03, 0x42}
	endpoint := []byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x03, 0x41}
	otherEndpoint := []byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x03, 0x42}
	createTestTenant(t, sqlxDB, tenant, "ReleaseEndpointTenant")
	createTestTenant(t, sqlxDB, otherTenant, "ReleaseEndpointOtherTenant")
	orgs := map[int64]uuid.UUID{tenant: uuid.New(), otherTenant: uuid.New()}
	for tenantID, orgID := range orgs {
		_, err := sqlxDB.Exec(`INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, $2, $3)`, orgID, tenantID, "release-endpoint-org-"+orgID.String())
		require.NoError(t, err)
	}
	cut := time.Date(2027, time.March, 1, 9, 0, 0, 0, time.UTC)
	before, after := cut.Add(-time.Minute).UnixNano(), cut.Add(time.Minute).UnixNano()
	seed := func(tenantID, queID int64, epEUI []byte, status string, bsEUI []byte, sentAt *int64) {
		_, err := sqlxDB.Exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, bs_eui, attempts, tx_time, earliest_at)
			VALUES ($1, $2, $3, '\x01', $4, $5, $6, 0, $7, NULL)`,
			epEUI, tenantID, orgs[tenantID], status, queID, bsEUI, sentAt)
		require.NoError(t, err)
	}
	seed(tenant, 840001, endpoint, "queued", station, &before)      // discarded by the station
	seed(tenant, 840002, endpoint, "reserved", station, nil)        // dispatch in flight
	seed(tenant, 840003, endpoint, "queued", station, &after)       // queued behind the attach propagate
	seed(tenant, 840004, otherEndpoint, "queued", station, &before) // another endpoint's
	seed(tenant, 840005, endpoint, "queued", otherStation, &before) // held by another station
	seed(otherTenant, 840006, endpoint, "queued", station, &before) // another tenant's endpoint
	seed(tenant, 840007, endpoint, "pending", nil, nil)             // held by no station
	seed(tenant, 840008, endpoint, "queued", station, nil)          // queued before send times were recorded

	logger.Initialize("error", "json")
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: logger.Get()}
	released, err := NewRepositories(db).Downlinks.ReleaseEndpointAtStation(t.Context(),
		tenant, 0x70B3D56770110341, 0x70B3D59CD0000341, cut)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint64{840001, 840008}, releasedQueueIDs(released))

	type row struct {
		Status   string `db:"status"`
		Holder   []byte `db:"bs_eui"`
		Attempts int    `db:"attempts"`
	}
	read := func(queID int64) row {
		var r row
		require.NoError(t, sqlxDB.Get(&r, `SELECT status, bs_eui, attempts FROM downlink_queue WHERE que_id = $1`, queID))
		return r
	}
	const releasedOnce = 1
	for _, queID := range []int64{840001, 840008} {
		assert.Equal(t, row{Status: "pending", Attempts: releasedOnce}, read(queID), "queue id %d is pending with no holder", queID)
	}
	for queID, status := range map[int64]string{840002: "reserved", 840003: "queued", 840004: "queued", 840005: "queued", 840006: "queued", 840007: "pending"} {
		r := read(queID)
		assert.Equal(t, status, r.Status, "queue id %d", queID)
		assert.Zero(t, r.Attempts, "queue id %d", queID)
	}
}

// releasedQueueIDs lists the queue ids of released downlinks.
func releasedQueueIDs(released []storage.PendingDownlink) []uint64 {
	ids := make([]uint64, len(released))
	for i, downlink := range released {
		ids[i] = downlink.QueID
	}
	return ids
}
