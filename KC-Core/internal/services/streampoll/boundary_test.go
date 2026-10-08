package streampoll

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/streamwake"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// tied is n rows named prefix-000... all stored at the same time.
func tied(prefix string, at time.Time, n int) []row {
	stored := make([]row, n)
	for i := range stored {
		stored[i] = row{id: fmt.Sprintf("%s-%03d", prefix, i), at: at}
	}
	return stored
}

// deliverOnce runs one drain of the cursor and returns what it delivered.
func deliverOnce(t *testing.T, c *cursor[row]) []string {
	t.Helper()
	out := make(chan row, 2*testBurst)
	require.True(t, c.deliver(testutil.TestContext(), out))
	return sent(out)
}

// A row stored after a drain, at the time of the last row delivered, is
// delivered by the next drain whichever way its id sorts, and only once.
func TestStream_ARowAtTheLastDeliveredTimeIsDeliveredByTheNextDrain(t *testing.T) {
	at := time.Now()
	store := &rows{}
	c := opened(store)
	store.add(row{id: "m", at: at})
	require.Equal(t, []string{"m"}, deliverOnce(t, c))

	store.add(row{id: "z", at: at})
	assert.Equal(t, []string{"z"}, deliverOnce(t, c), "an id sorting after the delivered row")
	store.add(row{id: "a", at: at})
	assert.Equal(t, []string{"a"}, deliverOnce(t, c), "an id sorting before the delivered rows")
	assert.Empty(t, deliverOnce(t, c), "a drain with nothing new delivers nothing")
}

// A row committed after a later-stamped one has streamed, stamped within the
// overlap behind it, is delivered; one stamped further back is not.
func TestStream_ARowCommittedLateWithinTheOverlapIsDelivered(t *testing.T) {
	at := time.Now()
	store := &rows{}
	c := opened(store)
	store.add(row{id: "newer", at: at})
	require.Equal(t, []string{"newer"}, deliverOnce(t, c))

	store.add(row{id: "late", at: at.Add(-testOverlap / 2)}, row{id: "too-late", at: at.Add(-2 * testOverlap)})

	assert.Equal(t, []string{"late"}, deliverOnce(t, c))
	assert.Empty(t, deliverOnce(t, c))
}

// A row stored again at a later time, as an event moved forward is, is
// delivered again, even behind a newer row already delivered.
func TestStream_ARowStoredAgainIsDeliveredAgain(t *testing.T) {
	at := time.Now()
	store := &rows{}
	c := opened(store)
	store.add(row{id: "moved", at: at}, row{id: "newer", at: at.Add(testOverlap / 2)})
	require.Equal(t, []string{"moved", "newer"}, deliverOnce(t, c))

	store.add(row{id: "moved", at: at.Add(testOverlap / 4)})

	assert.Equal(t, []string{"moved"}, deliverOnce(t, c))
}

// The rows within the overlap of the newest one when the stream opened are
// history, also when they fill several pages; a row stored later at their
// time is not.
func TestStream_TheHistoryWithinTheOverlapIsNotReplayed(t *testing.T) {
	at := time.Now()
	store := &rows{}
	store.add(row{id: "older", at: at.Add(-testOverlap / 2)})
	store.add(tied("hist", at, testBurst)...)
	c := opened(store)

	store.add(row{id: "hist-100a", at: at}, row{id: "a-new", at: at})

	assert.Equal(t, []string{"a-new", "hist-100a"}, deliverOnce(t, c))
}

// Tied rows filling several pages are delivered once each, and a row later
// stored at their time is delivered alone.
func TestStream_TiedRowsFillingSeveralPagesAreEachDeliveredOnce(t *testing.T) {
	at := time.Now()
	store := &rows{}
	c := opened(store)
	burst := tied("tie", at, testBurst)
	store.add(burst...)

	assert.Equal(t, ids(burst), deliverOnce(t, c))
	store.add(row{id: "tie-100a", at: at})
	assert.Equal(t, []string{"tie-100a"}, deliverOnce(t, c))
}

// The cursor remembers only the rows its next read still reaches.
func TestStream_TheCursorForgetsTheRowsBehindTheOverlap(t *testing.T) {
	at := time.Now()
	store := &rows{}
	c := opened(store)
	store.add(row{id: "old", at: at})
	require.Equal(t, []string{"old"}, deliverOnce(t, c))

	store.add(row{id: "new", at: at.Add(2 * testOverlap)})
	require.Equal(t, []string{"new"}, deliverOnce(t, c))

	assert.Equal(t, map[string]time.Time{"new": at.Add(2 * testOverlap)}, c.seen)
}

// Repeated wakes over rows sharing one time deliver each row once.
func TestStream_RepeatedWakesDeliverEachTiedRowOnce(t *testing.T) {
	at := time.Now()
	store := &rows{}
	wake := streamwake.NewSignal()
	ctx, cancel := testutil.TestContextWithCancel()
	defer cancel()

	ch := Stream(ctx, testWakeInterval, testBuffer, wokenSource(store, wake))
	require.Eventually(t, store.readAtLeast(1), testWindow, testInterval, "the baseline read")
	store.store(row{id: "a", at: at}, row{id: "b", at: at})
	wake.Notify()
	require.Eventually(t, store.readAtLeast(2), testWindow, testInterval, "the first woken read")
	store.store(row{id: "c", at: at})
	for range 3 {
		wake.Notify()
	}

	assert.Equal(t, []string{"a", "b", "c"}, drain(ch))
}
