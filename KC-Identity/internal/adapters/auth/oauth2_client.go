// Package auth provides authentication and user management services.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	authsvc "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
)

// oauth2Client handles authorization URL generation and token exchange with PKCE.
type oauth2Client struct {
	entropy      io.Reader
	httpClient   *http.Client
	authorizeURL string
	tokenURL     string
	userInfoURL  string
	clientID     string
	clientSecret string
	publicClient bool
	redirectURL  string
	scopes       []string
	pkceMethod   string
	userIDClaim  string
	emailClaim   string
	logger       logger.Logger
}

// OAuth2ClientConfig contains OAuth2 client configuration.
type OAuth2ClientConfig struct {
	AuthorizeURL string
	TokenURL     string
	UserInfoURL  string
	ClientID     string
	ClientSecret string
	PublicClient bool
	RedirectURL  string
	Scopes       []string
	PKCEMethod   string
	UserIDClaim  string
	EmailClaim   string
}

// Ensure oauth2Client implements authsvc.OAuth2Client interface.
var _ authsvc.OAuth2Client = (*oauth2Client)(nil)

// ErrInvalidPKCEMethod reports a configured PKCE code-challenge method that is
// neither S256 nor plain; an unknown value must not silently pick a branch.
var ErrInvalidPKCEMethod = errors.New("oauth2: pkce method must be S256 or plain")

// NewOAuth2Client creates a new OAuth2 PKCE client backed by crypto/rand.
func NewOAuth2Client(cfg OAuth2ClientConfig, log logger.Logger) (authsvc.OAuth2Client, error) {
	return newOAuth2ClientWithEntropy(cfg, log, rand.Reader)
}

// newOAuth2ClientWithEntropy constructs the client with an explicit randomness
// source so tests can force verifier-generation failure.
func newOAuth2ClientWithEntropy(cfg OAuth2ClientConfig, log logger.Logger, entropy io.Reader) (authsvc.OAuth2Client, error) {
	switch cfg.PKCEMethod {
	case authsvc.PKCEMethodS256, authsvc.PKCEMethodPlain:
	default:
		return nil, fmt.Errorf("%w: %q", ErrInvalidPKCEMethod, cfg.PKCEMethod)
	}
	return &oauth2Client{
		entropy: entropy,
		httpClient: &http.Client{
			Timeout: HTTPClientTimeout,
		},
		authorizeURL: cfg.AuthorizeURL,
		tokenURL:     cfg.TokenURL,
		userInfoURL:  cfg.UserInfoURL,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		publicClient: cfg.PublicClient,
		redirectURL:  cfg.RedirectURL,
		scopes:       cfg.Scopes,
		pkceMethod:   cfg.PKCEMethod,
		userIDClaim:  cfg.UserIDClaim,
		emailClaim:   cfg.EmailClaim,
		logger:       log,
	}, nil
}

// GetAuthorizationURL builds the OAuth2 authorization URL with state and PKCE.
// Returns URL and the code_verifier to store for token exchange. A verifier
// that cannot be generated from secure randomness fails the call.
func (c *oauth2Client) GetAuthorizationURL(state string) (authURL string, codeVerifier string, err error) {
	codeVerifier, err = c.generateCodeVerifier()
	if err != nil {
		return "", "", err
	}

	// Generate code_challenge from verifier
	codeChallenge := generateCodeChallenge(codeVerifier, c.pkceMethod)

	params := url.Values{}
	params.Set(authsvc.ParamResponseType, authsvc.ResponseTypeCode)
	params.Set(authsvc.ParamClientID, c.clientID)
	params.Set(authsvc.ParamRedirectURI, c.redirectURL)
	params.Set(authsvc.ParamScope, strings.Join(c.scopes, " "))
	params.Set(authsvc.ParamState, state)
	params.Set(authsvc.ParamCodeChallenge, codeChallenge)
	params.Set(authsvc.ParamCodeChallengeMethod, c.pkceMethod)

	authURL = c.authorizeURL + "?" + params.Encode()
	return authURL, codeVerifier, nil
}

// ExchangeCode exchanges authorization code for tokens using PKCE verifier.
func (c *oauth2Client) ExchangeCode(ctx context.Context, code, codeVerifier string) (*authsvc.OAuth2TokenResponse, error) {
	data := url.Values{}
	data.Set(authsvc.ParamGrantType, authsvc.GrantTypeAuthorizationCode)
	data.Set(authsvc.ParamCode, code)
	data.Set(authsvc.ParamRedirectURI, c.redirectURL)
	data.Set(authsvc.ParamClientID, c.clientID)
	data.Set(authsvc.ParamCodeVerifier, codeVerifier)

	// Include client_secret if not a public client
	if !c.publicClient && c.clientSecret != "" {
		data.Set(authsvc.ParamClientSecret, c.clientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set(HeaderContentType, MediaTypeFormURLEncoded)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.ErrorContext(ctx, logOAuth2TokenExchangeFailed, logger.FieldError, err)
		return nil, authsvc.ErrTokenExchangeFailed
	}
	defer closeResponseBody(ctx, c.logger, resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.ErrorContext(ctx, logOAuth2TokenExchangeFailed, logger.FieldStatus, resp.StatusCode, logger.FieldBody, string(body))
		return nil, authsvc.ErrTokenExchangeFailed
	}

	var tokenResp authsvc.OAuth2TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// GetUserInfo retrieves user info from the userinfo endpoint.
func (c *oauth2Client) GetUserInfo(ctx context.Context, accessToken string) (*authsvc.OAuth2UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(HeaderAuthorization, BearerPrefix+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.ErrorContext(ctx, logOAuth2UserInfoFailed, logger.FieldError, err)
		return nil, authsvc.ErrUserInfoFailed
	}
	defer closeResponseBody(ctx, c.logger, resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.ErrorContext(ctx, logOAuth2UserInfoFailed, logger.FieldStatus, resp.StatusCode)
		return nil, authsvc.ErrUserInfoFailed
	}

	// Parse into generic map first to handle dynamic claim names
	var rawInfo map[string]interface{}
	if err := json.Unmarshal(body, &rawInfo); err != nil {
		return nil, err
	}

	userInfo := &authsvc.OAuth2UserInfo{}

	// Extract user ID from configured claim
	if id, ok := rawInfo[c.userIDClaim].(string); ok {
		userInfo.Sub = id
	}
	if id, ok := rawInfo["id"].(string); ok {
		userInfo.ID = id
	}

	// Extract email from configured claim
	if email, ok := rawInfo[c.emailClaim].(string); ok {
		userInfo.Email = email
	}

	// Extract email_verified
	if verified, ok := rawInfo["email_verified"].(bool); ok {
		userInfo.EmailVerified = verified
	}

	// Extract name
	if name, ok := rawInfo["name"].(string); ok {
		userInfo.Name = name
	}

	return userInfo, nil
}

// generateCodeVerifier generates a cryptographically secure PKCE
// code_verifier from the injected entropy source. A read failure is returned;
// there is no predictable fallback.
func (c *oauth2Client) generateCodeVerifier() (string, error) {
	bytes := make([]byte, authsvc.PKCEVerifierLength)
	if _, err := io.ReadFull(c.entropy, bytes); err != nil {
		return "", fmt.Errorf("%w: %w", authsvc.ErrEntropyUnavailable, err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// generateCodeChallenge generates PKCE code_challenge from verifier.
func generateCodeChallenge(verifier, method string) string {
	if method == authsvc.PKCEMethodPlain {
		return verifier
	}
	// S256: SHA256(verifier) then base64url encode
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// ============================================================================
// OAuth2 Client Log Messages
// ============================================================================

const (
	// logOAuth2TokenExchangeFailed indicates token exchange failed.
	logOAuth2TokenExchangeFailed = "OAuth2 token exchange failed" //nolint:gosec // Log message, not actual credentials
	// logOAuth2UserInfoFailed indicates userinfo request failed.
	logOAuth2UserInfoFailed = "OAuth2 userinfo request failed"
)
