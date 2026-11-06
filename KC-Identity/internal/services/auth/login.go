package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
)

// Login authenticates user and returns tokens with user profile.
func (s *Service) Login(ctx context.Context, email, password string) (*grpcservices.AuthLoginResult, error) {
	email = NormalizeEmail(email)
	s.logger.InfoContext(ctx, logAuthLoginAttempt)

	if !s.localLoginEnabled {
		s.logger.WarnContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldReason, reasonLocalLoginDisabled)
		return nil, ErrLocalLoginDisabled
	}

	// Defensive nil check: tokenIssuer may be nil when token generation is disabled
	if s.tokenIssuer == nil {
		s.logger.WarnContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldReason, reasonTokenGenDisabled)
		return nil, ErrTokenGenerationDisabled
	}

	user, err := s.verifiedUser(ctx, email, password)
	if err != nil {
		return nil, err
	}

	memberships, err := s.loginMemberships(ctx, user, email)
	if err != nil {
		return nil, err
	}
	defaultOrgID := memberships[0].OrgID

	refreshToken, err := s.openRefreshSession(ctx, user, email)
	if err != nil {
		return nil, err
	}
	accessToken, err := s.tokenIssuer.IssueAccessToken(user.ID, &defaultOrgID)
	if err != nil {
		s.logger.ErrorContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, logAuthLoginSuccess, logger.FieldEmail, email, logger.FieldUserIDSnake, user.ID)
	return &grpcservices.AuthLoginResult{
		Tokens: &grpcservices.AuthTokens{
			AccessToken:      accessToken,
			RefreshToken:     refreshToken,
			AccessExpiresIn:  s.tokenIssuer.GetAccessTTL(),
			RefreshExpiresIn: s.tokenIssuer.GetRefreshTTL(),
		},
		Profile: newLoginProfile(user, memberships, defaultOrgID),
	}, nil
}

// verifiedUser returns the active user with email whose password matches.
func (s *Service) verifiedUser(ctx context.Context, email, password string) (*models.User, error) {
	user, err := s.userStore.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal whether email exists
		s.logger.WarnContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldReason, reasonUserNotFound)
		return nil, ErrInvalidCredentials
	}
	if err := requireActive(user); err != nil {
		s.logger.WarnContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldReason, reasonUserInactive)
		return nil, err
	}
	if user.PasswordHash == nil || *user.PasswordHash == "" {
		s.logger.WarnContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldReason, reasonNoPasswordSet)
		return nil, ErrNoPasswordSet
	}
	if err := VerifyPassword(password, *user.PasswordHash); err != nil {
		s.logger.WarnContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldReason, reasonInvalidPassword)
		return nil, ErrInvalidCredentials
	}
	return user, nil
}

// loginMemberships returns the user's memberships, first the default one; CE
// synthesizes its default organization for a user without any.
func (s *Service) loginMemberships(ctx context.Context, user *models.User, email string) ([]*models.OrganizationMembershipWithOrg, error) {
	memberships, err := s.membershipStore.ListUserMemberships(ctx, user.ID)
	if err != nil {
		s.logger.ErrorContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldError, err)
		return nil, err
	}
	if len(memberships) > 0 {
		return memberships, nil
	}
	if s.ceProvider == nil {
		s.logger.WarnContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldReason, reasonNoMemberships)
		return nil, ErrMembershipRequired
	}
	ceMembership, err := s.ceProvider.SynthesizeMembership(ctx, user)
	if err != nil {
		s.logger.ErrorContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldError, err)
		return nil, err
	}
	return []*models.OrganizationMembershipWithOrg{{
		OrgID:              ceMembership.OrgID,
		OrgName:            ceMembership.OrgName,
		Role:               ceMembership.Role,
		IsOrgAdmin:         ceMembership.IsOrgAdmin,
		IsBaseStationAdmin: ceMembership.IsBaseStationAdmin,
		IsEndpointAdmin:    ceMembership.IsEndpointAdmin,
	}}, nil
}

// openRefreshSession issues and stores a refresh token for the verified user;
// it returns no token when refresh tokens are off.
func (s *Service) openRefreshSession(ctx context.Context, user *models.User, email string) (string, error) {
	if !s.refreshTokenEnabled {
		return "", nil
	}
	refreshToken, err := s.tokenIssuer.IssueRefreshToken(user.ID)
	if err != nil {
		s.logger.ErrorContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldError, err)
		return "", err
	}
	rt := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: HashRefreshToken(refreshToken),
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: s.tokenIssuer.GetRefreshExpiresAt(),
	}
	if err := s.refreshTokenStore.CreateForPassword(ctx, rt, *user.PasswordHash); err != nil {
		return "", s.sessionRefused(ctx, email, err)
	}
	return refreshToken, nil
}

// sessionRefused maps a refused session to the error the caller sees: the
// user was deactivated, or their password changed, after it was verified.
func (s *Service) sessionRefused(ctx context.Context, email string, err error) error {
	switch {
	case errors.Is(err, storage.ErrUserInactive):
		s.logger.WarnContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldReason, reasonUserInactive)
		return ErrUserNotActive
	case errors.Is(err, storage.ErrUserCredentialsChanged):
		s.logger.WarnContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldReason, reasonCredentialsChanged)
		return ErrInvalidCredentials
	default:
		s.logger.ErrorContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errPrefixOpenRefreshSession, err)
	}
}
