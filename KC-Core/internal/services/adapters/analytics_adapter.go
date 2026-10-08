// Package adapters provides storage adapters bridging KC-DB to gRPC services.
package adapters

import (
	"context"
	"errors"
	"fmt"
	"time"

	analyticsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/analytics"
	miotyformat "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// errParseActivityDay wraps a daily activity row whose day does not parse.
var errParseActivityDay = errors.New("parse daily activity day")

// analyticsMessageStore covers the aggregate queries the analytics adapter
// reads from the message repository. Satisfied structurally by the KC-DB
// MIOTY message repository.
type analyticsMessageStore interface {
	GetAnalyticsOverview(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*mioty.AnalyticsOverviewStats, error)
	GetDailyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.DailyActivity, error)
	GetSignalQualityStats(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*mioty.SignalQualityStats, error)
	GetSignalQualityByBaseStation(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.BaseStationSignalQuality, error)
}

// AnalyticsMessageStoreAdapter adapts the KC-DB message repository to analyticsservice.MessageStore.
// Uses canonical formatters for EUI display.
type AnalyticsMessageStoreAdapter struct {
	repo analyticsMessageStore
}

// NewAnalyticsMessageStoreAdapter creates a new adapter for analytics.
func NewAnalyticsMessageStoreAdapter(repo analyticsMessageStore) *AnalyticsMessageStoreAdapter {
	return &AnalyticsMessageStoreAdapter{repo: repo}
}

// GetAnalyticsOverview returns analytics overview statistics.
func (a *AnalyticsMessageStoreAdapter) GetAnalyticsOverview(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*analyticsservice.Stats, error) {
	stats, err := a.repo.GetAnalyticsOverview(ctx, tenantID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	return &analyticsservice.Stats{
		TotalMessages:      stats.TotalMessages,
		ActiveEndpoints:    stats.ActiveEndpoints,
		ActiveBaseStations: stats.ActiveBaseStations,
		AvgRSSI:            stats.AvgRSSI,
		AvgSNR:             stats.AvgSNR,
		FirstMessage:       stats.FirstMessage,
		LastMessage:        stats.LastMessage,
	}, nil
}

// GetDailyActivity returns daily message activity.
func (a *AnalyticsMessageStoreAdapter) GetDailyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]analyticsservice.DailyActivity, error) {
	activities, err := a.repo.GetDailyActivity(ctx, tenantID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	result := make([]analyticsservice.DailyActivity, len(activities))
	for i, act := range activities {
		day, err := time.Parse(miotyformat.DateFormat, act.Day)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errParseActivityDay, err)
		}
		result[i] = analyticsservice.DailyActivity{
			Day:                day,
			MessageCount:       int64(act.MessageCount),
			UniqueEndpoints:    int64(act.UniqueEndpoints),
			UniqueBaseStations: int64(act.UniqueBaseStations),
		}
	}

	return result, nil
}

// GetSignalQualityStats returns signal quality statistics.
func (a *AnalyticsMessageStoreAdapter) GetSignalQualityStats(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*analyticsservice.SignalStats, error) {
	stats, err := a.repo.GetSignalQualityStats(ctx, tenantID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	return &analyticsservice.SignalStats{
		AvgRSSI:       stats.AvgRSSI,
		MinRSSI:       stats.MinRSSI,
		MaxRSSI:       stats.MaxRSSI,
		MedianRSSI:    stats.MedianRSSI,
		AvgSNR:        stats.AvgSNR,
		MinSNR:        stats.MinSNR,
		MaxSNR:        stats.MaxSNR,
		MedianSNR:     stats.MedianSNR,
		TotalMessages: stats.TotalMessages,
	}, nil
}

// GetSignalQualityByBaseStation returns the signal quality per receiving base station.
func (a *AnalyticsMessageStoreAdapter) GetSignalQualityByBaseStation(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]analyticsservice.BaseStationSignalStats, error) {
	stations, err := a.repo.GetSignalQualityByBaseStation(ctx, tenantID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	result := make([]analyticsservice.BaseStationSignalStats, len(stations))
	for i, station := range stations {
		result[i] = analyticsservice.BaseStationSignalStats{
			EUI:          mioty.FormatEUI64(station.BsEui),
			AvgRSSI:      station.AvgRSSI,
			AvgSNR:       station.AvgSNR,
			MessageCount: station.MessageCount,
		}
	}

	return result, nil
}

// Ensure AnalyticsMessageStoreAdapter implements analyticsservice.MessageStore
var _ analyticsservice.MessageStore = (*AnalyticsMessageStoreAdapter)(nil)
