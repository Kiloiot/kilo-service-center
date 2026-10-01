package postgres

import (
	"database/sql"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	// listenerWakeWithin bounds how long a stored row may take to wake its channel.
	listenerWakeWithin = 2 * time.Second
	// listenerQuietFor is how long a channel must stay quiet after a change that announces nothing.
	listenerQuietFor = 300 * time.Millisecond
	listenerTick     = 10 * time.Millisecond
	listenerStation  = "70b3d59cd00182a1"
)

// wakes records, per channel, that a notification ran its handler since the last take.
type wakes map[string]chan struct{}

func newWakes() wakes {
	return wakes{ChannelUplinkStored: make(chan struct{}, 1), ChannelEventStored: make(chan struct{}, 1)}
}

func (w wakes) handlers() map[string]func() {
	handlers := make(map[string]func(), len(w))
	for channel, woke := range w {
		handlers[channel] = func() {
			select {
			case woke <- struct{}{}:
			default:
			}
		}
	}
	return handlers
}

// took reports whether the channel woke since the last take, and clears it.
func (w wakes) took(channel string) bool {
	select {
	case <-w[channel]:
		return true
	default:
		return false
	}
}

func (w wakes) drain() {
	for channel := range w {
		w.took(channel)
	}
}

// runListener relays the database's notifications into w until the test ends.
func runListener(t *testing.T, dsn string, w wakes) {
	t.Helper()
	ctx, cancel := testutil.TestContextWithCancel()
	done := make(chan error, 1)
	go func() { done <- NewNotificationListener(dsn, w.handlers(), logger.NewNop()).Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		assert.NoError(t, <-done, "the listener stops cleanly with its context")
	})
}

// awaitListening stores rows until each channel wakes, which proves the
// listener listens on it; afterwards no wake is pending.
func awaitListening(t *testing.T, db *sql.DB, tenantID int64, w wakes) {
	t.Helper()
	require.Eventually(t, func() bool {
		storeListenerEvent(t, db, tenantID)
		return w.took(ChannelEventStored)
	}, listenerWakeWithin, listenerTick, "the listener listens on the event channel")
	require.Eventually(t, func() bool {
		storeListenerUplink(t, db, tenantID)
		return w.took(ChannelUplinkStored)
	}, listenerWakeWithin, listenerTick, "the listener listens on the uplink channel")
	time.Sleep(listenerQuietFor)
	w.drain()
}

func storeListenerEvent(t *testing.T, db *sql.DB, tenantID int64) string {
	t.Helper()
	var id string
	require.NoError(t, db.QueryRow(`INSERT INTO system_events (tenant_id, event_type, event_category, severity,
		source_type, source_name, title, description, data, status, occurred_at, recorded_at)
		VALUES ($1, 'basestation.updated', 'basestation', 'info', 'service_center', 'listener', 'event', 'event', '{}'::jsonb, 'new', NOW(), NOW())
		RETURNING id::text`, tenantID).Scan(&id))
	return id
}

func storeListenerMessage(t *testing.T, db *sql.DB, tenantID int64, commandType string) string {
	t.Helper()
	var id string
	require.NoError(t, db.QueryRow(
		`INSERT INTO messages (tenant_id, command_type, op_id, ep_eui, bs_eui, rx_time, packet_cnt, snr, rssi)
		 VALUES ($1, $2, 1, decode('70b3d5677011a182', 'hex'), decode($3, 'hex'), $4, 1, 1, -80)
		 RETURNING id`, tenantID, commandType, listenerStation, time.Now().UnixNano()).Scan(&id))
	return id
}

func storeListenerUplink(t *testing.T, db *sql.DB, tenantID int64) string {
	t.Helper()
	return storeListenerMessage(t, db, tenantID, mioty.CmdULData)
}

func seedListenerTenant(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var tenantID int64
	require.NoError(t, db.QueryRow(`INSERT INTO tenants (name) VALUES ($1) RETURNING id`, "listener-"+t.Name()).Scan(&tenantID))
	return tenantID
}

func migratedListenerDatabase(t *testing.T) (*sqlx.DB, string) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping listener integration test in short mode")
	}
	dsn, drop := testsupport.NewMigratedDatabase(t)
	t.Cleanup(drop)
	db, err := sqlx.Connect("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, db.Close()) })
	return db, dsn
}

// A stored uplink wakes the uplink channel and a stored event the event
// channel, each within the bound and neither the other's.
func TestNotificationListener_AStoredRowWakesItsChannel(t *testing.T) {
	db, dsn := migratedListenerDatabase(t)
	tenantID := seedListenerTenant(t, db.DB)
	w := newWakes()
	runListener(t, dsn, w)
	awaitListening(t, db.DB, tenantID, w)

	storeListenerUplink(t, db.DB, tenantID)
	require.Eventually(t, func() bool { return w.took(ChannelUplinkStored) }, listenerWakeWithin, listenerTick)
	assert.False(t, w.took(ChannelEventStored), "an uplink does not wake the event streams")

	storeListenerEvent(t, db.DB, tenantID)
	require.Eventually(t, func() bool { return w.took(ChannelEventStored) }, listenerWakeWithin, listenerTick)
	assert.False(t, w.took(ChannelUplinkStored), "an event does not wake the uplink streams")
}

// After its connection is terminated the listener reconnects, wakes every
// channel once for the notifications it may have missed, and relays again.
func TestNotificationListener_AReconnectWakesEveryChannel(t *testing.T) {
	db, dsn := migratedListenerDatabase(t)
	tenantID := seedListenerTenant(t, db.DB)
	w := newWakes()
	runListener(t, dsn, w)
	awaitListening(t, db.DB, tenantID, w)

	var terminated int
	require.NoError(t, db.QueryRow(`SELECT count(pg_terminate_backend(pid)) FROM pg_stat_activity
		WHERE datname = current_database() AND pid <> pg_backend_pid() AND query LIKE 'LISTEN%'`).Scan(&terminated))
	require.Equal(t, 1, terminated, "the listener's connection is terminated")

	require.Eventually(t, func() bool { return w.took(ChannelUplinkStored) }, listenerReconnectCap, listenerTick, "the reconnect wakes the uplink channel")
	require.Eventually(t, func() bool { return w.took(ChannelEventStored) }, listenerWakeWithin, listenerTick, "the reconnect wakes the event channel")

	storeListenerEvent(t, db.DB, tenantID)
	require.Eventually(t, func() bool { return w.took(ChannelEventStored) }, listenerWakeWithin, listenerTick, "the reconnected listener relays again")
}
