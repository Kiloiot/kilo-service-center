package blueprints

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// DecodePreview runs the blueprint decoder on a payload for preview.
func (s *Service) DecodePreview(ctx context.Context, blueprintID uuid.UUID, payload []byte, formatID uint8) (*grpcservices.DecodePreviewResult, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	bp, err := s.blueprintRepo.GetByID(ctx, tenantID, blueprintID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrBlueprintNotFound
		}
		return nil, fmt.Errorf("%s: %w", errOpGetBlueprint, err)
	}

	return s.runDecodePreview(ctx, bp, payload, formatID)
}

// DecodePreviewInline previews decoding against an unsaved inline spec (no catalog fetch).
func (s *Service) DecodePreviewInline(ctx context.Context, specJSON, payload []byte, formatID uint8) (*grpcservices.DecodePreviewResult, error) {
	return s.runDecodePreview(ctx, &models.Blueprint{SpecJSON: specJSON}, payload, formatID)
}

func (s *Service) runDecodePreview(ctx context.Context, bp *models.Blueprint, payload []byte, formatID uint8) (*grpcservices.DecodePreviewResult, error) {
	if s.decoder == nil {
		return nil, ErrDecoderNotConfigured
	}

	result, err := s.decoder.Decode(ctx, bp, payload, formatID, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errOpDecode, err)
	}

	if !result.Success {
		return &grpcservices.DecodePreviewResult{
			Success:          false,
			ErrorCode:        result.ErrorCode,
			ErrorDetail:      result.ErrorDetail,
			FormatID:         formatID,
			BlueprintVersion: bp.Version,
		}, nil
	}

	decoded, err := json.Marshal(result.DecodedData)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errOpMarshalDecodedData, err)
	}

	return &grpcservices.DecodePreviewResult{
		Success:          true,
		DecodedPayload:   decoded,
		FormatID:         formatID,
		BlueprintVersion: bp.Version,
	}, nil
}

// prepareBlueprintContent prepares the blueprint content for submission.
// Returns base64-encoded JSON content as a string (API expects base64-encoded content).
// Includes enriched metadata: manufacturer name, device model name and code.
func (s *Service) prepareBlueprintContent(bp *models.Blueprint, manufacturerName, deviceModelName, deviceModelCode string) (string, error) {
	// Create submission format with enriched metadata
	content := map[string]interface{}{
		"id":                bp.ID.String(),
		"device_model_id":   bp.DeviceModelID.String(),
		"version":           bp.Version,
		"type_eui":          mioty.FormatEUIBytes(bp.TypeEUI),
		"spec":              json.RawMessage(bp.SpecJSON),
		"is_default":        bp.IsDefault,
		"manufacturer_name": manufacturerName,
		"device_model_name": deviceModelName,
		"device_model_code": deviceModelCode,
		"created_at":        bp.CreatedAt,
		"updated_at":        bp.UpdatedAt,
	}

	jsonContent, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return "", fmt.Errorf("%s: %w", errOpMarshalBlueprint, err)
	}

	// Return base64-encoded string (not []byte to avoid double-encoding by JSON marshal)
	return base64.StdEncoding.EncodeToString(jsonContent), nil
}
