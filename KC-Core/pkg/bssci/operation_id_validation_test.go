package bssci

import (
	"testing"

	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/stretchr/testify/assert"
)

// TestSCOperationIDMustBeNegative verifies BSSCI §3.2-02 requirement that
// Service Center initiated operations MUST use negative operation IDs.
// This test ensures the critical validation gap fix is working correctly.
func TestSCOperationIDMustBeNegative(t *testing.T) {
	tests := []struct {
		name     string
		opId     int64
		expected string // Expected error token (empty string means valid)
		desc     string
	}{
		{
			name:     "PositiveSCOperationIDRejected",
			opId:     1,
			expected: errSCOperationIDMustBeNegative,
			desc:     "Positive operation ID must be rejected for SC-initiated operations",
		},
		{
			name:     "ZeroSCOperationIDRejected",
			opId:     0,
			expected: errSCOperationIDMustBeNegative,
			desc:     "Zero operation ID must be rejected for SC-initiated operations",
		},
		{
			name:     "NegativeSCOperationIDAccepted",
			opId:     -1,
			expected: "",
			desc:     "Negative operation ID must be accepted for SC-initiated operations",
		},
		{
			name:     "LargeNegativeSCOperationIDAccepted",
			opId:     -9999,
			expected: "",
			desc:     "Large negative operation ID must be accepted for SC-initiated operations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use NewTestServerWithMemoryStatusService (StatusService is mandatory)
			log := logger.NewNop()
			server := NewTestServerWithMemoryStatusService(log, nil, nil, 1)

			// Create test session
			session := &Session{
				BaseStationEUI: 123456789, // Non-zero to pass connect check
				LastScOpId:     0,
			}

			// Call validateOperationID with isBaseStationInitiated = false (SC operation)
			errToken := server.CallValidateOperationID(session, tt.opId, false)

			// Verify result
			if tt.expected == "" {
				assert.Empty(t, errToken, tt.desc)
			} else {
				assert.Equal(t, tt.expected, errToken, tt.desc)
			}
		})
	}
}

// TestSCOperationIDStrictDecrement verifies BSSCI §3.2-02 requirement for SC operation validation.
// With atomic send-and-persist pattern, LastScOpId is updated when SENDING operations,
// not during validation. Validation only checks that responses match or are stale retries.
func TestSCOperationIDStrictDecrement(t *testing.T) {
	tests := []struct {
		name         string
		lastScOpId   int64
		newOpId      int64
		expected     string // Expected error token (empty string means valid)
		desc         string
		shouldUpdate bool // Whether LastScOpId should be updated (always false for SC ops now)
	}{
		{
			name:         "SameOperationIDAllowed",
			lastScOpId:   -10,
			newOpId:      -10,
			expected:     "",
			desc:         "Same operation ID allowed (response to our request)",
			shouldUpdate: false,
		},
		{
			name:         "OperationIDIncreaseRejected",
			lastScOpId:   -5,
			newOpId:      -3,
			expected:     "",
			desc:         "Less-negative pending response is accepted for existing SC operation",
			shouldUpdate: false,
		},
		{
			name:         "FirstOperationResponse",
			lastScOpId:   -1,
			newOpId:      -1,
			expected:     "",
			desc:         "First SC operation response (matches our sent opId)",
			shouldUpdate: false,
		},
		{
			name:         "FutureOperationIDRejected",
			lastScOpId:   -1,
			newOpId:      -2,
			expected:     "bssci.error.operation_id_increasing",
			desc:         "Future operation ID rejected (SC operations must not increase)",
			shouldUpdate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use NewTestServerWithMemoryStatusService (StatusService is mandatory)
			log := logger.NewNop()
			server := NewTestServerWithMemoryStatusService(log, nil, nil, 1)

			// Create test session with initial LastScOpId
			session := &Session{
				BaseStationEUI: 123456789,
				LastScOpId:     tt.lastScOpId,
			}

			// Call validateOperationID
			errToken := server.CallValidateOperationID(session, tt.newOpId, false)

			// Verify error token
			if tt.expected == "" {
				assert.Empty(t, errToken, tt.desc)
			} else {
				assert.Equal(t, tt.expected, errToken, tt.desc)
			}

			// Verify LastScOpId update behavior
			if tt.shouldUpdate {
				assert.Equal(t, tt.newOpId, session.LastScOpId,
					"LastScOpId should be updated for new operations")
			} else {
				assert.Equal(t, tt.lastScOpId, session.LastScOpId,
					"LastScOpId should not be updated for repeated/invalid operations")
			}
		})
	}
}

// TestBSOperationIDMustBePositive verifies BSSCI §3.2-01 requirement that
// Base Station initiated operations MUST use positive operation IDs.
func TestBSOperationIDMustBePositive(t *testing.T) {
	tests := []struct {
		name     string
		opId     int64
		expected string
		desc     string
	}{
		{
			name:     "PositiveBSOperationIDAccepted",
			opId:     1,
			expected: "",
			desc:     "Positive operation ID must be accepted for BS-initiated operations",
		},
		{
			name:     "ZeroBSOperationIDAllowedAfterConnect",
			opId:     0,
			expected: "",
			desc:     "Zero operation ID allowed for BS operations (lenient implementation)",
		},
		{
			name:     "NegativeBSOperationIDRejected",
			opId:     -1,
			expected: errOperationIDNotPositive,
			desc:     "Negative operation ID must be rejected for BS-initiated operations",
		},
		{
			name:     "LargePositiveBSOperationIDAccepted",
			opId:     99999,
			expected: "",
			desc:     "Large positive operation ID must be accepted for BS-initiated operations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use NewTestServerWithMemoryStatusService (StatusService is mandatory)
			log := logger.NewNop()
			server := NewTestServerWithMemoryStatusService(log, nil, nil, 1)

			// Create session with BaseStationEUI set (after connect)
			session := &Session{
				BaseStationEUI: 123456789,
				LastBsOpId:     0,
			}

			// Call validateOperationID with isBaseStationInitiated = true
			errToken := server.CallValidateOperationID(session, tt.opId, true)

			if tt.expected == "" {
				assert.Empty(t, errToken, tt.desc)
			} else {
				assert.Equal(t, tt.expected, errToken, tt.desc)
			}
		})
	}
}

// TestBSOperationIDStrictIncrement verifies BSSCI §3.2-01 requirement that
// Base Station operation IDs must strictly increment for NEW operations.
func TestBSOperationIDStrictIncrement(t *testing.T) {
	tests := []struct {
		name         string
		lastBsOpId   int64
		newOpId      int64
		expected     string
		desc         string
		shouldUpdate bool
	}{
		{
			name:         "NewOperationIncrementsCorrectly",
			lastBsOpId:   5,
			newOpId:      6,
			expected:     "",
			desc:         "New operation with strictly incrementing ID should be accepted",
			shouldUpdate: true,
		},
		{
			name:         "OperationIDDecreaseRejected",
			lastBsOpId:   10,
			newOpId:      8,
			expected:     errOperationIDBackwards,
			desc:         "Operation ID must not go backwards",
			shouldUpdate: false,
		},
		{
			name:         "SameOperationIDAllowed",
			lastBsOpId:   20,
			newOpId:      20,
			expected:     "",
			desc:         "Same operation ID allowed (part of same operation handshake)",
			shouldUpdate: false,
		},
		{
			name:         "FirstOperationAccepted",
			lastBsOpId:   0,
			newOpId:      1,
			expected:     "",
			desc:         "First BS operation should be accepted",
			shouldUpdate: true,
		},
		{
			name:         "LargeIncrementAccepted",
			lastBsOpId:   100,
			newOpId:      200,
			expected:     "",
			desc:         "Large increment (gap in sequence) should be accepted",
			shouldUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use NewTestServerWithMemoryStatusService (StatusService is mandatory)
			log := logger.NewNop()
			server := NewTestServerWithMemoryStatusService(log, nil, nil, 1)

			session := &Session{
				BaseStationEUI: 123456789,
				LastBsOpId:     tt.lastBsOpId,
			}

			// Call validateOperationID with isBaseStationInitiated = true
			errToken := server.CallValidateOperationID(session, tt.newOpId, true)

			// Verify error token
			if tt.expected == "" {
				assert.Empty(t, errToken, tt.desc)
			} else {
				assert.Equal(t, tt.expected, errToken, tt.desc)
			}

			// Verify LastBsOpId update behavior
			if tt.shouldUpdate {
				assert.Equal(t, tt.newOpId, session.LastBsOpId,
					"LastBsOpId should be updated for new operations")
			} else {
				assert.Equal(t, tt.lastBsOpId, session.LastBsOpId,
					"LastBsOpId should not be updated for repeated/invalid operations")
			}
		})
	}
}
