package scaci

import (
	"context"
	"errors"
	"net"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// handleRegister processes Register messages per SCACI §3.6.1-3.6.3
//
// Handler Responsibilities (Transport Layer):
//   - Decode MessagePack payload
//   - Record operation for resume safety
//   - Send response frame
//   - Update operation state tracking
//
// Service Responsibilities (Business Logic):
//   - Field validation (epEui, nwkKey length)
//   - Endpoint existence check
//   - Create or update endpoint
//   - MIOTY field persistence
func (s *Server) handleRegister(conn net.Conn, session *Session, opId int64, payload []byte) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	// Step 1: Decode register payload (transport layer)
	var req Register
	if err := decodePayload(payload, &req); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIDecodeRegisterFailed,
			logger.FieldOpID, opId,
			logger.FieldError, err)
		s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, decodeFailureToken(err, errInvalidRegisterPayload, errInvalidNwkKeyLength))
		return nil
	}

	// Step 2: Record operation as pending (for resume safety) via operationRecorder
	if session.ID > 0 && s.operationRecorder != nil {
		opCtx, opCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer opCancel()

		// Record all §3.6.1 register fields except nwkKey (security exclusion)
		requestData := map[string]interface{}{
			MetadataKeyEpEui: mioty.FormatEUI64(req.EpEui),
			"bidi":           req.Bidi,
			"preAttach":      req.PreAttach,
			"shAddr":         req.ShAddr,
			"attachCnt":      req.AttachCnt,
			"packetCnt":      req.PacketCnt,
			"dualChan":       req.DualChan,
			"repetition":     req.Repetition,
			"wideCarrOff":    req.WideCarrOff,
			"longBlkDist":    req.LongBlkDist,
			// nwkKey explicitly excluded for security
		}
		if err := s.operationRecorder.Record(opCtx, session, opId, CmdRegister, models.OperationDirectionInbound, requestData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordRegisterOpFailed, logger.FieldError, err)
			// Continue - operation tracking is for resume, not critical path
		}
	}

	// Step 3: Delegate to EndpointService for business logic
	ctx := s.sessionContext(session)
	errToken := s.endpointSvc.Register(ctx, &req, session.TenantID)
	if errToken != "" {
		// Service returned error token - resolve and send
		s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errToken)
		return nil
	}

	// Step 4: Update session activity
	session.UpdateLastSeen(s.clock.Now())

	// Step 5: Send RegisterResponse (transport layer)
	resp := RegisterResponse{
		BaseMessage: BaseMessage{
			Command: CmdRegisterResponse,
			OpId:    opId,
		},
	}

	// Step 6: Update operation state to acknowledged
	if session.ID > 0 && s.operationRepo != nil {
		ackCtx, ackCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer ackCancel()

		if err := s.operationRepo.UpdateOperationState(ackCtx, session.ID, opId, models.OperationStateAcknowledged, nil); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIUpdateOperationStateFailed, logger.FieldError, err)
		}
	}

	return s.SendRegisterResponse(conn, session, &resp)
}

// handleRegisterComplete processes RegisterComplete messages per SCACI §3.6.3:
// it completes a reg the service center answered with regRsp and attaches the
// endpoint when the reg asked for pre-attachment. A reg answered with an
// error is completed by errorAck (SCACI §3.14), so a regCmp of any operation
// other than an answered reg is a protocol error.
func (s *Server) handleRegisterComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIRegisterHandshakeComplete,
		logger.FieldOpID, opId,
		logger.FieldTenantIDCamel, session.TenantID)

	session.UpdateLastSeen(s.clock.Now())

	ctx, cancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
	defer cancel()

	op, err := s.operationRepo.GetOperationByOpID(ctx, session.ID, opId)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		s.logger.WarnContext(s.sessionContext(session), LogSCACILoadRegisterOpFailed, logger.FieldOpID, opId, logger.FieldError, err)
		s.sendErrorWithCatalog(conn, session, opId, POSIX_EIO, errDatabaseError)
		return nil
	}
	if !answeredRegister(op) {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIUnexpectedRegisterComplete, logger.FieldOpID, opId)
		s.sendErrorWithCatalog(conn, session, opId, POSIX_EPROTO, errUnexpectedRegisterComplete)
		return nil
	}

	if err := s.operationRepo.UpdateOperationState(ctx, session.ID, opId, models.OperationStateCompleted, nil); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIMarkOperationCompleteFailed, logger.FieldError, err)
	}
	if epEui, ok := extractEUIFromJSON(op.RequestData, MetadataKeyEpEui); ok && epEui != 0 {
		s.preAttachRegisteredEndpoint(session, epEui)
	}

	return nil // No response per spec
}

// answeredRegister reports whether op is a reg the service center answered
// with regRsp, the only operation a regCmp completes.
func answeredRegister(op *models.SCACIOperation) bool {
	return op != nil && op.Command == CmdRegister && op.State == string(models.OperationStateAcknowledged)
}
