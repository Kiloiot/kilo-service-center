// Package registration provides self-service account registration.
package registration

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	"github.com/google/uuid"
)

// emailRegex validates basic email format per config.AuthEmailRegexPattern.
var emailRegex = regexp.MustCompile(config.AuthEmailRegexPattern)

// normalizeEmail lowercases and trims an email address for case-insensitive uniqueness.
func normalizeEmail(email string) string {
	return auth.NormalizeEmail(email)
}

// Service handles self-service account registration.
type Service struct {
	registrationRepo    interfaces.RegistrationRepository
	userStore           auth.UserStore
	tokenIssuer         *auth.TokenIssuer
	refreshTokenStore   auth.RefreshTokenStore
	membershipStore     auth.OrganizationMembershipStore
	registrationEnabled bool
	localLoginEnabled   bool
	refreshTokenEnabled bool
	isCommunityEdition  bool
	defaultTenantID     int64
	defaultOrgRepo      interfaces.OrganizationRepository // for CE default org lookup
	log                 logger.Logger
}

// NewService creates a new registration service.
func NewService(
	registrationRepo interfaces.RegistrationRepository,
	userStore auth.UserStore,
	tokenIssuer *auth.TokenIssuer,
	refreshTokenStore auth.RefreshTokenStore,
	membershipStore auth.OrganizationMembershipStore,
	registrationEnabled bool,
	localLoginEnabled bool,
	refreshTokenEnabled bool,
	log logger.Logger,
) *Service {
	return &Service{
		registrationRepo:    registrationRepo,
		userStore:           userStore,
		tokenIssuer:         tokenIssuer,
		refreshTokenStore:   refreshTokenStore,
		membershipStore:     membershipStore,
		registrationEnabled: registrationEnabled,
		localLoginEnabled:   localLoginEnabled,
		refreshTokenEnabled: refreshTokenEnabled,
		log:                 log.WithField("component", "registration-service"),
	}
}

// WithCEMode configures the service for Community Edition registration.
func (s *Service) WithCEMode(defaultTenantID int64, orgRepo interfaces.OrganizationRepository) *Service {
	s.isCommunityEdition = true
	s.defaultTenantID = defaultTenantID
	s.defaultOrgRepo = orgRepo
	return s
}

// RegisterAccount creates a new user account with an organization.
func (s *Service) RegisterAccount(ctx context.Context, req *grpcservices.RegisterAccountRequest) (*grpcservices.AuthLoginResult, error) {
	if !s.registrationEnabled {
		return nil, fmt.Errorf("%w", errRegistrationDisabled)
	}

	if req.Email == "" {
		return nil, fmt.Errorf("email: %w", errEmailRequired)
	}
	if req.FirstName == "" {
		return nil, fmt.Errorf("first_name: %w", errFirstNameRequired)
	}
	if req.LastName == "" {
		return nil, fmt.Errorf("last_name: %w", errLastNameRequired)
	}
	if req.CompanyName == "" && !s.isCommunityEdition {
		return nil, fmt.Errorf("company_name: %w", errCompanyNameRequired)
	}

	// Normalize email for case-insensitive uniqueness
	req.Email = normalizeEmail(req.Email)

	// Validate email format and length
	if len(req.Email) > config.AuthEmailMaxLength {
		return nil, fmt.Errorf("email: %w", errFieldTooLong)
	}
	if !emailRegex.MatchString(req.Email) {
		return nil, fmt.Errorf("email: %w", errEmailInvalidFormat)
	}

	// Validate field lengths
	if utf8.RuneCountInString(req.FirstName) > config.AuthNameMaxLength {
		return nil, fmt.Errorf("first_name: %w", errFieldTooLong)
	}
	if utf8.RuneCountInString(req.LastName) > config.AuthNameMaxLength {
		return nil, fmt.Errorf("last_name: %w", errFieldTooLong)
	}
	if req.CompanyName != "" && utf8.RuneCountInString(req.CompanyName) > config.AuthCompanyMaxLength {
		return nil, fmt.Errorf("company_name: %w", errFieldTooLong)
	}

	if err := auth.ValidatePassword(req.Password); err != nil {
		return nil, fmt.Errorf("password: %w", errWeakPassword)
	}

	// Check email uniqueness
	_, err := s.userStore.GetByEmail(ctx, req.Email)
	if err == nil {
		return nil, fmt.Errorf("email: %w", errEmailExists)
	}
	if !errors.Is(err, interfaces.ErrRecordNotFound) {
		s.log.ErrorContext(ctx, "failed to check email uniqueness", "error", err)
		return nil, fmt.Errorf("check email: %w", errRegistrationFailed)
	}

	// Generate salt and hash password
	salt := make([]byte, config.AuthPBKDF2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", errRegistrationFailed)
	}
	hash := auth.HashPassword(req.Password, salt, config.AuthPBKDF2Iterations)

	// Build user model
	user := &models.User{
		ID:            uuid.New(),
		Email:         req.Email,
		EmailVerified: false,
		PasswordHash:  &hash,
		IsAdmin:       false,
		IsActive:      true,
		FirstName:     &req.FirstName,
		LastName:      &req.LastName,
	}
	if req.CompanyName != "" {
		user.CompanyName = &req.CompanyName
	}

	// CE: register user on existing default org/tenant (no new org/tenant created)
	// ECE: atomic registration (user + tenant + org + membership)
	var result *interfaces.RegistrationResult
	if s.isCommunityEdition {
		defaultOrg, lookupErr := s.defaultOrgRepo.GetOrgByTenantID(ctx, s.defaultTenantID)
		if lookupErr != nil || defaultOrg == nil {
			s.log.ErrorContext(ctx, "CE default org lookup failed", "error", lookupErr)
			return nil, fmt.Errorf("register account: %w", errRegistrationFailed)
		}

		ceResult, ceErr := s.registrationRepo.RegisterCEAccount(ctx, &interfaces.CERegistrationParams{
			User:     user,
			TenantID: s.defaultTenantID,
			OrgID:    defaultOrg.OrgID,
		})
		if ceErr != nil {
			s.log.ErrorContext(ctx, "CE registration failed", "error", ceErr)
			return nil, fmt.Errorf("register account: %w", errRegistrationFailed)
		}
		result = ceResult
	} else {
		eceResult, eceErr := s.registrationRepo.RegisterAccount(ctx, &interfaces.RegistrationParams{
			User:        user,
			CompanyName: req.CompanyName,
		})
		if eceErr != nil {
			s.log.ErrorContext(ctx, "registration transaction failed", "error", eceErr)
			return nil, fmt.Errorf("register account: %w", errRegistrationFailed)
		}
		result = eceResult
	}

	// Issue access token
	accessToken, err := s.tokenIssuer.IssueAccessToken(result.User.ID, &result.Organization.OrgID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to issue access token after registration", "user_id", result.User.ID, "error", err)
		return nil, fmt.Errorf("issue token: %w", errRegistrationFailed)
	}

	tokens := &grpcservices.AuthTokens{
		AccessToken:     accessToken,
		AccessExpiresIn: s.tokenIssuer.GetAccessTTL(),
	}

	// Issue refresh token if enabled
	if s.refreshTokenEnabled {
		refreshToken, err := s.tokenIssuer.IssueRefreshToken(result.User.ID)
		if err != nil {
			s.log.ErrorContext(ctx, "failed to issue refresh token after registration", "user_id", result.User.ID, "error", err)
		} else {
			// Store refresh token hash
			tokenHash := auth.HashRefreshToken(refreshToken)
			rt := &models.RefreshToken{
				ID:        uuid.New(),
				UserID:    result.User.ID,
				TokenHash: tokenHash,
				IssuedAt:  time.Now().UTC(),
				ExpiresAt: s.tokenIssuer.GetRefreshExpiresAt(),
			}
			if storeErr := s.refreshTokenStore.Create(ctx, rt); storeErr != nil {
				s.log.ErrorContext(ctx, "failed to store refresh token after registration", "user_id", result.User.ID, "error", storeErr)
			} else {
				tokens.RefreshToken = refreshToken
				tokens.RefreshExpiresIn = s.tokenIssuer.GetRefreshTTL()
			}
		}
	}

	// Load memberships for profile
	memberships, err := s.membershipStore.ListUserMemberships(ctx, result.User.ID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to load memberships after registration", "user_id", result.User.ID, "error", err)
		memberships = nil
	}

	// Build profile
	profile := &grpcservices.UserProfile{
		ID:          result.User.ID,
		Email:       result.User.Email,
		IsAdmin:     result.User.IsAdmin,
		HasPassword: result.User.PasswordHash != nil,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
	}

	if result.Organization != nil {
		profile.DefaultOrgID = &result.Organization.OrgID
	}

	for _, m := range memberships {
		profile.Memberships = append(profile.Memberships, grpcservices.OrganizationMembership{
			OrgID:              m.OrgID,
			OrgName:            m.OrgName,
			Role:               m.Role,
			IsOrgAdmin:         m.IsOrgAdmin,
			IsBaseStationAdmin: m.IsBaseStationAdmin,
			IsEndpointAdmin:    m.IsEndpointAdmin,
			DisplayName:        m.OrgName,
		})
	}

	s.log.InfoContext(ctx, "account registered successfully",
		"user_id", result.User.ID,
		"org_id", result.Organization.OrgID)

	return &grpcservices.AuthLoginResult{
		Tokens:  tokens,
		Profile: profile,
	}, nil
}

// Sentinel errors for registration operations.
var (
	errRegistrationDisabled = errors.New(grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistrationDisabled))
	errEmailRequired        = errors.New(grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEmailRequired))
	errFirstNameRequired    = errors.New(grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenFirstNameRequired))
	errLastNameRequired     = errors.New(grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLastNameRequired))
	errCompanyNameRequired  = errors.New(grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCompanyNameRequired))
	errWeakPassword         = errors.New(grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenWeakPassword))
	errEmailExists          = errors.New(grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEmailAlreadyExists))
	errRegistrationFailed   = errors.New(grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistrationFailed))
	errEmailInvalidFormat   = errors.New(grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEmailInvalidFormat))
	errFieldTooLong         = errors.New(grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenFieldTooLong))
)
