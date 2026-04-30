package bssciservices

import (
	"sync"

	"github.com/kilocenter/KC-Core/pkg/basestation"
	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/org"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/postgres"
)

// BSSCIServiceBundle packages all BSSCI service dependencies
type BSSCIServiceBundle struct {
	SessionSvc          bssci.SessionService
	DownlinkSvc         bssci.DownlinkService
	StatusSvc           bssci.StatusService
	ConnectionSvc       bssci.ConnectionService
	Broadcaster         bssci.SCACIBroadcaster         // Initially unwired - set via SetSCACIServer
	EPStatusBroadcaster bssci.SCACIEPStatusBroadcaster // EPStatus adapter - set via SetSCACIEPStatusServer
	QueueSerializer     bssci.QueueSerializer
	AuditLogger         bssci.AuditLogger
	TenantResolver      bssci.TenantResolver
}

// BSSCIInfrastructure holds infrastructure that bssci.Server still accesses directly
// COMPLETE list based on current bssci.NewServer signature
// Note: KeyEncryptor and DeduplicatorSeed are NOT included - server creates them internally
type BSSCIInfrastructure struct {
	ConnectionMgr    *basestation.ConnectionManager
	Storage          interfaces.Storage          // Uses repository interfaces
	SystemEventStore interfaces.SystemEventStore // For event recording
	BasestationRepo  interfaces.BaseStationRepository
	EndpointRepo     interfaces.EndpointRepository
	PendingOps       *map[bssci.SessionOpKey]*bssci.PendingOperation
	PendingOpsMu     *sync.RWMutex
	OrgResolver      org.Resolver
	FallbackTenantID int64
}

// NewBSSCIServices creates all BSSCI services with explicit dependencies
func NewBSSCIServices(
	storage interfaces.Storage,
	systemEventStore interfaces.SystemEventStore,
	queueStore *postgres.DownlinkQueueReader,
	log logger.Logger,
	tenantID int64,
	orgResolver org.Resolver,
	pendingOps *map[bssci.SessionOpKey]*bssci.PendingOperation,
	pendingOpsMu *sync.RWMutex,
) *BSSCIServiceBundle {
	// Create services in dependency order
	// SessionService uses repository interfaces
	sessionSvc := NewSessionService(
		storage.BaseStationSessions(),
		storage.BaseStations(),
		systemEventStore,
		tenantID,
		log,
	)
	// StatusService uses PendingOperationRepository
	statusSvc := NewStatusService(pendingOps, pendingOpsMu, storage.PendingOperations(), log)
	connectionSvc := NewConnectionService(log)
	queueSerializer := NewQueueSerializer()
	auditLogger := NewAuditLogger(systemEventStore)
	tenantResolver := NewTenantResolver(queueStore)

	// Create broadcaster WITHOUT SCACI server wired
	// Will be wired via SetSCACIServer() after SCACI creation (Step 4.4)
	broadcaster := NewSCACIForwarder(log)

	// Create EPStatus adapter WITHOUT SCACI server wired
	// Will be wired via SetSCACIServer() on the adapter after SCACI creation
	epStatusBroadcaster := NewSCACIEPStatusAdapter(log)

	// DownlinkService uses interfaces.Storage with repository accessors
	downlinkSvc := NewDownlinkService(
		log,
		queueStore,
		tenantResolver,
		orgResolver,
		storage,
		broadcaster, // Unwired initially - SCACI server injected later
		auditLogger,
		queueSerializer,
	)

	return &BSSCIServiceBundle{
		SessionSvc:          sessionSvc,
		DownlinkSvc:         downlinkSvc,
		StatusSvc:           statusSvc,
		ConnectionSvc:       connectionSvc,
		Broadcaster:         broadcaster,         // Return for later SetSCACIServer call
		EPStatusBroadcaster: epStatusBroadcaster, // Return for later SetSCACIServer call
		QueueSerializer:     queueSerializer,
		AuditLogger:         auditLogger,
		TenantResolver:      tenantResolver,
	}
}
