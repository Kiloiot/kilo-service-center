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
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"

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

	// errNumericPrecision marks conversions rejected because a wire float value's
	// magnitude exceeds the exact-integer range of its IEEE 754 representation,
	// so the original integer cannot be recovered without loss
	errNumericPrecision = errors.New("numeric value beyond exact float integer range")
)

// precisionError wraps errNumericPrecision with the offending value for
// errors.Is() detection at the normalization boundary.
func precisionError(value interface{}) error {
	return fmt.Errorf("%w: %v (%T)", errNumericPrecision, value, value)
}

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
			logNumericPrecisionLoss(ctx, log, command, fieldSpec, err)
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
				logNumericPrecisionLoss(ctx, log, command, fieldSpec, err)
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

// logNumericPrecisionLoss reports precision-rejected conversions at the
// normalization boundary, where command and field context are available.
// EUI fields use the dedicated EUI precision log; other numeric fields use
// the generic normalization log.
func logNumericPrecisionLoss(ctx context.Context, log logger.Logger, command string, fieldSpec FieldSpec, err error) {
	if !errors.Is(err, errNumericPrecision) {
		return
	}
	msg := LogBSSCINumericPrecisionLoss
	if fieldSpec.EUI {
		msg = LogBSSCIEUIPrecisionLoss
	}
	log.WarnContext(ctx, msg,
		"command", command,
		"field", fieldSpec.Name,
		"error", err.Error(),
	)
}

// checkExactIntegerFloat rejects NaN, infinities, fractional values, and
// magnitudes beyond the exact-integer range of the source float width
// (2^53 for float64, 2^24 for float32).
func checkExactIntegerFloat(v float64, bound uint64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Errorf("non-finite value %v cannot be coerced to integer", v)
	}
	if v != math.Trunc(v) {
		return fmt.Errorf("fractional value %v cannot be coerced to integer", v)
	}
	if math.Abs(v) > float64(bound) {
		return precisionError(v)
	}
	return nil
}

// parseJSONNumber parses a json.Number into an exact rational value.
// big.Rat.SetString also accepts fractions ("1/2") and non-decimal forms that
// are not legal JSON numbers, so the token is validated as a JSON number first
// (wire input has already passed the decoder; manually constructed json.Number
// values have not).
func parseJSONNumber(n json.Number) (*big.Rat, error) {
	s := string(n)
	if s == "" || !json.Valid([]byte(s)) {
		return nil, fmt.Errorf("invalid JSON number %q", s)
	}
	if c := s[0]; c != '-' && (c < '0' || c > '9') {
		return nil, fmt.Errorf("invalid JSON number %q", s)
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, fmt.Errorf("invalid JSON number %q", s)
	}
	return r, nil
}

// jsonNumberToUint64 converts a json.Number to uint64 exactly. Integral JSON
// forms such as "1", "1.0", and "1e3" are accepted; fractional, negative, and
// out-of-range values are rejected. Values up to math.MaxUint64 survive
// without any float conversion.
func jsonNumberToUint64(n json.Number) (uint64, error) {
	r, err := parseJSONNumber(n)
	if err != nil {
		return 0, err
	}
	if !r.IsInt() {
		return 0, fmt.Errorf("fractional value %s cannot be coerced to uint64", n)
	}
	i := r.Num()
	if i.Sign() < 0 {
		return 0, fmt.Errorf("negative value cannot be coerced to uint64: %s", n)
	}
	if !i.IsUint64() {
		return 0, fmt.Errorf("value %s overflows uint64", n)
	}
	return i.Uint64(), nil
}

// jsonNumberToInt64 converts a json.Number to int64 exactly, accepting
// integral JSON forms and rejecting fractional or out-of-range values.
func jsonNumberToInt64(n json.Number) (int64, error) {
	r, err := parseJSONNumber(n)
	if err != nil {
		return 0, err
	}
	if !r.IsInt() {
		return 0, fmt.Errorf("fractional value %s cannot be coerced to int64", n)
	}
	i := r.Num()
	if !i.IsInt64() {
		return 0, fmt.Errorf("value %s overflows int64", n)
	}
	return i.Int64(), nil
}

// coerceInt64 converts wire numeric values to int64 with exact semantics:
// unsigned overflow, non-integral floats, and float magnitudes beyond the
// exact-integer range are rejected. Negative values are permitted (Service
// Center operation IDs are negative per BSSCI §3.2).
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
		if v > math.MaxInt64 {
			return 0, fmt.Errorf("value %d overflows int64", v)
		}
		return int64(v), nil
	case uint:
		if uint64(v) > math.MaxInt64 {
			return 0, fmt.Errorf("value %d overflows int64", v)
		}
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case float64:
		if err := checkExactIntegerFloat(v, maxExactFloat64Integer); err != nil {
			return 0, err
		}
		return int64(v), nil
	case float32:
		f := float64(v)
		if err := checkExactIntegerFloat(f, maxExactFloat32Integer); err != nil {
			return 0, err
		}
		return int64(f), nil
	case json.Number:
		return jsonNumberToInt64(v)
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", value)
	}
}

// coerceToUnsigned converts wire numeric values to an unsigned integer capped
// at targetMax with exact semantics. Unsigned integer widths are preserved
// exactly (the full EUI-64 range, including values above INT64_MAX, survives a
// MaxUint64 target); negative values, out-of-range magnitudes, non-integral
// floats, and float magnitudes beyond the exact-integer range are rejected.
// typeName names the target width in error messages.
func coerceToUnsigned(value interface{}, targetMax uint64, typeName string) (uint64, error) {
	switch v := value.(type) {
	case uint64:
		return unsignedToTarget(v, targetMax, typeName)
	case uint:
		return unsignedToTarget(uint64(v), targetMax, typeName)
	case uint32:
		return unsignedToTarget(uint64(v), targetMax, typeName)
	case uint16:
		return unsignedToTarget(uint64(v), targetMax, typeName)
	case uint8:
		return unsignedToTarget(uint64(v), targetMax, typeName)
	case int64:
		return signedToTarget(v, math.MaxInt64, targetMax, typeName)
	case int:
		return signedToTarget(int64(v), math.MaxInt64, targetMax, typeName)
	case int32:
		return signedToTarget(int64(v), math.MaxInt32, targetMax, typeName)
	case int16:
		return signedToTarget(int64(v), math.MaxInt16, targetMax, typeName)
	case int8:
		return signedToTarget(int64(v), math.MaxInt8, targetMax, typeName)
	case float64:
		return floatToTarget(v, maxExactFloat64Integer, targetMax, typeName)
	case float32:
		return floatToTarget(float64(v), maxExactFloat32Integer, targetMax, typeName)
	case json.Number:
		return jsonNumberToTarget(v, targetMax, typeName)
	default:
		return 0, fmt.Errorf("cannot convert %T to %s", value, typeName)
	}
}

// unsignedToTarget range-checks an unsigned wire value against the target width.
func unsignedToTarget(v, targetMax uint64, typeName string) (uint64, error) {
	if v > targetMax {
		return 0, fmt.Errorf("value %d out of range for %s", v, typeName)
	}
	return v, nil
}

// signedToTarget converts a signed wire value to the target width. When the
// source type can exceed the target range, negatives and overflows share one
// range error; when every non-negative source value fits, only negatives can
// fail and get a dedicated error.
func signedToTarget(v int64, sourceMax, targetMax uint64, typeName string) (uint64, error) {
	if sourceMax > targetMax {
		if v < 0 || uint64(v) > targetMax {
			return 0, fmt.Errorf("value %d out of range for %s", v, typeName)
		}
		return uint64(v), nil
	}
	if v < 0 {
		return 0, fmt.Errorf("negative value cannot be coerced to %s: %d", typeName, v)
	}
	return uint64(v), nil
}

// floatToTarget converts a float wire value to the target width, requiring an
// exactly representable integer within exactLimit. Sub-uint64 targets fold the
// sign check into the range error; a MaxUint64 target can only fail on sign.
func floatToTarget(f float64, exactLimit, targetMax uint64, typeName string) (uint64, error) {
	if err := checkExactIntegerFloat(f, exactLimit); err != nil {
		return 0, err
	}
	if targetMax < math.MaxUint64 {
		if f < 0 || f > float64(targetMax) {
			return 0, fmt.Errorf("value %f out of range for %s", f, typeName)
		}
		return uint64(f), nil
	}
	if f < 0 {
		return 0, fmt.Errorf("negative value cannot be coerced to %s: %v", typeName, f)
	}
	return uint64(f), nil
}

// jsonNumberToTarget parses a json.Number and range-checks it against the
// target width.
func jsonNumberToTarget(v json.Number, targetMax uint64, typeName string) (uint64, error) {
	u, err := jsonNumberToUint64(v)
	if err != nil {
		return 0, err
	}
	if u > targetMax {
		return 0, fmt.Errorf("value %d out of range for %s", u, typeName)
	}
	return u, nil
}

// coerceUint64 converts wire numeric values to uint64 with exact semantics.
func coerceUint64(value interface{}) (uint64, error) {
	return coerceToUnsigned(value, math.MaxUint64, "uint64")
}

// coerceUint32 converts various numeric types to uint32
func coerceUint32(value interface{}) (uint32, error) {
	u, err := coerceToUnsigned(value, math.MaxUint32, "uint32")
	if err != nil {
		return 0, err
	}
	return uint32(u), nil // #nosec G115 -- coerceToUnsigned enforces targetMax
}

// coerceUint16 converts various numeric types to uint16
func coerceUint16(value interface{}) (uint16, error) {
	u, err := coerceToUnsigned(value, math.MaxUint16, "uint16")
	if err != nil {
		return 0, err
	}
	return uint16(u), nil // #nosec G115 -- coerceToUnsigned enforces targetMax
}

// coerceFloat64 converts various numeric types to float64.
// Non-finite values are rejected: NaN and infinities are not representable in
// either BSSCI wire encoding's JSON form and are invalid field values.
func coerceFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, fmt.Errorf("non-finite value %v is not a valid field value", v)
		}
		return v, nil
	case float32:
		f := float64(v)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, fmt.Errorf("non-finite value %v is not a valid field value", f)
		}
		return f, nil
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return 0, fmt.Errorf("invalid JSON number %q: %w", string(v), err)
		}
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, fmt.Errorf("non-finite value %v is not a valid field value", f)
		}
		return f, nil
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
	case json.Number:
		u, err := jsonNumberToUint64(v)
		if err != nil {
			return 0, err
		}
		if u > 255 {
			return 0, fmt.Errorf("value %d out of byte range (0-255)", u)
		}
		return byte(u), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to byte", value)
	}
}
