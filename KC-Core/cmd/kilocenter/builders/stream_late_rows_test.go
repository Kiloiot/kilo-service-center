package builders

import (
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/adapters"
	eventsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/events"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	messagesservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/messages"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// lateBy is how far the late row's own time lies behind the row streamed before it.
const lateBy = time.Hour

// An uplink stored after a newer one has streamed, carrying an older
// reception time (a skewed station clock, a relayed or replayed uplink),
// reaches the open stream.
func TestStreams_AnUplinkStoredLateWithAnOlderReceptionTimeReachesTheStream(t *testing.T) {
	storage, repos, tenantID := wakeTestDatabase(t)
	ctx, cancel := testutil.TestContextWithCancel()
	defer cancel()
	wakes, stop := startStreamWakes(ctx, storage, logger.NewNop())
	t.Cleanup(stop)
	messages := messagesservice.New(adapters.NewMessageListingStoreAdapter(repos.Messages), nil, wakeTestPoll, wakeTestOverlap, wakes.uplinks, boundaryBatch, logger.NewNop())
	stream, err := messages.StreamMessages(ctx, tenantID, &grpcservices.MessageFilters{})
	require.NoError(t, err)
	uplinkID := func(m *mioty.ULDataMessage) string { return m.ID }
	var packetCnt uint32
	warmUp(t, stream, func() { packetCnt++; storeBoundaryUplink(t, repos, tenantID, packetCnt, time.Now().UnixNano()) })

	packetCnt++
	newer := storeBoundaryUplink(t, repos, tenantID, packetCnt, time.Now().Add(lateBy).UnixNano())
	require.Equal(t, []string{newer}, receiveIDs(t, stream, 1, uplinkID))
	packetCnt++
	late := storeBoundaryUplink(t, repos, tenantID, packetCnt, time.Now().Add(-lateBy).UnixNano())
	assert.Equal(t, []string{late}, receiveIDs(t, stream, 1, uplinkID))
}

// An event stored after a newer one has streamed, recorded as having occurred
// earlier, reaches the open stream.
func TestStreams_AnEventStoredLateWithAnEarlierOccurrenceReachesTheStream(t *testing.T) {
	storage, repos, tenantID := wakeTestDatabase(t)
	ctx, cancel := testutil.TestContextWithCancel()
	defer cancel()
	wakes, stop := startStreamWakes(ctx, storage, logger.NewNop())
	t.Cleanup(stop)
	events := eventsservice.New(adapters.NewSystemEventStoreAdapter(repos.SystemEvents),
		adapters.NewEUIResolver(repos.BaseStations, repos.Endpoints), nil, wakeTestPoll, wakeTestOverlap, wakes.events, boundaryBatch, logger.NewNop())
	stream, err := events.Stream(ctx, tenantID, nil)
	require.NoError(t, err)
	eventID := func(e *grpcservices.Event) string { return e.ID }
	warmUp(t, stream, func() { storeBoundaryEvent(t, repos, tenantID, time.Now()) })

	newer := storeBoundaryEvent(t, repos, tenantID, time.Now().Add(lateBy))
	require.Equal(t, []string{newer}, receiveIDs(t, stream, 1, eventID))
	late := storeBoundaryEvent(t, repos, tenantID, time.Now().Add(-lateBy))
	assert.Equal(t, []string{late}, receiveIDs(t, stream, 1, eventID))
}

// A refused connect that recurs moves its record forward, and the stream
// delivers the moved record again even when a newer event has streamed.
func TestStreams_AMovedRefusalReachesTheStreamBehindANewerEvent(t *testing.T) {
	storageCfg, repos, tenantID := wakeTestDatabase(t)
	ctx, cancel := testutil.TestContextWithCancel()
	defer cancel()
	wakes, stop := startStreamWakes(ctx, storageCfg, logger.NewNop())
	t.Cleanup(stop)
	stream := endpointManagerStream(t, repos, tenantID, wakes)
	refuse := func() {
		require.NoError(t, repos.SCACIEvents.RecordSessionEvent(testutil.TestContext(), &models.SCACISessionEvent{
			TenantID: tenantID, Category: models.EventCategorySCACI, EventType: models.EventTypeSCACIConnectRefused,
			AcEui: wakeTestAcEui, ErrorToken: wakeTestRefusalTok, ErrorMessage: t.Name(),
		}))
	}
	storeEndpointEvent := func(at time.Time) string {
		id := uuid.NewString()
		require.NoError(t, repos.SystemEvents.CreateEvent(testutil.TestContext(), &models.SystemEvent{
			ID: id, TenantID: strconv.FormatInt(tenantID, 10), EventType: models.EventTypeEndpointUpdated, Category: models.EventCategoryEndpoint,
			Severity: models.EventSeverityInfo, SourceType: models.SourceTypeServiceCenter, SourceName: t.Name(), Title: t.Name(), Description: t.Name(),
			CreatedAt: at,
		}))
		return id
	}
	warmUp(t, stream, func() { storeEndpointEvent(time.Now()) })

	refuse()
	first := next(t, stream, "the first refusal is stored")
	newer := storeEndpointEvent(time.Now().Add(lateBy))
	require.Equal(t, newer, next(t, stream, "a newer event streams").ID)

	refuse()
	moved := next(t, stream, "the recurrence moves the open record, still behind the newer event, and the stream delivers it again")
	assert.Equal(t, first.ID, moved.ID)
	assert.JSONEq(t, `2`, string(eventData(t, moved)[models.EventDetailKeyCount]))
}
