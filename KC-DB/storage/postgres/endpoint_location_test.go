package postgres

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// Fixture identities for the serving base station tests: the owner tenant,
// a tenant whose station the endpoint roams through, and three stations.
const (
	servingOwnerTenant   int64  = 331
	servingForeignTenant int64  = 332
	servingEndpointEUI   uint64 = 0x70B3D59CD0000331
	servingStationNear   uint64 = 0x70B3D59CD0000A01
	servingStationFar    uint64 = 0x70B3D59CD0000A02
	servingStationRoam   uint64 = 0x70B3D59CD0000A03
)

type servingFixture struct {
	t         *testing.T
	db        *sqlx.DB
	downlinks *MIOTYDownlinkRepository
}

func newServingFixture(t *testing.T) *servingFixture {
	t.Helper()
	downlinks, db, _ := applicationQueueIDFixture(t)
	createTestTenant(t, db, servingOwnerTenant, "ServingOwner")
	createTestTenant(t, db, servingForeignTenant, "ServingForeign")
	return &servingFixture{t: t, db: db, downlinks: downlinks}
}

func (f *servingFixture) endpoint(tenantID int64, eui uint64, attached bool) {
	f.t.Helper()
	f.introducedEndpoint(tenantID, eui, attached, nil)
}

// introducedEndpoint registers the endpoint; an attached one gets an active
// session, recorded as attached or propagated through introducedThrough when set.
func (f *servingFixture) introducedEndpoint(tenantID int64, eui uint64, attached bool, introducedThrough *uint64) {
	f.t.Helper()
	id := insertEndpoint(f.t, f.db, EndpointInsertParams{EpEUI: eui, Name: "serving-" + uuid.NewString(), TenantID: tenantID})
	if !attached {
		return
	}
	var primary *int64
	if introducedThrough != nil {
		var bsID int64
		require.NoError(f.t, f.db.QueryRow(`
			INSERT INTO basestations (tenant_id, bs_eui, name, connection_type, service_center_url)
			VALUES ($1, $2, $3, 'bssci', 'bssci://serving') RETURNING id`,
			tenantID, mioty.EUI64Bytes(*introducedThrough), "serving-bs-"+uuid.NewString()).Scan(&bsID))
		primary = &bsID
	}
	_, err := f.db.Exec(`INSERT INTO endpoint_sessions (endpoint_id, tenant_id, session_id, attach_cnt, status, primary_basestation_id)
		VALUES ($1, $2, $3, 1, 'active', $4)`, id, tenantID, uuid.NewString(), primary)
	require.NoError(f.t, err)
}

// uplink stores one ulData reception set of the endpoint under ownerTenant.
func (f *servingFixture) uplink(ownerTenant int64, eui uint64, firstReceiver uint64, rxTime time.Time, receptions ...mioty.BaseStationReception) {
	f.t.Helper()
	var baseStations *string
	if len(receptions) > 0 {
		encoded, err := json.Marshal(receptions)
		require.NoError(f.t, err)
		text := string(encoded)
		baseStations = &text
	}
	_, err := f.db.Exec(`
		INSERT INTO messages (id, tenant_id, owner_tenant_id, command_type, op_id, ep_eui, bs_eui,
			rx_time, packet_cnt, snr, rssi, dl_open, response_exp, dl_ack, received_at, base_stations)
		VALUES ($1, $2, $2, $3, 1, $4, $5, $6, 1, 1.0, -90.0, false, false, false, NOW(), $7::jsonb)`,
		uuid.NewString(), ownerTenant, mioty.CmdULData, mioty.EUI64Bytes(eui), mioty.EUI64Bytes(firstReceiver), rxTime.UnixNano(), baseStations)
	require.NoError(f.t, err)
}

func (f *servingFixture) location(tenantID int64, eui uint64) (storage.EndpointLocation, error) {
	return f.downlinks.GetEndpointLocation(f.t.Context(), tenantID, eui)
}

// TestGetEndpointLocation_ReceptionsOfTheLatestUplink: the location names
// every station that heard the endpoint's latest uplink, whichever tenant owns
// it, with the reception it reported.
func TestGetEndpointLocation_ReceptionsOfTheLatestUplink(t *testing.T) {
	f := newServingFixture(t)
	f.endpoint(servingOwnerTenant, servingEndpointEUI, true)
	earlier := time.Now().Add(-time.Hour)
	f.uplink(servingOwnerTenant, servingEndpointEUI, servingStationNear, earlier)
	latest := []mioty.BaseStationReception{{BsEui: servingStationFar, Snr: 2}, {BsEui: servingStationRoam, Snr: 9}}
	f.uplink(servingOwnerTenant, servingEndpointEUI, servingStationFar, earlier.Add(time.Minute), latest...)

	location, err := f.location(servingOwnerTenant, servingEndpointEUI)

	require.NoError(t, err)
	assert.Equal(t, latest, location.LatestReceptions)
	assert.Zero(t, location.AttachedThrough)
}

// TestGetEndpointLocation_FirstReceiverWithoutReceptionList covers an uplink
// stored without the merged reception list.
func TestGetEndpointLocation_FirstReceiverWithoutReceptionList(t *testing.T) {
	f := newServingFixture(t)
	f.endpoint(servingOwnerTenant, servingEndpointEUI, true)
	f.uplink(servingOwnerTenant, servingEndpointEUI, servingStationFar, time.Now())

	location, err := f.location(servingOwnerTenant, servingEndpointEUI)

	require.NoError(t, err)
	assert.Equal(t, []mioty.BaseStationReception{{BsEui: servingStationFar}}, location.LatestReceptions)
}

// TestGetEndpointLocation_AttachedThroughBesideTheUplink: the station the
// endpoint's active session was attached or propagated through is read
// whether or not the endpoint has been heard since.
func TestGetEndpointLocation_AttachedThroughBesideTheUplink(t *testing.T) {
	f := newServingFixture(t)
	introduced := servingStationNear
	f.introducedEndpoint(servingOwnerTenant, servingEndpointEUI, true, &introduced)

	location, err := f.location(servingOwnerTenant, servingEndpointEUI)
	require.NoError(t, err)
	assert.Equal(t, storage.EndpointLocation{AttachedThrough: servingStationNear}, location, "not heard yet")

	f.uplink(servingOwnerTenant, servingEndpointEUI, servingStationFar, time.Now())
	location, err = f.location(servingOwnerTenant, servingEndpointEUI)
	require.NoError(t, err)
	assert.Equal(t, storage.EndpointLocation{
		LatestReceptions: []mioty.BaseStationReception{{BsEui: servingStationFar}},
		AttachedThrough:  servingStationNear,
	}, location, "heard")
}

// TestGetEndpointLocation_HeardBeforeItsSessionExists: a pre-provisioned
// endpoint can be heard before the connect-time propagation gives it a session.
func TestGetEndpointLocation_HeardBeforeItsSessionExists(t *testing.T) {
	f := newServingFixture(t)
	f.endpoint(servingOwnerTenant, servingEndpointEUI, false)
	f.uplink(servingOwnerTenant, servingEndpointEUI, servingStationNear, time.Now())

	location, err := f.location(servingOwnerTenant, servingEndpointEUI)

	require.NoError(t, err)
	assert.Equal(t, []mioty.BaseStationReception{{BsEui: servingStationNear}}, location.LatestReceptions)
}

// TestGetEndpointLocation_KnownEndpointWithoutALocation: an endpoint neither
// heard nor attached through a station exists with an empty location, which
// is not the missing endpoint of TestGetEndpointLocation_ForeignTenantSeesNothing.
func TestGetEndpointLocation_KnownEndpointWithoutALocation(t *testing.T) {
	f := newServingFixture(t)
	f.endpoint(servingOwnerTenant, servingEndpointEUI, false)
	location, err := f.location(servingOwnerTenant, servingEndpointEUI)
	require.NoError(t, err, "a registered endpoint without a session or an uplink")
	assert.Equal(t, storage.EndpointLocation{}, location)

	unheardEUI := servingEndpointEUI + 1
	f.endpoint(servingOwnerTenant, unheardEUI, true)
	location, err = f.location(servingOwnerTenant, unheardEUI)
	require.NoError(t, err, "an attached endpoint neither heard nor introduced")
	assert.Equal(t, storage.EndpointLocation{}, location)
}

// TestGetEndpointLocation_ForeignTenantSeesNothing: another tenant asking for
// the endpoint learns nothing about where it is heard.
func TestGetEndpointLocation_ForeignTenantSeesNothing(t *testing.T) {
	f := newServingFixture(t)
	f.endpoint(servingOwnerTenant, servingEndpointEUI, true)
	f.uplink(servingOwnerTenant, servingEndpointEUI, servingStationNear, time.Now())

	_, err := f.location(servingForeignTenant, servingEndpointEUI)

	assert.ErrorIs(t, err, storage.ErrNotFound)
}
