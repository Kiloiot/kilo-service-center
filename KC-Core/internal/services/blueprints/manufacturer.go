package blueprints

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// CreateManufacturer creates a new manufacturer.
func (s *Service) CreateManufacturer(ctx context.Context, req *grpcservices.ManufacturerCreateRequest) (*models.Manufacturer, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var website *string
	if req.Website != "" {
		website = &req.Website
	}

	params := &models.ManufacturerCreateParams{
		TenantID: tenantID,
		IsSystem: req.IsSystem,
		Name:     req.Name,
		Website:  website,
	}

	manufacturer, err := s.manufacturerRepo.Create(ctx, params)
	if err != nil {
		s.logger.ErrorContext(ctx, LogManufacturerCreateFailed, logger.FieldName, req.Name, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpCreateManufacturer, err)
	}

	s.logger.InfoContext(ctx, LogManufacturerCreated, logger.FieldID, manufacturer.ID, logger.FieldName, manufacturer.Name)
	return manufacturer, nil
}

// GetManufacturer retrieves a manufacturer by ID.
func (s *Service) GetManufacturer(ctx context.Context, id uuid.UUID) (*models.Manufacturer, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	manufacturer, err := s.manufacturerRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrManufacturerNotFound
		}
		s.logger.ErrorContext(ctx, LogManufacturerGetFailed, logger.FieldID, id, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpGetManufacturer, err)
	}
	return manufacturer, nil
}

// UpdateManufacturer updates an existing manufacturer.
func (s *Service) UpdateManufacturer(ctx context.Context, id uuid.UUID, req *grpcservices.ManufacturerUpdateRequest) (*models.Manufacturer, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	existing, err := s.manufacturerRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrManufacturerNotFound
		}
		return nil, fmt.Errorf("%s: %w", errOpGetManufacturer, err)
	}

	// Build update params
	params := &models.ManufacturerUpdateParams{}
	if req.Name != nil {
		params.Name = req.Name
	}
	if req.Website != nil {
		params.Website = req.Website
	}

	if err := s.manufacturerRepo.Update(ctx, tenantID, existing.IsSystem, id, params); err != nil {
		s.logger.ErrorContext(ctx, LogManufacturerUpdateFailed, logger.FieldID, id, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpUpdateManufacturer, err)
	}

	// Fetch updated manufacturer
	manufacturer, err := s.manufacturerRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errOpGetUpdatedManufacturer, err)
	}

	s.logger.InfoContext(ctx, LogManufacturerUpdated, logger.FieldID, id)
	return manufacturer, nil
}

// DeleteManufacturer deletes a manufacturer.
func (s *Service) DeleteManufacturer(ctx context.Context, id uuid.UUID) error {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	existing, err := s.manufacturerRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrManufacturerNotFound
		}
		return fmt.Errorf("%s: %w", errOpGetManufacturer, err)
	}

	if err := s.manufacturerRepo.Delete(ctx, tenantID, existing.IsSystem, id); err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrManufacturerNotFound
		}
		s.logger.ErrorContext(ctx, LogManufacturerDeleteFailed, logger.FieldID, id, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpDeleteManufacturer, err)
	}

	s.logger.InfoContext(ctx, LogManufacturerDeleted, logger.FieldID, id)
	return nil
}

// ListManufacturers returns a list of manufacturers.
func (s *Service) ListManufacturers(ctx context.Context, isSystem bool, limit, offset int) ([]*models.Manufacturer, int64, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, 0, err
	}

	params := &models.ManufacturerListParams{
		TenantID: tenantID,
		IsSystem: isSystem,
		Limit:    limit,
		Offset:   offset,
	}

	manufacturers, err := s.manufacturerRepo.List(ctx, params)
	if err != nil {
		s.logger.ErrorContext(ctx, LogManufacturerListFailed, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errOpListManufacturers, err)
	}

	total, err := s.manufacturerRepo.Count(ctx, tenantID, isSystem)
	if err != nil {
		s.logger.ErrorContext(ctx, LogManufacturerCountFailed, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errOpCountManufacturers, err)
	}

	return manufacturers, total, nil
}
