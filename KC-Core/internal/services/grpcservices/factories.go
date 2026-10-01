package grpcservices

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
	stations BaseStationServiceDeps,
	results DownlinkResultsStore,
	queue DownlinkQueueLister,
	endpointIndex EndpointIndex,
	window RegistrationWindow,
) *GRPCServiceBundle {
	return &GRPCServiceBundle{
		EndpointSvc:      NewEndpointService(endpointStore, endpointIndex),
		BaseStationSvc:   NewBaseStationService(stations),
		DownlinkListings: NewDownlinkListingService(results, queue, window),
	}
}
