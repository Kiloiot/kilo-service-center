package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration169AcceptsApplicationQueueIDZero(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(167)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd000169a")
	h.seedDownlinkRow(tenantID, 16901, "pending")

	h.migrateTo(169)

	h.exec(`UPDATE downlink_queue SET ac_que_id = 0 WHERE que_id = 16901`)
	assert.Equal(t, int64(0), h.queryInt(`SELECT ac_que_id FROM downlink_queue WHERE que_id = 16901`))
	for _, refused := range []string{"-1", "18446744073709551616"} {
		_, err := h.db.Exec(`UPDATE downlink_queue SET ac_que_id = $1 WHERE que_id = 16901`, refused)
		assert.ErrorContains(t, err, "downlink_queue_ac_que_id_unsigned_64", "ac_que_id %s is outside the unsigned 64-bit range", refused)
	}

	err := h.m.Migrate(167)
	require.ErrorContains(t, err, "downlink_queue.ac_que_id is zero on 1 row(s)")
	require.NoError(t, h.m.Force(169))

	h.exec(`UPDATE downlink_queue SET ac_que_id = 7 WHERE que_id = 16901`)
	h.migrateTo(167)
	_, err = h.db.Exec(`UPDATE downlink_queue SET ac_que_id = 0 WHERE que_id = 16901`)
	assert.ErrorContains(t, err, "downlink_queue_ac_que_id_unsigned_64", "the downgrade refuses zero again")
	h.migrateTo(169)
}
