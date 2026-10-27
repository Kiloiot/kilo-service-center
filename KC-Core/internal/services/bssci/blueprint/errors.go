package blueprint

import "errors"

// Sentinel errors for payload decoding and expression evaluation. Decode
// errors surface to callers wrapped in blueprint.DecodeError details, so the
// exact message text is part of the decode diagnostics contract.
var (
	// errBytesTypeAlignment reports a bytes field whose offset or size is not byte-aligned.
	errBytesTypeAlignment = errors.New("bytes type requires byte-aligned offset and size")
	// errStringTypeAlignment reports a string field whose offset or size is not byte-aligned.
	errStringTypeAlignment = errors.New("string type requires byte-aligned offset and size")
	// errCalibrationNotProvided reports a calibration lookup without calibration data.
	errCalibrationNotProvided = errors.New("calibration data not provided")
	// errDivisionByZero reports a division by zero inside an arithmetic expression.
	errDivisionByZero = errors.New("division by zero")
	// errUnexpectedEndOfExpression reports an expression that ends before a factor.
	errUnexpectedEndOfExpression = errors.New("unexpected end of expression")
	// errMissingClosingParen reports an unbalanced parenthesized expression.
	errMissingClosingParen = errors.New("missing closing parenthesis")
)

// Error format strings for failures that carry positional or value context.
const (
	errFmtBitRangeExceedsPayload = "bit range [%d:%d] exceeds payload size %d bits"
	errFmtFormatIDNotInBlueprint = "format_id=%d not found in blueprint"
	errFmtPayloadBytesShort      = "need %d bytes, got %d"
	errFmtUnsupportedFieldType   = "unsupported field type: %s"
	errFmtBitSizeExceedsMax      = "bit size %d exceeds maximum of 64"
	errFmtCalibrationKeyNotFound = "calibration key '%s' not found"
	errFmtCannotEvaluateExpr     = "cannot evaluate expression: %s"
	errFmtCompareNonNumeric      = "cannot compare non-numeric values with operator %s"
	errFmtMissingClosingParenFor = "missing closing parenthesis for %s"
	errFmtExpectedNumberAtPos    = "expected number at position %d"
	errFmtInvalidNumber          = "invalid number: %s"
)

// errWrapListUnpinned prefixes a failed read of a tenant's endpoints without a snapshot.
const errWrapListUnpinned = "list endpoints without a snapshot"

// errNilSnapshotOnlyModes rejects a resolver built without the tenant switch reader.
var errNilSnapshotOnlyModes = errors.New("blueprint resolver: snapshot-only mode reader is required")

// errNilPinDependency rejects a pin service built without its resolver, endpoint store or clock.
var errNilPinDependency = errors.New("blueprint pin service: resolver, endpoint store and clock are required")
