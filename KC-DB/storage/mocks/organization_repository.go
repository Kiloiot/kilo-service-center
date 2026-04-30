package mocks

import (
	"context"

	"github.com/google/uuid"

	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
)

// MockOrganizationRepository implements interfaces.OrganizationRepository for testing.
// Nil function pointers return ErrMockNotConfigured to fail fast.
type MockOrganizationRepository struct {
	GetTenantByOrgIDFn    func(ctx context.Context, orgID uuid.UUID) (int64, error)
	GetOrgByTenantIDFn    func(ctx context.Context, tenantID int64) (*models.Organization, error)
	UpsertOrgFn           func(ctx context.Context, org *models.Organization) error
	CreateFn              func(ctx context.Context, org *models.Organization) error
	GetByIDFn             func(ctx context.Context, orgID uuid.UUID, tenantID int64) (*models.Organization, error)
	GetByIDUnscopedFn     func(ctx context.Context, orgID uuid.UUID) (*models.Organization, error)
	UpdateFn              func(ctx context.Context, orgID uuid.UUID, tenantID int64, updates map[string]interface{}) error
	DeleteFn              func(ctx context.Context, orgID uuid.UUID, tenantID int64) error
	ListOrgMembersFn      func(ctx context.Context, orgID uuid.UUID, status string) ([]*models.OrganizationMember, error)
	CheckUserMembershipFn func(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (bool, error)
	AddMemberFn           func(ctx context.Context, member *models.OrganizationMember) error
	RemoveMemberFn        func(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) error
	UpdateMemberRoleFn    func(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, role string) error
	ListUserMembershipsFn func(ctx context.Context, userID uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error)
	GetOrgByExternalIDFn  func(ctx context.Context, externalID string) (*models.Organization, error)

	// Organization Admin Methods
	ListOrganizationsFn           func(ctx context.Context, tenantID *int64, limit, offset int) ([]*models.Organization, int64, error)
	ListOrgMembersWithEmailFn     func(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*models.OrganizationMemberWithEmail, int64, error)
	GetOrgMemberWithEmailFn       func(ctx context.Context, orgID, userID uuid.UUID) (*models.OrganizationMemberWithEmail, error)
	CountOrgMembersFn             func(ctx context.Context, orgID uuid.UUID, status string) (int64, error)
	CheckBaseStationQuotaFn       func(ctx context.Context, orgID uuid.UUID) error
	CheckEndpointQuotaFn          func(ctx context.Context, orgID uuid.UUID) error
	UpdateMemberPermissionsFn     func(ctx context.Context, orgID, userID uuid.UUID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin bool) error
	CountOrgMembersByRoleFn       func(ctx context.Context, orgID uuid.UUID, role string) (int64, error)
	ListUserMembershipsByTenantFn func(ctx context.Context, userID uuid.UUID, tenantID int64) ([]*models.OrganizationMembershipWithOrg, error)
}

// Ensure MockOrganizationRepository implements interfaces.OrganizationRepository.
var _ interfaces.OrganizationRepository = (*MockOrganizationRepository)(nil)

// GetTenantByOrgID resolves organization UUID to numeric tenant ID.
func (m *MockOrganizationRepository) GetTenantByOrgID(ctx context.Context, orgID uuid.UUID) (int64, error) {
	if m.GetTenantByOrgIDFn == nil {
		return 0, ErrMockNotConfigured
	}
	return m.GetTenantByOrgIDFn(ctx, orgID)
}

// GetOrgByTenantID returns the default organization for a tenant.
func (m *MockOrganizationRepository) GetOrgByTenantID(ctx context.Context, tenantID int64) (*models.Organization, error) {
	if m.GetOrgByTenantIDFn == nil {
		return nil, ErrMockNotConfigured
	}
	return m.GetOrgByTenantIDFn(ctx, tenantID)
}

// UpsertOrg creates or updates organization.
func (m *MockOrganizationRepository) UpsertOrg(ctx context.Context, org *models.Organization) error {
	if m.UpsertOrgFn == nil {
		return ErrMockNotConfigured
	}
	return m.UpsertOrgFn(ctx, org)
}

// Create creates a new organization.
func (m *MockOrganizationRepository) Create(ctx context.Context, org *models.Organization) error {
	if m.CreateFn == nil {
		return ErrMockNotConfigured
	}
	return m.CreateFn(ctx, org)
}

// GetByID retrieves an organization by UUID, scoped to a tenant.
func (m *MockOrganizationRepository) GetByID(ctx context.Context, orgID uuid.UUID, tenantID int64) (*models.Organization, error) {
	if m.GetByIDFn == nil {
		return nil, ErrMockNotConfigured
	}
	return m.GetByIDFn(ctx, orgID, tenantID)
}

// GetByIDUnscoped retrieves an organization by UUID without tenant scoping.
func (m *MockOrganizationRepository) GetByIDUnscoped(ctx context.Context, orgID uuid.UUID) (*models.Organization, error) {
	if m.GetByIDUnscopedFn != nil {
		return m.GetByIDUnscopedFn(ctx, orgID)
	}
	return nil, ErrMockNotConfigured
}

// Update updates an existing organization, scoped to a tenant.
func (m *MockOrganizationRepository) Update(ctx context.Context, orgID uuid.UUID, tenantID int64, updates map[string]interface{}) error {
	if m.UpdateFn == nil {
		return ErrMockNotConfigured
	}
	return m.UpdateFn(ctx, orgID, tenantID, updates)
}

// Delete soft-deletes an organization, scoped to a tenant.
func (m *MockOrganizationRepository) Delete(ctx context.Context, orgID uuid.UUID, tenantID int64) error {
	if m.DeleteFn == nil {
		return ErrMockNotConfigured
	}
	return m.DeleteFn(ctx, orgID, tenantID)
}

// ListOrgMembers retrieves all members of an organization.
func (m *MockOrganizationRepository) ListOrgMembers(ctx context.Context, orgID uuid.UUID, status string) ([]*models.OrganizationMember, error) {
	if m.ListOrgMembersFn == nil {
		return nil, ErrMockNotConfigured
	}
	return m.ListOrgMembersFn(ctx, orgID, status)
}

// CheckUserMembership validates that a user belongs to an organization.
func (m *MockOrganizationRepository) CheckUserMembership(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (bool, error) {
	if m.CheckUserMembershipFn == nil {
		return false, ErrMockNotConfigured
	}
	return m.CheckUserMembershipFn(ctx, orgID, userID)
}

// AddMember adds a user to an organization with specified role.
func (m *MockOrganizationRepository) AddMember(ctx context.Context, member *models.OrganizationMember) error {
	if m.AddMemberFn == nil {
		return ErrMockNotConfigured
	}
	return m.AddMemberFn(ctx, member)
}

// RemoveMember soft-removes a user from an organization.
func (m *MockOrganizationRepository) RemoveMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) error {
	if m.RemoveMemberFn == nil {
		return ErrMockNotConfigured
	}
	return m.RemoveMemberFn(ctx, orgID, userID)
}

// UpdateMemberRole changes a member's role.
func (m *MockOrganizationRepository) UpdateMemberRole(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, role string) error {
	if m.UpdateMemberRoleFn == nil {
		return ErrMockNotConfigured
	}
	return m.UpdateMemberRoleFn(ctx, orgID, userID, role)
}

// ListUserMemberships retrieves all organizations a user belongs to with org details.
func (m *MockOrganizationRepository) ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error) {
	if m.ListUserMembershipsFn == nil {
		return nil, ErrMockNotConfigured
	}
	return m.ListUserMembershipsFn(ctx, userID)
}

// GetOrgByExternalID retrieves an organization by external IdP identifier.
func (m *MockOrganizationRepository) GetOrgByExternalID(ctx context.Context, externalID string) (*models.Organization, error) {
	if m.GetOrgByExternalIDFn == nil {
		return nil, ErrMockNotConfigured
	}
	return m.GetOrgByExternalIDFn(ctx, externalID)
}

// ============================================================================
// Organization Admin Methods
// ============================================================================

// ListOrganizations retrieves organizations with optional tenant filter and pagination.
func (m *MockOrganizationRepository) ListOrganizations(ctx context.Context, tenantID *int64, limit, offset int) ([]*models.Organization, int64, error) {
	if m.ListOrganizationsFn == nil {
		return nil, 0, ErrMockNotConfigured
	}
	return m.ListOrganizationsFn(ctx, tenantID, limit, offset)
}

// ListOrgMembersWithEmail returns members with email from users table (JOIN query).
func (m *MockOrganizationRepository) ListOrgMembersWithEmail(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*models.OrganizationMemberWithEmail, int64, error) {
	if m.ListOrgMembersWithEmailFn == nil {
		return nil, 0, ErrMockNotConfigured
	}
	return m.ListOrgMembersWithEmailFn(ctx, orgID, status, limit, offset)
}

// GetOrgMemberWithEmail returns a single member with email.
func (m *MockOrganizationRepository) GetOrgMemberWithEmail(ctx context.Context, orgID, userID uuid.UUID) (*models.OrganizationMemberWithEmail, error) {
	if m.GetOrgMemberWithEmailFn == nil {
		return nil, ErrMockNotConfigured
	}
	return m.GetOrgMemberWithEmailFn(ctx, orgID, userID)
}

// CountOrgMembers returns total count of org members matching status filter.
func (m *MockOrganizationRepository) CountOrgMembers(ctx context.Context, orgID uuid.UUID, status string) (int64, error) {
	if m.CountOrgMembersFn == nil {
		return 0, ErrMockNotConfigured
	}
	return m.CountOrgMembersFn(ctx, orgID, status)
}

// CheckBaseStationQuota validates BS quota for an organization.
func (m *MockOrganizationRepository) CheckBaseStationQuota(ctx context.Context, orgID uuid.UUID) error {
	if m.CheckBaseStationQuotaFn == nil {
		return ErrMockNotConfigured
	}
	return m.CheckBaseStationQuotaFn(ctx, orgID)
}

// CheckEndpointQuota validates endpoint quota for an organization.
func (m *MockOrganizationRepository) CheckEndpointQuota(ctx context.Context, orgID uuid.UUID) error {
	if m.CheckEndpointQuotaFn == nil {
		return ErrMockNotConfigured
	}
	return m.CheckEndpointQuotaFn(ctx, orgID)
}

// UpdateMemberPermissions updates member permission flags (explicit booleans).
func (m *MockOrganizationRepository) UpdateMemberPermissions(ctx context.Context, orgID, userID uuid.UUID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin bool) error {
	if m.UpdateMemberPermissionsFn == nil {
		return ErrMockNotConfigured
	}
	return m.UpdateMemberPermissionsFn(ctx, orgID, userID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin)
}

// CountOrgMembersByRole returns the count of active members with the given role.
func (m *MockOrganizationRepository) CountOrgMembersByRole(ctx context.Context, orgID uuid.UUID, role string) (int64, error) {
	if m.CountOrgMembersByRoleFn == nil {
		return 0, ErrMockNotConfigured
	}
	return m.CountOrgMembersByRoleFn(ctx, orgID, role)
}

// ListUserMembershipsByTenant retrieves organizations a user belongs to within a specific tenant.
func (m *MockOrganizationRepository) ListUserMembershipsByTenant(ctx context.Context, userID uuid.UUID, tenantID int64) ([]*models.OrganizationMembershipWithOrg, error) {
	if m.ListUserMembershipsByTenantFn == nil {
		return nil, ErrMockNotConfigured
	}
	return m.ListUserMembershipsByTenantFn(ctx, userID, tenantID)
}
