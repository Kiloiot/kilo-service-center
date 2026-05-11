package admin

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// UserAdminStore provides user persistence operations for admin.
type UserAdminStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*models.User, error)
	Count(ctx context.Context) (int64, error)
	SetPasswordHash(ctx context.Context, userID uuid.UUID, passwordHash string) error
}

// OrganizationStore provides organization persistence operations.
type OrganizationStore interface {
	GetByID(ctx context.Context, id uuid.UUID, tenantID int64) (*models.Organization, error)
	GetByIDUnscoped(ctx context.Context, orgID uuid.UUID) (*models.Organization, error)
	Create(ctx context.Context, org *models.Organization) error
	Update(ctx context.Context, id uuid.UUID, tenantID int64, updates map[string]interface{}) error
	Delete(ctx context.Context, id uuid.UUID, tenantID int64) error
	List(ctx context.Context, tenantID *int64, limit, offset int) ([]*models.Organization, int64, error)
}

// TenantStore provides tenant persistence operations.
type TenantStore interface {
	GetTenant(ctx context.Context, id int64) (*models.Tenant, error)
	CreateTenant(ctx context.Context, name, description string) (*models.Tenant, error)
	DeleteTenant(ctx context.Context, id int64) error
}

// OrganizationMemberStore provides membership persistence operations.
type OrganizationMemberStore interface {
	AddMember(ctx context.Context, member *models.OrganizationMember) error
	GetMember(ctx context.Context, orgID, userID uuid.UUID) (*models.OrganizationMemberWithEmail, error)
	UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role string) error
	UpdateMemberPermissions(ctx context.Context, orgID, userID uuid.UUID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin bool) error
	RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error
	ListMembers(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*models.OrganizationMemberWithEmail, int64, error)
	CountMembersByRole(ctx context.Context, orgID uuid.UUID, role string) (int64, error)
	ListUserMembershipsByTenant(ctx context.Context, userID uuid.UUID, tenantID int64) ([]*models.OrganizationMembershipWithOrg, error)
}
