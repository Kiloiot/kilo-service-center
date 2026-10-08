package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
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
// Returns storage.ErrNotFound if the endpoint does not exist.
func (r *RoamingRepository) GetEndpointOwner(ctx context.Context, epEui []byte) (int64, error) {
	query := `SELECT owner_tenant_id FROM endpoints WHERE ep_eui = $1`

	var ownerTenantID int64
	err := r.db.QueryRowContext(ctx, query, epEui).Scan(&ownerTenantID)
	if err == sql.ErrNoRows {
		return 0, storage.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapQueryEndpointOwner, err)
	}

	return ownerTenantID, nil
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
		return false, fmt.Errorf("%s: %w", errWrapQueryRoamingEnabled, err)
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
		return false, fmt.Errorf("%s: %w", errWrapQueryPartnership, err)
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
			return fmt.Errorf("%s: %w", errWrapMarshalEventMetadata, err)
		}
	}

	_, err := r.db.ExecContext(
		ctx, query,
		event.EventType, event.EpEUI, event.OwnerTenantID, event.ServingTenantID,
		event.FromBsEUI, event.ToBsEUI, event.Reason, metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapInsertRoamingEvent, err)
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
		return fmt.Errorf("%s: %w", errWrapAddRoamingEndpointSession, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapCheckRowsAffected, err)
	}
	if rows == 0 {
		return fmt.Errorf(errFmtSessionNotFound, sessionID)
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
		return fmt.Errorf("%s: %w", errWrapRemoveRoamingEndpointFromSession, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapCheckRowsAffected, err)
	}
	if rows == 0 {
		return fmt.Errorf(errFmtSessionNotFound, sessionID)
	}

	return nil
}
