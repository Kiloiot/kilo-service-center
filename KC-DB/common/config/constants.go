// Package config provides centralized configuration constants for KiloCenter
// These constants should be used instead of hardcoding values throughout the codebase
package config

import (
	"time"
)

// Database Configuration
const (
	// Query timeouts
	DefaultQueryTimeout = 30 * time.Second
)

// MIOTY Protocol Constants
const (
	// Note: BSSCI Protocol Version moved to KC-DB/storage/mioty/types.go (MIOTYProtocolVersion)

	// Ping mechanism
	PingInterval = 60 * time.Second

	// BSSCI Session Management (Section 3.3)
	// Maximum age for session resumption eligibility
	// After this duration, Base Stations must initiate new sessions
	MaxSessionResumptionAge = 24 * time.Hour

	// Max message sizes
	MaxMessageSize = 1024 * 1024 // 1MB

	// EUI sizes
	EUISize = 8 // 8 bytes for EUI64

	// SessionKeySize is the MIOTY network/application session key length in
	// bytes (nwkSnKey/appSnKey wire arrays are Numeric[16]).
	SessionKeySize = 16

	// Attach operation field sizes (BSSCI §3.6): nonce and signature are
	// 4-byte arrays on the wire.
	AttachNonceSize = 4
	AttachSignSize  = 4

	// AttachCounterMax is the highest attachment counter value the 24-bit
	// wire field can carry (BSSCI §3.6).
	AttachCounterMax = 0xFFFFFF

	// Attachment counter rollover window (BSSCI §3.6): a stored counter near
	// the 24-bit ceiling followed by a low received counter is a rollover,
	// not a replay.
	AttachCounterRolloverHigh = 0xFFFF00
	AttachCounterRolloverLow  = 0x100
)
