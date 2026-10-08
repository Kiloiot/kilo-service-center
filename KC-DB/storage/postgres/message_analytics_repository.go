package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// MessageAnalyticsRepository aggregates tenant-wide message activity.
type MessageAnalyticsRepository struct {
	db *sqlx.DB
}

var (
	_ interfaces.MIOTYMessageAnalytics     = (*MessageAnalyticsRepository)(nil)
	_ interfaces.MIOTYEndpointMessageStats = (*MessageAnalyticsRepository)(nil)
)

// sqlEndpointUplinkStatsFmt aggregates the uplinks of one endpoint: %[1]s
// binds the width of a day in nanoseconds, which counts the UTC days they
// were received on, and %[2]s is the uplink predicate.
const sqlEndpointUplinkStatsFmt = `
	SELECT COUNT(*), COUNT(DISTINCT bs_eui), COALESCE(AVG(rssi), 0), COALESCE(AVG(snr), 0),
		MIN(rx_time), MAX(rx_time), COUNT(DISTINCT rx_time / %[1]s)
	FROM messages
	WHERE %[2]s`

// nanosPerDay is the width of a UTC day in rx_time units.
const nanosPerDay = int64(24 * time.Hour)

// The tenant-wide analytics count the frames the tenant's endpoints
// transmitted; the propagations the service center sends share the table.
// $2 binds endpointFrameCommands and the window follows at $3 and $4.
const (
	sqlTenantEndpointFrames = `tenant_id = $1 AND command_type = ANY($2)`
	sqlFramesInWindow       = sqlTenantEndpointFrames + ` AND received_at BETWEEN $3 AND $4`
)

// endpointFrameCommands binds mioty.EndpointFrameCommands as one array.
func endpointFrameCommands() interface{} {
	return pq.Array(mioty.EndpointFrameCommands())
}

// NewMessageAnalyticsRepository creates the analytics repository on the given connection.
func NewMessageAnalyticsRepository(db *sqlx.DB) *MessageAnalyticsRepository {
	return &MessageAnalyticsRepository{db: db}
}

// GetMessageStatsByEndpoint returns the statistics of every stored uplink
// of the endpoint.
func (r *MessageAnalyticsRepository) GetMessageStatsByEndpoint(ctx context.Context, epEui uint64, tenantID int64) (*mioty.MessageStats, error) {
	stats, err := r.endpointUplinkStats(ctx, mioty.ULDataMessageFilter{TenantID: tenantID, EpEui: &epEui})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetMessageStatsByEndpoint, err)
	}
	return stats, nil
}

// GetMessageStatsByEndpointSince returns the statistics of the endpoint's
// uplinks received at or after since.
func (r *MessageAnalyticsRepository) GetMessageStatsByEndpointSince(ctx context.Context, epEui uint64, tenantID int64, since time.Time) (*mioty.MessageStats, error) {
	stats, err := r.endpointUplinkStats(ctx, mioty.ULDataMessageFilter{TenantID: tenantID, EpEui: &epEui, StartTime: &since})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetMessageStatsByEndpointSince, err)
	}
	return stats, nil
}

// endpointUplinkStats aggregates the uplinks the endpoint's uplink listing
// selects, so every figure agrees with it.
func (r *MessageAnalyticsRepository) endpointUplinkStats(ctx context.Context, filter mioty.ULDataMessageFilter) (*mioty.MessageStats, error) {
	scope := ulDataWhere(filter)
	query := fmt.Sprintf(sqlEndpointUplinkStatsFmt, scope.bind(nanosPerDay), scope.where())
	var stats mioty.MessageStats
	var first, last sql.NullInt64
	var activeDays int
	err := r.db.QueryRowContext(ctx, query, scope.args...).Scan(
		&stats.TotalCount,
		&stats.UniqueEndpoints, // unique base stations for a single endpoint
		&stats.AvgRSSI,
		&stats.AvgSNR,
		&first, &last, &activeDays,
	)
	if err != nil {
		return nil, err
	}
	stats.FirstSeen = nullUnixNanosPtr(first)
	stats.LastSeen = nullUnixNanosPtr(last)
	stats.ActiveDays = &activeDays
	return &stats, nil
}

// GetOverallStats returns overall message statistics
func (r *MessageAnalyticsRepository) GetOverallStats(ctx context.Context, tenantID int64) (*mioty.MessageStats, error) {
	query := `
		SELECT
			COUNT(*) as total_count,
			COUNT(DISTINCT ep_eui) as unique_endpoints,
			COALESCE(AVG(rssi), 0) as avg_rssi,
			COALESCE(AVG(snr), 0) as avg_snr
		FROM messages
		WHERE ` + sqlTenantEndpointFrames

	var stats mioty.MessageStats
	err := r.db.QueryRowContext(ctx, query, tenantID, endpointFrameCommands()).Scan(
		&stats.TotalCount,
		&stats.UniqueEndpoints,
		&stats.AvgRSSI,
		&stats.AvgSNR,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetOverallStats, err)
	}

	return &stats, nil
}

// GetAnalyticsOverview returns overview analytics within a time range
func (r *MessageAnalyticsRepository) GetAnalyticsOverview(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*mioty.AnalyticsOverviewStats, error) {
	query := `
		SELECT
			COUNT(*) as total_messages,
			COUNT(DISTINCT ep_eui) as active_endpoints,
			COUNT(DISTINCT bs_eui) as active_basestations,
			AVG(rssi) as avg_rssi,
			AVG(snr) as avg_snr,
			MIN(received_at) as first_message,
			MAX(received_at) as last_message
		FROM messages
		WHERE ` + sqlFramesInWindow

	var stats mioty.AnalyticsOverviewStats
	err := r.db.QueryRowContext(ctx, query, tenantID, endpointFrameCommands(), startTime, endTime).Scan(
		&stats.TotalMessages,
		&stats.ActiveEndpoints,
		&stats.ActiveBaseStations,
		&stats.AvgRSSI,
		&stats.AvgSNR,
		&stats.FirstMessage,
		&stats.LastMessage,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetAnalyticsOverview, err)
	}

	return &stats, nil
}

// GetHourlyActivity returns hourly message counts within a time range
func (r *MessageAnalyticsRepository) GetHourlyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.HourlyActivity, error) {
	query := `
		SELECT
			DATE_TRUNC('hour', received_at) as hour,
			COUNT(*) as message_count
		FROM messages
		WHERE ` + sqlFramesInWindow + `
		GROUP BY hour
		ORDER BY hour
	`

	var activities []mioty.HourlyActivity
	err := r.db.SelectContext(ctx, &activities, query, tenantID, endpointFrameCommands(), startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetHourlyActivity, err)
	}

	return activities, nil
}

// GetDailyActivity returns daily activity statistics within a time range
func (r *MessageAnalyticsRepository) GetDailyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.DailyActivity, error) {
	query := `
		SELECT
			DATE(received_at) as day,
			COUNT(*) as message_count,
			COUNT(DISTINCT ep_eui) as unique_endpoints,
			COUNT(DISTINCT bs_eui) as unique_basestations
		FROM messages
		WHERE ` + sqlFramesInWindow + `
		GROUP BY day
		ORDER BY day
	`

	var activities []mioty.DailyActivity
	err := r.db.SelectContext(ctx, &activities, query, tenantID, endpointFrameCommands(), startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetDailyActivity, err)
	}

	return activities, nil
}

// GetTopEndpointsByActivity returns the most active endpoints within a time range
func (r *MessageAnalyticsRepository) GetTopEndpointsByActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time, limit int) (endpoints []mioty.EndpointActivity, err error) {
	query := `
		SELECT
			ep_eui,
			COUNT(*) as message_count,
			MAX(received_at) as last_seen
		FROM messages
		WHERE ` + sqlFramesInWindow + `
		GROUP BY ep_eui
		ORDER BY message_count DESC
		LIMIT $5
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, endpointFrameCommands(), startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetTopEndpointsByActivity, err)
	}
	defer sqlcleanup.CloseRows(rows, errWrapIterateTopEndpoints, &err)

	for rows.Next() {
		var ep mioty.EndpointActivity
		var epEuiB []byte
		if err := rows.Scan(&epEuiB, &ep.MessageCount, &ep.LastSeen); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanTopEndpoint, err)
		}
		ep.EUI = mioty.EUI64FromBytes(epEuiB)
		endpoints = append(endpoints, ep)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateTopEndpoints, err)
	}

	return endpoints, nil
}

// GetSignalQualityStats returns signal quality statistics within a time range
func (r *MessageAnalyticsRepository) GetSignalQualityStats(ctx context.Context, tenantID int64, startTime, endTime time.Time) (*mioty.SignalQualityStats, error) {
	query := `
		SELECT
			COALESCE(AVG(rssi), 0) as avg_rssi,
			COALESCE(MIN(rssi), 0) as min_rssi,
			COALESCE(MAX(rssi), 0) as max_rssi,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY rssi), 0) as median_rssi,
			COALESCE(AVG(snr), 0) as avg_snr,
			COALESCE(MIN(snr), 0) as min_snr,
			COALESCE(MAX(snr), 0) as max_snr,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY snr), 0) as median_snr,
			COUNT(*) as total_messages
		FROM messages
		WHERE ` + sqlFramesInWindow

	var stats mioty.SignalQualityStats
	err := r.db.QueryRowContext(ctx, query, tenantID, endpointFrameCommands(), startTime, endTime).Scan(
		&stats.AvgRSSI,
		&stats.MinRSSI,
		&stats.MaxRSSI,
		&stats.MedianRSSI,
		&stats.AvgSNR,
		&stats.MinSNR,
		&stats.MaxSNR,
		&stats.MedianSNR,
		&stats.TotalMessages,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetSignalQualityStats, err)
	}

	return &stats, nil
}

// GetSignalQualityByBaseStation returns the signal quality of the tenant's
// uplinks per receiving base station, busiest station first.
func (r *MessageAnalyticsRepository) GetSignalQualityByBaseStation(ctx context.Context, tenantID int64, startTime, endTime time.Time) (stations []mioty.BaseStationSignalQuality, err error) {
	query := `
		SELECT
			bs_eui,
			AVG(rssi) as avg_rssi,
			AVG(snr) as avg_snr,
			COUNT(*) as message_count
		FROM messages
		WHERE ` + sqlFramesInWindow + `
		GROUP BY bs_eui
		ORDER BY message_count DESC, bs_eui`

	rows, err := r.db.QueryContext(ctx, query, tenantID, endpointFrameCommands(), startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetSignalQualityByBaseStation, err)
	}
	defer sqlcleanup.CloseRows(rows, errWrapIterateSignalQuality, &err)

	for rows.Next() {
		var eui []byte
		var station mioty.BaseStationSignalQuality
		if err := rows.Scan(&eui, &station.AvgRSSI, &station.AvgSNR, &station.MessageCount); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanSignalQuality, err)
		}
		station.BsEui = mioty.EUI64FromBytes(eui)
		stations = append(stations, station)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateSignalQuality, err)
	}
	return stations, nil
}

// GetMessageCountsByEndpoint returns uncapped message counts grouped by endpoint EUI.
func (r *MessageAnalyticsRepository) GetMessageCountsByEndpoint(ctx context.Context, tenantID int64, startTime, endTime time.Time) (counts map[string]int64, err error) {
	query := `
		SELECT ep_eui, COUNT(*) as message_count
		FROM messages
		WHERE ` + sqlFramesInWindow + `
		GROUP BY ep_eui`

	rows, err := r.db.QueryContext(ctx, query, tenantID, endpointFrameCommands(), startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetMessageCountsByEndpoint, err)
	}
	defer sqlcleanup.CloseRows(rows, errWrapIterateEndpointMessageCounts, &err)

	counts = make(map[string]int64)
	for rows.Next() {
		var eui []byte
		var count int64
		if err := rows.Scan(&eui, &count); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanEndpointMessageCount, err)
		}
		counts[mioty.FormatEUI64(mioty.EUI64FromBytes(eui))] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateEndpointMessageCounts, err)
	}

	return counts, nil
}

// GetWeeklyActivity returns message counts grouped by ISO week.
func (r *MessageAnalyticsRepository) GetWeeklyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.WeeklyActivity, error) {
	query := `
		SELECT DATE_TRUNC('week', received_at) AS week, COUNT(*) AS message_count
		FROM messages
		WHERE ` + sqlFramesInWindow + `
		GROUP BY week
		ORDER BY week`

	var activity []mioty.WeeklyActivity
	err := r.db.SelectContext(ctx, &activity, query, tenantID, endpointFrameCommands(), startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetWeeklyActivity, err)
	}
	return activity, nil
}

// GetMonthlyActivity returns message counts grouped by month.
func (r *MessageAnalyticsRepository) GetMonthlyActivity(ctx context.Context, tenantID int64, startTime, endTime time.Time) ([]mioty.MonthlyActivity, error) {
	query := `
		SELECT DATE_TRUNC('month', received_at) AS month, COUNT(*) AS message_count
		FROM messages
		WHERE ` + sqlFramesInWindow + `
		GROUP BY month
		ORDER BY month`

	var activity []mioty.MonthlyActivity
	err := r.db.SelectContext(ctx, &activity, query, tenantID, endpointFrameCommands(), startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetMonthlyActivity, err)
	}
	return activity, nil
}

// GetMessageCountsByBaseStation returns uncapped message counts grouped by base station EUI.
func (r *MessageAnalyticsRepository) GetMessageCountsByBaseStation(ctx context.Context, tenantID int64, startTime, endTime time.Time) (counts map[string]int64, err error) {
	query := `
		SELECT bs_eui, COUNT(*) as message_count
		FROM messages
		WHERE ` + sqlFramesInWindow + `
		GROUP BY bs_eui`

	rows, err := r.db.QueryContext(ctx, query, tenantID, endpointFrameCommands(), startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetMessageCountsByBaseStation, err)
	}
	defer sqlcleanup.CloseRows(rows, errWrapIterateBaseStationMessageCounts, &err)

	counts = make(map[string]int64)
	for rows.Next() {
		var eui []byte
		var count int64
		if err := rows.Scan(&eui, &count); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanBaseStationMessageCount, err)
		}
		counts[mioty.FormatEUI64(mioty.EUI64FromBytes(eui))] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateBaseStationMessageCounts, err)
	}

	return counts, nil
}
