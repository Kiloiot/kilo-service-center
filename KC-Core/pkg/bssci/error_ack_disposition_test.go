package bssci

import (
	"testing"
	"time"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newErrorAckFixture builds a server + session with one live pending SC
// operation recorded at opID, for exercising the errorAck disposition rules
// (BSSCI rev1 §5.17 / classic §3.17).
func newErrorAckFixture(t *testing.T, opID int64) (*Server, StatusService, *Session) {
	t.Helper()
	log := newRecordingLogger()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, storage := CreateTestServices(log, nil)
	server := NewTestServer(log, storage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	server.config = &Config{MessageEncoding: EncodingJSON}
	server.RegisterHandlers()
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "errorack-test",
			BaseStationEUI:    TestBsEui01,
			ResolvedTenantID:  1,
			DbSessionID:       1,
			Encoding:          EncodingJSON,
			HandshakeComplete: true,
			ConnectState:      ConnectStateComplete,
		},
		Conn: &bsscitest.TestConn{Encoding: EncodingJSON},
	}
	pendingOp := &PendingOperation{
		OperationType: mioty.CmdDLDataQueue,
		CreatedAt:     time.Now(),
		Metadata:      map[string]interface{}{},
	}
	require.NoError(t, statusSvc.RecordPendingOperation(testutil.TestContext(), session, opID, pendingOp, session.DbSessionID))
	return server, statusSvc, session
}

func errorAckMsg(opID int64) (*Message, map[string]interface{}) {
	data := map[string]interface{}{"command": mioty.CmdErrorAck, "opId": opID}
	return &Message{Command: mioty.CmdErrorAck, OpId: opID, Data: data}, data
}

// TestErrorAck_Unsolicited_RemovesNothing: an errorAck for an operation this
// service center never sent an error about must not touch the live pending
// operation - a spurious or forged errorAck cannot finalize in-flight work.
func TestErrorAck_Unsolicited_RemovesNothing(t *testing.T) {
	const opID = int64(-7)
	server, statusSvc, session := newErrorAckFixture(t, opID)

	msg, data := errorAckMsg(opID)
	require.NoError(t, server.handleErrorAck(server, session, msg, data))

	_, err := statusSvc.GetPendingOperation(session, opID)
	assert.NoError(t, err, "unsolicited errorAck must not remove the pending operation")
}

// TestErrorAck_AckOnly_RemovesNothing: a plain rejection error (sendError)
// solicits an errorAck that closes the exchange without touching pending
// state, even when a pending operation happens to share the opId.
func TestErrorAck_AckOnly_RemovesNothing(t *testing.T) {
	const opID = int64(-7)
	server, statusSvc, session := newErrorAckFixture(t, opID)

	require.NoError(t, server.sendError(session, opID, POSIX_EPROTO, "test rejection"))

	msg, data := errorAckMsg(opID)
	require.NoError(t, server.handleErrorAck(server, session, msg, data))

	_, err := statusSvc.GetPendingOperation(session, opID)
	assert.NoError(t, err, "ack-only errorAck must not remove the pending operation")
}

// TestErrorAck_Finalizing_RemovesExactlyOnce: an error that replaced a pending
// SC operation's normal sequence is completed by the errorAck, which finalizes
// that operation; a duplicate errorAck is ignored.
func TestErrorAck_Finalizing_RemovesExactlyOnce(t *testing.T) {
	const opID = int64(-7)
	server, statusSvc, session := newErrorAckFixture(t, opID)

	require.NoError(t, server.sendErrorReplacingOperation(session, opID, POSIX_EPROTO, "operation replaced"))

	msg, data := errorAckMsg(opID)
	require.NoError(t, server.handleErrorAck(server, session, msg, data))

	_, err := statusSvc.GetPendingOperation(session, opID)
	assert.Error(t, err, "finalizing errorAck completes the operation and removes its pending row")

	// A duplicate errorAck finds no awaited entry and changes nothing
	require.NoError(t, server.handleErrorAck(server, session, msg, data))
}

// TestErrorAck_WrongSession_RemovesNothing: the awaited-errorAck expectation
// is connection-scoped; an errorAck arriving on a different session for the
// same opId must not finalize the other session's operation.
func TestErrorAck_WrongSession_RemovesNothing(t *testing.T) {
	const opID = int64(-7)
	server, statusSvc, session := newErrorAckFixture(t, opID)

	require.NoError(t, server.sendErrorReplacingOperation(session, opID, POSIX_EPROTO, "operation replaced"))

	otherSession := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "other-session",
			BaseStationEUI:    TestBsEui02,
			ResolvedTenantID:  1,
			DbSessionID:       2,
			Encoding:          EncodingJSON,
			HandshakeComplete: true,
			ConnectState:      ConnectStateComplete,
		},
		Conn: &bsscitest.TestConn{Encoding: EncodingJSON},
	}

	msg, data := errorAckMsg(opID)
	require.NoError(t, server.handleErrorAck(server, otherSession, msg, data))

	_, err := statusSvc.GetPendingOperation(session, opID)
	assert.NoError(t, err, "an errorAck on a different session must not finalize this session's operation")
}

// TestErrorAck_PositiveOpID_NoPendingTouch: an errorAck acknowledging an error
// this service center sent about a BS-initiated (positive) operation consumes
// the expectation without any pending-operation removal.
func TestErrorAck_PositiveOpID_NoPendingTouch(t *testing.T) {
	const negOpID = int64(-7)
	const posOpID = int64(9)
	server, statusSvc, session := newErrorAckFixture(t, negOpID)

	require.NoError(t, server.sendError(session, posOpID, POSIX_EPROTO, "inbound command rejected"))

	msg, data := errorAckMsg(posOpID)
	require.NoError(t, server.handleErrorAck(server, session, msg, data))

	_, err := statusSvc.GetPendingOperation(session, negOpID)
	assert.NoError(t, err, "positive-opId errorAck must not touch SC pending operations")
}
