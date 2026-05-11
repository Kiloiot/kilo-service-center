// Package blueprints provides blueprint service implementation for gRPC layer.
package blueprints

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// TokenProvider resolves the Bearer token for registry API requests.
// Implementations include static tokens and GitHub App installation tokens.
type TokenProvider interface {
	Token(ctx context.Context) (string, error)
}

// staticTokenProvider returns a fixed token (PAT or other static credential).
type staticTokenProvider struct {
	token string
}

func (p *staticTokenProvider) Token(_ context.Context) (string, error) {
	return p.token, nil
}

// registryClient handles registry operations for blueprint submission.
// Compatible with any registry provider that supports the configured API.
type registryClient struct {
	httpClient    *http.Client
	apiURL        string // Registry provider API base URL (required)
	tokenProvider TokenProvider
	owner         string
	repo          string
	baseBranch    string
	branchPrefix  string
	blueprintPath string
	fileExtension string
	logger        logger.Logger
}

// newRegistryClient creates a new registry client from configuration.
func newRegistryClient(cfg *config.RegistryProviderConfig, log logger.Logger) (*registryClient, error) {
	timeout := time.Duration(cfg.HTTPTimeout) * time.Second
	if timeout == 0 {
		timeout = time.Duration(config.DefaultRegistryProviderHTTPTimeout) * time.Second
	}

	var tp TokenProvider
	switch cfg.AuthMode {
	case "github-app":
		provider, err := newGitHubAppTokenProvider(cfg, log)
		if err != nil {
			return nil, fmt.Errorf("initialize GitHub App token provider: %w", err)
		}
		tp = provider
	default: // "token" or empty
		tp = &staticTokenProvider{token: cfg.Token}
	}

	return &registryClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		apiURL:        cfg.APIURL,
		tokenProvider: tp,
		owner:         cfg.Owner,
		repo:          cfg.Repo,
		baseBranch:    cfg.BaseBranch,
		branchPrefix:  cfg.BranchPrefix,
		blueprintPath: cfg.BlueprintPath,
		fileExtension: cfg.FileExtension,
		logger:        log,
	}, nil
}

// getBaseBranchSHA retrieves the current SHA of the base branch.
func (c *registryClient) getBaseBranchSHA(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/git/refs/heads/%s", c.apiURL, c.owner, c.repo, c.baseBranch)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // Error from Close() in defer is not actionable

	if resp.StatusCode != http.StatusOK {
		return "", c.handleErrorResponse(ctx, resp)
	}

	var result refResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.Object.SHA, nil
}

// fileExists checks if a file exists in the repository at the given ref.
func (c *registryClient) fileExists(ctx context.Context, ref, path string) (bool, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", c.apiURL, c.owner, c.repo, path, ref)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // Error from Close() in defer is not actionable

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, c.handleErrorResponse(ctx, resp)
	}
}

// createBranch creates a new branch from the base branch.
func (c *registryClient) createBranch(ctx context.Context, baseSHA, newBranch string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/git/refs", c.apiURL, c.owner, c.repo)

	payload := createRefRequest{
		Ref: fmt.Sprintf("refs/heads/%s", newBranch),
		SHA: baseSHA,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // Error from Close() in defer is not actionable

	if resp.StatusCode != http.StatusCreated {
		return c.handleErrorResponse(ctx, resp)
	}

	return nil
}

// createOrUpdateFile creates or updates a file in the repository.
// content must be a base64-encoded string (API expects base64-encoded content).
func (c *registryClient) createOrUpdateFile(ctx context.Context, branch, path string, content string, message, contributorName, contributorEmail string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.apiURL, c.owner, c.repo, path)

	payload := createFileRequest{
		Message: message,
		Content: content,
		Branch:  branch,
		Committer: committer{
			Name:  contributorName,
			Email: contributorEmail,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // Error from Close() in defer is not actionable

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", c.handleFileErrorResponse(ctx, resp)
	}

	var result createFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.Commit.SHA, nil
}

// createSubmissionRequest creates a pull request / merge request.
func (c *registryClient) createSubmissionRequest(ctx context.Context, title, body, head, base string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls", c.apiURL, c.owner, c.repo)

	payload := createPRRequest{
		Title: title,
		Body:  body,
		Head:  head,
		Base:  base,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // Error from Close() in defer is not actionable

	if resp.StatusCode != http.StatusCreated {
		return "", c.handleErrorResponse(ctx, resp)
	}

	var result createPRResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.HTMLURL, nil
}

// setHeaders sets the common headers for API requests including Bearer token.
func (c *registryClient) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if c.tokenProvider != nil {
		token, err := c.tokenProvider.Token(req.Context())
		if err != nil {
			c.logger.ErrorContext(req.Context(), "failed to resolve registry token", "error", err)
			return
		}
		if token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		}
	}
}

// handleErrorResponse handles error responses from the API.
// Returns sentinel errors that can be mapped to gRPC tokens by the handler.
func (c *registryClient) handleErrorResponse(ctx context.Context, resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		c.logger.ErrorContext(ctx, "registry auth failed: invalid or expired token",
			"status", resp.StatusCode, "body", string(body))
		return ErrRegistryAuthFailed
	case http.StatusForbidden:
		c.logger.ErrorContext(ctx, "registry permission denied: token lacks required scopes",
			"status", resp.StatusCode, "body", string(body))
		return ErrRegistryPermissionDenied
	case http.StatusTooManyRequests:
		c.logger.WarnContext(ctx, "registry rate limited")
		return ErrRegistryRateLimited
	case http.StatusNotFound:
		c.logger.ErrorContext(ctx, "registry resource not found",
			"status", resp.StatusCode, "body", string(body))
		return fmt.Errorf("%w: resource not found", ErrRegistryAPIError)
	default:
		c.logger.ErrorContext(ctx, "registry API error",
			"status", resp.StatusCode, "body", string(body))
		return fmt.Errorf("%w: status %d", ErrRegistryAPIError, resp.StatusCode)
	}
}

// handleFileErrorResponse handles error responses from file creation/update operations.
// Applies precise conflict detection for 409 and 422 status codes to distinguish
// version-already-exists from other API errors.
func (c *registryClient) handleFileErrorResponse(ctx context.Context, resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusConflict:
		c.logger.WarnContext(ctx, "registry file conflict (409)", "body", string(body))
		return ErrRegistryVersionAlreadyExists
	case http.StatusUnprocessableEntity:
		if isCommitConflict422(body) {
			c.logger.WarnContext(ctx, "registry commit conflict (422)", "body", string(body))
			return ErrRegistryVersionAlreadyExists
		}
		c.logger.ErrorContext(ctx, "registry API error (422)", "body", string(body))
		return fmt.Errorf("%w: status %d", ErrRegistryAPIError, resp.StatusCode)
	default:
		return c.handleErrorResponse(ctx, resp)
	}
}

// isCommitConflict422 checks if a 422 response body contains known commit conflict phrases.
func isCommitConflict422(body []byte) bool {
	s := strings.ToLower(string(body))
	if strings.Contains(s, "update is not a fast forward") {
		return true
	}
	if strings.Contains(s, "is at") && strings.Contains(s, "but expected") {
		return true
	}
	if strings.Contains(s, `"sha" wasn't supplied`) || strings.Contains(s, `\"sha\" wasn't supplied`) {
		return true
	}
	return false
}
