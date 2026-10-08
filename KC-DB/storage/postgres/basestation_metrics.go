package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/jmoiron/sqlx"
)

// BaseStationMetricsRepository answers availability and throughput queries
// for one base station.
type BaseStationMetricsRepository struct {
	db *sqlx.DB
}

// NewBaseStationMetricsRepository creates the metrics repository.
func NewBaseStationMetricsRepository(db *sqlx.DB) *BaseStationMetricsRepository {
	return &BaseStationMetricsRepository{db: db}
}

// GetBaseStationOnlineIntervals returns the connected intervals for a base station
// that overlap [start, end), derived from its BSSCI sessions. An interval with a
// nil End is still active. These intervals are the source for time-weighted
// availability; callers bucket them. Results are ordered by start time.
func (r *BaseStationMetricsRepository) GetBaseStationOnlineIntervals(ctx context.Context, tenantID, baseStationID int64,
	start, end time.Time) (intervals []mioty.BaseStationOnlineInterval, err error) {

	const query = `
		SELECT started_at, ended_at
		FROM basestation_sessions
		WHERE tenant_id = $1
			AND basestation_id = $2
			AND started_at < $4
			AND (ended_at IS NULL OR ended_at > $3)
		ORDER BY started_at`

	rows, err := r.db.QueryContext(ctx, query, tenantID, baseStationID, start, end)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapQueryBaseStationOnlineIntervals, err)
	}
	defer sqlcleanup.CloseRows(rows, errWrapIterateBaseStationOnlineIntervals, &err)

	for rows.Next() {
		var startedAt time.Time
		var endedAt sql.NullTime
		if err := rows.Scan(&startedAt, &endedAt); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanBaseStationOnlineInterval, err)
		}
		interval := mioty.BaseStationOnlineInterval{Start: startedAt}
		if endedAt.Valid {
			ended := endedAt.Time
			interval.End = &ended
		}
		intervals = append(intervals, interval)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateBaseStationOnlineIntervals, err)
	}

	return intervals, nil
}

// CountBaseStationMessagesByBucket counts received uplink data messages ('ulData' only)
// per intervalSeconds bucket over [start, end), keyed by rx_time (BSSCI reception time, ns)
// / intervalSeconds. Absent buckets are zero-filled by the caller; secondary receivers
// (base_stations JSONB) are included. The base station must belong to the tenant; the
// receptions it counts belong to every tenant, so roaming traffic shows up as load
// without exposing whose it is.
func (r *BaseStationMetricsRepository) CountBaseStationMessagesByBucket(ctx context.Context, tenantID int64, bsEui []byte,
	start, end time.Time, intervalSeconds int64) (counts map[int64]int64, err error) {

	if intervalSeconds <= 0 {
		return nil, fmt.Errorf(errFmtIntervalSecondsMustBePositiveGot, intervalSeconds)
	}

	if len(bsEui) != dbconfig.EUISize {
		return nil, fmt.Errorf(errFmtBsEuiMustBe8BytesGot, len(bsEui))
	}
	// Containment compares the unsigned EUI exactly; a bigint cast fails on any EUI at or above 2^63.
	bsContainment := stationContainment(bsEui)

	const query = `
		WITH owned AS (
			SELECT 1 FROM basestations WHERE tenant_id = $1 AND bs_eui = $7
		),
		matched AS (
			SELECT m.rx_time
			FROM messages m
			WHERE EXISTS (SELECT 1 FROM owned)
				AND m.base_stations @> $2::jsonb
				AND m.command_type = $6
				AND m.rx_time >= $3
				AND m.rx_time < $4
			UNION ALL
			SELECT rx_time
			FROM messages
			WHERE EXISTS (SELECT 1 FROM owned)
				AND bs_eui = $7
				AND command_type = $6
				AND (base_stations IS NULL OR jsonb_array_length(base_stations) = 0)
				AND rx_time >= $3
				AND rx_time < $4
		)
		SELECT (rx_time / 1000000000 / $5)::bigint AS bucket_index,
			COUNT(*) AS message_count
		FROM matched
		GROUP BY bucket_index`

	// $7 matches the BYTEA bs_eui column; $2 the numeric bsEui in base_stations JSONB.
	rows, err := r.db.QueryContext(ctx, query, tenantID, bsContainment, start.UnixNano(), end.UnixNano(), intervalSeconds, mioty.CmdULData, bsEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapQueryBaseStationMessageBuckets, err)
	}
	defer sqlcleanup.CloseRows(rows, errWrapIterateBaseStationMessageBuckets, &err)

	counts = make(map[int64]int64)
	for rows.Next() {
		var bucketIndex, count int64
		if err := rows.Scan(&bucketIndex, &count); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanBaseStationMessageBucket, err)
		}
		counts[bucketIndex] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateBaseStationMessageBuckets, err)
	}

	return counts, nil
}
