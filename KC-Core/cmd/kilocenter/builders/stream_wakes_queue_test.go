package builders

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/adapters"
	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	eventsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/events"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	pkgclock "github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	wakeTestQueueID    = int64(918301)
	wakeTestAcEui      = "70B3D5E75E000183"
	wakeTestRefusalTok = "scaciErrCertificateUnknown"
)

// endpointManagerStream opens the event stream an endpoint manager holds:
// the categories its roles read, as the StreamEvents handler narrows them.
func endpointManagerStream(t *testing.T, repos *postgres.Repositories, tenantID int64, wakes streamWakes) <-chan *grpcservices.Event {
	t.Helper()
	categories, unrestricted := authz.VisibleEventCategories(authz.Roles{EndpointManager: true}, nil)
	require.False(t, unrestricted)
	events := eventsservice.New(adapters.NewSystemEventStoreAdapter(repos.SystemEvents),
		adapters.NewEUIResolver(repos.BaseStations, repos.Endpoints), nil, wakeTestPoll, wakeTestOverlap, wakes.events, 0, logger.NewNop())
	ctx, cancel := testutil.TestContextWithCancel()
	t.Cleanup(cancel)
	stream, err := events.Stream(ctx, tenantID, &grpcservices.EventFilters{Categories: categories})
	require.NoError(t, err)
	return stream
}

// warmUp stores rows the stream reads until one arrives, which proves the
// listener listens, then drains what the warm-up left.
func warmUp[T any](t *testing.T, stream <-chan T, store func()) {
	t.Helper()
	require.Eventually(t, func() bool {
		store()
		return received(stream, wakeTestLatency)
	}, wakeTestWarmUp, time.Millisecond, "the listener starts listening")
	leftover := received(stream, wakeTestQuiet)
	for leftover {
		leftover = received(stream, wakeTestQuiet)
	}
}

// next is the event the stream delivers within the wake latency.
func next(t *testing.T, stream <-chan *grpcservices.Event, why string) *grpcservices.Event {
	t.Helper()
	select {
	case event, open := <-stream:
		require.True(t, open, why)
		return event
	case <-time.After(wakeTestLatency):
		require.FailNow(t, "no event within the wake latency", why)
		return nil
	}
}

func eventData(t *testing.T, event *grpcservices.Event) map[string]json.RawMessage {
	t.Helper()
	var data map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(event.Data, &data))
	return data
}

// A repeated refused connect moves its open record to the new time instead of
// storing another; an open stream delivers the moved record at once, with
// its new count, long before the poll.
func TestStreamWakes_AMovedEventReachesItsStreamLongBeforeThePoll(t *testing.T) {
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
	warmUp(t, stream, func() {
		require.NoError(t, repos.SystemEvents.CreateEvent(testutil.TestContext(), &models.SystemEvent{
			TenantID: strconv.FormatInt(tenantID, 10), EventType: models.EventTypeEndpointUpdated, Category: models.EventCategoryEndpoint,
			Severity: models.EventSeverityInfo, SourceType: models.SourceTypeServiceCenter, SourceName: t.Name(), Title: t.Name(), Description: t.Name(),
		}))
	})

	refuse()
	first := next(t, stream, "the first refusal is stored")
	require.Equal(t, models.EventTypeSCACIConnectRefused, first.EventType)
	assert.JSONEq(t, `1`, string(eventData(t, first)[models.EventDetailKeyCount]))

	refuse()
	moved := next(t, stream, "the recurrence moves the open record, and the stream delivers it again")
	assert.Equal(t, first.ID, moved.ID, "the same record, not a new one")
	assert.True(t, moved.Timestamp.After(first.Timestamp))
	assert.JSONEq(t, `2`, string(eventData(t, moved)[models.EventDetailKeyCount]))
}

// Every change of the downlink queue the service center records reaches an
// endpoint manager's open event stream at once, naming the endpoint and the
// queue id its views are scoped by.
func TestStreamWakes_EveryQueueChangeReachesAnEndpointManagersStream(t *testing.T) {
	storageCfg, repos, tenantID := wakeTestDatabase(t)
	ctx, cancel := testutil.TestContextWithCancel()
	defer cancel()
	wakes, stop := startStreamWakes(ctx, storageCfg, logger.NewNop())
	t.Cleanup(stop)
	stream := endpointManagerStream(t, repos, tenantID, wakes)
	queue, err := bssciservices.NewAuditLogger(bssciservices.AuditLogDeps{
		Events: repos.SystemEvents, Downlinks: repos.Downlinks, Stations: repos.BaseStations,
		Clock: pkgclock.SystemClock{}, Logger: logger.NewNop(),
	})
	require.NoError(t, err)
	downlink := &storage.DownlinkMessage{
		EPEUI: mioty.FormatEUI64(wakeTestEpEui), TenantID: strconv.FormatInt(tenantID, 10), QueID: wakeTestQueueID, Payload: []byte{0x33},
	}
	warmUp(t, stream, func() { require.NoError(t, queue.RecordEnqueued(testutil.TestContext(), downlink)) })

	for eventType, record := range map[string]func() error{
		models.EventTypeDLDataEnqueued: func() error { return queue.RecordEnqueued(testutil.TestContext(), downlink) },
		models.EventTypeDLDataUpdated:  func() error { return queue.RecordPendingUpdated(testutil.TestContext(), downlink) },
		models.EventTypeDLDataRevoked:  func() error { return queue.RecordQueueRevoked(testutil.TestContext(), downlink) },
		models.EventTypeDLDataRequeued: func() error {
			return queue.RecordRequeued(testutil.TestContext(), storage.PendingDownlink{
				QueID: uint64(wakeTestQueueID), TenantID: tenantID, OrganizationID: uuid.New(), EpEUI: wakeTestEpEui,
			}, wakeTestBsEui)
		},
	} {
		require.NoError(t, record())
		event := next(t, stream, eventType)
		assert.Equal(t, eventType, event.EventType)
		assert.Equal(t, models.EventCategoryMessage, event.Category)
		data := eventData(t, event)
		assert.JSONEq(t, `"`+mioty.FormatEUI64(wakeTestEpEui)+`"`, string(data[models.EventDetailKeyEpEui]), eventType)
		assert.JSONEq(t, `918301`, string(data[models.EventDetailKeyQueID]), eventType)
	}
}
