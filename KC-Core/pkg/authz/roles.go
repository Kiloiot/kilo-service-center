// Package authz is the role model of KiloCenter: the roles a user holds in an
// organization, the requirements operations place on them and the event
// categories each role may read.
package authz

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Roles are the roles a user holds while acting in one organization.
type Roles struct {
	Admin              bool
	TenantManager      bool
	BaseStationManager bool
	EndpointManager    bool
}

// AllRoles is what an administrator holds.
var AllRoles = Roles{Admin: true, TenantManager: true, BaseStationManager: true, EndpointManager: true}

// Any reports whether at least one role is held.
func (r Roles) Any() bool {
	return r.Admin || r.TenantManager || r.BaseStationManager || r.EndpointManager
}

// Effective combines a user's own role flags with the flags of their
// membership in the organization. Administrators hold every role, inactive
// users hold none, and only an active membership grants anything.
func Effective(user *models.User, membership *models.OrganizationMember) Roles {
	if user == nil || !user.IsActive {
		return Roles{}
	}
	if user.IsAdmin {
		return AllRoles
	}
	roles := Roles{
		TenantManager:      user.IsTenantManager,
		BaseStationManager: user.IsBaseStationManager,
		EndpointManager:    user.IsEndpointManager,
	}
	if membership == nil || membership.Status != models.OrganizationMemberStatusActive {
		return roles
	}
	roles.TenantManager = roles.TenantManager || membership.IsOrgAdmin
	roles.BaseStationManager = roles.BaseStationManager || membership.IsBaseStationAdmin
	roles.EndpointManager = roles.EndpointManager || membership.IsEndpointAdmin
	return roles
}

// ServiceAccountRoles are what an organization's service-account key holds in that organization.
var ServiceAccountRoles = Roles{BaseStationManager: true, EndpointManager: true}

// ServiceAccount returns the roles of an API key acting in actingOrg: an
// active, unexpired service-account key manages the base stations and
// endpoints of its own organization and holds nothing anywhere else.
func ServiceAccount(key *models.APIKey, actingOrg uuid.UUID) Roles {
	if key == nil || !key.IsServiceAccount() || !key.IsActive || key.IsExpired() {
		return Roles{}
	}
	if actingOrg == uuid.Nil || key.OrgID != actingOrg {
		return Roles{}
	}
	return ServiceAccountRoles
}

type rolesKey struct{}

// WithRoles records the caller's roles on the request context.
func WithRoles(ctx context.Context, roles Roles) context.Context {
	return context.WithValue(ctx, rolesKey{}, roles)
}

// FromContext returns the caller's roles; a context without them holds none.
func FromContext(ctx context.Context) Roles {
	roles, _ := ctx.Value(rolesKey{}).(Roles)
	return roles
}
