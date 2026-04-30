package scaci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// ============================================================================
// Assembly Tests: Verify wire format contains only spec-defined keys
// Per SCACI §§2.4-2.5: Messages use MessagePack with spec-defined field names
// ============================================================================

// TestConnectResponseAssembly verifies ConnectResponse wire keys per SCACI §3.3.2
func TestConnectResponseAssembly(t *testing.T) {
	// Expected keys per §3.3.2:
	// Required: command, opId, version, scEui, snScUuid
	// Optional: vendor, model, name, swVersion, info, snResume
	expectedKeys := map[string]bool{
		"command":   true,
		"opId":      true,
		"version":   true,
		"scEui":     true,
		"snScUuid":  true,
		"vendor":    true,
		"model":     true,
		"name":      true,
		"swVersion": true,
		"info":      true,
		"snResume":  true,
	}

	version := ProtocolVersionString
	vendor := "KiloCenter"
	msg := ConnectResponse{
		BaseMessage: BaseMessage{Command: CmdConnectResponse, OpId: 0},
		Version:     &version,
		ScEui:       0x0102030405060708,
		SnScUUID:    UUID16{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00},
		Vendor:      &vendor,
		SnResume:    true,
	}

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "ConnectResponse should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "ConnectResponse should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in ConnectResponse wire format (spec: §3.3.2)", key)
	}

	// Verify mandatory keys present
	assert.Contains(t, decoded, "command", "ConnectResponse must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "ConnectResponse must contain 'opId' (spec: §3.2)")
	assert.Contains(t, decoded, "version", "ConnectResponse must contain 'version' (spec: §3.3.2)")
	assert.Contains(t, decoded, "scEui", "ConnectResponse must contain 'scEui' (spec: §3.3.2)")
	assert.Contains(t, decoded, "snScUuid", "ConnectResponse must contain 'snScUuid' (spec: §3.3.2)")
}

// TestStatusResponseAssembly verifies StatusResponse wire keys per SCACI §3.5.2
func TestStatusResponseAssembly(t *testing.T) {
	// Expected keys per §3.5.2:
	// Required: command, opId, code, message, time
	// Optional: uptime, baseStations
	expectedKeys := map[string]bool{
		"command":      true,
		"opId":         true,
		"code":         true,
		"message":      true,
		"time":         true,
		"uptime":       true,
		"baseStations": true,
	}

	uptime := int64(3600)
	msg := StatusResponse{
		BaseMessage:  BaseMessage{Command: CmdStatusResponse, OpId: 1},
		Code:         POSIX_OK,
		Message:      "Service Center operational",
		Time:         1234567890,
		Uptime:       &uptime,
		BaseStations: []BaseStationStatus{},
	}

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "StatusResponse should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "StatusResponse should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in StatusResponse wire format (spec: §3.5.2)", key)
	}

	// Verify mandatory keys present
	assert.Contains(t, decoded, "command", "StatusResponse must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "StatusResponse must contain 'opId' (spec: §3.2)")
	assert.Contains(t, decoded, "code", "StatusResponse must contain 'code' (spec: §3.5.2)")
	assert.Contains(t, decoded, "message", "StatusResponse must contain 'message' (spec: §3.5.2)")
	assert.Contains(t, decoded, "time", "StatusResponse must contain 'time' (spec: §3.5.2)")
}

// TestULDataAssembly verifies ULData wire keys per SCACI §3.8.1
func TestULDataAssembly(t *testing.T) {
	// Expected keys per §3.8.1:
	// Required: command, opId, epEui, baseStations, packetCnt, userData, dlOpen, responseExp, dlAck
	// Optional: format, duplicate
	expectedKeys := map[string]bool{
		"command":      true,
		"opId":         true,
		"epEui":        true,
		"baseStations": true,
		"packetCnt":    true,
		"userData":     true,
		"format":       true,
		"dlOpen":       true,
		"responseExp":  true,
		"dlAck":        true,
		"duplicate":    true,
	}

	format := uint8(1)
	msg := ULData{
		BaseMessage: BaseMessage{Command: CmdULData, OpId: -1},
		EpEui:       0x0102030405060708,
		BaseStations: []BaseStationReception{
			{BsEui: 0x1122334455667788, RxTime: 1234567890},
		},
		PacketCnt:   42,
		UserData:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
		Format:      &format,
		DlOpen:      true,
		ResponseExp: false,
		DlAck:       true,
	}

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "ULData should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "ULData should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in ULData wire format (spec: §3.8.1)", key)
	}

	// Verify mandatory keys present
	assert.Contains(t, decoded, "command", "ULData must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "ULData must contain 'opId' (spec: §3.2)")
	assert.Contains(t, decoded, "epEui", "ULData must contain 'epEui' (spec: §3.8.1)")
	assert.Contains(t, decoded, "baseStations", "ULData must contain 'baseStations' (spec: §3.8.1)")
	assert.Contains(t, decoded, "userData", "ULData must contain 'userData' (spec: §3.8.1)")
}

// TestErrorAssembly verifies Error wire keys per SCACI §3.14.1
func TestErrorAssembly(t *testing.T) {
	// Expected keys per §3.14.1:
	// Required: command, opId, code, message
	// Optional: errorToken (internal, not in spec but present in our impl)
	expectedKeys := map[string]bool{
		"command":    true,
		"opId":       true,
		"code":       true,
		"message":    true,
		"errorToken": true, // Internal extension for debugging
	}

	msg := Error{
		BaseMessage: BaseMessage{Command: CmdError, OpId: 1},
		Code:        POSIX_EINVAL,
		Message:     "Invalid operation",
		ErrorToken:  "scaci.error.test", // Internal tracking, not on wire per spec
	}

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "Error should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "Error should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in Error wire format (spec: §3.14.1)", key)
	}

	// Verify mandatory keys present
	assert.Contains(t, decoded, "command", "Error must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "Error must contain 'opId' (spec: §3.2)")
	assert.Contains(t, decoded, "code", "Error must contain 'code' (spec: §3.14.1)")
	assert.Contains(t, decoded, "message", "Error must contain 'message' (spec: §3.14.1)")
}

// TestPingResponseAssembly verifies PingResponse wire keys per SCACI §3.4.2
func TestPingResponseAssembly(t *testing.T) {
	// Expected keys per §3.4.2: command, opId (BaseMessage only)
	expectedKeys := map[string]bool{
		"command": true,
		"opId":    true,
	}

	msg := PingResponse{
		BaseMessage: BaseMessage{Command: CmdPingResponse, OpId: 5},
	}

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "PingResponse should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "PingResponse should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in PingResponse wire format (spec: §3.4.2)", key)
	}

	// Verify mandatory keys present
	assert.Contains(t, decoded, "command", "PingResponse must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "PingResponse must contain 'opId' (spec: §3.2)")
}

// TestRegisterResponseAssembly verifies RegisterResponse wire keys per SCACI §3.6.2
func TestRegisterResponseAssembly(t *testing.T) {
	// Expected keys per §3.6.2: command, opId (BaseMessage only)
	// RegisterResponse has no additional fields per spec
	expectedKeys := map[string]bool{
		"command": true,
		"opId":    true,
	}

	msg := RegisterResponse{
		BaseMessage: BaseMessage{Command: CmdRegisterResponse, OpId: 10},
	}

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "RegisterResponse should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "RegisterResponse should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in RegisterResponse wire format (spec: §3.6.2)", key)
	}

	// Verify mandatory keys present
	assert.Contains(t, decoded, "command", "RegisterResponse must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "RegisterResponse must contain 'opId' (spec: §3.2)")
}

// TestDLDataResultAssembly verifies DLDataResult wire keys per SCACI §3.12.1
func TestDLDataResultAssembly(t *testing.T) {
	// Expected keys per §3.12.1:
	// Required: command, opId, epEui, queId, result
	// Conditional (result=sent): bsEui, txTime, packetCnt
	expectedKeys := map[string]bool{
		"command":   true,
		"opId":      true,
		"epEui":     true,
		"queId":     true,
		"result":    true,
		"bsEui":     true,
		"txTime":    true,
		"packetCnt": true,
	}

	bsEui := uint64(0x1122334455667788)
	txTime := int64(1234567890)
	packetCnt := uint32(99)
	msg := DLDataResult{
		BaseMessage: BaseMessage{Command: CmdDLDataResult, OpId: -5},
		EpEui:       0x0102030405060708,
		QueID:       123,
		Result:      ResultSent,
		BsEui:       &bsEui,
		TxTime:      &txTime,
		PacketCnt:   &packetCnt,
	}

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "DLDataResult should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "DLDataResult should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in DLDataResult wire format (spec: §3.12.1)", key)
	}

	// Verify mandatory keys present
	assert.Contains(t, decoded, "command", "DLDataResult must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "DLDataResult must contain 'opId' (spec: §3.2)")
	assert.Contains(t, decoded, "epEui", "DLDataResult must contain 'epEui' (spec: §3.12.1)")
	assert.Contains(t, decoded, "queId", "DLDataResult must contain 'queId' (spec: §3.12.1)")
	assert.Contains(t, decoded, "result", "DLDataResult must contain 'result' (spec: §3.12.1)")
}

// TestEPStatusAssembly verifies EPStatus wire keys per SCACI §3.13.1
func TestEPStatusAssembly(t *testing.T) {
	// Expected keys per §3.13.1:
	// Required: command, opId, epEui, epStatus
	// Optional (OTA attach): attachCnt, nonce, sign
	expectedKeys := map[string]bool{
		"command":   true,
		"opId":      true,
		"epEui":     true,
		"epStatus":  true,
		"attachCnt": true,
		"nonce":     true,
		"sign":      true,
	}

	msg := EPStatus{
		BaseMessage: BaseMessage{Command: CmdEPStatus, OpId: -10},
		EpEui:       0x0102030405060708,
		EpStatus:    EPStatusAttached,
	}

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "EPStatus should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "EPStatus should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in EPStatus wire format (spec: §3.13.1)", key)
	}

	// Verify mandatory keys present
	assert.Contains(t, decoded, "command", "EPStatus must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "EPStatus must contain 'opId' (spec: §3.2)")
	assert.Contains(t, decoded, "epEui", "EPStatus must contain 'epEui' (spec: §3.13.1)")
	assert.Contains(t, decoded, "epStatus", "EPStatus must contain 'epStatus' (spec: §3.13.1)")
}

// TestDeregisterResponseAssembly verifies DeregisterResponse wire keys per SCACI §3.7.2
func TestDeregisterResponseAssembly(t *testing.T) {
	// Expected keys per §3.7.2: command, opId (BaseMessage only)
	expectedKeys := map[string]bool{
		"command": true,
		"opId":    true,
	}

	msg := DeregisterResponse{
		BaseMessage: BaseMessage{Command: CmdDeregisterResponse, OpId: 15},
	}

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "DeregisterResponse should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "DeregisterResponse should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in DeregisterResponse wire format (spec: §3.7.2)", key)
	}

	// Verify mandatory keys present
	assert.Contains(t, decoded, "command", "DeregisterResponse must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "DeregisterResponse must contain 'opId' (spec: §3.2)")
}

// TestULDataCompleteAssembly verifies ULDataComplete wire keys per SCACI §3.8.3
func TestULDataCompleteAssembly(t *testing.T) {
	// Expected keys per §3.8.3: command, opId (BaseMessage only)
	expectedKeys := map[string]bool{
		"command": true,
		"opId":    true,
	}

	msg := ULDataComplete{
		BaseMessage: BaseMessage{Command: CmdULDataComplete, OpId: 20},
	}

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "ULDataComplete should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "ULDataComplete should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in ULDataComplete wire format (spec: §3.8.3)", key)
	}

	// Verify mandatory keys present
	assert.Contains(t, decoded, "command", "ULDataComplete must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "ULDataComplete must contain 'opId' (spec: §3.2)")
}

// TestDLDataQueueResponseAssembly verifies DLDataQueueResponse wire keys per SCACI §3.10.2
// Note: DLDataQueueResponse is aliased to mioty.DLDataQueueResponse which embeds mioty.BaseMessage.
// The Go field is CommandType but serializes as "command" via msgpack tag.
func TestDLDataQueueResponseAssembly(t *testing.T) {
	// Expected keys per §3.10.2: command, opId (mioty.BaseMessage)
	// Wire key is "command" (via msgpack tag), not "commandType"
	expectedKeys := map[string]bool{
		"command": true,
		"opId":    true,
	}

	// DLDataQueueResponse is a mioty type alias - must use mioty.BaseMessage embedding
	msg := DLDataQueueResponse{}
	msg.CommandType = CmdDLDataQueueResponse
	msg.OpId = 25

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "DLDataQueueResponse should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "DLDataQueueResponse should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in DLDataQueueResponse wire format (spec: §3.10.2)", key)
	}

	// Verify mandatory keys present - wire key is "command" per msgpack tag
	assert.Contains(t, decoded, "command", "DLDataQueueResponse must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "DLDataQueueResponse must contain 'opId' (spec: §3.2)")
}

// TestDLDataRevokeResponseAssembly verifies DLDataRevokeResponse wire keys per SCACI §3.11.2
func TestDLDataRevokeResponseAssembly(t *testing.T) {
	// Expected keys per §3.11.2: command, opId (BaseMessage only)
	expectedKeys := map[string]bool{
		"command": true,
		"opId":    true,
	}

	msg := DLDataRevokeResponse{
		BaseMessage: BaseMessage{Command: CmdDLDataRevokeResponse, OpId: 30},
	}

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "DLDataRevokeResponse should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "DLDataRevokeResponse should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in DLDataRevokeResponse wire format (spec: §3.11.2)", key)
	}

	// Verify mandatory keys present
	assert.Contains(t, decoded, "command", "DLDataRevokeResponse must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "DLDataRevokeResponse must contain 'opId' (spec: §3.2)")
}

// TestDLDataResultCompleteAssembly verifies DLDataResultComplete wire keys per SCACI §3.12.3
func TestDLDataResultCompleteAssembly(t *testing.T) {
	// Expected keys per §3.12.3: command, opId (BaseMessage only)
	expectedKeys := map[string]bool{
		"command": true,
		"opId":    true,
	}

	msg := DLDataResultComplete{
		BaseMessage: BaseMessage{Command: CmdDLDataResultComplete, OpId: -35},
	}

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "DLDataResultComplete should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "DLDataResultComplete should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in DLDataResultComplete wire format (spec: §3.12.3)", key)
	}

	// Verify mandatory keys present
	assert.Contains(t, decoded, "command", "DLDataResultComplete must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "DLDataResultComplete must contain 'opId' (spec: §3.2)")
}

// ============================================================================
// §2.5 Wire-Key Tests for BaseMessage-Only Handshake Messages
// ============================================================================

// TestConnectCompleteAssembly verifies ConnectComplete wire keys per SCACI §3.3.3
// Per §3.3.3: conCmp contains only command and opId (BaseMessage fields)
func TestConnectCompleteAssembly(t *testing.T) {
	// Expected keys per §3.3.3: command, opId (BaseMessage only)
	expectedKeys := map[string]bool{
		"command": true,
		"opId":    true,
	}

	msg := ConnectComplete{
		BaseMessage: BaseMessage{Command: CmdConnectComplete, OpId: 0},
	}

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "ConnectComplete should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "ConnectComplete should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in ConnectComplete wire format (spec: §3.3.3)", key)
	}

	// Verify mandatory keys present
	assert.Contains(t, decoded, "command", "ConnectComplete must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "ConnectComplete must contain 'opId' (spec: §3.2)")

	// §3.3.3: opId must be 0 for conCmp
	assert.Equal(t, int64(0), decoded["opId"], "ConnectComplete opId must be 0 (spec: §3.3.3)")
}

// TestPingCompleteAssembly verifies PingComplete wire keys per SCACI §3.4.3
// Per §3.4.3: pingCmp contains only command and opId (BaseMessage fields)
func TestPingCompleteAssembly(t *testing.T) {
	// Expected keys per §3.4.3: command, opId (BaseMessage only)
	expectedKeys := map[string]bool{
		"command": true,
		"opId":    true,
	}

	msg := PingComplete{
		BaseMessage: BaseMessage{Command: CmdPingComplete, OpId: 42},
	}

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "PingComplete should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "PingComplete should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in PingComplete wire format (spec: §3.4.3)", key)
	}

	// Verify mandatory keys present
	assert.Contains(t, decoded, "command", "PingComplete must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "PingComplete must contain 'opId' (spec: §3.2)")
}

// TestStatusCompleteAssembly verifies StatusComplete wire keys per SCACI §3.5.3
// Per §3.5.3: statCmp contains only command and opId (BaseMessage fields)
func TestStatusCompleteAssembly(t *testing.T) {
	// Expected keys per §3.5.3: command, opId (BaseMessage only)
	expectedKeys := map[string]bool{
		"command": true,
		"opId":    true,
	}

	msg := StatusComplete{
		BaseMessage: BaseMessage{Command: CmdStatusComplete, OpId: 99},
	}

	data, err := msgpack.Marshal(msg)
	require.NoError(t, err, "StatusComplete should marshal to MessagePack")

	var decoded map[string]interface{}
	err = msgpack.Unmarshal(data, &decoded)
	require.NoError(t, err, "StatusComplete should unmarshal to map")

	for key := range decoded {
		assert.True(t, expectedKeys[key], "Unexpected key %q in StatusComplete wire format (spec: §3.5.3)", key)
	}

	// Verify mandatory keys present
	assert.Contains(t, decoded, "command", "StatusComplete must contain 'command' (spec: §3.2)")
	assert.Contains(t, decoded, "opId", "StatusComplete must contain 'opId' (spec: §3.2)")
}
