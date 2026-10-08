package builders

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	audit "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	dbadapters "github.com/Kiloiot/kilo-service-center/KC-DB/storage/adapters"
	authadapters "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/adapters/auth"
	identitygrpc "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/grpc"
	identityAdapters "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/adapters"
	adminservice "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/admin"
	authservice "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/registration"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/roles"
)

// IdentityResult holds the fully wired identity services and cleanup functions.
type IdentityResult struct {
	IdentityService         *identitygrpc.IdentityService
	IdentityInternalService *identitygrpc.IdentityInternalService
	CompatService           *identitygrpc.KiloCenterServiceCompatIdentity
	Cleanups                []func()
}

// BuildIdentityService constructs the IdentityService with admin, auth, and
// external auth wired. ctx is the process lifecycle context and bounds the
// startup connectivity checks of the external auth clients.
func BuildIdentityService(ctx context.Context, infra *Infrastructure) *IdentityResult {
	log := logger.Get()
	cfg := infra.Config
	var cleanups []func()

	auditEmitter, err := audit.NewEmitter(infra.Repos.SystemEvents, infra.Clock)
	if err != nil {
		log.Fatal(LogAuditEmitterCreateFailed, logger.FieldError, err)
	}
	auditDrops, err := audit.NewPrometheusDropCounter(prometheus.DefaultRegisterer)
	if err != nil {
		log.Fatal(LogAuditRecorderCreateFailed, logger.FieldError, err)
	}
	auditRecorder, err := audit.NewRecorder(auditEmitter, infra.Log, auditDrops)
	if err != nil {
		log.Fatal(LogAuditRecorderCreateFailed, logger.FieldError, err)
	}
	identityService, err := identitygrpc.NewIdentityService(log, auditRecorder, auditRecorder)
	if err != nil {
		log.Fatal(LogAuditRecorderCreateFailed, logger.FieldError, err)
	}
	identityService = identityService.WithEventWriter(infra.Repos.SystemEvents).
		WithPlatformTenantID(infra.TenantID)

	log.Info(LogWiringAuthAdminServices)

	// UserAdminService
	userStore := dbadapters.NewUserStoreAdapter(infra.Repos.Users)
	refreshTokenStore := dbadapters.NewRefreshTokenStoreAdapter(infra.Repos.RefreshTokens)
	userAdminSvc := adminservice.NewUserAdminService(userStore, refreshTokenStore, infra.Log)
	identityService = identityService.WithAdminUserService(userAdminSvc)
	log.Info(LogUserAdminServiceWired)

	roleResolver := roles.New(userStore, identityAdapters.NewOrganizationMemberStoreAdapter(infra.Repos.Organizations), infra.Repos.APIKeys)
	identityService = identityService.WithRoleResolver(roleResolver)

	// OrganizationAdminService and MembershipAdminService: enterprise-only.
	// In CE, these remain nil; handlers return ErrTokenServiceNotConfigured.
	if !pkgconfig.IsCommunityEdition(cfg.General.Edition) {
		orgStore := infra.Repos.Organizations
		tenantStoreWrapper := identityAdapters.NewTenantStoreAdapterWrapper(dbadapters.NewTenantStoreAdapter(infra.Repos.Tenants))
		orgAdminSvc := adminservice.NewOrganizationAdminService(orgStore, tenantStoreWrapper, infra.Log)
		identityService = identityService.WithOrganizationService(orgAdminSvc)
		log.Info(LogOrgAdminServiceWired)

		memberStore := identityAdapters.NewOrganizationMemberStoreAdapter(infra.Repos.Organizations)
		membershipSvc := adminservice.NewMembershipAdminService(memberStore, userStore, infra.Log)
		identityService = identityService.WithMembershipService(membershipSvc)
		log.Info(LogMembershipAdminWired)
	} else {
		log.Info(LogCEAdminServicesSkipped)
	}

	identityService = withAPIKeyAdministration(identityService, infra.Repos.APIKeys, infra.Repos.Organizations, infra.Log)
	log.Info(LogAPIKeyAdminServiceWired)

	// AuthService
	authMembershipStore := infra.Repos.Organizations
	var tokenIssuer authservice.TokenIssuer
	if cfg.Auth.LocalLoginEnabled || cfg.Auth.OIDC.Enabled || cfg.Auth.OAuth2.Enabled {
		tokenIssuer = authservice.NewJWTTokenIssuer(
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
		ceProvider = authservice.NewCEDefaultOrgProvider(infra.Repos.Organizations, cfg.General.TenantID)
		authSvc.WithCEProvider(ceProvider)
		log.Info(LogCEDefaultOrgProviderWired)
	}

	identityService = identityService.WithAuthService(authSvc)
	log.Info(LogAuthServiceWired, logger.FieldLocalLoginEnabled, cfg.Auth.LocalLoginEnabled)

	// RegistrationService (self-service signup, only when explicitly enabled)
	if cfg.Auth.RegistrationEnabled && cfg.Auth.LocalLoginEnabled {
		registrationAdapter := dbadapters.NewRegistrationAdapter(infra.Repos.Registrations)
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
			registrationSvc.WithCEMode(cfg.General.TenantID, infra.Repos.Organizations)
		}
		identityService = identityService.WithRegistrationService(registrationSvc)
		log.Info(LogRegistrationServiceWired)
	} else {
		log.Info(LogRegistrationServiceSkipped, logger.FieldRegistrationEnabled, cfg.Auth.RegistrationEnabled, logger.FieldLocalLoginEnabled, cfg.Auth.LocalLoginEnabled)
	}

	// ExternalAuthService (OIDC/OAuth2)
	if cfg.Auth.OIDC.Enabled || cfg.Auth.OAuth2.Enabled {
		redisClient, err := authadapters.NewRedisClient(ctx, &cfg.Redis, infra.Log)
		if err != nil {
			log.Fatal(LogRedisClientCreateFailed, logger.FieldError, err)
		}
		cleanups = append(cleanups, func() {
			if err := redisClient.Close(); err != nil {
				log.Error(LogRedisClientCloseFailed, logger.FieldError, err)
			}
		})

		oidcStateStore := identityAdapters.NewOIDCStateStoreAdapter(redisClient)
		oauth2StateStore := identityAdapters.NewOAuth2StateStoreAdapter(redisClient)

		var oidcClient authservice.OIDCClient
		if cfg.Auth.OIDC.Enabled {
			oidcClient, err = authadapters.NewOIDCClient(ctx, authadapters.OIDCClientConfig{
				ProviderURL:  cfg.Auth.OIDC.ProviderURL,
				ClientID:     cfg.Auth.OIDC.ClientID,
				ClientSecret: cfg.Auth.OIDC.ClientSecret,
				RedirectURL:  cfg.Auth.OIDC.RedirectURL,
				Scopes:       cfg.Auth.OIDC.Scopes,
			}, infra.Log)
			if err != nil {
				log.Fatal(LogOIDCClientCreateFailed, logger.FieldError, err)
			}
		}

		var oauth2Client authservice.OAuth2Client
		if cfg.Auth.OAuth2.Enabled {
			var oauth2Err error
			oauth2Client, oauth2Err = authadapters.NewOAuth2Client(authadapters.OAuth2ClientConfig{
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
			if oauth2Err != nil {
				log.Fatal(LogOAuth2ClientCreateFailed, logger.FieldError, oauth2Err)
			}
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
			log.Fatal(LogOrgResolverTypeMismatch)
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
			authadapters.NewRegistrationCallbackClient(infra.Log),
			externalAuthCfg,
			infra.Log,
		)
		if ceProvider != nil {
			externalAuthSvc.WithCEProvider(ceProvider)
		}
		identityService = identityService.WithExternalAuthService(externalAuthSvc)
		log.Info(LogExternalAuthServiceWired,
			logger.FieldOidcEnabled, cfg.Auth.OIDC.Enabled,
			logger.FieldOauth2Enabled, cfg.Auth.OAuth2.Enabled)
	}

	// Build IdentityInternalService for peer-to-peer RPCs
	apiKeyLookup := newAPIKeyLookupAdapter(infra.Repos.APIKeys)
	internalService := identitygrpc.NewIdentityInternalService(infra.OrgResolverSvc, apiKeyLookup, roleResolver, userAdminSvc, infra.Repos.SystemEvents, log)
	log.Info(LogIdentityInternalWired)

	// Build compat shim (28 identity RPCs on KiloCenterService)
	compatService := identitygrpc.NewKiloCenterServiceCompatIdentity(identityService)
	log.Info(LogCompatIdentityWired)

	return &IdentityResult{
		IdentityService:         identityService,
		IdentityInternalService: internalService,
		CompatService:           compatService,
		Cleanups:                cleanups,
	}
}

// withAPIKeyAdministration wires API key management in every edition: the key
// store and the organization directory that scopes each key to its tenant.
func withAPIKeyAdministration(svc *identitygrpc.IdentityService, keys adminservice.APIKeyStore, orgs grpcservices.OrganizationDirectory, log logger.Logger) *identitygrpc.IdentityService {
	return svc.WithAPIKeyService(adminservice.NewAPIKeyAdminService(keys, log)).WithOrganizationDirectory(orgs)
}
