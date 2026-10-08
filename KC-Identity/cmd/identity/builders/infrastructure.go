// Package builders provides composition-root builder functions for KC-Identity.
package builders

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/orgresolver"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
)

// Infrastructure holds shared platform dependencies for KC-Identity.
type Infrastructure struct {
	Config         *pkgconfig.Config
	Clock          clock.Clock
	Log            logger.Logger
	Storage        *postgres.DB
	Repos          *postgres.Repositories
	DB             *sql.DB
	OrgResolverSvc org.Resolver
	OrgRepo        interfaces.OrganizationRepository
	TenantID       int64
	Cleanups       []func()
}

const identitySourceName = "kc-identity"

// BuildInfrastructure constructs shared platform dependencies: storage, org resolver.
// identitySourceName identifies KC-Identity as a system event source.
func BuildInfrastructure(ctx context.Context, cfg *pkgconfig.Config) (*Infrastructure, error) {
	log := logger.Get()
	clk := clock.SystemClock{}
	var cleanups []func()

	// Build the mandatory key-material cipher before opening storage.
	cipher, err := keycrypto.NewCipherFromMasterKey(os.Getenv("KILOCENTER_MASTER_KEY"))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCipherInit, err)
	}

	// Initialize storage
	log.Info(LogInitializingStorage)
	storageOpts := postgres.Options{
		Clock:           clk,
		Host:            cfg.Storage.Host,
		Port:            cfg.Storage.Port,
		Database:        cfg.Storage.Database,
		Username:        cfg.Storage.Username,
		Password:        cfg.Storage.Password,
		SSLMode:         cfg.Storage.SSLMode,
		MaxOpenConns:    cfg.Storage.MaxConnections,
		MaxIdleConns:    cfg.Storage.MaxIdleConns,
		ConnMaxLifetime: time.Duration(cfg.Storage.ConnMaxLifetime) * time.Second,
		ConnMaxIdleTime: time.Duration(cfg.Storage.ConnMaxIdleTime) * time.Second,
	}
	storage, err := postgres.New(storageOpts, cipher)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStorageInit, err)
	}
	repos := postgres.NewRepositories(storage)
	cleanups = append(cleanups, func() {
		if err := storage.Close(); err != nil {
			log.Error(LogStorageCloseFailed, logger.FieldError, err)
		}
	})

	// Wait for database to be ready
	log.Info(LogWaitingForDatabase)
	if err := storage.WaitForConnection(ctx, databaseConnectTimeout); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseConnectTimeout, err)
	}

	// Run database migrations
	if cfg.Storage.EnableMigrations {
		log.Info(LogRunningMigrations)
		migrationConfig := &postgres.Config{
			Host:     cfg.Storage.Host,
			Port:     cfg.Storage.Port,
			Database: cfg.Storage.Database,
			Username: cfg.Storage.Username,
			Password: cfg.Storage.Password,
			SSLMode:  cfg.Storage.SSLMode,
		}
		runner, runnerErr := postgres.NewMigrationRunner(migrationConfig, postgres.UpgradeGates())
		if runnerErr != nil {
			return nil, fmt.Errorf("%w: %w", ErrMigrationsFailed, runnerErr)
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
				Description: fmt.Sprintf(models.EventDescriptionMigrationFailedFmt, identitySourceName, migErr.Error()),
				SourceName:  identitySourceName,
				SourceType:  models.SourceTypeSystem,
			}); evtErr != nil {
				log.Error(LogMigrationFailedEventError, logger.FieldError, evtErr)
			}
			return nil, fmt.Errorf("%w: %w", ErrMigrationsFailed, migErr)
		}

		// Emit migration success event
		if evtErr := repos.SystemEvents.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(cfg.General.TenantID, 10),
			EventType:   models.EventTypeMigrationApplied,
			Category:    models.EventCategorySystem,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleMigrationApplied,
			Description: fmt.Sprintf(models.EventDescriptionMigrationAppliedFmt, dbVersion, identitySourceName),
			SourceName:  identitySourceName,
			SourceType:  models.SourceTypeSystem,
		}); evtErr != nil {
			log.Error(LogMigrationAppliedEventError, logger.FieldError, evtErr)
		}

		log.Info(LogMigrationsComplete, logger.FieldVersion, dbVersion)
	}

	// Initialize organization resolver with caching
	orgRepo := repos.Organizations
	cacheTTL := time.Duration(cfg.General.OrgCacheTTLMinutes) * time.Minute
	cacheMaxEntries := cfg.General.OrgCacheMaxEntries

	loggerIface := logger.Get()

	var orgResolverSvc org.Resolver
	if pkgconfig.IsCommunityEdition(cfg.General.Edition) {
		log.Info(LogCESingleTenantResolver,
			logger.FieldTenantIDSnake, cfg.General.TenantID)

		defaultOrg, lookupErr := orgRepo.GetOrgByTenantID(ctx, cfg.General.TenantID)
		if lookupErr != nil || defaultOrg == nil {
			return nil, fmt.Errorf("%w: %d: %w", ErrCEDefaultOrgRequired, cfg.General.TenantID, lookupErr)
		}

		orgResolverSvc = org.NewCommunityResolver(cfg.General.TenantID, defaultOrg.OrgID)
	} else {
		log.Info(LogInitializingOrgResolver,
			logger.FieldCacheTTL, cacheTTL,
			logger.FieldMaxEntriesSnake, cacheMaxEntries)

		resolver, resolverErr := orgresolver.New(orgRepo, loggerIface, cacheTTL, cacheMaxEntries, clk)
		if resolverErr != nil {
			return nil, fmt.Errorf("%w: %w", ErrOrgResolverInit, resolverErr)
		}
		orgResolverSvc = resolver
	}

	return &Infrastructure{
		Clock:          clk,
		Config:         cfg,
		Log:            loggerIface,
		Storage:        storage,
		Repos:          repos,
		DB:             storage.GetDB(),
		OrgResolverSvc: orgResolverSvc,
		OrgRepo:        orgRepo,
		TenantID:       cfg.General.TenantID,
		Cleanups:       cleanups,
	}, nil
}
