package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigration164RecordsEndpointAcknowledgements(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(163)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd0000165")
	h.seedDownlinkRow(tenantID, 163001, "transmitted")

	h.migrateTo(164)

	assert.True(t, h.columnExists("downlink_queue", "endpoint_acked_at"))
	assert.Contains(t, h.indexDefinitionIfAny("idx_downlink_queue_ack_window"), "(tenant_id, ep_eui, transmission_packet_cnt)")
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM downlink_queue WHERE endpoint_acked_at IS NULL`),
		"an existing downlink carries no acknowledgement")

	h.exec(`UPDATE downlink_queue SET endpoint_acked_at = NOW()`)
	h.migrateTo(163)
	assert.False(t, h.columnExists("downlink_queue", "endpoint_acked_at"))
	assert.Empty(t, h.indexDefinitionIfAny("idx_downlink_queue_ack_window"))
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM downlink_queue`), "the downgrade keeps the downlinks")
	h.migrateTo(164)
}
