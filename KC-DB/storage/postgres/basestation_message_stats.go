package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// sqlStationUplinkStatsFmt aggregates the uplinks a station received with
// the radio values of its own reception; an uplink stored without receptions
// keeps its primary values. %[1]s binds the station's containment probe and
// %[2]s is the uplink predicate.
const sqlStationUplinkStatsFmt = `
	SELECT COUNT(*), COUNT(DISTINCT ep_eui),
		COALESCE(AVG(COALESCE(reception.station_rssi, rssi)), 0),
		COALESCE(AVG(COALESCE(reception.station_snr, snr)), 0),
		MIN(rx_time), MAX(rx_time)
	FROM messages
	LEFT JOIN LATERAL (
		SELECT (elem->>'rssi')::double precision AS station_rssi,
			(elem->>'snr')::double precision AS station_snr
		FROM jsonb_array_elements(messages.base_stations) AS elem
		WHERE jsonb_build_array(elem) @> %[1]s::jsonb
		LIMIT 1
	) AS reception ON TRUE
	WHERE %[2]s`

// Statements over an uplink predicate, which follows the WHERE.
const (
	sqlCountMessagesWhere   = `SELECT COUNT(*) FROM messages WHERE `
	sqlCountByEndpointWhere = `SELECT ep_eui, COUNT(*) FROM messages WHERE `
	sqlGroupByEndpoint      = ` GROUP BY ep_eui`
	sqlLastRxTimeWhere      = `SELECT MAX(rx_time) FROM messages WHERE `
)

// nullUnixNanosPtr is the instant of a nullable rx_time.
func nullUnixNanosPtr(nanos sql.NullInt64) *time.Time {
	if !nanos.Valid {
		return nil
	}
	instant := time.Unix(0, nanos.Int64)
	return &instant
}

// stationUplinks selects the uplinks the station received in the window:
// the predicate of the station's uplink listing, so every figure agrees
// with it. A nil bound leaves that side open.
func stationUplinks(tenantID int64, bsEui []byte, start, end *time.Time) *sqlScope {
	eui := mioty.EUI64FromBytes(bsEui)
	return ulDataWhere(mioty.ULDataMessageFilter{TenantID: tenantID, BsEui: &eui, StartTime: start, EndTime: end})
}

// GetBaseStationMessageStats aggregates the uplinks the base station
// received, as the primary receiver or as one of the receptions, within the
// window; a nil bound leaves it open. Today, this week and this month ignore
// the window.
func (r *MessageRepository) GetBaseStationMessageStats(ctx context.Context, tenantID int64, bsEui []byte,
	startTime, endTime *time.Time,
) (*mioty.BaseStationMessageStats, error) {
	scope := stationUplinks(tenantID, bsEui, startTime, endTime)
	query := fmt.Sprintf(sqlStationUplinkStatsFmt, scope.bind(stationContainment(bsEui)), scope.where())
	var stats mioty.BaseStationMessageStats
	var first, last sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, scope.args...).Scan(
		&stats.TotalMessages, &stats.TotalEndpoints, &stats.AvgRSSI, &stats.AvgSNR, &first, &last)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetBaseStationMessageStats, err)
	}
	stats.FirstMessageAt = nullUnixNanosPtr(first)
	stats.LastMessageAt = nullUnixNanosPtr(last)
	if err := r.countStationPeriods(ctx, tenantID, bsEui, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// countStationPeriods counts the uplinks the base station received since the
// start of today, of this week and of this month.
func (r *MessageRepository) countStationPeriods(ctx context.Context, tenantID int64, bsEui []byte, stats *mioty.BaseStationMessageStats) error {
	now := r.clock.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	periods := []struct {
		since   time.Time
		count   *int64
		errWrap string
	}{
		{todayStart, &stats.MessagesToday, errWrapGetTodaySMessageCount},
		{todayStart.AddDate(0, 0, -int(todayStart.Weekday())), &stats.MessagesThisWeek, errWrapGetThisWeekSMessageCount},
		{time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()), &stats.MessagesThisMonth, errWrapGetThisMonthSMessageCount},
	}
	for _, period := range periods {
		scope := stationUplinks(tenantID, bsEui, &period.since, nil)
		if err := r.db.QueryRowContext(ctx, sqlCountMessagesWhere+scope.where(), scope.args...).Scan(period.count); err != nil {
			return fmt.Errorf("%s: %w", period.errWrap, err)
		}
	}
	return nil
}

// stationContainment is the base_stations containment probe for the station.
func stationContainment(bsEui []byte) string {
	return fmt.Sprintf(bsContainmentFmt, mioty.EUI64FromBytes(bsEui))
}

// GetBaseStationEndpointCounts counts per endpoint the uplinks the base
// station received within the window, as the primary receiver or as one of
// the receptions; a nil bound leaves the window open.
func (r *MessageRepository) GetBaseStationEndpointCounts(ctx context.Context, tenantID int64, bsEui []byte,
	startTime, endTime *time.Time,
) (map[string]int64, error) {
	scope := stationUplinks(tenantID, bsEui, startTime, endTime)
	rows, err := r.db.QueryContext(ctx, sqlCountByEndpointWhere+scope.where()+sqlGroupByEndpoint, scope.args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetBaseStationEndpointCounts, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgCloseRowsMessages, logger.FieldError, err)
		}
	}()
	counts := make(map[string]int64)
	for rows.Next() {
		var epEuiBytes []byte
		var count int64
		if err := rows.Scan(&epEuiBytes, &count); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanEndpointCount, err)
		}
		counts[mioty.FormatEUI64(mioty.EUI64FromBytes(epEuiBytes))] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateEndpointCounts, err)
	}
	return counts, nil
}

// GetBaseStationLastSeen is when the base station last received an uplink,
// as the primary receiver or as one of the receptions; without any, when its
// last session started.
func (r *MessageRepository) GetBaseStationLastSeen(ctx context.Context, tenantID int64, bsEui []byte) (*time.Time, error) {
	scope := stationUplinks(tenantID, bsEui, nil, nil)
	var lastUplink sql.NullInt64
	err := r.db.QueryRowContext(ctx, sqlLastRxTimeWhere+scope.where(), scope.args...).Scan(&lastUplink)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetBaseStationLastSeen, err)
	}
	if lastUplink.Valid {
		return nullUnixNanosPtr(lastUplink), nil
	}
	var lastSeen sql.NullTime
	// basestation_sessions has no bs_eui, so join basestations.
	err = r.db.QueryRowContext(ctx,
		`SELECT MAX(s.started_at) FROM basestation_sessions s
		 JOIN basestations b ON b.id = s.basestation_id
		 WHERE s.tenant_id = $1 AND b.bs_eui = $2`,
		tenantID, bsEui).Scan(&lastSeen)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%s: %w", errWrapGetBaseStationSessionLastSeen, err)
	}
	return nullTimePtr(lastSeen), nil
}
