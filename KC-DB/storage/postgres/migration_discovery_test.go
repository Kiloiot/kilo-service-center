package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrationDiscovery validates that the auto-discovery mechanism correctly
// finds all migration files and extracts metadata
func TestMigrationDiscovery(t *testing.T) {
	migrations := discoverMigrations(t)

	// We have 127 migration files (numbered 1-133 with gaps at 24, 25, 26, 52, 78, 79)
	assert.Len(t, migrations, 127, "Should discover exactly 127 migrations")

	// Verify migrations are sorted by number
	for i := 1; i < len(migrations); i++ {
		assert.Less(t, migrations[i-1].number, migrations[i].number,
			"Migrations should be sorted by number")
	}

	// Verify first migration
	assert.Equal(t, 1, migrations[0].number, "First migration should be #1")
	assert.NotEmpty(t, migrations[0].description, "Migration should have description")

	// Verify last migration
	assert.Equal(t, 133, migrations[len(migrations)-1].number, "Last migration should be #133")

	// Verify specific migrations exist
	expectedMigrations := map[int]string{
		1:   "initial_schema",
		14:  "archive_tables",
		28:  "mioty_persistent_db_compliance",
		60:  "standardize_basestation_eui",
		61:  "add_session_encoding",
		62:  "add_endpoint_subpackets",
		63:  "relax_session_key_constraint",
		80:  "enforce_global_eui_uniqueness",
		81:  "add_downlink_reserved_queued_status",
		82:  "add_tls_metadata_to_scaci_sessions",
		83:  "add_negotiated_version_to_scaci_sessions",
		84:  "fix_eui_column_references",
		85:  "add_que_id_sequence",
		86:  "drop_obsolete_session_id_from_bs_sessions",
		87:  "fix_scaci_register_unsigned_fields",
		88:  "add_uldata_multi_bs_support",
		89:  "add_dl_rx_stat_qry",
		90:  "add_org_uuid_to_downlink_queue",
		91:  "add_scaci_completed_with_warnings_state",
		92:  "scaci_error_columns",
		93:  "enforce_positive_tenant_id",
		94:  "add_users_table",
		95:  "convert_org_members_user_id_uuid",
		96:  "add_refresh_tokens_table",
		97:  "add_org_external_id",
		98:  "add_user_role_flags",
		99:  "add_org_extended_fields",
		100: "add_org_member_flags",
		101: "reconcile_event_categories",
		102: "create_manufacturers_table",
		103: "create_device_models_table",
		104: "create_blueprints_table",
		105: "add_decoded_payload_to_messages",
		106: "add_device_model_to_endpoints",
		107: "simplify_manufacturers",
		108: "add_nwk_sn_key_to_mioty_messages",
		72:  "create_api_keys",
		109: "integrations",
		111: "create_legacy_compatibility_views",
		112: "drop_orphaned_endpoint_packet_function",
		113: "allow_nullable_blueprint_type_eui",
		114: "add_case_insensitive_name_indexes",
		115: "enforce_model_code_format",
		116: "identity_schema",
		117: "backfill_admin_manager_flags",
		118: "drop_downlink_port_confirmed",
		119: "add_user_profile_fields",
		120: "normalize_email_case",
		121: "gin_index_base_stations",
		122: "add_location_source",
		123: "create_ce_installation",
		124: "create_ce_registry",
		125: "create_federation_outbox",
		126: "create_federation_receipts",
		127: "federation_receipts_integrity",
		128: "drop_basestation_legacy_columns",
		129: "drop_legacy_compatibility_views",
		130: "backfill_downlink_organization_id",
		131: "normalize_event_taxonomy",
		132: "seed_default_admin",
		133: "add_endpoint_preshared_key",
	}

	for num, expectedDesc := range expectedMigrations {
		found := false
		for _, mig := range migrations {
			if mig.number == num {
				found = true
				assert.Contains(t, mig.description, expectedDesc,
					"Migration %d should have description containing '%s'", num, expectedDesc)
				break
			}
		}
		assert.True(t, found, "Migration %d should be discovered", num)
	}

	// Verify missing migrations (24, 25, 26, 52, 78, 79) are not present
	missingNumbers := []int{24, 25, 26, 52, 78, 79}
	for _, missing := range missingNumbers {
		for _, mig := range migrations {
			assert.NotEqual(t, missing, mig.number,
				"Migration %d should not exist (known gap)", missing)
		}
	}

	// Verify validators are mapped correctly for migrations 1-14
	validatedMigrations := []int{1, 2, 3, 4, 5, 6, 7, 8, 11, 12, 13, 14}
	for _, num := range validatedMigrations {
		for _, mig := range migrations {
			if mig.number == num {
				assert.NotNil(t, mig.validate,
					"Migration %d should have a validator function", num)
				break
			}
		}
	}

	// Log discovered migrations for debugging
	t.Logf("Discovered %d migrations:", len(migrations))
	for i, mig := range migrations {
		if i < 5 || i >= len(migrations)-5 {
			// Log first 5 and last 5
			validator := "no validator"
			if mig.validate != nil {
				validator = "has validator"
			}
			t.Logf("  Migration %d: %s (%s)", mig.number, mig.description, validator)
		} else if i == 5 {
			t.Logf("  ... (%d more) ...", len(migrations)-10)
		}
	}
}

// TestExtractMigrationInfo validates the filename parsing logic
func TestExtractMigrationInfo(t *testing.T) {
	testCases := []struct {
		filename     string
		expectedNum  int
		expectedDesc string
	}{
		{"001_initial_schema.up.sql", 1, "initial_schema"},
		{"014_archive_tables.up.sql", 14, "archive_tables"},
		{"028_mioty_persistent_db_compliance.up.sql", 28, "mioty_persistent_db_compliance"},
		{"000060_standardize_basestation_eui.up.sql", 60, "standardize_basestation_eui"},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			num, desc := extractMigrationInfo(tc.filename)
			assert.Equal(t, tc.expectedNum, num, "Extracted number should match")
			assert.Equal(t, tc.expectedDesc, desc, "Extracted description should match")
		})
	}

	// Test invalid formats
	invalidCases := []string{
		"invalid.up.sql",
		"no_number.up.sql",
		"001.up.sql",
	}

	for _, invalid := range invalidCases {
		t.Run("Invalid_"+invalid, func(t *testing.T) {
			num, _ := extractMigrationInfo(invalid)
			assert.Equal(t, 0, num, "Invalid format should return 0")
		})
	}
}

// TestMigrationValidatorsMapCompleteness verifies that all validators in the map
// correspond to actual migration files
func TestMigrationValidatorsMapCompleteness(t *testing.T) {
	migrations := discoverMigrations(t)

	// Build map of existing migration numbers
	existingNums := make(map[int]bool)
	for _, mig := range migrations {
		existingNums[mig.number] = true
	}

	// Verify all validators correspond to existing migrations
	for num := range migrationValidators {
		assert.True(t, existingNums[num],
			"Validator for migration %d exists but migration file doesn't", num)
	}

	// Count how many migrations have validators
	validatedCount := 0
	for _, mig := range migrations {
		if mig.validate != nil {
			validatedCount++
		}
	}

	t.Logf("Migration validation coverage: %d/%d (%.1f%%)",
		validatedCount, len(migrations),
		float64(validatedCount)/float64(len(migrations))*100)

	// Goal: eventually have validators for critical migrations
	// Current: 12 validators (migrations 1-14 minus 9, 10)
	require.GreaterOrEqual(t, validatedCount, 12,
		"Should have at least 12 validators (migrations 1-14)")
}
