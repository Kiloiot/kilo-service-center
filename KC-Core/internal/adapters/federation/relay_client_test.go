package federation

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/federation"

	"google.golang.org/grpc"

	federationpb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/federation/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// Timing and configuration knobs shared by the relay tests. Short cycles keep
// the reconnect and lifecycle tests fast while staying well clear of races.
const (
	testRelayEnabled        = true
	testReconnectMaxBackoff = 2 * time.Second
	testHeartbeatInterval   = 50 * time.Millisecond
	testStreamDropDelay     = 100 * time.Millisecond
	testQuickDropDelay      = 50 * time.Millisecond
	testTerminationTimeout  = 5 * time.Second
	testEstablishWait       = 200 * time.Millisecond
	testConnectPollInterval = 50 * time.Millisecond
	testCallerSettleDelay   = 100 * time.Millisecond
	testStatePollInterval   = 20 * time.Millisecond
)

// scriptedIngress is an in-process federation server whose behaviour each test
// controls: authenticate or reject, hold the stream open, drop it, or revoke.
type scriptedIngress struct {
	federationpb.UnimplementedFederationServiceServer

	authenticate bool
	revoke       bool
	dropAfter    time.Duration // close the stream this long after hello; 0 = hold open
	connects     atomic.Int32
}

func (s *scriptedIngress) Connect(stream federationpb.FederationService_ConnectServer) error {
	s.connects.Add(1)
	if _, err := stream.Recv(); err != nil { // CEHello
		return err
	}
	if err := stream.Send(&federationpb.ECEToCEMessage{
		Payload: &federationpb.ECEToCEMessage_Hello{
			Hello: &federationpb.ECEHello{Authenticated: s.authenticate, ErrorMessage: map[bool]string{false: "nope"}[s.authenticate]},
		},
	}); err != nil {
		return err
	}
	if !s.authenticate {
		return nil
	}
	if s.revoke {
		return stream.Send(&federationpb.ECEToCEMessage{
			Payload: &federationpb.ECEToCEMessage_Revoke{Revoke: &federationpb.RevokeNotice{Reason: "test"}},
		})
	}
	if s.dropAfter > 0 {
		time.Sleep(s.dropAfter)
		return nil // stream ends -> client sees a receive error
	}
	<-stream.Context().Done()
	return nil
}

func startScriptedIngress(t *testing.T, s *scriptedIngress) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	federationpb.RegisterFederationServiceServer(srv, s)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	return lis.Addr().String()
}

// relayInstallRepo serves one onboarded installation.
type relayInstallRepo struct {
	interfaces.CEInstallationRepository
	inst *models.CEInstallation
}

func (r *relayInstallRepo) Get(context.Context) (*models.CEInstallation, error) {
	return r.inst, nil
}
func (r *relayInstallRepo) Update(context.Context, map[string]interface{}) error { return nil }

// relayOutboxRepo is an empty outbox.
type relayOutboxRepo struct {
	interfaces.FederationOutboxRepository
}

func (r *relayOutboxRepo) ListPending(context.Context, int) ([]*models.FederationOutboxRecord, error) {
	return nil, nil
}
func (r *relayOutboxRepo) ListPendingOnly(context.Context, int) ([]*models.FederationOutboxRecord, error) {
	return nil, nil
}
func (r *relayOutboxRepo) MarkSent(context.Context, uuid.UUID) error { return nil }

func onboardedInstall() *models.CEInstallation {
	now := time.Now()
	tok := "token"
	return &models.CEInstallation{
		CEID:                  uuid.New(),
		CompanyName:           "Test CE",
		OnboardingCompletedAt: &now,
		FederationToken:       &tok,
	}
}

func newTestRelayClient(endpoint string) *RelayClient {
	return NewRelayClient(config.FederationConfig{
		Enabled:             testRelayEnabled,
		ECEEndpoint:         endpoint,
		ReconnectMaxBackoff: testReconnectMaxBackoff,
		HeartbeatInterval:   testHeartbeatInterval,
	}, &relayInstallRepo{inst: onboardedInstall()}, &relayOutboxRepo{}, logger.NewNop())
}

// TestRunStream_InitialConnectionFailure: a dead endpoint reports
// established=false, so the caller keeps escalating backoff.
func TestRunStream_InitialConnectionFailure(t *testing.T) {
	c := newTestRelayClient("127.0.0.1:1") // nothing listens here
	established, err := c.runStream(testutil.TestContext(), onboardedInstall())
	if established {
		t.Error("a failed dial must not report an established stream")
	}
	if err == nil {
		t.Error("a failed dial must report an error")
	}
}

// TestRunStream_RejectedAuthenticationNotEstablished: an authentication
// rejection is not an established stream either.
func TestRunStream_RejectedAuthenticationNotEstablished(t *testing.T) {
	addr := startScriptedIngress(t, &scriptedIngress{authenticate: false})
	c := newTestRelayClient(addr)
	established, err := c.runStream(testutil.TestContext(), onboardedInstall())
	if established || err == nil {
		t.Errorf("rejected auth must be (false, err); got (%v, %v)", established, err)
	}
}

// TestRunStream_EstablishedThenDropped: the normal termination - a stream that
// authenticates and later ends with a receive error - must still report
// established=true, which is what resets reconnect backoff. Both goroutines
// must have terminated by the time runStream returns.
func TestRunStream_EstablishedThenDropped(t *testing.T) {
	addr := startScriptedIngress(t, &scriptedIngress{authenticate: true, dropAfter: testStreamDropDelay})
	c := newTestRelayClient(addr)

	done := make(chan struct{})
	var established bool
	var err error
	go func() {
		established, err = c.runStream(testutil.TestContext(), onboardedInstall())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(testTerminationTimeout):
		t.Fatal("runStream did not terminate after the stream dropped - a goroutine is stuck")
	}
	if !established {
		t.Error("a stream that authenticated must report established=true however it ended")
	}
	if err == nil {
		t.Error("the drop must surface as an error so the loop reconnects")
	}
	if c.IsConnected() {
		t.Error("connected must clear when the stream ends")
	}
}

// TestRunStream_Revocation: a revoke notice halts the relay permanently and
// still tears both goroutines down.
func TestRunStream_Revocation(t *testing.T) {
	addr := startScriptedIngress(t, &scriptedIngress{authenticate: true, revoke: true})
	c := newTestRelayClient(addr)
	ctx, cancel := context.WithCancel(testutil.TestContext())
	defer cancel()

	done := make(chan struct{})
	var established bool
	var err error
	go func() {
		established, err = c.runStream(ctx, onboardedInstall())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(testTerminationTimeout):
		t.Fatal("runStream did not terminate after revocation")
	}
	if !established || !errors.Is(err, federation.ErrRelayRevoked) {
		t.Errorf("revocation is (true, ErrRelayRevoked); got (%v, %v)", established, err)
	}
	if !c.revoked.Load() {
		t.Error("revocation must be recorded permanently")
	}
}

// TestRunStream_ParentCancellation: cancelling the relay context ends the
// stream and both goroutines.
func TestRunStream_ParentCancellation(t *testing.T) {
	addr := startScriptedIngress(t, &scriptedIngress{authenticate: true})
	c := newTestRelayClient(addr)
	ctx, cancel := context.WithCancel(testutil.TestContext())

	done := make(chan struct{})
	go func() {
		_, _ = c.runStream(ctx, onboardedInstall())
		close(done)
	}()

	// Let it establish, then cancel the parent.
	time.Sleep(testEstablishWait)
	cancel()

	select {
	case <-done:
	case <-time.After(testTerminationTimeout):
		t.Fatal("runStream did not terminate on parent cancellation")
	}
}

// TestLoop_ReconnectsAtInitialDelayAfterEstablishedStream is the regression
// lock for the backoff defect: after a successfully established stream drops,
// the next reconnect must come quickly (initial delay), not after an escalated
// backoff carried over from before the connection.
func TestLoop_ReconnectsAtInitialDelayAfterEstablishedStream(t *testing.T) {
	ingress := &scriptedIngress{authenticate: true, dropAfter: testQuickDropDelay}
	addr := startScriptedIngress(t, ingress)
	c := newTestRelayClient(addr)

	ctx, cancel := context.WithCancel(testutil.TestContext())
	defer cancel()

	go c.loop(ctx, onboardedInstall())

	// Every cycle is ~50ms of stream plus ~1s of initial backoff. Three
	// connects inside five seconds are impossible if the delay doubles from an
	// escalated value instead of resetting after each established stream.
	deadline := time.After(testTerminationTimeout)
	for ingress.connects.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("only %d connects in 5s - backoff did not reset after established streams", ingress.connects.Load())
		case <-time.After(testConnectPollInterval):
		}
	}
}
