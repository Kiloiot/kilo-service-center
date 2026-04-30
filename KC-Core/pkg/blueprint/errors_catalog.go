// Package blueprint provides centralized error tokens for blueprint operations.
// All blueprint decode errors should use these tokens for consistent messaging
// and error tracking. Frontend display strings are in KC-Web messages.ts.
package blueprint

import "fmt"

// Error tokens for blueprint operations (centralized, no inline strings)
// These tokens map to human-readable messages via ResolveErrorMessage().
const (
	// Validation errors
	ErrInvalidBlueprintJSON = "blueprint.error.invalid_json"
	ErrMissingVersion       = "blueprint.error.missing_version"
	ErrMissingTypeEUI       = "blueprint.error.missing_type_eui"
	ErrInvalidTypeEUILength = "blueprint.error.invalid_type_eui_length"
	ErrMissingUplinkDefs    = "blueprint.error.missing_uplink_definitions"
	ErrInvalidSpecVersion   = "blueprint.error.invalid_spec_version"
	ErrDuplicateFormatID    = "blueprint.error.duplicate_format_id"

	// Decode errors
	ErrFormatIDNotFound     = "blueprint.error.format_id_not_found"
	ErrPayloadTooShort      = "blueprint.error.payload_too_short"
	ErrBitExtractionFailed  = "blueprint.error.bit_extraction_failed"
	ErrTypeConversionFailed = "blueprint.error.type_conversion_failed"
	ErrExpressionEvalFailed = "blueprint.error.expression_eval_failed"
	ErrConditionEvalFailed  = "blueprint.error.condition_eval_failed"
	ErrCalibrationNotFound  = "blueprint.error.calibration_not_found"
	ErrInvalidFieldType     = "blueprint.error.invalid_field_type"
	ErrInvalidBitOffset     = "blueprint.error.invalid_bit_offset"
	ErrInvalidBitSize       = "blueprint.error.invalid_bit_size"
	ErrOverflowDetected     = "blueprint.error.overflow_detected"
	ErrEnumValueNotMapped   = "blueprint.error.enum_value_not_mapped"

	// Component resolution errors
	ErrComponentRefNotFound = "blueprint.error.component_ref_not_found"

	// Resolution errors
	ErrBlueprintNotFound      = "blueprint.error.not_found"
	ErrNoDefaultBlueprint     = "blueprint.error.no_default_blueprint"
	ErrEndpointNoDeviceModel  = "blueprint.error.endpoint_no_device_model"
	ErrDeviceModelNoBlueprint = "blueprint.error.device_model_no_blueprint"

	// Internal errors
	ErrInternalDecodePanic    = "blueprint.error.internal_decode_panic"
	ErrInternalParseError     = "blueprint.error.internal_parse_error"
	ErrInvalidCryptoFieldType = "blueprint.error.invalid_crypto_field_type"
)

// errorMessages maps error tokens to human-readable messages.
// These messages are for logging and internal diagnostics only.
// User-facing messages are defined in KC-Web messages.ts.
var errorMessages = map[string]string{
	// Validation errors
	ErrInvalidBlueprintJSON: "Blueprint JSON is invalid or malformed",
	ErrMissingVersion:       "Blueprint is missing required 'version' field",
	ErrMissingTypeEUI:       "Blueprint is missing required 'typeEui' field",
	ErrInvalidTypeEUILength: "Type EUI must be exactly 8 bytes",
	ErrMissingUplinkDefs:    "Blueprint must have at least one uplink definition",
	ErrInvalidSpecVersion:   "Blueprint specification version is not supported",
	ErrDuplicateFormatID:    "Duplicate format ID found in blueprint definitions",

	// Decode errors
	ErrFormatIDNotFound:     "No payload format found for the given format ID",
	ErrPayloadTooShort:      "Payload is too short for the blueprint definition",
	ErrBitExtractionFailed:  "Failed to extract bits from payload",
	ErrTypeConversionFailed: "Failed to convert extracted value to target type",
	ErrExpressionEvalFailed: "Failed to evaluate func expression",
	ErrConditionEvalFailed:  "Failed to evaluate condition expression",
	ErrCalibrationNotFound:  "Referenced calibration key not found in endpoint data",
	ErrInvalidFieldType:     "Unknown or unsupported field type in blueprint",
	ErrInvalidBitOffset:     "Bit offset is invalid or out of range",
	ErrInvalidBitSize:       "Bit size is invalid or exceeds payload bounds",
	ErrOverflowDetected:     "Numeric overflow detected during decoding",
	ErrEnumValueNotMapped:   "Enum value not found in mapping",

	// Component resolution errors
	ErrComponentRefNotFound: "Component reference key not found in component definitions",

	// Resolution errors
	ErrBlueprintNotFound:      "No blueprint found for the given Type EUI",
	ErrNoDefaultBlueprint:     "Device model has no default blueprint set",
	ErrEndpointNoDeviceModel:  "Endpoint has no device model assigned",
	ErrDeviceModelNoBlueprint: "Device model has no blueprints defined",

	// Internal errors
	ErrInternalDecodePanic:    "Internal error: decoder panic recovered",
	ErrInternalParseError:     "Internal error: failed to parse blueprint specification",
	ErrInvalidCryptoFieldType: "Crypto field must be a string or integer",
}

// ResolveErrorMessage returns the human-readable message for an error token.
// Returns a formatted message if the token is unknown.
func ResolveErrorMessage(token string) string {
	if msg, ok := errorMessages[token]; ok {
		return msg
	}
	return fmt.Sprintf("Unknown blueprint error: %s", token)
}

// ErrorDefinition contains metadata about a blueprint error for diagnostics and tracking.
type ErrorDefinition struct {
	Token   string
	Message string
	// SpecSection could be added for MIOTY Application Layer spec references
}

// GetErrorDefinition returns the full error definition for a token.
func GetErrorDefinition(token string) *ErrorDefinition {
	msg, ok := errorMessages[token]
	if !ok {
		return &ErrorDefinition{
			Token:   ErrInternalParseError,
			Message: errorMessages[ErrInternalParseError],
		}
	}
	return &ErrorDefinition{
		Token:   token,
		Message: msg,
	}
}
