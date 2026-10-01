// Package config provides configuration types and loading for KiloCenter modules.
package config

import (
	"errors"
	"fmt"
)

// Validate validates the configuration; each rule group reports the first
// violation it finds, in a fixed order.
func (c *Config) Validate() error {
	for _, validate := range []func() error{
		c.validateService,
		c.validateInternalTrust,
		c.validateLocalAuth,
		c.validateExternalAuth,
		c.validateCommunityEdition,
		c.validateGatewayRateLimit,
		c.validateCertificates,
	} {
		if err := validate(); err != nil {
			return err
		}
	}
	return nil
}

// validateService checks the server identity, storage and gRPC listener.
func (c *Config) validateService() error {
	if c.General.ServerName == "" {
		return errors.New(ErrServerNameRequired)
	}
	if c.General.TenantID <= 0 {
		return errors.New(ErrPlatformTenantIDRequired)
	}
	if c.Storage.Type != StorageTypePostgres {
		return fmt.Errorf(ErrUnsupportedStorageTypeFmt, c.Storage.Type)
	}
	if c.Storage.Host == "" {
		return errors.New(ErrStorageHostRequired)
	}
	if c.Storage.Port <= 0 || c.Storage.Port > MaxPortNumber {
		return fmt.Errorf(ErrInvalidStoragePortFmt, c.Storage.Port)
	}
	if c.GRPC.Enabled && (c.GRPC.Port <= 0 || c.GRPC.Port > MaxPortNumber) {
		return fmt.Errorf(ErrInvalidGRPCPortFmt, c.GRPC.Port)
	}
	if c.GRPC.InternalTrustEnabled && c.GRPC.Web.Enabled {
		return errors.New(ErrInternalTrustWithGRPCWeb)
	}
	return nil
}

// validateLocalAuth enforces the local login, registration and refresh token
// rules.
func (c *Config) validateLocalAuth() error {
	localLogin := c.Auth.Enabled && c.Auth.LocalLoginEnabled
	switch {
	case c.Auth.LocalLoginEnabled && !c.Auth.Enabled:
		return errors.New(ErrLocalLoginRequiresAuth)
	case localLogin && c.Auth.HMACSecret == "":
		return errors.New(ErrLocalLoginHMACSecretRequired)
	case localLogin && len(c.Auth.HMACSecret) < AuthHMACSecretMinLength:
		return fmt.Errorf(ErrLocalLoginHMACSecretTooShortFmt, AuthHMACSecretMinLength, len(c.Auth.HMACSecret))
	case localLogin && c.Auth.JWKSEndpoint != "":
		return errors.New(ErrLocalLoginJWKSMutuallyExclusive)
	case c.Auth.RegistrationEnabled && !c.Auth.LocalLoginEnabled:
		return errors.New(ErrRegistrationRequiresLocalLogin)
	case c.Auth.RefreshTokenEnabled && !c.Auth.LocalLoginEnabled:
		return errors.New(ErrRefreshTokenRequiresLocalLogin)
	case c.Auth.RefreshTokenEnabled && c.Auth.RefreshTokenTTL <= c.Auth.AccessTokenTTL:
		return errors.New(ErrRefreshTokenTTLInvalid)
	}
	return nil
}

// validateExternalAuth enforces the OIDC and OAuth2 rules and the Redis store
// external auth needs. Local JWTs are still issued after the exchange, so the
// HMAC secret is required too.
func (c *Config) validateExternalAuth() error {
	if !c.Auth.OIDC.Enabled && !c.Auth.OAuth2.Enabled {
		return nil
	}
	switch {
	case !c.Auth.Enabled:
		return errors.New(MsgAuthMustBeEnabled)
	case c.Auth.HMACSecret == "":
		return errors.New(MsgHMACSecretRequired)
	case len(c.Auth.HMACSecret) < AuthHMACSecretMinLength:
		return fmt.Errorf(ErrLocalLoginHMACSecretTooShortFmt, AuthHMACSecretMinLength, len(c.Auth.HMACSecret))
	}
	if err := c.validateOIDC(); err != nil {
		return err
	}
	if err := c.validateOAuth2(); err != nil {
		return err
	}
	switch {
	case c.Redis.Host == "":
		return errors.New(MsgRedisRequiredForExternal)
	case c.Redis.Port <= 0 || c.Redis.Port > MaxPortNumber:
		return errors.New(MsgRedisPortInvalid)
	}
	return nil
}

func (c *Config) validateOIDC() error {
	oidc := c.Auth.OIDC
	if !oidc.Enabled {
		return nil
	}
	switch {
	case oidc.ProviderURL == "":
		return errors.New(MsgOIDCProviderURLRequired)
	case oidc.ClientID == "":
		return errors.New(MsgOIDCClientIDRequired)
	case oidc.ClientSecret == "":
		return errors.New(MsgOIDCClientSecretRequired)
	case oidc.RedirectURL == "":
		return errors.New(MsgOIDCRedirectURLRequired)
	case oidc.StateTTL <= 0:
		return errors.New(MsgStateTTLPositive)
	case oidc.NonceTTL <= 0:
		return errors.New(MsgNonceTTLPositive)
	case oidc.RegistrationEnabled && oidc.RegistrationCallbackURL == "":
		return errors.New(MsgOIDCRegCallbackURLRequired)
	}
	return nil
}

// validateOAuth2 requires a client secret unless the client is public
// (PKCE-only flow).
func (c *Config) validateOAuth2() error {
	oauth := c.Auth.OAuth2
	if !oauth.Enabled {
		return nil
	}
	switch {
	case oauth.AuthorizeURL == "":
		return errors.New(MsgOAuth2AuthorizeURLRequired)
	case oauth.TokenURL == "":
		return errors.New(MsgOAuth2TokenURLRequired)
	case oauth.UserInfoURL == "":
		return errors.New(MsgOAuth2UserInfoURLRequired)
	case oauth.ClientID == "":
		return errors.New(MsgOAuth2ClientIDRequired)
	case !oauth.PublicClient && oauth.ClientSecret == "":
		return errors.New(MsgOAuth2ClientSecretRequired)
	case oauth.RedirectURL == "":
		return errors.New(MsgOAuth2RedirectURLRequired)
	case oauth.StateTTL <= 0:
		return errors.New(MsgStateTTLPositive)
	case oauth.RegistrationEnabled && oauth.RegistrationCallbackURL == "":
		return errors.New(MsgOAuth2RegCallbackURLRequired)
	}
	return nil
}

func (c *Config) validateGatewayRateLimit() error {
	limit := c.Gateway.RateLimit
	if !limit.Enabled {
		return nil
	}
	switch {
	case limit.RequestsPerMin <= 0:
		return errors.New(ErrRateLimitRequestsPerMinPositive)
	case limit.Burst <= 0:
		return errors.New(ErrRateLimitBurstPositive)
	case limit.CleanupInterval <= 0:
		return errors.New(ErrRateLimitCleanupIntervalPositive)
	}
	return nil
}

// validateCertificates requires every certificate service setting; the
// loader's defaults are its only fallback.
func (c *Config) validateCertificates() error {
	certs := c.Certificates
	switch {
	case certs.CertGenPath == "":
		return errors.New(ErrCertificatesCertGenPathRequired)
	case certs.CertsDir == "":
		return errors.New(ErrCertificatesCertsDirRequired)
	case certs.TempDir == "":
		return errors.New(ErrCertificatesTempDirRequired)
	case certs.ServerValidityDays <= 0:
		return errors.New(ErrCertificatesServerValidityDaysPositive)
	case certs.CleanupIntervalMin <= 0:
		return errors.New(ErrCertificatesCleanupIntervalPositive)
	}
	return nil
}

// validateCommunityEdition refuses the organization settings the community
// edition, which serves one default tenant, cannot run with.
func (c *Config) validateCommunityEdition() error {
	if c.General.Edition != EditionCommunity {
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
	return nil
}

// validateEnterpriseOrgEnforcement refuses an ECE configuration without
// organization enforcement. KC-Core and KC-Gateway act on the setting, so
// only their validation applies it; KC-Identity never reads it.
func (c *Config) validateEnterpriseOrgEnforcement() error {
	if c.General.Edition == EditionECE && !c.General.OrgEnforcementEnabled {
		return errors.New(ErrECEOrgEnforcementRequired)
	}
	return nil
}

// validateSCACIListener refuses a SCACI listener that cannot tell the
// organizations' Application Centers apart (SCACI §1): ECE resolves every
// peer certificate strictly, and strict resolution needs the certificate's
// tenant mapping. Only KC-Core runs the listener, so only its validation
// applies it.
func (c *Config) validateSCACIListener() error {
	if !c.Protocol.SCACIEnabled {
		return nil
	}
	if c.General.Edition == EditionECE && !c.Protocol.StrictOrgResolution {
		return errors.New(ErrECEStrictOrgResolutionRequired)
	}
	if c.Protocol.StrictOrgResolution && !c.Protocol.SCACICertTenantMapping {
		return errors.New(ErrStrictOrgResolutionRequiresCertTenantMapping)
	}
	return nil
}
