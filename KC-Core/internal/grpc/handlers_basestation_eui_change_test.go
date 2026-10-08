package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	euiChangeOldEui = "AA00000000000001"
	euiChangeNewEui = "AA00000000000002"
)

// renamedBaseStations moves a station of testOwnerTenant to its new EUI.
type renamedBaseStations struct {
	mockBasestationSvcIsolation
}

func (renamedBaseStations) UpdateEUI(_ context.Context, tenantID int64, _, newEui []byte) (*models.BaseStation, error) {
	bs := &models.BaseStation{ID: testOwnedBaseStationID, TenantID: tenantID}
	copy(bs.EUI[:], newEui)
	return bs, nil
}

// liveSessions closes the sessions of the stations that hold one.
type liveSessions struct {
	connected map[uint64]bool
	closed    []uint64
}

func (l *liveSessions) CloseSessionByEUI(_ context.Context, eui uint64) bool {
	if !l.connected[eui] {
		return false
	}
	delete(l.connected, eui)
	l.closed = append(l.closed, eui)
	return true
}

type recordedConnectivityEvent struct {
	eui        [8]byte
	eventType  string
	occurredAt time.Time
	data       map[string]interface{}
}

type connectivityEvents struct{ events []recordedConnectivityEvent }

func (c *connectivityEvents) RecordEvent(_ context.Context, eui [8]byte, eventType string, occurredAt time.Time, data map[string]interface{}) error {
	c.events = append(c.events, recordedConnectivityEvent{eui, eventType, occurredAt, data})
	return nil
}

func changeStationEUI(t *testing.T, sessions *liveSessions, events *connectivityEvents) {
	t.Helper()
	svc := testCoreService(coreFields{basestationSvc: &renamedBaseStations{}, bssciSessionCloser: sessions, bsEventRecorder: events, log: &mockLogger{}})
	_, err := svc.UpdateBaseStationEui(contextForTenant(testOwnerTenant), &pb.UpdateBaseStationEuiRequest{BsEui: euiChangeOldEui, NewBsEui: euiChangeNewEui})
	require.NoError(t, err)
}

// A station that never connected has no connection to lose: its new EUI
// brings no "went offline" warning.
func TestUpdateBaseStationEui_NeverConnectedStationRecordsNoOffline(t *testing.T) {
	sessions := &liveSessions{connected: map[uint64]bool{}}
	events := &connectivityEvents{}

	changeStationEUI(t, sessions, events)

	assert.Empty(t, sessions.closed)
	assert.Empty(t, events.events, "no session was closed, so the station did not go offline")
}

// A connected station loses its session with its old identity and goes
// offline once, filed under the EUI it now has.
func TestUpdateBaseStationEui_ConnectedStationGoesOfflineOnce(t *testing.T) {
	oldEui := models.EUIFromString(euiChangeOldEui)
	newEui := models.EUIFromString(euiChangeNewEui)
	sessions := &liveSessions{connected: map[uint64]bool{oldEui.ToUint64(): true}}
	events := &connectivityEvents{}
	before := time.Now()

	changeStationEUI(t, sessions, events)

	assert.Equal(t, []uint64{oldEui.ToUint64()}, sessions.closed)
	require.Len(t, events.events, 1)
	offline := events.events[0]
	assert.Equal(t, models.EventTypeBaseStationOffline, offline.eventType)
	assert.Equal(t, [8]byte(newEui), offline.eui)
	assert.False(t, offline.occurredAt.Before(before), "the station went offline when its session closed")
	assert.Equal(t, map[string]interface{}{
		models.EventDetailKeyIsOnline: false,
		models.EventDetailKeyReason:   reasonEUIChanged,
	}, offline.data)
}
