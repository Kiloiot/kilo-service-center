package bssci

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	mioty "github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
)

// TestTestConnSingleWritePattern verifies the combined header+payload write the codec emits
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

	// Single write: header + payload
	n, writeErr := conn.Write(combined)
	assert.NoError(t, writeErr, "Combined write should succeed")
	assert.Equal(t, len(combined), n, "Should write all bytes")
	assert.Equal(t, 1, len(conn.SentMessages), "Should decode and add to SentMessages")

	// Verify captured message
	code, msg := conn.LastError()
	assert.Equal(t, 22, code, "LastError should return code 22")
	assert.Equal(t, "single write test", msg, "LastError should return message")
}
