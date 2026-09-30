package bssciservices

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

const (
	reclaimTestStation = uint64(0x70B3D59CD00009E6)
	reclaimTestTenant  = int64(78)
)

// reclaimTestReleased are the downlinks the reclaim test store releases.
var reclaimTestReleased = []storage.PendingDownlink{
	{QueID: 900001, TenantID: reclaimTestTenant, OrganizationID: uuid.New(), EpEUI: 0x70B3D56770111505},
	{QueID: 900002, TenantID: reclaimTestTenant + 1, OrganizationID: uuid.New(), EpEUI: 0x70B3D56770111506},
}

var errTestRequeueEvent = errors.New("event store down")

// requeue is one announced return to the queue.
type requeue struct {
	downlink storage.PendingDownlink
	bsEUI    uint64
}

// requeueRecorder records the announced returns to the queue.
type requeueRecorder struct {
	requeued []requeue
	err      error
}

func (r *requeueRecorder) RecordRequeued(_ context.Context, downlink storage.PendingDownlink, bsEUI uint64) error {
	r.requeued = append(r.requeued, requeue{downlink: downlink, bsEUI: bsEUI})
	return r.err
}

func mustReclaimer(t *testing.T, store ReservationReclaimer, events RequeueRecorder) bssci.DownlinkReclaimer {
	t.Helper()
	reclaimer, err := NewDownlinkReclaimer(store, events, &mockLoggerForDispatch{})
	require.NoError(t, err)
	return reclaimer
}

// announcedAtStation is the requeue record of every released downlink at the station.
func announcedAtStation(released []storage.PendingDownlink) []requeue {
	announced := make([]requeue, len(released))
	for i, downlink := range released {
		announced[i] = requeue{downlink: downlink, bsEUI: reclaimTestStation}
	}
	return announced
}

func TestNewDownlinkReclaimer_RejectsMissingCollaborators(t *testing.T) {
	store, events, log := &mockMIOTYDownlinksForDispatch{}, &requeueRecorder{}, logger.NewNop()
	cases := map[string]struct {
		store  ReservationReclaimer
		events RequeueRecorder
		log    logger.Logger
		want   error
	}{
		"nil store":    {events: events, log: log, want: ErrNilReservationReclaimer},
		"nil recorder": {store: store, log: log, want: ErrNilRequeueRecorder},
		"nil logger":   {store: store, events: events, want: ErrNilReclaimerLogger},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			reclaimer, err := NewDownlinkReclaimer(tc.store, tc.events, tc.log)
			require.ErrorIs(t, err, tc.want)
			assert.Nil(t, reclaimer)
		})
	}
}

// Each reclaim hands its scope to the store, counts the downlinks the store
// returned to pending and announces each one's return at the station, so
// every open view of the downlink queue shows it pending again; a failed
// release surfaces and announces nothing.
func TestReclaim_AnnouncesEveryDownlinkReturnedToTheQueue(t *testing.T) {
	propagatedAt := time.Unix(1_800_000_000, 0)
	cases := map[string]struct {
		reclaim func(bssci.DownlinkReclaimer) (int64, error)
		scope   func(*testing.T, *mockMIOTYDownlinksForDispatch)
		wrapped error
	}{
		"reservations of a fresh session": {
			reclaim: func(r bssci.DownlinkReclaimer) (int64, error) {
				return r.ReclaimReservations(testutil.TestContext(), reclaimTestStation, []int64{777})
			},
			scope: func(t *testing.T, store *mockMIOTYDownlinksForDispatch) {
				assert.Equal(t, []uint64{reclaimTestStation}, store.releasedStations)
				assert.Equal(t, [][]int64{{777}}, store.releaseKeeps, "the reissued queue id is kept")
			},
			wrapped: errReclaimReservations,
		},
		"queue a fresh session discarded": {
			reclaim: func(r bssci.DownlinkReclaimer) (int64, error) {
				return r.ReclaimDiscardedQueue(testutil.TestContext(), reclaimTestStation)
			},
			scope: func(t *testing.T, store *mockMIOTYDownlinksForDispatch) {
				assert.Equal(t, []uint64{reclaimTestStation}, store.releasedQueues)
				assert.Empty(t, store.releasedStations, "reservations are reclaimed separately")
			},
			wrapped: errReclaimDiscardedQueue,
		},
		"endpoint queue an attach propagate discarded": {
			reclaim: func(r bssci.DownlinkReclaimer) (int64, error) {
				return r.ReclaimEndpointQueue(testutil.TestContext(), reclaimTestTenant, 0x70B3D56770111505, reclaimTestStation, propagatedAt)
			},
			scope: func(t *testing.T, store *mockMIOTYDownlinksForDispatch) {
				assert.Equal(t, []endpointRelease{{tenantID: reclaimTestTenant, epEUI: 0x70B3D56770111505, bsEUI: reclaimTestStation, sentBy: propagatedAt}},
					store.endpointReleases)
				assert.Empty(t, store.releasedStations, "the station's other reservations are not touched")
				assert.Empty(t, store.releasedQueues, "the station's other queued downlinks are not touched")
			},
			wrapped: errReclaimEndpointQueue,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			store, events := &mockMIOTYDownlinksForDispatch{released: reclaimTestReleased}, &requeueRecorder{}
			reclaimer := mustReclaimer(t, store, events)

			released, err := tc.reclaim(reclaimer)

			require.NoError(t, err)
			assert.Equal(t, int64(len(reclaimTestReleased)), released)
			tc.scope(t, store)
			assert.Equal(t, announcedAtStation(reclaimTestReleased), events.requeued)

			store.releaseErr, events.requeued = errTestReserveDBDown, nil
			_, err = tc.reclaim(reclaimer)
			assert.ErrorIs(t, err, errTestReserveDBDown)
			assert.ErrorIs(t, err, tc.wrapped)
			assert.Empty(t, events.requeued)
		})
	}
}

// A release stands when its announcement fails: every downlink is still
// counted and the failure is logged.
func TestReclaim_AFailedAnnouncementLeavesTheReleaseInPlace(t *testing.T) {
	store, events := &mockMIOTYDownlinksForDispatch{released: reclaimTestReleased}, &requeueRecorder{err: errTestRequeueEvent}
	log := bsscitest.NewRecordingLogger()
	reclaimer, err := NewDownlinkReclaimer(store, events, log)
	require.NoError(t, err)

	released, err := reclaimer.ReclaimDiscardedQueue(testutil.TestContext(), reclaimTestStation)

	require.NoError(t, err)
	assert.Equal(t, int64(len(reclaimTestReleased)), released)
	assert.Len(t, events.requeued, len(reclaimTestReleased), "every downlink is still attempted")
	failures := log.FilterMessage(LogDownlinkRequeueEventFailed)
	require.Len(t, failures, len(reclaimTestReleased))
	assert.Equal(t, "ERROR", failures[0].Level)
}

// Nothing released, nothing announced.
func TestReclaim_NothingReleasedAnnouncesNothing(t *testing.T) {
	events := &requeueRecorder{}
	reclaimer := mustReclaimer(t, &mockMIOTYDownlinksForDispatch{}, events)

	released, err := reclaimer.ReclaimReservations(testutil.TestContext(), reclaimTestStation, nil)

	require.NoError(t, err)
	assert.Zero(t, released)
	assert.Empty(t, events.requeued)
}
