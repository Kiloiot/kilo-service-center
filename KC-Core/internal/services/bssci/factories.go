package bssciservices

import (
	"sync"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// BSSCIServiceBundle packages all BSSCI service dependencies
type BSSCIServiceBundle struct {
	SessionSvc          bssci.SessionService
	VersionNegotiator   bssci.VersionNegotiator
	DownlinkSvc         bssci.DownlinkService
	StatusSvc           bssci.StatusService
	ConnectionSvc       bssci.BaseStationConnectionRegistry
	Broadcaster         SCACIForwarderWithSetter       // Unwired until SCACI exists; the typed setter is called by the composition root
	EPStatusBroadcaster SCACIEPStatusAdapterWithSetter // EPStatus adapter; wired the same way
	QueueSerializer     bssci.QueueSerializer
	AuditLogger         *DownlinkAuditLog
	TenantResolver      bssci.TenantResolver
	// ResultReporter tells a downlink's originators its final result.
	ResultReporter *DownlinkResultReporter
	// ServingStations decides the base station serving an endpoint.
	ServingStations *ServingStationPolicy
}

// DownlinkQueueStores is the downlink queue the composition root hands the
// bundle: the repository fills the downlink service's queue ports and the
// endpoint locations the serving station policy decides on.
type DownlinkQueueStores interface {
	DownlinkOutcomeWriter
	DownlinkRevocationWriter
	DownlinkHolderWriter
	EndpointLocator
	DownlinkLookup
}

// NewBSSCIServices creates all BSSCI services with explicit dependencies
func NewBSSCIServices(
	sessionRepo BaseStationSessionStore,
	baseStationRepo BaseStationStore,
	pendingOpsRepo PendingOperationStore,
	downlinkRepo DownlinkQueueStores,
	systemEventStore SystemEventRecorder,
	queueStore DownlinkQueueOwnership,
	connectionMgr *basestation.ConnectionManager,
	log logger.Logger,
	tenantID int64,
	serviceCenterEUI uint64,
	pendingOps *map[bssci.SessionOpKey]*bssci.PendingOperation,
	pendingOpsMu *sync.RWMutex,
	supportedProtocolVersions []string,
	clk clock.Clock,
	mqttResults DownlinkResultPublisher,
	work BackgroundRunner,
	revokeNotHeldCodes []int,
) (*BSSCIServiceBundle, error) {
	// Create services in dependency order
	versionNegotiator, err := NewVersionNegotiator(supportedProtocolVersions, log)
	if err != nil {
		return nil, err
	}

	// SessionService uses repository interfaces
	sessionSvc := NewSessionService(
		sessionRepo,
		pendingOpsRepo,
		systemEventStore,
		tenantID,
		serviceCenterEUI,
		log,
	)
	// StatusService uses PendingOperationRepository
	statusSvc := NewStatusService(pendingOps, pendingOpsMu, pendingOpsRepo, log)
	connectionSvc := NewConnectionRegistry(connectionMgr, log)
	queueSerializer := NewQueueSerializer()
	auditLogger, err := NewAuditLogger(AuditLogDeps{
		Events: systemEventStore, Downlinks: downlinkRepo, Stations: baseStationRepo, Clock: clk, Logger: log,
	})
	if err != nil {
		return nil, err
	}
	tenantResolver := NewTenantResolver(queueStore)

	// Create broadcaster WITHOUT SCACI server wired
	// Will be wired via SetSCACIServer() after SCACI creation (Step 4.4)
	broadcaster := NewSCACIForwarder(log)

	// Create EPStatus adapter WITHOUT SCACI server wired
	// Will be wired via SetSCACIServer() on the adapter after SCACI creation
	epStatusBroadcaster := NewSCACIEPStatusAdapter(log)

	// The broadcaster reaches Application Centers once the SCACI server is wired.
	resultReporter, err := NewDownlinkResultReporter(broadcaster, mqttResults, auditLogger, auditLogger, work, log)
	if err != nil {
		return nil, err
	}
	servingStations, err := NewServingStationPolicy(downlinkRepo)
	if err != nil {
		return nil, err
	}
	revokeAnswers, err := NewRevokeAnswers(RevokeAnswerDeps{
		Logger: log, Tenants: tenantResolver, Revocations: downlinkRepo, Expiries: resultReporter,
		Serializer: queueSerializer, NotHeldCodes: revokeNotHeldCodes,
	})
	if err != nil {
		return nil, err
	}
	downlinkSvc, err := NewDownlinkService(DownlinkServiceDeps{
		Logger: log, Tenants: tenantResolver, Outcomes: downlinkRepo, Holders: downlinkRepo,
		Results: resultReporter, Serializer: queueSerializer, Clock: clk,
	}, revokeAnswers)
	if err != nil {
		return nil, err
	}

	return &BSSCIServiceBundle{
		SessionSvc:          sessionSvc,
		VersionNegotiator:   versionNegotiator,
		DownlinkSvc:         downlinkSvc,
		StatusSvc:           statusSvc,
		ConnectionSvc:       connectionSvc,
		Broadcaster:         broadcaster,         // Return for later SetSCACIServer call
		EPStatusBroadcaster: epStatusBroadcaster, // Return for later SetSCACIServer call
		QueueSerializer:     queueSerializer,
		AuditLogger:         auditLogger,
		TenantResolver:      tenantResolver,
		ResultReporter:      resultReporter,
		ServingStations:     servingStations,
	}, nil
}
