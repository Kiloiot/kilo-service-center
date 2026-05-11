package postgres

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/roaming"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/jmoiron/sqlx"
)

// RoamingRepository implements roaming-specific database operations
type RoamingRepository struct {
	db *sqlx.DB
}

// NewRoamingRepository creates a new roaming repository
func NewRoamingRepository(db *sqlx.DB) *RoamingRepository {
	return &RoamingRepository{db: db}
}

// GetEndpointOwner retrieves the owner tenant ID for an endpoint
// Returns roaming.ErrEndpointNotFound if endpoint doesn't exist
func (r *RoamingRepository) GetEndpointOwner(ctx context.Context, epEui []byte) (int64, error) {
	query := `SELECT owner_tenant_id FROM endpoints WHERE ep_eui = $1`

	var ownerTenantID int64
	err := r.db.QueryRowContext(ctx, query, epEui).Scan(&ownerTenantID)
	if err == sql.ErrNoRows {
		return 0, roaming.ErrEndpointNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("query endpoint owner: %w", err)
	}

	return ownerTenantID, nil
}

// GetEndpointWithOwnership retrieves an endpoint with ownership verification
// Returns endpoint if it exists and belongs to owner OR is being served by servingTenantID
func (r *RoamingRepository) GetEndpointWithOwnership(ctx context.Context, epEui []byte, servingTenantID int64) (*models.EndPoint, error) {
	query := `
		SELECT id, ep_eui, name, description, tenant_id, owner_tenant_id,
		       nwk_sn_key, app_key, crypto_mode, tags, sh_addr,
		       manufacturer, model, carrier_offset,
		       propagated, propagation_count, created_at, updated_at
		FROM endpoints
		WHERE ep_eui = $1
		  AND (owner_tenant_id = $2 OR tenant_id = $2)`

	var ep models.EndPoint
	err := r.db.QueryRowContext(ctx, query, epEui, servingTenantID).Scan(
		&ep.ID, &ep.EUI, &ep.Name, &ep.Description, &ep.TenantID, &ep.OwnerTenantID,
		&ep.NwkSnKey, &ep.AppKey, &ep.CryptoMode, &ep.Tags, &ep.ShAddr,
		&ep.Manufacturer, &ep.Model, &ep.CarrierOffset,
		&ep.Propagated, &ep.PropagationCount, &ep.CreatedAt, &ep.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, roaming.ErrEndpointNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query endpoint with ownership: %w", err)
	}

	return &ep, nil
}

// IsRoamingEnabled checks if roaming is enabled for a tenant
func (r *RoamingRepository) IsRoamingEnabled(ctx context.Context, tenantID int64) (bool, error) {
	query := `SELECT COALESCE(roaming_enabled, false) FROM tenants WHERE id = $1`

	var enabled bool
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&enabled)
	if err == sql.ErrNoRows {
		return false, nil // Tenant doesn't exist or roaming not configured
	}
	if err != nil {
		return false, fmt.Errorf("query roaming enabled: %w", err)
	}

	return enabled, nil
}

// AreTenantsPartners checks if two tenants have a roaming partnership
// Uses LEAST/GREATEST to handle bidirectional partnerships stored as (smaller_id, larger_id)
func (r *RoamingRepository) AreTenantsPartners(ctx context.Context, tenant1, tenant2 int64) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM roaming_partnerships
			WHERE tenant1_id = LEAST($1, $2)
			  AND tenant2_id = GREATEST($1, $2)
			  AND (expires_at IS NULL OR expires_at > NOW())
		)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, tenant1, tenant2).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("query partnership: %w", err)
	}

	return exists, nil
}

// RecordRoamingEvent inserts a roaming event into the audit trail
func (r *RoamingRepository) RecordRoamingEvent(ctx context.Context, event *models.RoamingEvent) error {
	query := `
		INSERT INTO roaming_events (
			event_type, ep_eui, owner_tenant_id, serving_tenant_id,
			from_bs_eui, to_bs_eui, reason, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	// Marshal metadata to JSONB if present
	var metadataJSON []byte
	if event.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("marshal event metadata: %w", err)
		}
	}

	_, err := r.db.ExecContext(ctx, query,
		event.EventType, event.EpEUI, event.OwnerTenantID, event.ServingTenantID,
		event.FromBsEUI, event.ToBsEUI, event.Reason, metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("insert roaming event: %w", err)
	}

	return nil
}

// AddRoamingEndpointToSession adds an endpoint to a session's roaming list
// Appends to the JSONB array and increments the roaming_endpoint_count
func (r *RoamingRepository) AddRoamingEndpointToSession(ctx context.Context, sessionID int64, epEui string, ownerTenantID int64) error {
	// Normalize EUI to uppercase hex for consistency
	epEuiUpper := strings.ToUpper(epEui)

	query := `
		UPDATE basestation_sessions
		SET roaming_endpoints = COALESCE(roaming_endpoints, '[]'::jsonb) ||
		                        jsonb_build_object(
		                            'eui', $2::text,
		                            'ownerTenantId', $3::bigint,
		                            'attachedAt', NOW()
		                        )::jsonb,
		    roaming_endpoint_count = COALESCE(roaming_endpoint_count, 0) + 1
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, sessionID, epEuiUpper, ownerTenantID)
	if err != nil {
		return fmt.Errorf("add roaming endpoint to session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("session %d not found", sessionID)
	}

	return nil
}

// RemoveRoamingEndpointFromSession removes an endpoint from a session's roaming list
// Filters the JSONB array and decrements the roaming_endpoint_count only if element was present
func (r *RoamingRepository) RemoveRoamingEndpointFromSession(ctx context.Context, sessionID int64, epEuiHex string) error {
	// Normalize to uppercase hex for consistent comparison
	epEuiHexUpper := strings.ToUpper(epEuiHex)

	// Use CTE to calculate actual removal count by comparing array lengths
	// COALESCE wrappers ensure empty arrays stay [] (not NULL) and counter logic works correctly
	query := `
		WITH pre_state AS (
		    SELECT
		        id,
		        COALESCE(roaming_endpoints, '[]'::jsonb) as old_array,
		        COALESCE(jsonb_array_length(roaming_endpoints), 0) as pre_len
		    FROM basestation_sessions
		    WHERE id = $1
		),
		filtered AS (
		    SELECT
		        COALESCE(jsonb_agg(elem), '[]'::jsonb) as new_array,
		        COUNT(*)::int as new_len
		    FROM jsonb_array_elements((SELECT old_array FROM pre_state)) elem
		    WHERE UPPER(elem->>'eui') != $2
		)
		UPDATE basestation_sessions
		SET roaming_endpoints = COALESCE((SELECT new_array FROM filtered), '[]'::jsonb),
		    roaming_endpoint_count = GREATEST(
		        COALESCE(roaming_endpoint_count, 0) -
		        ((SELECT pre_len FROM pre_state) - COALESCE((SELECT new_len FROM filtered), 0)),
		        0
		    )
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, sessionID, epEuiHexUpper)
	if err != nil {
		return fmt.Errorf("remove roaming endpoint from session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("session %d not found", sessionID)
	}

	return nil
}

// GetRoamingEndpointsInSession retrieves all roaming endpoints for a session
func (r *RoamingRepository) GetRoamingEndpointsInSession(ctx context.Context, sessionID int64) ([]models.RoamingEndpointInfo, error) {
	query := `SELECT COALESCE(roaming_endpoints, '[]'::jsonb) FROM basestation_sessions WHERE id = $1`

	var roamingJSON []byte
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(&roamingJSON)
	if err == sql.ErrNoRows {
		return []models.RoamingEndpointInfo{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query roaming endpoints: %w", err)
	}

	// Parse JSONB array into slice
	var endpoints []models.RoamingEndpointInfo
	if len(roamingJSON) > 0 && string(roamingJSON) != "null" {
		if err := json.Unmarshal(roamingJSON, &endpoints); err != nil {
			return nil, fmt.Errorf("unmarshal roaming endpoints: %w", err)
		}
	}

	return endpoints, nil
}

// GetRoamingStatistics retrieves roaming statistics for a tenant
func (r *RoamingRepository) GetRoamingStatistics(ctx context.Context, tenantID int64) (*models.RoamingStatistics, error) {
	query := `
		SELECT tenant_id, tenant_name,
		       COALESCE(owned_roaming_out, 0),
		       COALESCE(visiting_roaming_in, 0),
		       COALESCE(roaming_events_24h, 0),
		       roaming_enabled
		FROM roaming_statistics
		WHERE tenant_id = $1`

	var stats models.RoamingStatistics
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&stats.TenantID, &stats.TenantName,
		&stats.OwnedRoamingOut, &stats.VisitingRoamingIn,
		&stats.RoamingEvents24h, &stats.RoamingEnabled,
	)
	if err == sql.ErrNoRows {
		// Return empty stats if tenant not found
		return &models.RoamingStatistics{
			TenantID:       tenantID,
			RoamingEnabled: false,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query roaming statistics: %w", err)
	}

	return &stats, nil
}

// UpdateEndpointRoamingStatus updates an endpoint's serving tenant
// Used when endpoint roams to a new serving base station
func (r *RoamingRepository) UpdateEndpointRoamingStatus(ctx context.Context, epEui []byte, servingTenantID int64) error {
	query := `
		UPDATE endpoints
		SET tenant_id = $2,
		    updated_at = NOW()
		WHERE ep_eui = $1`

	result, err := r.db.ExecContext(ctx, query, epEui, servingTenantID)
	if err != nil {
		return fmt.Errorf("update endpoint roaming status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		epEuiHex := strings.ToUpper(hex.EncodeToString(epEui))
		return fmt.Errorf("endpoint %s not found", epEuiHex)
	}

	return nil
}

// GetTenantRoamingConfig retrieves roaming configuration for a tenant
func (r *RoamingRepository) GetTenantRoamingConfig(ctx context.Context, tenantID int64) (*models.TenantRoamingConfig, error) {
	query := `SELECT COALESCE(roaming_enabled, false) FROM tenants WHERE id = $1`

	var config models.TenantRoamingConfig
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&config.RoamingEnabled)
	if err == sql.ErrNoRows {
		// Return default config if tenant not found
		return &models.TenantRoamingConfig{
			RoamingEnabled: false,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query tenant roaming config: %w", err)
	}

	return &config, nil
}

// UpdateTenantRoamingConfig updates roaming configuration for a tenant
func (r *RoamingRepository) UpdateTenantRoamingConfig(ctx context.Context, tenantID int64, config *models.TenantRoamingConfig) error {
	query := `
		UPDATE tenants
		SET roaming_enabled = $2,
		    updated_at = NOW()
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, tenantID, config.RoamingEnabled)
	if err != nil {
		return fmt.Errorf("update tenant roaming config: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("tenant %d not found", tenantID)
	}

	return nil
}
