// Package adapter error catalog. Prose for errors raised while bridging
// KC-Identity RPC responses lives here so adapter call sites carry no
// hardcoded error strings.
package adapter

import "errors"

// ErrCertResolutionUnsupported is returned by ResolveCert: certificate-based
// org resolution is handled by KC-Core, never by the gateway.
var ErrCertResolutionUnsupported = errors.New("gateway does not support certificate-based org resolution")

const (
	// ErrIdentityValidateAPIKeyFailed prefixes ValidateAPIKey RPC transport failures.
	ErrIdentityValidateAPIKeyFailed = "identity RPC ValidateAPIKey failed" //nolint:gosec // G101: RPC failure prefix, not a credential
	// ErrInvalidAPIKeyID prefixes API key ID parse failures from identity responses.
	ErrInvalidAPIKeyID = "invalid API key ID from identity service" //nolint:gosec // G101: parse-failure prefix, not a credential
	// ErrInvalidOrgID prefixes organization ID parse failures from identity responses.
	ErrInvalidOrgID = "invalid org ID from identity service"
	// ErrInvalidUserID prefixes user ID parse failures from identity responses.
	ErrInvalidUserID = "invalid user ID from identity service"
	// ErrInvalidOrgUUID prefixes organization UUID parse failures from identity responses.
	ErrInvalidOrgUUID = "invalid org UUID from identity service"
)

// Wrap formats for identity RPC failures that carry the failing identifier.
const (
	// ErrFmtIdentityResolveOrgFailed wraps ResolveOrg RPC failures with the org UUID.
	ErrFmtIdentityResolveOrgFailed = "identity RPC ResolveOrg failed for org %s: %w"
	// ErrFmtIdentityGetDefaultOrgFailed wraps GetDefaultOrgForTenant RPC failures with the tenant ID.
	ErrFmtIdentityGetDefaultOrgFailed = "identity RPC GetDefaultOrgForTenant failed for tenant %d: %w"
)
