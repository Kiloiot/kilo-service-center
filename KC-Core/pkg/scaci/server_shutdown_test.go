package scaci

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	shutdownTestSessionID = int64(5)
	shutdownTestTenantID  = int64(1)
	shutdownTestServer    = "scaci.shutdown.test"
	shutdownSettleDelay   = 100 * time.Millisecond
	shutdownWaitTimeout   = 5 * time.Second
)

// blockingPendingRepo holds the resume replay inside its pending-operations
// lookup until released.
type blockingPendingRepo struct {
	recordingOperationRepo
	entered chan struct{}
	release chan struct{}
}

func (r *blockingPendingRepo) GetPendingOperations(context.Context, int64) ([]*models.SCACIOperation, error) {
	close(r.entered)
	<-r.release
	return nil, nil
}

// withLifecycle gives a directly built server the lifecycle Stop drives.
func withLifecycle(s *Server) *Server {
	s.shutdown = make(chan struct{})
	s.ctx, s.cancel = testutil.TestContextWithCancel()
	s.persistCtx, s.persistCancel = context.WithCancel(context.WithoutCancel(s.ctx))
	return s
}

// unhandshakenTLS is a TLS connection whose handshake never runs; the
// connect-complete handler only reads its (empty) connection state.
func unhandshakenTLS(t *testing.T) *tls.Conn {
	t.Helper()
	clientSide, serverSide := net.Pipe()
	t.Cleanup(func() {
		_ = clientSide.Close()
		_ = serverSide.Close()
	})
	return tls.Client(serverSide, &tls.Config{ServerName: shutdownTestServer, MinVersion: tls.VersionTLS12})
}

// The resume replay started by conCmp is server work: Stop waits for it.
func TestStop_WaitsForResumeReplay(t *testing.T) {
	repo := &blockingPendingRepo{entered: make(chan struct{}), release: make(chan struct{})}
	server := withLifecycle(newTestServerWithSessionRepo(new(mockSessionRepository)))
	server.operationRepo = repo
	session := &Session{ID: shutdownTestSessionID, TenantID: shutdownTestTenantID, State: StateConnecting, Resumed: true}
	server.registry.holder.Hold(testutil.TestContext(), &Session{ID: shutdownTestSessionID, TenantID: shutdownTestTenantID})
	conn := unhandshakenTLS(t)
	require.True(t, adopts(testutil.TestContext(), server.registry, conn, session))

	require.NoError(t, server.handleConnectComplete(conn, session, OpIDConnect))
	<-repo.entered

	stopped := make(chan struct{})
	go func() {
		_ = server.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
		t.Fatal("Stop returned while the resume replay was still running")
	case <-time.After(shutdownSettleDelay):
	}

	close(repo.release)
	select {
	case <-stopped:
	case <-time.After(shutdownWaitTimeout):
		t.Fatal("Stop did not return after the replay finished")
	}
}

// Heartbeat writes run detached from the connection but not from the server:
// Stop drains them.
func TestStop_DrainsHeartbeatWrite(t *testing.T) {
	heartbeats := &heartbeatFake{entered: make(chan struct{}), release: make(chan struct{})}
	server := withLifecycle(newTestServerWithSessionRepo(new(mockSessionRepository)))
	server.sessionPersistence = heartbeats
	session := &Session{ID: shutdownTestSessionID, TenantID: shutdownTestTenantID, State: StateActive}

	require.NoError(t, server.handlePingComplete(&mockConn{}, session, shutdownTestSessionID))
	<-heartbeats.entered

	stopped := make(chan struct{})
	go func() {
		_ = server.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
		t.Fatal("Stop returned while a heartbeat write was in flight")
	case <-time.After(shutdownSettleDelay):
	}
	close(heartbeats.release)
	<-stopped
	assert.Equal(t, []int64{shutdownTestSessionID}, heartbeats.heartbeats())
}

// A write requested after Stop drained the task group is refused rather than
// racing the drain.
func TestTaskGroup_RefusesTasksAfterClose(t *testing.T) {
	var group taskGroup
	require.True(t, group.closeAndWait(shutdownSettleDelay))

	ran := make(chan struct{})
	group.start(func() { close(ran) })

	select {
	case <-ran:
		t.Fatal("a closed task group started a task")
	case <-time.After(shutdownSettleDelay):
	}
}
