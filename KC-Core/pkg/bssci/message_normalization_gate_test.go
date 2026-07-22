package bssci

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
)

// TestNormalizationOnlyForBStoSCCommands verifies BSSCI §2.4 direction gating:
// Only BS->SC (inbound) commands should be normalized; SC->BS (outbound) responses must skip normalization.
//
// BSSCI §2.4 requires forward-compatible message interpretation for INBOUND messages only.
// Normalizing our own SC→BS responses would incorrectly validate server-generated payloads,
// which was the root cause of Issue #3's normalization regression.
func TestNormalizationOnlyForBStoSCCommands(t *testing.T) {
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
		{mioty.CmdStatus, false, "SC→BS", "Status (SC initiates, BSSCI 5.5)"},
		{mioty.CmdStatusComplete, false, "SC→BS", "Status completion (SC sends)"},
		{mioty.CmdDLDataQueueComplete, false, "SC→BS", "DL queue completion (SC sends)"},

		// Bidirectional: Can be initiated by either party (normalize when received)
		{mioty.CmdPing, true, "Bidirectional", "Ping (can be initiated by either party)"},
		{mioty.CmdPingResponse, true, "Bidirectional", "Ping response (either party)"},
		{mioty.CmdError, true, "Bidirectional", "Error (can be initiated by either party)"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			assert.Equal(t, tt.shouldNormalize, shouldNormalizeCommand(tt.command),
				"Command %s (%s) normalize decision incorrect: %s", tt.command, tt.direction, tt.description)
		})
	}
}

// TestNormalizationSkipsUnknownCommands verifies commands absent from
// CommandDirectionMap default to NOT normalizing (safe fallback).
// This prevents accidentally normalizing future protocol extensions or vendor-specific commands.
func TestNormalizationSkipsUnknownCommands(t *testing.T) {
	assert.False(t, shouldNormalizeCommand("unknownFutureCommand"),
		"Unknown commands should default to skipping normalization (safe fallback)")
}
