package postgres

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// migrationArtifacts represents database objects created/dropped by a migration
type migrationArtifacts struct {
	tables    []string
	indexes   []string
	triggers  []string
	functions []string
	types     []string
}

// TestDownScriptSymmetry validates that down scripts only drop objects
// created by the corresponding up script
func TestDownScriptSymmetry(t *testing.T) {
	migrations := discoverMigrations(t)

	for _, mig := range migrations {
		t.Run(fmt.Sprintf("Migration_%03d_Symmetry", mig.number), func(t *testing.T) {
			// Get migration file paths
			migrationsDir, err := filepath.Abs("../../migrations")
			require.NoError(t, err)

			upFile := findMigrationFile(t, migrationsDir, mig.number, "up")
			downFile := findMigrationFile(t, migrationsDir, mig.number, "down")

			if downFile == "" {
				t.Skipf("No down migration for %d", mig.number)
				return
			}

			// Parse artifacts from up and down scripts
			upArtifacts := parseCreates(t, upFile)
			downArtifacts := parseDrops(t, downFile)

			// Verify symmetry: down should only drop what up created
			checkSymmetry(t, mig.number, upArtifacts, downArtifacts)

			// Verify dependency order in down script
			checkDependencyOrder(t, mig.number, downFile)
		})
	}
}

// findMigrationFile finds the migration file for a given number and direction
func findMigrationFile(t *testing.T, dir string, num int, direction string) string {
	// Use zero-padded pattern to match exact migration numbers
	// Try different padding lengths: 001, 0001, 00001, 000001
	patterns := []string{
		filepath.Join(dir, fmt.Sprintf("%03d_*.%s.sql", num, direction)), // 001_*.up.sql
		filepath.Join(dir, fmt.Sprintf("%04d_*.%s.sql", num, direction)), // 0001_*.up.sql
		filepath.Join(dir, fmt.Sprintf("%06d_*.%s.sql", num, direction)), // 000001_*.up.sql
		filepath.Join(dir, fmt.Sprintf("0*%d_*.%s.sql", num, direction)), // fallback with leading zeros
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		require.NoError(t, err)

		if len(matches) == 1 {
			return matches[0]
		} else if len(matches) > 1 {
			// Multiple matches - this shouldn't happen with proper zero-padding
			t.Logf("Warning: Multiple matches for pattern %s: %v", pattern, matches)
		}
	}

	return ""
}

// stripSQLComments removes both single-line (--) and multi-line (/* */) comments from SQL
func stripSQLComments(sql string) string {
	// Remove multi-line comments: /* ... */
	blockCommentRe := regexp.MustCompile(`/\*[^*]*\*+(?:[^/*][^*]*\*+)*/`)
	sql = blockCommentRe.ReplaceAllString(sql, " ")

	// Remove single-line comments: -- ...
	lines := strings.Split(sql, "\n")
	var filtered []string
	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line != "" {
			filtered = append(filtered, line)
		}
	}

	return strings.Join(filtered, "\n")
}

// parseCreates extracts all CREATE statements from a migration file
func parseCreates(t *testing.T, filename string) migrationArtifacts {
	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	sql := stripSQLComments(string(content))
	var artifacts migrationArtifacts

	// Extract CREATE TABLE statements
	tableRe := regexp.MustCompile(`(?i)CREATE\s+TABLE(?:\s+IF\s+NOT\s+EXISTS)?\s+(\w+)`)
	for _, match := range tableRe.FindAllStringSubmatch(sql, -1) {
		if len(match) > 1 {
			artifacts.tables = append(artifacts.tables, strings.ToLower(match[1]))
		}
	}

	// Extract CREATE INDEX statements
	indexRe := regexp.MustCompile(`(?i)CREATE\s+(?:UNIQUE\s+)?INDEX(?:\s+IF\s+NOT\s+EXISTS)?\s+(\w+)`)
	for _, match := range indexRe.FindAllStringSubmatch(sql, -1) {
		if len(match) > 1 {
			artifacts.indexes = append(artifacts.indexes, strings.ToLower(match[1]))
		}
	}

	// Extract CREATE TRIGGER statements
	triggerRe := regexp.MustCompile(`(?i)CREATE\s+TRIGGER\s+(\w+)`)
	for _, match := range triggerRe.FindAllStringSubmatch(sql, -1) {
		if len(match) > 1 {
			artifacts.triggers = append(artifacts.triggers, strings.ToLower(match[1]))
		}
	}

	// Extract CREATE FUNCTION statements
	funcRe := regexp.MustCompile(`(?i)CREATE(?:\s+OR\s+REPLACE)?\s+FUNCTION\s+(\w+)`)
	for _, match := range funcRe.FindAllStringSubmatch(sql, -1) {
		if len(match) > 1 {
			artifacts.functions = append(artifacts.functions, strings.ToLower(match[1]))
		}
	}

	// Extract CREATE TYPE statements
	typeRe := regexp.MustCompile(`(?i)CREATE\s+TYPE\s+(\w+)`)
	for _, match := range typeRe.FindAllStringSubmatch(sql, -1) {
		if len(match) > 1 {
			artifacts.types = append(artifacts.types, strings.ToLower(match[1]))
		}
	}

	return artifacts
}

// parseDrops extracts all DROP statements from a migration file
func parseDrops(t *testing.T, filename string) migrationArtifacts {
	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	sql := stripSQLComments(string(content))
	var artifacts migrationArtifacts

	// Extract DROP TABLE statements
	tableRe := regexp.MustCompile(`(?i)DROP\s+TABLE(?:\s+IF\s+EXISTS)?\s+(\w+)`)
	for _, match := range tableRe.FindAllStringSubmatch(sql, -1) {
		if len(match) > 1 {
			artifacts.tables = append(artifacts.tables, strings.ToLower(match[1]))
		}
	}

	// Extract DROP INDEX statements
	indexRe := regexp.MustCompile(`(?i)DROP\s+INDEX(?:\s+IF\s+EXISTS)?\s+(\w+)`)
	for _, match := range indexRe.FindAllStringSubmatch(sql, -1) {
		if len(match) > 1 {
			artifacts.indexes = append(artifacts.indexes, strings.ToLower(match[1]))
		}
	}

	// Extract DROP TRIGGER statements
	triggerRe := regexp.MustCompile(`(?i)DROP\s+TRIGGER(?:\s+IF\s+EXISTS)?\s+(\w+)`)
	for _, match := range triggerRe.FindAllStringSubmatch(sql, -1) {
		if len(match) > 1 {
			artifacts.triggers = append(artifacts.triggers, strings.ToLower(match[1]))
		}
	}

	// Extract DROP FUNCTION statements
	funcRe := regexp.MustCompile(`(?i)DROP\s+FUNCTION(?:\s+IF\s+EXISTS)?\s+(\w+)`)
	for _, match := range funcRe.FindAllStringSubmatch(sql, -1) {
		if len(match) > 1 {
			artifacts.functions = append(artifacts.functions, strings.ToLower(match[1]))
		}
	}

	// Extract DROP TYPE statements
	typeRe := regexp.MustCompile(`(?i)DROP\s+TYPE(?:\s+IF\s+EXISTS)?\s+(\w+)`)
	for _, match := range typeRe.FindAllStringSubmatch(sql, -1) {
		if len(match) > 1 {
			artifacts.types = append(artifacts.types, strings.ToLower(match[1]))
		}
	}

	return artifacts
}

// checkSymmetry verifies that down script only drops what up script created
func checkSymmetry(t *testing.T, migNum int, up, down migrationArtifacts) {
	// Build map of created objects
	created := make(map[string]bool)
	for _, table := range up.tables {
		created["table:"+table] = true
	}
	for _, index := range up.indexes {
		created["index:"+index] = true
	}
	for _, trigger := range up.triggers {
		created["trigger:"+trigger] = true
	}
	for _, fn := range up.functions {
		created["function:"+fn] = true
	}
	for _, typ := range up.types {
		created["type:"+typ] = true
	}

	// Check tables
	for _, table := range down.tables {
		key := "table:" + table
		if !created[key] {
			// This is a warning, not a failure - migration might be cleaning up old stuff
			t.Logf("WARNING: Migration %d down script drops table '%s' not created in up script", migNum, table)
		}
	}

	// Check indexes - be more strict here
	for _, index := range down.indexes {
		key := "index:" + index
		if !created[key] {
			// Indexes should be symmetric - log as potential issue
			t.Logf("NOTICE: Migration %d down script drops index '%s' not created in up script (may be from earlier migration)", migNum, index)
		}
	}

	// Check triggers - must be symmetric
	for _, trigger := range down.triggers {
		key := "trigger:" + trigger
		if !created[key] {
			t.Errorf("Migration %d down script drops trigger '%s' not created in up script", migNum, trigger)
		}
	}

	// Check functions - must be symmetric
	for _, fn := range down.functions {
		key := "function:" + fn
		if !created[key] {
			t.Errorf("Migration %d down script drops function '%s' not created in up script", migNum, fn)
		}
	}

	// Check types - must be symmetric
	for _, typ := range down.types {
		key := "type:" + typ
		if !created[key] {
			t.Errorf("Migration %d down script drops type '%s' not created in up script", migNum, typ)
		}
	}
}

// checkDependencyOrder verifies that dependencies are dropped in correct order
func checkDependencyOrder(t *testing.T, migNum int, downFile string) {
	content, err := os.ReadFile(downFile)
	require.NoError(t, err)

	sql := string(content)
	lines := strings.Split(sql, "\n")

	// Find positions of different DROP statements
	triggerDrops := make(map[string]int)
	functionDrops := make(map[string]int)
	tableDrops := make(map[string]int)
	indexDrops := make(map[string]int)

	triggerRe := regexp.MustCompile(`(?i)DROP\s+TRIGGER(?:\s+IF\s+EXISTS)?\s+(\w+)`)
	funcRe := regexp.MustCompile(`(?i)DROP\s+FUNCTION(?:\s+IF\s+EXISTS)?\s+(\w+)`)
	tableRe := regexp.MustCompile(`(?i)DROP\s+TABLE(?:\s+IF\s+EXISTS)?\s+(\w+)`)
	indexRe := regexp.MustCompile(`(?i)DROP\s+INDEX(?:\s+IF\s+EXISTS)?\s+(\w+)`)

	for lineNum, line := range lines {
		if match := triggerRe.FindStringSubmatch(line); len(match) > 1 {
			triggerDrops[strings.ToLower(match[1])] = lineNum
		}
		if match := funcRe.FindStringSubmatch(line); len(match) > 1 {
			functionDrops[strings.ToLower(match[1])] = lineNum
		}
		if match := tableRe.FindStringSubmatch(line); len(match) > 1 {
			tableDrops[strings.ToLower(match[1])] = lineNum
		}
		if match := indexRe.FindStringSubmatch(line); len(match) > 1 {
			indexDrops[strings.ToLower(match[1])] = lineNum
		}
	}

	// Rule 1: Triggers must be dropped before functions they depend on
	for trigger, triggerLine := range triggerDrops {
		for fn, fnLine := range functionDrops {
			if triggerLine > fnLine {
				t.Errorf("Migration %d: Trigger '%s' (line %d) should be dropped BEFORE function '%s' (line %d)",
					migNum, trigger, triggerLine, fn, fnLine)
			}
		}
	}

	// Rule 2: Indexes should be dropped before tables (not critical, but clean)
	// This is advisory only - indexes are auto-dropped with tables
	for index, indexLine := range indexDrops {
		for table, tableLine := range tableDrops {
			if indexLine > tableLine {
				t.Logf("INFO: Migration %d: Index '%s' (line %d) dropped after table '%s' (line %d) - not critical but could be cleaner",
					migNum, index, indexLine, table, tableLine)
			}
		}
	}
}

// TestMigration014SymmetrySpecific validates our P0 fix for migration 014
func TestMigration014SymmetrySpecific(t *testing.T) {
	migrationsDir, err := filepath.Abs("../../migrations")
	require.NoError(t, err)

	upFile := findMigrationFile(t, migrationsDir, 14, "up")
	downFile := findMigrationFile(t, migrationsDir, 14, "down")

	require.NotEmpty(t, upFile, "Migration 014 up file should exist")
	require.NotEmpty(t, downFile, "Migration 014 down file should exist")

	// Parse artifacts
	upArtifacts := parseCreates(t, upFile)
	downArtifacts := parseDrops(t, downFile)

	t.Logf("Migration 014 UP creates:")
	t.Logf("  Tables: %v", upArtifacts.tables)
	t.Logf("  Indexes: %v", upArtifacts.indexes)
	t.Logf("  Triggers: %v", upArtifacts.triggers)
	t.Logf("  Functions: %v", upArtifacts.functions)

	t.Logf("Migration 014 DOWN drops:")
	t.Logf("  Tables: %v", downArtifacts.tables)
	t.Logf("  Indexes: %v", downArtifacts.indexes)
	t.Logf("  Triggers: %v", downArtifacts.triggers)
	t.Logf("  Functions: %v", downArtifacts.functions)

	// Verify all 4 triggers are in down script
	expectedTriggers := []string{
		"set_messages_archive_timestamp",
		"set_basestation_receptions_archive_timestamp",
		"set_endpoint_sessions_archive_timestamp",
		"set_endpoint_keys_archive_timestamp",
	}

	for _, trigger := range expectedTriggers {
		found := false
		for _, dt := range downArtifacts.triggers {
			if strings.EqualFold(dt, trigger) {
				found = true
				break
			}
		}
		assert.True(t, found, "Migration 014 down script should drop trigger '%s'", trigger)
	}

	// Verify function is dropped
	assert.Contains(t, downArtifacts.functions, "set_archived_at",
		"Migration 014 down script should drop function set_archived_at")

	// Verify dependency order
	checkDependencyOrder(t, 14, downFile)
}

// TestMigrationCoverage reports on which migrations have down scripts
func TestMigrationCoverage(t *testing.T) {
	migrations := discoverMigrations(t)
	migrationsDir, err := filepath.Abs("../../migrations")
	require.NoError(t, err)

	withDown := 0
	withoutDown := 0

	for _, mig := range migrations {
		downFile := findMigrationFile(t, migrationsDir, mig.number, "down")
		if downFile != "" {
			withDown++
		} else {
			withoutDown++
			t.Logf("Migration %d (%s) has no down script", mig.number, mig.description)
		}
	}

	t.Logf("Migration down script coverage: %d/%d (%.1f%%)",
		withDown, len(migrations), float64(withDown)/float64(len(migrations))*100)

	assert.Greater(t, withDown, 0, "Should have at least some down scripts")
}
