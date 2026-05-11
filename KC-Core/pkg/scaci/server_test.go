// Package scaci provides tests for SCACI Send* helper command validation.
//
// Coverage:
//   - SendPingResponse command mismatch detection (§3.4.2)
//   - SendRegisterResponse command mismatch detection (§3.6.2)
//   - SendDeregisterResponse command mismatch detection (§3.7.2)
//   - SendULDataComplete command mismatch detection (§3.8.3)
//   - SendULDataTransmitResponse CommandType mismatch detection (§3.9.2)
//   - SendDLDataQueueResponse CommandType mismatch detection (§3.10.2)
//   - SendDLDataRevokeResponse command mismatch detection (§3.11.2)
//   - SendDLDataResultComplete command mismatch detection (§3.12.3)
package scaci

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// testLogger creates a logger for testing that captures log output.
func testLogger() logger.Logger {
	core, _ := observer.New(zapcore.DebugLevel)
	return logger.FromZap(zap.New(core))
}

// ============================================================================
// Send Helper Command Mismatch Tests (SCACI §2.5)
//
// These tests verify that Send* helpers reject messages with wrong command
// fields, preventing misuse of BaseMessage-only response types.
// ============================================================================

// TestSendPingResponseCommandMismatch verifies §3.4.2 command validation.
func TestSendPingResponseCommandMismatch(t *testing.T) {
	s := &Server{logger: testLogger()}
	conn := &mockConn{}
	session := &Session{}

	msg := &PingResponse{}
	msg.Command = "wrongCommand" // Not CmdPingResponse
	msg.OpId = 1

	err := s.SendPingResponse(conn, session, msg)
	require.Error(t, err, "SendPingResponse should reject wrong command")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// TestSendRegisterResponseCommandMismatch verifies §3.6.2 command validation.
func TestSendRegisterResponseCommandMismatch(t *testing.T) {
	s := &Server{logger: testLogger()}
	conn := &mockConn{}
	session := &Session{}

	msg := &RegisterResponse{}
	msg.Command = "wrongCommand" // Not CmdRegisterResponse
	msg.OpId = 2

	err := s.SendRegisterResponse(conn, session, msg)
	require.Error(t, err, "SendRegisterResponse should reject wrong command")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// TestSendDeregisterResponseCommandMismatch verifies §3.7.2 command validation.
func TestSendDeregisterResponseCommandMismatch(t *testing.T) {
	s := &Server{logger: testLogger()}
	conn := &mockConn{}
	session := &Session{}

	msg := &DeregisterResponse{}
	msg.Command = "wrongCommand" // Not CmdDeregisterResponse
	msg.OpId = 3

	err := s.SendDeregisterResponse(conn, session, msg)
	require.Error(t, err, "SendDeregisterResponse should reject wrong command")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// TestSendULDataCompleteCommandMismatch verifies §3.8.3 command validation.
func TestSendULDataCompleteCommandMismatch(t *testing.T) {
	s := &Server{logger: testLogger()}
	conn := &mockConn{}
	session := &Session{}

	msg := &ULDataComplete{}
	msg.Command = "wrongCommand" // Not CmdULDataComplete
	msg.OpId = 4

	err := s.SendULDataComplete(conn, session, msg)
	require.Error(t, err, "SendULDataComplete should reject wrong command")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// TestSendULDataTransmitResponseCommandMismatch verifies §3.9.2 CommandType validation.
// Note: ULDataTransmitResponse is a mioty type alias using CommandType (not Command).
func TestSendULDataTransmitResponseCommandMismatch(t *testing.T) {
	s := &Server{logger: testLogger()}
	conn := &mockConn{}
	session := &Session{}

	msg := &ULDataTransmitResponse{}
	msg.CommandType = "wrongCommand" // Not CmdULDataTransmitResponse (uses CommandType!)
	msg.OpId = 5

	err := s.SendULDataTransmitResponse(conn, session, msg)
	require.Error(t, err, "SendULDataTransmitResponse should reject wrong CommandType")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// TestSendDLDataQueueResponseCommandMismatch verifies §3.10.2 CommandType validation.
// Note: DLDataQueueResponse is a mioty type alias using CommandType (not Command).
func TestSendDLDataQueueResponseCommandMismatch(t *testing.T) {
	s := &Server{logger: testLogger()}
	conn := &mockConn{}
	session := &Session{}

	msg := &DLDataQueueResponse{}
	msg.CommandType = "wrongCommand" // Not CmdDLDataQueueResponse (uses CommandType!)
	msg.OpId = 6

	err := s.SendDLDataQueueResponse(conn, session, msg)
	require.Error(t, err, "SendDLDataQueueResponse should reject wrong CommandType")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// TestSendDLDataRevokeResponseCommandMismatch verifies §3.11.2 command validation.
func TestSendDLDataRevokeResponseCommandMismatch(t *testing.T) {
	s := &Server{logger: testLogger()}
	conn := &mockConn{}
	session := &Session{}

	msg := &DLDataRevokeResponse{}
	msg.Command = "wrongCommand" // Not CmdDLDataRevokeResponse
	msg.OpId = 7

	err := s.SendDLDataRevokeResponse(conn, session, msg)
	require.Error(t, err, "SendDLDataRevokeResponse should reject wrong command")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// TestSendDLDataResultCompleteCommandMismatch verifies §3.12.3 command validation.
func TestSendDLDataResultCompleteCommandMismatch(t *testing.T) {
	s := &Server{logger: testLogger()}
	conn := &mockConn{}
	session := &Session{}

	msg := &DLDataResultComplete{}
	msg.Command = "wrongCommand" // Not CmdDLDataResultComplete
	msg.OpId = -8                // Negative for SC-originated

	err := s.SendDLDataResultComplete(conn, session, msg)
	require.Error(t, err, "SendDLDataResultComplete should reject wrong command")
	assert.Contains(t, err.Error(), "command mismatch", "Error should mention command mismatch")
}

// ============================================================================
// Constructor Validation Tests (Fail-Fast Pattern per BSSCI server.go:641-644)
//
// These tests verify that NewServer rejects nil dependencies at construction
// time rather than allowing runtime nil pointer panics.
//
// Each test provides valid values for parameters checked BEFORE the one being
// tested, and nil for parameters checked AFTER, to isolate the validation.
// ============================================================================

// TestNewServer_NilCfg verifies constructor rejects nil config.
func TestNewServer_NilCfg(t *testing.T) {
	// cfg is first validation - all other params can be nil
	_, err := NewServer(nil, testLogger(), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.Error(t, err, "NewServer should reject nil cfg")
	assert.Contains(t, err.Error(), "cfg is required")
}

// TestNewServer_NilLogger verifies constructor rejects nil logger.
func TestNewServer_NilLogger(t *testing.T) {
	cfg := &Config{ListenAddr: ":5001"}
	// logger is second validation - provide valid cfg, nil for rest
	_, err := NewServer(cfg, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.Error(t, err, "NewServer should reject nil logger")
	assert.Contains(t, err.Error(), "logger is required")
}

// TestNewServer_NilSessionRepo verifies constructor rejects nil sessionRepo.
func TestNewServer_NilSessionRepo(t *testing.T) {
	cfg := &Config{ListenAddr: ":5001"}
	// sessionRepo is third validation
	_, err := NewServer(cfg, testLogger(), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.Error(t, err, "NewServer should reject nil sessionRepo")
	assert.Contains(t, err.Error(), "sessionRepo is required")
}

// TestNewServer_NilOperationRepo verifies constructor rejects nil operationRepo.
func TestNewServer_NilOperationRepo(t *testing.T) {
	cfg := &Config{ListenAddr: ":5001"}
	mockSessionRepo := &mockSessionRepoStub{}
	// operationRepo is fourth validation
	_, err := NewServer(cfg, testLogger(), mockSessionRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.Error(t, err, "NewServer should reject nil operationRepo")
	assert.Contains(t, err.Error(), "operationRepo is required")
}

// TestNewServer_NilHandshakeSvc verifies constructor rejects nil handshakeSvc.
func TestNewServer_NilHandshakeSvc(t *testing.T) {
	cfg := &Config{ListenAddr: ":5001"}
	mockSessionRepo := &mockSessionRepoStub{}
	mockOpRepo := &mockOperationRepoStub{}
	_, err := NewServer(cfg, testLogger(), mockSessionRepo, mockOpRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.Error(t, err, "NewServer should reject nil handshakeSvc")
	assert.Contains(t, err.Error(), "handshakeSvc is required")
}

// TestNewServer_NilEndpointSvc verifies constructor rejects nil endpointSvc.
func TestNewServer_NilEndpointSvc(t *testing.T) {
	cfg := &Config{ListenAddr: ":5001"}
	mockSessionRepo := &mockSessionRepoStub{}
	mockOpRepo := &mockOperationRepoStub{}
	mockHandshake := &MockHandshakeService{}
	_, err := NewServer(cfg, testLogger(), mockSessionRepo, mockOpRepo, mockHandshake, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.Error(t, err, "NewServer should reject nil endpointSvc")
	assert.Contains(t, err.Error(), "endpointSvc is required")
}

// TestNewServer_NilULSvc verifies constructor rejects nil ulSvc.
func TestNewServer_NilULSvc(t *testing.T) {
	cfg := &Config{ListenAddr: ":5001"}
	mockSessionRepo := &mockSessionRepoStub{}
	mockOpRepo := &mockOperationRepoStub{}
	mockHandshake := &MockHandshakeService{}
	mockEndpoint := &MockEndpointService{}
	_, err := NewServer(cfg, testLogger(), mockSessionRepo, mockOpRepo, mockHandshake, mockEndpoint, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.Error(t, err, "NewServer should reject nil ulSvc")
	assert.Contains(t, err.Error(), "ulSvc is required")
}

// TestNewServer_NilDLSvc verifies constructor rejects nil dlSvc (§3.8 revocation).
func TestNewServer_NilDLSvc(t *testing.T) {
	cfg := &Config{ListenAddr: ":5001"}
	mockSessionRepo := &mockSessionRepoStub{}
	mockOpRepo := &mockOperationRepoStub{}
	mockHandshake := &MockHandshakeService{}
	mockEndpoint := &MockEndpointService{}
	mockUL := &MockULService{}
	_, err := NewServer(cfg, testLogger(), mockSessionRepo, mockOpRepo, mockHandshake, mockEndpoint, mockUL, nil, nil, nil, nil, nil, nil, nil, nil)
	require.Error(t, err, "NewServer should reject nil dlSvc")
	assert.Contains(t, err.Error(), "dlSvc is required")
}

// TestNewServer_NilStatusSvc verifies constructor rejects nil statusSvc.
func TestNewServer_NilStatusSvc(t *testing.T) {
	cfg := &Config{ListenAddr: ":5001"}
	mockSessionRepo := &mockSessionRepoStub{}
	mockOpRepo := &mockOperationRepoStub{}
	mockHandshake := &MockHandshakeService{}
	mockEndpoint := &MockEndpointService{}
	mockUL := &MockULService{}
	mockDL := &MockDLService{}
	_, err := NewServer(cfg, testLogger(), mockSessionRepo, mockOpRepo, mockHandshake, mockEndpoint, mockUL, mockDL, nil, nil, nil, nil, nil, nil, nil)
	require.Error(t, err, "NewServer should reject nil statusSvc")
	assert.Contains(t, err.Error(), "statusSvc is required")
}
