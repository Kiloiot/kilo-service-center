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

// migrationHarness drives a migrate instance over a fresh container and
// exposes the schema probes the 147-153 tests share.
type migrationHarness struct {
	t  *testing.T
	db *sql.DB
	m  *migrate.Migrate
	// dsn names the harness database, for connections of their own such as a notification listener.
	dsn string
}

func newMigrationHarness(t *testing.T) *migrationHarness {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping migration test in short mode")
	}
	db, cfg, cleanup := SetupPostgresContainerWithoutMigrations(t)
	t.Cleanup(cleanup)

	migrationsDir, err := filepath.Abs("../../migrations")
	require.NoError(t, err)
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	require.NoError(t, err)
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", filepath.ToSlash(migrationsDir)), "postgres", driver)
	require.NoError(t, err)
	return &migrationHarness{t: t, db: db.DB, m: m, dsn: cfg.DSN}
}

func (h *migrationHarness) migrateTo(version uint) {
	h.t.Helper()
	require.NoError(h.t, h.m.Migrate(version), "migrate to %d", version)
}

// migrateExpectingGuard runs a migration that must abort on its data guard and
// resets the version so the caller can repair the fixture and retry.
func (h *migrationHarness) migrateExpectingGuard(version uint, fragment string) {
	h.t.Helper()
	err := h.m.Migrate(version)
	require.Error(h.t, err, "migration %d must abort on its data guard", version)
	assert.Contains(h.t, err.Error(), fragment)
	require.NoError(h.t, h.m.Force(int(version-1)))
}

func (h *migrationHarness) exec(query string, args ...interface{}) {
	h.t.Helper()
	_, err := h.db.Exec(query, args...)
	require.NoError(h.t, err, query)
}

func (h *migrationHarness) queryInt(query string, args ...interface{}) int64 {
	h.t.Helper()
	var n int64
	require.NoError(h.t, h.db.QueryRow(query, args...).Scan(&n), query)
	return n
}

func (h *migrationHarness) tableExists(name string) bool {
	return h.queryInt(`SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1`, name) == 1
}

func (h *migrationHarness) columnExists(table, column string) bool {
	return h.queryInt(`SELECT count(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2`, table, column) == 1
}

func (h *migrationHarness) functionExists(name string) bool {
	return h.queryInt(`SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON p.pronamespace = n.oid WHERE n.nspname = 'public' AND p.proname = $1`, name) >= 1
}

func (h *migrationHarness) indexDefinition(name string) string {
	var def sql.NullString
	require.NoError(h.t, h.db.QueryRow(`SELECT indexdef FROM pg_indexes WHERE schemaname = 'public' AND indexname = $1`, name).Scan(&def))
	return def.String
}

func (h *migrationHarness) seedTenantAndBaseStation(eui string) (tenantID, basestationID int64) {
	h.t.Helper()
	require.NoError(h.t, h.db.QueryRow(`INSERT INTO tenants (name) VALUES ($1) RETURNING id`, "migration-"+eui).Scan(&tenantID))
	require.NoError(h.t, h.db.QueryRow(
		`INSERT INTO basestations (bs_eui, name, tenant_id, connection_type, service_center_url)
		 VALUES (decode($1, 'hex'), $2, $3, 'bssci', 'kilocenter.local:5000') RETURNING id`,
		eui, "bs-"+eui, tenantID).Scan(&basestationID))
	return tenantID, basestationID
}

func (h *migrationHarness) seedSession(tenantID, basestationID int64) int64 {
	h.t.Helper()
	var id int64
	require.NoError(h.t, h.db.QueryRow(
		`INSERT INTO basestation_sessions (basestation_id, tenant_id) VALUES ($1, $2) RETURNING id`,
		basestationID, tenantID).Scan(&id))
	return id
}

func (h *migrationHarness) archivedEvents(eventType string) int64 {
	return h.queryInt(`SELECT count(*) FROM system_events WHERE event_type = $1 AND status = 'resolved' AND source_type = 'migration'`, eventType)
}

func TestMigration148ArchivesLegacyOperationTracking(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(147)

	tenantID, bsID := h.seedTenantAndBaseStation("70b3d59cd0000148")
	sessionID := h.seedSession(tenantID, bsID)
	h.exec(`INSERT INTO bssci_operation_tracking (session_id, op_id, command, direction, state, response_data, error_message)
	        VALUES ($1, 5, 'attPrp', 'outbound', 'completed', '{"result": 0}', NULL),
	               ($1, 6, 'dlDataQue', 'outbound', 'failed', NULL, 'timeout'),
	               ($1, -7, 'detPrp', 'outbound', 'pending', NULL, NULL)`, sessionID)

	h.migrateExpectingGuard(148, "no canonical bssci_pending_operations row")

	h.exec(`INSERT INTO bssci_pending_operations (basestation_session_id, operation_id, operation_type, operation_data)
	        VALUES ($1, -7, 'detPrp', '{"command": "detPrp", "opId": -7}')`, sessionID)
	h.migrateTo(148)

	assert.False(t, h.tableExists("bssci_operation_tracking"))
	assert.Equal(t, int64(3), h.archivedEvents("bssci.operation.archived"))
	assert.Equal(t, int64(3), h.queryInt(
		`SELECT count(*) FROM system_events WHERE event_type = 'bssci.operation.archived' AND tenant_id = $1 AND basestation_id = $2 AND event_category = 'bssci'`,
		tenantID, bsID))
	assert.Equal(t, int64(1), h.queryInt(
		`SELECT count(*) FROM system_events WHERE event_type = 'bssci.operation.archived' AND severity = 'warning' AND data->>'errorMessage' = 'timeout' AND data->>'command' = 'dlDataQue'`))
	assert.Equal(t, int64(1), h.queryInt(
		`SELECT count(*) FROM system_events WHERE event_type = 'bssci.operation.archived' AND (data->'responseData'->>'result')::int = 0 AND data->>'state' = 'completed'`))

	h.migrateTo(147)
	assert.True(t, h.tableExists("bssci_operation_tracking"))
	assert.Equal(t, int64(0), h.queryInt(`SELECT count(*) FROM bssci_operation_tracking`))
	assert.Equal(t, int64(3), h.archivedEvents("bssci.operation.archived"), "rollback keeps the archive")
	h.migrateTo(148)
}

func TestMigration149ConsolidatesCertificateRecords(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(148)

	tenantID, bsID := h.seedTenantAndBaseStation("70b3d59cd0000149")
	_, keyedBsID := h.seedTenantAndBaseStation("70b3d59cd000014a")
	h.exec(`INSERT INTO basestation_certificates
	        (basestation_id, certificate_type, certificate_pem, private_key_encrypted, fingerprint, subject_dn, issuer_dn, not_before, not_after, is_active)
	        VALUES ($1, 'ca', 'CA-PEM', NULL, 'ca-fp', 'CN=ca', 'CN=ca', NOW() - INTERVAL '1 day', NOW() + INTERVAL '1 year', true),
	               ($1, 'client', 'CLIENT-PEM', NULL, 'client-fp', 'CN=bs', 'CN=ca', NOW() - INTERVAL '1 day', NOW() + INTERVAL '30 days', true),
	               ($1, 'client', 'OLD-CLIENT-PEM', NULL, 'old-fp', 'CN=bs', 'CN=ca', NOW() - INTERVAL '2 years', NOW() - INTERVAL '1 year', false),
	               ($2, 'client', 'KEYED-PEM', '\x0102'::bytea, 'keyed-fp', 'CN=keyed', 'CN=ca', NOW() - INTERVAL '1 day', NOW() + INTERVAL '30 days', true)`,
		bsID, keyedBsID)

	h.migrateExpectingGuard(149, "no canonical basestations.tls_key")

	h.exec(`UPDATE basestations SET tls_key = 'envelope:v1:redacted', tls_certificate = 'KEYED-PEM' WHERE id = $1`, keyedBsID)
	h.migrateTo(149)

	assert.False(t, h.tableExists("basestation_certificates"))
	var ca, cert, fingerprint string
	var expires sql.NullTime
	require.NoError(t, h.db.QueryRow(`SELECT tls_ca_certificate, tls_certificate, tls_cert_fingerprint, tls_cert_expires_at FROM basestations WHERE id = $1`, bsID).
		Scan(&ca, &cert, &fingerprint, &expires))
	assert.Equal(t, "CA-PEM", ca)
	assert.Equal(t, "CLIENT-PEM", cert, "only the active client certificate is consolidated")
	assert.Equal(t, "client-fp", fingerprint)
	assert.True(t, expires.Valid)

	assert.Equal(t, int64(4), h.archivedEvents("basestation.certificate.archived"))
	assert.Equal(t, int64(3), h.queryInt(`SELECT count(*) FROM system_events WHERE event_type = 'basestation.certificate.archived' AND tenant_id = $1`, tenantID))
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM system_events WHERE event_type = 'basestation.certificate.archived' AND (data->>'hadPrivateKey')::boolean`))
	assert.Equal(t, int64(0), h.queryInt(`SELECT count(*) FROM system_events WHERE event_type = 'basestation.certificate.archived' AND data::text LIKE '%PEM%'`),
		"archived metadata never carries certificate bodies")

	h.migrateTo(148)
	assert.True(t, h.tableExists("basestation_certificates"))
	h.migrateTo(149)
}

func TestMigration150ArchivesConnectionEvents(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(149)

	tenantID, bsID := h.seedTenantAndBaseStation("70b3d59cd0000150")
	h.exec(`INSERT INTO basestation_connection_events (basestation_id, event_type, event_data, remote_address)
	        VALUES ($1, 'connected', '{"tls": "1.3"}', '192.0.2.10'),
	               ($1, 'disconnected', '{"reason": "eof"}', NULL)`, bsID)

	h.migrateTo(150)

	assert.False(t, h.tableExists("basestation_connection_events"))
	assert.Equal(t, int64(2), h.archivedEvents("basestation.connection.archived"))
	assert.Equal(t, int64(1), h.queryInt(
		`SELECT count(*) FROM system_events WHERE event_type = 'basestation.connection.archived' AND tenant_id = $1 AND basestation_id = $2
		   AND data->>'eventType' = 'connected' AND data->>'remoteAddress' = '192.0.2.10' AND data->'eventData'->>'tls' = '1.3'`,
		tenantID, bsID))

	h.migrateTo(149)
	assert.True(t, h.tableExists("basestation_connection_events"))
	h.migrateTo(150)
}

func TestMigration151DropsDeadColumns(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(150)

	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd0000151")
	var orgID string
	require.NoError(t, h.db.QueryRow(`INSERT INTO organizations (tenant_id, name) VALUES ($1, 'org-151') RETURNING org_id`, tenantID).Scan(&orgID))
	h.exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, payload, organization_id, earliest_at, latest_at, correlation_id)
	        VALUES (decode('70b3d59cd0000001', 'hex'), $1, '\x01'::bytea, $2::uuid, NOW(), NOW() + INTERVAL '1 hour', gen_random_uuid())`, tenantID, orgID)

	h.migrateExpectingGuard(151, "correlation_id is populated")

	h.exec(`UPDATE downlink_queue SET correlation_id = NULL`)
	h.migrateTo(151)

	for _, column := range []string{"response_to_message_id", "retry_interval", "transmission_status", "tx_power_dbm", "transmitted_by_basestation_id", "correlation_id", "retry_count"} {
		assert.False(t, h.columnExists("downlink_queue", column), "downlink_queue.%s must be dropped", column)
	}
	assert.False(t, h.columnExists("basestations", "hardware_name"))
	assert.False(t, h.columnExists("endpoints", "nwk_addr"))
	assert.False(t, h.columnExists("endpoints", "ep_eui_alt"))
	for _, column := range []string{"messages_received", "messages_sent", "bytes_received", "bytes_sent"} {
		assert.False(t, h.columnExists("basestation_sessions", column), "basestation_sessions.%s must be dropped", column)
	}
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM downlink_queue`), "queue rows survive the column drop")
	assert.Contains(t, h.indexDefinition("idx_downlink_queue_mioty_prio"), "(status, prio DESC, created_at)")
	assert.NotContains(t, h.indexDefinition("idx_downlink_queue_mioty_prio"), "transmitted_by_basestation_id")

	h.migrateTo(150)
	assert.True(t, h.columnExists("downlink_queue", "retry_count"))
	assert.True(t, h.columnExists("basestation_sessions", "bytes_sent"))
	assert.Contains(t, h.indexDefinition("idx_downlink_queue_mioty_prio"), "transmitted_by_basestation_id")
	h.migrateTo(151)
}

func TestMigration152And153DropOrphanedStorage(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(151)

	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd0000152")
	h.exec(`INSERT INTO mioty_subpackets (message_id, message_received_at, tenant_id, subpacket_index) VALUES (1, NOW(), $1, 0)`, tenantID)
	h.migrateExpectingGuard(152, "mioty_subpackets holds 1 row")
	h.exec(`DELETE FROM mioty_subpackets`)
	h.migrateTo(152)
	assert.False(t, h.tableExists("mioty_subpackets"))

	// The original 012 audit trigger (still in force before 153 ever ran)
	// inserts into a system_events column that never existed, so the guard
	// fixture can only be seeded with that trigger disabled.
	h.exec(`ALTER TABLE encryption_keys DISABLE TRIGGER encryption_keys_audit`)
	h.exec(`INSERT INTO encryption_keys (key_id, key_type, expires_at) VALUES ('k1', 'data', NOW() + INTERVAL '1 day')`)
	h.migrateExpectingGuard(153, "key storage tables hold 1 row")
	h.exec(`DELETE FROM encryption_keys`)
	h.migrateTo(153)
	h.assertKeyStorageDropped()

	// Rolling 153 back restores working objects: every restored function,
	// the trigger and the view are exercised with the trigger enabled.
	h.migrateTo(152)
	assert.True(t, h.tableExists("encryption_keys"))
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM pg_trigger WHERE tgname = 'encryption_keys_audit' AND NOT tgisinternal AND tgenabled = 'O'`))
	h.exerciseRestoredKeyStorage()

	h.exec(`DELETE FROM key_usage_log`)
	h.exec(`DELETE FROM encrypted_fields`)
	h.exec(`DELETE FROM encryption_keys`)
	h.migrateTo(153)
	h.assertKeyStorageDropped()
}

func (h *migrationHarness) assertKeyStorageDropped() {
	h.t.Helper()
	for _, table := range []string{"encryption_keys", "key_usage_log", "encrypted_fields"} {
		assert.False(h.t, h.tableExists(table), "%s must be dropped", table)
	}
	for _, fn := range []string{"rotate_encryption_key", "check_key_expiration", "log_key_operation", "detect_anomalous_access"} {
		assert.False(h.t, h.functionExists(fn), "%s() must be dropped", fn)
	}
	assert.Equal(h.t, int64(0), h.queryInt(`SELECT count(*) FROM information_schema.views WHERE table_schema = 'public' AND table_name = 'v_active_encryption_keys'`))
}

// exerciseRestoredKeyStorage drives every object the 153 down migration
// recreates: the audit trigger on insert and on a status change, the
// expiration classifier, the active-keys view, key rotation and the anomaly
// detector.
func (h *migrationHarness) exerciseRestoredKeyStorage() {
	h.t.Helper()
	keys := map[string]string{
		"master": "NOW() + INTERVAL '100 days'",
		"data":   "NOW() + INTERVAL '20 days'",
		"field":  "NOW() + INTERVAL '1 day'",
		"backup": "NOW() - INTERVAL '1 day'",
	}
	for keyType, expiry := range keys {
		h.exec(`INSERT INTO encryption_keys (key_id, key_type, expires_at) VALUES ($1, $2, `+expiry+`)`, "restored-"+keyType, keyType)
	}
	assert.Equal(h.t, int64(4), h.queryInt(`SELECT count(*) FROM system_events WHERE event_type = 'security.key.created' AND data ? 'key_id'`), "the insert branch of the audit trigger writes system_events.data")

	h.exec(`UPDATE encryption_keys SET status = 'revoked' WHERE key_type = 'backup'`)
	assert.Equal(h.t, int64(1), h.queryInt(`SELECT count(*) FROM system_events WHERE event_type = 'security.key.status_changed' AND severity = 'warning' AND data->>'new_status' = 'revoked'`), "the status branch of the audit trigger writes system_events.data")

	assert.Equal(h.t, int64(3), h.queryInt(`SELECT count(*) FROM v_active_encryption_keys`), "the view counts the active keys")
	assert.Equal(h.t, int64(3), h.queryInt(`SELECT count(*) FROM check_key_expiration()`))
	for keyType, want := range map[string]string{"master": "ok", "data": "warning", "field": "critical"} {
		var status string
		require.NoError(h.t, h.db.QueryRow(`SELECT c.status FROM check_key_expiration() c JOIN encryption_keys k ON k.id = c.key_id WHERE k.key_type = $1`, keyType).Scan(&status))
		assert.Equal(h.t, want, status, "check_key_expiration classifies the %s key", keyType)
	}
	h.exec(`INSERT INTO encryption_keys (key_id, key_type, expires_at, status) VALUES ('restored-expired', 'backup', NOW() - INTERVAL '2 days', 'expired')`)
	assert.Equal(h.t, int64(0), h.queryInt(`SELECT count(*) FROM check_key_expiration() WHERE status = 'expired'`), "only active keys are classified; an already expired key is not")

	h.exec(`INSERT INTO encryption_keys (key_id, key_type, expires_at) VALUES ('restored-backup-overdue', 'backup', NOW() - INTERVAL '1 day')`)
	assert.Equal(h.t, int64(4), h.queryInt(`SELECT count(*) FROM check_key_expiration()`), "an active key past its expiry is still classified")
	var overdue string
	require.NoError(h.t, h.db.QueryRow(`SELECT c.status FROM check_key_expiration() c JOIN encryption_keys k ON k.id = c.key_id WHERE k.key_id = 'restored-backup-overdue'`).Scan(&overdue))
	assert.Equal(h.t, "expired", overdue, "check_key_expiration reports an active key whose expires_at has passed as expired")

	var oldKey, newKey string
	require.NoError(h.t, h.db.QueryRow(`SELECT id FROM encryption_keys WHERE key_type = 'data'`).Scan(&oldKey))
	require.NoError(h.t, h.db.QueryRow(`SELECT id FROM encryption_keys WHERE key_type = 'master'`).Scan(&newKey))
	h.exec(`SELECT rotate_encryption_key($1::uuid, $2::uuid, 'endpoint', ARRAY[gen_random_uuid(), gen_random_uuid()])`, oldKey, newKey)
	assert.Equal(h.t, int64(2), h.queryInt(`SELECT count(*) FROM key_usage_log WHERE key_id = $1::uuid AND operation = 'rotate'`, oldKey), "one usage row per rotated entity, no cross product")
	assert.Equal(h.t, int64(1), h.queryInt(`SELECT count(*) FROM encryption_keys WHERE id = $1::uuid AND status = 'rotating'`, oldKey))

	assert.Equal(h.t, int64(0), h.queryInt(`SELECT count(*) FROM detect_anomalous_access()`), "the anomaly detector runs against the live basestations and messages schema")
}

func TestMigration154RemovesOrganizationQuotas(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(153)

	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd0000154")
	h.exec(`INSERT INTO organizations (org_id, tenant_id, name, can_have_base_stations, max_base_station_count)
	        VALUES (gen_random_uuid(), $1, 'quota-org', false, 3)`, tenantID)

	h.migrateTo(154)
	for _, column := range []string{"can_have_base_stations", "max_base_station_count", "max_endpoint_count"} {
		assert.False(t, h.columnExists("organizations", column), "organizations.%s must be dropped", column)
	}
	assert.False(t, h.columnExists("tenants", "max_basestations"))
	assert.False(t, h.columnExists("tenants", "max_endpoints"))
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM organizations WHERE name = 'quota-org'`), "organization rows survive")

	h.migrateTo(153)
	assert.True(t, h.columnExists("organizations", "can_have_base_stations"))
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM organizations WHERE name = 'quota-org' AND can_have_base_stations`),
		"rollback restores the default, not the discarded value")
	h.migrateTo(154)
}

func TestMigration161BackfillsReceptionTimeFromTheFirstMessage(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(158)

	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd0000161")
	const messageID, rxTime = "0f5a3c1e-6b1d-4c2e-9a61-000000000161", int64(1790000000123456789)
	h.exec(`INSERT INTO messages (id, tenant_id, owner_tenant_id, ep_eui, bs_eui, op_id, packet_cnt, rx_time,
	            rssi, snr, command_type, dl_open, response_exp, dl_ack, received_at)
	        VALUES ($1, $2, $2, decode('70b3d56770111561', 'hex'), decode('70b3d59cd0000161', 'hex'), 0, 7, $3,
	            -80.0, 10.0, 'ulData', false, false, false, NOW())`, messageID, tenantID, rxTime)
	h.exec(`INSERT INTO mioty_message_deduplication (owner_tenant_id, ep_eui, packet_cnt, message_hash, first_message_id, first_bs_eui, first_received_at, last_received_at)
	        VALUES ($1, decode('70b3d56770111561', 'hex'), 7, decode('abcd', 'hex'), $2, decode('70b3d59cd0000161', 'hex'), NOW() - INTERVAL '1 hour', NOW())`,
		tenantID, messageID)

	h.migrateTo(161)
	assert.Equal(t, rxTime, h.queryInt(`SELECT first_rx_time FROM mioty_message_deduplication WHERE first_message_id = $1`, messageID),
		"an existing row takes the reception time of its first message")

	h.migrateTo(158)
	assert.False(t, h.columnExists("mioty_message_deduplication", "first_rx_time"))
	h.migrateTo(161)
}

func TestMigration161StoresTheFullDetachCounter(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(158)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd0001161")
	h.exec(`INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name) VALUES ($1, $1, decode('70b3d56770111562', 'hex'), 'detach-counter')`, tenantID)
	const fullRangeCounter = int64(4294967295)
	_, err := h.db.Exec(`UPDATE endpoints SET last_detach_packet_cnt = $1 WHERE ep_eui = decode('70b3d56770111562', 'hex')`, fullRangeCounter)
	require.Error(t, err, "before 161 the detach counter column is INTEGER")

	h.migrateTo(161)
	h.exec(`UPDATE endpoints SET last_detach_packet_cnt = $1 WHERE ep_eui = decode('70b3d56770111562', 'hex')`, fullRangeCounter)
	assert.Equal(t, fullRangeCounter, h.queryInt(`SELECT last_detach_packet_cnt FROM endpoints WHERE ep_eui = decode('70b3d56770111562', 'hex')`))
	_, err = h.db.Exec(`UPDATE endpoints SET last_detach_packet_cnt = $1 WHERE ep_eui = decode('70b3d56770111562', 'hex')`, fullRangeCounter+1)
	require.Error(t, err, "the counter stays within 32 bits")

	h.migrateTo(158)
	assert.Equal(t, int64(-1), h.queryInt(`SELECT last_detach_packet_cnt FROM endpoints WHERE ep_eui = decode('70b3d56770111562', 'hex')`),
		"the down migration wraps the counter into the signed range")
	h.migrateTo(161)
	assert.Equal(t, fullRangeCounter, h.queryInt(`SELECT last_detach_packet_cnt FROM endpoints WHERE ep_eui = decode('70b3d56770111562', 'hex')`),
		"the up migration unwraps a counter the down migration wrapped")
}

func TestMigration155ArchivesLegacyDeduplicationRows(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(154)

	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd0000155")
	h.exec(`INSERT INTO mioty_message_deduplication (tenant_id, ep_eui, packet_cnt, message_hash, first_received_at, first_message_id, duplicate_count)
	        VALUES ($1, decode('70b3d56770111505', 'hex'), 42, decode('abcd', 'hex'), NOW() - INTERVAL '1 hour', 12345, 2)`, tenantID)

	h.migrateTo(155)
	assert.Equal(t, int64(1), h.archivedEvents("message.dedup.archived"))
	assert.Equal(t, int64(1), h.queryInt(
		`SELECT count(*) FROM system_events WHERE event_type = 'message.dedup.archived' AND tenant_id = $1
		   AND (data->>'legacyFirstMessageId')::bigint = 12345 AND (data->>'duplicateCount')::int = 2`, tenantID))
	assert.Equal(t, int64(0), h.queryInt(`SELECT count(*) FROM mioty_message_deduplication`), "legacy rows are archived, not carried over")
	assert.True(t, h.columnExists("mioty_message_deduplication", "owner_tenant_id"))
	assert.True(t, h.columnExists("mioty_message_deduplication", "first_bs_eui"))
	assert.False(t, h.columnExists("mioty_message_deduplication", "id"))
	assert.True(t, h.tableExists("message_delivery_outbox"))
	assert.Equal(t, int64(1), h.queryInt(`SELECT count(*) FROM pg_type WHERE typname = 'message_delivery_channel'`))

	h.migrateTo(154)
	assert.True(t, h.columnExists("mioty_message_deduplication", "first_basestation_id"))
	assert.False(t, h.tableExists("message_delivery_outbox"))
	assert.Equal(t, int64(0), h.queryInt(`SELECT count(*) FROM pg_type WHERE typname = 'message_delivery_channel'`))
	h.migrateTo(155)
}
