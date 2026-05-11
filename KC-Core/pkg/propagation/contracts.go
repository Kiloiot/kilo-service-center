// Package propagation provides interfaces and types for automatic attach/detach
// propagation across base stations per BSSCI §5.8-5.8.3.
package propagation

import (
	"context"

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
}

// SessionSnapshotProvider provides access to connected BSSCI session snapshots
// Used by SCACI to trigger propagation without importing bssci package (avoids circular dependency)
// The BSSCI Server implements this interface via ConnectedSessionsSnapshot() method
//
// BSSCI §5.8-5.8.3: Cross-module integration pattern
type SessionSnapshotProvider interface {
	ConnectedSessionsSnapshot() []BaseStationSession
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
