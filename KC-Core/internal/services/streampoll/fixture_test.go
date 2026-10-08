package streampoll

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/streamwake"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	testInterval = 5 * time.Millisecond
	testWindow   = 100 * time.Millisecond
	testBuffer   = 4
	testPageSize = 100
	// testBurst is more rows than two pages hold, stored within one interval.
	testBurst = 250
	// testOverlap is how far each read reaches back behind the newest row read.
	testOverlap = time.Second
)

var errStoreDown = errors.New("store down")

// row is a stored row whose id is independent of the time it was stored at,
// as a table row's is: several rows can share a time.
type row struct {
	id string
	at time.Time
}

// compareRows orders rows the way the stores sort them: by time, then by id.
func compareRows(a, b row) int {
	if byTime := a.at.Compare(b.at); byTime != 0 {
		return byTime
	}
	return strings.Compare(a.id, b.id)
}

// rows is a store read newest first, a page at a time; failures makes the
// next reads fail, and onRead runs before each read is served.
type rows struct {
	mu       sync.Mutex
	stored   []row
	failures int
	errs     int
	reads    int
	onRead   func(read int)
}

// readAtLeast reports whether the stream has read the store n times.
func (r *rows) readAtLeast(n int) func() bool {
	return func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.reads >= n
	}
}

func (r *rows) fetch(_ context.Context, since *time.Time, offset int) ([]row, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reads++
	if r.onRead != nil {
		r.onRead(r.reads)
	}
	if r.failures > 0 {
		r.failures--
		return nil, errStoreDown
	}
	var inWindow []row
	for i := len(r.stored) - 1; i >= 0; i-- {
		if since == nil || !r.stored[i].at.Before(*since) {
			inWindow = append(inWindow, r.stored[i])
		}
	}
	if offset >= len(inWindow) {
		return nil, nil
	}
	return inWindow[offset:min(offset+testPageSize, len(inWindow))], nil
}

// add stores rows, keeping the store in its sort order; a row whose id is
// stored already is stored again at its new time.
func (r *rows) add(stored ...row) {
	for _, again := range stored {
		r.stored = slices.DeleteFunc(r.stored, func(x row) bool { return x.id == again.id })
	}
	r.stored = append(r.stored, stored...)
	slices.SortFunc(r.stored, compareRows)
}

func (r *rows) store(stored ...row) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.add(stored...)
}

func (r *rows) source() Source[row] {
	return Source[row]{
		Fetch:    r.fetch,
		PageSize: testPageSize,
		StoredAt: func(x row) time.Time { return x.at },
		Key:      func(x row) string { return x.id },
		Overlap:  testOverlap,
		OnError:  func(error) { r.mu.Lock(); r.errs++; r.mu.Unlock() },
		Wake:     streamwake.NewSignal(),
	}
}

// ids names rows in their order.
func ids(of []row) []string {
	names := make([]string, len(of))
	for i, x := range of {
		names[i] = x.id
	}
	return names
}

// drain collects the ids a stream sends within the test window.
func drain(ch <-chan row) []string {
	var got []string
	window := time.After(testWindow)
	for {
		select {
		case x, ok := <-ch:
			if !ok {
				return got
			}
			got = append(got, x.id)
		case <-window:
			return got
		}
	}
}

// sent closes out and returns the ids a cursor delivered into it.
func sent(out chan row) []string {
	close(out)
	var got []string
	for x := range out {
		got = append(got, x.id)
	}
	return got
}

// opened is a cursor whose baseline has read the store.
func opened(store *rows) *cursor[row] {
	c := &cursor[row]{src: store.source(), seen: make(map[string]time.Time)}
	c.baseline(testutil.TestContext())
	return c
}
