package builders

import (
	"context"
	"errors"
	"fmt"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/health"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/management"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
)

// BuildGRPCServer constructs the gRPC server with interceptor chain (auth, org resolver).
func BuildGRPCServer(infra *Infrastructure) (*grpc.Server, grpc.Config, error) {
	log := logger.Get()

	// Create gRPC organization resolver adapter (tenantID → orgUUID)
	grpcOrgResolver := grpc.NewOrgResolverAdapter(infra.Repos.Organizations, infra.LoggerIface)

	// Create tenant resolver adapter (orgUUID → tenantID)
	grpcTenantResolver := grpc.NewTenantResolverAdapter(infra.OrgResolverSvc, infra.LoggerIface)

	cfg := infra.Config

	var failClosedResolver org.Resolver
	if cfg.General.OrgEnforcementEnabled {
		failClosedResolver = infra.OrgResolverSvc
		log.Info(LogInitializingGRPCServerWithOrganizationEnforcementEnabled)
	} else {
		log.Info(LogInitializingGRPCServerCommunityMode)
	}

	roleSource, err := buildRoleSource(infra)
	if err != nil {
		return nil, grpc.Config{}, err
	}

	// Admin checker lets the fail-closed org resolver exempt server admins from the org-mismatch check.
	var adminChecker interceptors.AdminChecker
	if infra.IdentityInternalClient != nil {
		adminChecker = grpc.NewAdminOrgAdapter(infra.IdentityInternalClient, cfg.InternalAuth.PeerSecret)
		log.Info(LogAdminCheckerWiredIntoOrgResolverInterceptor)
	}

	grpcConfig := grpc.Config{
		Log:                  infra.LoggerIface,
		Port:                 cfg.GRPC.Port,
		Host:                 cfg.GRPC.Host,
		InternalTrustEnabled: cfg.GRPC.InternalTrustEnabled,
		PeerSecret:           cfg.InternalAuth.PeerSecret,
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
			EventWriter:      infra.Repos.SystemEvents,
			PlatformTenantID: cfg.General.TenantID,
			Logger:           infra.LoggerIface,
		},
		EventWriter:           infra.Repos.SystemEvents,
		PlatformTenantID:      cfg.General.TenantID,
		AdminChecker:          adminChecker,
		OrgResolver:           grpcOrgResolver,
		TenantResolver:        grpcTenantResolver,
		FailClosedOrgResolver: failClosedResolver,
		DefaultTenantID:       cfg.General.TenantID,
		DefaultOrgResolver:    infra.OrgResolverSvc,
		APIKeyAuth:            grpc.NewCoreAPIKeyAdapter(infra.Repos.APIKeys),
		RoleSource:            roleSource,
		EnableReflection:      cfg.GRPC.EnableReflection,
		EnableHealth:          cfg.GRPC.EnableHealth,
		HTTPConfig: grpc.HTTPServerConfig{
			ReadTimeout:  cfg.GRPC.HTTP.ReadTimeout,
			WriteTimeout: cfg.GRPC.HTTP.WriteTimeout,
			IdleTimeout:  cfg.GRPC.HTTP.IdleTimeout,
		},
		GRPCWeb: cfg.GRPC.Web,
	}

	grpcServer, err := grpc.NewServer(grpcConfig)
	if err != nil {
		return nil, grpc.Config{}, fmt.Errorf("%s: %w", errMsgFailedToCreateGRPCServer, err)
	}
	infra.statusBoard.require(statusNameGRPC, pkgconfig.ListenerProbeAddress(cfg.GRPC.Host, cfg.GRPC.Port),
		health.NewListenerChecker(grpcServer))

	return grpcServer, grpcConfig, nil
}

// buildRoleSource resolves every caller's roles through KC-Identity; without
// it no request could be authorized, so the service refuses to start.
func buildRoleSource(infra *Infrastructure) (interceptors.RoleSource, error) {
	log := logger.Get()
	cfg := infra.Config
	if infra.IdentityInternalClient == nil {
		if cfg.Identity.Address == "" {
			return nil, errors.New(errMsgRoleSourceNeedsIdentityAddress)
		}
		return nil, fmt.Errorf(errFmtRoleSourceIdentityClientMissing, cfg.Identity.Address)
	}
	cacheTTL := time.Duration(cfg.GRPC.RBACRoleCacheTTLSeconds) * time.Second
	if cfg.GRPC.RBACRoleCacheTTLSeconds <= 0 {
		cacheTTL = time.Duration(pkgconfig.DefaultRBACRoleCacheTTLSeconds) * time.Second
		log.Warn(LogInvalidRbacRoleCacheTTLSecondsUsingDefault, logger.FieldDefault, pkgconfig.DefaultRBACRoleCacheTTLSeconds)
	}
	log.Info(LogRoleEnforcementWired, logger.FieldCacheTTL, cacheTTL)
	return grpc.NewIdentityRoleSource(infra.IdentityInternalClient, cfg.InternalAuth.PeerSecret, cacheTTL), nil
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
	log.Info(LogGRPCServicesRegisteredCoreKiloCenterCompat)

	// Start gRPC server in a goroutine AFTER service registration
	go func() {
		log.Info(LogStartingGRPCServer, logger.FieldPort, grpcConfig.Port)
		if err := grpcServer.Start(); err != nil {
			log.Error(LogGRPCServerFailed, logger.Err(err))
			cancel()
		}
	}()

	// Create BSSCI manager for API access
	bssciManager := management.NewBSSCIManager(protocol.BSSCIServer, infra.LoggerIface)
	log.Info(LogBSSCIManagerInitialized)

	// Start internal HTTP server for BSSCI management
	go func() {
		if err := bssciManager.StartHTTPServer(infra.Config.Protocol.ManagementPort); err != nil {
			log.Error(LogFailedBSSCIMgmtServer, logger.Err(err))
		}
	}()
}
