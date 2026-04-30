// Package blueprint provides centralized constants and types for MIOTY blueprint payload decoding.
// This package is the single source of truth for decode status values and field type constants.
// Frontend (KC-Web) mirrors these constants in app.ts; display strings go in messages.ts.
package blueprint

// Decode status constants - used in DB (decode_status column) and API responses.
// These are the canonical values; KC-Web mirrors them in src/constants/app.ts.
// IMPORTANT: Never use inline strings for decode status - always reference these constants.
const (
	// DecodeStatusPending indicates the payload has not yet been decoded.
	// This is the default status when a message is first received.
	DecodeStatusPending = "pending"

	// DecodeStatusSuccess indicates the payload was successfully decoded.
	DecodeStatusSuccess = "success"

	// DecodeStatusFailed indicates decoding was attempted but failed.
	// When this status is set, DecodeErrorCode should contain the error token.
	DecodeStatusFailed = "failed"

	// DecodeStatusSkipped indicates decoding was skipped (no blueprint available).
	// This is set when no matching blueprint is found for the endpoint's Type EUI.
	DecodeStatusSkipped = "skipped"
)

// Field type constants for blueprint payload components.
// These define how to interpret bits extracted from the payload.
const (
	// FieldTypeUint interprets bits as an unsigned integer
	FieldTypeUint = "uint"

	// FieldTypeInt interprets bits as a signed integer (two's complement)
	FieldTypeInt = "int"

	// FieldTypeFloat interprets bits as a floating-point number
	// Typically used with 32-bit IEEE 754 single-precision format
	FieldTypeFloat = "float"

	// FieldTypeBool interprets a single bit as a boolean (0=false, 1=true)
	FieldTypeBool = "bool"

	// FieldTypeBytes interprets bits as raw bytes (no numeric conversion)
	FieldTypeBytes = "bytes"

	// FieldTypeString interprets bits as a UTF-8 string
	FieldTypeString = "string"

	// FieldTypeEnum maps integer values to named constants
	FieldTypeEnum = "enum"

	// FieldTypeBinary is an alias for bytes per MIOTY Application Layer Spec Table 14
	FieldTypeBinary = "binary"
)

// Blueprint specification version constants
const (
	// BlueprintSpecVersionV1 is the current blueprint spec version
	BlueprintSpecVersionV1 = "1.0"
)

// Type EUI length constant
const (
	// TypeEUILength is the required length of a MIOTY Type EUI in bytes
	TypeEUILength = 8
)

// Model code (slug) generation constants
const (
	// ModelCodeMaxLength is the maximum length of a generated model code slug
	ModelCodeMaxLength = 64

	// ModelCodeSeparator is the separator used in model code slugs
	ModelCodeSeparator = "-"

	// ModelCodeSuffixFormat is the format for collision-safe slug suffixes
	ModelCodeSuffixFormat = "-%d"

	// ModelCodeMaxSuffixAttempts limits how many suffix increments to try before failing
	ModelCodeMaxSuffixAttempts = 100
)

// Registry submission format constants for structured directory naming and PR metadata.
const (
	// RegistryCommitMsgFmt formats commit messages: manufacturer, model, version
	RegistryCommitMsgFmt = "Add %s %s %s blueprint"

	// RegistryPRTitleFmt formats pull request titles: manufacturer, model, version
	RegistryPRTitleFmt = "Add %s %s %s blueprint"

	// RegistryPRBodyFmt formats pull request body: manufacturer, model, version, type EUI, contributor name, email, notes
	RegistryPRBodyFmt = "## Blueprint Submission\n\n- **Manufacturer**: %s\n- **Model**: %s\n- **Version**: %s\n- **Type EUI**: %s\n- **Contributor**: %s <%s>\n\n%s"

	// RegistryVersionPrefix is the canonical version prefix for registry paths
	RegistryVersionPrefix = "v"
)
