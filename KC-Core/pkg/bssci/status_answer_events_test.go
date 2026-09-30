package bssci

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// statusAnswer is a well-formed statusRsp payload (BSSCI §5.5.2).
func statusAnswer() map[string]interface{} {
	return map[string]interface{}{"code": int64(0), "message": "ok", "time": int64(1)}
}

func answerStatus(t *testing.T, server *Server, session *Session, opID int64) {
	t.Helper()
	require.NoError(t, server.handleStatusResponse(session, &Message{Command: mioty.CmdStatusResponse, OpId: opID}, statusAnswer()))
}

// The station's answer to an operator's status request is recorded for the
// station with the request's opId, so the operator's view refreshes on it.
func TestOperatorStatusRequest_AnswerIsRecorded(t *testing.T) {
	events := &stationEventLog{}
	server, session, _ := newPingServer(t, events)

	opID, err := server.SendStatusRequest(session)
	require.NoError(t, err)
	answerStatus(t, server, session, opID)

	require.Len(t, events.events, 1)
	answered := events.events[0]
	assert.Equal(t, models.EventTypeBaseStationStatusAnswered, answered.eventType)
	assert.Equal(t, mioty.EUI64(uint64(TestBsEui01)).ToBytes(), answered.eui)
	assert.Equal(t, opID, answered.data[models.EventDetailKeyOpID])
	assert.Equal(t, pingTestTenant, answered.tenantID)
}

// The periodic status poll answers every interval; its answers are samples,
// not operator actions, and record nothing.
func TestPeriodicStatusRequest_AnswerIsNotRecorded(t *testing.T) {
	events := &stationEventLog{}
	server, session, _ := newPingServer(t, events)

	opID, err := server.requestPeriodicStatus(session)
	require.NoError(t, err)
	answerStatus(t, server, session, opID)

	assert.Empty(t, events.events)
}

// An answer is recorded once: a repeated statusRsp for the same opId is not
// an operator request any more.
func TestOperatorStatusRequest_AnswerIsRecordedOnce(t *testing.T) {
	events := &stationEventLog{}
	server, session, _ := newPingServer(t, events)

	opID, err := server.SendStatusRequest(session)
	require.NoError(t, err)
	answerStatus(t, server, session, opID)
	answerStatus(t, server, session, opID)

	assert.Len(t, events.events, 1)
}

// persistedRowOf is the row the production StatusService writes for a
// recorded pending operation.
func persistedRowOf(t *testing.T, op *PendingOperation) PersistedOperation {
	t.Helper()
	message, err := json.Marshal(op.Message)
	require.NoError(t, err)
	metadata, err := json.Marshal(op.Metadata)
	require.NoError(t, err)
	return PersistedOperation{OperationID: op.OperationID, OperationType: op.OperationType,
		OperationData: message, Metadata: metadata}
}

// resumeStation reconnects the station on a new connection and reissues the
// operations the previous connection left persisted.
func resumeStation(t *testing.T, server *Server, previous *Session) *Session {
	t.Helper()
	resumed := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "ping-session-resumed",
			BaseStationEUI:    previous.BaseStationEUI,
			Encoding:          EncodingJSON,
			HandshakeComplete: true,
			DbSessionID:       previous.DbSessionID,
		},
		Conn: &bsscitest.TestConn{Encoding: EncodingJSON},
	}
	server.RegisterSession(resumed)
	ops, err := server.pendingOps.load(testutil.TestContext(), resumed)
	require.NoError(t, err)
	require.NoError(t, server.reissueResumedOperations(testutil.TestContext(), resumed, ops))
	return resumed
}

// An operator's status request whose frame may have reached the wire is kept
// for resume; its reissue after the station reconnects is still the
// operator's request, so the answer to the reissued opId is recorded.
func TestOperatorStatusRequest_ReissuedAfterAmbiguousWrite_AnswerIsRecorded(t *testing.T) {
	events := &stationEventLog{}
	server, session, _ := newPingServer(t, events)
	session.Conn = &countingConn{failWrite: true}

	_, err := server.SendStatusRequest(session)
	require.ErrorIs(t, err, ErrAmbiguousWrite)
	opID := session.LastScOpId
	kept, err := server.statusSvc.GetPendingOperation(session, opID)
	require.NoError(t, err, "an ambiguous write keeps the pending row for resume")
	server.statusSvc.(*memoryStatusService).persistRows(session.DbSessionID, persistedRowOf(t, kept))

	resumed := resumeStation(t, server, session)
	answerStatus(t, server, resumed, opID)

	require.Len(t, events.events, 1, "the reissued operator request's answer is recorded")
	assert.Equal(t, models.EventTypeBaseStationStatusAnswered, events.events[0].eventType)
	assert.Equal(t, opID, events.events[0].data[models.EventDetailKeyOpID])
}

// A reissued periodic poll stays a sample: its answer records nothing.
func TestPeriodicStatusRequest_ReissuedAfterAmbiguousWrite_AnswerIsNotRecorded(t *testing.T) {
	events := &stationEventLog{}
	server, session, _ := newPingServer(t, events)
	session.Conn = &countingConn{failWrite: true}

	_, err := server.requestPeriodicStatus(session)
	require.ErrorIs(t, err, ErrAmbiguousWrite)
	opID := session.LastScOpId
	kept, err := server.statusSvc.GetPendingOperation(session, opID)
	require.NoError(t, err)
	server.statusSvc.(*memoryStatusService).persistRows(session.DbSessionID, persistedRowOf(t, kept))

	resumed := resumeStation(t, server, session)
	answerStatus(t, server, resumed, opID)

	assert.Empty(t, events.events)
}
