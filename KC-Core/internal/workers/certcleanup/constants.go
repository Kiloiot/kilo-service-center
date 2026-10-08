package certcleanup

import "errors"

// errFmtInvalidInterval reports an unusable cleanup interval.
const errFmtInvalidInterval = "%w: %v"

// Sentinel errors of the certificate bundle cleanup worker.
var (
	errMissingDependency = errors.New("certificate cleanup worker: missing dependency")
	errInvalidInterval   = errors.New("certificate cleanup worker: interval must be positive")
)
