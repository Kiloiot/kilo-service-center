package builders

import (
	federationservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/federation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
)

// defaultFederationWirer wires CE-only federation services (bootstrap/onboarding).
func defaultFederationWirer(fctx *FederationContext) (*FederationResult, error) {
	ceInstallRepo := postgres.NewCEInstallationRepository(fctx.SqlxDB)
	ceBootstrapSvc := federationservices.NewCEBootstrapService(ceInstallRepo, fctx.LoggerIface, fctx.Edition)
	if fctx.RelayClient != nil {
		ceBootstrapSvc = ceBootstrapSvc.
			WithRelayController(fctx.RelayClient).
			WithConnectedFn(fctx.RelayClient.IsConnected)
	}
	if fctx.DispositionResolver != nil {
		ceBootstrapSvc = ceBootstrapSvc.WithRelayGate(fctx.DispositionResolver)
	}
	return &FederationResult{
		BootstrapHandler: ceBootstrapSvc,
	}, nil
}
