// Package admin log message constants.
// Each Log* identifier names a structured log message produced by the admin
// services. Keeping the strings here lets the verify-constants gate enforce
// zero hardcoded log literals in this package.
package admin

// User admin log messages.
const (
	// LogUserEmailCheckFailed is logged when the duplicate-email lookup fails
	// with an error other than "not found".
	LogUserEmailCheckFailed = "failed to check email existence"

	// LogUserSaltGenerationFailed is logged when random salt generation for
	// password hashing fails.
	LogUserSaltGenerationFailed = "failed to generate salt"

	// LogUserCreateFailed is logged when the user store rejects a create.
	LogUserCreateFailed = "failed to create user"

	// LogUserCreated is logged after a user record is persisted.
	LogUserCreated = "user created"

	// LogUserGetFailed is logged when a user lookup fails with a store error.
	LogUserGetFailed = "failed to get user"

	// LogUserGetForUpdateFailed is logged when the pre-update user lookup fails.
	LogUserGetForUpdateFailed = "failed to get user for update"

	// LogUserUpdateFailed is logged when the user store rejects an update.
	LogUserUpdateFailed = "failed to update user"

	// LogUserUpdated is logged after a user record is updated.
	LogUserUpdated = "user updated"

	// LogUserGetForDeleteFailed is logged when the pre-delete user lookup fails.
	LogUserGetForDeleteFailed = "failed to get user for delete"

	// LogUserDeleteFailed is logged when the user store rejects a delete.
	LogUserDeleteFailed = "failed to delete user"

	// LogUserDeleted is logged after a user record is removed.
	LogUserDeleted = "user deleted"

	// LogUserListFailed is logged when the paginated user listing fails.
	LogUserListFailed = "failed to list users"

	// LogUserCountFailed is logged when the user count query fails.
	LogUserCountFailed = "failed to count users"

	// LogUserGetForPasswordChangeFailed is logged when the user lookup before a
	// password change fails.
	LogUserGetForPasswordChangeFailed = "failed to get user for password change"

	// LogUserSetPasswordHashFailed is logged when persisting a new password
	// hash fails.
	LogUserSetPasswordHashFailed = "failed to set password hash"

	// LogUserSessionsRevokeFailed is logged when a password reset cannot end
	// the user's refresh-token sessions.
	LogUserSessionsRevokeFailed = "failed to revoke user sessions after password reset"

	// LogUserPasswordChanged is logged after a user's password is changed.
	LogUserPasswordChanged = "user password changed"
)

// Organization admin log messages.
const (
	// LogOrgCallerTenantNotFound is logged when the tenant supplied on an org
	// create request does not exist.
	LogOrgCallerTenantNotFound = "caller tenant not found"

	// LogOrgTenantCreateFailed is logged when creating a fresh tenant for a new
	// organization fails.
	LogOrgTenantCreateFailed = "failed to create tenant for organization"

	// LogOrgTenantRollback is logged before deleting a freshly created tenant
	// after the org create failed.
	LogOrgTenantRollback = "rolling back tenant creation after org create failure"

	// LogOrgTenantRollbackFailed is logged when the tenant rollback delete fails.
	LogOrgTenantRollbackFailed = "failed to rollback tenant"

	// LogOrgCreatedWithTenant is logged after an organization and its tenant
	// are persisted.
	LogOrgCreatedWithTenant = "created organization with tenant"

	// LogOrgGetFailed is logged when a tenant-scoped organization lookup fails.
	LogOrgGetFailed = "failed to get organization"

	// LogOrgUpdateFailed is logged when the organization store rejects an update.
	LogOrgUpdateFailed = "failed to update organization"

	// LogOrgDeleteFailed is logged when the organization store rejects a delete.
	LogOrgDeleteFailed = "failed to delete organization"

	// LogOrgDeleted is logged after an organization is removed.
	LogOrgDeleted = "deleted organization"

	// LogOrgListFailed is logged when the tenant-scoped organization listing fails.
	LogOrgListFailed = "failed to list organizations"

	// LogOrgGetUnscopedFailed is logged when the unscoped organization lookup fails.
	LogOrgGetUnscopedFailed = "failed to get organization unscoped"
)

// Membership admin log messages.
const (
	// LogMemberAddFailed is logged when the member store rejects an add.
	LogMemberAddFailed = "failed to add member"

	// LogMemberAdded is logged after a user is added to an organization.
	LogMemberAdded = "member added"

	// LogMembershipGetFailed is logged when a membership lookup fails with a
	// store error.
	LogMembershipGetFailed = "failed to get membership"

	// LogMemberRoleUpdateFailed is logged when the member role update fails.
	LogMemberRoleUpdateFailed = "failed to update member role"

	// LogMemberRoleUpdated is logged after a member's role is updated.
	LogMemberRoleUpdated = "member role updated"

	// LogMemberPermissionsUpdateFailed is logged when the member permission
	// flags update fails.
	LogMemberPermissionsUpdateFailed = "failed to update member permissions"

	// LogMemberPermissionsUpdated is logged after a member's permission flags
	// are updated.
	LogMemberPermissionsUpdated = "member permissions updated"

	// LogMemberOwnerCountFailed is logged when counting an organization's
	// owners for last-owner protection fails.
	LogMemberOwnerCountFailed = "failed to count owners"

	// LogMemberRemoveFailed is logged when the member store rejects a removal.
	LogMemberRemoveFailed = "failed to remove member"

	// LogMemberRemoved is logged after a user is removed from an organization.
	LogMemberRemoved = "member removed"

	// LogMemberListFailed is logged when the organization member listing fails.
	LogMemberListFailed = "failed to list members"

	// LogMemberUserOrgListFailed is logged when listing a user's organizations
	// within a tenant fails.
	LogMemberUserOrgListFailed = "failed to list user organizations"
)

// API key admin log messages.
const (
	// LogAPIKeyGenerateFailed is logged when random key material generation fails.
	LogAPIKeyGenerateFailed = "failed to generate key bytes"

	// LogAPIKeyCreateFailed is logged when the API key store rejects a create.
	LogAPIKeyCreateFailed = "failed to create api key"

	// LogAPIKeyCreated is logged after an API key is persisted.
	LogAPIKeyCreated = "api key created"

	// LogAPIKeyGetFailed is logged when an API key lookup fails with a store error.
	LogAPIKeyGetFailed = "failed to get api key"

	// LogAPIKeyGetByOrgFailed is logged when the org-scoped API key lookup fails.
	LogAPIKeyGetByOrgFailed = "failed to get api key by org"

	// LogAPIKeyDeleteFailed is logged when the API key store rejects a delete.
	LogAPIKeyDeleteFailed = "failed to delete api key"

	// LogAPIKeyDeleted is logged after an API key is removed.
	LogAPIKeyDeleted = "api key deleted"

	// LogAPIKeyListFailed is logged when the API key listing fails.
	LogAPIKeyListFailed = "failed to list api keys"

	// LogAPIKeyCountFailed is logged when the API key count query fails.
	LogAPIKeyCountFailed = "failed to count api keys"
)
