package bssci

import (
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// handleDLDataResult handles dlDataRes from base station when downlink has been sent or discarded
func (s *Server) handleDLDataResult(session *Session, msg *Message, data map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Payload is normalized by handleMessage before dispatch
	// All mandatory fields guaranteed present and correctly typed
	// Conditional validation enforced: result="sent" requires txTime/packetCnt
	// Enum validation enforced: result must be "sent", "expired", or "invalid"

	// Build canonical MIOTY type from normalized payload
	dlResult := mioty.DLDataResult{
		BaseMessage: mioty.BaseMessage{
			CommandType: mioty.CmdDLDataResult,
			OpId:        msg.OpId,
		},
		EpEui:  data["epEui"].(uint64),  // Normalizer guarantees uint64
		QueId:  data["queId"].(uint64),  // Normalizer guarantees uint64
		Result: data["result"].(string), // Normalizer validates enum
	}

	// Extract optional conditional fields (nil when absent or forbidden)
	if txTime := data["txTime"]; txTime != nil {
		txTimeVal := txTime.(int64)
		dlResult.TxTime = &txTimeVal
	}
	if packetCnt := data["packetCnt"]; packetCnt != nil {
		packetCntVal := packetCnt.(uint32)
		dlResult.PacketCnt = &packetCntVal
	}

	ctx := s.sessionContext(session)

	// Delegate orchestration to DownlinkService (BSSCI §5.14): tenant
	// resolution, the queue update and the report to the downlink's originators.
	responseMsg, err := s.downlinkSvc.ProcessDLDataResult(ctx, session, &dlResult)
	if err != nil {
		catalogErr := catalogErrorOf(err)
		if sendErr := s.sendError(session, msg.OpId, catalogErr.Posix, ResolveErrorMessage(catalogErr.Token)); sendErr != nil {
			s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToSendErrorFrame, logger.FieldError, sendErr)
			return sendErr // Socket is broken, let session tear down
		}
		// Error frame sent successfully, keep session alive for errorAck
		return nil
	}

	// Runtime guard: Verify dlDataResRsp contains only canonical fields per BSSCI §5.14.2
	if len(responseMsg) != 2 {
		s.logger.WarnContext(ctx, LogBSSCIUnexpectedFieldsInDLDataResRsp,
			logger.FieldFieldCount, len(responseMsg),
			logger.FieldOpID, msg.OpId,
			logger.FieldExpectedFields, expectedEnvelopeFields)
	}
	// Verify specific keys
	if _, hasCommand := responseMsg["command"]; !hasCommand {
		s.logger.ErrorContext(ctx, LogBSSCIMissingCommandFieldInResponse, logger.FieldOpID, msg.OpId)
	}
	if _, hasOpId := responseMsg["opId"]; !hasOpId {
		s.logger.ErrorContext(ctx, LogBSSCIMissingOpIDFieldInResponse, logger.FieldOpID, msg.OpId)
	}

	// Send response message returned by service
	return s.sendMessage(session, responseMsg)
}

// ResultWithTransmitter returns the result as reported upstream: bsEui names
// the transmitting base station and only a sent result carries it (SCACI §3.12.1).
func ResultWithTransmitter(result mioty.DLDataResult, bsEUI uint64) mioty.DLDataResult {
	result.BsEui = nil
	if result.Result == mioty.ResultSent {
		result.BsEui = &bsEUI
	}
	return result
}

// handleDLDataResultResponse handles dlDataResRsp from base station
func (s *Server) handleDLDataResultResponse(session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	s.logger.DebugContext(s.sessionContext(session), LogBSSCIReceivedDLDataResRspFromBaseStation,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	// Send dlDataResCmp to complete the three-way handshake via queue serializer
	complete := s.queueSerializer.BuildDLDataResultComplete(msg.OpId)
	return s.sendMessage(session, complete)
}

// handleDLDataResultComplete handles dlDataResCmp from base station
func (s *Server) handleDLDataResultComplete(session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Clean up pending operation from database
	// BSSCI §§5.11-5.12.3 Gap 1: StatusService handles both cache and DB removal
	if err := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingOperationFromDB,
			logger.FieldSessionID, session.DbSessionID,
			logger.FieldOpID, msg.OpId,
			logger.FieldError, err)
	}
	// Note: No manual fallback - trust StatusService single-writer pattern

	s.logger.DebugContext(s.sessionContext(session), LogBSSCIDLDataResultOperationCompleted,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	return nil
}
