// Package audit provides the transport-neutral audit event surface: the
// event-writer port, the emitter that renders audit SystemEvents and the
// recorder that reports an audit event that could not be written.
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strconv"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// EventWriter writes system events to the event store.
type EventWriter interface {
	CreateEvent(ctx context.Context, event *models.SystemEvent) error
}

// ErrEmitterNotConfigured reports an audit emission with no event store behind it.
var ErrEmitterNotConfigured = errors.New("audit emitter not configured")

// ErrTenantRequired reports an audit event that names no tenant to file it under.
var ErrTenantRequired = errors.New("audit event: tenant is required")

// Construction faults of the emitter, the recorder and the drop counter.
var (
	ErrNilEventWriter = errors.New("audit emitter: event writer is nil")
	ErrNilClock       = errors.New("audit emitter: clock is nil")
	ErrNilEmitter     = errors.New("audit recorder: emitter is nil")
	ErrNilLogger      = errors.New("audit recorder: logger is nil")
	ErrNilDropCounter = errors.New("audit recorder: drop counter is nil")
	ErrNilRegisterer  = errors.New("audit drop counter: registerer is nil")
	ErrNilRecorder    = errors.New("audit recorder is nil")
)

const (
	errWrapMarshalDetails = "marshal audit details"
	errWrapWriteEvent     = "write audit event"
)

// Event carries the varying fields of an audit SystemEvent. Category defaults
// to the audit category and SourceType to the API. TenantID is required:
// server-level events are filed under the platform tenant. EndpointID and
// BaseStationID name the event's subject.
type Event struct {
	TenantID      int64
	SourceID      *uuid.UUID
	SourceType    string
	Category      string
	EventType     string
	Title         string
	Description   string
	SourceName    string
	UserID        string
	EndpointID    *int64
	BaseStationID *int64
	Details       map[string]any
}

// removalEvents are written after their subject row is deleted, so no foreign key can reference it.
var removalEvents = map[string]bool{
	models.EventTypeEndpointDeleted: true,
	models.EventTypeBSDeregistered:  true,
}

// Emitter writes audit SystemEvents at Severity=Info, filed under the
// category the caller names.
type Emitter struct {
	writer EventWriter
	clock  clock.Clock
}

// NewEmitter returns an Emitter backed by w, stamping events from clk; both
// are required, so a wiring fault surfaces at startup instead of on the first
// audited action.
func NewEmitter(w EventWriter, clk clock.Clock) (*Emitter, error) {
	if w == nil {
		return nil, ErrNilEventWriter
	}
	if clk == nil {
		return nil, ErrNilClock
	}
	return &Emitter{writer: w, clock: clk}, nil
}

// EmitAudit writes ev as an audit event and reports any failure to the caller.
func (e *Emitter) EmitAudit(ctx context.Context, ev Event) error {
	if e == nil || e.writer == nil {
		return ErrEmitterNotConfigured
	}
	if ev.TenantID <= 0 {
		return ErrTenantRequired
	}
	details, endpointID, baseStationID := subjectLinks(ev)
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapMarshalDetails, err)
	}
	sourceType := ev.SourceType
	if sourceType == "" {
		sourceType = models.SourceTypeAPI
	}
	category := ev.Category
	if category == "" {
		category = models.EventCategoryAudit
	}
	now := e.clock.Now()
	event := &models.SystemEvent{
		TenantID:      strconv.FormatInt(ev.TenantID, 10),
		EventType:     ev.EventType,
		Category:      category,
		Severity:      models.EventSeverityInfo,
		Title:         ev.Title,
		Description:   ev.Description,
		SourceType:    sourceType,
		SourceID:      ev.SourceID,
		SourceName:    ev.SourceName,
		EndpointID:    endpointID,
		BasestationID: baseStationID,
		UserID:        ev.UserID,
		Details:       detailsJSON,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := e.writer.CreateEvent(ctx, event); err != nil {
		return fmt.Errorf("%s: %w", errWrapWriteEvent, err)
	}
	return nil
}

// subjectLinks returns the details and the foreign keys an event is stored
// with; a removal event keeps its subject's ids in the details instead.
func subjectLinks(ev Event) (map[string]any, *int64, *int64) {
	if !removalEvents[ev.EventType] {
		return ev.Details, ev.EndpointID, ev.BaseStationID
	}
	details := maps.Clone(ev.Details)
	if details == nil {
		details = map[string]any{}
	}
	if ev.EndpointID != nil {
		details[models.EventDetailKeyEndpointID] = *ev.EndpointID
	}
	if ev.BaseStationID != nil {
		details[models.EventDetailKeyBaseStationID] = *ev.BaseStationID
	}
	return details, nil, nil
}
