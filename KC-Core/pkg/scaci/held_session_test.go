package scaci

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// Sessions of the held-session tests: two Application Centers of heldTenant.
const (
	heldTenant        = int64(61)
	heldSessionID     = int64(610)
	heldOtherACID     = int64(611)
	heldAcEui         = uint64(0x70B3D59CD0000B01)
	heldOtherAcEui    = uint64(0x70B3D59CD0000B02)
	heldLastIssued    = int64(-4)
	heldACQueID       = uint64(77)
	heldInternalQueID = int64(7_300_000_000_000_077)
)

var errHeldTestStore = errors.New("operation log unavailable")

// heldRecord is one operation the holder was asked to record for a session.
type heldRecord struct {
	sessionID int64
	command   string
}

// holderFake stands in for the resume holder: it keeps the held sessions by
// ID, releases what a session supersedes and records under each session's
// next SC opId, noting what it recorded anew.
type holderFake struct {
	mu       sync.Mutex
	held     map[int64]*Session
	records  []heldRecord
	failWith error
}

func newHolderFake() *holderFake { return &holderFake{held: make(map[int64]*Session)} }

func (h *holderFake) Hold(_ context.Context, session *Session) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, held := h.held[session.ID]; !held && session.ID > 0 {
		h.held[session.ID] = session
	}
}

func (h *holderFake) Release(_ context.Context, next *Session) (*Session, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	var resumed *Session
	for id, session := range h.held {
		if session.SupersededBy(next) {
			delete(h.held, id)
			if id == next.ID {
				resumed = session
			}
		}
	}
	return resumed, resumed != nil
}

func (h *holderFake) Record(_ context.Context, tenantID int64, reaches func(*Session) bool, command string, record OperationRecord) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.failWith != nil {
		return h.failWith
	}
	for id, session := range h.held {
		if session.TenantID != tenantID || !reaches(session) {
			continue
		}
		recording, err := record(session, session.NextScOpId())
		if err != nil {
			return err
		}
		if recording == RecordedNew {
			h.records = append(h.records, heldRecord{sessionID: id, command: command})
		}
	}
	return nil
}

func (h *holderFake) recorded() []heldRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]heldRecord(nil), h.records...)
}

func (h *holderFake) holds(id int64) (*Session, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	session, held := h.held[id]
	return session, held
}

// pendingLog serves each session's pending operations back, the way the
// operation log does.
type pendingLog struct {
	mockOperationRepoStub
	pending map[int64][]*models.SCACIOperation
}

func (l *pendingLog) GetPendingOperations(_ context.Context, sessionID int64) ([]*models.SCACIOperation, error) {
	return l.pending[sessionID], nil
}

func (l *pendingLog) UpdateOperationState(context.Context, int64, int64, models.OperationState, map[string]interface{}) error {
	return nil
}

func newHeldTestServer(holder *holderFake) *Server {
	return &Server{
		registry:           newTestRegistry(make(map[net.Conn]*Session), holder),
		codec:              testFrameCodec,
		commands:           mustTestCommandRegistry(),
		clock:              clock.SystemClock{},
		logger:             logger.NewNop(),
		config:             &Config{},
		sessionPersistence: sessionRowsOver{repo: acceptingCounterStore{}},
		operationRepo:      &pendingLog{},
		operationRecorder:  newOperationLedger(),
	}
}

// loseConnection serves a connected session and then drops its connection,
// the way a lost TCP connection ends (SCACI §1).
func loseConnection(server *Server, session *Session) {
	conn := &mockConn{}
	session.State = StateActive
	server.registry.sessions[conn] = session
	server.registry.release(testutil.TestContext(), conn, SessionPersistTimeout)
}

func heldSessionOf(id, tenantID int64, acEui uint64) *Session {
	return withOpIDs(&Session{ID: id, TenantID: tenantID, AcEui: acEui}, OpIDPair{SC: heldLastIssued})
}

func resumingSession() *Session {
	return &Session{ID: heldSessionID, TenantID: heldTenant, AcEui: heldAcEui, State: StateConnecting, Resumed: true}
}

func broadcastHeldUplink(t *testing.T, server *Server) error {
	t.Helper()
	return server.BroadcastULData(testutil.TestContext(), heldTenant, broadcastULDataFixture())
}

// A session that lost its connection stays resumable (SCACI §1): the server
// hands it to the holder.
func TestReleaseConnection_HoldsTheSessionThatLostItsConnection(t *testing.T) {
	holder := newHolderFake()
	server := newHeldTestServer(holder)
	session := heldSessionOf(heldSessionID, heldTenant, heldAcEui)

	loseConnection(server, session)

	held, ok := holder.holds(heldSessionID)
	require.True(t, ok, "the session is held for its resume")
	assert.Same(t, session, held)
}

// Every SC operation of the tenant is also recorded for its held sessions
// (SCACI §1, §3.8, §3.13).
func TestBroadcast_HasTheHolderRecordForHeldSessions(t *testing.T) {
	holder := newHolderFake()
	server := newHeldTestServer(holder)
	loseConnection(server, heldSessionOf(heldSessionID, heldTenant, heldAcEui))

	require.NoError(t, broadcastHeldUplink(t, server))
	require.NoError(t, server.BroadcastEPStatus(testutil.TestContext(), heldTenant,
		&EPStatusData{EpEui: 0x70B3D5677011150A, EpStatus: EPStatusDetached, Sign: &mioty.Numeric4{}}))

	assert.Equal(t, []heldRecord{{sessionID: heldSessionID, command: CmdULData}, {sessionID: heldSessionID, command: CmdEPStatus}},
		holder.recorded())
}

// F12 while offline: the holder is asked to record the result of a downlink
// Application Center X queued for X only, never for Application Center Y of
// the same tenant (SCACI §3.10.1, §3.12.1).
func TestBroadcastDLDataResult_HeldForTheQueuerOnly(t *testing.T) {
	holder := newHolderFake()
	server := newHeldTestServer(holder)
	loseConnection(server, heldSessionOf(heldSessionID, heldTenant, heldAcEui))
	loseConnection(server, heldSessionOf(heldOtherACID, heldTenant, heldOtherAcEui))

	require.NoError(t, server.BroadcastDLDataResult(testutil.TestContext(), ApplicationCenter{TenantID: heldTenant, AcEui: heldAcEui}, heldACQueID,
		&mioty.DLDataResult{EpEui: 0x70B3D5677011150A, QueId: uint64(heldInternalQueID), Result: mioty.ResultExpired}))

	assert.Equal(t, []heldRecord{{sessionID: heldSessionID, command: CmdDLDataResult}}, holder.recorded(),
		"Application Center Y never queued the downlink")
}

// An operation the holder cannot record fails the broadcast, so the delivery
// outbox retries it like a failed send.
func TestBroadcast_UnrecordableHeldOperationFails(t *testing.T) {
	holder := newHolderFake()
	holder.failWith = errHeldTestStore
	server := newHeldTestServer(holder)
	loseConnection(server, heldSessionOf(heldSessionID, heldTenant, heldAcEui))

	err := broadcastHeldUplink(t, server)

	require.ErrorIs(t, err, errBroadcastSessionFailed)
	assert.ErrorIs(t, err, errHeldTestStore)
}

// A resume continues the held session: its SC opIds keep decrementing from
// the last one issued (SCACI §3.2), and it stays held until its connect
// operation completes, so nothing produced meanwhile is lost.
func TestAdoptSession_ResumeContinuesTheHeldSession(t *testing.T) {
	holder := newHolderFake()
	server := newHeldTestServer(holder)
	loseConnection(server, heldSessionOf(heldSessionID, heldTenant, heldAcEui))
	resumed := resumingSession()

	require.True(t, adopts(testutil.TestContext(), server.registry, &mockConn{}, resumed))

	assert.Equal(t, heldLastIssued, resumed.OpIDs().SC)
	held, ok := holder.holds(heldSessionID)
	require.True(t, ok)
	assert.Same(t, resumed, held, "the resumed session is held until its connect operation completes")
}

// A resume of a session this server neither serves nor holds cannot continue
// it (SCACI §1): the connection is not adopted.
func TestAdoptSession_ResumeOfSessionNotHeldIsRefused(t *testing.T) {
	server := newHeldTestServer(newHolderFake())

	assert.False(t, adopts(testutil.TestContext(), server.registry, &mockConn{}, resumingSession()))
	assert.Empty(t, server.registry.sessions)
}

// A new session of the Application Center discards the previous session and
// what is held for it (SCACI §1); what is produced during its own connect
// operation is held for the new session (§3.3).
func TestAdoptSession_NewSessionDiscardsTheHeldSession(t *testing.T) {
	holder := newHolderFake()
	server := newHeldTestServer(holder)
	loseConnection(server, heldSessionOf(heldSessionID, heldTenant, heldAcEui))
	next := &Session{ID: heldSessionID + 100, TenantID: heldTenant, AcEui: heldAcEui, State: StateConnecting}

	require.True(t, adopts(testutil.TestContext(), server.registry, &mockConn{}, next))
	require.NoError(t, broadcastHeldUplink(t, server))

	assert.Equal(t, []heldRecord{{sessionID: next.ID, command: CmdULData}}, holder.recorded())
}

// An Application Center of another organization that claims the same acEui
// supersedes nothing: the owner keeps its live connection and its held
// session, and each keeps its own session.
func TestAdoptSession_AnotherOrganizationClaimingTheAcEuiSupersedesNothing(t *testing.T) {
	owner, claimant := uuid.New(), uuid.New()
	holder := newHolderFake()
	server := newHeldTestServer(holder)
	held := heldSessionOf(heldSessionID, heldTenant, heldAcEui)
	held.OrganizationID = owner
	loseConnection(server, held)
	liveConn := &mockConn{}
	live := &Session{ID: heldOtherACID, TenantID: heldTenant, OrganizationID: owner, AcEui: heldOtherAcEui, State: StateActive}
	server.registry.sessions[liveConn] = live

	for i, acEui := range []uint64{heldAcEui, heldOtherAcEui} {
		claim := &Session{ID: heldSessionID + 100 + int64(i), TenantID: heldTenant, OrganizationID: claimant, AcEui: acEui, State: StateConnecting}
		require.True(t, adopts(testutil.TestContext(), server.registry, &mockConn{}, claim))
	}

	assert.Same(t, live, server.registry.sessions[liveConn], "the owner's live connection is not evicted")
	stillHeld, ok := holder.holds(heldSessionID)
	assert.True(t, ok, "the owner's held session is not discarded")
	assert.Same(t, held, stillHeld)
	assert.Len(t, server.registry.sessions, 3, "each Application Center keeps its own session")
}

// Within one organization a new session of the Application Center still
// supersedes its earlier sessions, live and held (SCACI §1).
func TestAdoptSession_TheOrganizationsNewSessionSupersedesItsEarlierOnes(t *testing.T) {
	orgID := uuid.New()
	holder := newHolderFake()
	server := newHeldTestServer(holder)
	held := heldSessionOf(heldSessionID, heldTenant, heldAcEui)
	held.OrganizationID = orgID
	loseConnection(server, held)
	liveConn := &mockConn{}
	server.registry.sessions[liveConn] = &Session{ID: heldOtherACID, TenantID: heldTenant, OrganizationID: orgID, AcEui: heldAcEui, State: StateActive}

	require.True(t, adopts(testutil.TestContext(), server.registry, &mockConn{},
		&Session{ID: heldSessionID + 100, TenantID: heldTenant, OrganizationID: orgID, AcEui: heldAcEui, State: StateConnecting}))

	_, stillLive := server.registry.sessions[liveConn]
	assert.False(t, stillLive, "the earlier live session is evicted")
	_, stillHeld := holder.holds(heldSessionID)
	assert.False(t, stillHeld, "the earlier held session is discarded")
}

// The resume reissues what the session holds in the order it was issued: SC
// opIds strictly decrement (SCACI §3.2), whatever order the log returns.
func TestReplayPendingOperations_ReissuesInOpIDOrder(t *testing.T) {
	pending := func(opID int64) *models.SCACIOperation {
		return &models.SCACIOperation{OpId: opID, TenantID: heldTenant, Command: CmdDLDataResult,
			Direction:   string(models.OperationDirectionOutbound),
			RequestData: map[string]interface{}{MetadataKeyEpEui: "70B3D5677011150A", "queId": "77", "result": ResultExpired}}
	}
	server := newHeldTestServer(newHolderFake())
	server.operationRepo = &pendingLog{pending: map[int64][]*models.SCACIOperation{heldSessionID: {pending(-3), pending(-1), pending(-2)}}}
	conn := &mockConn{}

	server.replayPendingOperations(conn, heldSessionOf(heldSessionID, heldTenant, heldAcEui))

	assert.Equal(t, []int64{-1, -2, -3}, writtenOpIDs(t, conn.written))
}

// Nothing new goes out on a resumed session while its reissue runs: the
// Application Center sees the held operations first, then the new ones, in
// decrementing opId order (SCACI §1, §3.2).
func TestResume_NewOperationsWaitForTheReissue(t *testing.T) {
	repo := &blockingPendingRepo{entered: make(chan struct{}), release: make(chan struct{})}
	server := newHeldTestServer(newHolderFake())
	server.operationRepo = repo
	loseConnection(server, heldSessionOf(heldSessionID, heldTenant, heldAcEui))
	conn, _ := pipedTLS(t)
	resumed := resumingSession()
	require.True(t, adopts(testutil.TestContext(), server.registry, conn, resumed))
	require.NoError(t, server.handleConnectComplete(conn, resumed, OpIDConnect))
	<-repo.entered
	assert.Equal(t, StateReissuing, resumed.currentState(), "the session is not active before the reissue")

	sent := make(chan int64, 1)
	go func() {
		_ = server.initiateSCOperation(conn, resumed, scOperation{
			command: CmdULData,
			record:  func(*Session, int64) (Recording, error) { return RecordedNew, nil },
			send: func(_ net.Conn, _ *Session, opId int64) error {
				sent <- opId
				return nil
			},
		})
	}()
	select {
	case <-sent:
		t.Fatal("an operation started during the reissue went out before it")
	case <-time.After(shutdownSettleDelay):
	}

	close(repo.release)
	assert.Equal(t, heldLastIssued-1, <-sent)
	assert.Equal(t, StateActive, resumed.currentState())
}

// A reissued operation goes out through the same seam as a new one, so the
// Application Center's response to it is accepted and completed (SCACI §3.2).
func TestReplay_ReissuedOperationAwaitsItsResponse(t *testing.T) {
	const reissuedOpID = int64(-2)
	server := newHeldTestServer(newHolderFake())
	server.operationRepo = &pendingLog{pending: map[int64][]*models.SCACIOperation{heldSessionID: {{
		OpId: reissuedOpID, TenantID: heldTenant, Command: CmdDLDataResult, Direction: string(models.OperationDirectionOutbound),
		RequestData: map[string]interface{}{MetadataKeyEpEui: "70B3D5677011150A", "queId": "77", "result": ResultExpired},
	}}}}
	conn := &mockConn{}
	session := heldSessionOf(heldSessionID, heldTenant, heldAcEui)
	session.State = StateActive
	server.replayPendingOperations(conn, session)

	conn.written = nil
	sess := session
	require.NoError(t, server.routeMessage(conn, &sess, nil, CmdDLDataResultResponse, reissuedOpID, nil))

	var cmp DLDataResultComplete
	require.NoError(t, decodeResponse(conn.written, &cmp))
	assert.Equal(t, CmdDLDataResultComplete, cmp.Command, "the response to a reissued operation completes it")
}

// A resume whose session stopped being held while it connected (its
// resumability ended) is closed: the Application Center starts a new session.
func TestHandleConnectComplete_ResumeOfSessionNoLongerHeldIsClosed(t *testing.T) {
	server := newHeldTestServer(newHolderFake())
	conn, peer := pipedTLS(t)

	_ = server.handleConnectComplete(conn, resumingSession(), OpIDConnect)

	assert.ErrorIs(t, peer.SetReadDeadline(time.Now()), io.ErrClosedPipe, "the service center closes the connection")
}

// The connect operation completes once (SCACI §3.3): a second conCmp is a
// protocol error and does not replay anything again.
func TestHandleConnectComplete_SecondConnectCompleteIsRefused(t *testing.T) {
	server := newHeldTestServer(newHolderFake())
	conn := &mockConn{}
	active := &Session{ID: heldSessionID, TenantID: heldTenant, State: StateActive, Resumed: true}

	require.NoError(t, server.handleConnectComplete(conn, active, OpIDConnect))

	var refusal Error
	require.NoError(t, decodeResponse(conn.written, &refusal))
	assert.Equal(t, CmdError, refusal.Command)
	assert.Equal(t, POSIX_EPROTO, refusal.Code)
}

// pipedTLS is a TLS connection whose handshake never runs, and the peer end
// of its transport.
func pipedTLS(t *testing.T) (*tls.Conn, net.Conn) {
	t.Helper()
	peer, serverSide := net.Pipe()
	t.Cleanup(func() {
		_ = peer.Close()
		_ = serverSide.Close()
	})
	return tls.Client(serverSide, &tls.Config{ServerName: shutdownTestServer, MinVersion: tls.VersionTLS12}), peer
}

// writtenOpIDs decodes every frame written to a connection and returns their
// opIds in write order.
func writtenOpIDs(t *testing.T, written []byte) []int64 {
	t.Helper()
	reader := bytes.NewReader(written)
	var opIDs []int64
	for reader.Len() > 0 {
		frame, err := testFrameCodec.Read(reader)
		require.NoError(t, err)
		var msg BaseMessage
		require.NoError(t, decodeResponse(frame.Payload, &msg))
		opIDs = append(opIDs, msg.OpId)
	}
	return opIDs
}
