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
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
)

const (
	// boundaryBatch is a page smaller than the rows sharing one time, so the tie spans pages.
	boundaryBatch = 2
	// boundaryTied is how many rows are stored at the shared time before one more.
	boundaryTied = 3
)

// receiveIDs waits for n rows from the stream and then for a quiet spell,
// returning the ids of every row it received.
func receiveIDs[T any](t *testing.T, stream <-chan T, n int, id func(T) string) []string {
	t.Helper()
	var got []string
	for len(got) < n {
		select {
		case row, open := <-stream:
			require.True(t, open, "the stream stays open")
			got = append(got, id(row))
		case <-time.After(wakeTestLatency):
			return got
		}
	}
	select {
	case row := <-stream:
		return append(got, id(row))
	case <-time.After(wakeTestQuiet):
		return got
	}
}

func storeBoundaryEvent(t *testing.T, repos *postgres.Repositories, tenantID int64, at time.Time) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, repos.SystemEvents.CreateEvent(testutil.TestContext(), &models.SystemEvent{
		ID: id, TenantID: strconv.FormatInt(tenantID, 10), EventType: models.EventTypeBSUpdated, Category: models.EventCategoryBaseStation,
		Severity: models.EventSeverityInfo, SourceType: models.SourceTypeServiceCenter, SourceName: t.Name(),
		Title: t.Name(), Description: t.Name(), CreatedAt: at,
	}))
	return id
}

// Events recorded at one whole second, more than a page of them and then one
// more on a later read, all reach an open stream through the PostgreSQL store.
func TestStreams_EventsOnAWholeSecondReachTheStreamThroughPostgres(t *testing.T) {
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
	second := time.Now().Truncate(time.Second).Add(time.Minute)

	var first []string
	for range boundaryTied {
		first = append(first, storeBoundaryEvent(t, repos, tenantID, second))
	}
	assert.ElementsMatch(t, first, receiveIDs(t, stream, boundaryTied, eventID))

	later := storeBoundaryEvent(t, repos, tenantID, second)
	assert.Equal(t, []string{later}, receiveIDs(t, stream, 1, eventID))
}

func storeBoundaryUplink(t *testing.T, repos *postgres.Repositories, tenantID int64, packetCnt uint32, rxTime int64) string {
	t.Helper()
	id := uuid.NewString()
	_, err := repos.UplinkStore.Persist(testutil.TestContext(), models.UplinkPersistRequest{
		Message: &mioty.ULDataMessage{
			ID: id, CommandType: mioty.CmdULData, EpEui: wakeTestEpEui, BsEui: wakeTestBsEui,
			TenantID: tenantID, RxTime: rxTime, PacketCnt: packetCnt, SNR: 12.5, RSSI: -80, UserData: []byte{byte(packetCnt)},
			BaseStations: []mioty.BaseStationReception{{BsEui: wakeTestBsEui, RxTime: rxTime, Snr: 12.5, Rssi: -80}},
		},
		Window: wakeTestWindow,
	})
	require.NoError(t, err)
	return id
}

// Uplinks received at one instant, more than a page of them and then one more
// on a later read, all reach an open stream through the PostgreSQL store.
func TestStreams_UplinksSharingAReceptionTimeReachTheStreamThroughPostgres(t *testing.T) {
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
	rxTime := time.Now().Add(time.Minute).UnixNano()

	var first []string
	for range boundaryTied {
		packetCnt++
		first = append(first, storeBoundaryUplink(t, repos, tenantID, packetCnt, rxTime))
	}
	assert.ElementsMatch(t, first, receiveIDs(t, stream, boundaryTied, uplinkID))

	packetCnt++
	later := storeBoundaryUplink(t, repos, tenantID, packetCnt, rxTime)
	assert.Equal(t, []string{later}, receiveIDs(t, stream, 1, uplinkID))
}
