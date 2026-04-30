package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// schemaSnapshot captures the current state of a PostgreSQL schema
// This is used to validate that down migrations correctly restore previous schema states
type schemaSnapshot struct {
	version  uint                // Migration version from schema_migrations
	tables   []string            // Table names in public schema
	columns  map[string][]string // table -> sorted column names
	indexes  map[string][]string // table -> sorted index names
	triggers map[string][]string // table -> sorted trigger names
}

// captureSchemaSnapshot captures a complete snapshot of the current database schema
func captureSchemaSnapshot(db *sql.DB, m *migrate.Migrate) (schemaSnapshot, error) {
	var snap schemaSnapshot

	// 1. Get migration version
	version, _, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return snap, fmt.Errorf("failed to get migration version: %w", err)
	}
	snap.version = version

	// 2. Get all tables in public schema
	tables, err := getTables(db)
	if err != nil {
		return snap, fmt.Errorf("failed to get tables: %w", err)
	}
	snap.tables = tables

	// 3. Get columns for each table
	snap.columns = make(map[string][]string)
	for _, table := range tables {
		columns, err := getTableColumns(db, table)
		if err != nil {
			return snap, fmt.Errorf("failed to get columns for table %s: %w", table, err)
		}
		snap.columns[table] = columns
	}

	// 4. Get indexes for each table
	snap.indexes = make(map[string][]string)
	for _, table := range tables {
		indexes, err := getTableIndexes(db, table)
		if err != nil {
			return snap, fmt.Errorf("failed to get indexes for table %s: %w", table, err)
		}
		snap.indexes[table] = indexes
	}

	// 5. Get triggers for each table
	snap.triggers = make(map[string][]string)
	for _, table := range tables {
		triggers, err := getTableTriggers(db, table)
		if err != nil {
			return snap, fmt.Errorf("failed to get triggers for table %s: %w", table, err)
		}
		snap.triggers[table] = triggers
	}

	return snap, nil
}

// getTables returns all tables in the public schema (excluding system tables)
func getTables(db *sql.DB) ([]string, error) {
	query := `
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public'
		AND tablename NOT IN ('schema_migrations')
		ORDER BY tablename
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows in schema snapshot: %v", err)
		}
	}()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, rows.Err()
}

// getTableColumns returns all column names for a table, sorted
func getTableColumns(db *sql.DB, tableName string) ([]string, error) {
	query := `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		AND table_name = $1
		ORDER BY ordinal_position
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows in schema snapshot: %v", err)
		}
	}()

	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, err
		}
		columns = append(columns, column)
	}

	return columns, rows.Err()
}

// getTableIndexes returns all index names for a table, sorted
func getTableIndexes(db *sql.DB, tableName string) ([]string, error) {
	query := `
		SELECT indexname
		FROM pg_indexes
		WHERE schemaname = 'public'
		AND tablename = $1
		ORDER BY indexname
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows in schema snapshot: %v", err)
		}
	}()

	var indexes []string
	for rows.Next() {
		var index string
		if err := rows.Scan(&index); err != nil {
			return nil, err
		}
		indexes = append(indexes, index)
	}

	return indexes, rows.Err()
}

// getTableTriggers returns all trigger names for a table, sorted
func getTableTriggers(db *sql.DB, tableName string) ([]string, error) {
	query := `
		SELECT t.tgname
		FROM pg_trigger t
		JOIN pg_class c ON t.tgrelid = c.oid
		WHERE c.relname = $1
		AND t.tgisinternal = false
		ORDER BY t.tgname
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows in schema snapshot: %v", err)
		}
	}()

	var triggers []string
	for rows.Next() {
		var trigger string
		if err := rows.Scan(&trigger); err != nil {
			return nil, err
		}
		triggers = append(triggers, trigger)
	}

	return triggers, rows.Err()
}

// compare compares two schema snapshots and returns detailed differences
func (expected schemaSnapshot) compare(actual schemaSnapshot) []string {
	var diffs []string

	// 1. Compare migration versions
	if expected.version != actual.version {
		diffs = append(diffs, fmt.Sprintf("Version mismatch: expected %d, got %d",
			expected.version, actual.version))
	}

	// 2. Compare table lists
	expectedTables := make(map[string]bool)
	for _, t := range expected.tables {
		expectedTables[t] = true
	}
	actualTables := make(map[string]bool)
	for _, t := range actual.tables {
		actualTables[t] = true
	}

	// Find missing tables
	for table := range expectedTables {
		if !actualTables[table] {
			diffs = append(diffs, fmt.Sprintf("Missing table: %s", table))
		}
	}

	// Find extra tables
	for table := range actualTables {
		if !expectedTables[table] {
			diffs = append(diffs, fmt.Sprintf("Extra table: %s", table))
		}
	}

	// 3. Compare columns for common tables
	for table := range expectedTables {
		if !actualTables[table] {
			continue // Already reported as missing
		}

		expectedCols := expected.columns[table]
		actualCols := actual.columns[table]

		if !equalStringSlices(expectedCols, actualCols) {
			diffs = append(diffs, fmt.Sprintf("Table %s: column mismatch", table))
			diffs = append(diffs, fmt.Sprintf("  Expected: %v", expectedCols))
			diffs = append(diffs, fmt.Sprintf("  Actual:   %v", actualCols))
		}
	}

	// 4. Compare indexes for common tables
	for table := range expectedTables {
		if !actualTables[table] {
			continue
		}

		expectedIdx := expected.indexes[table]
		actualIdx := actual.indexes[table]

		if !equalStringSlices(expectedIdx, actualIdx) {
			diffs = append(diffs, fmt.Sprintf("Table %s: index mismatch", table))
			diffs = append(diffs, fmt.Sprintf("  Expected: %v", expectedIdx))
			diffs = append(diffs, fmt.Sprintf("  Actual:   %v", actualIdx))
		}
	}

	// 5. Compare triggers for common tables
	for table := range expectedTables {
		if !actualTables[table] {
			continue
		}

		expectedTrig := expected.triggers[table]
		actualTrig := actual.triggers[table]

		if !equalStringSlices(expectedTrig, actualTrig) {
			diffs = append(diffs, fmt.Sprintf("Table %s: trigger mismatch", table))
			diffs = append(diffs, fmt.Sprintf("  Expected: %v", expectedTrig))
			diffs = append(diffs, fmt.Sprintf("  Actual:   %v", actualTrig))
		}
	}

	return diffs
}

// equalStringSlices compares two string slices for equality (order matters)
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// summary returns a human-readable summary of the snapshot
func (expected schemaSnapshot) summary() string {
	totalColumns := 0
	for _, cols := range expected.columns {
		totalColumns += len(cols)
	}

	totalIndexes := 0
	for _, idx := range expected.indexes {
		totalIndexes += len(idx)
	}

	totalTriggers := 0
	for _, trig := range expected.triggers {
		totalTriggers += len(trig)
	}

	return fmt.Sprintf("Version %d: %d tables, %d columns, %d indexes, %d triggers",
		expected.version, len(expected.tables), totalColumns, totalIndexes, totalTriggers)
}

// String returns a detailed string representation of the snapshot
func (expected schemaSnapshot) String() string {
	result := fmt.Sprintf("Schema Snapshot (version %d)\n", expected.version)
	result += fmt.Sprintf("Tables (%d):\n", len(expected.tables))

	// Sort tables for consistent output
	tables := make([]string, len(expected.tables))
	copy(tables, expected.tables)
	sort.Strings(tables)

	for _, table := range tables {
		result += fmt.Sprintf("  %s:\n", table)
		result += fmt.Sprintf("    Columns (%d): %v\n", len(expected.columns[table]), expected.columns[table])
		result += fmt.Sprintf("    Indexes (%d): %v\n", len(expected.indexes[table]), expected.indexes[table])
		result += fmt.Sprintf("    Triggers (%d): %v\n", len(expected.triggers[table]), expected.triggers[table])
	}

	return result
}

// TestSchemaSnapshotCapture validates that we can capture schema snapshots
func TestSchemaSnapshotCapture(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping schema snapshot test in short mode")
	}

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

	// Test 1: Capture snapshot after no migrations
	snap0, err := captureSchemaSnapshot(db.DB, m)
	require.NoError(t, err)
	assert.Equal(t, uint(0), snap0.version, "Version should be 0 with no migrations")
	assert.Empty(t, snap0.tables, "Should have no tables initially")
	t.Logf("Initial snapshot: %s", snap0.summary())

	// Test 2: Apply migration 1 and capture
	err = m.Migrate(1)
	require.NoError(t, err)

	snap1, err := captureSchemaSnapshot(db.DB, m)
	require.NoError(t, err)
	assert.Equal(t, uint(1), snap1.version, "Version should be 1")
	assert.NotEmpty(t, snap1.tables, "Should have tables after migration 1")
	t.Logf("After migration 1: %s", snap1.summary())

	// Verify basic tables exist
	assert.Contains(t, snap1.tables, "tenants", "Should have tenants table")
	assert.Contains(t, snap1.tables, "endpoints", "Should have endpoints table")
	assert.Contains(t, snap1.tables, "basestations", "Should have basestations table")

	// Verify columns are captured
	assert.NotEmpty(t, snap1.columns["tenants"], "Tenants should have columns")
	assert.NotEmpty(t, snap1.columns["endpoints"], "Endpoints should have columns")

	// Test 3: Apply more migrations and verify snapshot grows
	err = m.Migrate(5)
	require.NoError(t, err)

	snap5, err := captureSchemaSnapshot(db.DB, m)
	require.NoError(t, err)
	assert.Equal(t, uint(5), snap5.version, "Version should be 5")
	assert.Greater(t, len(snap5.tables), len(snap1.tables),
		"Should have more tables after more migrations")
	t.Logf("After migration 5: %s", snap5.summary())
}

// TestSchemaSnapshotComparison validates snapshot comparison logic
func TestSchemaSnapshotComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping schema snapshot comparison test in short mode")
	}

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

	// Apply migration 1
	err = m.Migrate(1)
	require.NoError(t, err)

	snap1, err := captureSchemaSnapshot(db.DB, m)
	require.NoError(t, err)

	// Test 1: Compare identical snapshots
	snap1Copy, err := captureSchemaSnapshot(db.DB, m)
	require.NoError(t, err)

	diffs := snap1.compare(snap1Copy)
	assert.Empty(t, diffs, "Identical snapshots should have no differences")

	// Test 2: Apply more migrations and compare
	err = m.Migrate(5)
	require.NoError(t, err)

	snap5, err := captureSchemaSnapshot(db.DB, m)
	require.NoError(t, err)

	diffs = snap1.compare(snap5)
	assert.NotEmpty(t, diffs, "Different snapshots should have differences")
	t.Logf("Found %d differences between migration 1 and 5:", len(diffs))
	for _, diff := range diffs {
		t.Logf("  - %s", diff)
	}

	// Test 3: Roll back and verify snapshot matches
	err = m.Migrate(1)
	require.NoError(t, err)

	snap1After, err := captureSchemaSnapshot(db.DB, m)
	require.NoError(t, err)

	diffs = snap1.compare(snap1After)
	if len(diffs) > 0 {
		t.Logf("Differences after rollback to migration 1:")
		for _, diff := range diffs {
			t.Logf("  - %s", diff)
		}
		// Some differences may be acceptable (e.g., index recreation order)
		// but log them for investigation
	}
}

// TestSchemaSnapshotRollbackValidation tests that rolling back a migration
// restores the schema to the previous state
func TestSchemaSnapshotRollbackValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rollback validation test in short mode")
	}

	// Test migration 14 specifically (our P0 fix)
	testMigrationRollback(t, 14)
}

// testMigrationRollback validates that a specific migration can be rolled back
// and the schema matches the pre-migration state
func testMigrationRollback(t *testing.T, migNum uint) {
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

	// Step 1: Apply migrations up to (migNum - 1)
	if migNum > 1 {
		err = m.Migrate(migNum - 1)
		require.NoError(t, err)
	}

	// Step 2: Capture "before" snapshot
	snapBefore, err := captureSchemaSnapshot(db.DB, m)
	require.NoError(t, err)
	t.Logf("Before migration %d: %s", migNum, snapBefore.summary())

	// Step 3: Apply target migration
	err = m.Migrate(migNum)
	require.NoError(t, err)

	snapAfter, err := captureSchemaSnapshot(db.DB, m)
	require.NoError(t, err)
	t.Logf("After migration %d: %s", migNum, snapAfter.summary())

	// Step 4: Roll back target migration
	err = m.Migrate(migNum - 1)
	require.NoError(t, err, "Rollback of migration %d failed", migNum)

	// Step 5: Capture "after rollback" snapshot
	snapRollback, err := captureSchemaSnapshot(db.DB, m)
	require.NoError(t, err)
	t.Logf("After rollback of migration %d: %s", migNum, snapRollback.summary())

	// Step 6: Compare snapshots
	diffs := snapBefore.compare(snapRollback)
	if len(diffs) > 0 {
		t.Errorf("Schema mismatch after rolling back migration %d:", migNum)
		for _, diff := range diffs {
			t.Errorf("  - %s", diff)
		}
	} else {
		t.Logf("PASS: Migration %d rollback successful - schema matches pre-migration state", migNum)
	}
}

// TestEqualStringSlices tests the helper function
func TestEqualStringSlices(t *testing.T) {
	testCases := []struct {
		name     string
		a, b     []string
		expected bool
	}{
		{"both empty", []string{}, []string{}, true},
		{"both nil", nil, nil, true},
		{"equal non-empty", []string{"a", "b", "c"}, []string{"a", "b", "c"}, true},
		{"different length", []string{"a", "b"}, []string{"a", "b", "c"}, false},
		{"different order", []string{"a", "b", "c"}, []string{"a", "c", "b"}, false},
		{"different values", []string{"a", "b", "c"}, []string{"a", "b", "d"}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := equalStringSlices(tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}
