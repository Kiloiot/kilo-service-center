package bssci

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	// connectDispatchQueueID is the pending downlink the connecting station serves.
	connectDispatchQueueID = uint64(7_300_000_000_000_401)
	// connectQuietWindow is how long nothing more may happen once the connect settled.
	connectQuietWindow = 200 * time.Millisecond
)

// connectSteps records what a connecting station was sent, in order.
type connectSteps struct {
	mu    sync.Mutex
	steps []string
}

func (c *connectSteps) add(step string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.steps = append(c.steps, step)
}

func (c *connectSteps) recorded() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.steps...)
}

// stepReconciler records the reconciliation of the connecting station.
type stepReconciler struct {
	noopPropagationService
	steps *connectSteps
}

func (r stepReconciler) ReconcileBaseStation(context.Context, propagation.BaseStationSession, *models.BaseStation) error {
	r.steps.add("reconcile")
	return nil
}

// stepDispatcher records every exact dispatch.
type stepDispatcher struct {
	noopDownlinkDispatcher
	steps *connectSteps
}

func (d stepDispatcher) DispatchQueue(_ context.Context, _ int64, _ uuid.UUID, session *Session, queueID, _ uint64) (bool, error) {
	d.steps.add(fmt.Sprintf("dispatch %d to %x", queueID, session.BaseStationEUI))
	return true, nil
}

// onePendingDownlink lists one downlink of the test endpoint.
type onePendingDownlink struct {
	listed *atomic.Int32
}

func (p onePendingDownlink) ListPendingDownlinks(context.Context) ([]storage.PendingDownlink, error) {
	p.listed.Add(1)
	return []storage.PendingDownlink{{QueID: connectDispatchQueueID, TenantID: 1, OrganizationID: uuid.New(), EpEUI: TestEpEui01}}, nil
}

// servedBy names the one station serving every endpoint.
type servedBy uint64

func (s servedBy) ServingStation(context.Context, int64, uint64) (uint64, bool, error) {
	return uint64(s), true, nil
}

// A station that connects, with a new session or a resumed one, is first sent
// the attachments it lacks and then the pending downlinks it serves, so each
// dlDataQue reaches it behind the attPrp for its endpoint. A unidirectional
// station cannot transmit downlinks and is sent none.
func TestConnectCompleteSendsTheServedDownlinksAfterTheReconciliation(t *testing.T) {
	served := fmt.Sprintf("dispatch %d to %x", connectDispatchQueueID, TestBsEui01)
	for name, tc := range map[string]struct {
		session func() *Session
		want    []string
		listed  int32
	}{
		"new session":     {func() *Session { return bidirectional(newActivationSession("new-session", &countingConn{})) }, []string{"reconcile", served}, 1},
		"resumed session": {func() *Session { return bidirectional(newResumeSession(&countingConn{}, nil)) }, []string{"reconcile", served}, 1},
		"unidirectional":  {func() *Session { return newActivationSession("receive-only", &countingConn{}) }, []string{"reconcile"}, 0},
	} {
		t.Run(name, func(t *testing.T) {
			server := newResumeReissueServer(t)
			steps := &connectSteps{}
			listed := &atomic.Int32{}
			server.propagationSvc = stepReconciler{steps: steps}
			server.downlinkDispatcher = stepDispatcher{steps: steps}
			server.pendingDownlinks = onePendingDownlink{listed: listed}
			server.servingStations = servedBy(TestBsEui01)
			session := tc.session()
			t.Cleanup(func() { stopSessionStatus(session) })

			require.NoError(t, server.handleConnectComplete(session, &Message{Command: mioty.CmdConnectComplete}, nil))

			require.Eventually(t, func() bool { return len(steps.recorded()) == len(tc.want) }, testReconcileWait, testReconcilePoll)
			assert.Equal(t, tc.want, steps.recorded())
			assert.Never(t, func() bool { return listed.Load() != tc.listed }, connectQuietWindow, testReconcilePoll)
		})
	}
}

// A downlink of an endpoint that another station serves, or that no station
// has heard or attached yet, is not sent to the connecting station.
func TestConnectCompleteLeavesOtherStationsDownlinksPending(t *testing.T) {
	for name, locator := range map[string]ServingStationLocator{
		"served by another station": servedBy(TestBsEui01 + 1),
		"no serving station yet":    unknownServingStation{},
	} {
		t.Run(name, func(t *testing.T) {
			server := newResumeReissueServer(t)
			steps := &connectSteps{}
			listed := &atomic.Int32{}
			server.downlinkDispatcher = stepDispatcher{steps: steps}
			server.pendingDownlinks = onePendingDownlink{listed: listed}
			server.servingStations = locator
			session := bidirectional(newActivationSession("other-stations-endpoint", &countingConn{}))
			t.Cleanup(func() { stopSessionStatus(session) })

			require.NoError(t, server.handleConnectComplete(session, &Message{Command: mioty.CmdConnectComplete}, nil))

			require.Eventually(t, func() bool { return listed.Load() == 1 }, testReconcileWait, testReconcilePoll)
			assert.Never(t, func() bool { return len(steps.recorded()) > 0 }, connectQuietWindow, testReconcilePoll)
		})
	}
}

// holdingReconciler holds the reconciliation until the test releases it.
type holdingReconciler struct {
	noopPropagationService
	steps       *connectSteps
	reconciling chan struct{}
	release     chan struct{}
}

func (r holdingReconciler) ReconcileBaseStation(context.Context, propagation.BaseStationSession, *models.BaseStation) error {
	r.steps.add("reconcile")
	close(r.reconciling)
	<-r.release
	return nil
}

// laterServing knows no serving station until one is set.
type laterServing struct {
	station atomic.Uint64
}

func (l *laterServing) ServingStation(context.Context, int64, uint64) (uint64, bool, error) {
	station := l.station.Load()
	return station, station != 0, nil
}

// An endpoint the station comes to serve only after its connect completed -
// an uplink it reported during the reconciliation opened a downlink window -
// is left to that window's dispatch: the station is sent only the downlinks
// it served when it connected, so the window never gets a second downlink
// ahead of time (radio §3.6.1).
func TestConnectCompleteLeavesALaterDownlinkWindowToItsDispatch(t *testing.T) {
	server := newResumeReissueServer(t)
	steps := &connectSteps{}
	reconciler := holdingReconciler{steps: steps, reconciling: make(chan struct{}), release: make(chan struct{})}
	server.propagationSvc = reconciler
	server.downlinkDispatcher = stepDispatcher{steps: steps}
	server.pendingDownlinks = onePendingDownlink{listed: &atomic.Int32{}}
	serving := &laterServing{}
	server.servingStations = serving
	session := bidirectional(newActivationSession("heard-after-connect", &countingConn{}))
	t.Cleanup(func() { stopSessionStatus(session) })

	require.NoError(t, server.handleConnectComplete(session, &Message{Command: mioty.CmdConnectComplete}, nil))
	<-reconciler.reconciling
	serving.station.Store(TestBsEui01)
	close(reconciler.release)

	assert.Never(t, func() bool { return len(steps.recorded()) > 1 }, connectQuietWindow, testReconcilePoll)
	assert.Equal(t, []string{"reconcile"}, steps.recorded())
}

// unknownServingStation knows no station for any endpoint.
type unknownServingStation struct{}

func (unknownServingStation) ServingStation(context.Context, int64, uint64) (uint64, bool, error) {
	return 0, false, nil
}

func bidirectional(session *Session) *Session {
	session.Bidirectional = true
	return session
}
