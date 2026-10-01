package blueprints

import "errors"

// Registry failures the adapter maps HTTP responses onto. The blueprint service
// re-exports them so its callers keep matching on a single name.
var (
	// ErrRegistryAuthFailed reports a rejected registry credential.
	ErrRegistryAuthFailed = errors.New("registry authentication failed")
	// ErrRegistryPermissionDenied reports a credential without the required scope.
	ErrRegistryPermissionDenied = errors.New("registry token lacks required permissions")
	// ErrRegistryRateLimited reports a throttled registry.
	ErrRegistryRateLimited = errors.New("registry rate limit exceeded")
	// ErrRegistryAPIError reports any other registry API failure.
	ErrRegistryAPIError = errors.New("registry API error")
	// ErrRegistryVersionAlreadyExists reports a blueprint version the registry
	// already carries.
	ErrRegistryVersionAlreadyExists = errors.New("blueprint version already exists in registry")
)

// GitHub App configuration failures raised while building the token provider.
var (
	// ErrGitHubAppIDRequired reports a github-app auth mode without an app ID.
	ErrGitHubAppIDRequired = errors.New("github_app_id is required for github-app auth mode")
	// ErrGitHubAppInstallationIDRequired reports a github-app auth mode without an installation ID.
	ErrGitHubAppInstallationIDRequired = errors.New("github_app_installation_id is required for github-app auth mode")
	// ErrGitHubAppPrivateKeyRequired reports a github-app auth mode without a private key.
	ErrGitHubAppPrivateKeyRequired = errors.New("github_app_private_key is required for github-app auth mode")
)

// Error-wrap operation labels. Each names the HTTP or crypto operation whose
// failure is being wrapped, used as fmt.Errorf("%s: %w", errOp..., err).
const (
	errOpInitGitHubAppTokenProvider = "initialize GitHub App token provider" //nolint:gosec // operation label, not a credential
	errOpParseRSAPrivateKey         = "invalid RSA private key"
	errOpGenerateAppJWT             = "generate app JWT"
	errOpCreateInstallationToken    = "create installation token"
	errOpSignJWT                    = "sign JWT"
	errOpMarshalRequest             = "marshal request"
	errOpCreateRequest              = "create request"
	errOpExecuteRequest             = "execute request"
	errOpDecodeResponse             = "decode response"
)

// Error format strings for registry API failures. Kept as constants so the
// wrapped error text stays consistent across handlers.
const (
	// errFmtGitHubAPIStatus formats an unexpected GitHub API status and body.
	errFmtGitHubAPIStatus = "GitHub API returned %d: %s"
	// errFmtRegistryResourceNotFound wraps ErrRegistryAPIError for a 404 response.
	errFmtRegistryResourceNotFound = "%w: resource not found"
	// errFmtRegistryStatus wraps ErrRegistryAPIError with the failing status code.
	errFmtRegistryStatus = "%w: status %d"
	// errFmtUnreadableResponseBody stands in for an error body that cannot be read.
	errFmtUnreadableResponseBody = "unreadable response body: %v"
)
