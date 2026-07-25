package postgres

import (
	"context"
	"fmt"
)

// legacyArchiveEUIColumn enumerates the only EUI columns the preserved
// pre-000139 legacy archive shares with the canonical archive. The closed
// switch keeps every SQL identifier fixed in this file - callers can never
// supply table or column names.
type legacyArchiveEUIColumn int

const (
	legacyArchiveBsEUI legacyArchiveEUIColumn = iota
	legacyArchiveEpEUI
)

// updateLegacyArchiveEUI applies an EUI rename to the preserved legacy
// archive so identity maintenance covers both archive tables. The table is
// optional: a database whose legacy archive was empty (and therefore dropped
// by migration 000139) is a no-op, resolved via to_regclass inside the same
// transaction as the canonical rename.
func updateLegacyArchiveEUI(ctx context.Context, q sqlExecQuerier, column legacyArchiveEUIColumn, newEUI, oldEUI []byte) error {
	var exists *string
	if err := q.QueryRowContext(ctx, `SELECT to_regclass('messages_archive_pre000139')::text`).Scan(&exists); err != nil {
		return fmt.Errorf("check legacy archive presence: %w", err)
	}
	if exists == nil {
		return nil
	}

	var stmt string
	switch column {
	case legacyArchiveBsEUI:
		stmt = `UPDATE messages_archive_pre000139 SET bs_eui = $1 WHERE bs_eui = $2`
	case legacyArchiveEpEUI:
		stmt = `UPDATE messages_archive_pre000139 SET ep_eui = $1 WHERE ep_eui = $2`
	default:
		return fmt.Errorf("unknown legacy archive column %d", column)
	}

	if _, err := q.ExecContext(ctx, stmt, newEUI, oldEUI); err != nil {
		return fmt.Errorf("update legacy archive: %w", err)
	}
	return nil
}
