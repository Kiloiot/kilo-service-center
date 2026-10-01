package mioty

import "errors"

// Error-wrap contexts and error format strings, grouped by the file that
// uses them; the shared group serves multiple files.
const (
	// codec.go, checked.go
	errFmtNonceSignMustBe4ElementNumericArray  = "nonce/sign must be 4-element numeric array, got %d elements"
	errFmtNonceSignMustBe4ElementsGot          = "nonce/sign must be 4 elements, got %d"
	errWrapNonceSignMustBe4ElementNumericArray = "nonce/sign must be 4-element numeric array"
	errFmtUserDataEntry                        = "userData entry %d: %w"
	errFmtNumericValue                         = "%w: %v"
	errFmtNumericLength                        = "%w: %d values, want %d"
	errFmtNumericType                          = "numeric field holds %T"
	errFmtMissingField                         = "%w: %s"
)

// ErrNumericOutOfRange reports a Numeric value outside its field's range,
// such as an array element outside the 0-255 byte range.
var ErrNumericOutOfRange = errors.New("numeric value outside the field's range")

// ErrNumericLength reports a fixed-length Numeric array of another length.
var ErrNumericLength = errors.New("numeric array length differs from the field's")

// ErrMissingMandatoryField reports a decoded message without one of its
// mandatory fields.
var ErrMissingMandatoryField = errors.New("missing mandatory field")
