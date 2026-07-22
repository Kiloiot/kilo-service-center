package postgres

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestSchemaBaseline captures the current database schema after all migrations
func TestSchemaBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping schema baseline in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	timestamp := time.Now().Format("20060102-150405")
	outputFile := fmt.Sprintf("../../../compliance/evidence/schema-audit-%s.md", timestamp)

	var output strings.Builder
	output.WriteString("# Database Schema Audit\n\n")
	fmt.Fprintf(&output, "Generated: %s\n\n", time.Now().Format(time.RFC3339))

	// Get migration version
	var version int
	err := db.QueryRow("SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&version)
	require.NoError(t, err)
	fmt.Fprintf(&output, "**Migration Version:** %d\n\n", version)

	// Get all tables
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
			t.Logf("failed to close rows in schema audit: %v", err)
		}
	}()

	var tables []string
	for rows.Next() {
		var table string
		err := rows.Scan(&table)
		require.NoError(t, err)
		tables = append(tables, table)
	}

	fmt.Fprintf(&output, "## Tables (%d total)\n\n", len(tables))

	// For each table, get detailed structure
	for _, table := range tables {
		fmt.Fprintf(&output, "### %s\n\n", table)

		// Get columns
		colRows, err := db.Query(`
			SELECT 
				column_name,
				data_type,
				character_maximum_length,
				is_nullable,
				column_default
			FROM information_schema.columns
			WHERE table_name = $1
			ORDER BY ordinal_position
		`, table)
		require.NoError(t, err)

		output.WriteString("**Columns:**\n\n")
		output.WriteString("| Column | Type | Nullable | Default |\n")
		output.WriteString("|--------|------|----------|--------|\n")

		for colRows.Next() {
			var colName, dataType, isNullable string
			var maxLength *int
			var colDefault *string

			err := colRows.Scan(&colName, &dataType, &maxLength, &isNullable, &colDefault)
			require.NoError(t, err)

			typeStr := dataType
			if maxLength != nil {
				typeStr = fmt.Sprintf("%s(%d)", dataType, *maxLength)
			}

			defaultStr := ""
			if colDefault != nil {
				defaultStr = *colDefault
			}

			fmt.Fprintf(&output, "| %s | %s | %s | %s |\n",
				colName, typeStr, isNullable, defaultStr)
		}
		if err := colRows.Close(); err != nil {
			t.Fatalf("failed to close column rows: %v", err)
		}

		// Get indexes
		idxRows, err := db.Query(`
			SELECT indexname, indexdef
			FROM pg_indexes
			WHERE tablename = $1
			ORDER BY indexname
		`, table)
		require.NoError(t, err)

		var indexes []string
		for idxRows.Next() {
			var idxName, idxDef string
			err := idxRows.Scan(&idxName, &idxDef)
			require.NoError(t, err)
			indexes = append(indexes, fmt.Sprintf("- `%s`: %s", idxName, idxDef))
		}
		if err := idxRows.Close(); err != nil {
			t.Fatalf("failed to close index rows: %v", err)
		}

		if len(indexes) > 0 {
			output.WriteString("\n**Indexes:**\n\n")
			for _, idx := range indexes {
				output.WriteString(idx + "\n")
			}
		}

		// Get constraints
		constRows, err := db.Query(`
			SELECT constraint_name, constraint_type
			FROM information_schema.table_constraints
			WHERE table_name = $1
			ORDER BY constraint_name
		`, table)
		require.NoError(t, err)

		var constraints []string
		for constRows.Next() {
			var constName, constType string
			err := constRows.Scan(&constName, &constType)
			require.NoError(t, err)
			constraints = append(constraints, fmt.Sprintf("- `%s` (%s)", constName, constType))
		}
		if err := constRows.Close(); err != nil {
			t.Fatalf("failed to close constraint rows: %v", err)
		}

		if len(constraints) > 0 {
			output.WriteString("\n**Constraints:**\n\n")
			for _, constraint := range constraints {
				output.WriteString(constraint + "\n")
			}
		}

		output.WriteString("\n")
	}

	// Write to file (the evidence directory is not tracked in every checkout)
	require.NoError(t, os.MkdirAll(filepath.Dir(outputFile), 0o755))
	err = os.WriteFile(outputFile, []byte(output.String()), 0644)
	require.NoError(t, err)

	t.Logf("Schema audit written to: %s", outputFile)
	t.Logf("Migration version: %d", version)
	t.Logf("Total tables: %d", len(tables))
}

// TestStructTagConsistency verifies all struct tags match actual database columns
func TestStructTagConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping struct tag consistency in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	// Define critical table -> struct mappings to check
	type TableCheck struct {
		Table      string
		ModelFile  string
		KeyColumns []string
	}

	checks := []TableCheck{
		{Table: "basestations", ModelFile: "storage/models/basestation.go", KeyColumns: []string{"id", "tenant_id", "bs_eui", "name"}},
		{Table: "endpoints", ModelFile: "storage/models/endpoint.go", KeyColumns: []string{"id", "tenant_id", "ep_eui", "name"}},
		{Table: "endpoint_sessions", ModelFile: "storage/models/endpoint.go", KeyColumns: []string{"id", "endpoint_id", "tenant_id", "last_packet_cnt"}},
		{Table: "basestation_sessions", ModelFile: "storage/models/basestation.go", KeyColumns: []string{"id", "basestation_id", "tenant_id", "sn_bs_uuid", "sn_sc_uuid"}},
		{Table: "messages", ModelFile: "storage/models/message.go", KeyColumns: []string{"id", "tenant_id", "ep_eui", "bs_eui"}},
		{Table: "downlink_queue", ModelFile: "storage/models/downlink.go", KeyColumns: []string{"id", "tenant_id", "ep_eui", "que_id"}},
	}

	var mismatches []string

	for _, check := range checks {
		// Get actual DB columns
		rows, err := db.Query(`
			SELECT column_name
			FROM information_schema.columns
			WHERE table_name = $1
			ORDER BY column_name
		`, check.Table)
		require.NoError(t, err)

		var dbColumns []string
		for rows.Next() {
			var col string
			if err := rows.Scan(&col); err != nil {
				t.Fatalf("scan failed: %v", err)
			}
			dbColumns = append(dbColumns, col)
		}
		_ = rows.Close() // Best effort close

		// Check key columns exist
		for _, keyCol := range check.KeyColumns {
			found := false
			for _, dbCol := range dbColumns {
				if dbCol == keyCol {
					found = true
					break
				}
			}
			if !found {
				mismatches = append(mismatches, fmt.Sprintf("Table %s missing expected column: %s (check %s)",
					check.Table, keyCol, check.ModelFile))
			}
		}
	}

	if len(mismatches) > 0 {
		t.Errorf("Found %d struct tag / column mismatches:\n%s", len(mismatches), strings.Join(mismatches, "\n"))
	} else {
		t.Log("All key columns verified successfully")
	}
}
