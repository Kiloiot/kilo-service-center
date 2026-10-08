// Package integrations provides integration CRUD for event sink management.
package integrations

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// IntegrationStore persists integrations for a tenant.
type IntegrationStore interface {
	Create(ctx context.Context, integration *models.Integration) error
	GetByID(ctx context.Context, id int64, tenantID int64) (*models.Integration, error)
	ListByTenant(ctx context.Context, tenantID int64, limit, offset int) ([]*models.Integration, int64, error)
	Update(ctx context.Context, integration *models.Integration) error
	Delete(ctx context.Context, id int64, tenantID int64) error
}

// Service implements grpcservices.IntegrationService.
type Service struct {
	repo   IntegrationStore
	logger logger.Logger
}

// New creates a new integration service
func New(repo IntegrationStore, log logger.Logger) *Service {
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
		s.logger.ErrorContext(ctx, LogIntegrationCreateFailed, logger.FieldName, req.Name, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", ErrCreateIntegration, err)
	}

	s.logger.InfoContext(ctx, LogIntegrationCreated, logger.FieldID, integration.ID, logger.FieldName, integration.Name)
	return integration, nil
}

// GetByID retrieves an integration by ID
func (s *Service) GetByID(ctx context.Context, tenantID int64, id int64) (*models.Integration, error) {
	integration, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrIntegrationNotFound
		}
		s.logger.ErrorContext(ctx, LogIntegrationGetFailed, logger.FieldID, id, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", ErrGetIntegration, err)
	}
	return integration, nil
}

// Update updates an existing integration
func (s *Service) Update(ctx context.Context, tenantID int64, id int64, req *grpcservices.IntegrationUpdateRequest) (*models.Integration, error) {
	// Get existing integration
	integration, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrIntegrationNotFound
		}
		return nil, fmt.Errorf("%w: %w", ErrGetIntegration, err)
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
		s.logger.ErrorContext(ctx, LogIntegrationUpdateFailed, logger.FieldID, id, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", ErrUpdateIntegration, err)
	}

	s.logger.InfoContext(ctx, LogIntegrationUpdated, logger.FieldID, integration.ID, logger.FieldName, integration.Name)
	return integration, nil
}

// Delete deletes an integration
func (s *Service) Delete(ctx context.Context, tenantID int64, id int64) error {
	if err := s.repo.Delete(ctx, id, tenantID); err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrIntegrationNotFound
		}
		s.logger.ErrorContext(ctx, LogIntegrationDeleteFailed, logger.FieldID, id, logger.FieldError, err)
		return fmt.Errorf("%w: %w", ErrDeleteIntegration, err)
	}

	s.logger.InfoContext(ctx, LogIntegrationDeleted, logger.FieldID, id)
	return nil
}

// List returns paginated integrations for a tenant
func (s *Service) List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.Integration, int64, error) {
	integrations, count, err := s.repo.ListByTenant(ctx, tenantID, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, LogIntegrationListFailed, logger.FieldTenantIDSnake, tenantID, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%w: %w", ErrListIntegrations, err)
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
