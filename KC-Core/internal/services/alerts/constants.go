// Package alerts provides alerts service implementation for gRPC layer.
package alerts

import "github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"

// AlertSeverities defines severities that qualify as alerts (warning+).
var AlertSeverities = []string{
	models.EventSeverityWarning,
	models.EventSeverityError,
	models.EventSeverityCritical,
}

// DefaultRecentAlertsLimit is the default limit for recent alerts in summary.
// Use config alerts.recent_alerts_limit to override.
const DefaultRecentAlertsLimit = 5
