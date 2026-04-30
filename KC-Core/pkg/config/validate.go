// Package config provides configuration types and loading for KiloCenter modules.
package config

import (
	"errors"
	"fmt"
)

// Validate validates the configuration.
func (c *Config) Validate() error {
	// Validate general config
	if c.General.ServerName == "" {
		return errors.New(ErrServerNameRequired)
	}

	// Validate storage config
	if c.Storage.Type != StorageTypePostgres {
		return fmt.Errorf(ErrUnsupportedStorageTypeFmt, c.Storage.Type)
	}

	if c.Storage.Host == "" {
		return errors.New(ErrStorageHostRequired)
	}

	if c.Storage.Port <= 0 || c.Storage.Port > 65535 {
		return fmt.Errorf(ErrInvalidStoragePortFmt, c.Storage.Port)
	}

	// Validate gRPC config if enabled
	if c.GRPC.Enabled {
		if c.GRPC.Port <= 0 || c.GRPC.Port > 65535 {
			return fmt.Errorf(ErrInvalidGRPCPortFmt, c.GRPC.Port)
		}
	}

	// Internal trust mode safety checks
	if c.GRPC.InternalTrustEnabled && c.GRPC.Web.Enabled {
		return errors.New(ErrInternalTrustWithGRPCWeb)
	}

	// Auth configuration validation (SIX hard-fail rules)

	// Rule 1: local_login_enabled requires auth.enabled
	if c.Auth.LocalLoginEnabled && !c.Auth.Enabled {
		return errors.New(ErrLocalLoginRequiresAuth)
	}

	// Rule 2: hmac_secret required when local login enabled
	if c.Auth.Enabled && c.Auth.LocalLoginEnabled && c.Auth.HMACSecret == "" {
		return errors.New(ErrLocalLoginHMACSecretRequired)
	}

	// Rule 3: hmac_secret minimum length (use AuthHMACSecretMinLength constant)
	if c.Auth.Enabled && c.Auth.LocalLoginEnabled && len(c.Auth.HMACSecret) < AuthHMACSecretMinLength {
		return fmt.Errorf(ErrLocalLoginHMACSecretTooShortFmt, AuthHMACSecretMinLength, len(c.Auth.HMACSecret))
	}

	// Rule 4: local_login_enabled and jwks_endpoint are mutually exclusive
	if c.Auth.Enabled && c.Auth.LocalLoginEnabled && c.Auth.JWKSEndpoint != "" {
		return errors.New(ErrLocalLoginJWKSMutuallyExclusive)
	}

	// Rule: registration_enabled requires local_login_enabled
	if c.Auth.RegistrationEnabled && !c.Auth.LocalLoginEnabled {
		return errors.New(ErrRegistrationRequiresLocalLogin)
	}

	// Rule 5: refresh_token_enabled requires local_login_enabled
	if c.Auth.RefreshTokenEnabled && !c.Auth.LocalLoginEnabled {
		return errors.New(ErrRefreshTokenRequiresLocalLogin)
	}

	// Rule 6: refresh_token_ttl must exceed access_token_ttl when refresh enabled
	if c.Auth.RefreshTokenEnabled && c.Auth.RefreshTokenTTL <= c.Auth.AccessTokenTTL {
		return errors.New(ErrRefreshTokenTTLInvalid)
	}

	// =========================================================================
	// External auth validation rules
	// =========================================================================

	// Check if any external auth is enabled
	externalAuthEnabled := c.Auth.OIDC.Enabled || c.Auth.OAuth2.Enabled

	// Rule 7: External auth requires auth.enabled
	if externalAuthEnabled && !c.Auth.Enabled {
		return errors.New(MsgAuthMustBeEnabled)
	}

	// Rule 8: External auth requires HMAC secret (we still issue local JWTs after exchange)
	if externalAuthEnabled && c.Auth.HMACSecret == "" {
		return errors.New(MsgHMACSecretRequired)
	}

	// Rule 9: HMAC secret minimum length for external auth
	if externalAuthEnabled && len(c.Auth.HMACSecret) < AuthHMACSecretMinLength {
		return fmt.Errorf(ErrLocalLoginHMACSecretTooShortFmt, AuthHMACSecretMinLength, len(c.Auth.HMACSecret))
	}

	// OIDC validation (when OIDC enabled)
	if c.Auth.OIDC.Enabled {
		if c.Auth.OIDC.ProviderURL == "" {
			return errors.New(MsgOIDCProviderURLRequired)
		}
		if c.Auth.OIDC.ClientID == "" {
			return errors.New(MsgOIDCClientIDRequired)
		}
		if c.Auth.OIDC.ClientSecret == "" {
			return errors.New(MsgOIDCClientSecretRequired)
		}
		if c.Auth.OIDC.RedirectURL == "" {
			return errors.New(MsgOIDCRedirectURLRequired)
		}
		if c.Auth.OIDC.StateTTL <= 0 {
			return errors.New(MsgStateTTLPositive)
		}
		if c.Auth.OIDC.NonceTTL <= 0 {
			return errors.New(MsgNonceTTLPositive)
		}
		// registration_callback_url required when registration_enabled
		if c.Auth.OIDC.RegistrationEnabled && c.Auth.OIDC.RegistrationCallbackURL == "" {
			return errors.New(MsgOIDCRegCallbackURLRequired)
		}
	}

	// OAuth2 validation (when OAuth2 enabled)
	if c.Auth.OAuth2.Enabled {
		if c.Auth.OAuth2.AuthorizeURL == "" {
			return errors.New(MsgOAuth2AuthorizeURLRequired)
		}
		if c.Auth.OAuth2.TokenURL == "" {
			return errors.New(MsgOAuth2TokenURLRequired)
		}
		if c.Auth.OAuth2.UserInfoURL == "" {
			return errors.New(MsgOAuth2UserInfoURLRequired)
		}
		if c.Auth.OAuth2.ClientID == "" {
			return errors.New(MsgOAuth2ClientIDRequired)
		}
		// client_secret required unless public_client=true (PKCE-only flow)
		if !c.Auth.OAuth2.PublicClient && c.Auth.OAuth2.ClientSecret == "" {
			return errors.New(MsgOAuth2ClientSecretRequired)
		}
		if c.Auth.OAuth2.RedirectURL == "" {
			return errors.New(MsgOAuth2RedirectURLRequired)
		}
		if c.Auth.OAuth2.StateTTL <= 0 {
			return errors.New(MsgStateTTLPositive)
		}
		// registration_callback_url required when registration_enabled
		if c.Auth.OAuth2.RegistrationEnabled && c.Auth.OAuth2.RegistrationCallbackURL == "" {
			return errors.New(MsgOAuth2RegCallbackURLRequired)
		}
	}

	// Redis validation (required when any external auth enabled)
	if externalAuthEnabled {
		if c.Redis.Host == "" {
			return errors.New(MsgRedisRequiredForExternal)
		}
		if c.Redis.Port <= 0 || c.Redis.Port > 65535 {
			return errors.New(MsgRedisPortInvalid)
		}
		// ui_callback_url required for external auth redirects
		if c.Auth.UICallbackURL == "" {
			return errors.New(MsgUICallbackURLRequired)
		}
	}

	// CE edition incompatibility rules
	if IsCommunityEdition(c.General.Edition) {
		if c.General.OrgEnforcementEnabled {
			return errors.New(ErrCEOrgEnforcementIncompatible)
		}
		if c.Protocol.StrictOrgResolution {
			return errors.New(ErrCEStrictOrgResolutionIncompatible)
		}
		if c.Protocol.SCACICertTenantMapping {
			return errors.New(ErrCECertTenantMappingIncompatible)
		}
		if c.Auth.OIDC.ExternalOrgClaim != "" {
			return errors.New(ErrCEExternalOrgClaimIncompatible)
		}
		if c.General.TenantID <= 0 {
			return errors.New(ErrCETenantIDRequired)
		}
	}

	// Gateway rate limit validation
	if c.Gateway.RateLimit.Enabled {
		if c.Gateway.RateLimit.RequestsPerMin <= 0 {
			return errors.New(ErrRateLimitRequestsPerMinPositive)
		}
		if c.Gateway.RateLimit.Burst <= 0 {
			return errors.New(ErrRateLimitBurstPositive)
		}
		if c.Gateway.RateLimit.CleanupInterval <= 0 {
			return errors.New(ErrRateLimitCleanupIntervalPositive)
		}
	}

	return nil
}
