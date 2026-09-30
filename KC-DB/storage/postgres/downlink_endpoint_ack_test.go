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
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// ackEndpointEUI is the endpoint whose downlinks the acknowledgement tests seed.
const ackEndpointEUI uint64 = 0x70B3D59CD0000341

// ackStationEUI is the base station that received the acknowledging uplinks.
const ackStationEUI uint64 = 0x70B3D59CD0000342

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

// acknowledgement is the dlAck of an uplink stored as messageID for the
// window, queued on channels.
func acknowledgement(tenantID, window int64, messageID string, channels ...models.DeliveryChannel) models.EndpointAckRequest {
	return models.EndpointAckRequest{TenantID: tenantID, EpEUI: ackEndpointEUI, WindowPacketCnt: window, MessageID: messageID, Channels: channels}
}

// storeAcknowledgingUplink stores the tenant's uplink after the window and
// returns its message id.
func storeAcknowledgingUplink(t *testing.T, db *sqlx.DB, tenantID, window int64) string {
	t.Helper()
	req := uplinkRequest(tenantID, ackEndpointEUI, ackStationEUI, uint32(window+1), []byte{0x02})
	_, err := NewUplinkStore(db, clock.SystemClock{}, logger.Get()).Persist(t.Context(), req)
	require.NoError(t, err)
	return req.Message.ID
}

// ackDeliveryRow is the outbox row an acknowledgement is queued as.
type ackDeliveryRow struct {
	MessageID              string                 `db:"message_id"`
	Channel                models.DeliveryChannel `db:"channel"`
	OwnerTenantID          int64                  `db:"owner_tenant_id"`
	AcknowledgedDownlinkID *int64                 `db:"acknowledged_downlink_id"`
	Status                 string                 `db:"status"`
}

func ackDeliveryRows(t *testing.T, db *sqlx.DB) []ackDeliveryRow {
	t.Helper()
	var rows []ackDeliveryRow
	require.NoError(t, db.Select(&rows, `SELECT message_id::text, channel, owner_tenant_id, acknowledged_downlink_id, status
		FROM message_delivery_outbox WHERE acknowledged_downlink_id IS NOT NULL`))
	return rows
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

	acknowledged, marked, err := downlinks.MarkEndpointAcknowledged(t.Context(), acknowledgement(321, 41, uuid.NewString()))

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

	_, again, err := downlinks.MarkEndpointAcknowledged(t.Context(), acknowledgement(321, 41, uuid.NewString()))
	require.NoError(t, err)
	assert.False(t, again, "a repeated acknowledgement marks nothing new")
	assert.Equal(t, first, endpointAckedAt(t, db, 840001))
	assert.Empty(t, ackDeliveryRows(t, db), "an acknowledgement queued on no channel queues no row")
}

// TestMarkEndpointAcknowledged_QueuesTheAcknowledgementWithTheMark: the
// statement that marks the downlink queues one pending row of the uplink
// that carried the acknowledgement, naming the downlink, under its tenant; a
// repeated acknowledgement, by the same uplink or another, queues none.
func TestMarkEndpointAcknowledged_QueuesTheAcknowledgementWithTheMark(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	seedUplinkEndpoint(t, db, 321, ackEndpointEUI)
	seedWindowDownlink(t, db, 321, orgs[321], 840051, mioty.DLQueueStatusTransmitted, 41)
	uplink := storeAcknowledgingUplink(t, db, 321, 41)

	acknowledged, marked, err := downlinks.MarkEndpointAcknowledged(t.Context(),
		acknowledgement(321, 41, uplink, models.DeliveryChannelMQTTDownlinkAck))
	require.NoError(t, err)
	require.True(t, marked)

	want := []ackDeliveryRow{{
		MessageID: uplink, Channel: models.DeliveryChannelMQTTDownlinkAck, OwnerTenantID: 321,
		AcknowledgedDownlinkID: &acknowledged.ID, Status: models.DeliveryStatusPending,
	}}
	assert.Equal(t, want, ackDeliveryRows(t, db))

	_, again, err := downlinks.MarkEndpointAcknowledged(t.Context(),
		acknowledgement(321, 41, uplink, models.DeliveryChannelMQTTDownlinkAck))
	require.NoError(t, err)
	assert.False(t, again)
	assert.Equal(t, want, ackDeliveryRows(t, db), "the acknowledgement is queued once")
}

// TestMarkEndpointAcknowledged_AFailedQueueLeavesTheDownlinkUnmarked: the
// mark and its delivery row commit together, so an acknowledgement whose row
// cannot be queued is not recorded and the next reception of the uplink
// records it again.
func TestMarkEndpointAcknowledged_AFailedQueueLeavesTheDownlinkUnmarked(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	seedWindowDownlink(t, db, 321, orgs[321], 840061, mioty.DLQueueStatusTransmitted, 41)

	_, _, err := downlinks.MarkEndpointAcknowledged(t.Context(),
		acknowledgement(321, 41, uuid.NewString(), models.DeliveryChannelMQTTDownlinkAck))

	require.Error(t, err, "a row for an uplink that was never stored violates the outbox foreign key")
	assert.Nil(t, endpointAckedAt(t, db, 840061), "the downlink stays unacknowledged")
	assert.Empty(t, ackDeliveryRows(t, db))
}

// TestMarkEndpointAcknowledged_RefusesAnUnidentifiedUplink: the uplink that
// carried the acknowledgement names the delivery row, so a malformed id marks
// nothing.
func TestMarkEndpointAcknowledged_RefusesAnUnidentifiedUplink(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	seedWindowDownlink(t, db, 321, orgs[321], 840071, mioty.DLQueueStatusTransmitted, 41)

	_, _, err := downlinks.MarkEndpointAcknowledged(t.Context(), acknowledgement(321, 41, "not-a-message-id"))

	require.ErrorIs(t, err, storage.ErrInvalidInput)
	assert.Nil(t, endpointAckedAt(t, db, 840071))
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

	acknowledged, marked, err := downlinks.MarkEndpointAcknowledged(t.Context(), acknowledgement(321, 5, uuid.NewString()))

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
	_, err = first.Exec(sqlMarkEndpointAcknowledged, 321, mioty.EUI64Bytes(ackEndpointEUI), 9, mioty.DLQueueStatusTransmitted, time.Now(),
		uuid.New(), deliveryChannelArray(nil))
	require.NoError(t, err)

	second := make(chan bool)
	go func() {
		_, marked, err := downlinks.MarkEndpointAcknowledged(t.Context(), acknowledgement(321, 9, uuid.NewString()))
		assert.NoError(t, err)
		second <- marked
	}()
	waitForBlockedUpdate(t, db)
	require.NoError(t, first.Commit())

	assert.False(t, <-second, "the second reception finds the downlink already acknowledged")
}

// How long, and how often, waitForBlockedUpdate looks for a session waiting on a lock.
const (
	blockedUpdateWait = 10 * time.Second
	blockedUpdatePoll = 10 * time.Millisecond
)

// waitForBlockedUpdate waits until a session of the test database waits on a row lock.
func waitForBlockedUpdate(t *testing.T, db *sqlx.DB) {
	t.Helper()
	require.Eventually(t, func() bool {
		var waiting int
		require.NoError(t, db.QueryRow(`SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&waiting))
		return waiting > 0
	}, blockedUpdateWait, blockedUpdatePoll)
}

// TestGetAcknowledgedDownlink_ReadsTheAcknowledgedDownlinkOfItsTenant: the
// delivery of an acknowledgement reads the downlink back with its window and
// its ref; a downlink not acknowledged, or another tenant's, is not found.
func TestGetAcknowledgedDownlink_ReadsTheAcknowledgedDownlinkOfItsTenant(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	seedWindowDownlink(t, db, 321, orgs[321], 840081, mioty.DLQueueStatusTransmitted, 41)
	seedWindowDownlink(t, db, 321, orgs[321], 840082, mioty.DLQueueStatusTransmitted, 42)
	_, err := db.Exec(`UPDATE downlink_queue SET ref = $1 WHERE que_id = 840081`, fixtureCommandRef)
	require.NoError(t, err)
	acknowledged, marked, err := downlinks.MarkEndpointAcknowledged(t.Context(), acknowledgement(321, 41, uuid.NewString()))
	require.NoError(t, err)
	require.True(t, marked)
	var unacknowledgedID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM downlink_queue WHERE que_id = 840082`).Scan(&unacknowledgedID))

	read, err := downlinks.GetAcknowledgedDownlink(t.Context(), 321, acknowledged.ID)

	require.NoError(t, err)
	assert.Equal(t, int64(840081), read.QueID)
	assert.Equal(t, int64(41), read.TransmissionPacketCnt, "the window the acknowledgement names")
	assert.Equal(t, fixtureCommandRef, read.Ref)
	assert.Equal(t, orgs[321], *read.OrganizationID)
	assert.Equal(t, mioty.FormatEUIBytes(mioty.EUI64Bytes(ackEndpointEUI)), read.EPEUI)
	_, err = downlinks.GetAcknowledgedDownlink(t.Context(), 322, acknowledged.ID)
	assert.ErrorIs(t, err, storage.ErrNotFound, "another tenant never reads it")
	_, err = downlinks.GetAcknowledgedDownlink(t.Context(), 321, unacknowledgedID)
	assert.ErrorIs(t, err, storage.ErrNotFound, "a downlink its endpoint has not acknowledged is not read")
}

// TestGetDownlinkResults_ReportsTheEndpointAcknowledgement: the results
// listing carries the acknowledgement time.
func TestGetDownlinkResults_ReportsTheEndpointAcknowledgement(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	seedWindowDownlink(t, db, 321, orgs[321], 840011, mioty.DLQueueStatusTransmitted, 7)
	seedWindowDownlink(t, db, 321, orgs[321], 840012, mioty.DLQueueStatusTransmitted, 8)
	_, _, err := downlinks.MarkEndpointAcknowledged(t.Context(), acknowledgement(321, 7, uuid.NewString()))
	require.NoError(t, err)

	results, _, err := downlinks.GetDownlinkResults(t.Context(), 321, nil, storage.DownlinkResultFilter{}, 10, 0)

	require.NoError(t, err)
	acked := map[int64]bool{}
	for _, dl := range results {
		acked[dl.QueID] = dl.EndpointAckedAt != nil
	}
	assert.Equal(t, map[int64]bool{840011: true, 840012: false}, acked)
}
