package adapters

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/kilocenter/KC-DB/storage/postgres"
)

// OrganizationAdminAdapter wraps OrganizationRepository for org admin operations.
// Service adapters wrap KC-DB adapters, not postgres repos directly.
type OrganizationAdminAdapter struct {
	repo interfaces.OrganizationRepository
}

// NewOrganizationAdminAdapter creates a new adapter with the given database connection
func NewOrganizationAdminAdapter(db *sqlx.DB) *OrganizationAdminAdapter {
	return &OrganizationAdminAdapter{
		repo: postgres.NewOrganizationRepository(db),
	}
}

// ============================================================================
// Core CRUD Operations (matching interfaces/organization_repository.go:28-41)
// ============================================================================

// Create delegates to repo.Create (interfaces/organization_repository.go:30)
func (a *OrganizationAdminAdapter) Create(ctx context.Context, org *models.Organization) error {
	if err := a.repo.Create(ctx, org); err != nil {
		return fmt.Errorf("org admin adapter: create: %w", err)
	}
	return nil
}

// GetByID delegates to repo.GetByID, scoped to a tenant for defense-in-depth.
func (a *OrganizationAdminAdapter) GetByID(ctx context.Context, id uuid.UUID, tenantID int64) (*models.Organization, error) {
	org, err := a.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("org admin adapter: get: %w", err)
	}
	return org, nil
}

// GetByIDUnscoped retrieves an organization by UUID without tenant scoping.
func (a *OrganizationAdminAdapter) GetByIDUnscoped(ctx context.Context, orgID uuid.UUID) (*models.Organization, error) {
	org, err := a.repo.GetByIDUnscoped(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("org admin adapter: get_unscoped: %w", err)
	}
	return org, nil
}

// Update delegates to repo.Update, scoped to a tenant for defense-in-depth.
func (a *OrganizationAdminAdapter) Update(ctx context.Context, orgID uuid.UUID, tenantID int64, updates map[string]interface{}) error {
	if err := a.repo.Update(ctx, orgID, tenantID, updates); err != nil {
		return fmt.Errorf("org admin adapter: update: %w", err)
	}
	return nil
}

// Delete delegates to repo.Delete, scoped to a tenant for defense-in-depth.
func (a *OrganizationAdminAdapter) Delete(ctx context.Context, id uuid.UUID, tenantID int64) error {
	if err := a.repo.Delete(ctx, id, tenantID); err != nil {
		return fmt.Errorf("org admin adapter: delete: %w", err)
	}
	return nil
}

// ============================================================================
// Organization Admin Methods
// ============================================================================

// List delegates to repo.ListOrganizations
func (a *OrganizationAdminAdapter) List(ctx context.Context, tenantID *int64, limit, offset int) ([]*models.Organization, int64, error) {
	orgs, total, err := a.repo.ListOrganizations(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("org admin adapter: list: %w", err)
	}
	return orgs, total, nil
}

// CheckBaseStationQuota validates BS quota for an organization
func (a *OrganizationAdminAdapter) CheckBaseStationQuota(ctx context.Context, orgID uuid.UUID) error {
	if err := a.repo.CheckBaseStationQuota(ctx, orgID); err != nil {
		return fmt.Errorf("org admin adapter: check_bs_quota: %w", err)
	}
	return nil
}

// CheckEndpointQuota validates endpoint quota for an organization
func (a *OrganizationAdminAdapter) CheckEndpointQuota(ctx context.Context, orgID uuid.UUID) error {
	if err := a.repo.CheckEndpointQuota(ctx, orgID); err != nil {
		return fmt.Errorf("org admin adapter: check_ep_quota: %w", err)
	}
	return nil
}

// ============================================================================
// Organization Member Operations
// ============================================================================

// ListOrgMembers retrieves all members of an organization
func (a *OrganizationAdminAdapter) ListOrgMembers(ctx context.Context, orgID uuid.UUID, status string) ([]*models.OrganizationMember, error) {
	members, err := a.repo.ListOrgMembers(ctx, orgID, status)
	if err != nil {
		return nil, fmt.Errorf("org admin adapter: list_members: %w", err)
	}
	return members, nil
}

// ListOrgMembersWithEmail returns members with email from users table (JOIN query)
func (a *OrganizationAdminAdapter) ListOrgMembersWithEmail(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*models.OrganizationMemberWithEmail, int64, error) {
	members, total, err := a.repo.ListOrgMembersWithEmail(ctx, orgID, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("org admin adapter: list_members_with_email: %w", err)
	}
	return members, total, nil
}

// GetOrgMemberWithEmail returns a single member with email
func (a *OrganizationAdminAdapter) GetOrgMemberWithEmail(ctx context.Context, orgID, userID uuid.UUID) (*models.OrganizationMemberWithEmail, error) {
	member, err := a.repo.GetOrgMemberWithEmail(ctx, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("org admin adapter: get_member_with_email: %w", err)
	}
	return member, nil
}

// CountOrgMembers returns total count of org members matching status filter
func (a *OrganizationAdminAdapter) CountOrgMembers(ctx context.Context, orgID uuid.UUID, status string) (int64, error) {
	count, err := a.repo.CountOrgMembers(ctx, orgID, status)
	if err != nil {
		return 0, fmt.Errorf("org admin adapter: count_members: %w", err)
	}
	return count, nil
}

// AddMember adds a user to an organization with specified role
func (a *OrganizationAdminAdapter) AddMember(ctx context.Context, member *models.OrganizationMember) error {
	if err := a.repo.AddMember(ctx, member); err != nil {
		return fmt.Errorf("org admin adapter: add_member: %w", err)
	}
	return nil
}

// RemoveMember soft-removes a user from an organization
func (a *OrganizationAdminAdapter) RemoveMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) error {
	if err := a.repo.RemoveMember(ctx, orgID, userID); err != nil {
		return fmt.Errorf("org admin adapter: remove_member: %w", err)
	}
	return nil
}

// UpdateMemberRole changes a member's role
func (a *OrganizationAdminAdapter) UpdateMemberRole(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, role string) error {
	if err := a.repo.UpdateMemberRole(ctx, orgID, userID, role); err != nil {
		return fmt.Errorf("org admin adapter: update_member_role: %w", err)
	}
	return nil
}

// UpdateMemberPermissions updates member permission flags (explicit booleans)
func (a *OrganizationAdminAdapter) UpdateMemberPermissions(ctx context.Context, orgID, userID uuid.UUID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin bool) error {
	if err := a.repo.UpdateMemberPermissions(ctx, orgID, userID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin); err != nil {
		return fmt.Errorf("org admin adapter: update_member_permissions: %w", err)
	}
	return nil
}
