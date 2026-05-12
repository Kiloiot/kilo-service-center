// Package config provides configuration types and loading for KiloCenter modules.
package config

import (
	"errors"
	"fmt"
)

// Validate validates the configuration.
func (c *Config) Validate() error {
	if err := c.validateGeneralAndStorage(); err != nil {
		return err
	}
	if err := c.validateGRPC(); err != nil {
		return err
	}
	if err := c.validateLocalAuth(); err != nil {
		return err
	}
	if err := c.validateExternalAuth(); err != nil {
		return err
	}
	if err := c.validateCEEdition(); err != nil {
		return err
	}
	if err := c.validateRateLimit(); err != nil {
		return err
	}
	return nil
}

// validateGeneralAndStorage checks server name and storage driver/host/port.
func (c *Config) validateGeneralAndStorage() error {
	if c.General.ServerName == "" {
		return errors.New(ErrServerNameRequired)
	}
	if c.Storage.Type != StorageTypePostgres {
		return fmt.Errorf(ErrUnsupportedStorageTypeFmt, c.Storage.Type)
	}
	if c.Storage.Host == "" {
		return errors.New(ErrStorageHostRequired)
	}
	if c.Storage.Port <= 0 || c.Storage.Port > 65535 {
		return fmt.Errorf(ErrInvalidStoragePortFmt, c.Storage.Port)
	}
	return nil
}

// validateGRPC checks gRPC port range and internal-trust safety.
func (c *Config) validateGRPC() error {
	if c.GRPC.Enabled {
		if c.GRPC.Port <= 0 || c.GRPC.Port > 65535 {
			return fmt.Errorf(ErrInvalidGRPCPortFmt, c.GRPC.Port)
		}
	}
	if c.GRPC.InternalTrustEnabled && c.GRPC.Web.Enabled {
		return errors.New(ErrInternalTrustWithGRPCWeb)
	}
	return nil
}

// validateLocalAuth enforces the six hard-fail rules for local login, registration,
// and refresh-token configuration.
func (c *Config) validateLocalAuth() error {
	// Rule 1: local_login_enabled requires auth.enabled
	if c.Auth.LocalLoginEnabled && !c.Auth.Enabled {
		return errors.New(ErrLocalLoginRequiresAuth)
	}

	// Rule 2: hmac_secret required when local login enabled
	if c.Auth.Enabled && c.Auth.LocalLoginEnabled && c.Auth.HMACSecret == "" {
		return errors.New(ErrLocalLoginHMACSecretRequired)
	}

	// Rule 3: hmac_secret minimum length
	if c.Auth.Enabled && c.Auth.LocalLoginEnabled && len(c.Auth.HMACSecret) < AuthHMACSecretMinLength {
		return fmt.Errorf(ErrLocalLoginHMACSecretTooShortFmt, AuthHMACSecretMinLength, len(c.Auth.HMACSecret))
	}

	// Rule 4: local_login_enabled and jwks_endpoint are mutually exclusive
	if c.Auth.Enabled && c.Auth.LocalLoginEnabled && c.Auth.JWKSEndpoint != "" {
		return errors.New(ErrLocalLoginJWKSMutuallyExclusive)
	}

	// registration_enabled requires local_login_enabled
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

	return nil
}

// validateExternalAuth validates OIDC/OAuth2 provider settings, shared HMAC
// requirements, and Redis/UI callback dependencies.
func (c *Config) validateExternalAuth() error {
	externalAuthEnabled := c.Auth.OIDC.Enabled || c.Auth.OAuth2.Enabled
	if !externalAuthEnabled {
		return nil
	}

	// Rule 7: External auth requires auth.enabled
	if !c.Auth.Enabled {
		return errors.New(MsgAuthMustBeEnabled)
	}

	// Rule 8: External auth requires HMAC secret (local JWTs issued after exchange)
	if c.Auth.HMACSecret == "" {
		return errors.New(MsgHMACSecretRequired)
	}

	// Rule 9: HMAC secret minimum length for external auth
	if len(c.Auth.HMACSecret) < AuthHMACSecretMinLength {
		return fmt.Errorf(ErrLocalLoginHMACSecretTooShortFmt, AuthHMACSecretMinLength, len(c.Auth.HMACSecret))
	}

	if err := c.validateOIDC(); err != nil {
		return err
	}
	if err := c.validateOAuth2(); err != nil {
		return err
	}

	// Redis required when any external auth enabled
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

	return nil
}

// validateOIDC checks OIDC provider fields when OIDC is enabled.
func (c *Config) validateOIDC() error {
	if !c.Auth.OIDC.Enabled {
		return nil
	}
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
	if c.Auth.OIDC.RegistrationEnabled && c.Auth.OIDC.RegistrationCallbackURL == "" {
		return errors.New(MsgOIDCRegCallbackURLRequired)
	}
	return nil
}

// validateOAuth2 checks OAuth2 provider fields when OAuth2 is enabled.
func (c *Config) validateOAuth2() error {
	if !c.Auth.OAuth2.Enabled {
		return nil
	}
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
	if c.Auth.OAuth2.RegistrationEnabled && c.Auth.OAuth2.RegistrationCallbackURL == "" {
		return errors.New(MsgOAuth2RegCallbackURLRequired)
	}
	return nil
}

// validateCEEdition enforces Community Edition incompatibility rules.
func (c *Config) validateCEEdition() error {
	if !IsCommunityEdition(c.General.Edition) {
		return nil
	}
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
	return nil
}

// validateRateLimit checks gateway rate-limit fields when rate limiting is enabled.
func (c *Config) validateRateLimit() error {
	if !c.Gateway.RateLimit.Enabled {
		return nil
	}
	if c.Gateway.RateLimit.RequestsPerMin <= 0 {
		return errors.New(ErrRateLimitRequestsPerMinPositive)
	}
	if c.Gateway.RateLimit.Burst <= 0 {
		return errors.New(ErrRateLimitBurstPositive)
	}
	if c.Gateway.RateLimit.CleanupInterval <= 0 {
		return errors.New(ErrRateLimitCleanupIntervalPositive)
	}
	return nil
}
