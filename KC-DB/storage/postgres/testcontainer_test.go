package postgres

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
)

// The shared PostgreSQL harness lives in storage/testsupport; these aliases
// keep this package's many call sites on their established names.

// TestMain tears the shared container down after the package's tests.
func TestMain(m *testing.M) { os.Exit(testsupport.Main(m)) }

// SetupPostgresContainer provides a fully migrated per-test database.
func SetupPostgresContainer(t *testing.T) (*sqlx.DB, func()) {
	return testsupport.SetupPostgresContainer(t)
}

// SetupPostgresContainerWithoutMigrations provides an empty per-test database.
func SetupPostgresContainerWithoutMigrations(t *testing.T) (*sqlx.DB, *testsupport.TestContainerConfig, func()) {
	return testsupport.SetupPostgresContainerWithoutMigrations(t)
}
