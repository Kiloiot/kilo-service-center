package proxy

import (
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const identityServicePrefix = "/kilocenter.api.v1.IdentityService/"
const identityInternalServicePrefix = "/kilocenter.api.v1.IdentityInternalService/"

// compatIdentityMethods maps KiloCenterService compat methods that belong to identity.
var compatIdentityMethods = map[string]bool{
	"/kilocenter.api.v1.KiloCenterService/Login":                  true,
	"/kilocenter.api.v1.KiloCenterService/RefreshTokens":          true,
	"/kilocenter.api.v1.KiloCenterService/GetAuthSettings":        true,
	"/kilocenter.api.v1.KiloCenterService/ExchangeOIDC":           true,
	"/kilocenter.api.v1.KiloCenterService/ExchangeOAuth2":         true,
	"/kilocenter.api.v1.KiloCenterService/GetProfile":             true,
	"/kilocenter.api.v1.KiloCenterService/Logout":                 true,
	"/kilocenter.api.v1.KiloCenterService/ChangePassword":         true,
	"/kilocenter.api.v1.KiloCenterService/CreateUser":             true,
	"/kilocenter.api.v1.KiloCenterService/GetUser":                true,
	"/kilocenter.api.v1.KiloCenterService/UpdateUser":             true,
	"/kilocenter.api.v1.KiloCenterService/DeleteUser":             true,
	"/kilocenter.api.v1.KiloCenterService/ListUsers":              true,
	"/kilocenter.api.v1.KiloCenterService/UpdateUserPassword":     true,
	"/kilocenter.api.v1.KiloCenterService/CreateOrganization":     true,
	"/kilocenter.api.v1.KiloCenterService/GetOrganization":        true,
	"/kilocenter.api.v1.KiloCenterService/UpdateOrganization":     true,
	"/kilocenter.api.v1.KiloCenterService/DeleteOrganization":     true,
	"/kilocenter.api.v1.KiloCenterService/ListOrganizations":      true,
	"/kilocenter.api.v1.KiloCenterService/AddOrganizationUser":    true,
	"/kilocenter.api.v1.KiloCenterService/GetOrganizationUser":    true,
	"/kilocenter.api.v1.KiloCenterService/UpdateOrganizationUser": true,
	"/kilocenter.api.v1.KiloCenterService/RemoveOrganizationUser": true,
	"/kilocenter.api.v1.KiloCenterService/ListOrganizationUsers":  true,
	"/kilocenter.api.v1.KiloCenterService/ListUserOrganizations":  true,
	"/kilocenter.api.v1.KiloCenterService/CreateApiKey":           true,
	"/kilocenter.api.v1.KiloCenterService/GetApiKey":              true,
	"/kilocenter.api.v1.KiloCenterService/DeleteApiKey":           true,
	"/kilocenter.api.v1.KiloCenterService/ListApiKeys":            true,
	"/kilocenter.api.v1.KiloCenterService/RegisterAccount":        true,
}

// IsCompatIdentityMethod returns true if the method belongs to the
// KiloCenterService compat map and should route to KC-Identity.
func IsCompatIdentityMethod(fullMethod string) bool {
	return compatIdentityMethods[fullMethod]
}

// SelectUpstream routes external RPCs to the correct backend.
// IdentityInternalService methods are explicitly denied (internal-only).
func SelectUpstream(fullMethod string, core, identity grpc.ClientConnInterface) (grpc.ClientConnInterface, error) {
	// Hard deny: internal-only service must never be externally reachable
	if strings.HasPrefix(fullMethod, identityInternalServicePrefix) {
		return nil, status.Error(codes.Unimplemented, "unknown service")
	}

	// Route IdentityService RPCs to KC-Identity
	if strings.HasPrefix(fullMethod, identityServicePrefix) {
		return identity, nil
	}

	// Route compat identity methods to KC-Identity
	if compatIdentityMethods[fullMethod] {
		return identity, nil
	}

	// Everything else goes to KC-Core
	return core, nil
}
