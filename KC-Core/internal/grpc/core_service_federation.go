package grpc

import (
	"context"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetCEStatus returns CE installation status. Only meaningful in CE mode.
func (s *CoreService) GetCEStatus(ctx context.Context, req *pb.GetCEStatusRequest) (*pb.GetCEStatusResponse, error) {
	if s.ceBootstrapSvc == nil {
		return nil, status.Error(codes.Unimplemented, "CE status not available in this edition")
	}
	return s.ceBootstrapSvc.GetCEStatus(ctx, req)
}

// CompleteCEOnboarding stores the company name and completes CE onboarding. CE mode only.
func (s *CoreService) CompleteCEOnboarding(ctx context.Context, req *pb.CompleteCEOnboardingRequest) (*pb.CompleteCEOnboardingResponse, error) {
	if s.ceBootstrapSvc == nil {
		return nil, status.Error(codes.Unimplemented, "CE onboarding not available in this edition")
	}
	return s.ceBootstrapSvc.CompleteCEOnboarding(ctx, req)
}

// ListCEInstances returns paginated CE registry entries. ECE mode only.
// Restricted to server admins.
func (s *CoreService) ListCEInstances(ctx context.Context, req *pb.ListCEInstancesRequest) (*pb.ListCEInstancesResponse, error) {
	if s.ceRegistrySvc == nil {
		return nil, status.Error(codes.Unimplemented, "CE registry not available in this edition")
	}

	userID, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.adminChecker == nil {
		s.log.ErrorContext(ctx, "admin checker not configured")
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}
	isAdmin, err := s.adminChecker.IsServerAdmin(ctx, userID.String())
	if err != nil {
		s.log.ErrorContext(ctx, "server admin check failed", "user_id", userID, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}
	if !isAdmin {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAdminRequired))
	}

	return s.ceRegistrySvc.ListCEInstances(ctx, req)
}

// RevokeCEInstance marks a CE instance as revoked. ECE mode only.
// Restricted to server admins.
func (s *CoreService) RevokeCEInstance(ctx context.Context, req *pb.RevokeCEInstanceRequest) (*pb.RevokeCEInstanceResponse, error) {
	if s.ceRegistrySvc == nil {
		return nil, status.Error(codes.Unimplemented, "CE registry not available in this edition")
	}

	userID, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.adminChecker == nil {
		s.log.ErrorContext(ctx, "admin checker not configured")
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}
	isAdmin, err := s.adminChecker.IsServerAdmin(ctx, userID.String())
	if err != nil {
		s.log.ErrorContext(ctx, "server admin check failed", "user_id", userID, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}
	if !isAdmin {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAdminRequired))
	}

	return s.ceRegistrySvc.RevokeCEInstance(ctx, req)
}
