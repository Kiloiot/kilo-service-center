// Package propagation provides interfaces and types for automatic attach/detach
// propagation across base stations per BSSCI §5.8-5.8.3.
package propagation

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// BaseStationSession is a lightweight snapshot of active BSSCI session state
// Used to avoid exporting internal bssci.Session type to maintain clean boundaries
type BaseStationSession struct {
	ID                string
	BaseStationEUI    uint64
	TenantID          int64
	OrganizationID    *string // UUID as string
	HandshakeComplete bool
	// Resumed is true when the station resumed its previous session and so
	// kept the endpoints it held (BSSCI §1).
	Resumed bool
	// DisconnectedAt is when the connection a resumed session continues was
	// lost; nil when it is unknown.
	DisconnectedAt *time.Time
}

// Service orchestrates automatic attach/detach propagation across base stations
// Callers must enrich context with tenant/org values before invoking methods
type Service interface {
	// TriggerEndpointPropagate fans out attach propagate after successful attach
	// Context must already contain tenant/org values (caller enriches)
	// activeSessions is a snapshot of connected sessions from Server
	TriggerEndpointPropagate(ctx context.Context, endpointID int64, activeSessions []BaseStationSession) error

	// ReconcileBaseStation replays all endpoints to newly connected BS
	// Context must already contain tenant/org values (caller enriches)
	ReconcileBaseStation(ctx context.Context, session BaseStationSession, bs *models.BaseStation) error
}
