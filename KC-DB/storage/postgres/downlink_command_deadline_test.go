package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// commandDeadlineAhead puts a command's deadline well inside the lifetime.
const commandDeadlineAhead = 10 * time.Minute

// commandDownlink is the fixture downlink an MQTT command queued under ref
// with the deadline, nil for none.
func commandDownlink(queID int64, orgs map[int64]uuid.UUID, ref string, expiresAt *time.Time) *storage.DownlinkMessage {
	downlink := applicationDownlink(321, orgs[321], queID, nil)
	downlink.Ref = ref
	downlink.ExpiresAt = expiresAt
	return downlink
}

// latestAtOf reads when the downlink's lifetime ends.
func latestAtOf(t *testing.T, db *sqlx.DB, queID int64) time.Time {
	t.Helper()
	var latestAt time.Time
	require.NoError(t, db.Get(&latestAt, `SELECT latest_at FROM downlink_queue WHERE que_id = $1`, queID))
	return latestAt
}

// TestEnqueueDownlink_TheEarlierOfDeadlineAndLifetimeEndsTheWait: a command's
// deadline before the end of the lifetime ends the downlink's wait; a later
// one leaves the lifetime in charge.
func TestEnqueueDownlink_TheEarlierOfDeadlineAndLifetimeEndsTheWait(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	soon := time.Now().Add(commandDeadlineAhead).UTC().Truncate(time.Microsecond)
	late := time.Now().Add(2 * testDownlinkLifetime)

	_, err := downlinks.EnqueueDownlink(t.Context(), commandDownlink(840001, orgs, "order-1", &soon), testDownlinkLifetime)
	require.NoError(t, err)
	_, err = downlinks.EnqueueDownlink(t.Context(), commandDownlink(840002, orgs, "order-2", &late), testDownlinkLifetime)
	require.NoError(t, err)

	assert.True(t, soon.Equal(latestAtOf(t, db, 840001)), "the deadline ends the wait")
	assert.WithinDuration(t, time.Now().Add(testDownlinkLifetime), latestAtOf(t, db, 840002), lifetimeTolerance,
		"the lifetime ends the wait before a later deadline")
}

// TestEnqueueDownlink_ARefQueuesOneDownlink: a repeated ref queues nothing,
// whether the first downlink is in flight or finished and whether or not the
// repeat's deadline already passed; another endpoint or organization, and a
// downlink without a ref, are not affected.
func TestEnqueueDownlink_ARefQueuesOneDownlink(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Second)
	_, err := downlinks.EnqueueDownlink(ctx, commandDownlink(840011, orgs, "order-17", &future), testDownlinkLifetime)
	require.NoError(t, err)

	_, err = downlinks.EnqueueDownlink(ctx, commandDownlink(840012, orgs, "order-17", &future), testDownlinkLifetime)
	assert.ErrorIs(t, err, storage.ErrDownlinkRefTaken, "a repeat while the first is in flight")
	_, err = downlinks.EnqueueDownlink(ctx, commandDownlink(840013, orgs, "order-17", &past), testDownlinkLifetime)
	assert.ErrorIs(t, err, storage.ErrDownlinkRefTaken, "a repeat after its deadline is still a repeat, never refused as expired")

	_, err = db.Exec(`UPDATE downlink_queue SET status = $1, result = $2 WHERE que_id = 840011`, mioty.DLQueueStatusExpired, mioty.ResultExpired)
	require.NoError(t, err)
	_, err = downlinks.EnqueueDownlink(ctx, commandDownlink(840014, orgs, "order-17", nil), testDownlinkLifetime)
	assert.ErrorIs(t, err, storage.ErrDownlinkRefTaken, "a finished downlink still holds its ref")

	otherEndpoint := commandDownlink(840015, orgs, "order-17", &future)
	otherEndpoint.EPEUI = "70B3D59CD0000322"
	_, err = downlinks.EnqueueDownlink(ctx, otherEndpoint, testDownlinkLifetime)
	require.NoError(t, err, "another endpoint")
	otherOrg := applicationDownlink(322, orgs[322], 840016, nil)
	otherOrg.Ref = "order-17"
	_, err = downlinks.EnqueueDownlink(ctx, otherOrg, testDownlinkLifetime)
	require.NoError(t, err, "another tenant's organization")
	for _, queID := range []int64{840017, 840018} {
		_, err = downlinks.EnqueueDownlink(ctx, commandDownlink(queID, orgs, "", nil), testDownlinkLifetime)
		require.NoError(t, err, "downlinks without a ref")
	}

	var queued int
	require.NoError(t, db.Get(&queued, `SELECT count(*) FROM downlink_queue WHERE ref = 'order-17' AND tenant_id = 321 AND ep_eui = $1`,
		mioty.EUI64Bytes(0x70B3D59CD0000321)))
	assert.Equal(t, 1, queued, "one downlink per ref of the endpoint")
}

// TestEnqueueDownlink_ADeadlineThatPassedQueuesNothing: a new command whose
// deadline is not in the future is refused, with or without a ref, and
// leaves no row.
func TestEnqueueDownlink_ADeadlineThatPassedQueuesNothing(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	past := time.Now().Add(-time.Second)
	for queID, ref := range map[int64]string{840021: "order-21", 840022: ""} {
		_, err := downlinks.EnqueueDownlink(t.Context(), commandDownlink(queID, orgs, ref, &past), testDownlinkLifetime)
		assert.ErrorIs(t, err, storage.ErrDownlinkDeadlineElapsed, "ref %q", ref)
	}
	var rows int
	require.NoError(t, db.Get(&rows, `SELECT count(*) FROM downlink_queue WHERE que_id IN (840021, 840022)`))
	assert.Zero(t, rows)
}
