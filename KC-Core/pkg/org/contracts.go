// Package org provides organization UUID resolution for Kilo Cloud multi-tenant integration.
//
// The Resolver interface enables KC-Core to map organization UUIDs (from TLS certificates
// or X-Organization-ID headers) to numeric tenant IDs used throughout the database layer.
package org

import (
	"context"
	"crypto/x509"

	"github.com/google/uuid"
)

// OrganizationResolver resolves external org claims to local organization IDs.
// Used by external auth flows (OIDC/OAuth2) to map IdP org identifiers.
type OrganizationResolver interface {
	ResolveOrgByExternalID(ctx context.Context, externalID string) (uuid.UUID, error)
}

// Resolver provides organization UUID → tenant ID resolution for Kilo Cloud
// multi-tenant integration. Implementations should cache lookups for performance.
//
// Thread Safety: All methods must be safe for concurrent use.
type Resolver interface {
	// LookupTenant resolves an organization UUID to its numeric tenant ID.
	//
	// This is the hot path for organization-based requests. Implementations
	// should cache results to avoid database roundtrips on every request.
	//
	// Parameters:
	//   - ctx: Request context for cancellation/timeout
	//   - orgUUID: Organization UUID to resolve
	//
	// Returns:
	//   - tenantID: Numeric tenant ID for database sharding
	//   - error: ErrNotFound if organization doesn't exist, or database error
	//
	// Thread Safety: Must be safe for concurrent calls
	LookupTenant(ctx context.Context, orgUUID uuid.UUID) (int64, error)

	// ResolveCert extracts an organization UUID from a TLS certificate's
	// Common Name (CN) field and resolves it to a tenant ID.
	//
	// Certificate CN formats supported:
	//   - "org-{uuid}":  Standard format with prefix
	//   - "{uuid}":      Direct UUID (backward compatible)
	//
	// Parameters:
	//   - ctx: Request context for cancellation/timeout
	//   - cert: Client TLS certificate from mutual TLS handshake
	//
	// Returns:
	//   - orgUUID: Extracted organization UUID from cert CN
	//   - tenantID: Numeric tenant ID for database sharding
	//   - error: ErrInvalidCert if CN can't be parsed, or resolution error
	//
	// Usage: SCACI/BSSCI servers extract org context from client certificates
	// Thread Safety: Must be safe for concurrent calls
	ResolveCert(ctx context.Context, cert *x509.Certificate) (uuid.UUID, int64, error)

	// GetDefaultOrgForTenant returns the default organization UUID for a tenant.
	//
	// This is used as a fallback when:
	//   1. Certificate parsing fails (community self-hosted mode)
	//   2. X-Organization-ID header is missing (development/testing)
	//
	// Migration 000058 creates one default organization per tenant during
	// initial deployment, so this method should always succeed for valid tenants.
	//
	// Parameters:
	//   - ctx: Request context for cancellation/timeout
	//   - tenantID: Numeric tenant ID
	//
	// Returns:
	//   - orgUUID: Default organization UUID for the tenant
	//   - error: ErrNotFound if tenant has no organizations, or database error
	//
	// Usage: Community mode fallback when org enforcement is disabled
	// Thread Safety: Must be safe for concurrent calls
	GetDefaultOrgForTenant(ctx context.Context, tenantID int64) (uuid.UUID, error)
}
