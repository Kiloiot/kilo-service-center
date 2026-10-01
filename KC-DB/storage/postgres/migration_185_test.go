package postgres

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	storedUplinkIndex = "idx_messages_tenant_stored"
	storedEventIndex  = "idx_system_events_tenant_stored"
	storedEventStamp  = "system_events_restamp_moved"
	storedStation     = "70b3d59cd000a185"
)

// validateStreamStorageOrder: migration 000185 stamps system_events by the
// database clock on insert and on a move forward, and indexes both tables by
// storage order.
func validateStreamStorageOrder(t *testing.T, db *sql.DB) {
	var nullable, def string
	require.NoError(t, db.QueryRow(`SELECT is_nullable, column_default FROM information_schema.columns
		WHERE table_name = 'system_events' AND column_name = 'stored_at'`).Scan(&nullable, &def))
	assert.Equal(t, "NO", nullable)
	assert.Equal(t, "clock_timestamp()", def)

	var manipulation, timing, orientation, condition string
	require.NoError(t, db.QueryRow(`SELECT event_manipulation, action_timing, action_orientation, action_condition
		FROM information_schema.triggers WHERE trigger_name = $1 AND event_object_table = 'system_events'`,
		storedEventStamp).Scan(&manipulation, &timing, &orientation, &condition))
	assert.Equal(t, "UPDATE BEFORE ROW", manipulation+" "+timing+" "+orientation)
	assert.Equal(t, "(new.occurred_at > old.occurred_at)", condition)

	var indexes int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM pg_indexes WHERE indexname IN ($1, $2)`,
		storedUplinkIndex, storedEventIndex).Scan(&indexes))
	assert.Equal(t, 2, indexes)
}

func (h *migrationHarness) restampTriggerCount() int64 {
	return h.queryInt(`SELECT count(*) FROM pg_trigger WHERE tgname = $1`, storedEventStamp)
}

// storedLater reports whether the event's stored_at moved past the given one.
func (h *migrationHarness) storedLater(id, before string) bool {
	return h.queryInt(`SELECT count(*) FROM system_events WHERE id = $1::uuid AND stored_at > $2::timestamptz`, id, before) == 1
}

func (h *migrationHarness) storedAt(id string) string {
	h.t.Helper()
	var at string
	require.NoError(h.t, h.db.QueryRow(`SELECT stored_at::text FROM system_events WHERE id = $1::uuid`, id).Scan(&at))
	return at
}

// TestMigration185StampsTheStorageOrder: after 000185 the stored events keep
// -infinity, a new event is stamped by the database clock, an UPDATE moving
// occurred_at forward stamps it again while any other update leaves the
// stamp, and both storage-order indexes exist; the down migration removes
// them and a second up restores them.
func TestMigration185StampsTheStorageOrder(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(184)
	tenantID, _ := h.seedTenantAndBaseStation(storedStation)
	before := storeListenerEvent(t, h.db, tenantID)

	h.migrateTo(185)
	assert.Equal(t, "-infinity", h.storedAt(before), "a stored event stays behind every stream")
	assert.Contains(t, h.indexDefinition(storedUplinkIndex), "(tenant_id, created_at DESC, id DESC) WHERE ((command_type)::text = 'ulData'::text)")
	assert.Contains(t, h.indexDefinition(storedEventIndex), "(tenant_id, stored_at DESC, id DESC)")
	assert.Equal(t, int64(1), h.restampTriggerCount())

	event := storeListenerEvent(t, h.db, tenantID)
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM system_events WHERE id = $1::uuid AND stored_at > now() - interval '1 minute'`, event),
		"a new event is stamped by the database clock")
	stamp := h.storedAt(event)
	h.exec(`UPDATE system_events SET status = 'acknowledged' WHERE id = $1::uuid`, event)
	h.exec(`UPDATE system_events SET occurred_at = occurred_at - interval '1 minute' WHERE id = $1::uuid`, event)
	assert.Equal(t, stamp, h.storedAt(event), "an update that does not move the event forward keeps its stamp")
	h.exec(`UPDATE system_events SET occurred_at = occurred_at + interval '1 hour' WHERE id = $1::uuid`, event)
	assert.True(t, h.storedLater(event, stamp), "an event moved forward is stamped again")

	h.migrateTo(184)
	assert.False(t, h.columnExists("system_events", "stored_at"))
	assert.Empty(t, h.indexDefinitionIfAny(storedUplinkIndex))
	assert.Empty(t, h.indexDefinitionIfAny(storedEventIndex))
	assert.Zero(t, h.restampTriggerCount())
	assert.False(t, h.functionExists("restamp_moved_event"))

	h.migrateTo(185)
	require.True(t, h.columnExists("system_events", "stored_at"))
	assert.Equal(t, int64(1), h.restampTriggerCount(), "a second up restores the stamp")
}
