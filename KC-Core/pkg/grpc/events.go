package grpc

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// EventWriter writes system events to the event store.
type EventWriter interface {
	CreateEvent(ctx context.Context, event *models.SystemEvent) error
}

// AuditEvent carries the varying fields of an audit SystemEvent.
type AuditEvent struct {
	TenantID    int64
	SourceID    *uuid.UUID
	EventType   string
	Title       string
	Description string
	SourceName  string
	UserID      string
	Details     map[string]any
}

// AuditEmitter writes audit SystemEvents (Category=Audit, Severity=Info, SourceType=API).
type AuditEmitter struct {
	writer EventWriter
}

// NewAuditEmitter returns an AuditEmitter backed by w.
func NewAuditEmitter(w EventWriter) *AuditEmitter {
	return &AuditEmitter{writer: w}
}

// EmitAudit writes ev as an audit event; no-op if the emitter or its writer is nil.
func (e *AuditEmitter) EmitAudit(ctx context.Context, ev AuditEvent) {
	if e == nil || e.writer == nil {
		return
	}
	detailsJSON, _ := json.Marshal(ev.Details)
	_ = e.writer.CreateEvent(ctx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(ev.TenantID, 10),
		EventType:   ev.EventType,
		Category:    models.EventCategoryAudit,
		Severity:    models.EventSeverityInfo,
		Title:       ev.Title,
		Description: ev.Description,
		SourceType:  models.SourceTypeAPI,
		SourceID:    ev.SourceID,
		SourceName:  ev.SourceName,
		UserID:      ev.UserID,
		Details:     detailsJSON,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
}
