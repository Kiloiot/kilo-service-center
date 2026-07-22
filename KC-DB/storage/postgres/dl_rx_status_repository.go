package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// DLRXStatusRepository handles DL RX status persistence per BSSCI §3.15
type DLRXStatusRepository struct {
	db     *sqlx.DB
	logger logger.Logger
}

// NewDLRXStatusRepository creates a new DL RX status repository
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
		return fmt.Errorf("failed to insert DL RX status: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.logger.Warn("dlrx rows close failed", "error", err)
		}
	}()

	if rows.Next() {
		if err := rows.Scan(&status.ID, &status.CreatedAt, &status.UpdatedAt); err != nil {
			return fmt.Errorf("failed to scan returning values: %w", err)
		}
	}

	r.logger.Debug("Created DL RX status record",
		"id", status.ID,
		"tenantId", status.TenantID)

	return nil
}

// GetDLRXStatusByEndpoint retrieves DL RX status records for a specific endpoint
func (r *DLRXStatusRepository) GetDLRXStatusByEndpoint(ctx context.Context, tenantID int64, epEui []byte, limit, offset int, startTime, endTime *time.Time) ([]*mioty.DLRXStatus, int, error) {
	// Get total count
	var totalCount int

	// Build WHERE clause for count query using rx_time (nanoseconds)
	whereClauses := []string{"tenant_id = $1", "ep_eui = $2"}
	args := []interface{}{tenantID, epEui}
	argCount := 2

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
		return nil, 0, fmt.Errorf("failed to count DL RX status records: %w", err)
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
		return nil, 0, fmt.Errorf("failed to query DL RX status: %w", err)
	}

	return statuses, totalCount, nil
}

// GetLatestDLRXStatus retrieves the most recent DL RX status for an endpoint
func (r *DLRXStatusRepository) GetLatestDLRXStatus(ctx context.Context, tenantID int64, epEui []byte) (*mioty.DLRXStatus, error) {
	query := `
		SELECT id, tenant_id, ep_eui, bs_eui, rx_time, packet_cnt,
		       dl_rx_snr, dl_rx_rssi, created_at, updated_at
		FROM dl_rx_status
		WHERE tenant_id = $1 AND ep_eui = $2
		ORDER BY rx_time DESC
		LIMIT 1`

	var status mioty.DLRXStatus
	if err := r.db.GetContext(ctx, &status, query, tenantID, epEui); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest DL RX status: %w", err)
	}

	return &status, nil
}

// GetLatestDLRXStatusPerEndpoint retrieves the latest status for all endpoints (aggregate)
func (r *DLRXStatusRepository) GetLatestDLRXStatusPerEndpoint(ctx context.Context, tenantID int64) ([]*mioty.DLRXStatus, error) {
	query := `
		WITH latest_status AS (
			SELECT DISTINCT ON (ep_eui)
				id, tenant_id, ep_eui, bs_eui, rx_time, packet_cnt,
				dl_rx_snr, dl_rx_rssi, created_at, updated_at
			FROM dl_rx_status
			WHERE tenant_id = $1
			ORDER BY ep_eui, rx_time DESC
		)
		SELECT * FROM latest_status
		ORDER BY rx_time DESC`

	var statuses []*mioty.DLRXStatus
	if err := r.db.SelectContext(ctx, &statuses, query, tenantID); err != nil {
		return nil, fmt.Errorf("failed to get latest DL RX status per endpoint: %w", err)
	}

	return statuses, nil
}

// GetDLRXStatusByTimeRange retrieves DL RX status records within a time window
func (r *DLRXStatusRepository) GetDLRXStatusByTimeRange(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]*mioty.DLRXStatus, error) {
	// Convert times to nanoseconds for BSSCI §5.15.1 rx_time format
	startNs := startTime.UnixNano()
	endNs := endTime.UnixNano()

	query := `
		SELECT id, tenant_id, ep_eui, bs_eui, rx_time, packet_cnt,
		       dl_rx_snr, dl_rx_rssi, created_at, updated_at
		FROM dl_rx_status
		WHERE tenant_id = $1
		  AND rx_time >= $2
		  AND rx_time <= $3
		ORDER BY rx_time DESC`

	var statuses []*mioty.DLRXStatus
	if err := r.db.SelectContext(ctx, &statuses, query, tenantID, startNs, endNs); err != nil {
		return nil, fmt.Errorf("failed to query DL RX status by time range: %w", err)
	}

	return statuses, nil
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
		return 0, 0, 0, fmt.Errorf("failed to calculate average metrics: %w", err)
	}

	if avgSnrNull.Valid {
		avgSnr = avgSnrNull.Float64
	}
	if avgRssiNull.Valid {
		avgRssi = avgRssiNull.Float64
	}

	return avgSnr, avgRssi, count, nil
}

// DeleteOldDLRXStatus removes DL RX status records older than the retention period
func (r *DLRXStatusRepository) DeleteOldDLRXStatus(ctx context.Context, tenantID int64, retentionDays int) (int64, error) {
	query := `
		DELETE FROM dl_rx_status
		WHERE tenant_id = $1
		  AND created_at < NOW() - INTERVAL '%d days'`

	result, err := r.db.ExecContext(ctx, fmt.Sprintf(query, retentionDays), tenantID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old DL RX status records: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		r.logger.Info("Deleted old DL RX status records",
			"tenantId", tenantID,
			"retentionDays", retentionDays,
			"rowsDeleted", rowsAffected)
	}

	return rowsAffected, nil
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
		return fmt.Errorf("failed to create DL RX status query: %w", err)
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
		return false, fmt.Errorf("failed to mark DL RX status query received: %w", err)
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
		return 0, fmt.Errorf("failed to expire DL RX status queries: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		r.logger.Info("Expired DL RX status queries",
			"cutoff", cutoff,
			"rowsExpired", rowsAffected)
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
	argCount := 2

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
		return nil, 0, fmt.Errorf("failed to count DL RX status queries: %w", err)
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
		return nil, 0, fmt.Errorf("failed to query DL RX status query history: %w", err)
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
	argCount := 2

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
		return 0, 0, 0, fmt.Errorf("failed to query DL RX status query stats: %w", err)
	}

	// Map results to return values
	for _, result := range results {
		switch result.Status {
		case "pending":
			pending = result.Count
		case "received":
			received = result.Count
		case "timeout":
			timeout = result.Count
		}
	}

	return pending, received, timeout, nil
}

// GetLatestDLRXStatusByBaseStations returns latest DL RX status for each bs_eui in a single query (SCACI §3.8.1)
// Used for batch hydration of dlRxSnr/dlRxRssi in multi-BS UL data without N+1 queries
// Uses DISTINCT ON (Postgres-specific) to get one row per base station
// Signature uses []byte for consistency with other DL RX methods (caller does uint64→bytes conversion)
func (r *DLRXStatusRepository) GetLatestDLRXStatusByBaseStations(
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

	// DISTINCT ON is Postgres-specific - gets latest per bs_eui in single query
	// This avoids N+1 queries when hydrating DL RX metrics for multi-BS receptions
	query := `
		SELECT DISTINCT ON (bs_eui)
			id, tenant_id, ep_eui, bs_eui, dl_rx_snr, dl_rx_rssi, rx_time, packet_cnt, created_at, updated_at
		FROM dl_rx_status
		WHERE tenant_id = $1 AND ep_eui = $2 AND bs_eui = ANY($3)
		ORDER BY bs_eui, rx_time DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, epEui, pq.Array(bsEuis))
	if err != nil {
		return nil, fmt.Errorf("batch dl_rx_status query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []*mioty.DLRXStatus
	for rows.Next() {
		var status mioty.DLRXStatus
		if err := rows.Scan(
			&status.ID, &status.TenantID, &status.EpEui, &status.BsEui,
			&status.DlRxSnr, &status.DlRxRssi, &status.RxTime, &status.PacketCnt,
			&status.CreatedAt, &status.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan dl_rx_status: %w", err)
		}
		results = append(results, &status)
	}
	return results, rows.Err()
}
