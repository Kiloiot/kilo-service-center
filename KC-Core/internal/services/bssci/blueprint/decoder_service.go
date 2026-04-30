// Package blueprint provides the MIOTY Application Layer payload decoder service.
// This package is internal to KC-Core and implements the BlueprintDecoder interface
// defined in pkg/bssci/contracts.go per Dependency Inversion principle.
package blueprint

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kilocenter/KC-Core/pkg/blueprint"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/models"
)

// DecoderService implements the BlueprintDecoder interface for payload decoding.
// It parses blueprint specifications and decodes payloads according to the MIOTY
// Application Layer Specification.
type DecoderService struct {
	log       logger.Logger
	evaluator *ExpressionEvaluator
}

// NewDecoderService creates a new DecoderService instance.
func NewDecoderService(log logger.Logger) *DecoderService {
	return &DecoderService{
		log:       log.WithField("component", "blueprint-decoder"),
		evaluator: NewExpressionEvaluator(),
	}
}

// Decode decodes raw payload using blueprint spec for given format ID.
// Implements the BlueprintDecoder interface.
func (s *DecoderService) Decode(
	_ context.Context,
	bp *models.Blueprint,
	userData []byte,
	formatID uint8,
	calibration map[string]interface{},
) (*blueprint.DecodeResult, error) {
	// Parse the blueprint specification
	spec, err := s.parseBlueprint(bp.SpecJSON)
	if err != nil {
		s.log.Warn("Failed to parse blueprint specification",
			"blueprint_id", bp.ID,
			"error", err)
		return blueprint.NewDecodeError(blueprint.ErrInvalidBlueprintJSON, err.Error()), nil
	}

	// Validate the specification
	if err := spec.Validate(); err != nil {
		s.log.Warn("Blueprint validation failed",
			"blueprint_id", bp.ID,
			"error", err)
		if ve, ok := err.(*blueprint.ValidationError); ok {
			return blueprint.NewDecodeError(ve.Token, ve.Detail), nil
		}
		return blueprint.NewDecodeError(blueprint.ErrInvalidBlueprintJSON, err.Error()), nil
	}

	// Find the format definition for the given format ID
	format := spec.GetUplinkFormat(formatID)
	if format == nil {
		s.log.Debug("Format ID not found in blueprint",
			"blueprint_id", bp.ID,
			"format_id", formatID)
		return blueprint.NewDecodeError(
			blueprint.ErrFormatIDNotFound,
			fmt.Sprintf("format_id=%d not found in blueprint", formatID),
		), nil
	}

	// Check payload length
	requiredBits := s.calculateRequiredBits(format)
	requiredBytes := (requiredBits + 7) / 8
	if len(userData) < requiredBytes {
		s.log.Debug("Payload too short for format",
			"blueprint_id", bp.ID,
			"format_id", formatID,
			"required_bytes", requiredBytes,
			"actual_bytes", len(userData))
		return blueprint.NewDecodeError(
			blueprint.ErrPayloadTooShort,
			fmt.Sprintf("need %d bytes, got %d", requiredBytes, len(userData)),
		), nil
	}

	// Merge blueprint-level calibration defaults with per-device overrides
	mergedCalibration := mergeCalibration(spec.Calibration, calibration)

	// Decode the payload
	decodedData, err := s.decodePayload(userData, format, mergedCalibration)
	if err != nil {
		s.log.Debug("Payload decode failed",
			"blueprint_id", bp.ID,
			"format_id", formatID,
			"error", err)
		if de, ok := err.(*blueprint.DecodeError); ok {
			return blueprint.NewDecodeError(de.Token, de.Detail), nil
		}
		return blueprint.NewDecodeError(blueprint.ErrInternalDecodePanic, err.Error()), nil
	}

	s.log.Debug("Payload decoded successfully",
		"blueprint_id", bp.ID,
		"format_id", formatID,
		"field_count", len(decodedData))

	return blueprint.NewDecodeResult(decodedData, formatID, bp.Version), nil
}

// parseBlueprint parses the blueprint JSON specification.
func (s *DecoderService) parseBlueprint(specJSON json.RawMessage) (*blueprint.Spec, error) {
	return blueprint.ParseSpec(specJSON)
}

// calculateRequiredBits calculates the total bits required for a payload format.
func (s *DecoderService) calculateRequiredBits(format *blueprint.PayloadFormat) int {
	maxBitOffset := 0
	for _, component := range format.Components {
		endBit := 0
		if component.Offset != nil {
			endBit = *component.Offset + component.Size
		} else {
			// Sequential layout - calculate from previous components
			endBit = maxBitOffset + component.Size
		}
		if endBit > maxBitOffset {
			maxBitOffset = endBit
		}
	}
	return maxBitOffset
}

// decodePayload decodes the payload bytes according to the format definition.
func (s *DecoderService) decodePayload(
	userData []byte,
	format *blueprint.PayloadFormat,
	calibration map[string]interface{},
) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	currentBitOffset := 0

	for _, component := range format.Components {
		// Determine bit offset
		bitOffset := currentBitOffset
		if component.Offset != nil {
			bitOffset = *component.Offset
		}

		// Check condition if specified
		if component.Condition != "" {
			conditionMet, err := s.evaluator.EvaluateCondition(component.Condition, result, calibration)
			if err != nil {
				return nil, &blueprint.DecodeError{
					Token:     blueprint.ErrConditionEvalFailed,
					Component: component.Name,
					Detail:    err.Error(),
				}
			}
			if !conditionMet {
				// Skip this component
				currentBitOffset = bitOffset + component.Size
				continue
			}
		}

		// Extract value from payload (pass format-level LittleEndian per section 2.2.10)
		value, err := s.extractValue(userData, bitOffset, component.Size, component.Type, format.LittleEndian)
		if err != nil {
			return nil, &blueprint.DecodeError{
				Token:     blueprint.ErrBitExtractionFailed,
				Component: component.Name,
				Detail:    err.Error(),
			}
		}

		// Apply scale and bias for numeric types
		if numVal, ok := toFloat64(value); ok {
			if component.Scale != 0 {
				numVal = numVal * component.Scale
			}
			if component.Bias != 0 {
				numVal = numVal + component.Bias
			}
			value = numVal
		}

		// Apply calibration if referenced
		if component.Calibration != "" {
			calVal, err := s.evaluator.GetCalibrationValue(component.Calibration, calibration)
			if err != nil {
				return nil, &blueprint.DecodeError{
					Token:     blueprint.ErrCalibrationNotFound,
					Component: component.Name,
					Detail:    err.Error(),
				}
			}
			// Apply calibration as multiplier
			if numVal, ok := toFloat64(value); ok {
				if calFloat, ok := toFloat64(calVal); ok {
					value = numVal * calFloat
				}
			}
		}

		// Evaluate func expression if specified
		if component.Func != "" {
			funcResult, err := s.evaluator.EvaluateFunc(component.Func, value, result, calibration)
			if err != nil {
				return nil, &blueprint.DecodeError{
					Token:     blueprint.ErrExpressionEvalFailed,
					Component: component.Name,
					Detail:    err.Error(),
				}
			}
			value = funcResult
		}

		// Map enum value if specified
		if len(component.Enum) > 0 {
			if strKey := fmt.Sprintf("%v", value); strKey != "" {
				if enumVal, ok := component.Enum[strKey]; ok {
					value = enumVal
				}
			}
		}

		result[component.Name] = value
		currentBitOffset = bitOffset + component.Size
	}

	return result, nil
}

// extractValue extracts a value from the payload at the given bit offset and size.
// littleEndian controls byte order for multi-byte numeric types (per section 2.2.10).
func (s *DecoderService) extractValue(data []byte, bitOffset, bitSize int, fieldType string, littleEndian bool) (interface{}, error) {
	// Validate bounds
	totalBits := len(data) * 8
	if bitOffset < 0 || bitOffset+bitSize > totalBits {
		return nil, fmt.Errorf("bit range [%d:%d] exceeds payload size %d bits", bitOffset, bitOffset+bitSize, totalBits)
	}

	switch fieldType {
	case blueprint.FieldTypeBool:
		byteIndex := bitOffset / 8
		bitIndex := bitOffset % 8
		return (data[byteIndex] & (1 << (7 - bitIndex))) != 0, nil

	case blueprint.FieldTypeUint:
		return s.extractUint(data, bitOffset, bitSize, littleEndian)

	case blueprint.FieldTypeInt:
		uval, err := s.extractUint(data, bitOffset, bitSize, littleEndian)
		if err != nil {
			return nil, err
		}
		// Convert to signed using two's complement
		signBit := uint64(1) << (bitSize - 1)
		if uval&signBit != 0 {
			// Negative value - intentional signed conversion
			//nolint:gosec // G115: intentional signed integer conversion for two's complement
			return int64(uval) - int64(1<<bitSize), nil
		}
		//nolint:gosec // G115: intentional signed integer conversion
		return int64(uval), nil

	case blueprint.FieldTypeFloat:
		if bitSize == 32 {
			uval, err := s.extractUint(data, bitOffset, 32, littleEndian)
			if err != nil {
				return nil, err
			}
			//nolint:gosec // G115: intentional truncation for 32-bit float conversion
			return float64FromBits32(uint32(uval)), nil
		} else if bitSize == 64 {
			uval, err := s.extractUint(data, bitOffset, 64, littleEndian)
			if err != nil {
				return nil, err
			}
			return float64FromBits64(uval), nil
		}
		// For non-standard float sizes, extract as uint and convert
		uval, err := s.extractUint(data, bitOffset, bitSize, littleEndian)
		if err != nil {
			return nil, err
		}
		return float64(uval), nil

	case blueprint.FieldTypeBytes, blueprint.FieldTypeBinary:
		if bitOffset%8 != 0 || bitSize%8 != 0 {
			return nil, fmt.Errorf("bytes type requires byte-aligned offset and size")
		}
		startByte := bitOffset / 8
		numBytes := bitSize / 8
		result := make([]byte, numBytes)
		copy(result, data[startByte:startByte+numBytes])
		return result, nil

	case blueprint.FieldTypeString:
		if bitOffset%8 != 0 || bitSize%8 != 0 {
			return nil, fmt.Errorf("string type requires byte-aligned offset and size")
		}
		startByte := bitOffset / 8
		numBytes := bitSize / 8
		return string(data[startByte : startByte+numBytes]), nil

	case blueprint.FieldTypeEnum:
		// Enum values are stored as uint, mapped later
		return s.extractUint(data, bitOffset, bitSize, littleEndian)

	default:
		return nil, fmt.Errorf("unsupported field type: %s", fieldType)
	}
}

// extractUint extracts an unsigned integer from arbitrary bit positions.
// When littleEndian is true and the value is byte-aligned, bytes are read LSB-first
// per MIOTY Application Layer Spec section 2.2.10.
func (s *DecoderService) extractUint(data []byte, bitOffset, bitSize int, littleEndian bool) (uint64, error) {
	if bitSize > 64 {
		return 0, fmt.Errorf("bit size %d exceeds maximum of 64", bitSize)
	}

	// Little-endian fast path for byte-aligned multi-byte values
	if littleEndian && bitOffset%8 == 0 && bitSize%8 == 0 {
		startByte := bitOffset / 8
		numBytes := bitSize / 8
		var result uint64
		for i := 0; i < numBytes; i++ {
			result |= uint64(data[startByte+i]) << (i * 8)
		}
		return result, nil
	}

	// Big-endian (default) bit-by-bit extraction
	var result uint64
	for i := 0; i < bitSize; i++ {
		byteIndex := (bitOffset + i) / 8
		bitIndex := (bitOffset + i) % 8
		if (data[byteIndex] & (1 << (7 - bitIndex))) != 0 {
			result |= 1 << (bitSize - 1 - i)
		}
	}
	return result, nil
}

// mergeCalibration merges blueprint-level calibration defaults with per-device overrides.
// Device-level values take precedence over blueprint defaults.
func mergeCalibration(blueprintCal, deviceCal map[string]interface{}) map[string]interface{} {
	if len(blueprintCal) == 0 && len(deviceCal) == 0 {
		return nil
	}
	merged := make(map[string]interface{}, len(blueprintCal)+len(deviceCal))
	for k, v := range blueprintCal {
		merged[k] = v
	}
	for k, v := range deviceCal {
		merged[k] = v
	}
	return merged
}

// toFloat64 converts a value to float64 if possible.
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int64:
		return float64(val), true
	case int:
		return float64(val), true
	case uint64:
		return float64(val), true
	default:
		return 0, false
	}
}

// float64FromBits32 converts 32-bit IEEE 754 to float64.
func float64FromBits32(bits uint32) float64 {
	// Extract components
	sign := (bits >> 31) & 1
	exp := (bits >> 23) & 0xFF
	mantissa := bits & 0x7FFFFF

	// Handle special cases
	if exp == 0 && mantissa == 0 {
		// In Go, -0.0 and 0.0 are the same value
		return 0.0
	}
	if exp == 0xFF {
		if mantissa != 0 {
			return 0.0 // NaN -> 0 for simplicity
		}
		if sign == 1 {
			return -1e308 // -Inf approximation
		}
		return 1e308 // +Inf approximation
	}

	// Normal number
	f := float64(mantissa) / float64(1<<23)
	if exp == 0 {
		// Denormalized
		f = f * float64(int64(1)<<(exp-126))
	} else {
		f = (1.0 + f) * float64(int64(1)<<(int(exp)-127))
	}
	if sign == 1 {
		f = -f
	}
	return f
}

// float64FromBits64 converts 64-bit IEEE 754 to float64.
func float64FromBits64(bits uint64) float64 {
	// Use standard library math package via unsafe pointer
	// For safety, we implement a simple conversion
	sign := (bits >> 63) & 1
	exp := (bits >> 52) & 0x7FF
	mantissa := bits & 0xFFFFFFFFFFFFF

	if exp == 0 && mantissa == 0 {
		// In Go, -0.0 and 0.0 are the same value
		return 0.0
	}
	if exp == 0x7FF {
		if mantissa != 0 {
			return 0.0 // NaN -> 0
		}
		if sign == 1 {
			return -1e308
		}
		return 1e308
	}

	f := float64(mantissa) / float64(uint64(1)<<52)
	if exp == 0 {
		f = f * float64(int64(1)<<(int(exp)-1022)) //nolint:gosec // G115: intentional for IEEE 754 math
	} else {
		f = (1.0 + f) * float64(int64(1)<<(int(exp)-1023)) //nolint:gosec // G115: intentional for IEEE 754 math
	}
	if sign == 1 {
		f = -f
	}
	return f
}
