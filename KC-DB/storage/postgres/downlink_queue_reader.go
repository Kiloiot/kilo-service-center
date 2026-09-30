package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// DownlinkQueueReader implements read-only downlink queue operations
// Reuses storage.DownlinkMessage to avoid struct duplication
type DownlinkQueueReader struct {
	log logger.Logger
	db  *sqlx.DB
}

// Ensure DownlinkQueueReader implements the interfaces
var (
	_ interfaces.DownlinkQueueReader = (*DownlinkQueueReader)(nil)
	_ interfaces.DownlinkQueueStore  = (*DownlinkQueueReader)(nil)
)

// NewDownlinkQueueReader creates a new downlink queue reader
func NewDownlinkQueueReader(db *sqlx.DB, log logger.Logger) *DownlinkQueueReader {
	return &DownlinkQueueReader{
		log: log, db: db}
}

// downlinkQueueColumnsListed follow downlinkListingColumns in the queue
// listing: what a base station reported so far and the downlink window it
// waits for.
const downlinkQueueColumnsListed = `result, tx_time, earliest_at`

// ListTenantQueue lists one page of a tenant's queued downlinks, highest
// priority first.
func (r *DownlinkQueueReader) ListTenantQueue(ctx context.Context, tenantID int64, filter storage.DownlinkQueueFilter, limit, offset int) ([]*storage.DownlinkMessage, error) {
	scope := downlinkQueueScope(tenantID, filter)
	query := "SELECT " + downlinkListingColumns + ", " + downlinkQueueColumnsListed +
		" FROM downlink_queue WHERE " + scope.where() +
		" ORDER BY priority DESC, created_at ASC" + scope.page(limit, offset)
	rows, err := r.db.QueryContext(ctx, query, scope.args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapQueryDownlinkQueue, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgCloseRowsDownlinkQueue, logger.FieldError, err)
		}
	}()

	var messages []*storage.DownlinkMessage
	for rows.Next() {
		msg, err := scanQueueListingRow(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapErrorIteratingDownlinkMessages, err)
	}
	return messages, nil
}

// scanQueueListingRow reads one row of the queue listing.
func scanQueueListingRow(row rowScanner) (*storage.DownlinkMessage, error) {
	var result sql.NullString
	var txTime sql.NullInt64
	var earliestAt sql.NullTime
	msg, err := scanDownlinkListing(row, &result, &txTime, &earliestAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapScanDownlinkMessage, err)
	}
	msg.Result = result.String
	msg.TxTime = txTime.Int64
	msg.ScheduledAt = nullTimePtr(earliestAt)
	return msg, nil
}

// CountTenantQueue counts the downlinks ListTenantQueue lists for the filter.
func (r *DownlinkQueueReader) CountTenantQueue(ctx context.Context, tenantID int64, filter storage.DownlinkQueueFilter) (int64, error) {
	scope := downlinkQueueScope(tenantID, filter)
	var count int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM downlink_queue WHERE "+scope.where(), scope.args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapCountDownlinkQueue, err)
	}
	return count, nil
}

// GetTenantIDByQueueID resolves the tenant that owns a specific downlink queue
// Used by TenantResolver for cold-path tenant lookup during result processing
// Implements interfaces.DownlinkQueueStore
func (r *DownlinkQueueReader) GetTenantIDByQueueID(ctx context.Context, queueID uint64) (int64, error) {
	var tenantID int64
	query := `SELECT tenant_id FROM downlink_queue WHERE que_id = $1`
	err := r.db.QueryRowContext(ctx, query, queueID).Scan(&tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf(errFmtQueueIDNotFound, queueID)
		}
		return 0, fmt.Errorf(errFmtGetTenantForQueue, queueID, err)
	}
	return tenantID, nil
}

// downlinkQueueScope renders the predicate shared by the queue listing and
// its count so both always agree on what "in the queue" means. Terminal
// states belong to the results view; without an explicit status the queue
// covers every in-flight state, including rows a base station holds.
func downlinkQueueScope(tenantID int64, filter storage.DownlinkQueueFilter) *sqlScope {
	scope := newSQLScope()
	scope.equals(colTenantID, tenantID)
	if filter.Status != nil {
		scope.equals(colStatus, *filter.Status)
	} else {
		scope.and(sqlDownlinkInFlight)
	}
	if filter.EpEUI != nil {
		scope.equals(colEpEUI, filter.EpEUI[:])
	}
	if filter.BsEUI != nil {
		scope.equals(colBsEUI, filter.BsEUI[:])
	}
	if filter.Priority != nil {
		scope.equals(colPriority, *filter.Priority)
	}
	if filter.QueID != nil {
		scope.equals(colQueID, *filter.QueID)
	}
	scope.organization(filter.OrganizationID)
	if filter.QueuedFrom != nil {
		scope.atLeast(colCreatedAt, *filter.QueuedFrom)
	}
	return scope
}
