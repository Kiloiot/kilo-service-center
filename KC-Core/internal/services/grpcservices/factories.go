package grpcservices

import (
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
)

// GRPCServiceBundle packages all gRPC service dependencies.
// Note: SystemStatusService is wired directly in main.go alongside other
// optional services (BlueprintService, IntegrationService) that have
// dependencies on external systems (registry, certificates) not available here.
type GRPCServiceBundle struct {
	EndpointSvc      EndpointService
	BaseStationSvc   BaseStationService
	DownlinkListings *DownlinkListingService
}

// NewGRPCServices creates all gRPC services with explicit dependencies.
func NewGRPCServices(
	endpointStore EndpointStore,
	baseStationStore BaseStationStore,
	results DownlinkResultsStore,
	queue DownlinkQueueLister,
	endpointIndex EndpointIndex,
	window RegistrationWindow,
	protocolCfg *config.ProtocolConfig,
) *GRPCServiceBundle {
	return &GRPCServiceBundle{
		EndpointSvc:      NewEndpointService(endpointStore, endpointIndex),
		BaseStationSvc:   NewBaseStationService(baseStationStore, protocolCfg),
		DownlinkListings: NewDownlinkListingService(results, queue, window),
	}
}
