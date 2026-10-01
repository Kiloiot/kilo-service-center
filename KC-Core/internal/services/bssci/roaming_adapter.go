// Package bssciservices - Roaming adapter for BSSCI handlers.
package bssciservices

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/roaming"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// RoamingDetector decides whether an endpoint is roaming, whether the roam is
// permitted, and records the resulting attach/detach events.
type RoamingDetector interface {
	DetectRoaming(ctx context.Context, epEui []byte, servingTenantID int64) (isRoaming bool, ownerTenantID int64, err error)
	ValidateRoamingAllowed(ctx context.Context, ownerTenantID, servingTenantID int64) error
	RecordAttachEvent(ctx context.Context, epEui []byte, ownerTenantID, servingTenantID int64, bsEui []byte) error
	RecordDetachEvent(ctx context.Context, epEui []byte, ownerTenantID, servingTenantID int64, bsEui []byte) error
}

// RoamingSessionStore resolves endpoint ownership and maintains the roaming
// endpoint list attached to a base station session.
type RoamingSessionStore interface {
	GetEndpointOwner(ctx context.Context, epEui []byte) (ownerTenantID int64, err error)
	AddRoamingEndpointToSession(ctx context.Context, sessionID int64, epEui string, ownerTenantID int64) error
	RemoveRoamingEndpointFromSession(ctx context.Context, sessionID int64, epEui string) error
}

// RoamingService provides roaming detection and management for BSSCI.
// It is only constructed when roaming is enabled; a nil RoamingService on
// the server means endpoints are served by their connecting tenant.
type RoamingService struct {
	detector RoamingDetector
	sessions RoamingSessionStore
	logger   logger.Logger
}

// NewRoamingService creates a new roaming service.
func NewRoamingService(detector RoamingDetector, sessions RoamingSessionStore, log logger.Logger) *RoamingService {
	return &RoamingService{
		detector: detector,
		sessions: sessions,
		logger:   log,
	}
}

// ownershipResolverAdapter translates the storage layer's not-found sentinel
// into the roaming domain's ErrEndpointNotFound so the detector keeps its
// spec-defined behavior even though the persistence layer no longer depends on
// the roaming package.
type ownershipResolverAdapter struct {
	inner roaming.EndpointOwnershipResolver
}

func (a ownershipResolverAdapter) GetEndpointOwner(ctx context.Context, epEui []byte) (int64, error) {
	ownerTenantID, err := a.inner.GetEndpointOwner(ctx, epEui)
	if errors.Is(err, storage.ErrNotFound) {
		return 0, roaming.ErrEndpointNotFound
	}
	return ownerTenantID, err
}

func (a ownershipResolverAdapter) IsRoamingEnabled(ctx context.Context, tenantID int64) (bool, error) {
	return a.inner.IsRoamingEnabled(ctx, tenantID)
}

func (a ownershipResolverAdapter) AreTenantsPartners(ctx context.Context, tenant1, tenant2 int64) (bool, error) {
	return a.inner.AreTenantsPartners(ctx, tenant1, tenant2)
}

// NewRoamingDetector builds the configured detector over the storage layer's
// resolver, wrapped so storage not-found surfaces as roaming.ErrEndpointNotFound.
func NewRoamingDetector(config roaming.DetectorConfig, resolver roaming.EndpointOwnershipResolver, recorder roaming.EventRecorder,
	clk clock.Clock,
) (*roaming.Detector, error) {
	if resolver == nil {
		return nil, errNilRoamingResolver
	}
	return roaming.NewDetector(config, ownershipResolverAdapter{inner: resolver}, recorder, clk)
}

// DetectAndValidateRoaming checks if an endpoint is roaming and validates the operation
func (rs *RoamingService) DetectAndValidateRoaming(ctx context.Context, epEui []byte, servingTenantID int64) (isRoaming bool, ownerTenantID int64, err error) {
	// Detect roaming status
	isRoaming, ownerTenantID, err = rs.detector.DetectRoaming(ctx, epEui, servingTenantID)
	if err != nil {
		return false, 0, fmt.Errorf("%w: %w", errRoamingDetectionFailed, err)
	}

	// If roaming, validate it's allowed
	if isRoaming {
		if err := rs.detector.ValidateRoamingAllowed(ctx, ownerTenantID, servingTenantID); err != nil {
			rs.logger.WarnContext(ctx, bssci.LogBSSCIRoamingNotAllowed,
				logger.FieldEpEui, mioty.FormatEUIBytes(epEui),
				logger.FieldOwnerTenant, ownerTenantID,
				logger.FieldServingTenant, servingTenantID,
				logger.FieldError, err)
			return false, 0, fmt.Errorf("%w: %w", errRoamingNotAllowed, err)
		}
	}

	return isRoaming, ownerTenantID, nil
}

// RecordAttach records an endpoint attach with roaming awareness
func (rs *RoamingService) RecordAttach(ctx context.Context, epEui []byte, bsEui []byte, servingTenantID int64) error {
	ownerTenantID := rs.ownerOrServing(ctx, epEui, servingTenantID)
	return rs.detector.RecordAttachEvent(ctx, epEui, ownerTenantID, servingTenantID, bsEui)
}

// RecordDetach records an endpoint detach with roaming awareness
func (rs *RoamingService) RecordDetach(ctx context.Context, epEui []byte, bsEui []byte, servingTenantID int64) error {
	ownerTenantID := rs.ownerOrServing(ctx, epEui, servingTenantID)
	return rs.detector.RecordDetachEvent(ctx, epEui, ownerTenantID, servingTenantID, bsEui)
}

// UpdateSessionRoaming updates base station session with roaming endpoint info
func (rs *RoamingService) UpdateSessionRoaming(ctx context.Context, sessionID int64, epEui []byte, isAttach bool, servingTenantID int64) error {
	ownerTenantID := rs.ownerOrServing(ctx, epEui, servingTenantID)

	// Only track if actually roaming
	if ownerTenantID == servingTenantID {
		return nil
	}

	epEuiHex := mioty.FormatEUIBytes(epEui)
	if isAttach {
		// Add to roaming endpoints list
		return rs.sessions.AddRoamingEndpointToSession(ctx, sessionID, epEuiHex, ownerTenantID)
	}
	// Remove from roaming endpoints list
	return rs.sessions.RemoveRoamingEndpointFromSession(ctx, sessionID, epEuiHex)
}

// ownerOrServing resolves the owning tenant, falling back to the serving tenant
// when the endpoint is not yet known to the database.
func (rs *RoamingService) ownerOrServing(ctx context.Context, epEui []byte, servingTenantID int64) int64 {
	ownerTenantID, err := rs.sessions.GetEndpointOwner(ctx, epEui)
	if err != nil {
		return servingTenantID
	}
	return ownerTenantID
}
