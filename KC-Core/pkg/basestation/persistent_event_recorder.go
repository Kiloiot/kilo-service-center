package basestation

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// SystemEventWriter covers the event persistence the recorder performs.
// Satisfied structurally by the KC-DB system event store.
type SystemEventWriter interface {
	CreateEvent(ctx context.Context, event *models.SystemEvent) error
}

const eventKeyBsEui = models.EventDetailKeyBsEui

// PersistentEventRecorder implements EventRecorder and persists events to database
type PersistentEventRecorder struct {
	logger     logger.Logger
	eventStore SystemEventWriter
	tenantID   string
}

// NewPersistentEventRecorder creates a new persistent event recorder
func NewPersistentEventRecorder(log logger.Logger, eventStore SystemEventWriter, tenantID string) *PersistentEventRecorder {
	return &PersistentEventRecorder{
		logger:     log,
		eventStore: eventStore,
		tenantID:   tenantID,
	}
}

// RecordEvent persists a base station event that occurred at occurredAt for
// the tenant of ctx, naming the acting user of ctx; the store stamps when it
// was recorded. A failed write is logged, never returned, so the action the
// event records is not undone by its audit trail.
func (r *PersistentEventRecorder) RecordEvent(ctx context.Context, eui [8]byte, eventType string, occurredAt time.Time, data map[string]interface{}) error {
	euiStr := mioty.FormatEUIBytes(eui[:])
	text := describeEvent(eventType, euiStr, data)
	r.logger.InfoContext(ctx, LogBaseStationEvent,
		logger.FieldEui, euiStr,
		logger.FieldEventTypeSnake, eventType,
		logger.FieldSeverity, text.severity,
		logger.FieldData, data,
	)

	details, err := json.Marshal(withStationEUI(data, euiStr))
	if err != nil {
		r.logger.ErrorContext(ctx, LogFailedToPersistEvent, logger.FieldEui, euiStr, logger.FieldError, err)
		return nil
	}
	event := &models.SystemEvent{
		TenantID:    r.tenantOf(ctx),
		EventType:   eventType,
		Category:    models.EventCategoryBaseStation,
		Severity:    text.severity,
		Title:       text.title,
		Description: text.description,
		SourceType:  models.SourceTypeBaseStation,
		SourceName:  euiStr,
		UserID:      audit.ActingUser(ctx),
		Details:     details,
		CreatedAt:   occurredAt,
	}
	if err := r.eventStore.CreateEvent(ctx, event); err != nil {
		r.logger.ErrorContext(ctx, LogFailedToPersistEvent, logger.FieldEui, euiStr, logger.FieldError, err)
	}
	return nil
}

// tenantOf is the session- or request-scoped tenant of ctx, else the configured default.
func (r *PersistentEventRecorder) tenantOf(ctx context.Context) string {
	if tid, err := pkgcontext.GetTenantID(ctx); err == nil && tid > 0 {
		return strconv.FormatInt(tid, 10)
	}
	return r.tenantID
}

// withStationEUI copies data and names the station in it, so the station's
// activity feed and live updates find the event by its details.
func withStationEUI(data map[string]interface{}, euiStr string) map[string]interface{} {
	details := make(map[string]interface{}, len(data)+1)
	for k, v := range data {
		details[k] = v
	}
	details[eventKeyBsEui] = euiStr
	return details
}
