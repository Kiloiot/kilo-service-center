package postgres

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMigrationTestDB(t *testing.T) (*sql.DB, *Config, func()) {
	// Use testcontainers for isolated database (without running migrations)
	db, containerConfig, cleanup := SetupPostgresContainerWithoutMigrations(t)

	// Convert containerConfig.Port (string) to int
	port := 5432 // Default fallback
	if containerConfig.Port != "" {
		var parsedPort int
		if _, err := fmt.Sscanf(containerConfig.Port, "%d", &parsedPort); err == nil {
			port = parsedPort
		}
	}

	config := &Config{
		Host:     containerConfig.Host,
		Port:     port,
		Database: containerConfig.Database,
		Username: containerConfig.User,
		Password: containerConfig.Password,
		SSLMode:  "disable",
	}

	// Return sql.DB and Config for compatibility with migration tests
	return db.DB, config, cleanup
}

func TestMigrations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping migration tests in short mode")
	}

	// Test running migrations from path (for development)
	t.Run("RunMigrationsFromPath", func(t *testing.T) {
		// Fresh connection per subtest to avoid "database is closed" errors
		db, config, cleanup := setupMigrationTestDB(t)
		defer cleanup()

		err := RunMigrationsFromPath(db, config, "../../migrations")
		assert.NoError(t, err)

		// Verify tables exist (updated names: gateway→basestation, device→endpoint)
		tables := []string{
			"schema_migrations",
			"tenants",
			"endpoints",
			"basestations",
			"messages",
			"messages_archive",
			"downlink_queue",
			"roaming_agreements",
			"basestation_receptions",
			"endpoint_sessions",
			"endpoint_keys",
			"basestation_sessions",
			"system_events",
		}

		for _, table := range tables {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM information_schema.tables
					WHERE table_schema = 'public'
					AND table_name = $1
				)`, table).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Table %s should exist", table)
		}
	})

	// Test specific table structures
	t.Run("VerifyTableStructures", func(t *testing.T) {
		// Fresh connection per subtest
		db, config, cleanup := setupMigrationTestDB(t)
		defer cleanup()

		// Run migrations first
		err := RunMigrationsFromPath(db, config, "../../migrations")
		require.NoError(t, err)

		// Check endpoints table has MIOTY-specific columns
		endpointColumns := []string{
			"bidi", "pre_attach", "sh_addr", "attach_cnt",
			"packet_cnt", "dual_chan", "repetition",
			"wide_carr_off", "long_blk_dist", "ep_status",
		}

		for _, col := range endpointColumns {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM information_schema.columns
					WHERE table_name = 'endpoints'
					AND column_name = $1
				)`, col).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Column endpoints.%s should exist", col)
		}

		// Check basestations table has MIOTY-specific columns (current schema)
		basestationColumns := []string{
			"connection_type", "service_center_url", "is_online",
			"session_uuid", "vendor", "model", "sw_version",
			"bidi", "system_time", "duty_cycle",
		}

		for _, col := range basestationColumns {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM information_schema.columns
					WHERE table_name = 'basestations'
					AND column_name = $1
				)`, col).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Column basestations.%s should exist", col)
		}

		// Check downlink_queue has MIOTY enhancements
		downlinkColumns := []string{
			"dl_open", "res_exp", "dl_ack", "repetition",
			"earliest_at", "latest_at", "correlation_id",
		}

		for _, col := range downlinkColumns {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM information_schema.columns 
					WHERE table_name = 'downlink_queue' 
					AND column_name = $1
				)`, col).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Column downlink_queue.%s should exist", col)
		}
	})

	// Test indexes exist
	t.Run("VerifyIndexes", func(t *testing.T) {
		// Fresh connection per subtest
		db, config, cleanup := setupMigrationTestDB(t)
		defer cleanup()

		// Run migrations first
		err := RunMigrationsFromPath(db, config, "../../migrations")
		require.NoError(t, err)

		// Verify representative indexes that survive all migrations
		// (Some early indexes are dropped when tables are recreated in later migrations)
		indexes := []string{
			"idx_basestations_online",          // From migration 001, not recreated
			"idx_basestation_receptions_dedup", // From migration 010, table not recreated
			"idx_system_events_search",         // From migration 008, table not recreated
			"idx_messages_dedup",               // From migration 047 (messages table recreated)
			"idx_tenants_status",               // From migration 053 (tenants table)
		}

		for _, idx := range indexes {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM pg_indexes
					WHERE schemaname = 'public'
					AND indexname = $1
				)`, idx).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Index %s should exist", idx)
		}
	})

	// Test constraints
	t.Run("VerifyConstraints", func(t *testing.T) {
		// Fresh connection per subtest
		db, config, cleanup := setupMigrationTestDB(t)
		defer cleanup()

		// Run migrations first
		err := RunMigrationsFromPath(db, config, "../../migrations")
		require.NoError(t, err)

		// Test unique constraint on active endpoint sessions
		// Use integer IDs since tenant_id and endpoint_id are BIGINT
		// Use id=100 to avoid conflict with default tenant (id=1) from migration 053
		_, err = db.Exec(`
			INSERT INTO tenants (id, name, status)
			VALUES (100, 'test', 'active')`)
		require.NoError(t, err)

		// ep_eui is BYTEA type, use decode to convert hex string to bytea
		// owner_tenant_id is required after migration 075 (roaming support)
		_, err = db.Exec(`
			INSERT INTO endpoints (id, tenant_id, owner_tenant_id, ep_eui, name, endpoint_class)
			VALUES (100, 100, 100, decode('0123456789ABCDEF', 'hex'), 'test-endpoint', 'A')`)
		require.NoError(t, err)

		// Insert first active session
		_, err = db.Exec(`
			INSERT INTO endpoint_sessions (endpoint_id, tenant_id, session_id, attach_cnt, status)
			VALUES (100, 100, '33333333-3333-3333-3333-333333333333', 1, 'active')`)
		require.NoError(t, err)

		// Try to insert second active session - should fail
		_, err = db.Exec(`
			INSERT INTO endpoint_sessions (endpoint_id, tenant_id, session_id, attach_cnt, status)
			VALUES (100, 100, '44444444-4444-4444-4444-444444444444', 2, 'active')`)
		assert.Error(t, err, "Should not allow multiple active sessions per endpoint")
	})

	// Test functions
	t.Run("VerifyFunctions", func(t *testing.T) {
		// Fresh connection per subtest
		db, config, cleanup := setupMigrationTestDB(t)
		defer cleanup()

		// Run migrations first
		err := RunMigrationsFromPath(db, config, "../../migrations")
		require.NoError(t, err)

		// Verify representative functions created by migrations
		functions := []string{
			"update_endpoint_session_activity", // From migration 005
			"ensure_single_active_key",         // From migration 006
			"expire_old_system_events",         // From migration 008
		}

		for _, fn := range functions {
			var exists bool
			err := db.QueryRow(`
				SELECT EXISTS (
					SELECT FROM pg_proc
					WHERE proname = $1
				)`, fn).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Function %s should exist", fn)
		}
	})
}

func TestMigrationRunner(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping migration runner tests in short mode")
	}

	t.Run("Version", func(t *testing.T) {
		// Fresh connection per subtest
		_, config, cleanup := setupMigrationTestDB(t)
		defer cleanup()

		runner := NewMigrationRunner(config)

		// Initially no version
		version, dirty, err := runner.Version()
		assert.NoError(t, err)
		assert.Equal(t, uint(0), version)
		assert.False(t, dirty)
	})

	// Note: The embedded filesystem test would require the actual binary
	// For now, we test with the file path approach
	t.Run("RunFromPath", func(t *testing.T) {
		// Fresh connection per subtest
		db, config, cleanup := setupMigrationTestDB(t)
		defer cleanup()

		// Use container config instead of hardcoded testConfig
		err := RunMigrationsFromPath(db, config, "../../migrations")
		assert.NoError(t, err)

		// Check final version against the highest discovered migration number
		migrations := discoverMigrations(t)
		require.NotEmpty(t, migrations)
		expectedVersion := migrations[len(migrations)-1].number

		var version int
		err = db.QueryRow("SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&version)
		require.NoError(t, err)
		assert.Equal(t, expectedVersion, version,
			"Final migration version should match the highest migration file")
	})
}

// migrateToVersion runs migrations up to (and including) the given version using golang-migrate.
func migrateToVersion(db *sql.DB, config *Config, version uint) error {
	driver, err := migratepostgres.WithInstance(db, &migratepostgres.Config{
		DatabaseName: config.Database,
	})
	if err != nil {
		return fmt.Errorf("create postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", "../../migrations"),
		config.Database,
		driver,
	)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	if err := m.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate to version %d: %w", version, err)
	}
	return nil
}

// TestMigration110_OrphanRename verifies that migration 000110 handles all three
// collision vectors when orphan user-keys are converted to service-account keys:
//
//  1. Two orphan keys in the same org with identical names (orphan-orphan)
//  2. An orphan key sharing a name with an existing service-account key (orphan-vs-SA)
//  3. After migration, FK constraint api_keys_user_id_fkey exists
//  4. No data is lost (all 3 keys survive)
func TestMigration110_OrphanRename(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, config, cleanup := setupMigrationTestDB(t)
	defer cleanup()

	// Run migrations up to 109 (just before the FK migration)
	err := migrateToVersion(db, config, 109)
	require.NoError(t, err, "failed to migrate to version 109")

	// Set up test data: tenant, org
	// Use tenant ID 200 to avoid conflict with tenant 100 used by other tests
	tenantID := int64(200)
	_, err = db.Exec(`
		INSERT INTO tenants (id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, tenantID, "Migration110 Test Tenant", "Test tenant for migration 110")
	require.NoError(t, err)

	orgID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	_, err = db.Exec(`
		INSERT INTO organizations (org_id, tenant_id, name, can_have_base_stations, created_at, updated_at)
		VALUES ($1, $2, 'Migration Test Org', true, NOW(), NOW())
		ON CONFLICT (org_id) DO NOTHING
	`, orgID, tenantID)
	require.NoError(t, err)

	// Ghost user IDs — neither exists in users table (both are orphans).
	// Must be distinct to satisfy the pre-existing api_keys_user_name_uq partial
	// unique index ON (user_id, name) WHERE user_id IS NOT NULL.
	ghostUserID1 := uuid.MustParse("deadbeef-dead-beef-dead-beefdeadbeef")
	ghostUserID2 := uuid.MustParse("deadbeef-dead-beef-dead-beefdeadcafe")

	// Insert collision seed data: 3 keys all named "my-key" in same org
	orphan1ID := uuid.New()
	orphan2ID := uuid.New()
	saKeyID := uuid.New()

	// Orphan 1: user-type key referencing non-existent user
	_, err = db.Exec(`
		INSERT INTO api_keys (id, tenant_id, org_id, user_id, name, key_hash, key_prefix, key_type, is_active, created_at)
		VALUES ($1, $2, $3, $4, 'my-key', $5, 'kc_aaaa', 'user', true, NOW())
	`, orphan1ID, tenantID, orgID, ghostUserID1, fmt.Sprintf("hash_%s", orphan1ID))
	require.NoError(t, err, "insert orphan 1")

	// Orphan 2: different orphan user, same org+name (orphan-orphan collision vector)
	_, err = db.Exec(`
		INSERT INTO api_keys (id, tenant_id, org_id, user_id, name, key_hash, key_prefix, key_type, is_active, created_at)
		VALUES ($1, $2, $3, $4, 'my-key', $5, 'kc_bbbb', 'user', true, NOW())
	`, orphan2ID, tenantID, orgID, ghostUserID2, fmt.Sprintf("hash_%s", orphan2ID))
	require.NoError(t, err, "insert orphan 2")

	// Existing service-account key: user_id IS NULL, same name (orphan-vs-SA vector)
	_, err = db.Exec(`
		INSERT INTO api_keys (id, tenant_id, org_id, user_id, name, key_hash, key_prefix, key_type, is_active, created_at)
		VALUES ($1, $2, $3, NULL, 'my-key', $4, 'kc_cccc', 'service_account', true, NOW())
	`, saKeyID, tenantID, orgID, fmt.Sprintf("hash_%s", saKeyID))
	require.NoError(t, err, "insert existing SA key")

	// Run migration 110 — this is the migration under test
	err = migrateToVersion(db, config, 110)
	require.NoError(t, err, "migration 110 should succeed without collision errors")

	// Assertion 1: FK constraint exists
	var fkExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_constraint
			WHERE conname = 'api_keys_user_id_fkey'
			  AND conrelid = 'api_keys'::regclass
		)
	`).Scan(&fkExists)
	require.NoError(t, err)
	require.True(t, fkExists, "FK constraint api_keys_user_id_fkey should exist")

	// Assertion 2: No duplicate (org_id, name) WHERE user_id IS NULL
	var dupCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT org_id, name FROM api_keys
			WHERE user_id IS NULL
			GROUP BY org_id, name
			HAVING COUNT(*) > 1
		) dups
	`).Scan(&dupCount)
	require.NoError(t, err)
	require.Equal(t, 0, dupCount, "should have no duplicate (org_id, name) for NULL user_id keys")

	// Assertion 3: All 3 keys still exist (no data loss)
	var totalKeys int
	err = db.QueryRow(`SELECT COUNT(*) FROM api_keys WHERE id IN ($1, $2, $3)`, orphan1ID, orphan2ID, saKeyID).Scan(&totalKeys)
	require.NoError(t, err)
	require.Equal(t, 3, totalKeys, "all 3 keys should survive the migration")

	// Assertion 4: Orphan keys are deactivated, retyped, and have NULL user_id
	for _, oid := range []uuid.UUID{orphan1ID, orphan2ID} {
		var isActive bool
		var keyType string
		var userID *uuid.UUID
		var name string
		err = db.QueryRow(`
			SELECT is_active, key_type, user_id, name FROM api_keys WHERE id = $1
		`, oid).Scan(&isActive, &keyType, &userID, &name)
		require.NoError(t, err, "query orphan key %s", oid)
		require.False(t, isActive, "orphan key %s should be inactive", oid)
		require.Equal(t, "service_account", keyType, "orphan key %s should be service_account", oid)
		require.Nil(t, userID, "orphan key %s should have NULL user_id", oid)
		require.Contains(t, name, "_orphan_", "orphan key %s should have _orphan_ suffix", oid)
		require.NotEqual(t, "my-key", name, "orphan key %s should have been renamed from 'my-key'", oid)
	}

	// Assertion 5: Orphan keys received distinct names from each other
	var orphan1Name, orphan2Name string
	err = db.QueryRow(`SELECT name FROM api_keys WHERE id = $1`, orphan1ID).Scan(&orphan1Name)
	require.NoError(t, err)
	err = db.QueryRow(`SELECT name FROM api_keys WHERE id = $1`, orphan2ID).Scan(&orphan2Name)
	require.NoError(t, err)
	require.NotEqual(t, orphan1Name, orphan2Name, "migration should produce unique names for each orphan")

	// Assertion 6: Original SA key retains its original name
	var saName string
	err = db.QueryRow(`SELECT name FROM api_keys WHERE id = $1`, saKeyID).Scan(&saName)
	require.NoError(t, err)
	require.Equal(t, "my-key", saName, "original SA key should retain name 'my-key'")
}
