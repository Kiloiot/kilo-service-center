package basestation

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const removedStationName = "removed station"

var errStationUnreadable = errors.New("station unreadable")

// unreadableStationStore fails every station lookup.
type unreadableStationStore struct{ Store }

func (unreadableStationStore) GetBaseStation(context.Context, [8]byte) (*BaseStation, error) {
	return nil, errStationUnreadable
}

// countingStationStore counts the station lookups.
type countingStationStore struct {
	namedStationStore
	lookups int
}

func (c *countingStationStore) GetBaseStation(ctx context.Context, eui [8]byte) (*BaseStation, error) {
	c.lookups++
	return c.namedStationStore.GetBaseStation(ctx, eui)
}

// A ping the station answered reads with its registered name, as its
// connectivity events do; the caller's data and occurrence time are left as
// given, however long the lookup takes.
func TestNamingEventRecorder_NamesThePingsStation(t *testing.T) {
	events := &stationEvents{}
	recorder := NewNamingEventRecorder(namedStationStore{}, events, logger.NewNop())
	answered := map[string]interface{}{models.EventDetailKeyOpID: int64(-2691)}

	require.NoError(t, recorder.RecordEvent(testutil.TestContext(), testStationEUI, models.EventTypeBaseStationPingAnswered, recorderOccurredAt, answered))

	require.Len(t, events.events, 1)
	assert.Equal(t, testStationName, events.events[0].data[models.EventDetailKeyBaseStationName])
	assert.EqualValues(t, -2691, events.events[0].data[models.EventDetailKeyOpID])
	assert.Equal(t, recorderOccurredAt, events.events[0].occurredAt)
	assert.NotContains(t, answered, models.EventDetailKeyBaseStationName)
}

// An event that already names its station keeps that name without a lookup,
// as for a station just removed.
func TestNamingEventRecorder_KeepsTheNameTheEventCarries(t *testing.T) {
	events := &stationEvents{}
	stations := &countingStationStore{}
	recorder := NewNamingEventRecorder(stations, events, logger.NewNop())

	require.NoError(t, recorder.RecordEvent(testutil.TestContext(), testStationEUI, models.EventTypeBSDeregistered, recorderOccurredAt,
		map[string]interface{}{models.EventDetailKeyBaseStationName: removedStationName}))

	assert.Zero(t, stations.lookups)
	assert.Equal(t, removedStationName, events.events[0].data[models.EventDetailKeyBaseStationName])
}

// An unreadable station leaves the event unnamed; the event is still recorded.
func TestNamingEventRecorder_RecordsWithoutANameItCannotRead(t *testing.T) {
	events := &stationEvents{}
	recorder := NewNamingEventRecorder(unreadableStationStore{}, events, logger.NewNop())

	require.NoError(t, recorder.RecordEvent(testutil.TestContext(), testStationEUI, models.EventTypeBaseStationPingSent, recorderOccurredAt,
		map[string]interface{}{models.EventDetailKeyOpID: int64(-1)}))

	require.Len(t, events.events, 1)
	assert.NotContains(t, events.events[0].data, models.EventDetailKeyBaseStationName)
}
