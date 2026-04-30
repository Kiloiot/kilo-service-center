// Package blueprint provides blueprint resolution for MIOTY payload decoding.
package blueprint

import (
	"context"
	"encoding/hex"
	"encoding/json"

	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
)

// ResolverService implements the BlueprintResolver interface.
// It finds blueprints for endpoints based on Type EUI and device model associations.
type ResolverService struct {
	log             logger.Logger
	blueprintRepo   interfaces.BlueprintRepository
	deviceModelRepo interfaces.DeviceModelRepository
	endpointRepo    interfaces.EndpointRepository
}

// NewResolverService creates a new ResolverService instance.
func NewResolverService(
	log logger.Logger,
	blueprintRepo interfaces.BlueprintRepository,
	deviceModelRepo interfaces.DeviceModelRepository,
	endpointRepo interfaces.EndpointRepository,
) *ResolverService {
	return &ResolverService{
		log:             log.WithField("component", "blueprint-resolver"),
		blueprintRepo:   blueprintRepo,
		deviceModelRepo: deviceModelRepo,
		endpointRepo:    endpointRepo,
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
		s.log.Debug("No Type EUI provided, skipping blueprint resolution")
		return nil, nil
	}

	// Try to find blueprint by Type EUI directly
	bp, err := s.blueprintRepo.GetByTypeEUI(ctx, tenantID, typeEUI)
	if err == nil && bp != nil {
		s.log.Debug("Found blueprint by Type EUI",
			"blueprint_id", bp.ID,
			"type_eui", hex.EncodeToString(typeEUI),
			"version", bp.Version)
		return bp, nil
	}

	// Log if error is not "not found"
	if err != nil && !isNotFoundError(err) {
		s.log.Warn("Error looking up blueprint by Type EUI",
			"type_eui", hex.EncodeToString(typeEUI),
			"error", err)
	}

	s.log.Debug("No blueprint found for Type EUI",
		"type_eui", hex.EncodeToString(typeEUI))
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
	// Check if endpoint has a device model assigned
	if endpoint.DeviceModelID != nil {
		// TypeEUI precedence rule: Must get Type EUI from model's default blueprint
		// This prevents "loose" TypeEUI when a device model is assigned

		// DeviceModelID is now *uuid.UUID, use directly
		modelID := *endpoint.DeviceModelID

		// Get the default blueprint for this model
		bp, err := s.blueprintRepo.GetDefaultForModel(ctx, tenantID, modelID)
		if err != nil {
			s.log.Debug("Error getting default blueprint for model",
				"device_model_id", *endpoint.DeviceModelID,
				"error", err)
			return nil, nil
		}

		if bp != nil {
			s.log.Debug("Resolved blueprint from device model",
				"endpoint_eui", endpoint.EUI.String(),
				"device_model_id", *endpoint.DeviceModelID,
				"blueprint_id", bp.ID)
			return bp, nil
		}

		// Model has no default blueprint - return nil (decode will be skipped)
		s.log.Debug("Device model has no default blueprint",
			"endpoint_eui", endpoint.EUI.String(),
			"device_model_id", *endpoint.DeviceModelID)
		return nil, nil
	}

	// No device model - use endpoint's Type EUI for direct lookup
	if endpoint.TypeEUI != nil {
		typeEUIBytes := endpoint.TypeEUI[:]
		return s.ResolveBlueprint(ctx, tenantID, typeEUIBytes, formatID)
	}

	s.log.Debug("Endpoint has no Type EUI or device model",
		"endpoint_eui", endpoint.EUI.String())
	return nil, nil
}

// GetEndpointCalibration retrieves calibration data for an endpoint.
// Returns an empty map if no calibration data is available.
func (s *ResolverService) GetEndpointCalibration(
	_ context.Context, // ctx reserved for future async operations
	_ int64, // tenantID reserved for future multi-tenant calibration stores
	endpoint *models.EndPoint,
) map[string]interface{} {
	if len(endpoint.CalibrationData) == 0 {
		return make(map[string]interface{})
	}

	// Parse JSON calibration data
	var calibration map[string]interface{}
	if err := parseJSON(endpoint.CalibrationData, &calibration); err != nil {
		s.log.Warn("Failed to parse endpoint calibration data",
			"endpoint_eui", endpoint.EUI.String(),
			"error", err)
		return make(map[string]interface{})
	}

	return calibration
}

// isNotFoundError checks if an error indicates a "not found" condition.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check for common "not found" patterns
	errStr := err.Error()
	return errStr == "not found" ||
		errStr == "record not found" ||
		errStr == "no rows in result set"
}

// parseJSON parses JSON data into the target interface.
func parseJSON(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}
