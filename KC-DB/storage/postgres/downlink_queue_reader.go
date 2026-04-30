package postgres

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/lib/pq"
)

// DownlinkQueueReader implements read-only downlink queue operations
// Reuses storage.DownlinkMessage to avoid struct duplication
type DownlinkQueueReader struct {
	db *sqlx.DB
}

// Ensure DownlinkQueueReader implements the interfaces
var _ interfaces.DownlinkQueueReader = (*DownlinkQueueReader)(nil)
var _ interfaces.DownlinkQueueStore = (*DownlinkQueueReader)(nil)

// NewDownlinkQueueReader creates a new downlink queue reader
func NewDownlinkQueueReader(db *sqlx.DB) *DownlinkQueueReader {
	return &DownlinkQueueReader{db: db}
}

// ListTenantQueue retrieves pending downlink messages for a tenant
func (r *DownlinkQueueReader) ListTenantQueue(ctx context.Context, tenantID int64, epFilter *[8]byte, limit, offset int) ([]*storage.DownlinkMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	// Build query with optional endpoint filter
	query := `
		SELECT
			id, ep_eui, tenant_id, organization_id, payload,
			priority, status, attempts, max_attempts,
			que_id, cnt_depend, packet_cnt, format,
			response_exp, response_prio, dl_wind_req, exp_only, dl_rx_stat_qry,
			result, tx_time, bs_eui,
			created_at, transmitted_at, acknowledged_at, earliest_at, user_data
		FROM downlink_queue
		WHERE tenant_id = $1
		AND status IN ($2, $3)`

	args := []interface{}{tenantID, bssci.DLQueueStatusPending, bssci.DLQueueStatusScheduled}
	argIndex := 4

	// Add endpoint filter if provided
	if epFilter != nil {
		query += fmt.Sprintf(" AND ep_eui = $%d", argIndex)
		args = append(args, epFilter[:])
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY priority DESC, created_at ASC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query downlink queue: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// TODO: Repository lacks logger field - add for proper error tracking
			log.Printf("failed to close rows in downlink queue query: %v", err)
		}
	}()

	var messages []*storage.DownlinkMessage

	for rows.Next() {
		var msg storage.DownlinkMessage
		var epEuiBytes []byte
		var bsEuiBytes []byte
		var packetCntArray pq.Int64Array
		var result sql.NullString
		var txTime sql.NullInt64
		var transmittedAt, acknowledgedAt, earliestAt sql.NullTime
		var userDataJSON []byte
		var orgID *uuid.UUID

		err := rows.Scan(
			&msg.ID,
			&epEuiBytes,
			&tenantID,
			&orgID,
			&msg.Payload,
			&msg.Priority,
			&msg.Status,
			&msg.Attempts,
			&msg.MaxAttempts,
			&msg.QueID,
			&msg.CntDepend,
			&packetCntArray,
			&msg.Format,
			&msg.ResponseExp,
			&msg.ResponsePrio,
			&msg.DlWindReq,
			&msg.ExpOnly,
			&msg.DlRxStatQry,
			&result,
			&txTime,
			&bsEuiBytes,
			&msg.CreatedAt,
			&transmittedAt,
			&acknowledgedAt,
			&earliestAt,
			&userDataJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan downlink message: %w", err)
		}

		// Convert bytes to hex strings
		msg.EPEUI = hex.EncodeToString(epEuiBytes)
		msg.TenantID = strconv.FormatInt(tenantID, 10)
		msg.OrganizationID = orgID

		if len(bsEuiBytes) == 8 {
			msg.TxBSEUI = hex.EncodeToString(bsEuiBytes)
		}

		// Convert nullable fields
		if result.Valid {
			msg.Result = result.String
		}
		if txTime.Valid {
			msg.TxTime = txTime.Int64
		}
		if transmittedAt.Valid {
			msg.SentAt = &transmittedAt.Time
		}
		if earliestAt.Valid {
			msg.ScheduledAt = &earliestAt.Time
		}

		// Convert packet counter array
		if packetCntArray != nil {
			msg.PacketCntArray = []int64(packetCntArray)
		}

		// Nil-safe user_data JSON unmarshaling
		if len(userDataJSON) > 0 {
			var dlQueue mioty.DLDataQueue
			if err := json.Unmarshal(userDataJSON, &dlQueue); err == nil {
				msg.UserData = dlQueue.UserData
				// Populate single Payload field for backward compatibility
				if len(msg.UserData) > 0 && msg.Payload == nil {
					msg.Payload = msg.UserData[0]
				}
			}
			// Skip unmarshaling errors to handle NULL or empty JSON gracefully
		}

		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating downlink messages: %w", err)
	}

	return messages, nil
}

// CountTenantQueue returns the count of pending downlink messages for a tenant
// epFilter optionally filters by endpoint EUI (must match ListTenantQueue filter)
// Use parameterized status constants to avoid literal strings
func (r *DownlinkQueueReader) CountTenantQueue(ctx context.Context, tenantID int64, epFilter *[8]byte) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM downlink_queue
		WHERE tenant_id = $1
		AND status IN ($2, $3)`

	args := []interface{}{tenantID, bssci.DLQueueStatusPending, bssci.DLQueueStatusScheduled}

	// Add endpoint filter if provided (matches ListTenantQueue logic)
	if epFilter != nil {
		query += " AND ep_eui = $4"
		args = append(args, epFilter[:])
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count downlink queue: %w", err)
	}

	return count, nil
}

// GetTenantIDByQueueID resolves the tenant that owns a specific downlink queue
// Used by TenantResolver for cold-path tenant lookup during result processing
// Implements interfaces.DownlinkQueueStore
func (r *DownlinkQueueReader) GetTenantIDByQueueID(ctx context.Context, queueID uint64) (int64, error) {
	var tenantID int64
	// Reuse exact SQL from old Server.getDownlinkTenantByQueueID method
	query := `SELECT tenant_id FROM downlink_queue WHERE que_id = $1`
	err := r.db.QueryRowContext(ctx, query, queueID).Scan(&tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("queue ID not found: %d", queueID)
		}
		return 0, fmt.Errorf("failed to get tenant for queue %d: %w", queueID, err)
	}
	return tenantID, nil
}
