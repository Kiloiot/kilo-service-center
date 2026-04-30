package bssci

import (
	"testing"

	"github.com/kilocenter/KC-Core/pkg/bssci/testutil"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
)

// TestNormalizationOnlyForBStoSCCommands verifies BSSCI §2.4 direction gating:
// Only BS->SC (inbound) commands should be normalized; SC->BS (outbound) responses must skip normalization.
//
// BSSCI §2.4 requires forward-compatible message interpretation for INBOUND messages only.
// Normalizing our own SC→BS responses would incorrectly validate server-generated payloads,
// which was the root cause of Issue #3's normalization regression.
func TestNormalizationOnlyForBStoSCCommands(t *testing.T) {
	testLogger := logger.NewNop()

	tests := []struct {
		command         string
		shouldNormalize bool
		direction       string
		description     string
	}{
		// BS→SC: Inbound from base station (MUST normalize)
		{mioty.CmdAttach, true, "BS→SC", "Attach request from BS"},
		{mioty.CmdDetach, true, "BS→SC", "Detach request from BS"},
		{mioty.CmdULData, true, "BS→SC", "Uplink data from EP via BS"},
		{mioty.CmdDLDataResult, true, "BS→SC", "DL transmission result from BS"},
		{mioty.CmdConnect, true, "BS→SC", "Connect handshake from BS"},

		// SC→BS: Outbound to base station (MUST NOT normalize - they're our own responses)
		{mioty.CmdAttachPropagate, false, "SC→BS", "Attach propagate (SC initiates)"},
		{mioty.CmdDetachPropagate, false, "SC→BS", "Detach propagate (SC initiates)"},
		{mioty.CmdDLDataQueue, false, "SC→BS", "DL data queue (SC initiates)"},
		{mioty.CmdAttachResponse, false, "SC→BS", "Attach response to BS"},

		// Bidirectional: Can be initiated by either party (normalize when received)
		{mioty.CmdPing, true, "Bidirectional", "Ping (can be initiated by either party)"},
		{mioty.CmdStatus, true, "Bidirectional", "Status (can be initiated by either party)"},
		{mioty.CmdError, true, "Bidirectional", "Error (can be initiated by either party)"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			// Track normalization calls
			var spyCalled bool
			var spyCommand string
			var spyShouldNormalize bool

			// Create server with StatusService
			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
			server := NewTestServer(testLogger, mockStorage, nil, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc,
				broadcaster, queueSerializer, auditLogger, tenantResolver)

			// Inject test spy
			server.testNormalizationSpy = func(cmd string, normalized bool) {
				spyCalled = true
				spyCommand = cmd
				spyShouldNormalize = normalized
			}

			// Create minimal session with mock connection
			session := &Session{
				ID:                "test-session",
				DbSessionID:       1,
				Encoding:          "json",
				HandshakeComplete: true,                 // Allow non-handshake commands
				Conn:              &testutil.TestConn{}, // Mock connection to prevent nil pointer crash
			}

			// Build valid minimal payload for this command
			msg := &Message{
				Command: tt.command,
				OpId:    int64(100),
			}
			data := buildMinimalPayloadForCommand(tt.command)

			// Call handleMessage (will invoke spy before normalization)
			_ = server.handleMessage(session, msg, data)

			// Verify spy was called with correct parameters
			assert.True(t, spyCalled, "testNormalizationSpy should be called")
			assert.Equal(t, tt.command, spyCommand, "Spy should receive correct command")
			assert.Equal(t, tt.shouldNormalize, spyShouldNormalize,
				"Command %s (%s) normalize decision incorrect", tt.command, tt.direction)
		})
	}
}

// buildMinimalPayloadForCommand creates a minimal valid payload for testing normalization gate
// This ensures handleMessage doesn't crash on normalization errors before spy is called
func buildMinimalPayloadForCommand(command string) map[string]interface{} {
	// Base payload with command and opId
	payload := map[string]interface{}{
		"command": command,
		"opId":    int64(100),
	}

	// Add command-specific required fields
	switch command {
	case mioty.CmdAttach:
		payload["epEui"] = uint64(TestEpEui01)
		payload["rxTime"] = int64(1234567890)
		payload["attachCnt"] = uint32(1)
		payload["snr"] = float64(10.0)
		payload["rssi"] = float64(-80.0)
		payload["nonce"] = []byte{0x01, 0x02, 0x03, 0x04}
		payload["sign"] = []byte{0x05, 0x06, 0x07, 0x08}
		payload["dualChan"] = false
		payload["repetition"] = false
		payload["wideCarrOff"] = false
		payload["longBlkDist"] = false
	case mioty.CmdDetach:
		payload["epEui"] = uint64(TestEpEui01)
		payload["rxTime"] = int64(1234567890)
		payload["packetCnt"] = uint32(42)
		payload["snr"] = float64(10.0)
		payload["rssi"] = float64(-80.0)
		payload["sign"] = []byte{0x01, 0x02, 0x03, 0x04}
	case mioty.CmdULData:
		payload["epEui"] = uint64(TestEpEui01)
		payload["rxTime"] = int64(1234567890)
		payload["packetCnt"] = uint32(42)
		payload["userData"] = []byte{0xAA, 0xBB}
		payload["snr"] = float64(10.0)
		payload["rssi"] = float64(-80.0)
		payload["dlOpen"] = false
		payload["responseExp"] = false
		payload["dlAck"] = false
	case mioty.CmdDLDataResult:
		payload["epEui"] = uint64(TestEpEui01)
		payload["queId"] = uint64(12345)
		payload["result"] = "sent"
		payload["txTime"] = int64(1234567890)
		payload["packetCnt"] = uint32(42)
	case mioty.CmdConnect:
		payload["version"] = "1.0.0"
		payload["bsEui"] = uint64(TestBsEui01)
		payload["bidi"] = true
		payload["snBsUuid"] = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	default:
		// For commands without strict normalization requirements (responses, etc.)
		// Just return base payload - they won't be normalized anyway
	}

	return payload
}

// TestNormalizationSkipsUnknownCommands verifies safe fallback behavior:
// Commands not in CommandDirectionMap should default to skipping normalization.
// This prevents accidentally normalizing future protocol extensions or vendor-specific commands.
func TestNormalizationSkipsUnknownCommands(t *testing.T) {
	testLogger := logger.NewNop()

	// Track normalization calls
	var spyCalled bool
	var spyCommand string
	var spyShouldNormalize bool

	// Create server with StatusService
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc,
		broadcaster, queueSerializer, auditLogger, tenantResolver)

	// Inject test spy
	server.testNormalizationSpy = func(cmd string, normalized bool) {
		spyCalled = true
		spyCommand = cmd
		spyShouldNormalize = normalized
	}

	// Create minimal session
	session := &Session{
		ID:                "test-session",
		DbSessionID:       1,
		Encoding:          "json",
		HandshakeComplete: true,
		Conn:              &testutil.TestConn{},
	}

	// Test with unknown command
	unknownCmd := "unknownFutureCommand"
	msg := &Message{
		Command: unknownCmd,
		OpId:    int64(100),
	}
	data := map[string]interface{}{
		"command": unknownCmd,
		"opId":    int64(100),
	}

	// Call handleMessage
	_ = server.handleMessage(session, msg, data)

	// Verify unknown commands default to NOT normalizing (safe fallback)
	assert.True(t, spyCalled, "testNormalizationSpy should be called")
	assert.Equal(t, unknownCmd, spyCommand, "Spy should receive unknown command")
	assert.False(t, spyShouldNormalize, "Unknown commands should default to skipping normalization (safe fallback)")
}
