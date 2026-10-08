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

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
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
	1:   validateInitialSchema,
	2:   validateBasestationExtensions, // checks migration 002 actual artifacts
	3:   validateEndpointExtensions,    // checks migration 003 actual artifacts
	4:   validateBasestationReceptions, // checks migration 004 actual artifacts
	5:   validateEndpointSessions,      // checks migration 005 actual artifacts
	6:   validateEndpointKeys,          // checks migration 006 actual artifacts
	7:   validateBasestationSessions,   // checks migration 007 actual artifacts
	8:   validatePerformanceIndexes,
	11:  validateMessagePartitioning,
	12:  validateKeyStorageSchema, // TO UPDATE - needs migration 012 actual artifacts
	13:  validateSecurityConstraints,
	14:  validateArchiveTables,
	145: validateDropEndpointKeys,
	147: validateDropPlaceholderStatistics,
	148: validateArchiveOperationTracking,
	149: validateConsolidateCertificates,
	150: validateConsolidateConnectionEvents,
	151: validateDropDeadColumns,
	152: validateDropSubpackets,
	153: validateDropKeyStorage,
	154: validateRemoveOrganizationQuotas,
	155: validateCorrectMessageDeduplication,
	156: validateScaciFailedOperationsIndex,
	157: validateTrafficFilterIndexes,
	158: validateErrorCenterIndex,
	159: validateBaseStationSessionServiceCenter,
	160: validateDownlinkApplicationQueueID,
	161: validateUplinkReceptionTimeAndDetachCounter,
	162: validateDeadDownlinkQueueColumnsDropped,
	163: validateMessagePacketCounterReuse,
	164: validateDownlinkEndpointAcknowledgement,
	165: validateDownlinkWindowClaim,
	166: validateUnsignedApplicationQueueID,
	167: validateUnsignedMessagePacketCounter,
	168: validateEndpointStatusChangedAt,
	169: validateApplicationQueueIDZero,
	170: validateEndpointAttachmentChangedAt,
	171: validateLiveSessionPerOrganization,
	172: validateApplicationQueueIDInFlight,
	174: validateGPSWithoutFixCleared,
	175: validateSystemEventStatusNew,
	176: validateRoleGrandfathering,
	177: validateEventDeviceEUIsCanonical,
	178: validateBSSCIServiceCenterURLOptional,
	179: validateEventDetailKeysCamelCase,
	180: validateSCACISessionServiceCenter,
	181: validateCertificateGeneratedBaseStationCategory,
	182: validateStoredRowNotifications,
	183: validateMovedEventNotifications,
	184: validateDownlinkOrigin,
	185: validateStreamStorageOrder,
	186: validateSCACIUplinkDeliveryIdentity,
	187: validateDownlinkCommandRef,
	188: validateDownlinkAckDelivery,
	189: validateDownlinkCommandRefUnique,
	190: validateDownlinkRevoking,
	191: validateDownlinkSentAfterExpiry,
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

func validateDropEndpointKeys(t *testing.T, db *sql.DB) {
	// Migration 145 retires the endpoint_keys subsystem: the table, its archive,
	// and the two key-only functions are gone, while the shared archive/audit
	// functions remain for the other archive tables.
	assert.False(t, tableExists(t, db, "endpoint_keys"),
		"endpoint_keys should be dropped by migration 145")
	assert.False(t, tableExists(t, db, "endpoint_keys_archive"),
		"endpoint_keys_archive should be dropped by migration 145")

	assert.False(t, functionExists(t, db, "validate_key_format"),
		"validate_key_format() should be dropped by migration 145")
	assert.False(t, functionExists(t, db, "ensure_single_active_key"),
		"ensure_single_active_key() should be dropped by migration 145")

	assert.True(t, functionExists(t, db, "set_archived_at"),
		"shared set_archived_at() must survive migration 145")
	assert.True(t, functionExists(t, db, "update_audit_fields"),
		"shared update_audit_fields() must survive migration 145")
}

func validateDropPlaceholderStatistics(t *testing.T, db *sql.DB) {
	assert.False(t, tableExists(t, db, "table_statistics"))
	assert.False(t, functionExists(t, db, "update_table_statistics"))
}

func validateArchiveOperationTracking(t *testing.T, db *sql.DB) {
	assert.False(t, tableExists(t, db, "bssci_operation_tracking"))
	assert.True(t, tableExists(t, db, "bssci_pending_operations"), "canonical pending operations must survive")
}

func validateConsolidateCertificates(t *testing.T, db *sql.DB) {
	assert.False(t, tableExists(t, db, "basestation_certificates"))
	for _, column := range []string{"tls_ca_certificate", "tls_certificate", "tls_key", "tls_cert_fingerprint", "tls_cert_expires_at"} {
		assert.Contains(t, getColumnList(t, db, "basestations"), column)
	}
}

func validateConsolidateConnectionEvents(t *testing.T, db *sql.DB) {
	assert.False(t, tableExists(t, db, "basestation_connection_events"))
}

func validateDropDeadColumns(t *testing.T, db *sql.DB) {
	downlinkColumns := getColumnList(t, db, "downlink_queue")
	for _, column := range []string{"response_to_message_id", "retry_interval", "transmission_status", "tx_power_dbm", "transmitted_by_basestation_id", "correlation_id", "retry_count"} {
		assert.NotContains(t, downlinkColumns, column)
	}
	assert.NotContains(t, getColumnList(t, db, "basestations"), "hardware_name")
	endpointColumns := getColumnList(t, db, "endpoints")
	assert.NotContains(t, endpointColumns, "nwk_addr")
	assert.NotContains(t, endpointColumns, "ep_eui_alt")
	sessionColumns := getColumnList(t, db, "basestation_sessions")
	for _, column := range []string{"messages_received", "messages_sent", "bytes_received", "bytes_sent"} {
		assert.NotContains(t, sessionColumns, column)
	}
	downlinkIndexes := getIndexList(t, db, "downlink_queue")
	assert.Contains(t, downlinkIndexes, "idx_downlink_queue_mioty_prio")
	for _, index := range []string{"idx_downlink_queue_correlation", "idx_downlink_queue_response", "idx_downlink_queue_retry"} {
		assert.NotContains(t, downlinkIndexes, index)
	}
	assert.NotContains(t, getConstraintList(t, db, "downlink_queue"), "downlink_queue_retry_limit")
}

func validateDropSubpackets(t *testing.T, db *sql.DB) {
	assert.False(t, tableExists(t, db, "mioty_subpackets"))
	assert.True(t, tableExists(t, db, "mioty_message_deduplication"), "deduplication table is live and must survive")
}

func validateDropKeyStorage(t *testing.T, db *sql.DB) {
	for _, table := range []string{"encryption_keys", "key_usage_log", "encrypted_fields"} {
		assert.False(t, tableExists(t, db, table), "%s should be dropped by migration 153", table)
	}
	for _, fn := range []string{"rotate_encryption_key", "check_key_expiration", "log_key_operation", "detect_anomalous_access"} {
		assert.False(t, functionExists(t, db, fn), "%s() should be dropped by migration 153", fn)
	}
	assert.True(t, functionExists(t, db, "update_audit_fields"), "shared update_audit_fields() must survive migration 153")
	assert.True(t, functionExists(t, db, "enforce_tenant_isolation"), "enforce_tenant_isolation() must survive migration 153")
}

func validateRemoveOrganizationQuotas(t *testing.T, db *sql.DB) {
	orgColumns := getColumnList(t, db, "organizations")
	for _, column := range []string{"can_have_base_stations", "max_base_station_count", "max_endpoint_count"} {
		assert.NotContains(t, orgColumns, column, "organizations.%s should be dropped by migration 154", column)
	}
	tenantColumns := getColumnList(t, db, "tenants")
	for _, column := range []string{"max_basestations", "max_endpoints"} {
		assert.NotContains(t, tenantColumns, column, "tenants.%s should be dropped by migration 154", column)
	}
}

func validateCorrectMessageDeduplication(t *testing.T, db *sql.DB) {
	dedupColumns := getColumnList(t, db, "mioty_message_deduplication")
	for _, column := range []string{"owner_tenant_id", "ep_eui", "packet_cnt", "message_hash", "first_message_id", "first_bs_eui", "first_received_at", "last_received_at", "duplicate_count"} {
		assert.Contains(t, dedupColumns, column, "mioty_message_deduplication.%s should exist after migration 155", column)
	}
	assert.NotContains(t, dedupColumns, "id", "the legacy surrogate key should be gone")
	assert.True(t, tableExists(t, db, "message_delivery_outbox"), "message_delivery_outbox should exist after migration 155")
	assert.Contains(t, getIndexList(t, db, "message_delivery_outbox"), "idx_message_delivery_outbox_due")
}

func validateScaciFailedOperationsIndex(t *testing.T, db *sql.DB) {
	assert.Contains(t, getIndexList(t, db, "scaci_operation_log"), "idx_scaci_op_log_tenant_failed", "failed-operation groups need their partial index after migration 156")
}

func validateTrafficFilterIndexes(t *testing.T, db *sql.DB) {
	messageIndexes := getIndexList(t, db, "messages")
	for _, index := range []string{"idx_messages_tenant_duplicate", "idx_messages_tenant_dl_open", "idx_messages_tenant_profile_mode"} {
		assert.Contains(t, messageIndexes, index, "messages.%s should exist after migration 157", index)
	}
	assert.Contains(t, getIndexList(t, db, "downlink_queue"), "idx_downlink_queue_tenant_status_priority", "queue view ordering needs its index after migration 157")
}

func validateErrorCenterIndex(t *testing.T, db *sql.DB) {
	assert.Contains(t, getIndexList(t, db, "system_events"), "idx_system_events_tenant_failures", "the errors center scan needs its partial index after migration 158")
}

func validateBaseStationSessionServiceCenter(t *testing.T, db *sql.DB) {
	assert.Contains(t, getColumnList(t, db, "basestation_sessions"), "sc_eui", "session rows record their owning service center after migration 159")
	assert.Contains(t, getIndexList(t, db, "basestation_sessions"), "idx_basestation_sessions_active_sc_eui", "the startup reconciliation scan needs its partial index after migration 159")
}

func validateSCACISessionServiceCenter(t *testing.T, db *sql.DB) {
	assert.Contains(t, getColumnList(t, db, "scaci_sessions"), "sc_eui", "SCACI session rows record their owning service center after migration 180")
	assert.Contains(t, getIndexList(t, db, "scaci_sessions"), "idx_scaci_sessions_live_sc_eui", "the startup reconciliation scan needs its partial index after migration 180")
}

// sqlAuditCertificateGenerated counts the certificate.generated events still
// filed under audit; the preflight runs the same count.
const sqlAuditCertificateGenerated = `SELECT count(*) FROM system_events
	WHERE event_type = 'certificate.generated' AND event_category = 'audit'`

func validateCertificateGeneratedBaseStationCategory(t *testing.T, db *sql.DB) {
	var underAudit int
	require.NoError(t, db.QueryRow(sqlAuditCertificateGenerated).Scan(&underAudit))
	assert.Zero(t, underAudit, "every certificate.generated event is filed under basestation after migration 181")
}

func validateUplinkReceptionTimeAndDetachCounter(t *testing.T, db *sql.DB) {
	var dataType, nullable string
	require.NoError(t, db.QueryRow(`SELECT data_type, is_nullable FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'mioty_message_deduplication' AND column_name = 'first_rx_time'`).Scan(&dataType, &nullable))
	assert.Equal(t, "bigint", dataType, "first_rx_time holds a nanosecond radio time after migration 161")
	assert.Equal(t, "NO", nullable, "every classifier row carries its reception time after migration 161")
	for _, column := range []string{"packet_cnt", "last_packet_cnt", "last_detach_packet_cnt"} {
		require.NoError(t, db.QueryRow(`SELECT data_type FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'endpoints' AND column_name = $1`, column).Scan(&dataType))
		assert.Equal(t, "bigint", dataType, "endpoints.%s holds the full 32-bit packet counter", column)
	}
}

func validateDownlinkApplicationQueueID(t *testing.T, db *sql.DB) {
	assert.Contains(t, getColumnList(t, db, "downlink_queue"), "ac_que_id")
	assert.Contains(t, getIndexList(t, db, "downlink_queue"), "idx_downlink_queue_tenant_ac_que_id")
	assert.Contains(t, getConstraintList(t, db, "downlink_queue"), "downlink_queue_ac_que_id_positive")
	assert.Contains(t, getConstraintList(t, db, "downlink_queue"), "unique_queue_id", "que_id stays the installation-wide service center id")
}

func validateMessagePacketCounterReuse(t *testing.T, db *sql.DB) {
	assert.Contains(t, getColumnList(t, db, "messages"), "packet_cnt_reused")
	assert.Contains(t, getColumnList(t, db, "messages_archive"), "packet_cnt_reused", "the archive mirrors the live layout")
}

func validateDeadDownlinkQueueColumnsDropped(t *testing.T, db *sql.DB) {
	downlinkColumns := getColumnList(t, db, "downlink_queue")
	for _, column := range []string{"prio", "valid_until", "packet_cnt_array"} {
		assert.NotContains(t, downlinkColumns, column, "downlink_queue.%s is dropped after migration 162", column)
	}
	assert.Contains(t, downlinkColumns, "failure_reason")
	assert.NotContains(t, getIndexList(t, db, "downlink_queue"), "idx_downlink_queue_mioty_prio")
}

func validateDownlinkEndpointAcknowledgement(t *testing.T, db *sql.DB) {
	assert.Contains(t, getColumnList(t, db, "downlink_queue"), "endpoint_acked_at")
	assert.Contains(t, getIndexList(t, db, "downlink_queue"), "idx_downlink_queue_ack_window")
}

func validateDownlinkWindowClaim(t *testing.T, db *sql.DB) {
	assert.Contains(t, getColumnList(t, db, "messages"), "dl_window_claimed")
	assert.Contains(t, getColumnList(t, db, "messages_archive"), "dl_window_claimed", "the archive mirrors the live layout")
}

func validateUnsignedApplicationQueueID(t *testing.T, db *sql.DB) {
	var dataType string
	require.NoError(t, db.QueryRow(`SELECT data_type FROM information_schema.columns
		WHERE table_name = 'downlink_queue' AND column_name = 'ac_que_id'`).Scan(&dataType))
	assert.Equal(t, "numeric", dataType, "ac_que_id holds the full unsigned 64-bit range")
}

func validateApplicationQueueIDZero(t *testing.T, db *sql.DB) {
	var check string
	require.NoError(t, db.QueryRow(`SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conname = 'downlink_queue_ac_que_id_unsigned_64'`).Scan(&check))
	assert.Contains(t, check, ">= (0)", "ac_que_id accepts the Application Center queue id 0")
}

func validateUnsignedMessagePacketCounter(t *testing.T, db *sql.DB) {
	for _, table := range []string{"messages", "messages_archive"} {
		var dataType string
		require.NoError(t, db.QueryRow(`SELECT data_type FROM information_schema.columns
			WHERE table_name = $1 AND column_name = 'packet_cnt'`, table).Scan(&dataType))
		assert.Equal(t, "bigint", dataType, "%s.packet_cnt holds the full unsigned 32-bit range", table)
	}
}

func validateLiveSessionPerOrganization(t *testing.T, db *sql.DB) {
	var definition string
	require.NoError(t, db.QueryRow(`SELECT indexdef FROM pg_indexes
		WHERE schemaname = 'public' AND indexname = 'idx_scaci_sessions_unique_active'`).Scan(&definition))
	assert.Contains(t, definition, "(tenant_id, organization_id, ac_eui) NULLS NOT DISTINCT",
		"one live session per Application Center of an organization")
}

func validateSystemEventStatusNew(t *testing.T, db *sql.DB) {
	var columnDefault, nullable string
	require.NoError(t, db.QueryRow(`SELECT column_default, is_nullable FROM information_schema.columns
		WHERE table_name = 'system_events' AND column_name = 'status'`).Scan(&columnDefault, &nullable))
	assert.Contains(t, columnDefault, "'new'", "an unhandled event is new")
	assert.Equal(t, "NO", nullable, "every event has a status")
	var active int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM system_events WHERE status = 'active'`).Scan(&active))
	assert.Zero(t, active, "no event keeps the retired active status")
	var constraints []string
	rows, err := db.Query(`SELECT conname || ':' || convalidated FROM pg_constraint
		WHERE conrelid = 'system_events'::regclass AND contype = 'c' AND conname LIKE 'system_events_status%'`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	for rows.Next() {
		var c string
		require.NoError(t, rows.Scan(&c))
		constraints = append(constraints, c)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"system_events_status_check:true"}, constraints,
		"the status check is validated and the NOT NULL helper check is gone")
}

func validateApplicationQueueIDInFlight(t *testing.T, db *sql.DB) {
	var definition string
	require.NoError(t, db.QueryRow(`SELECT indexdef FROM pg_indexes
		WHERE schemaname = 'public' AND indexname = 'idx_downlink_queue_org_ac_que_id_in_flight'`).Scan(&definition))
	assert.Contains(t, definition, "(tenant_id, organization_id, ac_que_id)", "an Application Center queue id is scoped to its organization")
	assert.Contains(t, definition, "downlink_queue_in_flight", "and to the downlinks still in flight")
	assert.NotContains(t, getIndexList(t, db, "downlink_queue"), "idx_downlink_queue_tenant_ac_que_id")
	known := append([]mioty.DLQueueStatus{mioty.DLQueueStatusPending, mioty.DLQueueStatusScheduled,
		mioty.DLQueueStatusReserved, mioty.DLQueueStatusQueued}, mioty.TerminalStatuses()...)
	for _, status := range known {
		require.True(t, status.Known(), status)
		var inFlight bool
		require.NoError(t, db.QueryRow(`SELECT downlink_queue_in_flight($1)`, string(status)).Scan(&inFlight))
		assert.Equal(t, !status.Terminal(), inFlight, "downlink_queue_in_flight(%s) is the complement of Terminal", status)
	}
}

func validateRoleGrandfathering(t *testing.T, db *sql.DB) {
	for _, table := range []string{"role_grandfathered_users", "role_grandfathered_memberships"} {
		var n int
		require.NoError(t, db.QueryRow(`SELECT count(*) FROM information_schema.tables
			WHERE table_schema = 'identity' AND table_name = $1`, table).Scan(&n))
		assert.Equal(t, 1, n, "identity.%s records the switches 000176 turned on", table)
	}
}

func validateGPSWithoutFixCleared(t *testing.T, db *sql.DB) {
	var noFixPositions int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM basestations
		WHERE location_source = 'gps' AND latitude = 0 AND longitude = 0`).Scan(&noFixPositions))
	assert.Zero(t, noFixPositions, "no GPS report without a fix is stored as a position")
}

// sqlNonCanonicalEventDeviceEUIs counts the events whose bsEui or epEui detail
// is an EUI not in the canonical form; the preflight runs the same count.
const sqlNonCanonicalEventDeviceEUIs = `SELECT count(*) FROM system_events
	WHERE (data ? 'bsEui' OR data ? 'epEui') AND jsonb_typeof(data) = 'object' AND (
	     (jsonb_typeof(data->'bsEui') = 'string' AND data->>'bsEui' !~ '^[0-9A-F]{16}$' AND translate(data->>'bsEui', '-:', '') ~ '^[0-9A-Fa-f]{16}$')
	  OR (jsonb_typeof(data->'epEui') = 'string' AND data->>'epEui' !~ '^[0-9A-F]{16}$' AND translate(data->>'epEui', '-:', '') ~ '^[0-9A-Fa-f]{16}$')
	  OR (jsonb_typeof(data->'bsEui') = 'number' AND data->>'bsEui' ~ '^[0-9]{1,20}$' AND (data->>'bsEui')::numeric <= 18446744073709551615)
	  OR (jsonb_typeof(data->'epEui') = 'number' AND data->>'epEui' ~ '^[0-9]{1,20}$' AND (data->>'epEui')::numeric <= 18446744073709551615))`

func validateEventDeviceEUIsCanonical(t *testing.T, db *sql.DB) {
	var nonCanonical int
	require.NoError(t, db.QueryRow(sqlNonCanonicalEventDeviceEUIs).Scan(&nonCanonical))
	assert.Zero(t, nonCanonical, "every event names its device EUIs in the canonical form")
}

// sqlSnakeCaseEventDetailKeys counts the events that still hold a detail key
// migration 179 renames; the preflight runs the same count.
const sqlSnakeCaseEventDetailKeys = `SELECT count(*) FROM system_events
	WHERE jsonb_typeof(data) = 'object'
	  AND data ?| ARRAY['basestation_name', 'basestation_id', 'endpoint_id', 'operation_id', 'operation_type',
	                    'target_bs', 'target_bs_list', 'target_bs_count', 'is_online', 'connection_type', 'session_id']`

func validateEventDetailKeysCamelCase(t *testing.T, db *sql.DB) {
	var snakeCase int
	require.NoError(t, db.QueryRow(sqlSnakeCaseEventDetailKeys).Scan(&snakeCase))
	assert.Zero(t, snakeCase, "every event names its details in camelCase")
}

func validateBSSCIServiceCenterURLOptional(t *testing.T, db *sql.DB) {
	var checks int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM pg_constraint
		WHERE conrelid = 'basestations'::regclass AND conname = 'check_bssci_config'`).Scan(&checks))
	assert.Zero(t, checks, "a BSSCI base station may store an unknown Service Center URL as NULL")
}

func validateEndpointStatusChangedAt(t *testing.T, db *sql.DB) {
	var dataType string
	require.NoError(t, db.QueryRow(`SELECT data_type FROM information_schema.columns
		WHERE table_name = 'endpoints' AND column_name = 'status_changed_at'`).Scan(&dataType))
	assert.Equal(t, "timestamp with time zone", dataType, "endpoints.status_changed_at records the status decision time")
}

func validateEndpointAttachmentChangedAt(t *testing.T, db *sql.DB) {
	var dataType string
	require.NoError(t, db.QueryRow(`SELECT data_type FROM information_schema.columns
		WHERE table_name = 'endpoints' AND column_name = 'attachment_changed_at'`).Scan(&dataType))
	assert.Equal(t, "timestamp with time zone", dataType, "endpoints.attachment_changed_at records the attachment decision time")
	var renamedAway int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM information_schema.columns
		WHERE table_name = 'endpoints' AND column_name = 'status_changed_at'`).Scan(&renamedAway))
	assert.Zero(t, renamedAway, "endpoints.status_changed_at is renamed, not copied")
}

func validateFinalSchema(t *testing.T, db *sql.DB) {
	// Validate all tables exist
	expectedTables := []string{
		"tenants", "endpoints", "basestations", "messages", "downlink_queue",
		"roaming_agreements", "basestation_receptions", "endpoint_sessions",
		"basestation_sessions", "system_events",
		"messages_archive", "basestation_receptions_archive",
		"endpoint_sessions_archive",
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

func functionExists(t *testing.T, db *sql.DB, functionName string) bool {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_proc p
			JOIN pg_namespace n ON p.pronamespace = n.oid
			WHERE n.nspname = 'public' AND p.proname = $1
		)
	`, functionName).Scan(&exists)
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
