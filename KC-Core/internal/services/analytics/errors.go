package analytics

import "errors"

// ErrUnsupportedGranularity reports an activity bucket width the message
// store cannot aggregate by.
var ErrUnsupportedGranularity = errors.New("unsupported activity granularity")

// Operation-context sentinels wrapped around message store failures so
// callers can identify which analytics query failed.
var (
	errAnalyticsOverview      = errors.New("get analytics overview")
	errActivityTotals         = errors.New("get activity totals")
	errDailyActivity          = errors.New("get daily activity")
	errSignalQualityStats     = errors.New("get signal quality stats")
	errSignalQualityByStation = errors.New("get signal quality by base station")
)
