package bssci

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// testReconcileWait and testReconcilePoll bound the wait for the asynchronous reconciliation.
const (
	testReconcileWait = 2 * time.Second
	testReconcilePoll = 10 * time.Millisecond
)

// stationReconciler records the stations it is asked to reconcile.
type stationReconciler struct {
	noopPropagationService
	mu       sync.Mutex
	stations []propagation.BaseStationSession
}

func (r *stationReconciler) ReconcileBaseStation(_ context.Context, station propagation.BaseStationSession, _ *models.BaseStation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stations = append(r.stations, station)
	return nil
}

func (r *stationReconciler) reconciled() []propagation.BaseStationSession {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]propagation.BaseStationSession(nil), r.stations...)
}

// A resumed station keeps the session it had, so the resume takes over the
// time its previous connection was lost (BSSCI §1).
func TestResumeTakesOverThePreviousDisconnectTime(t *testing.T) {
	server := newResumeReissueServer(t)
	disconnectedAt := time.Unix(1_800_000_000, 0)
	prevUUID := make([]byte, 16)
	snBsUUID := make([]interface{}, 16)
	for i := range prevUUID {
		prevUUID[i] = byte(i + 1)
		snBsUUID[i] = int64(i + 1)
	}
	server.sessionSvc.(*mockSessionService).StoreSessionByUUID(&Session{ProtocolSessionState: ProtocolSessionState{
		ID: "previous-runtime-session", BaseStationEUI: TestBsEui01, SessionUUID: prevUUID, DbSessionID: 7,
		DisconnectedAt: &disconnectedAt,
	}})
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{ID: "resuming-connection", BaseStationEUI: TestBsEui01, Encoding: EncodingJSON},
		Conn:                 &testutil.TestConn{Encoding: EncodingJSON},
	}
	connectData := map[string]interface{}{
		"command": mioty.CmdConnect, "opId": int64(0), "version": mioty.MIOTYProtocolVersion,
		"bsEui": int64(TestBsEui01), "bidi": true, "snBsUuid": snBsUUID,
	}

	require.NoError(t, server.handleConnect(session, &Message{OpId: 0, Command: mioty.CmdConnect, Data: connectData}, connectData))

	require.True(t, session.IsResumed)
	require.NotNil(t, session.DisconnectedAt, "the resumed session knows when its connection was lost")
	assert.True(t, disconnectedAt.Equal(*session.DisconnectedAt))
}

// Reconciliation is told whether the station resumed its session and since
// when it was away, so it can send the detachments the station missed; a new
// session is reconciled as one that holds nothing.
func TestConnectCompleteTellsReconciliationHowTheStationReconnected(t *testing.T) {
	disconnectedAt := time.Unix(1_800_000_000, 0)
	resumed := newResumeSession(&countingConn{}, nil)
	resumed.DisconnectedAt = &disconnectedAt
	fresh := newActivationSession("fresh-session", &countingConn{})

	for name, tc := range map[string]struct {
		session        *Session
		resumed        bool
		disconnectedAt *time.Time
	}{
		"resumed session": {session: resumed, resumed: true, disconnectedAt: &disconnectedAt},
		"new session":     {session: fresh},
	} {
		t.Run(name, func(t *testing.T) {
			server := newResumeReissueServer(t)
			reconciler := &stationReconciler{}
			server.propagationSvc = reconciler
			t.Cleanup(func() { stopSessionStatus(tc.session) })

			require.NoError(t, server.handleConnectComplete(tc.session, &Message{Command: mioty.CmdConnectComplete}, nil))

			require.Eventually(t, func() bool { return len(reconciler.reconciled()) == 1 }, testReconcileWait, testReconcilePoll)
			station := reconciler.reconciled()[0]
			assert.Equal(t, tc.resumed, station.Resumed)
			assert.Equal(t, tc.disconnectedAt, station.DisconnectedAt)
		})
	}
}
