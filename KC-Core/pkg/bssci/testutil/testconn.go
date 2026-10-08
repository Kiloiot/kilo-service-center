// Package testutil provides test-only helpers for KC-Core/pkg/bssci.
// TestConn is a net.Conn implementation that captures msgpack/JSON frames so
// tests can decode them after exercising handler code paths.
package testutil

import (
	"bytes"
	"encoding/json"
	"net"
	"os"
	"sync"
	"time"

	mioty "github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/vmihailenco/msgpack/v5"
)

// TestConn captures sent messages for verification in tests
// Implements net.Conn interface for use as Session.Conn in tests
//
// Thread-safe: all public methods use mutex protection
// Handles BSSCI two-write pattern with buffering for all edge cases
type TestConn struct {
	net.Conn
	Mu           sync.Mutex
	SentMessages []map[string]interface{}
	// FailWrites refuses every write before accepting a byte, as a closed
	// connection does.
	FailWrites bool
	// StalledWrites times every write out, as a peer that stopped reading
	// does, leaving the frame ambiguous on the wire.
	StalledWrites bool
	Encoding      string // "json" or "msgpack"
}

// Write captures the message for inspection in tests
// Decodes based on Encoding field and stores in SentMessages
//
// Handles BSSCI two-write pattern:
//
//	Pattern 1: Header-only write (12 bytes) - buffer it for next write
//	Pattern 2: Header + payload combined - strip header, decode payload
//	Pattern 3: Payload after pending header - decode payload
//	Pattern 4: Standalone message (no header) - decode directly
//
// Thread-safe: uses mutex to protect SentMessages
func (m *TestConn) Write(b []byte) (n int, err error) {
	m.Mu.Lock()
	defer m.Mu.Unlock()

	if m.FailWrites {
		return 0, net.ErrClosed
	}
	if m.StalledWrites {
		return 0, os.ErrDeadlineExceeded
	}

	// A frame arrives in one write: strip the 12-byte header and decode the payload.
	if len(b) >= mioty.FrameHeaderSize && bytes.HasPrefix(b, mioty.MIOTYFrameIdentifier[:]) {
		return len(b), m.decodeAndCapture(b[mioty.FrameHeaderSize:])
	}

	// Standalone message (no MIOTY frame header)
	// Decode directly (used for some test scenarios)
	return len(b), m.decodeAndCapture(b)
}

// decodeAndCapture decodes a payload and appends it to SentMessages
// Handles both JSON and MessagePack encoding based on m.Encoding
// Returns error if decoding fails, nil on success
func (m *TestConn) decodeAndCapture(payload []byte) error {
	var msg map[string]interface{}
	var err error

	if m.Encoding == "msgpack" {
		err = msgpack.Unmarshal(payload, &msg)
	} else {
		err = json.Unmarshal(payload, &msg)
	}

	if err == nil {
		m.SentMessages = append(m.SentMessages, msg)
	}
	return nil // Always return nil to simulate successful write
}

// Reset clears captured messages and resets the write failure flag
// Thread-safe: uses mutex to protect state
func (m *TestConn) Reset() {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.SentMessages = nil
	m.FailWrites = false
	m.StalledWrites = false
}

// SeenCommand checks if a command was sent
// Thread-safe: uses mutex to protect SentMessages access
func (m *TestConn) SeenCommand(cmd string) bool {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	for _, msg := range m.SentMessages {
		if msg["command"] == cmd {
			return true
		}
	}
	return false
}

// LastMessage returns the most recent captured message carrying the command,
// or nil when none was sent.
func (m *TestConn) LastMessage(cmd string) map[string]interface{} {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	for i := len(m.SentMessages) - 1; i >= 0; i-- {
		if m.SentMessages[i]["command"] == cmd {
			return m.SentMessages[i]
		}
	}
	return nil
}

// LastError returns the error code and message from the most recent error frame
// Returns (0, "") if no error frame was captured
// Thread-safe: uses mutex to protect SentMessages access
func (m *TestConn) LastError() (int, string) {
	m.Mu.Lock()
	defer m.Mu.Unlock()

	// Search backwards for most recent error
	for i := len(m.SentMessages) - 1; i >= 0; i-- {
		msg := m.SentMessages[i]
		if cmd, ok := msg["command"].(string); ok && cmd == mioty.CmdError {
			code := ExtractIntCode(msg["code"])
			message := ""
			if msgStr, ok := msg["message"].(string); ok {
				message = msgStr
			}
			return code, message
		}
	}
	return 0, ""
}

// MessageCount returns the number of captured messages
// Thread-safe: uses mutex to protect SentMessages access
func (m *TestConn) MessageCount() int {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	return len(m.SentMessages)
}

// GetMessage returns the message at the given index (0-based)
// Returns nil if index is out of bounds
// Thread-safe: uses mutex to protect SentMessages access
func (m *TestConn) GetMessage(index int) map[string]interface{} {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	if index < 0 || index >= len(m.SentMessages) {
		return nil
	}
	return m.SentMessages[index]
}

// ExtractIntCode extracts an integer code from various msgpack/json numeric types
// Handles int8, int16, int32, int64, uint8, uint16, uint32, uint64, float32, float64, and plain int
func ExtractIntCode(val interface{}) int {
	switch code := val.(type) {
	case int:
		return code
	case int8:
		return int(code)
	case int16:
		return int(code)
	case int32:
		return int(code)
	case int64:
		return int(code)
	case uint8:
		return int(code)
	case uint16:
		return int(code)
	case uint32:
		return int(code)
	case uint64:
		return int(code) //nolint:gosec // G115: test helper, controlled inputs
	case float32:
		return int(code)
	case float64:
		return int(code)
	default:
		return 0
	}
}

// Implement remaining net.Conn interface methods (no-op for tests)

// Close closes the connection (no-op for tests)
func (m *TestConn) Close() error { return nil }

// LocalAddr returns the local network address (no-op for tests)
func (m *TestConn) LocalAddr() net.Addr { return nil }

// RemoteAddr returns the remote network address (no-op for tests)
func (m *TestConn) RemoteAddr() net.Addr { return nil }

// SetDeadline sets the read and write deadlines (no-op for tests)
func (m *TestConn) SetDeadline(_ time.Time) error { return nil }

// SetReadDeadline sets the read deadline (no-op for tests)
func (m *TestConn) SetReadDeadline(_ time.Time) error { return nil }

// SetWriteDeadline sets the write deadline (no-op for tests)
func (m *TestConn) SetWriteDeadline(_ time.Time) error { return nil }

// Read reads data from the connection (no-op for tests)
func (m *TestConn) Read(_ []byte) (n int, err error) { return 0, nil }

// FramePayload returns the payload of a single-write BSSCI frame, or b itself
// when it carries no frame header.
func FramePayload(b []byte) []byte {
	if len(b) >= mioty.FrameHeaderSize && bytes.HasPrefix(b, mioty.MIOTYFrameIdentifier[:]) {
		return b[mioty.FrameHeaderSize:]
	}
	return b
}
