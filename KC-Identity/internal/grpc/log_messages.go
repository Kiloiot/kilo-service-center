// Package grpc log message constants.
// Each Log* identifier names a structured log message produced by the
// identity gRPC handlers so log text stays consistent and greppable.
package grpc

// Log messages produced by the identity gRPC handlers, grouped by handler
// family. Message text is part of the operational log contract.
const (
	// LogAPIKeyLastUsedUpdateFailed is logged when the fire-and-forget
	// last-used timestamp update for an API key fails.
	LogAPIKeyLastUsedUpdateFailed = "failed to update API key last used" //nolint:gosec // G101: log message, not a credential

	// LogSecurityEventWriteFailed is logged when a security event cannot be recorded.
	LogSecurityEventWriteFailed = "failed to record security event"

	// Organization admin handlers.

	LogCreateOrganizationFailed     = "create organization failed"
	LogGetOrganizationFailed        = "get organization failed"
	LogUpdateOrganizationFailed     = "update organization failed"
	LogDeleteOrganizationFailed     = "delete organization failed"
	LogListOrgAPIKeysForAuditFailed = "list org api keys for audit failed" //nolint:gosec // G101: log message, not a credential
	LogListOrganizationsFailed      = "list organizations failed"

	// Organization membership handlers.

	LogEmailLookupFailed                         = "email lookup failed"
	LogAddOrgUserFailed                          = "add org user failed"
	LogUpdateOrgMemberPermissionsFailed          = "update org member permissions failed"
	LogGetNewlyAddedMemberFailed                 = "get newly added member failed"
	LogGetOrgUserFailed                          = "get org user failed"
	LogUpdateOrgUserRoleFailed                   = "update org user role failed"
	LogGetCurrentMemberForPermissionUpdateFailed = "get current member for permission update failed"
	LogGetUpdatedMemberFailed                    = "get updated member failed"
	LogRemoveOrgUserFailed                       = "remove org user failed"
	LogListOrgUsersFailed                        = "list org users failed"
	LogListUserOrganizationsFailed               = "list user organizations failed"

	// API key admin handlers.

	LogResolveTenantForAPIKeyFailed = "resolve tenant for API key failed" //nolint:gosec // G101: log message about API key resolution, not a credential
	LogCreateAPIKeyFailed           = "create API key failed"             //nolint:gosec // G101: log message, not a credential
	LogGetAPIKeyFailed              = "get API key failed"                //nolint:gosec // G101: log message, not a credential
	LogDeleteAPIKeyFailed           = "delete API key failed"             //nolint:gosec // G101: log message, not a credential
	LogListAPIKeysFailed            = "list API keys failed"              //nolint:gosec // G101: log message, not a credential

	// Authentication and user handlers.

	LogLoginFailed              = "login failed"
	LogTokenRefreshFailed       = "token refresh failed"
	LogGetProfileFailed         = "get profile failed"
	LogGetAuthSettingsFailed    = "get auth settings failed"
	LogLogoutFailed             = "logout failed"
	LogChangePasswordFailed     = "change password failed"
	LogCreateUserFailed         = "create user failed"
	LogGetUserFailed            = "get user failed"
	LogUpdateUserFailed         = "update user failed"
	LogDeleteUserFailed         = "delete user failed"
	LogListUsersFailed          = "list users failed"
	LogUpdateUserPasswordFailed = "update user password failed"

	// External identity-provider exchange handlers.

	LogOIDCExchangeFailed       = "OIDC exchange failed"
	LogOAuth2ExchangeFailed     = "OAuth2 exchange failed"
	LogBuildLoginResponseFailed = "Failed to build login response"

	// Registration handler.

	LogRegistrationFailed              = "registration failed"
	LogBuildRegistrationResponseFailed = "failed to build registration response"

	// Internal org-resolution handlers.

	LogOrgResolutionFailed       = "org resolution failed"
	LogDefaultOrgLookupFailed    = "default org lookup failed"
	LogRoleResolutionFailed      = "role resolution failed"
	LogServerAdminCheckFailed    = "server admin check failed"
	LogRecordPlatformEventFailed = "failed to record platform event"
)

// Security event detail strings recorded with authentication failures; the
// invalid-credentials prefix is completed with the attempted email address.
const (
	detailInvalidCredentialsForPrefix = "invalid credentials for " //nolint:gosec // G101: security-event detail prefix, not a credential
	detailOIDCExchangeFailed          = "OIDC code exchange failed"
	detailNonAdminAttemptedAdminOp    = "non-admin user attempted admin operation"
	detailNonManagerAttemptedMemberOp = "caller without the tenant manager role attempted a membership operation"

	// Operation names attached to security events.
	opNameLogin                = "Login"
	opNameRequireAdmin         = "requireServerAdmin"
	opNameResolveOrgAccess     = "resolveOrgAccess"
	opNameExchangeOIDC         = "ExchangeOIDC"
	opNameExchangeOAuth2       = "ExchangeOAuth2"
	detailOAuth2ExchangeFailed = "OAuth2 code exchange failed"
)
