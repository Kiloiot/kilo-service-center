package scacimonitoring

// Log messages owned by the SCACI monitoring service.
const (
	LogListSessionsFailed         = "failed to list SCACI sessions"
	LogCountOperationsFailed      = "failed to count SCACI session operations"
	LogGetSessionFailed           = "failed to get SCACI session"
	LogGetSessionStatisticsFailed = "failed to get session statistics"
	LogGetOperationSummaryFailed  = "failed to get operation summary"
	LogListErrorsFailed           = "failed to list failed SCACI operations"
	LogListQueueFailed            = "failed to list downlink queue"
	LogCountQueueFailed           = "failed to count queue"
	LogGetStatusStatisticsFailed  = "failed to get session statistics for status"
	LogGetPingSummaryFailed       = "failed to get ping summary"
	LogCountReconnectsFailed      = "failed to count reconnect attempts"
	LogGetLatestConnectFailed     = "failed to get latest connect operation"
)
