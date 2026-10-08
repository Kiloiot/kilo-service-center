package scaci

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// lifecycleRowLog applies the registry's lifecycle writes to one stored
// status, in the order they land; the first resume write waits for gate.
type lifecycleRowLog struct {
	mu      sync.Mutex
	stored  string
	gated   bool
	entered chan struct{}
	gate    chan struct{}
}

func newLifecycleRowLog() *lifecycleRowLog {
	return &lifecycleRowLog{entered: make(chan struct{}), gate: make(chan struct{})}
}

func (l *lifecycleRowLog) PersistResume(context.Context, *Session, string, string) error {
	l.mu.Lock()
	first := !l.gated
	l.gated = true
	l.mu.Unlock()
	if first {
		close(l.entered)
		<-l.gate
	}
	l.store(models.SCACISessionStatusActive)
	return nil
}

func (l *lifecycleRowLog) PersistDisconnect(context.Context, *Session) error {
	l.store(models.SCACISessionStatusDisconnected)
	return nil
}

func (l *lifecycleRowLog) store(status string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stored = status
}

func (l *lifecycleRowLog) status() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stored
}

// A resume whose row write is slow races the next connection of the same
// session: that connection resumes it too and is lost. The slow write must
// not land after the loss and leave the stored session active without a
// connection (SCACI §1).
func TestSessionRegistry_DisconnectRacingAResumeNeverLeavesTheSessionActive(t *testing.T) {
	ctx := testutil.TestContext()
	rows := newLifecycleRowLog()
	registry := newTestRegistryWithRows(nil, newHolderFake(), rows)
	registry.holder.Hold(ctx, heldSessionOf(heldSessionID, heldTenant, heldAcEui))

	slow, slowConn := resumingSession(), &mockConn{}
	require.True(t, adopts(ctx, registry, slowConn, slow))
	slowDone := make(chan error, 1)
	go func() { slowDone <- registry.complete(ctx, slowConn, slow, time.Now(), "", "") }()
	<-rows.entered

	next, nextConn := resumingSession(), &mockConn{}
	nextDone := make(chan struct{})
	go func() {
		defer close(nextDone)
		adopts(ctx, registry, nextConn, next)
		assert.NoError(t, registry.complete(ctx, nextConn, next, time.Now(), "", ""))
		registry.release(ctx, nextConn, SessionPersistTimeout)
	}()
	select {
	case <-nextDone:
	case <-time.After(asyncSettleDelay):
	}
	close(rows.gate)
	<-slowDone
	<-nextDone

	assert.Equal(t, models.SCACISessionStatusDisconnected, rows.status(),
		"no connection serves the session, so its row is not active")
	assert.Empty(t, registry.sessions)
}

// The lifecycle lock of a session ID exists only while it is in use.
func TestSessionLocks_ForgetAnUnusedID(t *testing.T) {
	var locks sessionLocks
	unlock := locks.lock(heldSessionID)
	assert.Len(t, locks.locks, 1)
	unlock()
	assert.Empty(t, locks.locks)
}

func TestNewSessionRegistry_RefusesAMissingCollaborator(t *testing.T) {
	for name, build := range map[string]func() (*SessionRegistry, error){
		"holder": func() (*SessionRegistry, error) { return NewSessionRegistry(nil, noRowWrites{}, testLogger()) },
		"rows":   func() (*SessionRegistry, error) { return NewSessionRegistry(newHolderFake(), nil, testLogger()) },
		"logger": func() (*SessionRegistry, error) { return NewSessionRegistry(newHolderFake(), noRowWrites{}, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			_, err := build()
			assert.ErrorIs(t, err, errMissingRegistryDependency)
		})
	}
}
