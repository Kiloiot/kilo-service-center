// Package builders provides composition-root builder functions for KiloCenter.
// Each builder constructs a focused set of dependencies and returns an explicit
// output struct, forcing clear dependency flow through the orchestrator (main.go).
package builders

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/downlinkid"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/health"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-MQTT/pkg/mqtt"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
	"github.com/Kiloiot/kilo-service-center/pkg/version"
)

// Infrastructure holds all shared platform dependencies constructed during startup.
type Infrastructure struct {
	Config      *pkgconfig.Config
	Clock       clock.Clock
	Log         logger.Logger
	LoggerIface logger.Logger
	Storage     *postgres.DB
	Repos       *postgres.Repositories
	// HealthService reports every dashboard row; ReadinessService only the
	// rows this process needs to serve its API.
	HealthService          *health.Service
	ReadinessService       *health.Service
	statusBoard            *statusBoard
	MQTTClient             mqtt.Publisher
	OrgResolverSvc         org.Resolver
	DownlinkQueueIDs       downlinkid.Allocator
	ConnectionMgr          *basestation.ConnectionManager
	EventRecorder          basestation.EventRecorder
	SystemEventStore       interfaces.SystemEventStore
	Cipher                 keycrypto.Cipher
	TenantID               int64
	ServiceStart           time.Time
	VersionInfo            *version.Info
	CanonicalSCURL         string
	IdentityInternalClient pb.IdentityInternalServiceClient
	Cleanups               []func()
}

// Archival scheduler health messages.
const (
	msgArchivalSchedulerRunning    = "Archival scheduler is running"
	msgArchivalSchedulerNotRunning = "Archival scheduler is not running"
	msgArchivalSchedulerDisabled   = "Archival is disabled by configuration"
)

// StorageOptions maps the loaded storage configuration onto the postgres
// connection options shared by every binary that opens primary storage.
func StorageOptions(cfg pkgconfig.StorageConfig) postgres.Options {
	return postgres.Options{
		Host:            cfg.Host,
		Port:            cfg.Port,
		Database:        cfg.Database,
		Username:        cfg.Username,
		Password:        cfg.Password,
		SSLMode:         cfg.SSLMode,
		MaxOpenConns:    cfg.MaxConnections,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: time.Duration(cfg.ConnMaxLifetime) * time.Second,
		ConnMaxIdleTime: time.Duration(cfg.ConnMaxIdleTime) * time.Second,
	}
}

// BuildInfrastructure constructs shared platform dependencies: storage, migrations,
// key encryption, archival, MQTT, health service, connection manager, org resolver.
// Pass opts to override edition-specific behavior (nil uses CE defaults).
func BuildInfrastructure(ctx context.Context, cfg *pkgconfig.Config, versionInfo *version.Info, opts *Options) (*Infrastructure, error) {
	log := logger.Get()
	clk := clock.SystemClock{}
	serviceStart := time.Now().UTC()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgInvalidConfiguration, err)
	}

	if cfg.General.Environment == "" {
		log.Warn(LogEnvironmentNotSetDefaultingToDevelopment)
		cfg.General.Environment = pkgconfig.DefaultGeneralEnvironment
	}

	if cfg.Protocol.MessageEncoding == "" {
		cfg.Protocol.MessageEncoding = bssci.EncodingMessagePack
		log.Info(LogMessageEncodingNotConfiguredUsingDefault, logger.FieldEncoding, cfg.Protocol.MessageEncoding)
	}

	log.Info(
		LogConfigurationLoaded,
		logger.FieldEnvironment, cfg.General.Environment,
		logger.FieldLogLevel, cfg.General.LogLevel,
		logger.FieldHealthCheckPort, cfg.General.HealthCheckPort,
	)

	var cleanups []func()

	// Build the mandatory key-material cipher before opening storage: all key
	// material (endpoints, endpoint sessions) is encrypted at rest through it,
	// so a missing or malformed master key is a fatal configuration error.
	cipher, err := keycrypto.NewCipherFromMasterKey(os.Getenv(envKilocenterMasterKey))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToBuildKeyMaterialCipher, err)
	}

	// Initialize storage
	log.Info(LogInitializingStorage)
	storageOpts := StorageOptions(cfg.Storage)
	storageOpts.Clock = clk
	storage, err := postgres.New(storageOpts, cipher)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToInitializeStorage, err)
	}
	cleanups = append(cleanups, func() {
		if err := storage.Close(); err != nil {
			log.Error(LogFailedCloseStorage, logger.Err(err))
		}
	})
	repos := postgres.NewRepositories(storage)

	// Wait for database to be ready
	log.Info(LogWaitingForDatabaseConnection)
	if err := storage.WaitForConnection(ctx, dbConnectionWaitTimeout); err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgDatabaseConnectionTimeout, err)
	}

	// Run database migrations (mandatory for UL persistence per BSSCI §5.10.1)
	if !cfg.Storage.EnableMigrations {
		return nil, errors.New(errMsgMigrationsRequiredForULPersistence)
	}

	log.Info(LogRunningDatabaseMigrations)
	migrationConfig := &postgres.Config{
		Host:     cfg.Storage.Host,
		Port:     cfg.Storage.Port,
		Database: cfg.Storage.Database,
		Username: cfg.Storage.Username,
		Password: cfg.Storage.Password,
		SSLMode:  cfg.Storage.SSLMode,
	}

	runner, err := postgres.NewMigrationRunner(migrationConfig, postgres.UpgradeGates())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedToRunDatabaseMigrations, err)
	}
	dbVersion, migErr := runner.Run(ctx)
	if migErr != nil {
		// Emit migration failure event if storage is available
		if evtErr := repos.SystemEvents.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(cfg.General.TenantID, 10),
			EventType:   models.EventTypeMigrationFailed,
			Category:    models.EventCategoryError,
			Severity:    models.EventSeverityError,
			Title:       models.EventTitleMigrationFailed,
			Description: fmt.Sprintf(models.EventDescriptionMigrationFailedFmt, CoreServiceSourceName, migErr.Error()),
			SourceName:  CoreServiceSourceName,
			SourceType:  models.SourceTypeSystem,
		}); evtErr != nil {
			log.Error(LogFailedToEmitMigrationFailedEvent, logger.Err(evtErr))
		}
		return nil, fmt.Errorf("%s: %w", errMsgFailedToRunDatabaseMigrations, migErr)
	}

	// Emit migration success event
	if evtErr := repos.SystemEvents.CreateEvent(ctx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(cfg.General.TenantID, 10),
		EventType:   models.EventTypeMigrationApplied,
		Category:    models.EventCategorySystem,
		Severity:    models.EventSeverityInfo,
		Title:       models.EventTitleMigrationApplied,
		Description: fmt.Sprintf(models.EventDescriptionMigrationAppliedFmt, dbVersion, CoreServiceSourceName),
		SourceName:  CoreServiceSourceName,
		SourceType:  models.SourceTypeSystem,
	}); evtErr != nil {
		log.Error(LogFailedToEmitMigrationAppliedEvent, logger.Err(evtErr))
	}

	// Enforce minimum schema version matches release manifest
	// Safe conversion: SchemaVersion is always positive (validated in manifest) and small
	if versionInfo.SchemaVersion >= 0 && dbVersion < uint(versionInfo.SchemaVersion) { // #nosec G115 -- SchemaVersion validated positive
		return nil, fmt.Errorf(errFmtSchemaVersionTooOld, dbVersion, versionInfo.SchemaVersion)
	}

	log.Info(LogDatabaseMigrationsComplete, logger.FieldVersion, dbVersion)

	// Initialize archival service and scheduler
	archivalService := postgres.NewArchivalService(storage.GetDB(), log, clk)

	archivalConfig := postgres.DefaultArchivalConfig()
	if cfg.Storage.MessageRetentionDays > 0 {
		archivalConfig.MessageRetentionDays = cfg.Storage.MessageRetentionDays
	}
	if cfg.Storage.ArchivalEnabled != nil {
		archivalConfig.ArchivalEnabled = *cfg.Storage.ArchivalEnabled
	}

	archivalScheduler := postgres.NewArchivalScheduler(archivalService, log, archivalConfig, clk)
	if err := archivalScheduler.Start(ctx); err != nil {
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
		log.Info(LogInitializingMQTTClient)
		mqttClient, err = mqtt.NewClient(&cfg.MQTT, log)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", errMsgFailedToInitializeMQTTClient, err)
		}
		cleanups = append(cleanups, func() { mqttClient.Disconnect(context.Background()) }) // context-root: shutdown
	}

	if mqttClient != nil {
		if err := mqttClient.Connect(ctx); err != nil {
			return nil, fmt.Errorf("%s: %w", errMsgFailedToConnectToMQTTBroker, err)
		}
		log.Info(LogMQTTClientConnected, logger.FieldBroker, pkgconfig.GetMQTTBrokerURL(cfg.MQTT))
	}

	// Create context-aware logger for services
	loggerIface := logger.Get()

	statusBoard := newStatusBoard(loggerIface, versionInfo.Version)

	var mqttConnected func() bool
	if mqttClient != nil {
		mqttConnected = mqttClient.IsConnected
	}
	var archivalRow health.Checker = health.NewDisabledChecker(msgArchivalSchedulerDisabled)
	if archivalConfig.ArchivalEnabled {
		archivalRow = &archivalHealthChecker{scheduler: archivalScheduler, log: loggerIface}
	}
	registerStatusCheckers(statusBoard, cfg, statusCheckerDeps{
		DB:            storage.GetDB(),
		MQTTConnected: mqttConnected,
		Archival:      archivalRow,
		Log:           loggerIface,
	})

	downlinkQueueIDs, err := downlinkid.New(downlinkid.NewCryptoGenerator(), downlinkid.DefaultAttempts)
	if err != nil {
		return nil, err
	}

	// Initialize connection manager for base station management
	log.Info(LogInitializingBaseStationConnectionManager)

	tenantID := cfg.General.TenantID
	canonicalServiceCenterURL := pkgconfig.GetServiceCenterURL(&cfg.Protocol)

	repositoryAdapter := basestation.NewRepositoryAdapter(repos.BaseStations, tenantID, canonicalServiceCenterURL, loggerIface)
	systemEventStore := repos.SystemEvents
	eventRecorder := basestation.NewNamingEventRecorder(repositoryAdapter,
		basestation.NewPersistentEventRecorder(loggerIface, systemEventStore, fmt.Sprintf("%d", tenantID)), loggerIface)
	connectionManager := basestation.NewConnectionManager(repositoryAdapter, eventRecorder)

	log.Info(LogMessageStorageReadyViaPostgresDB)

	// Create basestation repository with sqlx wrapper
	log.Info(LogInitializingBasestationRepository)

	// Initialize organization resolver (edition-specific).
	// CE uses single-tenant community resolver; ECE can override via opts.
	orgResolverFn := defaultOrgResolverBuilder
	if opts != nil && opts.OrgResolverBuilder != nil {
		orgResolverFn = opts.OrgResolverBuilder
	}
	orgResult, err := orgResolverFn(ctx, NewOrgResolverConfig(cfg), repos.Organizations, log)
	if err != nil {
		return nil, err
	}
	cleanups = append(cleanups, orgResult.Cleanups...)

	// Create downlink queue reader for tenant resolution

	return &Infrastructure{
		Config:                 cfg,
		Clock:                  clk,
		Log:                    log,
		LoggerIface:            loggerIface,
		Storage:                storage,
		Repos:                  repos,
		HealthService:          statusBoard.status,
		ReadinessService:       statusBoard.readiness,
		statusBoard:            statusBoard,
		MQTTClient:             mqttClient,
		OrgResolverSvc:         orgResult.Resolver,
		DownlinkQueueIDs:       downlinkQueueIDs,
		ConnectionMgr:          connectionManager,
		EventRecorder:          eventRecorder,
		SystemEventStore:       systemEventStore,
		Cipher:                 cipher,
		TenantID:               tenantID,
		ServiceStart:           serviceStart,
		VersionInfo:            versionInfo,
		CanonicalSCURL:         canonicalServiceCenterURL,
		IdentityInternalClient: orgResult.IdentityInternalClient,
		Cleanups:               cleanups,
	}, nil
}

// archivalStatusReporter is the capability the archival row reads.
type archivalStatusReporter interface {
	GetStatus() map[string]interface{}
}

// archivalHealthChecker reports the archival scheduler status as a health check.
type archivalHealthChecker struct {
	scheduler archivalStatusReporter
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
		check.Message = msgArchivalSchedulerRunning
		check.Metadata = status
	} else {
		check.Status = health.StatusUnhealthy
		check.Message = msgArchivalSchedulerNotRunning
	}

	return check
}
