// Package bssciservices - Roaming adapter for BSSCI handlers.
package bssciservices

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/roaming"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	log "github.com/sirupsen/logrus"
)

// RoamingAdapter implements roaming interfaces using the storage layer
type RoamingAdapter struct {
	storage storage.Storage
}

// NewRoamingAdapter creates a new roaming adapter
func NewRoamingAdapter(storage storage.Storage) *RoamingAdapter {
	return &RoamingAdapter{
		storage: storage,
	}
}

// GetEndpointOwner returns the owner tenant ID for an endpoint
func (r *RoamingAdapter) GetEndpointOwner(ctx context.Context, epEui []byte) (ownerTenantID int64, err error) {
	return r.storage.GetEndpointOwner(ctx, epEui)
}

// GetEndpointWithOwnership returns full endpoint data including ownership
func (r *RoamingAdapter) GetEndpointWithOwnership(ctx context.Context, epEui []byte, servingTenantID int64) (*models.EndPoint, error) {
	return r.storage.GetEndpointWithOwnership(ctx, epEui, servingTenantID)
}

// IsRoamingEnabled checks if roaming is enabled for a tenant
func (r *RoamingAdapter) IsRoamingEnabled(ctx context.Context, tenantID int64) (bool, error) {
	return r.storage.IsRoamingEnabled(ctx, tenantID)
}

// AreTenantsPartners checks if two tenants have a roaming agreement
func (r *RoamingAdapter) AreTenantsPartners(ctx context.Context, tenant1, tenant2 int64) (bool, error) {
	return r.storage.AreTenantsPartners(ctx, tenant1, tenant2)
}

// RecordRoamingEvent records a roaming event to the audit trail
func (r *RoamingAdapter) RecordRoamingEvent(ctx context.Context, event *models.RoamingEvent) error {
	return r.storage.RecordRoamingEvent(ctx, event)
}

// RoamingService provides roaming detection and management for BSSCI
type RoamingService struct {
	detector *roaming.Detector
	storage  storage.Storage
	enabled  bool
}

// NewRoamingService creates a new roaming service
func NewRoamingService(storage storage.Storage, enabled bool) *RoamingService {
	adapter := NewRoamingAdapter(storage)

	config := roaming.DefaultDetectorConfig()
	config.EnableAuditTrail = enabled
	config.EnableMetrics = enabled

	detector := roaming.NewDetector(config, adapter, adapter)

	return &RoamingService{
		detector: detector,
		storage:  storage,
		enabled:  enabled,
	}
}

// DetectAndValidateRoaming checks if an endpoint is roaming and validates the operation
func (rs *RoamingService) DetectAndValidateRoaming(ctx context.Context, epEui []byte, servingTenantID int64) (isRoaming bool, ownerTenantID int64, err error) {
	if !rs.enabled {
		// If roaming is disabled, owner always equals serving tenant
		return false, servingTenantID, nil
	}

	// Detect roaming status
	isRoaming, ownerTenantID, err = rs.detector.DetectRoaming(ctx, epEui, servingTenantID)
	if err != nil {
		return false, 0, fmt.Errorf("roaming detection failed: %w", err)
	}

	// If roaming, validate it's allowed
	if isRoaming {
		if err := rs.detector.ValidateRoamingAllowed(ctx, ownerTenantID, servingTenantID); err != nil {
			log.Warn("Roaming not allowed",
				"epEui", hex.EncodeToString(epEui),
				"ownerTenant", ownerTenantID,
				"servingTenant", servingTenantID,
				"error", err)
			return false, 0, fmt.Errorf("roaming not allowed: %w", err)
		}
	}

	return isRoaming, ownerTenantID, nil
}

// RecordAttach records an endpoint attach with roaming awareness
func (rs *RoamingService) RecordAttach(ctx context.Context, epEui []byte, bsEui []byte, servingTenantID int64) error {
	if !rs.enabled {
		return nil
	}

	// Get owner tenant ID
	ownerTenantID, err := rs.storage.GetEndpointOwner(ctx, epEui)
	if err != nil {
		// If endpoint not found, it might be a new endpoint
		// In this case, owner equals serving tenant
		ownerTenantID = servingTenantID
	}

	// Record the attach event
	return rs.detector.RecordAttachEvent(ctx, epEui, ownerTenantID, servingTenantID, bsEui)
}

// RecordDetach records an endpoint detach with roaming awareness
func (rs *RoamingService) RecordDetach(ctx context.Context, epEui []byte, bsEui []byte, servingTenantID int64) error {
	if !rs.enabled {
		return nil
	}

	// Get owner tenant ID
	ownerTenantID, err := rs.storage.GetEndpointOwner(ctx, epEui)
	if err != nil {
		// If endpoint not found, use serving tenant
		ownerTenantID = servingTenantID
	}

	// Record the detach event
	return rs.detector.RecordDetachEvent(ctx, epEui, ownerTenantID, servingTenantID, bsEui)
}

// UpdateSessionRoaming updates base station session with roaming endpoint info
func (rs *RoamingService) UpdateSessionRoaming(ctx context.Context, sessionID int64, epEui []byte, isAttach bool, servingTenantID int64) error {
	if !rs.enabled {
		return nil
	}

	epEuiHex := hex.EncodeToString(epEui)

	// Get owner tenant ID
	ownerTenantID, err := rs.storage.GetEndpointOwner(ctx, epEui)
	if err != nil {
		// If endpoint not found, use serving tenant
		ownerTenantID = servingTenantID
	}

	// Only track if actually roaming
	if ownerTenantID == servingTenantID {
		return nil
	}

	if isAttach {
		// Add to roaming endpoints list
		return rs.storage.AddRoamingEndpointToSession(ctx, sessionID, epEuiHex, ownerTenantID)
	}
	// Remove from roaming endpoints list
	return rs.storage.RemoveRoamingEndpointFromSession(ctx, sessionID, epEuiHex)
}

// GetMetrics returns roaming metrics
func (rs *RoamingService) GetMetrics() *roaming.MetricsSnapshot {
	if !rs.enabled || rs.detector == nil {
		return nil
	}

	metrics := rs.detector.GetMetrics()
	if metrics == nil {
		return nil
	}

	return metrics.GetSnapshot()
}

// InvalidateCache invalidates cache entry for an endpoint
func (rs *RoamingService) InvalidateCache(epEui []byte) {
	if rs.enabled && rs.detector != nil {
		rs.detector.InvalidateCache(epEui)
	}
}

// ClearCache clears the entire roaming cache
func (rs *RoamingService) ClearCache() {
	if rs.enabled && rs.detector != nil {
		rs.detector.ClearCache()
	}
}
