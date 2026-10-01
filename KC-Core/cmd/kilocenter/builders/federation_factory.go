package builders

import (
	"time"

	pkggrpc "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"

	coregrpc "github.com/Kiloiot/kilo-service-center/KC-Core/internal/grpc"
	federationservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/federation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/orgresolver"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// NewCEBootstrapHandler constructs and configures the CE bootstrap service,
// returning the public CEBootstrapHandler interface.
// This allows external ECE modules to reuse the CE bootstrap handler.
func NewCEBootstrapHandler(fctx *FederationContext) (pkggrpc.CEBootstrapHandler, error) {
	svc, err := federationservices.NewCEBootstrapService(fctx.CEInstallations, fctx.LoggerIface, fctx.Clock, fctx.Edition)
	if err != nil {
		return nil, err
	}
	if fctx.RelayClient != nil {
		svc = svc.
			WithRelayController(fctx.RelayClient).
			WithConnectedFn(fctx.RelayClient.IsConnected)
	}
	if fctx.DispositionResolver != nil {
		svc = svc.WithRelayGate(fctx.DispositionResolver)
	}
	// The domain service speaks installation records; the gRPC adapter presents
	// it over the protobuf contract that ECE and the core service expect.
	return coregrpc.NewCEBootstrapAdapter(svc), nil
}

// NewLocalDBOrgResolver creates the local DB-backed org resolver with caching
// that reads the process clock.
func NewLocalDBOrgResolver(
	orgRepo interfaces.OrgDirectoryRepository,
	log logger.Logger,
	cacheTTL time.Duration,
	maxEntries int,
) (org.Resolver, error) {
	return orgresolver.New(orgRepo, log, cacheTTL, maxEntries, clock.SystemClock{})
}
