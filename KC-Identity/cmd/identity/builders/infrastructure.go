// Package builders provides composition-root builder functions for KC-Identity.
package builders

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/orgresolver"
	"github.com/jmoiron/sqlx"
)

// Infrastructure holds shared platform dependencies for KC-Identity.
type Infrastructure struct {
	Config         *pkgconfig.Config
	Log            logger.Logger
	Storage        *postgres.DB
	SqlxDB         *sqlx.DB
	DB             *sql.DB
	OrgResolverSvc org.Resolver
	OrgRepo        interfaces.OrganizationRepository
	TenantID       int64
	Cleanups       []func()
}

// BuildInfrastructure constructs shared platform dependencies: storage, org resolver.
func BuildInfrastructure(ctx context.Context, cfg *pkgconfig.Config) (*Infrastructure, error) {
	log := logger.Get()
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

	// Run database migrations
	if cfg.Storage.EnableMigrations {
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
				Description: fmt.Sprintf(models.EventDescriptionMigrationFailedFmt, "kc-identity", migErr.Error()),
				SourceName:  "kc-identity",
				SourceType:  models.SourceTypeSystem,
			}); evtErr != nil {
				log.Error("failed to emit migration.failed event", "error", evtErr)
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
			Description: fmt.Sprintf(models.EventDescriptionMigrationAppliedFmt, dbVersion, "kc-identity"),
			SourceName:  "kc-identity",
			SourceType:  models.SourceTypeSystem,
		}); evtErr != nil {
			log.Error("failed to emit migration.applied event", "error", evtErr)
		}

		log.Info("Database migrations complete", "version", dbVersion)
	}

	sqlxDB := sqlx.NewDb(storage.GetDB(), "postgres")

	// Initialize organization resolver with caching
	orgRepo := storage.Organizations()
	cacheTTL := time.Duration(cfg.General.OrgCacheTTLMinutes) * time.Minute
	cacheMaxEntries := cfg.General.OrgCacheMaxEntries

	loggerIface := logger.Get()

	var orgResolverSvc org.Resolver
	if pkgconfig.IsCommunityEdition(cfg.General.Edition) {
		log.Info("Community Edition: initializing single-tenant org resolver",
			"tenant_id", cfg.General.TenantID)

		defaultOrg, lookupErr := orgRepo.GetOrgByTenantID(ctx, cfg.General.TenantID)
		if lookupErr != nil || defaultOrg == nil {
			return nil, fmt.Errorf("CE requires default org for tenant %d: %w", cfg.General.TenantID, lookupErr)
		}

		orgResolverSvc = org.NewCommunityResolver(cfg.General.TenantID, defaultOrg.OrgID)
	} else {
		log.Info("Initializing organization resolver with caching...",
			"cache_ttl", cacheTTL,
			"max_entries", cacheMaxEntries)

		orgResolverSvc = orgresolver.New(
			orgRepo,
			loggerIface,
			cacheTTL,
			cacheMaxEntries,
		)
	}

	return &Infrastructure{
		Config:         cfg,
		Log:            loggerIface,
		Storage:        storage,
		SqlxDB:         sqlxDB,
		DB:             storage.GetDB(),
		OrgResolverSvc: orgResolverSvc,
		OrgRepo:        orgRepo,
		TenantID:       cfg.General.TenantID,
		Cleanups:       cleanups,
	}, nil
}
