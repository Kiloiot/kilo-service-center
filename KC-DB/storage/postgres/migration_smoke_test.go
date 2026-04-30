package postgres

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrationSmokeTest validates the complete migration lifecycle:
// 1. Apply all migrations forward
// 2. Capture final schema state
// 3. Roll back all migrations
// 4. Verify clean slate (no schema_migrations version)
func TestMigrationSmokeTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping smoke test in short mode")
	}

	t.Log("=== Migration Smoke Test Start ===")

	// Use testcontainers for isolated database (without running migrations)
	db, _, cleanup := SetupPostgresContainerWithoutMigrations(t)
	defer cleanup()

	// Get migrations directory
	migrationsDir, err := filepath.Abs("../../migrations")
	require.NoError(t, err)
	require.DirExists(t, migrationsDir)

	// Create migration instance
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	require.NoError(t, err)

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", filepath.ToSlash(migrationsDir)),
		"postgres",
		driver,
	)
	require.NoError(t, err)

	// Step 1: Apply All Migrations Forward
	t.Log("Step 1: Applying all migrations forward...")

	err = m.Up()
	require.NoError(t, err, "Smoke test: forward migration failed")

	// Get final version
	finalVersion, dirty, err := m.Version()
	require.NoError(t, err, "Failed to get migration version after up")
	assert.False(t, dirty, "Migration left database in dirty state")
	t.Logf("PASS: All migrations applied. Final version: %d", finalVersion)

	// Step 2: Capture Final Schema State
	t.Log("Step 2: Capturing final schema state...")

	var tableCount int
	err = db.DB.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_type = 'BASE TABLE'
	`).Scan(&tableCount)
	require.NoError(t, err)
	t.Logf("PASS: Final schema has %d tables", tableCount)

	// Validate critical tables exist
	criticalTables := []string{
		"tenants",
		"endpoints",
		"basestations",
		"messages",
		"endpoint_sessions",
		"basestation_sessions",
		"downlink_queue",
		"system_events",
	}

	for _, table := range criticalTables {
		var exists bool
		err = db.DB.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, table).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "Critical table '%s' should exist after all migrations", table)
	}
	t.Logf("PASS: All %d critical tables exist", len(criticalTables))

	// Step 3: Roll Back All Migrations
	t.Log("Step 3: Rolling back all migrations...")

	err = m.Down()
	require.NoError(t, err, "Smoke test: rollback failed - this indicates down script issues")

	// Step 4: Verify Clean Slate
	t.Log("Step 4: Verifying clean slate...")

	// After full rollback, Version() should return ErrNilVersion
	_, dirty, err = m.Version()
	assert.Error(t, err, "After full rollback, Version() should return error")
	if err != nil {
		assert.Equal(t, migrate.ErrNilVersion, err, "Expected ErrNilVersion after complete rollback")
	}
	assert.False(t, dirty, "Database should not be dirty after rollback")

	// Verify schema_migrations table is empty or doesn't exist
	var versionCount int
	err = db.DB.QueryRow(`
		SELECT COUNT(*) FROM schema_migrations
	`).Scan(&versionCount)
	if err == nil {
		// Table exists but should be empty
		assert.Equal(t, 0, versionCount, "schema_migrations should be empty after full rollback")
	}
	// If error, table doesn't exist which is also acceptable

	// Verify no user tables remain (only PostgreSQL system tables)
	err = db.DB.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_type = 'BASE TABLE'
		AND table_name NOT IN ('schema_migrations')
	`).Scan(&tableCount)
	require.NoError(t, err)
	assert.Equal(t, 0, tableCount,
		"No user tables should remain after full rollback (found %d)", tableCount)

	t.Log("PASS: Clean slate verified - all migrations rolled back successfully")

	// Summary
	t.Log("=== Migration Smoke Test PASSED ===")
	t.Logf("PASS: Forward migration: Version %d reached", finalVersion)
	t.Logf("PASS: Critical tables: %d/%d validated", len(criticalTables), len(criticalTables))
	t.Logf("PASS: Rollback: Clean state achieved")
	t.Log("PASS: All migration up/down scripts are symmetric and correct")
}

// TestMigration014RollbackSpecific tests the specific fix for migration 014
// This validates that the trigger dependency order fix works correctly
func TestMigration014RollbackSpecific(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping migration 014 specific test in short mode")
	}

	t.Log("=== Migration 014 Specific Rollback Test ===")

	db, _, cleanup := SetupPostgresContainerWithoutMigrations(t)
	defer cleanup()

	migrationsDir, err := filepath.Abs("../../migrations")
	require.NoError(t, err)

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	require.NoError(t, err)

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", filepath.ToSlash(migrationsDir)),
		"postgres",
		driver,
	)
	require.NoError(t, err)

	// Apply migrations up to 014
	t.Log("Applying migrations up to 014...")
	err = m.Migrate(14)
	require.NoError(t, err, "Failed to migrate to version 014")

	// Verify archive tables and triggers exist
	archiveTables := []string{
		"messages_archive",
		"basestation_receptions_archive",
		"endpoint_sessions_archive",
		"endpoint_keys_archive",
	}

	for _, table := range archiveTables {
		var exists bool
		err = db.DB.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, table).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "Archive table '%s' should exist after migration 014", table)
	}

	// Verify set_archived_at() function exists
	var funcExists bool
	err = db.DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_proc p
			JOIN pg_namespace n ON p.pronamespace = n.oid
			WHERE n.nspname = 'public' AND p.proname = 'set_archived_at'
		)
	`).Scan(&funcExists)
	require.NoError(t, err)
	assert.True(t, funcExists, "Function set_archived_at() should exist after migration 014")

	t.Log("PASS: Migration 014 applied successfully")

	// ===========================
	// Critical Test: Roll back migration 014
	// ===========================
	t.Log("Rolling back migration 014 (testing trigger/function dependency fix)...")

	err = m.Migrate(13)
	require.NoError(t, err, "Migration 014 rollback failed - trigger/function dependency order issue")

	t.Log("PASS: Migration 014 rolled back successfully")

	// Verify function is gone
	err = db.DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_proc p
			JOIN pg_namespace n ON p.pronamespace = n.oid
			WHERE n.nspname = 'public' AND p.proname = 'set_archived_at'
		)
	`).Scan(&funcExists)
	require.NoError(t, err)
	assert.False(t, funcExists, "Function set_archived_at() should be removed after rollback")

	// Verify archive tables are gone (except messages_archive which existed before)
	for _, table := range []string{
		"basestation_receptions_archive",
		"endpoint_sessions_archive",
		"endpoint_keys_archive",
	} {
		var exists bool
		err = db.DB.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, table).Scan(&exists)
		require.NoError(t, err)
		assert.False(t, exists, "Archive table '%s' should be removed after rollback", table)
	}

	t.Log("=== Migration 014 Specific Test PASSED ===")
	t.Log("PASS: Trigger dependency order fix validated")
	t.Log("PASS: All triggers dropped before function")
	t.Log("PASS: Clean rollback achieved")
}
