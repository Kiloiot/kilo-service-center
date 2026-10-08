// Package blueprint provides centralized error tokens for blueprint operations.
// All blueprint decode errors should use these tokens for consistent messaging
// and error tracking. Frontend display strings are in KC-Web messages.ts.
package blueprint

import (
	"errors"
	"fmt"
)

// Error tokens for blueprint operations (centralized, no inline strings)
// These tokens map to human-readable messages via ResolveErrorMessage().
const (
	// Validation errors
	ErrInvalidBlueprintJSON = "blueprint.error.invalid_json"
	ErrMissingVersion       = "blueprint.error.missing_version"
	ErrMissingTypeEUI       = "blueprint.error.missing_type_eui"
	ErrInvalidTypeEUILength = "blueprint.error.invalid_type_eui_length"
	ErrInvalidTypeEUIFormat = "blueprint.error.invalid_type_eui_format"
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

	// Internal errors
	ErrInternalDecodePanic    = "blueprint.error.internal_decode_panic"
	ErrInternalParseError     = "blueprint.error.internal_parse_error"
	ErrInvalidCryptoFieldType = "blueprint.error.invalid_crypto_field_type"

	// Naming and registry-path validation.
	ErrInvalidRegistryPathSegment = "blueprint.error.invalid_registry_path_segment"
	ErrInvalidModelCode           = "blueprint.error.invalid_model_code"
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
	ErrInvalidTypeEUIFormat: "Type EUI must be 16 hexadecimal characters",
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

	// Internal errors
	ErrInternalDecodePanic:        "Internal error: decoder panic recovered",
	ErrInternalParseError:         "Internal error: failed to parse blueprint specification",
	ErrInvalidCryptoFieldType:     "Crypto field must be a string or integer",
	ErrInvalidRegistryPathSegment: "Registry path segment is empty or invalid",
	ErrInvalidModelCode:           "Model code must contain only lowercase alphanumeric characters and hyphens",
}

const unknownBlueprintErrorFmt = "Unknown blueprint error: %s"

// ResolveErrorMessage returns the human-readable message for an error token.
// Returns a formatted message if the token is unknown.
// unknownBlueprintErrorFmt labels tokens missing from the catalog.
func ResolveErrorMessage(token string) string {
	if msg, ok := errorMessages[token]; ok {
		return msg
	}
	return fmt.Sprintf(unknownBlueprintErrorFmt, token)
}

// ErrGitHubAppKeyPEM reports a GitHub App private key that is not valid PEM.
var ErrGitHubAppKeyPEM = errors.New("failed to decode PEM block from github_app_private_key")
