// Package activity implements the unified activity feed service.
// Merges events and messages for base stations and endpoints into a single timeline.
package activity

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sort"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// Service implements grpcservices.ActivityService.
type Service struct {
	eventSvc   grpcservices.EventService
	messageSvc grpcservices.MessageListingService
	log        logger.Logger
}

// New creates a new ActivityService.
func New(eventSvc grpcservices.EventService, messageSvc grpcservices.MessageListingService, log logger.Logger) *Service {
	return &Service{
		eventSvc:   eventSvc,
		messageSvc: messageSvc,
		log:        log,
	}
}

// pageTokenData encodes pagination state for cursor-based pagination.
type pageTokenData struct {
	EventOffset   int   `json:"eo"`
	MessageOffset int   `json:"mo"`
	LastTimestamp int64 `json:"lt"` // Unix nanoseconds
}

// ListBaseStationActivity returns merged events and messages for a base station.
// Items are sorted by timestamp descending with unified pagination.
func (s *Service) ListBaseStationActivity(
	ctx context.Context,
	tenantID int64,
	bsEui []byte,
	filters *grpcservices.ActivityFilters,
	pageSize int,
	pageToken string,
) (*grpcservices.ActivityListResult, error) {
	// Default page size
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Parse page token for cursor state
	var cursor pageTokenData
	if pageToken != "" {
		tokenBytes, err := base64.StdEncoding.DecodeString(pageToken)
		if err == nil {
			_ = json.Unmarshal(tokenBytes, &cursor)
		}
	}

	// Convert activity filters to service-specific filters
	eventFilters := &grpcservices.EventFilters{}
	messageFilters := &grpcservices.MessageFilters{}

	if filters != nil {
		eventFilters.StartTime = filters.StartTime
		eventFilters.EndTime = filters.EndTime
		messageFilters.StartTime = filters.StartTime
		messageFilters.EndTime = filters.EndTime
	}

	// Fetch events - use larger limit to allow for merging
	fetchLimit := pageSize * 2
	events, eventTotal, err := s.eventSvc.ListByBaseStation(ctx, tenantID, bsEui, eventFilters, fetchLimit, cursor.EventOffset)
	if err != nil {
		s.log.Error("failed to fetch events for activity feed", "error", err, "bsEui", bsEui)
		events = []*grpcservices.Event{}
		eventTotal = 0
	}

	// Fetch messages
	messages, messageTotal, err := s.messageSvc.ListBaseStationMessages(ctx, tenantID, bsEui, messageFilters, fetchLimit, cursor.MessageOffset)
	if err != nil {
		s.log.Error("failed to fetch messages for activity feed", "error", err, "bsEui", bsEui)
		messages = nil
		messageTotal = 0
	}

	// Merge into activity items
	items := make([]*grpcservices.ActivityItem, 0, len(events)+len(messages))

	for _, e := range events {
		items = append(items, &grpcservices.ActivityItem{
			Type:       grpcservices.ActivityItemTypeEvent,
			OccurredAt: e.Timestamp,
			Event:      e,
		})
	}

	for _, m := range messages {
		var occurredAt time.Time
		if m.RxTime > 0 {
			occurredAt = time.Unix(0, m.RxTime)
		} else {
			occurredAt = time.Now()
		}
		items = append(items, &grpcservices.ActivityItem{
			Type:       grpcservices.ActivityItemTypeMessage,
			OccurredAt: occurredAt,
			Message:    m,
		})
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].OccurredAt.After(items[j].OccurredAt)
	})

	// Apply pagination limit
	totalCount := eventTotal + messageTotal
	hasMore := len(items) > pageSize
	if len(items) > pageSize {
		items = items[:pageSize]
	}

	// Generate next page token
	var nextPageToken string
	if hasMore && len(items) > 0 {
		// Count how many events and messages we consumed in this page
		eventConsumed := 0
		messageConsumed := 0
		for _, item := range items {
			if item.Type == grpcservices.ActivityItemTypeEvent {
				eventConsumed++
			} else {
				messageConsumed++
			}
		}

		nextCursor := pageTokenData{
			EventOffset:   cursor.EventOffset + eventConsumed,
			MessageOffset: cursor.MessageOffset + messageConsumed,
			LastTimestamp: items[len(items)-1].OccurredAt.UnixNano(),
		}
		tokenBytes, _ := json.Marshal(nextCursor)
		nextPageToken = base64.StdEncoding.EncodeToString(tokenBytes)
	}

	return &grpcservices.ActivityListResult{
		Items:         items,
		NextPageToken: nextPageToken,
		TotalCount:    totalCount,
	}, nil
}

// ListEndpointActivity returns merged events and messages for an endpoint.
// Items are sorted by timestamp descending with unified pagination.
func (s *Service) ListEndpointActivity(
	ctx context.Context,
	tenantID int64,
	epEui []byte,
	filters *grpcservices.ActivityFilters,
	pageSize int,
	pageToken string,
) (*grpcservices.ActivityListResult, error) {
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var cursor pageTokenData
	if pageToken != "" {
		tokenBytes, err := base64.StdEncoding.DecodeString(pageToken)
		if err == nil {
			_ = json.Unmarshal(tokenBytes, &cursor)
		}
	}

	eventFilters := &grpcservices.EventFilters{}
	messageFilters := &grpcservices.MessageFilters{
		EpEui: epEui,
	}

	if filters != nil {
		eventFilters.StartTime = filters.StartTime
		eventFilters.EndTime = filters.EndTime
		messageFilters.StartTime = filters.StartTime
		messageFilters.EndTime = filters.EndTime
	}

	fetchLimit := pageSize * 2
	events, eventTotal, err := s.eventSvc.ListByEndPoint(ctx, tenantID, epEui, eventFilters, fetchLimit, cursor.EventOffset)
	if err != nil {
		s.log.Error("failed to fetch events for endpoint activity feed", "error", err, "epEui", epEui)
		events = []*grpcservices.Event{}
		eventTotal = 0
	}

	messages, messageTotal, err := s.messageSvc.ListMessages(ctx, tenantID, messageFilters, fetchLimit, cursor.MessageOffset)
	if err != nil {
		s.log.Error("failed to fetch messages for endpoint activity feed", "error", err, "epEui", epEui)
		messages = nil
		messageTotal = 0
	}

	items := make([]*grpcservices.ActivityItem, 0, len(events)+len(messages))

	for _, e := range events {
		items = append(items, &grpcservices.ActivityItem{
			Type:       grpcservices.ActivityItemTypeEvent,
			OccurredAt: e.Timestamp,
			Event:      e,
		})
	}

	for _, m := range messages {
		var occurredAt time.Time
		if m.RxTime > 0 {
			occurredAt = time.Unix(0, m.RxTime)
		} else {
			occurredAt = time.Now()
		}
		items = append(items, &grpcservices.ActivityItem{
			Type:       grpcservices.ActivityItemTypeMessage,
			OccurredAt: occurredAt,
			Message:    m,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].OccurredAt.After(items[j].OccurredAt)
	})

	totalCount := eventTotal + messageTotal
	hasMore := len(items) > pageSize
	if len(items) > pageSize {
		items = items[:pageSize]
	}

	var nextPageToken string
	if hasMore && len(items) > 0 {
		eventConsumed := 0
		messageConsumed := 0
		for _, item := range items {
			if item.Type == grpcservices.ActivityItemTypeEvent {
				eventConsumed++
			} else {
				messageConsumed++
			}
		}

		nextCursor := pageTokenData{
			EventOffset:   cursor.EventOffset + eventConsumed,
			MessageOffset: cursor.MessageOffset + messageConsumed,
			LastTimestamp: items[len(items)-1].OccurredAt.UnixNano(),
		}
		tokenBytes, _ := json.Marshal(nextCursor)
		nextPageToken = base64.StdEncoding.EncodeToString(tokenBytes)
	}

	return &grpcservices.ActivityListResult{
		Items:         items,
		NextPageToken: nextPageToken,
		TotalCount:    totalCount,
	}, nil
}

// Ensure Service implements ActivityService interface.
var _ grpcservices.ActivityService = (*Service)(nil)
