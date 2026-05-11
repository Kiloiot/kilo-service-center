package grpc

import (
	"context"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	"google.golang.org/grpc/metadata"
)

// roleResolverAdapter implements interceptors.RoleResolver via KC-Identity's internal RPC.
type roleResolverAdapter struct {
	client     pb.IdentityInternalServiceClient
	peerSecret string
}

// NewRoleResolver creates a RoleResolver that calls KC-Identity's GetUserMembership RPC.
func NewRoleResolver(client pb.IdentityInternalServiceClient, peerSecret string) interceptors.RoleResolver {
	return &roleResolverAdapter{client: client, peerSecret: peerSecret}
}

func (r *roleResolverAdapter) GetUserRole(ctx context.Context, orgID, userID string) (string, bool, error) {
	if r.peerSecret != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, grpcconst.MetadataKeyInternalPeerSecret, r.peerSecret)
	}

	resp, err := r.client.GetUserMembership(ctx, &pb.GetUserMembershipRequest{
		OrgId:  orgID,
		UserId: userID,
	})
	if err != nil {
		return "", false, err
	}

	return resp.Role, resp.IsActive, nil
}

// AdminChecker verifies whether a user has server-level admin privileges.
type AdminChecker interface {
	IsServerAdmin(ctx context.Context, userID string) (bool, error)
}

// OrgMapper maps internal tenant IDs to external organization UUIDs.
type OrgMapper interface {
	GetDefaultOrgForTenant(ctx context.Context, tenantID int64) (string, error)
}

// adminOrgAdapter implements AdminChecker and OrgMapper via KC-Identity internal RPC.
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
