// Package adapter bridges KC-Identity RPC responses to interceptor interfaces.
package adapter

import (
	"context"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcconstants "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

type tenantCacheEntry struct {
	tenantID  int64
	expiresAt time.Time
}

// IdentityRPCOrgAdapter satisfies org.Resolver, interceptors.OrganizationResolver,
// and interceptors.TenantResolver using KC-Identity's IdentityInternalService RPCs.
type IdentityRPCOrgAdapter struct {
	client     pb.IdentityInternalServiceClient
	peerSecret string
	log        logger.Logger
	mu         sync.RWMutex
	cache      map[uuid.UUID]tenantCacheEntry
	ttl        time.Duration
	maxSize    int
}

// NewIdentityRPCOrgAdapter creates an org adapter that resolves via KC-Identity RPCs.
func NewIdentityRPCOrgAdapter(
	client pb.IdentityInternalServiceClient,
	peerSecret string,
	log logger.Logger,
	ttl time.Duration,
	maxSize int,
) *IdentityRPCOrgAdapter {
	if maxSize <= 0 {
		maxSize = 1000
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &IdentityRPCOrgAdapter{
		client:     client,
		peerSecret: peerSecret,
		log:        log,
		cache:      make(map[uuid.UUID]tenantCacheEntry, maxSize),
		ttl:        ttl,
		maxSize:    maxSize,
	}
}

// LookupTenant resolves an organization UUID to its numeric tenant ID.
// Satisfies org.Resolver and interceptors.TenantResolver.
func (a *IdentityRPCOrgAdapter) LookupTenant(ctx context.Context, orgUUID uuid.UUID) (int64, error) {
	// Check cache first
	a.mu.RLock()
	if entry, ok := a.cache[orgUUID]; ok && time.Now().Before(entry.expiresAt) {
		a.mu.RUnlock()
		return entry.tenantID, nil
	}
	a.mu.RUnlock()

	// RPC fallback
	rpcCtx := a.withPeerAuth(ctx)
	resp, err := a.client.ResolveOrg(rpcCtx, &pb.ResolveOrgRequest{
		OrgId: orgUUID.String(),
	})
	if err != nil {
		return 0, fmt.Errorf("identity RPC ResolveOrg failed for org %s: %w", orgUUID, err)
	}

	tenantID := resp.GetTenantId()

	// Update cache
	a.mu.Lock()
	if len(a.cache) >= a.maxSize {
		now := time.Now()
		for k, v := range a.cache {
			if now.After(v.expiresAt) {
				delete(a.cache, k)
			}
		}
		if len(a.cache) >= a.maxSize {
			a.cache = make(map[uuid.UUID]tenantCacheEntry, a.maxSize)
		}
	}
	a.cache[orgUUID] = tenantCacheEntry{tenantID: tenantID, expiresAt: time.Now().Add(a.ttl)}
	a.mu.Unlock()

	return tenantID, nil
}

// ResolveCert is not supported in the gateway (protocol-level cert auth is handled by KC-Core).
func (a *IdentityRPCOrgAdapter) ResolveCert(_ context.Context, _ *x509.Certificate) (uuid.UUID, int64, error) {
	return uuid.Nil, 0, fmt.Errorf("gateway does not support certificate-based org resolution")
}

// GetDefaultOrgForTenant returns the default organization UUID for a tenant.
func (a *IdentityRPCOrgAdapter) GetDefaultOrgForTenant(ctx context.Context, tenantID int64) (uuid.UUID, error) {
	rpcCtx := a.withPeerAuth(ctx)
	resp, err := a.client.GetDefaultOrgForTenant(rpcCtx, &pb.GetDefaultOrgForTenantRequest{
		TenantId: tenantID,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("identity RPC GetDefaultOrgForTenant failed for tenant %d: %w", tenantID, err)
	}

	orgUUID, err := uuid.Parse(resp.GetOrgId())
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid org UUID from identity service: %w", err)
	}

	return orgUUID, nil
}

// ResolveOrganization maps a tenant ID to its default organization UUID.
// Satisfies interceptors.OrganizationResolver.
func (a *IdentityRPCOrgAdapter) ResolveOrganization(ctx context.Context, tenantID int64) (uuid.UUID, error) {
	return a.GetDefaultOrgForTenant(ctx, tenantID)
}

// IsServerAdmin checks if the given user is a server admin via KC-Identity.
func (a *IdentityRPCOrgAdapter) IsServerAdmin(ctx context.Context, userID string) (bool, error) {
	rpcCtx := a.withPeerAuth(ctx)
	resp, err := a.client.CheckServerAdmin(rpcCtx, &pb.CheckServerAdminRequest{UserId: userID})
	if err != nil {
		return false, err
	}
	return resp.IsAdmin, nil
}

// withPeerAuth creates a new outgoing context with peer secret metadata.
func (a *IdentityRPCOrgAdapter) withPeerAuth(ctx context.Context) context.Context {
	if a.peerSecret == "" {
		return ctx
	}
	md := metadata.Pairs(grpcconstants.MetadataKeyInternalPeerSecret, a.peerSecret)
	return metadata.NewOutgoingContext(ctx, md)
}
