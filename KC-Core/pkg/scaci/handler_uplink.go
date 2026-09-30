package scaci

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/encoding"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// handleULDataResponse processes ULDataResponse messages per SCACI §3.8.2
//
// AC acknowledges receipt of UL data. SC must then send ULDataComplete to finish handshake.
func (s *Server) handleULDataResponse(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIReceivedULDataResponse,
		logger.FieldOpID, opId,
		logger.FieldAcEui, mioty.FormatEUI64(session.AcEui))

	session.UpdateLastSeen(s.clock.Now())

	// Mark operation acknowledged (even if ulDataCmp send later fails)
	if session.ID > 0 && s.operationRepo != nil {
		ackCtx, ackCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer ackCancel()

		if err := s.operationRepo.UpdateOperationState(ackCtx, session.ID, opId,
			models.OperationStateAcknowledged, nil); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIUpdateULDataOpAckFailed,
				logger.FieldOpID, opId,
				logger.FieldError, err)
		}
	}

	// Send ULDataComplete per SCACI §3.8.3 (same opId, no new counter)
	session.WriteMu.Lock()
	cmpMsg := ULDataComplete{
		BaseMessage: BaseMessage{
			Command: CmdULDataComplete,
			OpId:    opId,
		},
	}
	err := s.SendULDataComplete(conn, session, &cmpMsg)
	session.WriteMu.Unlock()

	if err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACISendULDataCompleteFailed,
			logger.FieldOpID, opId,
			logger.FieldError, err)

		s.markOperationFailed(session, opId, map[string]interface{}{
			"errorToken":  errFailedRecordOperation,
			"errorDetail": fmt.Sprintf(errDetailFmtSendULDataCmp, err),
		})

		return err
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIULDataHandshakeComplete,
		logger.FieldOpID, opId)

	// Mark operation completed
	if session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId,
			models.OperationStateCompleted, nil); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIMarkULDataOpCompleteFailed,
				logger.FieldOpID, opId,
				logger.FieldError, err)
		}
	}

	return nil
}

// handleULDataTransmit processes UL Data Transmit messages per SCACI §3.9.1
//
// Handler Responsibilities (Transport Layer):
//   - Decode MessagePack payload
//   - Validate tenant ownership of specific BS (if requested)
//   - Record operation for resume safety
//   - Send response frame
//   - Update operation state tracking
//
// Service Responsibilities (Business Logic):
//   - Scheduler availability guard
//   - Delegation to BSSCI scheduler
//   - Error mapping (scheduler errors → SCACI tokens)
func (s *Server) handleULDataTransmit(conn net.Conn, session *Session, opId int64, payload []byte) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	// Step 1: Decode UL data transmit payload (transport layer)
	var req ULDataTransmit
	if err := decodePayload(payload, &req); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIDecodeULDataTxFailed,
			logger.FieldOpID, opId,
			logger.FieldError, err)
		s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, decodeFailureToken(err, errInvalidULDataTxPayload, errInvalidNwkSnKeyLength))
		return nil
	}

	// Step 1b: §2.4 - Validate mandatory fields before any processing or operation recording
	if errToken := ValidateULDataTransmit(&req); errToken != "" {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIULDataTxValidationFailed,
			logger.FieldOpID, opId,
			logger.FieldErrorToken, errToken)
		s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errToken)
		return nil
	}

	// Validate tenant ownership if specific BS requested
	// Note: Preference lookup for nil bsEui is handled in ULService.ScheduleULTransmit (§3.9.1)
	if req.BsEui != nil {
		bsCtx, bsCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer bsCancel()

		bsEuiBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(bsEuiBytes, *req.BsEui)
		_, err := s.statusSvc.GetBaseStation(bsCtx, session.TenantID, bsEuiBytes)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				s.logger.WarnContext(s.sessionContext(session), LogSCACIBaseStationNotFoundULTx,
					logger.FieldBsEui, *req.BsEui,
					logger.FieldTenantIDCamel, session.TenantID)
				s.sendErrorWithCatalog(conn, session, opId, POSIX_ENOENT, errBaseStationNotFound)
				return nil
			}
			s.logger.ErrorContext(s.sessionContext(session), LogSCACILookupBaseStationFailed,
				logger.FieldBsEui, *req.BsEui,
				logger.FieldError, err)
			s.sendErrorWithCatalog(conn, session, opId, POSIX_EIO, errFailedVerifyBS)
			return nil
		}
	}
	// If req.BsEui is nil, ULService will handle preference lookup and fallback selection

	// §2.4/§3.9.1: Normalize optional format field to default value 0
	format := uint8(0)
	if req.Format != nil {
		format = *req.Format
	}

	// Step 3: Record operation for resume safety (BEFORE scheduling)
	if session.ID > 0 && s.operationRepo != nil {
		recCtx, recCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer recCancel()

		// Build request data (NO raw keys)
		requestData := map[string]interface{}{
			"epEui":     mioty.FormatEUI64(req.EpEui),
			"shAddr":    req.ShAddr,
			"packetCnt": req.PacketCnt,
			"format":    format, // Always record normalized format (never nil)
		}
		if req.BsEui != nil {
			requestData["bsEui"] = mioty.FormatEUI64(*req.BsEui)
		}
		if req.Profile != nil {
			requestData["profile"] = *req.Profile
		}
		if req.UserData != nil {
			requestData["userData"] = encoding.EncodeUserData(req.UserData)
		}

		if err := s.operationRecorder.Record(recCtx, session, opId, CmdULDataTransmit, models.OperationDirectionInbound, requestData); err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogSCACIRecordULTxOpFailed,
				logger.FieldOpID, opId,
				logger.FieldError, err)
			s.sendErrorWithCatalog(conn, session, opId, POSIX_EIO, errFailedRecordOperation)
			return nil
		}
	}

	// Step 4: Delegate to ULService for business logic
	// Convert ULDataTransmit to mioty.ULDataTransmit for service call
	miotyReq := &mioty.ULDataTransmit{
		EpEui:     req.EpEui,
		ShAddr:    req.ShAddr,
		PacketCnt: req.PacketCnt,
		NwkSnKey:  req.NwkSnKey,
		UserData:  req.UserData,
		BsEui:     req.BsEui,
		Profile:   req.Profile,
		Format:    &format, // Always non-nil defaulted pointer - prevents downstream nil derefs
	}

	ctx := s.sessionContext(session)
	bssciOpID, actualBsEui, errToken := s.ulSvc.ScheduleULTransmit(ctx, miotyReq, session.TenantID)
	if errToken != "" {
		// Service returned error token - map to POSIX code and send error
		posixCode := POSIX_EINVAL
		switch errToken {
		case errULTransmitNotSupported:
			posixCode = POSIX_ENOTSUP
		case errBaseStationUnavailable:
			posixCode = POSIX_EAGAIN
		case errBaseStationNotFound:
			posixCode = POSIX_ENOENT
		case errFailedRecordOperation:
			posixCode = POSIX_EIO
		}

		s.markOperationFailed(session, opId, map[string]interface{}{
			"errorToken":  errToken,
			"errorDetail": errDetailULTransmitSchedFailed,
		})

		s.sendErrorWithCatalog(conn, session, opId, posixCode, errToken)
		return nil
	}

	// Step 5: Send response (transport layer)
	resp := ULDataTransmitResponse{
		BaseMessage: mioty.BaseMessage{
			CommandType: CmdULDataTransmitResponse,
			OpId:        opId,
		},
	}

	if err := s.SendULDataTransmitResponse(conn, session, &resp); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACISendULDataTxRspFailed,
			logger.FieldOpID, opId,
			logger.FieldError, err)
		return err
	}

	// Step 6: Mark operation acknowledged
	if session.ID > 0 && s.operationRepo != nil {
		ackCtx, ackCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer ackCancel()

		responseData := map[string]interface{}{
			"bssciOpID": bssciOpID,
			"bsEui":     mioty.FormatEUI64(actualBsEui),
		}

		if err := s.operationRepo.UpdateOperationState(ackCtx, session.ID, opId,
			models.OperationStateAcknowledged, responseData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIMarkULTxAckFailed,
				logger.FieldOpID, opId,
				logger.FieldError, err)
		}
	}

	// Step 7: Update session activity and log success
	session.UpdateLastSeen(s.clock.Now())

	s.logger.InfoContext(s.sessionContext(session), LogSCACIULDataTxScheduled,
		logger.FieldOpID, opId,
		logger.FieldBssciOpID, bssciOpID,
		logger.FieldEpEui, req.EpEui,
		logger.FieldBsEui, actualBsEui)

	return nil
}

// handleULDataTransmitComplete processes UL Data Transmit Complete messages per SCACI §3.9.3
//
// AC completes the three-way handshake after receiving ulDataTxRsp.
// No response per spec.
func (s *Server) handleULDataTransmitComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIProcessingULDataTxCmp,
		logger.FieldOpID, opId,
		logger.FieldAcEui, mioty.FormatEUI64(session.AcEui))

	// Update session activity
	session.UpdateLastSeen(s.clock.Now())

	// Mark operation completed
	if session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		completionData := map[string]interface{}{
			"completedAt": s.clock.Now().UTC().Format(time.RFC3339Nano),
		}

		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId,
			models.OperationStateCompleted, completionData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIMarkULTxOpCompleteFailed,
				logger.FieldOpID, opId,
				logger.FieldError, err)
		}
	}

	// No response per SCACI §3.9.3
	return nil
}

// rejectACIssuedULDataComplete answers an application-center ulDataCmp:
// the service center completes its own uplink handshake (SCACI §3.8.3).
func (s *Server) rejectACIssuedULDataComplete(conn net.Conn, session *Session, opId int64) error {
	return s.rejectACIssuedComplete(conn, session, opId, LogSCACIUnexpectedULDataCmp, errProtocolViolationULCmp, errDetailACSentULDataCmp)
}

// rejectACIssuedComplete fails the operation and answers EPROTO: the initiator
// of an operation is the party that completes it, so a complete from the
// application center for an SC-initiated operation is a protocol violation.
func (s *Server) rejectACIssuedComplete(conn net.Conn, session *Session, opId int64, logMsg, errToken, errDetail string) error {
	s.logger.ErrorContext(s.sessionContext(session), logMsg,
		logger.FieldOpID, opId,
		logger.FieldAcEui, mioty.FormatEUI64(session.AcEui),
		logger.FieldErrorToken, errToken)

	if session.ID > 0 && s.operationRepo != nil {
		errCtx, errCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer errCancel()

		if err := s.operationRepo.UpdateOperationState(errCtx, session.ID, opId,
			models.OperationStateFailed, map[string]interface{}{
				MetadataKeyErrorToken:  errToken,
				MetadataKeyErrorDetail: errDetail,
			}); err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogSCACIUpdateOperationStateFailed, logger.FieldError, err)
		}
	}

	s.sendErrorWithCatalog(conn, session, opId, POSIX_EPROTO, errToken)
	return nil
}

// rejectACIssuedULDataTransmitResponse answers an application-center
// ulDataTxRsp: the service center sends that response (SCACI §3.9.2).
func (s *Server) rejectACIssuedULDataTransmitResponse(conn net.Conn, session *Session, opId int64) error {
	// AC cannot send ulDataTxRsp - protocol violation per SCACI §3.9.2
	s.logger.ErrorContext(s.sessionContext(session), LogSCACIUnexpectedULDataTxRsp,
		logger.FieldOpID, opId,
		logger.FieldAcEui, mioty.FormatEUI64(session.AcEui))

	// Mark operation failed if it exists
	if session.ID > 0 && s.operationRepo != nil {
		errCtx, errCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer errCancel()

		if err := s.operationRepo.UpdateOperationState(errCtx, session.ID, opId,
			models.OperationStateFailed, map[string]interface{}{
				"errorToken":  errProtocolViolationULRsp,
				"errorDetail": errDetailACSentULDataTxRsp,
			}); err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogSCACIUpdateOperationStateFailed, logger.FieldError, err)
		}
	}

	s.sendErrorWithCatalog(conn, session, opId, POSIX_EPROTO, errProtocolViolationULRsp)
	return nil
}
