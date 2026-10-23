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

// SnapshotOnlyModeReader tells whether a tenant decodes only from pinned snapshots.
type SnapshotOnlyModeReader interface {
	IsSnapshotOnly(ctx context.Context, tenantID int64) (bool, error)
}

// ResolverService implements the BlueprintResolver interface.
// It finds blueprints for endpoints based on Type EUI and device model associations.
type ResolverService struct {
	log           logger.Logger
	blueprintRepo Reader
	modes         SnapshotOnlyModeReader
}

const componentBlueprintResolver = "blueprint-resolver"

// NewResolverService creates a new ResolverService instance.
// componentBlueprintResolver labels this component in structured logs.
func NewResolverService(
	log logger.Logger,
	blueprintRepo Reader,
	modes SnapshotOnlyModeReader,
) (*ResolverService, error) {
	if modes == nil {
		return nil, errNilSnapshotOnlyModes
	}
	return &ResolverService{
		log:           log.WithField(logger.FieldComponent, componentBlueprintResolver),
		blueprintRepo: blueprintRepo,
		modes:         modes,
	}, nil
}

// ResolveBlueprint finds the applicable blueprint for an endpoint's type EUI.
//
// Returns nil if no blueprint is found (not an error - means decoding should be skipped).
func (s *ResolverService) ResolveBlueprint(
	ctx context.Context,
	tenantID int64,
	typeEUI []byte,
	_ *uint8, // formatID reserved for future format-specific blueprint selection
) (*models.Blueprint, error) {
	if len(typeEUI) == 0 {
		s.log.DebugContext(ctx, LogBlueprintNoTypeEUI)
		return nil, nil
	}
	bp, err := s.blueprintByTypeEUI(ctx, tenantID, typeEUI)
	if err != nil {
		return nil, nil
	}
	return bp, nil
}

// ResolveBlueprintForEndpoint finds the applicable blueprint for an endpoint:
// its pinned snapshot, else - unless the tenant decodes only from snapshots -
// the catalog (catalogBlueprint). A failed catalog read skips decoding.
func (s *ResolverService) ResolveBlueprintForEndpoint(
	ctx context.Context,
	tenantID int64,
	endpoint *models.EndPoint,
	_ *uint8, // formatID reserved for future format-specific blueprint selection
) (*models.Blueprint, error) {
	if bp, ok := s.snapshotBlueprint(ctx, endpoint); ok {
		return bp, nil
	}
	if s.decodesOnlySnapshots(ctx, tenantID) {
		s.log.ErrorContext(ctx, LogBlueprintSnapshotOnlyNoSnapshot,
			logger.FieldEndpointEui, endpoint.EUI.String(), logger.FieldTenantIDSnake, tenantID)
		return nil, nil
	}
	bp, err := s.catalogBlueprint(ctx, tenantID, endpoint)
	if err != nil {
		return nil, nil
	}
	return bp, nil
}

// ResolveIgnoringMode finds the blueprint ResolveBlueprintForEndpoint would
// decode with if the tenant's switch were off. Unlike it, a failed catalog read
// is returned rather than taken for "nothing found".
func (s *ResolverService) ResolveIgnoringMode(ctx context.Context, tenantID int64, endpoint *models.EndPoint) (*models.Blueprint, error) {
	if bp, ok := s.snapshotBlueprint(ctx, endpoint); ok {
		return bp, nil
	}
	return s.catalogBlueprint(ctx, tenantID, endpoint)
}

// catalogBlueprint applies the TypeEUI precedence rule: an endpoint with a
// device model decodes with the model's default blueprint, which keeps a loose
// type EUI from overriding the model; otherwise with the blueprint of its type
// EUI. Nothing found is (nil, nil); a failed read is logged and returned.
func (s *ResolverService) catalogBlueprint(ctx context.Context, tenantID int64, endpoint *models.EndPoint) (*models.Blueprint, error) {
	if endpoint.DeviceModelID != nil {
		return s.modelDefaultBlueprint(ctx, tenantID, endpoint)
	}
	if endpoint.TypeEUI != nil {
		return s.blueprintByTypeEUI(ctx, tenantID, endpoint.TypeEUI[:])
	}
	s.log.DebugContext(ctx, LogBlueprintNoTypeEUIOrModel,
		logger.FieldEndpointEui, endpoint.EUI.String())
	return nil, nil
}

func (s *ResolverService) modelDefaultBlueprint(ctx context.Context, tenantID int64, endpoint *models.EndPoint) (*models.Blueprint, error) {
	modelID := *endpoint.DeviceModelID
	bp, err := s.blueprintRepo.GetDefaultForModel(ctx, tenantID, modelID)
	if err != nil && !isNotFoundError(err) {
		s.log.DebugContext(ctx, LogBlueprintModelDefaultError,
			logger.FieldDeviceModelID, modelID, logger.FieldError, err)
		return nil, err
	}
	if err != nil || bp == nil {
		s.log.DebugContext(ctx, LogBlueprintModelNoDefault,
			logger.FieldEndpointEui, endpoint.EUI.String(), logger.FieldDeviceModelID, modelID)
		return nil, nil
	}
	s.log.DebugContext(ctx, LogBlueprintResolvedFromModel,
		logger.FieldEndpointEui, endpoint.EUI.String(),
		logger.FieldDeviceModelID, modelID, logger.FieldBlueprintID, bp.ID)
	return bp, nil
}

func (s *ResolverService) blueprintByTypeEUI(ctx context.Context, tenantID int64, typeEUI []byte) (*models.Blueprint, error) {
	bp, err := s.blueprintRepo.GetByTypeEUI(ctx, tenantID, typeEUI)
	if err == nil && bp != nil {
		s.log.DebugContext(ctx, LogBlueprintFoundByTypeEUI,
			logger.FieldBlueprintID, bp.ID,
			logger.FieldTypeEui, mioty.FormatEUIBytes(typeEUI),
			logger.FieldVersion, bp.Version)
		return bp, nil
	}
	s.log.DebugContext(ctx, LogBlueprintNotFoundForTypeEUI, logger.FieldTypeEui, mioty.FormatEUIBytes(typeEUI))
	if err != nil && !isNotFoundError(err) {
		s.log.WarnContext(ctx, LogBlueprintTypeEUILookupError,
			logger.FieldTypeEui, mioty.FormatEUIBytes(typeEUI), logger.FieldError, err)
		return nil, err
	}
	return nil, nil
}

// snapshotBlueprint reads the endpoint's pinned snapshot, the highest-priority
// decode source; ok is false when there is none or it is malformed.
func (s *ResolverService) snapshotBlueprint(ctx context.Context, endpoint *models.EndPoint) (*models.Blueprint, bool) {
	if len(endpoint.BlueprintSnapshot) == 0 {
		return nil, false
	}
	var snap models.BlueprintSnapshot
	if err := json.Unmarshal(endpoint.BlueprintSnapshot, &snap); err != nil {
		s.log.WarnContext(ctx, LogBlueprintSnapshotParseFailed,
			logger.FieldEndpointEui, endpoint.EUI.String(), logger.FieldError, err)
		return nil, false
	}
	bp, err := snap.ToBlueprint()
	if err != nil {
		s.log.WarnContext(ctx, LogBlueprintSnapshotInvalid,
			logger.FieldEndpointEui, endpoint.EUI.String(), logger.FieldError, err)
		return nil, false
	}
	s.log.DebugContext(ctx, LogBlueprintResolvedFromSnapshot,
		logger.FieldEndpointEui, endpoint.EUI.String(),
		logger.FieldBlueprintID, bp.ID, logger.FieldVersion, bp.Version)
	return bp, true
}

// decodesOnlySnapshots reports the owner tenant's switch; an unreadable
// switch counts as on, so a failed read never decodes from the catalog.
func (s *ResolverService) decodesOnlySnapshots(ctx context.Context, tenantID int64) bool {
	on, err := s.modes.IsSnapshotOnly(ctx, tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, LogBlueprintSnapshotOnlyReadFailed,
			logger.FieldTenantIDSnake, tenantID, logger.FieldError, err)
		return true
	}
	return on
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
