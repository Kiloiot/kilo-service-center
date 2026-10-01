package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/lib/pq"
)

// CountOperationsBySession returns how many operations each of the given
// sessions logged; sessions without operations are absent from the map.
func (r *SCACIOperationRepository) CountOperationsBySession(ctx context.Context, tenantID int64, sessionIDs []int64) (map[int64]int64, error) {
	counts := make(map[int64]int64, len(sessionIDs))
	if len(sessionIDs) == 0 {
		return counts, nil
	}
	query := `
		SELECT session_id, COUNT(*)
		FROM scaci_operation_log
		WHERE tenant_id = $1 AND session_id = ANY($2)
		GROUP BY session_id`
	rows, err := r.db.QueryContext(ctx, query, tenantID, pq.Array(sessionIDs))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapCountSessionOperations, err)
	}
	defer r.closeRows(rows)
	for rows.Next() {
		var sessionID, count int64
		if err := rows.Scan(&sessionID, &count); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanSessionOperationCount, err)
		}
		counts[sessionID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapRowsIterationError, err)
	}
	return counts, nil
}

// ListFailedOperationGroups buckets the failed operations initiated inside
// [from, to] by command, error code and token, newest bucket first.
func (r *SCACIOperationRepository) ListFailedOperationGroups(ctx context.Context, tenantID int64, from, to time.Time, limit, offset int) ([]*models.SCACIOperationErrorGroup, int64, error) {
	const countQuery = `
		SELECT COUNT(*) FROM (
			SELECT 1
			FROM scaci_operation_log
			WHERE tenant_id = $1 AND state = $2
			  AND initiated_at >= $3 AND initiated_at <= $4
			GROUP BY command, error_code, error_token
		) AS failed_groups`
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, tenantID, models.OperationStateFailed, from, to).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapCountFailedOperationGroups, err)
	}
	const query = `
		SELECT
			command, error_code, error_token,
			(array_agg(error_message ORDER BY initiated_at DESC))[1],
			(array_agg(session_id ORDER BY initiated_at DESC))[1],
			MIN(initiated_at), MAX(initiated_at), COUNT(*)
		FROM scaci_operation_log
		WHERE tenant_id = $1 AND state = $2
		  AND initiated_at >= $3 AND initiated_at <= $4
		GROUP BY command, error_code, error_token
		ORDER BY MAX(initiated_at) DESC
		LIMIT $5 OFFSET $6`
	rows, err := r.db.QueryContext(ctx, query, tenantID, models.OperationStateFailed, from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapQueryFailedOperationGroups, err)
	}
	defer r.closeRows(rows)
	groups := []*models.SCACIOperationErrorGroup{}
	for rows.Next() {
		var g models.SCACIOperationErrorGroup
		var code sql.NullInt64
		var token, message sql.NullString
		if err := rows.Scan(&g.Command, &code, &token, &message, &g.SessionID, &g.FirstSeen, &g.LastSeen, &g.Count); err != nil {
			return nil, 0, fmt.Errorf("%s: %w", errWrapScanFailedOperationGroup, err)
		}
		if code.Valid {
			c := int(code.Int64)
			g.ErrorCode = &c
		}
		if token.Valid {
			g.ErrorToken = &token.String
		}
		if message.Valid {
			g.ErrorMessage = &message.String
		}
		groups = append(groups, &g)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapRowsIterationError, err)
	}
	return groups, total, nil
}

// GetPingSummary reports the latest completed ping and counts the pings in
// [from, to] that failed or have been pending longer than staleAfter.
func (r *SCACIOperationRepository) GetPingSummary(ctx context.Context, tenantID int64, command string, from, to time.Time, staleAfter time.Duration) (*models.SCACIPingSummary, error) {
	summary := &models.SCACIPingSummary{}
	latest := `
		SELECT initiated_at, completed_at
		FROM scaci_operation_log
		WHERE tenant_id = $1 AND command = $2 AND state = $3 AND completed_at IS NOT NULL
		ORDER BY initiated_at DESC
		LIMIT 1`
	var initiatedAt, completedAt time.Time
	err := r.db.QueryRowContext(ctx, latest, tenantID, command, models.OperationStateCompleted).Scan(&initiatedAt, &completedAt)
	switch {
	case err == nil:
		rtt := completedAt.Sub(initiatedAt)
		summary.LastPingAt = &initiatedAt
		summary.LastPingRTT = &rtt
	case err != sql.ErrNoRows:
		return nil, fmt.Errorf("%s: %w", errWrapQueryLatestPing, err)
	}
	missed := `
		SELECT COUNT(*)
		FROM scaci_operation_log
		WHERE tenant_id = $1 AND command = $2
		  AND initiated_at >= $3 AND initiated_at <= $4
		  AND (state = $5 OR (state IN ($6, $7) AND initiated_at < $8))`
	staleBefore := to.Add(-staleAfter)
	if err := r.db.QueryRowContext(ctx, missed, tenantID, command, from, to,
		models.OperationStateFailed, models.OperationStatePending, models.OperationStateAcknowledged, staleBefore,
	).Scan(&summary.MissedPings); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapCountMissedPings, err)
	}
	return summary, nil
}

// GetLatestOperation returns the tenant's most recent operation for a command,
// or nil when none was logged.
func (r *SCACIOperationRepository) GetLatestOperation(ctx context.Context, tenantID int64, command string) (*models.SCACIOperation, error) {
	query := sqlSelectSCACIOperations + `
		WHERE tenant_id = $1 AND command = $2
		ORDER BY initiated_at DESC
		LIMIT 1`
	op, err := r.scanOperation(r.db.QueryRowContext(ctx, query, tenantID, command))
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapQueryLatestOperation, err)
	}
	return op, nil
}

func (r *SCACIOperationRepository) closeRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		r.log.Warn(logMsgCloseRowsInSCACIOperationQuery, logger.FieldError, err)
	}
}
