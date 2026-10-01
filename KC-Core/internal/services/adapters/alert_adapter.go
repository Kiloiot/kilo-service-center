// Package adapters provides storage adapters bridging KC-DB to gRPC services.
package adapters

import (
	"context"
	"strconv"

	alertsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/alerts"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// unpaginated disables Limit/Offset so count queries span the entire filter range.
const unpaginated = 0

// alertEventStore is the slice of the system event store the alert adapter
// reads. Satisfied structurally by the KC-DB system event store.
type alertEventStore interface {
	GetActiveAlerts(ctx context.Context, filter models.AlertFilter) ([]*models.SystemEvent, error)
	CountActiveAlerts(ctx context.Context, filter models.AlertFilter) (int64, error)
	CountAlertsBySeverity(ctx context.Context, filter models.AlertFilter) (map[string]int64, error)
}

// AlertStoreAdapter adapts the system event store to alertsservice.AlertStore.
// Alerts are system events with severity warning, error, or critical.
type AlertStoreAdapter struct {
	store       alertEventStore
	recentLimit int
}

// NewAlertStoreAdapter creates a new adapter for alerts; the summary lists
// at most recentLimit recent alerts.
func NewAlertStoreAdapter(store alertEventStore, recentLimit int) *AlertStoreAdapter {
	return &AlertStoreAdapter{
		store:       store,
		recentLimit: recentLimit,
	}
}

// openAlerts selects the tenant's unresolved alerts of every alert severity,
// the rule both the list and the summary start from.
func openAlerts(tenantID int64) models.AlertFilter {
	return models.AlertFilter{
		TenantID:   strconv.FormatInt(tenantID, 10),
		Severities: alertsservice.AlertSeverities,
	}
}

// List returns the tenant's alerts in the filter's severities and statuses.
func (a *AlertStoreAdapter) List(ctx context.Context, tenantID int64, filter *alertsservice.AlertFilter, limit, offset int) ([]*alertsservice.Alert, int64, error) {
	dbFilter := openAlerts(tenantID)
	dbFilter.Limit = limit
	dbFilter.Offset = offset

	if filter != nil {
		if len(filter.Severity) > 0 {
			dbFilter.Severities = filter.Severity
		}
		dbFilter.Statuses = filter.Status
	}

	events, err := a.store.GetActiveAlerts(ctx, dbFilter)
	if err != nil {
		return nil, 0, err
	}

	countFilter := dbFilter
	countFilter.Limit = unpaginated
	countFilter.Offset = unpaginated
	total, err := a.store.CountActiveAlerts(ctx, countFilter)
	if err != nil {
		return nil, 0, err
	}

	return toAlerts(tenantID, events), total, nil
}

// GetSummary counts the open alerts per severity, each count the total the
// list returns for that severity, and returns the newest of them.
func (a *AlertStoreAdapter) GetSummary(ctx context.Context, tenantID int64) (*alertsservice.AlertSummary, error) {
	open := openAlerts(tenantID)

	counts, err := a.store.CountAlertsBySeverity(ctx, open)
	if err != nil {
		return nil, err
	}

	recent := open
	recent.Limit = a.recentLimit
	recentEvents, err := a.store.GetActiveAlerts(ctx, recent)
	if err != nil {
		return nil, err
	}

	return &alertsservice.AlertSummary{
		Critical: safeInt32(counts[models.EventSeverityCritical]),
		Error:    safeInt32(counts[models.EventSeverityError]),
		Warning:  safeInt32(counts[models.EventSeverityWarning]),
		Recent:   toAlerts(tenantID, recentEvents),
	}, nil
}

func toAlerts(tenantID int64, events []*models.SystemEvent) []*alertsservice.Alert {
	alerts := make([]*alertsservice.Alert, len(events))
	for i, e := range events {
		alerts[i] = &alertsservice.Alert{
			ID:          e.ID,
			TenantID:    tenantID,
			Category:    e.Category,
			Severity:    e.Severity,
			Title:       e.Title,
			Description: e.Description,
			SourceName:  e.SourceName,
			Status:      e.Status,
			CreatedAt:   e.CreatedAt.Unix(),
		}
	}
	return alerts
}

// safeInt32 safely converts int64 to int32, clamping to max value.
func safeInt32(v int64) int32 {
	const maxInt32 = int64(1<<31 - 1)
	if v > maxInt32 {
		return int32(maxInt32)
	}
	if v < -maxInt32-1 {
		return int32(-maxInt32 - 1)
	}
	return int32(v)
}

// Ensure AlertStoreAdapter implements alertsservice.AlertStore
var _ alertsservice.AlertStore = (*AlertStoreAdapter)(nil)
