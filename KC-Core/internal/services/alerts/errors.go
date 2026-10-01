package alerts

import "errors"

// ErrInvalidAlertStatus reports a status filter outside AlertStatuses.
var ErrInvalidAlertStatus = errors.New("invalid alert status")

// ErrInvalidAlertSeverity reports a severity filter outside AlertSeverities.
var ErrInvalidAlertSeverity = errors.New("invalid alert severity")

// Operation-context sentinels wrapped around alert store failures.
var (
	errListAlerts      = errors.New("list alerts")
	errGetAlertSummary = errors.New("get alert summary")
)
