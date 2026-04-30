// Package scaciservices implements SCACI service layer components.
//
// dl_service.go implements DLService interface.
//
// Extracted Logic:
//   - Scheduler error mapping (from handler_operations.go:1157-1250)
//   - DL revoke delegation to BSSCI
//
// Dependencies (injected):
//   - scheduler.DownlinkScheduler: BSSCI scheduler interface
//   - logger.Logger: Structured logging
//
// Error Handling:
//   - Maps scheduler errors → SCACI error tokens
//   - Returns error tokens (not Go errors) for consistency
//
// # This is a pure adapter - NO business logic, only error translation
//
// Architectural Note: QueueDownlink currently returns the queId from the request.
// The actual database persistence happens in the handler via storage.EnqueueDownlink(),
// and BSSCI picks up queued downlinks asynchronously from the database. This design
// aligns with SCACI spec requirements for asynchronous downlink processing.
package scaciservices

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/scaci"
	"github.com/kilocenter/KC-Core/pkg/scheduler"
	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/mioty"
)

// dlService implements DLService interface
//
// This is a SCACI-facing adapter around scheduler.DownlinkScheduler and storage.Storage.
// It delegates to the BSSCI scheduler and translates scheduler errors
// into SCACI error tokens for consistent error handling.
//
// Storage field encapsulates downlink queue persistence.
// Handlers access all storage operations through this service layer.
//
// Error Mapping (per SCACI §3.11):
//   - scheduler.ErrSchedulerQueueNotFound → errDownlinkNotFound (POSIX_ENOENT)
//   - scheduler.ErrSchedulerNoResources  → errSchedulerUnavailable (POSIX_EAGAIN)
//   - Other errors                       → errSchedulerUnavailable (POSIX_EIO)
//
// This adapter exists to:
//  1. Maintain error token pattern across all SCACI services
//  2. Avoid polluting scheduler package with SCACI concerns
//  3. Enable testing via interface mocking
//  4. Encapsulate downlink queue persistence logic
type dlService struct {
	dlScheduler scheduler.DownlinkScheduler // BSSCI scheduler
	storage     interfaces.Storage          // Downlink queue persistence via repository interfaces
	logger      logger.Logger
}

// NewDLService creates a new DL service.
//
// Parameters:
//   - dlScheduler: BSSCI DownlinkScheduler interface (from pkg/scheduler)
//   - storage: Storage layer for downlink queue persistence
//   - logger: Structured logger
//
// Returns:
//   - DLService: Service instance implementing interface
func NewDLService(
	dlScheduler scheduler.DownlinkScheduler,
	storage interfaces.Storage,
	log logger.Logger,
) scaci.DLService {
	return &dlService{
		dlScheduler: dlScheduler,
		storage:     storage,
		logger:      log,
	}
}

// QueueDownlink implements DLService.QueueDownlink
//
// Extracted from handler_operations.go:781-838
//
// Flow:
//  1. Check if scheduler is available
//  2. Delegate to BSSCI scheduler
//  3. Map scheduler errors → SCACI error tokens
//  4. Return (queuedQueId, bsEui, errToken)
//
// Scheduler Integration:
//   - Calls scheduler.QueueDownlink(req, tenantID)
//   - Scheduler selects bidirectional base station
//   - Returns actual queue ID (may normalize on collision) + BS EUI
//
// Parameters:
//   - ctx: Request context
//   - req: DL Data Queue message
//   - tenantID: Tenant scope
//
// Returns:
//   - queuedQueId: Actual queue ID assigned by scheduler (may differ from req.QueId)
//   - bsEui: Base station EUI selected for downlink
//   - errToken: Error token if queueing fails, "" on success
func (dls *dlService) QueueDownlink(
	ctx context.Context,
	req *mioty.DLDataQueue,
	tenantID int64,
) (uint64, uint64, string) {
	// Nil check for scheduler (feature guard)
	if dls.dlScheduler == nil {
		dls.logger.ErrorContext(ctx, scaci.LogSCACIDownlinkSchedulerNotConfigured)
		return 0, 0, scaci.ErrSchedulerUnavailable
	}

	dls.logger.Debug(scaci.LogSCACIDLQueueServiceInvoked,
		"queId", req.QueId,
		"epEui", req.EpEui,
		"tenantId", tenantID)

	// Delegate to BSSCI scheduler
	// IMPORTANT: queuedQueId may differ from req.QueId if BSSCI normalizes it
	queuedQueId, bsEui, err := dls.dlScheduler.QueueDownlink(req, tenantID)

	if err != nil {
		dls.logger.ErrorContext(ctx, scaci.LogSCACIEnqueueDownlinkFailed,
			"queId", req.QueId,
			"epEui", req.EpEui,
			"error", err)

		// Map scheduler errors to SCACI error tokens per §3.10
		var errorToken string
		switch {
		case errors.Is(err, scheduler.ErrSchedulerNoResources):
			// No bidirectional BS available - temporary
			errorToken = scaci.ErrBaseStationUnavailable
		case errors.Is(err, scheduler.ErrSchedulerResourceMissing):
			// Specific BS/EP not available - permanent
			errorToken = scaci.ErrBaseStationUnavailable
		case errors.Is(err, storage.ErrNotFound):
			// Endpoint/BS not found in database
			errorToken = scaci.ErrBaseStationUnavailable
		default:
			// Generic infrastructure error
			errorToken = scaci.ErrFailedRecordOperation
		}

		return 0, 0, errorToken
	}

	// Success - use queuedQueId from scheduler (may differ from request)
	dls.logger.Debug(scaci.LogSCACIDLDataQueueProcessed,
		"queId", queuedQueId,
		"bsEui", bsEui,
		"epEui", req.EpEui)

	return queuedQueId, bsEui, ""
}

// RevokeDownlink implements DLService.RevokeDownlink
//
// Extracted from handler_operations.go:1157-1250
//
// Flow:
//  1. Check if scheduler is available
//  2. Delegate to BSSCI scheduler
//  3. Map scheduler errors → SCACI error tokens
//  4. Return (bsEui, errToken)
//
// Scheduler Integration:
//   - Calls scheduler.RevokeDownlink(tenantID, queId)
//   - Scheduler removes from queue if still pending
//   - Returns BS EUI where downlink was queued
//
// Parameters:
//   - ctx: Request context
//   - queId: SC-issued queue ID (from prior QueueDownlink call)
//   - tenantID: Tenant scope for authorization
//
// Returns:
//   - bsEui: Base station EUI where downlink was queued
//   - errToken: Error token if revoke fails, "" on success
func (dls *dlService) RevokeDownlink(
	ctx context.Context,
	queId uint64,
	tenantID int64,
) (uint64, string) {
	// Nil check for scheduler (feature guard)
	if dls.dlScheduler == nil {
		dls.logger.ErrorContext(ctx, scaci.LogSCACIDownlinkSchedulerNotConfigured)
		return 0, scaci.ErrSchedulerUnavailable
	}

	// Delegate to BSSCI scheduler
	bsEui, err := dls.dlScheduler.RevokeDownlink(tenantID, queId)

	if err != nil {
		dls.logger.ErrorContext(ctx, scaci.LogSCACIRevokeDownlinkFailed,
			"queId", queId,
			"tenantId", tenantID,
			"error", err)

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
	dls.logger.Debug(scaci.LogSCACIDLRevokeSuccessful,
		"queId", queId,
		"bsEui", bsEui,
		"tenantId", tenantID)

	return bsEui, ""
}

// Storage persistence methods encapsulate downlink queue database operations.

// GetDownlinkByQueueID retrieves a downlink by queue ID
//
// Extracted from handler_operations.go:660
//
// Parameters:
//   - ctx: Request context
//   - queId: Queue ID to look up
//   - tenantID: Tenant identifier (string format)
//
// Returns:
//   - *storage.DownlinkMessage: Downlink message if found, nil otherwise
//   - error: Database error or nil
func (dls *dlService) GetDownlinkByQueueID(ctx context.Context, queId uint64, tenantID string) (*storage.DownlinkMessage, error) {
	if dls.storage == nil {
		return nil, storage.ErrStorageNotAvailable
	}
	return dls.storage.MIOTYDownlinks().GetDownlinkByQueueID(ctx, queId, tenantID)
}

// EnqueueDownlink persists a downlink message to the queue
//
// Extracted from handler_operations.go:741
//
// Parameters:
//   - ctx: Request context
//   - dlMsg: Downlink message to persist
//
// Returns:
//   - *storage.DownlinkMessage: Stored message with database ID assigned
//   - error: Database error or nil
func (dls *dlService) EnqueueDownlink(ctx context.Context, dlMsg *storage.DownlinkMessage) (*storage.DownlinkMessage, error) {
	if dls.storage == nil {
		return nil, storage.ErrStorageNotAvailable
	}
	return dls.storage.MIOTYDownlinks().EnqueueDownlink(ctx, dlMsg)
}

// UpdateDownlinkStatus updates the status field of a downlink message.
//
// Extracted from handler_operations.go:798,819.
// Accepts orgID for organization-filtered updates (SCACI §3.10).
//
// Parameters:
//   - ctx: Request context
//   - id: Downlink message ID (string format)
//   - status: New status value (e.g., "queued", "failed", "transmitted")
//   - orgID: Organization UUID for filtered updates (nil = no filter)
//
// Returns:
//   - error: Database error or nil
func (dls *dlService) UpdateDownlinkStatus(ctx context.Context, id string, status string, orgID *uuid.UUID) error {
	if dls.storage == nil {
		return storage.ErrStorageNotAvailable
	}
	return dls.storage.MIOTYDownlinks().UpdateDownlinkStatus(ctx, id, status, orgID)
}

// GetDownlinkByPacketCnt retrieves a downlink by endpoint and packet count
//
// Extracted from handler_operations.go:952
//
// Used by SCACI DL revoke operation which identifies downlinks by packetCnt
// instead of queId.
//
// Parameters:
//   - ctx: Request context
//   - tenantID: Tenant identifier (string format)
//   - epEui: Endpoint EUI (hex string format)
//   - packetCnt: Packet counter value
//
// Returns:
//   - *storage.DownlinkMessage: Downlink message if found, nil otherwise
//   - error: Database error or storage.ErrNotFound
func (dls *dlService) GetDownlinkByPacketCnt(ctx context.Context, tenantID string, epEui string, packetCnt uint32) (*storage.DownlinkMessage, error) {
	if dls.storage == nil {
		return nil, storage.ErrStorageNotAvailable
	}
	return dls.storage.MIOTYDownlinks().GetDownlinkByPacketCnt(ctx, tenantID, epEui, packetCnt)
}

// GetDownlinkQueue retrieves all pending/scheduled downlinks for an endpoint
//
// Extracted from handler_operations.go:1185
//
// Used by endpoint deregister flow to revoke all queued downlinks before
// marking endpoint inactive.
//
// Parameters:
//   - ctx: Request context
//   - deviceEUI: Endpoint EUI (hex string format)
//   - tenantID: Tenant identifier (string format)
//
// Returns:
//   - []*storage.DownlinkMessage: Slice of pending downlinks (may be empty)
//   - error: Database error or nil
func (dls *dlService) GetDownlinkQueue(ctx context.Context, deviceEUI string, tenantID string) ([]*storage.DownlinkMessage, error) {
	if dls.storage == nil {
		return nil, storage.ErrStorageNotAvailable
	}
	return dls.storage.MIOTYDownlinks().GetDownlinkQueue(ctx, deviceEUI, tenantID)
}

// RevokeDownlinkByID revokes a downlink by database ID
//
// Extracted from handler_operations.go:1204-1214
//
// This method uses type assertion to access RevokeDownlink method which is
// available on *postgres.DB but not on the storage.Storage interface.
//
// Parameters:
//   - ctx: Request context
//   - queId: Queue ID (int64 format as stored in database)
//   - tenantID: Tenant identifier (string format)
//
// Returns:
//   - error: Database error, ErrStorageNotAvailable, or ErrOperationNotSupported
func (dls *dlService) RevokeDownlinkByID(ctx context.Context, queId int64, tenantID string) error {
	if dls.storage == nil {
		return storage.ErrStorageNotAvailable
	}

	// Access RevokeDownlink method via MIOTYDownlinks repository
	type downlinkRevoker interface {
		RevokeDownlink(ctx context.Context, queId int64, tenantID string) error
	}

	revoker, ok := dls.storage.MIOTYDownlinks().(downlinkRevoker)
	if !ok {
		return storage.ErrOperationNotSupported
	}

	return revoker.RevokeDownlink(ctx, queId, tenantID)
}
