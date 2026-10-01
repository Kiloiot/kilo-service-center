package builders

import (
	"context"

	pkggrpc "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/federation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// FederationContext bundles the dependencies a FederationWirer needs.
// All types are public so external modules can implement FederationWirer.
type FederationContext struct {
	// Storage is the connection-only handle an enterprise wirer builds its
	// own repositories on.
	Storage             *postgres.DB
	CEInstallations     *postgres.CEInstallationRepository
	LoggerIface         logger.Logger
	Clock               clock.Clock
	Edition             string
	RelayClient         federation.RelayController
	DispositionResolver federation.RelayGate
}

// FederationResult holds the handlers produced by a FederationWirer.
type FederationResult struct {
	BootstrapHandler pkggrpc.CEBootstrapHandler
	RegistryHandler  pkggrpc.CERegistryHandler
}

// FederationWirer constructs edition-specific federation service handlers.
// CE wires only bootstrap; ECE can additionally provide registry handler.
type FederationWirer func(fctx *FederationContext) (*FederationResult, error)

// OrgResolverConfig holds the configuration values an OrgResolverBuilder needs.
type OrgResolverConfig struct {
	Edition              string
	TenantID             int64
	IdentityAddress      string
	PeerSecret           string
	InternalTrustEnabled bool
	OrgCacheTTLMinutes   int
	OrgCacheMaxEntries   int
}

// NewOrgResolverConfig derives the org resolver settings from the service
// configuration, so every binary that resolves organizations reads them alike.
func NewOrgResolverConfig(cfg *pkgconfig.Config) *OrgResolverConfig {
	return &OrgResolverConfig{
		Edition:              cfg.General.Edition,
		TenantID:             cfg.General.TenantID,
		IdentityAddress:      cfg.Identity.Address,
		PeerSecret:           cfg.InternalAuth.PeerSecret,
		InternalTrustEnabled: cfg.GRPC.InternalTrustEnabled,
		OrgCacheTTLMinutes:   cfg.General.OrgCacheTTLMinutes,
		OrgCacheMaxEntries:   cfg.General.OrgCacheMaxEntries,
	}
}

// OrgResolverResult holds the output of the edition-specific org resolver builder.
type OrgResolverResult struct {
	Resolver               org.Resolver
	IdentityInternalClient pb.IdentityInternalServiceClient
	Cleanups               []func()
}

// OrgResolverBuilder constructs an edition-specific org resolver.
type OrgResolverBuilder func(
	ctx context.Context,
	ocfg *OrgResolverConfig,
	orgRepo interfaces.OrgDirectoryRepository,
	log logger.Logger,
) (*OrgResolverResult, error)

// Options configures edition-specific behavior for the builder pipeline.
// Fields left nil use CE defaults.
type Options struct {
	OrgResolverBuilder OrgResolverBuilder
	FederationWirer    FederationWirer
}
