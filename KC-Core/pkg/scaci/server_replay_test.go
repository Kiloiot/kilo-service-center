// Package scaci provides tests for SCACI §1 pending operation replay logic
//
// Coverage:
//   - extractInt64FromJSON: JSON numeric type coercion
//   - ParseEUI64: Hex string to uint64 parsing
//   - replayableCommands: Command whitelist verification
//   - replayULData: Corrupted userData rejection, happy path
//   - Cross-tenant replay rejection
package scaci

import (
	"bytes"
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// ============================================================================
// R4a: Unit Tests for extractInt64FromJSON
// ============================================================================

func TestExtractInt64FromJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		key      string
		expected int64
		ok       bool
	}{
		{
			name:     "float64 value (JSON standard)",
			data:     map[string]interface{}{"count": float64(42)},
			key:      "count",
			expected: 42,
			ok:       true,
		},
		{
			name:     "int64 value",
			data:     map[string]interface{}{"count": int64(123)},
			key:      "count",
			expected: 123,
			ok:       true,
		},
		{
			name:     "int value",
			data:     map[string]interface{}{"count": int(999)},
			key:      "count",
			expected: 999,
			ok:       true,
		},
		{
			name:     "missing key",
			data:     map[string]interface{}{"other": float64(1)},
			key:      "count",
			expected: 0,
			ok:       false,
		},
		{
			name:     "nil map",
			data:     nil,
			key:      "count",
			expected: 0,
			ok:       false,
		},
		{
			name:     "string value (wrong type)",
			data:     map[string]interface{}{"count": "42"},
			key:      "count",
			expected: 0,
			ok:       false,
		},
		{
			name:     "negative float64",
			data:     map[string]interface{}{"opId": float64(-42)},
			key:      "opId",
			expected: -42,
			ok:       true,
		},
		{
			name:     "large float64",
			data:     map[string]interface{}{"ts": float64(1759348542000000000)},
			key:      "ts",
			expected: 1759348542000000000,
			ok:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := extractInt64FromJSON(tc.data, tc.key)
			assert.Equal(t, tc.expected, got)
			assert.Equal(t, tc.ok, ok)
		})
	}
}

// ============================================================================
// R4b: Unit Tests for ParseEUI64
// ============================================================================

func TestParseEUI64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected uint64
		wantErr  bool
	}{
		{
			name:     "lowercase hex",
			input:    "70b3d59cd00009e6",
			expected: 0x70b3d59cd00009e6,
			wantErr:  false,
		},
		{
			name:     "uppercase hex",
			input:    "70B3D59CD00009E6",
			expected: 0x70b3d59cd00009e6,
			wantErr:  false,
		},
		{
			name:     "mixed case hex",
			input:    "70B3d59cD00009E6",
			expected: 0x70b3d59cd00009e6,
			wantErr:  false,
		},
		{
			name:     "zero EUI",
			input:    "0000000000000000",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "max EUI",
			input:    "FFFFFFFFFFFFFFFF",
			expected: 0xFFFFFFFFFFFFFFFF,
			wantErr:  false,
		},
		{
			name:     "short string",
			input:    "70b3d59",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "long string",
			input:    "70b3d59cd00009e612345",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "invalid hex char at start",
			input:    "zz00000000000000",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseEUI64(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, got)
			}
		})
	}
}

// ============================================================================
// R4c: Unit Tests for replayableCommands whitelist
// ============================================================================

func TestReplayableCommands(t *testing.T) {
	// Only SC-originated operations with negative opIds should be replayable
	// Per SCACI §1: "only operations, which had not been completed before the connection loss are reissued"
	t.Run("ulData is replayable", func(t *testing.T) {
		assert.True(t, replayableCommands[CmdULData], "ulData MUST be replayable per SCACI §3.8")
	})

	t.Run("dlDataRes is replayable", func(t *testing.T) {
		assert.True(t, replayableCommands[CmdDLDataResult], "dlDataRes MUST be replayable per SCACI §3.12")
	})

	t.Run("AC-initiated commands are NOT replayable", func(t *testing.T) {
		acCommands := []string{
			CmdConnect, CmdConnectComplete,
			CmdRegister, CmdRegisterComplete,
			CmdDeregister, CmdDeregisterComplete,
			CmdULDataResponse, CmdULDataTransmit, CmdULDataTransmitComplete,
			CmdDLDataQueue, CmdDLDataQueueComplete,
			CmdDLDataRevoke, CmdDLDataRevokeComplete,
		}
		for _, cmd := range acCommands {
			assert.False(t, replayableCommands[cmd], "%s should NOT be replayable (AC-initiated)", cmd)
		}
	})
}

// ============================================================================
// R4d: Unit Test for corrupted userData rejection
// ============================================================================

func TestReplayULData_CorruptedUserData_ReturnsError(t *testing.T) {
	// Setup: operation with invalid hex userData
	op := &models.SCACIOperation{
		OpId:      -42,
		TenantID:  1,
		SessionID: 100,
		Command:   CmdULData,
		Direction: string(models.OperationDirectionOutbound),
		RequestData: map[string]interface{}{
			"epEui":    "0102030405060708",
			"userData": "ZZZZ_NOT_HEX", // Invalid hex
		},
	}

	session := &Session{
		ID:       100,
		TenantID: 1,
		WriteMu:  sync.Mutex{},
	}

	// Create server with nop logger
	s := &Server{
		logger: logger.NewNop(),
	}

	// Create net.Pipe for connection (won't actually be used due to early error)
	clientConn, serverConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()
	defer func() { _ = serverConn.Close() }()

	// Act: replayULData should return error for corrupted userData
	err := s.replayULData(serverConn, session, op)

	// Assert: error returned, contains descriptive message
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode userData for replay")
}

// ============================================================================
// R4e: Unit Test for cross-tenant replay rejection
// ============================================================================

// mockOperationRepo implements interfaces.SCACIOperationRepository for testing
type mockOperationRepo struct {
	operations []*models.SCACIOperation
	err        error
}

func (m *mockOperationRepo) RecordOperation(_ context.Context, _ *models.SCACIOperationRequest) (*models.SCACIOperation, error) {
	return nil, nil // Stub
}

func (m *mockOperationRepo) UpdateOperationState(_ context.Context, _ int64, _ int64, _ models.OperationState, _ map[string]interface{}) error {
	return nil // Stub
}

func (m *mockOperationRepo) GetOperationByOpID(_ context.Context, _ int64, _ int64) (*models.SCACIOperation, error) {
	return nil, nil // Stub
}

func (m *mockOperationRepo) GetPendingOperations(_ context.Context, _ int64) ([]*models.SCACIOperation, error) {
	return m.operations, m.err
}

func (m *mockOperationRepo) GetRecentOperations(_ context.Context, _ int64, _ int) ([]*models.SCACIOperation, error) {
	return nil, nil // Stub
}

func (m *mockOperationRepo) CleanupCompletedOperations(_ context.Context, _ int64) (int64, error) {
	return 0, nil // Stub
}

func (m *mockOperationRepo) GetTenantOperationSummary(_ context.Context, _ int64, _ int) (*models.SCACIOperationSummary, error) {
	return nil, nil // Stub
}

func (m *mockOperationRepo) UpdateOperationStateWithError(_ context.Context, _ int64, _ int64, _ models.OperationState, _ int, _ string, _ string, _ map[string]interface{}) error {
	return nil // Stub
}

func (m *mockOperationRepo) CompleteFailedOperation(_ context.Context, _ int64, _ int64, _ map[string]interface{}) error {
	return nil // Stub
}

// Compile-time check that mockOperationRepo satisfies the interface
var _ interfaces.SCACIOperationRepository = (*mockOperationRepo)(nil)

func TestReplayPendingOperations_CrossTenantRejected(t *testing.T) {
	// Setup: operation belongs to tenant 99, session belongs to tenant 1
	crossTenantOp := &models.SCACIOperation{
		OpId:      -1, // SC-originated
		TenantID:  99, // Different tenant
		SessionID: 100,
		Command:   CmdULData,
		Direction: string(models.OperationDirectionOutbound),
		RequestData: map[string]interface{}{
			"epEui":    "0102030405060708",
			"userData": "deadbeef",
		},
	}

	mockRepo := &mockOperationRepo{
		operations: []*models.SCACIOperation{crossTenantOp},
	}

	// Capture log output with observer
	observedLogs := bsscitest.NewRecordingLogger() // captures DEBUG+
	testLogger := observedLogs

	session := &Session{
		ID:       100,
		TenantID: 1, // Session tenant is 1
	}

	s := &Server{
		operationRepo: mockRepo,
		logger:        testLogger,
	}

	// Create connection
	clientConn, serverConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()
	defer func() { _ = serverConn.Close() }()

	// Act
	s.replayPendingOperations(serverConn, session)

	// Assert: cross-tenant rejection logged
	logs := observedLogs.AllAtLeast("DEBUG")
	found := false
	for _, entry := range logs {
		if entry.Message == LogSCACICrossTenantReplayRejected {
			found = true
			// Verify context fields are present
			entryFields := entry.FieldMap()
			assert.Equal(t, int64(99), entryFields["opTenantID"])
			break
		}
	}
	assert.True(t, found, "expected cross-tenant rejection log entry")
}

// ============================================================================
// R4f: Happy Path Test - OpId preservation and field reconstruction
// ============================================================================

func TestReplayULData_HappyPath_PreservesOpId(t *testing.T) {
	originalOpId := int64(-42)
	testEpEui := "0102030405060708"
	testUserData := "deadbeef" // Valid hex

	op := &models.SCACIOperation{
		OpId:      originalOpId,
		TenantID:  1,
		SessionID: 100,
		Command:   CmdULData,
		Direction: string(models.OperationDirectionOutbound),
		RequestData: map[string]interface{}{
			"epEui": testEpEui,
			"baseStations": []interface{}{
				map[string]interface{}{
					"bsEui":  "1122334455667788",
					"rxTime": float64(1234567890),
				},
			},
			"userData":    testUserData,
			"packetCnt":   float64(123), // JSON stores numbers as float64
			"dlOpen":      true,
			"responseExp": false,
		},
	}

	session := &Session{
		ID:       100,
		TenantID: 1,
		WriteMu:  sync.Mutex{},
	}

	s := &Server{
		logger: logger.NewNop(),
	}

	// Setup net.Pipe to capture wire output
	clientConn, serverConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()
	defer func() { _ = serverConn.Close() }()

	// Async reader to capture sent data
	var captured bytes.Buffer
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		// Read with timeout - expect small message
		_ = clientConn.SetReadDeadline(time.Now().Add(time.Second))
		_, _ = io.Copy(&captured, clientConn)
	}()

	// Act
	err := s.replayULData(serverConn, session, op)
	_ = serverConn.Close() // Trigger EOF on client side to complete read

	// Wait for read to complete
	<-readDone

	// Assert: no error
	require.NoError(t, err)

	// Assert: data was sent
	require.True(t, captured.Len() > 0, "expected wire data to be captured")

	// Parse the SCACI frame: MIOTYA01 header (8 bytes) + 4-byte size + MessagePack payload
	data := captured.Bytes()
	if len(data) < 12 {
		t.Fatalf("captured data too short: %d bytes", len(data))
	}

	// Verify MIOTYA01 header
	assert.Equal(t, "MIOTYA01", string(data[:8]), "frame header mismatch")

	// Extract payload (skip 8-byte header + 4-byte size)
	payload := data[12:]

	// Decode MessagePack payload
	var decoded map[string]interface{}
	err = msgpack.Unmarshal(payload, &decoded)
	require.NoError(t, err, "failed to decode msgpack payload")

	// Verify opId is preserved (critical for replay)
	opIdVal, ok := decoded["opId"]
	require.True(t, ok, "opId field missing from replay message")
	// msgpack may decode as int8/int16/int32/int64 depending on value
	var actualOpId int64
	switch v := opIdVal.(type) {
	case int8:
		actualOpId = int64(v)
	case int16:
		actualOpId = int64(v)
	case int32:
		actualOpId = int64(v)
	case int64:
		actualOpId = v
	default:
		t.Fatalf("unexpected opId type: %T", opIdVal)
	}
	assert.Equal(t, originalOpId, actualOpId, "opId must be preserved for replay")

	// Verify command is ulData
	cmdVal, ok := decoded["command"]
	require.True(t, ok, "command field missing")
	assert.Equal(t, CmdULData, cmdVal, "command mismatch")
}

// ============================================================================
// R4g: Test ReplayOperationTimeout constant is used
// ============================================================================

func TestReplayOperationTimeout_Constant(t *testing.T) {
	// Verify constant is defined and has expected value
	assert.Equal(t, 5*time.Second, ReplayOperationTimeout,
		"ReplayOperationTimeout should be 5s (matches ConnectPersistTimeout)")
}

// ============================================================================
// R9a: Test replayDLDataResult Happy Path - OpId preservation
// ============================================================================

func TestReplayDLDataResult_HappyPath_PreservesOpId(t *testing.T) {
	originalOpId := int64(-99) // SC-originated negative opId
	testEpEui := "0102030405060708"
	testQueId := float64(12345) // JSON stores as float64

	op := &models.SCACIOperation{
		OpId:      originalOpId,
		TenantID:  1,
		SessionID: 100,
		Command:   CmdDLDataResult,
		Direction: string(models.OperationDirectionOutbound),
		RequestData: map[string]interface{}{
			"epEui":  testEpEui,
			"queId":  testQueId,
			"result": ResultExpired, // Valid SCACI §3.12.1 enum value
		},
	}

	session := &Session{
		ID:       100,
		TenantID: 1,
		WriteMu:  sync.Mutex{},
	}

	s := &Server{
		logger: logger.NewNop(),
	}

	// Setup net.Pipe to capture wire output
	clientConn, serverConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()
	defer func() { _ = serverConn.Close() }()

	// Async reader to capture sent data
	var captured bytes.Buffer
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		_ = clientConn.SetReadDeadline(time.Now().Add(time.Second))
		_, _ = io.Copy(&captured, clientConn)
	}()

	// Act
	err := s.replayDLDataResult(serverConn, session, op)
	_ = serverConn.Close() // Trigger EOF

	// Wait for read to complete
	<-readDone

	// Assert: no error
	require.NoError(t, err)

	// Assert: data was sent
	require.True(t, captured.Len() > 0, "expected wire data to be captured")

	// Parse the SCACI frame and verify opId preservation
	data := captured.Bytes()
	if len(data) < 12 {
		t.Fatalf("captured data too short: %d bytes", len(data))
	}

	// Verify MIOTYA01 header
	assert.Equal(t, "MIOTYA01", string(data[:8]), "frame header mismatch")

	// Extract and decode payload
	payload := data[12:]
	var decoded map[string]interface{}
	err = msgpack.Unmarshal(payload, &decoded)
	require.NoError(t, err, "failed to decode msgpack payload")

	// Verify opId is preserved (critical for replay)
	opIdVal, ok := decoded["opId"]
	require.True(t, ok, "opId field missing from replay message")

	// msgpack decodes small negatives as int8/int16
	var actualOpId int64
	switch v := opIdVal.(type) {
	case int8:
		actualOpId = int64(v)
	case int16:
		actualOpId = int64(v)
	case int32:
		actualOpId = int64(v)
	case int64:
		actualOpId = v
	default:
		t.Fatalf("unexpected opId type: %T", opIdVal)
	}
	assert.Equal(t, originalOpId, actualOpId, "opId must be preserved for replay")

	// Verify command is dlDataRes
	cmdVal, ok := decoded["command"]
	require.True(t, ok, "command field missing")
	assert.Equal(t, CmdDLDataResult, cmdVal, "command mismatch")
}

// ============================================================================
// R9b: Test Positive Replay Path (Matching Tenant)
// ============================================================================

func TestReplayPendingOperations_PositivePath_MatchingTenant(t *testing.T) {
	// Setup: operation belongs to same tenant as session
	validOp := &models.SCACIOperation{
		OpId:      -1, // SC-originated
		TenantID:  1,  // Same as session
		SessionID: 100,
		Command:   CmdULData,
		Direction: string(models.OperationDirectionOutbound),
		RequestData: map[string]interface{}{
			"epEui":    "0102030405060708",
			"userData": "deadbeef", // Valid hex
		},
	}

	mockRepo := &mockOperationRepo{
		operations: []*models.SCACIOperation{validOp},
	}

	session := &Session{
		ID:       100,
		TenantID: 1, // Matches operation
		WriteMu:  sync.Mutex{},
	}

	// Capture log output with observer
	observedLogs := bsscitest.NewRecordingLogger() // captures DEBUG+
	testLogger := observedLogs

	s := &Server{
		operationRepo: mockRepo,
		logger:        testLogger,
	}

	// Create connection
	clientConn, serverConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()

	// Async reader to drain writes
	go func() {
		_, _ = io.Copy(io.Discard, clientConn)
	}()

	// Act
	s.replayPendingOperations(serverConn, session)
	_ = serverConn.Close()

	// Assert: replay was logged (positive path, not rejection)
	logs := observedLogs.AllAtLeast("DEBUG")
	foundReplayLog := false
	foundRejectionLog := false
	for _, entry := range logs {
		if entry.Message == LogSCACIReplayingPendingOp {
			foundReplayLog = true
		}
		if entry.Message == LogSCACICrossTenantReplayRejected {
			foundRejectionLog = true
		}
	}
	assert.True(t, foundReplayLog, "expected replay log entry for matching tenant")
	assert.False(t, foundRejectionLog, "should NOT have cross-tenant rejection for matching tenant")
}

// ============================================================================
// R9c: Test Corrupted UserData Logs Correct Token
// ============================================================================

func TestReplayULData_CorruptedUserData_LogsCorrectToken(t *testing.T) {
	// Setup: operation with invalid hex userData
	op := &models.SCACIOperation{
		OpId:      -42,
		TenantID:  1,
		SessionID: 100,
		Command:   CmdULData,
		Direction: string(models.OperationDirectionOutbound),
		RequestData: map[string]interface{}{
			"epEui":    "0102030405060708",
			"userData": "ZZZZ_NOT_HEX", // Invalid hex
		},
	}

	session := &Session{
		ID:       100,
		TenantID: 1,
		WriteMu:  sync.Mutex{},
	}

	// Capture log output with observer at Error level
	observedLogs := bsscitest.NewRecordingLogger() // captures ERROR+
	testLogger := observedLogs

	s := &Server{
		logger: testLogger,
	}

	// Create net.Pipe for connection
	clientConn, serverConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()
	defer func() { _ = serverConn.Close() }()

	// Act: replayULData should return error for corrupted userData
	err := s.replayULData(serverConn, session, op)

	// Assert: error returned
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode userData for replay")

	// Assert: LogSCACIReplayUserDataCorrupted was logged
	logs := observedLogs.AllAtLeast("ERROR")
	found := false
	for _, entry := range logs {
		if entry.Message == LogSCACIReplayUserDataCorrupted {
			found = true
			// Verify context fields
			entryFields := entry.FieldMap()
			assert.Equal(t, int64(-42), entryFields["opId"])
			assert.Equal(t, "ZZZZ_NOT_HEX", entryFields["storedValue"])
			break
		}
	}
	assert.True(t, found, "expected LogSCACIReplayUserDataCorrupted log entry")
}

// ============================================================================
// Gap 1 Test: Handler-path verification for POSIX_ENOTSUP (95) on wire
// Validates that sendErrorWithCatalog centralizes POSIXCode resolution
// ============================================================================

// TestHandleConnect_VersionMismatch_SendsPOSIXEnotsup verifies that when
// HandshakeService returns ErrVersionMismatchOnResume, the handler sends
// POSIX_ENOTSUP (95) on the wire - not the default POSIX_EINVAL (22).
// This validates the centralized POSIXCode resolution in sendErrorWithCatalog.
func TestHandleConnect_VersionMismatch_SendsPOSIXEnotsup(t *testing.T) {
	// Setup net.Pipe
	clientConn, serverConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()
	defer func() { _ = serverConn.Close() }()

	// Setup mock handshake service to return ErrVersionMismatchOnResume
	mockHS := new(MockHandshakeService)
	mockHS.On("ValidateConnect", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil, ErrVersionMismatchOnResume)

	// Setup mock session validator to pass (Connect field validation OK)
	mockValidator := new(MockSessionValidator)
	mockValidator.On("ValidateConnectFields", mock.Anything, mock.Anything).
		Return("") // Empty = no error

	// Create server with required dependencies
	s := &Server{
		logger:           logger.NewNop(),
		handshakeSvc:     mockHS,
		sessionValidator: mockValidator,
	}

	// Capture wire output async
	var captured bytes.Buffer
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		_ = clientConn.SetReadDeadline(time.Now().Add(time.Second))
		_, _ = io.Copy(&captured, clientConn)
	}()

	// Build real Connect payload (MessagePack)
	connectMsg := Connect{
		BaseMessage: BaseMessage{Command: CmdConnect, OpId: OpIDConnect},
		Version:     "1.0.0",
		AcEui:       0x0102030405060708,
		SnAcUUID:    UUID16{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}
	payload, err := msgpack.Marshal(&connectMsg)
	require.NoError(t, err, "failed to marshal Connect message")

	// Act: call handleConnect directly (bypasses TLS, like existing replay tests)
	session := (*Session)(nil)
	_ = s.handleConnect(serverConn, &session, nil, OpIDConnect, payload)
	_ = serverConn.Close()
	<-readDone

	// Assert: data was sent
	require.True(t, captured.Len() > 0, "expected wire data to be captured")

	// Parse MIOTYA01 frame: 8-byte header + 4-byte length + MessagePack payload
	data := captured.Bytes()
	require.True(t, len(data) >= 12, "captured data too short for SCACI frame")

	// Verify MIOTYA01 header
	assert.Equal(t, "MIOTYA01", string(data[:8]), "frame header mismatch")

	// Extract and decode payload
	framePayload := data[12:]
	var decoded map[string]interface{}
	err = msgpack.Unmarshal(framePayload, &decoded)
	require.NoError(t, err, "failed to decode msgpack payload")

	// Assert: error frame has command="error"
	cmdVal, ok := decoded["command"]
	require.True(t, ok, "command field missing from response")
	assert.Equal(t, CmdError, cmdVal, "expected error command")

	// Assert: error frame has code=POSIX_ENOTSUP (95)
	codeVal, ok := decoded["code"]
	require.True(t, ok, "code field missing from error response")

	// Handle msgpack integer type coercion
	var actualCode int
	switch v := codeVal.(type) {
	case int8:
		actualCode = int(v)
	case int16:
		actualCode = int(v)
	case int32:
		actualCode = int(v)
	case int64:
		actualCode = int(v)
	case uint8:
		actualCode = int(v)
	case uint16:
		actualCode = int(v)
	case uint32:
		actualCode = int(v)
	case uint64:
		actualCode = int(v)
	default:
		t.Fatalf("unexpected code type: %T", codeVal)
	}

	assert.Equal(t, POSIX_ENOTSUP, actualCode,
		"ErrVersionMismatchOnResume must send POSIX_ENOTSUP=95 on wire (not POSIX_EINVAL=22)")

	// Verify mocks were called as expected
	mockHS.AssertExpectations(t)
	mockValidator.AssertExpectations(t)
}
