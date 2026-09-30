package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// deadDownlinkQueueColumns are the downlink_queue columns no repository reads
// or writes; the dispatcher orders by priority, never by prio.
var deadDownlinkQueueColumns = []string{"prio", "valid_until", "packet_cnt_array"}

func TestMigration162DropsDeadDownlinkQueueColumns(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(161)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd0000162")
	h.seedDownlinkRow(tenantID, 162001, "pending")

	h.migrateTo(162)

	for _, column := range deadDownlinkQueueColumns {
		assert.False(t, h.columnExists("downlink_queue", column), "downlink_queue.%s must be dropped", column)
	}
	assert.Empty(t, h.indexDefinitionIfAny("idx_downlink_queue_mioty_prio"), "the index on the unread prio column goes with it")
	assert.True(t, h.columnExists("downlink_queue", "failure_reason"), "failure_reason records a base station's error answer")
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM downlink_queue`), "queue rows survive the column drop")

	h.migrateTo(161)
	for _, column := range deadDownlinkQueueColumns {
		assert.True(t, h.columnExists("downlink_queue", column), "downlink_queue.%s is restored", column)
	}
	assert.Contains(t, h.indexDefinitionIfAny("idx_downlink_queue_mioty_prio"), "(status, prio DESC, created_at)")
	assert.Equal(t, int64(0), h.queryInt(`SELECT count(*) FROM downlink_queue WHERE prio <> 0.0`), "prio is restored with its default")
	h.migrateTo(162)
}

func TestMigration162GuardRejectsPopulatedColumns(t *testing.T) {
	cases := map[string]struct {
		populate string
		fragment string
	}{
		"prio":             {`UPDATE downlink_queue SET prio = 2.5`, "downlink_queue.prio carries a non-default value"},
		"valid_until":      {`UPDATE downlink_queue SET valid_until = NOW()`, "downlink_queue.valid_until is populated"},
		"packet_cnt_array": {`UPDATE downlink_queue SET packet_cnt_array = ARRAY[1]::bigint[]`, "downlink_queue.packet_cnt_array is populated"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			h := newMigrationHarness(t)
			h.migrateTo(161)
			tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd0000163")
			h.seedDownlinkRow(tenantID, 162002, "pending")
			h.exec(tc.populate)

			h.migrateExpectingGuard(162, tc.fragment)
			assert.True(t, h.columnExists("downlink_queue", name), "a guarded migration drops nothing")
		})
	}
}

// indexDefinitionIfAny returns an index's definition, empty when it does not exist.
func (h *migrationHarness) indexDefinitionIfAny(name string) string {
	h.t.Helper()
	var def string
	for _, row := range h.queryStrings(`SELECT indexdef FROM pg_indexes WHERE schemaname = 'public' AND indexname = $1`, name) {
		def = row
	}
	return def
}

func (h *migrationHarness) queryStrings(query string, args ...interface{}) []string {
	h.t.Helper()
	rows, err := h.db.Query(query, args...)
	if err != nil {
		h.t.Fatalf("%s: %v", query, err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			h.t.Fatalf("%s: %v", query, err)
		}
		out = append(out, value)
	}
	if err := rows.Err(); err != nil {
		h.t.Fatalf("%s: %v", query, err)
	}
	return out
}
