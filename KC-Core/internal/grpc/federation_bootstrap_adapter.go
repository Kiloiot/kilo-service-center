package grpc

import (
	"context"
	"errors"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	pkgfederation "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/federation"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
)

// CEBootstrapAdapter presents the domain onboarding service over the generated
// protobuf contract. Keeping the translation here is what allows the service
// itself to carry no transport types.
type CEBootstrapAdapter struct {
	svc pkgfederation.CEBootstrapService
}

// NewCEBootstrapAdapter wraps a domain onboarding service for the gRPC surface.
func NewCEBootstrapAdapter(svc pkgfederation.CEBootstrapService) *CEBootstrapAdapter {
	return &CEBootstrapAdapter{svc: svc}
}

// GetCEStatus reports onboarding and relay state.
func (a *CEBootstrapAdapter) GetCEStatus(ctx context.Context, _ *pb.GetCEStatusRequest) (*pb.GetCEStatusResponse, error) {
	st, err := a.svc.Status(ctx)
	if err != nil {
		return nil, bootstrapStatusError(err)
	}

	return &pb.GetCEStatusResponse{
		OnboardingRequired:  st.OnboardingRequired,
		CeId:                st.CEID,
		CompanyName:         st.CompanyName,
		FederationConnected: st.FederationConnected,
	}, nil
}

// CompleteCEOnboarding records the company name and activates the relay.
func (a *CEBootstrapAdapter) CompleteCEOnboarding(ctx context.Context, req *pb.CompleteCEOnboardingRequest) (*pb.CompleteCEOnboardingResponse, error) {
	result, err := a.svc.CompleteOnboarding(ctx, req.GetCompanyName())
	if err != nil {
		return nil, bootstrapOnboardingError(err)
	}

	return &pb.CompleteCEOnboardingResponse{
		CeId:        result.CEID,
		CompanyName: result.CompanyName,
	}, nil
}

// bootstrapStatusError maps a status failure onto the catalog and renders it
// as a gRPC status carrying only catalog vocabulary.
func bootstrapStatusError(err error) error {
	if errors.Is(err, pkgfederation.ErrNotCommunityEdition) {
		return grpcerrors.ToStatusError(grpcerrors.NewTokenError(grpcerrors.ErrTokenCEStatusUnavailable, err))
	}
	return grpcerrors.ToStatusError(grpcerrors.NewTokenError(grpcerrors.ErrTokenInternalError, err))
}

// bootstrapOnboardingError maps an onboarding failure onto the catalog and
// renders it as a gRPC status carrying only catalog vocabulary.
func bootstrapOnboardingError(err error) error {
	switch {
	case errors.Is(err, pkgfederation.ErrNotCommunityEdition):
		return grpcerrors.ToStatusError(grpcerrors.NewTokenError(grpcerrors.ErrTokenCEOnboardingUnavailable, err))
	case errors.Is(err, pkgfederation.ErrCompanyNameRequired):
		return grpcerrors.ToStatusError(grpcerrors.NewTokenError(grpcerrors.ErrTokenCompanyNameRequired, err))
	case errors.Is(err, pkgfederation.ErrOnboardingAlreadyCompleted):
		return grpcerrors.ToStatusError(grpcerrors.NewTokenError(grpcerrors.ErrTokenCEOnboardingCompleted, err))
	default:
		return grpcerrors.ToStatusError(grpcerrors.NewTokenError(grpcerrors.ErrTokenInternalError, err))
	}
}
