package grpcservices

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

type endpointService struct {
	storage EndpointStore
	index   EndpointIndex
}

// NewEndpointService creates the endpoint service for the gRPC layer.
func NewEndpointService(storage EndpointStore, index EndpointIndex) EndpointService {
	return &endpointService{
		storage: storage,
		index:   index,
	}
}

func (s *endpointService) Create(ctx context.Context, endpoint *models.EndPoint) (*models.EndPoint, error) {
	if err := s.storage.Create(ctx, endpoint); err != nil {
		return nil, err
	}
	if s.index != nil {
		s.index.Add(ctx, endpoint.EUI)
	}
	return endpoint, nil
}

func (s *endpointService) CreateWithStatus(ctx context.Context, endpoint *models.EndPoint, status string) (*models.EndPoint, error) {
	if err := s.storage.CreateWithStatus(ctx, endpoint, status); err != nil {
		return nil, err
	}
	if s.index != nil {
		s.index.Add(ctx, endpoint.EUI)
	}
	return endpoint, nil
}

func (s *endpointService) GetByEUI(ctx context.Context, eui []byte, tenantID int64) (*models.EndPoint, error) {
	return s.storage.GetByEUI(ctx, tenantID, eui)
}

func (s *endpointService) Update(ctx context.Context, endpoint *models.EndPoint) (*models.EndPoint, error) {
	if err := s.storage.Update(ctx, endpoint); err != nil {
		return nil, err
	}
	return endpoint, nil
}

func (s *endpointService) UpdateWithEUI(ctx context.Context, tenantID int64, oldEui []byte, endpoint *models.EndPoint) (*models.EndPoint, error) {
	updated, err := s.storage.UpdateWithEUI(ctx, tenantID, oldEui, endpoint)
	if err != nil {
		return nil, err
	}
	if s.index != nil {
		var old models.EUI
		copy(old[:], oldEui)
		if old != updated.EUI {
			s.index.Remove(ctx, old)
		}
		s.index.Add(ctx, updated.EUI)
	}
	return updated, nil
}

func (s *endpointService) CheckEUIGloballyUnique(ctx context.Context, eui []byte) error {
	return s.storage.CheckEUIUnique(ctx, eui)
}

func (s *endpointService) Delete(ctx context.Context, eui []byte, tenantID int64) (int64, error) {
	removedID, err := s.storage.DeleteByTenant(ctx, tenantID, eui)
	if err != nil {
		return 0, err
	}
	if s.index != nil {
		var removed models.EUI
		copy(removed[:], eui)
		s.index.Remove(ctx, removed)
	}
	return removedID, nil
}

func (s *endpointService) List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.EndPoint, error) {
	return s.storage.ListByTenantPaginated(ctx, tenantID, limit, offset)
}

func (s *endpointService) ListByModelWithSnapshot(ctx context.Context, tenantID int64, deviceModelID uuid.UUID) ([]*models.EndPoint, error) {
	return s.storage.ListByModelWithSnapshot(ctx, tenantID, deviceModelID)
}
