package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// ackEndpointEUI is the endpoint whose downlinks the acknowledgement tests seed.
const ackEndpointEUI uint64 = 0x70B3D59CD0000341

// seedWindowDownlink stores a downlink of the endpoint in status, sent in the
// window of packetCnt.
func seedWindowDownlink(t *testing.T, db *sqlx.DB, tenantID int64, orgID uuid.UUID, queID int64, status mioty.DLQueueStatus, packetCnt int64) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO downlink_queue (que_id, ep_eui, tenant_id, organization_id, payload, status, priority,
			transmission_packet_cnt, transmitted_at, earliest_at, latest_at)
		VALUES ($1, $2, $3, $4, $5, $6, 0, $7, NOW(), NULL, NULL)`,
		queID, mioty.EUI64Bytes(ackEndpointEUI), tenantID, orgID, []byte{0x01}, status, packetCnt)
	require.NoError(t, err)
}

func endpointAckedAt(t *testing.T, db *sqlx.DB, queID int64) *string {
	t.Helper()
	var ackedAt *string
	require.NoError(t, db.QueryRow(`SELECT endpoint_acked_at::text FROM downlink_queue WHERE que_id = $1`, queID).Scan(&ackedAt))
	return ackedAt
}

// TestMarkEndpointAcknowledged_MarksTheDownlinkOfThePreviousWindow pins
// BSSCI §3.10.1: dlAck acknowledges the downlink sent in the window of
// packetCnt - 1, under the endpoint owner's tenant only, and only once.
func TestMarkEndpointAcknowledged_MarksTheDownlinkOfThePreviousWindow(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	seedWindowDownlink(t, db, 321, orgs[321], 840001, mioty.DLQueueStatusTransmitted, 41)
	seedWindowDownlink(t, db, 321, orgs[321], 840002, mioty.DLQueueStatusTransmitted, 40)
	seedWindowDownlink(t, db, 321, orgs[321], 840003, mioty.DLQueueStatusQueued, 41)
	seedWindowDownlink(t, db, 322, orgs[322], 840004, mioty.DLQueueStatusTransmitted, 41)

	acknowledged, marked, err := downlinks.MarkEndpointAcknowledged(t.Context(), 321, ackEndpointEUI, 41)

	require.NoError(t, err)
	assert.True(t, marked)
	assert.Equal(t, int64(840001), acknowledged.QueID, "the acknowledged downlink is named")
	assert.Equal(t, "321", acknowledged.TenantID)
	require.NotNil(t, acknowledged.OrganizationID)
	assert.Equal(t, orgs[321], *acknowledged.OrganizationID, "the row names the organization its acknowledgement is published to")
	first := endpointAckedAt(t, db, 840001)
	require.NotNil(t, first, "the downlink of window 41 is acknowledged")
	assert.Nil(t, endpointAckedAt(t, db, 840002), "another window's downlink is not")
	assert.Nil(t, endpointAckedAt(t, db, 840003), "a downlink not transmitted cannot have been received")
	assert.Nil(t, endpointAckedAt(t, db, 840004), "another tenant's downlink is never touched")

	_, again, err := downlinks.MarkEndpointAcknowledged(t.Context(), 321, ackEndpointEUI, 41)
	require.NoError(t, err)
	assert.False(t, again, "a repeated acknowledgement marks nothing new")
	assert.Equal(t, first, endpointAckedAt(t, db, 840001))
}

// TestMarkEndpointAcknowledged_AfterACounterResetMarksTheNewestTransmission:
// an endpoint whose counter restarted reuses a window, so its dlAck
// acknowledges the downlink transmitted last in that window; the older one
// stays unacknowledged.
func TestMarkEndpointAcknowledged_AfterACounterResetMarksTheNewestTransmission(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	seedWindowDownlink(t, db, 321, orgs[321], 840021, mioty.DLQueueStatusTransmitted, 5)
	seedWindowDownlink(t, db, 321, orgs[321], 840022, mioty.DLQueueStatusTransmitted, 5)
	_, err := db.Exec(`UPDATE downlink_queue SET transmitted_at = transmitted_at - INTERVAL '1 hour' WHERE que_id = 840021`)
	require.NoError(t, err)

	acknowledged, marked, err := downlinks.MarkEndpointAcknowledged(t.Context(), 321, ackEndpointEUI, 5)

	require.NoError(t, err)
	require.True(t, marked)
	assert.Equal(t, int64(840022), acknowledged.QueID)
	assert.NotNil(t, endpointAckedAt(t, db, 840022))
	assert.Nil(t, endpointAckedAt(t, db, 840021), "the transmission before the reset is not the one acknowledged")
}

// TestMarkEndpointAcknowledged_ConcurrentReceptionsMarkOnce: the receptions
// of one uplink by several base stations arrive together; a reception that
// picked the downlink while another held it marks nothing once the other
// commits, so the acknowledgement is reported once.
func TestMarkEndpointAcknowledged_ConcurrentReceptionsMarkOnce(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	seedWindowDownlink(t, db, 321, orgs[321], 840031, mioty.DLQueueStatusTransmitted, 9)
	first, err := db.BeginTxx(t.Context(), nil)
	require.NoError(t, err)
	_, err = first.Exec(sqlMarkEndpointAcknowledged, 321, mioty.EUI64Bytes(ackEndpointEUI), 9, mioty.DLQueueStatusTransmitted, time.Now())
	require.NoError(t, err)

	second := make(chan bool)
	go func() {
		_, marked, err := downlinks.MarkEndpointAcknowledged(t.Context(), 321, ackEndpointEUI, 9)
		assert.NoError(t, err)
		second <- marked
	}()
	waitForBlockedUpdate(t, db)
	require.NoError(t, first.Commit())

	assert.False(t, <-second, "the second reception finds the downlink already acknowledged")
}

// waitForBlockedUpdate waits until a session of the test database waits on a row lock.
func waitForBlockedUpdate(t *testing.T, db *sqlx.DB) {
	t.Helper()
	require.Eventually(t, func() bool {
		var waiting int
		require.NoError(t, db.QueryRow(`SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&waiting))
		return waiting > 0
	}, 10*time.Second, 10*time.Millisecond)
}

// TestGetDownlinkResults_ReportsTheEndpointAcknowledgement: the results
// listing carries the acknowledgement time.
func TestGetDownlinkResults_ReportsTheEndpointAcknowledgement(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	seedWindowDownlink(t, db, 321, orgs[321], 840011, mioty.DLQueueStatusTransmitted, 7)
	seedWindowDownlink(t, db, 321, orgs[321], 840012, mioty.DLQueueStatusTransmitted, 8)
	_, _, err := downlinks.MarkEndpointAcknowledged(t.Context(), 321, ackEndpointEUI, 7)
	require.NoError(t, err)

	results, _, err := downlinks.GetDownlinkResults(t.Context(), 321, nil, storage.DownlinkResultFilter{}, 10, 0)

	require.NoError(t, err)
	acked := map[int64]bool{}
	for _, dl := range results {
		acked[dl.QueID] = dl.EndpointAckedAt != nil
	}
	assert.Equal(t, map[int64]bool{840011: true, 840012: false}, acked)
}
