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

func (q *fakeQueue) RevokeOverdueHeld(_ context.Context, limit int) ([]*storage.DownlinkMessage, error) {
	return q.take(&q.held, limit)
}

func (q *fakeQueue) ClaimUnansweredRevocations(_ context.Context, connected []uint64, unansweredFor time.Duration, limit int) ([]*storage.DownlinkMessage, error) {
	q.mu.Lock()
	q.claims = append(q.claims, unansweredClaim{connected: connected, unansweredFor: unansweredFor})
	q.mu.Unlock()
	return q.take(&q.unanswered, limit)
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

// fakeReporter records every expired downlink it is asked to report.
type fakeReporter struct {
	mu       sync.Mutex
	reported []int64
	err      error
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
	reporter *fakeReporter
	revoker  *fakeRevoker
	tenants  *fakeQueueTenants
}

func (f *workerFixture) deps() Dependencies {
	return Dependencies{
		Queue:        f.queue,
		Reporter:     f.reporter,
		Revoker:      f.revoker,
		QueueTenants: f.tenants,
		Logger:       logger.NewNop(),
	}
}

func newWorkerFixture(batches ...[]*storage.DownlinkMessage) *workerFixture {
	return &workerFixture{
		queue:    &fakeQueue{unheld: fakeBatches{batches: batches}},
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

	f.worker(t).SweepOnce(testutil.TestContext())

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

	f.worker(t).SweepOnce(testutil.TestContext())

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

	f.worker(t).SweepOnce(testutil.TestContext())

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

	f.worker(t).SweepOnce(testutil.TestContext())

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

	f.worker(t).SweepOnce(testutil.TestContext())

	assert.Equal(t, []int64{510, 511, 512}, f.revoker.revoked)
	require.NotEmpty(t, f.queue.claims)
	assert.Equal(t, unansweredClaim{connected: []uint64{workerTestStation}, unansweredFor: workerTestInterval}, f.queue.claims[0],
		"only the connected stations, once per sweep interval")
	assert.Empty(t, f.reporter.reported, "time passing reports nothing")
	assert.Empty(t, f.tenants.forgotten)
}

// TestSweepOnce_NoConnectedStationIsAskedNothing: without a connected
// station there is nobody to ask again, and no revocation is claimed.
func TestSweepOnce_NoConnectedStationIsAskedNothing(t *testing.T) {
	f := newWorkerFixture()
	f.queue.unanswered.batches = [][]*storage.DownlinkMessage{{revokingRow(513)}}

	f.worker(t).SweepOnce(testutil.TestContext())

	assert.Empty(t, f.queue.claims)
	assert.Empty(t, f.revoker.revoked)
}

func TestSweepOnce_StopsOnAQueueFailure(t *testing.T) {
	f := newWorkerFixture()
	f.queue.err = errWorkerTestQueue

	f.worker(t).SweepOnce(testutil.TestContext())

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
