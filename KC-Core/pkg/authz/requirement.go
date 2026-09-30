package authz

// Requirement is the set of manager roles any one of which grants an
// operation; an administrator satisfies every requirement.
type Requirement uint8

const (
	grantTenantManager Requirement = 1 << iota
	grantBaseStationManager
	grantEndpointManager
)

// Requirements an operation can place on its caller.
const (
	AdminOnly          Requirement = 0
	TenantManager                  = grantTenantManager
	BaseStationManager             = grantBaseStationManager
	EndpointManager                = grantEndpointManager
	AnyManager                     = grantBaseStationManager | grantEndpointManager
	AnyRole                        = grantTenantManager | grantBaseStationManager | grantEndpointManager
)

// GrantedTo reports whether the roles satisfy the requirement.
func (q Requirement) GrantedTo(r Roles) bool {
	return r.Admin ||
		(q&grantTenantManager != 0 && r.TenantManager) ||
		(q&grantBaseStationManager != 0 && r.BaseStationManager) ||
		(q&grantEndpointManager != 0 && r.EndpointManager)
}
