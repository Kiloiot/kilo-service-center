package interfaces

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// OrgDirectoryRepository defines the CE-safe hot-path resolution methods.
// Used by community resolver and any caller that only needs org/tenant mapping.
type OrgDirectoryRepository interface {
	GetTenantByOrgID(ctx context.Context, orgID uuid.UUID) (int64, error)
	GetOrgByTenantID(ctx context.Context, tenantID int64) (*models.Organization, error)
	GetOrgByExternalID(ctx context.Context, externalID string) (*models.Organization, error)
}

// OrganizationRepository defines the interface for Organization data operations.
// Supports Kilo Cloud multi-tenant integration with org UUID → tenant ID resolution.
// Embeds OrgDirectoryRepository for backward compatibility.
type OrganizationRepository interface {
	OrgDirectoryRepository

	// UpsertOrg creates or updates organization (Kilo Cloud sync path)
	// Attempts update first; if not found, creates new organization
	// Used when syncing organizations from Kilo Cloud's master directory
	UpsertOrg(ctx context.Context, org *models.Organization) error

	// Create creates a new organization
	// Returns error if tenant_id foreign key constraint fails
	Create(ctx context.Context, org *models.Organization) error

	// GetByID retrieves an organization by UUID, scoped to a tenant for defense-in-depth.
	GetByID(ctx context.Context, orgID uuid.UUID, tenantID int64) (*models.Organization, error)

	// GetByIDUnscoped retrieves an organization by UUID without tenant scoping.
	// Used for server-admin cross-tenant operations.
	GetByIDUnscoped(ctx context.Context, orgID uuid.UUID) (*models.Organization, error)

	// Update updates an existing organization, scoped to a tenant for defense-in-depth.
	// Typically used for name changes or state transitions (active → suspended)
	Update(ctx context.Context, orgID uuid.UUID, tenantID int64, updates map[string]interface{}) error

	// Delete soft-deletes an organization, scoped to a tenant for defense-in-depth.
	// Hard delete via ON DELETE CASCADE when tenant is deleted
	Delete(ctx context.Context, orgID uuid.UUID, tenantID int64) error

	// ListOrgMembers retrieves all members of an organization
	// Filtered by status (e.g., only 'active' members)
	ListOrgMembers(ctx context.Context, orgID uuid.UUID, status string) ([]*models.OrganizationMember, error)

	// CheckUserMembership validates that a user belongs to an organization
	// Returns true if user has active membership, false otherwise
	CheckUserMembership(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (bool, error)

	// AddMember adds a user to an organization with specified role
	AddMember(ctx context.Context, member *models.OrganizationMember) error

	// RemoveMember soft-removes a user from an organization (sets status = 'removed')
	RemoveMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) error

	// UpdateMemberRole changes a member's role (e.g., member → admin)
	UpdateMemberRole(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, role string) error

	// ListUserMemberships retrieves all organizations a user belongs to with org details.
	// Returns memberships with org name, role, and status for auth profile.
	ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error)

	// ListUserMembershipsByTenant retrieves organizations a user belongs to within a specific tenant.
	// Filters by tenant_id to prevent cross-tenant data leak.
	ListUserMembershipsByTenant(ctx context.Context, userID uuid.UUID, tenantID int64) ([]*models.OrganizationMembershipWithOrg, error)

	// ============================================================================
	// Organization Admin Methods
	// ============================================================================

	// ListOrganizations retrieves organizations with optional tenant filter and pagination.
	// Returns organizations slice and total count for pagination.
	ListOrganizations(ctx context.Context, tenantID *int64, limit, offset int) ([]*models.Organization, int64, error)

	// ListOrgMembersWithEmail returns members with email from users table (JOIN query).
	// Used for organization user list views in admin UI.
	// Returns paginated results and total count matching the status filter.
	ListOrgMembersWithEmail(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*models.OrganizationMemberWithEmail, int64, error)

	// GetOrgMemberWithEmail returns a single member with email.
	// Used for organization user detail view in admin UI.
	GetOrgMemberWithEmail(ctx context.Context, orgID, userID uuid.UUID) (*models.OrganizationMemberWithEmail, error)

	// CountOrgMembers returns total count of org members matching status filter.
	// Required for list response pagination.
	CountOrgMembers(ctx context.Context, orgID uuid.UUID, status string) (int64, error)

	// CheckBaseStationQuota validates BS quota for an organization.
	// Returns error if quota exceeded or organization cannot have base stations.
	CheckBaseStationQuota(ctx context.Context, orgID uuid.UUID) error

	// CheckEndpointQuota validates endpoint quota for an organization.
	// Returns error if quota exceeded.
	CheckEndpointQuota(ctx context.Context, orgID uuid.UUID) error

	// UpdateMemberPermissions updates member permission flags (explicit booleans).
	// Permissions are independent of role - role is only for ownership semantics.
	UpdateMemberPermissions(ctx context.Context, orgID, userID uuid.UUID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin bool) error

	// CountOrgMembersByRole returns the count of active members with the given role.
	// Used for last-owner protection checks.
	CountOrgMembersByRole(ctx context.Context, orgID uuid.UUID, role string) (int64, error)
}
