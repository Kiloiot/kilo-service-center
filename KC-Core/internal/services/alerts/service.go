// Package alerts provides system alerting services.
package alerts

import (
	"context"
	"fmt"
	"slices"
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
	Severity []string
	Status   []string
}

// Alert represents a system alert from storage (internal format).
type Alert struct {
	ID          string
	TenantID    int64
	Category    string
	Severity    string
	Title       string
	Description string
	SourceName  string
	Status      string
	CreatedAt   int64
}

// AlertSummary represents alert counts (internal storage format).
type AlertSummary struct {
	Critical int32
	Error    int32
	Warning  int32
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
	if err := validateStatuses(filters); err != nil {
		return nil, 0, err
	}
	if err := validateSeverities(filters); err != nil {
		return nil, 0, err
	}
	// Convert grpcservices filters to internal format
	filter := convertFilters(filters)

	alerts, total, err := s.alertStore.List(ctx, tenantID, filter, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, LogAlertsListFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%w: %w", errListAlerts, err)
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
		s.logger.ErrorContext(ctx, LogAlertsSummaryFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", errGetAlertSummary, err)
	}

	// Convert recent alerts
	var recent []*grpcservices.Alert
	for _, a := range summary.Recent {
		recent = append(recent, convertAlert(a))
	}

	return &grpcservices.AlertSummary{
		Critical: summary.Critical,
		Error:    summary.Error,
		Warning:  summary.Warning,
		Recent:   recent,
	}, nil
}

// convertFilters converts grpcservices.AlertFilters to internal AlertFilter.
func convertFilters(filters *grpcservices.AlertFilters) *AlertFilter {
	if filters == nil {
		return nil
	}

	return &AlertFilter{
		Severity: filters.Severity,
		Status:   filters.Status,
	}
}

// convertAlert converts internal Alert to grpcservices.Alert.
func convertAlert(a *Alert) *grpcservices.Alert {
	return &grpcservices.Alert{
		ID:          a.ID,
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

// validateStatuses rejects a status filter outside AlertStatuses.
func validateStatuses(filters *grpcservices.AlertFilters) error {
	if filters == nil {
		return nil
	}
	for _, status := range filters.Status {
		if !slices.Contains(AlertStatuses, status) {
			return fmt.Errorf("%w: %s", ErrInvalidAlertStatus, status)
		}
	}
	return nil
}

// validateSeverities rejects a severity filter outside AlertSeverities.
func validateSeverities(filters *grpcservices.AlertFilters) error {
	if filters == nil {
		return nil
	}
	for _, severity := range filters.Severity {
		if !slices.Contains(AlertSeverities, severity) {
			return fmt.Errorf("%w: %s", ErrInvalidAlertSeverity, severity)
		}
	}
	return nil
}
