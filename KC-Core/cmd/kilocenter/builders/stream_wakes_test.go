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
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
)

const (
	// wakeTestPoll is a poll interval no assertion waits out, so every row arrives on a wake.
	wakeTestPoll = time.Minute
	// wakeTestOverlap is how far each stream read reaches back in storage order.
	wakeTestOverlap = time.Second
	// wakeTestLatency is the most a stored row may take to reach an open stream.
	wakeTestLatency = time.Second
	// wakeTestWarmUp bounds how long the listener may take to start listening.
	wakeTestWarmUp = 10 * time.Second
	// wakeTestQuiet is how long a stream stays quiet once the warm-up rows are all delivered.
	wakeTestQuiet  = 200 * time.Millisecond
	wakeTestRows   = 5
	wakeTestWindow = 5 * time.Minute
	wakeTestEpEui  = uint64(0x70B3D5677011A183)
	wakeTestBsEui  = uint64(0x70B3D59CD000A183)
)

// wakeTestDatabase opens a migrated database the way the service does,
// through the storage configuration, with one tenant and one endpoint.
func wakeTestDatabase(t *testing.T) (pkgconfig.StorageConfig, *postgres.Repositories, int64) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping stream wake integration test in short mode")
	}
	dsn, drop := testsupport.NewMigratedDatabase(t)
	t.Cleanup(drop)
	cfg, err := testsupport.ParseDSN(dsn)
	require.NoError(t, err)
	port, err := strconv.Atoi(cfg.Port)
	require.NoError(t, err)
	storage := pkgconfig.StorageConfig{Host: cfg.Host, Port: port, Database: cfg.Database,
		Username: cfg.User, Password: cfg.Password, SSLMode: "disable"}
	db, err := postgres.New(StorageOptions(storage), testsupport.TestCipher())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, db.Close()) })

	var tenantID int64
	require.NoError(t, db.Sqlx().QueryRow(`INSERT INTO tenants (name) VALUES ($1) RETURNING id`, t.Name()).Scan(&tenantID))
	_, err = db.Sqlx().Exec(`INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name) VALUES ($1, $1, $2, $3)`,
		tenantID, mioty.EUI64Bytes(wakeTestEpEui), t.Name())
	require.NoError(t, err)
	return storage, postgres.NewRepositories(db), tenantID
}

func storeWakeTestUplink(t *testing.T, repos *postgres.Repositories, tenantID int64, packetCnt uint32) {
	t.Helper()
	rxTime := time.Now().UnixNano()
	_, err := repos.UplinkStore.Persist(testutil.TestContext(), models.UplinkPersistRequest{
		Message: &mioty.ULDataMessage{
			ID: uuid.NewString(), CommandType: mioty.CmdULData, EpEui: wakeTestEpEui, BsEui: wakeTestBsEui,
			TenantID: tenantID, RxTime: rxTime, PacketCnt: packetCnt, SNR: 12.5, RSSI: -80, UserData: []byte{byte(packetCnt)},
			BaseStations: []mioty.BaseStationReception{{BsEui: wakeTestBsEui, RxTime: rxTime, Snr: 12.5, Rssi: -80}},
		},
		Window: wakeTestWindow,
	})
	require.NoError(t, err)
}

func storeWakeTestEvent(t *testing.T, repos *postgres.Repositories, tenantID int64) {
	t.Helper()
	require.NoError(t, repos.SystemEvents.CreateEvent(testutil.TestContext(), &models.SystemEvent{
		TenantID: strconv.FormatInt(tenantID, 10), EventType: models.EventTypeBSUpdated, Category: models.EventCategoryBaseStation,
		Severity: models.EventSeverityInfo, SourceType: models.SourceTypeServiceCenter, SourceName: t.Name(),
		Title: t.Name(), Description: t.Name(),
	}))
}

// received reports whether the stream delivers a row within d.
func received[T any](stream <-chan T, d time.Duration) bool {
	select {
	case _, open := <-stream:
		return open
	case <-time.After(d):
		return false
	}
}

// slowest stores rows one at a time and returns the longest a row took to
// reach the stream, after a warm-up that waits for the listener to listen.
func slowest[T any](t *testing.T, stream <-chan T, store func()) time.Duration {
	t.Helper()
	require.Eventually(t, func() bool {
		store()
		return received(stream, wakeTestLatency)
	}, wakeTestWarmUp, time.Millisecond, "the listener starts listening")
	// A warm-up row stored before the listener listened arrives with the next wake; it is not measured.
	leftover := received(stream, wakeTestQuiet)
	for leftover {
		leftover = received(stream, wakeTestQuiet)
	}
	var worst time.Duration
	for range wakeTestRows {
		start := time.Now()
		store()
		require.True(t, received(stream, wakeTestLatency), "a stored row reaches the stream within %s", wakeTestLatency)
		worst = max(worst, time.Since(start))
	}
	return worst
}

// With the poll a minute apart, a stored uplink and a stored event reach
// their open streams within a second: the database announces them.
func TestStreamWakes_AStoredRowReachesItsStreamLongBeforeThePoll(t *testing.T) {
	storage, repos, tenantID := wakeTestDatabase(t)
	log := logger.NewNop()
	ctx, cancel := testutil.TestContextWithCancel()
	defer cancel()
	wakes, stop := startStreamWakes(ctx, storage, log)
	t.Cleanup(stop)

	events := eventsservice.New(adapters.NewSystemEventStoreAdapter(repos.SystemEvents),
		adapters.NewEUIResolver(repos.BaseStations, repos.Endpoints), nil, wakeTestPoll, wakeTestOverlap, wakes.events, 0, log)
	eventStream, err := events.Stream(ctx, tenantID, nil)
	require.NoError(t, err)
	messages := messagesservice.New(adapters.NewMessageListingStoreAdapter(repos.Messages), nil, wakeTestPoll, wakeTestOverlap, wakes.uplinks, 0, log)
	uplinkStream, err := messages.StreamMessages(ctx, tenantID, &grpcservices.MessageFilters{})
	require.NoError(t, err)

	eventLatency := slowest(t, eventStream, func() { storeWakeTestEvent(t, repos, tenantID) })
	var packetCnt uint32
	uplinkLatency := slowest(t, uplinkStream, func() { packetCnt++; storeWakeTestUplink(t, repos, tenantID, packetCnt) })

	t.Logf("slowest stored-to-streamed latency over %d rows: event %s, uplink %s (poll interval %s)", wakeTestRows, eventLatency, uplinkLatency, wakeTestPoll)
	assert.Less(t, eventLatency, wakeTestLatency)
	assert.Less(t, uplinkLatency, wakeTestLatency)
}
