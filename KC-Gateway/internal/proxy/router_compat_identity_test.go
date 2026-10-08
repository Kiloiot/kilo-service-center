package proxy

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The compat identity table used to be hand-written; this pins the derived
// table to that list so a descriptor change is a deliberate diff here.
func TestCompatIdentityMethods_MatchesPinnedList(t *testing.T) {
	t.Parallel()
	pinned := []string{
		"/kilocenter.api.v1.KiloCenterService/Login",
		"/kilocenter.api.v1.KiloCenterService/RefreshTokens",
		"/kilocenter.api.v1.KiloCenterService/GetAuthSettings",
		"/kilocenter.api.v1.KiloCenterService/ExchangeOIDC",
		"/kilocenter.api.v1.KiloCenterService/ExchangeOAuth2",
		"/kilocenter.api.v1.KiloCenterService/GetProfile",
		"/kilocenter.api.v1.KiloCenterService/Logout",
		"/kilocenter.api.v1.KiloCenterService/ChangePassword",
		"/kilocenter.api.v1.KiloCenterService/CreateUser",
		"/kilocenter.api.v1.KiloCenterService/GetUser",
		"/kilocenter.api.v1.KiloCenterService/UpdateUser",
		"/kilocenter.api.v1.KiloCenterService/DeleteUser",
		"/kilocenter.api.v1.KiloCenterService/ListUsers",
		"/kilocenter.api.v1.KiloCenterService/UpdateUserPassword",
		"/kilocenter.api.v1.KiloCenterService/CreateOrganization",
		"/kilocenter.api.v1.KiloCenterService/GetOrganization",
		"/kilocenter.api.v1.KiloCenterService/UpdateOrganization",
		"/kilocenter.api.v1.KiloCenterService/DeleteOrganization",
		"/kilocenter.api.v1.KiloCenterService/ListOrganizations",
		"/kilocenter.api.v1.KiloCenterService/AddOrganizationUser",
		"/kilocenter.api.v1.KiloCenterService/GetOrganizationUser",
		"/kilocenter.api.v1.KiloCenterService/UpdateOrganizationUser",
		"/kilocenter.api.v1.KiloCenterService/RemoveOrganizationUser",
		"/kilocenter.api.v1.KiloCenterService/ListOrganizationUsers",
		"/kilocenter.api.v1.KiloCenterService/ListUserOrganizations",
		"/kilocenter.api.v1.KiloCenterService/CreateApiKey",
		"/kilocenter.api.v1.KiloCenterService/GetApiKey",
		"/kilocenter.api.v1.KiloCenterService/DeleteApiKey",
		"/kilocenter.api.v1.KiloCenterService/ListApiKeys",
		"/kilocenter.api.v1.KiloCenterService/RegisterAccount",
	}
	derived := make([]string, 0, len(compatIdentityMethods))
	for method := range compatIdentityMethods {
		derived = append(derived, method)
	}
	sort.Strings(derived)
	sort.Strings(pinned)
	assert.Equal(t, pinned, derived)
	for _, method := range pinned {
		assert.True(t, compatIdentityMethods[method], method)
	}
}
