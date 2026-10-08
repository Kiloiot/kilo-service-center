package scaci

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// Session row writes the ordering test observes.
const (
	rowWriteResumed      = "resumed"
	rowWriteDisconnected = "disconnected"
)

// sessionRowLog notes, in order, what the registry writes to a session row;
// the resume write waits for resumeGate.
type sessionRowLog struct {
	mu         sync.Mutex
	writes     []string
	resumeGate chan struct{}
}

func (l *sessionRowLog) note(write string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writes = append(l.writes, write)
}

func (l *sessionRowLog) PersistResume(context.Context, *Session, string, string) error {
	<-l.resumeGate
	l.note(rowWriteResumed)
	return nil
}

func (l *sessionRowLog) PersistDisconnect(context.Context, *Session) error {
	l.note(rowWriteDisconnected)
	return nil
}

func (l *sessionRowLog) recorded() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.writes...)
}

// A connection lost right after its resume completes leaves the row
// disconnected, never active without a connection: the resume is recorded
// before the handler reads on, so the loss is recorded after it (SCACI §1).
func TestResume_LostConnectionIsRecordedAfterTheResume(t *testing.T) {
	rows := &sessionRowLog{resumeGate: make(chan struct{})}
	server := withLifecycle(newHeldTestServer(newHolderFake()))
	server.registry.rows = rows
	server.registry.holder.Hold(testutil.TestContext(), heldSessionOf(heldSessionID, heldTenant, heldAcEui))
	conn, _ := pipedTLS(t)
	resumed := &Session{ID: heldSessionID, TenantID: heldTenant, AcEui: heldAcEui, State: StateConnecting, Resumed: true}
	require.True(t, adopts(testutil.TestContext(), server.registry, conn, resumed))

	done := make(chan error, 1)
	go func() { done <- server.handleConnectComplete(conn, resumed, OpIDConnect) }()
	returned := false
	select {
	case <-done:
		returned = true
	case <-time.After(shutdownSettleDelay):
		close(rows.resumeGate)
		<-done
	}
	server.registry.release(testutil.TestContext(), conn, SessionPersistTimeout)
	if returned {
		close(rows.resumeGate)
	}
	require.NoError(t, server.Stop())

	assert.Equal(t, []string{rowWriteResumed, rowWriteDisconnected}, rows.recorded())
}
