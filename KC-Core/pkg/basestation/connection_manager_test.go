package basestation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	testStationName  = "tims base"
	testConnectionID = "session-3"
)

var testStationEUI = [8]byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x09, 0xE6}

// namedStationStore holds one registered, connected station.
type namedStationStore struct{ Store }

func (namedStationStore) GetBaseStation(context.Context, [8]byte) (*BaseStation, error) {
	return &BaseStation{EUI: testStationEUI, Name: testStationName}, nil
}

func (namedStationStore) DisconnectIfCurrent(context.Context, [8]byte, string, time.Time) (bool, error) {
	return true, nil
}

func (namedStationStore) UpdateConnectionStatus(context.Context, [8]byte, *ConnectionStatus) error {
	return nil
}

// recordedStationEvent is one event the manager recorded.
type recordedStationEvent struct {
	eventType  string
	occurredAt time.Time
	data       map[string]interface{}
}

type stationEvents struct{ events []recordedStationEvent }

func (s *stationEvents) RecordEvent(_ context.Context, _ [8]byte, eventType string, occurredAt time.Time, data map[string]interface{}) error {
	s.events = append(s.events, recordedStationEvent{eventType, occurredAt, data})
	return nil
}

// A station going offline is named in its event, as when it connected: the
// manager records through the recorder that names the station.
func TestDisconnectBaseStationIfCurrent_NamesTheStation(t *testing.T) {
	events := &stationEvents{}
	manager := NewConnectionManager(namedStationStore{}, NewNamingEventRecorder(namedStationStore{}, events, logger.NewNop()))

	require.NoError(t, manager.DisconnectBaseStationIfCurrent(testutil.TestContext(), testStationEUI, testConnectionID))

	require.Len(t, events.events, 1)
	assert.Equal(t, models.EventTypeBaseStationOffline, events.events[0].eventType)
	assert.Equal(t, testStationName, events.events[0].data[models.EventDetailKeyBaseStationName])
}

// Connectivity event details are camelCase, as every detail the browser reads.
func TestConnectivityEvents_WriteCamelCaseDetails(t *testing.T) {
	events := &stationEvents{}
	manager := NewConnectionManager(namedStationStore{}, events)
	ctx := testutil.TestContext()

	require.NoError(t, manager.UpdateConnectionStatus(ctx, testStationEUI,
		&ConnectionStatus{IsOnline: true, ConnectionType: ConnectionTypeBSSCI, SessionID: testConnectionID}))
	require.NoError(t, manager.DisconnectBaseStationIfCurrent(ctx, testStationEUI, testConnectionID))

	require.Len(t, events.events, 2)
	assert.Equal(t, map[string]interface{}{
		models.EventDetailKeyIsOnline:       true,
		models.EventDetailKeyConnectionType: ConnectionTypeBSSCI,
		models.EventDetailKeySessionID:      testConnectionID,
	}, events.events[0].data)
	assert.Equal(t, map[string]interface{}{
		models.EventDetailKeyIsOnline:  false,
		models.EventDetailKeySessionID: testConnectionID,
	}, events.events[1].data)
	for _, event := range events.events {
		for key := range event.data {
			assert.Regexp(t, `^[a-z][a-zA-Z0-9]*$`, key)
		}
	}
}

// disconnectingStationStore keeps the moment a disconnect was stored.
type disconnectingStationStore struct {
	namedStationStore
	lastSeen time.Time
}

func (d *disconnectingStationStore) DisconnectIfCurrent(_ context.Context, _ [8]byte, _ string, lastSeen time.Time) (bool, error) {
	d.lastSeen = lastSeen
	return true, nil
}

// A station's online and offline events occurred when its connection changed,
// the moments the store keeps as last seen, not when naming them finished.
func TestConnectivityEvents_OccurWhenTheConnectionChanged(t *testing.T) {
	events := &stationEvents{}
	store := &disconnectingStationStore{}
	manager := NewConnectionManager(store, events)
	ctx := testutil.TestContext()

	require.NoError(t, manager.UpdateConnectionStatus(ctx, testStationEUI,
		&ConnectionStatus{IsOnline: true, LastSeen: recorderOccurredAt, ConnectionType: ConnectionTypeBSSCI, SessionID: testConnectionID}))
	require.NoError(t, manager.DisconnectBaseStationIfCurrent(ctx, testStationEUI, testConnectionID))

	require.Len(t, events.events, 2)
	assert.Equal(t, recorderOccurredAt, events.events[0].occurredAt)
	require.False(t, store.lastSeen.IsZero())
	assert.Equal(t, store.lastSeen, events.events[1].occurredAt)
}
