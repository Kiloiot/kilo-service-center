// Package statistics provides aggregated statistics across endpoints, base stations, and messages.
package statistics

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kilocenter/KC-Core/internal/services/grpcservices"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/interfaces"
)

// ErrUnsupportedGranularity is returned when the requested time series granularity is not recognized.
var ErrUnsupportedGranularity = errors.New("unsupported granularity")

// Service implements grpcservices.StatisticsService.
type Service struct {
	endpointRepo interfaces.EndpointRepository
	bsRepo       interfaces.BaseStationRepository
	msgRepo      interfaces.MIOTYMessageRepository
	logger       logger.Logger
}

// New creates a new statistics service.
func New(
	endpointRepo interfaces.EndpointRepository,
	bsRepo interfaces.BaseStationRepository,
	msgRepo interfaces.MIOTYMessageRepository,
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
		s.logger.ErrorContext(ctx, "failed to count endpoints", "tenantID", tenantID, "error", err)
		return nil, fmt.Errorf("count endpoints: %w", err)
	}
	result.TotalEndpoints = epCount

	bsStats, err := s.bsRepo.GetStatistics(ctx, tenantID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get base station stats", "tenantID", tenantID, "error", err)
		return nil, fmt.Errorf("get base station stats: %w", err)
	}
	result.TotalBaseStations = bsStats.TotalCount

	msgStats, err := s.msgRepo.GetOverallStats(ctx, tenantID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get message stats", "tenantID", tenantID, "error", err)
		return nil, fmt.Errorf("get message stats: %w", err)
	}
	result.TotalMessages = msgStats.TotalCount

	// Time series (message_counts)
	start, end := defaultTimeRange(startTime, endTime)
	timeSeries, err := s.getTimeSeries(ctx, tenantID, start, end, granularity)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get time series", "tenantID", tenantID, "error", err)
		return nil, fmt.Errorf("get time series: %w", err)
	}
	result.MessageCounts = timeSeries

	// Per-endpoint message counts (uncapped)
	epCounts, err := s.msgRepo.GetMessageCountsByEndpoint(ctx, tenantID, start, end)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get endpoint message counts", "tenantID", tenantID, "error", err)
		return nil, fmt.Errorf("get endpoint message counts: %w", err)
	}
	result.EndpointMessageCounts = epCounts

	// Per-base-station message counts (uncapped)
	bsCounts, err := s.msgRepo.GetMessageCountsByBaseStation(ctx, tenantID, start, end)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get base station message counts", "tenantID", tenantID, "error", err)
		return nil, fmt.Errorf("get base station message counts: %w", err)
	}
	result.BaseStationMessageCounts = bsCounts

	return result, nil
}

func (s *Service) getTimeSeries(ctx context.Context, tenantID int64, start, end time.Time, granularity string) ([]grpcservices.TimeSeriesPoint, error) {
	switch granularity {
	case "hour":
		hourly, err := s.msgRepo.GetHourlyActivity(ctx, tenantID, start, end)
		if err != nil {
			return nil, err
		}
		points := make([]grpcservices.TimeSeriesPoint, len(hourly))
		for i, h := range hourly {
			points[i] = grpcservices.TimeSeriesPoint{Timestamp: h.Hour, Value: int64(h.MessageCount)}
		}
		return points, nil
	case "day", "":
		daily, err := s.msgRepo.GetDailyActivity(ctx, tenantID, start, end)
		if err != nil {
			return nil, err
		}
		points := make([]grpcservices.TimeSeriesPoint, len(daily))
		for i, d := range daily {
			t, _ := time.Parse("2006-01-02", d.Day)
			points[i] = grpcservices.TimeSeriesPoint{Timestamp: t, Value: int64(d.MessageCount)}
		}
		return points, nil
	case "week":
		weekly, err := s.msgRepo.GetWeeklyActivity(ctx, tenantID, start, end)
		if err != nil {
			return nil, err
		}
		points := make([]grpcservices.TimeSeriesPoint, len(weekly))
		for i, w := range weekly {
			points[i] = grpcservices.TimeSeriesPoint{Timestamp: w.Week, Value: int64(w.MessageCount)}
		}
		return points, nil
	case "month":
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
		return nil, fmt.Errorf("%w: %q; supported: hour, day, week, month", ErrUnsupportedGranularity, granularity)
	}
}

func defaultTimeRange(startTime, endTime *time.Time) (time.Time, time.Time) {
	end := time.Now()
	start := end.Add(-30 * 24 * time.Hour) // Default: last 30 days
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
