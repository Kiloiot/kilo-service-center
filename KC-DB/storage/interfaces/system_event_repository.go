package interfaces

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// SystemEventStore defines the interface for system event operations
type SystemEventStore interface {
	// CreateEvent creates a new system event
	CreateEvent(ctx context.Context, event *models.SystemEvent) error

	// GetEvents retrieves events with filters
	GetEvents(ctx context.Context, filter SystemEventFilter) ([]*models.SystemEvent, error)

	// GetActiveAlerts retrieves unacknowledged events with severity warning or higher
	GetActiveAlerts(ctx context.Context, filter AlertFilter) ([]*models.SystemEvent, error)

	// GetEventStats retrieves event statistics
	GetEventStats(ctx context.Context, tenantID string, since time.Time) (*models.SystemEventStats, error)

	// RecordSCACIError records a SCACI protocol error event per SCACI §3.14
	RecordSCACIError(ctx context.Context, tenantID int64, sessionID int64, command string, opId int64, errorCode int, errorMsg string) error

	// CountEvents returns total count matching filter (for pagination)
	CountEvents(ctx context.Context, filter SystemEventFilter) (int64, error)

	// CountActiveAlerts returns total count of active alerts (for pagination)
	CountActiveAlerts(ctx context.Context, filter AlertFilter) (int64, error)
}

// SystemEventFilter defines filters for querying events
type SystemEventFilter struct {
	TenantID       string
	EventTypes     []string
	Categories     []string
	Severities     []string
	SourceTypes    []string
	SourceID       *uuid.UUID
	BaseStationID  *int64
	EndpointID     *int64
	BaseStationEUI string // Fallback: filter by source_name ILIKE for BS events
	EndpointEUI    string // Fallback: filter by source_name ILIKE for EP events
	Status         []string
	Since          *time.Time
	Until          *time.Time
	SearchText     string
	Limit          int
	Offset         int
	OrderBy        string // "occurred_at", "severity", etc.
	OrderDirection string // "asc" or "desc"
}

// AlertFilter defines filters specifically for alerts
type AlertFilter struct {
	TenantID     string
	Severities   []string // only warning, error, critical
	Categories   []string
	Acknowledged *bool
	Since        *time.Time
	Limit        int
	Offset       int
}
