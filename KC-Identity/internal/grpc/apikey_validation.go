package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// Validation messages for the internal API-key RPCs that have no catalog
// token; the strings are part of the internal wire contract.
const (
	statusMsgKeyHashRequired            = "key_hash is required"
	statusMsgAPIKeyServiceNotConfigured = "API key service not configured" //nolint:gosec // G101: status text about service wiring, not a credential
)

// ValidateAPIKey looks up an API key by hash and returns its metadata.
func (s *IdentityInternalService) ValidateAPIKey(ctx context.Context, req *pb.ValidateAPIKeyRequest) (*pb.ValidateAPIKeyResponse, error) {
	if req.KeyHash == "" {
		return nil, status.Error(codes.InvalidArgument, statusMsgKeyHashRequired)
	}

	if s.apiKeyRepo == nil {
		return nil, status.Error(codes.Internal, statusMsgAPIKeyServiceNotConfigured)
	}

	info, err := s.apiKeyRepo.GetByHash(ctx, req.KeyHash)
	if err != nil {
		return nil, status.Error(codes.NotFound, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenApiKeyNotFound))
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
		return nil, status.Error(codes.InvalidArgument, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	if s.apiKeyRepo == nil {
		return &pb.UpdateAPIKeyLastUsedResponse{}, nil
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidIDFormat))
	}

	// Fire-and-forget semantics: log but don't fail on update errors
	if err := s.apiKeyRepo.UpdateLastUsed(ctx, id); err != nil {
		s.log.ErrorContext(ctx, LogAPIKeyLastUsedUpdateFailed, logger.FieldID, req.Id, logger.FieldError, err)
	}

	return &pb.UpdateAPIKeyLastUsedResponse{}, nil
}
