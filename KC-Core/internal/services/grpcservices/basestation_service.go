// Package grpcservices provides service layer implementations for gRPC transport.
// These services wrap storage operations and provide business logic for the gRPC API layer.
package grpcservices

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// StationSessions closes the live BSSCI session of a base station and
// reports whether it held one.
type StationSessions interface {
	CloseSessionByEUI(ctx context.Context, eui uint64) bool
}

// StationDownlinks settles the downlinks a deleted base station held.
type StationDownlinks interface {
	ReleaseDeletedStation(ctx context.Context, bsEUI uint64)
}

// BaseStationServiceDeps are the base station service's collaborators.
type BaseStationServiceDeps struct {
	Store     BaseStationStore
	Protocol  *config.ProtocolConfig
	Sessions  StationSessions
	Downlinks StationDownlinks
}

type basestationService struct {
	storage        BaseStationStore
	protocolConfig *config.ProtocolConfig
	sessions       StationSessions
	downlinks      StationDownlinks
}

// NewBaseStationService creates a new basestation service for gRPC layer
func NewBaseStationService(deps BaseStationServiceDeps) BaseStationService {
	return &basestationService{
		storage:        deps.Store,
		protocolConfig: deps.Protocol,
		sessions:       deps.Sessions,
		downlinks:      deps.Downlinks,
	}
}

// Create creates a new base station
func (s *basestationService) Create(ctx context.Context, bs *models.BaseStation) (*models.BaseStation, error) {
	// Set default connection type if not specified
	if bs.ConnectionType == "" {
		bs.ConnectionType = models.ConnectionTypeBSSCI
	}

	// Set service center URL for BSSCI connections if not specified
	if bs.ConnectionType == models.ConnectionTypeBSSCI && bs.ServiceCenterURL == nil {
		bs.ServiceCenterURL = config.StoredServiceCenterURL(config.GetServiceCenterURL(s.protocolConfig))
	}

	if err := s.storage.Create(ctx, bs); err != nil {
		return nil, err
	}
	return bs, nil
}

// GetByEUI retrieves a base station by EUI
func (s *basestationService) GetByEUI(ctx context.Context, eui []byte, tenantID int64) (*models.BaseStation, error) {
	return s.storage.GetByEUI(ctx, tenantID, eui)
}

// Update updates a base station by looking up the existing record first to get the real DB ID,
// then merging the incoming fields and persisting.
func (s *basestationService) Update(ctx context.Context, bs *models.BaseStation) (*models.BaseStation, error) {
	existing, err := s.storage.GetByEUI(ctx, bs.TenantID, bs.EUI[:])
	if err != nil {
		return nil, err
	}

	// Carry the real DB ID so the storage layer's WHERE clause matches
	bs.ID = existing.ID

	if err := s.storage.UpdateProfile(ctx, bs); err != nil {
		return nil, err
	}
	return bs, nil
}

// UpdateEUI updates the EUI of a base station with cascade to all dependent tables.
// This operation atomically updates the EUI across all tables that reference it.
func (s *basestationService) UpdateEUI(ctx context.Context, tenantID int64, oldEui, newEui []byte) (*models.BaseStation, error) {
	return s.storage.UpdateEUI(ctx, tenantID, oldEui, newEui)
}

// Delete deletes a base station and returns the station it removed. Its live
// session is closed, so it can no longer transmit what it held, and only then
// are the downlinks it held settled.
func (s *basestationService) Delete(ctx context.Context, eui []byte, tenantID int64) (*models.BaseStation, error) {
	removed, err := s.storage.DeleteByEUI(ctx, tenantID, eui)
	if err != nil {
		return nil, err
	}
	bsEUI := removed.EUI.ToUint64()
	s.sessions.CloseSessionByEUI(ctx, bsEUI)
	s.downlinks.ReleaseDeletedStation(ctx, bsEUI)
	return removed, nil
}

// List lists base stations for a tenant
func (s *basestationService) List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.BaseStation, error) {
	stations, _, err := s.storage.List(ctx, &models.BaseStationFilter{TenantID: tenantID, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	return stations, nil
}

// ListAllLocations returns all base stations with coordinates across all tenants.
func (s *basestationService) ListAllLocations(ctx context.Context) ([]*models.BaseStation, error) {
	return s.storage.ListAllLocations(ctx)
}
