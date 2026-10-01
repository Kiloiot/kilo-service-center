package roaming

import "errors"

// errMissingDetectorDependency rejects a detector built without its resolver, recorder or clock.
var errMissingDetectorDependency = errors.New("roaming detector: missing dependency")

// Error format strings shared by this package's failure paths; verbs are filled at the point of failure.
const (
	errFmtNoRoamingAgreement                 = "no roaming agreement between tenants %d and %d"
	errFmtCheckPartnershipBetweenTenants     = "failed to check partnership between tenants: %w"
	errFmtRoamingNotEnabledForServingTenant  = "roaming not enabled for serving tenant %d"
	errFmtCheckRoamingStatusForServingTenant = "failed to check roaming status for serving tenant %d: %w"
	errFmtRoamingNotEnabledForOwnerTenant    = "roaming not enabled for owner tenant %d"
	errFmtCheckRoamingStatusForOwnerTenant   = "failed to check roaming status for owner tenant %d: %w"
	errFmtResolveOwnershipForEndpoint        = "failed to resolve ownership for endpoint %s: %w"
)
