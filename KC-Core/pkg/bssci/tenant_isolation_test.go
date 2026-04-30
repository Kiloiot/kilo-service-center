package bssci

import "testing"

// TestResolvedTenantUsesServerTenant validates that resolvedTenant() helper
// uses the server's configured tenantID (not defaultTenantID) as fallback.
// BSSCI Section 3: All session operations must respect tenant boundaries.
func TestResolvedTenantUsesServerTenant(t *testing.T) {
	tests := []struct {
		name           string
		session        *Session
		serverTenantID int64
		expected       int64
	}{
		{
			name:           "nil_session_uses_server_tenant",
			session:        nil,
			serverTenantID: 1,
			expected:       1,
		},
		{
			name:           "unresolved_session_uses_server_tenant",
			session:        &Session{ResolvedTenantID: 0},
			serverTenantID: 1,
			expected:       1,
		},
		{
			name:           "resolved_session_uses_own_tenant",
			session:        &Session{ResolvedTenantID: 42},
			serverTenantID: 1,
			expected:       42,
		},
		{
			name:           "cert_resolved_tenant_overrides_server_default",
			session:        &Session{ResolvedTenantID: 99},
			serverTenantID: 1,
			expected:       99,
		},
		{
			name:           "fallback_when_tenant_not_resolved",
			session:        &Session{ResolvedTenantID: 0, BaseStationEUI: 123456},
			serverTenantID: 5,
			expected:       5,
		},
		{
			name:           "different_server_tenant_fallback",
			session:        nil,
			serverTenantID: 999,
			expected:       999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolvedTenant(tt.session, tt.serverTenantID)
			if result != tt.expected {
				t.Errorf("resolvedTenant(%v, %d) = %d, want %d",
					tt.session, tt.serverTenantID, result, tt.expected)
			}
		})
	}
}

// TestResolvedTenantServerTenantNotDefaultTenant ensures that when
// serverTenantID != defaultTenantID (e.g., community mode or multi-tenant),
// the fallback uses serverTenantID, not defaultTenantID.
// This test documents the fix for the tenant isolation regression.
func TestResolvedTenantServerTenantNotDefaultTenant(t *testing.T) {
	tests := []struct {
		name             string
		session          *Session
		serverTenantID   int64
		defaultTenantID  int64 // Would be wrong fallback
		expectedTenantID int64
		description      string
	}{
		{
			name:             "community_mode_tenant_mismatch",
			session:          &Session{ResolvedTenantID: 0},
			serverTenantID:   1,
			defaultTenantID:  100, // Wrong fallback
			expectedTenantID: 1,   // Should use serverTenantID
			description:      "Community mode: server.tenantID=1, server.defaultTenantID=100",
		},
		{
			name:             "multi_tenant_deployment",
			session:          nil,
			serverTenantID:   5,
			defaultTenantID:  1,
			expectedTenantID: 5,
			description:      "Multi-tenant: each server instance has dedicated tenantID",
		},
		{
			name:             "cert_resolution_pending",
			session:          &Session{ResolvedTenantID: 0, BaseStationEUI: TestEpEui01},
			serverTenantID:   42,
			defaultTenantID:  1,
			expectedTenantID: 42,
			description:      "Before cert resolution completes, use serverTenantID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolvedTenant(tt.session, tt.serverTenantID)

			// Verify we get the server tenant, NOT the default tenant
			if result != tt.expectedTenantID {
				t.Errorf("%s: resolvedTenant() = %d, want %d (NOT defaultTenantID=%d)",
					tt.description, result, tt.expectedTenantID, tt.defaultTenantID)
			}

			// Explicitly check we're NOT using defaultTenantID as fallback
			if tt.session == nil || tt.session.ResolvedTenantID == 0 {
				if result == tt.defaultTenantID && tt.defaultTenantID != tt.serverTenantID {
					t.Errorf("%s: REGRESSION! Used defaultTenantID=%d instead of serverTenantID=%d",
						tt.description, tt.defaultTenantID, tt.serverTenantID)
				}
			}
		})
	}
}
