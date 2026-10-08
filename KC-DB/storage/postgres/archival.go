package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// archivedMessageColumns enumerates the messages columns copied into
// messages_archive (identical layout per migration 000139). Naming the columns
// makes schema drift between the two tables surface as an explicit SQL error
// instead of silently misaligned positional inserts.
const archivedMessageColumns = `id, tenant_id, command_type, op_id, ep_eui, bs_eui, rx_time, packet_cnt,
		snr, rssi, user_data, dl_open, response_exp, dl_ack, rx_duration, eq_snr, profile, mode, format,
		subpackets, received_at, processed_at, created_at, updated_at, org_uuid, owner_tenant_id,
		base_stations, duplicate, decoded_payload, blueprint_type_eui, blueprint_version_id,
		decode_status, decode_error_code, nwk_sn_key, archived, archived_at, packet_cnt_reused, dl_window_claimed`

// ArchivalService handles message archiving operations
type ArchivalService struct {
	clock  clock.Clock
	db     *sql.DB
	logger logger.Logger
}

const (
	bytesPerUnit = 1024
	bytesFmt     = "%d bytes"
)

var byteUnits = []string{"kB", "MB", "GB", "TB", "PB"}

// partitionNameFmt names the monthly message partitions.
const partitionNameFmt = "messages_%d_%02d"

// NewArchivalService creates a new archival service
// pg_size_pretty style byte formatting: the 1024 unit step, the plain-bytes
// format, and the unit ladder.
func NewArchivalService(db *sql.DB, logger logger.Logger, clk clock.Clock) *ArchivalService {
	return &ArchivalService{
		clock:  clk,
		db:     db,
		logger: logger,
	}
}

// ArchiveOldMessages archives messages older than the specified duration
func (s *ArchivalService) ArchiveOldMessages(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoffTime := s.clock.Now().Add(-olderThan)

	s.logger.Info(logMsgStartingMessageArchival,
		logger.FieldCutoffTime, cutoffTime,
		logger.FieldOlderThan, olderThan)

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapBeginTransaction, err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.logger.Warn(logMsgArchivalRollback, logger.FieldError, err)
		}
	}()

	// First, copy messages to archive table
	query := fmt.Sprintf(`
		INSERT INTO messages_archive (%s)
		SELECT %s FROM messages
		WHERE received_at < $1
		  AND archived = false
		  AND NOT EXISTS (
		    SELECT 1 FROM messages_archive
		    WHERE messages_archive.id = messages.id
		  )`, archivedMessageColumns, archivedMessageColumns)

	result, err := tx.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapCopyArchive, err)
	}

	copiedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapGetCopiedCount, err)
	}

	// Mark messages as archived in main table
	updateQuery := `
		UPDATE messages 
		SET archived = true, 
		    archived_at = CURRENT_TIMESTAMP 
		WHERE received_at < $1 
		  AND archived = false`

	_, err = tx.ExecContext(ctx, updateQuery, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapMarkAsArchived, err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapCommitTransaction, err)
	}

	s.logger.Info(logMsgMessageArchivalCompleted,
		logger.FieldMessagesArchived, copiedCount)

	return copiedCount, nil
}

// CreateMonthlyPartitions ensures partitions exist for the next N months
func (s *ArchivalService) CreateMonthlyPartitions(ctx context.Context, monthsAhead int) error {
	s.logger.Info(logMsgCreatingMonthlyPartitions, logger.FieldMonthsAhead, monthsAhead)

	now := s.clock.Now()
	for i := 0; i <= monthsAhead; i++ {
		targetDate := now.AddDate(0, i, 0)
		year := targetDate.Year()
		month := int(targetDate.Month())

		partitionName := fmt.Sprintf(partitionNameFmt, year, month)

		// Check if partition already exists
		var exists bool
		checkQuery := `
			SELECT EXISTS (
				SELECT 1 FROM pg_tables 
				WHERE schemaname = 'public' 
				  AND tablename = $1
			)`

		err := s.db.QueryRowContext(ctx, checkQuery, partitionName).Scan(&exists)
		if err != nil {
			return fmt.Errorf(errFmtCheckPartition, partitionName, err)
		}

		if exists {
			s.logger.Debug(logMsgPartitionAlreadyExists, logger.FieldPartition, partitionName)
			continue
		}

		// Create partition
		startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(0, 1, 0)

		// #nosec G201 -- partitionName is constructed from controlled year/month integers, dates from time.Time
		createQuery := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s PARTITION OF messages
			FOR VALUES FROM ('%s') TO ('%s')`,
			pq.QuoteIdentifier(partitionName),
			startDate.Format(time.DateOnly),
			endDate.Format(time.DateOnly))

		_, err = s.db.ExecContext(ctx, createQuery)
		if err != nil {
			return fmt.Errorf(errFmtCreatePartition, partitionName, err)
		}

		s.logger.Info(logMsgCreatedPartition, logger.FieldPartition, partitionName)
	}

	return nil
}

// PurgeArchivedMessages removes messages from the main table that have been archived
// This should only be run after verifying the archive is complete and backed up
func (s *ArchivalService) PurgeArchivedMessages(ctx context.Context, archivedBefore time.Time) (int64, error) {
	s.logger.Warn(logMsgStartingArchivedMessagePurge,
		logger.FieldArchivedBefore, archivedBefore)

	// Only delete messages that have been archived for at least the specified duration
	query := `
		DELETE FROM messages 
		WHERE archived = true 
		  AND archived_at < $1`

	result, err := s.db.ExecContext(ctx, query, archivedBefore)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapDeleteArchivedMessages, err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapGetDeletedCount, err)
	}

	s.logger.Info(logMsgArchivedMessagePurgeCompleted,
		logger.FieldMessagesDeleted, count)

	return count, nil
}

// GetArchivalStats returns statistics about archived messages
func (s *ArchivalService) GetArchivalStats(ctx context.Context) (*ArchivalStats, error) {
	stats := &ArchivalStats{}

	if err := s.queryMainTableStats(ctx, stats); err != nil {
		return nil, err
	}

	canonicalBytes, oldestArchive, newestArchive, err := s.queryCanonicalArchiveStats(ctx, stats)
	if err != nil {
		return nil, err
	}

	legacyBytes, oldestLegacy, newestLegacy, err := s.queryLegacyArchiveStats(ctx, stats)
	if err != nil {
		return nil, err
	}

	// Combined totals across both archive tables
	stats.ArchiveTableCount = stats.CanonicalArchiveCount + stats.LegacyArchiveCount
	stats.CanonicalArchiveSize = prettyBytes(canonicalBytes)
	stats.LegacyArchiveSize = prettyBytes(legacyBytes)
	stats.ArchiveTableSize = prettyBytes(canonicalBytes + legacyBytes)
	stats.OldestArchiveMessage = earliestValid(oldestArchive, oldestLegacy)
	stats.NewestArchiveMessage = latestValid(newestArchive, newestLegacy)

	partitions, err := s.queryPartitionInfo(ctx)
	if err != nil {
		return nil, err
	}
	stats.Partitions = partitions

	return stats, nil
}

// queryMainTableStats fills the main messages-table counters and timestamps.
func (s *ArchivalService) queryMainTableStats(ctx context.Context, stats *ArchivalStats) error {
	mainQuery := `
		SELECT
			COUNT(*) as total_messages,
			COUNT(*) FILTER (WHERE archived = true) as archived_messages,
			MIN(received_at) as oldest_message,
			MAX(received_at) as newest_message,
			pg_size_pretty(pg_total_relation_size('messages')) as table_size
		FROM messages`

	var oldestMain, newestMain sql.NullTime
	err := s.db.QueryRowContext(ctx, mainQuery).Scan(
		&stats.MainTableCount,
		&stats.MainTableArchived,
		&oldestMain,
		&newestMain,
		&stats.MainTableSize,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapGetMainTableStats, err)
	}

	if oldestMain.Valid {
		stats.OldestMainMessage = &oldestMain.Time
	}
	if newestMain.Valid {
		stats.NewestMainMessage = &newestMain.Time
	}
	return nil
}

// queryCanonicalArchiveStats reads the canonical archive table. Raw byte sizes
// are returned so combined totals can be summed numerically; pg_size_pretty
// strings cannot be added.
func (s *ArchivalService) queryCanonicalArchiveStats(ctx context.Context, stats *ArchivalStats) (tableBytes int64, oldest, newest sql.NullTime, err error) {
	archiveQuery := `
		SELECT
			COUNT(*) as total_messages,
			MIN(received_at) as oldest_message,
			MAX(received_at) as newest_message,
			pg_total_relation_size('messages_archive') as table_bytes
		FROM messages_archive`

	err = s.db.QueryRowContext(ctx, archiveQuery).Scan(
		&stats.CanonicalArchiveCount,
		&oldest,
		&newest,
		&tableBytes,
	)
	if err != nil && err != sql.ErrNoRows {
		return 0, sql.NullTime{}, sql.NullTime{}, fmt.Errorf("%s: %w", errWrapGetArchiveTableStats, err)
	}
	return tableBytes, oldest, newest, nil
}

// queryLegacyArchiveStats reads the pre-000139 archive table, which preserves
// rows the canonical rebuild could not losslessly project. It is optional
// (to_regclass) and combined statistics cover both tables.
func (s *ArchivalService) queryLegacyArchiveStats(ctx context.Context, stats *ArchivalStats) (tableBytes int64, oldest, newest sql.NullTime, err error) {
	var legacyExists *string
	if err := s.db.QueryRowContext(ctx, `SELECT to_regclass('messages_archive_pre000139')::text`).Scan(&legacyExists); err != nil {
		return 0, sql.NullTime{}, sql.NullTime{}, fmt.Errorf("%s: %w", errWrapCheckLegacyArchivePresence, err)
	}
	if legacyExists == nil {
		return 0, sql.NullTime{}, sql.NullTime{}, nil
	}

	legacyQuery := `
		SELECT
			COUNT(*) as total_messages,
			MIN(received_at) as oldest_message,
			MAX(received_at) as newest_message,
			pg_total_relation_size('messages_archive_pre000139') as table_bytes
		FROM messages_archive_pre000139`
	err = s.db.QueryRowContext(ctx, legacyQuery).Scan(
		&stats.LegacyArchiveCount,
		&oldest,
		&newest,
		&tableBytes,
	)
	if err != nil && err != sql.ErrNoRows {
		return 0, sql.NullTime{}, sql.NullTime{}, fmt.Errorf("%s: %w", errWrapGetLegacyArchiveStats, err)
	}
	return tableBytes, oldest, newest, nil
}

// queryPartitionInfo lists the monthly message partitions with pretty sizes.
func (s *ArchivalService) queryPartitionInfo(ctx context.Context) ([]PartitionInfo, error) {
	partitionQuery := `
		SELECT
			tablename,
			pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
		FROM pg_tables
		WHERE tablename LIKE 'messages_%'
		  AND tablename != 'messages_archive'
		  AND tablename != 'messages_archive_pre000139'
		ORDER BY tablename`

	rows, err := s.db.QueryContext(ctx, partitionQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetPartitionInfo, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn(logMsgRowsClose, logger.FieldError, err)
		}
	}()

	partitions := make([]PartitionInfo, 0)
	for rows.Next() {
		var info PartitionInfo
		if err := rows.Scan(&info.Name, &info.Size); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanPartitionInfo, err)
		}
		partitions = append(partitions, info)
	}
	return partitions, nil
}

// earliestValid returns the earliest valid candidate timestamp, or nil.
func earliestValid(candidates ...sql.NullTime) *time.Time {
	var earliest *time.Time
	for _, cand := range candidates {
		if cand.Valid && (earliest == nil || cand.Time.Before(*earliest)) {
			t := cand.Time
			earliest = &t
		}
	}
	return earliest
}

// latestValid returns the latest valid candidate timestamp, or nil.
func latestValid(candidates ...sql.NullTime) *time.Time {
	var latest *time.Time
	for _, cand := range candidates {
		if cand.Valid && (latest == nil || cand.Time.After(*latest)) {
			t := cand.Time
			latest = &t
		}
	}
	return latest
}

// ArchivalStats contains statistics about archived messages. The combined
// ArchiveTableCount/ArchiveTableSize cover the canonical archive plus the
// preserved pre-000139 legacy archive; the split fields expose each side.
type ArchivalStats struct {
	MainTableCount    int64
	MainTableArchived int64
	MainTableSize     string
	OldestMainMessage *time.Time
	NewestMainMessage *time.Time

	ArchiveTableCount     int64
	ArchiveTableSize      string
	CanonicalArchiveCount int64
	CanonicalArchiveSize  string
	LegacyArchiveCount    int64
	LegacyArchiveSize     string
	OldestArchiveMessage  *time.Time
	NewestArchiveMessage  *time.Time

	Partitions []PartitionInfo
}

// prettyBytes renders a byte count in the pg_size_pretty style used by the
// other size fields, computed client-side so combined totals stay numeric.
func prettyBytes(n int64) string {
	if n < bytesPerUnit {
		return fmt.Sprintf(bytesFmt, n)
	}
	value := float64(n)
	idx := -1
	for value >= bytesPerUnit && idx < len(byteUnits)-1 {
		value /= bytesPerUnit
		idx++
	}
	return fmt.Sprintf("%.0f %s", value, byteUnits[idx])
}

// PartitionInfo contains information about a message partition
type PartitionInfo struct {
	Name string
	Size string
}
