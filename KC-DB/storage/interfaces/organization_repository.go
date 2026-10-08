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

// OrganizationAdminRepository manages organization records.
type OrganizationAdminRepository interface {
	Create(ctx context.Context, org *models.Organization) error
	GetByID(ctx context.Context, orgID uuid.UUID, tenantID int64) (*models.Organization, error)
	GetByIDUnscoped(ctx context.Context, orgID uuid.UUID) (*models.Organization, error)
	Update(ctx context.Context, orgID uuid.UUID, tenantID int64, updates map[string]interface{}) error
	Delete(ctx context.Context, orgID uuid.UUID, tenantID int64) error
	ListOrganizations(ctx context.Context, tenantID *int64, limit, offset int) ([]*models.Organization, int64, error)
}

// OrganizationMembershipRepository manages organization membership records.
type OrganizationMembershipRepository interface {
	AddMember(ctx context.Context, member *models.OrganizationMember) error
	RemoveMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) error
	UpdateMemberRole(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, role string) error
	ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error)
	ListUserMembershipsByTenant(ctx context.Context, userID uuid.UUID, tenantID int64) ([]*models.OrganizationMembershipWithOrg, error)
	ListOrgMembersWithEmail(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*models.OrganizationMemberWithEmail, int64, error)
	GetOrgMemberWithEmail(ctx context.Context, orgID, userID uuid.UUID) (*models.OrganizationMemberWithEmail, error)
	UpdateMemberPermissions(ctx context.Context, orgID, userID uuid.UUID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin bool) error
	CountOrgMembersByRole(ctx context.Context, orgID uuid.UUID, role string) (int64, error)
}

// OrganizationRepository is the full organization persistence surface; consumers
// depend on the narrower interfaces and only the storage layer implements the union.
type OrganizationRepository interface {
	OrgDirectoryRepository
	OrganizationAdminRepository
	OrganizationMembershipRepository
}
