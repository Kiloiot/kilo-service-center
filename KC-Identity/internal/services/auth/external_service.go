// Package auth provides external OIDC/OAuth2 authentication services.
package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/kilocenter/KC-Identity/internal/services/grpcservices"
)

// ============================================================================
// External Auth Constants
// ============================================================================

const (
	// ProviderOIDC identifies OIDC provider in state storage.
	ProviderOIDC = "oidc"

	// ProviderOAuth2 identifies OAuth2 provider in state storage.
	ProviderOAuth2 = "oauth2"

	// StateTokenLength is the length of generated state tokens (32 bytes).
	StateTokenLength = 32

	// NonceLength is the length of generated OIDC nonces (32 bytes).
	NonceLength = 32

	// RegistrationCallbackTimeout is the HTTP timeout for registration callbacks.
	RegistrationCallbackTimeout = 10 * time.Second

	// RegistrationCallbackContentType is the Content-Type for callback payloads.
	RegistrationCallbackContentType = "application/json"
)

// ============================================================================
// External Auth Log Messages
// ============================================================================

const (
	logExternalAuthInitiated           = "external.auth.initiated"
	logExternalAuthStateFailed         = "external.auth.state.failed"
	logExternalAuthStateInvalid        = "external.auth.state.invalid"
	logExternalAuthEmailNotVerified    = "external.auth.email.not_verified"
	logExternalAuthRegDisabled         = "external.auth.registration.disabled"
	logExternalAuthUserCreateFailed    = "external.auth.user.create.failed"
	logExternalAuthUserCreated         = "external.auth.user.created"
	logExternalAuthLinkFailed          = "external.auth.link.failed"
	logExternalAuthNoMemberships       = "external.auth.no_memberships"
	logExternalAuthSuccess             = "external.auth.success"
	logRegistrationCallbackFailed      = "external.auth.callback.failed"
	logRegistrationCallbackSuccess     = "external.auth.callback.success"
	logExternalAuthOrgResolutionFailed = "external.auth.org.resolution.failed"
	logExternalAuthOrgResolved         = "external.auth.org.resolved"
	logExternalAuthOrgNotInMemberships = "external.auth.org.not_in_memberships"
)

// httpClientInterface abstracts HTTP client for testing.
type httpClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

// RegistrationCallbackPayload is the JSON payload sent to registration callback URLs.
type RegistrationCallbackPayload struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Provider  string    `json:"provider"`
	CreatedAt time.Time `json:"created_at"`
}

// ExternalAuthServiceConfig contains external auth service configuration.
type ExternalAuthServiceConfig struct {
	OIDCEnabled          bool
	OIDCLoginLabel       string
	OIDCLoginRedirect    bool
	OIDCStateTTL         time.Duration
	OIDCNonceTTL         time.Duration
	OIDCRegEnabled       bool
	OIDCAssumeVerified   bool
	OIDCRegCallbackURL   string
	OIDCExternalOrgClaim string
	OAuth2Enabled        bool
	OAuth2LoginLabel     string
	OAuth2LoginRedirect  bool
	OAuth2StateTTL       time.Duration
	OAuth2RegEnabled     bool
	OAuth2AssumeVerified bool
	OAuth2RegCallbackURL string
}

// ExternalAuthService implements the grpcservices.ExternalAuthService interface.
type ExternalAuthService struct {
	oidcClient           OIDCClient
	oauth2Client         OAuth2Client
	oidcStateStore       StateStore
	oauth2StateStore     StateStore
	userStore            ExternalUserStore
	membershipStore      OrganizationMembershipStore
	tokenIssuer          *TokenIssuer
	orgResolver          OrganizationResolver
	ceProvider           *CEDefaultOrgProvider // nil in ECE
	oidcEnabled          bool
	oauth2Enabled        bool
	oidcStateTTL         time.Duration
	oidcNonceTTL         time.Duration
	oidcRegEnabled       bool
	oidcAssumeVerified   bool
	oidcRegCallbackURL   string
	oidcExternalOrgClaim string
	oauth2StateTTL       time.Duration
	oauth2RegEnabled     bool
	oauth2AssumeVerified bool
	oauth2RegCallbackURL string
	logger               logger.Logger
	httpClient           httpClientInterface
}

// Ensure ExternalAuthService implements grpcservices.ExternalAuthService interface.
var _ grpcservices.ExternalAuthService = (*ExternalAuthService)(nil)

// NewExternalAuthService creates a new external auth service.
func NewExternalAuthService(
	oidcClient OIDCClient,
	oauth2Client OAuth2Client,
	oidcStateStore StateStore,
	oauth2StateStore StateStore,
	userStore ExternalUserStore,
	membershipStore OrganizationMembershipStore,
	tokenIssuer *TokenIssuer,
	orgResolver OrganizationResolver,
	cfg ExternalAuthServiceConfig,
	log logger.Logger,
) *ExternalAuthService {
	return &ExternalAuthService{
		oidcClient:           oidcClient,
		oauth2Client:         oauth2Client,
		oidcStateStore:       oidcStateStore,
		oauth2StateStore:     oauth2StateStore,
		userStore:            userStore,
		membershipStore:      membershipStore,
		tokenIssuer:          tokenIssuer,
		orgResolver:          orgResolver,
		oidcEnabled:          cfg.OIDCEnabled,
		oauth2Enabled:        cfg.OAuth2Enabled,
		oidcStateTTL:         cfg.OIDCStateTTL,
		oidcNonceTTL:         cfg.OIDCNonceTTL,
		oidcRegEnabled:       cfg.OIDCRegEnabled,
		oidcAssumeVerified:   cfg.OIDCAssumeVerified,
		oidcRegCallbackURL:   cfg.OIDCRegCallbackURL,
		oidcExternalOrgClaim: cfg.OIDCExternalOrgClaim,
		oauth2StateTTL:       cfg.OAuth2StateTTL,
		oauth2RegEnabled:     cfg.OAuth2RegEnabled,
		oauth2AssumeVerified: cfg.OAuth2AssumeVerified,
		oauth2RegCallbackURL: cfg.OAuth2RegCallbackURL,
		logger:               log,
		httpClient:           &http.Client{Timeout: RegistrationCallbackTimeout},
	}
}

// WithCEProvider sets the CE default org provider for community edition.
func (s *ExternalAuthService) WithCEProvider(provider *CEDefaultOrgProvider) *ExternalAuthService {
	s.ceProvider = provider
	return s
}

// extractClaimPath extracts a value from nested claims using dot notation.
func extractClaimPath(claims map[string]any, path string) string {
	if path == "" || claims == nil {
		return ""
	}

	parts := strings.Split(path, ".")
	current := claims

	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return ""
		}

		if i == len(parts)-1 {
			switch v := val.(type) {
			case string:
				return v
			case float64:
				return strconv.FormatFloat(v, 'f', -1, 64)
			default:
				return fmt.Sprintf("%v", v)
			}
		}

		nextMap, ok := val.(map[string]any)
		if !ok {
			return ""
		}
		current = nextMap
	}
	return ""
}

// InitiateOIDCLogin generates authorization URL and stores state with nonce.
func (s *ExternalAuthService) InitiateOIDCLogin(ctx context.Context) (string, error) {
	if !s.oidcEnabled {
		return "", ErrOIDCProviderDisabled
	}

	state := generateSecureToken(StateTokenLength)
	nonce := generateSecureToken(NonceLength)

	authState := State{
		Provider:  ProviderOIDC,
		CreatedAt: time.Now().Unix(),
		Nonce:     nonce,
	}

	stateJSON, err := json.Marshal(authState)
	if err != nil {
		return "", err
	}

	if err := s.oidcStateStore.StoreState(ctx, state, stateJSON, s.oidcStateTTL); err != nil {
		s.logger.ErrorContext(ctx, logExternalAuthStateFailed, "error", err)
		return "", ErrRedisUnavailable
	}

	authURL := s.oidcClient.GetAuthorizationURL(state, nonce)

	s.logger.InfoContext(ctx, logExternalAuthInitiated, "provider", ProviderOIDC)
	return authURL, nil
}

// CompleteOIDCLogin exchanges code for tokens and creates local session.
func (s *ExternalAuthService) CompleteOIDCLogin(ctx context.Context, code, state string) (*grpcservices.AuthLoginResult, error) {
	if !s.oidcEnabled {
		return nil, ErrOIDCProviderDisabled
	}

	stateJSON, err := s.oidcStateStore.GetState(ctx, state)
	if err != nil {
		s.logger.WarnContext(ctx, logExternalAuthStateInvalid, "error", err)
		return nil, ErrStateNotFound
	}

	var authState State
	if err := json.Unmarshal(stateJSON, &authState); err != nil {
		return nil, ErrInvalidState
	}

	if authState.Provider != ProviderOIDC {
		return nil, ErrInvalidState
	}

	tokenResp, err := s.oidcClient.ExchangeCode(ctx, code)
	if err != nil {
		return nil, ErrTokenExchangeFailed
	}

	claims, rawClaims, err := s.oidcClient.ValidateIDToken(ctx, tokenResp.IDToken, authState.Nonce)
	if err != nil {
		return nil, ErrIDTokenInvalid
	}

	// Extract and resolve external org claim if configured
	var resolvedOrgID uuid.UUID
	if s.oidcExternalOrgClaim != "" && s.orgResolver != nil {
		externalOrgValue := extractClaimPath(rawClaims, s.oidcExternalOrgClaim)
		if externalOrgValue != "" {
			orgID, err := s.orgResolver.ResolveOrgByExternalID(ctx, externalOrgValue)
			if err != nil {
				s.logger.WarnContext(ctx, logExternalAuthOrgResolutionFailed,
					"external_org_value", externalOrgValue, "error", err)
				return nil, ErrOrgResolutionFailed
			}
			if orgID != uuid.Nil {
				resolvedOrgID = orgID
				s.logger.InfoContext(ctx, logExternalAuthOrgResolved,
					"external_org_value", externalOrgValue, "resolved_org_id", orgID.String())
			}
		}
	}

	if !s.oidcAssumeVerified && !claims.EmailVerified {
		s.logger.WarnContext(ctx, logExternalAuthEmailNotVerified, "email", claims.Email)
		return nil, ErrEmailNotVerified
	}

	user, err := s.findOrCreateUserFromOIDC(ctx, claims)
	if err != nil {
		return nil, err
	}

	return s.issueLocalTokensWithOrg(ctx, user, resolvedOrgID)
}

// InitiateOAuth2Login generates authorization URL and stores state with PKCE verifier.
func (s *ExternalAuthService) InitiateOAuth2Login(ctx context.Context) (string, error) {
	if !s.oauth2Enabled {
		return "", ErrOAuth2ProviderDisabled
	}

	state := generateSecureToken(StateTokenLength)
	authURL, codeVerifier := s.oauth2Client.GetAuthorizationURL(state)

	authState := State{
		Provider:     ProviderOAuth2,
		CreatedAt:    time.Now().Unix(),
		CodeVerifier: codeVerifier,
	}

	stateJSON, err := json.Marshal(authState)
	if err != nil {
		return "", err
	}

	if err := s.oauth2StateStore.StoreState(ctx, state, stateJSON, s.oauth2StateTTL); err != nil {
		s.logger.ErrorContext(ctx, logExternalAuthStateFailed, "error", err)
		return "", ErrRedisUnavailable
	}

	s.logger.InfoContext(ctx, logExternalAuthInitiated, "provider", ProviderOAuth2)
	return authURL, nil
}

// CompleteOAuth2Login exchanges code for tokens and creates local session.
func (s *ExternalAuthService) CompleteOAuth2Login(ctx context.Context, code, state string) (*grpcservices.AuthLoginResult, error) {
	if !s.oauth2Enabled {
		return nil, ErrOAuth2ProviderDisabled
	}

	stateJSON, err := s.oauth2StateStore.GetState(ctx, state)
	if err != nil {
		s.logger.WarnContext(ctx, logExternalAuthStateInvalid, "error", err)
		return nil, ErrStateNotFound
	}

	var authState State
	if err := json.Unmarshal(stateJSON, &authState); err != nil {
		return nil, ErrInvalidState
	}

	if authState.Provider != ProviderOAuth2 {
		return nil, ErrInvalidState
	}

	tokenResp, err := s.oauth2Client.ExchangeCode(ctx, code, authState.CodeVerifier)
	if err != nil {
		return nil, ErrTokenExchangeFailed
	}

	userInfo, err := s.oauth2Client.GetUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, ErrUserInfoFailed
	}

	if !s.oauth2AssumeVerified && !userInfo.EmailVerified {
		s.logger.WarnContext(ctx, logExternalAuthEmailNotVerified, "email", userInfo.Email)
		return nil, ErrEmailNotVerified
	}

	user, err := s.findOrCreateUserFromOAuth2(ctx, userInfo)
	if err != nil {
		return nil, err
	}

	return s.issueLocalTokensWithOrg(ctx, user, uuid.Nil)
}

// findOrCreateUserFromOIDC finds existing user by external ID or email, or creates new user.
func (s *ExternalAuthService) findOrCreateUserFromOIDC(ctx context.Context, claims *OIDCClaims) (*models.User, error) {
	claims.Email = NormalizeEmail(claims.Email)

	user, err := s.userStore.GetByExternalID(ctx, claims.Sub)
	if err == nil && user != nil {
		return user, nil
	}

	user, err = s.userStore.GetByEmail(ctx, claims.Email)
	if err == nil && user != nil {
		user.ExternalID = &claims.Sub
		if err := s.userStore.Update(ctx, user); err != nil {
			s.logger.WarnContext(ctx, logExternalAuthLinkFailed, "error", err)
		}
		return user, nil
	}

	if !s.oidcRegEnabled {
		s.logger.WarnContext(ctx, logExternalAuthRegDisabled, "email", claims.Email)
		return nil, ErrRegistrationDisabled
	}

	newUser := &models.User{
		ID:         uuid.New(),
		Email:      claims.Email,
		ExternalID: &claims.Sub,
		IsActive:   true,
		IsAdmin:    false,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	if err := s.userStore.Create(ctx, newUser); err != nil {
		s.logger.ErrorContext(ctx, logExternalAuthUserCreateFailed, "error", err)
		return nil, err
	}

	s.logger.InfoContext(ctx, logExternalAuthUserCreated, "email", claims.Email, "provider", ProviderOIDC)
	s.sendRegistrationCallback(ctx, s.oidcRegCallbackURL, newUser, ProviderOIDC)

	return newUser, nil
}

// findOrCreateUserFromOAuth2 finds existing user by email or creates new user.
func (s *ExternalAuthService) findOrCreateUserFromOAuth2(ctx context.Context, userInfo *OAuth2UserInfo) (*models.User, error) {
	userInfo.Email = NormalizeEmail(userInfo.Email)

	externalID := userInfo.Sub
	if externalID == "" {
		externalID = userInfo.ID
	}

	if externalID != "" {
		user, err := s.userStore.GetByExternalID(ctx, externalID)
		if err == nil && user != nil {
			return user, nil
		}
	}

	user, err := s.userStore.GetByEmail(ctx, userInfo.Email)
	if err == nil && user != nil {
		if externalID != "" && (user.ExternalID == nil || *user.ExternalID == "") {
			user.ExternalID = &externalID
			if err := s.userStore.Update(ctx, user); err != nil {
				s.logger.WarnContext(ctx, logExternalAuthLinkFailed, "error", err)
			}
		}
		return user, nil
	}

	if !s.oauth2RegEnabled {
		s.logger.WarnContext(ctx, logExternalAuthRegDisabled, "email", userInfo.Email)
		return nil, ErrRegistrationDisabled
	}

	newUser := &models.User{
		ID:        uuid.New(),
		Email:     userInfo.Email,
		IsActive:  true,
		IsAdmin:   false,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if externalID != "" {
		newUser.ExternalID = &externalID
	}

	if err := s.userStore.Create(ctx, newUser); err != nil {
		s.logger.ErrorContext(ctx, logExternalAuthUserCreateFailed, "error", err)
		return nil, err
	}

	s.logger.InfoContext(ctx, logExternalAuthUserCreated, "email", userInfo.Email, "provider", ProviderOAuth2)
	s.sendRegistrationCallback(ctx, s.oauth2RegCallbackURL, newUser, ProviderOAuth2)

	return newUser, nil
}

// issueLocalTokensWithOrg issues local tokens for the user with optional resolved org.
func (s *ExternalAuthService) issueLocalTokensWithOrg(ctx context.Context, user *models.User, resolvedOrgID uuid.UUID) (*grpcservices.AuthLoginResult, error) {
	memberships, err := s.membershipStore.ListUserMemberships(ctx, user.ID)
	if err != nil {
		return nil, err
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
		s.logger.WarnContext(ctx, logExternalAuthNoMemberships, "user_id", user.ID)
		return nil, ErrMembershipRequired
	}

	// Determine default org
	var defaultOrgID uuid.UUID
	if resolvedOrgID != uuid.Nil {
		for _, m := range memberships {
			if m.OrgID == resolvedOrgID {
				defaultOrgID = resolvedOrgID
				break
			}
		}
		if defaultOrgID == uuid.Nil {
			s.logger.WarnContext(ctx, logExternalAuthOrgNotInMemberships,
				"user_id", user.ID, "resolved_org_id", resolvedOrgID.String())
			defaultOrgID = memberships[0].OrgID
		}
	} else {
		defaultOrgID = memberships[0].OrgID
	}

	accessToken, err := s.tokenIssuer.IssueAccessToken(user.ID, &defaultOrgID)
	if err != nil {
		return nil, err
	}

	// Build profile
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

	s.logger.InfoContext(ctx, logExternalAuthSuccess, "user_id", user.ID)

	return &grpcservices.AuthLoginResult{
		Tokens: &grpcservices.AuthTokens{
			AccessToken:     accessToken,
			AccessExpiresIn: s.tokenIssuer.GetAccessTTL(),
		},
		Profile: profile,
	}, nil
}

// sendRegistrationCallback sends notification to callback URL when user is created.
func (s *ExternalAuthService) sendRegistrationCallback(ctx context.Context, callbackURL string, user *models.User, provider string) {
	if callbackURL == "" {
		return
	}

	payload := RegistrationCallbackPayload{
		UserID:    user.ID.String(),
		Email:     user.Email,
		Provider:  provider,
		CreatedAt: user.CreatedAt,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		s.logger.WarnContext(ctx, logRegistrationCallbackFailed, "error", err, "stage", "marshal")
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(payloadJSON))
	if err != nil {
		s.logger.WarnContext(ctx, logRegistrationCallbackFailed, "error", err, "stage", "request_create")
		return
	}
	req.Header.Set("Content-Type", RegistrationCallbackContentType)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.WarnContext(ctx, logRegistrationCallbackFailed, "error", err, "stage", "request_send")
		return
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.logger.InfoContext(ctx, logRegistrationCallbackSuccess, "user_id", user.ID, "status", resp.StatusCode)
	} else {
		s.logger.WarnContext(ctx, logRegistrationCallbackFailed, "status", resp.StatusCode, "user_id", user.ID)
	}
}

// generateSecureToken generates a cryptographically secure random token.
func generateSecureToken(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		for i := range bytes {
			bytes[i] = byte(i)
		}
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}
