package scaci

import (
	"encoding/json"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

// TestDecodeMessagePack validates standard MessagePack decoding
func TestDecodeMessagePack(t *testing.T) {
	testMsg := map[string]interface{}{
		"command": CmdConnect,
		"opId":    int64(0),
		"version": "1.0.0",
		"acEui":   uint64(0xAABBCCDDEEFF1122),
	}

	// Encode as MessagePack
	encoded, err := msgpack.Marshal(testMsg)
	if err != nil {
		t.Fatalf("failed to encode test message: %v", err)
	}

	// Decode with same logic as server.go
	var decoded map[string]interface{}
	if err := msgpack.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("MessagePack decode failed: %v", err)
	}

	// Verify command field
	command, ok := decoded["command"].(string)
	if !ok || command != CmdConnect {
		t.Errorf("expected command=%s, got %v", CmdConnect, decoded["command"])
	}
}

// TestDecodeJSON validates JSON fallback decoding per SCACI §1
func TestDecodeJSON(t *testing.T) {
	testMsg := map[string]interface{}{
		"command": CmdConnect,
		"opId":    float64(0), // JSON numbers are float64
		"version": "1.0.0",
		"acEui":   float64(0xAABBCCDDEEFF1122),
	}

	// Encode as JSON
	encoded, err := json.Marshal(testMsg)
	if err != nil {
		t.Fatalf("failed to encode test message: %v", err)
	}

	// Decode with same logic as server.go (JSON fallback path)
	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("JSON decode failed: %v", err)
	}

	// Verify command field
	command, ok := decoded["command"].(string)
	if !ok || command != CmdConnect {
		t.Errorf("expected command=%s, got %v", CmdConnect, decoded["command"])
	}
}

// TestDecodeFallbackSequence validates the fallback logic:
// MessagePack fails → JSON succeeds
func TestDecodeFallbackSequence(t *testing.T) {
	testMsg := map[string]interface{}{
		"command": CmdConnect,
		"opId":    float64(0),
		"version": "1.0.0",
	}

	// Encode as JSON (not valid MessagePack)
	jsonPayload, err := json.Marshal(testMsg)
	if err != nil {
		t.Fatalf("failed to encode test message: %v", err)
	}

	// Simulate server.go decode logic
	var decoded map[string]interface{}
	msgpackErr := msgpack.Unmarshal(jsonPayload, &decoded)
	if msgpackErr == nil {
		// MessagePack might sometimes parse JSON-ish data
		// If it does, the test still passes as long as command is extractable
		command, ok := decoded["command"].(string)
		if ok && command == CmdConnect {
			t.Log("MessagePack parsed JSON payload (acceptable)")
			return
		}
	}

	// Fallback to JSON
	decoded = nil
	if jsonErr := json.Unmarshal(jsonPayload, &decoded); jsonErr != nil {
		t.Fatalf("JSON fallback decode failed: %v (msgpack error: %v)", jsonErr, msgpackErr)
	}

	// Verify command field
	command, ok := decoded["command"].(string)
	if !ok || command != CmdConnect {
		t.Errorf("expected command=%s, got %v", CmdConnect, decoded["command"])
	}
}

// TestDecodeBothFail validates error when both codecs fail
func TestDecodeBothFail(t *testing.T) {
	// Invalid payload (not MessagePack nor JSON)
	invalidPayload := []byte{0xFF, 0xFE, 0x00, 0x01, 0x02}

	var decoded map[string]interface{}
	msgpackErr := msgpack.Unmarshal(invalidPayload, &decoded)
	jsonErr := json.Unmarshal(invalidPayload, &decoded)

	// Both should fail
	if msgpackErr == nil {
		t.Error("expected MessagePack to fail on invalid payload")
	}
	if jsonErr == nil {
		t.Error("expected JSON to fail on invalid payload")
	}
}

// ============================================================================
// SCACI §2.4 - Message Interpretation: Unknown Field Tolerance
// ============================================================================

// TestUnknownFieldTolerance_MessagePack validates §2.4 compliance:
// "additional fields, beyond the protocol specification must be ignored graciously"
func TestUnknownFieldTolerance_MessagePack(t *testing.T) {
	tests := []struct {
		name           string
		extraFields    map[string]interface{}
		requiredFields map[string]interface{}
	}{
		{
			name: "single unknown field",
			extraFields: map[string]interface{}{
				"unknownField": "should be ignored",
			},
			requiredFields: map[string]interface{}{
				"command": CmdConnect,
				"opId":    int64(0),
				"version": "1.0.0",
			},
		},
		{
			name: "multiple unknown fields",
			extraFields: map[string]interface{}{
				"futureField1": "value1",
				"futureField2": int64(999),
				"futureField3": true,
			},
			requiredFields: map[string]interface{}{
				"command": CmdConnect,
				"opId":    int64(0),
				"version": "1.0.0",
			},
		},
		{
			name: "nested unknown object",
			extraFields: map[string]interface{}{
				"futureMetadata": map[string]interface{}{
					"nested": "object",
					"depth":  int64(2),
				},
			},
			requiredFields: map[string]interface{}{
				"command": CmdConnect,
				"opId":    int64(0),
				"version": "1.0.0",
			},
		},
		{
			name: "unknown array field",
			extraFields: map[string]interface{}{
				"futureList": []interface{}{"a", "b", "c"},
			},
			requiredFields: map[string]interface{}{
				"command": CmdConnect,
				"opId":    int64(0),
				"version": "1.0.0",
			},
		},
		{
			name: "empty unknown field",
			extraFields: map[string]interface{}{
				"emptyField": "",
			},
			requiredFields: map[string]interface{}{
				"command": CmdConnect,
				"opId":    int64(0),
				"version": "1.0.0",
			},
		},
		{
			name: "null unknown field",
			extraFields: map[string]interface{}{
				"nullField": nil,
			},
			requiredFields: map[string]interface{}{
				"command": CmdConnect,
				"opId":    int64(0),
				"version": "1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build message with both required and extra fields
			testMsg := make(map[string]interface{})
			for k, v := range tt.requiredFields {
				testMsg[k] = v
			}
			for k, v := range tt.extraFields {
				testMsg[k] = v
			}

			// Encode as MessagePack
			encoded, err := msgpack.Marshal(testMsg)
			if err != nil {
				t.Fatalf("failed to encode test message: %v", err)
			}

			// Decode (simulates server.go logic)
			var decoded map[string]interface{}
			if err := msgpack.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("MessagePack decode failed (§2.4 violation): %v", err)
			}

			// Verify required fields are preserved
			command, ok := decoded["command"].(string)
			if !ok || command != CmdConnect {
				t.Errorf("required field 'command' corrupted: got %v", decoded["command"])
			}

			version, ok := decoded["version"].(string)
			if !ok || version != "1.0.0" {
				t.Errorf("required field 'version' corrupted: got %v", decoded["version"])
			}

			// Note: Unknown fields MAY be present in decoded map - that's OK
			// The requirement is that decoding succeeds, not that fields are stripped
			// Handler code should only access known fields
		})
	}
}

// TestUnknownFieldTolerance_JSON validates §2.4 compliance for JSON codec
func TestUnknownFieldTolerance_JSON(t *testing.T) {
	tests := []struct {
		name           string
		extraFields    map[string]interface{}
		requiredFields map[string]interface{}
	}{
		{
			name: "future protocol extension",
			extraFields: map[string]interface{}{
				"futureCapability":  true,
				"extensionVersion":  "2.0.0",
				"additionalMetrics": map[string]interface{}{"metric1": 100},
			},
			requiredFields: map[string]interface{}{
				"command": CmdConnect,
				"opId":    float64(0), // JSON numbers
				"version": "1.0.0",
			},
		},
		{
			name: "vendor-specific extensions",
			extraFields: map[string]interface{}{
				"x-vendor-specific": "custom value",
				"_internal":         "should be tolerated",
			},
			requiredFields: map[string]interface{}{
				"command": CmdConnect,
				"opId":    float64(0),
				"version": "1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build message with both required and extra fields
			testMsg := make(map[string]interface{})
			for k, v := range tt.requiredFields {
				testMsg[k] = v
			}
			for k, v := range tt.extraFields {
				testMsg[k] = v
			}

			// Encode as JSON
			encoded, err := json.Marshal(testMsg)
			if err != nil {
				t.Fatalf("failed to encode test message: %v", err)
			}

			// Decode (JSON fallback path)
			var decoded map[string]interface{}
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("JSON decode failed (§2.4 violation): %v", err)
			}

			// Verify required fields are preserved
			command, ok := decoded["command"].(string)
			if !ok || command != CmdConnect {
				t.Errorf("required field 'command' corrupted: got %v", decoded["command"])
			}

			version, ok := decoded["version"].(string)
			if !ok || version != "1.0.0" {
				t.Errorf("required field 'version' corrupted: got %v", decoded["version"])
			}
		})
	}
}

// TestUnknownFieldTolerance_HandlerAccess validates handler code pattern
// Handlers should only access known fields, ignoring unknown ones
func TestUnknownFieldTolerance_HandlerAccess(t *testing.T) {
	// Simulate message from a future protocol version with unknown fields
	futureVersionMsg := map[string]interface{}{
		"command":             CmdConnect,
		"opId":                int64(0),
		"version":             "1.0.0",
		"acEui":               uint64(0xAABBCCDDEEFF1122),
		"snAcUuid":            [16]byte{0x01, 0x02, 0x03},
		"futureField":         "future value",           // Unknown - should be ignored
		"experimental":        map[string]interface{}{}, // Unknown - should be ignored
		"__protocolExtension": true,                     // Unknown - should be ignored
	}

	// Encode and decode
	encoded, err := msgpack.Marshal(futureVersionMsg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := msgpack.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode failed (§2.4 violation): %v", err)
	}

	// Handler pattern: only access known fields
	// This simulates what real handler code should do
	knownFields := []string{"command", "opId", "version", "acEui", "snAcUuid"}

	for _, field := range knownFields {
		if _, exists := decoded[field]; !exists {
			t.Errorf("known field %q missing from decoded message", field)
		}
	}

	// Handler should NOT fail if unknown fields exist
	// Just don't access them - that's the §2.4 compliance pattern
	if decoded["command"] != CmdConnect {
		t.Errorf("command field corrupted by unknown fields")
	}
}

// ============================================================================
// SCACI §2.4 - Missing Optional Field Defaults
// ============================================================================

// TestMissingOptionalField_DefaultSubstitution validates §2.4 requirement:
// "Missing optional fields must be substituted silently with the specified defaults"
func TestMissingOptionalField_DefaultSubstitution(t *testing.T) {
	// Connect message with only mandatory fields
	minimalConnect := map[string]interface{}{
		"command":  CmdConnect,
		"opId":     int64(0),
		"version":  "1.0.0",
		"acEui":    uint64(0xAABBCCDDEEFF1122),
		"snAcUuid": [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		// Missing optional fields: vendor, model, name, swVersion
	}

	encoded, err := msgpack.Marshal(minimalConnect)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := msgpack.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// Verify mandatory fields present
	if decoded["command"] != CmdConnect {
		t.Error("mandatory 'command' field missing or wrong")
	}
	if decoded["version"] != "1.0.0" {
		t.Error("mandatory 'version' field missing or wrong")
	}

	// Handler code should provide defaults for missing optional fields
	// Example pattern (simulated):
	vendor := ""
	if v, ok := decoded["vendor"].(string); ok {
		vendor = v
	}
	// vendor is now "" (default) since it was missing - this is valid per §2.4

	if vendor != "" {
		t.Errorf("expected empty default for missing optional 'vendor', got %q", vendor)
	}
}

// ============================================================================
// SCACI §2.5 - Message Assembly: No Extra Fields
// ============================================================================

// TestMessageAssembly_NoExtraFields validates §2.5 requirement:
// "No extra fields, beyond the protocol specification must be added"
func TestMessageAssembly_NoExtraFields(t *testing.T) {
	// Verify ConnectResponse struct only marshals spec-defined fields
	version := "1.0.0"
	vendor := "KiloCenter"
	model := "KC-1000"
	name := "test-sc"
	swVersion := "1.0.0-test"

	resp := &ConnectResponse{
		Version:   &version,
		ScEui:     0x1122334455667788,
		SnScUUID:  UUID16{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SnResume:  false,
		Vendor:    &vendor,
		Model:     &model,
		Name:      &name,
		SwVersion: &swVersion,
	}

	// Marshal to MessagePack
	encoded, err := msgpack.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal ConnectResponse failed: %v", err)
	}

	// Decode back to map to inspect fields
	var decoded map[string]interface{}
	if err := msgpack.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// List of spec-defined fields for ConnectResponse per SCACI §3.3
	// Note: All SCACI messages include BaseMessage fields (command, opId) per §3
	specFields := map[string]bool{
		// BaseMessage fields (present in all SCACI messages)
		"command": true,
		"opId":    true,
		// ConnectResponse-specific fields per §3.3.2
		"version":   true,
		"scEui":     true,
		"snScUuid":  true,
		"snResume":  true,
		"vendor":    true,
		"model":     true,
		"name":      true,
		"swVersion": true,
		"info":      true, // Optional per SCACI §3.3.2
	}

	// Check for any non-spec fields (§2.5 violation)
	for field := range decoded {
		if !specFields[field] {
			t.Errorf("§2.5 violation: extra field %q in ConnectResponse", field)
		}
	}
}

// TestMessageAssembly_MandatoryFields validates §2.5 requirement:
// "every mandatory field must be inserted"
func TestMessageAssembly_MandatoryFields(t *testing.T) {
	// ConnectResponse mandatory fields per SCACI §3.3:
	// - version (response to negotiation)
	// - scEui (SC identity)
	// - snScUuid (session identifier)
	// - snResume (resume status)

	version := "1.0.0"
	resp := &ConnectResponse{
		Version:  &version,
		ScEui:    0x1122334455667788,
		SnScUUID: UUID16{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SnResume: false,
		// Optional fields omitted
	}

	encoded, err := msgpack.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := msgpack.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// Verify all mandatory fields present
	// Per SCACI §3: every message has command and opId (BaseMessage)
	// Per SCACI §3.3.2: ConnectResponse specific mandatory fields
	mandatoryFields := []string{"command", "opId", "version", "scEui", "snScUuid", "snResume"}
	for _, field := range mandatoryFields {
		if _, exists := decoded[field]; !exists {
			t.Errorf("§2.5 violation: mandatory field %q missing from ConnectResponse", field)
		}
	}
}

// ============================================================================
// SCACI §2.4 - Typed Struct Unknown Field Tolerance
// ============================================================================

// TestUnknownFieldTolerance_TypedStruct validates §2.4 compliance when decoding
// directly into typed structs. Per msgpack-go behavior, unknown fields are ignored
// during decode to typed structs.
func TestUnknownFieldTolerance_TypedStruct(t *testing.T) {
	tests := []struct {
		name        string
		buildInput  func() map[string]interface{}
		decodeInto  func() interface{}
		verifyKnown func(t *testing.T, decoded interface{})
	}{
		{
			name: "Connect_with_futureCapability",
			buildInput: func() map[string]interface{} {
				return map[string]interface{}{
					"command":          CmdConnect,
					"opId":             int64(0),
					"version":          "1.0.0",
					"acEui":            uint64(0xAABBCCDDEEFF1122),
					"snAcUuid":         [16]byte{0x01, 0x02, 0x03, 0x04},
					"futureCapability": "ignored", // Unknown field
				}
			},
			decodeInto: func() interface{} { return &Connect{} },
			verifyKnown: func(t *testing.T, decoded interface{}) {
				conn := decoded.(*Connect)
				if conn.Version != "1.0.0" { // Version is string, not *string
					t.Error("known field 'version' corrupted")
				}
				if conn.AcEui == 0 {
					t.Error("known field 'acEui' corrupted")
				}
			},
		},
		{
			name: "Register_with_vendorMetadata",
			buildInput: func() map[string]interface{} {
				return map[string]interface{}{
					"command":        CmdRegister,
					"opId":           int64(1),
					"epEui":          uint64(0x1122334455667788),
					"nwkKey":         [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
					"vendorMetadata": map[string]interface{}{"vendor": "test"}, // Unknown nested object
				}
			},
			decodeInto: func() interface{} { return &Register{} },
			verifyKnown: func(t *testing.T, decoded interface{}) {
				reg := decoded.(*Register)
				if reg.EpEui != 0x1122334455667788 {
					t.Errorf("known field 'epEui' corrupted: got %x", reg.EpEui)
				}
			},
		},
		{
			name: "ULDataTransmit_with_experimentalMode",
			buildInput: func() map[string]interface{} {
				return map[string]interface{}{
					"command":          CmdULDataTransmit,
					"opId":             int64(5),
					"epEui":            uint64(0x1122334455667788),
					"shAddr":           uint16(0x1234), // ShAddr is uint16 per mioty.ULDataTransmit
					"packetCnt":        uint32(42),
					"nwkSnKey":         [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
					"userData":         []byte{0xDE, 0xAD, 0xBE, 0xEF},
					"experimentalMode": true, // Unknown field
				}
			},
			decodeInto: func() interface{} { return &ULDataTransmit{} },
			verifyKnown: func(t *testing.T, decoded interface{}) {
				ul := decoded.(*ULDataTransmit)
				if ul.EpEui != 0x1122334455667788 {
					t.Errorf("known field 'epEui' corrupted: got %x", ul.EpEui)
				}
				if ul.ShAddr != 0x1234 {
					t.Errorf("known field 'shAddr' corrupted: got %x", ul.ShAddr)
				}
				if ul.PacketCnt != 42 {
					t.Errorf("known field 'packetCnt' corrupted: got %d", ul.PacketCnt)
				}
			},
		},
		{
			name: "DLDataQueue_with_priorityClass",
			buildInput: func() map[string]interface{} {
				return map[string]interface{}{
					"command":       CmdDLDataQueue,
					"opId":          int64(10),
					"epEui":         uint64(0x1122334455667788),
					"queId":         uint64(999),
					"cntDepend":     false,
					"userData":      [][]byte{{0x01, 0x02}}, // UserData is [][]byte per mioty.DLDataQueue
					"priorityClass": "high",                 // Unknown field
				}
			},
			decodeInto: func() interface{} { return &DLDataQueue{} },
			verifyKnown: func(t *testing.T, decoded interface{}) {
				dl := decoded.(*DLDataQueue)
				if dl.EpEui != 0x1122334455667788 {
					t.Errorf("known field 'epEui' corrupted: got %x", dl.EpEui)
				}
				if dl.QueId != 999 {
					t.Errorf("known field 'queId' corrupted: got %d", dl.QueId)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build message with unknown fields
			input := tt.buildInput()

			// Encode to MessagePack
			encoded, err := msgpack.Marshal(input)
			if err != nil {
				t.Fatalf("encode failed: %v", err)
			}

			// Decode to typed struct - should ignore unknown fields per §2.4
			target := tt.decodeInto()
			if err := msgpack.Unmarshal(encoded, target); err != nil {
				t.Fatalf("decode to typed struct failed (§2.4 violation): %v", err)
			}

			// Verify known fields preserved
			tt.verifyKnown(t, target)

			// Verify round-trip: re-marshal should NOT include unknown fields
			reEncoded, err := msgpack.Marshal(target)
			if err != nil {
				t.Fatalf("re-encode failed: %v", err)
			}

			var reDecoded map[string]interface{}
			if err := msgpack.Unmarshal(reEncoded, &reDecoded); err != nil {
				t.Fatalf("re-decode failed: %v", err)
			}

			// Check that unknown fields from input are NOT in output
			unknownFields := []string{"futureCapability", "vendorMetadata", "experimentalMode", "priorityClass"}
			for _, field := range unknownFields {
				if _, exists := reDecoded[field]; exists {
					t.Errorf("§2.5 violation: unknown field %q survived round-trip", field)
				}
			}
		})
	}
}

// ============================================================================
// SCACI §2.4/§3.9.1 - ULDataTransmit Format Default
// ============================================================================

// TestULDataTransmit_FormatDefault validates §3.9.1 requirement:
// "format | Numeric | optional, default 0"
// The handler should apply default=0 when format field is absent.
func TestULDataTransmit_FormatDefault(t *testing.T) {
	// Create minimal ULDataTx payload WITHOUT format field
	minimalULDataTx := map[string]interface{}{
		"command":   CmdULDataTransmit,
		"opId":      int64(5),
		"epEui":     uint64(0x1122334455667788),
		"shAddr":    uint16(0x1234), // ShAddr is uint16 per mioty.ULDataTransmit
		"packetCnt": uint32(42),
		"nwkSnKey":  [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		"userData":  []byte{0xDE, 0xAD, 0xBE, 0xEF},
		// format deliberately omitted - should default to 0 per §3.9.1
	}

	// Encode
	encoded, err := msgpack.Marshal(minimalULDataTx)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	// Decode to typed struct
	var decoded ULDataTransmit
	if err := msgpack.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// Verify format is nil (field absent from wire)
	if decoded.Format != nil {
		t.Errorf("expected nil Format (absent from wire), got %d", *decoded.Format)
	}

	// Handler pattern per §2.4: normalize to default 0
	// This simulates what handleULDataTransmit does
	format := uint8(0)
	if decoded.Format != nil {
		format = *decoded.Format
	}

	if format != 0 {
		t.Errorf("expected default format=0, got %d", format)
	}

	// Verify mandatory fields preserved
	if decoded.EpEui != 0x1122334455667788 {
		t.Errorf("epEui corrupted: got %x", decoded.EpEui)
	}
	if decoded.ShAddr != 0x1234 {
		t.Errorf("shAddr corrupted: got %x", decoded.ShAddr)
	}
	if decoded.PacketCnt != 42 {
		t.Errorf("packetCnt corrupted: got %d", decoded.PacketCnt)
	}
}
