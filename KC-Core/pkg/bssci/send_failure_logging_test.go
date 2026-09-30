package bssci

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const sendFailureOpID = int64(41)

var errCloseRefused = errors.New("close refused")

// closeRefusingConn is a link whose Close fails.
type closeRefusingConn struct{ countingConn }

func (c *closeRefusingConn) Close() error { return errCloseRefused }

func newSendFailureServer(t *testing.T, log *recordingLogger) *Server {
	t.Helper()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, storage := CreateTestServices(log, nil)
	return NewTestServer(log, storage, nil, pingTestTenant,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver)
}

func sendFailureSession(conn *bsscitest.TestConn) *Session {
	return &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "send-failure-session",
			BaseStationEUI:    uint64(TestBsEui01),
			Encoding:          EncodingJSON,
			HandshakeComplete: true,
		},
		Conn: conn,
	}
}

func entriesWithMessage(log *recordingLogger, msg string) []logEntry {
	var matched []logEntry
	for _, entry := range log.entries {
		if entry.msg == msg {
			matched = append(matched, entry)
		}
	}
	return matched
}

// An error frame the station cannot be sent is logged with the station, the
// session and the opId it answered; the handler still reports the refusal.
func TestSendCatalogError_WriteFailureIsLoggedWithSessionAndOpID(t *testing.T) {
	log := newRecordingLogger()
	server := newSendFailureServer(t, log)
	session := sendFailureSession(&bsscitest.TestConn{Encoding: EncodingJSON, FailWrites: true})

	err := server.handleVMDLData(session, &Message{Command: mioty.CmdVMDLData, OpId: sendFailureOpID}, nil)

	require.EqualError(t, err, ResolveErrorMessage(errUnsupportedCommand))
	logged := entriesWithMessage(log, LogBSSCIFailedToSendCatalogError)
	require.Len(t, logged, 1, "the failed error frame is logged once")
	assert.Equal(t, "ERROR", logged[0].level)
	assert.Equal(t, uint64(TestBsEui01), logged[0].fields[logger.FieldBsEui])
	assert.Equal(t, session.ID, logged[0].fields[logger.FieldSessionID])
	assert.Equal(t, sendFailureOpID, logged[0].fields[logger.FieldOpID])
	assert.Equal(t, errUnsupportedCommand, logged[0].fields[logger.FieldErrorToken])
	assert.NotNil(t, logged[0].fields[logger.FieldError], "the transport error is logged")
}

// A sent error frame logs no failure.
func TestSendCatalogError_SentFrameLogsNoFailure(t *testing.T) {
	log := newRecordingLogger()
	server := newSendFailureServer(t, log)
	conn := &bsscitest.TestConn{Encoding: EncodingJSON}

	server.sendCatalogError(sendFailureSession(conn), sendFailureOpID, NewCatalogError(errUnsupportedCommand, POSIX_ENOSYS))

	assert.True(t, conn.SeenCommand(mioty.CmdError), "the error frame is written")
	assert.Empty(t, entriesWithMessage(log, LogBSSCIFailedToSendCatalogError))
}

// A transport that refuses to close after an ambiguous write is logged with
// the station and the opId whose write failed.
func TestCloseTransportAfterWriteFailure_CloseFailureIsLogged(t *testing.T) {
	log := newRecordingLogger()
	server := newSendFailureServer(t, log)
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{ID: "close-failure-session", BaseStationEUI: uint64(TestBsEui01)},
		Conn:                 &closeRefusingConn{},
	}

	server.closeTransportAfterWriteFailure(session, sendFailureOpID, ErrAmbiguousWrite)

	logged := entriesWithMessage(log, LogBSSCIFailedToCloseConnection)
	require.Len(t, logged, 1)
	assert.Equal(t, "WARN", logged[0].level)
	assert.Equal(t, sendFailureOpID, logged[0].fields[logger.FieldOpID])
	assert.Equal(t, uint64(TestBsEui01), logged[0].fields[logger.FieldBsEui])
	assert.Equal(t, errCloseRefused, logged[0].fields[logger.FieldError])
}
