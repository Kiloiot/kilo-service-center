// Package events provides system event storage and retrieval.
package events

import (
	"context"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// SystemEventStore provides event persistence primitives. Rows and total are
// separate so streaming (which discards the total) can skip the COUNT(*).
type SystemEventStore interface {
	GetEvents(ctx context.Context, tenantID int64, filter *EventFilter, limit, offset int) ([]*models.SystemEvent, error)
	CountEvents(ctx context.Context, tenantID int64, filter *EventFilter) (int64, error)
}

// EUIResolver maps a device EUI to its internal ID for event scoping. Best-effort:
// a nil ID means "unknown", so callers fall back to the source_name EUI filter.
type EUIResolver interface {
	ResolveBaseStationID(ctx context.Context, tenantID int64, bsEui []byte) (*int64, error)
	ResolveEndpointID(ctx context.Context, tenantID int64, epEui []byte) (*int64, error)
}

// EventFilter contains filtering criteria for events.
type EventFilter struct {
	Categories []string
	Severity   []string
	EventTypes []string
	StartTime  *int64 // Unix timestamp
	EndTime    *int64 // Unix timestamp

	// Scoping for base-station / endpoint views, set after EUI resolution.
	BaseStationID  *int64
	EndpointID     *int64
	BaseStationEUI string
	EndpointEUI    string
}

// DefaultEventStreamBatchSize is the default batch size for event streaming operations.
const DefaultEventStreamBatchSize = 100

// Service implements grpcservices.EventService with streaming support.
type Service struct {
	eventStore         SystemEventStore
	resolver           EUIResolver
	logger             logger.Logger
	streamPollInterval time.Duration
	streamBatchSize    int
}

// New creates a new events service.
func New(eventStore SystemEventStore, resolver EUIResolver, streamPollInterval time.Duration, streamBatchSize int, log logger.Logger) *Service {
	if streamBatchSize <= 0 {
		streamBatchSize = DefaultEventStreamBatchSize
	}
	return &Service{
		eventStore:         eventStore,
		resolver:           resolver,
		logger:             log,
		streamPollInterval: streamPollInterval,
		streamBatchSize:    streamBatchSize,
	}
}

// List returns events for the given tenant with optional filters.
func (s *Service) List(ctx context.Context, tenantID int64, filters *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error) {
	return s.listWithCount(ctx, tenantID, convertFilters(filters), limit, offset, "list events")
}

// ListByBaseStation returns events for a specific base station.
func (s *Service) ListByBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error) {
	filter := s.scopeBaseStation(ctx, tenantID, bsEui, convertFilters(filters))
	return s.listWithCount(ctx, tenantID, filter, limit, offset, "list base station events")
}

// ListByEndPoint returns events for a specific endpoint.
func (s *Service) ListByEndPoint(ctx context.Context, tenantID int64, epEui []byte, filters *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error) {
	filter := s.scopeEndPoint(ctx, tenantID, epEui, convertFilters(filters))
	return s.listWithCount(ctx, tenantID, filter, limit, offset, "list endpoint events")
}

// listWithCount fetches a page of events plus the matching total.
func (s *Service) listWithCount(ctx context.Context, tenantID int64, filter *EventFilter, limit, offset int, op string) ([]*grpcservices.Event, int64, error) {
	events, err := s.eventStore.GetEvents(ctx, tenantID, filter, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to "+op, "tenantID", tenantID, "error", err)
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}

	total, err := s.eventStore.CountEvents(ctx, tenantID, filter)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to "+op, "tenantID", tenantID, "error", err)
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}

	result := make([]*grpcservices.Event, len(events))
	for i, e := range events {
		result[i] = convertEvent(e)
	}

	return result, total, nil
}

// Stream streams events for the given tenant via poll-based realtime delivery.
func (s *Service) Stream(ctx context.Context, tenantID int64, filters *grpcservices.EventFilters) (<-chan *grpcservices.Event, error) {
	return s.stream(ctx, tenantID, filters, nil, "stream poll error"), nil
}

// StreamByBaseStation streams events for a specific base station via poll-based realtime delivery.
func (s *Service) StreamByBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.EventFilters) (<-chan *grpcservices.Event, error) {
	// Resolve once — EUI→ID is stable for the life of the stream.
	bsID, _ := s.resolver.ResolveBaseStationID(ctx, tenantID, bsEui)
	euiHex := fmt.Sprintf("%016x", bsEui)
	scope := func(f *EventFilter) {
		f.BaseStationID = bsID
		f.BaseStationEUI = euiHex
	}
	return s.stream(ctx, tenantID, filters, scope, "stream bs events poll error"), nil
}

// StreamByEndPoint streams events for a specific endpoint via poll-based realtime delivery.
func (s *Service) StreamByEndPoint(ctx context.Context, tenantID int64, epEui []byte, filters *grpcservices.EventFilters) (<-chan *grpcservices.Event, error) {
	epID, _ := s.resolver.ResolveEndpointID(ctx, tenantID, epEui)
	euiHex := fmt.Sprintf("%016x", epEui)
	scope := func(f *EventFilter) {
		f.EndpointID = epID
		f.EndpointEUI = euiHex
	}
	return s.stream(ctx, tenantID, filters, scope, "stream ep events poll error"), nil
}

// stream runs the shared poll loop. scope (optional) narrows each poll to a base
// station / endpoint. Uses GetEvents only: the poller discards totals, so a
// per-tick COUNT(*) would be pure waste.
func (s *Service) stream(ctx context.Context, tenantID int64, filters *grpcservices.EventFilters, scope func(*EventFilter), errMsg string) <-chan *grpcservices.Event {
	ch := make(chan *grpcservices.Event, s.streamBatchSize)

	go func() {
		defer close(ch)

		ticker := time.NewTicker(s.streamPollInterval)
		defer ticker.Stop()

		// Use filters.StartTime as moving window if provided.
		var lastTimestamp *time.Time
		if filters != nil && filters.StartTime != nil {
			lastTimestamp = filters.StartTime
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				queryFilters := &grpcservices.EventFilters{}
				if filters != nil {
					queryFilters.Categories = filters.Categories
					queryFilters.Severity = filters.Severity
					queryFilters.EndTime = filters.EndTime
				}
				if lastTimestamp != nil {
					queryFilters.StartTime = lastTimestamp
				}

				filter := convertFilters(queryFilters)
				if scope != nil {
					scope(filter)
				}

				events, err := s.eventStore.GetEvents(ctx, tenantID, filter, s.streamBatchSize, 0)
				if err != nil {
					s.logger.ErrorContext(ctx, errMsg, "tenantID", tenantID, "error", err)
					continue
				}

				// Emit events in chronological order (GetEvents returns DESC).
				// Capture prior cursor before loop to allow same-timestamp events in batch.
				prevTimestamp := lastTimestamp
				var latestTimestamp *time.Time

				for i := len(events) - 1; i >= 0; i-- {
					// Skip events at or before PRIOR cursor (cross-poll deduplication).
					if prevTimestamp != nil && !events[i].CreatedAt.After(*prevTimestamp) {
						continue
					}
					select {
					case ch <- convertEvent(events[i]):
						t := events[i].CreatedAt
						latestTimestamp = &t
					case <-ctx.Done():
						return
					}
				}

				if latestTimestamp != nil {
					lastTimestamp = latestTimestamp
				}
			}
		}
	}()

	return ch
}

// scopeBaseStation narrows a filter to a base station, resolving its EUI to an ID
// (best-effort) and always setting the EUI fallback.
func (s *Service) scopeBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filter *EventFilter) *EventFilter {
	if filter == nil {
		filter = &EventFilter{}
	}
	filter.BaseStationEUI = fmt.Sprintf("%016x", bsEui)
	if id, _ := s.resolver.ResolveBaseStationID(ctx, tenantID, bsEui); id != nil {
		filter.BaseStationID = id
	}
	return filter
}

// scopeEndPoint narrows a filter to an endpoint (see scopeBaseStation).
func (s *Service) scopeEndPoint(ctx context.Context, tenantID int64, epEui []byte, filter *EventFilter) *EventFilter {
	if filter == nil {
		filter = &EventFilter{}
	}
	filter.EndpointEUI = fmt.Sprintf("%016x", epEui)
	if id, _ := s.resolver.ResolveEndpointID(ctx, tenantID, epEui); id != nil {
		filter.EndpointID = id
	}
	return filter
}

// convertFilters converts grpcservices.EventFilters to internal EventFilter.
func convertFilters(filters *grpcservices.EventFilters) *EventFilter {
	if filters == nil {
		return nil
	}

	filter := &EventFilter{
		Categories: filters.Categories,
		Severity:   filters.Severity,
		EventTypes: filters.EventTypes,
	}

	if filters.StartTime != nil {
		ts := filters.StartTime.Unix()
		filter.StartTime = &ts
	}
	if filters.EndTime != nil {
		ts := filters.EndTime.Unix()
		filter.EndTime = &ts
	}

	return filter
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
		Timestamp:   e.CreatedAt,
		Data:        e.Details,
	}
}

// Ensure Service implements grpcservices.EventService
var _ grpcservices.EventService = (*Service)(nil)
