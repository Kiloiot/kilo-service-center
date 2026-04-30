package builders

import (
	"context"

	"github.com/jmoiron/sqlx"
	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	"github.com/kilocenter/KC-Core/pkg/federation"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/org"
	"github.com/kilocenter/KC-DB/storage/interfaces"
)

// FederationContext bundles the dependencies a FederationWirer needs.
// All types are public so external modules can implement FederationWirer.
type FederationContext struct {
	SqlxDB              *sqlx.DB
	LoggerIface         logger.Logger
	Edition             string
	RelayClient         federation.RelayController
	DispositionResolver federation.RelayGate
}

// FederationResult holds the handlers produced by a FederationWirer.
type FederationResult struct {
	BootstrapHandler federation.CEBootstrapHandler
	RegistryHandler  federation.CERegistryHandler
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
	orgRepo interfaces.OrganizationRepository,
	log logger.Logger,
) (*OrgResolverResult, error)

// Options configures edition-specific behavior for the builder pipeline.
// Fields left nil use CE defaults.
type Options struct {
	OrgResolverBuilder OrgResolverBuilder
	FederationWirer    FederationWirer
}
