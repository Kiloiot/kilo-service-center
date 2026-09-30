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

// fakeQueue hands out the configured batches, one per call.
type fakeQueue struct {
	mu      sync.Mutex
	batches [][]*storage.DownlinkMessage
	err     error
	calls   int
	limits  []int
	entered chan struct{}
	release chan struct{}
}

func (q *fakeQueue) ExpireOverdueDownlinks(_ context.Context, limit int) ([]*storage.DownlinkMessage, error) {
	if q.entered != nil {
		q.entered <- struct{}{}
		<-q.release
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.calls++
	q.limits = append(q.limits, limit)
	if q.err != nil {
		return nil, q.err
	}
	if len(q.batches) == 0 {
		return nil, nil
	}
	batch := q.batches[0]
	q.batches = q.batches[1:]
	return batch, nil
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

// fakeRevoker records every downlink it is asked to revoke at its station.
type fakeRevoker struct {
	mu      sync.Mutex
	revoked []int64
	err     error
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
		queue:    &fakeQueue{batches: batches},
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

// heldRow is a downlink that expired while workerTestStation held it.
func heldRow(queID int64) *storage.DownlinkMessage {
	row := expiredRow(queID)
	row.BsEui = workerTestStation
	return row
}

// TestSweepOnce_ReportsEachExpiredDownlink: every expired downlink is handed
// to the result reporter and its cached queue owner is forgotten; a full
// batch is followed by another.
func TestSweepOnce_ReportsEachExpiredDownlink(t *testing.T) {
	f := newWorkerFixture([]*storage.DownlinkMessage{expiredRow(501), expiredRow(502)}, nil)

	f.worker(t).SweepOnce(testutil.TestContext())

	assert.Equal(t, []int64{501, 502}, f.reporter.reported)
	assert.Equal(t, []int64{501, 502}, f.tenants.forgotten)
	assert.Equal(t, []int{workerTestBatch, workerTestBatch}, f.queue.limits, "a full batch is followed by another")
}

// TestSweepOnce_AReportFailureDoesNotStopTheSweep: an expired downlink the
// reporter cannot identify is logged and the rest are still reported.
func TestSweepOnce_AReportFailureDoesNotStopTheSweep(t *testing.T) {
	f := newWorkerFixture([]*storage.DownlinkMessage{expiredRow(503)})
	f.reporter.err = errWorkerTestUnknown

	f.worker(t).SweepOnce(testutil.TestContext())

	assert.Equal(t, []int64{503}, f.reporter.reported)
	assert.Equal(t, 1, f.queue.calls)
}

// TestSweepOnce_RevokesAHeldDownlinkAtItsStation: a downlink that expired
// while a base station held it is reported expired and revoked there with
// dlDataRev (BSSCI §3.13); one no station held is only reported.
func TestSweepOnce_RevokesAHeldDownlinkAtItsStation(t *testing.T) {
	f := newWorkerFixture([]*storage.DownlinkMessage{heldRow(505), expiredRow(506)})

	f.worker(t).SweepOnce(testutil.TestContext())

	assert.Equal(t, []int64{505, 506}, f.reporter.reported)
	assert.Equal(t, []int64{505}, f.revoker.revoked, "an expired downlink a station holds is revoked there")
}

// TestSweepOnce_AnUnreachableStationDoesNotStopTheSweep: a revoke that cannot
// reach the station is logged and the rest are still handled.
func TestSweepOnce_AnUnreachableStationDoesNotStopTheSweep(t *testing.T) {
	f := newWorkerFixture([]*storage.DownlinkMessage{heldRow(507), heldRow(508)})
	f.revoker.err = errWorkerTestStation

	f.worker(t).SweepOnce(testutil.TestContext())

	assert.Equal(t, []int64{507, 508}, f.revoker.revoked)
	assert.Equal(t, []int64{507, 508}, f.reporter.reported)
}

func TestSweepOnce_StopsOnAQueueFailure(t *testing.T) {
	f := newWorkerFixture()
	f.queue.err = errWorkerTestQueue

	f.worker(t).SweepOnce(testutil.TestContext())

	assert.Equal(t, 1, f.queue.calls)
	assert.Empty(t, f.reporter.reported)
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
