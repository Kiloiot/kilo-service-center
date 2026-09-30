package resilience

// retryableMethodPrefixes name the read-only RPC families that are safe to
// retry on UNAVAILABLE.
var retryableMethodPrefixes = []string{"Get", "List", "Search", "Export", "Query"}

// retryableExactMethods are read-only RPCs whose names fall outside the
// retryable prefixes.
var retryableExactMethods = map[string]bool{"DecodePreview": true}

const retryBackoffMultiplier = 2.0

const statusCodeUnavailable = "UNAVAILABLE"

// errFmtPanic records a panicking call as a breaker failure.
const errFmtPanic = "panic: %v"

// errFmtEncodeRetryServiceConfig wraps a failure to render the retry service config.
const errFmtEncodeRetryServiceConfig = "encode retry service config: %w"
