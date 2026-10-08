package interfaces

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// SystemEventStore defines the interface for system event operations
type SystemEventStore interface {
	// CreateEvent creates a new system event
	CreateEvent(ctx context.Context, event *models.SystemEvent) error

	// GetEvents retrieves events with filters
	GetEvents(ctx context.Context, filter models.SystemEventFilter) ([]*models.SystemEvent, error)

	// GetActiveAlerts retrieves unacknowledged events with severity warning or higher
	GetActiveAlerts(ctx context.Context, filter models.AlertFilter) ([]*models.SystemEvent, error)

	// GetEventStats retrieves event statistics
	GetEventStats(ctx context.Context, tenantID string, since time.Time) (*models.SystemEventStats, error)

	// CountEvents returns total count matching filter (for pagination)
	CountEvents(ctx context.Context, filter models.SystemEventFilter) (int64, error)

	// CountActiveAlerts returns total count of active alerts (for pagination)
	CountActiveAlerts(ctx context.Context, filter models.AlertFilter) (int64, error)
}

// SCACIEventStore records and reads the SCACI protocol events kept in the
// system event log.
type SCACIEventStore interface {
	RecordSCACIError(ctx context.Context, tenantID int64, sessionID int64, command string, opId int64, errorCode int, errorMsg string) error
	RecordSessionEvent(ctx context.Context, event *models.SCACISessionEvent) error
	ListSCACIEvents(ctx context.Context, tenantID int64, sessionID *int64, eventType string, limit, offset int) ([]*models.SystemEvent, error)
	CountSCACIEventsByFilter(ctx context.Context, tenantID int64, sessionID *int64, eventType string) (int64, error)
}
