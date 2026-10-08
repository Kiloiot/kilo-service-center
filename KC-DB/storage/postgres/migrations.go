package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/migrations"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Construction faults of a migration runner.
var (
	ErrNilMigrationConfig = errors.New("migration runner: config is nil")
	ErrNilSchemaGate      = errors.New("migration runner: schema gate is nil")
)

// MigrationRunner handles database migrations using dedicated short-lived connections.
// This design ensures that m.Close() can be safely called without affecting the main
// storage pool used by the application.
type MigrationRunner struct {
	config *Config
	gates  []SchemaGate
}

// SchemaGate holds an upgrade at a schema version until data that later
// migrations depend on is in place. Check only reads, so a refusal leaves
// the schema clean at Version.
type SchemaGate interface {
	Version() uint
	Check(ctx context.Context, db *sql.DB) error
}

// UpgradeGates returns the gates every upgrade of this schema must pass; a
// composition root hands them to NewMigrationRunner.
func UpgradeGates() []SchemaGate {
	return []SchemaGate{keyMaterialGate{}}
}

// allVersions targets every pending migration.
const allVersions = ^uint(0)

// NewMigrationRunner creates a migration runner that holds every upgrade at
// each of gates until the gate's check passes.
func NewMigrationRunner(config *Config, gates []SchemaGate) (*MigrationRunner, error) {
	if config == nil {
		return nil, ErrNilMigrationConfig
	}
	for _, gate := range gates {
		if gate == nil {
			return nil, ErrNilSchemaGate
		}
	}
	return &MigrationRunner{config: config, gates: append([]SchemaGate(nil), gates...)}, nil
}

// openMigrationConnection opens a dedicated short-lived connection for migrations
// with connectivity validation via Ping. This ensures clear error messages when
// credentials/host/SSL are misconfigured instead of failing deep in migrate.
func (mr *MigrationRunner) openMigrationConnection(ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open("postgres", mr.config.GetDSN())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapOpenMigrationConnection, err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, migrationPingTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		return nil, joinCloseError(fmt.Errorf("%s: %w", errWrapPingMigrationConnection, err), errWrapCloseMigrationConnection, db.Close())
	}

	return db, nil
}

// Run executes all pending migrations and returns the final version.
// Uses a dedicated short-lived DB connection to avoid closing the main pool.
func (mr *MigrationRunner) Run(ctx context.Context) (uint, error) {
	return mr.migrateUp(ctx, allVersions, func(m *migrate.Migrate) error {
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("%s: %w", errWrapRunMigrations, err)
		}
		return nil
	})
}

// RunTo applies pending migrations up to and including target and returns
// the resulting version. It never migrates down: a database already at or
// past target is left untouched, so a repeated upgrade step is harmless.
func (mr *MigrationRunner) RunTo(ctx context.Context, target uint) (uint, error) {
	return mr.migrateUp(ctx, target, func(m *migrate.Migrate) error {
		return migrateTo(m, target)
	})
}

// migrateUp moves a clean schema towards target: at every gate the upgrade
// crosses it first stops at the gate's version and requires the gate's check
// to pass, then finish applies the remaining migrations.
func (mr *MigrationRunner) migrateUp(ctx context.Context, target uint, finish func(*migrate.Migrate) error) (uint, error) {
	var version uint
	err := mr.withMigrate(ctx, func(m *migrate.Migrate, db *sql.DB) error {
		var err error
		version, err = mr.migrateUpWith(ctx, m, db, target, finish)
		return err
	})
	if err != nil {
		return 0, err
	}
	return version, nil
}

// migrateUpWith runs migrateUp's gated upgrade on an open migrate instance.
func (mr *MigrationRunner) migrateUpWith(ctx context.Context, m *migrate.Migrate, db *sql.DB, target uint, finish func(*migrate.Migrate) error) (uint, error) {
	current, err := cleanVersion(m)
	if err != nil {
		return 0, err
	}
	if current >= target {
		return current, nil
	}

	for _, gate := range mr.gates {
		if current > gate.Version() || target <= gate.Version() {
			continue
		}
		if err := migrateTo(m, gate.Version()); err != nil {
			return 0, err
		}
		if err := gate.Check(ctx, db); err != nil {
			return 0, err
		}
	}

	if err := finish(m); err != nil {
		return 0, err
	}
	return cleanVersion(m)
}

// migrateTo moves the schema to version; an already reached version is not
// an error.
func migrateTo(m *migrate.Migrate, version uint) error {
	if err := m.Migrate(version); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf(errFmtRunMigrationsToVersion, version, err)
	}
	return nil
}

// cleanVersion returns the applied schema version, zero for a database no
// migration has touched, and an error when a failed migration left it dirty.
func cleanVersion(m *migrate.Migrate) (uint, error) {
	version, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapGetMigrationVersion, err)
	}
	if dirty {
		return 0, fmt.Errorf(errFmtDatabaseIsInDirtyStateAtVersion, version)
	}
	return version, nil
}

// Version returns the current migration version
func (mr *MigrationRunner) Version(ctx context.Context) (uint, bool, error) {
	var version uint
	var dirty bool
	err := mr.withMigrate(ctx, func(m *migrate.Migrate, _ *sql.DB) error {
		var err error
		version, dirty, err = m.Version()
		if errors.Is(err, migrate.ErrNilVersion) {
			version, dirty = 0, false
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", errWrapGetMigrationVersion, err)
		}
		return nil
	})
	if err != nil {
		return 0, false, err
	}
	return version, dirty, nil
}

// withMigrate runs fn on a migrate instance over a dedicated short-lived
// connection and reports a failure to close either through the returned error.
func (mr *MigrationRunner) withMigrate(ctx context.Context, fn func(*migrate.Migrate, *sql.DB) error) (err error) {
	db, err := mr.openMigrationConnection(ctx)
	if err != nil {
		return err
	}
	defer func() { err = joinCloseError(err, errWrapCloseMigrationConnection, db.Close()) }()

	m, err := mr.getMigrate(db)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapFailedToCreateMigrateInstance, err)
	}
	defer func() {
		sourceErr, databaseErr := m.Close()
		err = joinCloseError(err, errWrapCloseMigrationSource, sourceErr)
		err = joinCloseError(err, errWrapCloseMigrationDatabase, databaseErr)
	}()
	return fn(m, db)
}

// joinCloseError joins a close failure, wrapped with wrap, onto err.
func joinCloseError(err error, wrap string, closeErr error) error {
	if closeErr == nil {
		return err
	}
	return errors.Join(err, fmt.Errorf("%s: %w", wrap, closeErr))
}

// getMigrate creates a new migrate instance with the provided connection
func (mr *MigrationRunner) getMigrate(db *sql.DB) (*migrate.Migrate, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{
		DatabaseName: mr.config.Database,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapFailedToCreatePostgresDriver, err)
	}

	source, err := iofs.New(migrations.MigrationsFS, ".")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapCreateMigrationSource, err)
	}

	m, err := migrate.NewWithInstance("iofs", source, mr.config.Database, driver)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapFailedToCreateMigrateInstance, err)
	}

	return m, nil
}

// Down rolls back all migrations
func (mr *MigrationRunner) Down(ctx context.Context) error {
	return mr.withMigrate(ctx, func(m *migrate.Migrate, _ *sql.DB) error {
		if err := m.Down(); err != nil {
			return fmt.Errorf("%s: %w", errWrapRollbackAllMigrations, err)
		}
		return nil
	})
}

// Steps runs n migration steps (positive = up, negative = down)
func (mr *MigrationRunner) Steps(ctx context.Context, n int) error {
	return mr.withMigrate(ctx, func(m *migrate.Migrate, _ *sql.DB) error {
		if err := m.Steps(n); err != nil {
			return fmt.Errorf(errFmtRunMigrationSteps, n, err)
		}
		return nil
	})
}

// Force sets the migration version without running actual migrations.
// Useful for recovering from a dirty state.
func (mr *MigrationRunner) Force(ctx context.Context, version int) error {
	return mr.withMigrate(ctx, func(m *migrate.Migrate, _ *sql.DB) error {
		if err := m.Force(version); err != nil {
			return fmt.Errorf(errFmtForceVersion, version, err)
		}
		return nil
	})
}
