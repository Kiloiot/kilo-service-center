package builders

import (
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	dbadapters "github.com/Kiloiot/kilo-service-center/KC-DB/storage/adapters"
	identitygrpc "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/grpc"
	identityAdapters "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/adapters"
	adminservice "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/admin"
	authservice "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/registration"
)

// IdentityResult holds the fully wired identity services and cleanup functions.
type IdentityResult struct {
	IdentityService         *identitygrpc.IdentityService
	IdentityInternalService *identitygrpc.IdentityInternalService
	CompatService           *identitygrpc.KiloCenterServiceCompatIdentity
	Cleanups                []func()
}

// BuildIdentityService constructs the IdentityService with admin, auth, and external auth wired.
func BuildIdentityService(infra *Infrastructure) *IdentityResult {
	log := logger.Get()
	cfg := infra.Config
	var cleanups []func()

	identityService := identitygrpc.NewIdentityService()
	identityService = identityService.WithEventWriter(infra.Storage.SystemEvents()).
		WithPlatformTenantID(infra.TenantID)

	log.Info("Wiring Auth and Admin services...")

	// UserAdminService
	userStore := dbadapters.NewUserStoreAdapter(infra.SqlxDB)
	userAdminSvc := adminservice.NewUserAdminService(userStore, infra.Log)
	identityService = identityService.WithAdminUserService(userAdminSvc)
	log.Info("UserAdminService wired")

	// OrganizationAdminService and MembershipAdminService: enterprise-only.
	// In CE, these remain nil; handlers return ErrTokenServiceNotConfigured.
	if !pkgconfig.IsCommunityEdition(cfg.General.Edition) {
		orgStore := dbadapters.NewOrganizationAdminAdapter(infra.SqlxDB)
		tenantStoreWrapper := identityAdapters.NewTenantStoreAdapterWrapper(dbadapters.NewTenantStoreAdapter(infra.SqlxDB))
		orgAdminSvc := adminservice.NewOrganizationAdminService(orgStore, tenantStoreWrapper, infra.Log)
		identityService = identityService.WithOrganizationService(orgAdminSvc)
		log.Info("OrganizationAdminService wired")

		memberStore := identityAdapters.NewOrganizationMemberStoreAdapter(infra.Storage.Organizations())
		membershipSvc := adminservice.NewMembershipAdminService(memberStore, userStore, infra.Log)
		identityService = identityService.WithMembershipService(membershipSvc)
		log.Info("MembershipAdminService wired")
	} else {
		log.Info("Community Edition: OrganizationAdminService and MembershipAdminService skipped")
	}

	// APIKeyAdminService
	apiKeySvc := adminservice.NewAPIKeyAdminService(infra.Storage.APIKeys(), infra.Log)
	identityService = identityService.WithAPIKeyService(apiKeySvc)
	log.Info("APIKeyAdminService wired")

	// AuthService
	authMembershipStore := dbadapters.NewOrgMembershipStoreAdapter(infra.SqlxDB)
	refreshTokenStore := dbadapters.NewRefreshTokenStoreAdapter(infra.SqlxDB)
	var tokenIssuer *authservice.TokenIssuer
	if cfg.Auth.LocalLoginEnabled || cfg.Auth.OIDC.Enabled || cfg.Auth.OAuth2.Enabled {
		tokenIssuer = authservice.NewTokenIssuer(
			[]byte(cfg.Auth.HMACSecret),
			cfg.Auth.TenantClaim,
			cfg.Auth.Issuer,
			cfg.Auth.Audience,
			cfg.Auth.AccessTokenTTL,
			cfg.Auth.RefreshTokenTTL,
		)
	}
	authSvc := authservice.NewService(
		userStore,
		authMembershipStore,
		refreshTokenStore,
		tokenIssuer,
		cfg.Auth.LocalLoginEnabled,
		cfg.Auth.RefreshTokenEnabled,
		infra.Log,
	)

	// CE: wire default-org membership synthesizer
	var ceProvider *authservice.CEDefaultOrgProvider
	if pkgconfig.IsCommunityEdition(cfg.General.Edition) {
		ceProvider = authservice.NewCEDefaultOrgProvider(infra.Storage.Organizations(), cfg.General.TenantID)
		authSvc.WithCEProvider(ceProvider)
		log.Info("CE default-org provider wired for AuthService")
	}

	identityService = identityService.WithAuthService(authSvc)
	log.Info("AuthService wired", "localLoginEnabled", cfg.Auth.LocalLoginEnabled)

	// RegistrationService (self-service signup, only when explicitly enabled)
	if cfg.Auth.RegistrationEnabled && cfg.Auth.LocalLoginEnabled {
		registrationAdapter := dbadapters.NewRegistrationAdapter(infra.SqlxDB)
		registrationSvc := registration.NewService(
			registrationAdapter,
			userStore,
			tokenIssuer,
			refreshTokenStore,
			authMembershipStore,
			cfg.Auth.RegistrationEnabled,
			cfg.Auth.LocalLoginEnabled,
			cfg.Auth.RefreshTokenEnabled,
			infra.Log,
		)
		if pkgconfig.IsCommunityEdition(cfg.General.Edition) {
			registrationSvc.WithCEMode(cfg.General.TenantID, infra.Storage.Organizations())
		}
		identityService = identityService.WithRegistrationService(registrationSvc)
		log.Info("RegistrationService wired")
	} else {
		log.Info("RegistrationService skipped", "registrationEnabled", cfg.Auth.RegistrationEnabled, "localLoginEnabled", cfg.Auth.LocalLoginEnabled)
	}

	// ExternalAuthService (OIDC/OAuth2)
	if cfg.Auth.OIDC.Enabled || cfg.Auth.OAuth2.Enabled {
		redisClient, err := authservice.NewRedisClient(&cfg.Redis, infra.Log)
		if err != nil {
			log.Fatal("failed to create Redis client for external auth", "error", err)
		}
		cleanups = append(cleanups, func() {
			if err := redisClient.Close(); err != nil {
				log.Error("failed to close Redis client", "error", err)
			}
		})

		oidcStateStore := identityAdapters.NewOIDCStateStoreAdapter(redisClient)
		oauth2StateStore := identityAdapters.NewOAuth2StateStoreAdapter(redisClient)

		var oidcClient authservice.OIDCClient
		if cfg.Auth.OIDC.Enabled {
			oidcClient, err = authservice.NewOIDCClient(authservice.OIDCClientConfig{
				ProviderURL:  cfg.Auth.OIDC.ProviderURL,
				ClientID:     cfg.Auth.OIDC.ClientID,
				ClientSecret: cfg.Auth.OIDC.ClientSecret,
				RedirectURL:  cfg.Auth.OIDC.RedirectURL,
				Scopes:       cfg.Auth.OIDC.Scopes,
			}, infra.Log)
			if err != nil {
				log.Fatal("failed to create OIDC client", "error", err)
			}
		}

		var oauth2Client authservice.OAuth2Client
		if cfg.Auth.OAuth2.Enabled {
			oauth2Client = authservice.NewOAuth2Client(authservice.OAuth2ClientConfig{
				AuthorizeURL: cfg.Auth.OAuth2.AuthorizeURL,
				TokenURL:     cfg.Auth.OAuth2.TokenURL,
				UserInfoURL:  cfg.Auth.OAuth2.UserInfoURL,
				ClientID:     cfg.Auth.OAuth2.ClientID,
				ClientSecret: cfg.Auth.OAuth2.ClientSecret,
				PublicClient: cfg.Auth.OAuth2.PublicClient,
				RedirectURL:  cfg.Auth.OAuth2.RedirectURL,
				Scopes:       cfg.Auth.OAuth2.Scopes,
				PKCEMethod:   cfg.Auth.OAuth2.PKCEMethod,
				UserIDClaim:  cfg.Auth.OAuth2.UserIDClaim,
				EmailClaim:   cfg.Auth.OAuth2.EmailClaim,
			}, infra.Log)
		}

		externalAuthCfg := authservice.ExternalAuthServiceConfig{
			OIDCEnabled:          cfg.Auth.OIDC.Enabled,
			OIDCLoginLabel:       cfg.Auth.OIDC.LoginLabel,
			OIDCLoginRedirect:    cfg.Auth.OIDC.LoginRedirect,
			OIDCStateTTL:         cfg.Auth.OIDC.StateTTL,
			OIDCNonceTTL:         cfg.Auth.OIDC.NonceTTL,
			OIDCRegEnabled:       cfg.Auth.OIDC.RegistrationEnabled,
			OIDCAssumeVerified:   cfg.Auth.OIDC.AssumeEmailVerified,
			OIDCRegCallbackURL:   cfg.Auth.OIDC.RegistrationCallbackURL,
			OIDCExternalOrgClaim: cfg.Auth.OIDC.ExternalOrgClaim,
			OAuth2Enabled:        cfg.Auth.OAuth2.Enabled,
			OAuth2LoginLabel:     cfg.Auth.OAuth2.LoginLabel,
			OAuth2LoginRedirect:  cfg.Auth.OAuth2.LoginRedirect,
			OAuth2StateTTL:       cfg.Auth.OAuth2.StateTTL,
			OAuth2RegEnabled:     cfg.Auth.OAuth2.RegistrationEnabled,
			OAuth2AssumeVerified: cfg.Auth.OAuth2.AssumeEmailVerified,
			OAuth2RegCallbackURL: cfg.Auth.OAuth2.RegistrationCallbackURL,
		}

		// Type assert orgResolverSvc to org.OrganizationResolver
		orgResolver, ok := infra.OrgResolverSvc.(org.OrganizationResolver)
		if !ok {
			log.Fatal("orgResolverSvc does not implement org.OrganizationResolver")
		}
		externalAuthSvc := authservice.NewExternalAuthService(
			oidcClient,
			oauth2Client,
			oidcStateStore,
			oauth2StateStore,
			userStore,
			authMembershipStore,
			tokenIssuer,
			orgResolver,
			externalAuthCfg,
			infra.Log,
		)
		if ceProvider != nil {
			externalAuthSvc.WithCEProvider(ceProvider)
		}
		identityService = identityService.WithExternalAuthService(externalAuthSvc)
		log.Info("ExternalAuthService wired",
			"oidcEnabled", cfg.Auth.OIDC.Enabled,
			"oauth2Enabled", cfg.Auth.OAuth2.Enabled)
	}

	// Build IdentityInternalService for peer-to-peer RPCs
	apiKeyLookup := newAPIKeyLookupAdapter(infra.Storage.APIKeys())
	membershipLookup := identitygrpc.NewMembershipLookup(infra.Storage.Organizations())
	internalService := identitygrpc.NewIdentityInternalService(infra.OrgResolverSvc, apiKeyLookup, membershipLookup, userAdminSvc, infra.Storage.SystemEvents())
	log.Info("IdentityInternalService wired")

	// Build compat shim (28 identity RPCs on KiloCenterService)
	compatService := identitygrpc.NewKiloCenterServiceCompatIdentity(identityService)
	log.Info("KiloCenterServiceCompatIdentity wired")

	return &IdentityResult{
		IdentityService:         identityService,
		IdentityInternalService: internalService,
		CompatService:           compatService,
		Cleanups:                cleanups,
	}
}
