package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// sqlAlertConditions selects the tenant's alert-level events ($1): those in
// the statuses $2, the ones not in status $3 when $2 is empty; of the
// severities $4, warning and above when $4 is empty; of the categories $5,
// any when empty; that occurred after $6 when it is set.
const sqlAlertConditions = ` WHERE tenant_id = $1
	AND (status = ANY($2::text[]) OR (cardinality($2::text[]) = 0 AND status <> $3))
	AND (severity = ANY($4::text[]) OR (cardinality($4::text[]) = 0 AND severity IN ('warning', 'error', 'critical')))
	AND (event_category = ANY($5::text[]) OR cardinality($5::text[]) = 0)
	AND ($6::timestamptz IS NULL OR occurred_at > $6::timestamptz)`

// sqlGetActiveAlerts pages the alerts, newest first: at most $7 (all when
// NULL) after skipping $8.
const sqlGetActiveAlerts = `SELECT id, tenant_id, event_type, event_category, severity,
		source_type, source_id, source_name, title, description, data,
		status, occurred_at, recorded_at
		FROM system_events` + sqlAlertConditions + `
		ORDER BY occurred_at DESC LIMIT $7 OFFSET $8`

// sqlCountActiveAlerts counts the alerts sqlGetActiveAlerts pages through.
const sqlCountActiveAlerts = `SELECT COUNT(*) FROM system_events` + sqlAlertConditions

// sqlCountAlertsBySeverity counts the same alerts per severity.
const sqlCountAlertsBySeverity = `SELECT severity, COUNT(*) FROM system_events` + sqlAlertConditions + `
		GROUP BY severity`

// alertConditionArgs binds the filter to sqlAlertConditions.
func alertConditionArgs(filter models.AlertFilter) ([]interface{}, error) {
	if filter.TenantID == "" {
		return nil, errTextTenantIDRequired
	}
	tenantIDInt, err := strconv.ParseInt(filter.TenantID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapInvalidTenantIDFormat, err)
	}
	var since interface{}
	if filter.Since != nil {
		since = *filter.Since
	}
	return []interface{}{
		tenantIDInt,
		textArray(filter.Statuses),
		models.EventStatusResolved,
		textArray(filter.Severities),
		textArray(filter.Categories),
		since,
	}, nil
}

// textArray binds values as a text[]; no values bind the empty array, never NULL.
func textArray(values []string) interface{} {
	return pq.Array(append([]string{}, values...))
}

// GetActiveAlerts retrieves the tenant's alert-level events in the filter's
// statuses, unresolved ones by default.
func (s *SystemEventStore) GetActiveAlerts(ctx context.Context, filter models.AlertFilter) ([]*models.SystemEvent, error) {
	args, err := alertConditionArgs(filter)
	if err != nil {
		return nil, err
	}
	var limit interface{}
	if filter.Limit > 0 {
		limit = filter.Limit
	}
	args = append(args, limit, max(filter.Offset, 0))

	rows, err := s.db.QueryContext(ctx, sqlGetActiveAlerts, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetActiveAlerts, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.log.Warn(logMsgCloseRowsActiveAlerts, logger.FieldError, err)
		}
	}()

	var events []*models.SystemEvent
	for rows.Next() {
		event := &models.SystemEvent{}
		var tenantIDDB int64
		var detailsJSON []byte
		var sourceID uuid.NullUUID
		if err := rows.Scan(
			&event.ID, &tenantIDDB, &event.EventType, &event.Category,
			&event.Severity, &event.SourceType, &sourceID, &event.SourceName,
			&event.Title, &event.Description, &detailsJSON,
			&event.Status, &event.CreatedAt, &event.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanActiveAlert, err)
		}
		if sourceID.Valid {
			event.SourceID = &sourceID.UUID
		}
		event.TenantID = strconv.FormatInt(tenantIDDB, 10)
		if len(detailsJSON) > 0 {
			event.Details = json.RawMessage(detailsJSON)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateActiveAlerts, err)
	}

	return events, nil
}

// CountActiveAlerts counts the alerts GetActiveAlerts pages through.
func (s *SystemEventStore) CountActiveAlerts(ctx context.Context, filter models.AlertFilter) (int64, error) {
	args, err := alertConditionArgs(filter)
	if err != nil {
		return 0, err
	}

	var count int64
	if err = s.db.QueryRowContext(ctx, sqlCountActiveAlerts, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapCountActiveAlerts, err)
	}

	return count, nil
}

// CountAlertsBySeverity counts, per severity, the alerts CountActiveAlerts
// counts. A severity without alerts is absent from the map.
func (s *SystemEventStore) CountAlertsBySeverity(ctx context.Context, filter models.AlertFilter) (map[string]int64, error) {
	args, err := alertConditionArgs(filter)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, sqlCountAlertsBySeverity, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapCountAlertsBySeverity, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.log.Warn(logMsgCloseRowsAlertSeverities, logger.FieldError, err)
		}
	}()

	counts := make(map[string]int64)
	for rows.Next() {
		var severity string
		var count int64
		if err := rows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanSeverityStat, err)
		}
		counts[severity] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateSeverityStats, err)
	}

	return counts, nil
}
