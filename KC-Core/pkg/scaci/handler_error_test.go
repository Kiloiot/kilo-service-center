package scaci

import (
	"net"
	"testing"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// TestHandleInboundError_ZeroCode_ReturnsEINVAL validates Fix 3:
// When AC sends error with code=0, SC must reject it with POSIX_EINVAL.
//
// Spec: SCACI §3.14.1 - Error messages must have non-zero POSIX code
func TestHandleInboundError_ZeroCode_ReturnsEINVAL(t *testing.T) {
	server := &Server{
		registry: newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:    testFrameCodec,
		commands: mustTestCommandRegistry(),
		clock:    clock.SystemClock{},
		logger:   testLogger(),
		config:   &Config{},
	}

	conn := &mockConn{}
	session := &Session{ID: 100, TenantID: 1, State: StateActive}

	// Create error with code=0 (invalid per §3.14.1)
	errMsg := Error{
		BaseMessage: BaseMessage{Command: CmdError, OpId: 1},
		Code:        0, // Invalid - POSIX_OK is not valid for Error messages
		Message:     "Some error message",
	}
	payload, err := msgpack.Marshal(&errMsg)
	require.NoError(t, err)

	handlerErr := server.handleInboundError(conn, session, 1, payload)
	assert.NoError(t, handlerErr)

	// Verify error response was sent
	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_EINVAL, errorResp.Code, "code=0 in Error must be rejected with POSIX_EINVAL")
	assertErrorToken(t, errorResp, errErrorMissingCode)
}

// TestHandleInboundError_EmptyMessage_ReturnsEINVAL validates Fix 3:
// When AC sends error with empty message, SC must reject it with POSIX_EINVAL.
//
// Spec: SCACI §3.14.1 - Error messages must have non-empty message field
func TestHandleInboundError_EmptyMessage_ReturnsEINVAL(t *testing.T) {
	server := &Server{
		registry: newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:    testFrameCodec,
		commands: mustTestCommandRegistry(),
		clock:    clock.SystemClock{},
		logger:   testLogger(),
		config:   &Config{},
	}

	conn := &mockConn{}
	session := &Session{ID: 100, TenantID: 1, State: StateActive}

	// Create error with empty message (invalid per §3.14.1)
	errMsg := Error{
		BaseMessage: BaseMessage{Command: CmdError, OpId: 1},
		Code:        POSIX_EINVAL, // Valid code
		Message:     "",           // Invalid - empty message
	}
	payload, err := msgpack.Marshal(&errMsg)
	require.NoError(t, err)

	handlerErr := server.handleInboundError(conn, session, 1, payload)
	assert.NoError(t, handlerErr)

	// Verify error response was sent
	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_EINVAL, errorResp.Code, "empty message in Error must be rejected with POSIX_EINVAL")
	assertErrorToken(t, errorResp, errErrorMissingMessage)
}

// TestHandleInboundError_Valid_SendsErrorAck validates that valid error messages
// are processed correctly and an errorAck is sent.
//
// Spec: SCACI §3.14 - Error handshake: error → errorAck
func TestHandleInboundError_Valid_SendsErrorAck(t *testing.T) {
	server := &Server{
		registry: newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:    testFrameCodec,
		commands: mustTestCommandRegistry(),
		clock:    clock.SystemClock{},
		logger:   testLogger(),
		config:   &Config{},
		// No errorRecorder - test basic flow without persistence
	}

	conn := &mockConn{}
	session := &Session{ID: 100, TenantID: 1, State: StateActive}

	// Create valid error message
	errMsg := Error{
		BaseMessage: BaseMessage{Command: CmdError, OpId: 1},
		Code:        POSIX_EIO, // Valid non-zero code
		Message:     "AC could not process operation",
	}
	payload, err := msgpack.Marshal(&errMsg)
	require.NoError(t, err)

	handlerErr := server.handleInboundError(conn, session, 1, payload)
	assert.NoError(t, handlerErr)

	// Verify errorAck was sent (not error)
	var response map[string]interface{}
	require.NoError(t, decodeResponse(conn.written, &response))

	cmdVal, ok := response["command"]
	require.True(t, ok, "response must have command field")
	assert.Equal(t, CmdErrorAck, cmdVal, "valid error must receive errorAck response")
}
