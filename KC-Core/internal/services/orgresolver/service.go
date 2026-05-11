// Package orgresolver implements the org.Resolver interface with in-memory caching.
//
// This service provides organization UUID → tenant ID resolution for Kilo Cloud
// multi-tenant integration, with explicit caching to optimize the hot path.
package orgresolver

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/google/uuid"
)

// service implements org.Resolver with in-memory caching
//
// Caching Strategy:
//   - Algorithm: Simple in-memory map with read/write mutex
//   - Concurrency: sync.RWMutex protecting map[uuid.UUID]cacheEntry
//   - TTL: Configurable (default 5 minutes)
//   - Max size: Configurable (default 10,000 entries)
//   - Eviction: Clear all entries when max size exceeded (log warning)
//   - Invalidation: None (assumes org changes are rare, stale reads acceptable)
//
// Thread Safety: All public methods use RWMutex for safe concurrent access
type service struct {
	orgRepo    interfaces.OrganizationRepository
	logger     logger.Logger
	cacheMu    sync.RWMutex
	cache      map[uuid.UUID]cacheEntry
	cacheTTL   time.Duration
	maxEntries int
}

// cacheEntry represents a cached organization → tenant mapping
type cacheEntry struct {
	tenantID int64
	cachedAt time.Time
}

// New creates a new organization resolver service with caching
//
// Parameters:
//   - orgRepo: Organization repository for database lookups
//   - logger: Structured logger
//   - cacheTTL: Cache entry time-to-live (recommend: 5 minutes)
//   - maxEntries: Maximum cache entries (recommend: 10,000)
//
// Returns:
//   - org.Resolver: Service instance implementing the interface
//
// Thread Safety: Returned service is safe for concurrent use
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

// LookupTenant implements org.Resolver.LookupTenant
//
// Flow:
//  1. Check cache (RLock)
//  2. If cache hit and not expired → return cached value
//  3. If cache miss → query database
//  4. Update cache (Lock) if query succeeds
//  5. If cache full → clear all entries and log warning
//
// Thread Safety: Uses RWMutex for safe concurrent access
func (s *service) LookupTenant(ctx context.Context, orgUUID uuid.UUID) (int64, error) {
	// Check cache first (read lock)
	s.cacheMu.RLock()
	entry, found := s.cache[orgUUID]
	s.cacheMu.RUnlock()

	if found && time.Since(entry.cachedAt) < s.cacheTTL {
		// Cache hit
		s.logger.DebugContext(ctx, "Org cache hit",
			"orgUUID", orgUUID.String(),
			"tenantID", entry.tenantID,
			"age", time.Since(entry.cachedAt))
		return entry.tenantID, nil
	}

	// Cache miss or expired: query database
	s.logger.DebugContext(ctx, "Org cache miss, querying database",
		"orgUUID", orgUUID.String())

	tenantID, err := s.orgRepo.GetTenantByOrgID(ctx, orgUUID)
	if err != nil {
		return 0, fmt.Errorf("org %s not found: %w", orgUUID, err)
	}

	// Update cache (write lock)
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	// Check cache size and clear if needed
	if len(s.cache) >= s.maxEntries {
		s.logger.WarnContext(ctx, "Org cache full, clearing all entries",
			"currentSize", len(s.cache),
			"maxEntries", s.maxEntries)
		s.cache = make(map[uuid.UUID]cacheEntry)
	}

	// Add to cache
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

// ResolveCert implements org.Resolver.ResolveCert
//
// Certificate CN parsing:
//   - "org-{uuid}":  Standard format with prefix (trim prefix)
//   - "{uuid}":      Direct UUID format (use as-is)
//
// Flow:
//  1. Extract CN from certificate
//  2. Trim "org-" prefix if present
//  3. Parse UUID from CN string
//  4. Call LookupTenant (leverages cache)
//
// Thread Safety: Delegates to LookupTenant which is thread-safe
func (s *service) ResolveCert(ctx context.Context, cert *x509.Certificate) (uuid.UUID, int64, error) {
	if cert == nil {
		return uuid.Nil, 0, fmt.Errorf("certificate is nil")
	}

	// Extract Common Name from certificate
	cn := cert.Subject.CommonName
	if cn == "" {
		return uuid.Nil, 0, fmt.Errorf("certificate CN is empty")
	}

	// Parse UUID from CN (trim "org-" prefix if present)
	uuidStr := strings.TrimPrefix(cn, "org-")

	orgUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("invalid org UUID in cert CN %q: %w", cn, err)
	}

	// Resolve tenant using cached lookup
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

// GetDefaultOrgForTenant implements org.Resolver.GetDefaultOrgForTenant
//
// Returns the first organization for a tenant (deterministic ordering by org_id).
// Migration 000058 creates one default organization per tenant, so this should
// always succeed for valid tenants.
//
// Flow:
//  1. Call GetOrgByTenantID repository method
//  2. Extract org_id from returned organization
//  3. Return org UUID
//
// Note: This method does NOT use caching (infrequent fallback path)
// Thread Safety: Delegates to repository which is thread-safe
func (s *service) GetDefaultOrgForTenant(ctx context.Context, tenantID int64) (uuid.UUID, error) {
	org, err := s.orgRepo.GetOrgByTenantID(ctx, tenantID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("no default org for tenant %d: %w", tenantID, err)
	}

	s.logger.DebugContext(ctx, "Resolved default org for tenant",
		"tenantID", tenantID,
		"orgUUID", org.OrgID.String(),
		"orgName", org.Name)

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
		s.logger.DebugContext(ctx, "External org ID not found",
			"externalID", externalID)
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to resolve external org %q: %w", externalID, err)
	}

	s.logger.DebugContext(ctx, "Resolved external org ID",
		"externalID", externalID,
		"orgUUID", org.OrgID.String(),
		"orgName", org.Name)

	return org.OrgID, nil
}
