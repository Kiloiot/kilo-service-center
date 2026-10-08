package interceptors

// Log messages for the interceptor chain. Message text is part of the
// operational log contract and stays byte-identical to the previous inline
// literals.
const (
	LogGrpcHandlerPanicRecovered                              = "gRPC handler panic recovered"
	LogGrpcOrgInterceptorInvalidXOrganizationIDFormat         = "gRPC org interceptor: invalid x-organization-id format"
	LogGrpcOrgInterceptorInvalidXUserIDFormat                 = "gRPC org interceptor: invalid x-user-id format"
	LogGrpcOrgInterceptorMissingMetadata                      = "gRPC org interceptor: missing metadata"
	LogGrpcOrgInterceptorMissingXOrganizationIDHeader         = "gRPC org interceptor: missing x-organization-id header"
	LogGrpcOrgInterceptorMissingXUserIDHeader                 = "gRPC org interceptor: missing x-user-id header"
	LogGrpcOrgInterceptorMissingXUserIDHeaderForUserPrincipal = "gRPC org interceptor: missing x-user-id header for user principal"
	LogGrpcOrgInterceptorOrgMismatchBetweenAuthAnd            = "gRPC org interceptor: org mismatch between auth and header"
	LogGrpcOrgInterceptorOrgResolutionFailed                  = "gRPC org interceptor: org resolution failed"
	LogGrpcOrgInterceptorResolvedOrgContext                   = "gRPC org interceptor: resolved org context"
	LogGrpcOrgInterceptorTenantMismatchBetweenAuthAnd         = "gRPC org interceptor: tenant mismatch between auth and header"
	LogGrpcOrgInterceptorUserMismatchBetweenAuthAnd           = "gRPC org interceptor: user mismatch between auth and header"
	LogGrpcOrgInterceptorValidatedAuthIdentityAgainstHeaders  = "gRPC org interceptor: validated auth identity against headers"
	LogGrpcOrgInterceptorXUserIDHeaderNot                     = "gRPC org interceptor: x-user-id header not allowed for service-account principal"
	LogInternalTrustIdentityExtracted                         = "internal trust: identity extracted"
	LogInternalTrustInvalidOrgHeader                          = "internal trust: invalid org header"
	LogInternalTrustInvalidTenantHeader                       = "internal trust: invalid tenant header"
	LogInternalTrustInvalidUserHeader                         = "internal trust: invalid user header"
	LogInternalTrustInvalidServiceAccountHeader               = "internal trust: invalid service account header"
	LogInternalTrustConflictingPrincipalHeaders               = "internal trust: user and service account headers together"
	LogInternalTrustMissingOrgHeaderForNonExempt              = "internal trust: missing org header for non-exempt method"
	LogInternalTrustDefaultOrgResolverMissing                 = "internal trust: community mode has no default organization resolver"
	LogInternalTrustDefaultOrgResolutionFailed                = "internal trust: default organization resolution failed"
	LogInternalTrustMissingTenantHeader                       = "internal trust: missing tenant header"
	LogInternalTrustPeerRejected                              = "internal trust: peer secret missing or wrong"
	LogSecurityEventRecordFailed                              = "security event could not be recorded"
	LogAuthAPIKeyLastUsedUpdateFailed                         = "auth interceptor: API key last-used time could not be updated"
	LogAuthOrganizationResolutionFailed                       = "auth interceptor: organization for the token's tenant could not be resolved; continuing without organization context"
	LogGrpcOrgInterceptorAdminCheckFailed                     = "gRPC org interceptor: server admin check failed; treating the caller as not an admin"
)

// errFmtJWKSFetchFailed reports a JWKS endpoint that could not be fetched at
// interceptor construction.
const errFmtJWKSFetchFailed = "failed to fetch JWKS from %s: %w"

// Security event detail strings recorded alongside auth, authorization, org
// resolution, and internal-trust failures.
const (
	detailInvalidAudience                     = "invalid audience"
	detailInvalidAuthorizationFormat          = "invalid authorization format"
	detailInvalidInternalOrgHeader            = "invalid internal org header"
	detailInvalidInternalTenantHeader         = "invalid internal tenant header"
	detailInvalidInternalUserHeader           = "invalid internal user header"
	detailInvalidInternalServiceAccountHeader = "invalid internal service account header"
	detailConflictingInternalPrincipalHeaders = "internal user and service account headers together"
	detailInternalPeerRejected                = "internal peer secret missing or wrong"
	detailInvalidOrgHeaderFormat              = "invalid x-organization-id format"
	detailInvalidTokenIssuer                  = "invalid token issuer"
	detailJWTValidationFailed                 = "JWT validation failed"
	detailUnrecognizedTokenFormat             = "unrecognized token format"
	detailAPIKeyNotFound                      = "API key not found" //nolint:gosec // G101: rejection detail, not a credential
	detailAPIKeyInactive                      = "API key inactive"  //nolint:gosec // G101: rejection detail, not a credential
	detailAPIKeyExpired                       = "API key expired"   //nolint:gosec // G101: rejection detail, not a credential
	detailNoSigningKeyConfigured              = "no signing key configured"
	detailTenantResolverNotConfigured         = "tenant resolver not configured"
	detailTenantClaimResolvedToZero           = "tenant claim resolved to zero"
	detailInsufficientRole                    = "insufficient role"
	detailMissingAudienceClaim                = "missing audience claim"
	detailMissingAuthorizationHeader          = "missing authorization header"
	detailMissingGRPCMetadata                 = "missing gRPC metadata"
	detailMissingInternalOrgHeader            = "missing internal org header"
	detailMissingInternalTenantHeader         = "missing internal tenant header"
	detailMissingOrgHeader                    = "missing x-organization-id header"
	detailMissingTenantClaim                  = "missing tenant claim"
	detailOrgResolutionFailed                 = "org resolution failed"
	detailOrgUUIDResolutionFailed             = "org UUID resolution failed"
	detailRoleResolutionFailed                = "role resolution failed"
	detailUnknownMethod                       = "method has no role requirement"
)
