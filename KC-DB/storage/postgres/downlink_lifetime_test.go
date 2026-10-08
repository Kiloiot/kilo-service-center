package postgres

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// testDownlinkLifetime is the lifetime the repository tests enqueue downlinks with.
const testDownlinkLifetime = 24 * time.Hour

// lifetimeTolerance absorbs the time between the enqueue and the check.
const lifetimeTolerance = time.Minute

// configuredLifetime is a lifetime other than the default day.
const configuredLifetime = 90 * time.Minute

// seedScheduledDownlink stores a downlink whose lifetime ends at latestAt
// relative to now; a zero offset leaves it without a lifetime.
func seedScheduledDownlink(t *testing.T, db *sqlx.DB, tenantID int64, orgID uuid.UUID, queID int64,
	status mioty.DLQueueStatus, latestIn time.Duration, acQueID *uint64,
) {
	t.Helper()
	if latestIn == 0 {
		_, err := db.Exec(`
			INSERT INTO downlink_queue (que_id, ac_que_id, ep_eui, tenant_id, organization_id, payload, status, priority, earliest_at, latest_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 0, NULL, NULL)`,
			queID, applicationQueueIDParam(acQueID), mioty.EUI64Bytes(uint64(queID)), tenantID, orgID, []byte{0x01}, status)
		require.NoError(t, err)
		return
	}
	_, err := db.Exec(`
		INSERT INTO downlink_queue (que_id, ac_que_id, ep_eui, tenant_id, organization_id, payload, status, priority, earliest_at, latest_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 0, NOW() - INTERVAL '1 day', NOW() + make_interval(secs => $8))`,
		queID, applicationQueueIDParam(acQueID), mioty.EUI64Bytes(uint64(queID)), tenantID, orgID, []byte{0x01}, status, latestIn.Seconds())
	require.NoError(t, err)
}

func downlinkStatusOf(t *testing.T, db *sqlx.DB, queID int64) (status string, result *string) {
	t.Helper()
	require.NoError(t, db.QueryRow(`SELECT status, result FROM downlink_queue WHERE que_id = $1`, queID).Scan(&status, &result))
	return status, result
}

// TestEnqueueDownlink_LatestAtFollowsTheLifetime: the configured lifetime,
// not a fixed day, bounds how long a downlink waits for a window.
func TestEnqueueDownlink_LatestAtFollowsTheLifetime(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	message := applicationDownlink(321, orgs[321], 830001, nil)

	stored, err := downlinks.EnqueueDownlink(t.Context(), message, configuredLifetime)
	require.NoError(t, err)

	var window time.Duration
	var seconds float64
	require.NoError(t, db.QueryRow(`SELECT EXTRACT(EPOCH FROM latest_at - created_at) FROM downlink_queue WHERE id = $1`, stored.ID).Scan(&seconds))
	window = time.Duration(seconds * float64(time.Second))
	assert.InDelta(t, float64(configuredLifetime), float64(window), float64(lifetimeTolerance))
}

// TestReservation_SkipsExpiredDownlinks: a pending downlink past its lifetime
// is never reserved, whether the dlOpen dispatch picks the next row or the
// immediate dispatch names it.
func TestReservation_SkipsExpiredDownlinks(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	org := orgs[321]
	seedScheduledDownlink(t, db, 321, org, 830011, mioty.DLQueueStatusPending, -time.Hour, nil)
	_, err := db.Exec(`UPDATE downlink_queue SET ep_eui = $1, priority = 9 WHERE que_id = 830011`, mioty.EUI64Bytes(830012))
	require.NoError(t, err)
	seedScheduledDownlink(t, db, 321, org, 830012, mioty.DLQueueStatusPending, time.Hour, nil)

	next, err := downlinks.ReserveNextPendingDownlink(t.Context(), 321, mioty.EUI64Bytes(830012), releaseStation)
	require.NoError(t, err)
	assert.Equal(t, int64(830012), next.QueID, "the higher-priority but expired row is skipped")

	_, err = downlinks.ReservePendingDownlinkByQueueID(t.Context(), 321, org, 830011, mioty.EUI64Bytes(830012), releaseStation)
	assert.ErrorIs(t, err, storage.ErrNotFound, "an expired row cannot be reserved by its queue id")
	status, _ := downlinkStatusOf(t, db, 830011)
	assert.Equal(t, string(mioty.DLQueueStatusPending), status)
}

// TestExpireOverdueUnheld_ExpiresOnlyOverduePendingRows: the sweep expires
// every tenant's pending downlink past its lifetime and nothing in time or
// finished, and returns what the originators must be told.
func TestExpireOverdueUnheld_ExpiresOnlyOverduePendingRows(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	acQueID := uint64(77)
	seedScheduledDownlink(t, db, 321, orgs[321], 830021, mioty.DLQueueStatusPending, -time.Hour, &acQueID)
	seedScheduledDownlink(t, db, 322, orgs[322], 830022, mioty.DLQueueStatusPending, -time.Minute, nil)
	seedScheduledDownlink(t, db, 321, orgs[321], 830023, mioty.DLQueueStatusPending, time.Hour, nil)
	seedScheduledDownlink(t, db, 321, orgs[321], 830024, mioty.DLQueueStatusTransmitted, -time.Hour, nil)
	seedScheduledDownlink(t, db, 321, orgs[321], 830025, mioty.DLQueueStatusPending, 0, nil)

	expired, err := downlinks.ExpireOverdueUnheld(t.Context(), 10)

	require.NoError(t, err)
	byQueue := map[int64]*storage.DownlinkMessage{}
	for _, dl := range expired {
		byQueue[dl.QueID] = dl
	}
	require.Len(t, byQueue, 2)
	require.Contains(t, byQueue, int64(830021))
	require.Contains(t, byQueue, int64(830022))
	assert.Equal(t, "321", byQueue[830021].TenantID)
	assert.Equal(t, orgs[321], *byQueue[830021].OrganizationID)
	assert.Equal(t, acQueID, *byQueue[830021].ACQueID)
	assert.Equal(t, "322", byQueue[830022].TenantID)
	assert.Nil(t, byQueue[830022].ACQueID)
	for _, queID := range []int64{830021, 830022} {
		status, result := downlinkStatusOf(t, db, queID)
		assert.Equal(t, string(mioty.DLQueueStatusExpired), status)
		require.NotNil(t, result)
		assert.Equal(t, mioty.ResultExpired, *result)
	}
	for queID, want := range map[int64]mioty.DLQueueStatus{
		830023: mioty.DLQueueStatusPending,
		830024: mioty.DLQueueStatusTransmitted,
		830025: mioty.DLQueueStatusPending,
	} {
		status, _ := downlinkStatusOf(t, db, queID)
		assert.Equal(t, string(want), status, "queue id %d", queID)
	}

	again, err := downlinks.ExpireOverdueUnheld(t.Context(), 10)
	require.NoError(t, err)
	assert.Empty(t, again, "an expired downlink is reported once")
}

func TestExpireOverdueUnheld_HonorsTheBatchLimit(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	for queID := int64(830031); queID <= 830033; queID++ {
		seedScheduledDownlink(t, db, 321, orgs[321], queID, mioty.DLQueueStatusPending, -time.Hour, nil)
	}

	expired, err := downlinks.ExpireOverdueUnheld(t.Context(), 2)

	require.NoError(t, err)
	assert.Len(t, expired, 2)
}

// TestOverdueSweep_RevokesQueuedRowsAndLeavesReservedOnes: a downlink a base
// station holds queued past its lifetime becomes revoking, keeps the station
// as its holder and names it so it is asked to drop it (BSSCI §3.13); its
// outcome is not decided yet. A reserved downlink, whose dlDataQue may not be
// on the wire, is left to its dispatch. A pending row expires naming no
// station, even with the station of a released reservation still on it.
func TestOverdueSweep_RevokesQueuedRowsAndLeavesReservedOnes(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	for queID, status := range map[int64]mioty.DLQueueStatus{
		830041: mioty.DLQueueStatusQueued, 830042: mioty.DLQueueStatusReserved, 830043: mioty.DLQueueStatusPending,
		830044: mioty.DLQueueStatusRevoking,
	} {
		seedScheduledDownlink(t, db, 321, orgs[321], queID, status, -time.Hour, nil)
		_, err := db.Exec(`UPDATE downlink_queue SET bs_eui = $1 WHERE que_id = $2`, mioty.EUI64Bytes(releaseStation), queID)
		require.NoError(t, err)
	}

	sweepStart := time.Now().Add(-time.Second)
	revoking, err := downlinks.RevokeOverdueHeld(t.Context(), sweepStart, 10)
	require.NoError(t, err)
	require.Len(t, revoking, 1)
	assert.Equal(t, int64(830041), revoking[0].QueID)
	assert.Equal(t, releaseStation, revoking[0].BsEui)
	assert.Equal(t, mioty.DLQueueStatusRevoking, revoking[0].Status)
	assert.Empty(t, revoking[0].Result, "no outcome while the station is asked")
	var askedAt, expiredAskedAt sql.NullTime
	require.NoError(t, db.Get(&askedAt, `SELECT revoke_asked_at FROM downlink_queue WHERE que_id = 830041`))
	assert.True(t, askedAt.Valid, "the ask is recorded so an unanswered one is repeated")
	assert.True(t, askedAt.Time.Equal(sweepStart.Truncate(storedTimePrecision)), "the ask is the sweep's start, so the next sweep finds it due")

	expired, err := downlinks.ExpireOverdueUnheld(t.Context(), 10)
	require.NoError(t, err)
	require.Len(t, expired, 1)
	assert.Equal(t, int64(830043), expired[0].QueID)
	assert.Zero(t, expired[0].BsEui, "a pending row is held by no station")
	require.NoError(t, db.Get(&expiredAskedAt, `SELECT revoke_asked_at FROM downlink_queue WHERE que_id = 830043`))
	assert.False(t, expiredAskedAt.Valid, "no station is asked about a row expired in the queue")

	for queID, want := range map[int64]mioty.DLQueueStatus{
		830041: mioty.DLQueueStatusRevoking, 830042: mioty.DLQueueStatusReserved,
		830043: mioty.DLQueueStatusExpired, 830044: mioty.DLQueueStatusRevoking,
	} {
		status, result := downlinkStatusOf(t, db, queID)
		assert.Equal(t, string(want), status, "queue id %d", queID)
		if want != mioty.DLQueueStatusExpired {
			assert.Nil(t, result, "queue id %d has no result", queID)
		}
	}
	again, err := downlinks.RevokeOverdueHeld(t.Context(), time.Now(), 10)
	require.NoError(t, err)
	assert.Empty(t, again, "a station is asked once per sweep transition")
}
