// Package analytics provides data analytics and aggregation services.
package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// MessageStore provides message analytics queries.
type MessageStore interface {
	GetAnalyticsOverview(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*Stats, error)
	GetDailyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]DailyActivity, error)
	GetSignalQualityStats(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*SignalStats, error)
	GetSignalQualityByBaseStation(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]BaseStationSignalStats, error)
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

// BaseStationSignalStats is the signal quality one base station received.
type BaseStationSignalStats struct {
	EUI          string
	AvgRSSI      float64
	AvgSNR       float64
	MessageCount int64
}

// Service implements grpcservices.AnalyticsService.
type Service struct {
	messageStore MessageStore
	logger       logger.Logger
	clock        clock.Clock
}

// New creates a new analytics service; clk ends an open window.
func New(messageStore MessageStore, log logger.Logger, clk clock.Clock) *Service {
	return &Service{
		messageStore: messageStore,
		logger:       log,
		clock:        clk,
	}
}

// GetOverview returns analytics overview for the given time range.
func (s *Service) GetOverview(ctx context.Context, tenantID int64, startTime, endTime *time.Time) (*grpcservices.AnalyticsOverview, error) {
	start, end := s.resolveWindow(startTime, endTime, defaultAnalyticsWindow)

	stats, err := s.messageStore.GetAnalyticsOverview(ctx, tenantID, start, end)
	if err != nil {
		s.logger.ErrorContext(ctx, LogAnalyticsOverviewFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", errAnalyticsOverview, err)
	}

	overview := &grpcservices.AnalyticsOverview{
		TotalMessages:      stats.TotalMessages,
		ActiveEndpoints:    stats.ActiveEndpoints,
		ActiveBaseStations: stats.ActiveBaseStations,
	}

	if stats.AvgRSSI != nil {
		overview.AverageRSSI = *stats.AvgRSSI
	}
	if stats.AvgSNR != nil {
		overview.AverageSNR = *stats.AvgSNR
	}

	return overview, nil
}

// GetActivity returns the window's message activity with one slot per day.
// The totals count distinct endpoints and base stations over the whole window.
func (s *Service) GetActivity(ctx context.Context, tenantID int64, startTime, endTime *time.Time, granularity string) (*grpcservices.ActivityAnalytics, error) {
	if granularity != "" && granularity != GranularityDay {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedGranularity, granularity)
	}
	start, end := s.resolveWindow(startTime, endTime, defaultActivityWindow)

	totals, err := s.messageStore.GetAnalyticsOverview(ctx, tenantID, start, end)
	if err != nil {
		s.logger.ErrorContext(ctx, LogAnalyticsActivityTotalsFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", errActivityTotals, err)
	}

	days, err := s.messageStore.GetDailyActivity(ctx, tenantID, start, end)
	if err != nil {
		s.logger.ErrorContext(ctx, LogAnalyticsDailyActivityFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", errDailyActivity, err)
	}

	slots := make([]grpcservices.ActivitySlot, len(days))
	for i, day := range days {
		slots[i] = grpcservices.ActivitySlot{
			Slot:          day.Day,
			MessageCount:  day.MessageCount,
			EndpointCount: day.UniqueEndpoints,
		}
	}

	return &grpcservices.ActivityAnalytics{
		StartTime:          start,
		EndTime:            end,
		TotalMessages:      totals.TotalMessages,
		UniqueEndpoints:    totals.ActiveEndpoints,
		UniqueBaseStations: totals.ActiveBaseStations,
		Slots:              slots,
	}, nil
}

// GetSignalQuality returns signal quality analytics for the given time range.
func (s *Service) GetSignalQuality(ctx context.Context, tenantID int64, startTime, endTime *time.Time) (*grpcservices.SignalQualityAnalytics, error) {
	start, end := s.resolveWindow(startTime, endTime, defaultAnalyticsWindow)

	stats, err := s.messageStore.GetSignalQualityStats(ctx, tenantID, start, end)
	if err != nil {
		s.logger.ErrorContext(ctx, LogAnalyticsSignalQualityFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", errSignalQualityStats, err)
	}

	stations, err := s.messageStore.GetSignalQualityByBaseStation(ctx, tenantID, start, end)
	if err != nil {
		s.logger.ErrorContext(ctx, LogAnalyticsSignalQualityByStationFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", errSignalQualityByStation, err)
	}

	byStation := make([]grpcservices.BaseStationSignalQuality, len(stations))
	for i, station := range stations {
		byStation[i] = grpcservices.BaseStationSignalQuality{
			EUI:          station.EUI,
			AverageRSSI:  station.AvgRSSI,
			AverageSNR:   station.AvgSNR,
			MessageCount: station.MessageCount,
		}
	}

	return &grpcservices.SignalQualityAnalytics{
		StartTime:     start,
		EndTime:       end,
		AverageRSSI:   stats.AvgRSSI,
		AverageSNR:    stats.AvgSNR,
		MedianRSSI:    stats.MedianRSSI,
		MedianSNR:     stats.MedianSNR,
		RSSIRange:     [2]float64{stats.MinRSSI, stats.MaxRSSI},
		SNRRange:      [2]float64{stats.MinSNR, stats.MaxSNR},
		ByBaseStation: byStation,
	}, nil
}

// resolveWindow fills an open window bound: the end defaults to now and the
// start to the lookback before the end.
func (s *Service) resolveWindow(startTime, endTime *time.Time, lookback time.Duration) (time.Time, time.Time) {
	end := s.clock.Now()
	if endTime != nil {
		end = *endTime
	}
	start := end.Add(-lookback)
	if startTime != nil {
		start = *startTime
	}
	return start, end
}

// Ensure Service implements grpcservices.AnalyticsService
var _ grpcservices.AnalyticsService = (*Service)(nil)

// GranularityDay is the only activity bucket width the message store aggregates by.
const GranularityDay = "day"

// defaultAnalyticsWindow is the lookback for overview and signal-quality
// queries when the caller provides no explicit start time (last 24 hours).
const defaultAnalyticsWindow = 24 * time.Hour

// defaultActivityWindow is the lookback for activity queries when the caller
// provides no explicit start time (last 7 days).
const defaultActivityWindow = 7 * 24 * time.Hour
