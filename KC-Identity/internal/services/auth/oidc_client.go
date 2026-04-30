// Package auth provides authentication and user management services.
package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kilocenter/KC-Core/pkg/logger"
)

// oidcClient handles OIDC discovery, token exchange, and ID token validation.
type oidcClient struct {
	httpClient   *http.Client
	providerURL  string
	clientID     string
	clientSecret string
	redirectURL  string
	scopes       []string
	discovery    *OIDCDiscovery
	logger       logger.Logger
}

// OIDCDiscovery contains OIDC provider discovery metadata.
type OIDCDiscovery struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	UserInfoEndpoint      string   `json:"userinfo_endpoint"`
	JWKSUri               string   `json:"jwks_uri"`
	ScopesSupported       []string `json:"scopes_supported"`
}

// OIDCClientConfig contains OIDC client configuration.
type OIDCClientConfig struct {
	ProviderURL  string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// Ensure oidcClient implements OIDCClient interface.
var _ OIDCClient = (*oidcClient)(nil)

// NewOIDCClient creates a new OIDC client with discovery.
func NewOIDCClient(cfg OIDCClientConfig, log logger.Logger) (OIDCClient, error) {
	client := &oidcClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		providerURL:  strings.TrimSuffix(cfg.ProviderURL, "/"),
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		redirectURL:  cfg.RedirectURL,
		scopes:       cfg.Scopes,
		logger:       log,
	}

	// Perform OIDC discovery
	if err := client.discover(context.Background()); err != nil {
		return nil, err
	}

	log.Info(LogOIDCClientCreated, "provider", cfg.ProviderURL)
	return client, nil
}

// discover fetches OIDC provider metadata from well-known endpoint.
func (c *oidcClient) discover(ctx context.Context) error {
	discoveryURL := c.providerURL + OIDCWellKnownPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error(LogOIDCDiscoveryFailed, "error", err)
		return errors.New(LogOIDCDiscoveryFailed + ": " + err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error(LogOIDCDiscoveryFailed, "status", resp.StatusCode)
		return errors.New(LogOIDCDiscoveryFailed + ": unexpected status " + resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var discovery OIDCDiscovery
	if err := json.Unmarshal(body, &discovery); err != nil {
		return err
	}

	c.discovery = &discovery
	c.logger.Info(LogOIDCDiscoveryComplete, "issuer", discovery.Issuer)
	return nil
}

// GetAuthorizationURL builds the OIDC authorization URL with state and nonce.
func (c *oidcClient) GetAuthorizationURL(state, nonce string) string {
	params := url.Values{}
	params.Set(ParamResponseType, ResponseTypeCode)
	params.Set(ParamClientID, c.clientID)
	params.Set(ParamRedirectURI, c.redirectURL)
	params.Set(ParamScope, strings.Join(c.scopes, " "))
	params.Set(ParamState, state)
	params.Set(ParamNonce, nonce)

	return c.discovery.AuthorizationEndpoint + "?" + params.Encode()
}

// ExchangeCode exchanges authorization code for tokens.
func (c *oidcClient) ExchangeCode(ctx context.Context, code string) (*OIDCTokenResponse, error) {
	data := url.Values{}
	data.Set(ParamGrantType, GrantTypeAuthorizationCode)
	data.Set(ParamCode, code)
	data.Set(ParamRedirectURI, c.redirectURL)
	data.Set(ParamClientID, c.clientID)
	data.Set(ParamClientSecret, c.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.discovery.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.ErrorContext(ctx, LogOIDCTokenExchangeFailed, "error", err)
		return nil, ErrTokenExchangeFailed
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.ErrorContext(ctx, LogOIDCTokenExchangeFailed, "status", resp.StatusCode, "body", string(body))
		return nil, ErrTokenExchangeFailed
	}

	var tokenResp OIDCTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// ValidateIDToken validates the ID token and extracts claims.
// Returns typed claims AND raw claim map for custom claim extraction.
// Note: This is a simplified validation that checks structure and nonce.
// Production deployments should use a proper JWT library with JWKS validation.
func (c *oidcClient) ValidateIDToken(ctx context.Context, idToken, expectedNonce string) (*OIDCClaims, map[string]any, error) {
	// Split JWT into parts
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		c.logger.ErrorContext(ctx, LogOIDCIDTokenInvalid, "reason", "invalid format")
		return nil, nil, ErrIDTokenInvalid
	}

	// Decode payload (middle part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		c.logger.ErrorContext(ctx, LogOIDCIDTokenInvalid, "reason", "base64 decode failed")
		return nil, nil, ErrIDTokenInvalid
	}

	// Unmarshal into raw map FIRST (preserves all claims including custom org claims)
	var rawClaims map[string]any
	if err := json.Unmarshal(payload, &rawClaims); err != nil {
		c.logger.ErrorContext(ctx, LogOIDCIDTokenInvalid, "reason", "json unmarshal raw failed")
		return nil, nil, ErrIDTokenInvalid
	}

	// Then unmarshal into typed struct
	var claims OIDCClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		c.logger.ErrorContext(ctx, LogOIDCIDTokenInvalid, "reason", "json unmarshal failed")
		return nil, nil, ErrIDTokenInvalid
	}

	// Validate issuer
	if claims.Issuer != c.discovery.Issuer {
		c.logger.ErrorContext(ctx, LogOIDCIDTokenInvalid, "reason", "issuer mismatch", "expected", c.discovery.Issuer, "got", claims.Issuer)
		return nil, nil, ErrIDTokenInvalid
	}

	// Validate audience (can be string or array in JWT)
	if claims.Audience != c.clientID {
		c.logger.ErrorContext(ctx, LogOIDCIDTokenInvalid, "reason", "audience mismatch")
		return nil, nil, ErrIDTokenInvalid
	}

	// Validate expiration
	if claims.ExpiresAt < time.Now().Unix() {
		c.logger.ErrorContext(ctx, LogOIDCIDTokenInvalid, "reason", "token expired")
		return nil, nil, ErrIDTokenInvalid
	}

	// Validate nonce
	if expectedNonce != "" && claims.Nonce != expectedNonce {
		c.logger.ErrorContext(ctx, LogOIDCNonceMismatch, "expected", expectedNonce, "got", claims.Nonce)
		return nil, nil, ErrNonceMismatch
	}

	return &claims, rawClaims, nil
}

// ============================================================================
// OIDC Client Log Messages
// ============================================================================

const (
	// LogOIDCClientCreated indicates OIDC client was created successfully.
	LogOIDCClientCreated = "OIDC client created"
	// LogOIDCDiscoveryFailed indicates OIDC discovery failed.
	LogOIDCDiscoveryFailed = "OIDC discovery failed"
	// LogOIDCDiscoveryComplete indicates OIDC discovery completed successfully.
	LogOIDCDiscoveryComplete = "OIDC discovery complete"
	// LogOIDCTokenExchangeFailed indicates token exchange failed.
	LogOIDCTokenExchangeFailed = "OIDC token exchange failed" //nolint:gosec // Log message, not actual credentials
	// LogOIDCUserInfoFailed indicates userinfo request failed.
	LogOIDCUserInfoFailed = "OIDC userinfo request failed"
	// LogOIDCIDTokenInvalid indicates ID token validation failed.
	LogOIDCIDTokenInvalid = "OIDC ID token invalid" //nolint:gosec // Log message, not actual credentials
	// LogOIDCNonceMismatch indicates nonce validation failed.
	LogOIDCNonceMismatch = "OIDC nonce mismatch"
)
