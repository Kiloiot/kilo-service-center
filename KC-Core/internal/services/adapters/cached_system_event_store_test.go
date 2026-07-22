package adapters

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	eventsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/events"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

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

func TestCachedCount_HitWithinTTL(t *testing.T) {
	inner := &fakeStore{countVal: 42}
	c := NewCachedSystemEventStore(inner, 10*time.Second)
	f := &eventsservice.EventFilter{Categories: []string{"a"}}

	v1, err := c.CountEvents(testutil.TestContext(), 1, f)
	if err != nil || v1 != 42 {
		t.Fatalf("first: got (%d,%v), want (42,nil)", v1, err)
	}
	v2, _ := c.CountEvents(testutil.TestContext(), 1, f)
	if v2 != 42 {
		t.Fatalf("second: got %d, want 42", v2)
	}
	if got := atomic.LoadInt32(&inner.countCalls); got != 1 {
		t.Fatalf("inner count calls = %d, want 1 (cache hit)", got)
	}
}

func TestCachedCount_ExpiryAfterTTL(t *testing.T) {
	inner := &fakeStore{countVal: 7}
	c := NewCachedSystemEventStore(inner, 10*time.Second)
	now := time.Unix(1000, 0)
	c.now = func() time.Time { return now }
	f := &eventsservice.EventFilter{Categories: []string{"a"}}

	_, _ = c.CountEvents(testutil.TestContext(), 1, f)
	now = now.Add(11 * time.Second) // past TTL
	_, _ = c.CountEvents(testutil.TestContext(), 1, f)

	if got := atomic.LoadInt32(&inner.countCalls); got != 2 {
		t.Fatalf("inner count calls = %d, want 2 (expired)", got)
	}
}

func TestCachedCount_ErrorsNotCached(t *testing.T) {
	inner := &fakeStore{countErr: errors.New("boom")}
	c := NewCachedSystemEventStore(inner, 10*time.Second)
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
	inner := &fakeStore{countVal: 1}
	c := NewCachedSystemEventStore(inner, 10*time.Second)
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
	inner := &fakeStore{countVal: 1}
	c := NewCachedSystemEventStore(inner, 10*time.Second)
	ctx := testutil.TestContext()

	_, _ = c.CountEvents(ctx, 1, &eventsservice.EventFilter{Categories: []string{"a", "b"}})
	_, _ = c.CountEvents(ctx, 1, &eventsservice.EventFilter{Categories: []string{"b", "a"}}) // same key

	if got := atomic.LoadInt32(&inner.countCalls); got != 1 {
		t.Fatalf("inner count calls = %d, want 1 (order-independent key)", got)
	}
}

func TestCachedCount_GetEventsPassthrough(t *testing.T) {
	inner := &fakeStore{}
	c := NewCachedSystemEventStore(inner, 10*time.Second)

	_, _ = c.GetEvents(testutil.TestContext(), 1, nil, 10, 0)
	_, _ = c.GetEvents(testutil.TestContext(), 1, nil, 10, 0)

	if got := atomic.LoadInt32(&inner.getCalls); got != 2 {
		t.Fatalf("inner GetEvents calls = %d, want 2 (never cached)", got)
	}
}

func TestCachedCount_SingleflightCollapse(t *testing.T) {
	gate := make(chan struct{})
	inner := &fakeStore{countVal: 5, countGate: gate}
	c := NewCachedSystemEventStore(inner, 10*time.Second)
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
	time.Sleep(100 * time.Millisecond)
	close(gate)
	wg.Wait()

	if got := atomic.LoadInt32(&inner.countCalls); got != 1 {
		t.Fatalf("inner count calls = %d, want 1 (singleflight)", got)
	}
	for i, v := range results {
		if v != 5 {
			t.Fatalf("result[%d] = %d, want 5", i, v)
		}
	}
}
