package certcleanup

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	testInterval = 5 * time.Millisecond
	testWait     = 2 * time.Second
	testSettle   = 10 * testInterval
)

var testNow = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

// recordingSweeper records the time of every sweep it is asked for.
type recordingSweeper struct {
	mu    sync.Mutex
	times []time.Time
}

func (r *recordingSweeper) CleanupExpiredCertificates(_ context.Context, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.times = append(r.times, now)
}

func (r *recordingSweeper) sweeps() []time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]time.Time(nil), r.times...)
}

func TestNewWorker_RefusesMissingDependenciesAndIntervals(t *testing.T) {
	clk := testutil.NewFakeClock(testNow)
	if _, err := NewWorker(nil, clk, testInterval); !errors.Is(err, errMissingDependency) {
		t.Fatalf("nil sweeper: err = %v", err)
	}
	if _, err := NewWorker(&recordingSweeper{}, nil, testInterval); !errors.Is(err, errMissingDependency) {
		t.Fatalf("nil clock: err = %v", err)
	}
	if _, err := NewWorker(&recordingSweeper{}, clk, 0); !errors.Is(err, errInvalidInterval) {
		t.Fatalf("zero interval: err = %v", err)
	}
}

// TestWorker_SweepsAtTheClockTimeUntilStopped runs the loop: every sweep is
// asked for at the injected clock's time, and none runs after stop returns.
func TestWorker_SweepsAtTheClockTimeUntilStopped(t *testing.T) {
	sweeper := &recordingSweeper{}
	worker, err := NewWorker(sweeper, testutil.NewFakeClock(testNow), testInterval)
	if err != nil {
		t.Fatalf("NewWorker: %v", err)
	}

	stop := worker.Start(testutil.TestContext())
	deadline := time.Now().Add(testWait)
	for len(sweeper.sweeps()) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("the running worker never swept")
		}
		time.Sleep(testInterval)
	}
	stop()

	stopped := len(sweeper.sweeps())
	time.Sleep(testSettle)
	sweeps := sweeper.sweeps()
	if len(sweeps) != stopped {
		t.Fatalf("a stopped worker still swept: %d sweeps after stop, %d before", len(sweeps), stopped)
	}
	for _, at := range sweeps {
		if !at.Equal(testNow) {
			t.Fatalf("sweep asked for at %v, want the clock's %v", at, testNow)
		}
	}
}
