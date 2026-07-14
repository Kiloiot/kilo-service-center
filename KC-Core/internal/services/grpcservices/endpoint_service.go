package grpcservices

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

type endpointService struct {
	storage storage.Storage
	logger  logger.Logger
}

// NewEndpointService creates a new endpoint service for gRPC layer
func NewEndpointService(storage storage.Storage, log logger.Logger) EndpointService {
	return &endpointService{
		storage: storage,
		logger:  log,
	}
}

// Create creates a new endpoint
func (s *endpointService) Create(ctx context.Context, endpoint *models.EndPoint) (*models.EndPoint, error) {
	return s.storage.CreateEndPoint(ctx, endpoint)
}

// GetByEUI retrieves an endpoint by EUI
func (s *endpointService) GetByEUI(ctx context.Context, eui []byte, tenantID int64) (*models.EndPoint, error) {
	return s.storage.GetEndPoint(ctx, eui, tenantID)
}

// Update updates an endpoint
func (s *endpointService) Update(ctx context.Context, endpoint *models.EndPoint) (*models.EndPoint, error) {
	return s.storage.UpdateEndPoint(ctx, endpoint)
}

// UpdateWithEUI atomically cascades an EUI change across all dependent tables and updates fields
func (s *endpointService) UpdateWithEUI(ctx context.Context, tenantID int64, oldEui []byte, endpoint *models.EndPoint) (*models.EndPoint, error) {
	return s.storage.UpdateEndPointWithEUI(ctx, tenantID, oldEui, endpoint)
}

// CheckEUIGloballyUnique verifies that no endpoint with the given EUI exists (any tenant)
func (s *endpointService) CheckEUIGloballyUnique(ctx context.Context, eui []byte) error {
	return s.storage.CheckEndPointEUIUnique(ctx, eui)
}

// Delete deletes an endpoint
func (s *endpointService) Delete(ctx context.Context, eui []byte, tenantID int64) error {
	return s.storage.DeleteEndPoint(ctx, eui, tenantID)
}

// List lists endpoints for a tenant
func (s *endpointService) List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.EndPoint, error) {
	return s.storage.ListEndPoints(ctx, tenantID, limit, offset)
}

// ListByModelWithSnapshot lists a tenant's snapshot-bearing endpoints for a device model.
func (s *endpointService) ListByModelWithSnapshot(ctx context.Context, tenantID int64, deviceModelID uuid.UUID) ([]*models.EndPoint, error) {
	return s.storage.ListEndPointsByModelWithSnapshot(ctx, tenantID, deviceModelID)
}
