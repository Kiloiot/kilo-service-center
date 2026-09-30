package builders

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	federationadapters "github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/federation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/health"
	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	blueprintresolver "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci/blueprint"
	federationservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/federation"
	grpcservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	scaciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/sessionreconcile"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/workers/delivery"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/workers/downlinkexpiry"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	pkgfederation "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/federation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/roaming"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/adapters"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-MQTT/pkg/mqtt"
	pkgversion "github.com/Kiloiot/kilo-service-center/pkg/version"
)

// BSSCIInfrastructure carries the repositories and resolver that the BSSCI and
// SCACI servers read directly, wired from the concrete storage layer.
type BSSCIInfrastructure struct {
	SystemEventStore interfaces.SystemEventStore
	BasestationRepo  interfaces.BaseStationRepository
	EndpointRepo     interfaces.EndpointRepository
	OrgResolver      org.Resolver
	FallbackTenantID int64
}

// ProtocolServers holds BSSCI and SCACI server instances and related resources.
type ProtocolServers struct {
	BSSCIServer         *bssci.Server
	BSSCIServices       *bssciservices.BSSCIServiceBundle
	SCACIServer         *scaci.Server // always built; scaci_enabled gates only its listener
	ServiceCenterEUI    uint64
	SoftwareVersion     string
	Cleanups            []func()
	RelayClient         federationservices.RelayController // nil unless CE mode with federation enabled
	DispositionResolver federationservices.RelayGate       // nil unless CE mode with federation enabled
	EndpointIndex       grpcservices.EndpointIndex         // keeps the disposition index in sync with endpoint CRUD
	AttachmentDecider   bssci.AttachmentDecider
	StatusNotifier      bssciservices.EndpointStatusNotifier
	Propagation         *bssciservices.AttachmentPropagation
}

// BuildProtocolServers constructs and starts BSSCI and (optionally) SCACI servers.
// Handles two-step BSSCI init: server first, then propagation/dispatcher injection.
func BuildProtocolServers(ctx context.Context, infra *Infrastructure) (*ProtocolServers, error) {
	log := logger.Get()
	cfg := infra.Config
	var cleanups []func()

	// Validate BSSCI configuration (BSSCI §1 enforcement)
	if err := pkgconfig.ValidateServiceCenterConfig(&cfg.Protocol); err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgBSSCIConfigurationInvalid, err)
	}

	log.Info(LogInitializingMandatoryBSSCIServer)

	// Service Center EUI is resolved and validated during config load (pkg/config Load)
	serviceCenterEUI := cfg.Protocol.SCEUIValue
	if cfg.Protocol.SCEUILegacyEnvUsed {
		log.WarnContext(ctx, pkgconfig.LogDeprecatedServiceCenterEUIEnv, logger.FieldScEui, cfg.Protocol.SCEUI)
	}

	// Resolve software version from release manifest with config fallback
	softwareVersion := infra.VersionInfo.Version
	if softwareVersion == "" || softwareVersion == pkgversion.DevVersion || softwareVersion == pkgversion.DevLocalVersion {
		if cfg.General.SoftwareVersion != "" {
			softwareVersion = cfg.General.SoftwareVersion
			log.Warn(LogUsingSoftwareVersionFromConfigFallback, logger.FieldVersion, softwareVersion)
		} else {
			softwareVersion = pkgversion.DevVersion
			log.Warn(LogUsingDefaultDevelopmentVersion, logger.FieldVersion, softwareVersion)
		}
	}

	bssciConfig := &bssci.Config{
		ListenAddr:                       fmt.Sprintf("%s:%d", cfg.Protocol.BSCIHost, cfg.Protocol.BSCIPort),
		TLSCert:                          cfg.Protocol.BSCITLS.CertFile,
		TLSKey:                           cfg.Protocol.BSCITLS.KeyFile,
		TLSCACert:                        cfg.Protocol.BSCITLS.CAFile,
		TLSMinVersion:                    cfg.Protocol.BSCITLS.MinVersion,
		ServiceCenterEUI:                 serviceCenterEUI,
		Vendor:                           cfg.Protocol.SCVendor,
		Model:                            cfg.Protocol.SCModel,
		Name:                             cfg.General.ServerName,
		SoftwareVersion:                  softwareVersion,
		OrgEnforcementEnabled:            cfg.General.OrgEnforcementEnabled,
		MessageEncoding:                  cfg.Protocol.MessageEncoding,
		DetachSignatureValidationEnabled: cfg.Protocol.DetachSignatureValidationEnabled,
		OperationAckTimeout:              time.Duration(cfg.Protocol.AckTimeout) * time.Millisecond,
		ConnectionEstablishmentTimeout:   time.Duration(cfg.Protocol.ConnectionEstablishmentTimeout) * time.Millisecond,
		SocketWriteTimeout:               time.Duration(cfg.Protocol.SocketWriteTimeout) * time.Millisecond,
		CertificatePollInterval:          cfg.Protocol.BSCICertificatePollInterval,
		StatusRequestInterval:            time.Duration(cfg.Protocol.StatusRequestInterval) * time.Second,
		StatusRequestInitialDelay:        time.Duration(cfg.Protocol.StatusRequestInitialDelay) * time.Second,
		DLRXQueryTimeout:                 time.Duration(cfg.Protocol.DLRXQueryTimeout) * time.Second,
		DLRXCleanupInterval:              time.Duration(cfg.Protocol.DLRXCleanupInterval) * time.Second,
	}

	// Create BSSCI service bundles
	log.Info(LogInitializingBSSCIServiceDependencies)

	// Shared pendingOps map using SessionOpKey composite key (BSSCI §5.11-5.12.3)
	pendingOps := make(map[bssci.SessionOpKey]*bssci.PendingOperation)
	var pendingOpsMu sync.RWMutex

	// MQTT event publisher for outbound device events (optional)
	var mqttAdapter bssci.MQTTEventPublisher
	var mqttResults bssciservices.DownlinkResultPublisher = bssciservices.DownlinkResultsWithoutMQTT{}
	var mqttAttachmentEvents bssciservices.AttachmentEventPublisher
	if infra.MQTTClient != nil {
		mqttPub := mqtt.NewPublisher(infra.MQTTClient, cfg.MQTT.TopicPrefix)
		mqttPublisher := bssciservices.NewMQTTAdapter(mqttPub)
		mqttAdapter = mqttPublisher
		mqttResults = mqttPublisher
		mqttAttachmentEvents = mqttPublisher
		log.Info(LogMQTTEventPublisherWiredToBSSCIServer)
	}

	// One background runner per process: the downlink results still being
	// delivered finish before shutdown completes.
	backgroundWork := bssciservices.NewBackgroundWork()

	bssciSvcBundle, err := bssciservices.NewBSSCIServices(
		infra.Repos.BaseStationSessions,
		infra.Repos.BaseStations,
		infra.Repos.PendingOperations,
		infra.Repos.Downlinks,
		infra.SystemEventStore,
		infra.Repos.DownlinkQueueReader,
		infra.ConnectionMgr,
		infra.LoggerIface,
		infra.TenantID,
		serviceCenterEUI,
		&pendingOps,
		&pendingOpsMu,
		[]string{mioty.MIOTYProtocolVersion},
		infra.Clock,
		mqttResults,
		backgroundWork,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildBSSCIServices, err)
	}

	bssciInfra := &BSSCIInfrastructure{
		SystemEventStore: infra.SystemEventStore,
		BasestationRepo:  infra.Repos.BaseStations,
		EndpointRepo:     infra.Repos.Endpoints,
		OrgResolver:      infra.OrgResolverSvc,
		FallbackTenantID: infra.TenantID,
	}

	// Initialize roaming service based on configuration
	var roamingSvc bssci.RoamingService
	if cfg.Protocol.Roaming.Enabled {
		log.Info(LogRoamingENABLEDInitializingRealService)
		detector, err := bssciservices.NewRoamingDetector(roamingDetectorConfig(cfg.Protocol.Roaming),
			infra.Repos.Roaming, infra.Repos.Roaming, infra.Clock)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildRoamingDetector, err)
		}
		roamingSvc = bssciservices.NewRoamingService(detector, infra.Repos.Roaming, infra.LoggerIface)
		log.Info(LogRoamingServiceInitialized,
			logger.FieldCacheEnabled, cfg.Protocol.Roaming.CacheEnabled,
			logger.FieldCacheTTL, cfg.Protocol.Roaming.CacheTTL,
			logger.FieldCacheMaxSize, cfg.Protocol.Roaming.CacheMaxSize)
	} else {
		log.Info(LogRoamingDisabled)
	}

	// Ingress disposition resolver: in CE mode with federation enabled,
	// unknown endpoints are relayed; otherwise dropped. Relay starts disabled
	// and is enabled at runtime once onboarding completes. The index is
	// pre-warmed from the store; a failed enumeration falls back to lazy
	// warming on cache-miss confirms.
	dispositionResolver := federationservices.NewDispositionResolver(bssciInfra.EndpointRepo, infra.Repos.Endpoints, false)
	if err := dispositionResolver.LoadFromDB(ctx); err != nil {
		log.Warn(LogEndpointIndexPrewarmFailed, logger.Err(err))
	}
	log.Info(LogIngressDispositionResolverWired, logger.FieldEdition, cfg.General.Edition)

	// Blueprint resolver and decoder for automatic payload decoding on uplinks
	resolverSvc := blueprintresolver.NewResolverService(infra.LoggerIface, infra.Repos.Blueprints)
	decoderSvc := blueprintresolver.NewDecoderService(infra.LoggerIface)

	attachments, err := buildAttachmentDecisions(infra, bssciInfra, bssciSvcBundle.EPStatusBroadcaster, mqttAttachmentEvents, backgroundWork)
	if err != nil {
		return nil, err
	}

	endpointOwners, err := bssciservices.NewEndpointOwnerResolver(bssciInfra.EndpointRepo)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildBSSCIServices, err)
	}

	// Shared uplink ingest service: tenant resolution, decoding, then one
	// transactional persist that classifies duplicates and enqueues delivery.
	// The delivery worker drains exactly the channels the ingest queues.
	channels := deliveryChannels(infra)
	endpointAcks, err := bssciservices.NewEndpointAckRecorder(infra.Repos.Downlinks, bssciSvcBundle.AuditLogger, infra.LoggerIface)
	if err != nil {
		return nil, err
	}
	uplinkIngestSvc, err := bssciservices.NewUplinkIngestService(
		infra.Repos.UplinkStore,
		uplinkWindows(cfg),
		channels,
		infra.Repos.DLRXStatus,
		bssciInfra.OrgResolver,
		roamingSvc,
		bssciInfra.EndpointRepo,
		endpointOwners,
		resolverSvc,
		decoderSvc,
		endpointAcks,
		infra.LoggerIface,
		infra.TenantID,
		0, // syntheticFederationBsEUI: zero until ECE federation-ingress is configured
	)
	if err != nil {
		return nil, err
	}

	// Detach signature validation is disabled by default: the MIOTY spec does
	// not define the detach CMAC construction, so no authoritative validator
	// ships with the community edition. Enabling the flag requires injecting a
	// real validator; without one the server refuses to start rather than
	// shipping a lookalike check with no cryptographic value.
	var detachValidator bssci.DetachSignatureValidator
	if cfg.Protocol.DetachSignatureValidationEnabled {
		return nil, errors.New(errMsgDetachValidationRequiresValidator)
	}
	log.Info(LogDetachSignatureValidationDisabled)

	// CE federation relay outbox writer (feature-controlled; the relay client
	// itself is wired after Start alongside onboarding)
	var relayOutboxWriter bssci.RelayOutboxWriter
	if cfg.General.Edition == pkgconfig.EditionCommunity && cfg.Protocol.Federation.Enabled {
		relayOutboxWriter = federationservices.NewOutboxWriter(
			infra.Repos.FederationOutbox, infra.LoggerIface,
		)
	}

	sessionReconciler, err := sessionreconcile.New(infra.Repos.BaseStationSessions, serviceCenterEUI,
		bssci.LogBSSCIReconciledAbandonedSessions, infra.LoggerIface)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildBSSCIServices, err)
	}

	sessionKeys, err := bssciservices.NewNetworkSessionKeySource(bssciInfra.EndpointRepo, infra.Repos.EndpointSessions)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildBSSCIServices, err)
	}

	attachPersistence, err := bssciservices.NewEndpointAttachmentPersistence(
		endpointSessionTxBridge{run: adapters.NewEndpointSessionTransactionAdapter(infra.Storage).Run}, bssciInfra.BasestationRepo, infra.Clock, log,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildBSSCIServices, err)
	}

	stationCertificates, err := bssci.NewStationCertificateBinder(
		bssciservices.NewRegisteredBaseStationDirectory(bssciInfra.BasestationRepo), infra.LoggerIface)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildBSSCIServices, err)
	}

	bssciServer, err := bssci.NewServer(bssciConfig, infra.LoggerIface, bssci.Dependencies{
		Clock: infra.Clock,
		Protocol: bssci.ProtocolServices{
			Session:            bssciSvcBundle.SessionSvc,
			VersionNegotiator:  bssciSvcBundle.VersionNegotiator,
			Downlink:           bssciSvcBundle.DownlinkSvc,
			Status:             bssciSvcBundle.StatusSvc,
			ConnectionRegistry: bssciSvcBundle.ConnectionSvc,
			QueueSerializer:    bssciSvcBundle.QueueSerializer,
			AuditLogger:        bssciSvcBundle.AuditLogger,
			TenantResolver:     bssciSvcBundle.TenantResolver,
			SessionReconciler:  sessionReconciler,
			ServingStations:    bssciSvcBundle.ServingStations,
			AttachmentDecider:  attachments.decider,
			StationEvents:      infra.EventRecorder,
		},
		Storage: bssci.StorageContracts{
			Events:            bssciInfra.SystemEventStore,
			BaseStations:      bssciInfra.BasestationRepo,
			Endpoints:         bssciInfra.EndpointRepo,
			EndpointOwners:    endpointOwners,
			AttachPersistence: attachPersistence,
			SessionKeys:       sessionKeys,
			ProtocolMessages:  infra.Repos.Messages,
			DLRXStatus:        infra.Repos.DLRXStatus,
			BaseStationStatus: infra.Repos.BaseStationStatus,
			DownlinkQueue:     infra.Repos.Downlinks,
			DownlinkRevoke:    infra.Repos.Downlinks,
			PendingDownlinks:  infra.Repos.Downlinks,
		},
		Identity: bssci.IdentityResolvers{
			OrgDirectory: bssciInfra.OrgResolver,
			// Certificate identity: the CE composite resolver handles EUI CNs
			// against the registered stations and delegates org-<UUID> CNs to the
			// deployment's org resolver
			CertIdentity: bssciservices.NewCertificateIdentityResolver(
				bssciInfra.BasestationRepo,
				bssciInfra.OrgResolver,
				infra.LoggerIface,
			),
			StationCertificates: stationCertificates,
			Cipher:              infra.Cipher,
		},
		Ingest: bssci.IngestPipeline{
			Uplink:            uplinkIngestSvc,
			Disposition:       dispositionResolver,
			BlueprintDecoder:  decoderSvc,
			BlueprintResolver: resolverSvc,
		},
		Features: bssci.FeatureCollaborators{
			Roaming:         roamingSvc,
			RelayOutbox:     relayOutboxWriter,
			DetachValidator: detachValidator,
			MQTT:            mqttAdapter,
		},
		TenantID:        infra.TenantID,
		DefaultTenantID: bssciInfra.FallbackTenantID,
	})
	if err != nil {
		return nil, fmt.Errorf(errFmtFailedToCreateBSSCIServer, err)
	}

	// Circular dependencies constructed against the live server, injected once
	// before Start
	attachmentPropagation, err := bssciservices.NewAttachmentPropagation(bssciServer, sessionKeys,
		infra.Repos.SystemEvents, backgroundWork, infra.Clock, infra.LoggerIface)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildBSSCIServices, err)
	}

	propagationSvc := bssciservices.NewPropagationService(
		bssciInfra.EndpointRepo,
		bssciServer,
		infra.LoggerIface,
	)
	downlinkDispatcher, err := bssciservices.NewDownlinkDispatcher(
		infra.LoggerIface,
		adapters.NewDownlinkReservationAdapter(infra.Storage),
		infra.Repos.Downlinks,
		infra.Repos.Messages,
		bssciServer.SendDLDataQueue,
		infra.Clock,
	)
	if err != nil {
		return nil, err
	}
	downlinkReclaimer, err := bssciservices.NewDownlinkReclaimer(infra.Repos.Downlinks, bssciSvcBundle.AuditLogger, infra.LoggerIface)
	if err != nil {
		return nil, err
	}
	if err := bssciServer.ConfigureRuntime(bssci.RuntimeDependencies{
		Propagation:        propagationSvc,
		DownlinkDispatcher: downlinkDispatcher,
		DownlinkReclaimer:  downlinkReclaimer,
	}); err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToConfigureBSSCIServerRuntime, err)
	}
	log.Info(LogDownlinkAutoDispatchEnabled)

	if err := bssciServer.Start(); err != nil {
		return nil, fmt.Errorf(errFmtFailedToStartBSSCIServer, err)
	}
	infra.statusBoard.track(statusNameBSSCI, pkgconfig.ListenerProbeAddress(cfg.Protocol.BSCIHost, cfg.Protocol.BSCIPort),
		health.NewListenerChecker(bssciServer))

	log.Info(LogBSSCIServerStartedSuccessfully,
		logger.FieldListenAddr, bssciConfig.ListenAddr,
		logger.FieldServiceCenterURL, infra.CanonicalSCURL,
		logger.FieldTLSMinVersion, cfg.Protocol.BSCITLS.MinVersion,
		logger.FieldTLSEnabled, cfg.Protocol.BSCITLS.Enabled,
		logger.FieldSpecCompliance, specComplianceBSSCI)

	// Forwarded downlink results drain only once BSSCI can start no more.
	cleanups = append(cleanups, func() {
		if err := bssciServer.Stop(); err != nil {
			log.Error(LogFailedStopBSSCIServer, logger.Err(err))
		}
		if err := backgroundWork.Stop(context.WithoutCancel(ctx)); err != nil {
			log.Error(LogFailedWaitDownlinkWork, logger.Err(err))
		}
	})

	// The SCACI server is always constructed: it carries the transport-neutral
	// downlink queueing core that gRPC and MQTT delegate to. scaci_enabled
	// gates only the external SCACI socket listener inside buildSCACIServer.
	scaciServer, scaciErr := buildSCACIServer(ctx, infra, bssciServer, bssciSvcBundle, bssciInfra, propagationSvc, attachments.decider, serviceCenterEUI)
	if scaciErr != nil {
		return nil, scaciErr
	}
	if cfg.Protocol.SCACIEnabled {
		infra.statusBoard.track(statusNameSCACI, pkgconfig.ListenerProbeAddress(cfg.Protocol.SCACIHost, cfg.Protocol.SCACIPort),
			health.NewListenerChecker(scaciServer))
	}

	cleanups = append(cleanups, func() {
		if err := scaciServer.Stop(); err != nil {
			log.Error(LogFailedStopSCACIServer, logger.Err(err))
		}
	})

	// Delivery worker: drains the outbox rows the ingest service enqueues,
	// fanning each stored uplink out to SCACI sessions and MQTT.
	deliveryWorker, err := buildDeliveryWorker(infra, channels, bssciSvcBundle.Broadcaster, mqttAdapter, bssciInfra.SystemEventStore)
	if err != nil {
		return nil, err
	}
	cleanups = append(cleanups, deliveryWorker.Start(ctx))
	log.Info(LogDeliveryWorkerStarted, logger.FieldChannels, channels)

	// Downlink expiry worker: expires the downlinks that outlived
	// protocol.downlink_expiry.lifetime in the queue and reports them.
	expiryWorker, err := buildDownlinkExpiryWorker(infra, bssciSvcBundle, bssciServer)
	if err != nil {
		return nil, err
	}
	cleanups = append(cleanups, expiryWorker.Start(ctx))
	log.Info(LogDownlinkExpiryWorkerStarted)

	// Wire MQTT command/down subscriber if enabled
	if infra.MQTTClient != nil && cfg.MQTT.EnableCommandSubscriptions {
		downlinkQueuer, err := bssciservices.NewSCACIDownlinkQueuer(scaciServer)
		if err != nil {
			return nil, err
		}
		downlinkAdapter, err := bssciservices.NewMQTTDownlinkAdapter(downlinkQueuer)
		if err != nil {
			return nil, err
		}
		cmdHandler := mqtt.NewCommandHandler(infra.MQTTClient, downlinkAdapter, infra.OrgResolverSvc, infra.LoggerIface, cfg.MQTT.TopicPrefix)
		go cmdHandler.Start(ctx)
		log.Info(LogMQTTCommandDownSubscriberStarted)
	}

	// Wire CE federation relay client and outbox (CE mode only)
	var protoRelayClient federationservices.RelayController
	var protoRelayGate federationservices.RelayGate
	if cfg.General.Edition == pkgconfig.EditionCommunity && cfg.Protocol.Federation.Enabled {
		outboxRepo := infra.Repos.FederationOutbox
		installRepo := infra.Repos.CEInstallations

		relayClient := federationadapters.NewRelayClient(
			cfg.Protocol.Federation,
			installRepo,
			outboxRepo,
			infra.LoggerIface,
		).WithCEVersion(infra.VersionInfo.Version).
			WithBsCountFn(func(bsCtx context.Context) int32 {
				stats, statsErr := infra.Repos.BaseStations.GetStatistics(bsCtx, infra.TenantID)
				if statsErr != nil {
					return 0
				}
				return int32(stats.OnlineCount) //nolint:gosec
			})

		protoRelayClient = relayClient
		protoRelayGate = dispositionResolver

		// Gate relay on onboarding completion: check DB at startup
		onboardingDone := false
		if inst, instErr := installRepo.Get(ctx); instErr == nil && inst != nil && inst.OnboardingCompletedAt != nil {
			onboardingDone = true
		}
		dispositionResolver.SetRelayEnabled(onboardingDone)

		if onboardingDone {
			if err := relayClient.EnsureStarted(ctx); err != nil {
				if errors.Is(err, pkgfederation.ErrRelayOnboardingIncomplete) {
					log.Info(LogCEOnboardingNotCompleteRelayDeferred)
				} else {
					log.Warn(LogFederationRelayClientCouldNotStart, logger.FieldError, err)
				}
			} else {
				log.Info(LogCEFederationRelayClientStarted, logger.FieldEceEndpoint, cfg.Protocol.Federation.ECEEndpoint)
			}
		} else {
			log.Info(LogCEOnboardingNotCompleteRelayDeferred)
		}

		cleanups = append(cleanups, func() { relayClient.Stop() })
	}

	return &ProtocolServers{
		BSSCIServer:         bssciServer,
		EndpointIndex:       dispositionResolver,
		BSSCIServices:       bssciSvcBundle,
		SCACIServer:         scaciServer,
		ServiceCenterEUI:    serviceCenterEUI,
		SoftwareVersion:     softwareVersion,
		Cleanups:            cleanups,
		RelayClient:         protoRelayClient,
		DispositionResolver: protoRelayGate,
		AttachmentDecider:   attachments.decider,
		StatusNotifier:      attachments.notifier,
		Propagation:         attachmentPropagation,
	}, nil
}

// buildSessionState builds the owner of the SCACI session rows, whose fresh
// sessions are created in a transaction that retires the earlier ones, and
// the registry of the sessions with the holder of the resumable ones that have
// no connection (SCACI §1). The sessions a previous process left live are
// disconnected first: no connection holds them any more.
func buildSessionState(ctx context.Context, infra *Infrastructure, serviceCenterEUI uint64) (*scaciservices.SessionRows, *scaci.SessionRegistry, error) {
	if err := reconcileSCACISessions(ctx, infra.Repos.SCACISessions, serviceCenterEUI, infra.LoggerIface); err != nil {
		return nil, nil, err
	}
	rows, err := scaciservices.NewSessionRows(
		scaciSessionTxBridge{run: adapters.NewSCACISessionTransactionAdapter(infra.Storage).Run},
		infra.Repos.SCACISessions, infra.Repos.SCACISessions, serviceCenterEUI)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToBuildSessionPersistence, err)
	}
	holder, err := buildResumeHolder(ctx, infra, rows)
	if err != nil {
		return nil, nil, err
	}
	registry, err := scaci.NewSessionRegistry(holder, rows, infra.LoggerIface)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errMsgFailedToBuildSessionRegistry, err)
	}
	return rows, registry, nil
}

// reconcileSCACISessions returns the sessions a previous process of this
// service center left live to the resumable disconnected state, before any
// Application Center connects.
func reconcileSCACISessions(ctx context.Context, sessions sessionreconcile.AbandonedSessionStore, serviceCenterEUI uint64, log logger.Logger) error {
	reconciler, err := sessionreconcile.New(sessions, serviceCenterEUI, scaci.LogSCACIReconciledAbandonedSessions, log)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgFailedToReconcileSCACISessions, err)
	}
	reconcileCtx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()
	if err := reconciler.ReconcileAbandonedSessions(reconcileCtx); err != nil {
		return fmt.Errorf("%s: %w", errMsgFailedToReconcileSCACISessions, err)
	}
	return nil
}

// buildResumeHolder builds the holder of the resumable SCACI sessions without
// a connection and holds those an earlier run left resumable (SCACI §1).
func buildResumeHolder(ctx context.Context, infra *Infrastructure, rows scaciservices.HeldSessionRows) (*scaciservices.ResumeHolder, error) {
	holder, err := scaciservices.NewResumeHolder(rows, infra.Repos.SCACIOperations,
		infra.Config.Protocol.SCACIResumeMaxPendingOperations, infra.LoggerIface)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildResumeHolder, err)
	}
	loadCtx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()
	if err := holder.Load(loadCtx, infra.Repos.SCACISessions); err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildResumeHolder, err)
	}
	return holder, nil
}

// buildSCACIServer constructs and starts the SCACI server, wiring BSSCI→SCACI forwarding.
func buildSCACIServer(
	ctx context.Context,
	infra *Infrastructure,
	bssciServer *bssci.Server,
	bssciSvcBundle *bssciservices.BSSCIServiceBundle,
	bssciInfra *BSSCIInfrastructure,
	propagationSvc propagation.Service,
	attachmentDecider bssci.AttachmentDecider,
	serviceCenterEUI uint64,
) (*scaci.Server, error) {
	log := logger.Get()
	cfg := infra.Config

	// Validate SCACI config before use (SCACI §1 compliance)
	if err := pkgconfig.ValidateSCACIConfig(&cfg.Protocol); err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgSCACIConfigurationInvalid, err)
	}
	log.Info(LogInitializingSCACIServer)

	scaciConfig := &scaci.Config{
		ListenAddr:            fmt.Sprintf("%s:%d", cfg.Protocol.SCACIHost, cfg.Protocol.SCACIPort),
		TLS:                   cfg.Protocol.SCACITLS,
		ServiceCenterEUI:      serviceCenterEUI,
		Vendor:                cfg.Protocol.SCVendor,
		Model:                 cfg.Protocol.SCModel,
		Name:                  cfg.General.ServerName,
		SoftwareVersion:       infra.VersionInfo.Version,
		OrgEnforcementEnabled: cfg.General.OrgEnforcementEnabled,
		LogPingOperations:     cfg.Protocol.SCALogPingOperations,
		LogStatusOperations:   cfg.Protocol.SCALogStatusOperations,

		ConnectionEstablishmentTimeout: time.Duration(cfg.Protocol.ConnectionEstablishmentTimeout) * time.Millisecond,
		SocketWriteTimeout:             time.Duration(cfg.Protocol.SocketWriteTimeout) * time.Millisecond,
		PlatformTenantID:               infra.TenantID,
	}

	scaciSessionRepo := infra.Repos.SCACISessions

	if err := guardSCACIOrgResolution(&cfg.Protocol, infra.OrgResolverSvc, log); err != nil {
		return nil, err
	}

	scaciSvcBundle, err := scaciservices.NewSCACIServices(
		scaciSessionRepo,
		infra.Repos.SCACIOperations,
		infra.Repos.Endpoints,
		infra.Repos.BaseStations,
		infra.Repos.Downlinks,
		bssciSvcBundle.AuditLogger,
		infra.Repos.SCACIEvents,
		bssciServer,
		attachmentDecider,
		infra.DownlinkQueueIDs,
		cfg.Protocol.DownlinkExpiry.Lifetime,
		infra.LoggerIface,
		infra.OrgResolverSvc,
		infra.TenantID,
		cfg.Protocol.StrictOrgResolution,
		scaciConfig.ServiceCenterEUI,
		scaciConfig.Vendor,
		scaciConfig.Model,
		scaciConfig.Name,
		scaciConfig.SoftwareVersion,
		infra.ServiceStart,
		infra.Clock,
	)
	if err != nil {
		return nil, err
	}
	rows, registry, err := buildSessionState(ctx, infra, serviceCenterEUI)
	if err != nil {
		return nil, err
	}

	scaciServer, err := scaci.NewServer(scaciConfig, infra.LoggerIface, scaci.Dependencies{
		Registry:     registry,
		Operations:   infra.Repos.SCACIOperations,
		Handshake:    scaciSvcBundle.HandshakeSvc,
		Endpoints:    scaciSvcBundle.EndpointSvc,
		UL:           scaciSvcBundle.ULSvc,
		DL:           scaciSvcBundle.DLSvc,
		Status:       scaciSvcBundle.StatusSvc,
		Validator:    scaciSvcBundle.SessionValidator,
		Recorder:     scaciSvcBundle.OperationRecorder,
		Persistence:  rows,
		OrgDirectory: bssciInfra.OrgResolver, // BSSCI/SCACI org context parity
		Snapshots:    bssciServer,
		Propagation:  propagationSvc, // BSSCI §5.8-5.8.3
		Errors:       scaciSvcBundle.ErrorRecorder,
		Clock:        infra.Clock,

		SessionEvents: infra.Repos.SCACIEvents,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToCreateSCACIServer, err)
	}
	log.Info(LogSCACIErrorRecorderWired)

	// scaci_enabled gates only the external socket listener. The application
	// core (downlink queueing for gRPC and MQTT, BSSCI forwarding) stays wired
	// either way; with no listener there are no AC sessions to broadcast to,
	// so forwarding is a no-op until the listener is enabled.
	if err := startAndWireSCACI(bssciSvcBundle, scaciServer, cfg.Protocol.SCACIEnabled); err != nil {
		return nil, err
	}

	if cfg.Protocol.SCACIEnabled {
		log.Info(LogSCACIServerStartedSuccessfully,
			logger.FieldHost, cfg.Protocol.SCACIHost,
			logger.FieldPort, cfg.Protocol.SCACIPort,
			logger.FieldSpecCompliance, specComplianceSCACI)
	} else {
		log.Warn(LogSCACIListenerDisabled)
	}
	log.Info(LogBSSCIToSCACIForwardingEnabled)
	log.Info(LogBSSCIToSCACIEPStatusAdapterWired)

	return scaciServer, nil
}

// guardSCACIOrgResolution refuses a SCACI listener whose strict org
// resolution has no resolver to map certificates to tenants (SCACI §1
// isolation); the configuration rules themselves are validated at load.
// Without a listener there is no application center peer to resolve.
func guardSCACIOrgResolution(protocol *pkgconfig.ProtocolConfig, resolver org.Resolver, log logger.Logger) error {
	if !protocol.SCACIEnabled {
		return nil
	}
	if protocol.StrictOrgResolution {
		if resolver == nil {
			return errors.New(errMsgStrictOrgResolutionNilOrgResolver)
		}
		log.Info(LogSCACIStrictOrgResolutionEnabled)
		return nil
	}
	if protocol.SCACICertTenantMapping && resolver == nil {
		log.Warn(LogSCACICertTenantMappingNilOrgResolver)
	}
	return nil
}

// scaciRuntime is the started-server surface the SCACI wiring step needs: the
// lifecycle Start plus both broadcaster surfaces the BSSCI forwarders relay
// onto. The typed setters mean a server that stops satisfying a broadcaster
// surface fails to compile instead of leaving forwarding silently
// disconnected.
type scaciRuntime interface {
	Start() error
	bssciservices.SCACIServerBroadcaster
	bssciservices.SCACIEPStatusServerBroadcaster
}

// startAndWireSCACI validates every BSSCI→SCACI wiring point, starts the
// socket listener when enabled, and applies both forwarding setters. Nothing
// starts while a wiring point is missing, and no setter runs unless Start
// succeeded, so a partially wired SCACI server can never serve traffic.
func startAndWireSCACI(bundle *bssciservices.BSSCIServiceBundle, server scaciRuntime, listenerEnabled bool) error {
	if bundle == nil {
		return errors.New(errMsgSCACIWiringBSSCIServiceBundleIsNil)
	}
	if bundle.Broadcaster == nil {
		return errors.New(errMsgSCACIForwardingBroadcasterIsNil)
	}
	if bundle.EPStatusBroadcaster == nil {
		return errors.New(errMsgSCACIEPStatusForwardingEPStatusBroadcasterIsNil)
	}
	if server == nil {
		return errors.New(errMsgSCACIWiringServerIsNil)
	}
	if listenerEnabled {
		if err := server.Start(); err != nil {
			return fmt.Errorf("%s: %w", errMsgFailedToStartSCACIServer, err)
		}
	}
	// Wire BSSCI→SCACI forwarding (uplink data + DL results, then EPStatus per
	// SCACI §3.13). The BSSCI server already holds the bundle's EPStatus
	// broadcaster as a constructor dependency; the setter activates it.
	bundle.Broadcaster.SetSCACIServer(server)
	bundle.EPStatusBroadcaster.SetSCACIServer(server)
	return nil
}

// BuildFederationIngestDeps constructs a fully-wired UplinkIngestService for use by the
// federation-ingress binary. It wires all real collaborators (org resolver, blueprint resolver,
// blueprint decoder, uplink store) so the ingress binary shares identical ingest behaviour.
func BuildFederationIngestDeps(_ context.Context, infra *Infrastructure) (*bssciservices.UplinkIngestServiceImpl, error) {
	resolverSvc := blueprintresolver.NewResolverService(
		infra.LoggerIface,
		infra.Repos.Blueprints,
	)
	decoderSvc := blueprintresolver.NewDecoderService(infra.LoggerIface)

	cfg := infra.Config
	syntheticEUI := syntheticBsEUI(cfg.Protocol.Federation.SyntheticBsEUI, infra.Log)

	downlinkEvents, err := bssciservices.NewAuditLogger(bssciservices.AuditLogDeps{
		Events: infra.SystemEventStore, Downlinks: infra.Repos.Downlinks, Stations: infra.Repos.BaseStations,
		Clock: infra.Clock, Logger: infra.LoggerIface,
	})
	if err != nil {
		return nil, err
	}
	endpointAcks, err := bssciservices.NewEndpointAckRecorder(infra.Repos.Downlinks, downlinkEvents, infra.LoggerIface)
	if err != nil {
		return nil, err
	}
	endpointOwners, err := bssciservices.NewEndpointOwnerResolver(infra.Repos.Endpoints)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildBSSCIServices, err)
	}
	return bssciservices.NewUplinkIngestService(
		infra.Repos.UplinkStore,
		uplinkWindows(cfg),
		deliveryChannels(infra),
		infra.Repos.DLRXStatus,
		infra.OrgResolverSvc,
		nil, // federation ingress resolves endpoint directly; no roaming service needed
		infra.Repos.Endpoints,
		endpointOwners,
		resolverSvc,
		decoderSvc,
		endpointAcks,
		infra.LoggerIface,
		infra.TenantID,
		syntheticEUI,
	)
}

// uplinkWindows converts the validated protocol settings into the windows the
// uplink store applies: same-counter receptions (seconds) and the wait for
// the other base stations' receptions before delivery.
func uplinkWindows(cfg *pkgconfig.Config) bssciservices.UplinkWindows {
	return bssciservices.UplinkWindows{
		Duplicate: time.Duration(cfg.Protocol.DuplicateWindow) * time.Second,
		Reception: cfg.Protocol.Delivery.ReceptionWindow,
	}
}

// deliveryChannels lists the outbox channels every stored uplink is fanned
// out to: SCACI always, MQTT when the deployment enables it. It reads the
// configuration, not this process's client, because another process may
// drain the rows.
func deliveryChannels(infra *Infrastructure) []models.DeliveryChannel {
	channels := []models.DeliveryChannel{models.DeliveryChannelSCACI}
	if infra.Config.MQTT.Enabled {
		channels = append(channels, models.DeliveryChannelMQTT)
	}
	return channels
}

// roamingDetectorConfig carries the protocol.roaming settings into the roaming detector.
func roamingDetectorConfig(cfg pkgconfig.RoamingConfig) roaming.DetectorConfig {
	return roaming.DetectorConfig{
		CacheEnabled:     cfg.CacheEnabled,
		CacheTTL:         cfg.CacheTTL,
		CacheMaxSize:     cfg.CacheMaxSize,
		EnableAuditTrail: cfg.EnableAuditTrail,
	}
}

// buildDeliveryWorker wires the outbox drain loop with a sender for every
// channel the ingest queues rows for.
func buildDeliveryWorker(infra *Infrastructure, channels []models.DeliveryChannel, scaci delivery.SCACIBroadcaster,
	mqttAdapter bssci.MQTTEventPublisher, events delivery.EventRecorder,
) (*delivery.Worker, error) {
	cfg := infra.Config.Protocol.Delivery
	retry, err := delivery.NewRetryPolicy(cfg.RetryBackoff, cfg.MaxBackoff)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildDeliveryWorker, err)
	}
	senders, err := deliverySenders(channels, scaci, mqttAdapter)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildDeliveryWorker, err)
	}
	worker, err := delivery.NewWorker(delivery.Dependencies{
		Outbox:   infra.Repos.DeliveryOutbox,
		Outcomes: infra.Repos.DeliveryOutbox,
		Messages: infra.Repos.Messages,
		Channels: senders,
		Events:   events,
		Clock:    infra.Clock,
		Logger:   infra.LoggerIface,
	}, delivery.Config{PollInterval: cfg.PollInterval, BatchSize: cfg.BatchSize, Retry: retry})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildDeliveryWorker, err)
	}
	return worker, nil
}

// deliverySenders binds a sender to every queued channel, so no queued row
// waits for a channel this process cannot drain.
func deliverySenders(channels []models.DeliveryChannel, scaci delivery.SCACIBroadcaster,
	mqttAdapter bssci.MQTTEventPublisher,
) (delivery.Channels, error) {
	var senders delivery.Channels
	for _, channel := range channels {
		switch {
		case channel == models.DeliveryChannelSCACI && scaci != nil:
			senders.SCACI = scaci
		case channel == models.DeliveryChannelMQTT && mqttAdapter != nil:
			senders.MQTT = mqttAdapter
		default:
			return delivery.Channels{}, fmt.Errorf(errFmtDeliveryChannelWithoutSender, channel)
		}
	}
	return senders, nil
}

// buildDownlinkExpiryWorker wires the sweep that expires overdue downlinks,
// has the downlink result reporter tell their originators and the BSSCI
// server revoke the ones a base station still holds.
func buildDownlinkExpiryWorker(infra *Infrastructure, bundle *bssciservices.BSSCIServiceBundle, revoker downlinkexpiry.StationRevoker) (*downlinkexpiry.Worker, error) {
	cfg := infra.Config.Protocol.DownlinkExpiry
	deps := downlinkexpiry.Dependencies{
		Queue:        infra.Repos.Downlinks,
		Reporter:     bundle.ResultReporter,
		Revoker:      revoker,
		QueueTenants: bundle.TenantResolver,
		Logger:       infra.LoggerIface,
	}
	worker, err := downlinkexpiry.NewWorker(deps, downlinkexpiry.Config{Interval: cfg.SweepInterval, BatchSize: cfg.BatchSize})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildDownlinkExpiryWorker, err)
	}
	return worker, nil
}

// syntheticBsEUI parses the configured federation base station EUI; an empty
// setting means the feature is unconfigured and a malformed one is logged and
// ignored so the ingress binary still starts.
func syntheticBsEUI(configured string, log logger.Logger) uint64 {
	if configured == "" {
		return 0
	}
	v, err := validation.ParseEUI(configured)
	if err != nil {
		log.Warn(LogInvalidSyntheticBsEuiFormat, logger.FieldValue, configured, logger.FieldError, err)
		return 0
	}
	return v
}
