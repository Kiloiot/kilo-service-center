package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// clockLead puts the injected clock far enough from the database's own
// clock that a statement reading NOW() is told apart from one bound to it.
const clockLead = 48 * time.Hour

// timestampPrecision is the resolution PostgreSQL stores timestamps at.
const timestampPrecision = time.Microsecond

// TestDownlinkQueue_TakesItsTimeFromTheClock: the queue's timestamps and its
// expiry decisions follow the injected clock, not the database server's.
func TestDownlinkQueue_TakesItsTimeFromTheClock(t *testing.T) {
	_, sqlxDB, orgs := downlinkRepositoriesFixture(t)
	now := time.Now().Add(clockLead).UTC().Truncate(timestampPrecision)
	downlinks := NewRepositories(&DB{clock: testutil.NewFakeClock(now), conn: sqlxDB.DB, sqlxDB: sqlxDB, log: logger.Get()}).Downlinks

	message := applicationDownlink(321, orgs[321], 830101, nil)
	stored, err := downlinks.EnqueueDownlink(t.Context(), message, configuredLifetime)
	require.NoError(t, err)
	var createdAt, latestAt time.Time
	require.NoError(t, sqlxDB.QueryRow(`SELECT created_at, latest_at FROM downlink_queue WHERE id = $1`, stored.ID).Scan(&createdAt, &latestAt))
	assert.True(t, now.Equal(createdAt), "created_at %s is the clock's %s", createdAt, now)
	assert.True(t, now.Add(configuredLifetime).Equal(latestAt), "latest_at %s is the clock's %s plus the lifetime", latestAt, now)

	seedScheduledDownlink(t, sqlxDB, 321, orgs[321], 830102, mioty.DLQueueStatusPending, time.Hour, nil)
	_, err = downlinks.ReservePendingDownlinkByQueueID(t.Context(), 321, orgs[321], 830102, mioty.EUI64Bytes(830102), releaseStation)
	assert.ErrorIs(t, err, storage.ErrNotFound, "a row overdue by the clock is not reserved")
	expired, err := downlinks.ExpireOverdueUnheld(t.Context(), 10)
	require.NoError(t, err)
	require.Len(t, expired, 1, "a row overdue by the clock is expired")
	assert.Equal(t, int64(830102), expired[0].QueID)
}
