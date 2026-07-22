// session_lookup_test.go
package bssci

import (
	"errors"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// createTestServerWithSessions creates a test server with pre-registered sessions
// Uses package bssci (not bssci_test) to access unexported fields like ctx, sessions
// StatusService is now mandatory, use NewTestServerWithMemoryStatusService
func createTestServerWithSessions(t *testing.T, sessions []*Session) *Server {
	t.Helper()

	log := logger.NewNop()

	// Use NewTestServerWithMemoryStatusService (StatusService now mandatory)
	server := NewTestServerWithMemoryStatusService(log, nil, nil, 1)

	// Initialize context to prevent nil pointer panics in logging
	server.SetContextForTests(testutil.TestContext())

	// Register test sessions
	for _, session := range sessions {
		server.RegisterSession(session)
	}

	return server
}

// TestFindSessionForEndpointAttachment_Success verifies the happy path
// where a session is found with correct handshake and bidirectional state
func TestFindSessionForEndpointAttachment_Success(t *testing.T) {
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "test-session-ready",
			BaseStationEUI:    TestBsEui04,
			HandshakeComplete: true,
			ClientVersion:     "1.0.0",
			NegotiatedVersion: "1.0.0",
		},
		Bidirectional: true,
		Connected:     time.Now(),
		LastSeen:      time.Now(),
	}

	server := createTestServerWithSessions(t, []*Session{session})

	sessionID, err := server.FindSessionForEndpointAttachment(TestBsEui04)

	require.NoError(t, err)
	assert.Equal(t, "test-session-ready", sessionID)
}

// TestFindSessionForEndpointAttachment_SessionNotFound verifies error handling
// when no session exists for the requested base station EUI
func TestFindSessionForEndpointAttachment_SessionNotFound(t *testing.T) {
	// Create server with no sessions
	server := createTestServerWithSessions(t, []*Session{})

	sessionID, err := server.FindSessionForEndpointAttachment(TestEpEuiUnknown)

	require.Error(t, err)
	assert.Empty(t, sessionID)
	assert.True(t, errors.Is(err, ErrSessionNotFound),
		"Error should wrap ErrSessionNotFound sentinel")
	assert.Contains(t, err.Error(), "session not found")
}

// TestFindSessionForEndpointAttachment_NotBidirectional verifies error handling
// when session exists but doesn't support bidirectional operations
func TestFindSessionForEndpointAttachment_NotBidirectional(t *testing.T) {
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "unidirectional-session",
			BaseStationEUI:    TestBsEui02,
			HandshakeComplete: true,
		},
		Bidirectional: false,
		// Not bidirectional
		Connected: time.Now(),
		LastSeen:  time.Now(),
	}

	server := createTestServerWithSessions(t, []*Session{session})

	sessionID, err := server.FindSessionForEndpointAttachment(TestBsEui02)

	require.Error(t, err)
	assert.Empty(t, sessionID)
	assert.True(t, errors.Is(err, ErrSessionNotBidirectional),
		"Error should wrap ErrSessionNotBidirectional sentinel")
}

// TestFindSessionForEndpointAttachment_HandshakeIncomplete verifies error handling
// when session exists but handshake hasn't completed (BSSCI §3.3)
func TestFindSessionForEndpointAttachment_HandshakeIncomplete(t *testing.T) {
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "incomplete-handshake",
			BaseStationEUI:    TestEpEui01,
			HandshakeComplete: false,
		},
		// Handshake not complete
		Bidirectional: true,
		Connected:     time.Now(),
		LastSeen:      time.Now(),
	}

	server := createTestServerWithSessions(t, []*Session{session})

	sessionID, err := server.FindSessionForEndpointAttachment(TestEpEui01)

	require.Error(t, err)
	assert.Empty(t, sessionID)
	assert.True(t, errors.Is(err, ErrSessionNotReady),
		"Error should wrap ErrSessionNotReady sentinel")
	assert.Contains(t, err.Error(), "handshake")
}

// TestFindSessionForEndpointAttachment_ErrorSentinels verifies all error sentinels
// can be detected using errors.Is() for proper error handling chains
func TestFindSessionForEndpointAttachment_ErrorSentinels(t *testing.T) {
	tests := []struct {
		name              string
		session           *Session
		bsEui             uint64
		expectedSentinel  error
		expectedInMessage string
	}{
		{
			name:              "SessionNotFound sentinel",
			session:           nil, // No session
			bsEui:             0xDEADBEEF,
			expectedSentinel:  ErrSessionNotFound,
			expectedInMessage: "session not found",
		},
		{
			name: "SessionNotReady sentinel",
			session: &Session{
				ProtocolSessionState: ProtocolSessionState{
					ID:                "not-ready",
					BaseStationEUI:    TestBsEuiMulti01,
					HandshakeComplete: false,
				},
				Bidirectional: true,
				Connected:     time.Now(),
				LastSeen:      time.Now(),
			},
			bsEui:             TestBsEuiMulti01,
			expectedSentinel:  ErrSessionNotReady,
			expectedInMessage: "handshake",
		},
		{
			name: "SessionNotBidirectional sentinel",
			session: &Session{
				ProtocolSessionState: ProtocolSessionState{
					ID:                "not-bidi",
					BaseStationEUI:    TestBsEuiMulti02,
					HandshakeComplete: true,
				},
				Bidirectional: false,
				Connected:     time.Now(),
				LastSeen:      time.Now(),
			},
			bsEui:             TestBsEuiMulti02,
			expectedSentinel:  ErrSessionNotBidirectional,
			expectedInMessage: "bidirectional",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sessions []*Session
			if tt.session != nil {
				sessions = []*Session{tt.session}
			}

			server := createTestServerWithSessions(t, sessions)

			_, err := server.FindSessionForEndpointAttachment(tt.bsEui)

			require.Error(t, err)
			assert.True(t, errors.Is(err, tt.expectedSentinel),
				"Error should wrap %v sentinel", tt.expectedSentinel)
			assert.Contains(t, err.Error(), tt.expectedInMessage,
				"Error message should contain '%s'", tt.expectedInMessage)
		})
	}
}

// TestFindSessionForEndpointAttachment_MultipleSessionsPicksCorrect verifies
// that when multiple sessions exist, the correct one is selected by EUI
func TestFindSessionForEndpointAttachment_MultipleSessionsPicksCorrect(t *testing.T) {
	sessions := []*Session{
		{
			ProtocolSessionState: ProtocolSessionState{
				ID:                "session-1",
				BaseStationEUI:    TestBsEuiMulti01,
				HandshakeComplete: true,
			},
			Bidirectional: true,
			Connected:     time.Now(),
			LastSeen:      time.Now(),
		},
		{
			ProtocolSessionState: ProtocolSessionState{
				ID:                "session-2-target",
				BaseStationEUI:    TestBsEuiMulti02,
				HandshakeComplete: true,
			},
			Bidirectional: true,
			Connected:     time.Now(),
			LastSeen:      time.Now(),
		},
		{
			ProtocolSessionState: ProtocolSessionState{
				ID:                "session-3",
				BaseStationEUI:    TestBsEuiMulti03,
				HandshakeComplete: true,
			},
			Bidirectional: true,
			Connected:     time.Now(),
			LastSeen:      time.Now(),
		},
	}

	server := createTestServerWithSessions(t, sessions)

	// Request session for second base station
	sessionID, err := server.FindSessionForEndpointAttachment(TestBsEuiMulti02)

	require.NoError(t, err)
	assert.Equal(t, "session-2-target", sessionID,
		"Should select the correct session by EUI match")
}

// TestSelectBidirectionalSession_DeterministicFallback verifies that when no specific
// base station is requested (targetBsEui == nil), the selection is deterministic:
// lowest EUI wins. This ensures reproducible behavior across Go runtime restarts
// regardless of map iteration order.
// SCACI §3.9.1 Production Readiness fix.
func TestSelectBidirectionalSession_DeterministicFallback(t *testing.T) {
	tenantID := int64(1)

	// Create sessions with different EUIs in non-sorted order
	// The session with lowest EUI (0x0001) should always be selected
	sessions := []*Session{
		{
			ProtocolSessionState: ProtocolSessionState{
				ID:                "session-high",
				BaseStationEUI:    0x0000000000009999,
				HandshakeComplete: true,
				ResolvedTenantID:  tenantID,
			},
			Bidirectional: true,
			Connected:     time.Now(),
			LastSeen:      time.Now(),
		},
		{
			ProtocolSessionState: ProtocolSessionState{
				ID:                "session-lowest",
				BaseStationEUI:    0x0000000000000001,
				HandshakeComplete: true,
				ResolvedTenantID:  tenantID,
			},
			Bidirectional: true,
			Connected:     time.Now(),
			LastSeen:      time.Now(),
		},
		{
			ProtocolSessionState: ProtocolSessionState{
				ID:                "session-mid",
				BaseStationEUI:    0x0000000000005555,
				HandshakeComplete: true,
				ResolvedTenantID:  tenantID,
			},
			Bidirectional: true,
			Connected:     time.Now(),
			LastSeen:      time.Now(),
		},
	}

	server := createTestServerWithSessions(t, sessions)

	// Run multiple times to verify determinism (map iteration order varies)
	for i := 0; i < 10; i++ {
		sessionID, actualBsEui, err := server.SelectBidirectionalSession(tenantID, nil)

		require.NoError(t, err)
		assert.Equal(t, "session-lowest", sessionID,
			"Should consistently select session with lowest EUI (iteration %d)", i)
		assert.Equal(t, uint64(0x0000000000000001), actualBsEui,
			"Should consistently return lowest EUI (iteration %d)", i)
	}
}

// TestSelectBidirectionalSession_DeterministicFallback_TenantIsolation verifies
// that deterministic fallback respects tenant boundaries - only sessions belonging
// to the requesting tenant are considered candidates.
func TestSelectBidirectionalSession_DeterministicFallback_TenantIsolation(t *testing.T) {
	tenant1 := int64(1)
	tenant2 := int64(2)

	sessions := []*Session{
		{
			ProtocolSessionState: ProtocolSessionState{
				ID:                "tenant2-lowest",
				BaseStationEUI:    0x0000000000000001, // Lowest overall, but wrong tenant
				HandshakeComplete: true,
				ResolvedTenantID:  tenant2,
			},
			Bidirectional: true,
			Connected:     time.Now(),
			LastSeen:      time.Now(),
		},
		{
			ProtocolSessionState: ProtocolSessionState{
				ID:                "tenant1-lowest",
				BaseStationEUI:    0x0000000000000100,
				HandshakeComplete: true,
				ResolvedTenantID:  tenant1,
			},
			Bidirectional: true,
			Connected:     time.Now(),
			LastSeen:      time.Now(),
		},
		{
			ProtocolSessionState: ProtocolSessionState{
				ID:                "tenant1-higher",
				BaseStationEUI:    0x0000000000000200,
				HandshakeComplete: true,
				ResolvedTenantID:  tenant1,
			},
			Bidirectional: true,
			Connected:     time.Now(),
			LastSeen:      time.Now(),
		},
	}

	server := createTestServerWithSessions(t, sessions)

	// Request for tenant1 - should ignore tenant2's session
	sessionID, actualBsEui, err := server.SelectBidirectionalSession(tenant1, nil)

	require.NoError(t, err)
	assert.Equal(t, "tenant1-lowest", sessionID,
		"Should select lowest EUI session belonging to tenant1")
	assert.Equal(t, uint64(0x0000000000000100), actualBsEui,
		"Should return tenant1's lowest EUI, not tenant2's")
}

// TestSelectBidirectionalSession_SpecificBsEui verifies that when a specific
// base station is requested, it is used regardless of other sessions.
func TestSelectBidirectionalSession_SpecificBsEui(t *testing.T) {
	tenantID := int64(1)
	targetEui := uint64(0x0000000000009999)

	sessions := []*Session{
		{
			ProtocolSessionState: ProtocolSessionState{
				ID:                "session-lowest",
				BaseStationEUI:    0x0000000000000001,
				HandshakeComplete: true,
				ResolvedTenantID:  tenantID,
			},
			Bidirectional: true,
			Connected:     time.Now(),
			LastSeen:      time.Now(),
		},
		{
			ProtocolSessionState: ProtocolSessionState{
				ID:                "session-target",
				BaseStationEUI:    targetEui,
				HandshakeComplete: true,
				ResolvedTenantID:  tenantID,
			},
			Bidirectional: true,
			Connected:     time.Now(),
			LastSeen:      time.Now(),
		},
	}

	server := createTestServerWithSessions(t, sessions)

	// Request specific BS - should use target, not lowest
	sessionID, actualBsEui, err := server.SelectBidirectionalSession(tenantID, &targetEui)

	require.NoError(t, err)
	assert.Equal(t, "session-target", sessionID,
		"Should select specifically requested session, not lowest")
	assert.Equal(t, targetEui, actualBsEui,
		"Should return requested EUI")
}
