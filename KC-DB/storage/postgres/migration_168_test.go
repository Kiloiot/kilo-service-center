package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	statusChangeDetachTime = int64(1_800_000_000_000_000_000)
	statusChangeDetachUnix = int64(1_800_000_000)
)

func (h *migrationHarness) seedEndpoint(tenantID int64, eui, status string, lastDetachTime *int64) int64 {
	h.t.Helper()
	return h.queryInt(`INSERT INTO endpoints (ep_eui, name, tenant_id, owner_tenant_id, ep_status, last_detach_time)
		VALUES (decode($1, 'hex'), $1, $2, $2, $3, $4) RETURNING id`, eui, tenantID, status, lastDetachTime)
}

// A detached endpoint that predates the column takes its last detach time as
// its status change; every other endpoint starts unknown, and the downgrade
// drops the column.
func TestMigration168RecordsTheStatusChangeTime(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(167)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd000a168")
	detachTime := statusChangeDetachTime
	detached := h.seedEndpoint(tenantID, "70b3d5677011a168", "detached", &detachTime)
	neverAttached := h.seedEndpoint(tenantID, "70b3d5677011a169", "detached", nil)
	attached := h.seedEndpoint(tenantID, "70b3d5677011a16a", "attached", &detachTime)

	h.migrateTo(168)

	assert.Equal(t, statusChangeDetachUnix, h.queryInt(`SELECT extract(epoch FROM status_changed_at)::bigint FROM endpoints WHERE id = $1`, detached),
		"a detached endpoint's last detach time is its closest known decision time")
	for _, id := range []int64{neverAttached, attached} {
		assert.Zero(t, h.queryInt(`SELECT count(*) FROM endpoints WHERE id = $1 AND status_changed_at IS NOT NULL`, id), "no known status change")
	}

	h.migrateTo(167)
	assert.False(t, h.columnExists("endpoints", "status_changed_at"), "the downgrade drops the column")
	h.migrateTo(168)
}
