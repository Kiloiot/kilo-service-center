package grpcservices

import (
	"github.com/kilocenter/KC-Core/pkg/config"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/postgres"
)

// GRPCServiceBundle packages all gRPC service dependencies.
// Note: SystemStatusService is wired directly in main.go alongside other
// optional services (BlueprintService, IntegrationService) that have
// dependencies on external systems (registry, certificates) not available here.
type GRPCServiceBundle struct {
	EndpointSvc    EndpointService
	BaseStationSvc BaseStationService
	MessageSvc     MessageService
}

// NewGRPCServices creates all gRPC services with explicit dependencies.
func NewGRPCServices(
	storageLayer storage.Storage,
	postgresDB *postgres.DB,
	log logger.Logger,
	protocolCfg *config.ProtocolConfig,
) *GRPCServiceBundle {
	return &GRPCServiceBundle{
		EndpointSvc:    NewEndpointService(storageLayer, log),
		BaseStationSvc: NewBaseStationService(storageLayer, log, protocolCfg),
		MessageSvc:     NewMessageService(postgresDB, log),
	}
}
