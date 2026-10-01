// Package events provides system event storage and retrieval.
package events

import (
	"context"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/streampoll"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// SystemEventStore provides event persistence primitives. Rows and total are
// separate so streaming (which discards the total) can skip the COUNT(*).
type SystemEventStore interface {
	GetEvents(ctx context.Context, tenantID int64, filter *EventFilter, limit, offset int) ([]*models.SystemEvent, error)
	CountEvents(ctx context.Context, tenantID int64, filter *EventFilter) (int64, error)
}

// EUIResolver maps a device EUI to its internal ID for event scoping. A nil
// ID means "unknown", so callers fall back to the source_name EUI filter; an
// error means the lookup itself failed.
type EUIResolver interface {
	ResolveBaseStationID(ctx context.Context, tenantID int64, bsEui []byte) (*int64, error)
	ResolveEndpointID(ctx context.Context, tenantID int64, epEui []byte) (*int64, error)
}

// EventFilter contains filtering criteria for events.
type EventFilter struct {
	Categories []string
	Severity   []string
	EventTypes []string
	StartTime  *time.Time
	EndTime    *time.Time

	// Scoping for base-station / endpoint views, set after EUI resolution.
	BaseStationID  *int64
	EndpointID     *int64
	BaseStationEUI string
	EndpointEUI    string

	OpID   *int64
	Search string

	// StoredSince keeps the events the database stored at or after it, and
	// OrderBy names the listing's sort column; empty sorts by occurrence.
	StoredSince *time.Time
	OrderBy     string
}

// RegistrationWindow tells where a view of an endpoint's history starts: the
// requested start, or its registration when that is later; registered is
// false for an EUI the tenant has not registered.
type RegistrationWindow interface {
	Start(ctx context.Context, tenantID int64, epEui []byte, requested *time.Time) (start *time.Time, registered bool, err error)
}

// DefaultEventStreamBatchSize is the default batch size for event streaming operations.
const DefaultEventStreamBatchSize = 100

// Service implements grpcservices.EventService with streaming support.
type Service struct {
	eventStore         SystemEventStore
	resolver           EUIResolver
	window             RegistrationWindow
	logger             logger.Logger
	streamPollInterval time.Duration
	streamOverlap      time.Duration
	streamWake         streampoll.Waker
	streamBatchSize    int
}

// Operation labels appended to failure log messages.
const (
	opListEvents            = "list events"
	opListBaseStationEvents = "list base station events"
	opListEndpointEvents    = "list endpoint events"
)

// New creates a new events service; streamWake announces stored events to the
// event streams, and each stream read reaches streamOverlap back in storage order.
func New(eventStore SystemEventStore, resolver EUIResolver, window RegistrationWindow, streamPollInterval, streamOverlap time.Duration, streamWake streampoll.Waker, streamBatchSize int, log logger.Logger) *Service {
	if streamBatchSize <= 0 {
		streamBatchSize = DefaultEventStreamBatchSize
	}
	return &Service{
		eventStore:         eventStore,
		resolver:           resolver,
		window:             window,
		logger:             log,
		streamPollInterval: streamPollInterval,
		streamOverlap:      streamOverlap,
		streamWake:         streamWake,
		streamBatchSize:    streamBatchSize,
	}
}

// List returns events for the given tenant with optional filters.
func (s *Service) List(ctx context.Context, tenantID int64, filters *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error) {
	filter, err := convertFilters(filters)
	if err != nil {
		return nil, 0, err
	}
	if filter != nil && filter.EndpointEUI != "" {
		epEui, err := validation.ParseEUIBytes(filter.EndpointEUI)
		if err != nil {
			return nil, 0, fmt.Errorf("%w: %w", ErrInvalidEndpointEUI, err)
		}
		return s.listEndpointWindow(ctx, tenantID, epEui, filter, limit, offset, opListEvents)
	}
	return s.listWithCount(ctx, tenantID, filter, limit, offset, opListEvents)
}

// ListByBaseStation returns events for a specific base station.
func (s *Service) ListByBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error) {
	filter, err := convertFilters(filters)
	if err != nil {
		return nil, 0, err
	}
	filter, err = s.scopeBaseStation(ctx, tenantID, bsEui, filter)
	if err != nil {
		return nil, 0, err
	}
	return s.listWithCount(ctx, tenantID, filter, limit, offset, opListBaseStationEvents)
}

// ListByEndPoint returns events for a specific endpoint.
func (s *Service) ListByEndPoint(ctx context.Context, tenantID int64, epEui []byte, filters *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error) {
	filter, err := convertFilters(filters)
	if err != nil {
		return nil, 0, err
	}
	filter, err = s.scopeEndPoint(ctx, tenantID, epEui, filter)
	if err != nil {
		return nil, 0, err
	}
	return s.listEndpointWindow(ctx, tenantID, epEui, filter, limit, offset, opListEndpointEvents)
}

// listEndpointWindow lists one endpoint's events from its registration on;
// an EUI the tenant has not registered has none.
func (s *Service) listEndpointWindow(ctx context.Context, tenantID int64, epEui []byte, filter *EventFilter, limit, offset int, op string) ([]*grpcservices.Event, int64, error) {
	if filter == nil {
		filter = &EventFilter{}
	}
	start, registered, err := s.window.Start(ctx, tenantID, epEui, filter.StartTime)
	if err != nil || !registered {
		return []*grpcservices.Event{}, 0, err
	}
	filter.StartTime = start
	return s.listWithCount(ctx, tenantID, filter, limit, offset, op)
}

// listWithCount fetches a page of events plus the matching total.
func (s *Service) listWithCount(ctx context.Context, tenantID int64, filter *EventFilter, limit, offset int, op string) ([]*grpcservices.Event, int64, error) {
	events, err := s.eventStore.GetEvents(ctx, tenantID, filter, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, logMsgFailedToPrefix+op, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}

	total, err := s.eventStore.CountEvents(ctx, tenantID, filter)
	if err != nil {
		s.logger.ErrorContext(ctx, logMsgFailedToPrefix+op, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}

	result := make([]*grpcservices.Event, len(events))
	for i, e := range events {
		result[i] = convertEvent(e)
	}

	return result, total, nil
}

// Stream streams the tenant's events recorded after the stream opened. Uses
// GetEvents only: the poller discards totals, so a per-poll COUNT(*) would be waste.
func (s *Service) Stream(ctx context.Context, tenantID int64, filters *grpcservices.EventFilters) (<-chan *grpcservices.Event, error) {
	base := grpcservices.EventFilters{}
	if filters != nil {
		base = grpcservices.EventFilters{Categories: filters.Categories, Severity: filters.Severity, StartTime: filters.StartTime, EndTime: filters.EndTime}
	}
	if _, err := convertFilters(&base); err != nil {
		return nil, err
	}
	return streampoll.Stream(ctx, s.streamPollInterval, s.streamBatchSize, streampoll.Source[*grpcservices.Event]{
		Fetch: func(ctx context.Context, since *time.Time, offset int) ([]*grpcservices.Event, error) {
			return s.fetchStored(ctx, tenantID, &base, since, offset)
		},
		PageSize: s.streamBatchSize,
		StoredAt: func(e *grpcservices.Event) time.Time { return e.StoredAt },
		Key:      func(e *grpcservices.Event) string { return e.ID },
		Overlap:  s.streamOverlap,
		OnError: func(err error) {
			s.logger.ErrorContext(ctx, logMsgStreamPollError, logger.FieldTenantID, tenantID, logger.FieldError, err)
		},
		Wake: s.streamWake,
	}), nil
}

// fetchStored reads a page of a stream's events stored at or after since,
// newest stored first.
func (s *Service) fetchStored(ctx context.Context, tenantID int64, query *grpcservices.EventFilters, since *time.Time, offset int) ([]*grpcservices.Event, error) {
	filter, err := convertFilters(query)
	if err != nil {
		return nil, err
	}
	filter.StoredSince = since
	filter.OrderBy = models.EventOrderByStored
	stored, err := s.eventStore.GetEvents(ctx, tenantID, filter, s.streamBatchSize, offset)
	if err != nil {
		return nil, err
	}
	events := make([]*grpcservices.Event, len(stored))
	for i, e := range stored {
		events[i] = convertEvent(e)
	}
	return events, nil
}

// scopeBaseStation narrows a filter to a base station, resolving its EUI to an
// ID when the station is known and always setting the EUI fallback.
func (s *Service) scopeBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filter *EventFilter) (*EventFilter, error) {
	if filter == nil {
		filter = &EventFilter{}
	}
	filter.BaseStationEUI = mioty.FormatEUIBytes(bsEui)
	id, err := s.resolver.ResolveBaseStationID(ctx, tenantID, bsEui)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errResolveEventScope, err)
	}
	filter.BaseStationID = id
	return filter, nil
}

// scopeEndPoint narrows a filter to an endpoint (see scopeBaseStation).
func (s *Service) scopeEndPoint(ctx context.Context, tenantID int64, epEui []byte, filter *EventFilter) (*EventFilter, error) {
	if filter == nil {
		filter = &EventFilter{}
	}
	filter.EndpointEUI = mioty.FormatEUIBytes(epEui)
	id, err := s.resolver.ResolveEndpointID(ctx, tenantID, epEui)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errResolveEventScope, err)
	}
	filter.EndpointID = id
	return filter, nil
}

// convertFilters converts grpcservices.EventFilters to internal EventFilter.
func convertFilters(filters *grpcservices.EventFilters) (*EventFilter, error) {
	if filters == nil {
		return nil, nil
	}
	severity, err := severitiesFor(filters.Outcome, filters.Severity)
	if err != nil {
		return nil, err
	}

	filter := &EventFilter{
		Categories:     filters.Categories,
		Severity:       severity,
		EventTypes:     filters.EventTypes,
		BaseStationEUI: filters.BsEUI,
		EndpointEUI:    filters.EpEUI,
		OpID:           filters.OpID,
		Search:         filters.Search,
		StartTime:      filters.StartTime,
		EndTime:        filters.EndTime,
	}

	return filter, nil
}

// convertEvent converts models.SystemEvent to grpcservices.Event.
func convertEvent(e *models.SystemEvent) *grpcservices.Event {
	// TenantID is string in models.SystemEvent, convert if needed
	var tenantID int64
	if e.TenantID != "" {
		if _, err := fmt.Sscanf(e.TenantID, "%d", &tenantID); err != nil {
			tenantID = 0
		}
	}

	return &grpcservices.Event{
		ID:          e.ID,
		TenantID:    tenantID,
		Category:    e.Category,
		EventType:   e.EventType,
		Severity:    e.Severity,
		Title:       e.Title,
		Description: e.Description,
		SourceName:  e.SourceName,
		UserID:      e.UserID,
		UserEmail:   e.UserEmail,
		Timestamp:   e.CreatedAt,
		Data:        projectDetails(e.EventType, e.Details),
		StoredAt:    e.StoredAt,
	}
}

// Ensure Service implements grpcservices.EventService
var _ grpcservices.EventService = (*Service)(nil)

// Log messages for event listing and streaming failures; the failed-to prefix
// is completed with the operation name at the call site.
const (
	logMsgFailedToPrefix  = "failed to "
	logMsgStreamPollError = "stream poll error"
)
