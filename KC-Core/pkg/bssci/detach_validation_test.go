package bssci

import (
	"context"
	"testing"

	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// mockDetachValidator implements DetachSignatureValidator for testing
type mockDetachValidator struct {
	returnResult *DetachValidationResult
	returnErr    error
}

func (m *mockDetachValidator) ValidateDetachSignature(_ context.Context, _ uint64, _ []byte) (*DetachValidationResult, error) {
	return m.returnResult, m.returnErr
}

// TestServer_DetachComplete_InvalidSignature_LogsTenantMetadata verifies server logs tenant metadata on signature failure
func TestServer_DetachComplete_InvalidSignature_LogsTenantMetadata(t *testing.T) {
	t.Parallel()

	const (
		opID     = int64(999)
		epEui    = TestEpEui01 // Use standard test EUI
		tenantID = int64(123)
		ownerID  = int64(456)
	)

	// Create mock validator that returns ErrDetachSignatureInvalid with metadata
	mockValidator := &mockDetachValidator{
		returnResult: &DetachValidationResult{
			Valid:            false,
			TenantID:         tenantID,
			OwnerTenantID:    ownerID,
			ValidationStatus: ValidationStatusInvalidSignature,
		},
		returnErr: ErrDetachSignatureInvalid,
	}

	// Use existing test environment helper with NO endpoint (unknown endpoint scenario)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: true,
		MessageEncoding:                  EncodingJSON,
	}, nil) // nil endpoint = unknown endpoint

	// Replace logger with observed logger to capture log output
	observedCore, observedLogs := observer.New(zap.WarnLevel)
	env.server.logger = logger.FromZap(zap.New(observedCore))

	// Install mock validator
	env.server.detachValidator = mockValidator

	// Build detach payload and message
	detachPayload := buildDetachPayload(epEui)
	detachMsg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: detachPayload}

	// Call handleDetach - validation fails, sends error directly (no detRsp, no three-way handshake)
	err := env.server.handleDetach(env.server, env.session, detachMsg, detachPayload)

	// Assert: handleDetach sends error but doesn't return error (BSSCI pattern: sendError returns nil)
	require.NoError(t, err, "handleDetach should send error but not return error")

	// Assert: Error message sent to basestation
	require.True(t, env.conn.SeenCommand("error"), "Server should send error command to basestation")

	// Verify error response contains correct POSIX code and message
	require.True(t, len(env.conn.SentMessages) > 0, "At least one message should be sent")
	errorMsg := env.conn.SentMessages[len(env.conn.SentMessages)-1]
	require.NotNil(t, errorMsg, "Error message should be sent")

	errorCode, _ := errorMsg["code"].(float64) // JSON numbers are float64
	assert.Equal(t, float64(POSIX_EACCES), errorCode, "Error code should be POSIX_EACCES")

	errorMessage, _ := errorMsg["message"].(string)
	expectedMsg := ResolveErrorMessage(errDetachSignatureInvalid)
	assert.Equal(t, expectedMsg, errorMessage, "Error message should match catalog token")

	// Assert: Log contains tenant metadata
	allLogs := observedLogs.All()
	foundLog := false
	for _, logEntry := range allLogs {
		if logEntry.Message == "Unknown endpoint detach signature invalid" {
			foundLog = true

			// Verify log context fields
			fields := logEntry.ContextMap()
			assert.Equal(t, epEui, fields["epEui"], "Log should contain epEui")
			assert.Equal(t, int64(tenantID), fields["tenant_id"], "Log should contain tenant_id")
			assert.Equal(t, int64(ownerID), fields["owner_tenant_id"], "Log should contain owner_tenant_id")
			assert.Equal(t, ValidationStatusInvalidSignature, fields["validation_status"], "Log should contain validation_status")
			break
		}
	}

	assert.True(t, foundLog, "Log message 'Unknown endpoint detach signature invalid' should be present")
}

// TestServer_DetachComplete_EndpointNotFound_SendsEnoent verifies 404 handling for unknown endpoints
func TestServer_DetachComplete_EndpointNotFound_SendsEnoent(t *testing.T) {
	t.Parallel()

	const (
		opID  = int64(888)
		epEui = TestEpEui02 // Use standard test EUI that doesn't overflow int64
	)

	// Mock validator returns ErrDetachValidationEndpointNotFound
	mockValidator := &mockDetachValidator{
		returnResult: nil,
		returnErr:    ErrDetachValidationEndpointNotFound,
	}

	// Use existing test environment helper with NO endpoint (unknown endpoint scenario)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: true,
		MessageEncoding:                  EncodingJSON,
	}, nil) // nil endpoint = unknown endpoint

	// Install mock validator
	env.server.detachValidator = mockValidator

	// Build detach payload and message
	detachPayload := buildDetachPayload(epEui)
	detachMsg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: detachPayload}

	// Call handleDetach - validation fails with 404, sends error directly (no detRsp, no three-way handshake)
	err := env.server.handleDetach(env.server, env.session, detachMsg, detachPayload)

	// Assert: handleDetach sends error but doesn't return error (BSSCI pattern: sendError returns nil)
	require.NoError(t, err, "handleDetach should send error but not return error")

	// Assert: Error sent with POSIX_ENOENT
	require.True(t, env.conn.SeenCommand("error"), "Server should send error command")
	require.True(t, len(env.conn.SentMessages) > 0, "At least one message should be sent")
	errorMsg := env.conn.SentMessages[len(env.conn.SentMessages)-1]
	errorCode, _ := errorMsg["code"].(float64)
	assert.Equal(t, float64(POSIX_ENOENT), errorCode, "Error code should be POSIX_ENOENT for not found")
}
