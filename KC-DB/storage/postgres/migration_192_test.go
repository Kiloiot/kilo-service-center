package postgres

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validateTenantSnapshotOnlyDecoding: migration 000192 adds the tenant's
// snapshot-only decoding switch, off unless set.
func validateTenantSnapshotOnlyDecoding(t *testing.T, db *sql.DB) {
	var dataType, nullable string
	var def sql.NullString
	require.NoError(t, db.QueryRow(`SELECT data_type, is_nullable, column_default FROM information_schema.columns
		WHERE table_name = 'tenants' AND column_name = 'is_snapshot_only_decoding'`).Scan(&dataType, &nullable, &def))
	assert.Equal(t, "boolean", dataType)
	assert.Equal(t, "NO", nullable)
	assert.Equal(t, "false", def.String)
}

// TC-CM-102: existing tenants get the switch off; one step down removes it.
func TestMigration192_ExistingTenantsStayOffAndOneStepDownRemovesTheColumn(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping migration test")
	}
	db, _, cleanup := SetupPostgresContainerWithoutMigrations(t)
	defer cleanup()
	migrationsDir, err := filepath.Abs("../../migrations")
	require.NoError(t, err)
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	require.NoError(t, err)
	m, err := migrate.NewWithDatabaseInstance(fmt.Sprintf("file://%s", filepath.ToSlash(migrationsDir)), "postgres", driver)
	require.NoError(t, err)

	require.NoError(t, m.Migrate(191))
	_, err = db.Exec(`INSERT INTO tenants (name, status, created_at, updated_at) VALUES ('before-192', 'active', NOW(), NOW())`)
	require.NoError(t, err)

	require.NoError(t, m.Migrate(192))
	version, dirty, err := m.Version()
	require.NoError(t, err)
	assert.Equal(t, uint(192), version)
	assert.False(t, dirty)
	var enabled bool
	require.NoError(t, db.QueryRow(`SELECT is_snapshot_only_decoding FROM tenants WHERE name = 'before-192'`).Scan(&enabled))
	assert.False(t, enabled, "a tenant that existed before the migration keeps decoding as before")

	require.NoError(t, m.Steps(-1))
	version, dirty, err = m.Version()
	require.NoError(t, err)
	assert.Equal(t, uint(191), version)
	assert.False(t, dirty)
	var columns int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM information_schema.columns
		WHERE table_name = 'tenants' AND column_name = 'is_snapshot_only_decoding'`).Scan(&columns))
	assert.Zero(t, columns, "one step down removes the column")
}
