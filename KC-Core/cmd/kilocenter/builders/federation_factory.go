package builders

import (
	"time"

	federationservices "github.com/kilocenter/KC-Core/internal/services/federation"
	orgresolver "github.com/kilocenter/KC-Core/internal/services/orgresolver"
	"github.com/kilocenter/KC-Core/pkg/federation"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/org"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/postgres"
)

// NewCEBootstrapHandler constructs and configures the CE bootstrap service,
// returning the public CEBootstrapHandler interface.
// This allows external ECE modules to reuse the CE bootstrap handler.
func NewCEBootstrapHandler(fctx *FederationContext) federation.CEBootstrapHandler {
	ceInstallRepo := postgres.NewCEInstallationRepository(fctx.SqlxDB)
	svc := federationservices.NewCEBootstrapService(ceInstallRepo, fctx.LoggerIface, fctx.Edition)
	if fctx.RelayClient != nil {
		svc = svc.
			WithRelayController(fctx.RelayClient).
			WithConnectedFn(fctx.RelayClient.IsConnected)
	}
	if fctx.DispositionResolver != nil {
		svc = svc.WithRelayGate(fctx.DispositionResolver)
	}
	return svc
}

// NewLocalDBOrgResolver creates a local DB-backed org resolver with caching.
// This allows external ECE modules to create the local resolver
// without importing internal packages.
func NewLocalDBOrgResolver(
	orgRepo interfaces.OrganizationRepository,
	log logger.Logger,
	cacheTTL time.Duration,
	maxEntries int,
) org.Resolver {
	return orgresolver.New(orgRepo, log, cacheTTL, maxEntries)
}
