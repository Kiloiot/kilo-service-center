package blueprints

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	blueprintconstants "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/blueprint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// CreateDeviceModel creates a new device model.
func (s *Service) CreateDeviceModel(ctx context.Context, req *grpcservices.DeviceModelCreateRequest) (*models.DeviceModel, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	mfr, err := s.manufacturerRepo.GetByID(ctx, tenantID, req.ManufacturerID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrManufacturerNotFound
		}
		return nil, fmt.Errorf("%s: %w", errOpGetManufacturer, err)
	}
	// Ownership homogeneity: a Custom model can't hang off a System manufacturer (or vice versa).
	if mfr.IsSystem != req.IsSystem {
		return nil, ErrOwnershipMismatch
	}

	// Canonicalize model code: generate slug from name if empty, validate if provided
	code := req.Code
	if code == "" {
		code = blueprintconstants.Slug(req.Name)
	} else {
		var err error
		code, err = blueprintconstants.CanonicalModelCode(code)
		if err != nil {
			return nil, err
		}
	}

	var description *string
	if req.Description != "" {
		description = &req.Description
	}
	var datasheetURL *string
	if req.DatasheetURL != "" {
		datasheetURL = &req.DatasheetURL
	}

	params := &models.DeviceModelCreateParams{
		TenantID:       tenantID,
		IsSystem:       req.IsSystem,
		ManufacturerID: req.ManufacturerID,
		Name:           req.Name,
		Code:           code,
		TypeEUI:        req.TypeEUI,
		Description:    description,
		DatasheetURL:   datasheetURL,
	}

	model, err := s.deviceModelRepo.Create(ctx, params)
	if err != nil {
		s.logger.ErrorContext(ctx, LogDeviceModelCreateFailed, logger.FieldName, req.Name, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpCreateDeviceModel, err)
	}

	s.logger.InfoContext(ctx, LogDeviceModelCreated, logger.FieldID, model.ID, logger.FieldName, model.Name)
	return model, nil
}

// GetDeviceModel retrieves a device model by ID.
func (s *Service) GetDeviceModel(ctx context.Context, id uuid.UUID) (*models.DeviceModel, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	model, err := s.deviceModelRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrDeviceModelNotFound
		}
		s.logger.ErrorContext(ctx, LogDeviceModelGetFailed, logger.FieldID, id, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpGetDeviceModel, err)
	}
	return model, nil
}

// GetDeviceModelForTenant retrieves a device model by ID using an explicit tenant ID
// instead of the service's default tenant. Used for request-context tenant validation.
func (s *Service) GetDeviceModelForTenant(
	ctx context.Context,
	tenantID int64,
	id uuid.UUID,
) (*models.DeviceModel, error) {
	model, err := s.deviceModelRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrDeviceModelNotFound
		}
		s.logger.ErrorContext(ctx, LogDeviceModelGetForTenantFailed,
			logger.FieldTenantID, tenantID, logger.FieldID, id, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpGetDeviceModelForTenant, err)
	}
	return model, nil
}

// UpdateDeviceModel updates an existing device model.
func (s *Service) UpdateDeviceModel(ctx context.Context, id uuid.UUID, req *grpcservices.DeviceModelUpdateRequest) (*models.DeviceModel, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	existing, err := s.deviceModelRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrDeviceModelNotFound
		}
		return nil, fmt.Errorf("%s: %w", errOpGetDeviceModel, err)
	}

	// Build update params
	params := &models.DeviceModelUpdateParams{}
	if req.Name != nil {
		params.Name = req.Name
	}
	if req.Code != nil {
		canonicalized, err := blueprintconstants.CanonicalModelCode(*req.Code)
		if err != nil {
			return nil, err
		}
		params.Code = &canonicalized
	}
	if req.TypeEUI != nil {
		params.TypeEUI = req.TypeEUI
	}
	if req.Description != nil {
		params.Description = req.Description
	}
	if req.DatasheetURL != nil {
		params.DatasheetURL = req.DatasheetURL
	}

	if err := s.deviceModelRepo.Update(ctx, tenantID, existing.IsSystem, id, params); err != nil {
		s.logger.ErrorContext(ctx, LogDeviceModelUpdateFailed, logger.FieldID, id, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpUpdateDeviceModel, err)
	}

	// Fetch updated model
	model, err := s.deviceModelRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errOpGetUpdatedDeviceModel, err)
	}

	s.logger.InfoContext(ctx, LogDeviceModelUpdated, logger.FieldID, id)
	return model, nil
}

// DeleteDeviceModel deletes a device model.
func (s *Service) DeleteDeviceModel(ctx context.Context, id uuid.UUID) error {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	existing, err := s.deviceModelRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrDeviceModelNotFound
		}
		return fmt.Errorf("%s: %w", errOpGetDeviceModel, err)
	}

	if err := s.deviceModelRepo.Delete(ctx, tenantID, existing.IsSystem, id); err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrDeviceModelNotFound
		}
		s.logger.ErrorContext(ctx, LogDeviceModelDeleteFailed, logger.FieldID, id, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpDeleteDeviceModel, err)
	}

	s.logger.InfoContext(ctx, LogDeviceModelDeleted, logger.FieldID, id)
	return nil
}

// ListDeviceModels returns a list of device models.
func (s *Service) ListDeviceModels(ctx context.Context, isSystem bool, manufacturerID *uuid.UUID, limit, offset int) ([]*models.DeviceModel, int64, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, 0, err
	}

	params := &models.DeviceModelListParams{
		TenantID: tenantID,
		IsSystem: isSystem,
		Limit:    limit,
		Offset:   offset,
	}
	if manufacturerID != nil {
		params.ManufacturerID = manufacturerID
	}

	models, err := s.deviceModelRepo.List(ctx, params)
	if err != nil {
		s.logger.ErrorContext(ctx, LogDeviceModelListFailed, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errOpListDeviceModels, err)
	}

	// Use appropriate count method based on filter
	var total int64
	if manufacturerID != nil {
		total, err = s.deviceModelRepo.CountByManufacturer(ctx, tenantID, isSystem, *manufacturerID)
	} else {
		total, err = s.deviceModelRepo.Count(ctx, tenantID, isSystem)
	}
	if err != nil {
		s.logger.ErrorContext(ctx, LogDeviceModelCountFailed, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errOpCountDeviceModels, err)
	}

	return models, total, nil
}

// CreateDeviceModelWithBlueprint creates a device model and default blueprint in a single transaction.
func (s *Service) CreateDeviceModelWithBlueprint(ctx context.Context, req *grpcservices.DeviceModelWithBlueprintRequest) (*models.DeviceModel, *models.Blueprint, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	if s.txRunner == nil {
		return nil, nil, ErrTxRunnerNotConfigured
	}

	if err := s.validateDeviceModelManufacturer(ctx, tenantID, req); err != nil {
		return nil, nil, err
	}

	// Extract typeEui from spec before transaction so it's available for model creation
	specTypeEUI, err := blueprintconstants.TypeEUIFromSpec(req.DecoderScript)
	if err != nil {
		return nil, nil, err
	}

	var (
		model     *models.DeviceModel
		blueprint *models.Blueprint
	)
	if err = s.txRunner.Run(ctx, func(tx BlueprintTx) error {
		model, err = s.createDeviceModelWithSlug(ctx, tx, tenantID, req, specTypeEUI)
		if err != nil {
			return err
		}

		bpParams := &models.BlueprintCreateParams{
			TenantID:      tenantID,
			IsSystem:      req.IsSystem,
			DeviceModelID: model.ID,
			Version:       req.Version,
			TypeEUI:       specTypeEUI,
			SpecJSON:      req.DecoderScript,
			IsDefault:     true,
		}

		blueprint, err = tx.CreateBlueprint(ctx, bpParams)
		if err != nil {
			return fmt.Errorf("%s: %w", errOpCreateBlueprint, err)
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}

	s.logger.InfoContext(ctx, LogDeviceModelWithBlueprintCreated,
		logger.FieldModelID, model.ID, logger.FieldBlueprintID, blueprint.ID, logger.FieldSlug, model.Code)

	return model, blueprint, nil
}

func (s *Service) validateDeviceModelManufacturer(ctx context.Context, tenantID int64, req *grpcservices.DeviceModelWithBlueprintRequest) error {
	mfr, err := s.manufacturerRepo.GetByID(ctx, tenantID, req.ManufacturerID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrManufacturerNotFound
		}
		return fmt.Errorf("%s: %w", errOpGetManufacturer, err)
	}
	// Ownership homogeneity: model+blueprint created here inherit req.IsSystem; parent must match.
	if mfr.IsSystem != req.IsSystem {
		return ErrOwnershipMismatch
	}
	return nil
}

// Retries with a suffixed slug on code collisions; ErrSlugGenerationFailed when all attempts fail.
func (s *Service) createDeviceModelWithSlug(ctx context.Context, tx BlueprintTx, tenantID int64, req *grpcservices.DeviceModelWithBlueprintRequest, specTypeEUI []byte) (*models.DeviceModel, error) {
	baseSlug := blueprintconstants.Slug(req.Name)
	slug := baseSlug
	for attempt := range blueprintconstants.ModelCodeMaxSuffixAttempts {
		if attempt > 0 {
			slug = fmt.Sprintf("%s"+blueprintconstants.ModelCodeSuffixFormat, baseSlug, attempt+1)
			if len(slug) > blueprintconstants.ModelCodeMaxLength {
				slug = slug[:blueprintconstants.ModelCodeMaxLength]
			}
		}
		modelParams := &models.DeviceModelCreateParams{
			TenantID:       tenantID,
			IsSystem:       req.IsSystem,
			ManufacturerID: req.ManufacturerID,
			Name:           req.Name,
			Code:           slug,
			TypeEUI:        specTypeEUI,
		}
		model, createErr := tx.CreateDeviceModel(ctx, modelParams)
		if createErr != nil {
			if errors.Is(createErr, storage.ErrDuplicateKey) {
				continue
			}
			return nil, fmt.Errorf("%s: %w", errOpCreateDeviceModel, createErr)
		}
		return model, nil
	}
	return nil, ErrSlugGenerationFailed
}
