package builders

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	blueprintresolver "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci/blueprint"
	federationservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/federation"
	scaciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-MQTT/pkg/mqtt"
)

// ProtocolServers holds BSSCI and SCACI server instances and related resources.
type ProtocolServers struct {
	BSSCIServer         *bssci.Server
	BSSCIServices       *bssciservices.BSSCIServiceBundle
	BSSCIInfra          *bssciservices.BSSCIInfrastructure
	SCACIServer         *scaci.Server // nil if disabled
	ServiceCenterEUI    uint64
	SoftwareVersion     string
	Cleanups            []func()
	RelayClient         federationservices.RelayController // nil unless CE mode with federation enabled
	DispositionResolver federationservices.RelayGate       // nil unless CE mode with federation enabled
}

// BuildProtocolServers constructs and starts BSSCI and (optionally) SCACI servers.
// Handles two-step BSSCI init: server first, then propagation/dispatcher injection.
func BuildProtocolServers(ctx context.Context, infra *Infrastructure) (*ProtocolServers, error) {
	log := logger.Get()
	cfg := infra.Config
	var cleanups []func()

	// Validate BSSCI configuration (BSSCI §1 enforcement)
	if err := pkgconfig.ValidateServiceCenterConfig(&cfg.Protocol); err != nil {
		return nil, fmt.Errorf("BSSCI configuration invalid: %w", err)
	}

	log.Info("Initializing mandatory BSSCI server...")

	// Service Center EUI is resolved and validated during config load (pkg/config Load)
	serviceCenterEUI := cfg.Protocol.SCEUIValue
	if cfg.Protocol.SCEUILegacyEnvUsed {
		log.WarnContext(ctx, pkgconfig.LogDeprecatedServiceCenterEUIEnv, "sc_eui", cfg.Protocol.SCEUI)
	}

	// Resolve software version from release manifest with config fallback
	softwareVersion := infra.VersionInfo.Version
	if softwareVersion == "" || softwareVersion == "dev" || softwareVersion == "dev-local" {
		if cfg.General.SoftwareVersion != "" {
			softwareVersion = cfg.General.SoftwareVersion
			log.Warn("Using software version from config fallback", "version", softwareVersion)
		} else {
			softwareVersion = "dev"
			log.Warn("Using default development version", "version", softwareVersion)
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
		DuplicateWindow:                  time.Duration(cfg.Protocol.DuplicateWindow) * time.Second,
		CertificatePollInterval:          cfg.Protocol.BSCICertificatePollInterval,
		StatusRequestInterval:            time.Duration(cfg.Protocol.StatusRequestInterval) * time.Second,
		StatusRequestInitialDelay:        time.Duration(cfg.Protocol.StatusRequestInitialDelay) * time.Second,
		DLRXQueryTimeout:                 time.Duration(cfg.Protocol.DLRXQueryTimeout) * time.Second,
		DLRXCleanupInterval:              time.Duration(cfg.Protocol.DLRXCleanupInterval) * time.Second,
	}

	// Create BSSCI service bundles
	log.Info("Initializing BSSCI service dependencies...")

	// Shared pendingOps map using SessionOpKey composite key (BSSCI §5.11-5.12.3)
	pendingOps := make(map[bssci.SessionOpKey]*bssci.PendingOperation)
	var pendingOpsMu sync.RWMutex

	bssciSvcBundle, err := bssciservices.NewBSSCIServices(
		infra.Storage,
		infra.SystemEventStore,
		infra.QueueStore,
		infra.ConnectionMgr,
		infra.LoggerIface,
		infra.TenantID,
		infra.OrgResolverSvc,
		&pendingOps,
		&pendingOpsMu,
		[]string{mioty.MIOTYProtocolVersion},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build BSSCI services: %w", err)
	}

	bssciInfra := &bssciservices.BSSCIInfrastructure{
		ConnectionMgr:    infra.ConnectionMgr,
		Storage:          infra.Storage,
		SystemEventStore: infra.SystemEventStore,
		BasestationRepo:  infra.BasestationRepo,
		EndpointRepo:     infra.Storage.EndPoints(),
		PendingOps:       &pendingOps,
		PendingOpsMu:     &pendingOpsMu,
		OrgResolver:      infra.OrgResolverSvc,
		FallbackTenantID: infra.TenantID,
	}

	// Two-step initialization: server first, then propagation service (BSSCI §5.8-5.8.3)
	bssciServer, err := bssci.NewServer(
		bssciConfig,
		infra.LoggerIface,
		bssciInfra.ConnectionMgr,
		bssciInfra.Storage,
		bssciInfra.SystemEventStore,
		bssciInfra.BasestationRepo,
		bssciInfra.EndpointRepo,
		infra.TenantID,
		bssciSvcBundle.SessionSvc,
		bssciSvcBundle.VersionNegotiator,
		bssciSvcBundle.DownlinkSvc,
		bssciSvcBundle.StatusSvc,
		bssciSvcBundle.ConnectionSvc,
		bssciSvcBundle.Broadcaster,
		bssciSvcBundle.QueueSerializer,
		bssciSvcBundle.AuditLogger,
		bssciSvcBundle.TenantResolver,
		bssciInfra.OrgResolver,
		bssciInfra.FallbackTenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create BSSCI server: %w (StatusService is mandatory for pending operation tracking)", err)
	}

	// Certificate identity: the CE composite resolver handles EUI CNs against
	// the registered stations and delegates org-<UUID> CNs to the deployment's
	// org resolver; the directory backs connect-time fingerprint enforcement
	bssciServer.SetCertificateIdentityResolver(bssciservices.NewCertificateIdentityResolver(
		bssciInfra.BasestationRepo,
		bssciInfra.OrgResolver,
		infra.LoggerIface,
	))
	bssciServer.SetBaseStationDirectory(bssciservices.NewRegisteredBaseStationDirectory(bssciInfra.BasestationRepo))

	// Create propagation service (depends on Server as AttachPropagateSender)
	propagationSvc := bssciservices.NewPropagationService(
		bssciInfra.EndpointRepo,
		bssciServer,
		infra.LoggerIface,
	)
	bssciServer.SetPropagationService(propagationSvc)

	// Create and inject downlink dispatcher for auto-dispatch on dlOpen
	downlinkDispatcher := bssciservices.NewDownlinkDispatcher(
		infra.LoggerIface,
		infra.Storage,
		bssciServer.SendDLDataQueue,
	)
	bssciServer.SetDownlinkDispatcher(downlinkDispatcher)
	log.Info("Downlink auto-dispatch enabled (BSSCI §5.10.2)")

	// Initialize roaming service based on configuration
	var roamingSvc bssci.RoamingService
	if cfg.Protocol.Roaming.Enabled {
		log.Info("Roaming ENABLED - initializing real service")
		roamingSvc = bssciservices.NewRoamingService(infra.Storage, true)
		log.Info("Roaming service initialized",
			"cache_enabled", cfg.Protocol.Roaming.CacheEnabled,
			"cache_ttl", cfg.Protocol.Roaming.CacheTTL,
			"cache_max_size", cfg.Protocol.Roaming.CacheMaxSize)
	} else {
		log.Info("Roaming DISABLED - using noop service")
		roamingSvc = bssciservices.NewNoopRoamingService()
	}
	bssciServer.SetRoamingService(roamingSvc)

	// Wire ingress disposition resolver.
	// In CE mode with federation enabled, unknown endpoints are relayed; otherwise they are dropped.
	// Relay is initially disabled; it is enabled at runtime once onboarding completes.
	dispositionResolver := federationservices.NewDispositionResolver(bssciInfra.EndpointRepo, false)
	bssciServer.SetDispositionResolver(dispositionResolver)
	log.Info("Ingress disposition resolver wired", "edition", cfg.General.Edition)

	// Wire shared uplink ingest service (dedup → tenant resolution → persist → SCACI → MQTT)
	uplinkIngestSvc := bssciservices.NewUplinkIngestService(
		bssciServer.GetDeduplicator(),
		bssciInfra.Storage,
		bssciInfra.OrgResolver,
		roamingSvc,
		bssciInfra.EndpointRepo,
		nil, // blueprintResolver injected after construction
		nil, // blueprintDecoder injected after construction
		bssciSvcBundle.Broadcaster,
		nil, // mqttPublisher injected after construction
		infra.LoggerIface,
		infra.TenantID,
		0, // syntheticFederationBsEUI: zero until ECE federation-ingress is configured
	)
	bssciServer.SetUplinkIngestService(uplinkIngestSvc)

	// Wire detach signature validator if enabled
	if cfg.Protocol.DetachSignatureValidationEnabled {
		log.Info("Detach signature validation ENABLED - initializing direct repository adapter")
		detachValidator := bssciservices.NewDetachValidatorDirectAdapter(
			infra.Storage.EndPoints(),
			log,
		)
		bssciServer.SetDetachValidator(detachValidator)
		log.Info("Detach validator wired to BSSCI server (direct repository mode)")
	} else {
		log.Info("Detach signature validation DISABLED - unknown endpoint detach will be rejected")
	}

	// Wire blueprint resolver and decoder for automatic payload decoding on uplinks
	resolverSvc := blueprintresolver.NewResolverService(infra.LoggerIface, infra.Storage.Blueprints(), infra.Storage.DeviceModels(), infra.Storage.EndPoints())
	bssciServer.SetBlueprintResolver(resolverSvc)
	decoderSvc := blueprintresolver.NewDecoderService(infra.LoggerIface)
	bssciServer.SetBlueprintDecoder(decoderSvc)
	uplinkIngestSvc.SetBlueprintResolver(resolverSvc)
	uplinkIngestSvc.SetBlueprintDecoder(decoderSvc)
	log.Info("Blueprint resolver and decoder wired to BSSCI server and ingest service")

	// Wire MQTT event publisher to BSSCI server for outbound device events
	if infra.MQTTClient != nil {
		mqttPub := mqtt.NewPublisher(infra.MQTTClient, cfg.MQTT.TopicPrefix)
		mqttAdapter := bssciservices.NewMQTTAdapter(mqttPub)
		bssciServer.SetMQTTPublisher(mqttAdapter)
		uplinkIngestSvc.SetMQTTPublisher(mqttAdapter)
		log.Info("MQTT event publisher wired to BSSCI server")
	}

	if err := bssciServer.Start(); err != nil {
		return nil, fmt.Errorf("failed to start mandatory BSSCI server: %w (cannot comply with MIOTY specification without TLS BSSCI endpoint)", err)
	}

	log.Info("BSSCI server started successfully",
		"listen_addr", bssciConfig.ListenAddr,
		"service_center_url", infra.CanonicalSCURL,
		"tls_min_version", cfg.Protocol.BSCITLS.MinVersion,
		"tls_enabled", cfg.Protocol.BSCITLS.Enabled,
		"spec_compliance", "MIOTY BSSCI v1.0.0")

	cleanups = append(cleanups, func() {
		if err := bssciServer.Stop(); err != nil {
			log.Error(LogFailedStopBSSCIServer, logger.Err(err))
		}
	})

	// Initialize SCACI server if enabled
	var scaciServer *scaci.Server
	if cfg.Protocol.SCACIEnabled {
		var scaciErr error
		scaciServer, scaciErr = buildSCACIServer(infra, bssciServer, bssciSvcBundle, bssciInfra, propagationSvc, serviceCenterEUI)
		if scaciErr != nil {
			return nil, scaciErr
		}

		cleanups = append(cleanups, func() {
			if err := scaciServer.Stop(); err != nil {
				log.Error(LogFailedStopSCACIServer, logger.Err(err))
			}
		})
	}

	// Wire MQTT command/down subscriber if enabled
	if infra.MQTTClient != nil && cfg.MQTT.EnableCommandSubscriptions && scaciServer != nil {
		downlinkQueuer := bssciservices.NewSCACIDownlinkQueuer(scaciServer)
		downlinkAdapter := bssciservices.NewMQTTDownlinkAdapter(downlinkQueuer)
		cmdHandler := mqtt.NewCommandHandler(infra.MQTTClient, downlinkAdapter, infra.OrgResolverSvc, infra.LoggerIface, cfg.MQTT.TopicPrefix)
		go cmdHandler.Start(ctx)
		log.Info("MQTT command/down subscriber started")
	}

	// Wire CE federation relay client and outbox (CE mode only)
	var protoRelayClient federationservices.RelayController
	var protoRelayGate federationservices.RelayGate
	if cfg.General.Edition == pkgconfig.EditionCommunity && cfg.Protocol.Federation.Enabled {
		sqlxDB := infra.SqlxDB
		outboxRepo := postgres.NewFederationOutboxRepository(sqlxDB)
		installRepo := postgres.NewCEInstallationRepository(sqlxDB)
		outboxWriter := federationservices.NewOutboxWriter(outboxRepo, infra.LoggerIface)
		bssciServer.SetRelayOutboxWriter(outboxWriter)

		relayClient := federationservices.NewRelayClient(
			cfg.Protocol.Federation,
			installRepo,
			outboxRepo,
			infra.LoggerIface,
		).WithCEVersion(infra.VersionInfo.Version).
			WithBsCountFn(func(bsCtx context.Context) int32 {
				stats, statsErr := infra.BasestationRepo.GetStatistics(bsCtx, infra.TenantID)
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
			if err := relayClient.Start(ctx); err != nil {
				log.Warn("Federation relay client could not start", "error", err)
			} else {
				log.Info("CE federation relay client started", "ece_endpoint", cfg.Protocol.Federation.ECEEndpoint)
			}
		} else {
			log.Info("CE onboarding not complete; federation relay deferred until onboarding")
		}

		cleanups = append(cleanups, func() { relayClient.Stop() })
	}

	return &ProtocolServers{
		BSSCIServer:         bssciServer,
		BSSCIServices:       bssciSvcBundle,
		BSSCIInfra:          bssciInfra,
		SCACIServer:         scaciServer,
		ServiceCenterEUI:    serviceCenterEUI,
		SoftwareVersion:     softwareVersion,
		Cleanups:            cleanups,
		RelayClient:         protoRelayClient,
		DispositionResolver: protoRelayGate,
	}, nil
}

// buildSCACIServer constructs and starts the SCACI server, wiring BSSCI→SCACI forwarding.
func buildSCACIServer(
	infra *Infrastructure,
	bssciServer *bssci.Server,
	bssciSvcBundle *bssciservices.BSSCIServiceBundle,
	bssciInfra *bssciservices.BSSCIInfrastructure,
	propagationSvc propagation.Service,
	serviceCenterEUI uint64,
) (*scaci.Server, error) {
	log := logger.Get()
	cfg := infra.Config

	// Validate SCACI config before use (SCACI §1 compliance)
	if err := pkgconfig.ValidateSCACIConfig(&cfg.Protocol); err != nil {
		return nil, fmt.Errorf("SCACI configuration invalid: %w", err)
	}
	log.Info("Initializing SCACI server...")

	scaciConfig := &scaci.Config{
		ListenAddr:            fmt.Sprintf("%s:%d", cfg.Protocol.SCACIHost, cfg.Protocol.SCACIPort),
		TLS:                   cfg.Protocol.SCACITLS,
		ServiceCenterEUI:      serviceCenterEUI,
		Vendor:                "KiloCenter",
		Model:                 "KiloCenter SC",
		Name:                  cfg.General.ServerName,
		SoftwareVersion:       infra.VersionInfo.Version,
		OrgEnforcementEnabled: cfg.General.OrgEnforcementEnabled,
		LogPingOperations:     cfg.Protocol.SCALogPingOperations,
		LogStatusOperations:   cfg.Protocol.SCALogStatusOperations,
	}

	scaciSessionRepo := infra.Storage.SCACISessions()

	// Strict org resolution startup guard (SCACI §1 isolation)
	if cfg.Protocol.StrictOrgResolution {
		if infra.OrgResolverSvc == nil {
			return nil, fmt.Errorf("StrictOrgResolution=true but orgResolverSvc is nil - cannot resolve certificates to tenants")
		}
		if !cfg.Protocol.SCACICertTenantMapping {
			return nil, fmt.Errorf("StrictOrgResolution=true requires SCACICertTenantMapping=true")
		}
		log.Info("SCACI strict org resolution enabled - certificates must resolve to tenant, no fallback allowed")
	} else if cfg.Protocol.SCACICertTenantMapping && infra.OrgResolverSvc == nil {
		log.Warn("SCACICertTenantMapping=true but orgResolverSvc is nil - will fall back to default tenant")
	}

	scaciSvcBundle := scaciservices.NewSCACIServices(
		scaciSessionRepo,
		infra.Storage.SCACIOperations(),
		infra.Storage.EndPoints(),
		infra.Storage.BaseStations(),
		infra.Storage,
		infra.SystemEventStore,
		bssciServer,
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
	)

	scaciServer, err := scaci.NewServer(
		scaciConfig,
		infra.LoggerIface,
		scaciSessionRepo,
		infra.Storage.SCACIOperations(),
		scaciSvcBundle.HandshakeSvc,
		scaciSvcBundle.EndpointSvc,
		scaciSvcBundle.ULSvc,
		scaciSvcBundle.DLSvc,
		scaciSvcBundle.StatusSvc,
		scaciSvcBundle.SessionValidator,
		scaciSvcBundle.OperationRecorder,
		scaciSvcBundle.SessionPersistence,
		bssciInfra.OrgResolver, // orgResolver (BSSCI/SCACI org context parity)
		bssciServer,            // sessionSnapshotProvider
		propagationSvc,         // propagationSvc (BSSCI §5.8-5.8.3)
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SCACI server: %w", err)
	}

	// Wire ErrorRecorder for §3.14 error persistence and event emission
	scaciServer.SetErrorRecorder(scaciSvcBundle.ErrorRecorder)
	log.Info("SCACI ErrorRecorder wired for §3.14 compliance")

	if err := scaciServer.Start(); err != nil {
		return nil, fmt.Errorf("failed to start SCACI server: %w", err)
	}

	log.Info("SCACI server started successfully",
		"host", cfg.Protocol.SCACIHost,
		"port", cfg.Protocol.SCACIPort,
		"spec_compliance", "MIOTY SCACI v1.0.0")

	// Wire BSSCI→SCACI forwarding via service bundle
	if forwarder, ok := bssciSvcBundle.Broadcaster.(interface{ SetSCACIServer(interface{}) }); ok {
		forwarder.SetSCACIServer(scaciServer)
		log.Info("BSSCI→SCACI forwarding enabled via service bundle")
	}

	// Wire BSSCI→SCACI EPStatus forwarding (SCACI §3.13)
	if bssciSvcBundle.EPStatusBroadcaster == nil {
		return nil, fmt.Errorf("SCACI EPStatus forwarding: EPStatusBroadcaster is nil")
	}
	adapter, ok := bssciSvcBundle.EPStatusBroadcaster.(bssciservices.SCACIEPStatusAdapterWithSetter)
	if !ok {
		return nil, errors.New(ErrSCACIEPStatusAdapterMissing)
	}
	adapter.SetSCACIServer(scaciServer)
	log.Info("BSSCI→SCACI EPStatus adapter wired")

	// Wire to bssciServer for direct forwarding from attach/detach handlers
	bssciServer.SetSCACIEPStatusBroadcaster(bssciSvcBundle.EPStatusBroadcaster)

	return scaciServer, nil
}

// BuildFederationIngestDeps constructs a fully-wired UplinkIngestService for use by the
// federation-ingress binary. It wires all real collaborators (org resolver, blueprint resolver,
// blueprint decoder, deduplicator) so the ingress binary shares identical ingest behaviour.
func BuildFederationIngestDeps(_ context.Context, infra *Infrastructure) (*bssciservices.UplinkIngestServiceImpl, error) {
	deduplicator := bssci.NewMessageDeduplicator(5 * time.Minute)
	resolverSvc := blueprintresolver.NewResolverService(
		infra.LoggerIface,
		infra.Storage.Blueprints(),
		infra.Storage.DeviceModels(),
		infra.Storage.EndPoints(),
	)
	decoderSvc := blueprintresolver.NewDecoderService(infra.LoggerIface)

	cfg := infra.Config
	syntheticEUI := parseSyntheticBsEUIFromConfig(cfg.Protocol.Federation.SyntheticBsEUI, infra.Log)

	svc := bssciservices.NewUplinkIngestService(
		deduplicator,
		infra.Storage,
		infra.OrgResolverSvc,
		nil, // federation ingress resolves endpoint directly; no roaming service needed
		infra.Storage.EndPoints(),
		resolverSvc,
		decoderSvc,
		nil, // no SCACI broadcaster in ingress binary
		nil, // no MQTT publisher in ingress binary
		infra.LoggerIface,
		infra.TenantID,
		syntheticEUI,
	)
	return svc, nil
}

// parseSyntheticBsEUIFromConfig converts the configured synthetic BS EUI hex string to uint64.
func parseSyntheticBsEUIFromConfig(s string, log logger.Logger) uint64 {
	if s == "" {
		return 0
	}
	var v uint64
	_, err := fmt.Sscanf(s, "%x", &v)
	if err != nil {
		log.Warn("Invalid synthetic_bs_eui format, using 0", "value", s, "error", err)
		return 0
	}
	return v
}
