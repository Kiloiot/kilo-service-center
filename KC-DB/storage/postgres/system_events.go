package postgres

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// SystemEventStore provides system event storage operations per MIOTY requirements
type SystemEventStore struct {
	db *sql.DB
}

// Ensure SystemEventStore implements the interface
var _ interfaces.SystemEventStore = (*SystemEventStore)(nil)

// NewSystemEventStore creates a new system event store
func NewSystemEventStore(db *sql.DB) *SystemEventStore {
	return &SystemEventStore{db: db}
}

// insertEvent is the unified insert helper that writes all 19 columns.
// All event creation paths (CreateEvent, recordBSSCIEvent, etc.) delegate here.
func (s *SystemEventStore) insertEvent(ctx context.Context, event *models.SystemEvent) error {
	// Generate UUID if not set
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// Set timestamps if zero
	now := time.Now()
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
		str := string(event.Details)
		dataJSON = &str
	} else {
		empty := "{}"
		dataJSON = &empty
	}

	// Convert string TenantID to int64 for BIGINT column
	var tenantIDInt int64
	if event.TenantID != "" {
		var err error
		tenantIDInt, err = strconv.ParseInt(event.TenantID, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid tenant ID format: %w", err)
		}
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

	query := `
		INSERT INTO system_events (
			id, tenant_id, event_type, event_category, severity,
			source_type, source_id, source_name, event_code,
			title, description, endpoint_id, basestation_id,
			message_id, user_id, data, tags, occurred_at, recorded_at
		) VALUES (
			$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		)`

	_, err := s.db.ExecContext(ctx, query,
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
	)
	if err != nil {
		return fmt.Errorf("failed to insert system event: %w", err)
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

// recordBSSCIEvent records a BSSCI protocol event per MIOTY BS-SC Interface v1.0.0.
// Accepts models.SystemEvent and a details map for JSONB data.
func (s *SystemEventStore) recordBSSCIEvent(ctx context.Context, event *models.SystemEvent, details map[string]interface{}) error {
	// Validate event category against centralized definitions
	if !models.ValidEventCategories[event.Category] {
		event.Category = models.EventCategoryProtocol
	}

	// Convert EUI from byte array to hex string if present in details
	if euiBytes, ok := details["epEui"].([8]byte); ok {
		details["epEui"] = hex.EncodeToString(euiBytes[:])
	}
	if euiBytes, ok := details["bsEui"].([8]byte); ok {
		details["bsEui"] = hex.EncodeToString(euiBytes[:])
	}

	// Marshal details map to json.RawMessage
	dataJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}
	event.Details = json.RawMessage(dataJSON)

	return s.insertEvent(ctx, event)
}

// appendDeviceScope narrows the query to a base station / endpoint: match by the FK
// when present, else fall back to the source_name EUI. OR, not AND — attach/detach and
// similar events carry only source_name, not the FK, so an AND would drop them.
func appendDeviceScope(query string, args []interface{}, argNum int, filter interfaces.SystemEventFilter) (string, []interface{}, int) {
	if filter.BaseStationID != nil && filter.BaseStationEUI != "" {
		query += fmt.Sprintf(" AND (basestation_id = $%d OR LOWER(source_name) = LOWER($%d))", argNum, argNum+1)
		args = append(args, *filter.BaseStationID, filter.BaseStationEUI)
		argNum += 2
	} else if filter.BaseStationID != nil {
		query += fmt.Sprintf(" AND basestation_id = $%d", argNum)
		args = append(args, *filter.BaseStationID)
		argNum++
	} else if filter.BaseStationEUI != "" {
		query += fmt.Sprintf(" AND LOWER(source_name) = LOWER($%d)", argNum)
		args = append(args, filter.BaseStationEUI)
		argNum++
	}
	if filter.EndpointID != nil && filter.EndpointEUI != "" {
		query += fmt.Sprintf(" AND (endpoint_id = $%d OR LOWER(source_name) = LOWER($%d))", argNum, argNum+1)
		args = append(args, *filter.EndpointID, filter.EndpointEUI)
		argNum += 2
	} else if filter.EndpointID != nil {
		query += fmt.Sprintf(" AND endpoint_id = $%d", argNum)
		args = append(args, *filter.EndpointID)
		argNum++
	} else if filter.EndpointEUI != "" {
		query += fmt.Sprintf(" AND LOWER(source_name) = LOWER($%d)", argNum)
		args = append(args, filter.EndpointEUI)
		argNum++
	}
	return query, args, argNum
}

// GetEvents retrieves events with filters
func (s *SystemEventStore) GetEvents(ctx context.Context, filter interfaces.SystemEventFilter) ([]*models.SystemEvent, error) {
	if filter.TenantID == "" {
		return nil, fmt.Errorf("tenant ID required")
	}

	query := `SELECT id, tenant_id, event_type, event_category, severity,
		source_type, source_id, source_name, title, description, data,
		status, occurred_at, recorded_at
		FROM system_events WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	// Convert string TenantID to int64 for BIGINT column
	if filter.TenantID != "" {
		tenantIDInt, err := strconv.ParseInt(filter.TenantID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid tenant ID format: %w", err)
		}
		query += fmt.Sprintf(" AND tenant_id = $%d", argNum)
		args = append(args, tenantIDInt)
		argNum++
	}
	if filter.Since != nil {
		query += fmt.Sprintf(" AND occurred_at > $%d", argNum)
		args = append(args, *filter.Since)
		argNum++
	}
	if filter.Until != nil {
		query += fmt.Sprintf(" AND occurred_at < $%d", argNum)
		args = append(args, *filter.Until)
		argNum++
	}
	if len(filter.Categories) > 0 {
		query += fmt.Sprintf(" AND event_category = ANY($%d)", argNum)
		args = append(args, pq.Array(filter.Categories))
		argNum++
	}
	if len(filter.Severities) > 0 {
		query += fmt.Sprintf(" AND severity = ANY($%d)", argNum)
		args = append(args, pq.Array(filter.Severities))
		argNum++
	}
	if len(filter.EventTypes) > 0 {
		query += fmt.Sprintf(" AND event_type = ANY($%d)", argNum)
		args = append(args, pq.Array(filter.EventTypes))
		argNum++
	}
	if len(filter.SourceTypes) > 0 {
		query += fmt.Sprintf(" AND source_type = ANY($%d)", argNum)
		args = append(args, pq.Array(filter.SourceTypes))
		argNum++
	}
	if len(filter.Status) > 0 {
		query += fmt.Sprintf(" AND status = ANY($%d)", argNum)
		args = append(args, pq.Array(filter.Status))
		argNum++
	}
	if filter.SourceID != nil {
		query += fmt.Sprintf(" AND source_id = $%d", argNum)
		args = append(args, *filter.SourceID)
		argNum++
	}

	query, args, argNum = appendDeviceScope(query, args, argNum, filter)

	// Order by occurred_at DESC (most recent first) unless specified otherwise
	orderBy := "occurred_at"
	orderDir := "DESC"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}
	if filter.OrderDirection != "" {
		orderDir = filter.OrderDirection
	}
	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, filter.Limit)
		argNum++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows in GetEvents: %v", err)
		}
	}()

	var events []*models.SystemEvent
	for rows.Next() {
		var e models.SystemEvent
		var tenantID int64
		var dataJSON []byte
		var sourceID sql.NullString
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
			&e.Title,
			&e.Description,
			&dataJSON,
			&status,
			&e.CreatedAt,
			&e.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}

		// Convert tenantID int64 to string for model
		e.TenantID = strconv.FormatInt(tenantID, 10)

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
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return events, nil
}

// GetActiveAlerts retrieves unresolved events with alert-level severity.
func (s *SystemEventStore) GetActiveAlerts(ctx context.Context, filter interfaces.AlertFilter) ([]*models.SystemEvent, error) {
	if filter.TenantID == "" {
		return nil, fmt.Errorf("tenant ID required")
	}

	query := `SELECT id, tenant_id, event_type, event_category, severity,
		source_type, source_id, source_name, title, description, data,
		status, occurred_at, recorded_at
		FROM system_events
		WHERE (status IS NULL OR status != 'resolved')`
	args := []interface{}{}
	argNum := 1

	tenantIDInt, err := strconv.ParseInt(filter.TenantID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID format: %w", err)
	}
	query += fmt.Sprintf(" AND tenant_id = $%d", argNum)
	args = append(args, tenantIDInt)
	argNum++

	if len(filter.Severities) > 0 {
		query += fmt.Sprintf(" AND severity = ANY($%d)", argNum)
		args = append(args, pq.Array(filter.Severities))
		argNum++
	} else {
		query += " AND severity IN ('warning', 'error', 'critical')"
	}
	if len(filter.Categories) > 0 {
		query += fmt.Sprintf(" AND event_category = ANY($%d)", argNum)
		args = append(args, pq.Array(filter.Categories))
		argNum++
	}
	if filter.Acknowledged != nil {
		if *filter.Acknowledged {
			query += " AND status = 'acknowledged'"
		} else {
			query += " AND (status IS NULL OR status != 'acknowledged')"
		}
	}
	if filter.Since != nil {
		query += fmt.Sprintf(" AND occurred_at > $%d", argNum)
		args = append(args, *filter.Since)
		argNum++
	}

	query += " ORDER BY occurred_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, filter.Limit)
		argNum++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get active alerts: %w", err)
	}
	defer func() { _ = rows.Close() }()

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
			return nil, fmt.Errorf("scan active alert: %w", err)
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
		return nil, fmt.Errorf("iterate active alerts: %w", err)
	}

	return events, nil
}

// GetEventStats retrieves event statistics grouped by severity for a tenant.
func (s *SystemEventStore) GetEventStats(ctx context.Context, tenantID string, since time.Time) (*models.SystemEventStats, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID required")
	}

	tenantIDInt, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID format: %w", err)
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
		return nil, fmt.Errorf("get event stats by severity: %w", err)
	}
	defer func() { _ = sevRows.Close() }()
	for sevRows.Next() {
		var severity string
		var count int64
		if err := sevRows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("scan severity stat: %w", err)
		}
		stats.EventsBySeverity[severity] = count
		stats.TotalEvents += count
	}
	if err := sevRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate severity stats: %w", err)
	}

	// Aggregate by status
	statusQuery := `SELECT COALESCE(status, 'new'), COUNT(*) FROM system_events
		WHERE tenant_id = $1 AND occurred_at > $2
		GROUP BY COALESCE(status, 'new')`
	statusRows, err := s.db.QueryContext(ctx, statusQuery, tenantIDInt, since)
	if err != nil {
		return nil, fmt.Errorf("get event stats by status: %w", err)
	}
	defer func() { _ = statusRows.Close() }()
	for statusRows.Next() {
		var eventStatus string
		var count int64
		if err := statusRows.Scan(&eventStatus, &count); err != nil {
			return nil, fmt.Errorf("scan status stat: %w", err)
		}
		stats.EventsByStatus[eventStatus] = count
		switch eventStatus {
		case "new":
			stats.NewEvents = count
		case "acknowledged":
			stats.AcknowledgedEvents = count
		case "resolved":
			stats.ResolvedEvents = count
		}
	}
	if err := statusRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate status stats: %w", err)
	}

	return stats, nil
}

// ============================================================================
// SCACI Event Recording Methods
// ============================================================================
// These methods record events for the SCACI (Service Center to Application
// Center Interface) protocol per MIOTY SCACI v1.0.0 specification.

// RecordSCACIError records a SCACI protocol error event
func (s *SystemEventStore) RecordSCACIError(ctx context.Context, tenantID int64, sessionID int64, command string, opId int64, errorCode int, errorMsg string) error {
	event := &models.SystemEvent{
		TenantID:    strconv.FormatInt(tenantID, 10),
		EventType:   scaci.EventTypeSCACIError,
		Category:    models.EventCategoryError,
		Severity:    models.EventSeverityError,
		SourceType:  models.SourceTypeServiceCenter,
		EventCode:   fmt.Sprintf("%d", errorCode),
		Title:       fmt.Sprintf(models.EventTitleSCACIError, command, opId),
		Description: fmt.Sprintf(models.EventDescriptionSCACIError, command, errorCode, errorMsg),
	}

	details := map[string]interface{}{
		"sessionID": sessionID,
		"command":   command,
		"opId":      opId,
		"errorCode": errorCode,
		"errorMsg":  errorMsg,
		"time":      time.Now().UnixNano(),
	}

	return s.recordBSSCIEvent(ctx, event, details)
}

// ListSCACIEvents retrieves SCACI-category events with optional filters.
// sessionID filtering uses bigint cast for index-friendly queries.
func (s *SystemEventStore) ListSCACIEvents(ctx context.Context, tenantID int64, sessionID *int64, eventType string, limit, offset int) ([]*models.SystemEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, tenant_id, event_type, event_category, severity,
			source_type, source_name, title, description,
			data, occurred_at, recorded_at
		FROM system_events
		WHERE event_category = 'scaci'
		  AND tenant_id = $1`

	args := []interface{}{tenantID}
	argIndex := 2

	if sessionID != nil {
		query += fmt.Sprintf(" AND (data->>'sessionID')::bigint = $%d", argIndex)
		args = append(args, *sessionID)
		argIndex++
	}

	if eventType != "" {
		query += fmt.Sprintf(" AND event_type = $%d", argIndex)
		args = append(args, eventType)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY occurred_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query SCACI events: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows in SCACI events query: %v", err)
		}
	}()

	var events []*models.SystemEvent
	for rows.Next() {
		var event models.SystemEvent
		var tenantIDDB int64
		var dataJSON []byte

		err := rows.Scan(
			&event.ID,
			&tenantIDDB,
			&event.EventType,
			&event.Category,
			&event.Severity,
			&event.SourceType,
			&event.SourceName,
			&event.Title,
			&event.Description,
			&dataJSON,
			&event.CreatedAt,
			&event.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan SCACI event: %w", err)
		}

		event.TenantID = strconv.FormatInt(tenantIDDB, 10)

		if len(dataJSON) > 0 && string(dataJSON) != "{}" {
			event.Details = json.RawMessage(dataJSON)
		}

		events = append(events, &event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return events, nil
}

// CountSCACIEvents counts SCACI events within a time window
func (s *SystemEventStore) CountSCACIEvents(ctx context.Context, tenantID int64, eventType string, lookbackHours int) (int64, error) {
	if lookbackHours <= 0 {
		lookbackHours = 24
	}

	query := `
		SELECT COUNT(*)
		FROM system_events
		WHERE event_category = 'scaci'
		  AND tenant_id = $1
		  AND occurred_at >= NOW() - INTERVAL '1 hour' * $2`

	args := []interface{}{tenantID, lookbackHours}

	if eventType != "" {
		query += " AND event_type = $3"
		args = append(args, eventType)
	}

	var count int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count SCACI events: %w", err)
	}

	return count, nil
}

// CountSCACIEventsByFilter counts SCACI events matching ListSCACIEvents filters.
// Used for accurate pagination metadata.
func (s *SystemEventStore) CountSCACIEventsByFilter(ctx context.Context, tenantID int64, sessionID *int64, eventType string) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM system_events
		WHERE event_category = 'scaci'
		  AND tenant_id = $1`

	args := []interface{}{tenantID}
	argIndex := 2

	if sessionID != nil {
		query += fmt.Sprintf(" AND (data->>'sessionID')::bigint = $%d", argIndex)
		args = append(args, *sessionID)
		argIndex++
	}

	if eventType != "" {
		query += fmt.Sprintf(" AND event_type = $%d", argIndex)
		args = append(args, eventType)
	}

	var count int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count SCACI events by filter: %w", err)
	}

	return count, nil
}

// CountEvents returns total count matching filter (for pagination)
func (s *SystemEventStore) CountEvents(ctx context.Context, filter interfaces.SystemEventFilter) (int64, error) {
	if filter.TenantID == "" {
		return 0, fmt.Errorf("tenant ID required")
	}

	query := `SELECT COUNT(*) FROM system_events WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	// Convert string TenantID to int64 for BIGINT column
	if filter.TenantID != "" {
		tenantIDInt, err := strconv.ParseInt(filter.TenantID, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid tenant ID format: %w", err)
		}
		query += fmt.Sprintf(" AND tenant_id = $%d", argNum)
		args = append(args, tenantIDInt)
		argNum++
	}
	if filter.Since != nil {
		query += fmt.Sprintf(" AND occurred_at > $%d", argNum)
		args = append(args, *filter.Since)
		argNum++
	}
	if filter.Until != nil {
		query += fmt.Sprintf(" AND occurred_at < $%d", argNum)
		args = append(args, *filter.Until)
		argNum++
	}
	if len(filter.Categories) > 0 {
		query += fmt.Sprintf(" AND event_category = ANY($%d)", argNum)
		args = append(args, pq.Array(filter.Categories))
		argNum++
	}
	if len(filter.Severities) > 0 {
		query += fmt.Sprintf(" AND severity = ANY($%d)", argNum)
		args = append(args, pq.Array(filter.Severities))
		argNum++
	}
	if len(filter.EventTypes) > 0 {
		query += fmt.Sprintf(" AND event_type = ANY($%d)", argNum)
		args = append(args, pq.Array(filter.EventTypes))
		argNum++
	}
	if len(filter.SourceTypes) > 0 {
		query += fmt.Sprintf(" AND source_type = ANY($%d)", argNum)
		args = append(args, pq.Array(filter.SourceTypes))
		argNum++
	}
	if len(filter.Status) > 0 {
		query += fmt.Sprintf(" AND status = ANY($%d)", argNum)
		args = append(args, pq.Array(filter.Status))
		argNum++
	}
	if filter.SourceID != nil {
		query += fmt.Sprintf(" AND source_id = $%d", argNum)
		args = append(args, *filter.SourceID)
		argNum++
	}

	query, args, _ = appendDeviceScope(query, args, argNum, filter)

	var count int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count events: %w", err)
	}

	return count, nil
}

// CountActiveAlerts returns total count of active alerts (for pagination)
func (s *SystemEventStore) CountActiveAlerts(ctx context.Context, filter interfaces.AlertFilter) (int64, error) {
	if filter.TenantID == "" {
		return 0, fmt.Errorf("tenant ID required")
	}

	query := `SELECT COUNT(*) FROM system_events
		WHERE (status IS NULL OR status != 'resolved')`
	args := []interface{}{}
	argNum := 1

	tenantIDInt, err := strconv.ParseInt(filter.TenantID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid tenant ID format: %w", err)
	}
	query += fmt.Sprintf(" AND tenant_id = $%d", argNum)
	args = append(args, tenantIDInt)
	argNum++

	if len(filter.Severities) > 0 {
		query += fmt.Sprintf(" AND severity = ANY($%d)", argNum)
		args = append(args, pq.Array(filter.Severities))
		argNum++
	} else {
		query += " AND severity IN ('warning', 'error', 'critical')"
	}
	if len(filter.Categories) > 0 {
		query += fmt.Sprintf(" AND event_category = ANY($%d)", argNum)
		args = append(args, pq.Array(filter.Categories))
		argNum++
	}
	if filter.Acknowledged != nil {
		if *filter.Acknowledged {
			query += " AND status = 'acknowledged'"
		} else {
			query += " AND (status IS NULL OR status != 'acknowledged')"
		}
	}
	if filter.Since != nil {
		query += fmt.Sprintf(" AND occurred_at > $%d", argNum)
		args = append(args, *filter.Since)
	}

	var count int64
	if err = s.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count active alerts: %w", err)
	}

	return count, nil
}
