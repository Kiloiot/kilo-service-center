package interfaces

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// MIOTYMessageWriter persists protocol messages that are not uplinks (uplinks go
// through UplinkStore).
type MIOTYMessageWriter interface {
	CreateDetachMessage(ctx context.Context, msg *mioty.DetachMessage, structuredMsg map[string]interface{}) error
	CreateAttachPropagateMessage(ctx context.Context, msg *mioty.AttachPropagateMessage) error
	CreateDetachPropagateMessage(ctx context.Context, msg *mioty.DetachPropagateMessage) error
}

// MIOTYMessageReader reads stored uplink messages and per-base-station message
// statistics.
type MIOTYMessageReader interface {
	GetULDataMessage(ctx context.Context, id string, tenantID int64) (*mioty.ULDataMessage, error)
	ListULDataMessages(ctx context.Context, filter mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, int64, error)
	GetBaseStationMessageStats(ctx context.Context, tenantID int64, bsEui []byte, startTime, endTime *time.Time) (*mioty.BaseStationMessageStats, error)
}

// MIOTYMessageAnalytics aggregates tenant-wide message activity for dashboards
// and statistics.
type MIOTYMessageAnalytics interface {
	GetMessageStatsByEndpoint(ctx context.Context, epEui uint64, tenantID int64) (*mioty.MessageStats, error)
	GetOverallStats(ctx context.Context, tenantID int64) (*mioty.MessageStats, error)
	GetAnalyticsOverview(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*mioty.AnalyticsOverviewStats, error)
	GetHourlyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.HourlyActivity, error)
	GetDailyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.DailyActivity, error)
	GetTopEndpointsByActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time, limit int) ([]mioty.EndpointActivity, error)
	GetSignalQualityStats(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*mioty.SignalQualityStats, error)
	GetSignalQualityByBaseStation(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.BaseStationSignalQuality, error)
	GetMessageCountsByEndpoint(ctx context.Context, tenantID int64, startTime, endTime time.Time) (map[string]int64, error)
	GetMessageCountsByBaseStation(ctx context.Context, tenantID int64, startTime, endTime time.Time) (map[string]int64, error)
	GetWeeklyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.WeeklyActivity, error)
	GetMonthlyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.MonthlyActivity, error)
}

// MIOTYEndpointMessageStats aggregates the uplinks an endpoint sent from a
// point in time on, such as its current registration.
type MIOTYEndpointMessageStats interface {
	GetMessageStatsByEndpointSince(ctx context.Context, epEui uint64, tenantID int64, since time.Time) (*mioty.MessageStats, error)
}

// MIOTYMessageRepository is the message store surface; analytics live on the
// separate MIOTYMessageAnalytics repository.
type MIOTYMessageRepository interface {
	MIOTYMessageWriter
	MIOTYMessageReader
}
