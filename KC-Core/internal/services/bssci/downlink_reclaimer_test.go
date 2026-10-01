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

// discardedRevocations hands out the overdue downlinks a station discarded.
type discardedRevocations struct {
	expired  []*storage.DownlinkMessage
	err      error
	stations []uint64
}

func (d *discardedRevocations) ExpireStationRevocations(_ context.Context, bsEUI uint64) ([]*storage.DownlinkMessage, error) {
	d.stations = append(d.stations, bsEUI)
	return d.expired, d.err
}

// stationExpiries records every downlink reported expired at its station.
type stationExpiries struct{ reported []int64 }

func (s *stationExpiries) ReportExpiredAtStation(_ context.Context, downlink *storage.DownlinkMessage) {
	s.reported = append(s.reported, downlink.QueID)
}

// forgottenQueues records every queue id whose cached owner is forgotten.
type forgottenQueues struct{ forgotten []int64 }

func (f *forgottenQueues) UnregisterQueueTenant(queueID int64) {
	f.forgotten = append(f.forgotten, queueID)
}

// reclaimerDeps are a reclaimer's collaborators around the store and the recorder.
func reclaimerDeps(store ReservationReclaimer, events RequeueRecorder, log logger.Logger) DownlinkReclaimerDeps {
	return DownlinkReclaimerDeps{
		Store: store, Revocations: &discardedRevocations{}, Events: events, Expiries: &stationExpiries{},
		Tenants: &forgottenQueues{}, Logger: log,
	}
}

func mustReclaimer(t *testing.T, store ReservationReclaimer, events RequeueRecorder) bssci.DownlinkReclaimer {
	t.Helper()
	reclaimer, err := NewDownlinkReclaimer(reclaimerDeps(store, events, &mockLoggerForDispatch{}))
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
	cases := map[string]struct {
		unset func(*DownlinkReclaimerDeps)
		want  error
	}{
		"nil store":       {func(d *DownlinkReclaimerDeps) { d.Store = nil }, ErrNilReservationReclaimer},
		"nil revocations": {func(d *DownlinkReclaimerDeps) { d.Revocations = nil }, ErrNilDiscardedRevocations},
		"nil recorder":    {func(d *DownlinkReclaimerDeps) { d.Events = nil }, ErrNilRequeueRecorder},
		"nil reporter":    {func(d *DownlinkReclaimerDeps) { d.Expiries = nil }, ErrNilDiscardedExpiryReporter},
		"nil tenants":     {func(d *DownlinkReclaimerDeps) { d.Tenants = nil }, ErrNilReclaimerQueueTenants},
		"nil logger":      {func(d *DownlinkReclaimerDeps) { d.Logger = nil }, ErrNilReclaimerLogger},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			deps := reclaimerDeps(&mockMIOTYDownlinksForDispatch{}, &requeueRecorder{}, logger.NewNop())
			tc.unset(&deps)
			reclaimer, err := NewDownlinkReclaimer(deps)
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
	reclaimer, err := NewDownlinkReclaimer(reclaimerDeps(store, events, log))
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

// A session that is not resumed discarded the overdue downlinks its station
// was asked to drop (BSSCI §1): they end expired and each is reported, and
// they are never counted as returned to the queue, and the owner cached for
// each is forgotten; a failed expiry surfaces after the queued downlinks were
// returned.
func TestReclaimDiscardedQueue_ExpiresTheStationsRevocations(t *testing.T) {
	store := &mockMIOTYDownlinksForDispatch{released: reclaimTestReleased}
	revocations := &discardedRevocations{expired: []*storage.DownlinkMessage{{QueID: 900011}, {QueID: 900012}}}
	expiries := &stationExpiries{}
	tenants := &forgottenQueues{}
	reclaimer, err := NewDownlinkReclaimer(DownlinkReclaimerDeps{
		Store: store, Revocations: revocations, Events: &requeueRecorder{}, Expiries: expiries, Tenants: tenants, Logger: logger.NewNop(),
	})
	require.NoError(t, err)

	released, err := reclaimer.ReclaimDiscardedQueue(testutil.TestContext(), reclaimTestStation)

	require.NoError(t, err)
	assert.Equal(t, int64(len(reclaimTestReleased)), released, "only queued downlinks return to the queue")
	assert.Equal(t, []uint64{reclaimTestStation}, revocations.stations)
	assert.Equal(t, []int64{900011, 900012}, expiries.reported)
	assert.Equal(t, []int64{900011, 900012}, tenants.forgotten, "an ended downlink's cached owner is forgotten")

	revocations.err, expiries.reported = errTestReserveDBDown, nil
	_, err = reclaimer.ReclaimDiscardedQueue(testutil.TestContext(), reclaimTestStation)
	assert.ErrorIs(t, err, errExpireDiscardedRevocations)
	assert.Empty(t, expiries.reported)

	_, err = reclaimer.ReclaimReservations(testutil.TestContext(), reclaimTestStation, nil)
	require.NoError(t, err)
	assert.Len(t, revocations.stations, 2, "a resumed session's reservations leave the revocations alone")
}

// A deleted station can never transmit what it held: its reservations and
// queued downlinks return to the queue for another station, and the ones it
// was asked to drop end expired and are reported; a failure is logged and the
// rest is still settled.
func TestReleaseDeletedStation_SettlesEverythingTheStationHeld(t *testing.T) {
	store := &mockMIOTYDownlinksForDispatch{released: reclaimTestReleased}
	revocations := &discardedRevocations{expired: []*storage.DownlinkMessage{{QueID: 900021}}}
	expiries := &stationExpiries{}
	events := &requeueRecorder{}
	reclaimer, err := NewDownlinkReclaimer(DownlinkReclaimerDeps{
		Store: store, Revocations: revocations, Events: events, Expiries: expiries, Tenants: &forgottenQueues{}, Logger: logger.NewNop(),
	})
	require.NoError(t, err)

	reclaimer.ReleaseDeletedStation(testutil.TestContext(), reclaimTestStation)

	assert.Equal(t, []uint64{reclaimTestStation}, store.releasedStations, "its reservations are released")
	assert.Equal(t, []uint64{reclaimTestStation}, store.releasedQueues, "its queue is released")
	assert.Equal(t, []uint64{reclaimTestStation}, revocations.stations)
	assert.Equal(t, []int64{900021}, expiries.reported)

	revocations.err, expiries.reported = errTestReserveDBDown, nil
	reclaimer.ReleaseDeletedStation(testutil.TestContext(), reclaimTestStation)
	assert.Empty(t, expiries.reported)
}
