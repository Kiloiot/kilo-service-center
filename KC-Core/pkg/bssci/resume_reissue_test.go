package bssci

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
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
	closes    int
	buf       bytes.Buffer
}

func (c *countingConn) Read(_ []byte) (int, error) { return 0, net.ErrClosed }
func (c *countingConn) Write(b []byte) (int, error) {
	if len(b) >= 8 && bytes.Equal(b[:8], mioty.MIOTYFrameIdentifier[:]) {
		c.frames++
	}
	if c.failWrite {
		return 0, net.ErrClosed
	}
	c.buf.Write(b)
	return len(b), nil
}

// writtenCommands decodes the buffered outbound frames and returns their
// command names in order.
func (c *countingConn) writtenCommands(t *testing.T) []string {
	t.Helper()
	var commands []string
	raw := c.buf.Bytes()
	for len(raw) >= HeaderSize {
		require.True(t, bytes.Equal(raw[:8], mioty.MIOTYFrameIdentifier[:]), "frame identifier")
		payloadLen := int(binary.LittleEndian.Uint32(raw[8:HeaderSize]))
		require.LessOrEqual(t, HeaderSize+payloadLen, len(raw), "complete frame")
		decoded, err := decodeMessage(raw[HeaderSize:HeaderSize+payloadLen], EncodingJSON)
		require.NoError(t, err)
		cmd, _ := decoded["command"].(string)
		commands = append(commands, cmd)
		raw = raw[HeaderSize+payloadLen:]
	}
	return commands
}

func (c *countingConn) Close() error {
	c.closes++
	return nil
}
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

// TestResumeReissueAbortsOnSendFailure: a reissue send failure aborts
// activation - handleConnectComplete errors, the ambiguous-write transport is
// closed, and status polling never starts. The rows stay persisted for the
// next resume.
func TestResumeReissueAbortsOnSendFailure(t *testing.T) {
	server := newResumeReissueServer(t)
	conn := &countingConn{failWrite: true}
	session := newResumeSession(conn, []*PendingOperation{
		statusReissueOp(-1),
		statusReissueOp(-2),
	})
	t.Cleanup(func() { stopSessionStatus(session) })

	msg := &Message{OpId: 0, Command: mioty.CmdConnectComplete}
	err := server.handleConnectComplete(server, session, msg, nil)
	require.Error(t, err, "a failed reissue must abort connect completion")
	require.ErrorIs(t, err, ErrAmbiguousWrite)

	assert.Equal(t, 1, conn.frames,
		"the first failed send must abort the reissue loop before the second operation")
	assert.Positive(t, conn.closes,
		"an ambiguous write must close the transport")

	session.mu.Lock()
	stopStatus := session.stopStatus
	session.mu.Unlock()
	assert.Nil(t, stopStatus, "status polling must not start after an aborted reissue")
}

// malformedResumeRow persists as a valid strict-decodable row whose metadata
// cannot be semantically reconstructed: ulDataTx and dlDataRev fail on the
// missing key/typed fields, dlDataQue on an undecodable payload.
func malformedResumeRow(opID int64, opType string) PersistedOperation {
	metadata := `{"bogus":true}`
	if opType == mioty.CmdDLDataQueue {
		metadata = `{"payloads":["%%%not-base64%%%"]}`
	}
	return PersistedOperation{
		OperationID:   opID,
		OperationType: opType,
		OperationData: []byte(fmt.Sprintf(`{"command":%q,"opId":%d}`, opType, opID)),
		Metadata:      []byte(metadata),
	}
}

// TestResumeRejectedWhenReconstructionFails: a persisted operation that
// cannot be semantically rebuilt rejects the whole resume with EAGAIN before
// conRsp - no row is deleted, no queue state changes, and no session
// activates. Covers all three payload-bearing operation types.
func TestResumeRejectedWhenReconstructionFails(t *testing.T) {
	for _, opType := range []string{mioty.CmdULDataTransmit, mioty.CmdDLDataQueue, mioty.CmdDLDataRevoke} {
		t.Run(opType, func(t *testing.T) {
			server := newResumeReissueServer(t)
			statusSvc := server.statusSvc.(*memoryStatusService)
			sessionSvc := server.sessionSvc.(*mockSessionService)

			// A resumable prior session identified by its snBsUuid
			prevUUID := make([]byte, 16)
			for i := range prevUUID {
				prevUUID[i] = byte(i + 1)
			}
			prev := &Session{ProtocolSessionState: ProtocolSessionState{
				ID:             "previous-runtime-session",
				BaseStationEUI: TestBsEui01,
				SessionUUID:    prevUUID,
				DbSessionID:    7,
				LastScOpId:     -1,
			}}
			sessionSvc.StoreSessionByUUID(prev)

			statusSvc.mu.Lock()
			statusSvc.persistedRows = []PersistedOperation{malformedResumeRow(-2, opType)}
			statusSvc.mu.Unlock()

			conn := &countingConn{}
			session := &Session{
				ProtocolSessionState: ProtocolSessionState{
					ID:             "resume-reject-session",
					BaseStationEUI: TestBsEui01,
					Encoding:       EncodingJSON,
				},
				Conn: conn,
			}

			snBsUUID := make([]interface{}, 16)
			for i := range snBsUUID {
				snBsUUID[i] = int64(i + 1)
			}
			connectData := map[string]interface{}{
				"command":  mioty.CmdConnect,
				"opId":     int64(0),
				"version":  mioty.MIOTYProtocolVersion,
				"bsEui":    int64(TestBsEui01),
				"bidi":     true,
				"snBsUuid": snBsUUID,
			}
			msg := &Message{OpId: 0, Command: mioty.CmdConnect, Data: connectData}
			require.NoError(t, server.handleConnect(server, session, msg, connectData),
				"a rejected connect awaits errorAck instead of failing the handler")

			commands := conn.writtenCommands(t)
			require.Equal(t, []string{mioty.CmdError}, commands,
				"the resume must be rejected with an error frame and never reach conRsp")

			statusSvc.mu.RLock()
			rows := len(statusSvc.persistedRows)
			removed := len(statusSvc.removedOps)
			statusSvc.mu.RUnlock()
			assert.Equal(t, 1, rows, "the malformed row must be preserved for operator inspection")
			assert.Zero(t, removed, "no persisted row may be deleted on a rejected resume")

			server.mu.Lock()
			live := len(server.sessions)
			server.mu.Unlock()
			assert.Zero(t, live, "no session may activate on a rejected resume")
		})
	}
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

// newActivationSession builds a fresh (non-resume) session in the
// awaiting-connect-complete state for the shared test base station, with a
// distinct snScUuid per connection.
func newActivationSession(id string, conn net.Conn) *Session {
	scUUID := make([]byte, 16)
	copy(scUUID, id)
	return &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               id,
			BaseStationEUI:   TestBsEui01,
			SessionUUID:      scUUID,
			Encoding:         EncodingJSON,
			ResolvedTenantID: 1,
			ConnectState:     ConnectStateAwaitingConnectComplete,
		},
		Conn:               conn,
		pendingBaseStation: &basestation.BaseStation{ID: 1, TenantID: 1, Name: "Displacement BS"},
	}
}

// TestActivationDisplacesLiveSessionForSameEUI: a second activation for a base
// station that still holds a live session leaves only the newer session
// reachable by EUI, with the displaced transport closed.
func TestActivationDisplacesLiveSessionForSameEUI(t *testing.T) {
	server := newResumeReissueServer(t)
	// keeps both activations' status goroutines dormant for the whole run
	server.config.StatusRequestInitialDelay = time.Hour

	displacedConn := &countingConn{}
	displaced := newActivationSession("displaced-session", displacedConn)
	t.Cleanup(func() { stopSessionStatus(displaced) })
	require.NoError(t, server.handleConnectComplete(server, displaced,
		&Message{OpId: 0, Command: mioty.CmdConnectComplete}, nil))

	firstLookup, ok := server.GetSessionByEUI(TestBsEui01).(*Session)
	require.True(t, ok, "the first activation must be reachable by EUI")
	require.Equal(t, displaced.ID, firstLookup.ID)

	currentConn := &countingConn{}
	current := newActivationSession("current-session", currentConn)
	t.Cleanup(func() { stopSessionStatus(current) })
	require.NoError(t, server.handleConnectComplete(server, current,
		&Message{OpId: 0, Command: mioty.CmdConnectComplete}, nil))

	server.mu.RLock()
	_, stillLive := server.sessions[displaced.ID]
	liveCount := len(server.sessions)
	server.mu.RUnlock()
	assert.False(t, stillLive, "the displaced session must leave the live map")
	// asserted before the lookup so a single live session makes its result unambiguous
	require.Equal(t, 1, liveCount, "one base station must hold exactly one live session")

	secondLookup, ok := server.GetSessionByEUI(TestBsEui01).(*Session)
	require.True(t, ok, "the newest activation must be reachable by EUI")
	assert.Equal(t, current.ID, secondLookup.ID,
		"a by-EUI lookup must never resolve to the displaced session")
	assert.Positive(t, displacedConn.closes, "the displaced transport must be closed")
	assert.Zero(t, currentConn.closes, "the activating transport must stay open")
}
