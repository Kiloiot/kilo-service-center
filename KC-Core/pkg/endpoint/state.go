// Package endpoint provides shared endpoint state management helpers
// used by both BSSCI and SCACI protocol handlers to maintain consistent
// endpoint lifecycle operations.
package endpoint

import (
	"context"
	"time"
)

// FieldUpdater is the single repository capability DetachEndpoint needs;
// both the full endpoint repository and narrower protocol-server views
// satisfy it.
type FieldUpdater interface {
	UpdateFields(ctx context.Context, tenantID int64, endpointID int64, updates map[string]interface{}) error
}

// DetachEndpoint builds the canonical endpoint detach update map and applies it.
//
// This helper mirrors the logic from KC-Core/pkg/bssci/server.go:1673 and :3573
// without modifying identity fields (nwk_key, app_key, etc.). It updates only
// the attachment state and telemetry fields that indicate the endpoint is no
// longer attached to a base station.
//
// Parameters:
//   - ctx: Context for database operations (should have timeout)
//   - repo: Endpoint repository for database updates
//   - tenantID: Tenant identifier for isolation
//   - endpointID: Database ID of the endpoint to detach
//   - telemetry: Optional map with "sign", "packetCnt", "rxDuration", "eqSnr", "profile" for BSSCI use
//
// Returns:
//   - error: Database error if update fails
//
// The helper is called by:
//   - BSSCI detach handler (KC-Core/pkg/bssci/server.go:1673) with telemetry from detach message
//   - BSSCI detach propagate complete (KC-Core/pkg/bssci/server.go:3573) with nil telemetry
//   - SCACI deregister handler (KC-Core/pkg/scaci/handler_operations.go) with nil telemetry
func DetachEndpoint(
	ctx context.Context,
	repo FieldUpdater,
	tenantID int64,
	endpointID int64,
	telemetry map[string]interface{},
) error {
	// Build canonical detach update map (exact fields from BSSCI)
	// Default to current time, but override with radio rxTime if provided
	detachTime := time.Now().UnixNano()

	updates := map[string]interface{}{
		"last_attached_bs_eui": nil, // Clear BS attachment
		"last_detach_time":     detachTime,
		"propagate_status":     PropagateStatusDetached, // Use shared constant
		"propagated":           false,
		"propagated_at":        nil,
		"ep_status":            EndpointStatusDetached,
	}

	// Add optional telemetry fields (BSSCI-specific, ignored when nil)
	if telemetry != nil {
		// Use radio reception time instead of server processing time
		if rxTime, ok := telemetry["rxTime"].(int64); ok {
			updates["last_detach_time"] = rxTime
		}

		// Store detach signature if provided (4-byte array from BSSCI)
		if sign, ok := telemetry["sign"].([]byte); ok && len(sign) == 4 {
			updates["last_detach_sign"] = sign
		}

		// Store packet counter from detach message (handles both uint32 and int64)
		if packetCnt, ok := telemetry["packetCnt"].(uint32); ok {
			updates["last_detach_packet_cnt"] = packetCnt
		} else if packetCnt, ok := telemetry["packetCnt"].(int64); ok {
			// Safe conversion with bounds check (gosec G115)
			// Silently skip invalid packet count rather than failing entire detach
			if packetCnt >= 0 && packetCnt <= 4294967295 {
				updates["last_detach_packet_cnt"] = uint32(packetCnt)
			}
		}

		// Note: rxDuration, eqSnr, profile handled by separate UpdateRadioMetrics call in BSSCI
		// This helper only manages the detach state fields
	}

	// Apply updates via repository with tenant isolation
	return repo.UpdateFields(ctx, tenantID, endpointID, updates)
}
