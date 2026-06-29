package postgres

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// GetBaseStationOnlineIntervals returns the connected intervals for a base station
// that overlap [start, end), derived from its BSSCI sessions. An interval with a
// nil End is still active. These intervals are the source for time-weighted
// availability; callers bucket them. Results are ordered by start time.
func (db *DB) GetBaseStationOnlineIntervals(ctx context.Context, tenantID, baseStationID int64,
	start, end time.Time) ([]mioty.BaseStationOnlineInterval, error) {

	const query = `
		SELECT started_at, ended_at
		FROM basestation_sessions
		WHERE tenant_id = $1
			AND basestation_id = $2
			AND started_at < $4
			AND (ended_at IS NULL OR ended_at > $3)
		ORDER BY started_at`

	rows, err := db.sqlxDB.QueryContext(ctx, query, tenantID, baseStationID, start, end)
	if err != nil {
		return nil, fmt.Errorf("query base station online intervals: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var intervals []mioty.BaseStationOnlineInterval
	for rows.Next() {
		var startedAt time.Time
		var endedAt sql.NullTime
		if err := rows.Scan(&startedAt, &endedAt); err != nil {
			return nil, fmt.Errorf("scan base station online interval: %w", err)
		}
		interval := mioty.BaseStationOnlineInterval{Start: startedAt}
		if endedAt.Valid {
			ended := endedAt.Time
			interval.End = &ended
		}
		intervals = append(intervals, interval)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate base station online intervals: %w", err)
	}

	return intervals, nil
}

// CountBaseStationMessagesByBucket counts received uplink data messages ('ulData' only)
// per intervalSeconds bucket over [start, end), keyed by rx_time (BSSCI reception time, ns)
// / intervalSeconds. Absent buckets are zero-filled by the caller; secondary receivers
// (base_stations JSONB) are included.
func (db *DB) CountBaseStationMessagesByBucket(ctx context.Context, tenantID int64, bsEui []byte,
	start, end time.Time, intervalSeconds int64) (map[int64]int64, error) {

	if intervalSeconds <= 0 {
		return nil, fmt.Errorf("interval seconds must be positive, got %d", intervalSeconds)
	}

	var bsEuiUint uint64
	if len(bsEui) == 8 {
		bsEuiUint = binary.BigEndian.Uint64(bsEui)
	}

	const query = `
		WITH matched AS (
			SELECT m.rx_time
			FROM messages m,
				jsonb_array_elements(m.base_stations) AS elem
			WHERE m.tenant_id = $1
				AND (elem->>'bsEui')::bigint = $2
				AND m.command_type = $6
				AND m.rx_time >= $3
				AND m.rx_time < $4
			UNION ALL
			SELECT rx_time
			FROM messages
			WHERE tenant_id = $1
				AND bs_eui = $2
				AND command_type = $6
				AND (base_stations IS NULL OR jsonb_array_length(base_stations) = 0)
				AND rx_time >= $3
				AND rx_time < $4
		)
		SELECT (rx_time / 1000000000 / $5)::bigint AS bucket_index,
			COUNT(*) AS message_count
		FROM matched
		GROUP BY bucket_index`

	rows, err := db.sqlxDB.QueryContext(ctx, query, tenantID, bsEuiUint, start.UnixNano(), end.UnixNano(), intervalSeconds, mioty.CmdULData)
	if err != nil {
		return nil, fmt.Errorf("query base station message buckets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	counts := make(map[int64]int64)
	for rows.Next() {
		var bucketIndex, count int64
		if err := rows.Scan(&bucketIndex, &count); err != nil {
			return nil, fmt.Errorf("scan base station message bucket: %w", err)
		}
		counts[bucketIndex] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate base station message buckets: %w", err)
	}

	return counts, nil
}
