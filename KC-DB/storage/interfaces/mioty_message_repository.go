package interfaces

import (
	"context"
	"time"

	"github.com/kilocenter/KC-DB/storage/mioty"
)

// MIOTYMessageRepository provides MIOTY-specific message storage operations
type MIOTYMessageRepository interface {
	// CRUD Operations (8 methods)
	CreateULDataMessage(ctx context.Context, msg *mioty.ULDataMessage) error
	CreateDetachMessage(ctx context.Context, msg *mioty.DetachMessage, structuredMsg map[string]interface{}) error
	CreateAttachMessage(ctx context.Context, msg *mioty.AttachMessage, structuredMsg map[string]interface{}) error
	CreateAttachPropagateMessage(ctx context.Context, msg *mioty.AttachPropagateMessage) error
	CreateDetachPropagateMessage(ctx context.Context, msg *mioty.DetachPropagateMessage) error
	GetULDataMessage(ctx context.Context, id string, tenantID int64) (*mioty.ULDataMessage, error)
	GetDetachMessage(ctx context.Context, id string, tenantID int64) (*mioty.DetachMessage, error) // FIX-5
	ListULDataMessages(ctx context.Context, filter mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, int64, error)

	// UpdateULDataBaseStations merges multi-BS reception data on duplicate detection per SCACI §3.8.1
	// Tenant-scoped, time-window enforced via rxTimeMin to match deduplication window
	UpdateULDataBaseStations(ctx context.Context, tenantID int64, epEui uint64, packetCnt uint32, rxTimeMin int64, baseStationsJSON []byte) error

	// Basic Statistics (4 methods)
	GetMessageStatsByBaseStation(ctx context.Context, bsEui uint64, tenantID int64) (*mioty.MessageStats, error)
	GetExtendedMessageStatsByBaseStation(ctx context.Context, bsEui uint64, tenantID int64) (*mioty.MessageStats, error)
	GetMessageStatsByEndpoint(ctx context.Context, epEui uint64, tenantID int64) (*mioty.MessageStats, error)
	GetOverallStats(ctx context.Context, tenantID int64) (*mioty.MessageStats, error)

	// Analytics - Time Series (4 methods)
	GetAnalyticsOverview(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*mioty.AnalyticsOverviewStats, error)
	GetHourlyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.HourlyActivity, error)
	GetDailyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.DailyActivity, error)
	GetTopEndpointsByActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time, limit int) ([]mioty.EndpointActivity, error)

	// Signal Quality (2 methods)
	GetSignalQualityStats(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*mioty.SignalQualityStats, error)
	GetSignalQualityByBaseStation(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.BaseStationSignalQuality, error)

	// Base Station Statistics
	// GetBaseStationMessageStats retrieves aggregated message statistics for a base station
	GetBaseStationMessageStats(ctx context.Context, tenantID int64, bsEui []byte, startTime, endTime *time.Time) (*mioty.BaseStationMessageStats, error)

	// Uncapped message count aggregations for statistics
	GetMessageCountsByEndpoint(ctx context.Context, tenantID int64, startTime, endTime time.Time) (map[string]int64, error)
	GetMessageCountsByBaseStation(ctx context.Context, tenantID int64, startTime, endTime time.Time) (map[string]int64, error)

	// Time series at weekly/monthly granularity
	GetWeeklyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.WeeklyActivity, error)
	GetMonthlyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.MonthlyActivity, error)
}
