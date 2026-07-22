package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// archivedMessageColumns enumerates the messages columns copied into
// messages_archive (identical layout per migration 000139). Naming the columns
// makes schema drift between the two tables surface as an explicit SQL error
// instead of silently misaligned positional inserts.
const archivedMessageColumns = `id, tenant_id, command_type, op_id, ep_eui, bs_eui, rx_time, packet_cnt,
		snr, rssi, user_data, dl_open, response_exp, dl_ack, rx_duration, eq_snr, profile, mode, format,
		subpackets, received_at, processed_at, created_at, updated_at, org_uuid, owner_tenant_id,
		base_stations, duplicate, decoded_payload, blueprint_type_eui, blueprint_version_id,
		decode_status, decode_error_code, nwk_sn_key, archived, archived_at`

// ArchivalService handles message archiving operations
type ArchivalService struct {
	db     *sql.DB
	logger logger.Logger
}

// NewArchivalService creates a new archival service
func NewArchivalService(db *sql.DB, logger logger.Logger) *ArchivalService {
	return &ArchivalService{
		db:     db,
		logger: logger,
	}
}

// ArchiveOldMessages archives messages older than the specified duration
func (s *ArchivalService) ArchiveOldMessages(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)

	s.logger.Info("Starting message archival",
		"cutoff_time", cutoffTime,
		"older_than", olderThan)

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.logger.Warn("archival rollback failed", "error", err)
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
		return 0, fmt.Errorf("copy to archive: %w", err)
	}

	copiedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get copied count: %w", err)
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
		return 0, fmt.Errorf("mark as archived: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	s.logger.Info("Message archival completed",
		"messages_archived", copiedCount)

	return copiedCount, nil
}

// ArchiveOldGatewayReceptions archives gateway receptions older than the specified duration
func (s *ArchivalService) ArchiveOldGatewayReceptions(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)

	s.logger.Info("Starting gateway reception archival",
		"cutoff_time", cutoffTime,
		"older_than", olderThan)

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.logger.Warn("archival rollback failed (partitions)", "error", err)
		}
	}()

	// First, copy gateway receptions to archive table
	query := `
		INSERT INTO gateway_receptions_archive 
		SELECT * FROM gateway_receptions 
		WHERE created_at < $1 
		  AND NOT EXISTS (
		    SELECT 1 FROM gateway_receptions_archive 
		    WHERE gateway_receptions_archive.id = gateway_receptions.id
		  )`

	result, err := tx.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("copy to archive: %w", err)
	}

	copiedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get copied count: %w", err)
	}

	// Delete from main table (gateway_receptions don't have an archived flag)
	deleteQuery := `
		DELETE FROM gateway_receptions 
		WHERE created_at < $1 
		  AND EXISTS (
		    SELECT 1 FROM gateway_receptions_archive 
		    WHERE gateway_receptions_archive.id = gateway_receptions.id
		  )`

	_, err = tx.ExecContext(ctx, deleteQuery, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("delete from main table: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	s.logger.Info("Gateway reception archival completed",
		"receptions_archived", copiedCount)

	return copiedCount, nil
}

// ArchiveOldDeviceSessions archives device sessions older than the specified duration
func (s *ArchivalService) ArchiveOldDeviceSessions(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)

	s.logger.Info("Starting device session archival",
		"cutoff_time", cutoffTime,
		"older_than", olderThan)

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.logger.Warn("archival rollback failed (retention)", "error", err)
		}
	}()

	// First, copy terminated/expired sessions to archive table
	query := `
		INSERT INTO device_sessions_archive 
		SELECT * FROM device_sessions 
		WHERE (status IN ('terminated', 'expired') AND ended_at < $1)
		  AND NOT EXISTS (
		    SELECT 1 FROM device_sessions_archive 
		    WHERE device_sessions_archive.id = device_sessions.id
		  )`

	result, err := tx.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("copy to archive: %w", err)
	}

	copiedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get copied count: %w", err)
	}

	// Delete from main table
	deleteQuery := `
		DELETE FROM device_sessions 
		WHERE (status IN ('terminated', 'expired') AND ended_at < $1)
		  AND EXISTS (
		    SELECT 1 FROM device_sessions_archive 
		    WHERE device_sessions_archive.id = device_sessions.id
		  )`

	_, err = tx.ExecContext(ctx, deleteQuery, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("delete from main table: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	s.logger.Info("Device session archival completed",
		"sessions_archived", copiedCount)

	return copiedCount, nil
}

// ArchiveOldDeviceKeys archives device keys older than the specified duration
func (s *ArchivalService) ArchiveOldDeviceKeys(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)

	s.logger.Info("Starting device key archival",
		"cutoff_time", cutoffTime,
		"older_than", olderThan)

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.logger.Warn("retention rollback failed", "error", err)
		}
	}()

	// First, copy inactive/expired keys to archive table
	query := `
		INSERT INTO device_keys_archive 
		SELECT * FROM device_keys 
		WHERE (is_active = false OR (valid_until IS NOT NULL AND valid_until < $1))
		  AND updated_at < $1
		  AND NOT EXISTS (
		    SELECT 1 FROM device_keys_archive 
		    WHERE device_keys_archive.id = device_keys.id
		  )`

	result, err := tx.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("copy to archive: %w", err)
	}

	copiedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get copied count: %w", err)
	}

	// Delete from main table (keep at least one inactive key per device for history)
	deleteQuery := `
		DELETE FROM device_keys dk1
		WHERE (is_active = false OR (valid_until IS NOT NULL AND valid_until < $1))
		  AND updated_at < $1
		  AND EXISTS (
		    SELECT 1 FROM device_keys_archive 
		    WHERE device_keys_archive.id = dk1.id
		  )
		  AND EXISTS (
		    SELECT 1 FROM device_keys dk2
		    WHERE dk2.device_id = dk1.device_id
		      AND dk2.key_type = dk1.key_type
		      AND dk2.id != dk1.id
		      AND (dk2.key_version > dk1.key_version OR dk2.is_active = true)
		  )`

	_, err = tx.ExecContext(ctx, deleteQuery, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("delete from main table: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	s.logger.Info("Device key archival completed",
		"keys_archived", copiedCount)

	return copiedCount, nil
}

// CreateMonthlyPartitions ensures partitions exist for the next N months
func (s *ArchivalService) CreateMonthlyPartitions(ctx context.Context, monthsAhead int) error {
	s.logger.Info("Creating monthly partitions", "months_ahead", monthsAhead)

	now := time.Now()
	for i := 0; i <= monthsAhead; i++ {
		targetDate := now.AddDate(0, i, 0)
		year := targetDate.Year()
		month := int(targetDate.Month())

		partitionName := fmt.Sprintf("messages_%d_%02d", year, month)

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
			return fmt.Errorf("check partition %s: %w", partitionName, err)
		}

		if exists {
			s.logger.Debug("Partition already exists", "partition", partitionName)
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
			startDate.Format("2006-01-02"),
			endDate.Format("2006-01-02"))

		_, err = s.db.ExecContext(ctx, createQuery)
		if err != nil {
			return fmt.Errorf("create partition %s: %w", partitionName, err)
		}

		s.logger.Info("Created partition", "partition", partitionName)
	}

	return nil
}

// PurgeArchivedMessages removes messages from the main table that have been archived
// This should only be run after verifying the archive is complete and backed up
func (s *ArchivalService) PurgeArchivedMessages(ctx context.Context, archivedBefore time.Time) (int64, error) {
	s.logger.Warn("Starting archived message purge",
		"archived_before", archivedBefore)

	// Only delete messages that have been archived for at least the specified duration
	query := `
		DELETE FROM messages 
		WHERE archived = true 
		  AND archived_at < $1`

	result, err := s.db.ExecContext(ctx, query, archivedBefore)
	if err != nil {
		return 0, fmt.Errorf("delete archived messages: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get deleted count: %w", err)
	}

	s.logger.Info("Archived message purge completed",
		"messages_deleted", count)

	return count, nil
}

// GetArchivalStats returns statistics about archived messages
func (s *ArchivalService) GetArchivalStats(ctx context.Context) (*ArchivalStats, error) {
	stats := &ArchivalStats{}

	// Get main table stats
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
		return nil, fmt.Errorf("get main table stats: %w", err)
	}

	if oldestMain.Valid {
		stats.OldestMainMessage = &oldestMain.Time
	}
	if newestMain.Valid {
		stats.NewestMainMessage = &newestMain.Time
	}

	// Get archive table stats
	archiveQuery := `
		SELECT 
			COUNT(*) as total_messages,
			MIN(received_at) as oldest_message,
			MAX(received_at) as newest_message,
			pg_size_pretty(pg_total_relation_size('messages_archive')) as table_size
		FROM messages_archive`

	var oldestArchive, newestArchive sql.NullTime
	err = s.db.QueryRowContext(ctx, archiveQuery).Scan(
		&stats.ArchiveTableCount,
		&oldestArchive,
		&newestArchive,
		&stats.ArchiveTableSize,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("get archive table stats: %w", err)
	}

	if oldestArchive.Valid {
		stats.OldestArchiveMessage = &oldestArchive.Time
	}
	if newestArchive.Valid {
		stats.NewestArchiveMessage = &newestArchive.Time
	}

	// Get partition info
	partitionQuery := `
		SELECT 
			tablename,
			pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
		FROM pg_tables 
		WHERE tablename LIKE 'messages_%' 
		  AND tablename != 'messages_archive'
		ORDER BY tablename`

	rows, err := s.db.QueryContext(ctx, partitionQuery)
	if err != nil {
		return nil, fmt.Errorf("get partition info: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn("rows close failed", "error", err)
		}
	}()

	stats.Partitions = make([]PartitionInfo, 0)
	for rows.Next() {
		var info PartitionInfo
		if err := rows.Scan(&info.Name, &info.Size); err != nil {
			return nil, fmt.Errorf("scan partition info: %w", err)
		}
		stats.Partitions = append(stats.Partitions, info)
	}

	return stats, nil
}

// ArchivalStats contains statistics about archived messages
type ArchivalStats struct {
	MainTableCount    int64
	MainTableArchived int64
	MainTableSize     string
	OldestMainMessage *time.Time
	NewestMainMessage *time.Time

	ArchiveTableCount    int64
	ArchiveTableSize     string
	OldestArchiveMessage *time.Time
	NewestArchiveMessage *time.Time

	Partitions []PartitionInfo
}

// PartitionInfo contains information about a message partition
type PartitionInfo struct {
	Name string
	Size string
}
