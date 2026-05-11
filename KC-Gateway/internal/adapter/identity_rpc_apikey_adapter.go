// Package adapter bridges KC-Identity RPC responses to interceptor interfaces.
package adapter

import (
	"context"
	"fmt"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcconstants "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

// IdentityRPCAPIKeyAdapter validates API keys via KC-Identity's IdentityInternalService.
type IdentityRPCAPIKeyAdapter struct {
	client     pb.IdentityInternalServiceClient
	peerSecret string
}

// NewIdentityRPCAPIKeyAdapter creates an API key adapter that calls KC-Identity RPCs.
func NewIdentityRPCAPIKeyAdapter(client pb.IdentityInternalServiceClient, peerSecret string) *IdentityRPCAPIKeyAdapter {
	return &IdentityRPCAPIKeyAdapter{client: client, peerSecret: peerSecret}
}

// LookupByHash retrieves and converts an API key by its SHA-256 hash via RPC.
func (a *IdentityRPCAPIKeyAdapter) LookupByHash(ctx context.Context, hash string) (*interceptors.APIKeyRecord, error) {
	rpcCtx := a.withPeerAuth(ctx)
	resp, err := a.client.ValidateAPIKey(rpcCtx, &pb.ValidateAPIKeyRequest{
		KeyHash: hash,
	})
	if err != nil {
		return nil, fmt.Errorf("identity RPC ValidateAPIKey failed: %w", err)
	}

	id, err := uuid.Parse(resp.GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid API key ID from identity service: %w", err)
	}

	orgID, err := uuid.Parse(resp.GetOrganizationId())
	if err != nil {
		return nil, fmt.Errorf("invalid org ID from identity service: %w", err)
	}

	var userID *uuid.UUID
	if resp.GetUserId() != "" {
		parsed, err := uuid.Parse(resp.GetUserId())
		if err != nil {
			return nil, fmt.Errorf("invalid user ID from identity service: %w", err)
		}
		userID = &parsed
	}

	return &interceptors.APIKeyRecord{
		ID:             id,
		TenantID:       resp.GetTenantId(),
		OrganizationID: orgID,
		UserID:         userID,
		IsActive:       resp.GetIsActive(),
		IsExpired:      resp.GetIsExpired(),
	}, nil
}

// UpdateLastUsed fires-and-forgets an update to the API key's last used timestamp via RPC.
func (a *IdentityRPCAPIKeyAdapter) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	rpcCtx := a.withPeerAuth(ctx)
	_, err := a.client.UpdateAPIKeyLastUsed(rpcCtx, &pb.UpdateAPIKeyLastUsedRequest{
		Id: id.String(),
	})
	return err
}

// withPeerAuth creates a new outgoing context with peer secret metadata.
func (a *IdentityRPCAPIKeyAdapter) withPeerAuth(ctx context.Context) context.Context {
	if a.peerSecret == "" {
		return ctx
	}
	md := metadata.Pairs(grpcconstants.MetadataKeyInternalPeerSecret, a.peerSecret)
	return metadata.NewOutgoingContext(ctx, md)
}
