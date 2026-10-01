package grpc

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/google/uuid"
)

// orgResolverAdapter implements OrganizationResolver by wrapping OrganizationRepository.
// Bridges the repository layer (GetOrgByTenantID) to the auth interceptor (ResolveOrganization)
// for tenant→organization UUID resolution.
type orgResolverAdapter struct {
	repo interfaces.OrgDirectoryRepository
	log  logger.Logger
}

// NewOrgResolverAdapter creates an OrganizationResolver that wraps a repository
func NewOrgResolverAdapter(repo interfaces.OrgDirectoryRepository, log logger.Logger) OrganizationResolver {
	return &orgResolverAdapter{
		repo: repo,
		log:  log,
	}
}

// ResolveOrganization maps a tenant ID to its organization UUID. The auth
// interceptor reports a failure; organization context is optional there.
func (a *orgResolverAdapter) ResolveOrganization(ctx context.Context, tenantID int64) (uuid.UUID, error) {
	org, err := a.repo.GetOrgByTenantID(ctx, tenantID)
	if err != nil {
		return uuid.Nil, err
	}

	a.log.DebugContext(ctx, LogResolvedOrganizationForTenant,
		logger.FieldTenantIDSnake, tenantID,
		logger.FieldOrgIDSnake, org.OrgID)

	return org.OrgID, nil
}

// tenantResolverAdapter implements TenantResolver by wrapping org.Resolver.
// Enables org UUID → tenant ID resolution for JWT claims that contain
// organization UUIDs instead of numeric tenant IDs.
type tenantResolverAdapter struct {
	resolver org.Resolver
	log      logger.Logger
}

// NewTenantResolverAdapter creates a TenantResolver that wraps org.Resolver.
func NewTenantResolverAdapter(resolver org.Resolver, log logger.Logger) TenantResolver {
	return &tenantResolverAdapter{
		resolver: resolver,
		log:      log,
	}
}

// LookupTenant maps an organization UUID to its numeric tenant ID.
func (a *tenantResolverAdapter) LookupTenant(ctx context.Context, orgID uuid.UUID) (int64, error) {
	tenantID, err := a.resolver.LookupTenant(ctx, orgID)
	if err != nil {
		a.log.WarnContext(ctx, LogFailedToResolveTenantForOrganization,
			logger.FieldOrgIDSnake, orgID,
			logger.FieldError, err)
		return 0, err
	}

	a.log.DebugContext(ctx, LogResolvedTenantForOrganization,
		logger.FieldOrgIDSnake, orgID,
		logger.FieldTenantIDSnake, tenantID)

	return tenantID, nil
}
