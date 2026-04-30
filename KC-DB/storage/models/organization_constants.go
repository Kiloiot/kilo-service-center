package models

// Organization states - used for Organization.State field.
const (
	OrganizationStateActive    = "active"
	OrganizationStateSuspended = "suspended"
	OrganizationStateArchived  = "archived"
)

// Organization member roles - used for OrganizationMember.Role field.
const (
	OrganizationRoleOwner  = "owner"
	OrganizationRoleAdmin  = "admin"
	OrganizationRoleMember = "member"
)

// Organization member statuses - used for OrganizationMember.Status field.
const (
	OrganizationMemberStatusActive  = "active"
	OrganizationMemberStatusInvited = "invited"
	OrganizationMemberStatusRemoved = "removed"
)

// Organization member status filters - used for list/query inputs.
// These are API-facing filter values, not stored statuses.
const (
	OrganizationMemberStatusFilterAll      = "all"
	OrganizationMemberStatusFilterInactive = "inactive"
)

// Organization member field names - used for update map keys.
const (
	MemberFieldRole = "role"
)
