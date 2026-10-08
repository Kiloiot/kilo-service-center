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
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	"github.com/google/uuid"
)

// Request field names and registration stages referenced in wrapped
// validation and failure errors, so callers can match on a stable prefix.
const (
	fieldEmail       = "email"
	fieldFirstName   = "first_name"
	fieldLastName    = "last_name"
	fieldCompanyName = "company_name"
	fieldPassword    = "password"

	stageCheckEmail      = "check email"
	stageGenerateSalt    = "generate salt"
	stageRegisterAccount = "register account"
	stageIssueToken      = "issue token"
)

// emailRegex validates basic email format per config.AuthEmailRegexPattern.
var emailRegex = regexp.MustCompile(config.AuthEmailRegexPattern)

// normalizeEmail lowercases and trims an email address for case-insensitive uniqueness.
// componentRegistrationService labels this component in structured logs.
const componentRegistrationService = "registration-service"

func normalizeEmail(email string) string {
	return auth.NormalizeEmail(email)
}

// SessionTokenIssuer is the token capability registration consumes to open
// the first session after signup: issuing both token kinds and reporting
// their lifetimes.
type SessionTokenIssuer interface {
	IssueAccessToken(userID uuid.UUID, orgID *uuid.UUID) (string, error)
	IssueRefreshToken(userID uuid.UUID) (string, error)
	GetAccessTTL() int64
	GetRefreshTTL() int64
	GetRefreshExpiresAt() time.Time
}

// AccountRegistrar atomically provisions an account (user, tenant, org, membership).
type AccountRegistrar interface {
	RegisterAccount(ctx context.Context, params *models.RegistrationParams) (*models.RegistrationResult, error)
	RegisterCEAccount(ctx context.Context, params *models.CERegistrationParams) (*models.RegistrationResult, error)
}

// Service handles self-service account registration.
type Service struct {
	registrationRepo    AccountRegistrar
	userStore           auth.UserStore
	tokenIssuer         SessionTokenIssuer
	refreshTokenStore   auth.RefreshTokenStore
	membershipStore     auth.OrganizationMembershipStore
	registrationEnabled bool
	localLoginEnabled   bool
	refreshTokenEnabled bool
	isCommunityEdition  bool
	defaultTenantID     int64
	defaultOrgRepo      auth.OrgDirectory // for CE default org lookup
	log                 logger.Logger
}

// NewService creates a new registration service.
func NewService(
	registrationRepo AccountRegistrar,
	userStore auth.UserStore,
	tokenIssuer SessionTokenIssuer,
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
		log:                 log.WithField(logger.FieldComponent, componentRegistrationService),
	}
}

// WithCEMode configures the service for Community Edition registration.
func (s *Service) WithCEMode(defaultTenantID int64, orgRepo auth.OrgDirectory) *Service {
	s.isCommunityEdition = true
	s.defaultTenantID = defaultTenantID
	s.defaultOrgRepo = orgRepo
	return s
}

// RegisterAccount creates a new user account with an organization.
func (s *Service) RegisterAccount(ctx context.Context, req *grpcservices.RegisterAccountRequest) (*grpcservices.AuthLoginResult, error) {
	if !s.registrationEnabled {
		return nil, fmt.Errorf("%w", ErrRegistrationDisabled)
	}

	if req.Email == "" {
		return nil, fmt.Errorf("%s: %w", fieldEmail, ErrEmailRequired)
	}
	if req.FirstName == "" {
		return nil, fmt.Errorf("%s: %w", fieldFirstName, ErrFirstNameRequired)
	}
	if req.LastName == "" {
		return nil, fmt.Errorf("%s: %w", fieldLastName, ErrLastNameRequired)
	}
	if req.CompanyName == "" && !s.isCommunityEdition {
		return nil, fmt.Errorf("%s: %w", fieldCompanyName, ErrCompanyNameRequired)
	}

	// Normalize email for case-insensitive uniqueness
	req.Email = normalizeEmail(req.Email)

	// Validate email format and length
	if len(req.Email) > config.AuthEmailMaxLength {
		return nil, fmt.Errorf("%s: %w", fieldEmail, ErrFieldTooLong)
	}
	if !emailRegex.MatchString(req.Email) {
		return nil, fmt.Errorf("%s: %w", fieldEmail, ErrEmailInvalidFormat)
	}

	// Validate field lengths
	if utf8.RuneCountInString(req.FirstName) > config.AuthNameMaxLength {
		return nil, fmt.Errorf("%s: %w", fieldFirstName, ErrFieldTooLong)
	}
	if utf8.RuneCountInString(req.LastName) > config.AuthNameMaxLength {
		return nil, fmt.Errorf("%s: %w", fieldLastName, ErrFieldTooLong)
	}
	if req.CompanyName != "" && utf8.RuneCountInString(req.CompanyName) > config.AuthCompanyMaxLength {
		return nil, fmt.Errorf("%s: %w", fieldCompanyName, ErrFieldTooLong)
	}

	if err := auth.ValidatePassword(req.Password); err != nil {
		return nil, fmt.Errorf("%s: %w", fieldPassword, ErrWeakPassword)
	}

	// Check email uniqueness
	_, err := s.userStore.GetByEmail(ctx, req.Email)
	if err == nil {
		return nil, fmt.Errorf("%s: %w", fieldEmail, ErrEmailExists)
	}
	if !errors.Is(err, storage.ErrRecordNotFound) {
		s.log.ErrorContext(ctx, LogRegistrationEmailUniquenessCheckFailed, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", stageCheckEmail, ErrRegistrationFailed)
	}

	// Generate salt and hash password
	salt := make([]byte, config.AuthPBKDF2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("%s: %w", stageGenerateSalt, ErrRegistrationFailed)
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
	var result *models.RegistrationResult
	if s.isCommunityEdition {
		defaultOrg, lookupErr := s.defaultOrgRepo.GetOrgByTenantID(ctx, s.defaultTenantID)
		if lookupErr != nil || defaultOrg == nil {
			s.log.ErrorContext(ctx, LogRegistrationCEDefaultOrgLookupFailed, logger.FieldError, lookupErr)
			return nil, fmt.Errorf("%s: %w", stageRegisterAccount, ErrRegistrationFailed)
		}

		ceResult, ceErr := s.registrationRepo.RegisterCEAccount(ctx, &models.CERegistrationParams{
			User:     user,
			TenantID: s.defaultTenantID,
			OrgID:    defaultOrg.OrgID,
		})
		if ceErr != nil {
			s.log.ErrorContext(ctx, LogRegistrationCERegistrationFailed, logger.FieldError, ceErr)
			return nil, fmt.Errorf("%s: %w", stageRegisterAccount, ErrRegistrationFailed)
		}
		result = ceResult
	} else {
		eceResult, eceErr := s.registrationRepo.RegisterAccount(ctx, &models.RegistrationParams{
			User:        user,
			CompanyName: req.CompanyName,
		})
		if eceErr != nil {
			s.log.ErrorContext(ctx, LogRegistrationTransactionFailed, logger.FieldError, eceErr)
			return nil, fmt.Errorf("%s: %w", stageRegisterAccount, ErrRegistrationFailed)
		}
		result = eceResult
	}

	// Issue access token
	accessToken, err := s.tokenIssuer.IssueAccessToken(result.User.ID, &result.Organization.OrgID)
	if err != nil {
		s.log.ErrorContext(ctx, LogRegistrationAccessTokenIssueFailed, logger.FieldUserIDSnake, result.User.ID, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", stageIssueToken, ErrRegistrationFailed)
	}

	tokens := &grpcservices.AuthTokens{
		AccessToken:     accessToken,
		AccessExpiresIn: s.tokenIssuer.GetAccessTTL(),
	}

	// Issue refresh token if enabled
	if s.refreshTokenEnabled {
		refreshToken, err := s.tokenIssuer.IssueRefreshToken(result.User.ID)
		if err != nil {
			s.log.ErrorContext(ctx, LogRegistrationRefreshTokenIssueFailed, logger.FieldUserIDSnake, result.User.ID, logger.FieldError, err)
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
				s.log.ErrorContext(ctx, LogRegistrationRefreshTokenStoreFailed, logger.FieldUserIDSnake, result.User.ID, logger.FieldError, storeErr)
			} else {
				tokens.RefreshToken = refreshToken
				tokens.RefreshExpiresIn = s.tokenIssuer.GetRefreshTTL()
			}
		}
	}

	// Load memberships for profile
	memberships, err := s.membershipStore.ListUserMemberships(ctx, result.User.ID)
	if err != nil {
		s.log.ErrorContext(ctx, LogRegistrationMembershipLoadFailed, logger.FieldUserIDSnake, result.User.ID, logger.FieldError, err)
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

	s.log.InfoContext(ctx, LogRegistrationAccountRegistered,
		logger.FieldUserIDSnake, result.User.ID,
		logger.FieldOrgIDSnake, result.Organization.OrgID)

	return &grpcservices.AuthLoginResult{
		Tokens:  tokens,
		Profile: profile,
	}, nil
}

// Domain sentinels for registration failures. The gRPC delivery layer maps
// each sentinel onto its catalog token; the service carries no transport
// vocabulary.
var (
	// ErrRegistrationDisabled reports self-service registration being turned off.
	ErrRegistrationDisabled = errors.New("registration disabled")
	// ErrEmailRequired reports a registration without an email address.
	ErrEmailRequired = errors.New("email required")
	// ErrFirstNameRequired reports a registration without a first name.
	ErrFirstNameRequired = errors.New("first name required")
	// ErrLastNameRequired reports a registration without a last name.
	ErrLastNameRequired = errors.New("last name required")
	// ErrCompanyNameRequired reports a registration without a company name.
	ErrCompanyNameRequired = errors.New("company name required")
	// ErrWeakPassword reports a password below the strength policy.
	ErrWeakPassword = errors.New("password too weak")
	// ErrEmailExists reports an email already registered.
	ErrEmailExists = errors.New("email already registered")
	// ErrRegistrationFailed reports a registration that could not be completed.
	ErrRegistrationFailed = errors.New("registration failed")
	// ErrEmailInvalidFormat reports a malformed email address.
	ErrEmailInvalidFormat = errors.New("invalid email format")
	// ErrFieldTooLong reports a field exceeding its length limit.
	ErrFieldTooLong = errors.New("field too long")
)
