package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// A command queued 600ns into a microsecond with a deadline 900ns into it:
// both are stored as the same microsecond.
const (
	enqueueOffsetInStoredMoment  = 600 * time.Nanosecond
	deadlineOffsetInStoredMoment = 900 * time.Nanosecond
)

// firstReceptionWait is how long a repeat must stay blocked behind a first
// reception that has not committed yet.
const firstReceptionWait = 300 * time.Millisecond

// uncommittedFirstReception starts the first reception of the command under
// ref in an open transaction, as EnqueueDownlink does, and leaves it open.
func uncommittedFirstReception(t *testing.T, db *sqlx.DB, downlink *storage.DownlinkMessage) *sqlx.Tx {
	t.Helper()
	tx, err := db.BeginTxx(t.Context(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	command := storage.DownlinkCommandRef{
		TenantID: 321, OrganizationID: *downlink.OrganizationID, EpEUI: 0x70B3D59CD0000321, Ref: downlink.Ref,
	}
	_, err = tx.ExecContext(t.Context(), sqlLockCommandRef, command.TenantID, command.OrganizationID, mioty.EUI64Bytes(command.EpEUI), command.Ref)
	require.NoError(t, err)
	userData, err := encodeDownlinkUserData(downlink.CntDepend, downlink.UserData)
	require.NoError(t, err)
	enqueuedAt := time.Now().Truncate(storedTimePrecision)
	require.NoError(t, insertQueuedDownlink(t.Context(), tx, downlink, mioty.EUI64Bytes(command.EpEUI), 321, userData, enqueuedAt, testDownlinkLifetime))
	return tx
}

// enqueueInBackground runs EnqueueDownlink and delivers its error.
func enqueueInBackground(ctx context.Context, downlinks *MIOTYDownlinkRepository, downlink *storage.DownlinkMessage) <-chan error {
	done := make(chan error, 1)
	go func() {
		_, err := downlinks.EnqueueDownlink(ctx, downlink, testDownlinkLifetime)
		done <- err
	}()
	return done
}

// TestEnqueueDownlink_ARepeatPastItsDeadlineWaitsForTheFirstReception: a
// repeat whose deadline passed while the first reception of the command is
// still being queued waits for that reception and then reports the repeat,
// never a false expiry of a command that was accepted.
func TestEnqueueDownlink_ARepeatPastItsDeadlineWaitsForTheFirstReception(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Second)
	first := uncommittedFirstReception(t, db, commandDownlink(860001, orgs, "order-86", &future))

	done := enqueueInBackground(t.Context(), downlinks, commandDownlink(860002, orgs, "order-86", &past))
	select {
	case err := <-done:
		t.Fatalf("the repeat did not wait for the first reception: %v", err)
	case <-time.After(firstReceptionWait):
	}
	require.NoError(t, first.Commit())

	assert.ErrorIs(t, <-done, storage.ErrDownlinkRefTaken)
	assert.Equal(t, 1, refRows(t, db, "order-86"))
}

// TestEnqueueDownlink_ARepeatBeforeItsDeadlineWaitsForTheFirstReception: a
// repeat inside its deadline that races the first reception reports the
// repeat once that reception commits, and queues nothing.
func TestEnqueueDownlink_ARepeatBeforeItsDeadlineWaitsForTheFirstReception(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	future := time.Now().Add(time.Hour)
	first := uncommittedFirstReception(t, db, commandDownlink(860011, orgs, "order-87", &future))

	done := enqueueInBackground(t.Context(), downlinks, commandDownlink(860012, orgs, "order-87", &future))
	select {
	case err := <-done:
		t.Fatalf("the repeat did not wait for the first reception: %v", err)
	case <-time.After(firstReceptionWait):
	}
	require.NoError(t, first.Commit())

	assert.ErrorIs(t, <-done, storage.ErrDownlinkRefTaken)
	assert.Equal(t, 1, refRows(t, db, "order-87"))
}

// TestEnqueueDownlink_ConcurrentReceptionsQueueOneDownlink: receptions of one
// command racing each other queue one downlink; every other one reports the
// repeat and none fails otherwise.
func TestEnqueueDownlink_ConcurrentReceptionsQueueOneDownlink(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	future := time.Now().Add(time.Hour)
	const receptions = 8
	results := make([]<-chan error, receptions)
	for i := range receptions {
		results[i] = enqueueInBackground(t.Context(), downlinks, commandDownlink(860021+int64(i), orgs, "order-88", &future))
	}

	queued := 0
	for _, done := range results {
		err := <-done
		if err == nil {
			queued++
			continue
		}
		assert.ErrorIs(t, err, storage.ErrDownlinkRefTaken)
	}
	assert.Equal(t, 1, queued)
	assert.Equal(t, 1, refRows(t, db, "order-88"))
}

// TestEnqueueDownlink_ADeadlineInsideTheStoredPrecisionIsExpired: a deadline
// less than a microsecond after the moment of queueing is the same stored
// moment, so the command is refused as expired rather than failing the
// lifetime constraint.
func TestEnqueueDownlink_ADeadlineInsideTheStoredPrecisionIsExpired(t *testing.T) {
	_, db, orgs := applicationQueueIDFixture(t)
	microsecond := time.Now().Truncate(time.Microsecond)
	enqueuedAt := microsecond.Add(enqueueOffsetInStoredMoment)
	deadline := microsecond.Add(deadlineOffsetInStoredMoment)
	writer := &DownlinkQueueWriter{db: db, clock: fixedClock{now: enqueuedAt}}

	_, err := writer.EnqueueDownlink(t.Context(), commandDownlink(860031, orgs, "order-89", &deadline), testDownlinkLifetime)

	assert.ErrorIs(t, err, storage.ErrDownlinkDeadlineElapsed)
	assert.Zero(t, refRows(t, db, "order-89"))
}

// refRows counts the downlinks queued under ref.
func refRows(t *testing.T, db *sqlx.DB, ref string) int {
	t.Helper()
	var rows int
	require.NoError(t, db.Get(&rows, `SELECT count(*) FROM downlink_queue WHERE ref = $1`, ref))
	return rows
}
