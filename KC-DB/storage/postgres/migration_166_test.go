package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const beyondSignedApplicationQueueID = "9223372036854775813" // 2^63 + 5

func TestMigration166StoresApplicationQueueIDsBeyondSigned64(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(165)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd000166a")
	h.seedDownlinkRow(tenantID, 16601, "pending")
	h.exec(`UPDATE downlink_queue SET ac_que_id = 42 WHERE que_id = 16601`)

	h.migrateTo(166)

	assert.Equal(t, int64(42), h.queryInt(`SELECT ac_que_id FROM downlink_queue WHERE que_id = 16601`),
		"an existing Application Center queue id survives the type change")
	h.seedDownlinkRow(tenantID, 16602, "pending")
	h.exec(`UPDATE downlink_queue SET ac_que_id = $1 WHERE que_id = 16602`, beyondSignedApplicationQueueID)
	var stored string
	require.NoError(t, h.db.QueryRow(`SELECT ac_que_id::text FROM downlink_queue WHERE que_id = 16602`).Scan(&stored))
	assert.Equal(t, beyondSignedApplicationQueueID, stored)

	for _, refused := range []string{"0", "18446744073709551616"} {
		_, err := h.db.Exec(`UPDATE downlink_queue SET ac_que_id = $1 WHERE que_id = 16601`, refused)
		assert.ErrorContains(t, err, "downlink_queue_ac_que_id_unsigned_64", "ac_que_id %s is outside the unsigned 64-bit range", refused)
	}
	_, err := h.db.Exec(`UPDATE downlink_queue SET ac_que_id = $1 WHERE que_id = 16601`, beyondSignedApplicationQueueID)
	assert.ErrorContains(t, err, "idx_downlink_queue_tenant_ac_que_id", "the id stays unique within its tenant")

	err = h.m.Migrate(165)
	require.ErrorContains(t, err, "downlink_queue.ac_que_id exceeds BIGINT on 1 row(s)")
	require.NoError(t, h.m.Force(166))

	h.exec(`UPDATE downlink_queue SET ac_que_id = 43 WHERE que_id = 16602`)
	h.migrateTo(165)
	assert.Equal(t, int64(43), h.queryInt(`SELECT ac_que_id FROM downlink_queue WHERE que_id = 16602`))
	_, err = h.db.Exec(`UPDATE downlink_queue SET ac_que_id = -1 WHERE que_id = 16602`)
	assert.ErrorContains(t, err, "downlink_queue_ac_que_id_positive", "the downgrade restores the positive check")
	h.migrateTo(166)
}
