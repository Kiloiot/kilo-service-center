// Package grpcservices provides service layer implementations for gRPC transport.
// These services wrap storage operations and provide business logic for the gRPC API layer.
package grpcservices

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

type basestationService struct {
	storage        BaseStationStore
	protocolConfig *config.ProtocolConfig
}

// NewBaseStationService creates a new basestation service for gRPC layer
func NewBaseStationService(storage BaseStationStore, protocolCfg *config.ProtocolConfig) BaseStationService {
	return &basestationService{
		storage:        storage,
		protocolConfig: protocolCfg,
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

// Delete deletes a base station and returns the station it removed.
func (s *basestationService) Delete(ctx context.Context, eui []byte, tenantID int64) (*models.BaseStation, error) {
	return s.storage.DeleteByEUI(ctx, tenantID, eui)
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
