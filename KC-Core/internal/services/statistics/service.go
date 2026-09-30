// Package statistics provides aggregated statistics across endpoints, base stations, and messages.
package statistics

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// ErrUnsupportedGranularity is returned when the requested time series granularity is not recognized.
var ErrUnsupportedGranularity = errors.New("unsupported granularity")

// EndpointCounter counts a tenant's endpoints.
type EndpointCounter interface {
	CountByTenant(ctx context.Context, tenantID int64) (int64, error)
}

// BaseStationStatsReader reads aggregated base station statistics for a tenant.
type BaseStationStatsReader interface {
	GetStatistics(ctx context.Context, tenantID int64) (*models.BaseStationStatistics, error)
}

// MessageStatsReader reads aggregated message statistics and time series for a tenant.
type MessageStatsReader interface {
	GetOverallStats(ctx context.Context, tenantID int64) (*mioty.MessageStats, error)
	GetHourlyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.HourlyActivity, error)
	GetDailyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.DailyActivity, error)
	GetWeeklyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.WeeklyActivity, error)
	GetMonthlyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.MonthlyActivity, error)
	GetMessageCountsByEndpoint(ctx context.Context, tenantID int64, startTime, endTime time.Time) (map[string]int64, error)
	GetMessageCountsByBaseStation(ctx context.Context, tenantID int64, startTime, endTime time.Time) (map[string]int64, error)
}

// Service implements grpcservices.StatisticsService.
type Service struct {
	endpointRepo EndpointCounter
	bsRepo       BaseStationStatsReader
	msgRepo      MessageStatsReader
	logger       logger.Logger
}

// New creates a new statistics service.
func New(
	endpointRepo EndpointCounter,
	bsRepo BaseStationStatsReader,
	msgRepo MessageStatsReader,
	log logger.Logger,
) *Service {
	return &Service{
		endpointRepo: endpointRepo,
		bsRepo:       bsRepo,
		msgRepo:      msgRepo,
		logger:       log,
	}
}

// GetStatistics returns aggregated statistics for a tenant.
func (s *Service) GetStatistics(ctx context.Context, tenantID int64, startTime, endTime *time.Time, granularity string) (*grpcservices.StatisticsResult, error) {
	result := &grpcservices.StatisticsResult{
		EndpointMessageCounts:    make(map[string]int64),
		BaseStationMessageCounts: make(map[string]int64),
	}

	// Totals
	epCount, err := s.endpointRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		s.logger.ErrorContext(ctx, LogStatsCountEndpointsFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", errCountEndpoints, err)
	}
	result.TotalEndpoints = epCount

	bsStats, err := s.bsRepo.GetStatistics(ctx, tenantID)
	if err != nil {
		s.logger.ErrorContext(ctx, LogStatsBaseStationStatsFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", errBaseStationStats, err)
	}
	result.TotalBaseStations = bsStats.TotalCount

	msgStats, err := s.msgRepo.GetOverallStats(ctx, tenantID)
	if err != nil {
		s.logger.ErrorContext(ctx, LogStatsMessageStatsFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", errMessageStats, err)
	}
	result.TotalMessages = msgStats.TotalCount

	// Time series (message_counts)
	start, end := defaultTimeRange(startTime, endTime)
	timeSeries, err := s.getTimeSeries(ctx, tenantID, start, end, granularity)
	if err != nil {
		s.logger.ErrorContext(ctx, LogStatsTimeSeriesFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", errTimeSeries, err)
	}
	result.MessageCounts = timeSeries

	// Per-endpoint message counts (uncapped)
	epCounts, err := s.msgRepo.GetMessageCountsByEndpoint(ctx, tenantID, start, end)
	if err != nil {
		s.logger.ErrorContext(ctx, LogStatsEndpointMessageCountsFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", errEndpointMessageCounts, err)
	}
	result.EndpointMessageCounts = epCounts

	// Per-base-station message counts (uncapped)
	bsCounts, err := s.msgRepo.GetMessageCountsByBaseStation(ctx, tenantID, start, end)
	if err != nil {
		s.logger.ErrorContext(ctx, LogStatsBaseStationMessageCountsFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", errBaseStationMessageCounts, err)
	}
	result.BaseStationMessageCounts = bsCounts

	return result, nil
}

func (s *Service) getTimeSeries(ctx context.Context, tenantID int64, start, end time.Time, granularity string) ([]grpcservices.TimeSeriesPoint, error) {
	switch granularity {
	case granularityHour:
		hourly, err := s.msgRepo.GetHourlyActivity(ctx, tenantID, start, end)
		if err != nil {
			return nil, err
		}
		points := make([]grpcservices.TimeSeriesPoint, len(hourly))
		for i, h := range hourly {
			points[i] = grpcservices.TimeSeriesPoint{Timestamp: h.Hour, Value: int64(h.MessageCount)}
		}
		return points, nil
	case granularityDay, "":
		daily, err := s.msgRepo.GetDailyActivity(ctx, tenantID, start, end)
		if err != nil {
			return nil, err
		}
		points := make([]grpcservices.TimeSeriesPoint, len(daily))
		for i, d := range daily {
			t, err := time.Parse(time.DateOnly, d.Day)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", errParseActivityDay, err)
			}
			points[i] = grpcservices.TimeSeriesPoint{Timestamp: t, Value: int64(d.MessageCount)}
		}
		return points, nil
	case granularityWeek:
		weekly, err := s.msgRepo.GetWeeklyActivity(ctx, tenantID, start, end)
		if err != nil {
			return nil, err
		}
		points := make([]grpcservices.TimeSeriesPoint, len(weekly))
		for i, w := range weekly {
			points[i] = grpcservices.TimeSeriesPoint{Timestamp: w.Week, Value: int64(w.MessageCount)}
		}
		return points, nil
	case granularityMonth:
		monthly, err := s.msgRepo.GetMonthlyActivity(ctx, tenantID, start, end)
		if err != nil {
			return nil, err
		}
		points := make([]grpcservices.TimeSeriesPoint, len(monthly))
		for i, m := range monthly {
			points[i] = grpcservices.TimeSeriesPoint{Timestamp: m.Month, Value: int64(m.MessageCount)}
		}
		return points, nil
	default:
		return nil, fmt.Errorf("%w: %q; %s", ErrUnsupportedGranularity, granularity, supportedGranularitiesHint)
	}
}

// defaultStatisticsWindow is the lookback applied when the caller provides
// no explicit start time (last 30 days).
const defaultStatisticsWindow = 30 * 24 * time.Hour

func defaultTimeRange(startTime, endTime *time.Time) (time.Time, time.Time) {
	end := time.Now()
	start := end.Add(-defaultStatisticsWindow)
	if startTime != nil {
		start = *startTime
	}
	if endTime != nil {
		end = *endTime
	}
	return start, end
}

// Compile-time verification
var _ grpcservices.StatisticsService = (*Service)(nil)
