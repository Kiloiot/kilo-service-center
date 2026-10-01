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

const (
	supersedingSessionID int64 = 502
	supersedeClockStep         = time.Second
)

var sessionEventOrderStart = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

// openingGate is an event log whose write of one session's opened event
// waits, the way a slow store write does, until it is released.
type openingGate struct {
	*sessionEventLog
	sessionID int64
	once      sync.Once
	entered   chan struct{}
	release   chan struct{}
}

func gateOpening(log *sessionEventLog, sessionID int64) *openingGate {
	return &openingGate{sessionEventLog: log, sessionID: sessionID, entered: make(chan struct{}), release: make(chan struct{})}
}

func (g *openingGate) RecordSessionEvent(ctx context.Context, event *models.SCACISessionEvent) error {
	if event.SessionID == g.sessionID && event.EventType == models.EventTypeSCACISessionOpened {
		g.once.Do(func() {
			close(g.entered)
			<-g.release
		})
	}
	return g.sessionEventLog.RecordSessionEvent(ctx, event)
}

func eventOf(t *testing.T, events []models.SCACISessionEvent, sessionID int64, eventType string) models.SCACISessionEvent {
	t.Helper()
	for _, event := range events {
		if event.SessionID == sessionID && event.EventType == eventType {
			return event
		}
	}
	t.Fatalf("no %s event of session %d in %v", eventType, sessionID, events)
	return models.SCACISessionEvent{}
}

// A session superseded while its opened event is still on its way to the
// log reaches the log after its closed event; the moments they carry still
// put its opening first, and the new session's opening after its close.
func TestSessionEvents_ASessionSupersededWhileItOpensOpensBeforeItCloses(t *testing.T) {
	server, log, handshake := newSessionEventServer()
	clk := testutil.NewFakeClock(sessionEventOrderStart)
	server.clock = clk
	gate := gateOpening(log, freshSessionID)
	server.sessionEvents = gate
	oldConn, _ := pipedTLS(t)
	old := &Session{ID: freshSessionID, TenantID: heldTenant, AcEui: heldAcEui, State: StateConnecting}
	require.True(t, adopts(testutil.TestContext(), server.registry, oldConn, old))

	completed := make(chan error, 1)
	go func() { completed <- server.handleConnectComplete(oldConn, old, OpIDConnect) }()
	<-gate.entered
	clk.Advance(supersedeClockStep)
	next := &Session{ID: supersedingSessionID, TenantID: heldTenant, AcEui: heldAcEui, State: StateConnecting}
	connectAs(t, server, handshake, next)
	close(gate.release)
	require.NoError(t, <-completed)

	events := log.filed()
	require.Len(t, events, 2)
	require.Equal(t, models.EventTypeSCACISessionClosed, events[0].EventType, "the close reaches the log first")
	opened := eventOf(t, events, freshSessionID, models.EventTypeSCACISessionOpened)
	closed := eventOf(t, events, freshSessionID, models.EventTypeSCACISessionClosed)
	assert.True(t, opened.OccurredAt.Equal(sessionEventOrderStart), "opened at the completion of its connect operation")
	assert.True(t, opened.OccurredAt.Before(closed.OccurredAt), "opened %s, closed %s", opened.OccurredAt, closed.OccurredAt)
}

// A connect that completes after a newer session of its application center
// took its connection over never opens: the superseded session does not go
// live and files no opened event (SCACI §1).
func TestHandleConnectComplete_ASessionSupersededBeforeItsConnectCompletesNeverOpens(t *testing.T) {
	server, log, handshake := newSessionEventServer()
	oldConn, _ := pipedTLS(t)
	old := &Session{ID: freshSessionID, TenantID: heldTenant, AcEui: heldAcEui, State: StateConnecting}
	require.True(t, adopts(testutil.TestContext(), server.registry, oldConn, old))
	connectAs(t, server, handshake, &Session{ID: supersedingSessionID, TenantID: heldTenant, AcEui: heldAcEui, State: StateConnecting})
	require.Empty(t, log.filed(), "a session that never went live closes nothing")

	_ = server.handleConnectComplete(oldConn, old, OpIDConnect)

	assert.False(t, old.connected(), "the superseded session does not go live")
	for _, event := range log.filed() {
		assert.NotEqual(t, models.EventTypeSCACISessionOpened, event.EventType)
	}
}

// A lost connection is filed at the moment it was found lost.
func TestEndConnection_FilesTheLossAtItsMoment(t *testing.T) {
	server, log, _ := newSessionEventServer()
	server.clock = testutil.NewFakeClock(sessionEventOrderStart)
	live := &mockConn{}
	server.registry.sessions[live] = &Session{ID: heldSessionID, TenantID: heldTenant, AcEui: heldAcEui, State: StateActive}

	server.endConnection(live)

	require.Len(t, log.filed(), 1)
	assert.True(t, log.filed()[0].OccurredAt.Equal(sessionEventOrderStart))
}
