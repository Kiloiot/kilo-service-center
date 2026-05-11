// Package alerts provides system alerting services.
package alerts

import (
	"context"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// AlertStore provides alert persistence operations.
type AlertStore interface {
	List(ctx context.Context, tenantID int64, filter *AlertFilter, limit, offset int) ([]*Alert, int64, error)
	GetSummary(ctx context.Context, tenantID int64) (*AlertSummary, error)
}

// AlertFilter contains filtering criteria for alerts.
type AlertFilter struct {
	Severity  []string
	Status    []string
	StartTime *int64 // Unix timestamp
	EndTime   *int64 // Unix timestamp
}

// Alert represents a system alert from storage (internal format).
type Alert struct {
	ID          int64
	TenantID    int64
	Category    string
	Severity    string
	Title       string
	Description string
	SourceName  string
	Status      string
	CreatedAt   int64
	ResolvedAt  *int64
}

// AlertSummary represents alert counts (internal storage format).
type AlertSummary struct {
	Critical int32
	Warning  int32
	Info     int32
	Recent   []*Alert
}

// Service implements grpcservices.AlertService.
type Service struct {
	alertStore AlertStore
	logger     logger.Logger
}

// New creates a new alerts service.
func New(alertStore AlertStore, log logger.Logger) *Service {
	return &Service{
		alertStore: alertStore,
		logger:     log,
	}
}

// List returns alerts for the given tenant with optional filters.
func (s *Service) List(ctx context.Context, tenantID int64, filters *grpcservices.AlertFilters, limit, offset int) ([]*grpcservices.Alert, int64, error) {
	// Convert grpcservices filters to internal format
	filter := convertFilters(filters)

	alerts, total, err := s.alertStore.List(ctx, tenantID, filter, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list alerts", "tenantID", tenantID, "error", err)
		return nil, 0, fmt.Errorf("list alerts: %w", err)
	}

	result := make([]*grpcservices.Alert, len(alerts))
	for i, a := range alerts {
		result[i] = convertAlert(a)
	}

	return result, total, nil
}

// GetSummary returns alert summary statistics for the given tenant.
func (s *Service) GetSummary(ctx context.Context, tenantID int64) (*grpcservices.AlertSummary, error) {
	summary, err := s.alertStore.GetSummary(ctx, tenantID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get alert summary", "tenantID", tenantID, "error", err)
		return nil, fmt.Errorf("get alert summary: %w", err)
	}

	// Convert recent alerts
	var recent []*grpcservices.Alert
	for _, a := range summary.Recent {
		recent = append(recent, convertAlert(a))
	}

	return &grpcservices.AlertSummary{
		Critical: summary.Critical,
		Warning:  summary.Warning,
		Info:     summary.Info,
		Recent:   recent,
	}, nil
}

// convertFilters converts grpcservices.AlertFilters to internal AlertFilter.
func convertFilters(filters *grpcservices.AlertFilters) *AlertFilter {
	if filters == nil {
		return nil
	}

	filter := &AlertFilter{
		Severity: filters.Severity,
		Status:   filters.Status,
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

// convertAlert converts internal Alert to grpcservices.Alert.
func convertAlert(a *Alert) *grpcservices.Alert {
	return &grpcservices.Alert{
		ID:          fmt.Sprintf("%d", a.ID),
		TenantID:    a.TenantID,
		Category:    a.Category,
		Severity:    a.Severity,
		Title:       a.Title,
		Description: a.Description,
		SourceName:  a.SourceName,
		Timestamp:   time.Unix(a.CreatedAt, 0),
		Status:      a.Status,
	}
}

// Ensure Service implements grpcservices.AlertService
var _ grpcservices.AlertService = (*Service)(nil)
