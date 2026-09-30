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
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// EventReader lists the events of one base station or endpoint.
type EventReader interface {
	ListByBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error)
	ListByEndPoint(ctx context.Context, tenantID int64, epEui []byte, filters *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error)
}

// MessageReader lists the uplinks of one base station or endpoint.
type MessageReader interface {
	ListBaseStationMessages(ctx context.Context, tenantID int64, bsEui []byte, filters *grpcservices.MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error)
	ListMessages(ctx context.Context, tenantID int64, filters *grpcservices.MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error)
}

// Service implements grpcservices.ActivityService.
type Service struct {
	eventSvc   EventReader
	messageSvc MessageReader
	log        logger.Logger
}

// overfetchFactor over-reads each source page so merged pagination can
// tolerate uneven interleaving between sources.
const overfetchFactor = 2

// New creates a new ActivityService.
func New(eventSvc EventReader, messageSvc MessageReader, log logger.Logger) *Service {
	return &Service{eventSvc: eventSvc, messageSvc: messageSvc, log: log}
}

// pageTokenData encodes pagination state for cursor-based pagination.
type pageTokenData struct {
	EventOffset   int `json:"eo"`
	MessageOffset int `json:"mo"`
}

// feed is one device's two activity sources, read from an offset each.
type feed struct {
	events   func(filters *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error)
	messages func(filters *grpcservices.MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error)
	// messageScope narrows the uplink filters to the device, when the reader needs it there.
	messageScope   func(*grpcservices.MessageFilters)
	eventsFailed   string
	messagesFailed string
	device         []byte
}

// ListBaseStationActivity returns merged events and messages for a base station.
// Items are sorted by timestamp descending with unified pagination.
func (s *Service) ListBaseStationActivity(ctx context.Context, tenantID int64, bsEui []byte,
	filters *grpcservices.ActivityFilters, pageSize int, pageToken string,
) (*grpcservices.ActivityListResult, error) {
	return s.list(ctx, feed{
		events: func(f *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error) {
			return s.eventSvc.ListByBaseStation(ctx, tenantID, bsEui, f, limit, offset)
		},
		messages: func(f *grpcservices.MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error) {
			return s.messageSvc.ListBaseStationMessages(ctx, tenantID, bsEui, f, limit, offset)
		},
		messageScope: func(*grpcservices.MessageFilters) {},
		eventsFailed: LogActivityBSEventsFetchFailed, messagesFailed: LogActivityBSMessagesFetchFailed, device: bsEui,
	}, filters, pageSize, pageToken), nil
}

// ListEndpointActivity returns merged events and messages for an endpoint;
// both readers start the endpoint's history at its registration.
func (s *Service) ListEndpointActivity(ctx context.Context, tenantID int64, epEui []byte,
	filters *grpcservices.ActivityFilters, pageSize int, pageToken string,
) (*grpcservices.ActivityListResult, error) {
	return s.list(ctx, feed{
		events: func(f *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error) {
			return s.eventSvc.ListByEndPoint(ctx, tenantID, epEui, f, limit, offset)
		},
		messages: func(f *grpcservices.MessageFilters, limit, offset int) ([]*mioty.ULDataMessage, int64, error) {
			return s.messageSvc.ListMessages(ctx, tenantID, f, limit, offset)
		},
		messageScope: func(f *grpcservices.MessageFilters) { f.EpEui = epEui },
		eventsFailed: LogActivityEPEventsFetchFailed, messagesFailed: LogActivityEPMessagesFetchFailed, device: epEui,
	}, filters, pageSize, pageToken), nil
}

// list merges one page of a device's events and uplinks, newest first.
func (s *Service) list(ctx context.Context, f feed, window *grpcservices.ActivityFilters, pageSize int, pageToken string) *grpcservices.ActivityListResult {
	pageSize = clampPageSize(pageSize)
	cursor := decodeCursor(pageToken)
	eventFilters, messageFilters := &grpcservices.EventFilters{}, &grpcservices.MessageFilters{}
	if window != nil {
		eventFilters.StartTime, eventFilters.EndTime = window.StartTime, window.EndTime
		messageFilters.StartTime, messageFilters.EndTime = window.StartTime, window.EndTime
	}
	f.messageScope(messageFilters)
	fetchLimit := pageSize * overfetchFactor
	events, eventTotal, err := readableEvents(ctx, eventFilters, func(ef *grpcservices.EventFilters) ([]*grpcservices.Event, int64, error) {
		return f.events(ef, fetchLimit, cursor.EventOffset)
	})
	if err != nil {
		s.log.ErrorContext(ctx, f.eventsFailed, logger.FieldError, err, logger.FieldEui, f.device)
		events, eventTotal = nil, 0
	}
	messages, messageTotal, err := f.messages(messageFilters, fetchLimit, cursor.MessageOffset)
	if err != nil {
		s.log.ErrorContext(ctx, f.messagesFailed, logger.FieldError, err, logger.FieldEui, f.device)
		messages, messageTotal = nil, 0
	}
	items, next := page(merge(events, messages), pageSize, cursor)
	return &grpcservices.ActivityListResult{Items: items, NextPageToken: next, TotalCount: eventTotal + messageTotal}
}

func clampPageSize(pageSize int) int {
	switch {
	case pageSize <= 0:
		return defaultPageSize
	case pageSize > maxPageSize:
		return maxPageSize
	}
	return pageSize
}

// decodeCursor reads a page token; an unreadable one starts at the top.
func decodeCursor(pageToken string) pageTokenData {
	var cursor pageTokenData
	if raw, err := base64.StdEncoding.DecodeString(pageToken); err == nil {
		if json.Unmarshal(raw, &cursor) != nil {
			return pageTokenData{}
		}
	}
	return cursor
}

// merge orders events and uplinks newest first.
func merge(events []*grpcservices.Event, messages []*mioty.ULDataMessage) []*grpcservices.ActivityItem {
	items := make([]*grpcservices.ActivityItem, 0, len(events)+len(messages))
	for _, e := range events {
		items = append(items, &grpcservices.ActivityItem{Type: grpcservices.ActivityItemTypeEvent, OccurredAt: e.Timestamp, Event: e})
	}
	for _, m := range messages {
		items = append(items, &grpcservices.ActivityItem{Type: grpcservices.ActivityItemTypeMessage, OccurredAt: time.Unix(0, m.RxTime), Message: m})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].OccurredAt.After(items[j].OccurredAt) })
	return items
}

// page cuts one page and the token of the next: the offsets each source consumed.
func page(items []*grpcservices.ActivityItem, pageSize int, cursor pageTokenData) ([]*grpcservices.ActivityItem, string) {
	if len(items) <= pageSize {
		return items, ""
	}
	items = items[:pageSize]
	next := cursor
	for _, item := range items {
		if item.Type == grpcservices.ActivityItemTypeEvent {
			next.EventOffset++
		} else {
			next.MessageOffset++
		}
	}
	raw, err := json.Marshal(next)
	if err != nil {
		return items, ""
	}
	return items, base64.StdEncoding.EncodeToString(raw)
}

// Ensure Service implements ActivityService interface.
var _ grpcservices.ActivityService = (*Service)(nil)

// readableEvents narrows fetch to the event categories the caller's roles may read.
func readableEvents(
	ctx context.Context,
	filters *grpcservices.EventFilters,
	fetch func(*grpcservices.EventFilters) ([]*grpcservices.Event, int64, error),
) ([]*grpcservices.Event, int64, error) {
	categories, unrestricted := authz.VisibleEventCategories(authz.FromContext(ctx), nil)
	if !unrestricted && len(categories) == 0 {
		return nil, 0, nil
	}
	filters.Categories = categories
	return fetch(filters)
}
