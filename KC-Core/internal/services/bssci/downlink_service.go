package bssciservices

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/org"
	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/mioty"
)

type downlinkService struct {
	logger           logger.Logger
	tenantResolver   bssci.TenantResolver
	orgResolver      org.Resolver
	queueStore       interfaces.DownlinkQueueStore
	storage          interfaces.Storage
	scaciBroadcaster bssci.SCACIBroadcaster
	auditLogger      bssci.AuditLogger
	queueSerializer  bssci.QueueSerializer
}

// NewDownlinkService creates a new downlink service with all required dependencies.
func NewDownlinkService(
	log logger.Logger,
	queueStore interfaces.DownlinkQueueStore,
	tenantResolver bssci.TenantResolver,
	orgResolver org.Resolver,
	stor interfaces.Storage,
	scaciBroadcaster bssci.SCACIBroadcaster,
	auditLogger bssci.AuditLogger,
	queueSerializer bssci.QueueSerializer,
) bssci.DownlinkService {
	return &downlinkService{
		logger:           log,
		tenantResolver:   tenantResolver,
		orgResolver:      orgResolver,
		queueStore:       queueStore,
		storage:          stor,
		scaciBroadcaster: scaciBroadcaster,
		auditLogger:      auditLogger,
		queueSerializer:  queueSerializer,
	}
}

// EnqueueDownlink enqueues a downlink message using the storage interface.
func (d *downlinkService) EnqueueDownlink(ctx context.Context, epEui uint64, userData []byte, priority float32, tenantID int64) (int64, error) {
	// Create downlink message using real storage.DownlinkMessage type
	downlink := &storage.DownlinkMessage{
		EPEUI:    fmt.Sprintf("%016X", epEui),
		TenantID: fmt.Sprintf("%d", tenantID),
		Payload:  userData,
		Priority: priority,
		Status:   "queued",
	}

	// Delegate to storage.MIOTYDownlinks().EnqueueDownlink method
	// This method uses MIOTY-specific types and returns strongly-typed results
	result, err := d.storage.MIOTYDownlinks().EnqueueDownlink(ctx, downlink)
	if err != nil {
		d.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToEnqueueDownlink,
			"error", err,
			"epEui", epEui)
		return 0, bssci.NewCatalogError(bssci.ErrFailedToEnqueueDownlink, bssci.POSIX_EAGAIN)
	}

	d.logger.InfoContext(ctx, bssci.LogBSSCIEnqueuedDownlink,
		"queId", result.QueID,
		"epEui", epEui)

	// Register queue-to-tenant mapping for result processing (BSSCI §5.12)
	d.tenantResolver.RegisterQueueTenant(result.QueID, strconv.FormatInt(tenantID, 10))

	return result.QueID, nil
}

// UpdateDownlinkStatus updates downlink message status via the storage interface.
func (d *downlinkService) UpdateDownlinkStatus(ctx context.Context, queId uint64, _ string, status string) error {
	return d.storage.MIOTYDownlinks().UpdateDownlinkStatus(ctx, fmt.Sprintf("%d", queId), status, nil)
}

// ProcessDLDataResult handles complete downlink result processing (BSSCI §5.14)
// Extracts orchestration logic from handleDLDataResult handler
func (d *downlinkService) ProcessDLDataResult(ctx context.Context, session *bssci.Session, result *mioty.DLDataResult) (map[string]interface{}, error) {
	// Resolve the owning tenant for this downlink using queue ID
	var ownerTenantID int64
	var ownerTenantStr string

	// Range guard for uint64 -> int64 conversion
	if result.QueId > math.MaxInt64 {
		d.logger.ErrorContext(ctx, bssci.LogBSSCIQueueIDOutOfRange,
			"queId", result.QueId,
			"maxInt64", math.MaxInt64)
		return nil, fmt.Errorf("queue ID out of range: %d", result.QueId)
	}

	tidStr, err := d.tenantResolver.ResolveTenant(ctx, int64(result.QueId))
	if err != nil {
		d.logger.ErrorContext(ctx, bssci.LogBSSCICannotResolveTenantForDownlinkResult,
			"queId", result.QueId,
			"error", err)
		return nil, fmt.Errorf("cannot resolve tenant for queue %d: %w", result.QueId, err)
	}

	ownerTenantStr = tidStr
	if tid, err := strconv.ParseInt(tidStr, 10, 64); err == nil {
		ownerTenantID = tid
	} else {
		d.logger.ErrorContext(ctx, bssci.LogBSSCIInvalidTenantIDFormat,
			"tenantStr", tidStr,
			"error", err)
		return nil, fmt.Errorf("invalid tenant ID format: %s", tidStr)
	}

	d.logger.InfoContext(ctx, bssci.LogBSSCIReceivedDLDataResFromBaseStation,
		"bsEui", session.BaseStationEUI,
		"epEui", result.EpEui,
		"queId", result.QueId,
		"result", result.Result,
		"txTime", result.TxTime,
		"packetCnt", result.PacketCnt,
		"tenantId", ownerTenantStr)

	// Update downlink queue in database with transmission result
	if d.storage != nil {
		// Convert base station EUI to bytes for storage
		bsEuiBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(bsEuiBytes, session.BaseStationEUI)

		// Convert endpoint EUI to bytes for validation
		epEuiBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(epEuiBytes, result.EpEui)

		// Convert txTime to int64 pointer for storage
		var txTimePtr *int64
		if result.TxTime != nil {
			txTimePtr = result.TxTime
		}

		// Update the downlink result in the database with tenant and endpoint validation
		//nolint:gosec // G115: uint64->int64 safe, guarded by range check above
		err := d.storage.MIOTYDownlinks().UpdateDownlinkResult(ctx, int64(result.QueId), result.Result, txTimePtr, result.PacketCnt, bsEuiBytes, epEuiBytes, ownerTenantStr, nil)
		if err != nil {
			d.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateDownlinkResult,
				"queId", result.QueId,
				"result", result.Result,
				"error", err)

			// Return appropriate error based on error type
			if errors.Is(err, storage.ErrDownlinkNotFound) {
				return nil, fmt.Errorf("queue ID not found: %d", result.QueId)
			}
			return nil, fmt.Errorf("database update failed: %w", err)
		}

		d.logger.DebugContext(ctx, bssci.LogBSSCIUpdatedDownlinkResult,
			"queId", result.QueId,
			"result", result.Result,
			"txTime", result.TxTime)

		// Forward result to Application Centers via SCACI (non-blocking)
		if d.scaciBroadcaster != nil {
			// Create isolated copy and set bsEui only for "sent" per SCACI §3.12.1
			resultForAC := *result
			if result.Result == mioty.ResultSent {
				resultForAC.BsEui = new(uint64)
				*resultForAC.BsEui = session.BaseStationEUI
			} else {
				resultForAC.BsEui = nil
			}

			// Derive timeout context from caller context (preserves tenant/org metadata)
			ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)

			// Pass derived context and cancel into goroutine explicitly
			go func(ctx context.Context, cancel context.CancelFunc, tenantID int64, payload mioty.DLDataResult) {
				defer cancel() // Ensure timeout cleanup

				if err := d.scaciBroadcaster.BroadcastDLDataResult(ctx, tenantID, &payload); err != nil {
					d.logger.WarnContext(ctx, bssci.LogBSSCIFailedToBroadcastDLResultToSCACI,
						"queId", payload.QueId,
						"error", err)
				}
			}(ctxWithTimeout, cancel, ownerTenantID, resultForAC)
		}
	}

	// Record event based on result via audit logger
	if err := d.auditLogger.RecordDLResult(ctx, ownerTenantStr, session, result); err != nil {
		d.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToRecordDLDataResultEvent, "error", err)
	}

	// Clean up queue-to-tenant mapping after result processed
	//nolint:gosec // G115: uint64->int64 safe, guarded by range check above
	d.tenantResolver.UnregisterQueueTenant(int64(result.QueId))

	// Build dlDataResRsp response via queue serializer
	response := d.queueSerializer.BuildDLDataResultResponse(result.OpId, result)
	return response, nil
}

// ProcessRevokeResponse handles downlink revoke response processing (BSSCI §5.13)
// Extracts orchestration logic from handleDLDataRevokeResponse handler
func (d *downlinkService) ProcessRevokeResponse(ctx context.Context, session *bssci.Session, opId int64, queueID int64, endpointEUI uint64) (map[string]interface{}, error) {
	// Validate queue ID
	if queueID <= 0 {
		d.logger.ErrorContext(ctx, bssci.LogBSSCIInvalidQueueIDInRevokeResponse,
			"queueId", queueID,
			"opId", opId)
		return nil, bssci.NewCatalogError(bssci.ErrInvalidQueueID, bssci.POSIX_EINVAL)
	}

	// Resolve the owning tenant for this downlink using queue ID
	tidStr, err := d.tenantResolver.ResolveTenant(ctx, queueID)
	if err != nil {
		d.logger.ErrorContext(ctx, bssci.LogBSSCICannotResolveTenantForRevoke,
			"queueId", queueID,
			"opId", opId,
			"error", err)
		return nil, bssci.NewCatalogError(bssci.ErrCannotResolveTenantForQueue, bssci.POSIX_EIO)
	}

	ownerTenantStr := tidStr

	d.logger.InfoContext(ctx, bssci.LogBSSCIProcessingRevokeResponse,
		"bsEui", session.BaseStationEUI,
		"epEui", endpointEUI,
		"queId", queueID,
		"tenantId", ownerTenantStr)

	// Clean up queue-to-tenant mapping AFTER successful tenant resolution
	d.tenantResolver.UnregisterQueueTenant(queueID)

	// Update database to mark as revoked (BSSCI §5.13.3 requires persistence before completion)
	if d.storage == nil {
		d.logger.ErrorContext(ctx, bssci.LogBSSCIStorageUnavailableForRevoke,
			"queId", queueID,
			"opId", opId)
		return nil, bssci.NewCatalogError(bssci.ErrDatabaseUpdateFailed, bssci.POSIX_EIO)
	}

	err = d.storage.MIOTYDownlinks().RevokeDownlink(ctx, queueID, ownerTenantStr)
	if err != nil {
		d.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToUpdateDownlinkAsRevoked,
			"queId", queueID,
			"error", err)
		return nil, bssci.NewCatalogError(bssci.ErrDatabaseUpdateFailed, bssci.POSIX_EIO)
	}

	// Record event via audit logger
	// Note: This would need RecordDLRevokeResponse method on AuditLogger interface
	// For now, manually create event similar to original handler
	// TODO: Add RecordDLRevokeResponse to AuditLogger interface

	// Build dlDataRevCmp complete response using serializer (canonical constant enforcement)
	response := d.queueSerializer.BuildDLDataRevokeComplete(opId)
	return response, nil
}
