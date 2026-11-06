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

// RefreshTokens exchanges refresh token for new token pair.
func (s *Service) RefreshTokens(ctx context.Context, refreshToken string) (*grpcservices.AuthTokens, error) {
	s.logger.InfoContext(ctx, logAuthTokenRefresh)

	if err := s.refreshAvailable(); err != nil {
		return nil, err
	}

	userID, _, err := s.tokenIssuer.ParseRefreshToken(refreshToken)
	if err != nil {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonInvalidToken)
		return nil, err
	}

	storedToken, err := s.usableRefreshToken(ctx, userID, HashRefreshToken(refreshToken))
	if err != nil {
		return nil, err
	}

	user, err := s.activeRefreshUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	defaultOrgID, err := s.refreshDefaultOrg(ctx, user)
	if err != nil {
		return nil, err
	}

	return s.rotateRefreshToken(ctx, storedToken, user.ID, defaultOrgID)
}

// refreshAvailable reports whether this deployment renews tokens at all.
func (s *Service) refreshAvailable() error {
	if !s.localLoginEnabled {
		return ErrLocalLoginDisabled
	}
	if !s.refreshTokenEnabled {
		return ErrRefreshTokenDisabled
	}
	// Defensive nil check: tokenIssuer may be nil when token generation is disabled
	if s.tokenIssuer == nil {
		return ErrTokenGenerationDisabled
	}
	return nil
}

// usableRefreshToken returns the stored token with tokenHash when it is
// neither replaced, revoked nor expired. Presenting a replaced token is reuse,
// which revokes every token of the user.
func (s *Service) usableRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string) (*models.RefreshToken, error) {
	storedToken, err := s.refreshTokenStore.GetByHash(ctx, tokenHash)
	if err != nil {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonTokenLookupFailed, logger.FieldError, err)
		return nil, ErrInvalidRefreshToken
	}
	if storedToken == nil {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonTokenNotFound)
		return nil, ErrInvalidRefreshToken
	}

	if storedToken.IsReplaced() {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonTokenReuseDetected, logger.FieldUserIDSnake, userID)
		s.revokeUserTokens(ctx, userID)
		return nil, ErrRefreshTokenReused
	}
	if storedToken.IsRevoked() {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonTokenRevoked)
		return nil, ErrRefreshTokenRevoked
	}
	if storedToken.IsExpired() {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonTokenExpired)
		return nil, ErrInvalidRefreshToken
	}
	return storedToken, nil
}

// activeRefreshUser loads the token owner and refuses a deactivated one,
// ending every session it still holds.
func (s *Service) activeRefreshUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonUserNotFound, logger.FieldUserIDSnake, userID)
		return nil, ErrInvalidRefreshToken
	}
	if err := requireActive(user); err != nil {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonUserInactive, logger.FieldUserIDSnake, userID)
		s.revokeUserTokens(ctx, userID)
		return nil, err
	}
	return user, nil
}

// revokeUserTokens ends every refresh-token session of the user; a failure is
// logged because the caller refuses the request either way.
func (s *Service) revokeUserTokens(ctx context.Context, userID uuid.UUID) {
	if err := s.refreshTokenStore.RevokeByUserID(ctx, userID); err != nil {
		s.logger.ErrorContext(ctx, logAuthRevokeTokenFailed, logger.FieldUserIDSnake, userID, logger.FieldError, err)
	}
}

// refreshDefaultOrg returns the organization the renewed access token is
// scoped to: the user's first membership, or the CE default organization.
func (s *Service) refreshDefaultOrg(ctx context.Context, user *models.User) (uuid.UUID, error) {
	memberships, err := s.membershipStore.ListUserMemberships(ctx, user.ID)
	if err != nil {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonNoMemberships, logger.FieldUserIDSnake, user.ID)
		return uuid.Nil, ErrMembershipRequired
	}
	if len(memberships) > 0 {
		return memberships[0].OrgID, nil
	}
	if s.ceProvider == nil {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonNoMemberships, logger.FieldUserIDSnake, user.ID)
		return uuid.Nil, ErrMembershipRequired
	}
	ceMembership, err := s.ceProvider.SynthesizeMembership(ctx, user)
	if err != nil {
		return uuid.Nil, err
	}
	return ceMembership.OrgID, nil
}

// rotateRefreshToken issues a new token pair and atomically replaces the
// presented refresh token with the new one.
func (s *Service) rotateRefreshToken(ctx context.Context, old *models.RefreshToken, userID, defaultOrgID uuid.UUID) (*grpcservices.AuthTokens, error) {
	newAccessToken, err := s.tokenIssuer.IssueAccessToken(userID, &defaultOrgID)
	if err != nil {
		return nil, err
	}
	newRefreshToken, err := s.tokenIssuer.IssueRefreshToken(userID)
	if err != nil {
		return nil, err
	}

	newRT := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: HashRefreshToken(newRefreshToken),
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: s.tokenIssuer.GetRefreshExpiresAt(),
	}
	if err := s.refreshTokenStore.Rotate(ctx, old.ID, newRT); err != nil {
		return nil, s.rotationRefused(ctx, userID, err)
	}

	s.logger.InfoContext(ctx, logAuthTokenRotated, logger.FieldUserIDSnake, userID)
	return &grpcservices.AuthTokens{
		AccessToken:      newAccessToken,
		RefreshToken:     newRefreshToken,
		AccessExpiresIn:  s.tokenIssuer.GetAccessTTL(),
		RefreshExpiresIn: s.tokenIssuer.GetRefreshTTL(),
	}, nil
}

// rotationRefused maps a failed rotation to the error the caller sees. A token
// another request claimed first is merely invalid, not reuse.
func (s *Service) rotationRefused(ctx context.Context, userID uuid.UUID, err error) error {
	switch {
	case errors.Is(err, storage.ErrRefreshTokenUserInactive):
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonUserInactive, logger.FieldUserIDSnake, userID)
		return ErrUserNotActive
	case errors.Is(err, storage.ErrRefreshTokenNotClaimable):
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonTokenNotClaimable, logger.FieldUserIDSnake, userID)
		return ErrInvalidRefreshToken
	default:
		s.logger.ErrorContext(ctx, logAuthTokenRefreshFailed, logger.FieldUserIDSnake, userID, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errPrefixRotateRefreshToken, err)
	}
}
