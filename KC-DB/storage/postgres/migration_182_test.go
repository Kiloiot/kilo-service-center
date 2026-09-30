package postgres

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// storedRowTriggers maps each trigger of migration 000182 to its table.
var storedRowTriggers = map[string]string{
	"messages_notify_uplink_stored":     "messages",
	"system_events_notify_event_stored": "system_events",
}

const sqlTriggerEvents = `SELECT event_manipulation, action_timing, action_orientation
	FROM information_schema.triggers WHERE trigger_name = $1 AND event_object_table = $2`

// validateStoredRowNotifications: each announcing trigger fires after a
// stored row and on no other change.
func validateStoredRowNotifications(t *testing.T, db *sql.DB) {
	for trigger, table := range storedRowTriggers {
		rows, err := db.Query(sqlTriggerEvents, trigger, table)
		require.NoError(t, err)
		var events []string
		for rows.Next() {
			var manipulation, timing, orientation string
			require.NoError(t, rows.Scan(&manipulation, &timing, &orientation))
			events = append(events, manipulation+" "+timing+" "+orientation)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
		assert.Equal(t, []string{"INSERT AFTER ROW"}, events, "%s fires after each row inserted into %s, and only then", trigger, table)
	}
}

func (h *migrationHarness) triggerCount() int64 {
	return h.queryInt(`SELECT count(*) FROM pg_trigger WHERE tgname IN ('messages_notify_uplink_stored', 'system_events_notify_event_stored')`)
}

// assertQuiet fails when a channel wakes within the quiet period.
func assertQuiet(t *testing.T, w wakes, why string) {
	t.Helper()
	time.Sleep(listenerQuietFor)
	assert.False(t, w.took(ChannelUplinkStored), why)
	assert.False(t, w.took(ChannelEventStored), why)
}

// TestMigration182AnnouncesStoredRowsOnly: after 000182 an inserted uplink or
// event wakes its channel, while an update, a delete or a stored propagate
// message wakes nothing; the down migration stops the announcements and a
// second up restores them.
func TestMigration182AnnouncesStoredRowsOnly(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(181)
	assert.Zero(t, h.triggerCount(), "no row is announced before 000182")
	tenantID, _ := h.seedTenantAndBaseStation(listenerStation)

	h.migrateTo(182)
	validateStoredRowNotifications(t, h.db)
	w := newWakes()
	runListener(t, h.dsn, w)
	awaitListening(t, h.db, tenantID, w)

	uplink := storeListenerUplink(t, h.db, tenantID)
	event := storeListenerEvent(t, h.db, tenantID)
	require.Eventually(t, func() bool { return w.took(ChannelUplinkStored) }, listenerWakeWithin, listenerTick)
	require.Eventually(t, func() bool { return w.took(ChannelEventStored) }, listenerWakeWithin, listenerTick)

	h.exec(`UPDATE messages SET duplicate = true WHERE id = $1`, uplink)
	h.exec(`UPDATE system_events SET status = 'acknowledged' WHERE id = $1::uuid`, event)
	assertQuiet(t, w, "an update announces nothing")
	h.exec(`DELETE FROM messages WHERE id = $1`, uplink)
	h.exec(`DELETE FROM system_events WHERE id = $1::uuid`, event)
	assertQuiet(t, w, "a delete announces nothing")
	storeListenerMessage(t, h.db, tenantID, mioty.CmdAttachPropagate)
	assertQuiet(t, w, "a propagate message is not an uplink")

	h.migrateTo(181)
	assert.Zero(t, h.triggerCount())
	storeListenerUplink(t, h.db, tenantID)
	storeListenerEvent(t, h.db, tenantID)
	assertQuiet(t, w, "the down migration stops the announcements")

	h.migrateTo(182)
	storeListenerEvent(t, h.db, tenantID)
	require.Eventually(t, func() bool { return w.took(ChannelEventStored) }, listenerWakeWithin, listenerTick, "a second up announces again")
}
