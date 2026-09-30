package postgres

import (
	"testing"

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

	queID, marked, err := downlinks.MarkEndpointAcknowledged(t.Context(), 321, ackEndpointEUI, 41)

	require.NoError(t, err)
	assert.True(t, marked)
	assert.Equal(t, int64(840001), queID, "the acknowledged downlink is named")
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
