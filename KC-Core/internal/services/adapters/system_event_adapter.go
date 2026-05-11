// Package adapters provides storage adapters bridging KC-DB to gRPC services.
package adapters

import (
	"context"
	"fmt"
	"strconv"
	"time"

	eventsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/events"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// SystemEventStoreAdapter adapts interfaces.SystemEventStore to eventsservice.SystemEventStore.
// Bridges KC-DB system event repository to the gRPC service interface.
type SystemEventStoreAdapter struct {
	store  interfaces.SystemEventStore
	bsRepo interfaces.BaseStationRepository
	epRepo interfaces.EndpointRepository
}

// NewSystemEventStoreAdapter creates a new adapter for system events.
// Accepts repositories for EUI-to-ID lookup.
func NewSystemEventStoreAdapter(
	store interfaces.SystemEventStore,
	bsRepo interfaces.BaseStationRepository,
	epRepo interfaces.EndpointRepository,
) *SystemEventStoreAdapter {
	return &SystemEventStoreAdapter{
		store:  store,
		bsRepo: bsRepo,
		epRepo: epRepo,
	}
}

// List returns events for the given tenant with optional filters.
func (a *SystemEventStoreAdapter) List(ctx context.Context, tenantID int64, filter *eventsservice.EventFilter, limit, offset int) ([]*models.SystemEvent, int64, error) {
	dbFilter := convertEventFilter(tenantID, filter)
	dbFilter.Limit = limit
	dbFilter.Offset = offset

	events, err := a.store.GetEvents(ctx, dbFilter)
	if err != nil {
		return nil, 0, err
	}

	// Use CountEvents for accurate total count
	countFilter := convertEventFilter(tenantID, filter)
	total, err := a.store.CountEvents(ctx, countFilter)
	if err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

// ListByBaseStation returns events for a specific base station.
// Uses repository injection for EUI-to-ID lookup with EUI fallback.
func (a *SystemEventStoreAdapter) ListByBaseStation(ctx context.Context, tenantID int64, bsEui []byte, filter *eventsservice.EventFilter, limit, offset int) ([]*models.SystemEvent, int64, error) {
	dbFilter := convertEventFilter(tenantID, filter)
	dbFilter.Limit = limit
	dbFilter.Offset = offset

	// Resolve EUI to base station ID via repository
	euiHex := fmt.Sprintf("%016x", bsEui)
	if len(bsEui) >= 8 && a.bsRepo != nil {
		bs, err := a.bsRepo.GetByEUI(ctx, tenantID, bsEui)
		if err == nil && bs != nil {
			dbFilter.BaseStationID = &bs.ID
		}
	}
	// Always set EUI fallback for events that may lack the FK
	dbFilter.BaseStationEUI = euiHex

	events, err := a.store.GetEvents(ctx, dbFilter)
	if err != nil {
		return nil, 0, err
	}

	// Use CountEvents for accurate total count
	countFilter := convertEventFilter(tenantID, filter)
	if dbFilter.BaseStationID != nil {
		countFilter.BaseStationID = dbFilter.BaseStationID
	}
	countFilter.BaseStationEUI = euiHex
	total, err := a.store.CountEvents(ctx, countFilter)
	if err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

// ListByEndPoint returns events for a specific endpoint.
// Uses repository injection for EUI-to-ID lookup with EUI fallback.
func (a *SystemEventStoreAdapter) ListByEndPoint(ctx context.Context, tenantID int64, epEui []byte, filter *eventsservice.EventFilter, limit, offset int) ([]*models.SystemEvent, int64, error) {
	dbFilter := convertEventFilter(tenantID, filter)
	dbFilter.Limit = limit
	dbFilter.Offset = offset

	// Resolve EUI to endpoint ID via repository
	euiHex := fmt.Sprintf("%016x", epEui)
	if len(epEui) >= 8 && a.epRepo != nil {
		ep, err := a.epRepo.GetByEUI(ctx, tenantID, epEui)
		if err == nil && ep != nil {
			dbFilter.EndpointID = &ep.ID
		}
	}
	// Always set EUI fallback for events that may lack the FK
	dbFilter.EndpointEUI = euiHex

	events, err := a.store.GetEvents(ctx, dbFilter)
	if err != nil {
		return nil, 0, err
	}

	// Use CountEvents for accurate total count
	countFilter := convertEventFilter(tenantID, filter)
	if dbFilter.EndpointID != nil {
		countFilter.EndpointID = dbFilter.EndpointID
	}
	countFilter.EndpointEUI = euiHex
	total, err := a.store.CountEvents(ctx, countFilter)
	if err != nil {
		return nil, 0, err
	}

	return events, total, nil
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

// CreateEvent delegates to the underlying SystemEventStore.
func (a *SystemEventStoreAdapter) CreateEvent(ctx context.Context, event *models.SystemEvent) error {
	return a.store.CreateEvent(ctx, event)
}
