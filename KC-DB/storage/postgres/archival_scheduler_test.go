package postgres

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// Scheduler timing for the tests: every clock read falls inside the archival
// hour, and waits are bounded so a regression fails instead of hanging.
const (
	schedulerTestHour        = 2
	schedulerTestWait        = 2 * time.Second
	schedulerTestTick        = time.Millisecond
	schedulerTestObservation = 200 * time.Millisecond
	schedulerTestPoll        = 5 * time.Millisecond
)

// fixedClock reports one instant inside the archival hour.
type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

// blockingArchiver holds every archival run until it is released or its
// context ends, and records how many runs are in flight at once.
type blockingArchiver struct {
	mu       sync.Mutex
	inFlight int
	peak     int
	started  chan struct{}
	release  chan struct{}
	once     sync.Once
}

func newBlockingArchiver() *blockingArchiver {
	return &blockingArchiver{started: make(chan struct{}, 1), release: make(chan struct{})}
}

func (a *blockingArchiver) ArchiveOldMessages(ctx context.Context, _ time.Duration) (int64, error) {
	a.mu.Lock()
	a.inFlight++
	if a.inFlight > a.peak {
		a.peak = a.inFlight
	}
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.inFlight--
		a.mu.Unlock()
	}()

	select {
	case a.started <- struct{}{}:
	default:
	}
	select {
	case <-a.release:
		return 0, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func (a *blockingArchiver) CreateMonthlyPartitions(context.Context, int) error { return nil }

func (a *blockingArchiver) PurgeArchivedMessages(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (a *blockingArchiver) releaseAll() { a.once.Do(func() { close(a.release) }) }

func (a *blockingArchiver) counts() (inFlight, peak int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.inFlight, a.peak
}

func (a *blockingArchiver) awaitStart(t *testing.T) {
	t.Helper()
	select {
	case <-a.started:
	case <-time.After(schedulerTestWait):
		t.Fatal("the archival run never started")
	}
}

// stopWithin stops the scheduler and fails the test if Stop does not return.
func stopWithin(t *testing.T, scheduler *ArchivalScheduler) {
	t.Helper()
	stopped := make(chan error, 1)
	go func() { stopped <- scheduler.Stop() }()
	select {
	case err := <-stopped:
		require.NoError(t, err)
	case <-time.After(schedulerTestWait):
		t.Fatal("Stop did not return")
	}
}

func newTestScheduler(archiver messageArchiver, checkInterval time.Duration) *ArchivalScheduler {
	return NewArchivalScheduler(archiver, logger.NewNop(), ArchivalConfig{
		MessageRetentionDays: defaultMessageRetentionDays,
		MessageArchivalHour:  schedulerTestHour,
		ArchivalEnabled:      defaultArchivalEnabled,
		CheckInterval:        checkInterval,
	}, fixedClock{now: time.Date(2026, time.September, 27, schedulerTestHour, 30, 0, 0, time.UTC)})
}

// TestArchivalScheduler_StopWaitsForTheRunningArchival proves Stop does not
// return while an archival run is still using the database.
func TestArchivalScheduler_StopWaitsForTheRunningArchival(t *testing.T) {
	archiver := newBlockingArchiver()
	t.Cleanup(archiver.releaseAll)
	scheduler := newTestScheduler(archiver, time.Hour)

	require.NoError(t, scheduler.Start(testutil.TestContext()))
	archiver.awaitStart(t)
	stopWithin(t, scheduler)

	inFlight, _ := archiver.counts()
	assert.Zero(t, inFlight, "Stop must return only after the running archival has ended")
}

// TestArchivalScheduler_RunsNeverOverlap proves a run that outlasts the check
// interval is not joined by a second run.
func TestArchivalScheduler_RunsNeverOverlap(t *testing.T) {
	archiver := newBlockingArchiver()
	t.Cleanup(archiver.releaseAll)
	scheduler := newTestScheduler(archiver, schedulerTestTick)

	require.NoError(t, scheduler.Start(testutil.TestContext()))
	archiver.awaitStart(t)

	assert.Never(t, func() bool {
		_, peak := archiver.counts()
		return peak > 1
	}, schedulerTestObservation, schedulerTestPoll, "archival runs must not overlap")
	archiver.releaseAll()
	stopWithin(t, scheduler)
}

// TestArchivalScheduler_RestartsAfterStop proves the scheduler can be started
// again once stopped.
func TestArchivalScheduler_RestartsAfterStop(t *testing.T) {
	archiver := newBlockingArchiver()
	archiver.releaseAll()
	scheduler := newTestScheduler(archiver, time.Hour)

	require.NoError(t, scheduler.Start(testutil.TestContext()))
	stopWithin(t, scheduler)
	require.NoError(t, scheduler.Start(testutil.TestContext()))
	stopWithin(t, scheduler)
}
