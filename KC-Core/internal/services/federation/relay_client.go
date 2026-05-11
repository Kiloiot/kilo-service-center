package federation

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	federationpb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/federation/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

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
	started     atomic.Bool
	cancelRelay context.CancelFunc

	// bsCountFn returns the current number of connected base stations for heartbeats.
	bsCountFn func(context.Context) int32
	// ceVersion is the software version string included in CEHello.
	ceVersion string
}

// NewRelayClient constructs a RelayClient for CE mode.
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

// EnsureStarted starts the relay loop if it has not been started yet.
// It is idempotent and safe to call concurrently.
func (c *RelayClient) EnsureStarted(ctx context.Context) error {
	if c.started.Load() {
		return nil
	}
	return c.Start(ctx)
}

// Start begins the relay loop in a background goroutine.
// It is idempotent; subsequent calls after the first are no-ops.
func (c *RelayClient) Start(ctx context.Context) error {
	if !c.started.CompareAndSwap(false, true) {
		return nil
	}

	inst, err := c.installRepo.Get(ctx)
	if err != nil {
		c.started.Store(false)
		return fmt.Errorf("relay client: cannot read CE installation: %w", err)
	}
	if inst == nil || inst.OnboardingCompletedAt == nil {
		c.logger.InfoContext(ctx, "CE onboarding not complete; federation relay will not start")
		c.started.Store(false)
		return nil
	}
	c.ceID = inst.CEID
	c.company = inst.CompanyName

	relayCtx, cancel := context.WithCancel(ctx)
	c.cancelRelay = cancel
	go c.loop(relayCtx, inst)
	return nil
}

// IsConnected reports whether the relay stream is currently active.
func (c *RelayClient) IsConnected() bool {
	return c.connected.Load()
}

// Stop shuts down the relay loop.
func (c *RelayClient) Stop() {
	if c.cancelRelay != nil {
		c.cancelRelay()
	}
}

// loop implements exponential backoff reconnection.
func (c *RelayClient) loop(ctx context.Context, inst *interfaces.CEInstallation) {
	backoff := time.Second
	maxBackoff := c.cfg.ReconnectMaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = 5 * time.Minute
	}

	for {
		if c.revoked.Load() {
			c.logger.InfoContext(ctx, "CE revoked; relay loop exiting permanently")
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := c.runStream(ctx, inst); err != nil {
			if c.revoked.Load() {
				return
			}
			c.logger.WarnContext(ctx, "Federation relay stream closed; reconnecting",
				"backoff", backoff, "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = min(backoff*2, maxBackoff)
		} else {
			backoff = time.Second // reset on clean close
		}
	}
}

// buildDialCredentials returns the gRPC transport credentials based on the TLS configuration.
func (c *RelayClient) buildDialCredentials(ctx context.Context) (grpc.DialOption, error) {
	tlsCfg := c.cfg.TLS
	if !tlsCfg.Enabled {
		c.logger.WarnContext(ctx, "Federation relay transport is unencrypted; enable TLS for production use")
		return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
	}

	tlsConf := &tls.Config{
		InsecureSkipVerify: tlsCfg.InsecureSkipVerify, //nolint:gosec // operator-controlled flag
	}

	if tlsCfg.CAFile != "" {
		caCert, err := os.ReadFile(tlsCfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA cert from %s", tlsCfg.CAFile)
		}
		tlsConf.RootCAs = pool
	}

	if tlsCfg.CertFile != "" && tlsCfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client cert/key: %w", err)
		}
		tlsConf.Certificates = []tls.Certificate{cert}
	}

	return grpc.WithTransportCredentials(credentials.NewTLS(tlsConf)), nil
}

// runStream opens a single Connect() stream, authenticates, replays the outbox, and
// processes incoming messages until the stream closes.
func (c *RelayClient) runStream(ctx context.Context, inst *interfaces.CEInstallation) error {
	dialOpt, err := c.buildDialCredentials(ctx)
	if err != nil {
		return fmt.Errorf("build dial credentials: %w", err)
	}
	conn, err := grpc.NewClient(c.cfg.ECEEndpoint, dialOpt)
	if err != nil {
		return fmt.Errorf("dial ECE: %w", err)
	}
	defer func() { _ = conn.Close() }()

	client := federationpb.NewFederationServiceClient(conn)
	stream, err := client.Connect(ctx)
	if err != nil {
		return fmt.Errorf("open federation stream: %w", err)
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
		return fmt.Errorf("send CEHello: %w", err)
	}

	// Wait for ECEHello
	hello, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("recv ECEHello: %w", err)
	}
	eceHello := hello.GetHello()
	if eceHello == nil {
		return fmt.Errorf("expected ECEHello, got %T", hello.Payload)
	}
	if !eceHello.Authenticated {
		// If ECE has revoked this CE, mark permanently and stop retrying.
		if strings.Contains(strings.ToLower(eceHello.ErrorMessage), "revoked") {
			c.revoked.Store(true)
		}
		return fmt.Errorf("ECE rejected authentication: %s", eceHello.ErrorMessage)
	}

	// Persist updated token if ECE issued/reissued one
	if eceHello.FederationToken != "" && eceHello.FederationToken != token {
		updates := map[string]interface{}{
			"federation_token": eceHello.FederationToken,
			"token_issued_at":  time.Now(),
		}
		if updateErr := c.installRepo.Update(ctx, updates); updateErr != nil {
			c.logger.WarnContext(ctx, "Failed to persist new federation token", "error", updateErr)
		}
		inst.FederationToken = &eceHello.FederationToken
	}

	c.connected.Store(true)
	defer c.connected.Store(false)
	c.logger.InfoContext(ctx, "Federation relay stream established", "ce_id", c.ceID)

	// Replay the full outbox (pending + sent) on reconnect, then switch to pending-only drain.
	outboxDone := make(chan error, 1)
	go func() { outboxDone <- c.drainOutbox(ctx, stream, inst) }()

	// Process incoming messages (receipts, heartbeat acks, revoke notices)
	for {
		msg, recvErr := stream.Recv()
		if recvErr != nil {
			<-outboxDone
			return recvErr
		}
		switch p := msg.Payload.(type) {
		case *federationpb.ECEToCEMessage_Receipt:
			c.handleReceipt(ctx, p.Receipt)
		case *federationpb.ECEToCEMessage_HeartbeatAck:
			c.logger.DebugContext(ctx, "Heartbeat acknowledged", "acked_at_ns", p.HeartbeatAck.AckedAtNs)
		case *federationpb.ECEToCEMessage_Revoke:
			c.logger.ErrorContext(ctx, "CE instance revoked by ECE; relay permanently halted",
				"reason", p.Revoke.Reason)
			c.revoked.Store(true)
			c.Stop()
			<-outboxDone
			return nil
		}
	}
}

// drainOutbox replays the full outbox on initial connect, then continuously sends new
// pending records and emits heartbeats on each tick.
func (c *RelayClient) drainOutbox(ctx context.Context, stream federationpb.FederationService_ConnectClient, _ *interfaces.CEInstallation) error {
	heartbeatInterval := c.cfg.HeartbeatInterval
	if heartbeatInterval <= 0 {
		heartbeatInterval = 60 * time.Second
	}
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	// On reconnect, replay pending+sent records first to recover any unacknowledged sends.
	reconnectRecords, err := c.outboxRepo.ListPending(ctx, 0)
	if err != nil {
		c.logger.WarnContext(ctx, "Failed to read outbox for reconnect replay", "error", err)
	}
	for _, r := range reconnectRecords {
		if r.Status == interfaces.FederationOutboxStatusAcked || r.Status == interfaces.FederationOutboxStatusRejected {
			continue
		}
		sendErr := stream.Send(&federationpb.CEToECEMessage{
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
		})
		if sendErr != nil {
			return sendErr
		}
		_ = c.outboxRepo.MarkSent(ctx, r.RelayID)
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
			_ = stream.Send(&federationpb.CEToECEMessage{
				Payload: &federationpb.CEToECEMessage_Heartbeat{
					Heartbeat: &federationpb.Heartbeat{
						CeId:          c.ceID.String(),
						SentAtNs:      time.Now().UnixNano(),
						ActiveBsCount: bsCount,
					},
				},
			})
		default:
			records, listErr := c.outboxRepo.ListPendingOnly(ctx, 50)
			if listErr != nil {
				c.logger.WarnContext(ctx, "Failed to read outbox", "error", listErr)
				time.Sleep(time.Second)
				continue
			}
			for _, r := range records {
				sendErr := stream.Send(&federationpb.CEToECEMessage{
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
				})
				if sendErr != nil {
					return sendErr
				}
				_ = c.outboxRepo.MarkSent(ctx, r.RelayID)
			}
			if len(records) == 0 {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// handleReceipt processes an UplinkReceipt from ECE.
func (c *RelayClient) handleReceipt(ctx context.Context, receipt *federationpb.UplinkReceipt) {
	relayID, err := uuid.Parse(receipt.RelayId)
	if err != nil {
		c.logger.WarnContext(ctx, "Invalid relay_id in receipt", "relay_id", receipt.RelayId)
		return
	}
	if receipt.Accepted {
		_ = c.outboxRepo.MarkAcked(ctx, relayID)
	} else {
		_ = c.outboxRepo.MarkRejected(ctx, relayID, receipt.ErrorMessage)
		c.logger.WarnContext(ctx, "Relay rejected by ECE",
			"relay_id", relayID, "error", receipt.ErrorMessage)
	}
}
