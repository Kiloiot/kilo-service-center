package postgres

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMigrator builds a migrate instance over the container database.
func newMigrator(t *testing.T, db *sqlx.DB) *migrate.Migrate {
	t.Helper()
	migrationsDir, err := filepath.Abs("../../migrations")
	require.NoError(t, err)

	driver, err := migratepostgres.WithInstance(db.DB, &migratepostgres.Config{})
	require.NoError(t, err)

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", filepath.ToSlash(migrationsDir)),
		"postgres", driver)
	require.NoError(t, err)
	return m
}

// seedLegacyArchiveRow inserts a row into the pre-000139 legacy archive
// layout (001-era messages columns).
func seedLegacyArchiveRow(t *testing.T, db *sqlx.DB, id int64, epEUI, bsEUI []byte, frameCount int) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO messages_archive (
			id, ep_eui, bs_eui, tenant_id, payload, frame_count,
			rssi, snr, eq_snr, frequency, received_at
		) VALUES ($1, $2, $3, 1, $4, $5, -80, 10, 9.5, 868300000, NOW())`,
		id, epEUI, bsEUI, []byte{0x01, 0x02}, frameCount)
	require.NoError(t, err, "seed legacy archive row")
}

// TestMigration000139LegacyPreservation drives up/down/up with a POPULATED
// legacy archive: rows (including high-bit EUIs) survive field-for-field
// under messages_archive_pre000139, EUI maintenance and archival statistics
// cover both tables, down restores the original table, and a second up
// preserves it again without duplication or loss.
func TestMigration000139LegacyPreservation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping migration test in short mode")
	}
	db, _, cleanup := SetupPostgresContainerWithoutMigrations(t)
	defer cleanup()
	m := newMigrator(t, db)
	ctx := testutil.TestContext()

	// Schema state immediately before the archive rebuild
	require.NoError(t, m.Migrate(138), "migrate to 000138")

	// High-bit EUIs prove BYTEA rows survive without numeric projection
	highEp := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFE}
	highBs := []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}
	lowEp := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02}
	lowBs := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03}
	seedLegacyArchiveRow(t, db, 1, highEp, highBs, 42)
	seedLegacyArchiveRow(t, db, 2, lowEp, lowBs, 7)

	// Up: rebuild preserves the populated legacy table
	require.NoError(t, m.Migrate(139), "migrate to 000139")

	var legacyCount int
	require.NoError(t, db.Get(&legacyCount, `SELECT COUNT(*) FROM messages_archive_pre000139`))
	assert.Equal(t, 2, legacyCount, "every legacy row must be preserved")

	var frameCount int
	require.NoError(t, db.Get(&frameCount,
		`SELECT frame_count FROM messages_archive_pre000139 WHERE ep_eui = $1`, highEp))
	assert.Equal(t, 42, frameCount, "high-bit EUI row fields must survive intact")

	var canonicalCount int
	require.NoError(t, db.Get(&canonicalCount, `SELECT COUNT(*) FROM messages_archive`))
	assert.Zero(t, canonicalCount, "the corrected canonical archive starts empty")

	// EUI maintenance covers the legacy table via the shared helper
	renamedBs := []byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x99}
	require.NoError(t, updateLegacyArchiveEUI(ctx, db, legacyArchiveBsEUI, renamedBs, highBs))
	var renamedCount int
	require.NoError(t, db.Get(&renamedCount,
		`SELECT COUNT(*) FROM messages_archive_pre000139 WHERE bs_eui = $1`, renamedBs))
	assert.Equal(t, 1, renamedCount, "legacy archive must follow BS EUI renames")

	// Archival statistics cover both tables: combined plus split counts
	svc := NewArchivalService(db.DB, logger.NewNop())
	stats, err := svc.GetArchivalStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.LegacyArchiveCount)
	assert.Equal(t, int64(0), stats.CanonicalArchiveCount)
	assert.Equal(t, int64(2), stats.ArchiveTableCount, "combined count covers both tables")
	assert.NotEmpty(t, stats.LegacyArchiveSize)
	assert.NotNil(t, stats.OldestArchiveMessage, "oldest spans both tables")
	for _, p := range stats.Partitions {
		assert.NotEqual(t, "messages_archive", p.Name)
		assert.NotEqual(t, "messages_archive_pre000139", p.Name,
			"archive tables are not message partitions")
	}

	// Down (canonical archive empty): the preserved table is restored
	require.NoError(t, m.Migrate(138), "migrate down to 000138")
	require.NoError(t, db.Get(&legacyCount, `SELECT COUNT(*) FROM messages_archive`))
	assert.Equal(t, 2, legacyCount, "down restores the preserved legacy archive")
	var preExists *string
	require.NoError(t, db.Get(&preExists, `SELECT to_regclass('messages_archive_pre000139')::text`))
	assert.Nil(t, preExists, "the preserved name is released on down")

	// Up again: preserved once more, no duplication or loss
	require.NoError(t, m.Migrate(139), "migrate up to 000139 again")
	require.NoError(t, db.Get(&legacyCount, `SELECT COUNT(*) FROM messages_archive_pre000139`))
	assert.Equal(t, 2, legacyCount, "second up preserves the same rows exactly once")
}

// TestMigration000139DownRefusesWithCanonicalRows verifies the down migration
// blocks with its named diagnostic when the canonical archive holds rows,
// instead of silently destroying archived messages.
func TestMigration000139DownRefusesWithCanonicalRows(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping migration test in short mode")
	}
	db, _, cleanup := SetupPostgresContainerWithoutMigrations(t)
	defer cleanup()
	m := newMigrator(t, db)

	require.NoError(t, m.Migrate(139), "migrate to 000139")

	_, err := db.Exec(`
		INSERT INTO messages_archive (
			tenant_id, op_id, ep_eui, bs_eui, rx_time, packet_cnt, snr, rssi
		) VALUES (1, 1, $1, $2, 1, 1, 10, -80)`,
		[]byte{0, 0, 0, 0, 0, 0, 0, 1}, []byte{0, 0, 0, 0, 0, 0, 0, 2})
	require.NoError(t, err, "seed canonical archive row")

	err = m.Migrate(138)
	require.Error(t, err, "down with canonical archive rows must refuse")
	assert.Contains(t, err.Error(), "KC-MIG-000139-DOWN",
		"the refusal must carry its named diagnostic")
}

// TestMigration000140PurgeRules verifies both purge rules: session-ownership
// deletion for terminated/non-resumable sessions, and the fixed-cutoff purge
// of pre-completion-fix status polls - while every other operation of a
// disconnected-but-resumable session survives.
func TestMigration000140PurgeRules(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping migration test in short mode")
	}
	db, _, cleanup := SetupPostgresContainerWithoutMigrations(t)
	defer cleanup()
	m := newMigrator(t, db)

	require.NoError(t, m.Migrate(139), "migrate to 000139")

	// Seed tenant → base station → sessions
	var tenantID int64
	require.NoError(t, db.QueryRow(`
		INSERT INTO tenants (name, status, created_at, updated_at)
		VALUES ('mig140', 'active', NOW(), NOW()) RETURNING id`).Scan(&tenantID))

	var bsID int64
	require.NoError(t, db.QueryRow(`
		INSERT INTO basestations (tenant_id, bs_eui, name, connection_type, service_center_url)
		VALUES ($1, $2, 'mig140-bs', 'bssci', 'bssci://test') RETURNING id`,
		tenantID, []byte{0, 0, 0, 0, 0, 0, 0, 9}).Scan(&bsID))

	bsUUID := make([]byte, 16)
	scUUID := make([]byte, 16)
	insertSession := func(status string, canResume bool, mark byte) int64 {
		bsUUID[15] = mark
		scUUID[15] = mark
		var id int64
		require.NoError(t, db.QueryRow(`
			INSERT INTO basestation_sessions (
				basestation_id, tenant_id, sn_bs_uuid, sn_sc_uuid, sn_bs_op_id, sn_sc_op_id,
				status, can_resume, encoding, started_at
			) VALUES ($1, $2, $3, $4, 0, 0, $5, $6, 'msgpack', NOW()) RETURNING id`,
			bsID, tenantID, bsUUID, scUUID, status, canResume).Scan(&id))
		return id
	}
	resumable := insertSession("disconnected", true, 1)
	terminated := insertSession("terminated", false, 2)

	cutoff := time.Date(2026, 7, 21, 18, 36, 9, 0, time.UTC)
	insertOp := func(sessionID, opID int64, opType string, createdAt time.Time) {
		_, err := db.Exec(`
			INSERT INTO bssci_pending_operations (
				basestation_session_id, operation_id, operation_type, operation_data, created_at
			) VALUES ($1, $2, $3, '{}', $4)`,
			sessionID, opID, opType, createdAt)
		require.NoError(t, err, "seed pending op")
	}
	insertOp(resumable, -1, "status", cutoff.Add(-time.Hour))    // pre-fix leak: purged
	insertOp(resumable, -2, "status", cutoff.Add(time.Hour))     // post-fix poll: survives
	insertOp(resumable, -3, "dlDataQue", cutoff.Add(-time.Hour)) // non-status: survives
	insertOp(terminated, -4, "dlDataQue", cutoff.Add(time.Hour)) // terminated session: purged

	require.NoError(t, m.Migrate(140), "migrate to 000140")

	surviving := map[int64]bool{}
	rows, err := db.Query(`SELECT operation_id FROM bssci_pending_operations`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int64
		require.NoError(t, rows.Scan(&id))
		surviving[id] = true
	}
	require.NoError(t, rows.Err())

	assert.False(t, surviving[-1], "pre-fix leaked status poll must be purged")
	assert.True(t, surviving[-2], "post-fix status poll of a resumable session must survive")
	assert.True(t, surviving[-3], "non-status operation of a resumable session must survive")
	assert.False(t, surviving[-4], "operations of a terminated session must be purged")
	assert.Len(t, surviving, 2)
}
