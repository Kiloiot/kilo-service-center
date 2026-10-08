package builders

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/keymaterial"
	"github.com/Kiloiot/kilo-service-center/pkg/version"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/activity"
	coreAdapters "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/adapters"
	alertsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/alerts"
	analyticsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/analytics"
	blueprintsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/blueprints"
	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	blueprintresolver "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci/blueprint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/capabilities"
	certificatesservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/certificates"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/diagnostics"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/downlinks"
	errorgroupsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/errorgroups"
	eventsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/events"
	grpcservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	integrationsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/integrations"
	messagesservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/messages"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/registrationscope"
	scacimonitoringservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/scaci_monitoring"
	statisticsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/statistics"
	systemstatusservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/systemstatus"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/workers/certcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/adapters"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
)

// CoreServiceSourceName identifies KC-Core as the source of system events.
const CoreServiceSourceName = "kc-core"

// CoreServiceDisplayName labels KC-Core in event titles.
const CoreServiceDisplayName = "KC-Core"

// listenInfoFmt summarizes the listening ports for the started event.
const listenInfoFmt = "gRPC=%d, BSSCI=%d, SCACI=%d"

// CoreResult holds the fully wired CoreService, the shared system event
// adapter and the stops of the background work the core services started.
type CoreResult struct {
	Service            *grpc.CoreService
	SystemEventAdapter grpcservices.EventWriter
	Cleanups           []func()
}

// The gRPC service layer consumes narrow ports; *postgres.DB satisfies each of
// them. Asserting that here keeps the services package free of any dependency on
// the concrete store.
var (
	_ grpcservices.EndpointStore        = (*postgres.EndPointRepository)(nil)
	_ grpcservices.BaseStationStore     = (*postgres.BaseStationRepository)(nil)
	_ grpcservices.DownlinkResultsStore = (*postgres.MIOTYDownlinkRepository)(nil)
	_ downlinks.PendingEditor           = (*postgres.MIOTYDownlinkRepository)(nil)

	_ grpc.DLRXStatusQueryStorage = (*postgres.DLRXStatusRepository)(nil)
	_ grpc.DLRXStatusReader       = (*postgres.DLRXStatusRepository)(nil)
	_ grpc.MessageStore           = (*postgres.MessageRepository)(nil)

	_ grpc.BaseStationAvailabilityReader  = (*postgres.BaseStationMetricsRepository)(nil)
	_ grpc.BaseStationMessageBucketReader = (*postgres.BaseStationMetricsRepository)(nil)
)

// BuildCoreService constructs the CoreService with all domain services wired via With* methods.
// Pass opts to override edition-specific federation wiring (nil uses CE defaults).
func BuildCoreService(ctx context.Context, infra *Infrastructure, protocol *ProtocolServers, opts *Options) *CoreResult {
	log := logger.Get()
	cfg := infra.Config

	log.Info(LogInitializingGRPCServiceLayer)

	// Create grpcservices bundle for gRPC service integration
	registrationWindow := registrationscope.New(infra.Repos.Endpoints)
	grpcSvcBundle := grpcservices.NewGRPCServices(
		infra.Repos.Endpoints,
		grpcservices.BaseStationServiceDeps{
			Store: infra.Repos.BaseStations, Protocol: &cfg.Protocol,
			Sessions: protocol.BSSCIServer, Downlinks: protocol.DownlinkReclaimer,
		},
		infra.Repos.Downlinks,
		infra.Repos.DownlinkQueueReader,
		protocol.EndpointIndex,
		registrationWindow,
	)

	// Extract Service Center identity for gRPC GetReleaseInfo
	scVendor := cfg.Protocol.SCVendor
	scModel := cfg.Protocol.SCModel
	hostname := ServiceHostname(ctx, log)
	scName := cfg.General.ServerName
	if scName == "" {
		scName = hostname
	}

	// Wire BSSCI session closer for EUI change handling

	// Wire base station event recorder for EUI change events

	// Wire base station time-series metrics readers (availability + received messages)

	// Wire system event recorder for CRUD event emissions
	systemEventAdapter := coreAdapters.NewSystemEventStoreAdapter(infra.Repos.SystemEvents)
	log.Info(LogEventWriterWired)

	emitServiceStarted(ctx, log, systemEventAdapter, cfg, infra, hostname)

	// ScaciMonitoringService
	scaciMonitoringDeps := scacimonitoringservice.Deps{
		Sessions:     infra.Repos.SCACISessions,
		Operations:   infra.Repos.SCACIOperations,
		Queue:        infra.Repos.DownlinkQueueReader,
		Listener:     protocol.SCACIServer,
		ServiceStart: infra.ServiceStart,
		SCEui:        protocol.ServiceCenterEUI,
		Clock:        infra.Clock,
		Log:          infra.LoggerIface,
	}
	scaciMonitoringSvc := scacimonitoringservice.New(scaciMonitoringDeps)
	log.Info(LogScaciMonitoringServiceWired)

	auditRecorder := buildAuditRecorder(infra, log)

	// CertificateService: repository and encryptor are mandatory constructor
	// inputs (ownership verification and persistence are part of issuance)
	certificateSvc, certErr := certificatesservice.New(ctx, cfg, infra.LoggerIface, infra.Repos.BaseStations,
		keymaterial.NewCipherKeyEncryptor(infra.Cipher), certificatesservice.ExecCertGen, infra.Clock, auditRecorder)
	if certErr != nil {
		log.Fatal(LogFailedCreateCoreService, logger.Err(certErr))
	}
	certificateCleanup, certErr := certcleanup.NewWorker(certificateSvc, infra.Clock,
		time.Duration(cfg.Certificates.CleanupIntervalMin)*time.Minute)
	if certErr != nil {
		log.Fatal(LogFailedCreateCoreService, logger.Err(certErr))
	}
	stopCertificateCleanup := certificateCleanup.Start(ctx)
	log.Info(LogCertificateServiceWired)

	// BlueprintService
	blueprintSvc := blueprintsservice.New(
		infra.Repos.Manufacturers,
		infra.Repos.DeviceModels,
		infra.Repos.Blueprints,
		infra.TenantID,
		&cfg.RegistryProvider,
		infra.LoggerIface,
		blueprintTxBridge{run: adapters.NewBlueprintTransactionAdapter(infra.Storage).Run},
		blueprintresolver.NewDecoderService(infra.LoggerIface),
	)
	log.Info(LogBlueprintServiceWired)

	// IntegrationService
	integrationSvc := integrationsservice.New(
		infra.Repos.Integrations,
		infra.LoggerIface,
	)
	log.Info(LogIntegrationServiceWired)

	// EndpointStatsStore
	log.Info(LogEndpointStatsStoreWired)

	// OperationStatusAdapter
	opStatusAdapter := coreAdapters.NewOperationStatusAdapter(infra.Repos.OperationStatus)
	log.Info(LogOperationStatusAdapterWired)

	// SystemStatusService
	endpointURLs := make([]systemstatusservice.EndpointURL, 0, len(infra.statusBoard.rows))
	for _, row := range infra.statusBoard.rows {
		endpointURLs = append(endpointURLs, systemstatusservice.EndpointURL{Name: row.Name, URL: row.URL})
	}
	systemStatusSvc := systemstatusservice.New(
		infra.Repos.BaseStations,
		infra.Repos.Endpoints,
		infra.Repos.MessageAnalytics,
		infra.LoggerIface,
		infra.HealthService,
		endpointURLs,
	)
	log.Info(LogSystemStatusServiceWired)

	// Cache the heavy COUNT(*) so dashboard polling stays off Postgres.
	eventStoreAdapter := coreAdapters.NewSystemEventStoreAdapter(infra.Repos.SystemEvents)
	eventStore, countCacheErr := coreAdapters.NewCachedSystemEventStore(eventStoreAdapter, cfg.GRPC.CountCacheTTL, config.CountCacheComputeTimeout, infra.Clock)
	if countCacheErr != nil {
		log.Fatal(LogFailedCreateCoreService, logger.Err(countCacheErr))
	}
	euiResolver := coreAdapters.NewEUIResolver(infra.Repos.BaseStations, infra.Repos.Endpoints)
	wakes, stopStreamWakes := startStreamWakes(ctx, cfg.Storage, infra.LoggerIface)

	eventsSvc := eventsservice.New(
		eventStore,
		euiResolver,
		registrationWindow,
		cfg.GRPC.StreamPollInterval,
		cfg.GRPC.StreamOverlap,
		wakes.events,
		cfg.GRPC.StreamBatchSize,
		infra.LoggerIface,
	)
	log.Info(LogEventServiceWired)

	// AlertService
	alertsSvc := alertsservice.New(
		coreAdapters.NewAlertStoreAdapter(
			infra.Repos.SystemEvents,
			cfg.Alerts.RecentAlertsLimit,
		),
		infra.LoggerIface,
	)
	log.Info(LogAlertServiceWired)

	// AnalyticsService
	analyticsSvc := analyticsservice.New(
		coreAdapters.NewAnalyticsMessageStoreAdapter(infra.Repos.MessageAnalytics),
		infra.LoggerIface,
		infra.Clock,
	)
	log.Info(LogAnalyticsServiceWired)

	// MessageListingService
	messagesSvc := messagesservice.New(
		coreAdapters.NewMessageListingStoreAdapter(infra.Repos.Messages),
		registrationWindow,
		cfg.GRPC.StreamPollInterval,
		cfg.GRPC.StreamOverlap,
		wakes.uplinks,
		cfg.GRPC.StreamBatchSize,
		infra.LoggerIface,
	)
	log.Info(LogMessageListingServiceWired)

	// StatisticsService
	statisticsSvc := statisticsservice.New(
		infra.Repos.Endpoints,
		infra.Repos.BaseStations,
		infra.Repos.MessageAnalytics,
		infra.LoggerIface,
	)
	log.Info(LogStatisticsServiceWired)

	// ActivityService
	activitySvc := activity.New(eventsSvc, messagesSvc, infra.LoggerIface)
	log.Info(LogActivityServiceWired)

	// OrgMapper names each station's organization in the cross-tenant location API
	var orgMapper grpc.OrgMapper
	if infra.IdentityInternalClient != nil {
		orgMapper = grpc.NewAdminOrgAdapter(infra.IdentityInternalClient, cfg.InternalAuth.PeerSecret)
		log.Info(LogOrgMapperWired)
	}

	// EndpointAttachmentService
	endpointAttachmentSvc, err := bssciservices.NewEndpointAttachmentService(
		infra.Repos.Endpoints,
		grpcSvcBundle.EndpointSvc,
		protocol.AttachmentDecider,
		protocol.StatusNotifier,
		protocol.Propagation,
	)
	if err != nil {
		log.Fatal(LogEndpointAttachmentServiceCreateFailed, logger.Err(err))
	}
	log.Info(LogEndpointAttachmentServiceWired)

	// Federation services (edition-specific wiring)
	federationFn := defaultFederationWirer
	if opts != nil && opts.FederationWirer != nil {
		federationFn = opts.FederationWirer
	}
	fctx := &FederationContext{
		Storage:         infra.Storage,
		CEInstallations: infra.Repos.CEInstallations,
		LoggerIface:     infra.LoggerIface,
		Clock:           infra.Clock,
		Edition:         cfg.General.Edition,
	}
	if protocol.RelayClient != nil {
		fctx.RelayClient = protocol.RelayClient
	}
	if protocol.DispositionResolver != nil {
		fctx.DispositionResolver = protocol.DispositionResolver
	}
	var federation grpc.FederationHandlerDeps
	fedResult, fedErr := federationFn(fctx)
	if fedErr != nil {
		log.Error(LogFailedToWireFederationServices, logger.Err(fedErr))
	} else {
		federation.Bootstrap = fedResult.BootstrapHandler
		federation.Registry = fedResult.RegistryHandler
	}

	downlinkCommands, err := downlinks.NewService(downlinks.Deps{
		Queuer:    protocol.SCACIServer,
		Editor:    infra.Repos.Downlinks,
		Revoker:   protocol.BSSCIServer,
		Endpoints: grpcSvcBundle.EndpointSvc,
		Events:    protocol.BSSCIServices.AuditLogger,
		Audit:     auditRecorder,
		Log:       infra.LoggerIface,
	})
	if err != nil {
		log.Fatal(LogDownlinkServiceCreateFailed, logger.Err(err))
	}
	coreService, err := grpc.NewCoreService(grpc.CoreServiceDeps{
		Log:   infra.LoggerIface,
		Audit: auditRecorder,
		Endpoints: grpc.EndpointHandlerDeps{
			KeyReveals:      auditRecorder,
			Endpoints:       grpcSvcBundle.EndpointSvc,
			Blueprints:      blueprintSvc,
			Attachment:      endpointAttachmentSvc,
			Stats:           infra.Repos.MessageAnalytics,
			Registrations:   registrationWindow,
			Operations:      opStatusAdapter,
			ActivityWindow:  time.Duration(cfg.General.ActivityWindowHours) * time.Hour,
			Clock:           infra.Clock,
			ServingStations: protocol.BSSCIServices.ServingStations,
		},
		BaseStations: grpc.BaseStationHandlerDeps{
			BaseStations:   grpcSvcBundle.BaseStationSvc,
			Stats:          infra.Repos.Messages,
			StatusReq:      protocol.BSSCIServer,
			Ping:           protocol.BSSCIServer,
			Sessions:       protocol.BSSCIServer,
			SessionCloser:  protocol.BSSCIServer,
			EventRecorder:  infra.EventRecorder,
			Availability:   infra.Repos.BaseStationMetrics,
			MessageBuckets: infra.Repos.BaseStationMetrics,
			Orgs:           orgMapper,
		},
		Downlinks: grpc.DownlinkHandlerDeps{
			Commands: downlinkCommands,
			Queue:    grpcSvcBundle.DownlinkListings,
			Results:  grpcSvcBundle.DownlinkListings,
		},
		ULTransmit: grpc.ULTransmitHandlerDeps{
			Sessions:     protocol.BSSCIServer,
			Transmitter:  protocol.BSSCIServer,
			BaseStations: grpcSvcBundle.BaseStationSvc,
		},
		DLRX: grpc.DLRXHandlerDeps{
			Queries:   infra.Repos.DLRXStatus,
			Statuses:  infra.Repos.DLRXStatus,
			Stations:  protocol.BSSCIServices.ServingStations,
			Commander: protocol.BSSCIServer,
			Sessions:  protocol.BSSCIServer,
		},
		Messages:     grpc.MessageHandlerDeps{Listing: messagesSvc, Activity: activitySvc},
		Analytics:    grpc.AnalyticsHandlerDeps{Analytics: analyticsSvc, Events: eventsSvc, Alerts: alertsSvc, ErrorGroups: errorgroupsservice.New(infra.Repos.SystemEvents, infra.Clock, infra.LoggerIface)},
		Certificates: grpc.CertificateHandlerDeps{Certificates: certificateSvc, PlatformTenantID: infra.TenantID},
		Blueprints:   grpc.BlueprintHandlerDeps{Blueprints: blueprintSvc, Endpoints: grpcSvcBundle.EndpointSvc},
		Integrations: grpc.IntegrationHandlerDeps{Integrations: integrationSvc},
		Scaci:        grpc.ScaciHandlerDeps{Monitoring: scaciMonitoringSvc},
		Federation:   federation,
		System: grpc.SystemHandlerDeps{
			StartedAt:    infra.ServiceStart,
			Statistics:   statisticsSvc,
			SystemStatus: systemStatusSvc,
			SCEui:        protocol.ServiceCenterEUI,
			SCVendor:     scVendor,
			SCModel:      scModel,
			SCName:       scName,
			SCSwVersion:  protocol.SoftwareVersion,
			Edition:      cfg.General.Edition,
			Capabilities: capabilities.FromConfig(cfg),
			Diagnostics: diagnostics.New(diagnostics.Deps{
				Release:    version.Get,
				Config:     cfg,
				Events:     eventsSvc,
				SCACI:      infra.Repos.SCACISessions,
				BSSCI:      protocol.BSSCIServer,
				Limits:     diagnostics.Limits{MaxBundleBytes: config.DiagnosticsMaxBundleBytes, MaxEvents: config.DiagnosticsMaxEvents, MaxSessions: config.DiagnosticsMaxSessions, Timeout: config.DiagnosticsGenerationTimeout},
				Clock:      infra.Clock,
				Log:        infra.LoggerIface,
				ServerName: cfg.General.ServerName,
			}),
		},
	})
	if err != nil {
		log.Fatal(LogFailedCreateCoreService, logger.Err(err))
	}

	log.Info(LogGRPCServicesWiringComplete)

	return &CoreResult{
		Service:            coreService,
		SystemEventAdapter: systemEventAdapter,
		Cleanups:           []func(){stopCertificateCleanup, stopStreamWakes},
	}
}
