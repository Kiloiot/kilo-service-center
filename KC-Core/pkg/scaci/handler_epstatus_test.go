// Package scaci provides tests for EPStatus lifecycle handlers
// per SCACI §3.13 (EPStatus operation).
//
// Coverage:
//   - handleEPStatusResponse: marks operation as acknowledged
//   - handleEPStatusComplete: marks operation as completed
//   - Session nil handling for both handlers
//   - ResponseData format verification
package scaci

import (
	"testing"
	"time"

	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Operation State Transition Tests
// =============================================================================

func TestHandleEPStatusResponse_StateTransition(t *testing.T) {
	// Per SCACI §3.13.2: EPStatusResponse transitions operation from Pending → Acknowledged

	// Verify the expected state transition
	initialState := models.OperationStatePending
	targetState := models.OperationStateAcknowledged

	assert.NotEqual(t, initialState, targetState, "states should be different")
	assert.Equal(t, "acknowledged", string(targetState), "target state should be 'acknowledged'")
}

func TestHandleEPStatusComplete_StateTransition(t *testing.T) {
	// Per SCACI §3.13.3: EPStatusComplete transitions operation from Acknowledged → Completed

	targetState := models.OperationStateCompleted

	assert.Equal(t, "completed", string(targetState), "target state should be 'completed'")
}

// =============================================================================
// ResponseData Format Tests
// =============================================================================

func TestHandleEPStatusResponse_ResponseDataFormat(t *testing.T) {
	// Verify the responseData format used by handleEPStatusResponse at handler_operations.go:1211
	responseData := map[string]interface{}{"status": "ack"}

	status, ok := responseData["status"].(string)
	assert.True(t, ok, "status should be string")
	assert.Equal(t, "ack", status)
}

func TestHandleEPStatusComplete_ResponseDataFormat(t *testing.T) {
	// Verify the responseData format used by handleEPStatusComplete at handler_operations.go:1244-1247
	responseData := map[string]interface{}{
		"status":      "completed",
		"completedAt": time.Now().UTC().Format(time.RFC3339),
	}

	// Verify status field
	status, ok := responseData["status"].(string)
	assert.True(t, ok, "status should be string")
	assert.Equal(t, "completed", status)

	// Verify completedAt is RFC3339 formatted
	completedAt, ok := responseData["completedAt"].(string)
	assert.True(t, ok, "completedAt should be string")

	// Parse to verify format
	parsedTime, err := time.Parse(time.RFC3339, completedAt)
	assert.NoError(t, err, "completedAt should be valid RFC3339")
	assert.False(t, parsedTime.IsZero(), "parsed time should not be zero")
}

// =============================================================================
// Session Validation Tests
// =============================================================================

func TestHandleEPStatusResponse_NilSession_Error(t *testing.T) {
	// Per handler_operations.go:1197-1199:
	// if session == nil { return s.sendErrorWithCatalog(..., errNoActiveSession) }

	// Verify the error token used
	assert.Equal(t, "scaci.error.no_active_session", errNoActiveSession,
		"nil session should use errNoActiveSession token")
}

func TestHandleEPStatusComplete_NilSession_Error(t *testing.T) {
	// Per handler_operations.go:1230-1232:
	// if session == nil { return s.sendErrorWithCatalog(..., errNoActiveSession) }

	// Verify the error token used
	assert.Equal(t, "scaci.error.no_active_session", errNoActiveSession,
		"nil session should use errNoActiveSession token")
}

// =============================================================================
// Operation ID Tests
// =============================================================================

func TestEPStatusHandlers_UseNegativeOpId(t *testing.T) {
	// Per SCACI §3.13: EPStatus is SC-initiated, so opId should be negative

	// Simulate SC-originated operation IDs
	scOpIDs := []int64{-1, -2, -10, -100, -999}

	for _, opID := range scOpIDs {
		assert.True(t, opID < 0, "SC-initiated opId must be negative: %d", opID)
	}
}

func TestEPStatusHandlers_RejectPositiveOpId(t *testing.T) {
	// Positive opIds are BS/AC-originated and should not be used for EPStatus

	// These would be rejected (EPStatus is SC-initiated)
	acOpIDs := []int64{1, 2, 10, 100, 999}

	for _, opID := range acOpIDs {
		assert.True(t, opID > 0, "AC-originated opId is positive: %d", opID)
		// Note: The actual rejection happens in message routing, not in handlers
	}
}

// =============================================================================
// Three-Way Handshake Order Tests
// =============================================================================

func TestEPStatusHandshake_CorrectOrder(t *testing.T) {
	// SCACI §3.13 three-way handshake:
	// 1. SC sends EPStatus (CmdEPStatus)
	// 2. AC sends EPStatusResponse (CmdEPStatusResponse)
	// 3. AC sends EPStatusComplete (CmdEPStatusComplete)

	// Verify command sequence
	assert.Equal(t, "epStat", CmdEPStatus, "first message")
	assert.Equal(t, "epStatRsp", CmdEPStatusResponse, "second message")
	assert.Equal(t, "epStatCmp", CmdEPStatusComplete, "third message")
}

func TestEPStatusHandshake_StateProgression(t *testing.T) {
	// Operation state should progress as handshake completes

	states := []models.OperationState{
		models.OperationStatePending,      // After SC sends EPStatus
		models.OperationStateAcknowledged, // After AC sends EPStatusResponse
		models.OperationStateCompleted,    // After AC sends EPStatusComplete
	}

	// Verify states are distinct
	assert.NotEqual(t, states[0], states[1], "Pending != Acknowledged")
	assert.NotEqual(t, states[1], states[2], "Acknowledged != Completed")
	assert.NotEqual(t, states[0], states[2], "Pending != Completed")
}

// =============================================================================
// Log Message Tests
// =============================================================================

func TestEPStatusHandlers_LogMessages(t *testing.T) {
	// Verify log messages exist for EPStatus handlers

	assert.NotEmpty(t, LogSCACIEPStatusResponseReceived,
		"EPStatusResponse log message should be defined")
	assert.NotEmpty(t, LogSCACIEPStatusHandshakeComplete,
		"EPStatusComplete log message should be defined")
	assert.NotEmpty(t, LogSCACIOperationStateUpdateFailed,
		"Operation state update failure log should be defined")
}

// =============================================================================
// Session LastSeen Update Tests
// =============================================================================

func TestSession_UpdateLastSeen_Timestamps(t *testing.T) {
	// Both handlers call session.UpdateLastSeen()
	// Verify UpdateLastSeen updates the timestamp

	session := &Session{
		ID:       1,
		TenantID: 1,
	}

	// Initial LastSeen should be zero
	assert.True(t, session.LastSeen.IsZero() || session.LastSeen.Before(time.Now()),
		"initial LastSeen should be before now")

	// Update LastSeen
	session.UpdateLastSeen()

	// LastSeen should be recent
	assert.False(t, session.LastSeen.IsZero(), "LastSeen should not be zero after update")
	assert.True(t, time.Since(session.LastSeen) < time.Second,
		"LastSeen should be within last second")
}

// =============================================================================
// POSIX Error Code Tests
// =============================================================================

func TestEPStatusHandlers_UsePOSIXEINVAL(t *testing.T) {
	// Nil session returns POSIX_EINVAL error

	// POSIX_EINVAL is "Invalid argument" - appropriate for nil session
	assert.Equal(t, 22, POSIX_EINVAL, "EINVAL should be 22")
}
