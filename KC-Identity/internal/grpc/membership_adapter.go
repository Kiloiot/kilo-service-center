package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
)

// membershipRepoAdapter implements MembershipLookup using the organization repository.
type membershipRepoAdapter struct {
	orgRepo interfaces.OrganizationRepository
}

// NewMembershipLookup creates a MembershipLookup backed by the organization repository.
func NewMembershipLookup(orgRepo interfaces.OrganizationRepository) MembershipLookup {
	return &membershipRepoAdapter{orgRepo: orgRepo}
}

func (a *membershipRepoAdapter) GetMembership(ctx context.Context, orgID, userID string) (string, bool, error) {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return "", false, err
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return "", false, err
	}

	member, err := a.orgRepo.GetOrgMemberWithEmail(ctx, orgUUID, userUUID)
	if err != nil {
		return "", false, err
	}

	isActive := member.Status == models.OrganizationMemberStatusActive
	return member.Role, isActive, nil
}
