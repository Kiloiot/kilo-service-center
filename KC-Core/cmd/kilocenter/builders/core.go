package builders

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/activity"
	coreAdapters "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/adapters"
	alertsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/alerts"
	analyticsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/analytics"
	blueprintsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/blueprints"
	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	blueprintresolver "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci/blueprint"
	certificatesservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/certificates"
	eventsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/events"
	grpcservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	integrationsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/integrations"
	messagesservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/messages"
	scacimonitoringservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/scaci_monitoring"
	statisticsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/statistics"
	systemstatusservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/systemstatus"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// CoreResult holds the fully wired CoreService and shared system event adapter.
type CoreResult struct {
	Service            *grpc.CoreService
	SystemEventAdapter grpcservices.EventWriter
}

// BuildCoreService constructs the CoreService with all domain services wired via With* methods.
// Pass opts to override edition-specific federation wiring (nil uses CE defaults).
func BuildCoreService(ctx context.Context, infra *Infrastructure, protocol *ProtocolServers, opts *Options) *CoreResult {
	log := logger.Get()
	cfg := infra.Config

	log.Info("Initializing gRPC service layer...")

	// Create grpcservices bundle for gRPC service integration
	grpcSvcBundle := grpcservices.NewGRPCServices(
		infra.Storage,
		infra.Storage,
		infra.LoggerIface,
		&cfg.Protocol,
	)

	// Extract Service Center identity for gRPC GetReleaseInfo
	scVendor := cfg.Protocol.SCVendor
	scModel := cfg.Protocol.SCModel
	scName := cfg.General.ServerName
	if scName == "" {
		hostname, _ := os.Hostname()
		scName = hostname
	}

	coreService, err := grpc.NewCoreService(grpc.CoreServiceDeps{
		EndpointSvc:       grpcSvcBundle.EndpointSvc,
		BasestationSvc:    grpcSvcBundle.BaseStationSvc,
		MessageSvc:        grpcSvcBundle.MessageSvc,
		DownlinkSvc:       protocol.BSSCIServices.DownlinkSvc,
		StatusSvc:         protocol.BSSCIServices.StatusSvc,
		DownlinkCmd:       protocol.BSSCIServer,
		DownlinkScheduler: protocol.BSSCIServer,
		SessionDir:        protocol.BSSCIServer,
		ULTransmit:        protocol.BSSCIServer,
		StatusReq:         protocol.BSSCIServer,
		PingCmd:           protocol.BSSCIServer,
		StatsStore:        infra.Storage,
		DownlinkStore:     infra.Storage,
		DLRXStorage:       infra.Storage,
		SCEui:             protocol.ServiceCenterEUI,
		SCVendor:          scVendor,
		SCModel:           scModel,
		SCName:            scName,
		SCSwVersion:       protocol.SoftwareVersion,
		Edition:           cfg.General.Edition,
	})
	if err != nil {
		log.Fatal(LogFailedCreateCoreService, logger.Err(err))
	}

	// Optional: Wire SCACI queuer when SCACI is configured
	if protocol.SCACIServer != nil {
		coreService = coreService.WithSCACIQueuer(protocol.SCACIServer)
	}

	// Wire BSSCI session closer for EUI change handling
	coreService = coreService.WithBSSCISessionCloser(protocol.BSSCIServer)

	// Wire base station event recorder for EUI change events
	coreService = coreService.WithBSEventRecorder(infra.EventRecorder)

	// Wire base station time-series metrics readers (availability + received messages)
	coreService = coreService.WithBaseStationMetricsReaders(infra.Storage, infra.Storage)

	// Wire system event recorder for CRUD event emissions
	systemEventAdapter := coreAdapters.NewSystemEventStoreAdapter(infra.Storage.SystemEvents())
	coreService = coreService.WithEventWriter(systemEventAdapter)
	log.Info("EventWriter wired")

	// Emit service started event
	hostname, _ := os.Hostname()
	listenInfo := fmt.Sprintf("gRPC=%d, BSSCI=%d, SCACI=%d",
		cfg.GRPC.Port, cfg.Protocol.BSCIPort, cfg.Protocol.SCACIPort)
	startDetails, _ := json.Marshal(map[string]interface{}{
		"service":   "kc-core",
		"version":   infra.VersionInfo.Version,
		"gitCommit": infra.VersionInfo.GitCommit,
		"host":      hostname,
		"ports": map[string]int{
			"grpc":  cfg.GRPC.Port,
			"bssci": cfg.Protocol.BSCIPort,
			"scaci": cfg.Protocol.SCACIPort,
		},
	})
	_ = systemEventAdapter.CreateEvent(ctx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(infra.TenantID, 10),
		EventType:   models.EventTypeServiceStarted,
		Category:    models.EventCategorySystem,
		Severity:    models.EventSeverityInfo,
		Title:       fmt.Sprintf(models.EventTitleServiceStartedFmt, "KC-Core", hostname),
		Description: fmt.Sprintf(models.EventDescriptionServiceStartedFmt, "KC-Core", infra.VersionInfo.Version, listenInfo),
		SourceType:  models.SourceTypeSystem,
		SourceName:  "kc-core",
		Details:     json.RawMessage(startDetails),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	log.Info("Service started event emitted")

	// ScaciMonitoringService
	scaciMonitoringSvc := scacimonitoringservice.New(
		infra.Storage.SCACISessions(),
		infra.Storage.SCACIOperations(),
		infra.QueueStore,
		infra.LoggerIface,
	)
	coreService = coreService.WithScaciMonitoringService(scaciMonitoringSvc)
	log.Info("ScaciMonitoringService wired")

	// CertificateService
	certificateSvc := certificatesservice.New(cfg, infra.LoggerIface).
		WithBaseStationRepo(infra.Storage.BaseStations()).
		WithKeyEncryptor(infra.KeyEncryptor)
	coreService = coreService.WithCertificateService(certificateSvc)
	log.Info("CertificateService wired")

	// BlueprintService
	blueprintSvc := blueprintsservice.New(
		infra.Storage.Manufacturers(),
		infra.Storage.DeviceModels(),
		infra.Storage.Blueprints(),
		infra.TenantID,
		&cfg.RegistryProvider,
		infra.LoggerIface,
	).WithTxStarter(infra.Storage).WithDecoder(blueprintresolver.NewDecoderService(infra.LoggerIface))
	coreService = coreService.WithBlueprintService(blueprintSvc)
	log.Info("BlueprintService wired")

	// IntegrationService
	integrationSvc := integrationsservice.New(
		infra.Storage.Integrations(),
		infra.LoggerIface,
	)
	coreService = coreService.WithIntegrationService(integrationSvc)
	log.Info("IntegrationService wired")

	// EndpointStatsStore
	coreService = coreService.WithEndpointStatsStore(infra.Storage.MIOTYMessages())
	log.Info("EndpointStatsStore wired")

	// OperationStatusAdapter
	opStatusAdapter := coreAdapters.NewOperationStatusAdapter(infra.Storage.OperationStatus())
	coreService = coreService.WithOperationStatusAdapter(opStatusAdapter)
	log.Info("OperationStatusAdapter wired")

	// SystemStatusService
	systemStatusSvc := systemstatusservice.New(
		infra.Storage.BaseStations(),
		infra.Storage.EndPoints(),
		infra.Storage.MIOTYMessages(),
		infra.LoggerIface,
	)

	var endpointURLs []systemstatusservice.EndpointURL
	for _, ep := range cfg.Status.Endpoints {
		url := ep.URL
		if url == "" && ep.Host != "" && ep.Port > 0 {
			url = ep.GetAddress()
		}
		endpointURLs = append(endpointURLs, systemstatusservice.EndpointURL{
			Name: ep.Name,
			URL:  url,
		})
	}
	systemStatusSvc = systemStatusSvc.WithHealthService(infra.HealthService, endpointURLs)
	coreService = coreService.WithSystemStatusService(systemStatusSvc)
	log.Info("SystemStatusService wired")

	// EventService
	eventsSvc := eventsservice.New(
		coreAdapters.NewSystemEventStoreAdapter(infra.Storage.SystemEvents()),
		coreAdapters.NewEUIResolver(infra.Storage.BaseStations(), infra.Storage.EndPoints()),
		cfg.GRPC.StreamPollInterval,
		cfg.GRPC.StreamBatchSize,
		infra.LoggerIface,
	)
	coreService = coreService.WithEventService(eventsSvc)
	log.Info("EventService wired")

	// AlertService
	alertsSvc := alertsservice.New(
		coreAdapters.NewAlertStoreAdapter(
			infra.Storage.SystemEvents(),
			cfg.Alerts.SummaryLookbackHours,
			cfg.Alerts.RecentAlertsLimit,
		),
		infra.LoggerIface,
	)
	coreService = coreService.WithAlertService(alertsSvc)
	log.Info("AlertService wired")

	// AnalyticsService
	analyticsSvc := analyticsservice.New(
		coreAdapters.NewAnalyticsMessageStoreAdapter(infra.Storage.MIOTYMessages()),
		infra.LoggerIface,
	)
	coreService = coreService.WithAnalyticsService(analyticsSvc)
	log.Info("AnalyticsService wired")

	// MessageListingService
	messagesSvc := messagesservice.New(
		coreAdapters.NewMessageListingStoreAdapter(infra.Storage.MIOTYMessages()),
		cfg.GRPC.StreamPollInterval,
		cfg.GRPC.StreamBatchSize,
		infra.LoggerIface,
	)
	coreService = coreService.WithMessageListingService(messagesSvc)
	log.Info("MessageListingService wired")

	// StatisticsService
	statisticsSvc := statisticsservice.New(
		infra.Storage.EndPoints(),
		infra.Storage.BaseStations(),
		infra.Storage.MIOTYMessages(),
		infra.LoggerIface,
	)
	coreService = coreService.WithStatisticsService(statisticsSvc)
	log.Info("StatisticsService wired")

	// ActivityService
	activitySvc := activity.New(eventsSvc, messagesSvc, infra.LoggerIface)
	coreService = coreService.WithActivityService(activitySvc)
	log.Info("ActivityService wired")

	// StreamPollInterval
	coreService = coreService.WithStreamPollInterval(cfg.GRPC.StreamPollInterval)
	log.Info("StreamPollInterval configured", "interval", cfg.GRPC.StreamPollInterval)

	// Endpoint activity window
	coreService = coreService.WithEndpointActivityWindow(
		time.Duration(cfg.General.ActivityWindowHours) * time.Hour,
	)

	// AdminChecker and OrgMapper for cross-tenant location API
	if infra.IdentityInternalClient != nil {
		adminOrgAdapter := grpc.NewAdminOrgAdapter(infra.IdentityInternalClient, cfg.InternalAuth.PeerSecret)
		coreService = coreService.WithAdminChecker(adminOrgAdapter).WithOrgMapper(adminOrgAdapter)
		log.Info("AdminChecker and OrgMapper wired")
	}

	// EndpointAttachmentService
	endpointAttachmentSvc := bssciservices.NewEndpointAttachmentService(
		infra.Storage.EndPoints(),
		infra.Storage.SystemEvents(),
		protocol.BSSCIServer,
		infra.LoggerIface,
	)
	coreService = coreService.WithEndpointAttachmentService(endpointAttachmentSvc)
	log.Info("EndpointAttachmentService wired")

	// Federation services (edition-specific wiring)
	federationFn := defaultFederationWirer
	if opts != nil && opts.FederationWirer != nil {
		federationFn = opts.FederationWirer
	}
	fctx := &FederationContext{
		SqlxDB:      infra.SqlxDB,
		LoggerIface: infra.LoggerIface,
		Edition:     cfg.General.Edition,
	}
	if protocol.RelayClient != nil {
		fctx.RelayClient = protocol.RelayClient
	}
	if protocol.DispositionResolver != nil {
		fctx.DispositionResolver = protocol.DispositionResolver
	}
	fedResult, fedErr := federationFn(fctx)
	if fedErr != nil {
		log.Error("Failed to wire federation services", logger.Err(fedErr))
	} else {
		if fedResult.BootstrapHandler != nil {
			coreService = coreService.WithCEBootstrapHandler(fedResult.BootstrapHandler)
		}
		if fedResult.RegistryHandler != nil {
			coreService = coreService.WithCERegistryHandler(fedResult.RegistryHandler)
		}
	}

	log.Info("gRPC services wiring complete")

	return &CoreResult{
		Service:            coreService,
		SystemEventAdapter: systemEventAdapter,
	}
}
