package bssci

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// Stations of the serving-station tests: two of the owner tenant and one of
// the tenant an endpoint roams into.
const (
	servingOwnerTenant   int64  = 1
	servingRoamTenant    int64  = 4
	servingStationLow    uint64 = 0x70B3D59CD0000001
	servingStationHigh   uint64 = 0x70B3D59CD0000002
	servingStationRoam   uint64 = 0x70B3D59CD0000003
	servingStationAbsent uint64 = 0x70B3D59CD0000009
	servingQueueID       uint64 = 7_300_000_000_000_101
)

// servingLocator decides the station serving the owner tenant's endpoint;
// zero means no station heard or attached it yet, and another tenant has no
// such endpoint.
type servingLocator struct {
	station uint64
	asked   []int64
}

func (l *servingLocator) ServingStation(_ context.Context, tenantID int64, _ uint64) (uint64, bool, error) {
	l.asked = append(l.asked, tenantID)
	if tenantID != servingOwnerTenant {
		return 0, false, storage.ErrNotFound
	}
	return l.station, l.station != 0, nil
}

// sessionRecordingDispatcher records the session every exact dispatch goes through.
type sessionRecordingDispatcher struct {
	noopDownlinkDispatcher
	stations []uint64
}

func (d *sessionRecordingDispatcher) DispatchQueue(_ context.Context, _ int64, _ uuid.UUID, session *Session, _, _ uint64) (bool, error) {
	d.stations = append(d.stations, session.BaseStationEUI)
	return true, nil
}

func newServingStationServer(t *testing.T, locator *servingLocator) (*Server, *sessionRecordingDispatcher) {
	t.Helper()
	server := NewTestServerWithMemoryStatusService(newRecordingLogger(), nil, nil, servingOwnerTenant)
	dispatcher := &sessionRecordingDispatcher{}
	server.downlinkDispatcher = dispatcher
	server.servingStations = locator
	for _, station := range []struct {
		eui    uint64
		tenant int64
		bidi   bool
	}{
		{servingStationLow, servingOwnerTenant, true},
		{servingStationHigh, servingOwnerTenant, true},
		{servingStationRoam, servingRoamTenant, true},
		{servingStationAbsent - 1, servingOwnerTenant, false},
	} {
		server.RegisterSession(&Session{
			ProtocolSessionState: ProtocolSessionState{
				ID:                mioty.FormatEUI64(station.eui),
				BaseStationEUI:    station.eui,
				ResolvedTenantID:  station.tenant,
				HandshakeComplete: true,
			},
			Bidirectional: station.bidi,
		})
	}
	return server, dispatcher
}

func queueServingDownlink(server *Server, tenantID int64) (uint64, error) {
	_, bsEUI, err := server.QueueDownlink(testutil.TestContext(),
		&mioty.DLDataQueue{EpEui: TestEpEui01, QueId: servingQueueID, UserData: [][]byte{{0x01}}},
		tenantID, uuid.New())
	return bsEUI, err
}

// TestQueueDownlink_GoesToTheStationThatLastHeardTheEndpoint pins radio
// §3.6.1: a downlink queued ahead of time is only useful at a station that
// hears the endpoint's next uplink, so it goes to the station that received
// the endpoint last, not to the tenant's lowest-EUI station.
func TestQueueDownlink_GoesToTheStationThatLastHeardTheEndpoint(t *testing.T) {
	locator := &servingLocator{station: servingStationHigh}
	server, dispatcher := newServingStationServer(t, locator)

	bsEUI, err := queueServingDownlink(server, servingOwnerTenant)

	require.NoError(t, err)
	assert.Equal(t, servingStationHigh, bsEUI)
	assert.Equal(t, []uint64{servingStationHigh}, dispatcher.stations)
	assert.Equal(t, []int64{servingOwnerTenant}, locator.asked, "the endpoint is looked up under the queue row's tenant")
}

// TestQueueDownlink_RoamingStationServesTheOwnersDownlink: the station that
// hears a roaming endpoint belongs to another tenant and still transmits the
// owner's downlink.
func TestQueueDownlink_RoamingStationServesTheOwnersDownlink(t *testing.T) {
	server, dispatcher := newServingStationServer(t, &servingLocator{station: servingStationRoam})

	bsEUI, err := queueServingDownlink(server, servingOwnerTenant)

	require.NoError(t, err)
	assert.Equal(t, servingStationRoam, bsEUI)
	assert.Equal(t, []uint64{servingStationRoam}, dispatcher.stations)
}

// TestQueueDownlink_DefersWhileTheServingStationCannotTransmit: an endpoint
// known to be heard by a station that is offline or unidirectional is out of
// reach of every other station, so the row stays pending for the endpoint's
// next downlink window instead of going to a station that cannot reach it.
func TestQueueDownlink_DefersWhileTheServingStationCannotTransmit(t *testing.T) {
	for name, station := range map[string]uint64{
		"station not connected":  servingStationAbsent,
		"station unidirectional": servingStationAbsent - 1,
	} {
		t.Run(name, func(t *testing.T) {
			server, dispatcher := newServingStationServer(t, &servingLocator{station: station})

			_, err := queueServingDownlink(server, servingOwnerTenant)

			require.ErrorIs(t, err, scheduler.ErrSchedulerNoResources)
			assert.Empty(t, dispatcher.stations, "no other station is chosen")
		})
	}
}

// TestQueueDownlink_UnlocatedEndpointWaitsForItsFirstUplink: an endpoint no
// station heard or attached yet has no station that can reach it, so its
// downlink stays pending until the window of its first dlOpen uplink and is
// never handed to an arbitrary station of its tenant.
func TestQueueDownlink_UnlocatedEndpointWaitsForItsFirstUplink(t *testing.T) {
	server, dispatcher := newServingStationServer(t, &servingLocator{})

	_, err := queueServingDownlink(server, servingOwnerTenant)

	require.ErrorIs(t, err, scheduler.ErrSchedulerNoResources)
	assert.Empty(t, dispatcher.stations, "no station of the tenant is chosen")
}

// TestQueueDownlink_MissingEndpointIsAMissingResource: a tenant without the
// endpoint is told so, not that its location is unknown.
func TestQueueDownlink_MissingEndpointIsAMissingResource(t *testing.T) {
	server, dispatcher := newServingStationServer(t, &servingLocator{station: servingStationHigh})

	_, err := queueServingDownlink(server, servingRoamTenant)

	require.ErrorIs(t, err, scheduler.ErrSchedulerResourceMissing)
	assert.Empty(t, dispatcher.stations)
}
