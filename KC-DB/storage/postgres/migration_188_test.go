package postgres

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validateDownlinkAckDelivery: migration 000188 lets the delivery outbox carry
// an endpoint acknowledgement, one row per downlink, on its own channel.
func validateDownlinkAckDelivery(t *testing.T, db *sql.DB) {
	var channelExists bool
	require.NoError(t, db.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_enum e JOIN pg_type t ON t.oid = e.enumtypid
		WHERE t.typname = 'message_delivery_channel' AND e.enumlabel = 'mqtt_downlink_ack')`).Scan(&channelExists))
	assert.True(t, channelExists)
	var dataType, nullable string
	require.NoError(t, db.QueryRow(`SELECT data_type, is_nullable FROM information_schema.columns
		WHERE table_name = 'message_delivery_outbox' AND column_name = 'acknowledged_downlink_id'`).Scan(&dataType, &nullable))
	assert.Equal(t, "bigint", dataType)
	assert.Equal(t, "YES", nullable)
	var check, index string
	require.NoError(t, db.QueryRow(`SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conname = 'message_delivery_outbox_acknowledged_downlink'`).Scan(&check))
	assert.Equal(t, "CHECK ((((channel)::text = 'mqtt_downlink_ack'::text) = (acknowledged_downlink_id IS NOT NULL)))", check)
	require.NoError(t, db.QueryRow(`SELECT indexdef FROM pg_indexes
		WHERE indexname = 'uq_message_delivery_outbox_acknowledged_downlink'`).Scan(&index))
	assert.Contains(t, index, "UNIQUE INDEX")
	assert.Contains(t, index, "WHERE (acknowledged_downlink_id IS NOT NULL)")
}

// TestMigration188CarriesOneAcknowledgementPerDownlink: the outbox keeps its
// uplink rows, takes one acknowledgement row per downlink and only on the
// acknowledgement channel, drops it with its downlink, and the downgrade
// removes the acknowledgement rows and keeps the uplink rows.
func TestMigration188CarriesOneAcknowledgementPerDownlink(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(187)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd000188a")
	uplink := h.seedMessage(tenantID, "messages", "42")
	other := h.seedMessage(tenantID, "messages", "43")
	h.exec(`INSERT INTO message_delivery_outbox (message_id, channel, owner_tenant_id) VALUES ($1, 'mqtt', $2)`, uplink, tenantID)
	h.seedDownlinkRow(tenantID, 188001, "transmitted")
	downlinkID := h.queryInt(`SELECT id FROM downlink_queue WHERE que_id = 188001`)

	h.migrateTo(188)

	queueAck := func(messageID, channel string, downlink interface{}) error {
		_, err := h.db.Exec(`INSERT INTO message_delivery_outbox (message_id, channel, owner_tenant_id, acknowledged_downlink_id)
			VALUES ($1, $2, $3, $4)`, messageID, channel, tenantID, downlink)
		return err
	}
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM message_delivery_outbox WHERE acknowledged_downlink_id IS NULL`),
		"the uplink row stored before is kept")
	require.NoError(t, queueAck(uplink, "mqtt_downlink_ack", downlinkID))
	assert.Error(t, queueAck(other, "mqtt_downlink_ack", downlinkID), "one acknowledgement row per downlink")
	assert.Error(t, queueAck(other, "mqtt_downlink_ack", nil), "an acknowledgement row names its downlink")
	assert.Error(t, queueAck(other, "scaci", downlinkID), "only the acknowledgement channel names a downlink")

	h.exec(`DELETE FROM downlink_queue WHERE id = $1`, downlinkID)
	assert.Equal(t, int64(0), h.queryInt(`SELECT count(*) FROM message_delivery_outbox WHERE acknowledged_downlink_id IS NOT NULL`),
		"deleting the downlink deletes its acknowledgement row")

	h.seedDownlinkRow(tenantID, 188002, "transmitted")
	require.NoError(t, queueAck(uplink, "mqtt_downlink_ack", h.queryInt(`SELECT id FROM downlink_queue WHERE que_id = 188002`)))
	h.migrateTo(187)
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM message_delivery_outbox`), "the downgrade keeps only the uplink row")
	h.migrateTo(188)
}
