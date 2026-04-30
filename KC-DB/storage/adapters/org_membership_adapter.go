package adapters

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/kilocenter/KC-DB/storage/postgres"
)

// OrgMembershipStoreAdapter adapts postgres.OrganizationRepository to provide
// organization membership operations for authentication.
type OrgMembershipStoreAdapter struct {
	repo *postgres.OrganizationRepository
}

// NewOrgMembershipStoreAdapter creates a new adapter with the given database connection
func NewOrgMembershipStoreAdapter(db *sqlx.DB) *OrgMembershipStoreAdapter {
	repo := postgres.NewOrganizationRepository(db)
	return &OrgMembershipStoreAdapter{
		repo: repo.(*postgres.OrganizationRepository),
	}
}

// ListUserMemberships retrieves all organizations a user belongs to with org details
func (a *OrgMembershipStoreAdapter) ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error) {
	memberships, err := a.repo.ListUserMemberships(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("org membership adapter: list_user_memberships: %w", err)
	}
	return memberships, nil
}

// CheckUserMembership validates that a user belongs to an organization
func (a *OrgMembershipStoreAdapter) CheckUserMembership(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (bool, error) {
	isMember, err := a.repo.CheckUserMembership(ctx, orgID, userID)
	if err != nil {
		return false, fmt.Errorf("org membership adapter: check_user_membership: %w", err)
	}
	return isMember, nil
}
