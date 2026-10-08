package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	largestPacketCounter = "4294967295" // 2^32 - 1
	beyondPacketCounter  = "4294967296"
	firstUnsignedCounter = "2147483648" // 2^31
)

func (h *migrationHarness) seedMessage(tenantID int64, table, packetCnt string) string {
	h.t.Helper()
	var id string
	require.NoError(h.t, h.db.QueryRow(
		`INSERT INTO `+table+` (tenant_id, op_id, ep_eui, bs_eui, rx_time, packet_cnt, snr, rssi)
		 VALUES ($1, 1, decode('70b3d5677011a167', 'hex'), decode('70b3d59cd000a167', 'hex'), 1, $2, 1, -80)
		 RETURNING id`, tenantID, packetCnt).Scan(&id))
	return id
}

func TestMigration167StoresTheFullUnsigned32BitPacketCounter(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(166)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd000a167")
	kept := h.seedMessage(tenantID, "messages", "7")

	h.migrateTo(167)

	assert.Equal(t, int64(7), h.queryInt(`SELECT packet_cnt FROM messages WHERE id = $1`, kept), "an existing counter survives the type change")
	for _, table := range []string{"messages", "messages_archive"} {
		id := h.seedMessage(tenantID, table, largestPacketCounter)
		assert.Equal(t, int64(4294967295), h.queryInt(`SELECT packet_cnt FROM `+table+` WHERE id = $1`, id), table)
		_, err := h.db.Exec(`UPDATE `+table+` SET packet_cnt = $1 WHERE id = $2`, beyondPacketCounter, id)
		assert.ErrorContains(t, err, "valid_packet_cnt", "%s refuses a counter beyond 32 bits", table)
	}
	err := h.m.Migrate(166)
	require.ErrorContains(t, err, "packet counter exceeds INTEGER")
	require.NoError(t, h.m.Force(167))

	h.exec(`DELETE FROM messages WHERE packet_cnt > 2147483647`)
	h.exec(`DELETE FROM messages_archive WHERE packet_cnt > 2147483647`)
	h.migrateTo(166)
	_, err = h.db.Exec(`UPDATE messages SET packet_cnt = $1 WHERE id = $2`, firstUnsignedCounter, kept)
	assert.ErrorContains(t, err, "out of range", "the downgrade restores INTEGER")
	_, err = h.db.Exec(`UPDATE messages SET packet_cnt = -1 WHERE id = $1`, kept)
	assert.ErrorContains(t, err, "valid_packet_cnt", "the downgrade restores the non-negative check")
	h.migrateTo(167)
}
