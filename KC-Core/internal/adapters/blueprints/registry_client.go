// Package blueprints holds the outbound clients the blueprint service uses to
// reach the external blueprint registry. They live outside the service package
// so the service depends on ports rather than on HTTP.
package blueprints

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/blueprint"

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

// authModeGitHubApp selects GitHub App authentication for the registry
// provider; the default is a static token.
const authModeGitHubApp = "github-app"

// GitHub REST API path templates used by the registry client.
const (
	ghPathBranchRef     = "%s/repos/%s/%s/git/refs/heads/%s"
	ghPathContentsAtRef = "%s/repos/%s/%s/contents/%s?ref=%s"
	ghPathRefs          = "%s/repos/%s/%s/git/refs"
	ghPathContents      = "%s/repos/%s/%s/contents/%s"
	ghPathPulls         = "%s/repos/%s/%s/pulls"
	ghBranchRefFmt      = "refs/heads/%s"
)

// Lowercased GitHub 422 body phrases that identify a commit conflict.
const (
	ghConflictNotFastForward        = "update is not a fast forward"
	ghConflictIsAt                  = "is at"
	ghConflictButExpected           = "but expected"
	ghConflictShaNotSupplied        = `"sha" wasn't supplied`
	ghConflictShaNotSuppliedEscaped = `\"sha\" wasn't supplied`
)

func (p *staticTokenProvider) Token(_ context.Context) (string, error) {
	return p.token, nil
}

// RegistryClient handles registry operations for blueprint submission.
// Compatible with any registry provider that supports the configured API.
type RegistryClient struct {
	httpClient    *http.Client
	apiURL        string // Registry provider API base URL (required)
	tokenProvider TokenProvider
	owner         string
	repo          string
	baseBranch    string
	branchPrefix  string
	fileExtension string
	logger        logger.Logger
}

// NewRegistryClient creates a new registry client from configuration.
func NewRegistryClient(cfg *config.RegistryProviderConfig, log logger.Logger) (*RegistryClient, error) {
	timeout := time.Duration(cfg.HTTPTimeout) * time.Second
	if timeout == 0 {
		timeout = time.Duration(config.DefaultRegistryProviderHTTPTimeout) * time.Second
	}

	var tp TokenProvider
	switch cfg.AuthMode {
	case authModeGitHubApp:
		provider, err := newGitHubAppTokenProvider(cfg, log)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", errOpInitGitHubAppTokenProvider, err)
		}
		tp = provider
	default: // "token" or empty
		tp = &staticTokenProvider{token: cfg.Token}
	}

	return &RegistryClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		apiURL:        cfg.APIURL,
		tokenProvider: tp,
		owner:         cfg.Owner,
		repo:          cfg.Repo,
		baseBranch:    cfg.BaseBranch,
		branchPrefix:  cfg.BranchPrefix,
		fileExtension: cfg.FileExtension,
		logger:        log,
	}, nil
}

// BaseBranchSHA retrieves the current SHA of the base branch.
func (c *RegistryClient) BaseBranchSHA(ctx context.Context) (string, error) {
	url := fmt.Sprintf(ghPathBranchRef, c.apiURL, c.owner, c.repo, c.baseBranch)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errOpCreateRequest, err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errOpExecuteRequest, err)
	}
	defer closeResponseBody(ctx, c.logger, resp)

	if resp.StatusCode != http.StatusOK {
		return "", c.handleErrorResponse(ctx, resp)
	}

	var result refResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("%s: %w", errOpDecodeResponse, err)
	}

	return result.Object.SHA, nil
}

// FileExists checks if a file exists in the repository at the given ref.
func (c *RegistryClient) FileExists(ctx context.Context, ref, path string) (bool, error) {
	url := fmt.Sprintf(ghPathContentsAtRef, c.apiURL, c.owner, c.repo, path, ref)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("%s: %w", errOpCreateRequest, err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("%s: %w", errOpExecuteRequest, err)
	}
	defer closeResponseBody(ctx, c.logger, resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, c.handleErrorResponse(ctx, resp)
	}
}

// CreateBranch creates a new branch from the base branch.
func (c *RegistryClient) CreateBranch(ctx context.Context, baseSHA, newBranch string) error {
	url := fmt.Sprintf(ghPathRefs, c.apiURL, c.owner, c.repo)

	payload := createRefRequest{
		Ref: fmt.Sprintf(ghBranchRefFmt, newBranch),
		SHA: baseSHA,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s: %w", errOpMarshalRequest, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s: %w", errOpCreateRequest, err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", errOpExecuteRequest, err)
	}
	defer closeResponseBody(ctx, c.logger, resp)

	if resp.StatusCode != http.StatusCreated {
		return c.handleErrorResponse(ctx, resp)
	}

	return nil
}

// CreateOrUpdateFile creates or updates a file in the repository.
// content must be a base64-encoded string (API expects base64-encoded content).
func (c *RegistryClient) CreateOrUpdateFile(ctx context.Context, branch, path string, content string, message, contributorName, contributorEmail string) (string, error) {
	url := fmt.Sprintf(ghPathContents, c.apiURL, c.owner, c.repo, path)

	payload := CreateFileRequest{
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
		return "", fmt.Errorf("%s: %w", errOpMarshalRequest, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("%s: %w", errOpCreateRequest, err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errOpExecuteRequest, err)
	}
	defer closeResponseBody(ctx, c.logger, resp)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", c.handleFileErrorResponse(ctx, resp)
	}

	var result createFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("%s: %w", errOpDecodeResponse, err)
	}

	return result.Commit.SHA, nil
}

// CreateSubmissionRequest creates a pull request / merge request.
func (c *RegistryClient) CreateSubmissionRequest(ctx context.Context, title, body, head, base string) (string, error) {
	url := fmt.Sprintf(ghPathPulls, c.apiURL, c.owner, c.repo)

	payload := CreatePRRequest{
		Title: title,
		Body:  body,
		Head:  head,
		Base:  base,
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errOpMarshalRequest, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("%s: %w", errOpCreateRequest, err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errOpExecuteRequest, err)
	}
	defer closeResponseBody(ctx, c.logger, resp)

	if resp.StatusCode != http.StatusCreated {
		return "", c.handleErrorResponse(ctx, resp)
	}

	var result createPRResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("%s: %w", errOpDecodeResponse, err)
	}

	return result.HTMLURL, nil
}

// setHeaders sets the common headers for API requests including Bearer token.
func (c *RegistryClient) setHeaders(req *http.Request) {
	req.Header.Set(headerAccept, blueprint.MediaTypeJSON)
	req.Header.Set(blueprint.HeaderContentType, blueprint.MediaTypeJSON)
	if c.tokenProvider != nil {
		token, err := c.tokenProvider.Token(req.Context())
		if err != nil {
			c.logger.ErrorContext(req.Context(), blueprint.LogRegistryTokenResolveFailed, logger.FieldError, err)
			return
		}
		if token != "" {
			req.Header.Set(blueprint.HeaderAuthorization, blueprint.BearerPrefix+token)
		}
	}
}

// handleErrorResponse handles error responses from the API.
// Returns sentinel errors that can be mapped to gRPC tokens by the handler.
func (c *RegistryClient) handleErrorResponse(ctx context.Context, resp *http.Response) error {
	body := errorBody(resp)

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		c.logger.ErrorContext(ctx, blueprint.LogRegistryAuthFailed,
			logger.FieldStatus, resp.StatusCode, logger.FieldBody, string(body))
		return ErrRegistryAuthFailed
	case http.StatusForbidden:
		c.logger.ErrorContext(ctx, LogRegistryPermissionDenied,
			logger.FieldStatus, resp.StatusCode, logger.FieldBody, string(body))
		return ErrRegistryPermissionDenied
	case http.StatusTooManyRequests:
		c.logger.WarnContext(ctx, LogRegistryRateLimited)
		return ErrRegistryRateLimited
	case http.StatusNotFound:
		c.logger.ErrorContext(ctx, LogRegistryResourceNotFound,
			logger.FieldStatus, resp.StatusCode, logger.FieldBody, string(body))
		return fmt.Errorf(errFmtRegistryResourceNotFound, ErrRegistryAPIError)
	default:
		c.logger.ErrorContext(ctx, blueprint.LogRegistryAPIError,
			logger.FieldStatus, resp.StatusCode, logger.FieldBody, string(body))
		return fmt.Errorf(errFmtRegistryStatus, ErrRegistryAPIError, resp.StatusCode)
	}
}

// handleFileErrorResponse handles error responses from file creation/update operations.
// Applies precise conflict detection for 409 and 422 status codes to distinguish
// version-already-exists from other API errors.
func (c *RegistryClient) handleFileErrorResponse(ctx context.Context, resp *http.Response) error {
	body := errorBody(resp)

	switch resp.StatusCode {
	case http.StatusConflict:
		c.logger.WarnContext(ctx, LogRegistryFileConflict, logger.FieldBody, string(body))
		return ErrRegistryVersionAlreadyExists
	case http.StatusUnprocessableEntity:
		if isCommitConflict422(body) {
			c.logger.WarnContext(ctx, LogRegistryCommitConflict, logger.FieldBody, string(body))
			return ErrRegistryVersionAlreadyExists
		}
		c.logger.ErrorContext(ctx, blueprint.LogRegistryAPIUnprocessable, logger.FieldBody, string(body))
		return fmt.Errorf(errFmtRegistryStatus, ErrRegistryAPIError, resp.StatusCode)
	default:
		return c.handleErrorResponse(ctx, resp)
	}
}

// isCommitConflict422 checks if a 422 response body contains known commit conflict phrases.
func isCommitConflict422(body []byte) bool {
	s := strings.ToLower(string(body))
	if strings.Contains(s, ghConflictNotFastForward) {
		return true
	}
	if strings.Contains(s, ghConflictIsAt) && strings.Contains(s, ghConflictButExpected) {
		return true
	}
	if strings.Contains(s, ghConflictShaNotSupplied) || strings.Contains(s, ghConflictShaNotSuppliedEscaped) {
		return true
	}
	return false
}
