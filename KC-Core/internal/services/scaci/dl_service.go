// Package scaciservices implements SCACI service layer components.
//
// dl_service.go implements the DLService interface: it maps scheduler errors
// to SCACI error tokens and delegates revokes to the BSSCI scheduler.
//
// Dependencies (injected):
//   - scheduler.DownlinkScheduler: BSSCI scheduler interface
//   - QueueIDAllocator: draws the service center queue id of every downlink
//   - EnqueueRecorder: announces every downlink the queue accepts
//   - logger.Logger: Structured logging
//
// Error Handling:
//   - Maps scheduler errors → SCACI error tokens
//   - Returns error tokens (not Go errors) for consistency
//
// EnqueueDownlink persists every downlink under a service center queue id
// drawn by the allocator; the Application Center's own queue id is stored
// beside it and never reaches a base station.
package scaciservices

import (
	"context"
	"errors"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
)

// DownlinkStore is the downlink queue surface the SCACI DL service reads and
// writes; revocations go through the scheduler, never the store directly.
type DownlinkStore interface {
	ListInFlightDownlinks(ctx context.Context, tenantID int64, filter storage.DownlinkQueueFilter) ([]*storage.DownlinkMessage, error)
	ListPacketCounterDownlinks(ctx context.Context, query storage.PacketCounterDownlinks) ([]*storage.DownlinkMessage, error)
	EnqueueDownlink(ctx context.Context, downlink *storage.DownlinkMessage, lifetime time.Duration) (*storage.DownlinkMessage, error)
}

// EnqueueRecorder records a downlink entering the service center queue.
type EnqueueRecorder interface {
	RecordEnqueued(ctx context.Context, downlink *storage.DownlinkMessage) error
}

// QueueIDAllocator draws the service center queue id of a downlink and runs
// the enqueue again under a fresh id while the drawn one is already assigned.
type QueueIDAllocator interface {
	Allocate(ctx context.Context, enqueue func(ctx context.Context, queID int64) error) (int64, error)
}

// dlService implements DLService interface
//
// This is a SCACI-facing adapter around scheduler.DownlinkScheduler and the
// narrow downlink repository ports above. It delegates to the BSSCI scheduler
// and translates scheduler errors into SCACI error tokens for consistent
// error handling.
//
// Error Mapping (per SCACI §3.10 / §3.11):
//   - scheduler.ErrSchedulerNoResources on queue → deferred delivery, no error
//   - scheduler.ErrSchedulerQueueNotFound → errDownlinkNotFound (POSIX_ENOENT)
//   - scheduler.ErrSchedulerNoResources on revoke → errSchedulerUnavailable (POSIX_EAGAIN)
//   - Other errors                       → errSchedulerUnavailable / errFailedRecordOperation (POSIX_EIO)
//
// This adapter exists to:
//  1. Maintain error token pattern across all SCACI services
//  2. Avoid polluting scheduler package with SCACI concerns
//  3. Enable testing via interface mocking
//  4. Encapsulate downlink queue persistence logic
type dlService struct {
	dlScheduler scheduler.DownlinkScheduler // BSSCI scheduler
	downlinks   DownlinkStore               // Downlink queue persistence
	queueIDs    QueueIDAllocator            // Service center queue id of every downlink
	lifetime    time.Duration               // How long a new downlink waits for a downlink window
	events      EnqueueRecorder             // Announces every downlink the queue accepts
	logger      logger.Logger
}

// NewDLService creates a new DL service.
//
// Parameters:
//   - dlScheduler: BSSCI DownlinkScheduler interface (from pkg/scheduler)
//   - downlinks: Downlink queue repository
//   - queueIDs: Allocator of the service center queue ids
//   - lifetime: How long a new downlink waits for a downlink window before it expires
//   - events: Records every downlink the queue accepts
//   - logger: Structured logger
//
// Returns:
//   - DLService: Service instance implementing interface
//   - error: a nil collaborator's sentinel (ErrNilDownlinkScheduler, ErrNilDownlinkStore,
//     ErrNilQueueIDAllocator, ErrNilEnqueueRecorder, ErrNilDLServiceLogger), ErrNonPositiveDownlinkLifetime
//     without a lifetime
func NewDLService(
	dlScheduler scheduler.DownlinkScheduler,
	downlinks DownlinkStore,
	queueIDs QueueIDAllocator,
	lifetime time.Duration,
	events EnqueueRecorder,
	log logger.Logger,
) (scaci.DLService, error) {
	switch {
	case dlScheduler == nil:
		return nil, ErrNilDownlinkScheduler
	case downlinks == nil:
		return nil, ErrNilDownlinkStore
	case queueIDs == nil:
		return nil, ErrNilQueueIDAllocator
	case events == nil:
		return nil, ErrNilEnqueueRecorder
	case log == nil:
		return nil, ErrNilDLServiceLogger
	case lifetime <= 0:
		return nil, ErrNonPositiveDownlinkLifetime
	}
	return &dlService{
		dlScheduler: dlScheduler,
		downlinks:   downlinks,
		queueIDs:    queueIDs,
		lifetime:    lifetime,
		events:      events,
		logger:      log,
	}, nil
}

// QueueDownlink implements DLService.QueueDownlink
//
// Flow:
//  1. Check if scheduler is available
//  2. Delegate to BSSCI scheduler
//  3. Map scheduler errors → SCACI error tokens
//  4. Return the outcome and errToken
//
// Scheduler Integration:
//   - Calls scheduler.QueueDownlink(req, tenantID)
//   - Scheduler hands the row to the base station serving the endpoint
//   - Returns actual queue ID (may normalize on collision) + BS EUI
//   - No connected bidirectional serving station is not a failure: the
//     persisted row stays pending and is delivered in the endpoint's next
//     downlink window (SCACI §3.10 permits queueing a priori)
//
// Parameters:
//   - ctx: Request context
//   - req: DL Data Queue message
//   - tenantID: Tenant scope
//
// Returns:
//   - outcome: queue ID assigned by the scheduler and the delivering base
//     station EUI, or Deferred with the request's queue ID
//   - errToken: Error token if queueing fails, "" on success
func (dls *dlService) QueueDownlink(
	ctx context.Context,
	req *mioty.DLDataQueue,
	tenantID int64,
	organizationID uuid.UUID,
) (scaci.DownlinkQueueOutcome, string) {
	dls.logger.DebugContext(ctx, scaci.LogSCACIDLQueueServiceInvoked,
		logger.FieldQueID, req.QueId,
		logger.FieldEpEui, req.EpEui,
		logger.FieldTenantIDCamel, tenantID)

	// Delegate to BSSCI scheduler
	// IMPORTANT: queuedQueId may differ from req.QueId if BSSCI normalizes it
	queuedQueId, bsEui, err := dls.dlScheduler.QueueDownlink(ctx, req, tenantID, organizationID)
	if errors.Is(err, scheduler.ErrSchedulerNoResources) {
		dls.logger.InfoContext(ctx, scaci.LogSCACIDLDataQueueDeferred,
			logger.FieldQueID, req.QueId,
			logger.FieldEpEui, req.EpEui,
			logger.FieldTenantIDCamel, tenantID)
		return scaci.DownlinkQueueOutcome{QueID: req.QueId, Deferred: true}, ""
	}
	if err != nil {
		dls.logger.ErrorContext(ctx, scaci.LogSCACIEnqueueDownlinkFailed,
			logger.FieldQueID, req.QueId,
			logger.FieldEpEui, req.EpEui,
			logger.FieldError, err)

		var errorToken string
		switch {
		case errors.Is(err, scheduler.ErrSchedulerQueueNotFound):
			// The pending row vanished between persistence and dispatch
			errorToken = scaci.ErrDownlinkNotFound
		default:
			// Generic infrastructure error
			errorToken = scaci.ErrFailedRecordOperation
		}

		return scaci.DownlinkQueueOutcome{}, errorToken
	}

	// Success - use queuedQueId from scheduler (may differ from request)
	dls.logger.DebugContext(ctx, scaci.LogSCACIDLDataQueueProcessed,
		logger.FieldQueID, queuedQueId,
		logger.FieldBsEui, bsEui,
		logger.FieldEpEui, req.EpEui)

	return scaci.DownlinkQueueOutcome{QueID: queuedQueId, BsEui: bsEui}, ""
}

// RevokeDownlink implements DLService.RevokeDownlink: the BSSCI scheduler
// revokes the downlink the reference names and its failure maps to the SCACI
// error token (§3.11).
func (dls *dlService) RevokeDownlink(ctx context.Context, ref scheduler.DownlinkRef) (uint64, string) {
	bsEui, err := dls.dlScheduler.RevokeDownlink(ctx, ref)
	if err != nil {
		dls.logger.ErrorContext(ctx, scaci.LogSCACIRevokeDownlinkFailed,
			logger.FieldQueID, ref.QueID,
			logger.FieldTenantIDCamel, ref.TenantID,
			logger.FieldError, err)

		// Map scheduler errors to SCACI error tokens per §3.11
		var errorToken string
		switch {
		case errors.Is(err, scheduler.ErrSchedulerQueueNotFound):
			// Queue entry not found or already transmitted
			errorToken = scaci.ErrDownlinkNotFound
		case errors.Is(err, scheduler.ErrSchedulerNoResources):
			// Scheduler unavailable or in degraded state
			errorToken = scaci.ErrSchedulerUnavailable
		default:
			// Generic scheduler error
			errorToken = scaci.ErrSchedulerUnavailable
		}

		return 0, errorToken
	}

	// Success
	dls.logger.DebugContext(ctx, scaci.LogSCACIDLRevokeSuccessful,
		logger.FieldQueID, ref.QueID,
		logger.FieldBsEui, bsEui,
		logger.FieldTenantIDCamel, ref.TenantID)

	return bsEui, ""
}

// Storage persistence methods encapsulate downlink queue database operations.

// EnqueueDownlink persists a downlink message under a service center queue
// id drawn by the allocator, drawing again while the id is already assigned,
// and announces it; the downlink stays queued when the event cannot be recorded.
//
// Parameters:
//   - ctx: Request context
//   - dlMsg: Downlink message to persist; its QueID is assigned here
//
// Returns:
//   - *storage.DownlinkMessage: Stored message with database ID and queue ID assigned
//   - error: storage.ErrDuplicateKey when the Application Center queue id is
//     already in flight in its organization, or another persistence error
func (dls *dlService) EnqueueDownlink(ctx context.Context, dlMsg *storage.DownlinkMessage) (*storage.DownlinkMessage, error) {
	var stored *storage.DownlinkMessage
	_, err := dls.queueIDs.Allocate(ctx, func(ctx context.Context, queID int64) error {
		dlMsg.QueID = queID
		persisted, err := dls.downlinks.EnqueueDownlink(ctx, dlMsg, dls.lifetime)
		stored = persisted
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := dls.events.RecordEnqueued(ctx, stored); err != nil {
		dls.logger.ErrorContext(ctx, scaci.LogSCACIRecordEnqueuedEventFailed,
			logger.FieldQueID, stored.QueID,
			logger.FieldEpEui, stored.EPEUI,
			logger.FieldError, err)
	}
	return stored, nil
}

// GetDownlinksByPacketCnt lists the downlinks a SCACI dlDataRev names: the
// in-flight counter-dependent downlinks the query scopes (SCACI §3.11.1),
// newest first; no match is an empty list.
func (dls *dlService) GetDownlinksByPacketCnt(ctx context.Context, query storage.PacketCounterDownlinks) ([]*storage.DownlinkMessage, error) {
	return dls.downlinks.ListPacketCounterDownlinks(ctx, query)
}

// GetDownlinkQueue lists the tenant's in-flight downlinks the filter narrows;
// the deregistration revokes an endpoint's through it.
func (dls *dlService) GetDownlinkQueue(ctx context.Context, tenantID int64, filter storage.DownlinkQueueFilter) ([]*storage.DownlinkMessage, error) {
	return dls.downlinks.ListInFlightDownlinks(ctx, tenantID, filter)
}
