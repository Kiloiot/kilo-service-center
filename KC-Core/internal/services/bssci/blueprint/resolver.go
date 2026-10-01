// Package blueprint provides blueprint resolution for MIOTY payload decoding.
package blueprint

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Reader resolves blueprints by Type EUI and by device model default.
type Reader interface {
	GetByTypeEUI(ctx context.Context, tenantID int64, typeEUI []byte) (*models.Blueprint, error)
	GetDefaultForModel(ctx context.Context, tenantID int64, deviceModelID uuid.UUID) (*models.Blueprint, error)
}

// ResolverService implements the BlueprintResolver interface.
// It finds blueprints for endpoints based on Type EUI and device model associations.
type ResolverService struct {
	log           logger.Logger
	blueprintRepo Reader
}

const componentBlueprintResolver = "blueprint-resolver"

// NewResolverService creates a new ResolverService instance.
// componentBlueprintResolver labels this component in structured logs.
func NewResolverService(
	log logger.Logger,
	blueprintRepo Reader,
) *ResolverService {
	return &ResolverService{
		log:           log.WithField(logger.FieldComponent, componentBlueprintResolver),
		blueprintRepo: blueprintRepo,
	}
}

// ResolveBlueprint finds the applicable blueprint for an endpoint's type EUI.
// It implements the TypeEUI precedence rule:
// 1. If endpoint has device_model_id, get the default blueprint from that model
// 2. Otherwise, look up blueprint by Type EUI directly
//
// Returns nil if no blueprint is found (not an error - means decoding should be skipped).
func (s *ResolverService) ResolveBlueprint(
	ctx context.Context,
	tenantID int64,
	typeEUI []byte,
	_ *uint8, // formatID reserved for future format-specific blueprint selection
) (*models.Blueprint, error) {
	// If typeEUI is nil or empty, we can't resolve
	if len(typeEUI) == 0 {
		s.log.DebugContext(ctx, LogBlueprintNoTypeEUI)
		return nil, nil
	}

	// Try to find blueprint by Type EUI directly
	bp, err := s.blueprintRepo.GetByTypeEUI(ctx, tenantID, typeEUI)
	if err == nil && bp != nil {
		s.log.DebugContext(ctx, LogBlueprintFoundByTypeEUI,
			logger.FieldBlueprintID, bp.ID,
			logger.FieldTypeEui, mioty.FormatEUIBytes(typeEUI),
			logger.FieldVersion, bp.Version)
		return bp, nil
	}

	// Log if error is not "not found"
	if err != nil && !isNotFoundError(err) {
		s.log.WarnContext(ctx, LogBlueprintTypeEUILookupError,
			logger.FieldTypeEui, mioty.FormatEUIBytes(typeEUI),
			logger.FieldError, err)
	}

	s.log.DebugContext(ctx, LogBlueprintNotFoundForTypeEUI,
		logger.FieldTypeEui, mioty.FormatEUIBytes(typeEUI))
	return nil, nil
}

// ResolveBlueprintForEndpoint finds the applicable blueprint for an endpoint.
// It implements the TypeEUI precedence rule:
// 1. If endpoint has device_model_id, get the default blueprint from that model
// 2. Otherwise, fall back to endpoint's TypeEUI for direct lookup
func (s *ResolverService) ResolveBlueprintForEndpoint(
	ctx context.Context,
	tenantID int64,
	endpoint *models.EndPoint,
	formatID *uint8,
) (*models.Blueprint, error) {
	// Snapshot is the highest-priority decode source; on a malformed snapshot, fall through to catalog resolution.
	if len(endpoint.BlueprintSnapshot) > 0 {
		var snap models.BlueprintSnapshot
		if err := json.Unmarshal(endpoint.BlueprintSnapshot, &snap); err != nil {
			s.log.WarnContext(ctx, LogBlueprintSnapshotParseFailed,
				logger.FieldEndpointEui, endpoint.EUI.String(), logger.FieldError, err)
		} else if bp, err := snap.ToBlueprint(); err != nil {
			s.log.WarnContext(ctx, LogBlueprintSnapshotInvalid,
				logger.FieldEndpointEui, endpoint.EUI.String(), logger.FieldError, err)
		} else {
			s.log.DebugContext(ctx, LogBlueprintResolvedFromSnapshot,
				logger.FieldEndpointEui, endpoint.EUI.String(),
				logger.FieldBlueprintID, bp.ID, logger.FieldVersion, bp.Version)
			return bp, nil
		}
	}

	// Check if endpoint has a device model assigned
	if endpoint.DeviceModelID != nil {
		// TypeEUI precedence rule: Must get Type EUI from model's default blueprint
		// This prevents "loose" TypeEUI when a device model is assigned

		// DeviceModelID is now *uuid.UUID, use directly
		modelID := *endpoint.DeviceModelID

		// Get the default blueprint for this model
		bp, err := s.blueprintRepo.GetDefaultForModel(ctx, tenantID, modelID)
		if err != nil {
			s.log.DebugContext(ctx, LogBlueprintModelDefaultError,
				logger.FieldDeviceModelID, *endpoint.DeviceModelID,
				logger.FieldError, err)
			return nil, nil
		}

		if bp != nil {
			s.log.DebugContext(ctx, LogBlueprintResolvedFromModel,
				logger.FieldEndpointEui, endpoint.EUI.String(),
				logger.FieldDeviceModelID, *endpoint.DeviceModelID,
				logger.FieldBlueprintID, bp.ID)
			return bp, nil
		}

		// Model has no default blueprint - return nil (decode will be skipped)
		s.log.DebugContext(ctx, LogBlueprintModelNoDefault,
			logger.FieldEndpointEui, endpoint.EUI.String(),
			logger.FieldDeviceModelID, *endpoint.DeviceModelID)
		return nil, nil
	}

	// No device model - use endpoint's Type EUI for direct lookup
	if endpoint.TypeEUI != nil {
		typeEUIBytes := endpoint.TypeEUI[:]
		return s.ResolveBlueprint(ctx, tenantID, typeEUIBytes, formatID)
	}

	s.log.DebugContext(ctx, LogBlueprintNoTypeEUIOrModel,
		logger.FieldEndpointEui, endpoint.EUI.String())
	return nil, nil
}

// GetEndpointCalibration retrieves calibration data for an endpoint.
// Returns an empty map if no calibration data is available.
func (s *ResolverService) GetEndpointCalibration(
	ctx context.Context,
	_ int64, // tenantID reserved for future multi-tenant calibration stores
	endpoint *models.EndPoint,
) map[string]interface{} {
	if len(endpoint.CalibrationData) == 0 {
		return make(map[string]interface{})
	}

	// Parse JSON calibration data
	var calibration map[string]interface{}
	if err := parseJSON(endpoint.CalibrationData, &calibration); err != nil {
		s.log.WarnContext(ctx, LogBlueprintCalibrationParseFailed,
			logger.FieldEndpointEui, endpoint.EUI.String(),
			logger.FieldError, err)
		return make(map[string]interface{})
	}

	return calibration
}

// isNotFoundError reports a missing record from either storage sentinel,
// including wrapped errors.
func isNotFoundError(err error) bool {
	return errors.Is(err, storage.ErrNotFound) || errors.Is(err, storage.ErrRecordNotFound)
}

// parseJSON parses JSON data into the target interface.
func parseJSON(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}
