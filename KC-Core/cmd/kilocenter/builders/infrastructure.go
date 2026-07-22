// Package builders provides composition-root builder functions for KiloCenter.
// Each builder constructs a focused set of dependencies and returns an explicit
// output struct, forcing clear dependency flow through the orchestrator (main.go).
package builders

import (
	"context"
	"fmt"
	"strconv"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/health"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-MQTT/pkg/mqtt"
	"github.com/Kiloiot/kilo-service-center/pkg/version"
	"github.com/jmoiron/sqlx"
)

// Infrastructure holds all shared platform dependencies constructed during startup.
type Infrastructure struct {
	Config                 *pkgconfig.Config
	Log                    logger.Logger
	LoggerIface            logger.Logger
	Storage                *postgres.DB
	SqlxDB                 *sqlx.DB
	HealthService          *health.Service
	MQTTClient             mqtt.Publisher
	OrgResolverSvc         org.Resolver
	ConnectionMgr          *basestation.ConnectionManager
	EventRecorder          basestation.EventRecorder
	SystemEventStore       interfaces.SystemEventStore
	KeyEncryptor           *crypto.KeyEncryptor
	TenantID               int64
	ServiceStart           time.Time
	VersionInfo            *version.Info
	BasestationRepo        interfaces.BaseStationRepository
	QueueStore             *postgres.DownlinkQueueReader
	CanonicalSCURL         string
	IdentityInternalClient pb.IdentityInternalServiceClient
	Cleanups               []func()
}

// BuildInfrastructure constructs shared platform dependencies: storage, migrations,
// key encryption, archival, MQTT, health service, connection manager, org resolver.
// Pass opts to override edition-specific behavior (nil uses CE defaults).
func BuildInfrastructure(ctx context.Context, cfg *pkgconfig.Config, versionInfo *version.Info, opts *Options) (*Infrastructure, error) {
	log := logger.Get()
	serviceStart := time.Now().UTC()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	if cfg.General.Environment == "" {
		log.Warn("Environment not set, defaulting to 'development'")
		cfg.General.Environment = "development"
	}

	if cfg.Protocol.MessageEncoding == "" {
		cfg.Protocol.MessageEncoding = bssci.EncodingMessagePack
		log.Info("Message encoding not configured, using default", "encoding", cfg.Protocol.MessageEncoding)
	}

	log.Info("Configuration loaded",
		"environment", cfg.General.Environment,
		"log_level", cfg.General.LogLevel,
		"health_check_port", cfg.General.HealthCheckPort,
	)

	var cleanups []func()

	// Initialize storage
	log.Info("Initializing storage...")
	storage, err := postgres.New(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}
	cleanups = append(cleanups, func() { _ = storage.Close() })

	// Wait for database to be ready
	log.Info("Waiting for database connection...")
	if err := storage.WaitForConnection(ctx, 30*time.Second); err != nil {
		return nil, fmt.Errorf("database connection timeout: %w", err)
	}

	// Run database migrations (mandatory for UL persistence per BSSCI §5.10.1)
	if !cfg.Storage.EnableMigrations {
		return nil, fmt.Errorf("migrations are required for UL persistence; enable_migrations=false")
	}

	log.Info("Running database migrations...")
	migrationConfig := &postgres.Config{
		Host:     cfg.Storage.Host,
		Port:     cfg.Storage.Port,
		Database: cfg.Storage.Database,
		Username: cfg.Storage.Username,
		Password: cfg.Storage.Password,
		SSLMode:  cfg.Storage.SSLMode,
	}

	runner := postgres.NewMigrationRunner(migrationConfig)
	dbVersion, migErr := runner.Run()
	if migErr != nil {
		// Emit migration failure event if storage is available
		if evtErr := storage.SystemEvents().CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(cfg.General.TenantID, 10),
			EventType:   models.EventTypeMigrationFailed,
			Category:    models.EventCategoryError,
			Severity:    models.EventSeverityError,
			Title:       models.EventTitleMigrationFailed,
			Description: fmt.Sprintf(models.EventDescriptionMigrationFailedFmt, coreServiceSourceName, migErr.Error()),
			SourceName:  coreServiceSourceName,
			SourceType:  models.SourceTypeSystem,
		}); evtErr != nil {
			log.Error("failed to emit migration.failed event", logger.Err(evtErr))
		}
		return nil, fmt.Errorf("failed to run database migrations: %w", migErr)
	}

	// Emit migration success event
	if evtErr := storage.SystemEvents().CreateEvent(ctx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(cfg.General.TenantID, 10),
		EventType:   models.EventTypeMigrationApplied,
		Category:    models.EventCategorySystem,
		Severity:    models.EventSeverityInfo,
		Title:       models.EventTitleMigrationApplied,
		Description: fmt.Sprintf(models.EventDescriptionMigrationAppliedFmt, dbVersion, coreServiceSourceName),
		SourceName:  coreServiceSourceName,
		SourceType:  models.SourceTypeSystem,
	}); evtErr != nil {
		log.Error("failed to emit migration.applied event", logger.Err(evtErr))
	}

	// Enforce minimum schema version matches release manifest
	// Safe conversion: SchemaVersion is always positive (validated in manifest) and small
	if versionInfo.SchemaVersion >= 0 && dbVersion < uint(versionInfo.SchemaVersion) { // #nosec G115 -- SchemaVersion validated positive
		return nil, fmt.Errorf("schema version too old: current=%d, required=%d", dbVersion, versionInfo.SchemaVersion)
	}

	log.Info("Database migrations complete", "version", dbVersion)

	// Initialize key encryptor for session key lazy migration (BSSCI §5.6)
	log.Info("Initializing key encryptor for session key migration...")
	keyEncryptor, err := crypto.NewKeyEncryptor()
	if err != nil {
		log.Warn(LogFailedKeyEncryptorInit, logger.Err(err))
		keyEncryptor = nil
	}

	if keyEncryptor != nil {
		storage.SetKeyEncryptor(keyEncryptor)
		log.Info("Session key lazy migration enabled in storage layer")
	}

	// Initialize archival service and scheduler
	archivalService := postgres.NewArchivalService(storage.GetDB(), log)

	archivalConfig := postgres.DefaultArchivalConfig()
	if cfg.Storage.MessageRetentionDays > 0 {
		archivalConfig.MessageRetentionDays = cfg.Storage.MessageRetentionDays
	}
	if cfg.Storage.ArchivalEnabled != nil {
		archivalConfig.ArchivalEnabled = *cfg.Storage.ArchivalEnabled
	}

	archivalScheduler := postgres.NewArchivalScheduler(archivalService, log, archivalConfig)
	if err := archivalScheduler.Start(); err != nil {
		log.Error(LogFailedStartArchivalScheduler, logger.Err(err))
	}
	cleanups = append(cleanups, func() {
		if err := archivalScheduler.Stop(); err != nil {
			log.Error(LogFailedStopArchivalScheduler, logger.Err(err))
		}
	})

	// Initialize MQTT client if enabled
	var mqttClient mqtt.Publisher
	if cfg.MQTT.Enabled {
		log.Info("Initializing MQTT client...")
		mqttClient, err = mqtt.NewClient(&cfg.MQTT, log)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize MQTT client: %w", err)
		}
		cleanups = append(cleanups, func() { mqttClient.Disconnect(context.Background()) })
	}

	if mqttClient != nil {
		if err := mqttClient.Connect(ctx); err != nil {
			return nil, fmt.Errorf("failed to connect to MQTT broker: %w", err)
		}
		log.Info("MQTT client connected", "broker", pkgconfig.GetMQTTBrokerURL(cfg.MQTT))
	}

	// Create context-aware logger for services
	loggerIface := logger.Get()

	// Initialize health check service
	healthService := health.NewService(loggerIface, versionInfo.Version)

	for _, ep := range cfg.Status.Endpoints {
		switch ep.Type {
		case "http":
			healthService.RegisterChecker(ep.Name, health.NewHTTPChecker(ep.URL, cfg.Status.Timeout, loggerIface))
		case "tcp":
			healthService.RegisterChecker(ep.Name, health.NewTCPChecker(ep.GetAddress(), cfg.Status.Timeout, loggerIface))
		case "db":
			healthService.RegisterChecker(ep.Name, health.NewPostgreSQLChecker(storage.GetDB(), loggerIface))
		case "mqtt":
			if mqttClient != nil {
				healthService.RegisterChecker(ep.Name, health.NewMQTTChecker(
					func() bool { return mqttClient.IsConnected() },
					loggerIface,
				))
			}
		case "archival":
			healthService.RegisterChecker(ep.Name, &archivalHealthChecker{
				scheduler: archivalScheduler,
				log:       loggerIface,
			})
		}
	}

	// Initialize connection manager for base station management
	log.Info("Initializing base station connection manager...")

	tenantID := cfg.General.TenantID
	canonicalServiceCenterURL := pkgconfig.GetServiceCenterURL(&cfg.Protocol)

	repositoryAdapter := basestation.NewRepositoryAdapter(storage.BaseStations(), tenantID, canonicalServiceCenterURL)
	systemEventStore := storage.SystemEvents()
	eventRecorder := basestation.NewPersistentEventRecorder(loggerIface, systemEventStore, fmt.Sprintf("%d", tenantID))
	connectionManager := basestation.NewConnectionManager(repositoryAdapter, eventRecorder, loggerIface)

	log.Info("Message storage ready via postgres.DB...")

	// Create basestation repository with sqlx wrapper
	log.Info("Initializing basestation repository...")
	sqlxDB := sqlx.NewDb(storage.GetDB(), "postgres")
	basestationRepo := postgres.NewBaseStationRepository(sqlxDB)

	// Initialize organization resolver (edition-specific).
	// CE uses single-tenant community resolver; ECE can override via opts.
	orgResolverFn := defaultOrgResolverBuilder
	if opts != nil && opts.OrgResolverBuilder != nil {
		orgResolverFn = opts.OrgResolverBuilder
	}
	ocfg := &OrgResolverConfig{
		Edition:              cfg.General.Edition,
		TenantID:             tenantID,
		IdentityAddress:      cfg.Identity.Address,
		PeerSecret:           cfg.InternalAuth.PeerSecret,
		InternalTrustEnabled: cfg.GRPC.InternalTrustEnabled,
		OrgCacheTTLMinutes:   cfg.General.OrgCacheTTLMinutes,
		OrgCacheMaxEntries:   cfg.General.OrgCacheMaxEntries,
	}
	orgResult, err := orgResolverFn(ctx, ocfg, storage.Organizations(), log)
	if err != nil {
		return nil, err
	}
	cleanups = append(cleanups, orgResult.Cleanups...)

	// Create downlink queue reader for tenant resolution
	queueStore := postgres.NewDownlinkQueueReader(sqlxDB)

	return &Infrastructure{
		Config:                 cfg,
		Log:                    log,
		LoggerIface:            loggerIface,
		Storage:                storage,
		SqlxDB:                 sqlxDB,
		HealthService:          healthService,
		MQTTClient:             mqttClient,
		OrgResolverSvc:         orgResult.Resolver,
		ConnectionMgr:          connectionManager,
		EventRecorder:          eventRecorder,
		SystemEventStore:       systemEventStore,
		KeyEncryptor:           keyEncryptor,
		TenantID:               tenantID,
		ServiceStart:           serviceStart,
		VersionInfo:            versionInfo,
		BasestationRepo:        basestationRepo,
		QueueStore:             queueStore,
		CanonicalSCURL:         canonicalServiceCenterURL,
		IdentityInternalClient: orgResult.IdentityInternalClient,
		Cleanups:               cleanups,
	}, nil
}

// archivalHealthChecker reports the archival scheduler status as a health check.
// Defined in this package to avoid exporting the archivalScheduler instance.
type archivalHealthChecker struct {
	scheduler *postgres.ArchivalScheduler
	log       logger.Logger
}

func (c *archivalHealthChecker) Check(_ context.Context) *health.Check {
	start := time.Now()
	status := c.scheduler.GetStatus()

	check := &health.Check{
		Name:      "archival_scheduler",
		Timestamp: start,
		Duration:  time.Since(start),
	}

	if running, ok := status["running"].(bool); ok && running {
		check.Status = health.StatusHealthy
		check.Message = "Archival scheduler is running"
		check.Metadata = status
	} else {
		check.Status = health.StatusUnhealthy
		check.Message = "Archival scheduler is not running"
	}

	return check
}
