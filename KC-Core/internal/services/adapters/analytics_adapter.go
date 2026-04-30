// Package adapters provides storage adapters bridging KC-DB to gRPC services.
package adapters

import (
	"context"
	"time"

	analyticsservice "github.com/kilocenter/KC-Core/internal/services/analytics"
	miotyformat "github.com/kilocenter/KC-Core/pkg/mioty"
	"github.com/kilocenter/KC-DB/storage/interfaces"
)

// AnalyticsMessageStoreAdapter adapts interfaces.MIOTYMessageRepository to analyticsservice.MessageStore.
// Uses canonical formatters for EUI display.
type AnalyticsMessageStoreAdapter struct {
	repo interfaces.MIOTYMessageRepository
}

// NewAnalyticsMessageStoreAdapter creates a new adapter for analytics.
func NewAnalyticsMessageStoreAdapter(repo interfaces.MIOTYMessageRepository) *AnalyticsMessageStoreAdapter {
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
		day, _ := time.Parse(miotyformat.DateFormat, act.Day)
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

// GetTopEndpointsByActivity returns top endpoints by message count.
func (a *AnalyticsMessageStoreAdapter) GetTopEndpointsByActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time, limit int) ([]analyticsservice.EndpointActivityStats, error) {
	activities, err := a.repo.GetTopEndpointsByActivity(ctx, tenantID, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}

	result := make([]analyticsservice.EndpointActivityStats, len(activities))
	for i, act := range activities {
		result[i] = analyticsservice.EndpointActivityStats{
			EUI:          act.EUI,
			EUIFormatted: miotyformat.FormatEUI64(act.EUI), // Use canonical formatter
			MessageCount: int64(act.MessageCount),
			LastSeen:     &act.LastSeen,
		}
	}

	return result, nil
}

// Ensure AnalyticsMessageStoreAdapter implements analyticsservice.MessageStore
var _ analyticsservice.MessageStore = (*AnalyticsMessageStoreAdapter)(nil)
