// Package federation holds the outbound relay client a community installation
// uses to reach an enterprise ingress. It lives outside the federation service
// package so that package carries no gRPC transport.
package federation

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/federation"

	federationpb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/federation/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// relayState is the lifecycle position of the relay client.
type relayState int

const (
	// relayIdle: no loop is running and a start attempt may begin.
	relayIdle relayState = iota
	// relayStarting: one caller owns an in-flight start attempt; others wait on it.
	relayStarting
	// relayRunning: the relay loop is active.
	relayRunning
	// relayStopped: permanently stopped by Stop or revocation; never restarts.
	relayStopped
)

// startAttempt carries one start attempt's shared outcome to every caller that
// arrived while it was in flight.
type startAttempt struct {
	done chan struct{}
	err  error
}

// RelayClient manages the persistent CE→ECE federation stream.
// It connects to the ECE federation-ingress, sends CEHello, relays outbox records,
// and processes UplinkReceipts and control messages from ECE.
type RelayClient struct {
	cfg         config.FederationConfig
	installRepo interfaces.CEInstallationRepository
	outboxRepo  interfaces.FederationOutboxRepository
	logger      logger.Logger
	ceID        uuid.UUID
	company     string
	connected   atomic.Bool
	revoked     atomic.Bool

	// mu guards the lifecycle state, the lifecycle cancel function, the loop
	// completion channel, and the in-flight start attempt.
	mu      sync.Mutex
	state   relayState
	cancel  context.CancelFunc
	done    chan struct{}
	attempt *startAttempt

	// bsCountFn returns the current number of connected base stations for heartbeats.
	bsCountFn func(context.Context) int32
	// ceVersion is the software version string included in CEHello.
	ceVersion string
}

const backoffGrowthFactor = 2

// revokedErrorText, matched case-insensitively in an ECE hello rejection,
// marks this CE as permanently revoked.
const revokedErrorText = "revoked"

// outboxDrainBatchSize bounds each outbox drain read.
const outboxDrainBatchSize = 50

// NewRelayClient constructs a RelayClient for CE mode.
// backoffGrowthFactor doubles the reconnect backoff up to its cap.
func NewRelayClient(
	cfg config.FederationConfig,
	installRepo interfaces.CEInstallationRepository,
	outboxRepo interfaces.FederationOutboxRepository,
	log logger.Logger,
) *RelayClient {
	return &RelayClient{
		cfg:         cfg,
		installRepo: installRepo,
		outboxRepo:  outboxRepo,
		logger:      log,
	}
}

// WithBsCountFn injects a function that returns the active base station count for heartbeat messages.
func (c *RelayClient) WithBsCountFn(fn func(context.Context) int32) *RelayClient {
	c.bsCountFn = fn
	return c
}

// WithCEVersion sets the software version string sent in CEHello.
func (c *RelayClient) WithCEVersion(v string) *RelayClient {
	c.ceVersion = v
	return c
}

// EnsureStarted starts the relay loop if it is not already running. It is safe
// to call concurrently: the first caller to find the client idle owns the
// start attempt, callers arriving while that attempt is in flight wait for it
// and share its result, a running client returns nil immediately, and a
// stopped client permanently returns federation.ErrRelayStopped.
//
// An attempt that fails to read the installation, or finds onboarding
// incomplete, returns the client to idle so a later call can try again.
func (c *RelayClient) EnsureStarted(ctx context.Context) error {
	c.mu.Lock()
	switch c.state {
	case relayRunning:
		c.mu.Unlock()
		return nil
	case relayStopped:
		c.mu.Unlock()
		return federation.ErrRelayStopped
	case relayStarting:
		att := c.attempt
		c.mu.Unlock()
		<-att.done
		return att.err
	case relayIdle:
	}

	// This caller owns the attempt. The lifecycle context is created before
	// the installation lookup so a concurrent Stop can cancel the lookup, and
	// it is detached from the caller's context because the loop must outlive
	// request-scoped callers (CompleteCEOnboarding): Stop() is the only
	// shutdown path.
	att := &startAttempt{done: make(chan struct{})}
	lifecycleCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	c.state = relayStarting
	c.cancel = cancel
	c.attempt = att
	c.mu.Unlock()

	att.err = c.resolveStartAttempt(lifecycleCtx, cancel)
	close(att.done)
	return att.err
}

// resolveStartAttempt performs the installation lookup for an owned start
// attempt and either launches the relay loop or returns the client to its
// terminal or idle state.
func (c *RelayClient) resolveStartAttempt(lifecycleCtx context.Context, cancel context.CancelFunc) error {
	inst, err := c.installRepo.Get(lifecycleCtx)

	c.mu.Lock()
	defer c.mu.Unlock()

	// A Stop that arrived during the lookup latched the terminal state and
	// already cancelled the lifecycle context; every waiter sees the stop.
	if c.state == relayStopped {
		return federation.ErrRelayStopped
	}

	if err != nil {
		c.state = relayIdle
		cancel()
		return fmt.Errorf("%w: %w", ErrRelayInstallationRead, err)
	}
	if inst == nil || inst.OnboardingCompletedAt == nil {
		c.logger.InfoContext(lifecycleCtx, LogOnboardingIncomplete)
		c.state = relayIdle
		cancel()
		return federation.ErrRelayOnboardingIncomplete
	}

	c.ceID = inst.CEID
	c.company = inst.CompanyName
	c.state = relayRunning
	done := make(chan struct{})
	c.done = done
	go func() {
		defer func() {
			// The loop only exits on Stop or revocation; both are permanent,
			// so the lifecycle context is released here (Stop cancels it
			// itself; revocation would otherwise leak it).
			cancel()
			c.mu.Lock()
			c.state = relayStopped
			c.mu.Unlock()
			close(done)
		}()
		c.loop(lifecycleCtx, inst)
	}()
	return nil
}

// IsConnected reports whether the relay stream is currently active.
func (c *RelayClient) IsConnected() bool {
	return c.connected.Load()
}

// Stop permanently shuts down the relay client. It cancels an in-flight start
// attempt or the running loop, waits for a running loop to finish, and leaves
// the client in the stopped state; it is safe to call repeatedly and
// concurrently.
func (c *RelayClient) Stop() {
	c.mu.Lock()
	c.state = relayStopped
	cancel := c.cancel
	done := c.done
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// loop implements exponential backoff reconnection.
func (c *RelayClient) loop(ctx context.Context, inst *models.CEInstallation) {
	backoff := time.Second
	maxBackoff := c.cfg.ReconnectMaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = time.Duration(config.DefaultFederationReconnectMaxBackoffSeconds) * time.Second
	}

	for {
		if c.revoked.Load() {
			c.logger.InfoContext(ctx, LogLoopRevokedExit)
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}

		established, err := c.runStream(ctx, inst)
		if established {
			// A stream that authenticated was a successful connection no
			// matter how it ended - a normal termination arrives as a receive
			// error, so resetting only on a nil return kept an escalated
			// delay after every real connection.
			backoff = time.Second
		}
		if errors.Is(err, federation.ErrRelayRevoked) {
			c.logger.InfoContext(ctx, LogLoopRevokedExit)
			return
		}
		if err != nil {
			if c.revoked.Load() {
				return
			}
			c.logger.WarnContext(ctx, LogStreamClosedReconnecting,
				logger.FieldBackoff, backoff, logger.FieldError, err)
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			backoff = min(backoff*backoffGrowthFactor, maxBackoff)
		}
	}
}

// buildDialCredentials returns the gRPC transport credentials based on the TLS configuration.
func (c *RelayClient) buildDialCredentials(ctx context.Context) (grpc.DialOption, error) {
	tlsCfg := c.cfg.TLS
	if !tlsCfg.Enabled {
		c.logger.WarnContext(ctx, LogTransportUnencrypted)
		return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
	}

	tlsConf := &tls.Config{
		InsecureSkipVerify: tlsCfg.InsecureSkipVerify, //nolint:gosec // operator-controlled flag
	}

	if tlsCfg.CAFile != "" {
		caCert, err := os.ReadFile(tlsCfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrCACertRead, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("%w: %s", federation.ErrRelayCACertParse, tlsCfg.CAFile)
		}
		tlsConf.RootCAs = pool
	}

	if tlsCfg.CertFile != "" && tlsCfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrClientCertLoad, err)
		}
		tlsConf.Certificates = []tls.Certificate{cert}
	}

	return grpc.WithTransportCredentials(credentials.NewTLS(tlsConf)), nil
}

// runStream opens a single Connect() stream, authenticates, replays the outbox,
// and processes incoming messages until the stream closes. The first return
// value reports whether a stream was established (authenticated); the caller
// uses it to reset reconnect backoff regardless of how the stream later ended.
func (c *RelayClient) runStream(ctx context.Context, inst *models.CEInstallation) (bool, error) {
	dialOpt, err := c.buildDialCredentials(ctx)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrDialCredentials, err)
	}
	conn, err := grpc.NewClient(c.cfg.ECEEndpoint, dialOpt)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrDialECE, err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			c.logger.WarnContext(ctx, LogRelayConnCloseFailed, logger.FieldError, closeErr)
		}
	}()

	// Everything on this stream lives under one per-attempt context, so a
	// failure on either side - receive or send - tears the whole stream down
	// instead of leaving the sibling blocked indefinitely.
	streamCtx, cancelStream := context.WithCancel(ctx)
	defer cancelStream()

	client := federationpb.NewFederationServiceClient(conn)
	stream, err := client.Connect(streamCtx)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrStreamOpen, err)
	}

	// Send CEHello
	token := ""
	if inst.FederationToken != nil {
		token = *inst.FederationToken
	}

	var activeBsCount int32
	if c.bsCountFn != nil {
		activeBsCount = c.bsCountFn(ctx)
	}

	if err := stream.Send(&federationpb.CEToECEMessage{
		Payload: &federationpb.CEToECEMessage_Hello{
			Hello: &federationpb.CEHello{
				CeId:            c.ceID.String(),
				FederationToken: token,
				CompanyName:     c.company,
				ActiveBsCount:   activeBsCount,
				CeVersion:       c.ceVersion,
			},
		},
	}); err != nil {
		return false, fmt.Errorf("%w: %w", ErrHelloSend, err)
	}

	// Wait for ECEHello
	hello, err := stream.Recv()
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrHelloRecv, err)
	}
	eceHello := hello.GetHello()
	if eceHello == nil {
		return false, fmt.Errorf("%w %T", ErrUnexpectedHelloPayload, hello.Payload)
	}
	if !eceHello.Authenticated {
		// If ECE has revoked this CE, mark permanently and stop retrying.
		if strings.Contains(strings.ToLower(eceHello.ErrorMessage), revokedErrorText) {
			c.revoked.Store(true)
		}
		return false, fmt.Errorf("%w: %s", ErrAuthRejected, eceHello.ErrorMessage)
	}

	// Persist updated token if ECE issued/reissued one
	if eceHello.FederationToken != "" && eceHello.FederationToken != token {
		updates := map[string]interface{}{
			"federation_token": eceHello.FederationToken,
			"token_issued_at":  time.Now(),
		}
		if updateErr := c.installRepo.Update(ctx, updates); updateErr != nil {
			c.logger.WarnContext(ctx, federation.LogRelayTokenPersistFailed, logger.FieldError, updateErr)
		}
		inst.FederationToken = &eceHello.FederationToken
	}

	c.connected.Store(true)
	defer c.connected.Store(false)
	c.logger.InfoContext(ctx, LogStreamEstablished, logger.FieldCeID, c.ceID)

	// Receive processing and outbox draining run as siblings. Whichever
	// terminates first ends the stream: the per-stream context is cancelled
	// immediately, which unblocks the other side, and both are waited for
	// before the connection closes.
	done := make(chan error, 2)
	go func() { done <- c.drainOutbox(streamCtx, stream, inst) }()
	go func() { done <- c.receiveLoop(streamCtx, stream) }()

	firstErr := <-done
	cancelStream()
	<-done

	return true, firstErr
}

// receiveLoop processes incoming messages (receipts, heartbeat acks, revoke
// notices) until the stream ends.
func (c *RelayClient) receiveLoop(ctx context.Context, stream federationpb.FederationService_ConnectClient) error {
	for {
		msg, recvErr := stream.Recv()
		if recvErr != nil {
			return recvErr
		}
		switch p := msg.Payload.(type) {
		case *federationpb.ECEToCEMessage_Receipt:
			c.handleReceipt(ctx, p.Receipt)
		case *federationpb.ECEToCEMessage_HeartbeatAck:
			c.logger.DebugContext(ctx, LogHeartbeatAcked, logger.FieldAckedAtNs, p.HeartbeatAck.AckedAtNs)
		case *federationpb.ECEToCEMessage_Revoke:
			// The sentinel travels back to the reconnect loop, which exits and
			// lets the lifecycle goroutine latch the stopped state. Calling the
			// public blocking Stop from inside the loop's own call tree would
			// deadlock on the loop-completion wait.
			c.logger.ErrorContext(ctx, LogRevokedByECE,
				logger.FieldReason, p.Revoke.Reason)
			c.revoked.Store(true)
			return federation.ErrRelayRevoked
		}
	}
}

// sleepCtx waits for d or until ctx is done, reporting whether the full wait
// elapsed. Used instead of time.Sleep so outbox pacing cannot outlive the
// stream it paces.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// drainOutbox replays the full outbox on initial connect, then continuously sends new
// pending records and emits heartbeats on each tick.
func (c *RelayClient) drainOutbox(ctx context.Context, stream federationpb.FederationService_ConnectClient, _ *models.CEInstallation) error {
	heartbeatInterval := c.cfg.HeartbeatInterval
	if heartbeatInterval <= 0 {
		heartbeatInterval = time.Duration(config.DefaultFederationHeartbeatIntervalSeconds) * time.Second
	}
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	// On reconnect, replay pending+sent records first to recover any unacknowledged sends.
	reconnectRecords, err := c.outboxRepo.ListPending(ctx, 0)
	if err != nil {
		c.logger.WarnContext(ctx, federation.LogRelayOutboxReplayReadFailed, logger.FieldError, err)
	}
	for _, r := range reconnectRecords {
		if r.Status == models.FederationOutboxStatusAcked || r.Status == models.FederationOutboxStatusRejected {
			continue
		}
		if err := c.relayRecord(ctx, stream, r); err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			var bsCount int32
			if c.bsCountFn != nil {
				bsCount = c.bsCountFn(ctx)
			}
			if hbErr := stream.Send(&federationpb.CEToECEMessage{
				Payload: &federationpb.CEToECEMessage_Heartbeat{
					Heartbeat: &federationpb.Heartbeat{
						CeId:          c.ceID.String(),
						SentAtNs:      time.Now().UnixNano(),
						ActiveBsCount: bsCount,
					},
				},
			}); hbErr != nil {
				// A heartbeat that cannot be written means the stream is gone;
				// treat it as a stream failure so reconnection starts now
				// rather than at the next uplink.
				return fmt.Errorf("%w: %w", ErrHeartbeatSend, hbErr)
			}
		default:
			records, listErr := c.outboxRepo.ListPendingOnly(ctx, outboxDrainBatchSize)
			if listErr != nil {
				c.logger.WarnContext(ctx, federation.LogRelayOutboxReadFailed, logger.FieldError, listErr)
				if !sleepCtx(ctx, time.Second) {
					return ctx.Err()
				}
				continue
			}
			for _, r := range records {
				if err := c.relayRecord(ctx, stream, r); err != nil {
					return err
				}
			}
			if len(records) == 0 {
				if !sleepCtx(ctx, outboxIdlePollInterval) {
					return ctx.Err()
				}
			}
		}
	}
}

// handleReceipt processes an UplinkReceipt from ECE.
func (c *RelayClient) handleReceipt(ctx context.Context, receipt *federationpb.UplinkReceipt) {
	relayID, err := uuid.Parse(receipt.RelayId)
	if err != nil {
		c.logger.WarnContext(ctx, LogInvalidReceiptRelayID, logger.FieldRelayID, receipt.RelayId)
		return
	}
	if receipt.Accepted {
		if err := c.outboxRepo.MarkAcked(ctx, relayID); err != nil {
			c.logger.WarnContext(ctx, LogRelayReceiptNotRecorded, logger.FieldRelayID, relayID, logger.FieldError, err)
		}
		return
	}
	c.logger.WarnContext(ctx, LogRelayRejected,
		logger.FieldRelayID, relayID, logger.FieldError, receipt.ErrorMessage)
	if err := c.outboxRepo.MarkRejected(ctx, relayID, receipt.ErrorMessage); err != nil {
		c.logger.WarnContext(ctx, LogRelayReceiptNotRecorded, logger.FieldRelayID, relayID, logger.FieldError, err)
	}
}

// relayRecord writes one outbox record to the stream and marks it sent. A
// record left unmarked is resent on every drain pass, so a failed mark ends
// the stream and reconnection backs off.
func (c *RelayClient) relayRecord(ctx context.Context, stream federationpb.FederationService_ConnectClient, r *models.FederationOutboxRecord) error {
	if err := stream.Send(&federationpb.CEToECEMessage{
		Payload: &federationpb.CEToECEMessage_Uplink{
			Uplink: &federationpb.RelayedUplink{
				RelayId:       r.RelayID.String(),
				CeId:          c.ceID.String(),
				EpEui:         uint64(r.EpEUI), //nolint:gosec // G115: EUI fits uint64
				BsEui:         uint64(r.BsEUI), //nolint:gosec // G115: EUI fits uint64
				RawBssciFrame: r.RawFrame,
				ReceivedAtNs:  r.ReceivedAtNs,
			},
		},
	}); err != nil {
		return err
	}
	if err := c.outboxRepo.MarkSent(ctx, r.RelayID); err != nil {
		return fmt.Errorf("%w: %w", ErrOutboxMarkSent, err)
	}
	return nil
}
