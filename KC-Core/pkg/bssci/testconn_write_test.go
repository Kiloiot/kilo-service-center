package bssci

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	mioty "github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
)

// TestTestConnTwoWritePattern verifies TestConn correctly handles BSSCI two-write pattern
// This is a diagnostic test for Issue #3-4 Fix A2
func TestTestConnTwoWritePattern(t *testing.T) {
	conn := &testutil.TestConn{Encoding: "msgpack"}

	// Create error message exactly like sendError() does
	errorMsg := mioty.Error{
		BaseMessage: mioty.BaseMessage{
			CommandType: mioty.CmdError,
			OpId:        123,
		},
		Code:    71, // POSIX_EPROTO
		Message: "test error message",
	}

	// Encode the message
	payload, err := msgpack.Marshal(errorMsg)
	assert.NoError(t, err, "Marshal should succeed")

	// Create header (12 bytes: MIOTYB01 + 4-byte size)
	header := make([]byte, 12)
	copy(header[:8], mioty.MIOTYFrameIdentifier[:])
	// Put payload size in little-endian
	payloadSize := uint32(len(payload))
	header[8] = byte(payloadSize)
	header[9] = byte(payloadSize >> 8)
	header[10] = byte(payloadSize >> 16)
	header[11] = byte(payloadSize >> 24)

	// First write: header only (Pattern 1)
	n1, err1 := conn.Write(header)
	assert.NoError(t, err1, "Header write should succeed")
	assert.Equal(t, 12, n1, "Should write 12 bytes")
	assert.Equal(t, 0, len(conn.SentMessages), "Header-only write should not add to SentMessages")
	assert.NotNil(t, conn.PendingHeader, "Should buffer header")

	// Second write: payload only (Pattern 3)
	n2, err2 := conn.Write(payload)
	assert.NoError(t, err2, "Payload write should succeed")
	assert.Equal(t, len(payload), n2, "Should write all payload bytes")
	assert.Equal(t, 1, len(conn.SentMessages), "Payload write should decode and add to SentMessages")
	assert.Nil(t, conn.PendingHeader, "Should clear pending header")

	// Verify captured message
	captured := conn.SentMessages[0]
	assert.Equal(t, "error", captured["command"], "Command should be 'error'")
	assert.Equal(t, int64(123), captured["opId"], "OpId should be 123")

	// Check error code extraction
	code, msg := conn.LastError()
	assert.Equal(t, 71, code, "LastError should return code 71")
	assert.Equal(t, "test error message", msg, "LastError should return message")
}

// TestTestConnSingleWritePattern verifies combined header+payload write (Pattern 2)
func TestTestConnSingleWritePattern(t *testing.T) {
	conn := &testutil.TestConn{Encoding: "msgpack"}

	// Create error message
	errorMsg := mioty.Error{
		BaseMessage: mioty.BaseMessage{
			CommandType: mioty.CmdError,
			OpId:        456,
		},
		Code:    22, // POSIX_EINVAL
		Message: "single write test",
	}

	// Encode the message
	payload, err := msgpack.Marshal(errorMsg)
	assert.NoError(t, err, "Marshal should succeed")

	// Create combined buffer: header + payload
	combined := make([]byte, 12+len(payload))
	copy(combined[:8], mioty.MIOTYFrameIdentifier[:])
	payloadSize := uint32(len(payload))
	combined[8] = byte(payloadSize)
	combined[9] = byte(payloadSize >> 8)
	combined[10] = byte(payloadSize >> 16)
	combined[11] = byte(payloadSize >> 24)
	copy(combined[12:], payload)

	// Single write: header + payload (Pattern 2)
	n, writeErr := conn.Write(combined)
	assert.NoError(t, writeErr, "Combined write should succeed")
	assert.Equal(t, len(combined), n, "Should write all bytes")
	assert.Equal(t, 1, len(conn.SentMessages), "Should decode and add to SentMessages")
	assert.Nil(t, conn.PendingHeader, "Should not have pending header")

	// Verify captured message
	code, msg := conn.LastError()
	assert.Equal(t, 22, code, "LastError should return code 22")
	assert.Equal(t, "single write test", msg, "LastError should return message")
}
