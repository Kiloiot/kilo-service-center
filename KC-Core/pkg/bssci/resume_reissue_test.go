package bssci

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// countingConn is a net.Conn that records started frames (writes beginning
// with the MIOTY frame identifier) and optionally fails every write, standing
// in for a base-station link that breaks mid-reissue.
type countingConn struct {
	frames    int
	failWrite bool
}

func (c *countingConn) Read(_ []byte) (int, error) { return 0, net.ErrClosed }
func (c *countingConn) Write(b []byte) (int, error) {
	if len(b) >= 8 && bytes.Equal(b[:8], mioty.MIOTYFrameIdentifier[:]) {
		c.frames++
	}
	if c.failWrite {
		return 0, net.ErrClosed
	}
	return len(b), nil
}
func (c *countingConn) Close() error                       { return nil }
func (c *countingConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (c *countingConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (c *countingConn) SetDeadline(_ time.Time) error      { return nil }
func (c *countingConn) SetReadDeadline(_ time.Time) error  { return nil }
func (c *countingConn) SetWriteDeadline(_ time.Time) error { return nil }

// newResumeReissueServer assembles a server the same way the interop harness
// does, without a wire transport: connect-complete is invoked directly on a
// hand-built session carrying a resume snapshot.
func newResumeReissueServer(t *testing.T) *Server {
	t.Helper()
	log := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(log, nil)
	server := NewTestServer(log, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, interopConnectionService{}, broadcaster,
		queueSerializer, auditLogger, tenantResolver)
	server.config = &Config{
		ServiceCenterEUI:      TestScEui01,
		Vendor:                "test-vendor",
		Model:                 "test-model",
		Name:                  "test-sc",
		SoftwareVersion:       "1.0.0",
		MessageEncoding:       EncodingJSON,
		OperationAckTimeout:   2 * time.Second,
		StatusRequestInterval: time.Hour,
	}
	ctx, cancel := context.WithCancel(testutil.TestContext())
	server.ctx = ctx
	server.cancel = cancel
	t.Cleanup(cancel)
	return server
}

// newResumeSession builds a session in the awaiting-connect-complete state
// with a resume snapshot attached, as handleConnect leaves it for a resumed
// base station.
func newResumeSession(conn net.Conn, ops []*PendingOperation) *Session {
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:             "resume-reissue-session",
			BaseStationEUI: TestBsEui01,
			SessionUUID:    make([]byte, 16),
			ConnectState:   ConnectStateAwaitingConnectComplete,
		},
		Conn:               conn,
		pendingBaseStation: &basestation.BaseStation{ID: 1, TenantID: 1, Name: "Resume BS"},
	}
	session.IsResumed = true
	session.resumePendingOps = ops
	return session
}

func stopSessionStatus(session *Session) {
	session.mu.Lock()
	if session.stopStatus != nil {
		close(session.stopStatus)
		session.stopStatus = nil
	}
	session.mu.Unlock()
}

// statusReissueOp is a minimal resumable SC operation (status request) whose
// frame passes outbound validation without reconstitution.
func statusReissueOp(opID int64) *PendingOperation {
	return &PendingOperation{
		OperationID:   opID,
		OperationType: mioty.CmdStatus,
		Message: map[string]interface{}{
			"command": mioty.CmdStatus,
			"opId":    opID,
		},
	}
}

// TestResumeReissueAbortsOnSendFailure: once a reissue send fails the
// connection is broken, so the loop stops instead of attempting the remaining
// operations; their rows stay persisted for the next resume.
func TestResumeReissueAbortsOnSendFailure(t *testing.T) {
	server := newResumeReissueServer(t)
	conn := &countingConn{failWrite: true}
	session := newResumeSession(conn, []*PendingOperation{
		statusReissueOp(-1),
		statusReissueOp(-2),
	})
	t.Cleanup(func() { stopSessionStatus(session) })

	msg := &Message{OpId: 0, Command: mioty.CmdConnectComplete}
	require.NoError(t, server.handleConnectComplete(server, session, msg, nil))

	assert.Equal(t, 1, conn.frames,
		"the first failed send must abort the reissue loop before the second operation")
}

// TestResumeRemovesIrrecoverableOperation: an operation whose payload cannot
// be reconstituted is deleted durably so it does not resurface on every
// future resume, and the loop continues with the remaining operations.
func TestResumeRemovesIrrecoverableOperation(t *testing.T) {
	server := newResumeReissueServer(t)
	statusSvc := server.statusSvc.(*memoryStatusService)
	conn := &countingConn{}
	corrupt := &PendingOperation{
		OperationID:   -3,
		OperationType: mioty.CmdULDataTransmit,
		Message:       map[string]interface{}{"command": mioty.CmdULDataTransmit, "opId": int64(-3)},
		Metadata:      map[string]interface{}{"bogus": true},
	}
	session := newResumeSession(conn, []*PendingOperation{
		corrupt,
		statusReissueOp(-4),
	})
	t.Cleanup(func() { stopSessionStatus(session) })

	msg := &Message{OpId: 0, Command: mioty.CmdConnectComplete}
	require.NoError(t, server.handleConnectComplete(server, session, msg, nil))

	statusSvc.mu.RLock()
	removed := append([]int64(nil), statusSvc.removedOps...)
	statusSvc.mu.RUnlock()
	assert.Equal(t, []int64{-3}, removed,
		"the irrecoverable operation's persisted row must be removed")
	assert.Equal(t, 1, conn.frames,
		"the valid operation after the corrupt one must still be reissued")
}

// TestTeardownEvictsCacheKeepsRows: when an active session's connection is
// lost, its cached operations are swept (the runtime session ID dies with the
// connection) while the persisted rows are preserved for resume.
func TestTeardownEvictsCacheKeepsRows(t *testing.T) {
	h := startInteropServer(t, EncodingJSON)
	h.writeFrame(connectPayload(mioty.MIOTYProtocolVersion, uint64(TestBsEui01)))
	conRsp := h.readFrame()
	require.Equal(t, mioty.CmdConnectResponse, frameCommand(conRsp))
	h.writeFrame(map[string]interface{}{"command": mioty.CmdConnectComplete, "opId": int64(0)})
	h.writeFrame(map[string]interface{}{"command": mioty.CmdPing, "opId": int64(1)})
	require.Equal(t, mioty.CmdPingResponse, frameCommand(h.readFrame()))

	h.server.mu.Lock()
	require.Len(t, h.server.sessions, 1, "the activated session must be live")
	var live *Session
	for _, s := range h.server.sessions {
		live = s
	}
	h.server.mu.Unlock()

	statusSvc := h.server.statusSvc.(*memoryStatusService)
	foreign := &Session{ProtocolSessionState: ProtocolSessionState{ID: "other-session"}}
	require.NoError(t, statusSvc.RecordPendingOperation(testutil.TestContext(), live, -10, statusReissueOp(-10), live.DbSessionID))
	require.NoError(t, statusSvc.RecordPendingOperation(testutil.TestContext(), foreign, -10, statusReissueOp(-10), 99))

	require.NoError(t, h.conn.Close())
	<-h.done

	statusSvc.mu.RLock()
	_, liveCached := (*statusSvc.pendingOps)[SessionOpKey{SessionID: live.ID, OperationID: -10}]
	_, foreignCached := (*statusSvc.pendingOps)[SessionOpKey{SessionID: foreign.ID, OperationID: -10}]
	deleteCalls := statusSvc.deleteSessionCalls
	statusSvc.mu.RUnlock()

	assert.False(t, liveCached, "teardown must evict the dead session's cached operations")
	assert.True(t, foreignCached, "other sessions' cached operations must survive")
	assert.Zero(t, deleteCalls,
		"an active session's persisted rows must be preserved for resume (no durable delete)")
}
