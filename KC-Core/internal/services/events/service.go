// Package events provides system event storage and retrieval.
package events

import (
	"context"
	"fmt"
	"time"

	"github.com/kilocenter/KC-Core/internal/services/grpcservices"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/models"
)

// SystemEventStore provides event persistence operations.
type SystemEventStore interface {
	List(ctx context.Context, tenantID int64, filter *EventFilter, limit, offset int) ([]*models.SystemEvent, int64, error)
	ListByBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filter *EventFilter, limit, offset int) ([]*models.SystemEvent, int64, error)
	ListByEndPoint(ctx context.Context, tenantID int64, epEui []byte, filter *EventFilter, limit, offset int) ([]*models.SystemEvent, int64, error)
}

// EventFilter contains filtering criteria for events.
type EventFilter struct {
	Categories []string
	Severity   []string
	EventTypes []string
	StartTime  *int64 // Unix timestamp
	EndTime    *int64 // Unix timestamp
}

// DefaultEventStreamBatchSize is the default batch size for event streaming operations.
const DefaultEventStreamBatchSize = 100

// Service implements grpcservices.EventService with streaming support.
type Service struct {
	eventStore         SystemEventStore
	logger             logger.Logger
	streamPollInterval time.Duration
	streamBatchSize    int
}

// New creates a new events service.
func New(eventStore SystemEventStore, streamPollInterval time.Duration, streamBatchSize int, log logger.Logger) *Service {
	if streamBatchSize <= 0 {
		streamBatchSize = DefaultEventStreamBatchSize
	}
	return &Service{
		eventStore:         eventStore,
		logger:             log,
		streamPollInterval: streamPollInterval,
		streamBatchSize:    streamBatchSize,
	}
}

// List returns events for the given tenant with optional filters.
func (s *Service) List(ctx context.Context, tenantID int64, filters *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error) {
	// Convert grpcservices filters to internal format
	filter := convertFilters(filters)

	events, total, err := s.eventStore.List(ctx, tenantID, filter, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list events", "tenantID", tenantID, "error", err)
		return nil, 0, fmt.Errorf("list events: %w", err)
	}

	result := make([]*grpcservices.Event, len(events))
	for i, e := range events {
		result[i] = convertEvent(e)
	}

	return result, total, nil
}

// ListByBaseStation returns events for a specific base station.
func (s *Service) ListByBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error) {
	filter := convertFilters(filters)

	events, total, err := s.eventStore.ListByBaseStation(ctx, tenantID, bsEui, filter, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list base station events", "tenantID", tenantID, "error", err)
		return nil, 0, fmt.Errorf("list base station events: %w", err)
	}

	result := make([]*grpcservices.Event, len(events))
	for i, e := range events {
		result[i] = convertEvent(e)
	}

	return result, total, nil
}

// ListByEndPoint returns events for a specific endpoint.
func (s *Service) ListByEndPoint(ctx context.Context, tenantID int64, epEui []byte, filters *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error) {
	filter := convertFilters(filters)

	events, total, err := s.eventStore.ListByEndPoint(ctx, tenantID, epEui, filter, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list endpoint events", "tenantID", tenantID, "error", err)
		return nil, 0, fmt.Errorf("list endpoint events: %w", err)
	}

	result := make([]*grpcservices.Event, len(events))
	for i, e := range events {
		result[i] = convertEvent(e)
	}

	return result, total, nil
}

// Stream streams events for the given tenant via poll-based realtime delivery.
func (s *Service) Stream(ctx context.Context, tenantID int64, filters *grpcservices.EventFilters) (<-chan *grpcservices.Event, error) {
	ch := make(chan *grpcservices.Event, s.streamBatchSize)

	go func() {
		defer close(ch)

		ticker := time.NewTicker(s.streamPollInterval)
		defer ticker.Stop()

		// Use filters.StartTime as moving window if provided
		var lastTimestamp *time.Time
		if filters != nil && filters.StartTime != nil {
			lastTimestamp = filters.StartTime
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Build query filters with updated start time
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
				events, _, err := s.eventStore.List(ctx, tenantID, filter, s.streamBatchSize, 0)
				if err != nil {
					s.logger.ErrorContext(ctx, "stream poll error", "tenantID", tenantID, "error", err)
					continue
				}

				// Emit events in chronological order (List returns DESC)
				// Capture prior cursor before loop to allow same-timestamp events in batch
				prevTimestamp := lastTimestamp
				var latestTimestamp *time.Time

				for i := len(events) - 1; i >= 0; i-- {
					// Skip events at or before PRIOR cursor (cross-poll deduplication)
					if prevTimestamp != nil && !events[i].CreatedAt.After(*prevTimestamp) {
						continue
					}
					select {
					case ch <- convertEvent(events[i]):
						// Track newest timestamp in this batch
						t := events[i].CreatedAt
						latestTimestamp = &t
					case <-ctx.Done():
						return
					}
				}

				// Update cursor after loop for next poll
				if latestTimestamp != nil {
					lastTimestamp = latestTimestamp
				}
			}
		}
	}()

	return ch, nil
}

// StreamByBaseStation streams events for a specific base station via poll-based realtime delivery.
func (s *Service) StreamByBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.EventFilters) (<-chan *grpcservices.Event, error) {
	ch := make(chan *grpcservices.Event, s.streamBatchSize)

	go func() {
		defer close(ch)

		ticker := time.NewTicker(s.streamPollInterval)
		defer ticker.Stop()

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
				events, _, err := s.eventStore.ListByBaseStation(ctx, tenantID, bsEui, filter, s.streamBatchSize, 0)
				if err != nil {
					s.logger.ErrorContext(ctx, "stream bs events poll error", "tenantID", tenantID, "error", err)
					continue
				}

				// Capture prior cursor before loop to allow same-timestamp events in batch
				prevTimestamp := lastTimestamp
				var latestTimestamp *time.Time

				for i := len(events) - 1; i >= 0; i-- {
					// Skip events at or before PRIOR cursor (cross-poll deduplication)
					if prevTimestamp != nil && !events[i].CreatedAt.After(*prevTimestamp) {
						continue
					}
					select {
					case ch <- convertEvent(events[i]):
						// Track newest timestamp in this batch
						t := events[i].CreatedAt
						latestTimestamp = &t
					case <-ctx.Done():
						return
					}
				}

				// Update cursor after loop for next poll
				if latestTimestamp != nil {
					lastTimestamp = latestTimestamp
				}
			}
		}
	}()

	return ch, nil
}

// StreamByEndPoint streams events for a specific endpoint via poll-based realtime delivery.
func (s *Service) StreamByEndPoint(ctx context.Context, tenantID int64, epEui []byte, filters *grpcservices.EventFilters) (<-chan *grpcservices.Event, error) {
	ch := make(chan *grpcservices.Event, s.streamBatchSize)

	go func() {
		defer close(ch)

		ticker := time.NewTicker(s.streamPollInterval)
		defer ticker.Stop()

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
				events, _, err := s.eventStore.ListByEndPoint(ctx, tenantID, epEui, filter, s.streamBatchSize, 0)
				if err != nil {
					s.logger.ErrorContext(ctx, "stream ep events poll error", "tenantID", tenantID, "error", err)
					continue
				}

				// Capture prior cursor before loop to allow same-timestamp events in batch
				prevTimestamp := lastTimestamp
				var latestTimestamp *time.Time

				for i := len(events) - 1; i >= 0; i-- {
					// Skip events at or before PRIOR cursor (cross-poll deduplication)
					if prevTimestamp != nil && !events[i].CreatedAt.After(*prevTimestamp) {
						continue
					}
					select {
					case ch <- convertEvent(events[i]):
						// Track newest timestamp in this batch
						t := events[i].CreatedAt
						latestTimestamp = &t
					case <-ctx.Done():
						return
					}
				}

				// Update cursor after loop for next poll
				if latestTimestamp != nil {
					lastTimestamp = latestTimestamp
				}
			}
		}
	}()

	return ch, nil
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
