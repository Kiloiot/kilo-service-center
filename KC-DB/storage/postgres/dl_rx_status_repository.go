package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// DLRXStatusRepository handles DL RX status persistence per BSSCI §3.15
type DLRXStatusRepository struct {
	db     *sqlx.DB
	logger logger.Logger
}

const (
	dlRxStatusPending  = "pending"
	dlRxStatusReceived = "received"
	dlRxStatusTimeout  = "timeout"
)

// NewDLRXStatusRepository creates a new DL RX status repository
// DL RX status lifecycle values persisted in dl_rx_status.status.
func NewDLRXStatusRepository(db *sqlx.DB, logger logger.Logger) *DLRXStatusRepository {
	return &DLRXStatusRepository{
		db:     db,
		logger: logger,
	}
}

// Use the shared DLRXStatus type from the mioty package

// CreateDLRXStatus persists a new DL RX status report
func (r *DLRXStatusRepository) CreateDLRXStatus(ctx context.Context, status *mioty.DLRXStatus) error {
	query := `
		INSERT INTO dl_rx_status (
			tenant_id, org_uuid, ep_eui, bs_eui, rx_time, packet_cnt,
			dl_rx_snr, dl_rx_rssi, created_at, updated_at
		) VALUES (
			:tenant_id, :org_uuid, :ep_eui, :bs_eui, :rx_time, :packet_cnt,
			:dl_rx_snr, :dl_rx_rssi, NOW(), NOW()
		) RETURNING id, created_at, updated_at`

	// Create binding struct with int64 for PostgreSQL BIGINT compatibility
	// PacketCnt is uint32 in storage model but BIGINT in database
	type bindParams struct {
		TenantID       int64   `db:"tenant_id"`
		OrganizationID *string `db:"org_uuid"` // UUID as string for binding (nullable)
		EpEui          []byte  `db:"ep_eui"`
		BsEui          []byte  `db:"bs_eui"`
		RxTime         int64   `db:"rx_time"`
		PacketCnt      int64   `db:"packet_cnt"` // uint32 → int64 for BIGINT binding
		DlRxSnr        float64 `db:"dl_rx_snr"`
		DlRxRssi       float64 `db:"dl_rx_rssi"`
	}

	// Convert UUID to string for binding (nil if not set)
	var orgUUID *string
	if status.OrganizationID != nil {
		s := status.OrganizationID.String()
		orgUUID = &s
	}

	params := bindParams{
		TenantID:       status.TenantID,
		OrganizationID: orgUUID,
		EpEui:          status.EpEui,
		BsEui:          status.BsEui,
		RxTime:         status.RxTime,
		PacketCnt:      int64(status.PacketCnt), // Safe: uint32 fits in int64
		DlRxSnr:        status.DlRxSnr,
		DlRxRssi:       status.DlRxRssi,
	}

	rows, err := r.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapInsertDLRXStatus, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.logger.Warn(logMsgDlrxRowsClose, logger.FieldError, err)
		}
	}()

	if rows.Next() {
		if err := rows.Scan(&status.ID, &status.CreatedAt, &status.UpdatedAt); err != nil {
			return fmt.Errorf("%s: %w", errWrapScanReturningValues, err)
		}
	}

	r.logger.Debug(logMsgCreatedDLRXStatusRecord,
		logger.FieldID, status.ID,
		logger.FieldTenantIDCamel, status.TenantID)

	return nil
}

// GetDLRXStatusByEndpoint retrieves DL RX status records for a specific endpoint
func (r *DLRXStatusRepository) GetDLRXStatusByEndpoint(ctx context.Context, tenantID int64, epEui []byte, limit, offset int, startTime, endTime *time.Time) ([]*mioty.DLRXStatus, int, error) {
	// Get total count
	var totalCount int

	// Build WHERE clause for count query using rx_time (nanoseconds)
	whereClauses := []string{"tenant_id = $1", "ep_eui = $2"}
	args := []interface{}{tenantID, epEui}
	argCount := len(args)

	if startTime != nil {
		startNs := startTime.UnixNano()
		argCount++
		whereClauses = append(whereClauses, fmt.Sprintf("rx_time >= $%d", argCount))
		args = append(args, startNs)
	}
	if endTime != nil {
		endNs := endTime.UnixNano()
		argCount++
		whereClauses = append(whereClauses, fmt.Sprintf("rx_time <= $%d", argCount))
		args = append(args, endNs)
	}

	whereClause := "WHERE " + whereClauses[0]
	for i := 1; i < len(whereClauses); i++ {
		whereClause += " AND " + whereClauses[i]
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM dl_rx_status %s", whereClause)
	if err := r.db.GetContext(ctx, &totalCount, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapCountDLRXStatusRecords, err)
	}

	// Get paginated results using rx_time per BSSCI §5.15.1
	query := fmt.Sprintf(`
		SELECT id, tenant_id, ep_eui, bs_eui, rx_time, packet_cnt,
		       dl_rx_snr, dl_rx_rssi, created_at, updated_at
		FROM dl_rx_status
		%s
		ORDER BY rx_time DESC
		LIMIT $%d OFFSET $%d`, whereClause, argCount+1, argCount+2)

	args = append(args, limit, offset)

	var statuses []*mioty.DLRXStatus
	if err := r.db.SelectContext(ctx, &statuses, query, args...); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapQueryDLRXStatus, err)
	}

	return statuses, totalCount, nil
}

// GetAverageDLRXMetrics calculates average SNR and RSSI for an endpoint over a time period
func (r *DLRXStatusRepository) GetAverageDLRXMetrics(ctx context.Context, tenantID int64, epEui []byte, startTime, endTime *time.Time) (avgSnr, avgRssi float64, count int, err error) {
	query := `
		SELECT
			AVG(dl_rx_snr) as avg_snr,
			AVG(dl_rx_rssi) as avg_rssi,
			COUNT(*) as count
		FROM dl_rx_status
		WHERE tenant_id = $1
		  AND ep_eui = $2`

	args := []interface{}{tenantID, epEui}

	// Convert times to nanoseconds for BSSCI §5.15.1 rx_time format
	if startTime != nil {
		startNs := startTime.UnixNano()
		query += " AND rx_time >= $3"
		args = append(args, startNs)
	}
	if endTime != nil {
		endNs := endTime.UnixNano()
		query += fmt.Sprintf(" AND rx_time <= $%d", len(args)+1)
		args = append(args, endNs)
	}

	row := r.db.QueryRowContext(ctx, query, args...)

	var avgSnrNull, avgRssiNull sql.NullFloat64
	err = row.Scan(&avgSnrNull, &avgRssiNull, &count)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%s: %w", errWrapCalculateAverageMetrics, err)
	}

	if avgSnrNull.Valid {
		avgSnr = avgSnrNull.Float64
	}
	if avgRssiNull.Valid {
		avgRssi = avgRssiNull.Float64
	}

	return avgSnr, avgRssi, count, nil
}

// CreateDLRXStatusQuery tracks a dlRxStatQry request for correlation (BSSCI §5.15 audit trail)
func (r *DLRXStatusRepository) CreateDLRXStatusQuery(ctx context.Context, tenantID int64, orgUUID *uuid.UUID, epEui, bsEui []byte, opId int64) error {
	ctx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()

	query := `
		INSERT INTO dl_rx_status_queries (
			tenant_id, org_uuid, ep_eui, bs_eui, op_id, status, requested_at
		) VALUES (
			$1, $2, $3, $4, $5, 'pending', NOW()
		)
		ON CONFLICT (tenant_id, ep_eui, op_id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query, tenantID, orgUUID, epEui, bsEui, opId)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapCreateDLRXStatusQuery, err)
	}

	return nil
}

// MarkDLRXStatusReceived marks the OLDEST pending query for the tenant, endpoint,
// AND expected base station as received (BSSCI §5.15). Correlating on the expected
// bs_eui prevents a report from one base station satisfying a concurrent query
// issued for a different base station. The actual BS EUI/opId are recorded for
// audit. Returns true if a matching pending query was found and updated.
func (r *DLRXStatusRepository) MarkDLRXStatusReceived(ctx context.Context, tenantID int64, epEui []byte, bsEui []byte, bsOpID int64) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()

	// CTE ensures atomic lookup + update in single statement (no race between handlers)
	query := `
		WITH target_query AS (
			SELECT id FROM dl_rx_status_queries
			WHERE tenant_id = $1
			  AND ep_eui = $2
			  AND bs_eui = $3
			  AND status = 'pending'
			ORDER BY requested_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE dl_rx_status_queries
		SET status = 'received',
		    received_at = NOW(),
		    bs_eui_actual = $3,
		    bs_op_id_actual = $4
		FROM target_query
		WHERE dl_rx_status_queries.id = target_query.id
		RETURNING dl_rx_status_queries.id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, tenantID, epEui, bsEui, bsOpID).Scan(&id)
	if err == sql.ErrNoRows {
		// No pending query found - this is not an error, just unsolicited dlRxStat
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", errWrapMarkDLRXStatusQueryReceived, err)
	}

	return true, nil
}

// ExpireDLRXStatusQuery marks queries older than cutoff as 'timeout'
// Returns count of expired queries (used by cleanup job)
func (r *DLRXStatusRepository) ExpireDLRXStatusQuery(ctx context.Context, cutoff time.Time) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()

	query := `
		UPDATE dl_rx_status_queries
		SET status = 'timeout',
		    received_at = NOW()
		WHERE status = 'pending'
		  AND requested_at < $1`

	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapExpireDLRXStatusQueries, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapFailedToGetRowsAffected, err)
	}

	if rowsAffected > 0 {
		r.logger.Info(logMsgExpiredDLRXStatusQueries,
			logger.FieldCutoff, cutoff,
			logger.FieldRowsExpired, rowsAffected)
	}

	return rowsAffected, nil
}

// GetDLRXStatusQueryHistory retrieves query tracking records for an endpoint (BSSCI §5.15 telemetry)
func (r *DLRXStatusRepository) GetDLRXStatusQueryHistory(ctx context.Context, tenantID int64, epEui []byte, limit, offset int, startTime, endTime *time.Time) ([]*mioty.DLRXStatusQuery, int, error) {
	ctx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()

	// Build WHERE clause with filters
	whereClauses := []string{"tenant_id = $1", "ep_eui = $2"}
	args := []interface{}{tenantID, epEui}
	argCount := len(args)

	if startTime != nil {
		argCount++
		whereClauses = append(whereClauses, fmt.Sprintf("requested_at >= $%d", argCount))
		args = append(args, *startTime)
	}
	if endTime != nil {
		argCount++
		whereClauses = append(whereClauses, fmt.Sprintf("requested_at <= $%d", argCount))
		args = append(args, *endTime)
	}

	whereClause := "WHERE " + whereClauses[0]
	for i := 1; i < len(whereClauses); i++ {
		whereClause += " AND " + whereClauses[i]
	}

	// Get total count
	var totalCount int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM dl_rx_status_queries %s", whereClause)
	if err := r.db.GetContext(ctx, &totalCount, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapCountDLRXStatusQueries, err)
	}

	// Get paginated results ordered by requested_at DESC (most recent first)
	query := fmt.Sprintf(`
		SELECT id, tenant_id, org_uuid, ep_eui, bs_eui, op_id, status, requested_at, received_at
		FROM dl_rx_status_queries
		%s
		ORDER BY requested_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argCount+1, argCount+2)

	args = append(args, limit, offset)

	var queries []*mioty.DLRXStatusQuery
	if err := r.db.SelectContext(ctx, &queries, query, args...); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapQueryDLRXStatusQueryHistory, err)
	}

	return queries, totalCount, nil
}

// GetDLRXStatusQueryStats calculates query status distribution for an endpoint (BSSCI §5.15 telemetry)
func (r *DLRXStatusRepository) GetDLRXStatusQueryStats(ctx context.Context, tenantID int64, epEui []byte, startTime, endTime *time.Time) (pending, received, timeout int64, err error) {
	ctx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()

	// Build WHERE clause with filters
	whereClauses := []string{"tenant_id = $1", "ep_eui = $2"}
	args := []interface{}{tenantID, epEui}
	argCount := len(args)

	if startTime != nil {
		argCount++
		whereClauses = append(whereClauses, fmt.Sprintf("requested_at >= $%d", argCount))
		args = append(args, *startTime)
	}
	if endTime != nil {
		argCount++
		whereClauses = append(whereClauses, fmt.Sprintf("requested_at <= $%d", argCount))
		args = append(args, *endTime)
	}

	whereClause := "WHERE " + whereClauses[0]
	for i := 1; i < len(whereClauses); i++ {
		whereClause += " AND " + whereClauses[i]
	}

	// Execute GROUP BY query to get status distribution
	query := fmt.Sprintf(`
		SELECT status, COUNT(*) as count
		FROM dl_rx_status_queries
		%s
		GROUP BY status`, whereClause)

	type statusCount struct {
		Status string `db:"status"`
		Count  int64  `db:"count"`
	}

	var results []statusCount
	if err := r.db.SelectContext(ctx, &results, query, args...); err != nil {
		return 0, 0, 0, fmt.Errorf("%s: %w", errWrapQueryDLRXStatusQueryStats, err)
	}

	// Map results to return values
	for _, result := range results {
		switch result.Status {
		case dlRxStatusPending:
			pending = result.Count
		case dlRxStatusReceived:
			received = result.Count
		case dlRxStatusTimeout:
			timeout = result.Count
		}
	}

	return pending, received, timeout, nil
}

// GetDLRXStatusSinceLastHeard returns, per base station, the latest DL RX status
// reported after the endpoint was last heard, so each report is attached to the
// first uplink after it (SCACI §3.8.1 "previous DL reception") in a single query.
func (r *DLRXStatusRepository) GetDLRXStatusSinceLastHeard(
	ctx context.Context,
	tenantID int64,
	epEui []byte,
	bsEuis [][]byte,
) ([]*mioty.DLRXStatus, error) {
	if len(bsEuis) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
	defer cancel()

	// Arrival order bounds the report: a base station may send dlRxStat before or after the ulData it rode in.
	query := `
		SELECT DISTINCT ON (d.bs_eui)
			d.id, d.tenant_id, d.ep_eui, d.bs_eui, d.dl_rx_snr, d.dl_rx_rssi, d.rx_time, d.packet_cnt, d.created_at, d.updated_at
		FROM dl_rx_status d
		WHERE d.tenant_id = $1 AND d.ep_eui = $2 AND d.bs_eui = ANY($3)
		  AND d.created_at > COALESCE(
		      (SELECT e.last_seen_at FROM endpoints e WHERE e.ep_eui = $2 AND e.owner_tenant_id = $1),
		      '-infinity'::timestamptz)
		ORDER BY d.bs_eui, d.rx_time DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, epEui, pq.Array(bsEuis))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapBatchDlRxStatusQuery, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.logger.Warn(logMsgDlrxRowsClose, logger.FieldError, err)
		}
	}()

	var results []*mioty.DLRXStatus
	for rows.Next() {
		var status mioty.DLRXStatus
		if err := rows.Scan(
			&status.ID, &status.TenantID, &status.EpEui, &status.BsEui,
			&status.DlRxSnr, &status.DlRxRssi, &status.RxTime, &status.PacketCnt,
			&status.CreatedAt, &status.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanDlRxStatus, err)
		}
		results = append(results, &status)
	}
	return results, rows.Err()
}
