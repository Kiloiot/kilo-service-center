package scaci

import (
	"errors"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	errorFrameTestSessionID = int64(77)
	errorFrameTestAcEui     = uint64(0x70B3D59CD0000A01)
	errorFrameTestOpID      = int64(5)
)

var errCloseRefused = errors.New("close refused")

// closeRefusingConn delivers writes to its peer but refuses to close.
type closeRefusingConn struct{ net.Conn }

func (c closeRefusingConn) Close() error {
	if err := c.Conn.Close(); err != nil {
		return err
	}
	return errCloseRefused
}

func newErrorFrameServer(log logger.Logger) *Server {
	return &Server{
		codec:    testFrameCodec,
		commands: mustTestCommandRegistry(),
		clock:    clock.SystemClock{},
		logger:   log,
		config:   &Config{},
	}
}

// brokenConn returns a connection whose peer is gone, so every write fails.
func brokenConn(t *testing.T) net.Conn {
	t.Helper()
	conn, peer := net.Pipe()
	require.NoError(t, peer.Close())
	t.Cleanup(func() { assert.NoError(t, conn.Close()) })
	return conn
}

// drainedPeer returns a connection whose peer reads everything it is sent.
func drainedPeer(t *testing.T) net.Conn {
	t.Helper()
	conn, peer := net.Pipe()
	copied := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(io.Discard, peer)
		copied <- copyErr
	}()
	t.Cleanup(func() {
		assert.NoError(t, conn.Close())
		assert.NoError(t, <-copied, "the peer drains until the connection closes")
	})
	return conn
}

// An error frame the Application Center cannot be sent is logged with the
// application center, the session, the opId and the catalog token.
func TestErrorFrame_WriteFailureIsLoggedWithSessionAndToken(t *testing.T) {
	log := bsscitest.NewRecordingLogger()
	server := newErrorFrameServer(log)
	session := &Session{ID: errorFrameTestSessionID, AcEui: errorFrameTestAcEui}

	_, _, ok := server.frameHeader(brokenConn(t), session, []byte{0xc1})

	require.False(t, ok)
	logged := log.FilterMessage(LogSCACISendErrorFailed)
	require.Len(t, logged, 1, "the failed error frame is logged once")
	fields := logged[0].FieldMap()
	assert.Equal(t, "ERROR", logged[0].Level)
	assert.Equal(t, mioty.FormatEUI64(errorFrameTestAcEui), fields[logger.FieldAcEui])
	assert.Equal(t, errorFrameTestSessionID, fields[logger.FieldSessionID])
	assert.Equal(t, int64(0), fields[logger.FieldOpID])
	assert.Equal(t, errInvalidMessageFormat, fields[logger.FieldErrorToken])
	assert.NotNil(t, fields[logger.FieldError], "the transport error is logged")
}

// A sent error frame logs no failure.
func TestErrorFrame_SentFrameLogsNoFailure(t *testing.T) {
	log := bsscitest.NewRecordingLogger()
	server := newErrorFrameServer(log)

	_, _, ok := server.frameHeader(drainedPeer(t), &Session{ID: errorFrameTestSessionID}, []byte{0xc1})

	require.False(t, ok)
	assert.Empty(t, log.FilterMessage(LogSCACISendErrorFailed))
}

// A refused connect whose connection will not close is logged at WARN with
// the peer it belongs to.
func TestRefusedConnect_CloseFailureIsLogged(t *testing.T) {
	log := bsscitest.NewRecordingLogger()
	server := newErrorFrameServer(log)
	payload, err := msgpack.Marshal(&Connect{
		BaseMessage: BaseMessage{Command: CmdConnect, OpId: errorFrameTestOpID},
		Version:     ProtocolVersionString,
		AcEui:       errorFrameTestAcEui,
	})
	require.NoError(t, err)
	var session *Session

	err = server.handleConnect(closeRefusingConn{drainedPeer(t)}, &session, nil, errorFrameTestOpID, payload)

	require.ErrorIs(t, err, errInvalidConnectOpId)
	logged := log.FilterMessage(LogSCACICloseConnectionFailed)
	require.Len(t, logged, 1)
	fields := logged[0].FieldMap()
	assert.Equal(t, "WARN", logged[0].Level)
	assert.NotEmpty(t, fields[logger.FieldRemote])
	assert.Equal(t, errCloseRefused, fields[logger.FieldError])
}

var errStateWriteRefused = errors.New("state write refused")

// A failed operation whose failure cannot be recorded is logged with the
// session and opId.
func TestMarkOperationFailed_UnrecordedFailureIsLogged(t *testing.T) {
	log := bsscitest.NewRecordingLogger()
	server := newErrorFrameServer(log)
	repo := &MockSCACIOperationRepository{}
	repo.On("UpdateOperationState", mock.Anything, errorFrameTestSessionID, errorFrameTestOpID,
		models.OperationStateFailed, mock.Anything).Return(errStateWriteRefused)
	server.operationRepo = repo

	server.markOperationFailed(&Session{ID: errorFrameTestSessionID}, errorFrameTestOpID, nil)

	repo.AssertExpectations(t)
	logged := log.FilterMessage(LogSCACIUpdateOperationStateFailed)
	require.Len(t, logged, 1)
	fields := logged[0].FieldMap()
	assert.Equal(t, errorFrameTestSessionID, fields[logger.FieldSessionID])
	assert.Equal(t, errorFrameTestOpID, fields[logger.FieldOpID])
	assert.Equal(t, errStateWriteRefused, fields[logger.FieldError])
}

// A recorded reception whose subpackets cannot be decoded is replayed without
// them, and the undecodable field is logged.
func TestReplayReception_UndecodableSubpacketsAreLogged(t *testing.T) {
	log := bsscitest.NewRecordingLogger()
	server := newErrorFrameServer(log)

	reception := server.replayReception(&Session{ID: errorFrameTestSessionID}, errorFrameTestOpID,
		map[string]interface{}{metadataKeySubpackets: []interface{}{}})

	assert.Nil(t, reception.Subpackets)
	logged := log.FilterMessage(LogSCACIReplayULDataFieldDecodeErr)
	require.Len(t, logged, 1)
	assert.Equal(t, metadataKeySubpackets, logged[0].FieldMap()[logger.FieldField])
}
