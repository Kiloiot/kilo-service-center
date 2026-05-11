// Package adapters provides storage adapters bridging KC-DB to gRPC services.
package adapters

import (
	"context"
	"errors"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	adminservice "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/admin"
	"github.com/google/uuid"
)

// OrganizationMemberStoreAdapter wraps OrganizationRepository to implement admin.OrganizationMemberStore.
type OrganizationMemberStoreAdapter struct {
	repo interfaces.OrganizationRepository
}

// NewOrganizationMemberStoreAdapter creates a new adapter.
func NewOrganizationMemberStoreAdapter(repo interfaces.OrganizationRepository) *OrganizationMemberStoreAdapter {
	return &OrganizationMemberStoreAdapter{repo: repo}
}

// AddMember adds a user to an organization.
func (a *OrganizationMemberStoreAdapter) AddMember(ctx context.Context, member *models.OrganizationMember) error {
	return a.repo.AddMember(ctx, member)
}

// GetMember retrieves a member with email via JOIN query.
func (a *OrganizationMemberStoreAdapter) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*models.OrganizationMemberWithEmail, error) {
	member, err := a.repo.GetOrgMemberWithEmail(ctx, orgID, userID)
	if err != nil {
		return nil, translateNotFound(err)
	}
	return member, nil
}

// UpdateMemberRole updates a member's role.
func (a *OrganizationMemberStoreAdapter) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	return translateNotFound(a.repo.UpdateMemberRole(ctx, orgID, userID, role))
}

// UpdateMemberPermissions updates member permission flags.
func (a *OrganizationMemberStoreAdapter) UpdateMemberPermissions(ctx context.Context, orgID, userID uuid.UUID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin bool) error {
	return translateNotFound(a.repo.UpdateMemberPermissions(ctx, orgID, userID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin))
}

// RemoveMember removes a user from an organization.
func (a *OrganizationMemberStoreAdapter) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error {
	return translateNotFound(a.repo.RemoveMember(ctx, orgID, userID))
}

// ListMembers retrieves members with email via JOIN query and SQL-level pagination.
func (a *OrganizationMemberStoreAdapter) ListMembers(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*models.OrganizationMemberWithEmail, int64, error) {
	repoStatus := status
	if status == models.OrganizationMemberStatusFilterAll {
		repoStatus = ""
	}
	// "inactive" filter passes through to repo which generates `status != 'active'` SQL
	return a.repo.ListOrgMembersWithEmail(ctx, orgID, repoStatus, limit, offset)
}

// CountMembersByRole returns the count of active members with the given role.
func (a *OrganizationMemberStoreAdapter) CountMembersByRole(ctx context.Context, orgID uuid.UUID, role string) (int64, error) {
	return a.repo.CountOrgMembersByRole(ctx, orgID, role)
}

// ListUserMembershipsByTenant returns organizations a user belongs to within a specific tenant.
func (a *OrganizationMemberStoreAdapter) ListUserMembershipsByTenant(ctx context.Context, userID uuid.UUID, tenantID int64) ([]*models.OrganizationMembershipWithOrg, error) {
	return a.repo.ListUserMembershipsByTenant(ctx, userID, tenantID)
}

// translateNotFound converts storage.ErrNotFound to interfaces.ErrRecordNotFound
// so callers using errors.Is(err, interfaces.ErrRecordNotFound) see a consistent sentinel.
func translateNotFound(err error) error {
	if err != nil && errors.Is(err, storage.ErrNotFound) {
		return interfaces.ErrRecordNotFound
	}
	return err
}

// Ensure OrganizationMemberStoreAdapter implements admin.OrganizationMemberStore
var _ adminservice.OrganizationMemberStore = (*OrganizationMemberStoreAdapter)(nil)
