// Package grpc provides gRPC service constants and utilities for the KiloCenter API.
package grpc

// PublicMethods lists gRPC methods that skip authentication.
// This is the single source of truth — consumed by auth interceptor and gateway.
var PublicMethods = map[string]bool{
	"/grpc.health.v1.Health/Check":                                   true,
	"/grpc.health.v1.Health/Watch":                                   true,
	"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo": true,
	"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo":      true,
	"/kilocenter.api.v1.KiloCenterService/Login":                     true,
	"/kilocenter.api.v1.KiloCenterService/RefreshTokens":             true,
	"/kilocenter.api.v1.KiloCenterService/GetAuthSettings":           true,
	"/kilocenter.api.v1.KiloCenterService/ExchangeOIDC":              true,
	"/kilocenter.api.v1.KiloCenterService/ExchangeOAuth2":            true,
	"/kilocenter.api.v1.KiloCenterService/GetReleaseInfo":            true,
	"/kilocenter.api.v1.KiloCenterService/RegisterAccount":           true,
	"/kilocenter.api.v1.IdentityService/Login":                       true,
	"/kilocenter.api.v1.IdentityService/RegisterAccount":             true,
	"/kilocenter.api.v1.IdentityService/RefreshTokens":               true,
	"/kilocenter.api.v1.IdentityService/GetAuthSettings":             true,
	"/kilocenter.api.v1.IdentityService/ExchangeOIDC":                true,
	"/kilocenter.api.v1.IdentityService/ExchangeOAuth2":              true,
	"/kilocenter.api.v1.CoreService/GetReleaseInfo":                  true,
	// CE onboarding does not require authentication (the installation has no credentials yet)
	"/kilocenter.api.v1.KiloCenterService/GetCEStatus":          true,
	"/kilocenter.api.v1.KiloCenterService/CompleteCEOnboarding": true,
	"/kilocenter.api.v1.CoreService/GetCEStatus":                true,
	"/kilocenter.api.v1.CoreService/CompleteCEOnboarding":       true,
}

// OrgExemptMethods lists gRPC methods that skip the org resolver interceptor.
// Superset of PublicMethods — includes methods that need auth but not org context.
// Identity admin RPCs validate org access via explicit request fields (validateOrgAccess)
// instead of the org resolver interceptor, so they are exempt.
var OrgExemptMethods = map[string]bool{
	// All public methods are also org-exempt
	"/grpc.health.v1.Health/Check":                                   true,
	"/grpc.health.v1.Health/Watch":                                   true,
	"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo": true,
	"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo":      true,
	"/kilocenter.api.v1.KiloCenterService/Login":                     true,
	"/kilocenter.api.v1.KiloCenterService/RefreshTokens":             true,
	"/kilocenter.api.v1.KiloCenterService/GetAuthSettings":           true,
	"/kilocenter.api.v1.KiloCenterService/ExchangeOIDC":              true,
	"/kilocenter.api.v1.KiloCenterService/ExchangeOAuth2":            true,
	"/kilocenter.api.v1.KiloCenterService/GetReleaseInfo":            true,
	"/kilocenter.api.v1.KiloCenterService/RegisterAccount":           true,
	"/kilocenter.api.v1.KiloCenterService/GetSystemStatus":           true,
	"/kilocenter.api.v1.IdentityService/Login":                       true,
	"/kilocenter.api.v1.IdentityService/RegisterAccount":             true,
	"/kilocenter.api.v1.IdentityService/RefreshTokens":               true,
	"/kilocenter.api.v1.IdentityService/GetAuthSettings":             true,
	"/kilocenter.api.v1.IdentityService/ExchangeOIDC":                true,
	"/kilocenter.api.v1.IdentityService/ExchangeOAuth2":              true,
	"/kilocenter.api.v1.CoreService/GetReleaseInfo":                  true,
	"/kilocenter.api.v1.CoreService/GetSystemStatus":                 true,

	// Identity admin RPCs (users, orgs, memberships) — org context handled via request fields
	"/kilocenter.api.v1.IdentityService/CreateUser":               true,
	"/kilocenter.api.v1.IdentityService/GetUser":                  true,
	"/kilocenter.api.v1.IdentityService/UpdateUser":               true,
	"/kilocenter.api.v1.IdentityService/DeleteUser":               true,
	"/kilocenter.api.v1.IdentityService/ListUsers":                true,
	"/kilocenter.api.v1.IdentityService/UpdateUserPassword":       true,
	"/kilocenter.api.v1.IdentityService/GetProfile":               true,
	"/kilocenter.api.v1.IdentityService/Logout":                   true,
	"/kilocenter.api.v1.IdentityService/ChangePassword":           true,
	"/kilocenter.api.v1.IdentityService/CreateOrganization":       true,
	"/kilocenter.api.v1.IdentityService/GetOrganization":          true,
	"/kilocenter.api.v1.IdentityService/UpdateOrganization":       true,
	"/kilocenter.api.v1.IdentityService/DeleteOrganization":       true,
	"/kilocenter.api.v1.IdentityService/ListOrganizations":        true,
	"/kilocenter.api.v1.IdentityService/AddOrganizationUser":      true,
	"/kilocenter.api.v1.IdentityService/GetOrganizationUser":      true,
	"/kilocenter.api.v1.IdentityService/UpdateOrganizationUser":   true,
	"/kilocenter.api.v1.IdentityService/RemoveOrganizationUser":   true,
	"/kilocenter.api.v1.IdentityService/ListOrganizationUsers":    true,
	"/kilocenter.api.v1.IdentityService/ListUserOrganizations":    true,
	"/kilocenter.api.v1.KiloCenterService/CreateUser":             true,
	"/kilocenter.api.v1.KiloCenterService/GetUser":                true,
	"/kilocenter.api.v1.KiloCenterService/UpdateUser":             true,
	"/kilocenter.api.v1.KiloCenterService/DeleteUser":             true,
	"/kilocenter.api.v1.KiloCenterService/ListUsers":              true,
	"/kilocenter.api.v1.KiloCenterService/UpdateUserPassword":     true,
	"/kilocenter.api.v1.KiloCenterService/GetProfile":             true,
	"/kilocenter.api.v1.KiloCenterService/Logout":                 true,
	"/kilocenter.api.v1.KiloCenterService/ChangePassword":         true,
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

	// Global coverage API (authenticated, server admin only — no org context needed)
	"/kilocenter.api.v1.CoreService/ListAllBaseStationLocations":       true,
	"/kilocenter.api.v1.KiloCenterService/ListAllBaseStationLocations": true,

	// CE onboarding (unauthenticated; no org context)
	"/kilocenter.api.v1.CoreService/GetCEStatus":                true,
	"/kilocenter.api.v1.CoreService/CompleteCEOnboarding":       true,
	"/kilocenter.api.v1.KiloCenterService/GetCEStatus":          true,
	"/kilocenter.api.v1.KiloCenterService/CompleteCEOnboarding": true,
}

// IsPublicMethod checks if a gRPC method should skip authentication.
func IsPublicMethod(fullMethod string) bool {
	return PublicMethods[fullMethod]
}

// IsOrgExemptMethod checks if a gRPC method should skip org resolver.
func IsOrgExemptMethod(fullMethod string) bool {
	return OrgExemptMethods[fullMethod]
}
