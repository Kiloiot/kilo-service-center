package postgres

import (
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// archivalMessageParams describes a minimal messages row for archival tests.
type archivalMessageParams struct {
	TenantID   int64
	EpEUI      uint64
	BsEUI      uint64
	ReceivedAt time.Time
}

// insertArchivalMessage inserts a messages row satisfying all NOT NULL columns
// of the current schema (migration 000047 layout with BYTEA EUIs per 000135).
// Returns the generated message UUID.
func insertArchivalMessage(t *testing.T, db *sqlx.DB, p archivalMessageParams) string {
	t.Helper()

	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, p.EpEUI)
	bsEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEUIBytes, p.BsEUI)

	var id string
	err := db.QueryRow(`
		INSERT INTO messages (
			tenant_id, op_id, ep_eui, bs_eui, rx_time, packet_cnt,
			snr, rssi, user_data, received_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, p.TenantID, p.ReceivedAt.UnixNano(), epEUIBytes, bsEUIBytes,
		p.ReceivedAt.UnixNano(), 1, 10.0, -80.0, []byte{0x42}, p.ReceivedAt).Scan(&id)
	require.NoError(t, err, "Failed to insert test message")

	return id
}

// TestArchivalServiceMessages verifies the message archival lifecycle against
// the current messages/messages_archive schema: copy-then-mark on the first
// sweep, idempotency on re-run, BYTEA EUI round-trip for high-bit EUI64
// values, archival statistics, and the purge of already-archived rows.
func TestArchivalServiceMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	createTestTenant(t, sqlxDB, 100, "TestTenantArchival")

	logger.Initialize("error", "json")
	log := logger.Get()
	service := NewArchivalService(sqlxDB.DB, log)

	ctx := context.Background()

	// High-bit EUI64 (> max int64) proves the BYTEA storage path survives the
	// archival round-trip without signed overflow.
	const highBitEpEUI = uint64(0xCAFECAFECAFECAFE)
	const bsEUI = uint64(0x70B3D59CD00009E6)
	const retention = 90 * 24 * time.Hour

	oldTime := time.Now().Add(-100 * 24 * time.Hour)
	for i := 0; i < 5; i++ {
		insertArchivalMessage(t, sqlxDB, archivalMessageParams{
			TenantID:   100,
			EpEUI:      highBitEpEUI,
			BsEUI:      bsEUI,
			ReceivedAt: oldTime.Add(time.Duration(i) * time.Hour),
		})
	}

	recentTime := time.Now().Add(-1 * time.Hour)
	for i := 0; i < 3; i++ {
		insertArchivalMessage(t, sqlxDB, archivalMessageParams{
			TenantID:   100,
			EpEUI:      highBitEpEUI,
			BsEUI:      bsEUI,
			ReceivedAt: recentTime.Add(time.Duration(i) * time.Minute),
		})
	}

	t.Run("ArchiveOldMessages", func(t *testing.T) {
		count, err := service.ArchiveOldMessages(ctx, retention)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count, "only messages older than retention must be archived")

		var archivedCount int
		require.NoError(t, sqlxDB.QueryRow("SELECT COUNT(*) FROM messages_archive").Scan(&archivedCount))
		assert.Equal(t, 5, archivedCount)

		var markedCount int
		require.NoError(t, sqlxDB.QueryRow(
			"SELECT COUNT(*) FROM messages WHERE archived = true").Scan(&markedCount))
		assert.Equal(t, 5, markedCount, "archived rows must be flagged in the main table")

		var unmarkedCount int
		require.NoError(t, sqlxDB.QueryRow(
			"SELECT COUNT(*) FROM messages WHERE archived = false").Scan(&unmarkedCount))
		assert.Equal(t, 3, unmarkedCount, "recent rows must remain unflagged")

		var stampedCount int
		require.NoError(t, sqlxDB.QueryRow(
			"SELECT COUNT(*) FROM messages_archive WHERE archived_at IS NOT NULL").Scan(&stampedCount))
		assert.Equal(t, 5, stampedCount, "archive trigger must stamp archived_at on every copied row")
	})

	t.Run("HighBitEUIRoundTrip", func(t *testing.T) {
		rows, err := sqlxDB.Query("SELECT ep_eui, bs_eui FROM messages_archive")
		require.NoError(t, err)
		defer func() { _ = rows.Close() }()

		count := 0
		for rows.Next() {
			var epEUIBytes, bsEUIBytes []byte
			require.NoError(t, rows.Scan(&epEUIBytes, &bsEUIBytes))
			require.Len(t, epEUIBytes, 8)
			require.Len(t, bsEUIBytes, 8)
			assert.Equal(t, highBitEpEUI, binary.BigEndian.Uint64(epEUIBytes),
				"high-bit EP EUI must survive the archival round-trip unchanged")
			assert.Equal(t, bsEUI, binary.BigEndian.Uint64(bsEUIBytes))
			count++
		}
		require.NoError(t, rows.Err())
		assert.Equal(t, 5, count)
	})

	t.Run("SecondRunIsIdempotent", func(t *testing.T) {
		count, err := service.ArchiveOldMessages(ctx, retention)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count, "re-running archival must not copy rows twice")

		var archivedCount int
		require.NoError(t, sqlxDB.QueryRow("SELECT COUNT(*) FROM messages_archive").Scan(&archivedCount))
		assert.Equal(t, 5, archivedCount)
	})

	t.Run("ArchivalStats", func(t *testing.T) {
		stats, err := service.GetArchivalStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(8), stats.MainTableCount)
		assert.Equal(t, int64(5), stats.MainTableArchived)
		assert.Equal(t, int64(5), stats.ArchiveTableCount)
	})

	t.Run("PurgeArchivedMessages", func(t *testing.T) {
		purged, err := service.PurgeArchivedMessages(ctx, time.Now().Add(time.Minute))
		require.NoError(t, err)
		assert.Equal(t, int64(5), purged, "purge must remove exactly the archived rows")

		var remaining int
		require.NoError(t, sqlxDB.QueryRow("SELECT COUNT(*) FROM messages").Scan(&remaining))
		assert.Equal(t, 3, remaining, "recent unarchived rows must survive the purge")

		var archiveCount int
		require.NoError(t, sqlxDB.QueryRow("SELECT COUNT(*) FROM messages_archive").Scan(&archiveCount))
		assert.Equal(t, 5, archiveCount, "purge must not touch the archive table")
	})
}
