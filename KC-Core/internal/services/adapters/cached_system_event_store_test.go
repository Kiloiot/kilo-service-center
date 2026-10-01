package adapters

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	eventsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/events"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// Timing knobs and count seeds for the cache tests.
const (
	testCountTTL          = 10 * time.Second
	testPastTTL           = testCountTTL + time.Second
	testGoroutineSettle   = 100 * time.Millisecond
	testCountTimeout      = 5 * time.Second
	testShortCountTimeout = 50 * time.Millisecond

	testCachedCount       = 42
	testExpiredCount      = 7
	testKeyedCount        = 1
	testSingleflightCount = 5
	testSharedCount       = 9
)

// errCountFailure is the failure the fake store returns for count queries.
var errCountFailure = errors.New("boom")

type fakeStore struct {
	countCalls int32
	getCalls   int32
	countVal   int64
	countErr   error
	countGate  chan struct{} // if non-nil, CountEvents blocks until closed
}

func (f *fakeStore) GetEvents(_ context.Context, _ int64, _ *eventsservice.EventFilter, _, _ int) ([]*models.SystemEvent, error) {
	atomic.AddInt32(&f.getCalls, 1)
	return nil, nil
}

func (f *fakeStore) CountEvents(_ context.Context, _ int64, _ *eventsservice.EventFilter) (int64, error) {
	atomic.AddInt32(&f.countCalls, 1)
	if f.countGate != nil {
		<-f.countGate
	}
	if f.countErr != nil {
		return 0, f.countErr
	}
	return f.countVal, nil
}

var testClockStart = time.Unix(1000, 0)

// steppingClock is a clock the test moves forward by hand.
type steppingClock struct{ now time.Time }

func (c *steppingClock) Now() time.Time { return c.now }

func newTestCountCache(t *testing.T, inner eventsservice.SystemEventStore, clk clock.Clock) *CachedSystemEventStore {
	t.Helper()
	c, err := NewCachedSystemEventStore(inner, testCountTTL, testCountTimeout, clk)
	require.NoError(t, err)
	return c
}

// blockingStore holds every count until release closes or its context ends.
type blockingStore struct {
	fakeStore
	started chan struct{}
	release chan struct{}
	once    sync.Once
	mu      sync.Mutex
	lastErr error
}

func (b *blockingStore) CountEvents(ctx context.Context, _ int64, _ *eventsservice.EventFilter) (int64, error) {
	atomic.AddInt32(&b.countCalls, 1)
	b.once.Do(func() { close(b.started) })
	var err error
	select {
	case <-b.release:
	case <-ctx.Done():
		err = ctx.Err()
	}
	b.mu.Lock()
	b.lastErr = err
	b.mu.Unlock()
	if err != nil {
		return 0, err
	}
	return testSharedCount, nil
}

func (b *blockingStore) countErr() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastErr
}

func TestCachedCount_CallerCancellationDoesNotCancelTheSharedCount(t *testing.T) {
	inner := &blockingStore{started: make(chan struct{}), release: make(chan struct{})}
	c := newTestCountCache(t, inner, clock.SystemClock{})
	f := &eventsservice.EventFilter{Categories: []string{"a"}}

	first, cancelFirst := context.WithCancel(testutil.TestContext())
	firstDone := make(chan error, 1)
	go func() {
		_, err := c.CountEvents(first, 1, f)
		firstDone <- err
	}()
	<-inner.started

	type countResult struct {
		n   int64
		err error
	}
	secondDone := make(chan countResult, 1)
	go func() {
		n, err := c.CountEvents(testutil.TestContext(), 1, f)
		secondDone <- countResult{n: n, err: err}
	}()
	time.Sleep(testGoroutineSettle)

	cancelFirst()
	require.ErrorIs(t, <-firstDone, context.Canceled, "the first caller stops waiting on its own context")

	close(inner.release)
	second := <-secondDone
	require.NoError(t, second.err, "another caller's cancellation must not fail the shared count")
	assert.Equal(t, int64(testSharedCount), second.n)
	assert.NoError(t, inner.countErr(), "the shared count ran to completion")
	assert.Equal(t, int32(1), atomic.LoadInt32(&inner.countCalls), "both callers shared one count")
}

func TestCachedCount_SharedCountHasItsOwnTimeout(t *testing.T) {
	inner := &blockingStore{started: make(chan struct{}), release: make(chan struct{})}
	c, err := NewCachedSystemEventStore(inner, testCountTTL, testShortCountTimeout, clock.SystemClock{})
	require.NoError(t, err)

	_, err = c.CountEvents(testutil.TestContext(), 1, &eventsservice.EventFilter{Categories: []string{"a"}})
	require.ErrorIs(t, err, context.DeadlineExceeded, "a count that never finishes gives up after the cache's timeout")
}

func TestNewCachedSystemEventStore_RejectsMissingCollaborators(t *testing.T) {
	_, err := NewCachedSystemEventStore(nil, testCountTTL, testCountTimeout, clock.SystemClock{})
	assert.ErrorIs(t, err, ErrNilCountedStore)
	_, err = NewCachedSystemEventStore(&fakeStore{}, testCountTTL, testCountTimeout, nil)
	assert.ErrorIs(t, err, ErrNilCountClock)
}

func TestCachedCount_HitWithinTTL(t *testing.T) {
	inner := &fakeStore{countVal: testCachedCount}
	c := newTestCountCache(t, inner, clock.SystemClock{})
	f := &eventsservice.EventFilter{Categories: []string{"a"}}

	v1, err := c.CountEvents(testutil.TestContext(), 1, f)
	if err != nil || v1 != testCachedCount {
		t.Fatalf("first: got (%d,%v), want (42,nil)", v1, err)
	}
	v2, _ := c.CountEvents(testutil.TestContext(), 1, f)
	if v2 != testCachedCount {
		t.Fatalf("second: got %d, want 42", v2)
	}
	if got := atomic.LoadInt32(&inner.countCalls); got != 1 {
		t.Fatalf("inner count calls = %d, want 1 (cache hit)", got)
	}
}

func TestCachedCount_ExpiryAfterTTL(t *testing.T) {
	inner := &fakeStore{countVal: testExpiredCount}
	clk := &steppingClock{now: testClockStart}
	c := newTestCountCache(t, inner, clk)
	f := &eventsservice.EventFilter{Categories: []string{"a"}}

	_, _ = c.CountEvents(testutil.TestContext(), 1, f)
	clk.now = clk.now.Add(testPastTTL)
	_, _ = c.CountEvents(testutil.TestContext(), 1, f)

	if got := atomic.LoadInt32(&inner.countCalls); got != 2 {
		t.Fatalf("inner count calls = %d, want 2 (expired)", got)
	}
}

func TestCachedCount_ErrorsNotCached(t *testing.T) {
	inner := &fakeStore{countErr: errCountFailure}
	c := newTestCountCache(t, inner, clock.SystemClock{})
	f := &eventsservice.EventFilter{Categories: []string{"a"}}

	if _, err := c.CountEvents(testutil.TestContext(), 1, f); err == nil {
		t.Fatal("want error")
	}
	if _, err := c.CountEvents(testutil.TestContext(), 1, f); err == nil {
		t.Fatal("want error")
	}
	if got := atomic.LoadInt32(&inner.countCalls); got != 2 {
		t.Fatalf("inner count calls = %d, want 2 (errors not cached)", got)
	}
}

func TestCachedCount_KeyVariesByFilterAndTenant(t *testing.T) {
	inner := &fakeStore{countVal: testKeyedCount}
	c := newTestCountCache(t, inner, clock.SystemClock{})
	ctx := testutil.TestContext()

	_, _ = c.CountEvents(ctx, 1, &eventsservice.EventFilter{Categories: []string{"a"}})      // miss -> 1
	_, _ = c.CountEvents(ctx, 1, &eventsservice.EventFilter{Categories: []string{"b"}})      // miss -> 2
	_, _ = c.CountEvents(ctx, 2, &eventsservice.EventFilter{Categories: []string{"a"}})      // diff tenant -> 3
	_, _ = c.CountEvents(ctx, 1, &eventsservice.EventFilter{Categories: []string{"b", "a"}}) // sorted key "a,b" -> miss 4
	_, _ = c.CountEvents(ctx, 1, &eventsservice.EventFilter{Categories: []string{"a"}})      // same as first -> hit

	if got := atomic.LoadInt32(&inner.countCalls); got != 4 {
		t.Fatalf("inner count calls = %d, want 4", got)
	}
}

func TestCachedCount_SortedCategoriesShareKey(t *testing.T) {
	inner := &fakeStore{countVal: testKeyedCount}
	c := newTestCountCache(t, inner, clock.SystemClock{})
	ctx := testutil.TestContext()

	_, _ = c.CountEvents(ctx, 1, &eventsservice.EventFilter{Categories: []string{"a", "b"}})
	_, _ = c.CountEvents(ctx, 1, &eventsservice.EventFilter{Categories: []string{"b", "a"}}) // same key

	if got := atomic.LoadInt32(&inner.countCalls); got != 1 {
		t.Fatalf("inner count calls = %d, want 1 (order-independent key)", got)
	}
}

func TestCachedCount_GetEventsPassthrough(t *testing.T) {
	inner := &fakeStore{}
	c := newTestCountCache(t, inner, clock.SystemClock{})

	_, _ = c.GetEvents(testutil.TestContext(), 1, nil, 10, 0)
	_, _ = c.GetEvents(testutil.TestContext(), 1, nil, 10, 0)

	if got := atomic.LoadInt32(&inner.getCalls); got != 2 {
		t.Fatalf("inner GetEvents calls = %d, want 2 (never cached)", got)
	}
}

func TestCachedCount_SingleflightCollapse(t *testing.T) {
	gate := make(chan struct{})
	inner := &fakeStore{countVal: testSingleflightCount, countGate: gate}
	c := newTestCountCache(t, inner, clock.SystemClock{})
	f := &eventsservice.EventFilter{Categories: []string{"a"}}

	const n = 10
	var wg sync.WaitGroup
	results := make([]int64, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v, _ := c.CountEvents(testutil.TestContext(), 1, f)
			results[i] = v
		}(i)
	}

	// Give all goroutines time to reach singleflight.Do (leader is blocked on gate).
	time.Sleep(testGoroutineSettle)
	close(gate)
	wg.Wait()

	if got := atomic.LoadInt32(&inner.countCalls); got != 1 {
		t.Fatalf("inner count calls = %d, want 1 (singleflight)", got)
	}
	for i, v := range results {
		if v != testSingleflightCount {
			t.Fatalf("result[%d] = %d, want 5", i, v)
		}
	}
}
