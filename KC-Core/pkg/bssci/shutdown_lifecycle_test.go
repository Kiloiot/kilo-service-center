package bssci

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// disconnectContextSpy reports the context state each session was handed
// back resumable on.
type disconnectContextSpy struct {
	SessionService
	markedOn chan error
}

func (d *disconnectContextSpy) MarkDisconnected(ctx context.Context, session *Session) error {
	d.markedOn <- ctx.Err()
	return d.SessionService.MarkDisconnected(ctx, session)
}

// A session lost while the server stops is still handed back resumable: the
// teardown persistence outlives the server context.
func TestTeardownDuringStopRecordsResumableSession(t *testing.T) {
	spy := &disconnectContextSpy{markedOn: make(chan error, 1)}
	h := startInteropServerWithSetup(t, EncodingJSON, func(server *Server) {
		spy.SessionService = server.sessionSvc
		server.sessionSvc = spy
		server.config.StatusRequestInitialDelay = interopIODeadline * interopQuietStatusFactor
	})
	h.writeFrame(connectPayload(mioty.MIOTYProtocolVersion, uint64(TestBsEui01)))
	require.Equal(t, mioty.CmdConnectResponse, frameCommand(h.readFrame()))
	h.writeFrame(map[string]interface{}{"command": mioty.CmdConnectComplete, "opId": int64(0)})
	h.ping(1)

	h.server.cancel()
	require.NoError(t, h.conn.Close())
	<-h.done

	select {
	case ctxErr := <-spy.markedOn:
		assert.NoError(t, ctxErr, "the session must be marked disconnected on a live context")
	case <-time.After(interopIODeadline):
		t.Fatal("the lost session was never handed back resumable")
	}
}

// counterWriteSpy reports the context of every counter write made after the
// server stopped.
type counterWriteSpy struct {
	SessionService
	serverStopped atomic.Bool
	mu            sync.Mutex
	afterStop     []error
}

func (c *counterWriteSpy) UpdateSessionCounters(ctx context.Context, session *Session) error {
	if c.serverStopped.Load() {
		c.mu.Lock()
		c.afterStop = append(c.afterStop, ctx.Err())
		c.mu.Unlock()
	}
	return c.SessionService.UpdateSessionCounters(ctx, session)
}

// A lost session persists its final operation counters with its teardown, so
// a resume continues from them even when the server was stopping.
func TestTeardownPersistsFinalCounters(t *testing.T) {
	spy := &counterWriteSpy{}
	h := startInteropServerWithSetup(t, EncodingJSON, func(server *Server) {
		spy.SessionService = server.sessionSvc
		server.sessionSvc = spy
		server.config.StatusRequestInitialDelay = interopIODeadline * interopQuietStatusFactor
	})
	h.writeFrame(connectPayload(mioty.MIOTYProtocolVersion, uint64(TestBsEui01)))
	require.Equal(t, mioty.CmdConnectResponse, frameCommand(h.readFrame()))
	h.writeFrame(map[string]interface{}{"command": mioty.CmdConnectComplete, "opId": int64(0)})
	h.ping(1)

	h.server.cancel()
	spy.serverStopped.Store(true)
	require.NoError(t, h.conn.Close())
	<-h.done
	require.NoError(t, h.server.Stop())

	spy.mu.Lock()
	defer spy.mu.Unlock()
	assert.Contains(t, spy.afterStop, nil, "the teardown writes the final counters on a live context")
}

// blockingReconciler holds the attach-propagate reconciliation until the
// server context ends and records that it returned.
type blockingReconciler struct {
	noopPropagationService
	entered  chan struct{}
	returned atomic.Bool
}

func (b *blockingReconciler) ReconcileBaseStation(ctx context.Context, _ propagation.BaseStationSession, _ *models.BaseStation) error {
	b.entered <- struct{}{}
	<-ctx.Done()
	time.Sleep(interopShortTimeout)
	b.returned.Store(true)
	return ctx.Err()
}

// Stop waits for the background work a connection started.
func TestStopWaitsForConnectionBackgroundWork(t *testing.T) {
	reconciler := &blockingReconciler{entered: make(chan struct{}, 1)}
	h := startInteropServerWithSetup(t, EncodingJSON, func(server *Server) {
		server.propagationSvc = reconciler
		server.config.StatusRequestInitialDelay = interopIODeadline * interopQuietStatusFactor
	})
	h.writeFrame(connectPayload(mioty.MIOTYProtocolVersion, uint64(TestBsEui01)))
	require.Equal(t, mioty.CmdConnectResponse, frameCommand(h.readFrame()))
	h.writeFrame(map[string]interface{}{"command": mioty.CmdConnectComplete, "opId": int64(0)})
	<-reconciler.entered

	require.NoError(t, h.conn.Close())
	<-h.done
	require.NoError(t, h.server.Stop())

	assert.True(t, reconciler.returned.Load(), "Stop returned before the reconciliation it started")
}

// Stopping the server closes a connection whose base station stays quiet, so
// its handler returns instead of holding Stop.
func TestServerCancelClosesQuietConnection(t *testing.T) {
	h := startActiveInteropSession(t)
	h.ping(1)
	// The synchronous pipe accepts a partial frame header only once the
	// handler reads, which then waits on the quiet peer for the rest.
	require.NoError(t, h.conn.SetWriteDeadline(time.Now().Add(interopIODeadline)))
	_, err := h.conn.Write(mioty.MIOTYFrameIdentifier[:1])
	require.NoError(t, err)

	h.server.cancel()
	select {
	case <-h.done:
	case <-time.After(interopIODeadline):
		t.Fatal("a quiet connection held its handler after the server stopped")
	}
}

// A peer that opens TCP and never starts TLS is bounded by the connection
// establishment timeout instead of holding its handler forever.
func TestTLSHandshakeBoundedByEstablishmentTimeout(t *testing.T) {
	log := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, storage := CreateTestServices(log, nil)
	server := NewTestServer(log, storage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, interopConnectionService{}, broadcaster,
		queueSerializer, auditLogger, tenantResolver)
	server.config = &Config{ConnectionEstablishmentTimeout: interopShortTimeout}

	silentPeer, serverSide := net.Pipe()
	t.Cleanup(func() {
		_ = silentPeer.Close()
		_ = serverSide.Close()
	})

	done := make(chan struct{})
	server.wg.Add(1)
	go func() {
		defer close(done)
		server.handleConnection(tls.Server(serverSide, &tls.Config{MinVersion: tls.VersionTLS12}))
	}()

	select {
	case <-done:
	case <-time.After(interopIODeadline):
		t.Fatal("the TLS handshake of a silent peer outlived the establishment timeout")
	}
}
