package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	acceptingStation = uint64(0x70B3D59CD00009E6)
	// nextHolder is the station a released downlink is reserved for next.
	nextHolder = uint64(0x70B3D59CD00009E7)
)

// The queue names the base station that accepted a downlink (dlDataQueRsp,
// BSSCI §3.12) and when; a downlink released back to pending is no longer
// accepted by anyone.
func TestDownlinkQueue_NamesTheStationThatAcceptedADownlinkUntilItIsReleased(t *testing.T) {
	repos, db, orgs := downlinkRepositoriesFixture(t)
	ctx := t.Context()
	const queID = int64(870001)
	_, err := repos.Downlinks.EnqueueDownlink(ctx, applicationDownlink(321, orgs[321], queID, nil), testDownlinkLifetime)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE downlink_queue SET status = 'queued', bs_eui = $1 WHERE que_id = $2`, mioty.EUI64Bytes(acceptingStation), queID)
	require.NoError(t, err)

	listed := listQueuedDownlink(t, repos)
	assert.Nil(t, listed.AcceptedAt, "sent but not yet accepted")

	require.NoError(t, repos.Downlinks.UpdateDownlinkBaseStation(ctx, uint64(queID), 321, acceptingStation))
	listed = listQueuedDownlink(t, repos)
	assert.Equal(t, acceptingStation, listed.BsEui)
	require.NotNil(t, listed.AcceptedAt, "the station's acceptance is recorded")

	_, err = repos.Downlinks.ReleaseStationQueue(ctx, acceptingStation)
	require.NoError(t, err)
	listed = listQueuedDownlink(t, repos)
	assert.Zero(t, listed.BsEui)
	assert.Nil(t, listed.AcceptedAt, "a released downlink awaits a new station")
}

// A dlDataQueRsp that arrives after the downlink was released, from the
// station that no longer holds it, records nothing: the row stays pending, or
// stays with the station now holding it.
func TestDownlinkQueue_ALateAcceptanceFromAStationThatNoLongerHoldsTheDownlinkIsRefused(t *testing.T) {
	repos, db, orgs := downlinkRepositoriesFixture(t)
	ctx := t.Context()
	const queID = int64(870002)
	_, err := repos.Downlinks.EnqueueDownlink(ctx, applicationDownlink(321, orgs[321], queID, nil), testDownlinkLifetime)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE downlink_queue SET status = 'reserved', bs_eui = $1 WHERE que_id = $2`, mioty.EUI64Bytes(acceptingStation), queID)
	require.NoError(t, err)
	_, err = repos.Downlinks.ReleaseStationReservations(ctx, acceptingStation, nil)
	require.NoError(t, err)

	err = repos.Downlinks.UpdateDownlinkBaseStation(ctx, uint64(queID), 321, acceptingStation)
	require.ErrorIs(t, err, storage.ErrDownlinkNotFound)
	released := listQueuedDownlink(t, repos)
	assert.Equal(t, "pending", string(released.Status))
	assert.Nil(t, released.AcceptedAt, "a released downlink is not accepted by the station that let it go")

	_, err = db.Exec(`UPDATE downlink_queue SET status = 'reserved', bs_eui = $1 WHERE que_id = $2`, mioty.EUI64Bytes(nextHolder), queID)
	require.NoError(t, err)
	err = repos.Downlinks.UpdateDownlinkBaseStation(ctx, uint64(queID), 321, acceptingStation)
	require.ErrorIs(t, err, storage.ErrDownlinkNotFound)
	err = repos.Downlinks.MarkReservedAsQueued(ctx, uint64(queID), 321, acceptingStation, 1, nil, nil)
	require.ErrorIs(t, err, ErrDownlinkAlreadyReserved)
	held := listQueuedDownlink(t, repos)
	assert.Equal(t, "reserved", string(held.Status))
	assert.Equal(t, nextHolder, held.BsEui, "the station now holding the downlink keeps it")
	assert.Nil(t, held.AcceptedAt)

	require.NoError(t, repos.Downlinks.UpdateDownlinkBaseStation(ctx, uint64(queID), 321, nextHolder))
	require.NotNil(t, listQueuedDownlink(t, repos).AcceptedAt, "the holder's own acceptance is recorded")
}

func listQueuedDownlink(t *testing.T, repos *Repositories) *storage.DownlinkMessage {
	t.Helper()
	queue, err := repos.DownlinkQueueReader.ListTenantQueue(t.Context(), 321, storage.DownlinkQueueFilter{}, 10, 0)
	require.NoError(t, err)
	require.Len(t, queue, 1)
	return queue[0]
}
