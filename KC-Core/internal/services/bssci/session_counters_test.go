package bssciservices

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	repodoubles "github.com/Kiloiot/kilo-service-center/KC-Core/internal/testsupport/repodoubles"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// counterWriteBound is how long a test waits before declaring a counter write
// held back by another one.
const counterWriteBound = 2 * time.Second

// landedCounters records counter writes in the order they complete; the first
// write stalls until released, like a slow database round trip.
type landedCounters struct {
	*repodoubles.BaseStationSessionRepo
	mu      sync.Mutex
	calls   int
	entered chan struct{}
	release chan struct{}
	landed  [][2]int64
}

func (l *landedCounters) UpdateOperationIDs(_ context.Context, _, _ int64, bsOpID, scOpID int64) error {
	l.mu.Lock()
	l.calls++
	first := l.calls == 1
	l.mu.Unlock()
	if first {
		l.entered <- struct{}{}
		<-l.release
	}
	l.mu.Lock()
	l.landed = append(l.landed, [2]int64{bsOpID, scOpID})
	l.mu.Unlock()
	return nil
}

// A slow counter write does not hold back a later one of the same session:
// the store only moves counters forward, so the snapshots need no ordering
// and each write carries the counters current when it started.
func TestUpdateSessionCountersDoesNotWaitForAnEarlierWrite(t *testing.T) {
	store := &landedCounters{
		BaseStationSessionRepo: repodoubles.NewBaseStationSessionRepo(),
		entered:                make(chan struct{}, 1),
		release:                make(chan struct{}),
	}
	svc := NewSessionService(store, newMockPendingOpsStore(),
		&repodoubles.SystemEventStore{}, 1, bssci.TestScEui01, logger.NewNop())
	session := &bssci.Session{ProtocolSessionState: bssci.ProtocolSessionState{DbSessionID: 5, ResolvedTenantID: 1, LastBsOpId: 1, LastScOpId: -1}}
	ctx := testutil.TestContext()

	stalled := make(chan error, 1)
	go func() { stalled <- svc.UpdateSessionCounters(ctx, session) }()
	<-store.entered

	session.NextScOpID()
	later := make(chan error, 1)
	go func() { later <- svc.UpdateSessionCounters(ctx, session) }()
	select {
	case err := <-later:
		require.NoError(t, err)
	case <-time.After(counterWriteBound):
		close(store.release)
		t.Fatal("a stalled counter write held back the next write of its session")
	}
	close(store.release)
	require.NoError(t, <-stalled)

	require.Len(t, store.landed, 2)
	assert.Equal(t, [2]int64{1, -2}, store.landed[0], "the later write lands with the newer counters")
	assert.Equal(t, [2]int64{1, -1}, store.landed[1], "the stalled write keeps the counters it started with")
}
