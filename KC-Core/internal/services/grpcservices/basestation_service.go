// Package grpcservices provides service layer implementations for gRPC transport.
// These services wrap storage operations and provide business logic for the gRPC API layer.
package grpcservices

import (
	"context"

	"github.com/kilocenter/KC-Core/pkg/config"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/models"
)

type basestationService struct {
	storage        storage.Storage
	logger         logger.Logger
	protocolConfig *config.ProtocolConfig
}

// NewBaseStationService creates a new basestation service for gRPC layer
func NewBaseStationService(storage storage.Storage, log logger.Logger, protocolCfg *config.ProtocolConfig) BaseStationService {
	return &basestationService{
		storage:        storage,
		logger:         log,
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
		url := config.GetServiceCenterURL(s.protocolConfig)
		bs.ServiceCenterURL = &url
	}

	return s.storage.CreateBaseStation(ctx, bs)
}

// GetByEUI retrieves a base station by EUI
func (s *basestationService) GetByEUI(ctx context.Context, eui []byte, tenantID int64) (*models.BaseStation, error) {
	return s.storage.GetBaseStation(ctx, eui, tenantID)
}

// Update updates a base station by looking up the existing record first to get the real DB ID,
// then merging the incoming fields and persisting.
func (s *basestationService) Update(ctx context.Context, bs *models.BaseStation) (*models.BaseStation, error) {
	existing, err := s.storage.GetBaseStation(ctx, bs.EUI[:], bs.TenantID)
	if err != nil {
		return nil, err
	}

	// Carry the real DB ID so the storage layer's WHERE clause matches
	bs.ID = existing.ID

	return s.storage.UpdateBaseStation(ctx, bs)
}

// UpdateEUI updates the EUI of a base station with cascade to all dependent tables.
// This operation atomically updates the EUI across all tables that reference it.
func (s *basestationService) UpdateEUI(ctx context.Context, tenantID int64, oldEui, newEui []byte) (*models.BaseStation, error) {
	return s.storage.UpdateBaseStationEUI(ctx, tenantID, oldEui, newEui)
}

// Delete deletes a base station
func (s *basestationService) Delete(ctx context.Context, eui []byte, tenantID int64) error {
	return s.storage.DeleteBaseStation(ctx, eui, tenantID)
}

// List lists base stations for a tenant
func (s *basestationService) List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.BaseStation, error) {
	return s.storage.ListBaseStations(ctx, tenantID, limit, offset)
}

// ListAllLocations returns all base stations with coordinates across all tenants.
func (s *basestationService) ListAllLocations(ctx context.Context) ([]*models.BaseStation, error) {
	return s.storage.ListAllBaseStationLocations(ctx)
}
