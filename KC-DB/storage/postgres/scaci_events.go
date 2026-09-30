package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// SCACIEventStore records and reads SCACI protocol events in the system event log.
type SCACIEventStore struct {
	log    logger.Logger
	clock  clock.Clock
	db     *sql.DB
	events *SystemEventStore
}

var _ interfaces.SCACIEventStore = (*SCACIEventStore)(nil)

// NewSCACIEventStore creates a SCACI event store that writes through the system event store.
func NewSCACIEventStore(db *sql.DB, events *SystemEventStore, clk clock.Clock, log logger.Logger) *SCACIEventStore {
	return &SCACIEventStore{
		log: log, clock: clk, db: db, events: events}
}

// RecordSCACIError records a SCACI protocol error event
func (s *SCACIEventStore) RecordSCACIError(ctx context.Context, tenantID int64, sessionID int64, command string, opId int64, errorCode int, errorMsg string) error {
	event := &models.SystemEvent{
		TenantID:    strconv.FormatInt(tenantID, 10),
		EventType:   models.EventTypeSCACIError,
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
		"time":      s.clock.Now().UnixNano(),
	}

	return s.recordProtocolEvent(ctx, event, details)
}

// scaciSessionEventKind is how one application center session event type is
// filed: its severity and how its title and description read.
type scaciSessionEventKind struct {
	severity string
	title    string
	describe func(event *models.SCACISessionEvent, ac string) string
}

func describeSession(format string) func(*models.SCACISessionEvent, string) string {
	return func(event *models.SCACISessionEvent, ac string) string {
		return fmt.Sprintf(format, event.SessionID, ac)
	}
}

var scaciSessionEventKinds = map[string]scaciSessionEventKind{
	models.EventTypeSCACISessionOpened: {
		severity: models.EventSeverityInfo,
		title:    models.EventTitleSCACISessionOpened,
		describe: describeSession(models.EventDescriptionSCACISessionOpened),
	},
	models.EventTypeSCACISessionResumed: {
		severity: models.EventSeverityInfo,
		title:    models.EventTitleSCACISessionResumed,
		describe: describeSession(models.EventDescriptionSCACISessionResumed),
	},
	models.EventTypeSCACISessionClosed: {
		severity: models.EventSeverityWarning,
		title:    models.EventTitleSCACISessionClosed,
		describe: func(event *models.SCACISessionEvent, ac string) string {
			if event.Reason == models.SCACISessionClosedSuperseded {
				return describeSession(models.EventDescriptionSCACISessionSuperseded)(event, ac)
			}
			return describeSession(models.EventDescriptionSCACISessionClosed)(event, ac)
		},
	},
	models.EventTypeSCACIConnectRefused: {
		severity: models.EventSeverityError,
		title:    models.EventTitleSCACIConnectRefused,
		describe: func(event *models.SCACISessionEvent, ac string) string {
			return fmt.Sprintf(models.EventDescriptionSCACIConnectRefused, ac, event.ErrorMessage)
		},
	},
}

// scaciSessionEventCategories are the categories a session event is filed in:
// the tenant's SCACI events, or the administrators' security events.
var scaciSessionEventCategories = map[string]bool{
	models.EventCategorySCACI:    true,
	models.EventCategorySecurity: true,
}

// A refused connect is one record per cause while it is not resolved: the
// transaction locks the cause (tenant, type, category, application center,
// error token), and a recurrence moves the open record to the new time and
// counts it instead of adding a record.
const (
	sqlLockRefusalCause = `SELECT pg_advisory_xact_lock(hashtextextended(concat_ws(chr(31), $1::text, $2::text, $3::text, $4::text, $5::text), 0))`
	sqlCoalesceRefusal  = `UPDATE system_events
		SET occurred_at = $6, recorded_at = $6,
			data = data || jsonb_build_object(
				$8::text, COALESCE((data->>$8::text)::bigint, 1) + 1,
				$9::text, $7::text)
				|| CASE WHEN $11::text <> '' THEN jsonb_build_object($10::text, $11::text) ELSE '{}'::jsonb END
		WHERE tenant_id = $1 AND event_type = $2 AND event_category = $3
			AND source_name = $4 AND event_code = $5 AND status <> $12`
)

// RecordSessionEvent files a step in the lifecycle of an application center
// session (SCACI §1, §3.3) under the event's tenant and category.
func (s *SCACIEventStore) RecordSessionEvent(ctx context.Context, event *models.SCACISessionEvent) error {
	kind, ok := scaciSessionEventKinds[event.EventType]
	if !ok {
		return fmt.Errorf("%w: %s", errTextUnknownSCACISessionEvent, event.EventType)
	}
	if !scaciSessionEventCategories[event.Category] {
		return fmt.Errorf("%w: %s", errTextSCACISessionEventCategory, event.Category)
	}
	ac := event.AcEui
	if ac == "" {
		ac = models.ApplicationCenterUnknown
	}

	details := map[string]interface{}{models.EventDetailKeyAcEui: event.AcEui}
	if event.SessionID > 0 {
		details[models.EventDetailKeySessionID] = event.SessionID
	}
	if event.RemoteAddr != "" {
		details[models.EventDetailKeyRemoteAddr] = event.RemoteAddr
	}
	if event.Reason != "" {
		details[models.EventDetailKeyReason] = event.Reason
	}
	if event.ErrorToken != "" {
		details[models.EventDetailKeyErrorToken] = event.ErrorToken
		details[models.EventDetailKeyMessage] = event.ErrorMessage
	}

	systemEvent := &models.SystemEvent{
		TenantID:    strconv.FormatInt(event.TenantID, 10),
		EventType:   event.EventType,
		Category:    event.Category,
		Severity:    kind.severity,
		SourceType:  models.SourceTypeApplicationCenter,
		SourceName:  event.AcEui,
		EventCode:   event.ErrorToken,
		Title:       fmt.Sprintf(kind.title, ac),
		Description: kind.describe(event, ac),
	}
	if event.EventType == models.EventTypeSCACIConnectRefused {
		return s.recordRefusal(ctx, event, systemEvent, details)
	}
	systemEvent.CreatedAt = event.OccurredAt
	return s.recordProtocolEvent(ctx, systemEvent, details)
}

// recordRefusal counts a recurrence of an open refused connect of the same
// cause, or files the cause's first record.
func (s *SCACIEventStore) recordRefusal(ctx context.Context, event *models.SCACISessionEvent, systemEvent *models.SystemEvent, details map[string]interface{}) error {
	now := s.clock.Now().UTC()
	seen := now.Format(time.RFC3339Nano)
	details[models.EventDetailKeyCount] = 1
	details[models.EventDetailKeyFirstSeen] = seen
	details[models.EventDetailKeyLastSeen] = seen
	dataJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapMarshalEventData, err)
	}
	systemEvent.Details = json.RawMessage(dataJSON)
	systemEvent.CreatedAt, systemEvent.UpdatedAt = now, now

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapBeginTransaction, err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.log.Warn(logMsgRefusalRollback, logger.FieldError, err)
		}
	}()

	cause := []interface{}{event.TenantID, event.EventType, event.Category, event.AcEui, event.ErrorToken}
	if _, err := tx.ExecContext(ctx, sqlLockRefusalCause, cause...); err != nil {
		return fmt.Errorf("%s: %w", errWrapLockRefusalCause, err)
	}
	result, err := tx.ExecContext(ctx, sqlCoalesceRefusal, append(cause,
		now, seen,
		models.EventDetailKeyCount, models.EventDetailKeyLastSeen,
		models.EventDetailKeyRemoteAddr, event.RemoteAddr,
		models.EventStatusResolved)...)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapCoalesceRefusal, err)
	}
	counted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapCoalesceRefusal, err)
	}
	if counted == 0 {
		if err := s.events.insertEventWith(ctx, tx, systemEvent); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", errWrapCommitTransaction, err)
	}
	return nil
}

// ListSCACIEvents retrieves SCACI-category events with optional filters.
// sessionID filtering uses bigint cast for index-friendly queries.
func (s *SCACIEventStore) ListSCACIEvents(ctx context.Context, tenantID int64, sessionID *int64, eventType string, limit, offset int) ([]*models.SystemEvent, error) {
	if limit <= 0 {
		limit = defaultSCACIEventsLimit
	}
	if limit > maxSCACIEventsLimit {
		limit = maxSCACIEventsLimit
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

	query += fmt.Sprintf(" ORDER BY occurred_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1) //nolint:gosec // G202: appends parameter placeholders, values are bound
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapQuerySCACIEvents, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.log.Warn(logMsgCloseRowsSCACIEvents, logger.FieldError, err)
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
			return nil, fmt.Errorf("%s: %w", errWrapScanSCACIEvent, err)
		}

		event.TenantID = strconv.FormatInt(tenantIDDB, 10)

		if len(dataJSON) > 0 && string(dataJSON) != "{}" {
			event.Details = json.RawMessage(dataJSON)
		}

		events = append(events, &event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapRowsIterationError, err)
	}

	return events, nil
}

// CountSCACIEventsByFilter counts SCACI events matching ListSCACIEvents filters.
// Used for accurate pagination metadata.
func (s *SCACIEventStore) CountSCACIEventsByFilter(ctx context.Context, tenantID int64, sessionID *int64, eventType string) (int64, error) {
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
		return 0, fmt.Errorf("%s: %w", errWrapCountSCACIEventsByFilter, err)
	}

	return count, nil
}

// recordProtocolEvent normalizes the detail map and stores the event.
func (s *SCACIEventStore) recordProtocolEvent(ctx context.Context, event *models.SystemEvent, details map[string]interface{}) error {
	// Validate event category against centralized definitions
	if !models.IsValidEventCategory(event.Category) {
		event.Category = models.EventCategoryProtocol
	}

	// Convert EUI from byte array to hex string if present in details
	if euiBytes, ok := details["epEui"].([8]byte); ok {
		details["epEui"] = mioty.FormatEUIBytes(euiBytes[:])
	}
	if euiBytes, ok := details["bsEui"].([8]byte); ok {
		details["bsEui"] = mioty.FormatEUIBytes(euiBytes[:])
	}

	// Marshal details map to json.RawMessage
	dataJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapMarshalEventData, err)
	}
	event.Details = json.RawMessage(dataJSON)

	return s.events.CreateEvent(ctx, event)
}
