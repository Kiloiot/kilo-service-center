// Package bssci provides BSSCI v1.0.0 message normalization per §2.4
//
// Implements BSSCI §2.4 requirements for:
// - Forward compatibility: silently ignore unknown fields
// - Optional field defaults: apply spec-mandated defaults when fields absent
// - Mandatory field validation: ensure required fields present and correctly typed
// - Conditional rules: enforce context-dependent field requirements
//
// References:
// - BSSCI §2.4: Message Interpretation
// - All log calls use *Context methods for structured context propagation
package bssci

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// ============================================================================
// Normalization Sentinel Errors
// ============================================================================

// Normalization sentinel errors for errors.Is() detection
// These errors reuse catalog token constants to ensure alignment if token strings change
var (
	// ErrMandatoryFieldMissing indicates a required field was absent in the payload
	ErrMandatoryFieldMissing = errors.New(errMandatoryFieldMissing)

	// ErrInvalidFieldType indicates a field had wrong type (e.g., string instead of int64)
	ErrInvalidFieldType = errors.New(errInvalidFieldType)

	// ErrConditionalRuleFailed indicates a context-dependent requirement was violated
	ErrConditionalRuleFailed = errors.New(errConditionalRuleFailed)

	// ErrResponseExpRequiresDlOpen indicates responseExp=true requires dlOpen=true per BSSCI §5.10.1
	ErrResponseExpRequiresDlOpen = errors.New(errResponseExpRequiresDlOpen)
)

// ============================================================================
// Normalization Function
// ============================================================================

// normalizePayload applies BSSCI §2.4 message interpretation rules to incoming payloads
//
// This function MUST be called before handler dispatch to ensure:
// 1. All mandatory fields are present and correctly typed
// 2. Missing optional fields receive spec-mandated defaults
// 3. Unknown fields are logged at WARN level and dropped (forward compatibility)
// 4. Conditional field requirements are enforced
//
// Parameters:
//   - ctx: Request context for context-aware logging
//   - log: Logger instance for structured logging via *Context methods
//   - command: MIOTY command mnemonic (e.g., "statusRsp", "att", "det")
//   - data: Raw payload map decoded from MessagePack/JSON
//
// Returns:
//   - Normalized payload with guaranteed field presence and types
//   - Error if normalization fails (mapped to errors_catalog.go tokens by caller)
//
// CRITICAL: All log calls MUST use *Context methods (WarnContext, ErrorContext, InfoContext)
// Never use non-context methods (Warn, Error, Info).
//
// Example normalization flow:
//
//	// Before normalization (raw payload)
//	data := map[string]interface{}{
//	    "code": 0,
//	    "message": "ok",
//	    "time": 1699564800000000000,
//	    "unknownField": "future_feature",  // Unknown field
//	    // dutyCycle absent (optional field)
//	}
//
//	// After normalization
//	normalized := map[string]interface{}{
//	    "code": int64(0),          // Type-coerced to int64
//	    "message": "ok",           // Validated string
//	    "time": int64(1699564800000000000),
//	    "dutyCycle": nil,          // Default applied (nil for pointer types)
//	    // unknownField dropped + WARN log emitted
//	}
func normalizePayload(ctx context.Context, log logger.Logger, command string, data map[string]interface{}) (map[string]interface{}, error) {
	// Look up message specification in registry
	spec, exists := messageRegistry[command]
	if !exists {
		// Command not registered - pass through without normalization
		// This allows handlers for unregistered commands to continue working
		// as they currently do (backward compatibility during incremental rollout)
		return data, nil
	}

	normalized := make(map[string]interface{})
	seenFields := make(map[string]bool)

	// ========================================================================
	// Step 1: Validate mandatory fields and copy to normalized payload
	// ========================================================================

	for _, fieldSpec := range spec.MandatoryFields {
		value, exists := data[fieldSpec.Name]
		if !exists {
			// Mandatory field missing - fail normalization
			// Sentinel error enables caller to use errors.Is() for token mapping
			return nil, fmt.Errorf("%w: %s (spec: %s)", ErrMandatoryFieldMissing, fieldSpec.Name, fieldSpec.SpecRef)
		}

		// Type-coerce and validate field
		coerced, err := coerceFieldType(value, fieldSpec)
		if err != nil {
			// Invalid field type - fail normalization
			// Sentinel error enables caller to use errors.Is() for token mapping
			return nil, fmt.Errorf("%w for field %s: %v (spec: %s)", ErrInvalidFieldType, fieldSpec.Name, err, fieldSpec.SpecRef)
		}

		// Custom validation if specified
		if fieldSpec.Validator != nil && !fieldSpec.Validator(coerced) {
			return nil, fmt.Errorf("validation failed for field %s (spec: %s)", fieldSpec.Name, fieldSpec.SpecRef)
		}

		normalized[fieldSpec.Name] = coerced
		seenFields[fieldSpec.Name] = true
	}

	// ========================================================================
	// Step 2: Apply defaults to optional fields when absent
	// ========================================================================

	for _, fieldSpec := range spec.OptionalFields {
		if value, exists := data[fieldSpec.Name]; exists {
			// Optional field present - type-coerce and validate
			coerced, err := coerceFieldType(value, fieldSpec)
			if err != nil {
				// Sentinel error enables caller to use errors.Is() for token mapping
				return nil, fmt.Errorf("%w for optional field %s: %v (spec: %s)", ErrInvalidFieldType, fieldSpec.Name, err, fieldSpec.SpecRef)
			}

			// Custom validation if specified
			if fieldSpec.Validator != nil && !fieldSpec.Validator(coerced) {
				return nil, fmt.Errorf("validation failed for optional field %s (spec: %s)", fieldSpec.Name, fieldSpec.SpecRef)
			}

			normalized[fieldSpec.Name] = coerced
		} else {
			// Optional field absent
			// Special case: Detach eqSnr defaults to snr value when absent (BSSCI §5.7.1)
			if command == "det" && fieldSpec.Name == "eqSnr" {
				// Copy snr value as eqSnr default
				if snr, ok := normalized["snr"]; ok {
					normalized["eqSnr"] = snr
				}
			} else {
				// Always add optional field to map (even with nil value)
				// This allows callers to distinguish "present but nil" from "never sent"
				normalized[fieldSpec.Name] = fieldSpec.DefaultValue
			}
		}

		seenFields[fieldSpec.Name] = true
	}

	// ========================================================================
	// Step 3: Detect and log unknown fields (forward compatibility per §2.4)
	// ========================================================================

	// BaseMessage fields (present in ALL messages) - not unknown
	baseFields := map[string]bool{
		"command": true,
		"opId":    true,
	}

	for fieldName := range data {
		if !seenFields[fieldName] && !baseFields[fieldName] {
			// Unknown field detected - log at WARN level and drop
			// Use WarnContext for context-aware logging
			log.WarnContext(ctx, "Unknown field in message - dropping for forward compatibility",
				"command", command,
				"field", fieldName,
				"specSection", spec.SpecSection,
				"bssciRef", "§2.4",
			)
		}
	}

	// ========================================================================
	// Step 4: Enforce conditional field requirements
	// ========================================================================

	if spec.ConditionalRules != nil {
		for _, rule := range spec.ConditionalRules {
			if rule.Condition(normalized) {
				// Condition met - validate required/forbidden fields
				for _, requiredField := range rule.Required {
					// Check both key existence AND non-nil value
					// Optional fields are added to normalized with nil when absent,
					// so we must verify the value is actually present, not just the key
					if value, exists := normalized[requiredField]; !exists || value == nil {
						// Conditional field requirement violated
						// Sentinel error enables caller to use errors.Is() for token mapping
						return nil, fmt.Errorf("%w: %s (%s)", ErrConditionalRuleFailed, requiredField, rule.ErrorMsg)
					}
				}

				for _, forbiddenField := range rule.Forbidden {
					// Check if field is present with non-nil value
					// Optional fields with nil value are considered absent
					if value, exists := normalized[forbiddenField]; exists && value != nil {
						// Sentinel error enables caller to use errors.Is() for token mapping
						return nil, fmt.Errorf("%w: %s must not be present (%s)", ErrConditionalRuleFailed, forbiddenField, rule.ErrorMsg)
					}
				}
			}
		}
	}

	// ========================================================================
	// Step 5: Special-case value validation for ULData responseExp constraint
	// ========================================================================
	// BSSCI/MIOTY Radio Protocol §3.6.5.1: When responseExp=true, dlOpen must be true
	// ConditionalRule above ensures dlOpen is *present*; this validates its *value*
	if command == "ulData" {
		if responseExp, ok := normalized["responseExp"].(bool); ok && responseExp {
			if dlOpen, ok := normalized["dlOpen"].(bool); !ok || !dlOpen {
				return nil, fmt.Errorf("%w (MIOTY Radio Protocol §3.6.5.1)", ErrResponseExpRequiresDlOpen)
			}
		}
	}

	return normalized, nil
}

// ============================================================================
// Type Coercion Helpers
// ============================================================================

// coerceFieldType converts a raw interface{} value to the expected field type
// Reuses existing type coercion patterns from server.go helpers
func coerceFieldType(value interface{}, fieldSpec FieldSpec) (interface{}, error) {
	switch fieldSpec.Type {
	case TypeInt64:
		return coerceInt64(value)

	case TypeUint64:
		return coerceUint64(value)

	case TypeUint32:
		return coerceUint32(value)

	case TypeUint16:
		return coerceUint16(value)

	case TypeFloat64:
		return coerceFloat64(value)

	case TypeString:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		return str, nil

	case TypeBool:
		b, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool, got %T", value)
		}
		return b, nil

	case TypeBytes:
		// Accept []byte, []interface{} (numeric array), or [N]byte
		switch v := value.(type) {
		case []byte:
			if fieldSpec.ByteLength > 0 && len(v) != fieldSpec.ByteLength {
				return nil, fmt.Errorf("expected %d bytes, got %d", fieldSpec.ByteLength, len(v))
			}
			return v, nil
		case []interface{}:
			// Convert numeric array to []byte
			// MessagePack can produce any of 12 numeric types (uint8, int8, uint16, int16, uint32, int32, uint64, int64, int, uint, float32, float64)
			bytes := make([]byte, len(v))
			for i, elem := range v {
				b, err := numericToByte(elem)
				if err != nil {
					return nil, fmt.Errorf("array element at index %d: %w", i, err)
				}
				bytes[i] = b
			}
			if fieldSpec.ByteLength > 0 && len(bytes) != fieldSpec.ByteLength {
				return nil, fmt.Errorf("expected %d bytes, got %d", fieldSpec.ByteLength, len(bytes))
			}
			return bytes, nil
		default:
			return nil, fmt.Errorf("expected byte array, got %T", value)
		}

	case TypeArray, TypeGeoLocation, TypeSubpackets, TypeStruct, TypeInterface:
		// Pass through complex types without validation
		// Handlers will perform structure-specific validation
		return value, nil

	default:
		return nil, fmt.Errorf("unsupported field type: %s", fieldSpec.Type)
	}
}

// coerceInt64 converts various numeric types to int64
// Mirrors server.go:getNumericField behavior
func coerceInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case uint64:
		//nolint:gosec // G115: Protocol EUIs/opIds are < 2^48, safe to cast to int64
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case float64:
		// JSON numbers come as float64
		return int64(v), nil
	case float32:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", value)
	}
}

// coerceUint64 converts various numeric types to uint64
func coerceUint64(value interface{}) (uint64, error) {
	switch v := value.(type) {
	case uint64:
		return v, nil
	case uint32:
		// uint32 always fits in uint64
		return uint64(v), nil
	case uint16:
		// uint16 always fits in uint64
		return uint64(v), nil
	case uint8:
		// uint8 always fits in uint64
		return uint64(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("negative value cannot be coerced to uint64: %d", v)
		}
		return uint64(v), nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("negative value cannot be coerced to uint64: %d", v)
		}
		return uint64(v), nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("negative value cannot be coerced to uint64: %f", v)
		}
		return uint64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to uint64", value)
	}
}

// coerceUint32 converts various numeric types to uint32
func coerceUint32(value interface{}) (uint32, error) {
	switch v := value.(type) {
	case uint32:
		return v, nil
	case uint16:
		// uint16 always fits in uint32 (0-65535)
		return uint32(v), nil
	case uint64:
		if v > uint64(math.MaxUint32) {
			return 0, fmt.Errorf("value %d out of range for uint32", v)
		}
		return uint32(v), nil
	case uint8:
		// uint8 always fits in uint32 (0-255)
		return uint32(v), nil
	case int64:
		if v < 0 || v > 4294967295 {
			return 0, fmt.Errorf("value %d out of range for uint32", v)
		}
		return uint32(v), nil
	case int:
		if v < 0 || v > 4294967295 {
			return 0, fmt.Errorf("value %d out of range for uint32", v)
		}
		return uint32(v), nil
	case float64:
		if v < 0 || v > 4294967295 {
			return 0, fmt.Errorf("value %f out of range for uint32", v)
		}
		return uint32(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to uint32", value)
	}
}

// coerceUint16 converts various numeric types to uint16
func coerceUint16(value interface{}) (uint16, error) {
	switch v := value.(type) {
	case uint16:
		return v, nil
	case uint64:
		if v > uint64(math.MaxUint16) {
			return 0, fmt.Errorf("value %d out of range for uint16", v)
		}
		return uint16(v), nil
	case uint32:
		if v > uint32(math.MaxUint16) {
			return 0, fmt.Errorf("value %d out of range for uint16", v)
		}
		return uint16(v), nil
	case uint8:
		return uint16(v), nil
	case int64:
		if v < 0 || v > 65535 {
			return 0, fmt.Errorf("value %d out of range for uint16", v)
		}
		return uint16(v), nil
	case int:
		if v < 0 || v > 65535 {
			return 0, fmt.Errorf("value %d out of range for uint16", v)
		}
		return uint16(v), nil
	case float64:
		if v < 0 || v > 65535 {
			return 0, fmt.Errorf("value %f out of range for uint16", v)
		}
		return uint16(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to uint16", value)
	}
}

// coerceFloat64 converts various numeric types to float64
// Mirrors server.go:getFloatFieldValidated behavior
func coerceFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// numericToByte converts any numeric type to byte (uint8)
// Handles all 12 numeric types that MessagePack can produce
// Validates range 0-255 for types larger than byte
func numericToByte(value interface{}) (byte, error) {
	switch v := value.(type) {
	case uint8:
		return v, nil
	case int8:
		if v < 0 {
			return 0, fmt.Errorf("negative value %d cannot be converted to byte", v)
		}
		return byte(v), nil
	case uint16:
		if v > 255 {
			return 0, fmt.Errorf("value %d out of byte range (0-255)", v)
		}
		return byte(v), nil
	case int16:
		if v < 0 || v > 255 {
			return 0, fmt.Errorf("value %d out of byte range (0-255)", v)
		}
		return byte(v), nil
	case uint32:
		if v > 255 {
			return 0, fmt.Errorf("value %d out of byte range (0-255)", v)
		}
		return byte(v), nil
	case int32:
		if v < 0 || v > 255 {
			return 0, fmt.Errorf("value %d out of byte range (0-255)", v)
		}
		return byte(v), nil
	case uint64:
		if v > 255 {
			return 0, fmt.Errorf("value %d out of byte range (0-255)", v)
		}
		return byte(v), nil
	case int64:
		if v < 0 || v > 255 {
			return 0, fmt.Errorf("value %d out of byte range (0-255)", v)
		}
		return byte(v), nil
	case int:
		if v < 0 || v > 255 {
			return 0, fmt.Errorf("value %d out of byte range (0-255)", v)
		}
		return byte(v), nil
	case uint:
		if v > 255 {
			return 0, fmt.Errorf("value %d out of byte range (0-255)", v)
		}
		return byte(v), nil
	case float32:
		// Reject fractional values - bytes must be whole numbers
		if v != float32(int64(v)) {
			return 0, fmt.Errorf("value %f out of byte range (0-255)", v)
		}
		if v < 0 || v > 255 {
			return 0, fmt.Errorf("value %f out of byte range (0-255)", v)
		}
		return byte(v), nil
	case float64:
		// Reject fractional values - bytes must be whole numbers
		if v != float64(int64(v)) {
			return 0, fmt.Errorf("value %f out of byte range (0-255)", v)
		}
		if v < 0 || v > 255 {
			return 0, fmt.Errorf("value %f out of byte range (0-255)", v)
		}
		return byte(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to byte", value)
	}
}
