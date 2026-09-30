package bssci

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	// burstPings is how many pings arrive while the first counter write stalls.
	burstPings = 5
	// counterPollInterval paces the wait for the coalesced write.
	counterPollInterval = 10 * time.Millisecond
)

// stalledCounterWrites stalls the first counter write until released and
// records how many writes overlapped and which counters each carried.
type stalledCounterWrites struct {
	SessionService
	entered     chan struct{}
	release     chan struct{}
	mu          sync.Mutex
	calls       int
	inFlight    int
	maxInFlight int
	lastBsOpID  int64
}

func (c *stalledCounterWrites) UpdateSessionCounters(ctx context.Context, session *Session) error {
	c.mu.Lock()
	c.calls++
	c.inFlight++
	c.maxInFlight = max(c.maxInFlight, c.inFlight)
	first := c.calls == 1
	c.mu.Unlock()
	if first {
		c.entered <- struct{}{}
		<-c.release
	}
	lastBsOpID, _ := session.OperationCounters()
	c.mu.Lock()
	c.inFlight--
	c.lastBsOpID = lastBsOpID
	c.mu.Unlock()
	return c.SessionService.UpdateSessionCounters(ctx, session)
}

func (c *stalledCounterWrites) snapshot() (calls, inFlight, maxInFlight int, lastBsOpID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls, c.inFlight, c.maxInFlight, c.lastBsOpID
}

// Frames that arrive while a counter write is in flight start no writer of
// their own: the running one writes the newest counters once it is free.
func TestFrameBurstCoalescesCounterWrites(t *testing.T) {
	spy := &stalledCounterWrites{entered: make(chan struct{}, 1), release: make(chan struct{})}
	h := startInteropServerWithSetup(t, EncodingJSON, func(server *Server) {
		spy.SessionService = server.sessionSvc
		server.sessionSvc = spy
		server.config.StatusRequestInitialDelay = interopIODeadline * interopQuietStatusFactor
	})
	h.writeFrame(connectPayload(mioty.MIOTYProtocolVersion, uint64(TestBsEui01)))
	require.Equal(t, mioty.CmdConnectResponse, frameCommand(h.readFrame()))
	h.writeFrame(map[string]interface{}{"command": mioty.CmdConnectComplete, "opId": int64(0)})

	h.ping(1)
	<-spy.entered
	for opID := int64(2); opID <= burstPings; opID++ {
		h.ping(opID)
	}
	calls, _, _, _ := spy.snapshot()
	assert.Equal(t, 1, calls, "frames arriving during a write start no further writer")

	close(spy.release)
	require.Eventually(t, func() bool {
		_, inFlight, _, lastBsOpID := spy.snapshot()
		return inFlight == 0 && lastBsOpID == burstPings
	}, interopIODeadline, counterPollInterval, "the newest counters are written once the stalled write finished")
	_, _, maxInFlight, _ := spy.snapshot()
	assert.Equal(t, 1, maxInFlight, "one counter write per session is in flight at a time")
}
