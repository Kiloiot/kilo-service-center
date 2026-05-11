package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
)

// ValidateAPIKey looks up an API key by hash and returns its metadata.
func (s *IdentityInternalService) ValidateAPIKey(ctx context.Context, req *pb.ValidateAPIKeyRequest) (*pb.ValidateAPIKeyResponse, error) {
	if req.KeyHash == "" {
		return nil, status.Error(codes.InvalidArgument, "key_hash is required")
	}

	if s.apiKeyRepo == nil {
		return nil, status.Error(codes.Internal, "API key service not configured")
	}

	info, err := s.apiKeyRepo.GetByHash(ctx, req.KeyHash)
	if err != nil {
		return nil, status.Error(codes.NotFound, "API key not found")
	}

	resp := &pb.ValidateAPIKeyResponse{
		Id:             info.ID.String(),
		TenantId:       info.TenantID,
		OrganizationId: info.OrganizationID.String(),
		IsActive:       info.IsActive,
		IsExpired:      info.IsExpired,
	}
	if info.UserID != uuid.Nil {
		resp.UserId = info.UserID.String()
	}
	return resp, nil
}

// UpdateAPIKeyLastUsed updates the last-used timestamp for an API key.
func (s *IdentityInternalService) UpdateAPIKeyLastUsed(ctx context.Context, req *pb.UpdateAPIKeyLastUsedRequest) (*pb.UpdateAPIKeyLastUsedResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if s.apiKeyRepo == nil {
		return &pb.UpdateAPIKeyLastUsedResponse{}, nil
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id format")
	}

	// Fire-and-forget semantics: log but don't fail on update errors
	if err := s.apiKeyRepo.UpdateLastUsed(ctx, id); err != nil {
		s.log.ErrorContext(ctx, "failed to update API key last used", "id", req.Id, "error", err)
	}

	return &pb.UpdateAPIKeyLastUsedResponse{}, nil
}
