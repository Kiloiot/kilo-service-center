package auth

import (
	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
)

// newLoginProfile builds the profile a sign-in returns from the loaded user,
// their memberships and the organization the access token is scoped to.
func newLoginProfile(user *models.User, memberships []*models.OrganizationMembershipWithOrg, defaultOrgID uuid.UUID) *grpcservices.UserProfile {
	profile := &grpcservices.UserProfile{
		ID:           user.ID,
		Email:        user.Email,
		IsAdmin:      user.IsAdmin,
		HasPassword:  user.PasswordHash != nil && *user.PasswordHash != "",
		DefaultOrgID: &defaultOrgID,
		Memberships:  make([]grpcservices.OrganizationMembership, len(memberships)),
	}
	if user.FirstName != nil {
		profile.FirstName = *user.FirstName
	}
	if user.LastName != nil {
		profile.LastName = *user.LastName
	}
	for i, m := range memberships {
		profile.Memberships[i] = grpcservices.OrganizationMembership{
			OrgID:              m.OrgID,
			OrgName:            m.OrgName,
			Role:               m.Role,
			DisplayName:        m.OrgName,
			IsOrgAdmin:         m.IsOrgAdmin,
			IsBaseStationAdmin: m.IsBaseStationAdmin,
			IsEndpointAdmin:    m.IsEndpointAdmin,
		}
	}
	return profile
}
