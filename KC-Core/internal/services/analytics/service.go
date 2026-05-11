// Package analytics provides data analytics and aggregation services.
package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// MessageStore provides message analytics queries.
type MessageStore interface {
	GetAnalyticsOverview(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*Stats, error)
	GetDailyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]DailyActivity, error)
	GetSignalQualityStats(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*SignalStats, error)
	GetTopEndpointsByActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time, limit int) ([]EndpointActivityStats, error)
}

// Stats represents analytics overview statistics.
type Stats struct {
	TotalMessages      int64
	ActiveEndpoints    int64
	ActiveBaseStations int64
	AvgRSSI            *float64
	AvgSNR             *float64
	FirstMessage       *time.Time
	LastMessage        *time.Time
}

// DailyActivity represents daily message activity.
type DailyActivity struct {
	Day                time.Time
	MessageCount       int64
	UniqueEndpoints    int64
	UniqueBaseStations int64
}

// SignalStats represents signal quality statistics.
type SignalStats struct {
	AvgRSSI       float64
	MinRSSI       float64
	MaxRSSI       float64
	MedianRSSI    float64
	AvgSNR        float64
	MinSNR        float64
	MaxSNR        float64
	MedianSNR     float64
	TotalMessages int64
}

// EndpointActivityStats represents endpoint activity metrics.
type EndpointActivityStats struct {
	EUI          uint64
	EUIFormatted string
	MessageCount int64
	LastSeen     *time.Time
}

// Service implements grpcservices.AnalyticsService.
type Service struct {
	messageStore MessageStore
	logger       logger.Logger
}

// New creates a new analytics service.
func New(messageStore MessageStore, log logger.Logger) *Service {
	return &Service{
		messageStore: messageStore,
		logger:       log,
	}
}

// GetOverview returns analytics overview for the given time range.
func (s *Service) GetOverview(ctx context.Context, tenantID int64, startTime, endTime *time.Time) (*grpcservices.AnalyticsOverview, error) {
	// Default to last 24 hours if not specified
	end := time.Now()
	start := end.Add(-24 * time.Hour)
	if startTime != nil {
		start = *startTime
	}
	if endTime != nil {
		end = *endTime
	}

	stats, err := s.messageStore.GetAnalyticsOverview(ctx, tenantID, start, end)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get analytics overview", "tenantID", tenantID, "error", err)
		return nil, fmt.Errorf("get analytics overview: %w", err)
	}

	overview := &grpcservices.AnalyticsOverview{
		TotalMessages:     stats.TotalMessages,
		ActiveEndpoints:   stats.ActiveEndpoints,
		TotalBaseStations: stats.ActiveBaseStations,
	}

	if stats.AvgRSSI != nil {
		overview.AverageRSSI = *stats.AvgRSSI
	}
	if stats.AvgSNR != nil {
		overview.AverageSNR = *stats.AvgSNR
	}

	return overview, nil
}

// GetActivity returns activity analytics for the given time range.
func (s *Service) GetActivity(ctx context.Context, tenantID int64, startTime, endTime *time.Time) (*grpcservices.ActivityAnalytics, error) {
	// Default to last 7 days if not specified
	end := time.Now()
	start := end.Add(-7 * 24 * time.Hour)
	if startTime != nil {
		start = *startTime
	}
	if endTime != nil {
		end = *endTime
	}

	// Get daily activity
	dailyActivities, err := s.messageStore.GetDailyActivity(ctx, tenantID, start, end)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get daily activity", "tenantID", tenantID, "error", err)
		return nil, fmt.Errorf("get daily activity: %w", err)
	}

	// Get top endpoints
	topEndpoints, err := s.messageStore.GetTopEndpointsByActivity(ctx, tenantID, start, end, 10)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get top endpoints", "tenantID", tenantID, "error", err)
		return nil, fmt.Errorf("get top endpoints: %w", err)
	}

	activity := &grpcservices.ActivityAnalytics{
		MessagesPerHour:  make(map[string]int64),
		EndpointActivity: make(map[string]int64),
		TopEndpoints:     make([]grpcservices.EndpointActivity, 0, len(topEndpoints)),
	}

	// Populate daily stats as messages per day (hour granularity requires different query)
	for _, day := range dailyActivities {
		activity.MessagesPerHour[day.Day.Format("2006-01-02")] = day.MessageCount
	}

	// Populate top endpoints
	for _, ep := range topEndpoints {
		activity.TopEndpoints = append(activity.TopEndpoints, grpcservices.EndpointActivity{
			EPEUI:        ep.EUIFormatted,
			MessageCount: ep.MessageCount,
			LastSeen:     ep.LastSeen,
		})
		activity.EndpointActivity[ep.EUIFormatted] = ep.MessageCount
	}

	return activity, nil
}

// GetSignalQuality returns signal quality analytics for the given time range.
func (s *Service) GetSignalQuality(ctx context.Context, tenantID int64, startTime, endTime *time.Time) (*grpcservices.SignalQualityAnalytics, error) {
	// Default to last 24 hours if not specified
	end := time.Now()
	start := end.Add(-24 * time.Hour)
	if startTime != nil {
		start = *startTime
	}
	if endTime != nil {
		end = *endTime
	}

	stats, err := s.messageStore.GetSignalQualityStats(ctx, tenantID, start, end)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get signal quality stats", "tenantID", tenantID, "error", err)
		return nil, fmt.Errorf("get signal quality stats: %w", err)
	}

	// Determine quality rating based on average RSSI
	quality := "poor"
	if stats.AvgRSSI > -70 {
		quality = "excellent"
	} else if stats.AvgRSSI > -85 {
		quality = "good"
	} else if stats.AvgRSSI > -100 {
		quality = "fair"
	}

	return &grpcservices.SignalQualityAnalytics{
		AverageRSSI: stats.AvgRSSI,
		AverageSNR:  stats.AvgSNR,
		RSSIRange:   [2]float64{stats.MinRSSI, stats.MaxRSSI},
		SNRRange:    [2]float64{stats.MinSNR, stats.MaxSNR},
		Quality:     quality,
	}, nil
}

// Ensure Service implements grpcservices.AnalyticsService
var _ grpcservices.AnalyticsService = (*Service)(nil)
