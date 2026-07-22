package bssci

import (
	"errors"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// buildMinimalValidPayload creates payloads matching actual sendMessage() output
// This helper ensures tests use the exact JSON shapes from real message constructors
func buildMinimalValidPayload(command string) map[string]interface{} {
	fields, _ := mioty.AllowedOutboundFields(command)
	payload := map[string]interface{}{
		"command": command,
		"opId":    int64(-1),
	}

	for _, field := range fields {
		switch field {
		// Array fields requiring []interface{} conversion (CRITICAL: match exact types from constructors)
		case "nwkSnKey":
			// Per server.go:1955-1961 (AttachResponse) and server.go:3755 (AttachPropagate)
			// AttachResponse uses int(byte), AttachPropagate uses uint8(byte)
			// Use uint8 for minimal test (works for both validation paths)
			arr := make([]interface{}, 16)
			for i := 0; i < 16; i++ {
				arr[i] = uint8(0)
			}
			payload[field] = arr

		case "sign":
			// Per server.go:2187-2194 (DetachResponse) - 4 bytes as ints
			arr := make([]interface{}, 4)
			for i := 0; i < 4; i++ {
				arr[i] = int(0)
			}
			payload[field] = arr

		case "userData":
			// Per ul_transmit.go:136-157 - converted to []interface{} with uint8
			// Use minimal array to match real constructor pattern
			payload[field] = []interface{}{uint8(0)}

		case "packetCnt":
			// Can be single uint32 or []interface{} array for DLDataQueue
			// Use single value for minimal test
			payload[field] = uint32(0)

		case "snScUuid", "snBsUuid":
			// SessionUUID [16]byte - marshals as []interface{} with ints
			arr := make([]interface{}, 16)
			for i := 0; i < 16; i++ {
				arr[i] = int(0)
			}
			payload[field] = arr

		// Numeric fields (EUI/address types)
		case "epEui", "bsEui", "scEui":
			payload[field] = uint64(1)
		case "shAddr":
			payload[field] = uint16(1)
		case "queId", "startTime", "endTime", "rxTime", "txTime":
			payload[field] = int64(1)

		// Small numeric fields
		case "code", "macType", "prio", "format":
			payload[field] = uint8(0)
		case "rssi", "snr", "dlRxRssi", "dlRxSnr", "eqSnr":
			payload[field] = float64(0)
		case "lastPacketCnt":
			payload[field] = uint32(0)

		// String fields
		case "version", "vendor", "model", "name", "swVersion", "message", "profile":
			payload[field] = "test"

		// Boolean fields
		case "bidi", "snResume", "cntDepend", "dualChan", "repetition", "wideCarrOff", "longBlkDist", "cntIncr", "dlTime", "ackReq", "responseExp", "dlWindReq", "expOnly":
			payload[field] = false

		// Optional/complex fields (nil for minimal test)
		case "info", "geoLocation", "config", "dutyCycle", "uptime", "temp", "cpuLoad", "memLoad":
			payload[field] = nil

		// Numeric fields that vary by context
		case "limit", "offset":
			payload[field] = int64(100)
		case "responsePrio":
			payload[field] = uint8(0)

		default:
			// Unknown optional field - set to nil
			payload[field] = nil
		}
	}

	return payload
}

// TestValidateAllCatalogCommands iterates all 52 OutboundFieldCatalog entries with minimal payloads
func TestValidateAllCatalogCommands(t *testing.T) {
	// Get all commands from catalog
	commands := []string{
		mioty.CmdConnectResponse, mioty.CmdConnectComplete,
		mioty.CmdPingResponse, mioty.CmdPingComplete,
		mioty.CmdStatusComplete,
		mioty.CmdAttachResponse, mioty.CmdAttachComplete,
		mioty.CmdAttachPropagate, mioty.CmdAttachPropagateComplete,
		mioty.CmdDetachResponse, mioty.CmdDetachComplete,
		mioty.CmdDetachPropagate, mioty.CmdDetachPropagateComplete,
		mioty.CmdULDataResponse, mioty.CmdULDataComplete,
		mioty.CmdULDataTransmit, mioty.CmdULDataTransmitComplete,
		mioty.CmdDLDataQueue, mioty.CmdDLDataQueueResponse, mioty.CmdDLDataQueueComplete,
		mioty.CmdDLDataRevoke, mioty.CmdDLDataRevokeResponse, mioty.CmdDLDataRevokeComplete,
		mioty.CmdDLDataResultResponse, mioty.CmdDLDataResultComplete,
		mioty.CmdDLRxStatusQuery, mioty.CmdDLRxStatusComplete,
		mioty.CmdVMActivate, mioty.CmdVMActivateResponse, mioty.CmdVMActivateComplete,
		mioty.CmdVMDeactivate, mioty.CmdVMDeactivateResponse, mioty.CmdVMDeactivateComplete,
		mioty.CmdVMStatus, mioty.CmdVMStatusResponse, mioty.CmdVMStatusComplete,
		mioty.CmdVMDLData, mioty.CmdVMDLDataResponse, mioty.CmdVMDLDataComplete,
		mioty.CmdError, mioty.CmdErrorAck,
	}

	logger := logger.NewNop()
	server := &Server{
		config: &Config{},
		logger: logger,
	}
	server.broadcastFn = server.SendAttachPropagateToAll

	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:          "test-session",
			DbSessionID: 1,
			Encoding:    "json",
		},
	}

	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			payload := buildMinimalValidPayload(command)

			err := server.validateOutboundMessage(session, payload)
			if err != nil {
				t.Errorf("Validation failed for %s: %v", command, err)
			}
		})
	}
}

// TestValidationMissingCommand detects missing "command" field
func TestValidationMissingCommand(t *testing.T) {
	logger := logger.NewNop()
	server := &Server{
		config: &Config{},
		logger: logger,
	}
	server.broadcastFn = server.SendAttachPropagateToAll

	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:          "test-session",
			DbSessionID: 1,
			Encoding:    "json",
		},
	}

	// Message without command field
	invalidMsg := map[string]interface{}{
		"opId": int64(-1),
	}

	err := server.validateOutboundMessage(session, invalidMsg)
	if err == nil {
		t.Error("Expected error for missing command field")
	}

	// Verify it's the correct error token
	var catalogErr *CatalogError
	if !errors.As(err, &catalogErr) || catalogErr.Token != errOutboundMissingCommand {
		t.Errorf("Expected errOutboundMissingCommand, got: %v", err)
	}
}

// TestValidationMissingOpId verifies that opId is mandatory per MIOTY §2.5.1
// validateOutboundMessage enforces opId presence and type validation
func TestValidationMissingOpId(t *testing.T) {
	logger := logger.NewNop()
	server := &Server{
		config: &Config{},
		logger: logger,
	}
	server.broadcastFn = server.SendAttachPropagateToAll

	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:          "test-session",
			DbSessionID: 1,
			Encoding:    "json",
		},
	}

	// Message without opId field - should fail validation (MIOTY requires opId on all frames)
	msgWithoutOpId := map[string]interface{}{
		"command": mioty.CmdPingResponse,
	}

	err := server.validateOutboundMessage(session, msgWithoutOpId)
	if err == nil {
		t.Errorf("Validation should fail when opId is missing")
	} else {
		// Verify correct error token
		if catalogErr, ok := err.(*CatalogError); ok {
			if catalogErr.Token != errOutboundMissingField {
				t.Errorf("Expected errOutboundMissingField, got: %s", catalogErr.Token)
			}
		}
	}

	// Message with opId field - should pass
	msgWithOpId := map[string]interface{}{
		"command": mioty.CmdPingResponse,
		"opId":    int64(-1),
	}

	err = server.validateOutboundMessage(session, msgWithOpId)
	if err != nil {
		t.Errorf("Validation should pass when opId is present: %v", err)
	}

	// Message with opId=0 should pass (valid for conCmp per BSSCI §3.3)
	msgWithOpIDZero := map[string]interface{}{
		"command": mioty.CmdPingResponse,
		"opId":    int64(0),
	}

	err = server.validateOutboundMessage(session, msgWithOpIDZero)
	if err != nil {
		t.Errorf("Validation should pass when opId is 0: %v", err)
	}

	// Message with negative opId should pass (SC-initiated operations)
	msgWithNegativeOpID := map[string]interface{}{
		"command": mioty.CmdPingResponse,
		"opId":    int64(-42),
	}

	err = server.validateOutboundMessage(session, msgWithNegativeOpID)
	if err != nil {
		t.Errorf("Validation should pass when opId is negative: %v", err)
	}

	// Message with invalid opId type should fail
	msgWithInvalidOpIDType := map[string]interface{}{
		"command": mioty.CmdPingResponse,
		"opId":    "not-a-number",
	}

	err = server.validateOutboundMessage(session, msgWithInvalidOpIDType)
	if err == nil {
		t.Errorf("Validation should fail when opId has invalid type")
	} else {
		// Verify correct error token
		if catalogErr, ok := err.(*CatalogError); ok {
			if catalogErr.Token != errOutboundInvalidFieldType {
				t.Errorf("Expected errOutboundInvalidFieldType, got: %s", catalogErr.Token)
			}
		}
	}
}

// TestValidationExtraField injects extra field that should fail validation
func TestValidationExtraField(t *testing.T) {
	logger := logger.NewNop()
	server := &Server{
		config: &Config{},
		logger: logger,
	}
	server.broadcastFn = server.SendAttachPropagateToAll

	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:          "test-session",
			DbSessionID: 1,
			Encoding:    "json",
		},
	}

	// Valid payload with extra field
	invalidMsg := map[string]interface{}{
		"command":    mioty.CmdPingResponse,
		"opId":       int64(-1),
		"bogusField": "this_should_not_be_here",
	}

	err := server.validateOutboundMessage(session, invalidMsg)
	if err == nil {
		t.Error("Expected error for extra field")
	}

	// Verify it's the correct error token
	var catalogErr *CatalogError
	if !errors.As(err, &catalogErr) || catalogErr.Token != errOutboundExtraField {
		t.Errorf("Expected errOutboundExtraField, got: %v", err)
	}
}

// TestValidationUnknownCommand uses command not in catalog
func TestValidationUnknownCommand(t *testing.T) {
	logger := logger.NewNop()
	server := &Server{
		config: &Config{},
		logger: logger,
	}
	server.broadcastFn = server.SendAttachPropagateToAll

	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:          "test-session",
			DbSessionID: 1,
			Encoding:    "json",
		},
	}

	// Message with unknown command
	invalidMsg := map[string]interface{}{
		"command": "fakeCommand",
		"opId":    int64(-1),
	}

	err := server.validateOutboundMessage(session, invalidMsg)
	if err == nil {
		t.Error("Expected error for unknown command")
	}

	// Verify it's the correct error token
	var catalogErr *CatalogError
	if !errors.As(err, &catalogErr) || catalogErr.Token != errOutboundUnknownCommand {
		t.Errorf("Expected errOutboundUnknownCommand, got: %v", err)
	}
}

// TestValidationMarshalFailure tests unmarshalable message
func TestValidationMarshalFailure(t *testing.T) {
	logger := logger.NewNop()
	server := &Server{
		config: &Config{},
		logger: logger,
	}

	server.broadcastFn = server.SendAttachPropagateToAll
	_ = server

	// Create a struct with unmarshalable field (channel type)
	type UnmarshalableMsg struct {
		Command string
		OpId    int64
		Chan    chan int // Cannot be marshaled to JSON
	}

	invalidMsg := UnmarshalableMsg{
		Command: mioty.CmdPingResponse,
		OpId:    int64(-1),
		Chan:    make(chan int),
	}

	_, err := outboundValidationProjection(invalidMsg)
	if err == nil {
		t.Error("Expected error for unmarshalable message")
	}

	// Verify it's the correct error token
	var catalogErr *CatalogError
	if !errors.As(err, &catalogErr) || catalogErr.Token != errOutboundMarshalFailed {
		t.Errorf("Expected errOutboundMarshalFailed, got: %v", err)
	}
}

// TestValidationStructMessages tests ConnectResponse and VMDLData struct messages
func TestValidationStructMessages(t *testing.T) {
	logger := logger.NewNop()
	server := &Server{
		config: &Config{},
		logger: logger,
	}

	server.broadcastFn = server.SendAttachPropagateToAll
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:          "test-session",
			DbSessionID: 1,
			Encoding:    "json",
		},
	}

	t.Run("ConnectResponse_struct", func(t *testing.T) {
		// Create ConnectResponse struct (from types.go)
		var uuid mioty.SessionUUID
		for i := 0; i < 16; i++ {
			uuid[i] = byte(i)
		}

		vendor := "test-vendor"
		connectRsp := mioty.ConnectResponse{
			BaseMessage: mioty.BaseMessage{
				CommandType: mioty.CmdConnectResponse,
				OpId:        int64(-1),
			},
			Version:  "1.0.0",
			ScEui:    TestScEui01,
			Vendor:   &vendor,
			SnResume: false,
			SnScUuid: uuid,
		}

		projection, projErr := outboundValidationProjection(connectRsp)
		if projErr != nil {
			t.Fatalf("Projection failed for ConnectResponse struct: %v", projErr)
		}
		err := server.validateOutboundMessage(session, projection)
		if err != nil {
			t.Errorf("Validation failed for ConnectResponse struct: %v", err)
		}
	})

	t.Run("VMDLData_struct", func(t *testing.T) {
		// Create VMDLData struct (from types.go)
		txTime := int64(1234567890)
		vmDlData := mioty.VMDLData{
			BaseMessage: mioty.BaseMessage{
				CommandType: mioty.CmdVMDLData,
				OpId:        int64(-1),
			},
			EpEui:    TestBsEui01,
			MACType:  uint8(1),
			UserData: []byte{0x00, 0x01, 0x02},
			TxTime:   &txTime,
		}

		projection, projErr := outboundValidationProjection(vmDlData)
		if projErr != nil {
			t.Fatalf("Projection failed for VMDLData struct: %v", projErr)
		}
		err := server.validateOutboundMessage(session, projection)
		if err != nil {
			t.Errorf("Validation failed for VMDLData struct: %v", err)
		}
	})
}
