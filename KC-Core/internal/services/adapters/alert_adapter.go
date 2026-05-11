// Package adapters provides storage adapters bridging KC-DB to gRPC services.
package adapters

import (
	"context"
	"fmt"
	"strconv"
	"time"

	alertsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/alerts"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// AlertStoreAdapter adapts interfaces.SystemEventStore to alertsservice.AlertStore.
// Alerts are system events with severity warning, error, or critical.
type AlertStoreAdapter struct {
	store         interfaces.SystemEventStore
	lookbackHours int
	recentLimit   int
}

// NewAlertStoreAdapter creates a new adapter for alerts.
// Accepts lookbackHours and recentLimit from config.
func NewAlertStoreAdapter(store interfaces.SystemEventStore, lookbackHours, recentLimit int) *AlertStoreAdapter {
	return &AlertStoreAdapter{
		store:         store,
		lookbackHours: lookbackHours,
		recentLimit:   recentLimit,
	}
}

// List returns alerts (high-severity events) for the given tenant.
func (a *AlertStoreAdapter) List(ctx context.Context, tenantID int64, filter *alertsservice.AlertFilter, limit, offset int) ([]*alertsservice.Alert, int64, error) {
	dbFilter := interfaces.AlertFilter{
		TenantID:   strconv.FormatInt(tenantID, 10),
		Severities: alertsservice.AlertSeverities, // Use centralized constant
		Limit:      limit,
		Offset:     offset,
	}

	if filter != nil {
		if len(filter.Severity) > 0 {
			dbFilter.Severities = filter.Severity
		}
		if filter.StartTime != nil {
			t := time.Unix(*filter.StartTime, 0)
			dbFilter.Since = &t
		}
	}

	events, err := a.store.GetActiveAlerts(ctx, dbFilter)
	if err != nil {
		return nil, 0, err
	}

	// Convert to alertsservice.Alert
	alerts := make([]*alertsservice.Alert, len(events))
	for i, e := range events {
		alerts[i] = &alertsservice.Alert{
			ID:          parseEventID(e.ID),
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

	// Get count
	countFilter := dbFilter
	countFilter.Limit = 0
	countFilter.Offset = 0
	total, err := a.store.CountActiveAlerts(ctx, countFilter)
	if err != nil {
		return alerts, int64(len(alerts)), nil // Return what we have
	}

	return alerts, total, nil
}

// GetSummary returns alert counts by severity.
func (a *AlertStoreAdapter) GetSummary(ctx context.Context, tenantID int64) (*alertsservice.AlertSummary, error) {
	tenantStr := strconv.FormatInt(tenantID, 10)
	since := time.Now().Add(-time.Duration(a.lookbackHours) * time.Hour)

	stats, err := a.store.GetEventStats(ctx, tenantStr, since)
	if err != nil {
		return &alertsservice.AlertSummary{}, nil // Return empty on error
	}

	summary := &alertsservice.AlertSummary{
		Critical: safeInt32(stats.EventsBySeverity[models.EventSeverityCritical]),
		Warning:  safeInt32(stats.EventsBySeverity[models.EventSeverityWarning]),
		Info:     safeInt32(stats.EventsBySeverity[models.EventSeverityInfo]),
	}

	// Get recent alerts using config limit
	recentFilter := interfaces.AlertFilter{
		TenantID:   tenantStr,
		Severities: alertsservice.AlertSeverities, // Use centralized constant
		Since:      &since,
		Limit:      a.recentLimit, // Use config value
	}
	recentEvents, _ := a.store.GetActiveAlerts(ctx, recentFilter)
	for _, e := range recentEvents {
		summary.Recent = append(summary.Recent, &alertsservice.Alert{
			ID:          parseEventID(e.ID),
			TenantID:    tenantID,
			Category:    e.Category,
			Severity:    e.Severity,
			Title:       e.Title,
			Description: e.Description,
			SourceName:  e.SourceName,
			Status:      e.Status,
			CreatedAt:   e.CreatedAt.Unix(),
		})
	}

	return summary, nil
}

// parseEventID converts string event ID to int64.
func parseEventID(id string) int64 {
	var result int64
	_, _ = fmt.Sscanf(id, "%d", &result)
	return result
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
