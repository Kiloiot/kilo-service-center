package postgres

import (
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Register file source driver for migrations
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Register PostgreSQL driver
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	testcontainerspostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// TestContainerConfig holds parsed connection details from testcontainer DSN
type TestContainerConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

// SetupPostgresContainerWithoutMigrations starts a PostgreSQL testcontainer without running migrations
// Useful for migration tests that need to test applying migrations themselves
// Returns *sqlx.DB connection, parsed config with dynamic port, and cleanup function
func SetupPostgresContainerWithoutMigrations(t *testing.T) (*sqlx.DB, *TestContainerConfig, func()) {
	ctx := testutil.TestContext()

	// Start PostgreSQL container
	postgresContainer, err := testcontainerspostgres.Run(ctx,
		"docker.io/postgres:17.5-alpine",
		testcontainerspostgres.WithDatabase("kilocenter_test"),
		testcontainerspostgres.WithUsername("kilocenter"),
		testcontainerspostgres.WithPassword("changeme"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")

	// Get connection string from container with sslmode disabled
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "Failed to get connection string")

	// Parse DSN to extract connection details (including dynamic port)
	config, err := parseDSN(connStr)
	require.NoError(t, err, "Failed to parse DSN")

	// Connect using sqlx
	db, err := sqlx.Connect("postgres", connStr)
	require.NoError(t, err, "Failed to connect to test database")

	// Return cleanup function (no migrations run)
	cleanup := func() {
		if err := db.Close(); err != nil {
			t.Logf("Warning: failed to close database connection: %v", err)
		}
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Warning: failed to terminate container: %v", err)
		}
	}

	return db, config, cleanup
}

// SetupPostgresContainer starts a PostgreSQL testcontainer and runs migrations
// Returns *sqlx.DB connection and cleanup function
// The DSN is parsed to extract dynamic host/port for CI compatibility
func SetupPostgresContainer(t *testing.T) (*sqlx.DB, func()) {
	ctx := testutil.TestContext()

	// Start PostgreSQL container
	postgresContainer, err := testcontainerspostgres.Run(ctx,
		"docker.io/postgres:17.5-alpine",
		testcontainerspostgres.WithDatabase("kilocenter_test"),
		testcontainerspostgres.WithUsername("kilocenter"),
		testcontainerspostgres.WithPassword("changeme"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")

	// Get connection string from container with sslmode disabled
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "Failed to get connection string")

	// Connect using sqlx
	db, err := sqlx.Connect("postgres", connStr)
	require.NoError(t, err, "Failed to connect to test database")

	// Run migrations using golang-migrate library
	err = runMigrationsWithLibrary(db.DB)
	require.NoError(t, err, "Failed to run migrations")

	// **CRITICAL**: Verify migration 000080 applied (global uniqueness constraints exist)
	// Without these constraints, cross-tenant duplicate tests will produce false negatives
	var epConstraintExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_constraint
			WHERE conname = 'unique_ep_eui'
		)
	`).Scan(&epConstraintExists)
	require.NoError(t, err, "Failed to check migration 000080 ep_eui constraint")
	require.True(t, epConstraintExists, "Migration 000080 did not create unique_ep_eui constraint - tests will produce false negatives")

	var bsConstraintExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_constraint
			WHERE conname = 'unique_bs_eui'
		)
	`).Scan(&bsConstraintExists)
	require.NoError(t, err, "Failed to check migration 000080 bs_eui constraint")
	require.True(t, bsConstraintExists, "Migration 000080 did not create unique_bs_eui constraint - tests will produce false negatives")

	// Return cleanup function
	cleanup := func() {
		if err := db.Close(); err != nil {
			t.Logf("Warning: failed to close database connection: %v", err)
		}
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Warning: failed to terminate container: %v", err)
		}
	}

	return db, cleanup
}

// parseDSN extracts connection details from PostgreSQL DSN
// Example: postgres://user:pass@localhost:32768/dbname?sslmode=disable
func parseDSN(dsn string) (*TestContainerConfig, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	password, _ := u.User.Password()

	config := &TestContainerConfig{
		Host:     u.Hostname(),
		Port:     u.Port(),
		User:     u.User.Username(),
		Password: password,
		Database: u.Path[1:], // Remove leading slash
	}

	return config, nil
}

// runMigrationsWithLibrary runs database migrations using golang-migrate library
// Takes *sql.DB and applies all migrations from the migrations directory
// No external binary required - uses golang-migrate/migrate/v4
func runMigrationsWithLibrary(db *sql.DB) error {
	// Find migrations directory relative to this file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("failed to get current file path")
	}

	// Navigate from KC-DB/storage/postgres/testcontainer.go to KC-DB/migrations
	migrationsPath := filepath.Join(filepath.Dir(filename), "..", "..", "migrations")

	// Create postgres driver instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	// Create migrate instance with file source
	sourceURL := fmt.Sprintf("file://%s", migrationsPath)

	m, err := migrate.NewWithDatabaseInstance(
		sourceURL,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Run all migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}
