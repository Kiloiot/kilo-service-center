package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/migrations"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// MigrationRunner handles database migrations using dedicated short-lived connections.
// This design ensures that m.Close() can be safely called without affecting the main
// storage pool used by the application.
type MigrationRunner struct {
	config *Config
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(config *Config) *MigrationRunner {
	return &MigrationRunner{config: config}
}

// openMigrationConnection opens a dedicated short-lived connection for migrations
// with connectivity validation via Ping. This ensures clear error messages when
// credentials/host/SSL are misconfigured instead of failing deep in migrate.
func (mr *MigrationRunner) openMigrationConnection() (*sql.DB, error) {
	db, err := sql.Open("postgres", mr.config.GetDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open migration connection: %w", err)
	}

	// Validate connectivity with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping migration connection: %w", err)
	}

	return db, nil
}

// Run executes all pending migrations and returns the final version.
// Uses a dedicated short-lived DB connection to avoid closing the main pool.
func (mr *MigrationRunner) Run() (uint, error) {
	db, err := mr.openMigrationConnection()
	if err != nil {
		return 0, err
	}
	defer func() { _ = db.Close() }()

	m, err := mr.getMigrate(db)
	if err != nil {
		return 0, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		_, _ = m.Close() // Safe to close - dedicated connection
	}()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return 0, fmt.Errorf("failed to run migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, fmt.Errorf("failed to get migration version: %w", err)
	}

	if dirty {
		return 0, fmt.Errorf("database is in dirty state at version %d", version)
	}

	return version, nil
}

// Version returns the current migration version
func (mr *MigrationRunner) Version() (uint, bool, error) {
	db, err := mr.openMigrationConnection()
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = db.Close() }()

	m, err := mr.getMigrate(db)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	version, dirty, err := m.Version()
	if err == migrate.ErrNilVersion {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}

// Rollback rolls back the last migration
func (mr *MigrationRunner) Rollback() error {
	db, err := mr.openMigrationConnection()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	m, err := mr.getMigrate(db)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Steps(-1); err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	return nil
}

// Reset drops all tables and reruns migrations
func (mr *MigrationRunner) Reset() error {
	db, err := mr.openMigrationConnection()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	m, err := mr.getMigrate(db)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Drop(); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	if err := m.Up(); err != nil {
		return fmt.Errorf("failed to run migrations after reset: %w", err)
	}

	return nil
}

// getMigrate creates a new migrate instance with the provided connection
func (mr *MigrationRunner) getMigrate(db *sql.DB) (*migrate.Migrate, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{
		DatabaseName: mr.config.Database,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	source, err := iofs.New(migrations.MigrationsFS, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, mr.config.Database, driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return m, nil
}

// Down rolls back all migrations
func (mr *MigrationRunner) Down() error {
	db, err := mr.openMigrationConnection()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	m, err := mr.getMigrate(db)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Down(); err != nil {
		return fmt.Errorf("failed to rollback all migrations: %w", err)
	}

	return nil
}

// Steps runs n migration steps (positive = up, negative = down)
func (mr *MigrationRunner) Steps(n int) error {
	db, err := mr.openMigrationConnection()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	m, err := mr.getMigrate(db)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Steps(n); err != nil {
		return fmt.Errorf("failed to run %d migration steps: %w", n, err)
	}

	return nil
}

// Force sets the migration version without running actual migrations.
// Useful for recovering from a dirty state.
func (mr *MigrationRunner) Force(version int) error {
	db, err := mr.openMigrationConnection()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	m, err := mr.getMigrate(db)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Force(version); err != nil {
		return fmt.Errorf("failed to force version %d: %w", version, err)
	}

	return nil
}

// RunMigrationsFromPath runs migrations from a file system path (for development)
func RunMigrationsFromPath(db *sql.DB, config *Config, path string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{
		DatabaseName: config.Database,
	})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", path),
		config.Database,
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	// Note: Do NOT defer m.Close() here because it closes the *sql.DB which is owned by the caller
	// The migrate instance will be garbage collected after this function returns

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
