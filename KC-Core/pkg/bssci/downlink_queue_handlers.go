package bssci

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler" // Import neutral scheduler contracts
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
)

// validatePayloadSizes checks each payload entry against MaxDLUserDataBytes (MIOTY radio protocol §3.6.6.3).
func validatePayloadSizes(payloads [][]byte) error {
	for i, p := range payloads {
		if len(p) > mioty.MaxDLUserDataBytes {
			return fmt.Errorf(errFmtPayloadEntryTooLarge,
				ResolveErrorMessage(errDLPayloadTooLarge), i, len(p), mioty.MaxDLUserDataBytes)
		}
	}
	return nil
}

// handleDLDataQueueResponse handles dlDataQueRsp from base station
func (s *Server) handleDLDataQueueResponse(session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// According to MIOTY BSSCI v1.0.0 spec, dlDataQueRsp only contains:
	// - command (String dlDataQueRsp)
	// - opId (Numeric ID of the operation)

	s.logger.InfoContext(s.sessionContext(session), LogBSSCIReceivedDLDataQueRspFromBaseStation,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	s.recordQueueAcknowledgement(s.sessionContext(session), session, msg.OpId)

	// The service center completes its own SC-initiated dlDataQue operation
	// (BSSCI §3.12): it sends dlDataQueCmp and finalizes the pending operation.
	// A spec-compliant base station never returns dlDataQueCmp, so the pending
	// row is removed here or it leaks.
	complete := s.queueSerializer.BuildDLDataQueueComplete(msg.OpId)
	if err := s.sendMessage(session, complete); err != nil {
		return err
	}
	if err := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingOperationFromDatabase,
			logger.FieldError, err, logger.FieldOpID, msg.OpId)
	}
	return nil
}

// recordQueueAcknowledgement files a dlDataQueRsp under the tenant owning
// the downlink, which a roaming station's session tenant is not (BSSCI §5.12).
func (s *Server) recordQueueAcknowledgement(ctx context.Context, session *Session, opID int64) {
	endpointEUI, queueID, recordedTenant, orgID := s.statusSvc.ExtractQueueMetadata(session, opID)
	if queueID <= 0 {
		return
	}
	owner, owned := s.queueOwnerTenant(ctx, recordedTenant, queueID)
	if !owned {
		return
	}
	ack := QueueAcknowledgement{QueueID: queueID, OwnerTenant: owner, OrganizationID: orgID}
	if err := s.downlinkSvc.ProcessQueueAck(ctx, session, ack); err != nil {
		s.logger.WarnContext(ctx, LogBSSCIFailedToConfirmDownlinkQueued,
			logger.FieldQueID, queueID,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldError, err)
	}
	if endpointEUI == 0 {
		return
	}
	if err := s.auditLogger.RecordQueueAck(ctx, owner, session, endpointEUI, queueID, opID); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToRecordDLDataQueueAcknowledgedEvent, logger.FieldError, err)
	}
}

// rejectQueuedDownlink fails the downlink of a dlDataQue the base station
// answered with error: the station will never transmit it (BSSCI §3.17).
func (s *Server) rejectQueuedDownlink(ctx context.Context, session *Session, opID int64, code int, message string) {
	endpointEUI, queueID, recordedTenant, _ := s.statusSvc.ExtractQueueMetadata(session, opID)
	if queueID <= 0 {
		return
	}
	owner, owned := s.queueOwnerTenant(ctx, recordedTenant, queueID)
	if !owned {
		return
	}
	rejection := QueueRejection{QueueID: queueID, EndpointEUI: endpointEUI, OwnerTenant: owner, Code: code, Message: message}
	if err := s.downlinkSvc.ProcessQueueError(ctx, session, rejection); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToFailRejectedDownlink,
			logger.FieldQueID, queueID,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldError, err)
	}
}

// queueOwnerTenant is the tenant owning the downlink of a dlDataQue
// operation: the one recorded with the operation, else the queue row's owner.
func (s *Server) queueOwnerTenant(ctx context.Context, recorded string, queueID int64) (string, bool) {
	if recorded != "" {
		return recorded, true
	}
	owner, err := s.tenantResolver.ResolveTenant(ctx, queueID)
	if err != nil {
		s.logger.WarnContext(ctx, LogBSSCIQueueOwnerUnresolved, logger.FieldQueID, queueID, logger.FieldError, err)
		return "", false
	}
	return owner, true
}

// QueueDownlink dispatches a SCACI-queued downlink message (SCACI §3.10).
//
// This method satisfies the DownlinkScheduler interface defined in
// KC-Core/pkg/scheduler and is called by SCACI after the queue row has been
// persisted with status 'pending'. Delivery goes through the downlink
// dispatcher's DispatchQueue so both the immediate (SCACI) and deferred
// (dlOpen auto-dispatch) paths share one pending→reserved→queued lifecycle
// and both honor the dlRxStatQry pairing (SCACI §3.10.1, BSSCI rev1 §5.16 /
// classic §3.16).
//
// Returns:
//   - queuedQueId: Actual queue ID assigned (same as requested if no collision)
//   - bsEui: EUI of base station that will deliver the message
//   - error: scheduler.ErrSchedulerNoResources when no station heard or
//     attached the endpoint yet, or the one serving it cannot transmit now (the
//     row waits for the endpoint's next downlink window),
//     scheduler.ErrSchedulerResourceMissing when the tenant has no such
//     endpoint, scheduler.ErrSchedulerQueueNotFound when the row is no longer
//     pending
func (s *Server) QueueDownlink(ctx context.Context, req *mioty.DLDataQueue, tenantID int64, organizationID uuid.UUID) (queuedQueId uint64, bsEui uint64, err error) {
	// Validate individual payload sizes (MIOTY radio protocol §3.6.6.3)
	if err := validatePayloadSizes(req.UserData); err != nil {
		return 0, 0, err
	}

	// Range guard for uint64 -> int64 conversion
	if req.QueId > math.MaxInt64 {
		return 0, 0, fmt.Errorf(errFmtTokenWithQueID, ResolveErrorMessage(errQueueIDOutOfRange), req.QueId)
	}

	session, err := s.servingSession(ctx, tenantID, req.EpEui)
	if err != nil {
		return 0, 0, err
	}

	// Dispatch the exact persisted queue row; the dispatcher reserves it,
	// sends (with dlRxStatQry pairing when the row requests it), and confirms
	// reserved→queued. The reservation is scoped to the organization the row
	// was enqueued under - never the base station session's organization,
	// which under roaming belongs to a different tenant entirely.
	dispatched, err := s.downlinkDispatcher.DispatchQueue(ctx, tenantID, organizationID, session, req.QueId, req.EpEui)
	if err != nil {
		return 0, 0, err
	}
	if !dispatched {
		// No matching pending row: it was never persisted, already dispatched,
		// or revoked in the meantime.
		return 0, 0, scheduler.ErrSchedulerQueueNotFound
	}

	return req.QueId, session.BaseStationEUI, nil
}

// servingSession is the live bidirectional session of the base station that
// serves the tenant's endpoint (radio §3.6.1: only a station hearing the
// endpoint can use its downlink window). The station may belong to any
// tenant, since the lookup is scoped to the tenant's own endpoint (roaming).
// While no station heard or attached the endpoint, or the serving station
// cannot transmit, the row waits for the endpoint's next downlink window.
func (s *Server) servingSession(ctx context.Context, tenantID int64, epEUI uint64) (*Session, error) {
	bsEUI, known, err := s.servingStations.ServingStation(ctx, tenantID, epEUI)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, fmt.Errorf("%w: %w", scheduler.ErrSchedulerResourceMissing, err)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ResolveErrorMessage(errServingStationLookupFailed), err)
	}
	if !known {
		s.logger.DebugContext(ctx, LogBSSCIServingStationUnknown, logger.FieldEpEui, epEUI)
		return nil, scheduler.ErrSchedulerNoResources
	}
	session := s.sessions.byEUI(bsEUI, func(session *Session) bool {
		return session.HandshakeComplete && session.Bidirectional
	})
	if session == nil {
		s.logger.DebugContext(ctx, LogBSSCIServingStationUnavailable,
			logger.FieldEpEui, epEUI, logger.FieldBsEui, bsEUI)
		return nil, scheduler.ErrSchedulerNoResources
	}
	return session, nil
}
