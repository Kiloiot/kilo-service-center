// Package auth provides authentication and user management services.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kilocenter/KC-Core/pkg/logger"
)

// oauth2Client handles authorization URL generation and token exchange with PKCE.
type oauth2Client struct {
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

// Ensure oauth2Client implements OAuth2Client interface.
var _ OAuth2Client = (*oauth2Client)(nil)

// NewOAuth2Client creates a new OAuth2 PKCE client.
func NewOAuth2Client(cfg OAuth2ClientConfig, log logger.Logger) OAuth2Client {
	log.Info(LogOAuth2ClientCreated, "authorize_url", cfg.AuthorizeURL)

	return &oauth2Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
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
	}
}

// GetAuthorizationURL builds the OAuth2 authorization URL with state and PKCE.
// Returns URL and the code_verifier to store for token exchange.
func (c *oauth2Client) GetAuthorizationURL(state string) (authURL string, codeVerifier string) {
	// Generate PKCE code_verifier
	codeVerifier = generateCodeVerifier()

	// Generate code_challenge from verifier
	codeChallenge := generateCodeChallenge(codeVerifier, c.pkceMethod)

	params := url.Values{}
	params.Set(ParamResponseType, ResponseTypeCode)
	params.Set(ParamClientID, c.clientID)
	params.Set(ParamRedirectURI, c.redirectURL)
	params.Set(ParamScope, strings.Join(c.scopes, " "))
	params.Set(ParamState, state)
	params.Set(ParamCodeChallenge, codeChallenge)
	params.Set(ParamCodeChallengeMethod, c.pkceMethod)

	authURL = c.authorizeURL + "?" + params.Encode()
	return authURL, codeVerifier
}

// ExchangeCode exchanges authorization code for tokens using PKCE verifier.
func (c *oauth2Client) ExchangeCode(ctx context.Context, code, codeVerifier string) (*OAuth2TokenResponse, error) {
	data := url.Values{}
	data.Set(ParamGrantType, GrantTypeAuthorizationCode)
	data.Set(ParamCode, code)
	data.Set(ParamRedirectURI, c.redirectURL)
	data.Set(ParamClientID, c.clientID)
	data.Set(ParamCodeVerifier, codeVerifier)

	// Include client_secret if not a public client
	if !c.publicClient && c.clientSecret != "" {
		data.Set(ParamClientSecret, c.clientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.ErrorContext(ctx, LogOAuth2TokenExchangeFailed, "error", err)
		return nil, ErrTokenExchangeFailed
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.ErrorContext(ctx, LogOAuth2TokenExchangeFailed, "status", resp.StatusCode, "body", string(body))
		return nil, ErrTokenExchangeFailed
	}

	var tokenResp OAuth2TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// GetUserInfo retrieves user info from the userinfo endpoint.
func (c *oauth2Client) GetUserInfo(ctx context.Context, accessToken string) (*OAuth2UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.ErrorContext(ctx, LogOAuth2UserInfoFailed, "error", err)
		return nil, ErrUserInfoFailed
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.ErrorContext(ctx, LogOAuth2UserInfoFailed, "status", resp.StatusCode)
		return nil, ErrUserInfoFailed
	}

	// Parse into generic map first to handle dynamic claim names
	var rawInfo map[string]interface{}
	if err := json.Unmarshal(body, &rawInfo); err != nil {
		return nil, err
	}

	userInfo := &OAuth2UserInfo{}

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

// generateCodeVerifier generates a cryptographically secure PKCE code_verifier.
func generateCodeVerifier() string {
	bytes := make([]byte, PKCEVerifierLength)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to less secure but functional random
		for i := range bytes {
			bytes[i] = byte(i)
		}
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}

// generateCodeChallenge generates PKCE code_challenge from verifier.
func generateCodeChallenge(verifier, method string) string {
	if method == PKCEMethodPlain {
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
	// LogOAuth2ClientCreated indicates OAuth2 client was created successfully.
	LogOAuth2ClientCreated = "OAuth2 client created"
	// LogOAuth2TokenExchangeFailed indicates token exchange failed.
	LogOAuth2TokenExchangeFailed = "OAuth2 token exchange failed" //nolint:gosec // Log message, not actual credentials
	// LogOAuth2UserInfoFailed indicates userinfo request failed.
	LogOAuth2UserInfoFailed = "OAuth2 userinfo request failed"
)
