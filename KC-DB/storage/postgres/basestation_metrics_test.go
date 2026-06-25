package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// metricsWindow is a fixed UTC window used across the bucketed-read tests.
func metricsWindow() (start, end time.Time) {
	start = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return start, start.Add(3 * time.Hour)
}

func insertTestBaseStation(t *testing.T, db *sqlx.DB, eui models.EUI, tenantID int64, name string) int64 {
	t.Helper()
	var id int64
	// basestations.bs_eui is BYTEA(8) (unlike messages.bs_eui which is BIGINT);
	// a bssci base station requires service_center_url (check_bssci_config).
	err := db.QueryRow(`
		INSERT INTO basestations (tenant_id, bs_eui, name, connection_type, service_center_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		tenantID, eui[:], name, string(models.ConnectionTypeBSSCI), "bssci://test",
	).Scan(&id)
	require.NoError(t, err, "insert basestation")
	return id
}

func insertTestSession(t *testing.T, db *sqlx.DB, tenantID, bsID int64, snUUID byte,
	startedAt time.Time, endedAt *time.Time) {
	t.Helper()
	bsUUID := make([]byte, 16)
	scUUID := make([]byte, 16)
	bsUUID[15] = snUUID
	scUUID[15] = snUUID
	_, err := db.Exec(`
		INSERT INTO basestation_sessions (
			basestation_id, tenant_id, sn_bs_uuid, sn_sc_uuid, sn_bs_op_id, sn_sc_op_id,
			status, remote_addr, can_resume, encoding, started_at, ended_at
		) VALUES ($1, $2, $3, $4, 0, 0, 'terminated', '127.0.0.1', false, 'msgpack', $5, $6)`,
		bsID, tenantID, bsUUID, scUUID, startedAt, endedAt,
	)
	require.NoError(t, err, "insert session")
}

func insertTestMessage(t *testing.T, db *sqlx.DB, tenantID, bsEui, epEui int64,
	receivedAt time.Time, baseStations *string) {
	t.Helper()
	var bs interface{}
	if baseStations != nil {
		bs = *baseStations
	}
	_, err := db.Exec(`
		INSERT INTO messages (
			id, tenant_id, ep_eui, bs_eui, op_id, packet_cnt, rx_time,
			rssi, snr, command_type, dl_open, response_exp, dl_ack, received_at, base_stations
		) VALUES ($1, $2, $3, $4, 0, 1, $5, -80.0, 10.0, 'ulData', false, false, false, $6, $7::jsonb)`,
		uuid.NewString(), tenantID, epEui, bsEui, receivedAt.UnixNano(), receivedAt, bs,
	)
	require.NoError(t, err, "insert message")
}

func TestGetBaseStationOnlineIntervals(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	db := &DB{sqlxDB: sqlxDB}
	ctx := context.Background()
	start, end := metricsWindow()

	createTestTenant(t, sqlxDB, 100, "TestTenant100")
	bsID := insertTestBaseStation(t, sqlxDB, models.EUI{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0, 0, 5}, 100, "metrics-bs")

	// In-window session [start, start+90m], and one entirely before the window.
	endedIn := start.Add(90 * time.Minute)
	insertTestSession(t, sqlxDB, 100, bsID, 1, start, &endedIn)
	before := start.Add(-10 * time.Hour)
	beforeEnd := start.Add(-9 * time.Hour)
	insertTestSession(t, sqlxDB, 100, bsID, 2, before, &beforeEnd)

	intervals, err := db.GetBaseStationOnlineIntervals(ctx, 100, bsID, start, end)
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
	db := &DB{sqlxDB: sqlxDB}
	ctx := context.Background()
	start, end := metricsWindow()

	createTestTenant(t, sqlxDB, 200, "TestTenant200")
	bsID := insertTestBaseStation(t, sqlxDB, models.EUI{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0, 0, 6}, 200, "tenant-b-bs")
	endedIn := start.Add(time.Hour)
	insertTestSession(t, sqlxDB, 200, bsID, 1, start, &endedIn)

	// Querying as a different tenant must return nothing.
	intervals, err := db.GetBaseStationOnlineIntervals(ctx, 999, bsID, start, end)
	require.NoError(t, err)
	assert.Empty(t, intervals)
}

func TestCountBaseStationMessagesByBucket_PrimaryAndSecondary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()
	db := &DB{sqlxDB: sqlxDB}
	ctx := context.Background()
	start, end := metricsWindow()
	const tenantID, bsEui int64 = 100, 5
	intervalSeconds := int64(3600)

	// Bucket 0: two primary messages. Bucket 2: one secondary (base_stations JSONB). Bucket 1: none.
	insertTestMessage(t, sqlxDB, tenantID, bsEui, 1, start.Add(10*time.Minute), nil)
	insertTestMessage(t, sqlxDB, tenantID, bsEui, 2, start.Add(20*time.Minute), nil)
	secondary := `[{"bsEui":5,"rssi":-85.0,"snr":8.0}]`
	insertTestMessage(t, sqlxDB, tenantID, 99, 3, start.Add(2*time.Hour+5*time.Minute), &secondary)

	counts, err := db.CountBaseStationMessagesByBucket(ctx, tenantID, []byte{0, 0, 0, 0, 0, 0, 0, 5}, start, end, intervalSeconds)
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
	db := &DB{sqlxDB: sqlxDB}
	ctx := context.Background()
	start, end := metricsWindow()

	insertTestMessage(t, sqlxDB, 200, 5, 1, start.Add(10*time.Minute), nil)

	counts, err := db.CountBaseStationMessagesByBucket(ctx, 100, []byte{0, 0, 0, 0, 0, 0, 0, 5}, start, end, 3600)
	require.NoError(t, err)
	assert.Empty(t, counts, "another tenant's messages are not counted")
}
