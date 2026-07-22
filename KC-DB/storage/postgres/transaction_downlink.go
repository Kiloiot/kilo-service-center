package postgres

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// transactionalMIOTYDownlinkRepository wraps MIOTYDownlinkRepository for transactions
// Implements FOR UPDATE SKIP LOCKED pattern for concurrent dispatch safety
type transactionalMIOTYDownlinkRepository struct {
	tx *sql.Tx
	db *DB
}

var _ interfaces.MIOTYDownlinkRepository = (*transactionalMIOTYDownlinkRepository)(nil)

// ReserveNextPendingDownlink atomically selects+reserves highest-priority pending downlink
// Uses FOR UPDATE SKIP LOCKED to avoid blocking concurrent dispatchers
// Returns nil, nil if no pending downlinks available
// Returns ErrDownlinkAlreadyReserved ONLY when UPDATE affects 0 rows due to race
// orgID filters by organization; nil = no org filter (backward compatible)
func (r *transactionalMIOTYDownlinkRepository) ReserveNextPendingDownlink(
	ctx context.Context,
	tenantID int64,
	epEUI []byte,
	bsEUI uint64,
	orgIDFilter *uuid.UUID,
) (*storage.DownlinkMessage, error) {
	// Step 1: Select pending with lock (SKIP LOCKED prevents race)
	// Use column names matching downlink_queue schema
	baseQuery := `
		SELECT id, que_id, ep_eui, tenant_id, organization_id, payload, priority, status,
		       cnt_depend, packet_cnt, format, response_exp, response_prio,
		       dl_wind_req, exp_only, dl_rx_stat_qry, user_data, created_at
		FROM downlink_queue
		WHERE tenant_id = $1
		  AND ep_eui = $2
		  AND status = $3`

	var selectQuery string
	var selectArgs []interface{}

	if orgIDFilter != nil {
		selectQuery = baseQuery + ` AND (organization_id = $4 OR organization_id IS NULL)
		ORDER BY priority DESC, created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`
		selectArgs = []interface{}{tenantID, epEUI, bssci.DLQueueStatusPending, *orgIDFilter}
	} else {
		selectQuery = baseQuery + `
		ORDER BY priority DESC, created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`
		selectArgs = []interface{}{tenantID, epEUI, bssci.DLQueueStatusPending}
	}

	dl, err := scanDownlinkQueueRow(r.tx.QueryRowContext(ctx, selectQuery, selectArgs...))
	if err == sql.ErrNoRows {
		return nil, nil // No pending downlinks - not an error
	}
	if err != nil {
		return nil, fmt.Errorf("select pending downlink: %w", err)
	}

	// Step 2: Mark as reserved (still within transaction lock)
	// Convert bsEUI to bytea
	bsEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEUIBytes, bsEUI)

	reserveQuery := `
		UPDATE downlink_queue
		SET status = $1, bs_eui = $2, updated_at = NOW()
		WHERE que_id = $3 AND tenant_id = $4 AND status = $5
	`
	result, err := r.tx.ExecContext(ctx, reserveQuery,
		bssci.DLQueueStatusReserved,
		bsEUIBytes,
		dl.QueID,
		tenantID,
		bssci.DLQueueStatusPending,
	)
	if err != nil {
		return nil, fmt.Errorf("reserve downlink: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		// ONLY return this error when row was locked/taken by concurrent tx
		return nil, ErrDownlinkAlreadyReserved
	}

	dl.Status = bssci.DLQueueStatusReserved
	dl.BsEui = bsEUI
	return dl, nil
}

// ReservePendingDownlinkByQueueID atomically reserves one exact pending queue
// row (see interface contract). Shares the single-statement implementation
// with the non-transactional repository.
func (r *transactionalMIOTYDownlinkRepository) ReservePendingDownlinkByQueueID(
	ctx context.Context,
	tenantID int64,
	organizationID *uuid.UUID,
	queueID uint64,
	epEUI []byte,
	bsEUI uint64,
) (*storage.DownlinkMessage, error) {
	return reservePendingDownlinkByQueueID(ctx, r.tx, tenantID, organizationID, queueID, epEUI, bsEUI)
}

// MarkReservedAsQueued transitions reserved → queued with transmission metadata.
// Idempotent per the interface contract; shares the implementation with the
// non-transactional repository.
func (r *transactionalMIOTYDownlinkRepository) MarkReservedAsQueued(
	ctx context.Context,
	queID uint64,
	tenantID int64,
	bsEUI uint64,
	txTime int64,
	packetCnt *uint32,
	orgID *uuid.UUID,
) error {
	return markReservedAsQueued(ctx, r.tx, queID, tenantID, bsEUI, txTime, packetCnt, orgID)
}

// sqlExecQuerier is the subset of *sql.Tx / *sql.DB used by the shared
// downlink dispatch statements so the transactional and regular repositories
// share one implementation.
type sqlExecQuerier interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// downlinkQueueColumns is the shared column list scanned by
// scanDownlinkQueueRow for dispatch reads and RETURNING clauses.
const downlinkQueueColumns = `id, que_id, ep_eui, tenant_id, organization_id, payload, priority, status,
	       cnt_depend, packet_cnt, format, response_exp, response_prio,
	       dl_wind_req, exp_only, dl_rx_stat_qry, user_data, created_at`

// scanDownlinkQueueRow scans one downlink_queue row (downlinkQueueColumns
// order) into a storage.DownlinkMessage. Returns sql.ErrNoRows unwrapped so
// callers can map "no matching row" to their contract.
func scanDownlinkQueueRow(row *sql.Row) (*storage.DownlinkMessage, error) {
	var dl storage.DownlinkMessage
	var epEuiBytes []byte
	var rowTenantID int64
	var packetCntArray pq.Int64Array
	var userDataJSON []byte
	var orgID *uuid.UUID

	err := row.Scan(
		&dl.ID, &dl.QueID, &epEuiBytes, &rowTenantID, &orgID, &dl.Payload, &dl.Priority,
		&dl.Status, &dl.CntDepend, &packetCntArray, &dl.Format,
		&dl.ResponseExp, &dl.ResponsePrio, &dl.DlWindReq, &dl.ExpOnly, &dl.DlRxStatQry,
		&userDataJSON, &dl.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert bytea to string for storage struct
	dl.EPEUI = hex.EncodeToString(epEuiBytes)
	dl.TenantID = fmt.Sprintf("%d", rowTenantID)
	dl.OrganizationID = orgID

	// Convert packet counter array
	if packetCntArray != nil {
		dl.PacketCntArray = []int64(packetCntArray)
	}

	// Nil-safe user_data JSON unmarshaling
	if len(userDataJSON) > 0 {
		var dlQueue mioty.DLDataQueue
		if err := json.Unmarshal(userDataJSON, &dlQueue); err == nil {
			dl.UserData = dlQueue.UserData
			// Populate single Payload field for backward compatibility
			if len(dl.UserData) > 0 && len(dl.Payload) == 0 {
				dl.Payload = dl.UserData[0]
			}
		}
		// Skip unmarshaling errors to handle NULL or empty JSON gracefully
	}

	return &dl, nil
}

// reservePendingDownlinkByQueueID performs the exact-match pending → reserved
// transition as one atomic UPDATE ... RETURNING statement. The row must match
// que_id, tenant_id, and ep_eui and be in 'pending' state; an org-scoped
// request additionally requires organization_id to match exactly (never
// OR organization_id IS NULL).
func reservePendingDownlinkByQueueID(
	ctx context.Context,
	q sqlExecQuerier,
	tenantID int64,
	organizationID *uuid.UUID,
	queueID uint64,
	epEUI []byte,
	bsEUI uint64,
) (*storage.DownlinkMessage, error) {
	bsEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEUIBytes, bsEUI)

	orgFilter := ""
	args := []interface{}{
		bssci.DLQueueStatusReserved,
		bsEUIBytes,
		queueID,
		tenantID,
		epEUI,
		bssci.DLQueueStatusPending,
	}
	if organizationID != nil {
		orgFilter = " AND organization_id = $7"
		args = append(args, *organizationID)
	}

	// nolint:gosec // G201: orgFilter is a static string with a parameterized value, not user input
	query := fmt.Sprintf(`
		UPDATE downlink_queue
		SET status = $1, bs_eui = $2, updated_at = NOW()
		WHERE que_id = $3 AND tenant_id = $4 AND ep_eui = $5 AND status = $6%s
		RETURNING %s
	`, orgFilter, downlinkQueueColumns)

	dl, err := scanDownlinkQueueRow(q.QueryRowContext(ctx, query, args...))
	if err == sql.ErrNoRows {
		return nil, nil // No matching pending row - not an error
	}
	if err != nil {
		return nil, fmt.Errorf("reserve downlink by queue id: %w", err)
	}
	dl.BsEui = bsEUI
	return dl, nil
}

// markReservedAsQueued transitions reserved → queued with transmission
// metadata. Idempotent: a row already 'queued' succeeds unchanged; any other
// state (pending, failed, completed, revoked) is an error because the caller's
// reservation no longer holds.
func markReservedAsQueued(
	ctx context.Context,
	q sqlExecQuerier,
	queID uint64,
	tenantID int64,
	bsEUI uint64,
	txTime int64,
	packetCnt *uint32,
	orgID *uuid.UUID,
) error {
	// Convert bsEUI to bytea
	bsEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEUIBytes, bsEUI)

	// Convert packet counter to int64 for database storage
	var transmissionPacketCnt *int64
	if packetCnt != nil {
		val := int64(*packetCnt)
		transmissionPacketCnt = &val
	}

	// Build organization filter for backward compatibility (nil = no filter)
	orgFilter := ""
	args := []interface{}{
		bssci.DLQueueStatusQueued, // queued = sent to BS, awaiting actual transmission
		txTime,
		transmissionPacketCnt, // nullable - nil preserves NULL
		bsEUIBytes,
		queID,
		tenantID,
		bssci.DLQueueStatusReserved,
	}
	if orgID != nil {
		orgFilter = " AND (organization_id = $8 OR organization_id IS NULL)"
		args = append(args, *orgID)
	}

	// NOTE: transmission_result stays NULL here (not 'sent') because the result arrives via dlDataRes (BSSCI 5.14)
	// Status 'queued' indicates "sent to BS, awaiting actual transmission"
	// nolint:gosec // G201: orgFilter is a static string (empty or " AND (...)" with parameterized value), not user input
	query := fmt.Sprintf(`
		UPDATE downlink_queue
		SET status = $1,
		    transmission_time = $2,
		    transmission_packet_cnt = $3,
		    bs_eui = $4,
		    tx_time = $2,
		    updated_at = NOW()
		WHERE que_id = $5 AND tenant_id = $6 AND status = $7%s
	`, orgFilter)

	result, err := q.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("mark downlink queued: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		return nil
	}

	// Zero rows: idempotent success when the row is already queued (a repair
	// path or crash-recovery retry already confirmed the send); anything else
	// means the reservation no longer holds.
	var status string
	if err := q.QueryRowContext(ctx, `
		SELECT status FROM downlink_queue WHERE que_id = $1 AND tenant_id = $2
	`, queID, tenantID).Scan(&status); err != nil {
		if err == sql.ErrNoRows {
			return ErrDownlinkAlreadyReserved
		}
		return fmt.Errorf("mark downlink queued status check: %w", err)
	}
	if status == bssci.DLQueueStatusQueued {
		return nil
	}
	return ErrDownlinkAlreadyReserved
}

// Delegate ALL existing interface methods to non-transactional DB
// Full delegation to non-transactional DB for read-only methods

func (r *transactionalMIOTYDownlinkRepository) GetDownlinkQueue(ctx context.Context, deviceEUI string, tenantID string) ([]*storage.DownlinkMessage, error) {
	return r.db.MIOTYDownlinks().GetDownlinkQueue(ctx, deviceEUI, tenantID)
}

func (r *transactionalMIOTYDownlinkRepository) GetDownlinkByQueueID(ctx context.Context, queId uint64, tenantID string) (*storage.DownlinkMessage, error) {
	return r.db.MIOTYDownlinks().GetDownlinkByQueueID(ctx, queId, tenantID)
}

func (r *transactionalMIOTYDownlinkRepository) GetDownlinkByPacketCnt(ctx context.Context, tenantID string, epEui string, packetCnt uint32) (*storage.DownlinkMessage, error) {
	return r.db.MIOTYDownlinks().GetDownlinkByPacketCnt(ctx, tenantID, epEui, packetCnt)
}

func (r *transactionalMIOTYDownlinkRepository) GetDownlinkResults(ctx context.Context, deviceEUI string, tenantID string, orgID *uuid.UUID, statusFilter string, timeFrom, timeTo *time.Time, limit, offset int) ([]*storage.DownlinkMessage, int, error) {
	return r.db.MIOTYDownlinks().GetDownlinkResults(ctx, deviceEUI, tenantID, orgID, statusFilter, timeFrom, timeTo, limit, offset)
}

func (r *transactionalMIOTYDownlinkRepository) EnqueueDownlink(ctx context.Context, dl *storage.DownlinkMessage) (*storage.DownlinkMessage, error) {
	return r.db.MIOTYDownlinks().EnqueueDownlink(ctx, dl)
}

func (r *transactionalMIOTYDownlinkRepository) UpdateDownlinkStatus(ctx context.Context, id string, status string, orgID *uuid.UUID) error {
	return r.db.MIOTYDownlinks().UpdateDownlinkStatus(ctx, id, status, orgID)
}

func (r *transactionalMIOTYDownlinkRepository) UpdateDownlinkResult(ctx context.Context, queId int64, result string, txTime *int64, packetCnt *uint32, bsEUI []byte, epEUI []byte, tenantID string, orgID *uuid.UUID) error {
	return r.db.MIOTYDownlinks().UpdateDownlinkResult(ctx, queId, result, txTime, packetCnt, bsEUI, epEUI, tenantID, orgID)
}

func (r *transactionalMIOTYDownlinkRepository) UpdateDownlinkBaseStation(ctx context.Context, queId uint64, tenantID string, bsEUI uint64) error {
	return r.db.MIOTYDownlinks().UpdateDownlinkBaseStation(ctx, queId, tenantID, bsEUI)
}

func (r *transactionalMIOTYDownlinkRepository) RevokeDownlink(ctx context.Context, queId int64, tenantID string) error {
	return r.db.MIOTYDownlinks().RevokeDownlink(ctx, queId, tenantID)
}
