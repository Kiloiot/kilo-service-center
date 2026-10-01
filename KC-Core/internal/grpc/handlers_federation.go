package grpc

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"google.golang.org/grpc/status"
)

// FederationHandlers serves the CE bootstrap and registry RPCs.
type FederationHandlers struct {
	ceBootstrapSvc CEBootstrapHandler
	ceRegistrySvc  CERegistryHandler
	log            logger.Logger
}

// FederationHandlerDeps wires FederationHandlers; nil handlers answer their
// RPCs as unavailable.
type FederationHandlerDeps struct {
	Bootstrap CEBootstrapHandler
	Registry  CERegistryHandler
}

// NewFederationHandlers builds the group.
func NewFederationHandlers(d FederationHandlerDeps, log logger.Logger) *FederationHandlers {
	return &FederationHandlers{
		ceBootstrapSvc: d.Bootstrap,
		ceRegistrySvc:  d.Registry,
		log:            log,
	}
}

// GetCEStatus returns CE installation status. Only meaningful in CE mode.
func (s *FederationHandlers) GetCEStatus(ctx context.Context, req *pb.GetCEStatusRequest) (*pb.GetCEStatusResponse, error) {
	if s.ceBootstrapSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCEStatusUnavailable),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCEStatusUnavailable))
	}
	return s.ceBootstrapSvc.GetCEStatus(ctx, req)
}

// CompleteCEOnboarding stores the company name and completes CE onboarding. CE mode only.
func (s *FederationHandlers) CompleteCEOnboarding(ctx context.Context, req *pb.CompleteCEOnboardingRequest) (*pb.CompleteCEOnboardingResponse, error) {
	if s.ceBootstrapSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCEOnboardingUnavailable),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCEOnboardingUnavailable))
	}
	return s.ceBootstrapSvc.CompleteCEOnboarding(ctx, req)
}

// ListCEInstances returns paginated CE registry entries. ECE mode only.
// Restricted to server admins.
func (s *FederationHandlers) ListCEInstances(ctx context.Context, req *pb.ListCEInstancesRequest) (*pb.ListCEInstancesResponse, error) {
	if s.ceRegistrySvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCERegistryUnavailable),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCERegistryUnavailable))
	}

	return s.ceRegistrySvc.ListCEInstances(ctx, req)
}

// RevokeCEInstance marks a CE instance as revoked. ECE mode only.
// Restricted to server admins.
func (s *FederationHandlers) RevokeCEInstance(ctx context.Context, req *pb.RevokeCEInstanceRequest) (*pb.RevokeCEInstanceResponse, error) {
	if s.ceRegistrySvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCERegistryUnavailable),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCERegistryUnavailable))
	}

	return s.ceRegistrySvc.RevokeCEInstance(ctx, req)
}

// CEBootstrapHandler is the canonical interface from pkg/grpc.
type CEBootstrapHandler = grpcerrors.CEBootstrapHandler

// CERegistryHandler is the canonical interface from pkg/grpc.
type CERegistryHandler = grpcerrors.CERegistryHandler
