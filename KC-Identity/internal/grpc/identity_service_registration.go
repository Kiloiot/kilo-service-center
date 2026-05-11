package grpc

import (
	"context"
	"strings"

	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
)

// RegisterAccount handles self-service account registration.
func (s *IdentityService) RegisterAccount(ctx context.Context, req *pb.RegisterAccountRequest) (*pb.LoginResponse, error) {
	if s.registrationSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistrationDisabled),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistrationDisabled))
	}

	if req.Email == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEmailRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEmailRequired))
	}
	if req.Password == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenPasswordRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenPasswordRequired))
	}
	if req.FirstName == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenFirstNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenFirstNameRequired))
	}
	if req.LastName == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenLastNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLastNameRequired))
	}
	// CompanyName is optional in CE (validated by registration service)
	// In ECE, the registration service enforces this requirement

	svcReq := &grpcservices.RegisterAccountRequest{
		Email:       req.Email,
		Password:    req.Password,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		CompanyName: req.CompanyName,
	}

	result, err := s.registrationSvc.RegisterAccount(ctx, svcReq)
	if err != nil {
		s.log.ErrorContext(ctx, "registration failed", logger.Err(err))
		return nil, mapRegistrationError(err)
	}

	resp, err := buildLoginResponse(result)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to build registration response", logger.Err(err))
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistrationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistrationFailed))
	}
	return resp, nil
}

// mapRegistrationError maps registration service errors to gRPC status codes.
func mapRegistrationError(err error) error {
	msg := err.Error()
	switch {
	case containsToken(msg, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistrationDisabled)):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistrationDisabled),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistrationDisabled))
	case containsToken(msg, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEmailAlreadyExists)):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidArgument),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidArgument))
	case containsToken(msg, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenWeakPassword)):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenWeakPassword),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenWeakPassword))
	case containsToken(msg, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenFirstNameRequired)):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenFirstNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenFirstNameRequired))
	case containsToken(msg, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLastNameRequired)):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenLastNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLastNameRequired))
	case containsToken(msg, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCompanyNameRequired)):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCompanyNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCompanyNameRequired))
	case containsToken(msg, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEmailInvalidFormat)):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEmailInvalidFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEmailInvalidFormat))
	case containsToken(msg, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenFieldTooLong)):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenFieldTooLong),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenFieldTooLong))
	default:
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistrationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistrationFailed))
	}
}

func containsToken(msg, token string) bool {
	return strings.Contains(msg, token)
}
