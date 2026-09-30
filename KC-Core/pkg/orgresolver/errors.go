package orgresolver

import "errors"

// Domain sentinels for organization resolution failures.
var (
	// ErrNilCertificate reports a resolve request without a certificate.
	ErrNilCertificate = errors.New("certificate is nil")
	// ErrEmptyCertCN reports a certificate whose subject carries no common name.
	ErrEmptyCertCN = errors.New("certificate CN is empty")
	// ErrOrgNotFound reports an organization UUID with no tenant mapping.
	ErrOrgNotFound = errors.New("org not found")
	// ErrInvalidOrgUUIDInCN reports a certificate CN that does not parse as an org UUID.
	ErrInvalidOrgUUIDInCN = errors.New("invalid org UUID in cert CN")
	// ErrTenantResolveFailed reports a tenant lookup failure for an organization.
	ErrTenantResolveFailed = errors.New("failed to resolve tenant for org")
	// ErrNoDefaultOrg reports a tenant without a default organization.
	ErrNoDefaultOrg = errors.New("no default org for tenant")
	// ErrExternalOrgResolveFailed reports a database failure resolving an external org ID.
	ErrExternalOrgResolveFailed = errors.New("failed to resolve external org")
	// ErrMissingDependency reports a resolver built without its directory, logger or clock.
	ErrMissingDependency = errors.New("org resolver: missing dependency")
)
