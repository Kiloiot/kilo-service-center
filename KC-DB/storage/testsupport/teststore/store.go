// Package teststore opens the concrete PostgreSQL store over a migrated
// per-test database from the shared harness. It lives beside testsupport so
// the postgres package's own tests can use the harness without a cycle.
package teststore

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
)

// warnFmtCloseTestStore reports a store that could not be closed at cleanup;
// the failure never fails the test.
const warnFmtCloseTestStore = "Warning: failed to close test store: %v"

// Setup provides a fully migrated per-test database opened as *postgres.DB.
// Any setup failure fails the test.
func Setup(t *testing.T) (*postgres.DB, func()) {
	t.Helper()
	dsn, drop := testsupport.NewMigratedDatabase(t)
	cfg, err := testsupport.ParseDSN(dsn)
	require.NoError(t, err, "parse test database DSN")

	port, err := strconv.Atoi(cfg.Port)
	require.NoError(t, err, "parse test database port")
	store, err := postgres.New(postgres.Options{
		Host:     cfg.Host,
		Port:     port,
		Username: cfg.User,
		Password: cfg.Password,
		Database: cfg.Database,
		SSLMode:  "disable",
	}, testsupport.TestCipher())
	require.NoError(t, err, "open test store")

	cleanup := func() {
		if err := store.Close(); err != nil {
			t.Logf(warnFmtCloseTestStore, err)
		}
		drop()
	}
	return store, cleanup
}
