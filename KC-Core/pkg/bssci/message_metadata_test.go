package bssci

import (
	"testing"

	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessageSpecCoverage verifies that all 56 BSSCI v1.0.0 commands
// have MessageSpec entries in the registry
func TestMessageSpecCoverage(t *testing.T) {
	// All 56 BSSCI v1.0.0 commands from types.go
	allCommands := []string{
		// Connection (§3.1)
		mioty.CmdConnect, mioty.CmdConnectResponse, mioty.CmdConnectComplete,
		// Ping (§3.2)
		mioty.CmdPing, mioty.CmdPingResponse, mioty.CmdPingComplete,
		// Status (§5.8)
		mioty.CmdStatus, mioty.CmdStatusResponse, mioty.CmdStatusComplete,
		// Attach (§5.6)
		mioty.CmdAttach, mioty.CmdAttachResponse, mioty.CmdAttachComplete,
		// Detach (§5.7)
		mioty.CmdDetach, mioty.CmdDetachResponse, mioty.CmdDetachComplete,
		// Attach Propagate (§5.9)
		mioty.CmdAttachPropagate, mioty.CmdAttachPropagateResponse, mioty.CmdAttachPropagateComplete,
		// Detach Propagate (§5.10)
		mioty.CmdDetachPropagate, mioty.CmdDetachPropagateResponse, mioty.CmdDetachPropagateComplete,
		// UL Data (§5.11)
		mioty.CmdULData, mioty.CmdULDataResponse, mioty.CmdULDataComplete,
		// UL Data Transmit (§5.11)
		mioty.CmdULDataTransmit, mioty.CmdULDataTransmitResponse, mioty.CmdULDataTransmitComplete,
		// DL Data Queue (§5.12)
		mioty.CmdDLDataQueue, mioty.CmdDLDataQueueResponse, mioty.CmdDLDataQueueComplete,
		// DL Data Revoke (§5.13)
		mioty.CmdDLDataRevoke, mioty.CmdDLDataRevokeResponse, mioty.CmdDLDataRevokeComplete,
		// DL Data Result (§5.14)
		mioty.CmdDLDataResult, mioty.CmdDLDataResultResponse, mioty.CmdDLDataResultComplete,
		// DL RX Status (§5.15)
		mioty.CmdDLRxStatus, mioty.CmdDLRxStatusResponse, mioty.CmdDLRxStatusComplete,
		// DL RX Status Query (§5.16)
		mioty.CmdDLRxStatusQuery, mioty.CmdDLRxStatusQueryResponse, mioty.CmdDLRxStatusQueryComplete,
		// Error Handling (§4)
		mioty.CmdError, mioty.CmdErrorAck,
		// VM Operations (§5.17-5.20)
		mioty.CmdVMActivate, mioty.CmdVMActivateResponse, mioty.CmdVMActivateComplete,
		mioty.CmdVMDeactivate, mioty.CmdVMDeactivateResponse, mioty.CmdVMDeactivateComplete,
		mioty.CmdVMStatus, mioty.CmdVMStatusResponse, mioty.CmdVMStatusComplete,
		mioty.CmdVMDLData, mioty.CmdVMDLDataResponse, mioty.CmdVMDLDataComplete,
	}

	// Verify count
	if len(allCommands) != 56 {
		t.Fatalf("Expected 56 commands, got %d", len(allCommands))
	}

	// Verify each command has a MessageSpec entry
	missing := []string{}
	for _, cmd := range allCommands {
		if _, exists := messageRegistry[cmd]; !exists {
			missing = append(missing, cmd)
		}
	}

	if len(missing) > 0 {
		t.Errorf("Missing MessageSpec entries for %d commands: %v", len(missing), missing)
	}

	// Verify registry count matches
	if len(messageRegistry) != 56 {
		t.Errorf("messageRegistry has %d entries, expected exactly 56", len(messageRegistry))
	}

	t.Logf("PASS: All 56 BSSCI commands have MessageSpec entries")
}

// TestMessageSpecFieldCompleteness verifies that commands with outbound fields
// have complete FieldSpec definitions (mandatory + optional)
func TestMessageSpecFieldCompleteness(t *testing.T) {
	// Commands that MUST have field specifications (17 commands with outbound data)
	// These are requests initiated by BS or SC that carry payload data
	commandsRequiringFields := map[string]struct {
		minMandatory int
		minOptional  int
		description  string
	}{
		// BS→SC requests (observed OTA data)
		mioty.CmdConnect: {4, 8, "BSSCI §3.1 - BS session metadata"},
		mioty.CmdAttach:  {11, 5, "BSSCI §5.6 - OTA attach with crypto"},
		mioty.CmdDetach:  {6, 4, "BSSCI §5.7 - OTA detach with signature"},
		mioty.CmdULData:  {9, 6, "BSSCI §5.11 - Uplink user data"},

		// BS→SC status reports
		mioty.CmdStatusResponse: {3, 7, "BSSCI §5.8 - BS operational status"},
		mioty.CmdDLDataResult:   {3, 3, "BSSCI §5.14 - Downlink transmission result"},
		mioty.CmdDLRxStatus:     {5, 0, "BSSCI §5.15 - EP downlink reception metrics"},

		// SC→BS commands (queue operations)
		mioty.CmdAttachPropagate: {9, 0, "BSSCI §5.9 - SC-initiated attach"},
		mioty.CmdDetachPropagate: {2, 0, "BSSCI §5.10 - SC-initiated detach"},
		mioty.CmdDLDataQueue:     {4, 7, "BSSCI §5.12 - Queue downlink data"},
		mioty.CmdDLDataRevoke:    {2, 0, "BSSCI §5.13 - Cancel queued downlink"},
		mioty.CmdULDataTransmit:  {5, 2, "BSSCI §5.11 - Command uplink transmission"},
		mioty.CmdDLRxStatusQuery: {1, 4, "BSSCI §5.16 - Query EP downlink metrics"},

		// VM Operations
		mioty.CmdVMActivate:   {2, 0, "BSSCI §5.17 - Activate VM mode"},
		mioty.CmdVMDeactivate: {2, 0, "BSSCI §5.18 - Deactivate VM mode"},
		mioty.CmdVMStatus:     {1, 0, "BSSCI §5.19 - Query VM status"},
		mioty.CmdVMDLData:     {3, 1, "BSSCI §5.20 - VM downlink data"},
	}

	for cmd, reqs := range commandsRequiringFields {
		spec, exists := messageRegistry[cmd]
		if !exists {
			t.Errorf("%s: Missing MessageSpec entry (%s)", cmd, reqs.description)
			continue
		}

		mandatoryCount := len(spec.MandatoryFields)
		optionalCount := len(spec.OptionalFields)

		// Check mandatory fields
		if mandatoryCount < reqs.minMandatory {
			t.Errorf("%s: Expected at least %d mandatory fields, got %d (%s)",
				cmd, reqs.minMandatory, mandatoryCount, reqs.description)
		}

		// Check optional fields
		if optionalCount < reqs.minOptional {
			t.Errorf("%s: Expected at least %d optional fields, got %d (%s)",
				cmd, reqs.minOptional, optionalCount, reqs.description)
		}

		// Verify FieldSpec structure for each field
		for _, field := range spec.MandatoryFields {
			if field.Name == "" {
				t.Errorf("%s: Mandatory field missing Name", cmd)
			}
			if field.Type == "" {
				t.Errorf("%s: Field %s missing Type", cmd, field.Name)
			}
			if !field.Required {
				t.Errorf("%s: Mandatory field %s has Required=false", cmd, field.Name)
			}
			if field.SpecRef == "" {
				t.Errorf("%s: Field %s missing SpecRef", cmd, field.Name)
			}
		}

		for _, field := range spec.OptionalFields {
			if field.Name == "" {
				t.Errorf("%s: Optional field missing Name", cmd)
			}
			if field.Type == "" {
				t.Errorf("%s: Field %s missing Type", cmd, field.Name)
			}
			if field.Required {
				t.Errorf("%s: Optional field %s has Required=true", cmd, field.Name)
			}
			if field.SpecRef == "" {
				t.Errorf("%s: Field %s missing SpecRef", cmd, field.Name)
			}
		}

		t.Logf("PASS: %s: %d mandatory + %d optional fields (%s)",
			cmd, mandatoryCount, optionalCount, reqs.description)
	}
}

// TestMessageSpecResponseCommands verifies that response/complete commands
// have empty or minimal field specifications (acknowledgment-only)
func TestMessageSpecResponseCommands(t *testing.T) {
	// Response and complete commands should have empty field specs
	// (they're acknowledgments only, no payload data per BSSCI spec)
	acknowledgmentCommands := []string{
		// Response commands (SC→BS acknowledgments)
		mioty.CmdConnectResponse, mioty.CmdAttachResponse, mioty.CmdDetachResponse,
		mioty.CmdULDataResponse, mioty.CmdAttachPropagateResponse, mioty.CmdDetachPropagateResponse,
		mioty.CmdDLDataQueueResponse, mioty.CmdDLDataRevokeResponse, mioty.CmdDLDataResultResponse,
		mioty.CmdULDataTransmitResponse, mioty.CmdDLRxStatusResponse, mioty.CmdDLRxStatusQueryResponse,
		mioty.CmdVMActivateResponse, mioty.CmdVMDeactivateResponse, mioty.CmdVMStatusResponse,
		mioty.CmdVMDLDataResponse,

		// Complete commands (handshake finalization)
		mioty.CmdConnectComplete, mioty.CmdPingComplete, mioty.CmdStatusComplete,
		mioty.CmdAttachComplete, mioty.CmdDetachComplete,
		mioty.CmdAttachPropagateComplete, mioty.CmdDetachPropagateComplete,
		mioty.CmdULDataComplete, mioty.CmdULDataTransmitComplete,
		mioty.CmdDLDataQueueComplete, mioty.CmdDLDataRevokeComplete, mioty.CmdDLDataResultComplete,
		mioty.CmdDLRxStatusComplete, mioty.CmdDLRxStatusQueryComplete,
		mioty.CmdVMActivateComplete, mioty.CmdVMDeactivateComplete, mioty.CmdVMStatusComplete,
		mioty.CmdVMDLDataComplete,
	}

	for _, cmd := range acknowledgmentCommands {
		spec, exists := messageRegistry[cmd]
		if !exists {
			t.Errorf("%s: Missing MessageSpec entry (acknowledgment command)", cmd)
			continue
		}

		// Most acknowledgment commands have zero fields
		// A few like CmdPingResponse may have empty slices (nil vs []FieldSpec{})
		// Both are valid for acknowledgments
		if len(spec.MandatoryFields) > 0 {
			t.Errorf("%s: Acknowledgment command should have no mandatory fields, got %d",
				cmd, len(spec.MandatoryFields))
		}

		if len(spec.OptionalFields) > 0 {
			t.Errorf("%s: Acknowledgment command should have no optional fields, got %d",
				cmd, len(spec.OptionalFields))
		}
	}

	t.Logf("PASS: All %d acknowledgment commands have empty field specs", len(acknowledgmentCommands))
}

// TestCommandDirectionConsistency verifies that commands with MessageSpec entries
// also have direction metadata in the generated CommandDirectionMap
func TestCommandDirectionConsistency(t *testing.T) {
	// Verify that all commands in the registry have consistent data
	for cmd, spec := range messageRegistry {
		if spec == nil {
			t.Errorf("messageRegistry[%s] is nil", cmd)
			continue
		}
		if spec.Command != cmd {
			t.Errorf("messageRegistry[%s] has Command=%s", cmd, spec.Command)
		}
	}

	// Verify RegisteredCommands helper
	registeredCmds := RegisteredCommands()
	if len(registeredCmds) != len(messageRegistry) {
		t.Errorf("RegisteredCommands() returned %d commands, registry has %d",
			len(registeredCmds), len(messageRegistry))
	}

	t.Logf("PASS: MessageSpec registry consistency validated for %d commands", len(messageRegistry))
}

// TestMessageSpecDirectionPopulated verifies that all 56 BSSCI v1.0.0 commands
// have their Direction field populated from CommandDirectionMap (Issue #3-4 Fix B1).
// This ensures normalization gating can correctly identify BS→SC, SC→BS, and bidirectional commands.
func TestMessageSpecDirectionPopulated(t *testing.T) {
	// All 56 BSSCI v1.0.0 commands as defined in CommandDirectionMap
	expectedCommands := []string{
		// Connection (§3.1)
		mioty.CmdConnect, mioty.CmdConnectResponse, mioty.CmdConnectComplete,

		// Ping (§3.2)
		mioty.CmdPing, mioty.CmdPingResponse, mioty.CmdPingComplete,

		// Status (§5.8)
		mioty.CmdStatus, mioty.CmdStatusResponse, mioty.CmdStatusComplete,

		// Attach (§5.6)
		mioty.CmdAttach, mioty.CmdAttachResponse, mioty.CmdAttachComplete,

		// Detach (§5.7)
		mioty.CmdDetach, mioty.CmdDetachResponse, mioty.CmdDetachComplete,

		// Attach Propagate (§5.9)
		mioty.CmdAttachPropagate, mioty.CmdAttachPropagateResponse, mioty.CmdAttachPropagateComplete,

		// Detach Propagate (§5.10)
		mioty.CmdDetachPropagate, mioty.CmdDetachPropagateResponse, mioty.CmdDetachPropagateComplete,

		// UL Data (§5.11)
		mioty.CmdULData, mioty.CmdULDataResponse, mioty.CmdULDataComplete,

		// UL Data Transmit (§5.11)
		mioty.CmdULDataTransmit, mioty.CmdULDataTransmitResponse, mioty.CmdULDataTransmitComplete,

		// DL Data Queue (§5.12)
		mioty.CmdDLDataQueue, mioty.CmdDLDataQueueResponse, mioty.CmdDLDataQueueComplete,

		// DL Data Revoke (§5.13)
		mioty.CmdDLDataRevoke, mioty.CmdDLDataRevokeResponse, mioty.CmdDLDataRevokeComplete,

		// DL Data Result (§5.14)
		mioty.CmdDLDataResult, mioty.CmdDLDataResultResponse, mioty.CmdDLDataResultComplete,

		// DL RX Status (§5.15)
		mioty.CmdDLRxStatus, mioty.CmdDLRxStatusResponse, mioty.CmdDLRxStatusComplete,

		// DL RX Status Query (§5.16)
		mioty.CmdDLRxStatusQuery, mioty.CmdDLRxStatusQueryResponse, mioty.CmdDLRxStatusQueryComplete,

		// Error Handling (§4)
		mioty.CmdError, mioty.CmdErrorAck,

		// VM Operations (§5.17-5.20)
		mioty.CmdVMActivate, mioty.CmdVMActivateResponse, mioty.CmdVMActivateComplete,
		mioty.CmdVMDeactivate, mioty.CmdVMDeactivateResponse, mioty.CmdVMDeactivateComplete,
		mioty.CmdVMStatus, mioty.CmdVMStatusResponse, mioty.CmdVMStatusComplete,
		mioty.CmdVMDLData, mioty.CmdVMDLDataResponse, mioty.CmdVMDLDataComplete,
	}

	require.Equal(t, 56, len(expectedCommands), "BSSCI v1.0.0 defines 56 commands")

	// Verify each command has Direction populated in messageRegistry
	for _, cmd := range expectedCommands {
		t.Run(cmd, func(t *testing.T) {
			spec, exists := messageRegistry[cmd]
			require.True(t, exists, "Command %s must be in messageRegistry", cmd)

			// Verify Direction is populated (not zero value "")
			assert.NotEqual(t, CommandDirection(""), spec.Direction,
				"Command %s must have Direction field populated from CommandDirectionMap", cmd)

			// Verify Direction is one of the three valid values
			assert.Contains(t, []CommandDirection{DirectionBStoSC, DirectionSCtoBS, DirectionBidirectional},
				spec.Direction,
				"Command %s Direction must be BS→SC, SC→BS, or Bidirectional", cmd)

			// Verify Direction matches CommandDirectionMap
			expectedDir, exists := CommandDirectionMap[cmd]
			require.True(t, exists, "Command %s must be in CommandDirectionMap", cmd)
			assert.Equal(t, expectedDir, spec.Direction,
				"Command %s Direction in MessageSpec must match CommandDirectionMap", cmd)
		})
	}
}

// TestMessageRegistryCompleteness ensures messageRegistry contains all 56 BSSCI commands.
// This is a sanity check to catch missing entries during spec updates.
func TestMessageRegistryCompleteness(t *testing.T) {
	// SCACI-only commands that are in CommandDirectionMap but not in messageRegistry
	// These are SCACI §3.12 commands that have BSSCI direction mappings but no BSSCI message spec
	scaciOnlyCommands := map[string]bool{
		mioty.CmdDLDataResultResponseSCACI: true, // SCACI §3.12.2: txDataResRsp
		mioty.CmdDLDataResultCompleteSCACI: true, // SCACI §3.12.3: txDataResCmp
	}

	// Get all commands from CommandDirectionMap (the authoritative source)
	commandsInMap := make(map[string]bool)
	for cmd := range CommandDirectionMap {
		commandsInMap[cmd] = true
	}

	// Verify messageRegistry contains all BSSCI commands from CommandDirectionMap
	// (skip SCACI-only commands which don't have BSSCI message specs)
	for cmd := range commandsInMap {
		if scaciOnlyCommands[cmd] {
			continue // SCACI commands don't need BSSCI message specs
		}
		t.Run(cmd, func(t *testing.T) {
			_, exists := messageRegistry[cmd]
			assert.True(t, exists, "Command %s in CommandDirectionMap must have MessageSpec in messageRegistry", cmd)
		})
	}

	// Verify count matches expected 58 commands (56 BSSCI + 2 SCACI §3.12)
	assert.Equal(t, 58, len(commandsInMap), "CommandDirectionMap should contain all 58 protocol commands (56 BSSCI + 2 SCACI)")
	assert.Equal(t, len(commandsInMap)-len(scaciOnlyCommands), len(messageRegistry),
		"messageRegistry should have 56 BSSCI commands (58 total minus 2 SCACI-only)")
}
