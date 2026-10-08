package streampoll

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/streamwake"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// testWakeInterval is a poll interval no test waits out, so only a wake reads.
const testWakeInterval = time.Minute

func wokenSource(store *rows, wake Waker) Source[row] {
	src := store.source()
	src.Wake = wake
	return src
}

// A wake reads at once: the row stored after the stream opened arrives long
// before the poll interval.
func TestStream_AWakeDeliversTheStoredRowBeforeTheInterval(t *testing.T) {
	now := time.Now()
	store := &rows{}
	wake := streamwake.NewSignal()
	ctx, cancel := testutil.TestContextWithCancel()
	defer cancel()

	ch := Stream(ctx, testWakeInterval, testBuffer, wokenSource(store, wake))
	require.Eventually(t, store.readAtLeast(1), testWindow, testInterval, "the baseline read")
	store.store(row{id: "stored", at: now})
	wake.Notify()

	assert.Equal(t, []string{"stored"}, drain(ch), "the wake reads the row within the window")
}

// A notification that arrives while a woken read runs wakes the stream again,
// so a row stored during that read does not wait for the interval.
func TestStream_ANotificationDuringAReadWakesTheStreamAgain(t *testing.T) {
	now := time.Now()
	store := &rows{}
	wake := streamwake.NewSignal()
	store.onRead = func(read int) {
		if read == 2 {
			store.add(row{id: "during", at: now.Add(time.Second)})
			wake.Notify()
		}
	}
	ctx, cancel := testutil.TestContextWithCancel()
	defer cancel()

	ch := Stream(ctx, testWakeInterval, testBuffer, wokenSource(store, wake))
	require.Eventually(t, store.readAtLeast(1), testWindow, testInterval, "the baseline read")
	store.store(row{id: "before", at: now})
	wake.Notify()

	assert.Equal(t, []string{"before", "during"}, drain(ch))
}
