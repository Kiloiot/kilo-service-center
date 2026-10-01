package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Every endpoint starts with no recorded station profile change, and the
// downgrade drops the column.
func TestMigration173RecordsTheProfileChangeTime(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(172)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd000a173")
	attached := h.seedEndpoint(tenantID, "70b3d5677011a173", "attached", nil)

	h.migrateTo(173)

	assert.Zero(t, h.queryInt(`SELECT count(*) FROM endpoints WHERE id = $1 AND profile_changed_at IS NOT NULL`, attached),
		"no known profile change")
	assert.Contains(t, h.columnComment("endpoints", "profile_changed_at"), "attach propagate parameters")

	h.migrateTo(172)

	assert.False(t, h.columnExists("endpoints", "profile_changed_at"), "the downgrade drops the column")
	h.migrateTo(173)
}
