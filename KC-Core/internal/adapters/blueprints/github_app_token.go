package blueprints

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/blueprint"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// githubAppTokenProvider generates short-lived installation access tokens
// from a GitHub App's private key. Tokens are cached until 5 minutes before expiry.
type githubAppTokenProvider struct {
	appID          int64
	installationID int64
	privateKey     *rsa.PrivateKey
	apiURL         string
	httpClient     *http.Client
	logger         logger.Logger

	mu          sync.Mutex
	cachedToken string
	expiresAt   time.Time
}

type installationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// newGitHubAppTokenProvider creates a token provider from config.
// JWT header fields for the GitHub App token exchange.
const (
	jwtHeaderAlg = "alg"
	jwtHeaderTyp = "typ"
	jwtAlgRS256  = "RS256"
	jwtTypJWT    = "JWT"
)

// ghPathInstallationTokens is the GitHub REST path exchanging an app JWT for
// an installation token.
const ghPathInstallationTokens = "%s/app/installations/%d/access_tokens" //nolint:gosec // G101: REST path template, not a credential

func newGitHubAppTokenProvider(cfg *config.RegistryProviderConfig, log logger.Logger) (*githubAppTokenProvider, error) {
	if cfg.GitHubAppID == 0 {
		return nil, ErrGitHubAppIDRequired
	}
	if cfg.GitHubAppInstallationID == 0 {
		return nil, ErrGitHubAppInstallationIDRequired
	}
	if cfg.GitHubAppPrivateKey == "" {
		return nil, ErrGitHubAppPrivateKeyRequired
	}

	block, _ := pem.Decode([]byte(cfg.GitHubAppPrivateKey))
	if block == nil {
		return nil, blueprint.ErrGitHubAppKeyPEM
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errOpParseRSAPrivateKey, err)
	}

	timeout := time.Duration(cfg.HTTPTimeout) * time.Second
	if timeout == 0 {
		timeout = time.Duration(config.DefaultRegistryProviderHTTPTimeout) * time.Second
	}

	return &githubAppTokenProvider{
		appID:          cfg.GitHubAppID,
		installationID: cfg.GitHubAppInstallationID,
		privateKey:     key,
		apiURL:         cfg.APIURL,
		httpClient:     &http.Client{Timeout: timeout},
		logger:         log,
	}, nil
}

// Token returns a valid installation access token, refreshing if needed.
func (p *githubAppTokenProvider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cachedToken != "" && time.Now().Add(blueprint.GitHubTokenRefreshSkew).Before(p.expiresAt) {
		return p.cachedToken, nil
	}

	appJWT, err := p.generateAppJWT()
	if err != nil {
		return "", fmt.Errorf("%s: %w", errOpGenerateAppJWT, err)
	}

	token, expiresAt, err := p.createInstallationToken(ctx, appJWT)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errOpCreateInstallationToken, err)
	}

	p.cachedToken = token
	p.expiresAt = expiresAt
	p.logger.InfoContext(ctx, LogGitHubTokenRefreshed,
		logger.FieldExpiresAt, expiresAt.Format(time.RFC3339))

	return token, nil
}

// generateAppJWT creates a short-lived RS256 JWT signed with the app's private key.
// Uses stdlib crypto only — no external JWT library needed.
func (p *githubAppTokenProvider) generateAppJWT() (string, error) {
	now := time.Now()
	header := map[string]string{jwtHeaderAlg: jwtAlgRS256, jwtHeaderTyp: jwtTypJWT}
	payload := map[string]interface{}{
		"iat": now.Add(-blueprint.GitHubJWTIssuedAtSkew).Unix(),
		"exp": now.Add(blueprint.GitHubJWTLifetime).Unix(),
		"iss": p.appID,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := headerB64 + "." + payloadB64

	hash := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, p.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("%s: %w", errOpSignJWT, err)
	}

	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)
	return signingInput + "." + signatureB64, nil
}

// createInstallationToken exchanges the app JWT for an installation access token.
func (p *githubAppTokenProvider) createInstallationToken(ctx context.Context, appJWT string) (string, time.Time, error) {
	apiURL := strings.TrimRight(p.apiURL, "/")
	url := fmt.Sprintf(ghPathInstallationTokens, apiURL, p.installationID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%s: %w", errOpCreateRequest, err)
	}
	req.Header.Set(headerAccept, mediaTypeGitHubJSON)
	req.Header.Set(blueprint.HeaderAuthorization, blueprint.BearerPrefix+appJWT)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%s: %w", errOpExecuteRequest, err)
	}
	defer closeResponseBody(ctx, p.logger, resp)

	if resp.StatusCode != http.StatusCreated {
		return "", time.Time{}, fmt.Errorf(errFmtGitHubAPIStatus, resp.StatusCode, string(errorBody(resp)))
	}

	var result installationTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", time.Time{}, fmt.Errorf("%s: %w", errOpDecodeResponse, err)
	}

	return result.Token, result.ExpiresAt, nil
}
