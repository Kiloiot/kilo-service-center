// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseFrame parses a SCACI frame from bytes (test-only helper)
func parseFrame(data []byte) (*Frame, error) {
	if len(data) < MinFrameSize {
		return nil, fmt.Errorf("frame too short: need at least %d bytes, got %d", MinFrameSize, len(data))
	}

	frame := &Frame{}
	copy(frame.Identifier[:], data[0:8])

	if frame.Identifier != MIOTYA01Identifier {
		return nil, fmt.Errorf("invalid frame identifier: expected MIOTYA01, got %s", string(frame.Identifier[:]))
	}

	frame.PayloadSize = binary.LittleEndian.Uint32(data[8:12])

	if frame.PayloadSize > MaxPayloadSize {
		return nil, fmt.Errorf("payload size too large: %d bytes exceeds maximum %d bytes", frame.PayloadSize, MaxPayloadSize)
	}

	totalSize := MinFrameSize + int(frame.PayloadSize)
	if len(data) < totalSize {
		return nil, fmt.Errorf("incomplete frame: expected %d bytes payload (total %d bytes), got %d total bytes",
			frame.PayloadSize, totalSize, len(data))
	}

	if frame.PayloadSize > 0 {
		frame.Payload = data[MinFrameSize:totalSize]
	}

	return frame, nil
}

// TestFrameHeaderValidation validates MIOTYA01 identifier acceptance and rejection per §3.1
func TestFrameHeaderValidation(t *testing.T) {
	tests := []struct {
		name       string
		identifier [8]byte
		wantErr    bool
		errContain string
	}{
		{
			name:       "valid_MIOTYA01",
			identifier: MIOTYA01Identifier,
			wantErr:    false,
		},
		{
			name:       "invalid_MIOTYA02",
			identifier: [8]byte{'M', 'I', 'O', 'T', 'Y', 'A', '0', '2'},
			wantErr:    true,
			errContain: "invalid frame identifier",
		},
		{
			name:       "invalid_lowercase",
			identifier: [8]byte{'m', 'i', 'o', 't', 'y', 'a', '0', '1'},
			wantErr:    true,
			errContain: "invalid frame identifier",
		},
		{
			name:       "invalid_random_bytes",
			identifier: [8]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07},
			wantErr:    true,
			errContain: "invalid frame identifier",
		},
		{
			name:       "invalid_BSSCI_identifier",
			identifier: [8]byte{'M', 'I', 'O', 'T', 'Y', 'B', '0', '1'},
			wantErr:    true,
			errContain: "invalid frame identifier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build header: 8-byte identifier + 4-byte LE size (0)
			header := make([]byte, MinFrameSize)
			copy(header[0:8], tt.identifier[:])
			binary.LittleEndian.PutUint32(header[8:12], 0)

			payloadSize, err := ParseFrameHeader(header)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
			} else {
				require.NoError(t, err)
				assert.Equal(t, uint32(0), payloadSize)
			}
		})
	}
}

// TestFramePayloadSizeEncoding validates little-endian payload size encoding per §3.1
func TestFramePayloadSizeEncoding(t *testing.T) {
	tests := []struct {
		name        string
		payloadSize uint32
		expectedLE  []byte // Expected little-endian bytes
	}{
		{
			name:        "size_1",
			payloadSize: 1,
			expectedLE:  []byte{0x01, 0x00, 0x00, 0x00},
		},
		{
			name:        "size_256",
			payloadSize: 256,
			expectedLE:  []byte{0x00, 0x01, 0x00, 0x00},
		},
		{
			name:        "size_65536",
			payloadSize: 65536,
			expectedLE:  []byte{0x00, 0x00, 0x01, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create payload of specified size
			payload := make([]byte, tt.payloadSize)

			// Create and serialize frame
			frame, err := NewFrame(payload)
			require.NoError(t, err)

			data, err := frame.Serialize()
			require.NoError(t, err)

			// Verify MIOTYA01 header
			assert.Equal(t, MIOTYA01Identifier[:], data[0:8], "MIOTYA01 identifier mismatch")

			// Verify little-endian size bytes
			assert.Equal(t, tt.expectedLE, data[8:12], "LE payload size mismatch")

			// Round-trip: parse header and verify size
			parsedSize, err := ParseFrameHeader(data[:MinFrameSize])
			require.NoError(t, err)
			assert.Equal(t, tt.payloadSize, parsedSize)
		})
	}
}

// TestFrameBoundaries validates payload size boundary conditions per §3.1
func TestFrameBoundaries(t *testing.T) {
	t.Run("empty_payload", func(t *testing.T) {
		payload := []byte{}

		frame, err := NewFrame(payload)
		require.NoError(t, err)
		assert.Equal(t, uint32(0), frame.PayloadSize)

		data, err := frame.Serialize()
		require.NoError(t, err)
		assert.Len(t, data, MinFrameSize) // Header only

		// Parse back
		parsed, err := parseFrame(data)
		require.NoError(t, err)
		assert.Len(t, parsed.Payload, 0)
	})

	t.Run("at_max_payload", func(t *testing.T) {
		// Create payload exactly at MaxPayloadSize
		payload := make([]byte, MaxPayloadSize)
		for i := range payload {
			payload[i] = byte(i % 256)
		}

		frame, err := NewFrame(payload)
		require.NoError(t, err)
		assert.Equal(t, uint32(MaxPayloadSize), frame.PayloadSize)

		data, err := frame.Serialize()
		require.NoError(t, err)
		assert.Len(t, data, MinFrameSize+MaxPayloadSize)

		// Parse header should accept
		parsedSize, err := ParseFrameHeader(data[:MinFrameSize])
		require.NoError(t, err)
		assert.Equal(t, uint32(MaxPayloadSize), parsedSize)
	})

	t.Run("over_max_ParseHeader", func(t *testing.T) {
		// Craft header with size > MaxPayloadSize
		header := make([]byte, MinFrameSize)
		copy(header[0:8], MIOTYA01Identifier[:])
		binary.LittleEndian.PutUint32(header[8:12], MaxPayloadSize+1)

		_, err := ParseFrameHeader(header)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "payload size too large")
	})

	t.Run("over_max_ParseFrame", func(t *testing.T) {
		// Craft frame data with declared size > MaxPayloadSize
		data := make([]byte, MinFrameSize+100) // Some extra bytes
		copy(data[0:8], MIOTYA01Identifier[:])
		binary.LittleEndian.PutUint32(data[8:12], MaxPayloadSize+1)

		_, err := parseFrame(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "payload size too large")
	})
}

// TestFrameTruncation validates handling of truncated frames per §3.1
func TestFrameTruncation(t *testing.T) {
	t.Run("short_header_11_bytes", func(t *testing.T) {
		header := make([]byte, 11) // Less than MinFrameSize
		copy(header[0:8], MIOTYA01Identifier[:])

		_, err := ParseFrameHeader(header)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "header too short")
	})

	t.Run("short_header_8_bytes", func(t *testing.T) {
		header := make([]byte, 8) // Identifier only
		copy(header[0:8], MIOTYA01Identifier[:])

		_, err := ParseFrameHeader(header)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "header too short")
	})

	t.Run("short_header_0_bytes", func(t *testing.T) {
		header := []byte{}

		_, err := ParseFrameHeader(header)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "header too short")
	})

	t.Run("declared_exceeds_available", func(t *testing.T) {
		// Header declares 100 bytes payload, but only 50 available
		data := make([]byte, MinFrameSize+50)
		copy(data[0:8], MIOTYA01Identifier[:])
		binary.LittleEndian.PutUint32(data[8:12], 100) // Claim 100 bytes

		_, err := parseFrame(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incomplete frame")
	})

	t.Run("declared_exceeds_available_exact", func(t *testing.T) {
		// Header declares 100 bytes payload, but only 99 available
		data := make([]byte, MinFrameSize+99)
		copy(data[0:8], MIOTYA01Identifier[:])
		binary.LittleEndian.PutUint32(data[8:12], 100)

		_, err := parseFrame(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incomplete frame")
	})
}

// TestFrameRoundTrip validates NewFrame → Serialize → ParseFrame integrity
func TestFrameRoundTrip(t *testing.T) {
	tests := []struct {
		name        string
		payloadSize int
	}{
		{
			name:        "small_payload",
			payloadSize: 16,
		},
		{
			name:        "medium_payload",
			payloadSize: 4096,
		},
		{
			name:        "large_payload",
			payloadSize: 65536,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create deterministic payload
			originalPayload := make([]byte, tt.payloadSize)
			for i := range originalPayload {
				originalPayload[i] = byte(i % 256)
			}

			// Create frame
			frame, err := NewFrame(originalPayload)
			require.NoError(t, err)
			assert.Equal(t, MIOTYA01Identifier, frame.Identifier)
			assert.Equal(t, uint32(tt.payloadSize), frame.PayloadSize)

			// Serialize
			data, err := frame.Serialize()
			require.NoError(t, err)
			assert.Len(t, data, MinFrameSize+tt.payloadSize)

			// Parse back
			parsed, err := parseFrame(data)
			require.NoError(t, err)

			// Verify integrity
			assert.Equal(t, MIOTYA01Identifier, parsed.Identifier)
			assert.Equal(t, uint32(tt.payloadSize), parsed.PayloadSize)
			assert.Equal(t, originalPayload, parsed.Payload)
		})
	}
}

// TestFrameServerSendReceive validates bidirectional frame exchange using net.Pipe per §3.1
func TestFrameServerSendReceive(t *testing.T) {
	t.Run("bidirectional", func(t *testing.T) {
		// Create pipe for bidirectional communication
		scConn, acConn := net.Pipe()
		defer func() { _ = scConn.Close() }()
		defer func() { _ = acConn.Close() }()

		// Test payloads
		scPayload := []byte(`{"command":"connectResponse","opId":0}`)
		acPayload := []byte(`{"command":"connect","opId":0}`)

		// Channel for errors
		errCh := make(chan error, 2)

		// SC → AC direction
		go func() {
			frame, err := NewFrame(scPayload)
			if err != nil {
				errCh <- err
				return
			}
			data, err := frame.Serialize()
			if err != nil {
				errCh <- err
				return
			}
			_, err = scConn.Write(data)
			errCh <- err
		}()

		// Read on AC side
		go func() {
			// Read header
			header := make([]byte, MinFrameSize)
			if _, err := io.ReadFull(acConn, header); err != nil {
				errCh <- err
				return
			}

			// Verify MIOTYA01 header bytes explicitly
			assert.Equal(t, MIOTYA01Identifier[:], header[0:8], "SC→AC: MIOTYA01 identifier mismatch")

			// Verify LE size
			expectedSize := uint32(len(scPayload))
			actualSize := binary.LittleEndian.Uint32(header[8:12])
			assert.Equal(t, expectedSize, actualSize, "SC→AC: LE payload size mismatch")

			// Read payload
			payload := make([]byte, actualSize)
			if _, err := io.ReadFull(acConn, payload); err != nil {
				errCh <- err
				return
			}
			assert.Equal(t, scPayload, payload, "SC→AC: payload mismatch")

			errCh <- nil
		}()

		// Wait for SC→AC
		require.NoError(t, <-errCh)
		require.NoError(t, <-errCh)

		// AC → SC direction
		go func() {
			frame, err := NewFrame(acPayload)
			if err != nil {
				errCh <- err
				return
			}
			data, err := frame.Serialize()
			if err != nil {
				errCh <- err
				return
			}
			_, err = acConn.Write(data)
			errCh <- err
		}()

		// Read on SC side
		go func() {
			// Read header
			header := make([]byte, MinFrameSize)
			if _, err := io.ReadFull(scConn, header); err != nil {
				errCh <- err
				return
			}

			// Verify MIOTYA01 header bytes explicitly
			assert.Equal(t, MIOTYA01Identifier[:], header[0:8], "AC→SC: MIOTYA01 identifier mismatch")

			// Verify LE size
			expectedSize := uint32(len(acPayload))
			actualSize := binary.LittleEndian.Uint32(header[8:12])
			assert.Equal(t, expectedSize, actualSize, "AC→SC: LE payload size mismatch")

			// Read payload
			payload := make([]byte, actualSize)
			if _, err := io.ReadFull(scConn, payload); err != nil {
				errCh <- err
				return
			}
			assert.Equal(t, acPayload, payload, "AC→SC: payload mismatch")

			errCh <- nil
		}()

		// Wait for AC→SC
		require.NoError(t, <-errCh)
		require.NoError(t, <-errCh)
	})

	t.Run("multiple_frames_sequential", func(t *testing.T) {
		scConn, acConn := net.Pipe()
		defer func() { _ = scConn.Close() }()
		defer func() { _ = acConn.Close() }()

		payloads := [][]byte{
			[]byte(`{"command":"ping","opId":1}`),
			[]byte(`{"command":"ping","opId":2}`),
			[]byte(`{"command":"ping","opId":3}`),
		}

		// Send multiple frames
		go func() {
			for i, payload := range payloads {
				frame, err := NewFrame(payload)
				require.NoError(t, err, "frame %d NewFrame", i)

				data, err := frame.Serialize()
				require.NoError(t, err, "frame %d Serialize", i)

				_, err = scConn.Write(data)
				require.NoError(t, err, "frame %d Write", i)
			}
		}()

		// Read and verify each frame
		for i, expectedPayload := range payloads {
			header := make([]byte, MinFrameSize)
			_, err := io.ReadFull(acConn, header)
			require.NoError(t, err, "frame %d header read", i)

			size, err := ParseFrameHeader(header)
			require.NoError(t, err, "frame %d header parse", i)

			payload := make([]byte, size)
			_, err = io.ReadFull(acConn, payload)
			require.NoError(t, err, "frame %d payload read", i)

			assert.Equal(t, expectedPayload, payload, "frame %d payload mismatch", i)
		}
	})
}

// TestFramePartialIdentifier validates rejection of partial/truncated identifiers
func TestFramePartialIdentifier(t *testing.T) {
	tests := []struct {
		name   string
		header []byte
	}{
		{
			name:   "7_bytes_identifier",
			header: []byte{'M', 'I', 'O', 'T', 'Y', 'A', '0'},
		},
		{
			name:   "5_bytes_identifier",
			header: []byte{'M', 'I', 'O', 'T', 'Y'},
		},
		{
			name:   "1_byte_identifier",
			header: []byte{'M'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFrameHeader(tt.header)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "header too short")
		})
	}
}

// TestFrameSerializeIdentifierBytes verifies exact MIOTYA01 byte sequence
func TestFrameSerializeIdentifierBytes(t *testing.T) {
	frame, err := NewFrame([]byte("test"))
	require.NoError(t, err)

	data, err := frame.Serialize()
	require.NoError(t, err)

	// Verify exact byte sequence per spec §3.1
	expectedIdentifier := []byte{0x4d, 0x49, 0x4f, 0x54, 0x59, 0x41, 0x30, 0x31} // "MIOTYA01"
	assert.Equal(t, expectedIdentifier, data[0:8], "MIOTYA01 exact bytes")

	// Also verify via bytes.Equal for clarity
	assert.True(t, bytes.Equal([]byte("MIOTYA01"), data[0:8]))
}
