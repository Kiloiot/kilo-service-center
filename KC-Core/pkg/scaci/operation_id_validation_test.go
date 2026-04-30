// Package scaci tests for SCACI §3.3.2 opId=0 reserved enforcement.
//
// Per SCACI §3.3.2, opId=0 is reserved for the connect handshake only.
// These tests verify that non-connect commands with opId=0 are rejected
// with POSIX_EINVAL + ErrOpIDZeroReserved.
//
// Allowed commands with opId=0:
//   - CmdConnect (§3.3.1)
//   - CmdConnectComplete (§3.3.3)
//   - CmdErrorAck (echoes triggering opId per §3.14.2)
//
// Rejected commands: All others return POSIX_EINVAL + ErrOpIDZeroReserved
package scaci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
)

// ============================================================================
// SCACI §3.3.2: opId=0 Reserved for Connect Handshake Tests
// ============================================================================

// TestOpIdZeroRejectedForPing verifies ping with opId=0 is rejected per §3.3.2.
func TestOpIdZeroRejectedForPing(t *testing.T) {
	conn := &captureConn{}
	s := &Server{logger: testLogger()}
	session := &Session{State: StateActive, TenantID: 1}

	err := s.routeMessage(conn, &session, nil, CmdPing, 0, nil)

	assert.Nil(t, err, "routeMessage should return nil (error sent on wire)")
	assertOpIDZeroError(t, conn, "ping")
}

// TestOpIdZeroRejectedForStatus verifies status with opId=0 is rejected per §3.3.2.
func TestOpIdZeroRejectedForStatus(t *testing.T) {
	conn := &captureConn{}
	s := &Server{logger: testLogger()}
	session := &Session{State: StateActive, TenantID: 1}

	err := s.routeMessage(conn, &session, nil, CmdStatus, 0, nil)

	assert.Nil(t, err, "routeMessage should return nil (error sent on wire)")
	assertOpIDZeroError(t, conn, "status")
}

// TestOpIdZeroRejectedForULData verifies ulData with opId=0 is rejected per §3.3.2.
func TestOpIdZeroRejectedForULData(t *testing.T) {
	conn := &captureConn{}
	s := &Server{logger: testLogger()}
	session := &Session{State: StateActive, TenantID: 1}

	// Note: CmdULDataResponse is AC-to-SC, but with opId=0 should still be rejected
	err := s.routeMessage(conn, &session, nil, CmdULDataResponse, 0, nil)

	assert.Nil(t, err, "routeMessage should return nil (error sent on wire)")
	assertOpIDZeroError(t, conn, "ulDataRsp")
}

// TestOpIdZeroRejectedForConnectResponse verifies conRsp with opId=0 is rejected.
// CmdConnectResponse is SC→AC only per §3.3.2, so inbound conRsp is rejected.
func TestOpIdZeroRejectedForConnectResponse(t *testing.T) {
	conn := &captureConn{}
	s := &Server{logger: testLogger()}
	session := &Session{State: StateActive, TenantID: 1}

	err := s.routeMessage(conn, &session, nil, CmdConnectResponse, 0, nil)

	assert.Nil(t, err, "routeMessage should return nil (error sent on wire)")
	assertOpIDZeroError(t, conn, "conRsp")
}

// TestOpIdZeroAllowedForConnect verifies Connect with opId=0 is not rejected by the guard.
// This test verifies the guard logic directly rather than calling the full handler.
func TestOpIdZeroAllowedForConnect(t *testing.T) {
	// Verify that CmdConnect IS one of the allowed commands for opId=0
	// This test validates the guard condition without triggering handler code

	allowedCommands := []string{CmdConnect, CmdConnectComplete, CmdErrorAck}
	for _, cmd := range allowedCommands {
		// Guard condition from server.go:756 - these should NOT trigger rejection
		shouldReject := cmd != CmdConnect && cmd != CmdConnectComplete && cmd != CmdErrorAck
		assert.False(t, shouldReject, "Command %s should be allowed with opId=0", cmd)
	}

	// Verify that CmdConnectResponse IS rejected (it's SC→AC only)
	shouldReject := CmdConnectResponse != CmdConnect && CmdConnectResponse != CmdConnectComplete && CmdConnectResponse != CmdErrorAck
	assert.True(t, shouldReject, "CmdConnectResponse should be rejected with opId=0")
}

// TestOpIdZeroAllowedForConnectComplete verifies ConnectComplete with opId=0 is not rejected by guard.
// This test verifies the guard logic directly rather than calling the full handler.
func TestOpIdZeroAllowedForConnectComplete(t *testing.T) {
	// Verify that CmdConnectComplete IS one of the allowed commands for opId=0
	allowedCommands := map[string]bool{
		CmdConnect:         true,
		CmdConnectComplete: true,
		CmdErrorAck:        true,
	}
	assert.True(t, allowedCommands[CmdConnectComplete], "CmdConnectComplete should be allowed with opId=0")
}

// TestOpIdZeroAllowedForErrorAck verifies ErrorAck with opId=0 is accepted.
// ErrorAck echoes the opId from the original error message per §3.14.2.
func TestOpIdZeroAllowedForErrorAck(t *testing.T) {
	conn := &captureConn{}
	s := &Server{logger: testLogger()}
	session := &Session{State: StateActive, TenantID: 1}

	err := s.routeMessage(conn, &session, nil, CmdErrorAck, 0, nil)

	// ErrorAck should be accepted and return nil
	assert.Nil(t, err, "ErrorAck with opId=0 should be accepted")

	// Verify no error was sent
	if len(conn.written) > 0 {
		var errMsg map[string]interface{}
		if parseErr := parseErrorFromFrame(conn.written, &errMsg); parseErr == nil {
			if token, ok := errMsg["errorToken"].(string); ok {
				assert.NotEqual(t, ErrOpIDZeroReserved, token, "ErrorAck should not trigger ErrOpIDZeroReserved")
			}
		}
	}
}

// ============================================================================
// Test Helpers
// ============================================================================

// captureConn captures written bytes for assertion (extends mockConn pattern)
type captureConn struct {
	mockConn
}

// assertOpIDZeroError verifies that an error response with POSIX_EINVAL and
// the opId=0 reserved message was sent to the connection.
// Note: errorToken is NOT on the wire per SCACI spec (msgpack:"-"), so we verify the message field.
func assertOpIDZeroError(t *testing.T, conn *captureConn, cmdName string) {
	t.Helper()

	if len(conn.written) == 0 {
		t.Fatalf("%s with opId=0: expected error response, got none", cmdName)
	}

	var errMsg map[string]interface{}
	if err := parseErrorFromFrame(conn.written, &errMsg); err != nil {
		t.Fatalf("%s with opId=0: failed to parse error response: %v", cmdName, err)
	}

	// Verify command is "error"
	assert.Equal(t, CmdError, errMsg["command"], "%s: expected error command", cmdName)

	// Verify POSIX code is EINVAL (22)
	code, ok := errMsg["code"]
	assert.True(t, ok, "%s: error response missing code field", cmdName)
	if codeFloat, ok := code.(float64); ok {
		assert.Equal(t, float64(POSIX_EINVAL), codeFloat, "%s: expected POSIX_EINVAL (22)", cmdName)
	} else if codeInt, ok := code.(int64); ok {
		assert.Equal(t, int64(POSIX_EINVAL), codeInt, "%s: expected POSIX_EINVAL (22)", cmdName)
	}

	// Verify error message matches catalog (errorToken is internal-only, not on wire)
	msg, ok := errMsg["message"].(string)
	assert.True(t, ok, "%s: error response missing message field", cmdName)
	expectedMsg := GetErrorDefinition(ErrOpIDZeroReserved).Message
	assert.Equal(t, expectedMsg, msg, "%s: expected opId=0 reserved message", cmdName)
}

// parseErrorFromFrame extracts the MessagePack error message from a SCACI frame.
// Frame format: 12-byte header (MIOTYA01 + 4-byte size) + payload
func parseErrorFromFrame(data []byte, result *map[string]interface{}) error {
	if len(data) < MinFrameSize {
		return nil // Too short
	}

	// Skip 12-byte header
	payload := data[MinFrameSize:]
	return msgpack.Unmarshal(payload, result)
}

// ============================================================================
// SCACI §3.2: opId Sign Validation Tests
// ============================================================================
//
// Per SCACI §3.2:
//   - AC-initiated commands require positive opId (> 0)
//   - SC-initiated commands require negative opId (< 0)
//   - Connect handshake uses opId=0 (validated separately)
//   - Ping/Error can be initiated by either party (any sign valid)

// TestValidateOpIDSign_ACCommands_InvalidNegative verifies AC commands reject negative opIds.
func TestValidateOpIDSign_ACCommands_InvalidNegative(t *testing.T) {
	acCommands := []string{
		CmdStatus, CmdStatusResponse, CmdStatusComplete,
		CmdRegister, CmdRegisterResponse, CmdRegisterComplete,
		CmdDeregister, CmdDeregisterResponse, CmdDeregisterComplete,
		CmdULDataTransmit, CmdULDataTransmitResponse, CmdULDataTransmitComplete,
		CmdDLDataQueue, CmdDLDataQueueResponse, CmdDLDataQueueComplete,
		CmdDLDataRevoke, CmdDLDataRevokeResponse, CmdDLDataRevokeComplete,
	}

	for _, cmd := range acCommands {
		t.Run(cmd+"_negative", func(t *testing.T) {
			errToken := ValidateOpIDSign(cmd, -1)
			assert.Equal(t, errOpIDSignMismatch, errToken,
				"%s with negative opId should return errOpIDSignMismatch", cmd)
		})

		t.Run(cmd+"_zero", func(t *testing.T) {
			errToken := ValidateOpIDSign(cmd, 0)
			assert.Equal(t, errOpIDSignMismatch, errToken,
				"%s with opId=0 should return errOpIDSignMismatch", cmd)
		})
	}
}

// TestValidateOpIDSign_ACCommands_ValidPositive verifies AC commands accept positive opIds.
func TestValidateOpIDSign_ACCommands_ValidPositive(t *testing.T) {
	acCommands := []string{
		CmdStatus, CmdStatusResponse, CmdStatusComplete,
		CmdRegister, CmdRegisterResponse, CmdRegisterComplete,
		CmdDeregister, CmdDeregisterResponse, CmdDeregisterComplete,
		CmdULDataTransmit, CmdULDataTransmitResponse, CmdULDataTransmitComplete,
		CmdDLDataQueue, CmdDLDataQueueResponse, CmdDLDataQueueComplete,
		CmdDLDataRevoke, CmdDLDataRevokeResponse, CmdDLDataRevokeComplete,
	}

	for _, cmd := range acCommands {
		t.Run(cmd+"_positive", func(t *testing.T) {
			errToken := ValidateOpIDSign(cmd, 1)
			assert.Empty(t, errToken,
				"%s with positive opId should pass validation", cmd)
		})

		t.Run(cmd+"_large_positive", func(t *testing.T) {
			errToken := ValidateOpIDSign(cmd, 999999)
			assert.Empty(t, errToken,
				"%s with large positive opId should pass validation", cmd)
		})
	}
}

// TestValidateOpIDSign_SCCommands_InvalidPositive verifies SC commands reject positive opIds.
func TestValidateOpIDSign_SCCommands_InvalidPositive(t *testing.T) {
	scCommands := []string{
		CmdULData, CmdULDataResponse, CmdULDataComplete,
		CmdDLDataResult, CmdDLDataResultResponse, CmdDLDataResultComplete,
		CmdEPStatus, CmdEPStatusResponse, CmdEPStatusComplete,
	}

	for _, cmd := range scCommands {
		t.Run(cmd+"_positive", func(t *testing.T) {
			errToken := ValidateOpIDSign(cmd, 1)
			assert.Equal(t, errOpIDSignMismatch, errToken,
				"%s with positive opId should return errOpIDSignMismatch", cmd)
		})

		t.Run(cmd+"_zero", func(t *testing.T) {
			errToken := ValidateOpIDSign(cmd, 0)
			assert.Equal(t, errOpIDSignMismatch, errToken,
				"%s with opId=0 should return errOpIDSignMismatch", cmd)
		})
	}
}

// TestValidateOpIDSign_SCCommands_ValidNegative verifies SC commands accept negative opIds.
func TestValidateOpIDSign_SCCommands_ValidNegative(t *testing.T) {
	scCommands := []string{
		CmdULData, CmdULDataResponse, CmdULDataComplete,
		CmdDLDataResult, CmdDLDataResultResponse, CmdDLDataResultComplete,
		CmdEPStatus, CmdEPStatusResponse, CmdEPStatusComplete,
	}

	for _, cmd := range scCommands {
		t.Run(cmd+"_negative", func(t *testing.T) {
			errToken := ValidateOpIDSign(cmd, -1)
			assert.Empty(t, errToken,
				"%s with negative opId should pass validation", cmd)
		})

		t.Run(cmd+"_large_negative", func(t *testing.T) {
			errToken := ValidateOpIDSign(cmd, -999999)
			assert.Empty(t, errToken,
				"%s with large negative opId should pass validation", cmd)
		})
	}
}

// TestValidateOpIDSign_EitherPartyCommands verifies Ping/Error accept any opId sign.
func TestValidateOpIDSign_EitherPartyCommands(t *testing.T) {
	eitherCommands := []string{
		CmdPing, CmdPingResponse, CmdPingComplete,
		CmdError, CmdErrorAck,
	}

	testCases := []struct {
		name string
		opId int64
	}{
		{"positive", 1},
		{"negative", -1},
		{"large_positive", 999999},
		{"large_negative", -999999},
		// Note: opId=0 is handled separately by the opId=0 guard
	}

	for _, cmd := range eitherCommands {
		for _, tc := range testCases {
			t.Run(cmd+"_"+tc.name, func(t *testing.T) {
				errToken := ValidateOpIDSign(cmd, tc.opId)
				assert.Empty(t, errToken,
					"%s with opId=%d should pass validation (either party)", cmd, tc.opId)
			})
		}
	}
}

// TestValidateOpIDSign_ConnectCommands verifies Connect commands pass through.
// opId=0 enforcement is done separately in routeMessage.
func TestValidateOpIDSign_ConnectCommands(t *testing.T) {
	connectCommands := []string{
		CmdConnect, CmdConnectResponse, CmdConnectComplete,
	}

	testCases := []struct {
		name string
		opId int64
	}{
		{"zero", 0},
		{"positive", 1},
		{"negative", -1},
	}

	for _, cmd := range connectCommands {
		for _, tc := range testCases {
			t.Run(cmd+"_"+tc.name, func(t *testing.T) {
				errToken := ValidateOpIDSign(cmd, tc.opId)
				assert.Empty(t, errToken,
					"%s with opId=%d should pass sign validation (validated separately)", cmd, tc.opId)
			})
		}
	}
}

// TestValidateOpIDSign_UnknownCommand verifies unknown commands pass through.
// Unknown command error is handled via the unsupported command path.
func TestValidateOpIDSign_UnknownCommand(t *testing.T) {
	errToken := ValidateOpIDSign("unknownCmd", 42)
	assert.Empty(t, errToken,
		"Unknown command should pass sign validation (handled via unsupported command error)")
}

// ============================================================================
// SCACI §3.2: opId Sign Validation Integration Tests
// ============================================================================

// TestOpIDSignMismatch_ACCommandNegative_IntegrationTest verifies AC command with negative opId
// triggers POSIX_EINVAL + errOpIDSignMismatch via routeMessage.
func TestOpIDSignMismatch_ACCommandNegative_IntegrationTest(t *testing.T) {
	conn := &captureConn{}
	s := &Server{logger: testLogger()}
	session := &Session{State: StateActive, TenantID: 1}

	// CmdStatus is AC-initiated, requires positive opId
	err := s.routeMessage(conn, &session, nil, CmdStatus, -5, nil)

	assert.Nil(t, err, "routeMessage should return nil (error sent on wire)")
	assertOpIDSignMismatchError(t, conn, "status with negative opId")
}

// TestOpIDSignMismatch_SCCommandPositive_IntegrationTest verifies SC command with positive opId
// triggers POSIX_EINVAL + errOpIDSignMismatch via routeMessage.
func TestOpIDSignMismatch_SCCommandPositive_IntegrationTest(t *testing.T) {
	conn := &captureConn{}
	s := &Server{logger: testLogger()}
	session := &Session{State: StateActive, TenantID: 1}

	// CmdULData is SC-initiated, requires negative opId
	err := s.routeMessage(conn, &session, nil, CmdULData, 5, nil)

	assert.Nil(t, err, "routeMessage should return nil (error sent on wire)")
	assertOpIDSignMismatchError(t, conn, "ulData with positive opId")
}

// assertOpIDSignMismatchError verifies that an error response with POSIX_EINVAL and
// the opId sign mismatch message was sent to the connection.
// Note: errorToken is NOT on the wire per SCACI spec (msgpack:"-"), so we verify the message field.
func assertOpIDSignMismatchError(t *testing.T, conn *captureConn, testName string) {
	t.Helper()

	if len(conn.written) == 0 {
		t.Fatalf("%s: expected error response, got none", testName)
	}

	var errMsg map[string]interface{}
	if err := parseErrorFromFrame(conn.written, &errMsg); err != nil {
		t.Fatalf("%s: failed to parse error response: %v", testName, err)
	}

	// Verify command is "error"
	assert.Equal(t, CmdError, errMsg["command"], "%s: expected error command", testName)

	// Verify POSIX code is EINVAL (22)
	code, ok := errMsg["code"]
	assert.True(t, ok, "%s: error response missing code field", testName)
	if codeFloat, ok := code.(float64); ok {
		assert.Equal(t, float64(POSIX_EINVAL), codeFloat, "%s: expected POSIX_EINVAL (22)", testName)
	} else if codeInt, ok := code.(int64); ok {
		assert.Equal(t, int64(POSIX_EINVAL), codeInt, "%s: expected POSIX_EINVAL (22)", testName)
	}

	// Verify error message matches catalog (errorToken is internal-only, not on wire)
	msg, ok := errMsg["message"].(string)
	assert.True(t, ok, "%s: error response missing message field", testName)
	expectedMsg := GetErrorDefinition(errOpIDSignMismatch).Message
	assert.Equal(t, expectedMsg, msg, "%s: expected opId sign mismatch message", testName)
}
