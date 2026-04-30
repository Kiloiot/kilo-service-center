package builders

import (
	"context"
	"fmt"
	"time"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	"github.com/kilocenter/KC-Core/internal/grpc"
	pkgconfig "github.com/kilocenter/KC-Core/pkg/config"
	"github.com/kilocenter/KC-Core/pkg/grpc/interceptors"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/management"
	"github.com/kilocenter/KC-Core/pkg/org"
	"github.com/kilocenter/KC-DB/storage/models"
)

// BuildGRPCServer constructs the gRPC server with interceptor chain (auth, org resolver).
func BuildGRPCServer(infra *Infrastructure) (*grpc.Server, grpc.Config, error) {
	log := logger.Get()

	// Create gRPC organization resolver adapter (tenantID → orgUUID)
	grpcOrgResolver := grpc.NewOrgResolverAdapter(infra.Storage.Organizations(), infra.LoggerIface)

	// Create tenant resolver adapter (orgUUID → tenantID)
	grpcTenantResolver := grpc.NewTenantResolverAdapter(infra.OrgResolverSvc, infra.LoggerIface)

	cfg := infra.Config

	var failClosedResolver org.Resolver
	if cfg.General.OrgEnforcementEnabled {
		failClosedResolver = infra.OrgResolverSvc
		log.Info("Initializing gRPC server with organization enforcement enabled")
	} else {
		log.Info("Initializing gRPC server in community mode (org enforcement disabled)")
	}

	// RBAC policies are always defined (fail-closed when identity unavailable)
	rbacPolicies := map[string][]string{
		pb.KiloCenterService_GenerateServerCertificates_FullMethodName: {models.OrganizationRoleOwner, models.OrganizationRoleAdmin},
		pb.KiloCenterService_RenewServerCertificates_FullMethodName:    {models.OrganizationRoleOwner, models.OrganizationRoleAdmin},
		pb.CoreService_GenerateServerCertificates_FullMethodName:       {models.OrganizationRoleOwner, models.OrganizationRoleAdmin},
		pb.CoreService_RenewServerCertificates_FullMethodName:          {models.OrganizationRoleOwner, models.OrganizationRoleAdmin},
	}

	// Role resolver requires KC-Identity connection
	var roleResolver interceptors.RoleResolver
	if infra.IdentityInternalClient != nil {
		roleResolver = grpc.NewRoleResolver(infra.IdentityInternalClient, cfg.InternalAuth.PeerSecret)
		log.Info("RBAC policies configured for certificate management RPCs")
	}

	// Resolve RBAC cache TTL from config with bounds check
	rbacCacheTTL := time.Duration(cfg.GRPC.RBACRoleCacheTTLSeconds) * time.Second
	if cfg.GRPC.RBACRoleCacheTTLSeconds <= 0 {
		rbacCacheTTL = time.Duration(pkgconfig.DefaultRBACRoleCacheTTLSeconds) * time.Second
		log.Warn("Invalid rbac_role_cache_ttl_seconds, using default", "default", pkgconfig.DefaultRBACRoleCacheTTLSeconds)
	}

	grpcConfig := grpc.Config{
		Port:                 cfg.GRPC.Port,
		Host:                 cfg.GRPC.Host,
		InternalTrustEnabled: cfg.GRPC.InternalTrustEnabled,
		TLSCert:              cfg.GRPC.TLSCert,
		TLSKey:               cfg.GRPC.TLSKey,
		EnableTLS:            cfg.GRPC.EnableTLS,
		Auth: grpc.AuthConfig{
			Enabled:          cfg.Auth.Enabled,
			JWKSEndpoint:     cfg.Auth.JWKSEndpoint,
			Issuer:           cfg.Auth.Issuer,
			Audience:         cfg.Auth.Audience,
			TenantClaim:      cfg.Auth.TenantClaim,
			Algorithm:        cfg.Auth.Algorithm,
			UserClaim:        cfg.Auth.OAuth2.UserIDClaim,
			HMACSecret:       cfg.Auth.HMACSecret,
			EventWriter:      infra.Storage.SystemEvents(),
			PlatformTenantID: cfg.General.TenantID,
		},
		EventWriter:           infra.Storage.SystemEvents(),
		PlatformTenantID:      cfg.General.TenantID,
		OrgResolver:           grpcOrgResolver,
		TenantResolver:        grpcTenantResolver,
		FailClosedOrgResolver: failClosedResolver,
		DefaultTenantID:       cfg.General.TenantID,
		APIKeyAuth:            grpc.NewCoreAPIKeyAdapter(infra.Storage.APIKeys()),
		RoleResolver:          roleResolver,
		RBACPolicies:          rbacPolicies,
		RBACRoleCacheTTL:      rbacCacheTTL,
		EnableReflection:      cfg.GRPC.EnableReflection,
		EnableHealth:          cfg.GRPC.EnableHealth,
		HTTPConfig: grpc.HTTPServerConfig{
			ReadTimeout:  cfg.GRPC.HTTP.ReadTimeout,
			WriteTimeout: cfg.GRPC.HTTP.WriteTimeout,
			IdleTimeout:  cfg.GRPC.HTTP.IdleTimeout,
		},
		GRPCWeb: grpc.WebConfig{
			Enabled:          cfg.GRPC.Web.Enabled,
			AllowedOrigins:   cfg.GRPC.Web.AllowedOrigins,
			AllowedHeaders:   cfg.GRPC.Web.AllowedHeaders,
			ExposeHeaders:    cfg.GRPC.Web.ExposeHeaders,
			AllowCredentials: cfg.GRPC.Web.AllowCredentials,
			MaxAge:           cfg.GRPC.Web.MaxAge,
			AllowAllOrigins:  cfg.GRPC.Web.AllowAllOrigins,
			EnableWebsockets: cfg.GRPC.Web.EnableWebsockets,
			AllowedMethods:   cfg.GRPC.Web.AllowedMethods,
		},
	}

	grpcServer, err := grpc.NewServer(grpcConfig)
	if err != nil {
		return nil, grpc.Config{}, fmt.Errorf("failed to create gRPC server: %w", err)
	}

	return grpcServer, grpcConfig, nil
}

// RegisterAndServe registers gRPC services and starts the gRPC + management HTTP servers.
func RegisterAndServe(
	grpcServer *grpc.Server,
	grpcConfig grpc.Config,
	core *grpc.CoreService,
	protocol *ProtocolServers,
	infra *Infrastructure,
	cancel context.CancelFunc,
) {
	log := logger.Get()

	compatService := grpc.NewKiloCenterServiceCompat(core)
	pb.RegisterCoreServiceServer(grpcServer.GetServer(), core)
	pb.RegisterKiloCenterServiceServer(grpcServer.GetServer(), compatService)
	log.Info("gRPC services registered (Core + KiloCenterCompat)")

	// Start gRPC server in a goroutine AFTER service registration
	go func() {
		log.Info("Starting gRPC server", "port", grpcConfig.Port)
		if err := grpcServer.Start(); err != nil {
			log.Error(LogGRPCServerFailed, logger.Err(err))
			cancel()
		}
	}()

	// Create BSSCI manager for API access
	bssciManager := management.NewBSSCIManager(protocol.BSSCIServer, infra.Storage, infra.TenantID, infra.LoggerIface)
	log.Info("BSSCI manager initialized")

	// Start internal HTTP server for BSSCI management
	go func() {
		if err := bssciManager.StartHTTPServer(8081); err != nil {
			log.Error(LogFailedBSSCIMgmtServer, logger.Err(err))
		}
	}()
}
