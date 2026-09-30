package postgres

import (
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"

	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// minutes and hours express test timeline offsets relative to the metrics window.
func minutes(n int) time.Duration { return time.Duration(n) * time.Minute }

// metricsEUI is the fixture base station EUI for a small integer identifier.
func metricsEUI(n int64) models.EUI {
	var eui models.EUI
	binary.BigEndian.PutUint64(eui[:], uint64(n))
	return eui
}

func hours(n int) time.Duration { return time.Duration(n) * time.Hour }

// metricsWindow is a fixed UTC window used across the bucketed-read tests.
func metricsWindow() (start, end time.Time) {
	start = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return start, start.Add(hours(3))
}

func insertTestBaseStation(t *testing.T, db *sqlx.DB, eui models.EUI, tenantID int64, name string) int64 {
	t.Helper()
	var id int64
	// basestations.bs_eui and messages.ep_eui/bs_eui are BYTEA(8).
	err := db.QueryRow(
		`
		INSERT INTO basestations (tenant_id, bs_eui, name, connection_type, service_center_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		tenantID, eui[:], name, string(models.ConnectionTypeBSSCI), "bssci://test",
	).Scan(&id)
	require.NoError(t, err, "insert basestation")
	return id
}

func insertTestSession(t *testing.T, db *sqlx.DB, tenantID, bsID int64, snUUID byte,
	startedAt time.Time, endedAt *time.Time,
) {
	t.Helper()
	bsUUID := make([]byte, 16)
	scUUID := make([]byte, 16)
	bsUUID[15] = snUUID
	scUUID[15] = snUUID
	_, err := db.Exec(
		`
		INSERT INTO basestation_sessions (
			basestation_id, tenant_id, sn_bs_uuid, sn_sc_uuid, sn_bs_op_id, sn_sc_op_id,
			status, remote_addr, can_resume, encoding, started_at, ended_at
		) VALUES ($1, $2, $3, $4, 0, 0, 'terminated', '127.0.0.1', false, 'msgpack', $5, $6)`,
		bsID, tenantID, bsUUID, scUUID, startedAt, endedAt,
	)
	require.NoError(t, err, "insert session")
}

func insertTestMessage(t *testing.T, db *sqlx.DB, tenantID, bsEui, epEui int64,
	receivedAt time.Time, baseStations *string,
) {
	t.Helper()
	insertTestMessageCmd(t, db, tenantID, bsEui, epEui, "ulData", receivedAt.UnixNano(), receivedAt, baseStations)
}

// insertTestMessageCmd inserts a message with explicit command_type and rx_time (ns).
func insertTestMessageCmd(t *testing.T, db *sqlx.DB, tenantID, bsEui, epEui int64,
	commandType string, rxTimeNs int64, receivedAt time.Time, baseStations *string,
) {
	t.Helper()
	var bs any
	if baseStations != nil {
		bs = *baseStations
	}
	_, err := db.Exec(
		`
		INSERT INTO messages (
			id, tenant_id, ep_eui, bs_eui, op_id, packet_cnt, rx_time,
			rssi, snr, command_type, dl_open, response_exp, dl_ack, received_at, base_stations
		) VALUES ($1, $2, $3, $4, 0, 1, $5, -80.0, 10.0, $6, false, false, false, $7, $8::jsonb)`,
		uuid.NewString(), tenantID, mioty.EUI64Bytes(uint64(epEui)), mioty.EUI64Bytes(uint64(bsEui)), rxTimeNs, commandType, receivedAt, bs,
	)
	require.NoError(t, err, "insert message")
}

func TestGetBaseStationOnlineIntervals(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	db := &DB{clock: clock.SystemClock{}, sqlxDB: sqlxDB}
	ctx := testutil.TestContext()
	start, end := metricsWindow()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")
	bsID := insertTestBaseStation(t, sqlxDB, models.EUI{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0, 0, 5}, 100, "metrics-bs")

	// In-window session [start, start+90m], and one entirely before the window.
	endedIn := start.Add(minutes(90))
	insertTestSession(t, sqlxDB, 100, bsID, 1, start, &endedIn)
	before := start.Add(-hours(10))
	beforeEnd := start.Add(-hours(9))
	insertTestSession(t, sqlxDB, 100, bsID, 2, before, &beforeEnd)

	intervals, err := NewRepositories(db).BaseStationMetrics.GetBaseStationOnlineIntervals(ctx, 100, bsID, start, end)
	require.NoError(t, err)
	require.Len(t, intervals, 1, "only the overlapping session is returned")
	assert.True(t, intervals[0].Start.Equal(start))
	require.NotNil(t, intervals[0].End)
	assert.True(t, intervals[0].End.Equal(endedIn))
}

func TestGetBaseStationOnlineIntervals_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	db := &DB{clock: clock.SystemClock{}, sqlxDB: sqlxDB}
	ctx := testutil.TestContext()
	start, end := metricsWindow()

	createTestTenant(t, sqlxDB, 200, "TestTenant200")
	bsID := insertTestBaseStation(t, sqlxDB, models.EUI{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0, 0, 6}, 200, "tenant-b-bs")
	endedIn := start.Add(time.Hour)
	insertTestSession(t, sqlxDB, 200, bsID, 1, start, &endedIn)

	// Querying as a different tenant must return nothing.
	intervals, err := NewRepositories(db).BaseStationMetrics.GetBaseStationOnlineIntervals(ctx, 999, bsID, start, end)
	require.NoError(t, err)
	assert.Empty(t, intervals)
}

func TestCountBaseStationMessagesByBucket_PrimaryAndSecondary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	db := &DB{clock: clock.SystemClock{}, sqlxDB: sqlxDB}
	ctx := testutil.TestContext()
	start, end := metricsWindow()
	const tenantID, bsEui int64 = 100, 5
	intervalSeconds := int64(3600)
	createTestTenant(t, sqlxDB, tenantID, "bucket-primary-tenant")
	insertTestBaseStation(t, sqlxDB, metricsEUI(bsEui), tenantID, "bucket-primary")

	// Bucket 0: two primary messages. Bucket 2: one secondary (base_stations JSONB). Bucket 1: none.
	insertTestMessage(t, sqlxDB, tenantID, bsEui, 1, start.Add(minutes(10)), nil)
	insertTestMessage(t, sqlxDB, tenantID, bsEui, 2, start.Add(minutes(20)), nil)
	secondary := `[{"bsEui":5,"rssi":-85.0,"snr":8.0}]`
	insertTestMessage(t, sqlxDB, tenantID, 99, 3, start.Add(hours(2)+minutes(5)), &secondary)

	counts, err := NewRepositories(db).BaseStationMetrics.CountBaseStationMessagesByBucket(ctx, tenantID, []byte{0, 0, 0, 0, 0, 0, 0, 5}, start, end, intervalSeconds)
	require.NoError(t, err)

	bucket0 := start.Unix() / intervalSeconds
	assert.Equal(t, int64(2), counts[bucket0], "two primary messages in bucket 0")
	assert.Equal(t, int64(1), counts[bucket0+2], "one secondary-receiver message in bucket 2")
	_, hasBucket1 := counts[bucket0+1]
	assert.False(t, hasBucket1, "empty bucket is absent (caller zero-fills)")
}

func TestCountBaseStationMessagesByBucket_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	db := &DB{clock: clock.SystemClock{}, sqlxDB: sqlxDB}
	ctx := testutil.TestContext()
	start, end := metricsWindow()

	createTestTenant(t, sqlxDB, 200, "bucket-owner-tenant")
	insertTestBaseStation(t, sqlxDB, metricsEUI(5), 200, "owned-by-200")
	insertTestMessage(t, sqlxDB, 100, 5, 1, start.Add(minutes(10)), nil)

	counts, err := NewRepositories(db).BaseStationMetrics.CountBaseStationMessagesByBucket(ctx, 100, []byte{0, 0, 0, 0, 0, 0, 0, 5}, start, end, 3600)
	require.NoError(t, err)
	assert.Empty(t, counts, "a base station the tenant does not own reports nothing, even for the tenant's own messages")

	owner, err := NewRepositories(db).BaseStationMetrics.CountBaseStationMessagesByBucket(ctx, 200, []byte{0, 0, 0, 0, 0, 0, 0, 5}, start, end, 3600)
	require.NoError(t, err)
	assert.Equal(t, int64(1), owner[start.Unix()/3600], "the owner counts receptions of another tenant's endpoint as load")
}

// TestCountBaseStationMessagesByBucket_FullRangeEUIs covers EUIs with the top
// bit set: a reception from one anywhere must not break another tenant's
// metrics, and a station with such an EUI counts its own receptions.
func TestCountBaseStationMessagesByBucket_FullRangeEUIs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	db := &DB{clock: clock.SystemClock{}, sqlxDB: sqlxDB}
	ctx := testutil.TestContext()
	start, end := metricsWindow()
	const intervalSeconds = int64(3600)
	const tenantA, tenantB, lowBs int64 = 100, 200, 5
	highBsEui := uint64(0xF0B3D59CD0000161)
	var highBs models.EUI
	binary.BigEndian.PutUint64(highBs[:], highBsEui)
	createTestTenant(t, sqlxDB, tenantA, "bucket-low-eui-tenant")
	createTestTenant(t, sqlxDB, tenantB, "bucket-high-eui-tenant")
	insertTestBaseStation(t, sqlxDB, metricsEUI(lowBs), tenantA, "low-eui-station")
	insertTestBaseStation(t, sqlxDB, highBs, tenantB, "high-eui-station")

	lowReception := `[{"bsEui":5,"rssi":-85.0,"snr":8.0}]`
	insertTestMessage(t, sqlxDB, tenantA, 99, 1, start.Add(minutes(10)), &lowReception)
	highReception := fmt.Sprintf(`[{"bsEui":%d,"rssi":-85.0,"snr":8.0}]`, highBsEui)
	insertTestMessage(t, sqlxDB, tenantB, 99, 2, start.Add(minutes(20)), &highReception)

	bucket0 := start.Unix() / intervalSeconds
	metrics := NewRepositories(db).BaseStationMetrics
	lowEui := metricsEUI(lowBs)
	low, err := metrics.CountBaseStationMessagesByBucket(ctx, tenantA, lowEui[:], start, end, intervalSeconds)
	require.NoError(t, err, "another tenant's full-range EUI reception must not break these metrics")
	assert.Equal(t, int64(1), low[bucket0])

	high, err := metrics.CountBaseStationMessagesByBucket(ctx, tenantB, highBs[:], start, end, intervalSeconds)
	require.NoError(t, err, "a station whose EUI has the top bit set has metrics too")
	assert.Equal(t, int64(1), high[bucket0])
}

func TestCountBaseStationMessagesByBucket_ExcludesNonUlData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	db := &DB{clock: clock.SystemClock{}, sqlxDB: sqlxDB}
	ctx := testutil.TestContext()
	start, end := metricsWindow()
	const tenantID, bsEui int64 = 100, 7
	intervalSeconds := int64(3600)
	createTestTenant(t, sqlxDB, tenantID, "bucket-uldata-tenant")
	insertTestBaseStation(t, sqlxDB, metricsEUI(bsEui), tenantID, "bucket-ulData")

	// The only row that should be counted.
	insertTestMessageCmd(t, sqlxDB, tenantID, bsEui, 1, "ulData", start.Add(minutes(10)).UnixNano(), start.Add(minutes(10)), nil)
	// Propagate/completion rows (rx_time=0 in production) must be excluded.
	insertTestMessageCmd(t, sqlxDB, tenantID, bsEui, 2, "attPrp", 0, start.Add(minutes(15)), nil)
	insertTestMessageCmd(t, sqlxDB, tenantID, bsEui, 3, "detPrp", 0, start.Add(minutes(20)), nil)
	insertTestMessageCmd(t, sqlxDB, tenantID, bsEui, 4, "attPrpCmp", 0, start.Add(minutes(25)), nil)

	counts, err := NewRepositories(db).BaseStationMetrics.CountBaseStationMessagesByBucket(ctx, tenantID, []byte{0, 0, 0, 0, 0, 0, 0, 7}, start, end, intervalSeconds)
	require.NoError(t, err)

	bucket0 := start.Unix() / intervalSeconds
	assert.Equal(t, int64(1), counts[bucket0], "only the ulData uplink is counted")
	assert.Len(t, counts, 1, "propagate/completion messages are excluded")
}

func TestGetBaseStationEndpointCounts_PrimaryAndSecondary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	repo := NewMessageRepository(sqlxDB, clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()
	start, _ := metricsWindow()
	const tenantID, bsEui int64 = 100, 5

	// ep 10 primary (x2); ep 20 secondary receiver (JSONB); ep 30 another bs (excluded).
	insertTestMessage(t, sqlxDB, tenantID, bsEui, 10, start.Add(minutes(1)), nil)
	insertTestMessage(t, sqlxDB, tenantID, bsEui, 10, start.Add(minutes(2)), nil)
	secondary := `[{"bsEui":5,"rssi":-85.0,"snr":8.0}]`
	insertTestMessage(t, sqlxDB, tenantID, 99, 20, start.Add(minutes(3)), &secondary)
	insertTestMessage(t, sqlxDB, tenantID, 99, 30, start.Add(minutes(4)), nil)
	insertTestMessage(t, sqlxDB, 200, bsEui, 10, start.Add(minutes(5)), nil) // other tenant

	counts, err := repo.GetBaseStationEndpointCounts(ctx, tenantID, []byte{0, 0, 0, 0, 0, 0, 0, 5}, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, int64(2), counts[mioty.FormatEUI64(uint64(10))], "two primary messages for ep 10")
	assert.Equal(t, int64(1), counts[mioty.FormatEUI64(uint64(20))], "one secondary-receiver message for ep 20")
	_, has30 := counts[mioty.FormatEUI64(uint64(30))]
	assert.False(t, has30, "endpoint seen only by another bs is excluded")
	assert.Len(t, counts, 2, "only ep 10 and ep 20 for this bs/tenant")
}

func TestGetBaseStationLastSeen_FromMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	repo := NewMessageRepository(sqlxDB, clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()
	start, _ := metricsWindow()
	const tenantID, bsEui int64 = 100, 5

	insertTestMessage(t, sqlxDB, tenantID, bsEui, 10, start.Add(minutes(1)), nil) // primary, older
	latest := start.Add(minutes(5))
	secondary := `[{"bsEui":5,"rssi":-80.0,"snr":9.0}]`
	insertTestMessage(t, sqlxDB, tenantID, 99, 20, latest, &secondary) // secondary, newest

	lastSeen, err := repo.GetBaseStationLastSeen(ctx, tenantID, []byte{0, 0, 0, 0, 0, 0, 0, 5})
	require.NoError(t, err)
	require.NotNil(t, lastSeen)
	assert.WithinDuration(t, latest, *lastSeen, time.Second, "newest received_at, including secondary receiver")
}

func TestGetBaseStationLastSeen_SessionFallbackAndEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	repo := NewMessageRepository(sqlxDB, clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()
	start, _ := metricsWindow()
	const tenantID int64 = 100

	createTestTenant(t, sqlxDB, tenantID, "TestTenant100")
	bsID := insertTestBaseStation(t, sqlxDB, models.EUI{0, 0, 0, 0, 0, 0, 0, 5}, tenantID, "no-msg-bs")
	sessionStart := start.Add(minutes(30))
	ended := sessionStart.Add(time.Hour)
	insertTestSession(t, sqlxDB, tenantID, bsID, 1, sessionStart, &ended)

	// no messages → session fallback (started_at).
	lastSeen, err := repo.GetBaseStationLastSeen(ctx, tenantID, []byte{0, 0, 0, 0, 0, 0, 0, 5})
	require.NoError(t, err)
	require.NotNil(t, lastSeen)
	assert.WithinDuration(t, sessionStart, *lastSeen, time.Second, "falls back to session start")

	// Never-connected bs (no messages, no sessions) → nil, not a 502.
	none, err := repo.GetBaseStationLastSeen(ctx, tenantID, []byte{0, 0, 0, 0, 0, 0, 0, 9})
	require.NoError(t, err)
	assert.Nil(t, none, "no messages and no sessions returns nil")
}
