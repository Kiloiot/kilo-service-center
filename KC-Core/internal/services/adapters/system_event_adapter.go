// Package adapters provides storage adapters bridging KC-DB to gRPC services.
package adapters

import (
	"context"
	"strconv"
	"time"

	eventsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/events"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// SystemEventStoreAdapter adapts interfaces.SystemEventStore to eventsservice.SystemEventStore.
type SystemEventStoreAdapter struct {
	store interfaces.SystemEventStore
}

// NewSystemEventStoreAdapter creates a new adapter for system events.
func NewSystemEventStoreAdapter(store interfaces.SystemEventStore) *SystemEventStoreAdapter {
	return &SystemEventStoreAdapter{store: store}
}

// GetEvents returns a page of events (no total).
func (a *SystemEventStoreAdapter) GetEvents(ctx context.Context, tenantID int64, filter *eventsservice.EventFilter, limit, offset int) ([]*models.SystemEvent, error) {
	dbFilter := convertEventFilter(tenantID, filter)
	dbFilter.Limit = limit
	dbFilter.Offset = offset

	return a.store.GetEvents(ctx, dbFilter)
}

// CountEvents returns the total matching the filter (ignores limit/offset).
func (a *SystemEventStoreAdapter) CountEvents(ctx context.Context, tenantID int64, filter *eventsservice.EventFilter) (int64, error) {
	return a.store.CountEvents(ctx, convertEventFilter(tenantID, filter))
}

// CreateEvent delegates to the underlying SystemEventStore.
func (a *SystemEventStoreAdapter) CreateEvent(ctx context.Context, event *models.SystemEvent) error {
	return a.store.CreateEvent(ctx, event)
}

// convertEventFilter converts service filter to DB filter.
func convertEventFilter(tenantID int64, filter *eventsservice.EventFilter) interfaces.SystemEventFilter {
	dbFilter := interfaces.SystemEventFilter{
		TenantID: strconv.FormatInt(tenantID, 10),
	}

	if filter != nil {
		dbFilter.Categories = filter.Categories
		dbFilter.Severities = filter.Severity
		dbFilter.EventTypes = filter.EventTypes
		dbFilter.BaseStationID = filter.BaseStationID
		dbFilter.EndpointID = filter.EndpointID
		dbFilter.BaseStationEUI = filter.BaseStationEUI
		dbFilter.EndpointEUI = filter.EndpointEUI
		if filter.StartTime != nil {
			t := time.Unix(*filter.StartTime, 0)
			dbFilter.Since = &t
		}
		if filter.EndTime != nil {
			t := time.Unix(*filter.EndTime, 0)
			dbFilter.Until = &t
		}
	}

	return dbFilter
}
