package statistics

// Supported time series granularities for message count aggregation.
const (
	granularityHour  = "hour"
	granularityDay   = "day"
	granularityWeek  = "week"
	granularityMonth = "month"
)

// supportedGranularitiesHint names the accepted granularity values in errors.
const supportedGranularitiesHint = "supported: " + granularityHour + ", " + granularityDay + ", " + granularityWeek + ", " + granularityMonth
