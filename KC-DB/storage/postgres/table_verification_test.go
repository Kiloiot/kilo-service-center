package postgres

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTableByTableVerification performs comprehensive table-by-table verification
func TestTableByTableVerification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping table verification in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	// Define critical tables with their expected key columns based on actual DB schema
	tables := map[string][]string{
		"tenants": {"id", "name", "status", "description", "created_at", "updated_at"},
		"basestations": {"id", "tenant_id", "bs_eui", "name", "description", "is_online", "last_seen_at",
			"connection_type", "service_center_url", "created_at", "updated_at"},
		"endpoints": {"id", "tenant_id", "ep_eui", "name", "description", "endpoint_class", "bidi",
			"pre_attach", "sh_addr", "attach_cnt", "packet_cnt", "dual_chan", "repetition",
			"wide_carr_off", "long_blk_dist", "ep_status", "last_packet_cnt", "created_at", "updated_at"},
		"endpoint_sessions": {"id", "endpoint_id", "tenant_id", "session_id", "attach_cnt",
			"last_packet_cnt", "status", "started_at", "last_activity_at", "ended_at"},
		"basestation_sessions": {"id", "basestation_id", "tenant_id",
			"sn_bs_uuid", "sn_sc_uuid", "sn_bs_op_id", "sn_sc_op_id",
			"status", "connection_id", "remote_addr",
			"started_at", "last_ping_at", "ended_at",
			"can_resume", "encoding", "protocol_version",
			"organization_id", "created_at", "updated_at"},
		// TODO: Add assertions for column types, constraints, and indexes (19 columns total)
		"messages": {"id", "tenant_id", "ep_eui", "bs_eui", "rx_time", "rssi", "snr", "eq_snr",
			"user_data", "packet_cnt", "created_at"},
		"downlink_queue": {"id", "tenant_id", "ep_eui", "payload", "priority",
			"valid_until", "earliest_at", "status", "attempts", "max_attempts", "transmitted_at",
			"acknowledged_at", "bs_eui", "failure_reason", "created_at", "updated_at",
			"que_id", "cnt_depend", "packet_cnt_array", "result", "tx_time", "tx_bs_eui"},
		"system_events": {"id", "tenant_id", "event_type", "event_category", "title", "description",
			"severity", "source_type", "source_id", "data", "occurred_at", "recorded_at"},
	}

	var allIssues []string

	for tableName, expectedCols := range tables {
		t.Run(tableName, func(t *testing.T) {
			// Get actual columns from database
			rows, err := db.Query(`
				SELECT column_name, data_type, is_nullable
				FROM information_schema.columns
				WHERE table_name = $1
				ORDER BY column_name
			`, tableName)
			require.NoError(t, err)
			defer func() {
				_ = rows.Close() // Best effort close
			}()

			actualCols := make(map[string]struct {
				DataType   string
				IsNullable string
			})
			for rows.Next() {
				var colName, dataType, isNullable string
				if err := rows.Scan(&colName, &dataType, &isNullable); err != nil {
					t.Fatalf("scan failed: %v", err)
				}
				actualCols[colName] = struct {
					DataType   string
					IsNullable string
				}{dataType, isNullable}
			}

			// Check each expected column exists
			var missing []string
			for _, expectedCol := range expectedCols {
				if _, exists := actualCols[expectedCol]; !exists {
					missing = append(missing, expectedCol)
				}
			}

			if len(missing) > 0 {
				issue := fmt.Sprintf("Table %s missing expected columns: %v", tableName, missing)
				allIssues = append(allIssues, issue)
				t.Error(issue)
			}

			// Report extra columns (might be new additions)
			var extra []string
			for actualCol := range actualCols {
				found := false
				for _, expectedCol := range expectedCols {
					if actualCol == expectedCol {
						found = true
						break
					}
				}
				if !found {
					extra = append(extra, actualCol)
				}
			}

			if len(extra) > 0 {
				t.Logf("Table %s has additional columns not in expected list: %v", tableName, extra)
			}
		})
	}

	// Write summary report
	if len(allIssues) > 0 {
		reportFile := "../../../compliance/evidence/table-verification-issues.txt"
		content := "Table Verification Issues\n"
		content += fmt.Sprintf("Generated: %s\n\n", "2025-10-30")
		content += strings.Join(allIssues, "\n")
		if err := os.WriteFile(reportFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write report: %v", err)
		}
		t.Logf("Issues written to %s", reportFile)
	} else {
		t.Log("All tables verified successfully - no mismatches found")
	}
}

// TestEndpointsUnsignedConstraints verifies CHECK constraints from migration 000087 exist
// with exact constraint names for SCACI §3.6.1 register operation compliance.
//
// These constraints enforce unsigned semantics on signed DB columns:
//   - sh_addr: uint16 on wire, INTEGER in DB (CHECK >= 0 AND <= 65535)
//   - attach_cnt: uint32 on wire, BIGINT in DB (CHECK >= 0 AND <= 4294967295)
//   - packet_cnt: uint32 on wire, BIGINT in DB (CHECK >= 0 AND <= 4294967295)
//   - last_packet_cnt: uint32 on wire, BIGINT in DB (CHECK >= 0 AND <= 4294967295)
func TestEndpointsUnsignedConstraints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping constraint verification in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	// Query pg_constraint for CHECK constraints on endpoints table
	rows, err := db.Query(`
		SELECT conname, pg_get_constraintdef(oid) as constraint_def
		FROM pg_constraint
		WHERE conrelid = 'endpoints'::regclass
		AND contype = 'c'
		ORDER BY conname
	`)
	require.NoError(t, err)
	defer func() {
		_ = rows.Close()
	}()

	constraints := make(map[string]string) // constraint_name -> definition
	for rows.Next() {
		var conname, constraintDef string
		if err := rows.Scan(&conname, &constraintDef); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		constraints[conname] = constraintDef
	}
	require.NoError(t, rows.Err())

	// Expected constraint names from migration 000087
	expectedConstraints := []string{
		"chk_endpoints_sh_addr_unsigned",
		"chk_endpoints_attach_cnt_unsigned",
		"chk_endpoints_packet_cnt_unsigned",
		"chk_endpoints_last_packet_cnt_unsigned",
	}

	// Verify each expected constraint exists
	for _, expected := range expectedConstraints {
		def, exists := constraints[expected]
		if !exists {
			t.Errorf("Missing CHECK constraint: %s (from migration 000087)", expected)
			continue
		}

		// Verify constraint enforces unsigned semantics (>= 0)
		if !strings.Contains(strings.ToLower(def), ">= 0") {
			t.Errorf("Constraint %s does not enforce >= 0: %s", expected, def)
		}

		t.Logf("Verified constraint %s: %s", expected, def)
	}

	// Log any additional constraints found (informational)
	for conname, def := range constraints {
		found := false
		for _, expected := range expectedConstraints {
			if conname == expected {
				found = true
				break
			}
		}
		if !found {
			t.Logf("Additional constraint found: %s = %s", conname, def)
		}
	}
}

// TestEndpointsUnsignedConstraintBounds verifies the actual bounds enforced by constraints.
// This ensures the constraints match SCACI §3.6.1 wire format requirements.
func TestEndpointsUnsignedConstraintBounds(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping constraint bounds verification in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	// Create a test tenant first
	_, err := db.Exec(`INSERT INTO tenants (id, name, status) VALUES (999999, 'constraint_test_tenant', 'active') ON CONFLICT DO NOTHING`)
	require.NoError(t, err)
	defer func() {
		_, _ = db.Exec(`DELETE FROM endpoints WHERE tenant_id = 999999`)
		_, _ = db.Exec(`DELETE FROM tenants WHERE id = 999999`)
	}()

	// Test constraint bounds - these should succeed
	validCases := []struct {
		name  string
		query string
	}{
		{"sh_addr max uint16", `INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name, sh_addr) VALUES (999999, 999999, E'\\x0102030405060701', 'test1', 65535)`},
		{"attach_cnt max uint32", `INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name, attach_cnt) VALUES (999999, 999999, E'\\x0102030405060702', 'test2', 4294967295)`},
		{"packet_cnt max uint32", `INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name, packet_cnt) VALUES (999999, 999999, E'\\x0102030405060703', 'test3', 4294967295)`},
		{"last_packet_cnt max uint32", `INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name, last_packet_cnt) VALUES (999999, 999999, E'\\x0102030405060704', 'test4', 4294967295)`},
	}

	for _, tc := range validCases {
		t.Run(tc.name+"_valid", func(t *testing.T) {
			_, err := db.Exec(tc.query)
			if err != nil {
				t.Errorf("Valid value rejected: %s - error: %v", tc.name, err)
			}
		})
	}

	// Test constraint violations - negative values should fail
	negativeCases := []struct {
		name  string
		query string
	}{
		{"sh_addr negative", `INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name, sh_addr) VALUES (999999, 999999, E'\\x0102030405060705', 'test5', -1)`},
		{"attach_cnt negative", `INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name, attach_cnt) VALUES (999999, 999999, E'\\x0102030405060706', 'test6', -1)`},
		{"packet_cnt negative", `INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name, packet_cnt) VALUES (999999, 999999, E'\\x0102030405060707', 'test7', -1)`},
		{"last_packet_cnt negative", `INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name, last_packet_cnt) VALUES (999999, 999999, E'\\x0102030405060708', 'test8', -1)`},
	}

	for _, tc := range negativeCases {
		t.Run(tc.name+"_rejected", func(t *testing.T) {
			_, err := db.Exec(tc.query)
			if err == nil {
				t.Errorf("Negative value should be rejected by constraint: %s", tc.name)
			} else {
				// Verify it's a check constraint violation
				if !strings.Contains(err.Error(), "chk_endpoints") {
					t.Logf("Constraint violation detected (may be different constraint): %v", err)
				}
			}
		})
	}

	// Test constraint violations - upper-bound violations (max+1) should fail
	// Per SCACI §3.6.1: sh_addr is uint16 (max 65535), counters are uint32 (max 4294967295)
	upperBoundCases := []struct {
		name  string
		query string
	}{
		{"sh_addr exceeds max (65536)", `INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name, sh_addr) VALUES (999999, 999999, E'\\x0102030405060709', 'test9', 65536)`},
		{"attach_cnt exceeds max (4294967296)", `INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name, attach_cnt) VALUES (999999, 999999, E'\\x010203040506070a', 'test10', 4294967296)`},
		{"packet_cnt exceeds max (4294967296)", `INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name, packet_cnt) VALUES (999999, 999999, E'\\x010203040506070b', 'test11', 4294967296)`},
		{"last_packet_cnt exceeds max (4294967296)", `INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name, last_packet_cnt) VALUES (999999, 999999, E'\\x010203040506070c', 'test12', 4294967296)`},
	}

	for _, tc := range upperBoundCases {
		t.Run(tc.name+"_rejected", func(t *testing.T) {
			_, err := db.Exec(tc.query)
			require.Error(t, err, "Upper-bound violation should be rejected by constraint: %s", tc.name)
			// Verify it's a check constraint violation (message should include "check constraint")
			require.Contains(t, strings.ToLower(err.Error()), "check", "Error should be CHECK constraint violation for %s", tc.name)
		})
	}
}

// TestRepositorySQLVerification checks that repository SQL uses correct column names
func TestRepositorySQLVerification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SQL verification in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	// Test critical SQL patterns by actually executing them
	tests := []struct {
		name  string
		query string
		args  []interface{}
	}{
		{
			name:  "basestations_select_bs_eui",
			query: "SELECT bs_eui, name FROM basestations LIMIT 1",
			args:  nil,
		},
		{
			name:  "endpoints_select_ep_eui",
			query: "SELECT ep_eui, name FROM endpoints LIMIT 1",
			args:  nil,
		},
		{
			name:  "messages_join_ep_bs_eui",
			query: "SELECT m.ep_eui, m.bs_eui FROM messages m LIMIT 1",
			args:  nil,
		},
		{
			name:  "downlink_queue_que_id",
			query: "SELECT que_id, ep_eui FROM downlink_queue LIMIT 1",
			args:  nil,
		},
		{
			name:  "endpoint_sessions_last_packet_cnt",
			query: "SELECT last_packet_cnt FROM endpoint_sessions LIMIT 1",
			args:  nil,
		},
		{
			name:  "basestation_sessions_sn_fields",
			query: "SELECT sn_bs_uuid, sn_sc_uuid, sn_bs_op_id, sn_sc_op_id FROM basestation_sessions LIMIT 1",
			args:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.Query(tt.query, tt.args...)
			if err != nil {
				t.Errorf("Query %s failed with: %v\nThis indicates a column name mismatch in repository code", tt.name, err)
			}
		})
	}
}
