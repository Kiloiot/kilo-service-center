package scaci

import (
	"context"
	"net"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// handleDLDataQueue processes DLDataQueue messages per SCACI §3.10.1
//
// AC queues downlink data for SC to transmit to endpoint.
// This handler delegates to processDLDataQueueCore for single-source business logic,
// then handles socket-specific I/O (response send, operation state updates).
func (s *Server) handleDLDataQueue(conn net.Conn, session *Session, opId int64, payload []byte) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	// Unmarshal request (transport layer responsibility)
	var req DLDataQueue
	if err := decodePayload(payload, &req); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIUnmarshalDLDataQueueFailed, logger.FieldError, err)
		s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, decodeFailureToken(err, errMalformedPayload, errMalformedPayload))
		return nil
	}
	// SCACI §3.10.1: the Application Center assigns the queue id, any
	// 64-bit value, zero included.
	ctx := s.sessionContext(session)
	result, errToken, posixCode := s.processDLDataQueueCore(ctx, session, opId, &req, &req.QueId)
	if errToken != "" {
		s.sendErrorWithCatalog(conn, session, opId, posixCode, errToken)
		return nil
	}

	// Success - send response (transport layer)
	resp := DLDataQueueResponse{
		BaseMessage: mioty.BaseMessage{
			CommandType: CmdDLDataQueueResponse,
			OpId:        opId,
		},
	}

	if err := s.SendDLDataQueueResponse(conn, session, &resp); err != nil {
		return err
	}

	// Mark operation as acknowledged with queId/bsEui metadata (socket path only)
	if session.ID > 0 && s.operationRepo != nil {
		ackCtx, ackCancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
		defer ackCancel()

		responseData := map[string]interface{}{
			"queId":          operationLogUint64(req.QueId),
			"status":         result.Status(),
			"acknowledgedAt": s.clock.Now().UTC().Format(time.RFC3339Nano),
		}
		if !result.Deferred {
			responseData["bsEui"] = mioty.FormatEUI64(result.BsEui)
		}
		if err := s.operationRepo.UpdateOperationState(ackCtx, session.ID, opId,
			models.OperationStateAcknowledged, responseData); err != nil {
			s.logger.WarnContext(ctx, LogSCACIUpdateOperationStateFailed, logger.FieldError, err)
		}
	}

	session.UpdateLastSeen(s.clock.Now())

	return nil
}

// handleDLDataQueueComplete processes DLDataQueueComplete messages per SCACI §3.10.3
func (s *Server) handleDLDataQueueComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	// Mark operation as completed (guard session.ID). No response data: the
	// acknowledgement recorded the downlink's status, which completion keeps.
	if session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId,
			models.OperationStateCompleted, nil); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIMarkOperationCompleteFailed, logger.FieldError, err)
		}
	}

	session.UpdateLastSeen(s.clock.Now())

	s.logger.DebugContext(s.sessionContext(session), LogSCACIDLDataQueueHandshakeComplete,
		logger.FieldOpID, opId)

	// No response per SCACI §3.10.3
	return nil
}

// handleDLDataRevoke processes DLDataRevoke messages per SCACI §3.11 DL Data Revoke
//
// Handler Responsibilities (Transport Layer):
//   - Decode MessagePack payload
//   - Validate mandatory fields (epEui, packetCnt)
//   - Resolve packetCnt → every downlink scheduled for it via database lookup
//   - Record operation for resume safety
//   - Send response frame
//   - Update operation state tracking
//
// Service Responsibilities (Business Logic):
//   - Scheduler availability guard
//   - Delegation to BSSCI scheduler (RevokeDownlink)
//   - Error mapping (scheduler errors → SCACI tokens)
//
// Three-way handshake flow:
//  1. AC sends DLDataRevoke (this handler)
//  2. SC sends DLDataRevokeResponse
//  3. AC sends DLDataRevokeComplete (handled by handleDLDataRevokeComplete)
//
// POSIX error mapping per SCACI §3.9:
//   - POSIX_ENOENT (2): Queue entry not found
//   - POSIX_EAGAIN (11): Base station temporarily unavailable
//   - POSIX_ENOTSUP (95): Downlink scheduler not configured
func (s *Server) handleDLDataRevoke(conn net.Conn, session *Session, opId int64, payload []byte) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	// Step 1: Decode MessagePack payload (transport layer)
	var req DLDataRevoke
	if err := decodePayload(payload, &req); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIUnmarshalDLDataRevokeFailed, logger.FieldError, err)
		s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, decodeFailureToken(err, errInvalidDLDataRevPayload, errInvalidDLDataRevPayload))
		return nil
	}

	// Step 2: Validate mandatory fields per SCACI §3.11.1; the decoder already
	// refused an absent one, and zero is a valid packet counter
	if req.EpEui == 0 {
		s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errMissingEpEui)
		return nil
	}
	packetCnt := req.PacketCnt

	s.logger.InfoContext(s.sessionContext(session), LogSCACIProcessingDLDataRevoke,
		logger.FieldOpID, opId,
		logger.FieldPacketCnt, packetCnt,
		logger.FieldEpEui, mioty.FormatEUI64(req.EpEui))

	// Step 3: Resolve packetCnt → the scheduled downlinks via database lookup
	// SCACI §3.11.1 names data by packet counter alone, so a counter-independent downlink cannot be revoked here.
	owner := session.ApplicationCenter()
	lookupCtx, lookupCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
	defer lookupCancel()

	downlinks, err := s.dlSvc.GetDownlinksByPacketCnt(lookupCtx, owner.counterDownlinks(req.EpEui, packetCnt))
	if err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACILookupDownlinkByPacketCntFailed,
			logger.FieldPacketCnt, packetCnt,
			logger.FieldError, err)

		s.markOperationFailed(session, opId, map[string]interface{}{
			"errorToken":  errDatabaseError,
			"errorDetail": err.Error(),
			"packetCnt":   packetCnt,
			"posix_code":  POSIX_EIO,
		})

		s.sendErrorWithCatalog(conn, session, opId, POSIX_EIO, errDatabaseError)
		return nil
	}

	// Nothing scheduled for the counter already holds what the revoke asks for.
	if len(downlinks) == 0 {
		s.logger.InfoContext(s.sessionContext(session), LogSCACIDownlinkNotFoundForPacketCnt,
			logger.FieldPacketCnt, packetCnt,
			logger.FieldEpEui, mioty.FormatEUI64(req.EpEui),
			logger.FieldTenantIDCamel, session.TenantID)
	}

	// The base stations know the downlinks by their service center queue ids;
	// the operation records carry the ids the Application Center knows them by.
	queIDs := make([]uint64, 0, len(downlinks))
	acQueIDs := make([]string, 0, len(downlinks))
	for _, downlink := range downlinks {
		queID, validQueID := downlink.WireQueueID()
		if !validQueID {
			s.logger.ErrorContext(s.sessionContext(session), LogSCACIInvalidQueueIDFromDB,
				logger.FieldQueID, downlink.QueID)
			s.sendErrorWithCatalog(conn, session, opId, POSIX_EIO, errDatabaseError)
			return nil
		}
		queIDs = append(queIDs, queID)
		acQueID := queID
		if downlink.ACQueID != nil {
			acQueID = *downlink.ACQueID
		}
		acQueIDs = append(acQueIDs, operationLogUint64(acQueID))
	}

	// Step 4: Record operation before BSSCI coordination (for session resume)
	if session.ID > 0 && s.operationRecorder != nil {
		recCtx, recCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer recCancel()

		requestData := map[string]interface{}{
			"packetCnt":  packetCnt,
			"epEui":      mioty.FormatEUI64(req.EpEui),
			"queIds":     acQueIDs,
			"receivedAt": s.clock.Now().UTC().Format(time.RFC3339Nano),
		}
		if err := s.operationRecorder.Record(recCtx, session, opId, CmdDLDataRevoke, models.OperationDirectionInbound, requestData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordDLDataRevokeOpFailed, logger.FieldError, err)
		}
	}

	session.UpdateLastSeen(s.clock.Now())

	// Step 5: Revoke every scheduled downlink through the DL service; a
	// failure does not stop the others, and the first one answers the AC.
	ctx := s.sessionContext(session)
	holders := make([]string, 0, len(queIDs))
	var errToken string
	for _, queID := range queIDs {
		bsEui, revokeErrToken := s.dlSvc.RevokeDownlink(ctx, owner.downlinkRef(req.EpEui, queID))
		if revokeErrToken != "" {
			if errToken == "" {
				errToken = revokeErrToken
			}
			continue
		}
		// A zero EUI means the downlink was revoked in the queue, held by no station.
		if bsEui != 0 {
			holders = append(holders, mioty.FormatEUI64(bsEui))
		}
	}
	if errToken != "" {
		// Service returned error token - map to POSIX code and send error
		posixCode := POSIX_EINVAL
		switch errToken {
		case errDownlinkNotFound:
			posixCode = POSIX_ENOENT
		case errSchedulerUnavailable:
			posixCode = POSIX_ENOTSUP
		case errDatabaseError:
			posixCode = POSIX_EIO
		}

		s.markOperationFailed(session, opId, map[string]interface{}{
			"errorToken":  errToken,
			"errorDetail": errDetailDLRevokeFailed,
			"packetCnt":   packetCnt,
			"queIds":      acQueIDs,
			"posix_code":  posixCode,
		})

		s.sendErrorWithCatalog(conn, session, opId, posixCode, errToken)
		return nil
	}

	// Step 6: Mark operation as acknowledged (BSSCI coordination initiated)
	if session.ID > 0 && s.operationRepo != nil {
		ackCtx, ackCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer ackCancel()

		ackData := map[string]interface{}{
			"packetCnt":      packetCnt,
			"queIds":         acQueIDs,
			"acknowledgedAt": s.clock.Now().UTC().Format(time.RFC3339Nano),
		}
		if len(holders) > 0 {
			ackData["bsEuis"] = holders
		}
		if err := s.operationRepo.UpdateOperationState(ackCtx, session.ID, opId,
			models.OperationStateAcknowledged, ackData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIUpdateOperationStateFailed, logger.FieldError, err)
		}
	}

	// Step 7: Send response per SCACI §3.11.2 (transport layer)
	resp := DLDataRevokeResponse{
		BaseMessage: BaseMessage{
			Command: CmdDLDataRevokeResponse,
			OpId:    opId,
		},
	}

	s.logger.InfoContext(s.sessionContext(session), LogSCACIDLDataRevokeInitiated,
		logger.FieldOpID, opId,
		logger.FieldPacketCnt, packetCnt,
		logger.FieldCount, len(queIDs))

	return s.SendDLDataRevokeResponse(conn, session, &resp)
}

// handleDLDataRevokeComplete processes DLDataRevokeComplete messages per SCACI §3.11.3
//
// This handler completes the three-way handshake for DL Data Revoke. By the time this
// message arrives, the BSSCI layer has already sent dlDataRev to the base station and
// received dlDataRevRsp. The AC is now acknowledging receipt of our DLDataRevokeResponse.
//
// This handler:
//  1. Marks the operation as completed in the operation log
//  2. Updates session last-seen timestamp
//  3. No response is sent per SCACI §3.11.3
func (s *Server) handleDLDataRevokeComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	// Mark operation as completed (guard session.ID)
	if session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		completedAt := s.clock.Now().UTC().Format(time.RFC3339Nano)
		completionData := map[string]interface{}{
			"completedAt": completedAt,
			"status":      ResultRevoked,
		}
		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId,
			models.OperationStateCompleted, completionData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIMarkOperationCompleteFailed, logger.FieldError, err)
		}
	}

	session.UpdateLastSeen(s.clock.Now())

	s.logger.DebugContext(s.sessionContext(session), LogSCACIDLDataRevokeHandshakeComplete,
		logger.FieldOpID, opId)

	// No response per SCACI §3.11.3
	return nil
}

// handleDLDataResultResponse processes DLDataResultResponse messages per SCACI §3.12.2
//
// AC acknowledges receipt of DL data result. SC must then send DLDataResultComplete to finish handshake.
func (s *Server) handleDLDataResultResponse(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIReceivedDLDataResultResponse,
		logger.FieldOpID, opId,
		logger.FieldAcEui, mioty.FormatEUI64(session.AcEui))

	session.UpdateLastSeen(s.clock.Now())

	// Mark operation acknowledged
	if session.ID > 0 && s.operationRepo != nil {
		ackCtx, ackCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer ackCancel()

		if err := s.operationRepo.UpdateOperationState(ackCtx, session.ID, opId,
			models.OperationStateAcknowledged, nil); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIUpdateDLResultOpAckFailed,
				logger.FieldOpID, opId,
				logger.FieldError, err)
		}
	}

	// Send DLDataResultComplete per SCACI §3.12.3 (same opId, no new counter)
	session.WriteMu.Lock()
	cmpMsg := DLDataResultComplete{
		BaseMessage: BaseMessage{
			Command: CmdDLDataResultComplete,
			OpId:    opId,
		},
	}
	err := s.SendDLDataResultComplete(conn, session, &cmpMsg)
	session.WriteMu.Unlock()

	if err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACISendDLResultCompleteFailed,
			logger.FieldOpID, opId,
			logger.FieldError, err)

		responseData := map[string]interface{}{
			MetadataKeyErrorToken:  errFailedRecordOperation,
			MetadataKeyErrorDetail: err.Error(),
		}
		s.markOperationFailed(session, opId, responseData)
		return err
	}

	// Mark operation completed after successful send
	if session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		responseData := map[string]interface{}{
			metadataKeyCompletedAt: s.clock.Now().UTC().Format(time.RFC3339),
		}
		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId,
			models.OperationStateCompleted, responseData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIUpdateDLResultOpCompleteFailed,
				logger.FieldOpID, opId,
				logger.FieldError, err)
		}
	}

	return nil
}

// rejectACIssuedDLDataResultComplete answers an application-center
// txDataResCmp: the Service Center initiates and completes the DL data result
// operation (§3.12.3).
func (s *Server) rejectACIssuedDLDataResultComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}
	return s.rejectACIssuedComplete(conn, session, opId, LogSCACIUnexpectedDLResultComplete, errProtocolViolationDLResCmp, errDetailACSentTxDataResCmp)
}
