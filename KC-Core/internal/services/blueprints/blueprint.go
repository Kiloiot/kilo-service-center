package blueprints

import (
	"bytes"
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

// CreateBlueprint creates a new blueprint.
func (s *Service) CreateBlueprint(ctx context.Context, req *grpcservices.BlueprintCreateRequest) (*models.Blueprint, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	model, err := s.deviceModelRepo.GetByID(ctx, tenantID, req.DeviceModelID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrDeviceModelNotFound
		}
		return nil, fmt.Errorf("%s: %w", errOpGetDeviceModel, err)
	}
	// Ownership homogeneity: a Custom blueprint can't hang off a System model (or vice versa).
	if model.IsSystem != req.IsSystem {
		return nil, ErrOwnershipMismatch
	}

	// Extract typeEui from spec JSON if provided (per MIOTY spec section 2.2.3)
	typeEUI := req.TypeEUI
	if len(req.SpecJSON) > 0 {
		specTypeEUI, err := blueprintconstants.TypeEUIFromSpec(req.SpecJSON)
		if err != nil {
			return nil, err
		}
		// Cross-validate: if request also provides TypeEUI, it must match spec
		if len(typeEUI) > 0 && !bytes.Equal(typeEUI, specTypeEUI) {
			return nil, ErrInvalidTypeEUIFormat
		}
		typeEUI = specTypeEUI
	}
	if len(typeEUI) == 0 {
		typeEUI = model.TypeEUI
	}
	if len(typeEUI) > 0 && len(typeEUI) != blueprintconstants.TypeEUILength {
		return nil, ErrInvalidTypeEUIFormat
	}

	params := &models.BlueprintCreateParams{
		TenantID:      tenantID,
		IsSystem:      req.IsSystem,
		DeviceModelID: req.DeviceModelID,
		Version:       req.Version,
		TypeEUI:       typeEUI,
		SpecJSON:      req.SpecJSON,
		IsDefault:     req.IsDefault,
	}

	// Auto-default: when the model has no current default, this blueprint becomes it so snapshot-less endpoints still resolve.
	if !params.IsDefault {
		def, defErr := s.blueprintRepo.GetDefaultForModel(ctx, tenantID, req.DeviceModelID)
		if defErr != nil {
			return nil, fmt.Errorf("%s: %w", errOpCheckDefaultForModel, defErr)
		}
		if def == nil {
			params.IsDefault = true
		}
	}

	blueprint, err := s.blueprintRepo.Create(ctx, params)
	if err != nil {
		s.logger.ErrorContext(ctx, LogBlueprintCreateFailed, logger.FieldVersion, req.Version, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpCreateBlueprint, err)
	}

	s.logger.InfoContext(ctx, LogBlueprintCreated, logger.FieldID, blueprint.ID, logger.FieldVersion, blueprint.Version)
	return blueprint, nil
}

// GetBlueprint retrieves a blueprint by ID.
func (s *Service) GetBlueprint(ctx context.Context, id uuid.UUID) (*models.Blueprint, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	blueprint, err := s.blueprintRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrBlueprintNotFound
		}
		s.logger.ErrorContext(ctx, LogBlueprintGetFailed, logger.FieldID, id, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpGetBlueprint, err)
	}
	return blueprint, nil
}

// UpdateBlueprint updates an existing blueprint.
func (s *Service) UpdateBlueprint(ctx context.Context, id uuid.UUID, req *grpcservices.BlueprintUpdateRequest) (*models.Blueprint, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	existing, err := s.blueprintRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrBlueprintNotFound
		}
		return nil, fmt.Errorf("%s: %w", errOpGetBlueprint, err)
	}

	// Build update params
	params := &models.BlueprintUpdateParams{}
	if req.Version != nil {
		params.Version = req.Version
	}
	if req.TypeEUI != nil {
		params.TypeEUI = req.TypeEUI
	}
	// If SpecJSON is updated, extract and persist typeEui from it (per section 2.2.3)
	if len(req.SpecJSON) > 0 {
		params.SpecJSON = req.SpecJSON
		specTypeEUI, err := blueprintconstants.TypeEUIFromSpec(req.SpecJSON)
		if err != nil {
			return nil, err
		}
		// Cross-validate: if request also provides TypeEUI, it must match spec
		if req.TypeEUI != nil && !bytes.Equal(req.TypeEUI, specTypeEUI) {
			return nil, ErrInvalidTypeEUIFormat
		}
		params.TypeEUI = specTypeEUI
	}
	if req.IsDefault != nil {
		params.IsDefault = req.IsDefault
	}

	if err := s.blueprintRepo.Update(ctx, tenantID, existing.IsSystem, id, params); err != nil {
		s.logger.ErrorContext(ctx, LogBlueprintUpdateFailed, logger.FieldID, id, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpUpdateBlueprint, err)
	}

	// Fetch updated blueprint
	blueprint, err := s.blueprintRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errOpGetUpdatedBlueprint, err)
	}

	s.logger.InfoContext(ctx, LogBlueprintUpdated, logger.FieldID, id)
	return blueprint, nil
}

// DeleteBlueprint deletes a blueprint.
func (s *Service) DeleteBlueprint(ctx context.Context, id uuid.UUID) error {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	existing, err := s.blueprintRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrBlueprintNotFound
		}
		return fmt.Errorf("%s: %w", errOpGetBlueprint, err)
	}

	if err := s.blueprintRepo.Delete(ctx, tenantID, existing.IsSystem, id); err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrBlueprintNotFound
		}
		s.logger.ErrorContext(ctx, LogBlueprintDeleteFailed, logger.FieldID, id, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpDeleteBlueprint, err)
	}

	s.logger.InfoContext(ctx, LogBlueprintDeleted, logger.FieldID, id)
	return nil
}

// ListBlueprints returns a list of blueprints.
func (s *Service) ListBlueprints(ctx context.Context, isSystem bool, deviceModelID *uuid.UUID, limit, offset int) ([]*models.Blueprint, int64, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, 0, err
	}

	params := &models.BlueprintListParams{
		TenantID: tenantID,
		IsSystem: isSystem,
		Limit:    limit,
		Offset:   offset,
	}
	if deviceModelID != nil {
		params.DeviceModelID = deviceModelID
	}

	blueprints, err := s.blueprintRepo.List(ctx, params)
	if err != nil {
		s.logger.ErrorContext(ctx, LogBlueprintListFailed, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errOpListBlueprints, err)
	}

	// Use appropriate count method based on filter
	var total int64
	if deviceModelID != nil {
		total, err = s.blueprintRepo.CountByDeviceModel(ctx, tenantID, isSystem, *deviceModelID)
	} else {
		total, err = s.blueprintRepo.Count(ctx, tenantID, isSystem)
	}
	if err != nil {
		s.logger.ErrorContext(ctx, LogBlueprintCountFailed, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errOpCountBlueprints, err)
	}

	return blueprints, total, nil
}

// SetDefaultBlueprint sets a blueprint as the default for its device model.
func (s *Service) SetDefaultBlueprint(ctx context.Context, id uuid.UUID) error {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	existing, err := s.blueprintRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrBlueprintNotFound
		}
		return fmt.Errorf("%s: %w", errOpGetBlueprint, err)
	}

	if err := s.blueprintRepo.SetDefault(ctx, tenantID, existing.IsSystem, id); err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrBlueprintNotFound
		}
		s.logger.ErrorContext(ctx, LogBlueprintSetDefaultFailed, logger.FieldID, id, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpSetDefaultBlueprint, err)
	}

	s.logger.InfoContext(ctx, LogBlueprintDefaultSet, logger.FieldID, id)
	return nil
}

// GetDefaultForModel returns the device model's default blueprint (nil, nil when none).
func (s *Service) GetDefaultForModel(ctx context.Context, tenantID int64, modelID uuid.UUID) (*models.Blueprint, error) {
	return s.blueprintRepo.GetDefaultForModel(ctx, tenantID, modelID)
}

// ResolveEffectiveTypeEUI returns the default blueprint's 8-byte TypeEUI for a model, or nil when there's no default or valid TypeEUI (no fallback to DeviceModel.TypeEUI).
func (s *Service) ResolveEffectiveTypeEUI(ctx context.Context, tenantID int64, modelID uuid.UUID) (*models.EUI, error) {
	bp, err := s.blueprintRepo.GetDefaultForModel(ctx, tenantID, modelID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errOpResolveEffectiveTypeEUI, err)
	}
	if bp == nil {
		return nil, nil
	}
	if len(bp.TypeEUI) == 8 {
		var eui models.EUI
		copy(eui[:], bp.TypeEUI)
		return &eui, nil
	}
	return nil, nil
}
