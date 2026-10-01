package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

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
	logMarkTokenReplacedFailed   = "auth.mark_replaced.failed"
	logAuthPasswordRevokeFailed  = "auth.password.change.revoke_failed"

	reasonUserNotFound       = "user_not_found"
	reasonUserInactive       = "user_inactive"
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

	// Look up user by email
	user, err := s.userStore.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal whether email exists
		s.logger.WarnContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldReason, reasonUserNotFound)
		return nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !user.IsActive {
		s.logger.WarnContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldReason, reasonUserInactive)
		return nil, ErrUserNotActive
	}

	// Check if user has a password set
	if user.PasswordHash == nil || *user.PasswordHash == "" {
		s.logger.WarnContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldReason, reasonNoPasswordSet)
		return nil, ErrNoPasswordSet
	}

	// Verify password
	if err := VerifyPassword(password, *user.PasswordHash); err != nil {
		s.logger.WarnContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldReason, reasonInvalidPassword)
		return nil, ErrInvalidCredentials
	}

	// Get user memberships
	memberships, err := s.membershipStore.ListUserMemberships(ctx, user.ID)
	if err != nil {
		s.logger.ErrorContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldError, err)
		return nil, err
	}

	// User must have at least one active membership (CE synthesizes one if needed)
	if len(memberships) == 0 && s.ceProvider != nil {
		ceMembership, ceErr := s.ceProvider.SynthesizeMembership(ctx, user)
		if ceErr != nil {
			s.logger.ErrorContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldError, ceErr)
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
	} else if len(memberships) == 0 {
		s.logger.WarnContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldReason, reasonNoMemberships)
		return nil, ErrMembershipRequired
	}

	// Get default org (first membership by created_at)
	defaultOrgID := memberships[0].OrgID

	// Issue tokens
	accessToken, err := s.tokenIssuer.IssueAccessToken(user.ID, &defaultOrgID)
	if err != nil {
		s.logger.ErrorContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldError, err)
		return nil, err
	}

	var refreshToken string
	if s.refreshTokenEnabled {
		refreshToken, err = s.tokenIssuer.IssueRefreshToken(user.ID)
		if err != nil {
			s.logger.ErrorContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldError, err)
			return nil, err
		}

		// Store refresh token hash
		tokenHash := HashRefreshToken(refreshToken)
		rt := &models.RefreshToken{
			ID:        uuid.New(),
			UserID:    user.ID,
			TokenHash: tokenHash,
			IssuedAt:  time.Now().UTC(),
			ExpiresAt: s.tokenIssuer.GetRefreshExpiresAt(),
		}
		if err := s.refreshTokenStore.Create(ctx, rt); err != nil {
			s.logger.ErrorContext(ctx, logAuthLoginFailed, logger.FieldEmail, email, logger.FieldError, err)
			return nil, err
		}
	}

	s.logger.InfoContext(ctx, logAuthLoginSuccess, logger.FieldEmail, email, logger.FieldUserIDSnake, user.ID)

	// Build user profile from already-loaded user and memberships
	hasPassword := user.PasswordHash != nil && *user.PasswordHash != ""
	profile := &grpcservices.UserProfile{
		ID:           user.ID,
		Email:        user.Email,
		IsAdmin:      user.IsAdmin,
		HasPassword:  hasPassword,
		DefaultOrgID: &defaultOrgID,
		Memberships:  make([]grpcservices.OrganizationMembership, len(memberships)),
	}

	if user.FirstName != nil {
		profile.FirstName = *user.FirstName
	}
	if user.LastName != nil {
		profile.LastName = *user.LastName
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

	return &grpcservices.AuthLoginResult{
		Tokens: &grpcservices.AuthTokens{
			AccessToken:      accessToken,
			RefreshToken:     refreshToken,
			AccessExpiresIn:  s.tokenIssuer.GetAccessTTL(),
			RefreshExpiresIn: s.tokenIssuer.GetRefreshTTL(),
		},
		Profile: profile,
	}, nil
}

// RefreshTokens exchanges refresh token for new token pair.
func (s *Service) RefreshTokens(ctx context.Context, refreshToken string) (*grpcservices.AuthTokens, error) {
	s.logger.InfoContext(ctx, logAuthTokenRefresh)

	if !s.localLoginEnabled {
		return nil, ErrLocalLoginDisabled
	}
	if !s.refreshTokenEnabled {
		return nil, ErrRefreshTokenDisabled
	}

	// Defensive nil check: tokenIssuer may be nil when token generation is disabled
	if s.tokenIssuer == nil {
		return nil, ErrTokenGenerationDisabled
	}

	// Parse and validate the refresh token
	userID, _, err := s.tokenIssuer.ParseRefreshToken(refreshToken)
	if err != nil {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonInvalidToken)
		return nil, err
	}

	// Look up stored token by hash
	tokenHash := HashRefreshToken(refreshToken)
	storedToken, err := s.refreshTokenStore.GetByHash(ctx, tokenHash)
	if err != nil {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonTokenLookupFailed, logger.FieldError, err)
		return nil, ErrInvalidRefreshToken
	}
	if storedToken == nil {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonTokenNotFound)
		return nil, ErrInvalidRefreshToken
	}

	// Check for reuse (token already replaced)
	if storedToken.IsReplaced() {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonTokenReuseDetected, logger.FieldUserIDSnake, userID)
		// Revoke entire token family
		if err := s.refreshTokenStore.RevokeByUserID(ctx, userID); err != nil {
			s.logger.ErrorContext(ctx, logAuthRevokeTokenFailed, logger.FieldError, err)
		}
		return nil, ErrRefreshTokenReused
	}

	// Check if revoked
	if storedToken.IsRevoked() {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonTokenRevoked)
		return nil, ErrRefreshTokenRevoked
	}

	// Check expiration
	if storedToken.IsExpired() {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonTokenExpired)
		return nil, ErrInvalidRefreshToken
	}

	// Get user to find default org
	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonUserNotFound, logger.FieldUserIDSnake, userID)
		return nil, ErrInvalidRefreshToken
	}

	// Get memberships for default org (CE synthesizes one if needed)
	memberships, err := s.membershipStore.ListUserMemberships(ctx, userID)
	if err != nil {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonNoMemberships, logger.FieldUserIDSnake, userID)
		return nil, ErrMembershipRequired
	}
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
	} else if len(memberships) == 0 {
		s.logger.WarnContext(ctx, logAuthTokenRefreshFailed, logger.FieldReason, reasonNoMemberships, logger.FieldUserIDSnake, userID)
		return nil, ErrMembershipRequired
	}
	defaultOrgID := memberships[0].OrgID

	// Issue new tokens
	newAccessToken, err := s.tokenIssuer.IssueAccessToken(user.ID, &defaultOrgID)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := s.tokenIssuer.IssueRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	// Store new refresh token
	newTokenHash := HashRefreshToken(newRefreshToken)
	newRT := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: newTokenHash,
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: s.tokenIssuer.GetRefreshExpiresAt(),
	}
	if err := s.refreshTokenStore.Create(ctx, newRT); err != nil {
		return nil, err
	}

	// Mark old token as replaced
	if err := s.refreshTokenStore.MarkReplaced(ctx, storedToken.ID, newRT.ID); err != nil {
		s.logger.WarnContext(ctx, logMarkTokenReplacedFailed, logger.FieldError, err)
		// Non-fatal, continue
	}

	s.logger.InfoContext(ctx, logAuthTokenRotated, logger.FieldUserIDSnake, userID)

	return &grpcservices.AuthTokens{
		AccessToken:      newAccessToken,
		RefreshToken:     newRefreshToken,
		AccessExpiresIn:  s.tokenIssuer.GetAccessTTL(),
		RefreshExpiresIn: s.tokenIssuer.GetRefreshTTL(),
	}, nil
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

	// Revoke all refresh tokens (force re-login)
	if s.refreshTokenEnabled {
		if err := s.refreshTokenStore.RevokeByUserID(ctx, userID); err != nil {
			// Log but don't fail - password was changed successfully
			s.logger.WarnContext(ctx, logAuthPasswordRevokeFailed, logger.FieldUserIDSnake, userID, logger.FieldError, err)
		}
	}

	s.logger.InfoContext(ctx, logAuthPasswordChangeSuccess, logger.FieldUserIDSnake, userID)
	return nil
}

// Ensure Service implements grpcservices.AuthService
var _ grpcservices.AuthService = (*Service)(nil)
