package federation

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/federation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// errDBDown is the injected repository failure the lifecycle tests assert on.
var errDBDown = errors.New("db down")

// blockingInstallRepo lets tests hold the installation lookup open until
// released, to exercise the starting state.
type blockingInstallRepo struct {
	relayInstallRepo
	gate    chan struct{}
	getErr  error
	lookups atomic.Int32
}

func (r *blockingInstallRepo) Get(ctx context.Context) (*models.CEInstallation, error) {
	r.lookups.Add(1)
	if r.gate != nil {
		select {
		case <-r.gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.inst, nil
}

func newLifecycleClient(endpoint string, repo interfaces.CEInstallationRepository) *RelayClient {
	return NewRelayClient(config2(endpoint), repo, &relayOutboxRepo{}, logger.NewNop())
}

func TestEnsureStarted_ConcurrentCallersShareOneAttempt(t *testing.T) {
	addr := startScriptedIngress(t, &scriptedIngress{authenticate: true})
	repo := &blockingInstallRepo{relayInstallRepo: relayInstallRepo{inst: onboardedInstall()}, gate: make(chan struct{})}
	c := newLifecycleClient(addr, repo)
	defer c.Stop()

	const callers = 8
	var wg sync.WaitGroup
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = c.EnsureStarted(testutil.TestContext())
		}(i)
	}
	time.Sleep(testCallerSettleDelay)
	close(repo.gate)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("caller %d: unexpected error %v", i, err)
		}
	}
	if got := repo.lookups.Load(); got != 1 {
		t.Errorf("one installation lookup expected for the shared attempt, got %d", got)
	}
}

func TestEnsureStarted_ConcurrentCallersShareRepositoryFailure(t *testing.T) {
	cause := errDBDown
	repo := &blockingInstallRepo{gate: make(chan struct{}), getErr: cause}
	c := newLifecycleClient("127.0.0.1:1", repo)

	const callers = 6
	var wg sync.WaitGroup
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = c.EnsureStarted(testutil.TestContext())
		}(i)
	}
	time.Sleep(testCallerSettleDelay)
	close(repo.gate)
	wg.Wait()

	for i, err := range errs {
		if !errors.Is(err, cause) {
			t.Errorf("caller %d: expected the shared repository failure, got %v", i, err)
		}
	}

	// The failure returned the client to idle: a fresh attempt runs again and
	// reports the incomplete onboarding as its typed sentinel.
	repo.getErr = nil
	repo.gate = nil
	repo.inst = nil // onboarding incomplete resolves to the typed sentinel and idle
	if err := c.EnsureStarted(testutil.TestContext()); !errors.Is(err, federation.ErrRelayOnboardingIncomplete) {
		t.Errorf("post-failure attempt must run from idle and report incomplete onboarding, got %v", err)
	}
}

func TestEnsureStarted_StopDuringStartCancelsAllWaiters(t *testing.T) {
	repo := &blockingInstallRepo{relayInstallRepo: relayInstallRepo{inst: onboardedInstall()}, gate: make(chan struct{})}
	c := newLifecycleClient("127.0.0.1:1", repo)

	const callers = 5
	var wg sync.WaitGroup
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = c.EnsureStarted(testutil.TestContext())
		}(i)
	}
	time.Sleep(testCallerSettleDelay)
	c.Stop() // cancels the in-flight lookup; the owner resolves to stopped
	wg.Wait()

	for i, err := range errs {
		if !errors.Is(err, federation.ErrRelayStopped) {
			t.Errorf("caller %d: expected ErrRelayStopped after Stop during start, got %v", i, err)
		}
	}
	if err := c.EnsureStarted(testutil.TestContext()); !errors.Is(err, federation.ErrRelayStopped) {
		t.Errorf("stopped is permanent, got %v", err)
	}
}

func TestEnsureStarted_ConcurrentWithStop(t *testing.T) {
	addr := startScriptedIngress(t, &scriptedIngress{authenticate: true})
	c := newLifecycleClient(addr, &relayInstallRepo{inst: onboardedInstall()})

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := c.EnsureStarted(testutil.TestContext())
			if err != nil && !errors.Is(err, federation.ErrRelayStopped) {
				t.Errorf("EnsureStarted racing Stop may only return nil or ErrRelayStopped, got %v", err)
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.Stop()
	}()
	wg.Wait()
	c.Stop() // repeated Stop is safe
}

func TestStop_BeforeStart(t *testing.T) {
	c := newLifecycleClient("127.0.0.1:1", &relayInstallRepo{inst: onboardedInstall()})
	c.Stop()
	if err := c.EnsureStarted(testutil.TestContext()); !errors.Is(err, federation.ErrRelayStopped) {
		t.Errorf("start after stop must return ErrRelayStopped, got %v", err)
	}
}

func TestStop_RepeatedAfterRunning(t *testing.T) {
	addr := startScriptedIngress(t, &scriptedIngress{authenticate: true})
	c := newLifecycleClient(addr, &relayInstallRepo{inst: onboardedInstall()})
	if err := c.EnsureStarted(testutil.TestContext()); err != nil {
		t.Fatalf("start: %v", err)
	}
	c.Stop()
	c.Stop()
	if err := c.EnsureStarted(testutil.TestContext()); !errors.Is(err, federation.ErrRelayStopped) {
		t.Errorf("restart after stop must return ErrRelayStopped, got %v", err)
	}
}

func TestRevocation_LatchesStoppedWithoutPublicStop(t *testing.T) {
	addr := startScriptedIngress(t, &scriptedIngress{authenticate: true, revoke: true})
	c := newLifecycleClient(addr, &relayInstallRepo{inst: onboardedInstall()})
	if err := c.EnsureStarted(testutil.TestContext()); err != nil {
		t.Fatalf("start: %v", err)
	}

	// The revocation notice must end the loop by itself - no Stop call here.
	deadline := time.After(testTerminationTimeout)
	for {
		c.mu.Lock()
		state := c.state
		c.mu.Unlock()
		if state == relayStopped {
			break
		}
		select {
		case <-deadline:
			t.Fatal("loop did not latch stopped after revocation")
		case <-time.After(testStatePollInterval):
		}
	}
	if err := c.EnsureStarted(testutil.TestContext()); !errors.Is(err, federation.ErrRelayStopped) {
		t.Errorf("revocation is permanent, got %v", err)
	}
	c.Stop() // Stop after a revocation-driven exit must not deadlock
}

// config2 builds the fast-cycling federation config the lifecycle tests use.
func config2(endpoint string) config.FederationConfig {
	return config.FederationConfig{
		Enabled:             testRelayEnabled,
		ECEEndpoint:         endpoint,
		ReconnectMaxBackoff: testReconnectMaxBackoff,
		HeartbeatInterval:   testHeartbeatInterval,
	}
}
