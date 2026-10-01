package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// TestExpireRemovedStationDownlinks_EndsEverythingTheStationHeld: a deleted
// station's reserved, queued and revoking downlinks of every tenant end
// expired, none returns to pending; its pending and finished rows and another
// station's held rows keep their state.
func TestExpireRemovedStationDownlinks_EndsEverythingTheStationHeld(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	seedHeldDownlink(t, db, 321, orgs[321], 860001, mioty.DLQueueStatusReserved, releaseStation)
	seedHeldDownlink(t, db, 322, orgs[322], 860002, mioty.DLQueueStatusQueued, releaseStation)
	seedHeldDownlink(t, db, 321, orgs[321], 860003, mioty.DLQueueStatusRevoking, releaseStation)
	seedHeldDownlink(t, db, 321, orgs[321], 860004, mioty.DLQueueStatusPending, releaseStation)
	seedHeldDownlink(t, db, 321, orgs[321], 860005, mioty.DLQueueStatusTransmitted, releaseStation)
	seedHeldDownlink(t, db, 321, orgs[321], 860006, mioty.DLQueueStatusQueued, neighborStation)

	expired, err := downlinks.ExpireRemovedStationDownlinks(ctx, releaseStation)

	require.NoError(t, err)
	queIDs := make([]int64, 0, len(expired))
	for _, downlink := range expired {
		queIDs = append(queIDs, downlink.QueID)
		assert.Equal(t, mioty.ResultExpired, downlink.Result)
		assert.Equal(t, releaseStation, downlink.BsEui, "reported as expired at the deleted station")
	}
	assert.ElementsMatch(t, []int64{860001, 860002, 860003}, queIDs, "a roaming tenant's downlink ends too")
	for queID, want := range map[int64]mioty.DLQueueStatus{
		860001: mioty.DLQueueStatusExpired, 860002: mioty.DLQueueStatusExpired, 860003: mioty.DLQueueStatusExpired,
		860004: mioty.DLQueueStatusPending, 860005: mioty.DLQueueStatusTransmitted, 860006: mioty.DLQueueStatusQueued,
	} {
		assert.Equal(t, string(want), heldDownlink(t, db, queID).Status, "queue id %d", queID)
	}
	again, err := downlinks.ExpireRemovedStationDownlinks(ctx, releaseStation)
	require.NoError(t, err)
	assert.Empty(t, again, "each is expired once")
}

// TestExpireHeldAtRemovedStations_EndsWhatADeletionLeftHeld: the sweep ends
// expired the held downlinks of a station with no base station row, in
// batches, and leaves those of a registered station alone.
func TestExpireHeldAtRemovedStations_EndsWhatADeletionLeftHeld(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	ctx := t.Context()
	createTestBaseStation(t, db, 860100, neighborStation, 321, "TestBS-Registered")
	seedHeldDownlink(t, db, 321, orgs[321], 860101, mioty.DLQueueStatusReserved, releaseStation)
	seedHeldDownlink(t, db, 322, orgs[322], 860102, mioty.DLQueueStatusRevoking, releaseStation)
	seedHeldDownlink(t, db, 321, orgs[321], 860103, mioty.DLQueueStatusQueued, neighborStation)
	seedHeldDownlink(t, db, 321, orgs[321], 860104, mioty.DLQueueStatusPending, releaseStation)

	first, err := downlinks.ExpireHeldAtRemovedStations(ctx, 1)
	require.NoError(t, err)
	require.Len(t, first, 1)
	rest, err := downlinks.ExpireHeldAtRemovedStations(ctx, 10)
	require.NoError(t, err)
	require.Len(t, rest, 1)

	assert.ElementsMatch(t, []int64{860101, 860102}, []int64{first[0].QueID, rest[0].QueID})
	assert.Equal(t, releaseStation, first[0].BsEui, "each is returned with the station that held it")
	assert.Equal(t, mioty.ResultExpired, rest[0].Result)
	assert.Equal(t, string(mioty.DLQueueStatusQueued), heldDownlink(t, db, 860103).Status, "a registered station keeps what it holds")
	assert.Equal(t, string(mioty.DLQueueStatusPending), heldDownlink(t, db, 860104).Status, "a pending row is held by no station")
	_, err = downlinks.ExpireHeldAtRemovedStations(ctx, 0)
	assert.Error(t, err, "a batch must be positive")
}
