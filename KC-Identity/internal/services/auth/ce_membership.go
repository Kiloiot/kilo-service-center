package auth

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
)

// CEDefaultOrgProvider synthesizes a membership for users in Community Edition.
// CE has a single hidden default org; users are assigned membership based on their
// server admin status rather than explicit org membership records.
type CEDefaultOrgProvider struct {
	orgDirectory    interfaces.OrganizationRepository
	defaultTenantID int64
}

// NewCEDefaultOrgProvider creates a CE membership synthesizer.
func NewCEDefaultOrgProvider(orgDirectory interfaces.OrganizationRepository, defaultTenantID int64) *CEDefaultOrgProvider {
	return &CEDefaultOrgProvider{
		orgDirectory:    orgDirectory,
		defaultTenantID: defaultTenantID,
	}
}

// SynthesizeMembership returns a single CE membership for the default org.
// Permissions are derived from user.IsAdmin.
func (p *CEDefaultOrgProvider) SynthesizeMembership(ctx context.Context, user *models.User) (*grpcservices.OrganizationMembership, error) {
	defaultOrg, err := p.orgDirectory.GetOrgByTenantID(ctx, p.defaultTenantID)
	if err != nil {
		return nil, fmt.Errorf("CE default org lookup failed for tenant %d: %w", p.defaultTenantID, err)
	}
	return &grpcservices.OrganizationMembership{
		OrgID:              defaultOrg.OrgID,
		OrgName:            defaultOrg.Name,
		Role:               models.OrganizationRoleMember,
		DisplayName:        defaultOrg.Name,
		IsOrgAdmin:         false,
		IsBaseStationAdmin: user.IsAdmin,
		IsEndpointAdmin:    user.IsAdmin,
	}, nil
}
