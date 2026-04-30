// Package scaciservices implements the SCACI (Service Center Application Center Interface) protocol server.
//
// ul_service.go implements ULService interface.
//
// Extracted Logic:
//   - Scheduler error mapping (from handler_operations.go:586-620)
//   - UL transmit delegation to BSSCI
//   - Preference lookup for BS selection (SCACI §3.9.1)
//
// Dependencies (injected):
//   - scheduler.ULTransmitScheduler: BSSCI scheduler interface
//   - scaci.StatusService: For preference lookup (last_attached_bs_eui)
//   - logger.Logger: Structured logging
//
// Error Handling:
//   - Maps scheduler errors → SCACI error tokens
//   - Returns error tokens (not Go errors) for consistency
//
// BS Selection Policy (SCACI §3.9.1):
//  1. Explicit bsEui from request takes priority
//  2. If nil, query endpoint's last_attached_bs_eui preference
//  3. If no preference or lookup fails, scheduler uses deterministic fallback (lowest EUI)
package scaciservices

import (
	"context"
	"encoding/binary"
	"errors"

	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/scaci"
	"github.com/kilocenter/KC-Core/pkg/scheduler"
	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/mioty"
)

// ulService implements ULService interface
//
// This is a SCACI-facing adapter around scheduler.ULTransmitScheduler.
// It delegates to the BSSCI scheduler and translates scheduler errors
// into SCACI error tokens for consistent error handling.
//
// Error Mapping (per SCACI §3.9):
//   - scheduler.ErrSchedulerNoResources  → errBaseStationUnavailable (POSIX_EAGAIN)
//   - scheduler.ErrSchedulerResourceMissing → errBaseStationUnavailable (POSIX_ENOENT)
//   - storage.ErrNotFound                → errBaseStationNotFound (POSIX_ENOENT)
//   - Other errors                       → errFailedRecordOperation (POSIX_EIO)
//
// This adapter exists to:
//  1. Maintain error token pattern across all SCACI services
//  2. Avoid polluting scheduler package with SCACI concerns
//  3. Enable testing via interface mocking
//  4. Encapsulate BS preference lookup (SCACI §3.9.1)
type ulService struct {
	ulScheduler scheduler.ULTransmitScheduler // BSSCI scheduler
	statusSvc   scaci.StatusService           // For preference lookup (§3.9.1)
	logger      logger.Logger
}

// NewULService creates a new UL service
//
// Parameters:
//   - ulScheduler: BSSCI ULTransmitScheduler interface (from pkg/scheduler)
//   - statusSvc: StatusService for preference lookup (§3.9.1), may be nil
//   - logger: Structured logger
//
// Returns:
//   - ULService: Service instance implementing interface
func NewULService(
	ulScheduler scheduler.ULTransmitScheduler,
	statusSvc scaci.StatusService,
	log logger.Logger,
) scaci.ULService {
	return &ulService{
		ulScheduler: ulScheduler,
		statusSvc:   statusSvc,
		logger:      log,
	}
}

// ScheduleULTransmit implements ULService.ScheduleULTransmit
//
// Extracted from handler_operations.go:578-620
//
// Flow:
//  1. Determine target BS (preference lookup if nil)
//  2. Delegate to BSSCI scheduler
//  3. Map scheduler errors → SCACI error tokens
//  4. Return (opId, bsEui, errToken)
//
// BS Selection Policy (SCACI §3.9.1):
//   - Explicit bsEui from request takes priority
//   - If nil, query endpoint's last_attached_bs_eui preference
//   - If no preference or lookup fails, scheduler uses deterministic fallback
//
// Scheduler Integration:
//   - Calls scheduler.ScheduleULDataTransmit()
//   - Scheduler selects bidirectional base station
//   - Returns BSSCI operation ID + selected BS EUI
//
// Parameters:
//   - ctx: Request context
//   - req: UL Data Transmit message (includes epEui, userData, nwkSnKey, etc.)
//   - tenantID: Tenant scope for endpoint/BS lookup
//
// Returns:
//   - opId: BSSCI operation ID assigned to this UL transmission
//   - bsEui: Base station EUI selected for transmission
//   - errToken: Error token if scheduling fails, "" on success
func (uls *ulService) ScheduleULTransmit(
	ctx context.Context,
	req *mioty.ULDataTransmit,
	tenantID int64,
) (int64, uint64, string) {
	// Nil check for scheduler (feature guard)
	if uls.ulScheduler == nil {
		uls.logger.WarnContext(ctx, scaci.LogSCACIULDataTxNotSupported,
			"epEui", req.EpEui)
		return 0, 0, scaci.ErrULTransmitNotSupported
	}

	// §3.9.1 Preference lookup: explicit bsEui wins, else try preference, else fallback
	targetBsEui := req.BsEui
	if targetBsEui == nil && uls.statusSvc != nil {
		// Query preferred BS based on endpoint's last_attached_bs_eui
		epEuiBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(epEuiBytes, req.EpEui)
		preferredBs, found, lookupErr := uls.statusSvc.GetPreferredBaseStation(ctx, tenantID, epEuiBytes)
		if lookupErr != nil {
			// Log warning but don't fail - proceed with fallback selection
			uls.logger.WarnContext(ctx, scaci.LogSCACIPreferenceLookupFailed,
				"epEui", req.EpEui,
				"error", lookupErr)
		} else if found && preferredBs != nil {
			targetBsEui = preferredBs
			uls.logger.DebugContext(ctx, scaci.LogSCACIUsingPreferredBS,
				"epEui", req.EpEui,
				"preferredBsEui", *preferredBs)
		}
		// If not found or NULL, targetBsEui remains nil → scheduler uses deterministic fallback
	}

	// Delegate to BSSCI scheduler with resolved target
	bssciOpID, actualBsEui, err := uls.ulScheduler.ScheduleULDataTransmit(
		tenantID, req, targetBsEui)

	if err != nil {
		uls.logger.ErrorContext(ctx, scaci.LogSCACIScheduleULTxFailed,
			"epEui", req.EpEui,
			"error", err)

		// Map scheduler errors to SCACI error tokens per §3.9
		var errorToken string
		switch {
		case errors.Is(err, scheduler.ErrSchedulerNoResources):
			// No bidirectional BS available - temporary condition, AC should retry
			errorToken = scaci.ErrBaseStationUnavailable
		case errors.Is(err, scheduler.ErrSchedulerResourceMissing):
			// Specific BS not available - resource missing
			errorToken = scaci.ErrBaseStationUnavailable
		case errors.Is(err, bssci.ErrBaseStationTenantMismatch):
			// Security boundary: requested BS belongs to another tenant
			errorToken = scaci.ErrBaseStationNotFound
		case errors.Is(err, bssci.ErrSessionNotReady), errors.Is(err, bssci.ErrSessionNotBidirectional):
			// Session exists but not ready or not bidirectional yet
			errorToken = scaci.ErrBaseStationUnavailable
		case errors.Is(err, storage.ErrNotFound):
			// Endpoint/BS not found in database
			errorToken = scaci.ErrBaseStationNotFound
		default:
			// True transport/I/O failure
			errorToken = scaci.ErrFailedRecordOperation
		}

		return 0, 0, errorToken
	}

	// Success
	uls.logger.Debug(scaci.LogSCACIULDataTxScheduled,
		"bssciOpID", bssciOpID,
		"bsEui", actualBsEui,
		"epEui", req.EpEui)

	return bssciOpID, actualBsEui, ""
}
