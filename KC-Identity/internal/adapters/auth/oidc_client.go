// Package auth provides authentication and user management services.
package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	authsvc "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
)

// oidcClient handles OIDC discovery, token exchange, and ID token validation.
type oidcClient struct {
	httpClient   *http.Client
	providerURL  string
	clientID     string
	clientSecret string
	redirectURL  string
	scopes       []string
	discovery    *oidcDiscovery
	logger       logger.Logger
}

// oidcDiscovery contains OIDC provider discovery metadata.
type oidcDiscovery struct {
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

// Ensure oidcClient implements authsvc.OIDCClient interface.
var _ authsvc.OIDCClient = (*oidcClient)(nil)

const errFmtOIDCDiscoveryStatus = "%s: unexpected status %s"

// NewOIDCClient creates a new OIDC client with discovery. ctx scopes the
// provider metadata fetch performed at construction.
// errFmtOIDCDiscoveryStatus wraps a non-200 discovery response status.
func NewOIDCClient(ctx context.Context, cfg OIDCClientConfig, log logger.Logger) (authsvc.OIDCClient, error) {
	client := &oidcClient{
		httpClient: &http.Client{
			Timeout: HTTPClientTimeout,
		},
		providerURL:  strings.TrimSuffix(cfg.ProviderURL, "/"),
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		redirectURL:  cfg.RedirectURL,
		scopes:       cfg.Scopes,
		logger:       log,
	}

	// Perform OIDC discovery
	if err := client.discover(ctx); err != nil {
		return nil, err
	}

	log.Info(logOIDCClientCreated, logger.FieldProvider, cfg.ProviderURL)
	return client, nil
}

// discover fetches OIDC provider metadata from well-known endpoint.
func (c *oidcClient) discover(ctx context.Context) error {
	discoveryURL := c.providerURL + authsvc.OIDCWellKnownPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.ErrorContext(ctx, logOIDCDiscoveryFailed, logger.FieldError, err)
		return errors.New(logOIDCDiscoveryFailed + ": " + err.Error())
	}
	defer closeResponseBody(ctx, c.logger, resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.logger.ErrorContext(ctx, logOIDCDiscoveryFailed, logger.FieldStatus, resp.StatusCode)
		return fmt.Errorf(errFmtOIDCDiscoveryStatus, logOIDCDiscoveryFailed, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var discovery oidcDiscovery
	if err := json.Unmarshal(body, &discovery); err != nil {
		return err
	}

	c.discovery = &discovery
	c.logger.InfoContext(ctx, logOIDCDiscoveryComplete, logger.FieldIssuer, discovery.Issuer)
	return nil
}

// GetAuthorizationURL builds the OIDC authorization URL with state and nonce.
func (c *oidcClient) GetAuthorizationURL(state, nonce string) string {
	params := url.Values{}
	params.Set(authsvc.ParamResponseType, authsvc.ResponseTypeCode)
	params.Set(authsvc.ParamClientID, c.clientID)
	params.Set(authsvc.ParamRedirectURI, c.redirectURL)
	params.Set(authsvc.ParamScope, strings.Join(c.scopes, " "))
	params.Set(authsvc.ParamState, state)
	params.Set(authsvc.ParamNonce, nonce)

	return c.discovery.AuthorizationEndpoint + "?" + params.Encode()
}

// ExchangeCode exchanges authorization code for tokens.
func (c *oidcClient) ExchangeCode(ctx context.Context, code string) (*authsvc.OIDCTokenResponse, error) {
	data := url.Values{}
	data.Set(authsvc.ParamGrantType, authsvc.GrantTypeAuthorizationCode)
	data.Set(authsvc.ParamCode, code)
	data.Set(authsvc.ParamRedirectURI, c.redirectURL)
	data.Set(authsvc.ParamClientID, c.clientID)
	data.Set(authsvc.ParamClientSecret, c.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.discovery.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set(HeaderContentType, MediaTypeFormURLEncoded)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.ErrorContext(ctx, logOIDCTokenExchangeFailed, logger.FieldError, err)
		return nil, authsvc.ErrTokenExchangeFailed
	}
	defer closeResponseBody(ctx, c.logger, resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.ErrorContext(ctx, logOIDCTokenExchangeFailed, logger.FieldStatus, resp.StatusCode, logger.FieldBody, string(body))
		return nil, authsvc.ErrTokenExchangeFailed
	}

	var tokenResp authsvc.OIDCTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// ValidateIDToken validates the ID token and extracts claims.
// Returns typed claims AND raw claim map for custom claim extraction.
// Note: This is a simplified validation that checks structure and nonce.
// Production deployments should use a proper JWT library with JWKS validation.
func (c *oidcClient) ValidateIDToken(ctx context.Context, idToken, expectedNonce string) (*authsvc.OIDCClaims, map[string]any, error) {
	// Split JWT into parts
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		c.logger.ErrorContext(ctx, logOIDCIDTokenInvalid, logger.FieldReason, oidcReasonInvalidFormat)
		return nil, nil, authsvc.ErrIDTokenInvalid
	}

	// Decode payload (middle part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		c.logger.ErrorContext(ctx, logOIDCIDTokenInvalid, logger.FieldReason, oidcReasonBase64DecodeFailed)
		return nil, nil, authsvc.ErrIDTokenInvalid
	}

	// Unmarshal into raw map FIRST (preserves all claims including custom org claims)
	var rawClaims map[string]any
	if err := json.Unmarshal(payload, &rawClaims); err != nil {
		c.logger.ErrorContext(ctx, logOIDCIDTokenInvalid, logger.FieldReason, oidcReasonUnmarshalRawFailed)
		return nil, nil, authsvc.ErrIDTokenInvalid
	}

	// Then unmarshal into typed struct
	var claims authsvc.OIDCClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		c.logger.ErrorContext(ctx, logOIDCIDTokenInvalid, logger.FieldReason, oidcReasonUnmarshalFailed)
		return nil, nil, authsvc.ErrIDTokenInvalid
	}

	// Validate issuer
	if claims.Issuer != c.discovery.Issuer {
		c.logger.ErrorContext(ctx, logOIDCIDTokenInvalid, logger.FieldReason, oidcReasonIssuerMismatch, logger.FieldExpected, c.discovery.Issuer, logger.FieldGot, claims.Issuer)
		return nil, nil, authsvc.ErrIDTokenInvalid
	}

	// Validate audience (can be string or array in JWT)
	if claims.Audience != c.clientID {
		c.logger.ErrorContext(ctx, logOIDCIDTokenInvalid, logger.FieldReason, oidcReasonAudienceMismatch)
		return nil, nil, authsvc.ErrIDTokenInvalid
	}

	// Validate expiration
	if claims.ExpiresAt < time.Now().Unix() {
		c.logger.ErrorContext(ctx, logOIDCIDTokenInvalid, logger.FieldReason, oidcReasonTokenExpired)
		return nil, nil, authsvc.ErrIDTokenInvalid
	}

	// Validate nonce
	if expectedNonce != "" && claims.Nonce != expectedNonce {
		c.logger.ErrorContext(ctx, logOIDCNonceMismatch, logger.FieldExpected, expectedNonce, logger.FieldGot, claims.Nonce)
		return nil, nil, authsvc.ErrNonceMismatch
	}

	return &claims, rawClaims, nil
}

// ============================================================================
// OIDC Client Log Messages
// ============================================================================

const (
	// logOIDCClientCreated indicates OIDC client was created successfully.
	logOIDCClientCreated = "OIDC client created"
	// logOIDCDiscoveryFailed indicates OIDC discovery failed.
	logOIDCDiscoveryFailed = "OIDC discovery failed"
	// logOIDCDiscoveryComplete indicates OIDC discovery completed successfully.
	logOIDCDiscoveryComplete = "OIDC discovery complete"
	// logOIDCTokenExchangeFailed indicates token exchange failed.
	logOIDCTokenExchangeFailed = "OIDC token exchange failed" //nolint:gosec // Log message, not actual credentials
	// logOIDCIDTokenInvalid indicates ID token validation failed.
	logOIDCIDTokenInvalid = "OIDC ID token invalid" //nolint:gosec // Log message, not actual credentials
	// logOIDCNonceMismatch indicates nonce validation failed.
	logOIDCNonceMismatch = "OIDC nonce mismatch"
)
