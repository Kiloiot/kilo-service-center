package admin

import "errors"

// Admin service errors.

// User admin errors
var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUserEmailExists  = errors.New("user email already exists")
	ErrUserDeleteFailed = errors.New("user delete failed")
)

// Organization admin errors
var (
	ErrOrganizationNotFound     = errors.New("organization not found")
	ErrOrganizationNameRequired = errors.New("organization name is required")
	ErrTenantCreationFailed     = errors.New("failed to create tenant for organization")
)

// Membership errors
var (
	ErrMemberNotFound        = errors.New("organization member not found")
	ErrMemberAlreadyExists   = errors.New("user is already a member of this organization")
	ErrCannotRemoveLastOwner = errors.New("cannot remove the last owner of an organization")
)

// API key errors
var (
	ErrAPIKeyNotFound = errors.New("api key not found")
)

// Error-wrap operation labels. Each names the store or crypto operation whose
// failure is being wrapped, used as fmt.Errorf("%s: %w", errOp..., err).
const (
	errOpCheckEmail     = "check email"
	errOpGenerateSalt   = "generate salt"
	errOpCreateUser     = "create user"
	errOpGetUser        = "get user"
	errOpGetUserByEmail = "get user by email"
	errOpUpdateUser     = "update user"
	errOpListUsers      = "list users"
	errOpCountUsers     = "count users"
	errOpSetPassword    = "set password"
	errOpRevokeSessions = "revoke sessions"

	errOpValidateTenant           = "validate tenant"
	errOpCreateOrganization       = "create organization"
	errOpGetOrganization          = "get organization"
	errOpGetOrganizationForUpdate = "get organization for update"
	errOpGetUpdatedOrganization   = "get updated organization"
	errOpGetOrganizationForDelete = "get organization for delete"
	errOpGetOrganizationUnscoped  = "get organization unscoped"
	errOpUpdateOrganization       = "update organization"
	errOpDeleteOrganization       = "delete organization"
	errOpListOrganizations        = "list organizations"

	errOpCheckMembership       = "check membership"
	errOpAddMember             = "add member"
	errOpGetMembership         = "get membership"
	errOpUpdateRole            = "update role"
	errOpUpdatePermissions     = "update permissions"
	errOpCountOwners           = "count owners"
	errOpRemoveMember          = "remove member"
	errOpListMembers           = "list members"
	errOpListUserOrganizations = "list user organizations"

	errOpGenerateKey  = "generate key"
	errOpCreateAPIKey = "create api key"
	errOpGetAPIKey    = "get api key" //nolint:gosec // G101: operation label, not a credential
	errOpDeleteAPIKey = "delete api key"
	errOpListAPIKeys  = "list api keys" //nolint:gosec // G101: operation label, not a credential
	errOpCountAPIKeys = "count api keys"
)
