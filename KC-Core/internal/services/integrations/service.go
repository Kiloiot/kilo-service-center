// Package integrations provides integration CRUD for event sink management.
package integrations

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Sentinel errors for integration operations
var (
	ErrIntegrationNotFound = errors.New("integration not found")
	ErrInvalidType         = errors.New("invalid integration type")
	ErrInvalidStatus       = errors.New("invalid integration status")
)

// Service implements grpcservices.IntegrationService.
type Service struct {
	repo   interfaces.IntegrationRepository
	logger logger.Logger
}

// New creates a new integration service
func New(repo interfaces.IntegrationRepository, log logger.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: log,
	}
}

// Create creates a new integration
func (s *Service) Create(ctx context.Context, tenantID int64, req *grpcservices.IntegrationCreateRequest) (*models.Integration, error) {
	// Validate type
	if !isValidType(req.Type) {
		return nil, ErrInvalidType
	}

	// Set default delivery format
	deliveryFormat := req.DeliveryFormat
	if deliveryFormat == "" {
		deliveryFormat = models.IntegrationDeliveryFormatJSON
	}

	// Build integration model
	integration := &models.Integration{
		OrgID:          req.OrgID,
		TenantID:       tenantID,
		Name:           req.Name,
		Type:           req.Type,
		Config:         req.Config,
		EventFilter:    models.NullJSON{Valid: req.EventFilter != nil, Data: req.EventFilter},
		DeliveryFormat: deliveryFormat,
		Status:         models.IntegrationStatusActive,
	}

	// Set optional fields
	if req.Description != "" {
		integration.Description = &req.Description
	}
	if req.CreatedBy != "" {
		integration.CreatedBy = &req.CreatedBy
	}

	// Create integration
	if err := s.repo.Create(ctx, integration); err != nil {
		s.logger.ErrorContext(ctx, "failed to create integration", "name", req.Name, "error", err)
		return nil, fmt.Errorf("create integration: %w", err)
	}

	s.logger.InfoContext(ctx, "integration created", "id", integration.ID, "name", integration.Name)
	return integration, nil
}

// GetByID retrieves an integration by ID
func (s *Service) GetByID(ctx context.Context, tenantID int64, id int64) (*models.Integration, error) {
	integration, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrIntegrationNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get integration", "id", id, "error", err)
		return nil, fmt.Errorf("get integration: %w", err)
	}
	return integration, nil
}

// Update updates an existing integration
func (s *Service) Update(ctx context.Context, tenantID int64, id int64, req *grpcservices.IntegrationUpdateRequest) (*models.Integration, error) {
	// Get existing integration
	integration, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrIntegrationNotFound
		}
		return nil, fmt.Errorf("get integration: %w", err)
	}

	// Apply updates
	if req.Name != nil {
		integration.Name = *req.Name
	}
	if req.Description != nil {
		integration.Description = req.Description
	}
	if req.Config != nil {
		integration.Config = req.Config
	}
	if req.EventFilter != nil {
		integration.EventFilter = models.NullJSON{Valid: true, Data: req.EventFilter}
	}
	if req.Status != nil {
		if !isValidStatus(*req.Status) {
			return nil, ErrInvalidStatus
		}
		integration.Status = *req.Status
	}
	if req.UpdatedBy != "" {
		integration.UpdatedBy = &req.UpdatedBy
	}

	// Update integration
	if err := s.repo.Update(ctx, integration); err != nil {
		s.logger.ErrorContext(ctx, "failed to update integration", "id", id, "error", err)
		return nil, fmt.Errorf("update integration: %w", err)
	}

	s.logger.InfoContext(ctx, "integration updated", "id", integration.ID, "name", integration.Name)
	return integration, nil
}

// Delete deletes an integration
func (s *Service) Delete(ctx context.Context, tenantID int64, id int64) error {
	if err := s.repo.Delete(ctx, id, tenantID); err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrIntegrationNotFound
		}
		s.logger.ErrorContext(ctx, "failed to delete integration", "id", id, "error", err)
		return fmt.Errorf("delete integration: %w", err)
	}

	s.logger.InfoContext(ctx, "integration deleted", "id", id)
	return nil
}

// List returns paginated integrations for a tenant
func (s *Service) List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.Integration, int64, error) {
	integrations, count, err := s.repo.ListByTenant(ctx, tenantID, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list integrations", "tenant_id", tenantID, "error", err)
		return nil, 0, fmt.Errorf("list integrations: %w", err)
	}
	return integrations, count, nil
}

// isValidType checks if the integration type is valid
func isValidType(t string) bool {
	switch t {
	case models.IntegrationTypeHTTP, models.IntegrationTypeMQTT, models.IntegrationTypeDatabase:
		return true
	default:
		return false
	}
}

// isValidStatus checks if the integration status is valid
func isValidStatus(s string) bool {
	switch s {
	case models.IntegrationStatusActive, models.IntegrationStatusPaused, models.IntegrationStatusDisabled:
		return true
	default:
		return false
	}
}
