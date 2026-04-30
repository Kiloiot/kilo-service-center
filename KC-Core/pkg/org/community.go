package org

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-DB/storage"
)

// CommunityResolver implements Resolver and OrganizationResolver for
// Community Edition deployments. It maps all requests to a single default
// tenant/org pair and rejects any other organization lookups.
type CommunityResolver struct {
	defaultTenantID int64
	defaultOrgUUID  uuid.UUID
}

// Compile-time interface checks.
var (
	_ Resolver             = (*CommunityResolver)(nil)
	_ OrganizationResolver = (*CommunityResolver)(nil)
)

// NewCommunityResolver creates a resolver that maps all requests to the given
// default tenant and organization. The defaultOrgUUID must be loaded from the
// database at startup; callers should fail hard if the default org is missing.
func NewCommunityResolver(defaultTenantID int64, defaultOrgUUID uuid.UUID) *CommunityResolver {
	return &CommunityResolver{
		defaultTenantID: defaultTenantID,
		defaultOrgUUID:  defaultOrgUUID,
	}
}

// LookupTenant resolves an organization UUID to its tenant ID.
// Only the hidden default org resolves; all others return ErrNotFound.
func (r *CommunityResolver) LookupTenant(_ context.Context, orgUUID uuid.UUID) (int64, error) {
	if orgUUID == r.defaultOrgUUID {
		return r.defaultTenantID, nil
	}
	return 0, fmt.Errorf("org %s not found: %w", orgUUID, storage.ErrNotFound)
}

// ResolveCert returns the default org and tenant for any certificate.
// CE certs are not org-scoped; all connections belong to the single tenant.
func (r *CommunityResolver) ResolveCert(_ context.Context, cert *x509.Certificate) (uuid.UUID, int64, error) {
	if cert == nil {
		return uuid.Nil, 0, fmt.Errorf("certificate is nil")
	}
	return r.defaultOrgUUID, r.defaultTenantID, nil
}

// GetDefaultOrgForTenant returns the default org UUID for the configured tenant.
// Returns ErrNotFound for any other tenant.
func (r *CommunityResolver) GetDefaultOrgForTenant(_ context.Context, tenantID int64) (uuid.UUID, error) {
	if tenantID == r.defaultTenantID {
		return r.defaultOrgUUID, nil
	}
	return uuid.Nil, fmt.Errorf("no default org for tenant %d: %w", tenantID, storage.ErrNotFound)
}

// ResolveOrgByExternalID always returns ErrNotFound in CE.
// External IdP org claims are an enterprise feature.
func (r *CommunityResolver) ResolveOrgByExternalID(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, storage.ErrNotFound
}
