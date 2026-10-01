package postgres

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const movedEventTrigger = "system_events_notify_event_moved"

// validateMovedEventNotifications: the trigger of migration 000183 fires
// after each system_events row whose occurred_at an UPDATE moves forward, and
// on no other change.
func validateMovedEventNotifications(t *testing.T, db *sql.DB) {
	var manipulation, timing, orientation, condition string
	require.NoError(t, db.QueryRow(`SELECT event_manipulation, action_timing, action_orientation, action_condition
		FROM information_schema.triggers WHERE trigger_name = $1 AND event_object_table = 'system_events'`,
		movedEventTrigger).Scan(&manipulation, &timing, &orientation, &condition))
	assert.Equal(t, "UPDATE AFTER ROW", manipulation+" "+timing+" "+orientation)
	assert.Equal(t, "(new.occurred_at > old.occurred_at)", condition)

	rows, err := db.Query(`SELECT event_object_column FROM information_schema.triggered_update_columns
		WHERE trigger_name = $1`, movedEventTrigger)
	require.NoError(t, err)
	var columns []string
	for rows.Next() {
		var column string
		require.NoError(t, rows.Scan(&column))
		columns = append(columns, column)
	}
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
	assert.Equal(t, []string{"occurred_at"}, columns, "only an update naming occurred_at is considered")
}

func (h *migrationHarness) movedEventTriggerCount() int64 {
	return h.queryInt(`SELECT count(*) FROM pg_trigger WHERE tgname = '` + movedEventTrigger + `'`)
}

// TestMigration183AnnouncesAnEventMovedForward: after 000183 an event whose
// occurred_at an UPDATE moves forward wakes the event channel, while an
// update that leaves it or moves it back, and an update of an uplink, wake
// nothing; the down migration stops the announcement and a second up
// restores it.
func TestMigration183AnnouncesAnEventMovedForward(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(182)
	tenantID, _ := h.seedTenantAndBaseStation(listenerStation)
	w := newWakes()
	runListener(t, h.dsn, w)
	awaitListening(t, h.db, tenantID, w)
	event := storeListenerEvent(t, h.db, tenantID)
	uplink := storeListenerUplink(t, h.db, tenantID)
	awaitStoredWakes(t, w)
	moveForward := func() {
		h.exec(`UPDATE system_events SET occurred_at = occurred_at + interval '1 second' WHERE id = $1::uuid`, event)
	}

	moveForward()
	assertQuiet(t, w, "before 000183 an event moved forward is not announced")

	h.migrateTo(183)
	assert.Equal(t, int64(1), h.movedEventTriggerCount())
	validateMovedEventNotifications(t, h.db)
	moveForward()
	require.Eventually(t, func() bool { return w.took(ChannelEventStored) }, listenerWakeWithin, listenerTick,
		"an event moved forward wakes the event channel")

	h.exec(`UPDATE system_events SET status = 'acknowledged' WHERE id = $1::uuid`, event)
	assertQuiet(t, w, "an update leaving occurred_at out announces nothing")
	h.exec(`UPDATE system_events SET occurred_at = occurred_at WHERE id = $1::uuid`, event)
	assertQuiet(t, w, "an event kept at its time announces nothing")
	h.exec(`UPDATE system_events SET occurred_at = occurred_at - interval '1 minute' WHERE id = $1::uuid`, event)
	assertQuiet(t, w, "an event moved back announces nothing")
	h.exec(`UPDATE messages SET duplicate = true WHERE id = $1`, uplink)
	assertQuiet(t, w, "an updated uplink announces nothing")

	h.migrateTo(182)
	assert.Zero(t, h.movedEventTriggerCount())
	moveForward()
	assertQuiet(t, w, "the down migration stops the announcement")

	h.migrateTo(183)
	moveForward()
	require.Eventually(t, func() bool { return w.took(ChannelEventStored) }, listenerWakeWithin, listenerTick,
		"a second up announces again")
}

// awaitStoredWakes waits out the wakes of the rows a test just stored.
func awaitStoredWakes(t *testing.T, w wakes) {
	t.Helper()
	require.Eventually(t, func() bool { return w.took(ChannelEventStored) }, listenerWakeWithin, listenerTick)
	require.Eventually(t, func() bool { return w.took(ChannelUplinkStored) }, listenerWakeWithin, listenerTick)
	assertQuiet(t, w, "the stored rows woke their channels once")
}
