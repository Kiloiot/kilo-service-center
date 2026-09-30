package streampoll

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

func TestStream_BaselineWaitsForAReadThatSucceeds(t *testing.T) {
	now := time.Now()
	store := &rows{stored: []row{{id: "history", at: now.Add(-time.Minute)}}, failures: 2}
	ctx, cancel := testutil.TestContextWithCancel()
	defer cancel()

	ch := Stream(ctx, testInterval, testBuffer, store.source())
	require.Eventually(t, store.readAtLeast(3), testWindow, testInterval, "two failed reads, then the baseline")
	store.store(row{id: "new", at: now})

	assert.Equal(t, []string{"new"}, drain(ch), "history stays out even when the first reads failed")
	store.mu.Lock()
	defer store.mu.Unlock()
	assert.Equal(t, 2, store.errs, "each failed read is reported")
}

func TestStream_DeliversOldestFirstAndClosesWithTheContext(t *testing.T) {
	now := time.Now()
	store := &rows{}
	ctx, cancel := testutil.TestContextWithCancel()

	ch := Stream(ctx, testInterval, testBuffer, store.source())
	require.Eventually(t, store.readAtLeast(1), testWindow, testInterval)
	store.store(row{id: "first", at: now})
	store.store(row{id: "second", at: now.Add(time.Second)})
	assert.Equal(t, []string{"first", "second"}, drain(ch))

	cancel()
	_, open := <-ch
	assert.False(t, open)
}

// burst is testBurst rows a millisecond apart after base.
func burst(base time.Time) []row {
	stored := make([]row, testBurst)
	for i := range stored {
		stored[i] = row{id: fmt.Sprintf("burst-%03d", i), at: base.Add(time.Duration(i+1) * time.Millisecond)}
	}
	return stored
}

// More rows than a page holds, stored within one interval, are all delivered.
func TestStream_DrainsABurstLargerThanAPage(t *testing.T) {
	base := time.Now()
	store := &rows{stored: []row{{id: "base", at: base}}}
	c := opened(store)
	want := burst(base)
	store.add(want...)

	out := make(chan row, testBurst)
	require.True(t, c.deliver(testutil.TestContext(), out))

	assert.Equal(t, ids(want), sent(out), "every row since the cursor, oldest first")
	assert.Equal(t, want[len(want)-1].at, *c.latest)
}

// A newer row stored while the pages are read shifts the later pages; the
// rows read twice are delivered once, and the newer row, landing in a page
// already read, waits for the next drain.
func TestStream_DeliversEachRowOnceWhenRowsArriveDuringADrain(t *testing.T) {
	base := time.Now()
	store := &rows{stored: []row{{id: "base", at: base}}}
	c := opened(store)
	want := burst(base)
	store.add(want...)
	late := row{id: "late", at: base.Add(time.Microsecond)}
	newer := row{id: "newer", at: want[len(want)-1].at.Add(time.Second)}
	store.onRead = func(read int) {
		if read == 4 {
			store.add(late, newer)
		}
	}

	out := make(chan row, 2*testBurst)
	require.True(t, c.deliver(testutil.TestContext(), out))

	assert.Equal(t, append([]string{late.id}, ids(want)...), sent(out))
}

func TestStream_ADrainThatFailsDeliversNothingAndRetries(t *testing.T) {
	base := time.Now()
	store := &rows{stored: []row{{id: "base", at: base}}}
	c := opened(store)
	store.add(burst(base)...)
	store.onRead = func(read int) {
		if read == 4 {
			store.failures = 1
		}
	}

	out := make(chan row, testBurst)
	require.True(t, c.deliver(testutil.TestContext(), out))
	assert.Empty(t, out, "a partial drain is not delivered")
	assert.Equal(t, base, *c.latest)
	assert.Equal(t, 1, store.errs)

	require.True(t, c.deliver(testutil.TestContext(), out))
	assert.Len(t, out, testBurst)
}
