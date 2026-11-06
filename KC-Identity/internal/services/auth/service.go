package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/passwordpolicy"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"

	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
)

const defaultLoginLabel = "Sign In"

// NormalizeEmail lowercases and trims an email for case-insensitive uniqueness.
// defaultLoginLabel is the built-in provider's login button label.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Log field and message constants
//
//nolint:gosec // Log tokens/fields are identifiers, not credentials.
const (
	logAuthLoginAttempt          = "auth.login.attempt"
	logAuthLoginFailed           = "auth.login.failed"
	logAuthLoginSuccess          = "auth.login.success"
	logAuthTokenRefresh          = "auth.token.refresh"
	logAuthTokenRefreshFailed    = "auth.token.refresh.failed"
	logAuthTokenRotated          = "auth.token.rotated"
	logAuthLogout                = "auth.logout"
	logAuthLogoutFailed          = "auth.logout.failed"
	logAuthTokenRevoked          = "auth.token.revoked"
	logAuthRevokeTokenFailed     = "auth.revoke.token_family.failed"
	logAuthPasswordChange        = "auth.password.change"
	logAuthPasswordChangeFailed  = "auth.password.change.failed"
	logAuthPasswordChangeSuccess = "auth.password.change.success"
	logAuthProfileFetched        = "auth.profile.fetched"
	logAuthPasswordRevokeFailed  = "auth.password.change.revoke_failed"

	reasonUserNotFound       = "user_not_found"
	reasonUserInactive       = "user_inactive"
	reasonCredentialsChanged = "credentials_changed"
	reasonNoPasswordSet      = "no_password_set"
	reasonInvalidPassword    = "invalid_password"
	reasonNoMemberships      = "no_memberships"
	reasonLocalLoginDisabled = "local_login_disabled"
	reasonInvalidToken       = "invalid_token"
	reasonTokenLookupFailed  = "token_lookup_failed"
	reasonTokenNotFound      = "token_not_found"
	reasonTokenReuseDetected = "token_reuse_detected"
	reasonTokenRevoked       = "token_revoked"
	reasonTokenExpired       = "token_expired"
	reasonTokenNotClaimable  = "token_not_claimable"
	reasonTokenGenDisabled   = "token_generation_disabled"
	reasonPasswordWeak       = "password_weak"
)

// Service implements the grpcservices.AuthService interface.
type Service struct {
	userStore           UserStore
	membershipStore     OrganizationMembershipStore
	refreshTokenStore   RefreshTokenStore
	tokenIssuer         TokenIssuer
	localLoginEnabled   bool
	refreshTokenEnabled bool
	logger              logger.Logger
	ceProvider          MembershipSynthesizer // nil in ECE
}

// NewService creates a new authentication service.
func NewService(
	userStore UserStore,
	membershipStore OrganizationMembershipStore,
	refreshTokenStore RefreshTokenStore,
	tokenIssuer TokenIssuer,
	localLoginEnabled bool,
	refreshTokenEnabled bool,
	log logger.Logger,
) *Service {
	return &Service{
		userStore:           userStore,
		membershipStore:     membershipStore,
		refreshTokenStore:   refreshTokenStore,
		tokenIssuer:         tokenIssuer,
		localLoginEnabled:   localLoginEnabled,
		refreshTokenEnabled: refreshTokenEnabled,
		logger:              log,
	}
}

// WithCEProvider sets the CE default org provider for community edition.
func (s *Service) WithCEProvider(provider MembershipSynthesizer) *Service {
	s.ceProvider = provider
	return s
}

// GetProfile retrieves user profile with memberships.
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*grpcservices.UserProfile, error) {
	s.logger.InfoContext(ctx, logAuthProfileFetched, logger.FieldUserIDSnake, userID)

	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	memberships, err := s.membershipStore.ListUserMemberships(ctx, userID)
	if err != nil {
		return nil, err
	}

	// CE: synthesize default org membership when user has no memberships
	if len(memberships) == 0 && s.ceProvider != nil {
		ceMembership, ceErr := s.ceProvider.SynthesizeMembership(ctx, user)
		if ceErr != nil {
			return nil, ceErr
		}
		memberships = append(memberships, &models.OrganizationMembershipWithOrg{
			OrgID:              ceMembership.OrgID,
			OrgName:            ceMembership.OrgName,
			Role:               ceMembership.Role,
			IsOrgAdmin:         ceMembership.IsOrgAdmin,
			IsBaseStationAdmin: ceMembership.IsBaseStationAdmin,
			IsEndpointAdmin:    ceMembership.IsEndpointAdmin,
		})
	}

	// Check if user has a password set
	hasPassword := user.PasswordHash != nil && *user.PasswordHash != ""

	profile := &grpcservices.UserProfile{
		ID:          user.ID,
		Email:       user.Email,
		IsAdmin:     user.IsAdmin,
		HasPassword: hasPassword,
		Memberships: make([]grpcservices.OrganizationMembership, len(memberships)),
	}

	if user.FirstName != nil {
		profile.FirstName = *user.FirstName
	}
	if user.LastName != nil {
		profile.LastName = *user.LastName
	}

	// Set default org ID if user has memberships
	if len(memberships) > 0 {
		profile.DefaultOrgID = &memberships[0].OrgID
	}

	for i, m := range memberships {
		profile.Memberships[i] = grpcservices.OrganizationMembership{
			OrgID:              m.OrgID,
			OrgName:            m.OrgName,
			Role:               m.Role,
			DisplayName:        m.OrgName,
			IsOrgAdmin:         m.IsOrgAdmin,
			IsBaseStationAdmin: m.IsBaseStationAdmin,
			IsEndpointAdmin:    m.IsEndpointAdmin,
		}
	}

	return profile, nil
}

// GetAuthSettings returns auth provider configuration.
func (s *Service) GetAuthSettings(_ context.Context) (*grpcservices.AuthSettings, error) {
	return &grpcservices.AuthSettings{
		Enabled:             authSettingsEnabled,
		LocalLoginEnabled:   s.localLoginEnabled,
		LoginURL:            "", // No custom login URL
		LoginLabel:          defaultLoginLabel,
		LoginRedirect:       false,
		LogoutURL:           "",
		RefreshTokenEnabled: s.refreshTokenEnabled,
		OIDCEnabled:         authSettingsOIDCEnabled,
		OIDCProviderURL:     "",
		PasswordPolicy:      passwordpolicy.Rules,
	}, nil
}

// Logout revokes a user's session (all refresh tokens).
func (s *Service) Logout(ctx context.Context, userID uuid.UUID) error {
	s.logger.InfoContext(ctx, logAuthLogout, logger.FieldUserIDSnake, userID)

	if !s.localLoginEnabled {
		return ErrLocalLoginDisabled
	}

	// Revoke all refresh tokens for user
	if err := s.refreshTokenStore.RevokeByUserID(ctx, userID); err != nil {
		s.logger.WarnContext(ctx, logAuthLogoutFailed, logger.FieldUserIDSnake, userID, logger.FieldError, err)
		return err
	}

	s.logger.InfoContext(ctx, logAuthTokenRevoked, logger.FieldUserIDSnake, userID)
	return nil
}

// ChangePassword changes the authenticated user's password.
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	s.logger.InfoContext(ctx, logAuthPasswordChange, logger.FieldUserIDSnake, userID)

	if !s.localLoginEnabled {
		return ErrLocalLoginDisabled
	}

	// Get user to verify current password
	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	// Verify current password if set
	if user.PasswordHash != nil && *user.PasswordHash != "" {
		if err := VerifyPassword(currentPassword, *user.PasswordHash); err != nil {
			s.logger.WarnContext(ctx, logAuthPasswordChangeFailed, logger.FieldUserIDSnake, userID, logger.FieldReason, reasonInvalidPassword)
			return ErrInvalidCredentials
		}
	}

	// Validate new password
	if err := ValidatePassword(newPassword); err != nil {
		s.logger.WarnContext(ctx, logAuthPasswordChangeFailed, logger.FieldUserIDSnake, userID, logger.FieldReason, reasonPasswordWeak)
		return err
	}

	// Generate salt
	salt := make([]byte, config.AuthPBKDF2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		s.logger.ErrorContext(ctx, logAuthPasswordChangeFailed, logger.FieldUserIDSnake, userID, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errPrefixGenerateSalt, err)
	}

	// Hash password
	passwordHash := HashPassword(newPassword, salt, config.AuthPBKDF2Iterations)

	// Update via store
	if err := s.userStore.SetPasswordHash(ctx, userID, passwordHash); err != nil {
		s.logger.ErrorContext(ctx, logAuthPasswordChangeFailed, logger.FieldUserIDSnake, userID, logger.FieldError, err)
		return err
	}

	if err := s.endSessionsAfterPasswordChange(ctx, userID); err != nil {
		return err
	}

	s.logger.InfoContext(ctx, logAuthPasswordChangeSuccess, logger.FieldUserIDSnake, userID)
	return nil
}

// endSessionsAfterPasswordChange revokes every refresh token of the user, so
// a session opened with the old password cannot be renewed; a failure is
// reported because the caller would otherwise believe the old sessions ended.
func (s *Service) endSessionsAfterPasswordChange(ctx context.Context, userID uuid.UUID) error {
	if !s.refreshTokenEnabled {
		return nil
	}
	if err := s.refreshTokenStore.RevokeByUserID(ctx, userID); err != nil {
		s.logger.ErrorContext(ctx, logAuthPasswordRevokeFailed, logger.FieldUserIDSnake, userID, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errPrefixRevokeSessions, err)
	}
	return nil
}

// Ensure Service implements grpcservices.AuthService
var _ grpcservices.AuthService = (*Service)(nil)
