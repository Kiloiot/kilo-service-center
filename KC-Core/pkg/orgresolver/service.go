// Package orgresolver implements org.Resolver over the organization
// directory with in-memory TTL caches. KC-Core and KC-Identity share it for
// organization UUID to tenant ID resolution.
package orgresolver

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/google/uuid"
)

// OrgDirectory resolves organization identity to and from tenant ids.
type OrgDirectory interface {
	GetTenantByOrgID(ctx context.Context, orgID uuid.UUID) (int64, error)
	GetOrgByTenantID(ctx context.Context, tenantID int64) (*models.Organization, error)
	GetOrgByExternalID(ctx context.Context, externalID string) (*models.Organization, error)
}

// service implements org.Resolver over the organization directory. Both
// lookups that run on the request path - organization to tenant and tenant
// to its default organization - are served from bounded TTL caches; org
// changes are rare, so a read may be stale for up to one TTL. A peer
// certificate is resolved once per connection, from the directory.
type service struct {
	orgRepo     OrgDirectory
	logger      logger.Logger
	maxEntries  int
	tenants     *ttlCache[uuid.UUID, int64]
	defaultOrgs *ttlCache[int64, uuid.UUID]
}

// orgCNPrefix is the certificate common-name prefix carrying an organization
// UUID.
const orgCNPrefix = "org-"

// New creates an organization resolver whose caches keep an entry for
// cacheTTL, hold at most maxEntries each, and read time from clk.
func New(
	orgRepo OrgDirectory,
	log logger.Logger,
	cacheTTL time.Duration,
	maxEntries int,
	clk clock.Clock,
) (org.Resolver, error) {
	if orgRepo == nil || log == nil || clk == nil {
		return nil, ErrMissingDependency
	}
	return &service{
		orgRepo:     orgRepo,
		logger:      log,
		maxEntries:  maxEntries,
		tenants:     newTTLCache[uuid.UUID, int64](cacheTTL, maxEntries, clk),
		defaultOrgs: newTTLCache[int64, uuid.UUID](cacheTTL, maxEntries, clk),
	}, nil
}

// logCached records a cache write, and the flush when it found the cache full.
func (s *service) logCached(ctx context.Context, dropped, size int, fields ...interface{}) {
	if dropped > 0 {
		s.logger.WarnContext(ctx, LogOrgCacheFull,
			logger.FieldCurrentSize, dropped,
			logger.FieldMaxEntries, s.maxEntries)
	}
	s.logger.DebugContext(ctx, LogOrgCached, append(fields, logger.FieldCacheSize, size)...)
}

// LookupTenant resolves an organization UUID to its tenant, from the cache
// while the entry is fresh and from the directory otherwise; failures are not
// cached.
func (s *service) LookupTenant(ctx context.Context, orgUUID uuid.UUID) (int64, error) {
	if tenantID, age, ok := s.tenants.get(orgUUID); ok {
		s.logger.DebugContext(ctx, LogOrgCacheHit,
			logger.FieldOrgUUID, orgUUID.String(),
			logger.FieldTenantID, tenantID,
			logger.FieldAge, age)
		return tenantID, nil
	}

	s.logger.DebugContext(ctx, LogOrgCacheMiss,
		logger.FieldOrgUUID, orgUUID.String())

	tenantID, err := s.readTenant(ctx, orgUUID)
	if err != nil {
		return 0, err
	}

	dropped, size := s.tenants.put(orgUUID, tenantID)
	s.logCached(ctx, dropped, size,
		logger.FieldOrgUUID, orgUUID.String(),
		logger.FieldTenantID, tenantID)

	return tenantID, nil
}

// readTenant resolves an organization UUID to its tenant from the directory.
func (s *service) readTenant(ctx context.Context, orgUUID uuid.UUID) (int64, error) {
	tenantID, err := s.orgRepo.GetTenantByOrgID(ctx, orgUUID)
	if err != nil {
		return 0, fmt.Errorf("%w: %s: %w", ErrOrgNotFound, orgUUID, err)
	}
	return tenantID, nil
}

// ResolveCert implements org.Resolver.ResolveCert
//
// Certificate CN parsing:
//   - "org-{uuid}":  Standard format with prefix (trim prefix)
//   - "{uuid}":      Direct UUID format (use as-is)
//
// The tenant is read from the directory, not the cache: a certificate is
// resolved once per connection, and a cached mapping would keep a peer of an
// organization moved or deleted within the TTL in its old tenant for the
// life of the connection.
func (s *service) ResolveCert(ctx context.Context, cert *x509.Certificate) (uuid.UUID, int64, error) {
	if cert == nil {
		return uuid.Nil, 0, ErrNilCertificate
	}

	// Extract Common Name from certificate
	cn := cert.Subject.CommonName
	if cn == "" {
		return uuid.Nil, 0, ErrEmptyCertCN
	}

	// Parse UUID from CN (trim "org-" prefix if present)
	uuidStr := strings.TrimPrefix(cn, orgCNPrefix)

	orgUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("%w %q: %w", ErrInvalidOrgUUIDInCN, cn, err)
	}

	tenantID, err := s.readTenant(ctx, orgUUID)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("%w %s: %w", ErrTenantResolveFailed, orgUUID, err)
	}

	s.logger.DebugContext(ctx, LogOrgCertResolved,
		logger.FieldCn, cn,
		logger.FieldOrgUUID, orgUUID.String(),
		logger.FieldTenantID, tenantID)

	return orgUUID, tenantID, nil
}

// GetDefaultOrgForTenant returns the first organization of a tenant
// (deterministic ordering by org_id). Community-mode requests resolve it on
// every call, so it is cached like LookupTenant; failures are not cached.
func (s *service) GetDefaultOrgForTenant(ctx context.Context, tenantID int64) (uuid.UUID, error) {
	if orgID, age, ok := s.defaultOrgs.get(tenantID); ok {
		s.logger.DebugContext(ctx, LogOrgCacheHit,
			logger.FieldTenantID, tenantID,
			logger.FieldOrgUUID, orgID.String(),
			logger.FieldAge, age)
		return orgID, nil
	}

	org, err := s.orgRepo.GetOrgByTenantID(ctx, tenantID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w %d: %w", ErrNoDefaultOrg, tenantID, err)
	}

	s.logger.DebugContext(ctx, LogOrgDefaultResolved,
		logger.FieldTenantID, tenantID,
		logger.FieldOrgUUID, org.OrgID.String(),
		logger.FieldOrgName, org.Name)

	dropped, size := s.defaultOrgs.put(tenantID, org.OrgID)
	s.logCached(ctx, dropped, size,
		logger.FieldTenantID, tenantID,
		logger.FieldOrgUUID, org.OrgID.String())

	return org.OrgID, nil
}

// Compile-time check that service implements org.OrganizationResolver.
var _ org.OrganizationResolver = (*service)(nil)

// ResolveOrgByExternalID resolves an external IdP organization identifier to a local
// organization UUID. Used by external auth flows (OIDC/OAuth2) when the identity
// provider includes an org claim.
//
// Returns:
//   - uuid.Nil with nil error if external ID not found (not an error for external auth)
//   - uuid.Nil with error on database errors
//   - org.OrgID on success
//
// Thread Safety: Delegates to repository which is thread-safe
func (s *service) ResolveOrgByExternalID(ctx context.Context, externalID string) (uuid.UUID, error) {
	org, err := s.orgRepo.GetOrgByExternalID(ctx, externalID)
	if errors.Is(err, storage.ErrNotFound) {
		// Not found is not an error for external auth - org mapping is optional
		s.logger.DebugContext(ctx, LogOrgExternalMissing,
			logger.FieldExternalID, externalID)
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w %q: %w", ErrExternalOrgResolveFailed, externalID, err)
	}

	s.logger.DebugContext(ctx, LogOrgExternalResolved,
		logger.FieldExternalID, externalID,
		logger.FieldOrgUUID, org.OrgID.String(),
		logger.FieldOrgName, org.Name)

	return org.OrgID, nil
}
