package grpc

import (
	"context"
	"sync"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type cachedRoles struct {
	roles     authz.Roles
	expiresAt time.Time
}

// identityRoleSource resolves a caller's roles through KC-Identity and keeps
// them for cacheTTL, which bounds how long a role change takes to apply.
type identityRoleSource struct {
	client     pb.IdentityInternalServiceClient
	peerSecret string
	cacheTTL   time.Duration
	cache      sync.Map // "orgID:kind:principalID" -> cachedRoles
}

// NewIdentityRoleSource creates the RoleSource backed by KC-Identity's GetUserRoles RPC.
func NewIdentityRoleSource(client pb.IdentityInternalServiceClient, peerSecret string, cacheTTL time.Duration) interceptors.RoleSource {
	return &identityRoleSource{client: client, peerSecret: peerSecret, cacheTTL: cacheTTL}
}

const (
	roleCacheKeySeparator       = ":"
	principalKindUser           = "user"
	principalKindServiceAccount = "service_account"
)

// rolesRequest names the caller: the signed-in user, or else the
// service-account key it authenticated with. A request without an
// organization (only org-exempt methods get this far without one) is judged
// on the caller's organization-independent roles, which never exceed what any
// organization grants. It returns the request and the cache key for the answer.
func rolesRequest(ctx context.Context) (*pb.GetUserRolesRequest, string, error) {
	var org string
	if orgID, err := pkgcontext.RequireOrganizationID(ctx); err == nil {
		org = orgID.String()
	}
	if userID, err := pkgcontext.GetUserID(ctx); err == nil {
		return &pb.GetUserRolesRequest{OrgId: org, UserId: userID},
			org + roleCacheKeySeparator + principalKindUser + roleCacheKeySeparator + userID, nil
	}
	keyID, err := pkgcontext.GetServiceAccountID(ctx)
	if err != nil {
		return nil, "", grpcconst.NewTokenError(grpcconst.ErrTokenMissingUserCtx, err)
	}
	return &pb.GetUserRolesRequest{OrgId: org, ServiceAccountId: keyID.String()},
		org + roleCacheKeySeparator + principalKindServiceAccount + roleCacheKeySeparator + keyID.String(), nil
}

func (s *identityRoleSource) Roles(ctx context.Context) (authz.Roles, error) {
	req, key, err := rolesRequest(ctx)
	if err != nil {
		return authz.Roles{}, err
	}
	if cached, ok := s.cache.Load(key); ok {
		entry := cached.(cachedRoles)
		if time.Now().Before(entry.expiresAt) {
			return entry.roles, nil
		}
		s.cache.Delete(key)
	}

	if s.peerSecret != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, grpcconst.MetadataKeyInternalPeerSecret, s.peerSecret)
	}
	resp, err := s.client.GetUserRoles(ctx, req)
	var roles authz.Roles
	switch {
	case status.Code(err) == codes.NotFound:
		// An unknown user or key holds no roles.
	case err != nil:
		return authz.Roles{}, grpcconst.NewTokenError(grpcconst.ErrTokenInternalError, err)
	default:
		roles = rolesFromProto(resp.GetRoles())
	}
	s.cache.Store(key, cachedRoles{roles: roles, expiresAt: time.Now().Add(s.cacheTTL)})
	return roles, nil
}

func rolesFromProto(r *pb.UserRoles) authz.Roles {
	return authz.Roles{
		Admin:              r.GetAdmin(),
		TenantManager:      r.GetTenantManager(),
		BaseStationManager: r.GetBaseStationManager(),
		EndpointManager:    r.GetEndpointManager(),
	}
}

// OrgMapper maps internal tenant IDs to external organization UUIDs.
type OrgMapper interface {
	GetDefaultOrgForTenant(ctx context.Context, tenantID int64) (string, error)
}

// adminOrgAdapter implements the org resolver's AdminChecker and OrgMapper via KC-Identity internal RPC.
type adminOrgAdapter struct {
	client     pb.IdentityInternalServiceClient
	peerSecret string
}

// NewAdminOrgAdapter creates an adapter for admin checks and org mapping.
func NewAdminOrgAdapter(client pb.IdentityInternalServiceClient, peerSecret string) *adminOrgAdapter { //nolint:revive // unexported return is intentional; callers use the interface
	return &adminOrgAdapter{client: client, peerSecret: peerSecret}
}

func (a *adminOrgAdapter) IsServerAdmin(ctx context.Context, userID string) (bool, error) {
	if a.peerSecret != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, grpcconst.MetadataKeyInternalPeerSecret, a.peerSecret)
	}
	resp, err := a.client.CheckServerAdmin(ctx, &pb.CheckServerAdminRequest{UserId: userID})
	if err != nil {
		return false, err
	}
	return resp.IsAdmin, nil
}

func (a *adminOrgAdapter) GetDefaultOrgForTenant(ctx context.Context, tenantID int64) (string, error) {
	if a.peerSecret != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, grpcconst.MetadataKeyInternalPeerSecret, a.peerSecret)
	}
	resp, err := a.client.GetDefaultOrgForTenant(ctx, &pb.GetDefaultOrgForTenantRequest{TenantId: tenantID})
	if err != nil {
		return "", err
	}
	return resp.OrgId, nil
}
