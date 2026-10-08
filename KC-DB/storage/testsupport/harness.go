// Package testsupport owns the shared PostgreSQL test harness: one container
// per test binary, migrations applied once to a template database, and a
// millisecond-cheap clone of that template per test. It is a public package so
// every module's database-backed tests - KC-DB repositories and adapters, and
// KC-Core protocol integration tests - share one lifecycle instead of
// hardcoding ports and downgrading connection failures to skips.
//
// TEST_DB_* environment variables switch the harness onto an externally
// managed server as an explicit developer override; the template/clone flow is
// identical there. Absent the override, the isolated container starts, and any
// setup failure fails the test - never a skip.
package testsupport

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
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

	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// postgresImage is the canonical PostgreSQL version for the repository. Keep it
// in step with docker-compose.yml, compose.gates.yml and the CI service
// container; a version skew between them hides ordering and syntax differences
// until deployment.
const postgresImage = "docker.io/postgres:18.1-alpine"

// containerStartupTimeout bounds the shared postgres container boot.
const containerStartupTimeout = 60 * time.Second

// Names of the fixed databases inside the shared container. Per-test databases
// are cloned from templateDatabase and are named with testDatabasePrefix.
const (
	maintenanceDatabase = "kilocenter_test"
	templateDatabase    = "kilocenter_template"
	testDatabasePrefix  = "kc_test_"
)

// TestContainerConfig holds parsed connection details from testcontainer DSN
type TestContainerConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	// DSN is the connection string the config was parsed from.
	DSN string
}

// The package shares a single PostgreSQL container across all of its tests.
// Migrations are applied once to a template database; every test then clones
// that template, which is a file copy inside the server rather than a fresh
// container start plus a full migration run. Isolation is unchanged - each test
// still gets its own database - but the per-test cost drops from tens of
// seconds to milliseconds, which is what keeps a non-short `go test ./...`
// inside Go's default timeout.
//
// Startup is lazy so that `go test -short`, where nearly every database test
// skips, never starts a container at all.
var (
	sharedOnce      sync.Once
	sharedContainer *testcontainerspostgres.PostgresContainer
	sharedAdminDB   *sqlx.DB
	sharedBaseDSN   string
	sharedInitErr   error
	testDatabaseSeq atomic.Uint64
)

// Main wraps a package's TestMain run and tears the shared container down
// afterwards. Every package using this harness calls it:
//
//	func TestMain(m *testing.M) { os.Exit(testsupport.Main(m)) }
func Main(m *testing.M) int {
	code := m.Run()
	teardownSharedContainer()
	return code
}

func teardownSharedContainer() {
	if sharedContainer == nil && sharedAdminDB != nil {
		dropSharedTemplate()
	}
	if sharedAdminDB != nil {
		if err := sharedAdminDB.Close(); err != nil {
			fmt.Fprintf(os.Stderr, warnFmtCloseSharedAdminDB, err)
		}
	}
	if sharedContainer != nil {
		if err := sharedContainer.Terminate(testutil.TestContext()); err != nil {
			fmt.Fprintf(os.Stderr, warnFmtTerminateContainer, err)
		}
	}
}

// ensureSharedContainer starts the container and builds the migrated template
// database on first use.
func ensureSharedContainer(t *testing.T) {
	t.Helper()

	sharedOnce.Do(func() {
		ctx := testutil.TestContext()

		// Explicit developer override: an externally managed server. The
		// template/clone flow is unchanged; only the base DSN differs.
		if host := os.Getenv("TEST_DB_HOST"); host != "" {
			port := envOr("TEST_DB_PORT", defaultPostgresPort)
			user := envOr("TEST_DB_USER", "kilocenter")
			pass := envOr("TEST_DB_PASSWORD", "changeme")
			dbname := envOr("TEST_DB_NAME", "postgres")
			sslmode := envOr("TEST_DB_SSLMODE", "disable")
			baseDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, dbname, sslmode)
			sharedBaseDSN = baseDSN
			adminDB, err := sqlx.Connect("postgres", baseDSN)
			if err != nil {
				sharedInitErr = fmt.Errorf("%s: %w", errWrapConnectTESTDBOverride, err)
				return
			}
			sharedAdminDB = adminDB
			sharedInitErr = prepareTemplate(baseDSN, adminDB)
			return
		}

		container, err := testcontainerspostgres.Run(ctx,
			postgresImage,
			testcontainerspostgres.WithDatabase(maintenanceDatabase),
			testcontainerspostgres.WithUsername("kilocenter"),
			testcontainerspostgres.WithPassword("changeme"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(containerStartupTimeout)),
		)
		if err != nil {
			sharedInitErr = fmt.Errorf("%s: %w", errWrapStartPostgresContainer, err)
			return
		}
		sharedContainer = container

		baseDSN, err := container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			sharedInitErr = fmt.Errorf("%s: %w", errWrapReadContainerConnectionString, err)
			return
		}
		sharedBaseDSN = baseDSN

		// The admin connection targets the maintenance database so that CREATE
		// DATABASE ... TEMPLATE never runs while connected to the template.
		adminDB, err := sqlx.Connect("postgres", baseDSN)
		if err != nil {
			sharedInitErr = fmt.Errorf("%s: %w", errWrapConnectMaintenanceDatabase, err)
			return
		}
		sharedAdminDB = adminDB

		sharedInitErr = prepareTemplate(baseDSN, adminDB)
	})

	require.NoError(t, sharedInitErr, "Failed to prepare shared PostgreSQL container")
}

// prepareTemplate creates and migrates the template database on the server
// behind baseDSN.
func prepareTemplate(baseDSN string, adminDB *sqlx.DB) error {
	if _, err := adminDB.Exec("DROP DATABASE IF EXISTS " + templateName() + " WITH (FORCE)"); err != nil {
		return fmt.Errorf("%s: %w", errWrapDropStaleTemplateDatabase, err)
	}
	if _, err := adminDB.Exec("CREATE DATABASE " + templateName()); err != nil {
		return fmt.Errorf("%s: %w", errWrapCreateTemplateDatabase, err)
	}
	templateDB, err := sqlx.Connect("postgres", dsnForDatabase(baseDSN, templateName()))
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapConnectTemplateDatabase, err)
	}
	migrateErr := runMigrationsWithLibrary(templateDB.DB)
	// The template must have no open connections before it can be cloned.
	closeErr := templateDB.Close()
	if migrateErr != nil {
		return fmt.Errorf("%s: %w", errWrapMigrateTemplateDatabase, migrateErr)
	}
	if closeErr != nil {
		return fmt.Errorf("%s: %w", errWrapCloseTemplateDatabase, closeErr)
	}
	return nil
}

// envOr returns the environment value or a fallback.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// newTestDatabase creates a database for a single test and returns its name and
// DSN. When template is empty the database starts out with no schema.
func newTestDatabase(t *testing.T, template string) (string, string) {
	t.Helper()
	ensureSharedContainer(t)

	name := testDatabaseName(testDatabaseSeq.Add(1))
	stmt := "CREATE DATABASE " + name
	if template != "" {
		stmt += " TEMPLATE " + template
	}
	_, err := sharedAdminDB.Exec(stmt)
	require.NoErrorf(t, err, "Failed to create test database %s", name)

	return name, dsnForDatabase(sharedBaseDSN, name)
}

// dropTestDatabase removes a per-test database. FORCE detaches any connection
// the test left behind so one leak cannot cascade into later failures.
func dropTestDatabase(t *testing.T, name string) {
	t.Helper()
	if _, err := sharedAdminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", name)); err != nil {
		t.Logf(warnFmtDropTestDatabase, name, err)
	}
}

// SetupPostgresContainerWithoutMigrations provides an empty database for tests
// that apply migrations themselves.
// Returns *sqlx.DB connection, parsed config with dynamic port, and cleanup function
func SetupPostgresContainerWithoutMigrations(t *testing.T) (*sqlx.DB, *TestContainerConfig, func()) {
	name, dsn := newTestDatabase(t, "")

	config, err := parseDSN(dsn)
	require.NoError(t, err, "Failed to parse DSN")

	db, err := sqlx.Connect("postgres", dsn)
	require.NoError(t, err, "Failed to connect to test database")

	cleanup := func() {
		if err := db.Close(); err != nil {
			t.Logf(warnFmtCloseDBConnection, err)
		}
		dropTestDatabase(t, name)
	}

	return db, config, cleanup
}

// SetupPostgresContainer provides a fully migrated database cloned from the
// package template.
// Returns *sqlx.DB connection and cleanup function
func SetupPostgresContainer(t *testing.T) (*sqlx.DB, func()) {
	name, dsn := newTestDatabase(t, templateName())

	db, err := sqlx.Connect("postgres", dsn)
	require.NoError(t, err, "Failed to connect to test database")

	// Verify migration 000080 applied (global uniqueness constraints exist).
	// Without these constraints, cross-tenant duplicate tests will produce false negatives.
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

	cleanup := func() {
		if err := db.Close(); err != nil {
			t.Logf(warnFmtCloseDBConnection, err)
		}
		dropTestDatabase(t, name)
	}

	return db, cleanup
}

// dsnForDatabase rewrites the database component of a PostgreSQL DSN.
func dsnForDatabase(dsn, database string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	u.Path = "/" + database
	return u.String()
}

// parseDSN extracts connection details from PostgreSQL DSN
// Example: postgres://user:pass@localhost:<mapped-port>/dbname?sslmode=disable
func parseDSN(dsn string) (*TestContainerConfig, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapParseDSN, err)
	}

	password, _ := u.User.Password()

	config := &TestContainerConfig{
		Host:     u.Hostname(),
		Port:     u.Port(),
		User:     u.User.Username(),
		Password: password,
		Database: u.Path[1:], // Remove leading slash
		DSN:      dsn,
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
		return errTextGetCurrentFilePath
	}

	// Navigate from KC-DB/storage/postgres/testcontainer.go to KC-DB/migrations
	migrationsPath := filepath.Join(filepath.Dir(filename), "..", "..", "migrations")

	// Create postgres driver instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapCreatePostgresDriver, err)
	}

	// Create migrate instance with file source
	sourceURL := fmt.Sprintf("file://%s", migrationsPath)

	m, err := migrate.NewWithDatabaseInstance(
		sourceURL,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapCreateMigrateInstance, err)
	}
	// The postgres driver checks out a dedicated connection for its advisory
	// lock and holds it until closed. Closing the sqlx pool alone leaves that
	// connection open, which later blocks CREATE DATABASE ... TEMPLATE with
	// "source database is being accessed by other users".
	defer func() {
		if sourceErr, dbErr := m.Close(); sourceErr != nil || dbErr != nil {
			fmt.Fprintf(os.Stderr, warnFmtCloseMigrateInstance, sourceErr, dbErr)
		}
	}()

	// Run all migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("%s: %w", errWrapMigration, err)
	}

	return nil
}

// NewMigratedDatabase creates a migrated per-test database and returns its
// DSN plus a cleanup dropping it. For callers that open their own connection
// type over the database (e.g. the concrete store).
func NewMigratedDatabase(t *testing.T) (string, func()) {
	name, dsn := newTestDatabase(t, templateName())
	return dsn, func() { dropTestDatabase(t, name) }
}

// ParseDSN exposes DSN parsing for callers building typed configs.
func ParseDSN(dsn string) (*TestContainerConfig, error) { return parseDSN(dsn) }

// dropSharedTemplate removes this process's template from an external server,
// which outlives the test run; a container takes it along when it stops.
func dropSharedTemplate() {
	if _, err := sharedAdminDB.Exec("DROP DATABASE IF EXISTS " + templateName() + " WITH (FORCE)"); err != nil {
		fmt.Fprintf(os.Stderr, warnFmtDropTemplateDatabase, err)
	}
}
