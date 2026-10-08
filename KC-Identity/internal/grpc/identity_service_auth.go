// Package grpc provides gRPC service implementations.
package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
)

// Login authenticates a user and returns tokens with user profile in a single round-trip.
func (s *IdentityService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if s.authSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	// Validate request
	if req.Email == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEmailRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEmailRequired))
	}
	if req.Password == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenPasswordRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenPasswordRequired))
	}

	result, err := s.authSvc.Login(ctx, req.Email, req.Password)
	if err != nil {
		s.log.ErrorContext(ctx, LogLoginFailed, userFieldEmail, req.Email, logger.FieldError, err)
		s.emitSecurityEvent(ctx, models.EventTypeAuthLoginFailed, models.EventTitleAuthLoginFailed, opNameLogin, detailInvalidCredentialsForPrefix+req.Email)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidCredentials),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidCredentials))
	}

	// Convert profile memberships to proto
	var memberships []*pb.UserMembership
	for _, m := range result.Profile.Memberships {
		memberships = append(memberships, &pb.UserMembership{
			OrgId:              m.OrgID.String(),
			OrgName:            m.OrgName,
			Role:               m.Role,
			IsOrgAdmin:         m.IsOrgAdmin,
			IsBaseStationAdmin: m.IsBaseStationAdmin,
			IsEndpointAdmin:    m.IsEndpointAdmin,
		})
	}

	resp := &pb.LoginResponse{
		Tokens: &pb.AuthTokens{
			AccessToken:      result.Tokens.AccessToken,
			RefreshToken:     result.Tokens.RefreshToken,
			AccessExpiresIn:  result.Tokens.AccessExpiresIn,
			RefreshExpiresIn: result.Tokens.RefreshExpiresIn,
		},
		User: &pb.UserProfile{
			Id:          result.Profile.ID.String(),
			Email:       result.Profile.Email,
			IsAdmin:     result.Profile.IsAdmin,
			HasPassword: result.Profile.HasPassword,
			FirstName:   result.Profile.FirstName,
			LastName:    result.Profile.LastName,
			Memberships: memberships,
		},
	}
	if result.Profile.DefaultOrgID != nil {
		resp.User.DefaultOrgId = result.Profile.DefaultOrgID.String()
	}
	return resp, nil
}

// RefreshTokens refreshes the access token using a refresh token.
func (s *IdentityService) RefreshTokens(ctx context.Context, req *pb.RefreshTokensRequest) (*pb.RefreshTokensResponse, error) {
	if s.authSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.RefreshToken == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRefreshTokenRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRefreshTokenRequired))
	}

	tokens, err := s.authSvc.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		s.log.ErrorContext(ctx, LogTokenRefreshFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidRefreshToken),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidRefreshToken))
	}

	return &pb.RefreshTokensResponse{
		Tokens: &pb.AuthTokens{
			AccessToken:      tokens.AccessToken,
			RefreshToken:     tokens.RefreshToken,
			AccessExpiresIn:  tokens.AccessExpiresIn,
			RefreshExpiresIn: tokens.RefreshExpiresIn,
		},
	}, nil
}

// GetProfile returns the current user's profile.
func (s *IdentityService) GetProfile(ctx context.Context, _ *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	if s.authSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	userID, err := grpcerrors.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	profile, err := s.authSvc.GetProfile(ctx, userID)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetProfileFailed, logger.FieldUserIDSnake, userID, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetProfileFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetProfileFailed))
	}

	// Convert memberships to proto
	var memberships []*pb.UserMembership
	for _, m := range profile.Memberships {
		memberships = append(memberships, &pb.UserMembership{
			OrgId:              m.OrgID.String(),
			OrgName:            m.OrgName,
			Role:               m.Role,
			IsOrgAdmin:         m.IsOrgAdmin,
			IsBaseStationAdmin: m.IsBaseStationAdmin,
			IsEndpointAdmin:    m.IsEndpointAdmin,
		})
	}

	roles, err := s.profileRoles(ctx, userID, profile.DefaultOrgID)
	if err != nil {
		s.log.ErrorContext(ctx, LogRoleResolutionFailed, logger.FieldUserIDSnake, userID, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetProfileFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetProfileFailed))
	}

	resp := &pb.GetProfileResponse{
		User: &pb.UserProfile{
			Id:          profile.ID.String(),
			Email:       profile.Email,
			IsAdmin:     profile.IsAdmin,
			HasPassword: profile.HasPassword,
			FirstName:   profile.FirstName,
			LastName:    profile.LastName,
			Memberships: memberships,
			Roles:       rolesToProto(roles),
		},
	}
	if profile.DefaultOrgID != nil {
		resp.User.DefaultOrgId = profile.DefaultOrgID.String()
	}
	return resp, nil
}

// errRoleResolverNotConfigured reports a profile request on a service built without a role resolver.
var errRoleResolverNotConfigured = grpcerrors.NewTokenError(grpcerrors.ErrTokenServiceNotConfigured, nil)

// profileRoles resolves the caller's roles in the organization it is acting
// in, falling back to its default organization.
func (s *IdentityService) profileRoles(ctx context.Context, userID uuid.UUID, defaultOrgID *uuid.UUID) (authz.Roles, error) {
	if s.roles == nil {
		return authz.Roles{}, errRoleResolverNotConfigured
	}
	orgID, err := pkgcontext.GetOrganizationID(ctx)
	if err != nil && defaultOrgID != nil {
		orgID = *defaultOrgID
	}
	return s.roles.Resolve(ctx, userID, orgID)
}

// GetAuthSettings returns authentication settings (public endpoint).
func (s *IdentityService) GetAuthSettings(ctx context.Context, _ *pb.GetAuthSettingsRequest) (*pb.GetAuthSettingsResponse, error) {
	if s.authSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	settings, err := s.authSvc.GetAuthSettings(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetAuthSettingsFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetAuthSettingsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetAuthSettingsFailed))
	}

	resp := &pb.GetAuthSettingsResponse{
		Settings: &pb.AuthSettings{
			Enabled:             settings.Enabled,
			LocalLoginEnabled:   settings.LocalLoginEnabled,
			LoginUrl:            settings.LoginURL,
			LoginLabel:          settings.LoginLabel,
			LoginRedirect:       settings.LoginRedirect,
			LogoutUrl:           settings.LogoutURL,
			RefreshTokenEnabled: settings.RefreshTokenEnabled,
			RegistrationEnabled: s.registrationSvc != nil,
			PasswordPolicy:      passwordPolicyToProto(settings.PasswordPolicy),
		},
	}
	if settings.OIDCEnabled {
		resp.Settings.Oidc = &pb.ProviderSettings{
			Enabled:  settings.OIDCEnabled,
			LoginUrl: settings.OIDCProviderURL,
		}
	}
	return resp, nil
}

func passwordPolicyToProto(policy grpcservices.PasswordPolicy) *pb.PasswordPolicy {
	return &pb.PasswordPolicy{
		MinLength:      policy.MinLength,
		MaxLength:      policy.MaxLength,
		RequiresLetter: policy.RequiresLetter,
		RequiresDigit:  policy.RequiresDigit,
	}
}

// Logout invalidates the user's tokens.
func (s *IdentityService) Logout(ctx context.Context, _ *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	if s.authSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	userID, err := grpcerrors.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.authSvc.Logout(ctx, userID); err != nil {
		s.log.ErrorContext(ctx, LogLogoutFailed, logger.FieldUserIDSnake, userID, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenLogoutFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLogoutFailed))
	}

	return &pb.LogoutResponse{Success: true}, nil
}

// buildLoginResponse builds a LoginResponse from an AuthLoginResult.
// Returns error if result, tokens, or profile are nil.
func buildLoginResponse(result *grpcservices.AuthLoginResult) (*pb.LoginResponse, error) {
	if result == nil || result.Tokens == nil || result.Profile == nil {
		return nil, errors.New(grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}

	var memberships []*pb.UserMembership
	for _, m := range result.Profile.Memberships {
		memberships = append(memberships, &pb.UserMembership{
			OrgId:              m.OrgID.String(),
			OrgName:            m.OrgName,
			Role:               m.Role,
			IsOrgAdmin:         m.IsOrgAdmin,
			IsBaseStationAdmin: m.IsBaseStationAdmin,
			IsEndpointAdmin:    m.IsEndpointAdmin,
		})
	}

	resp := &pb.LoginResponse{
		Tokens: &pb.AuthTokens{
			AccessToken:      result.Tokens.AccessToken,
			RefreshToken:     result.Tokens.RefreshToken,
			AccessExpiresIn:  result.Tokens.AccessExpiresIn,
			RefreshExpiresIn: result.Tokens.RefreshExpiresIn,
		},
		User: &pb.UserProfile{
			Id:          result.Profile.ID.String(),
			Email:       result.Profile.Email,
			IsAdmin:     result.Profile.IsAdmin,
			HasPassword: result.Profile.HasPassword,
			FirstName:   result.Profile.FirstName,
			LastName:    result.Profile.LastName,
			Memberships: memberships,
		},
	}
	if result.Profile.DefaultOrgID != nil {
		resp.User.DefaultOrgId = result.Profile.DefaultOrgID.String()
	}
	return resp, nil
}
