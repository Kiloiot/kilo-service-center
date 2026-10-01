package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
)

// ============================================================================
// External auth handlers (OIDC/OAuth2 exchange)
// ============================================================================

// ExchangeOIDC exchanges an OIDC authorization code for tokens.
func (s *IdentityService) ExchangeOIDC(ctx context.Context, req *pb.ExchangeOIDCRequest) (*pb.LoginResponse, error) {
	if s.externalAuthSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthOIDCDisabled),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthOIDCDisabled))
	}

	if req.Code == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCodeRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCodeRequired))
	}
	if req.State == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenStateRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenStateRequired))
	}

	result, err := s.externalAuthSvc.CompleteOIDCLogin(ctx, req.Code, req.State)
	if err != nil {
		s.log.ErrorContext(ctx, LogOIDCExchangeFailed, logger.FieldError, err)
		s.emitSecurityEvent(ctx, models.EventTypeAuthOIDCExchangeFailed, models.EventTitleAuthOIDCExchangeFailed, opNameExchangeOIDC, detailOIDCExchangeFailed)
		return nil, mapExternalAuthError(err)
	}

	resp, err := buildLoginResponse(result)
	if err != nil {
		s.log.ErrorContext(ctx, LogBuildLoginResponseFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}
	return resp, nil
}

// ExchangeOAuth2 exchanges an OAuth2 authorization code for tokens.
func (s *IdentityService) ExchangeOAuth2(ctx context.Context, req *pb.ExchangeOAuth2Request) (*pb.LoginResponse, error) {
	if s.externalAuthSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthOAuth2Disabled),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthOAuth2Disabled))
	}

	if req.Code == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCodeRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCodeRequired))
	}
	if req.State == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenStateRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenStateRequired))
	}

	result, err := s.externalAuthSvc.CompleteOAuth2Login(ctx, req.Code, req.State)
	if err != nil {
		s.log.ErrorContext(ctx, LogOAuth2ExchangeFailed, logger.FieldError, err)
		s.emitSecurityEvent(ctx, models.EventTypeAuthOAuth2ExchangeFailed, models.EventTitleAuthOAuth2ExchangeFailed, opNameExchangeOAuth2, detailOAuth2ExchangeFailed)
		return nil, mapExternalAuthError(err)
	}

	resp, err := buildLoginResponse(result)
	if err != nil {
		s.log.ErrorContext(ctx, LogBuildLoginResponseFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}
	return resp, nil
}

// mapExternalAuthError maps external auth service errors to gRPC status errors.
// Uses errors.Is() for robust error matching that handles wrapped errors.
func mapExternalAuthError(err error) error {
	switch {
	case errors.Is(err, auth.ErrOIDCProviderDisabled):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthOIDCDisabled),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthOIDCDisabled))
	case errors.Is(err, auth.ErrOAuth2ProviderDisabled):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthOAuth2Disabled),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthOAuth2Disabled))
	case errors.Is(err, auth.ErrStateNotFound):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthStateNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthStateNotFound))
	case errors.Is(err, auth.ErrInvalidState):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthStateInvalid),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthStateInvalid))
	case errors.Is(err, auth.ErrTokenExchangeFailed):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthTokenExchangeFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthTokenExchangeFailed))
	case errors.Is(err, auth.ErrIDTokenInvalid):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthIDTokenInvalid),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthIDTokenInvalid))
	case errors.Is(err, auth.ErrNonceMismatch):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthIDTokenInvalid),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthIDTokenInvalid))
	case errors.Is(err, auth.ErrEmailNotVerified):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthEmailNotVerified),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthEmailNotVerified))
	case errors.Is(err, auth.ErrRegistrationDisabled):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthRegistrationDisabled),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthRegistrationDisabled))
	case errors.Is(err, auth.ErrRedisUnavailable):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthRedisUnavailable),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthRedisUnavailable))
	case errors.Is(err, auth.ErrUserInfoFailed):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthUserInfoFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthUserInfoFailed))
	case errors.Is(err, auth.ErrOrgResolutionFailed):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthExternalOrgFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthExternalOrgFailed))
	case errors.Is(err, auth.ErrMembershipRequired):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMembershipRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMembershipRequired))
	default:
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}
}
