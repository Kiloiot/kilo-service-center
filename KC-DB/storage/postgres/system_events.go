package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// SystemEventStore provides system event storage operations per MIOTY requirements
type SystemEventStore struct {
	log   logger.Logger
	clock clock.Clock
	db    *sql.DB
}

// Ensure SystemEventStore implements the interface
var _ interfaces.SystemEventStore = (*SystemEventStore)(nil)

// NewSystemEventStore creates a new system event store
func NewSystemEventStore(db *sql.DB, clk clock.Clock, log logger.Logger) *SystemEventStore {
	return &SystemEventStore{
		log: log, clock: clk, db: db}
}

// eventExecer runs the event insert on the pool or inside a transaction.
type eventExecer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// insertEvent is the unified insert helper that writes all 20 columns.
// All event creation paths delegate here.
func (s *SystemEventStore) insertEvent(ctx context.Context, event *models.SystemEvent) error {
	return s.insertEventWith(ctx, s.db, event)
}

// insertEventWith inserts the event through exec, so a caller's transaction
// can hold it.
func (s *SystemEventStore) insertEventWith(ctx context.Context, exec eventExecer, event *models.SystemEvent) error {
	// Generate UUID if not set
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// Set timestamps if zero
	now := s.clock.Now()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	if event.UpdatedAt.IsZero() {
		event.UpdatedAt = now
	}

	// Convert Details JSON — lib/pq sends []byte as binary format which JSONB rejects,
	// so convert to *string for text format.
	var dataJSON interface{}
	if len(event.Details) > 0 {
		str := string(canonicalEventDetails(event.Details))
		dataJSON = &str
	} else {
		empty := "{}"
		dataJSON = &empty
	}

	// Every event is filed under an existing tenant; server-level events use the platform tenant.
	if event.TenantID == "" {
		return errTextTenantIDRequired
	}
	tenantIDInt, err := strconv.ParseInt(event.TenantID, 10, 64)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapInvalidTenantIDFormat, err)
	}

	// Handle SourceID — pass nil if not set
	var sourceIDParam interface{}
	if event.SourceID != nil {
		sourceIDParam = event.SourceID.String()
	}

	// Handle tags — convert []string to hstore-compatible format
	var tagsParam interface{}
	if len(event.Tags) > 0 {
		tagsParam = pq.Array(event.Tags)
	}

	if event.Status == "" {
		event.Status = models.EventStatusNew
	}

	query := `
		INSERT INTO system_events (
			id, tenant_id, event_type, event_category, severity,
			source_type, source_id, source_name, event_code,
			title, description, endpoint_id, basestation_id,
			message_id, user_id, data, tags, occurred_at, recorded_at, status
		) VALUES (
			$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)`

	_, err = exec.ExecContext(
		ctx, query,
		event.ID,
		tenantIDInt,
		event.EventType,
		event.Category,
		event.Severity,
		event.SourceType,
		sourceIDParam,
		event.SourceName,
		event.EventCode,
		event.Title,
		event.Description,
		event.EndpointID,
		event.BasestationID,
		event.MessageID,
		event.UserID,
		dataJSON,
		tagsParam,
		event.CreatedAt,
		event.UpdatedAt,
		event.Status,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapInsertSystemEvent, err)
	}

	return nil
}

// CreateEvent creates a new system event per MIOTY requirements.
// Delegates to the unified insertEvent helper.
func (s *SystemEventStore) CreateEvent(ctx context.Context, event *models.SystemEvent) error {
	// Use SourceName if provided, otherwise fall back to SourceID
	if event.SourceName == "" && event.SourceID != nil {
		event.SourceName = event.SourceID.String()
	}

	return s.insertEvent(ctx, event)
}

// appendDeviceScope narrows the query to a base station / endpoint: an event
// belongs to the device when its FK names it, when the device is its source,
// or when its details name the device (a detach propagate is filed under the
// endpoint and names the station). OR, not AND: most device events carry only
// one of these.
func appendDeviceScope(query string, args []interface{}, argNum int, filter models.SystemEventFilter) (string, []interface{}, int) {
	query, args, argNum = appendOneDeviceScope(query, args, argNum,
		deviceScope{idColumn: "basestation_id", id: filter.BaseStationID, eui: filter.BaseStationEUI, detailKey: models.EventDetailKeyBsEui})
	return appendOneDeviceScope(query, args, argNum,
		deviceScope{idColumn: "endpoint_id", id: filter.EndpointID, eui: filter.EndpointEUI, detailKey: models.EventDetailKeyEpEui})
}

// deviceScope is how events name one device.
type deviceScope struct {
	idColumn  string
	id        *int64
	eui       string
	detailKey string
}

func appendOneDeviceScope(query string, args []interface{}, argNum int, scope deviceScope) (string, []interface{}, int) {
	var matches []string
	if scope.id != nil {
		matches = append(matches, fmt.Sprintf("%s = $%d", scope.idColumn, argNum))
		args = append(args, *scope.id)
		argNum++
	}
	if scope.eui != "" {
		matches = append(matches, fmt.Sprintf("LOWER(source_name) = LOWER($%d)", argNum))
		args = append(args, scope.eui)
		argNum++
	}
	if canonical, ok := canonicalEventEUI(scope.eui); ok {
		matches = append(matches, fmt.Sprintf("data @> jsonb_build_object('%s', $%d::text)", scope.detailKey, argNum))
		args = append(args, canonical)
		argNum++
	}
	if len(matches) == 0 {
		return query, args, argNum
	}
	return query + " AND (" + strings.Join(matches, " OR ") + ")", args, argNum
}

// appendEventSearch adds the free-text and opId predicates shared by the
// listing and the count.
func appendEventSearch(query string, args []interface{}, argNum int, filter models.SystemEventFilter) (string, []interface{}, int) {
	if filter.SearchText != "" {
		query += fmt.Sprintf(" AND searchable_text @@ plainto_tsquery($%d)", argNum)
		args = append(args, filter.SearchText)
		argNum++
	}
	if filter.OpID != nil {
		query += fmt.Sprintf(" AND data @> jsonb_build_object('opId', $%d::bigint)", argNum)
		args = append(args, *filter.OpID)
		argNum++
	}
	return query, args, argNum
}

// sqlActorEmail resolves an event's acting user to an email only when that
// user belongs or belonged to an organization of the event's own tenant, so a
// listing never reveals the email of another tenant's user.
const sqlActorEmail = `COALESCE((
		SELECT u.email FROM users u
		JOIN organization_members om ON om.user_id = u.id
		JOIN organizations o ON o.org_id = om.org_id
		WHERE u.id::text = system_events.user_id AND o.tenant_id = system_events.tenant_id
		LIMIT 1), '')`

// sqlSelectEvents is the event listing's projection; filters append to its WHERE.
const sqlSelectEvents = `SELECT id, tenant_id, event_type, event_category, severity,
		source_type, source_id, source_name, user_id, ` + sqlActorEmail + `, title, description, data,
		status, occurred_at, recorded_at, stored_at
		FROM system_events WHERE 1=1`

// GetEvents retrieves events with filters
func (s *SystemEventStore) GetEvents(ctx context.Context, filter models.SystemEventFilter) ([]*models.SystemEvent, error) {
	where, args, err := eventFilterWhere(filter)
	if err != nil {
		return nil, err
	}
	column, direction, err := eventOrdering(filter)
	if err != nil {
		return nil, err
	}
	var limit interface{}
	if filter.Limit > 0 {
		limit = filter.Limit
	}
	query := sqlSelectEvents
	query += where
	query += fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", eventOrderClause(column, direction), len(args)+1, len(args)+2)
	args = append(args, limit, max(filter.Offset, 0))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapQueryEvents, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.log.Warn(logMsgCloseRowsGetEvents, logger.FieldError, err)
		}
	}()

	var events []*models.SystemEvent
	for rows.Next() {
		var e models.SystemEvent
		var tenantID int64
		var dataJSON []byte
		var sourceID sql.NullString
		var userID sql.NullString
		var status sql.NullString

		err := rows.Scan(
			&e.ID,
			&tenantID,
			&e.EventType,
			&e.Category,
			&e.Severity,
			&e.SourceType,
			&sourceID,
			&e.SourceName,
			&userID,
			&e.UserEmail,
			&e.Title,
			&e.Description,
			&dataJSON,
			&status,
			&e.CreatedAt,
			&e.UpdatedAt,
			&e.StoredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanEvent, err)
		}

		// Convert tenantID int64 to string for model
		e.TenantID = strconv.FormatInt(tenantID, 10)
		e.UserID = userID.String

		// Handle nullable source_id (UUID column)
		if sourceID.Valid {
			if parsed, err := uuid.Parse(sourceID.String); err == nil {
				e.SourceID = &parsed
			}
		}

		// Handle nullable status
		if status.Valid {
			e.Status = status.String
		}

		// Set Details from JSONB data column
		if len(dataJSON) > 0 {
			e.Details = json.RawMessage(dataJSON)
		}

		events = append(events, &e)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapRowsIterationError, err)
	}

	return events, nil
}

// GetEventStats retrieves event statistics grouped by severity for a tenant.
func (s *SystemEventStore) GetEventStats(ctx context.Context, tenantID string, since time.Time) (*models.SystemEventStats, error) {
	if tenantID == "" {
		return nil, errTextTenantIDRequired
	}

	tenantIDInt, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapInvalidTenantIDFormat, err)
	}

	stats := &models.SystemEventStats{
		TenantID:         tenantID,
		EventsByType:     make(map[string]int64),
		EventsBySeverity: make(map[string]int64),
		EventsByStatus:   make(map[string]int64),
	}

	// Aggregate by severity
	sevQuery := `SELECT severity, COUNT(*) FROM system_events
		WHERE tenant_id = $1 AND occurred_at > $2
		GROUP BY severity`
	sevRows, err := s.db.QueryContext(ctx, sevQuery, tenantIDInt, since)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetEventStatsBySeverity, err)
	}
	defer func() {
		if err := sevRows.Close(); err != nil {
			s.log.Warn(logMsgCloseRowsEventStats, logger.FieldError, err)
		}
	}()
	for sevRows.Next() {
		var severity string
		var count int64
		if err := sevRows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanSeverityStat, err)
		}
		stats.EventsBySeverity[severity] = count
		stats.TotalEvents += count
	}
	if err := sevRows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateSeverityStats, err)
	}

	// Aggregate by status
	statusQuery := `SELECT status, COUNT(*) FROM system_events
		WHERE tenant_id = $1 AND occurred_at > $2
		GROUP BY status`
	statusRows, err := s.db.QueryContext(ctx, statusQuery, tenantIDInt, since)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetEventStatsByStatus, err)
	}
	defer func() {
		if err := statusRows.Close(); err != nil {
			s.log.Warn(logMsgCloseRowsEventStats, logger.FieldError, err)
		}
	}()
	for statusRows.Next() {
		var eventStatus string
		var count int64
		if err := statusRows.Scan(&eventStatus, &count); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanStatusStat, err)
		}
		stats.EventsByStatus[eventStatus] = count
		switch eventStatus {
		case models.EventStatusNew:
			stats.NewEvents = count
		case models.EventStatusAcknowledged:
			stats.AcknowledgedEvents = count
		case models.EventStatusResolved:
			stats.ResolvedEvents = count
		}
	}
	if err := statusRows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateStatusStats, err)
	}

	return stats, nil
}

// ============================================================================
// SCACI Event Recording Methods
// ============================================================================
// These methods record events for the SCACI (Service Center to Application
// Center Interface) protocol per MIOTY SCACI v1.0.0 specification.

// CountEvents returns total count matching filter (for pagination)
func (s *SystemEventStore) CountEvents(ctx context.Context, filter models.SystemEventFilter) (int64, error) {
	where, args, err := eventFilterWhere(filter)
	if err != nil {
		return 0, err
	}
	var count int64
	if err := s.db.QueryRowContext(ctx, sqlCountEvents+where, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapCountEvents, err)
	}
	return count, nil
}

// The grouped branches are rendered once and wrapped by these templates.
const (
	errorGroupCountQueryFmt = "WITH failure_groups AS (%s) SELECT COUNT(*) FROM failure_groups"
	errorGroupListQueryFmt  = "WITH failure_groups AS (%s) SELECT event_type, code, message, source_name, first_seen, last_seen, occurrences, last_op_id FROM failure_groups ORDER BY last_seen DESC LIMIT $%d OFFSET $%d"
)

// ListErrorGroups groups the failures of the filter by event type, code and
// subject; SCACI operation failures join the result under their command.
func (s *SystemEventStore) ListErrorGroups(ctx context.Context, filter models.ErrorGroupFilter) ([]*models.EventErrorGroup, int64, error) {
	branches, args := errorGroupBranches(filter)
	var total int64
	// nolint:gosec // G201: branches is static SQL with bound parameter placeholders, not user input
	countQuery := fmt.Sprintf(errorGroupCountQueryFmt, branches)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapCountErrorGroups, err)
	}
	// nolint:gosec // G201: branches is static SQL with bound parameter placeholders, not user input
	listQuery := fmt.Sprintf(errorGroupListQueryFmt, branches, len(args)+1, len(args)+2)
	args = append(args, filter.Limit, filter.Offset)
	rows, err := s.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapQueryErrorGroups, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.log.Warn(logMsgCloseRowsErrorGroups, logger.FieldError, err)
		}
	}()
	groups := []*models.EventErrorGroup{}
	for rows.Next() {
		var g models.EventErrorGroup
		var code, message, sourceName, lastOpID sql.NullString
		if err := rows.Scan(&g.EventType, &code, &message, &sourceName, &g.FirstSeen, &g.LastSeen, &g.Count, &lastOpID); err != nil {
			return nil, 0, fmt.Errorf("%s: %w", errWrapScanErrorGroup, err)
		}
		g.Code, g.Message, g.SourceName, g.LastOpID = code.String, message.String, sourceName.String, lastOpID.String
		groups = append(groups, &g)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapRowsIterationError, err)
	}
	return groups, total, nil
}

// errorGroupBranches renders the grouped system_events branch and, when asked,
// the grouped scaci_operation_log branch, sharing one argument list.
func errorGroupBranches(filter models.ErrorGroupFilter) (string, []interface{}) {
	args := []interface{}{filter.TenantID, filter.From, filter.To, pq.Array(filter.Categories), pq.Array(filter.Severities)}
	eventsWhere := "tenant_id = $1 AND occurred_at >= $2 AND occurred_at <= $3 AND event_category = ANY($4) AND severity = ANY($5)"
	if len(filter.EventTypePrefixes) > 0 {
		args = append(args, pq.Array(filter.EventTypePrefixes))
		eventsWhere += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM unnest($%d::text[]) AS prefix WHERE event_type LIKE prefix || '%%')", len(args))
	}
	branches := `
		SELECT
			event_type,
			COALESCE(event_code, '') AS code,
			(array_agg(description ORDER BY occurred_at DESC))[1] AS message,
			COALESCE(source_name, '') AS source_name,
			MIN(occurred_at) AS first_seen,
			MAX(occurred_at) AS last_seen,
			COUNT(*) AS occurrences,
			(array_agg(data->>'opId' ORDER BY occurred_at DESC))[1] AS last_op_id
		FROM system_events
		WHERE ` + eventsWhere + `
		GROUP BY event_type, event_code, source_name`
	if filter.IncludeSCACIFailures {
		args = append(args, models.OperationStateFailed)
		branches += fmt.Sprintf(`
		UNION ALL
		SELECT
			command AS event_type,
			COALESCE(error_code::text, '') AS code,
			(array_agg(error_message ORDER BY initiated_at DESC))[1] AS message,
			'' AS source_name,
			MIN(initiated_at) AS first_seen,
			MAX(initiated_at) AS last_seen,
			COUNT(*) AS occurrences,
			(array_agg(op_id::text ORDER BY initiated_at DESC))[1] AS last_op_id
		FROM scaci_operation_log
		WHERE tenant_id = $1 AND initiated_at >= $2 AND initiated_at <= $3 AND state = $%d
		GROUP BY command, error_code, error_token`, len(args))
	}
	return branches, args
}
