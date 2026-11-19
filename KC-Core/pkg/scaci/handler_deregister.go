package scaci

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// handleDeregister processes Deregister messages per SCACI §3.7.1
//
// Handler Responsibilities (Transport Layer):
//   - Decode MessagePack payload
//   - Record operation for resume safety
//   - Send response frame
//   - Update operation state tracking
//
// Service Responsibilities (Business Logic):
//   - Field validation (epEui)
//   - Endpoint existence check (tenant-scoped)
//   - Mark endpoint inactive (DetachEndpoint)
//
// NOTE: BSSCI detach propagation happens in handleDeregisterComplete, not here.
// This maintains spec-compliant three-way handshake (wait for AC confirmation).
func (s *Server) handleDeregister(conn net.Conn, session *Session, opId int64, payload []byte) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	// Step 1: Decode deregister payload (transport layer)
	var req Deregister
	if err := decodePayload(payload, &req); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIDecodeDeregisterFailed,
			logger.FieldOpID, opId,
			logger.FieldError, err)
		s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, decodeFailureToken(err, errInvalidDeregisterPayload, errInvalidDeregisterPayload))
		return nil
	}

	// Step 2: Record operation as pending (for resume safety) via operationRecorder
	if session.ID > 0 && s.operationRecorder != nil {
		opCtx, opCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer opCancel()

		requestData := map[string]interface{}{MetadataKeyEpEui: mioty.FormatEUI64(req.EpEui)}
		if err := s.operationRecorder.Record(opCtx, session, opId, CmdDeregister, models.OperationDirectionInbound, requestData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordDeregisterOpFailed, logger.FieldError, err)
			// Continue - operation tracking is for resume, not critical path
		}
	}

	// Step 3: Delegate to EndpointService for business logic
	ctx := s.sessionContext(session)
	errToken := s.endpointSvc.Deregister(ctx, req.EpEui, session.TenantID)
	if errToken != "" {
		s.sendErrorWithCatalog(conn, session, opId, deregisterFailureCode(errToken), errToken)
		return nil
	}

	// Step 4: Update session activity
	session.UpdateLastSeen(s.clock.Now())

	return s.acknowledgeDeregister(conn, session, opId)
}

// deregisterFailureCode is the POSIX code a deregistration refused with errToken answers with.
func deregisterFailureCode(errToken string) int {
	switch errToken {
	case errEndpointNotFound:
		return POSIX_ENOENT
	case errDatabaseError, errFailedUpdateEndpoint:
		return POSIX_EIO
	default:
		return POSIX_EINVAL
	}
}

// acknowledgeDeregister answers a deregistration with deregRsp, announces the
// detachment it made, and records the operation acknowledged or failed.
func (s *Server) acknowledgeDeregister(conn net.Conn, session *Session, opId int64) error {
	resp := DeregisterResponse{
		BaseMessage: BaseMessage{
			Command: CmdDeregisterResponse,
			OpId:    opId,
		},
	}

	// The operation is acknowledged only after deregRsp is sent.
	sendErr := s.SendDeregisterResponse(conn, session, &resp)
	if sendErr != nil {
		failMeta := map[string]interface{}{
			MetadataKeyErrorToken:  errSendDeregisterResponseFailed,
			MetadataKeyErrorDetail: sendErr.Error(),
		}
		s.markOperationFailed(session, opId, failMeta)
		return sendErr
	}

	if session.ID > 0 && s.operationRepo != nil {
		ackCtx, ackCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer ackCancel()

		if err := s.operationRepo.UpdateOperationState(ackCtx, session.ID, opId, models.OperationStateAcknowledged, nil); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIUpdateOperationStateFailed, logger.FieldError, err)
		}
	}

	return nil
}

// handleDeregisterComplete processes DeregisterComplete messages per SCACI
// §3.7.3: it completes a dereg the service center answered with deregRsp,
// revokes the endpoint's downlinks and sends the tenant's endpoint detPrp. A
// dereg answered with an error is completed by errorAck (SCACI §3.14), so a
// deregCmp of any operation other than an answered dereg is a protocol error.
func (s *Server) handleDeregisterComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIDeregisterHandshakeComplete, logger.FieldOpID, opId)

	session.UpdateLastSeen(s.clock.Now())

	ctx, cancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
	defer cancel()

	op, err := s.operationRepo.GetOperationByOpID(ctx, session.ID, opId)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		s.logger.WarnContext(s.sessionContext(session), LogSCACILoadDeregisterOpFailed, logger.FieldOpID, opId, logger.FieldError, err)
		s.sendErrorWithCatalog(conn, session, opId, POSIX_EIO, errDatabaseError)
		return nil
	}
	epEui, ok := deregisteredEndpoint(op)
	if !ok {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIUnexpectedDeregisterComplete, logger.FieldOpID, opId)
		s.sendErrorWithCatalog(conn, session, opId, POSIX_EPROTO, errUnexpectedDeregisterComplete)
		return nil
	}

	cleanup := s.detachDeregisteredEndpoint(ctx, session, opId, epEui)
	if err := s.operationRepo.UpdateOperationState(ctx, session.ID, opId, cleanup.state(), cleanup.responseData()); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIMarkOperationCompleteFailed, logger.FieldError, err)
	}

	return nil // No response per spec
}

// answeredDeregister reports whether op is a dereg the service center
// answered with deregRsp, the only operation a deregCmp completes.
func answeredDeregister(op *models.SCACIOperation) bool {
	return op != nil && op.Command == CmdDeregister && op.State == string(models.OperationStateAcknowledged)
}

// deregisteredEndpoint is the endpoint op deregistered, when op is a dereg
// answered with deregRsp.
func deregisteredEndpoint(op *models.SCACIOperation) (uint64, bool) {
	if !answeredDeregister(op) {
		return 0, false
	}
	epEui, ok := extractEUIFromJSON(op.RequestData, MetadataKeyEpEui)
	return epEui, ok && epEui != 0
}

// deregisterCleanup is what completing a deregistration did to the
// endpoint's downlinks and to the base stations (SCACI §3.7.3).
type deregisterCleanup struct {
	epEui        uint64
	revoked      int
	revokeErr    error
	detachErrors int
}

// state is completed when the cleanup succeeded and completed with warnings
// when a revoke or a detach propagation failed.
func (c deregisterCleanup) state() models.OperationState {
	if c.revokeErr != nil || c.detachErrors > 0 {
		return models.OperationStateCompletedWithWarnings
	}
	return models.OperationStateCompleted
}

// responseData records the cleanup in the operation log.
func (c deregisterCleanup) responseData() map[string]interface{} {
	status := opStatusSuccess
	if c.state() == models.OperationStateCompletedWithWarnings {
		status = opStatusPartialFailure
	}
	data := map[string]interface{}{
		MetadataKeyEpEui:            mioty.FormatEUI64(c.epEui),
		MetadataKeyRevokedCount:     c.revoked,
		MetadataKeyDetachErrorCount: c.detachErrors,
		MetadataKeyCleanupStatus:    status,
	}
	if c.revokeErr != nil {
		data[MetadataKeyErrorToken] = errRevokeDownlinksFailed
		data[MetadataKeyErrorDetail] = c.revokeErr.Error()
	}
	return data
}

// detachDeregisteredEndpoint revokes the tenant's endpoint's downlinks and
// sends its detPrp to every connected base station.
func (s *Server) detachDeregisteredEndpoint(ctx context.Context, session *Session, opId int64, epEui uint64) deregisterCleanup {
	s.logger.DebugContext(s.sessionContext(session), LogSCACIDeregisterCleanupStart,
		logger.FieldOpID, opId,
		logger.FieldEpEui, mioty.FormatEUI64(epEui))

	cleanup := deregisterCleanup{epEui: epEui}
	cleanup.revoked, cleanup.revokeErr = s.revokeEndpointDownlinks(ctx, session.ApplicationCenter(), epEui)
	if cleanup.revokeErr != nil {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIRevokeDownlinksFailed, logger.FieldEpEui, epEui, logger.FieldError, cleanup.revokeErr)
	}

	if errs := s.endpointSvc.PropagateDetachToAll(ctx, session.TenantID, epEui); len(errs) > 0 {
		cleanup.detachErrors = len(errs)
		s.logger.WarnContext(s.sessionContext(session), LogSCACIDetachPropagationErrors, logger.FieldCount, cleanup.detachErrors)
	} else {
		s.logger.DebugContext(s.sessionContext(session), LogSCACIDetachPropagationSent, logger.FieldEpEui, epEui)
	}
	return cleanup
}

// revokeEndpointDownlinks revokes every in-flight downlink the Application
// Center queued for the endpoint through the revoke path dlDataRev uses, and
// reports how many it revoked; the count stands when some revokes fail.
func (s *Server) revokeEndpointDownlinks(ctx context.Context, owner ApplicationCenter, epEui uint64) (int, error) {
	deviceEUI := mioty.FormatEUI64(epEui)
	if s.dlSvc == nil {
		s.logger.WarnContext(ctx, LogSCACIStorageNotAvailableRevoke,
			logger.FieldTenantIDCamel, owner.TenantID,
			logger.FieldEpEui, deviceEUI)
		return 0, errDownlinkServiceUnavailable
	}

	s.logger.DebugContext(ctx, LogSCACIRevokingDownlinksForEndpoint,
		logger.FieldTenantIDCamel, owner.TenantID,
		logger.FieldEpEui, deviceEUI)

	downlinks, err := s.dlSvc.GetDownlinkQueue(ctx, owner.TenantID, owner.endpointQueue(epEui))
	if err != nil {
		s.logger.ErrorContext(ctx, LogSCACIQueryDownlinkQueueFailed,
			logger.FieldEpEui, deviceEUI,
			logger.FieldError, err)
		return 0, fmt.Errorf(errFmtQueryDownlinkQueue, err)
	}

	if len(downlinks) == 0 {
		s.logger.DebugContext(ctx, LogSCACINoPendingDownlinksToRevoke, logger.FieldEpEui, deviceEUI)
		return 0, nil
	}

	// The revoke path revokes a downlink still in the queue there and one a
	// base station holds at that station with dlDataRev (BSSCI §3.13).
	var failed, revokedCount int
	for _, dl := range downlinks {
		errToken := ErrDownlinkNotFound
		if dl.QueID > 0 {
			_, errToken = s.dlSvc.RevokeDownlink(ctx, owner.downlinkRef(epEui, uint64(dl.QueID)))
		}
		if errToken != "" {
			s.logger.WarnContext(ctx, LogSCACIRevokeDownlinkItemFailed,
				logger.FieldQueID, dl.QueID,
				logger.FieldEpEui, deviceEUI,
				logger.FieldErrorToken, errToken)
			failed++
			continue
		}
		revokedCount++
	}

	s.logger.InfoContext(ctx, LogSCACIDownlinkRevocationComplete,
		logger.FieldEpEui, deviceEUI,
		logger.FieldTotal, len(downlinks),
		logger.FieldRevoked, revokedCount,
		logger.FieldFailed, failed)

	if failed > 0 {
		return revokedCount, fmt.Errorf(errFmtRevokeDownlinksPartial, failed, len(downlinks))
	}

	return revokedCount, nil
}
