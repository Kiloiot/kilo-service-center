package blueprints

// Log messages for registry client and GitHub App token operations. Shared
// registry log vocabulary lives in KC-Core/pkg/blueprint; these messages are
// specific to this adapter.
const (
	// LogGitHubTokenRefreshed reports a renewed GitHub App installation token.
	LogGitHubTokenRefreshed = "GitHub App installation token refreshed" //nolint:gosec // log message, not a credential
	// LogFailedCloseResponseBody reports a registry response body that failed to close.
	LogFailedCloseResponseBody = "failed to close registry response body"
	// LogRegistryPermissionDenied reports a 403 from the registry.
	LogRegistryPermissionDenied = "registry permission denied: token lacks required scopes"
	// LogRegistryRateLimited reports a 429 from the registry.
	LogRegistryRateLimited = "registry rate limited"
	// LogRegistryResourceNotFound reports a 404 from the registry.
	LogRegistryResourceNotFound = "registry resource not found"
	// LogRegistryFileConflict reports a 409 file conflict from the registry.
	LogRegistryFileConflict = "registry file conflict (409)"
	// LogRegistryCommitConflict reports a 422 commit conflict from the registry.
	LogRegistryCommitConflict = "registry commit conflict (422)"
)
