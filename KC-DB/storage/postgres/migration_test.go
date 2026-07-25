package postgres

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// migrationTest represents a single migration with optional validation
type migrationTest struct {
	number      int
	description string
	validate    func(*testing.T, *sql.DB)
}

// discoverMigrations scans the migrations directory and returns all available migrations
// sorted by migration number
func discoverMigrations(t *testing.T) []migrationTest {
	migrationsDir, err := filepath.Abs("../../migrations")
	require.NoError(t, err)

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "No migration files found")

	var tests []migrationTest
	for _, f := range files {
		num, desc := extractMigrationInfo(filepath.Base(f))
		if num > 0 {
			tests = append(tests, migrationTest{
				number:      num,
				description: desc,
				validate:    migrationValidators[num], // nil if no validator exists
			})
		}
	}

	// Sort by migration number
	sort.Slice(tests, func(i, j int) bool {
		return tests[i].number < tests[j].number
	})

	return tests
}

// extractMigrationInfo extracts migration number and description from filename
// Example: "014_archive_tables.up.sql" -> (14, "archive_tables")
func extractMigrationInfo(filename string) (int, string) {
	// Remove .up.sql suffix
	name := strings.TrimSuffix(filename, ".up.sql")

	// Extract number and description: "014_archive_tables" -> 14, "archive_tables"
	re := regexp.MustCompile(`^0*(\d+)_(.+)$`)
	matches := re.FindStringSubmatch(name)
	if len(matches) != 3 {
		return 0, ""
	}

	num, _ := strconv.Atoi(matches[1])
	desc := matches[2]

	return num, desc
}

// migrationValidators maps migration numbers to their validation functions
// This allows us to add validators incrementally for critical migrations
var migrationValidators = map[int]func(*testing.T, *sql.DB){
	1:  validateInitialSchema,
	2:  validateBasestationExtensions, // checks migration 002 actual artifacts
	3:  validateEndpointExtensions,    // checks migration 003 actual artifacts
	4:  validateBasestationReceptions, // checks migration 004 actual artifacts
	5:  validateEndpointSessions,      // checks migration 005 actual artifacts
	6:  validateEndpointKeys,          // checks migration 006 actual artifacts
	7:  validateBasestationSessions,   // checks migration 007 actual artifacts
	8:  validatePerformanceIndexes,
	11: validateMessagePartitioning,
	12: validateKeyStorageSchema, // TO UPDATE - needs migration 012 actual artifacts
	13: validateSecurityConstraints,
	14: validateArchiveTables,
	// Additional validators can be added here as they are implemented
	// 28: validateMiotyPersistentCompliance,
	// 31: validateFoo,
	// etc.
}

// TestMigrationUpDown tests that all migrations can be applied and rolled back
func TestMigrationUpDown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping migration test")
	}

	// Use testcontainers for isolated database (without running migrations - this test applies them)
	db, _, cleanup := SetupPostgresContainerWithoutMigrations(t)
	defer cleanup()

	// Get migrations directory
	migrationsDir, err := filepath.Abs("../../migrations")
	require.NoError(t, err)
	require.DirExists(t, migrationsDir)

	// Create migration instance using db.DB (sql.DB)
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	require.NoError(t, err)

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", filepath.ToSlash(migrationsDir)),
		"postgres",
		driver,
	)
	require.NoError(t, err)

	// Test individual migrations
	t.Run("IndividualMigrations", func(t *testing.T) {
		testIndividualMigrations(t, m, db.DB)
	})

	// Test full migration cycle
	t.Run("FullMigrationCycle", func(t *testing.T) {
		testFullMigrationCycle(t, m, db.DB)
	})

	// Test migration idempotency
	t.Run("MigrationIdempotency", func(t *testing.T) {
		testMigrationIdempotency(t, m, db.DB)
	})
}

func testIndividualMigrations(t *testing.T, m *migrate.Migrate, db *sql.DB) {
	// Get current version (should be none)
	_, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		require.NoError(t, err)
	}
	assert.False(t, dirty)

	// Discover all available migrations
	migrations := discoverMigrations(t)
	t.Logf("Testing %d migrations (from discovery)", len(migrations))

	// Count migrations with validators
	withValidators := 0
	for _, mig := range migrations {
		if mig.validate != nil {
			withValidators++
		}
	}
	t.Logf("Migrations with validators: %d/%d (%.1f%%)",
		withValidators, len(migrations), float64(withValidators)/float64(len(migrations))*100)

	// Test each migration up
	for _, mig := range migrations {
		t.Run(fmt.Sprintf("Migration_%03d_%s", mig.number, mig.description), func(t *testing.T) {
			// Migrate up to this version
			err := m.Migrate(uint(mig.number))
			require.NoError(t, err)

			// Validate migration if validator exists
			if mig.validate != nil {
				mig.validate(t, db)
			} else {
				t.Logf("No validator for migration %d - testing version only", mig.number)
			}

			// Check version
			currentVersion, dirty, err := m.Version()
			require.NoError(t, err)
			assert.Equal(t, uint(mig.number), currentVersion)
			assert.False(t, dirty)
		})
	}

	// Test migrating down
	t.Run("MigrateDown", func(t *testing.T) {
		// Migrate down one version at a time
		for i := len(migrations) - 1; i >= 0; i-- {
			targetVersion := uint(0)
			if i > 0 {
				targetVersion = uint(migrations[i-1].number)
			}

			// Special case: when rolling back to version 0, use Down() instead of Migrate(0)
			if targetVersion == 0 {
				err := m.Down()
				// Allow ErrNoChange if already at version 0
				if err != nil && err != migrate.ErrNoChange {
					require.NoError(t, err)
				}
			} else {
				err := m.Migrate(targetVersion)
				require.NoError(t, err)
			}

			// Verify version
			if targetVersion > 0 {
				currentVersion, dirty, err := m.Version()
				require.NoError(t, err)
				assert.Equal(t, targetVersion, currentVersion)
				assert.False(t, dirty)
			}
		}

		// Should be back at no version
		_, _, err := m.Version()
		assert.Equal(t, migrate.ErrNilVersion, err)
	})
}

func testFullMigrationCycle(t *testing.T, m *migrate.Migrate, db *sql.DB) {
	// Reset to clean state
	err := m.Down()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err)
	}

	// Migrate all the way up
	err = m.Up()
	require.NoError(t, err)

	// Validate final schema state
	validateFinalSchema(t, db)

	// Migrate all the way down
	err = m.Down()
	require.NoError(t, err)

	// Verify clean state (filter out schema_migrations which is expected to remain)
	tables := getTableList(t, db)
	var userTables []string
	for _, table := range tables {
		if table != "schema_migrations" {
			userTables = append(userTables, table)
		}
	}
	assert.Empty(t, userTables, "All user tables should be dropped after full down migration (schema_migrations is expected)")
}

func testMigrationIdempotency(t *testing.T, m *migrate.Migrate, _ *sql.DB) {
	// Reset to clean state
	err := m.Down()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err)
	}

	// Migrate up
	err = m.Up()
	require.NoError(t, err)

	// Try to migrate up again (should be no change)
	err = m.Up()
	assert.Equal(t, migrate.ErrNoChange, err)

	// Get current version
	version1, dirty1, err := m.Version()
	require.NoError(t, err)
	assert.False(t, dirty1)

	// Force re-run of current migration (simulate interrupted migration)
	err = m.Force(int(version1))
	require.NoError(t, err)

	// Migrate up again
	err = m.Up()
	assert.Equal(t, migrate.ErrNoChange, err)

	// Version should be the same
	version2, dirty2, err := m.Version()
	require.NoError(t, err)
	assert.Equal(t, version1, version2)
	assert.False(t, dirty2)
}

// Validation functions for each migration
func validateInitialSchema(t *testing.T, db *sql.DB) {
	tables := []string{
		"tenants", "endpoints", "basestations", "messages",
		"downlink_queue", "roaming_agreements", "schema_migrations",
	}

	existingTables := getTableList(t, db)
	for _, table := range tables {
		assert.Contains(t, existingTables, table, "Table %s should exist", table)
	}
}

// Schema-accurate validators for migrations 2-7

func validateBasestationExtensions(t *testing.T, db *sql.DB) {
	// Migration 002 adds columns to basestations table
	columns := getColumnList(t, db, "basestations")
	assert.Contains(t, columns, "vendor")
	assert.Contains(t, columns, "model")
	assert.Contains(t, columns, "version")
	assert.Contains(t, columns, "session_uuid")
	assert.Contains(t, columns, "session_started_at")
	assert.Contains(t, columns, "bidi")
	assert.Contains(t, columns, "uptime")
	assert.Contains(t, columns, "ul_load")
	assert.Contains(t, columns, "dl_load")
	assert.Contains(t, columns, "status_code")
	assert.Contains(t, columns, "status_message")

	// Check index
	indexes := getIndexList(t, db, "basestations")
	assert.Contains(t, indexes, "idx_basestations_session")
}

func validateEndpointExtensions(t *testing.T, db *sql.DB) {
	// Migration 003 adds columns to endpoints table
	columns := getColumnList(t, db, "endpoints")
	assert.Contains(t, columns, "bidi")
	assert.Contains(t, columns, "pre_attach")
	assert.Contains(t, columns, "sh_addr")
	assert.Contains(t, columns, "attach_cnt")
	assert.Contains(t, columns, "packet_cnt")
	assert.Contains(t, columns, "dual_chan")
	assert.Contains(t, columns, "repetition")
	assert.Contains(t, columns, "wide_carr_off")
	assert.Contains(t, columns, "long_blk_dist")
	assert.Contains(t, columns, "ep_status")

	// Check index
	indexes := getIndexList(t, db, "endpoints")
	assert.Contains(t, indexes, "idx_endpoints_status")
}

func validateBasestationReceptions(t *testing.T, db *sql.DB) {
	// Migration 004 creates basestation_receptions table
	assert.True(t, tableExists(t, db, "basestation_receptions"))

	// Check indexes
	indexes := getIndexList(t, db, "basestation_receptions")
	assert.Contains(t, indexes, "idx_basestation_receptions_message")
	assert.Contains(t, indexes, "idx_basestation_receptions_basestation")
	assert.Contains(t, indexes, "idx_basestation_receptions_tenant")
	assert.Contains(t, indexes, "idx_basestation_receptions_rssi")
}

func validateEndpointSessions(t *testing.T, db *sql.DB) {
	// Migration 005 creates endpoint_sessions table
	assert.True(t, tableExists(t, db, "endpoint_sessions"))

	// Check indexes
	indexes := getIndexList(t, db, "endpoint_sessions")
	assert.Contains(t, indexes, "idx_endpoint_sessions_endpoint")
	assert.Contains(t, indexes, "idx_endpoint_sessions_tenant")
	assert.Contains(t, indexes, "idx_endpoint_sessions_session")
	assert.Contains(t, indexes, "idx_endpoint_sessions_activity")
	assert.Contains(t, indexes, "idx_endpoint_sessions_unique_active")
}

func validateEndpointKeys(t *testing.T, db *sql.DB) {
	// Migration 006 creates endpoint_keys table
	assert.True(t, tableExists(t, db, "endpoint_keys"))

	// Check indexes
	indexes := getIndexList(t, db, "endpoint_keys")
	assert.Contains(t, indexes, "idx_endpoint_keys_endpoint")
	assert.Contains(t, indexes, "idx_endpoint_keys_tenant")
	assert.Contains(t, indexes, "idx_endpoint_keys_validity")
	assert.Contains(t, indexes, "idx_endpoint_keys_rotation")
	assert.Contains(t, indexes, "idx_endpoint_keys_unique_active")
}

func validateBasestationSessions(t *testing.T, db *sql.DB) {
	// Migration 007 creates basestation_sessions table
	assert.True(t, tableExists(t, db, "basestation_sessions"))

	// Check indexes
	indexes := getIndexList(t, db, "basestation_sessions")
	assert.Contains(t, indexes, "idx_basestation_sessions_bs")
	assert.Contains(t, indexes, "idx_basestation_sessions_tenant")
	assert.Contains(t, indexes, "idx_basestation_sessions_session")
	assert.Contains(t, indexes, "idx_basestation_sessions_activity")
	assert.Contains(t, indexes, "idx_basestation_sessions_errors")
	assert.Contains(t, indexes, "idx_basestation_sessions_unique_active")
}

func validatePerformanceIndexes(t *testing.T, db *sql.DB) {
	// Check composite indexes
	indexes := getIndexList(t, db, "messages")
	assert.Contains(t, indexes, "idx_messages_tenant")

	indexes = getIndexList(t, db, "endpoints")
	assert.Contains(t, indexes, "idx_endpoints_status")
}

func validateMessagePartitioning(t *testing.T, db *sql.DB) {
	// Check if messages table has partitioning enabled
	// This is PostgreSQL specific
	var isPartitioned bool
	err := db.QueryRow(`
		SELECT relkind = 'p' 
		FROM pg_class 
		WHERE relname = 'messages' AND relnamespace = 'public'::regnamespace
	`).Scan(&isPartitioned)
	require.NoError(t, err)
	assert.True(t, isPartitioned, "Messages table should be partitioned")
}

func validateKeyStorageSchema(t *testing.T, db *sql.DB) {
	// Migration 012 creates 3 tables for encryption infrastructure
	assert.True(t, tableExists(t, db, "encryption_keys"))
	assert.True(t, tableExists(t, db, "key_usage_log"))
	assert.True(t, tableExists(t, db, "encrypted_fields"))

	// Check encryption_keys indexes
	encKeysIndexes := getIndexList(t, db, "encryption_keys")
	assert.Contains(t, encKeysIndexes, "idx_encryption_keys_unique_active")
	assert.Contains(t, encKeysIndexes, "idx_encryption_keys_status")
	assert.Contains(t, encKeysIndexes, "idx_encryption_keys_expires")
	assert.Contains(t, encKeysIndexes, "idx_encryption_keys_type")

	// Check key_usage_log indexes
	usageLogIndexes := getIndexList(t, db, "key_usage_log")
	assert.Contains(t, usageLogIndexes, "idx_key_usage_log_key")
	assert.Contains(t, usageLogIndexes, "idx_key_usage_log_entity")
	assert.Contains(t, usageLogIndexes, "idx_key_usage_log_time")

	// Check encrypted_fields indexes
	encFieldsIndexes := getIndexList(t, db, "encrypted_fields")
	assert.Contains(t, encFieldsIndexes, "idx_encrypted_fields_unique_active")
	assert.Contains(t, encFieldsIndexes, "idx_encrypted_fields_table")

	// Check columns added to endpoint_keys
	endpointKeysColumns := getColumnList(t, db, "endpoint_keys")
	assert.Contains(t, endpointKeysColumns, "encryption_key_id")
	assert.Contains(t, endpointKeysColumns, "encrypted_at")
	assert.Contains(t, endpointKeysColumns, "encryption_metadata")
}

func validateSecurityConstraints(t *testing.T, db *sql.DB) {
	// Migration 013 creates endpoints_eui_length; migration 000084 renames it
	// to endpoints_ep_eui_length. Accept either name so the validator holds at
	// any schema version from 013 onward.
	constraints := getConstraintList(t, db, "endpoints")
	hasLengthConstraint := false
	for _, name := range constraints {
		if name == "endpoints_eui_length" || name == "endpoints_ep_eui_length" {
			hasLengthConstraint = true
			break
		}
	}
	assert.True(t, hasLengthConstraint,
		"endpoints must have an EUI length CHECK constraint (endpoints_eui_length pre-084, endpoints_ep_eui_length post-084), got: %v", constraints)

	// Note: endpoint_keys table has no named constraints
	// Validation is trigger-based (validate_key_format function from migration 013)
}

func validateArchiveTables(t *testing.T, db *sql.DB) {
	archiveTables := []string{
		"messages_archive",
		"basestation_receptions_archive",
		"endpoint_sessions_archive",
		"endpoint_keys_archive",
	}

	existingTables := getTableList(t, db)
	for _, table := range archiveTables {
		assert.Contains(t, existingTables, table, "Archive table %s should exist", table)
	}
}

func validateFinalSchema(t *testing.T, db *sql.DB) {
	// Validate all tables exist
	expectedTables := []string{
		"tenants", "endpoints", "basestations", "messages", "downlink_queue",
		"roaming_agreements", "basestation_receptions", "endpoint_sessions",
		"endpoint_keys", "basestation_sessions", "system_events",
		"messages_archive", "basestation_receptions_archive",
		"endpoint_sessions_archive", "endpoint_keys_archive",
	}

	existingTables := getTableList(t, db)
	for _, table := range expectedTables {
		assert.Contains(t, existingTables, table, "Table %s should exist in final schema", table)
	}
}

func tableExists(t *testing.T, db *sql.DB, tableName string) bool {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = $1
		)
	`, tableName).Scan(&exists)
	require.NoError(t, err)
	return exists
}

func getTableList(t *testing.T, db *sql.DB) []string {
	rows, err := db.Query(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	require.NoError(t, err)
	defer func() {
		if err := rows.Close(); err != nil {
			t.Logf("rows close failed in getTableList: %v", err)
		}
	}()

	var tables []string
	for rows.Next() {
		var table string
		err := rows.Scan(&table)
		require.NoError(t, err)
		tables = append(tables, table)
	}
	return tables
}

func getIndexList(t *testing.T, db *sql.DB, tableName string) []string {
	rows, err := db.Query(`
		SELECT indexname
		FROM pg_indexes
		WHERE schemaname = 'public' AND tablename = $1
	`, tableName)
	require.NoError(t, err)
	defer func() {
		if err := rows.Close(); err != nil {
			t.Logf("rows close failed in getIndexList: %v", err)
		}
	}()

	var indexes []string
	for rows.Next() {
		var index string
		err := rows.Scan(&index)
		require.NoError(t, err)
		indexes = append(indexes, index)
	}
	return indexes
}

func getColumnList(t *testing.T, db *sql.DB, tableName string) []string {
	rows, err := db.Query(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`, tableName)
	require.NoError(t, err)
	defer func() {
		if err := rows.Close(); err != nil {
			t.Logf("rows close failed in getColumnList: %v", err)
		}
	}()

	var columns []string
	for rows.Next() {
		var column string
		err := rows.Scan(&column)
		require.NoError(t, err)
		columns = append(columns, column)
	}
	return columns
}

func getConstraintList(t *testing.T, db *sql.DB, tableName string) []string {
	rows, err := db.Query(`
		SELECT constraint_name
		FROM information_schema.table_constraints
		WHERE table_schema = 'public' AND table_name = $1
	`, tableName)
	require.NoError(t, err)
	defer func() {
		if err := rows.Close(); err != nil {
			t.Logf("rows close failed in getConstraintList: %v", err)
		}
	}()

	var constraints []string
	for rows.Next() {
		var constraint string
		err := rows.Scan(&constraint)
		require.NoError(t, err)
		constraints = append(constraints, constraint)
	}
	return constraints
}
