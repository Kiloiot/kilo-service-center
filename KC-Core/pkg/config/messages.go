// Package config provides configuration types and loading for KiloCenter modules.
package config

const (
	// ErrConfigReadFmt formats config file read errors.
	ErrConfigReadFmt = "error reading config file: %w"
	// ErrConfigUnmarshalFmt formats config unmarshal errors.
	ErrConfigUnmarshalFmt = "error unmarshaling config: %w"

	// ErrSCEUIInvalidFmt formats Service Center EUI parse failures with the offending source.
	ErrSCEUIInvalidFmt = "invalid Service Center EUI from %s: %w"

	// LogDeprecatedServiceCenterEUIEnv warns that the legacy SERVICE_CENTER_EUI variable supplied
	// the Service Center EUI. Emitted by the consumer because config loading has no logger.
	LogDeprecatedServiceCenterEUIEnv = "SERVICE_CENTER_EUI is deprecated; set KILOCENTER_PROTOCOL_SC_EUI or protocol.sc_eui instead"

	// ErrServerNameRequired indicates server_name is missing.
	ErrServerNameRequired = "server name is required"
	// ErrUnsupportedStorageTypeFmt formats unsupported storage type errors.
	ErrUnsupportedStorageTypeFmt = "unsupported storage type: %s"
	// ErrStorageHostRequired indicates storage host is missing.
	ErrStorageHostRequired = "storage host is required"
	// ErrInvalidStoragePortFmt formats storage port range errors.
	ErrInvalidStoragePortFmt = "invalid storage port: %d"
	// ErrInvalidAPIPortFmt formats API port range errors.
	ErrInvalidAPIPortFmt = "invalid API port: %d"
	// ErrInvalidGRPCPortFmt formats gRPC port range errors.
	ErrInvalidGRPCPortFmt = "invalid gRPC port: %d"
	// ErrInternalTrustWithGRPCWeb indicates internal_trust_enabled=true is incompatible with grpc.web.enabled=true.
	ErrInternalTrustWithGRPCWeb = "internal_trust_enabled=true requires grpc.web.enabled=false (gateway handles gRPC-web)"

	// Auth configuration validation errors (fatal — abort startup).

	// ErrLocalLoginRequiresAuth indicates local_login_enabled requires auth.enabled.
	ErrLocalLoginRequiresAuth = "local_login_enabled requires auth.enabled=true"
	// ErrLocalLoginHMACSecretRequired indicates hmac_secret is missing when local login enabled.
	ErrLocalLoginHMACSecretRequired = "auth.hmac_secret is required when local_login_enabled=true" //nolint:gosec // Error message, not actual credentials
	// ErrLocalLoginHMACSecretTooShortFmt formats hmac_secret minimum length errors.
	ErrLocalLoginHMACSecretTooShortFmt = "auth.hmac_secret must be at least %d bytes (got %d)" //nolint:gosec // Error message, not actual credentials
	// ErrLocalLoginJWKSMutuallyExclusive indicates local_login and jwks_endpoint are mutually exclusive.
	ErrLocalLoginJWKSMutuallyExclusive = "local_login_enabled and jwks_endpoint are mutually exclusive"
	// ErrRefreshTokenRequiresLocalLogin indicates refresh_token_enabled requires local_login_enabled.
	ErrRefreshTokenRequiresLocalLogin = "refresh_token_enabled requires local_login_enabled=true"
	// ErrRefreshTokenTTLInvalid indicates refresh_token_ttl must exceed access_token_ttl.
	ErrRefreshTokenTTLInvalid = "refresh_token_ttl must be greater than access_token_ttl"

	// =========================================================================
	// External auth validation messages
	// =========================================================================

	// MsgAuthMustBeEnabled indicates external auth requires auth.enabled.
	MsgAuthMustBeEnabled = "auth.enabled must be true when OIDC or OAuth2 is enabled"
	// MsgHMACSecretRequired indicates HMAC secret is required for external providers (local JWTs still issued).
	MsgHMACSecretRequired = "auth.hmac_secret required for external providers (local JWTs still issued)" //nolint:gosec // Error message, not actual credentials

	// MsgOIDCClientIDRequired indicates client_id is missing when OIDC enabled.
	MsgOIDCClientIDRequired = "auth.oidc.client_id is required when OIDC enabled"
	// MsgOIDCClientSecretRequired indicates client_secret is missing when OIDC enabled.
	MsgOIDCClientSecretRequired = "auth.oidc.client_secret is required when OIDC enabled" //nolint:gosec // Error message, not actual credentials
	// MsgOIDCRedirectURLRequired indicates redirect_url is missing when OIDC enabled.
	MsgOIDCRedirectURLRequired = "auth.oidc.redirect_url is required when OIDC enabled" //nolint:gosec // Error message, not actual credentials
	// MsgOIDCProviderURLRequired indicates provider_url is missing when OIDC enabled.
	MsgOIDCProviderURLRequired = "auth.oidc.provider_url is required when OIDC enabled"

	// MsgOAuth2ClientIDRequired indicates client_id is missing when OAuth2 enabled.
	MsgOAuth2ClientIDRequired = "auth.oauth2.client_id is required when OAuth2 enabled"
	// MsgOAuth2ClientSecretRequired indicates client_secret is missing when OAuth2 enabled (unless public_client=true).
	MsgOAuth2ClientSecretRequired = "auth.oauth2.client_secret is required when OAuth2 enabled (unless public_client=true)" //nolint:gosec // Error message, not actual credentials
	// MsgOAuth2RedirectURLRequired indicates redirect_url is missing when OAuth2 enabled.
	MsgOAuth2RedirectURLRequired = "auth.oauth2.redirect_url is required when OAuth2 enabled" //nolint:gosec // Error message, not actual credentials
	// MsgOAuth2AuthorizeURLRequired indicates authorize_url is missing when OAuth2 enabled.
	MsgOAuth2AuthorizeURLRequired = "auth.oauth2.authorize_url is required when OAuth2 enabled"
	// MsgOAuth2TokenURLRequired indicates token_url is missing when OAuth2 enabled.
	MsgOAuth2TokenURLRequired = "auth.oauth2.token_url is required when OAuth2 enabled" //nolint:gosec // Error message, not actual credentials
	// MsgOAuth2UserInfoURLRequired indicates userinfo_url is missing when OAuth2 enabled.
	MsgOAuth2UserInfoURLRequired = "auth.oauth2.userinfo_url is required when OAuth2 enabled"

	// MsgStateTTLPositive indicates state_ttl must be positive.
	MsgStateTTLPositive = "state_ttl must be positive"
	// MsgNonceTTLPositive indicates nonce_ttl must be positive.
	MsgNonceTTLPositive = "nonce_ttl must be positive"

	// MsgRedisRequiredForExternal indicates Redis is required when OIDC or OAuth2 enabled.
	MsgRedisRequiredForExternal = "redis.host is required when OIDC or OAuth2 enabled"
	// MsgRedisPortInvalid indicates Redis port is out of valid range.
	MsgRedisPortInvalid = "redis.port must be between 1 and 65535"

	// MsgUICallbackURLRequired indicates ui_callback_url is required when external auth enabled.
	MsgUICallbackURLRequired = "auth.ui_callback_url is required when OIDC or OAuth2 is enabled"

	// MsgOIDCRegCallbackURLRequired indicates registration_callback_url is required when registration enabled.
	MsgOIDCRegCallbackURLRequired = "auth.oidc.registration_callback_url is required when registration_enabled=true"

	// MsgOAuth2RegCallbackURLRequired indicates registration_callback_url is required when registration enabled.
	MsgOAuth2RegCallbackURLRequired = "auth.oauth2.registration_callback_url is required when registration_enabled=true"

	// ErrRegistrationRequiresLocalLogin indicates registration_enabled requires local_login_enabled.
	ErrRegistrationRequiresLocalLogin = "registration_enabled requires local_login_enabled=true"

	// ErrRateLimitRequestsPerMinPositive indicates requests_per_min must be positive.
	ErrRateLimitRequestsPerMinPositive = "gateway.rate_limit.requests_per_min must be positive when rate limiting enabled"

	// ErrRateLimitBurstPositive indicates burst must be positive.
	ErrRateLimitBurstPositive = "gateway.rate_limit.burst must be positive when rate limiting enabled"

	// ErrRateLimitCleanupIntervalPositive indicates cleanup_interval must be positive.
	ErrRateLimitCleanupIntervalPositive = "gateway.rate_limit.cleanup_interval must be positive when rate limiting enabled"

	// CE edition incompatibility errors (fatal — abort startup).

	// ErrCEOrgEnforcementIncompatible indicates CE cannot use org enforcement.
	ErrCEOrgEnforcementIncompatible = "edition=ce is incompatible with org_enforcement_enabled=true"
	// ErrCEStrictOrgResolutionIncompatible indicates CE cannot use strict org resolution.
	ErrCEStrictOrgResolutionIncompatible = "edition=ce is incompatible with protocol.strict_org_resolution=true"
	// ErrCECertTenantMappingIncompatible indicates CE cannot use cert-based tenant mapping.
	ErrCECertTenantMappingIncompatible = "edition=ce is incompatible with protocol.scaci_cert_tenant_mapping=true"
	// ErrCEExternalOrgClaimIncompatible indicates CE cannot use external org claims.
	ErrCEExternalOrgClaimIncompatible = "edition=ce is incompatible with oidc.external_org_claim"
	// ErrCETenantIDRequired indicates CE requires a configured tenant ID.
	ErrCETenantIDRequired = "edition=ce requires general.tenant_id > 0"
)
