// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
//
// SCACI protocol per Fraunhofer MIOTY SCACI Specification v1.0.0, Section 3.1
package scaci

import (
	"encoding/binary"
	"fmt"
	"math"
)

// MIOTYA01Identifier is the 8-byte ASCII identifier for SCACI frames per spec §3.1
var MIOTYA01Identifier = [8]byte{'M', 'I', 'O', 'T', 'Y', 'A', '0', '1'} // 0x4d 49 4f 54 59 41 30 31

// Frame represents the binary frame structure that wraps all SCACI messages
// per MIOTY SCACI v1.0.0 §3.1
//
// Frame format:
//
//	Identifier: 8 bytes - ASCII "MIOTYA01"
//	PayloadSize: 4 bytes - Little-endian unsigned integer
//	Payload: n bytes - JSON/MessagePack encoded message
type Frame struct {
	Identifier  [8]byte // ASCII "MIOTYA01" (0x4d 49 4f 54 59 41 30 31)
	PayloadSize uint32  // Size of following JSON/MessagePack object (little-endian)
	Payload     []byte  // The actual JSON or MessagePack encoded message
}

// MinFrameSize is the minimum valid frame size (header only, no payload)
const MinFrameSize = 12 // 8 bytes identifier + 4 bytes size

// MaxPayloadSize is the maximum allowed payload size (1MB safety limit)
const MaxPayloadSize = 1024 * 1024

// ParseFrameHeader parses just the SCACI frame header (12 bytes) per MIOTY SCACI v1.0.0 §3.1
//
// This is the first step in frame parsing - read header to get payload size,
// then read that many bytes for the payload.
//
// Parameters:
//   - data: 12-byte header buffer
//
// Returns:
//   - payloadSize: Size of payload to read next
//   - error: Validation error if header is malformed
//
// Errors:
//   - Header too short (< 12 bytes)
//   - Invalid frame identifier (not "MIOTYA01")
//   - Payload size too large (> MaxPayloadSize)
func ParseFrameHeader(data []byte) (uint32, error) {
	if len(data) < MinFrameSize {
		return 0, fmt.Errorf("header too short: need %d bytes, got %d", MinFrameSize, len(data))
	}

	// Validate identifier (8 bytes)
	var identifier [8]byte
	copy(identifier[:], data[0:8])
	if identifier != MIOTYA01Identifier {
		return 0, fmt.Errorf("invalid frame identifier: expected MIOTYA01, got %s", string(identifier[:]))
	}

	// Parse payload size (4 bytes, little-endian)
	payloadSize := binary.LittleEndian.Uint32(data[8:12])

	// Safety check: prevent memory exhaustion attacks
	if payloadSize > MaxPayloadSize {
		return 0, fmt.Errorf("payload size too large: %d bytes exceeds maximum %d bytes", payloadSize, MaxPayloadSize)
	}

	return payloadSize, nil
}

// Serialize converts a Frame to bytes per MIOTY SCACI v1.0.0 §3.1
//
// Returns:
//   - []byte: Serialized frame ready for transmission
//   - error: Error if payload size exceeds uint32 max
//
// Frame layout:
//
//	[0:8]   - Identifier: "MIOTYA01"
//	[8:12]  - PayloadSize: Little-endian uint32
//	[12:n]  - Payload: MessagePack/JSON encoded message
func (f *Frame) Serialize() ([]byte, error) {
	payloadLen := len(f.Payload)
	if payloadLen > math.MaxUint32 {
		return nil, fmt.Errorf("payload too large: %d bytes (max %d)", payloadLen, math.MaxUint32)
	}

	payloadSize := uint32(payloadLen)
	result := make([]byte, MinFrameSize+payloadLen)

	// Write identifier
	copy(result[0:8], MIOTYA01Identifier[:])

	// Write payload size (little-endian)
	binary.LittleEndian.PutUint32(result[8:12], payloadSize)

	// Write payload
	if payloadLen > 0 {
		copy(result[12:], f.Payload)
	}

	return result, nil
}

// NewFrame creates a new SCACI frame with the given payload
//
// Parameters:
//   - payload: MessagePack or JSON encoded message bytes
//
// Returns:
//   - *Frame: Frame ready for serialization
//   - error: Error if payload size exceeds uint32 max
func NewFrame(payload []byte) (*Frame, error) {
	payloadLen := len(payload)
	if payloadLen > math.MaxUint32 {
		return nil, fmt.Errorf("payload too large: %d bytes (max %d)", payloadLen, math.MaxUint32)
	}

	return &Frame{
		Identifier:  MIOTYA01Identifier,
		PayloadSize: uint32(payloadLen),
		Payload:     payload,
	}, nil
}
