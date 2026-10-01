// Package alerts provides alerts service implementation for gRPC layer.
package alerts

import "github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"

// AlertStatuses are the states an alert can be listed by.
var AlertStatuses = []string{
	models.EventStatusNew,
	models.EventStatusAcknowledged,
	models.EventStatusResolved,
}

// AlertSeverities defines severities that qualify as alerts (warning+).
var AlertSeverities = []string{
	models.EventSeverityWarning,
	models.EventSeverityError,
	models.EventSeverityCritical,
}
