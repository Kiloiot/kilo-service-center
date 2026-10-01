package scaci

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	ownershipTestSessionID = int64(100)
	ownershipTestTenantID  = int64(1)
	ownershipTestOpID      = int64(-7)
)

// stateTransition is one UpdateOperationState call as the handlers issue it.
type stateTransition struct {
	opID  int64
	state models.OperationState
	data  map[string]interface{}
}

// recordingOperationRepo keeps every state transition in call order.
type recordingOperationRepo struct {
	mockOperationRepoStub
	transitions []stateTransition
}

func (r *recordingOperationRepo) UpdateOperationState(_ context.Context, _ int64, opID int64, state models.OperationState, data map[string]interface{}) error {
	r.transitions = append(r.transitions, stateTransition{opID: opID, state: state, data: data})
	return nil
}

func newOwnershipServer(repo *recordingOperationRepo) *Server {
	sessions := &mockSessionRepository{}
	sessions.On("UpdateOperationIDs", mock.Anything, ownershipTestTenantID, ownershipTestSessionID, mock.Anything, mock.Anything).Return(nil)
	return &Server{
		registry:           newTestRegistry(nil, newHolderFake()),
		codec:              testFrameCodec,
		commands:           mustTestCommandRegistry(),
		clock:              clock.SystemClock{},
		logger:             testLogger(),
		operationRepo:      repo,
		sessionPersistence: sessionRowsOver{repo: sessions},
		config:             &Config{},
	}
}

func ownershipSession() *Session {
	return &Session{ID: ownershipTestSessionID, TenantID: ownershipTestTenantID, State: StateActive}
}

func TestHandleEPStatusResponse_ServiceCenterCompletesTheOperation(t *testing.T) {
	repo := &recordingOperationRepo{}
	conn := &mockConn{}

	s := newOwnershipServer(repo)
	require.NoError(t, s.handleEPStatusResponse(conn, ownershipSession(), ownershipTestOpID))
	s.persistTasks.wg.Wait()

	var cmp EPStatusComplete
	require.NoError(t, decodeResponse(conn.written, &cmp))
	assert.Equal(t, CmdEPStatusComplete, cmp.Command, "the initiator closes the handshake with epStatCmp")
	assert.Equal(t, ownershipTestOpID, cmp.OpId, "the complete reuses the operation id")

	require.Len(t, repo.transitions, 2)
	assert.Equal(t, models.OperationStateAcknowledged, repo.transitions[0].state)
	assert.Equal(t, models.OperationStateCompleted, repo.transitions[1].state)
	assert.Equal(t, ownershipTestOpID, repo.transitions[1].opID)
}

func TestRejectACIssuedCompletes_FailTheOperationAndAnswerEPROTO(t *testing.T) {
	cases := []struct {
		name   string
		reject func(*Server, net.Conn, *Session, int64) error
		token  string
		detail string
	}{
		{name: CmdULDataComplete, reject: (*Server).rejectACIssuedULDataComplete, token: errProtocolViolationULCmp, detail: errDetailACSentULDataCmp},
		{name: CmdDLDataResultComplete, reject: (*Server).rejectACIssuedDLDataResultComplete, token: errProtocolViolationDLResCmp, detail: errDetailACSentTxDataResCmp},
		{name: CmdEPStatusComplete, reject: (*Server).rejectACIssuedEPStatusComplete, token: errProtocolViolationEPStatCmp, detail: errDetailACSentEPStatusCmp},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &recordingOperationRepo{}
			conn := &mockConn{}

			require.NoError(t, tc.reject(newOwnershipServer(repo), conn, ownershipSession(), ownershipTestOpID))

			var errorResp Error
			require.NoError(t, decodeResponse(conn.written, &errorResp))
			assert.Equal(t, CmdError, errorResp.Command)
			assert.Equal(t, ownershipTestOpID, errorResp.OpId)
			assert.Equal(t, POSIX_EPROTO, errorResp.Code)
			assertErrorToken(t, errorResp, tc.token)

			require.Len(t, repo.transitions, 1)
			assert.Equal(t, models.OperationStateFailed, repo.transitions[0].state)
			assert.Equal(t, tc.token, repo.transitions[0].data[MetadataKeyErrorToken])
			assert.Equal(t, tc.detail, repo.transitions[0].data[MetadataKeyErrorDetail])
		})
	}
}

func TestRejectACIssuedCompletes_NoSessionAnswersEINVAL(t *testing.T) {
	cases := []struct {
		name   string
		reject func(*Server, net.Conn, *Session, int64) error
	}{
		{name: CmdDLDataResultComplete, reject: (*Server).rejectACIssuedDLDataResultComplete},
		{name: CmdEPStatusComplete, reject: (*Server).rejectACIssuedEPStatusComplete},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &recordingOperationRepo{}
			conn := &mockConn{}

			require.NoError(t, tc.reject(newOwnershipServer(repo), conn, nil, ownershipTestOpID))

			var errorResp Error
			require.NoError(t, decodeResponse(conn.written, &errorResp))
			assert.Equal(t, POSIX_EINVAL, errorResp.Code)
			assertErrorToken(t, errorResp, errNoActiveSession)
			assert.Empty(t, repo.transitions)
		})
	}
}

// scResponses are the application center's answers to service-center
// operations (SCACI §3.4.2, §3.8.2, §3.12.2, §3.13.2).
var scResponses = []string{CmdPingResponse, CmdULDataResponse, CmdDLDataResultResponse, CmdEPStatusResponse}

func routeResponse(t *testing.T, s *Server, conn net.Conn, session *Session, command string, opID int64) {
	t.Helper()
	sess := session
	require.NoError(t, s.routeMessage(conn, &sess, nil, command, opID, nil))
}

// A response for an operation the service center never started is answered
// with an error (SCACI §3.14), never with a completion.
func TestRouteMessage_UnsolicitedResponseIsAnsweredWithAnError(t *testing.T) {
	for _, command := range scResponses {
		t.Run(command, func(t *testing.T) {
			conn := &mockConn{}
			s := newOwnershipServer(&recordingOperationRepo{})

			routeResponse(t, s, conn, &Session{TenantID: ownershipTestTenantID, State: StateActive}, command, ownershipTestOpID)

			var reply Error
			require.NoError(t, decodeResponse(conn.written, &reply))
			assert.Equal(t, CmdError, reply.Command, "no completion for an operation the service center did not start")
			assert.Equal(t, ownershipTestOpID, reply.OpId)
			assert.Equal(t, POSIX_EPROTO, reply.Code)
			assertErrorToken(t, reply, errUnsolicitedResponse)
		})
	}
}

// The response to a broadcast operation completes it once; a repeated
// response is unsolicited.
func TestRouteMessage_ResponseSettlesTheBroadcastOperationOnce(t *testing.T) {
	conn := &mockConn{}
	s := newOwnershipServer(&recordingOperationRepo{})
	session := &Session{TenantID: ownershipTestTenantID, State: StateActive}
	s.registry.sessions = map[net.Conn]*Session{conn: session}
	require.NoError(t, s.BroadcastULData(testutil.TestContext(), ownershipTestTenantID, broadcastULDataFixture()))
	opID := session.OpIDs().SC

	conn.written = nil
	routeResponse(t, s, conn, session, CmdULDataResponse, opID)
	var cmp ULDataComplete
	require.NoError(t, decodeResponse(conn.written, &cmp))
	assert.Equal(t, CmdULDataComplete, cmp.Command)

	conn.written = nil
	routeResponse(t, s, conn, session, CmdULDataResponse, opID)
	var reply Error
	require.NoError(t, decodeResponse(conn.written, &reply))
	assert.Equal(t, CmdError, reply.Command)
}

func TestRouteMessage_PingResponseSettlesTheServiceCenterPing(t *testing.T) {
	conn := &mockConn{}
	s := newOwnershipServer(&recordingOperationRepo{})
	session := &Session{TenantID: ownershipTestTenantID, State: StateActive}
	require.NoError(t, s.initiatePing(conn, session))

	conn.written = nil
	routeResponse(t, s, conn, session, CmdPingResponse, session.OpIDs().SC)

	var cmp PingComplete
	require.NoError(t, decodeResponse(conn.written, &cmp))
	assert.Equal(t, CmdPingComplete, cmp.Command)
}

// pendingOperationRepo serves the pending operations of a resumed session.
type pendingOperationRepo struct {
	recordingOperationRepo
	pending []*models.SCACIOperation
}

func (r *pendingOperationRepo) GetPendingOperations(context.Context, int64) ([]*models.SCACIOperation, error) {
	return r.pending, nil
}

// A reissued operation is awaited like a new one (SCACI §1).
func TestRouteMessage_ResponseSettlesTheReplayedOperation(t *testing.T) {
	conn := &mockConn{}
	uplink := broadcastULDataFixture()
	repo := &pendingOperationRepo{pending: []*models.SCACIOperation{{
		OpId: ownershipTestOpID, TenantID: ownershipTestTenantID, Command: CmdULData,
		Direction: string(models.OperationDirectionOutbound),
		RequestData: map[string]interface{}{
			MetadataKeyEpEui: mioty.FormatEUI64(uplink.EpEui),
			"baseStations": []interface{}{map[string]interface{}{
				"bsEui": mioty.FormatEUI64(uplink.BaseStations[0].BsEui), "rxTime": float64(uplink.BaseStations[0].RxTime),
			}},
		},
	}}}
	s := newOwnershipServer(&repo.recordingOperationRepo)
	s.operationRepo = repo
	session := ownershipSession()

	s.replayPendingOperations(conn, session)
	conn.written = nil
	routeResponse(t, s, conn, session, CmdULDataResponse, ownershipTestOpID)

	var cmp ULDataComplete
	require.NoError(t, decodeResponse(conn.written, &cmp))
	assert.Equal(t, CmdULDataComplete, cmp.Command)
}
