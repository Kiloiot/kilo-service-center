package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insertEventWithDefaultStatus stores an event the way the insert did before
// 175: without a status, so the column default decides it.
func (h *migrationHarness) insertEventWithDefaultStatus(tenantID int64, title string) {
	h.exec(`INSERT INTO system_events (tenant_id, event_type, event_category, severity, source_type, source_name, title)
		VALUES ($1, 'connection_error', 'basestation', 'warning', 'basestation', 'bs', $2)`, tenantID, title)
}

// TestMigration175MakesUnhandledEventsNew: events stored with the retired
// 'active' default become 'new', later ones default to 'new', the downgrade
// hands them back as 'active', and upgrading again repeats the upgrade.
func TestMigration175MakesUnhandledEventsNew(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(172)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd0000175")
	h.insertEventWithDefaultStatus(tenantID, "before")
	h.exec(`INSERT INTO system_events (tenant_id, event_type, event_category, severity, source_type, source_name, title, status)
		VALUES ($1, 'connection_error', 'basestation', 'warning', 'basestation', 'bs', 'handled', 'acknowledged')`, tenantID)

	h.migrateTo(175)

	h.insertEventWithDefaultStatus(tenantID, "after")
	assert.Equal(t, int64(2), h.queryInt(`SELECT count(*) FROM system_events WHERE status = 'new'`))
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM system_events WHERE status = 'acknowledged'`))
	_, err := h.db.Exec(`UPDATE system_events SET status = 'active' WHERE title = 'after'`)
	require.ErrorContains(t, err, "system_events_status_check", "the retired status cannot come back")

	_, err = h.db.Exec(`UPDATE system_events SET status = NULL WHERE title = 'after'`)
	require.Error(t, err, "every event has a status")

	require.NoError(t, h.m.Migrate(172))
	assert.Equal(t, int64(2), h.queryInt(`SELECT count(*) FROM system_events WHERE status = 'active'`))
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM pg_constraint
		WHERE conrelid = 'system_events'::regclass AND conname = 'system_events_status_check' AND convalidated`))

	h.migrateTo(175)
	assert.Equal(t, int64(2), h.queryInt(`SELECT count(*) FROM system_events WHERE status = 'new'`))
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM pg_constraint
		WHERE conrelid = 'system_events'::regclass AND contype = 'c' AND conname LIKE 'system_events_status%'`),
		"only the validated status check remains")
}
