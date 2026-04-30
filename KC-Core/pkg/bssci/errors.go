package bssci

import "errors"

// Sentinel errors for structured error handling with errors.Is pattern

// ErrResumeCounterMismatch is returned when resume operation counters do not match persisted session state (BSSCI §5.2)
var ErrResumeCounterMismatch = errors.New("resume rejected: operation counters do not match persisted session state")

// ErrDetachValidationEndpointNotFound is returned by detach validator when endpoint not found in database.
// Enables typed error detection via errors.Is in server.go detach handler.
var ErrDetachValidationEndpointNotFound = errors.New("detach validation endpoint not found")

// ErrDetachSignatureInvalid is returned by detach validator when signature cryptographic check fails.
// Maps to errDetachSignatureInvalid catalog token and POSIX error code from internal validator.
var ErrDetachSignatureInvalid = errors.New("detach signature validation failed")

// CatalogError wraps error catalog tokens for structured error handling
// This enables services to return tokens that transports resolve via ResolveErrorMessage
type CatalogError struct {
	Token string // Exact token from errors_catalog.go (e.g., errVersionIncompatible)
	Posix int    // POSIX error code for wire protocol
}

// Error implements the error interface
// Returns the token exactly - no prefixes or modifications
func (e *CatalogError) Error() string {
	return e.Token
}

// NewCatalogError creates a new catalog error with a token and POSIX code
func NewCatalogError(token string, posix int) *CatalogError {
	return &CatalogError{
		Token: token,
		Posix: posix,
	}
}
