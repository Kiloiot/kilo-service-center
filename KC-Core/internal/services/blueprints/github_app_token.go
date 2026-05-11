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
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

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
func newGitHubAppTokenProvider(cfg *config.RegistryProviderConfig, log logger.Logger) (*githubAppTokenProvider, error) {
	if cfg.GitHubAppID == 0 {
		return nil, fmt.Errorf("github_app_id is required for github-app auth mode")
	}
	if cfg.GitHubAppInstallationID == 0 {
		return nil, fmt.Errorf("github_app_installation_id is required for github-app auth mode")
	}
	if cfg.GitHubAppPrivateKey == "" {
		return nil, fmt.Errorf("github_app_private_key is required for github-app auth mode")
	}

	block, _ := pem.Decode([]byte(cfg.GitHubAppPrivateKey))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from github_app_private_key")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("invalid RSA private key: %w", err)
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

	if p.cachedToken != "" && time.Now().Add(5*time.Minute).Before(p.expiresAt) {
		return p.cachedToken, nil
	}

	appJWT, err := p.generateAppJWT()
	if err != nil {
		return "", fmt.Errorf("generate app JWT: %w", err)
	}

	token, expiresAt, err := p.createInstallationToken(ctx, appJWT)
	if err != nil {
		return "", fmt.Errorf("create installation token: %w", err)
	}

	p.cachedToken = token
	p.expiresAt = expiresAt
	p.logger.InfoContext(ctx, "GitHub App installation token refreshed",
		"expires_at", expiresAt.Format(time.RFC3339))

	return token, nil
}

// generateAppJWT creates a short-lived RS256 JWT signed with the app's private key.
// Uses stdlib crypto only — no external JWT library needed.
func (p *githubAppTokenProvider) generateAppJWT() (string, error) {
	now := time.Now()
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	payload := map[string]interface{}{
		"iat": now.Add(-60 * time.Second).Unix(),
		"exp": now.Add(10 * time.Minute).Unix(),
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
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)
	return signingInput + "." + signatureB64, nil
}

// createInstallationToken exchanges the app JWT for an installation access token.
func (p *githubAppTokenProvider) createInstallationToken(ctx context.Context, appJWT string) (string, time.Time, error) {
	apiURL := strings.TrimRight(p.apiURL, "/")
	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", apiURL, p.installationID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", appJWT))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", time.Time{}, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var result installationTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", time.Time{}, fmt.Errorf("decode response: %w", err)
	}

	return result.Token, result.ExpiresAt, nil
}
