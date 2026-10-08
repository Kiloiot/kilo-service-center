package downlinkexpiry

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	workerTestBatch    = 2
	workerTestInterval = time.Millisecond
	workerTestSettle   = 50 * time.Millisecond
)

const workerTestStation uint64 = 0x70B3D59CD00009E6

// workerTestStart is when the swept sweep started.
var workerTestStart = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

var (
	errWorkerTestQueue   = errors.New("queue down")
	errWorkerTestUnknown = errors.New("unidentified downlink")
	errWorkerTestStation = errors.New("base station disconnected")
)

// fakeBatches hands out the configured batches, one per call.
type fakeBatches struct {
	batches [][]*storage.DownlinkMessage
	limits  []int
}

func (b *fakeBatches) next(limit int) []*storage.DownlinkMessage {
	b.limits = append(b.limits, limit)
	if len(b.batches) == 0 {
		return nil
	}
	batch := b.batches[0]
	b.batches = b.batches[1:]
	return batch
}

// unansweredClaim is one claim of the revocations connected stations left unanswered.
type unansweredClaim struct {
	connected     []uint64
	askedAt       time.Time
	unansweredFor time.Duration
}

// fakeQueue hands out the overdue downlinks no station holds, those a station
// holds and the unanswered revocations, each from its own batches.
type fakeQueue struct {
	mu         sync.Mutex
	unheld     fakeBatches
	held       fakeBatches
	unanswered fakeBatches
	claims     []unansweredClaim
	revokedAt  []time.Time
	err        error
	calls      int
	entered    chan struct{}
	release    chan struct{}
}

func (q *fakeQueue) ExpireOverdueUnheld(_ context.Context, limit int) ([]*storage.DownlinkMessage, error) {
	if q.entered != nil {
		q.entered <- struct{}{}
		<-q.release
	}
	return q.take(&q.unheld, limit)
}

func (q *fakeQueue) RevokeOverdueHeld(_ context.Context, askedAt time.Time, limit int) ([]*storage.DownlinkMessage, error) {
	q.mu.Lock()
	q.revokedAt = append(q.revokedAt, askedAt)
	q.mu.Unlock()
	return q.take(&q.held, limit)
}

func (q *fakeQueue) ClaimUnansweredRevocations(_ context.Context, connected []uint64, askedAt time.Time, unansweredFor time.Duration, limit int) ([]*storage.DownlinkMessage, error) {
	q.mu.Lock()
	q.claims = append(q.claims, unansweredClaim{connected: connected, askedAt: askedAt, unansweredFor: unansweredFor})
	q.mu.Unlock()
	return q.take(&q.unanswered, limit)
}

// askTimes lists the ask times the sweeps recorded on overdue held downlinks.
func (q *fakeQueue) askTimes() []time.Time {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]time.Time(nil), q.revokedAt...)
}

// fakeRemovedStations hands out the downlinks held by deleted stations.
type fakeRemovedStations struct {
	orphaned fakeBatches
	err      error
}

func (r *fakeRemovedStations) ExpireHeldAtRemovedStations(_ context.Context, limit int) ([]*storage.DownlinkMessage, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.orphaned.next(limit), nil
}

func (q *fakeQueue) take(from *fakeBatches, limit int) ([]*storage.DownlinkMessage, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.calls++
	if q.err != nil {
		return nil, q.err
	}
	return from.next(limit), nil
}

// fakeReporter records every expired downlink it is asked to report, in the
// queue or at the station that held it.
type fakeReporter struct {
	mu        sync.Mutex
	reported  []int64
	atStation []int64
	err       error
}

func (r *fakeReporter) ReportExpiredAtStation(_ context.Context, downlink *storage.DownlinkMessage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.atStation = append(r.atStation, downlink.QueID)
}

func (r *fakeReporter) ReportExpiredInQueue(_ context.Context, downlink *storage.DownlinkMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reported = append(r.reported, downlink.QueID)
	return r.err
}

// fakeRevoker records every downlink it is asked to revoke at its station
// and names the connected stations.
type fakeRevoker struct {
	mu        sync.Mutex
	revoked   []int64
	connected []uint64
	err       error
}

func (r *fakeRevoker) ConnectedStations() []uint64 {
	return r.connected
}

func (r *fakeRevoker) RevokeHeldDownlink(_ context.Context, downlink *storage.DownlinkMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.revoked = append(r.revoked, downlink.QueID)
	return r.err
}

type fakeQueueTenants struct{ forgotten []int64 }

func (f *fakeQueueTenants) UnregisterQueueTenant(queueID int64) {
	f.forgotten = append(f.forgotten, queueID)
}

type workerFixture struct {
	queue    *fakeQueue
	removed  *fakeRemovedStations
	reporter *fakeReporter
	revoker  *fakeRevoker
	tenants  *fakeQueueTenants
}

func (f *workerFixture) deps() Dependencies {
	return Dependencies{
		Queue:        f.queue,
		Removed:      f.removed,
		Reporter:     f.reporter,
		Revoker:      f.revoker,
		QueueTenants: f.tenants,
		Logger:       logger.NewNop(),
	}
}

func newWorkerFixture(batches ...[]*storage.DownlinkMessage) *workerFixture {
	return &workerFixture{
		queue:    &fakeQueue{unheld: fakeBatches{batches: batches}},
		removed:  &fakeRemovedStations{},
		reporter: &fakeReporter{},
		revoker:  &fakeRevoker{},
		tenants:  &fakeQueueTenants{},
	}
}

func (f *workerFixture) worker(t *testing.T) *Worker {
	t.Helper()
	w, err := NewWorker(f.deps(), Config{Interval: workerTestInterval, BatchSize: workerTestBatch})
	require.NoError(t, err)
	return w
}

func expiredRow(queID int64) *storage.DownlinkMessage {
	return &storage.DownlinkMessage{QueID: queID, Status: mioty.DLQueueStatusExpired, Result: mioty.ResultExpired}
}

// revokingRow is a downlink whose lifetime ended while workerTestStation held it.
func revokingRow(queID int64) *storage.DownlinkMessage {
	return &storage.DownlinkMessage{QueID: queID, Status: mioty.DLQueueStatusRevoking, BsEui: workerTestStation}
}

// TestSweepOnce_ReportsEachExpiredDownlink: every expired downlink is handed
// to the result reporter and its cached queue owner is forgotten; a full
// batch is followed by another.
func TestSweepOnce_ReportsEachExpiredDownlink(t *testing.T) {
	f := newWorkerFixture([]*storage.DownlinkMessage{expiredRow(501), expiredRow(502)}, nil)

	f.worker(t).SweepOnce(testutil.TestContext(), workerTestStart)

	assert.Equal(t, []int64{501, 502}, f.reporter.reported)
	assert.Equal(t, []int64{501, 502}, f.tenants.forgotten)
	assert.Equal(t, []int{workerTestBatch, workerTestBatch}, f.queue.unheld.limits, "a full batch is followed by another")
	assert.Empty(t, f.revoker.revoked)
}

// TestSweepOnce_AReportFailureDoesNotStopTheSweep: an expired downlink the
// reporter cannot identify is logged and the rest are still reported.
func TestSweepOnce_AReportFailureDoesNotStopTheSweep(t *testing.T) {
	f := newWorkerFixture([]*storage.DownlinkMessage{expiredRow(503)})
	f.reporter.err = errWorkerTestUnknown

	f.worker(t).SweepOnce(testutil.TestContext(), workerTestStart)

	assert.Equal(t, []int64{503}, f.reporter.reported)
	assert.Equal(t, 2, f.queue.calls, "both sweeps run")
}

// TestSweepOnce_AsksTheHoldingStationAndReportsNothing: a downlink whose
// lifetime ended while a base station held it is revoked there with dlDataRev
// (BSSCI §3.13), batch by batch, and neither reported nor forgotten: only the
// station's answer, or a result it reports first, decides its outcome.
func TestSweepOnce_AsksTheHoldingStationAndReportsNothing(t *testing.T) {
	f := newWorkerFixture()
	f.queue.held.batches = [][]*storage.DownlinkMessage{{revokingRow(505), revokingRow(506)}, {revokingRow(507)}}

	f.worker(t).SweepOnce(testutil.TestContext(), workerTestStart)

	assert.Equal(t, []int64{505, 506, 507}, f.revoker.revoked)
	assert.Empty(t, f.reporter.reported, "no outcome is reported before the station answers")
	assert.Empty(t, f.tenants.forgotten, "the queue owner stays cached for the answer")
	assert.Equal(t, []int{workerTestBatch, workerTestBatch}, f.queue.held.limits)
}

// TestSweepOnce_AnUnreachableStationDoesNotStopTheSweep: a revoke that cannot
// reach the station is logged, the downlink stays revoking for the station's
// next session, and the rest are still asked.
func TestSweepOnce_AnUnreachableStationDoesNotStopTheSweep(t *testing.T) {
	f := newWorkerFixture()
	f.queue.held.batches = [][]*storage.DownlinkMessage{{revokingRow(508), revokingRow(509)}}
	f.revoker.err = errWorkerTestStation

	f.worker(t).SweepOnce(testutil.TestContext(), workerTestStart)

	assert.Equal(t, []int64{508, 509}, f.revoker.revoked)
	assert.Empty(t, f.reporter.reported)
}

// TestSweepOnce_AsksAConnectedHolderThatLeftARevokeUnansweredAgain: a
// revoking downlink whose connected holder did not settle it within a sweep
// interval is asked for again, batch by batch, and nothing is reported or
// forgotten: the holder's answer still decides its outcome.
func TestSweepOnce_AsksAConnectedHolderThatLeftARevokeUnansweredAgain(t *testing.T) {
	f := newWorkerFixture()
	f.revoker.connected = []uint64{workerTestStation}
	f.queue.unanswered.batches = [][]*storage.DownlinkMessage{{revokingRow(510), revokingRow(511)}, {revokingRow(512)}}

	f.worker(t).SweepOnce(testutil.TestContext(), workerTestStart)

	assert.Equal(t, []int64{510, 511, 512}, f.revoker.revoked)
	require.NotEmpty(t, f.queue.claims)
	assert.Equal(t, unansweredClaim{connected: []uint64{workerTestStation}, askedAt: workerTestStart, unansweredFor: workerTestInterval}, f.queue.claims[0],
		"only the connected stations, once per sweep interval, asked at the sweep's start")
	assert.Empty(t, f.reporter.reported, "time passing reports nothing")
	assert.Empty(t, f.tenants.forgotten)
}

// TestSweepOnce_NoConnectedStationIsAskedNothing: without a connected
// station there is nobody to ask again, and no revocation is claimed.
func TestSweepOnce_NoConnectedStationIsAskedNothing(t *testing.T) {
	f := newWorkerFixture()
	f.queue.unanswered.batches = [][]*storage.DownlinkMessage{{revokingRow(513)}}

	f.worker(t).SweepOnce(testutil.TestContext(), workerTestStart)

	assert.Empty(t, f.queue.claims)
	assert.Empty(t, f.revoker.revoked)
}

func TestSweepOnce_StopsOnAQueueFailure(t *testing.T) {
	f := newWorkerFixture()
	f.queue.err = errWorkerTestQueue

	f.worker(t).SweepOnce(testutil.TestContext(), workerTestStart)

	assert.Equal(t, 2, f.queue.calls, "each sweep stops at its first failure")
	assert.Empty(t, f.reporter.reported)
	assert.Empty(t, f.revoker.revoked)
}

// TestStart_StopWaitsForTheSweepInFlight: the composition root stops the
// worker at shutdown and a started sweep finishes reporting first.
func TestStart_StopWaitsForTheSweepInFlight(t *testing.T) {
	f := newWorkerFixture([]*storage.DownlinkMessage{expiredRow(504)})
	f.queue.entered = make(chan struct{}, 1)
	f.queue.release = make(chan struct{})
	stop := f.worker(t).Start(testutil.TestContext())
	<-f.queue.entered

	stopped := make(chan struct{})
	go func() {
		stop()
		close(stopped)
	}()
	select {
	case <-stopped:
		t.Fatal("stop returned while a sweep was still expiring downlinks")
	case <-time.After(workerTestSettle):
	}

	close(f.queue.release)
	<-stopped
	assert.Equal(t, []int64{504}, f.reporter.reported, "the expired downlink was reported before the worker stopped")
}

func TestNewWorker_RejectsMissingCollaboratorsAndBadConfig(t *testing.T) {
	f := newWorkerFixture()
	valid := Config{Interval: workerTestInterval, BatchSize: workerTestBatch}
	missing := map[string]func(*Dependencies){
		"queue":         func(d *Dependencies) { d.Queue = nil },
		"removed":       func(d *Dependencies) { d.Removed = nil },
		"reporter":      func(d *Dependencies) { d.Reporter = nil },
		"revoker":       func(d *Dependencies) { d.Revoker = nil },
		"queue tenants": func(d *Dependencies) { d.QueueTenants = nil },
		"logger":        func(d *Dependencies) { d.Logger = nil },
	}
	for name, unset := range missing {
		t.Run(name, func(t *testing.T) {
			deps := f.deps()
			unset(&deps)
			_, err := NewWorker(deps, valid)
			assert.ErrorIs(t, err, errMissingDependency)
		})
	}
	for name, cfg := range map[string]Config{
		"interval":   {BatchSize: workerTestBatch},
		"batch size": {Interval: workerTestInterval},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewWorker(f.deps(), cfg)
			assert.ErrorIs(t, err, errInvalidConfig)
		})
	}
}

// TestSweepOnce_EndsWhatADeletedStationLeftHeld: a downlink held by a station
// that no longer exists ends expired, is reported as expired at that station
// and its cached owner is forgotten, batch by batch.
func TestSweepOnce_EndsWhatADeletedStationLeftHeld(t *testing.T) {
	f := newWorkerFixture()
	f.removed.orphaned.batches = [][]*storage.DownlinkMessage{{revokingRow(520), revokingRow(521)}, {revokingRow(522)}}

	f.worker(t).SweepOnce(testutil.TestContext(), workerTestStart)

	assert.Equal(t, []int64{520, 521, 522}, f.reporter.atStation)
	assert.Equal(t, []int64{520, 521, 522}, f.tenants.forgotten)
	assert.Empty(t, f.reporter.reported, "they ended at a station, not in the queue")
	assert.Equal(t, []int{workerTestBatch, workerTestBatch}, f.removed.orphaned.limits)

	failing := newWorkerFixture()
	failing.removed.err = errWorkerTestQueue
	failing.queue.held.batches = [][]*storage.DownlinkMessage{{revokingRow(523)}}
	failing.worker(t).SweepOnce(testutil.TestContext(), workerTestStart)
	assert.Equal(t, []int64{523}, failing.revoker.revoked, "a failure there leaves the other sweeps running")
}

// TestSweepOnce_RecordsTheSweepStartAsTheAsk: the overdue downlinks a sweep
// moves to revoking are asked at its start, so the next sweep, one interval
// later, finds an unanswered one due.
func TestSweepOnce_RecordsTheSweepStartAsTheAsk(t *testing.T) {
	f := newWorkerFixture()

	f.worker(t).SweepOnce(testutil.TestContext(), workerTestStart)

	assert.Equal(t, []time.Time{workerTestStart}, f.queue.askTimes())
}

// TestStart_AsksAtTheScheduledTicks: the running worker starts each sweep on
// the grid of whole intervals, so consecutive asks lie whole intervals apart
// however late a tick fires.
func TestStart_AsksAtTheScheduledTicks(t *testing.T) {
	f := newWorkerFixture()
	stop := f.worker(t).Start(testutil.TestContext())
	require.Eventually(t, func() bool { return len(f.queue.askTimes()) >= 3 }, workerTestSettle*10, workerTestInterval)
	stop()

	asks := f.queue.askTimes()
	for i := 1; i < len(asks); i++ {
		gap := asks[i].Sub(asks[i-1])
		assert.Positive(t, gap)
		assert.Zero(t, gap%workerTestInterval, "sweep %d asked %v after the one before", i, gap)
	}
}
