// Package orgresolver implements the org.Resolver interface with in-memory caching
// for KC-Identity's local organization → tenant resolution.
package orgresolver

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/org"
	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/interfaces"
)

// service implements org.Resolver and org.OrganizationResolver with in-memory caching.
//
// Caching Strategy:
//   - Algorithm: Simple in-memory map with read/write mutex
//   - Concurrency: sync.RWMutex protecting map[uuid.UUID]cacheEntry
//   - TTL: Configurable (default 5 minutes)
//   - Max size: Configurable (default 10,000 entries)
//   - Eviction: Clear all entries when max size exceeded (log warning)
type service struct {
	orgRepo    interfaces.OrganizationRepository
	logger     logger.Logger
	cacheMu    sync.RWMutex
	cache      map[uuid.UUID]cacheEntry
	cacheTTL   time.Duration
	maxEntries int
}

// cacheEntry represents a cached organization → tenant mapping.
type cacheEntry struct {
	tenantID int64
	cachedAt time.Time
}

// New creates a new organization resolver service with caching.
func New(
	orgRepo interfaces.OrganizationRepository,
	log logger.Logger,
	cacheTTL time.Duration,
	maxEntries int,
) org.Resolver {
	return &service{
		orgRepo:    orgRepo,
		logger:     log,
		cache:      make(map[uuid.UUID]cacheEntry),
		cacheTTL:   cacheTTL,
		maxEntries: maxEntries,
	}
}

// LookupTenant implements org.Resolver.LookupTenant.
// Checks in-memory cache first, falls back to database on miss/expiry.
func (s *service) LookupTenant(ctx context.Context, orgUUID uuid.UUID) (int64, error) {
	s.cacheMu.RLock()
	entry, found := s.cache[orgUUID]
	s.cacheMu.RUnlock()

	if found && time.Since(entry.cachedAt) < s.cacheTTL {
		s.logger.DebugContext(ctx, "Org cache hit",
			"orgUUID", orgUUID.String(),
			"tenantID", entry.tenantID,
			"age", time.Since(entry.cachedAt))
		return entry.tenantID, nil
	}

	s.logger.DebugContext(ctx, "Org cache miss, querying database",
		"orgUUID", orgUUID.String())

	tenantID, err := s.orgRepo.GetTenantByOrgID(ctx, orgUUID)
	if err != nil {
		return 0, fmt.Errorf("org %s not found: %w", orgUUID, err)
	}

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	if len(s.cache) >= s.maxEntries {
		s.logger.WarnContext(ctx, "Org cache full, clearing all entries",
			"currentSize", len(s.cache),
			"maxEntries", s.maxEntries)
		s.cache = make(map[uuid.UUID]cacheEntry)
	}

	s.cache[orgUUID] = cacheEntry{
		tenantID: tenantID,
		cachedAt: time.Now(),
	}

	s.logger.DebugContext(ctx, "Org cached",
		"orgUUID", orgUUID.String(),
		"tenantID", tenantID,
		"cacheSize", len(s.cache))

	return tenantID, nil
}

// ResolveCert implements org.Resolver.ResolveCert.
// Extracts organization UUID from certificate CN and resolves its tenant.
func (s *service) ResolveCert(ctx context.Context, cert *x509.Certificate) (uuid.UUID, int64, error) {
	if cert == nil {
		return uuid.Nil, 0, fmt.Errorf("certificate is nil")
	}

	cn := cert.Subject.CommonName
	if cn == "" {
		return uuid.Nil, 0, fmt.Errorf("certificate CN is empty")
	}

	uuidStr := strings.TrimPrefix(cn, "org-")

	orgUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("invalid org UUID in cert CN %q: %w", cn, err)
	}

	tenantID, err := s.LookupTenant(ctx, orgUUID)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("failed to resolve tenant for org %s: %w", orgUUID, err)
	}

	s.logger.DebugContext(ctx, "Resolved cert to org and tenant",
		"cn", cn,
		"orgUUID", orgUUID.String(),
		"tenantID", tenantID)

	return orgUUID, tenantID, nil
}

// GetDefaultOrgForTenant implements org.Resolver.GetDefaultOrgForTenant.
// Returns the first organization for a tenant (deterministic ordering by org_id).
func (s *service) GetDefaultOrgForTenant(ctx context.Context, tenantID int64) (uuid.UUID, error) {
	o, err := s.orgRepo.GetOrgByTenantID(ctx, tenantID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("no default org for tenant %d: %w", tenantID, err)
	}

	s.logger.DebugContext(ctx, "Resolved default org for tenant",
		"tenantID", tenantID,
		"orgUUID", o.OrgID.String(),
		"orgName", o.Name)

	return o.OrgID, nil
}

// Compile-time check that service implements org.OrganizationResolver.
var _ org.OrganizationResolver = (*service)(nil)

// ResolveOrgByExternalID implements org.OrganizationResolver.
// Resolves an external IdP organization identifier to a local organization UUID.
func (s *service) ResolveOrgByExternalID(ctx context.Context, externalID string) (uuid.UUID, error) {
	o, err := s.orgRepo.GetOrgByExternalID(ctx, externalID)
	if errors.Is(err, storage.ErrNotFound) {
		s.logger.DebugContext(ctx, "External org ID not found",
			"externalID", externalID)
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to resolve external org %q: %w", externalID, err)
	}

	s.logger.DebugContext(ctx, "Resolved external org ID",
		"externalID", externalID,
		"orgUUID", o.OrgID.String(),
		"orgName", o.Name)

	return o.OrgID, nil
}
