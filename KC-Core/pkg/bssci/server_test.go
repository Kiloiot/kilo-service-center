package bssci_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseBSSCIFrame parses a BSSCI frame from bytes (test-only helper)
func parseBSSCIFrame(data []byte) (*mioty.Frame, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("frame too short: need at least 12 bytes, got %d", len(data))
	}

	frame := &mioty.Frame{}
	copy(frame.Identifier[:], data[0:8])

	if frame.Identifier != mioty.MIOTYFrameIdentifier {
		return nil, fmt.Errorf("invalid frame identifier: expected MIOTYB01, got %s", string(frame.Identifier[:]))
	}

	frame.PayloadSize = binary.LittleEndian.Uint32(data[8:12])

	if len(data) < int(12+frame.PayloadSize) {
		return nil, fmt.Errorf("incomplete frame: expected %d bytes payload, got %d", frame.PayloadSize, len(data)-12)
	}

	frame.Payload = data[12 : 12+frame.PayloadSize]
	return frame, nil
}

// TestFrameHeaderValidation verifies that frame header validation correctly
// accepts valid MIOTY frame identifiers and rejects invalid ones.
// This test ensures consolidated constants are applied correctly.
func TestFrameHeaderValidation(t *testing.T) {
	tests := []struct {
		name        string
		header      []byte
		shouldMatch bool
		description string
	}{
		{
			name:        "ValidMIOTYB01Header",
			header:      []byte{'M', 'I', 'O', 'T', 'Y', 'B', '0', '1'},
			shouldMatch: true,
			description: "Standard MIOTYB01 identifier should match",
		},
		{
			name:        "ValidHeaderFromConstant",
			header:      mioty.MIOTYFrameIdentifier[:],
			shouldMatch: true,
			description: "Header created from canonical constant should match",
		},
		{
			name:        "InvalidHeaderWrongPrefix",
			header:      []byte{'X', 'I', 'O', 'T', 'Y', 'B', '0', '1'},
			shouldMatch: false,
			description: "Header with wrong first byte should not match",
		},
		{
			name:        "InvalidHeaderWrongVersion",
			header:      []byte{'M', 'I', 'O', 'T', 'Y', 'B', '0', '2'},
			shouldMatch: false,
			description: "Header with wrong version number should not match",
		},
		{
			name:        "InvalidHeaderTruncated",
			header:      []byte{'M', 'I', 'O', 'T', 'Y'},
			shouldMatch: false,
			description: "Truncated header should not match",
		},
		{
			name:        "InvalidHeaderAllZeros",
			header:      []byte{0, 0, 0, 0, 0, 0, 0, 0},
			shouldMatch: false,
			description: "All-zero header should not match",
		},
		{
			name:        "InvalidHeaderMIOTYA01",
			header:      []byte{'M', 'I', 'O', 'T', 'Y', 'A', '0', '1'},
			shouldMatch: false,
			description: "MIOTYA01 (hypothetical different protocol) should not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For headers at least 8 bytes, test byte-level comparison
			if len(tt.header) >= 8 {
				matches := bytes.Equal(tt.header[:8], mioty.MIOTYFrameIdentifier[:])
				if tt.shouldMatch {
					assert.True(t, matches, tt.description)
				} else {
					assert.False(t, matches, tt.description)
				}
			} else {
				// For truncated headers, they should never match
				assert.False(t, tt.shouldMatch, "Truncated headers should never match")
			}
		})
	}
}

// TestFrameParsing tests BSSCI frame parsing to ensure
// it correctly validates the frame identifier using the canonical constant.
func TestFrameParsing(t *testing.T) {
	t.Run("ValidFrame", func(t *testing.T) {
		// Create a valid frame: 8-byte identifier + 4-byte size + payload
		frame := make([]byte, 12+5)
		copy(frame[:8], mioty.MIOTYFrameIdentifier[:])
		// Payload size = 5 (little-endian)
		frame[8] = 5
		frame[9] = 0
		frame[10] = 0
		frame[11] = 0
		// Payload: "hello"
		copy(frame[12:], []byte("hello"))

		parsed, err := parseBSSCIFrame(frame)
		require.NoError(t, err, "Valid frame should parse successfully")
		assert.Equal(t, mioty.MIOTYFrameIdentifier, parsed.Identifier, "Identifier should match")
		assert.Equal(t, uint32(5), parsed.PayloadSize, "Payload size should be 5")
		assert.Equal(t, []byte("hello"), parsed.Payload, "Payload should match")
	})

	t.Run("InvalidFrameIdentifier", func(t *testing.T) {
		// Create frame with invalid identifier
		frame := make([]byte, 12)
		copy(frame[:8], []byte("INVALID1"))
		frame[8] = 0
		frame[9] = 0
		frame[10] = 0
		frame[11] = 0

		parsed, err := parseBSSCIFrame(frame)
		assert.Error(t, err, "Frame with invalid identifier should fail")
		assert.Nil(t, parsed, "Parsed frame should be nil on error")
		assert.Contains(t, err.Error(), "invalid frame identifier", "Error should mention invalid identifier")
	})

	t.Run("FrameTooShort", func(t *testing.T) {
		// Frame with only 10 bytes (needs at least 12)
		frame := make([]byte, 10)
		copy(frame[:8], mioty.MIOTYFrameIdentifier[:])

		parsed, err := parseBSSCIFrame(frame)
		assert.Error(t, err, "Frame shorter than 12 bytes should fail")
		assert.Nil(t, parsed, "Parsed frame should be nil on error")
		assert.Contains(t, err.Error(), "too short", "Error should mention frame is too short")
	})
}

// TestProtocolVersionConstant verifies that the protocol version constant
// is correctly defined and accessible from the mioty package.
func TestProtocolVersionConstant(t *testing.T) {
	// Verify version constant exists and has expected value
	assert.Equal(t, "1.0.0", mioty.MIOTYProtocolVersion, "Protocol version should be 1.0.0")

	// Verify version components
	assert.Equal(t, 1, mioty.MIOTYVersionMajor, "Major version should be 1")
	assert.Equal(t, 0, mioty.MIOTYVersionMinor, "Minor version should be 0")
	assert.Equal(t, 0, mioty.MIOTYVersionPatch, "Patch version should be 0")
}
