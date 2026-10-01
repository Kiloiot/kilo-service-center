// Package scheduler provides neutral contracts for SCACI<->BSSCI scheduling operations.
package scheduler

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
)

// ULTransmitScheduler allows SCACI to schedule uplink transmissions via BSSCI without import cycles.
//
// This interface is satisfied by the BSSCI Server and passed to SCACI during initialization.
// When an Application Center initiates an uplink transmission (SCACI §3.9), SCACI uses this
// interface to schedule the transmission through an appropriate base station.
//
// Call graph:
//   - Interface defined in neutral pkg/scheduler package
//   - Implemented by BSSCI Server (KC-Core/pkg/bssci/ul_transmit.go)
//   - Used by SCACI handlers (KC-Core/pkg/scaci/handler_operations.go)
//   - No circular dependencies between SCACI and BSSCI
type ULTransmitScheduler interface {
	// ScheduleULDataTransmit schedules an uplink transmission for an endpoint.
	//
	// Parameters:
	//   - tenantID: Tenant context for session selection
	//   - req: UL data transmit request containing endpoint details and payload
	//   - targetBsEui: Optional specific base station EUI (nil = SC chooses automatically)
	//
	// Returns:
	//   - opId: BSSCI operation ID for tracking
	//   - actualBsEui: EUI of base station that will transmit
	//   - error: ErrSchedulerNoResources if temporary, ErrSchedulerResourceMissing for permanent failures
	//
	// Error Semantics:
	//   - ErrSchedulerNoResources: No bidirectional BS available (retry later)
	//   - ErrSchedulerResourceMissing: Requested BS not available (permanent for these params)
	//   - Other errors: Infrastructure failures (database, network, etc.)
	ScheduleULDataTransmit(tenantID int64, req *mioty.ULDataTransmit, targetBsEui *uint64) (opId int64, actualBsEui uint64, err error)
}

// DownlinkScheduler allows SCACI to queue downlink messages via BSSCI without import cycles.
//
// This interface is satisfied by the BSSCI Server and passed to SCACI during initialization.
// When an Application Center queues a downlink message (SCACI §3.10), SCACI uses this
// interface to queue and schedule the message through an appropriate bidirectional base station.
//
// Call graph:
//   - Interface defined in neutral pkg/scheduler package
//   - Implemented by BSSCI Server (KC-Core/pkg/bssci/downlink_handlers.go)
//   - Used by SCACI handlers (KC-Core/pkg/scaci/handler_operations.go)
//   - No circular dependencies between SCACI and BSSCI
type DownlinkScheduler interface {
	// QueueDownlink dispatches a persisted pending downlink message to an endpoint.
	//
	// Parameters:
	//   - ctx: Request context for cancellation and tenant metadata
	//   - req: DL data queue request containing endpoint, payload, and delivery options
	//   - tenantID: Tenant context for base station selection and queue dispatch
	//   - organizationID: Organization the downlink was enqueued under; the
	//     exact-row dispatch is scoped to it (never the base station session's
	//     organization)
	//
	// Returns:
	//   - queuedQueId: Actual queue ID assigned (may differ from requested if collision)
	//   - bsEui: EUI of the base station that now holds the message: the one serving the endpoint, or a station of its tenant when its location is unknown
	//   - error: ErrSchedulerNoResources when no connected bidirectional station can take it
	//
	// Error Semantics:
	//   - ErrSchedulerNoResources: The serving station, or every station of the tenant for an endpoint with no known location, is offline or unidirectional; the row waits for the next downlink window
	//   - ErrSchedulerQueueNotFound: No matching pending queue row to dispatch
	//   - Other errors: Infrastructure failures (database, network, etc.)
	QueueDownlink(ctx context.Context, req *mioty.DLDataQueue, tenantID int64, organizationID uuid.UUID) (queuedQueId uint64, bsEui uint64, err error)

	// RevokeDownlink cancels a queued downlink message before delivery (SCACI §3.11).
	//
	// Parameters:
	//   - ctx: Request context for cancellation and tenant metadata
	//   - ref: The tenant's queue id to revoke, narrowed to an owner when set
	//
	// Returns:
	//   - bsEui: EUI of base station that had the queue entry
	//   - error: ErrSchedulerQueueNotFound if entry doesn't exist, ErrSchedulerResourceMissing if BS disconnected
	//
	// Error Semantics:
	//   - ErrSchedulerQueueNotFound: Queue entry doesn't exist or already processed
	//   - ErrSchedulerResourceMissing: Base station no longer connected
	//   - Other errors: Infrastructure failures (database, network, etc.)
	RevokeDownlink(ctx context.Context, ref DownlinkRef) (bsEui uint64, err error)
}

// DownlinkRef names the downlink a revoke ends: the tenant's service center
// queue id, narrowed to the organization and endpoint when they are set so
// another owner's queue id reads as not found.
type DownlinkRef struct {
	TenantID       int64
	QueID          uint64
	OrganizationID *uuid.UUID
	EpEUI          *uint64
}
