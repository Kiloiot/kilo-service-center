package scaci

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/vmihailenco/msgpack/v5"
)

// writeFrame encodes and sends one frame. A failed send may leave part of the
// frame on the TLS stream, so the connection is closed: its read loop ends and
// the session is removed instead of stalling every later writer.
func (s *Server) writeFrame(conn net.Conn, payload []byte) error {
	encoded, err := s.codec.Encode(payload)
	if err != nil {
		return err
	}
	if err := s.codec.Send(conn, encoded); err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			s.logger.DebugContext(s.safeCtx(), LogSCACICloseAfterWriteFailed, logger.FieldError, closeErr)
		}
		return err
	}
	return nil
}

// sendErrorAck sends error acknowledgement per SCACI §3.14.2
//
// Called when SC receives CmdError from AC to complete the error sequence.
//
// Parameters:
//   - conn: Connection to send on
//   - session: Session for logging context
//   - opId: Operation ID from the error message (echoed back)
//
// Returns:
//   - error: Send error
func (s *Server) sendErrorAck(conn net.Conn, session *Session, opId int64) error {
	ack := ErrorAck{
		BaseMessage: BaseMessage{
			Command: CmdErrorAck,
			OpId:    opId,
		},
	}

	if err := s.sendResponse(conn, &ack); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACISentErrorAck, logger.FieldOpID, opId, logger.FieldError, err)
		return err
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACISentErrorAck, logger.FieldOpID, opId)
	return nil
}

// sendResponse sends a message to the AC per SCACI §3
//
// Parameters:
//   - conn: Connection to send on
//   - msg: Message struct (must have Command and OpId)
//
// Returns:
//   - error: Serialization or send error
func (s *Server) sendResponse(conn net.Conn, msg interface{}) error {
	payload, err := msgpack.Marshal(msg)
	if err != nil {
		return fmt.Errorf(errFmtMarshalResponse, err)
	}

	if err := s.writeFrame(conn, payload); err != nil {
		return fmt.Errorf(errFmtSendFrame, err)
	}

	// Log response for debugging
	if msgJSON, err := json.Marshal(msg); err == nil {
		s.logger.DebugContext(s.safeCtx(), LogSCACIResponseSent, logger.FieldMessage, string(msgJSON))
	}

	return nil
}

// SendDLDataResult validates and sends a DLDataResult message.
// All DLDataResult senders MUST call this helper to ensure validation.
// DO NOT bypass this function—validation is mandatory per SCACI §3.12.1.
func (s *Server) SendDLDataResult(conn net.Conn, session *Session, msg *DLDataResult) error {
	if errToken := ValidateDLDataResult(msg); errToken != "" {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIDLResultValidationFailed,
			logger.FieldSpec, specSectionDLDataResult, logger.FieldToken, errToken, logger.FieldOpID, msg.OpId, logger.FieldEpEui, mioty.FormatEUI64(msg.EpEui))
		return fmt.Errorf(errFmtDLDataResultValidationFailed, errToken)
	}
	return s.sendResponse(conn, msg)
}

// SendEPStatus validates and sends an EPStatus message.
// All future EP status notifiers MUST call this helper to ensure validation.
// DO NOT bypass this function—validation is mandatory per SCACI §§3.2, 3.13.1.
func (s *Server) SendEPStatus(conn net.Conn, session *Session, msg *EPStatus) error {
	if errToken := ValidateEPStatus(msg, msg.OpId); errToken != "" {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIEPStatusValidationFailed,
			logger.FieldSpec, specSectionEPStatus, logger.FieldToken, errToken, logger.FieldOpID, msg.OpId, logger.FieldEpEui, mioty.FormatEUI64(msg.EpEui))
		return fmt.Errorf(errFmtEPStatusValidationFailed, errToken)
	}
	return s.sendResponse(conn, msg)
}

// SendConnectResponse validates and sends a ConnectResponse message.
// DO NOT bypass this function—validation is mandatory per SCACI §3.3.2.
//
// SCACI §3.3.2: version is OPTIONAL - defaults to negotiated version from session
// or ProtocolVersionString if session is nil/empty.
func (s *Server) SendConnectResponse(conn net.Conn, session *Session, msg *ConnectResponse) error {
	// Apply version default per SCACI §3.3.2 before validation
	if msg.Version == nil || *msg.Version == "" {
		defaultVer := ProtocolVersionString
		if session != nil && session.NegotiatedVersion != "" {
			defaultVer = session.NegotiatedVersion
		}
		msg.Version = &defaultVer
	}

	if errToken := ValidateConnectResponse(msg); errToken != "" {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIConnectRspValidationFailed,
			logger.FieldSpec, specSectionConnectRsp, logger.FieldToken, errToken, logger.FieldOpID, msg.OpId)
		return fmt.Errorf(errFmtConnectResponseValidationFailed, errToken)
	}
	return s.sendResponse(conn, msg)
}

// SendStatusResponse validates and sends a StatusResponse message.
// DO NOT bypass this function—validation is mandatory per SCACI §3.5.2.
func (s *Server) SendStatusResponse(conn net.Conn, session *Session, msg *StatusResponse) error {
	if errToken := ValidateStatusResponse(msg); errToken != "" {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIStatusRspValidationFailed,
			logger.FieldSpec, specSectionStatusRsp, logger.FieldToken, errToken, logger.FieldOpID, msg.OpId)
		return fmt.Errorf(errFmtStatusResponseValidationFailed, errToken)
	}
	return s.sendResponse(conn, msg)
}

// SendULData validates and sends a ULData message.
// DO NOT bypass this function—validation is mandatory per SCACI §3.8.1.
func (s *Server) SendULData(conn net.Conn, session *Session, msg *ULData) error {
	if errToken := ValidateULData(msg); errToken != "" {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIULDataValidationFailed,
			logger.FieldSpec, specSectionULData, logger.FieldToken, errToken, logger.FieldOpID, msg.OpId, logger.FieldEpEui, mioty.FormatEUI64(msg.EpEui))
		return fmt.Errorf(errFmtULDataValidationFailed, errToken)
	}
	return s.sendResponse(conn, msg)
}

// SendError validates and sends an Error message.
// DO NOT bypass this function—validation is mandatory per SCACI §3.14.1.
func (s *Server) SendError(conn net.Conn, session *Session, msg *Error) error {
	if errToken := ValidateError(msg); errToken != "" {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIErrorMsgValidationFailed,
			logger.FieldSpec, specSectionError, logger.FieldToken, errToken, logger.FieldOpID, msg.OpId)
		return fmt.Errorf(errFmtErrorMessageValidationFailed, errToken)
	}
	return s.sendResponse(conn, msg)
}

// SendPingResponse validates command and sends a PingResponse message.
// Command validation prevents misuse of BaseMessage-only response types.
func (s *Server) SendPingResponse(conn net.Conn, session *Session, msg *PingResponse) error {
	if msg.Command != CmdPingResponse {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			logger.FieldSpec, specSectionPingRsp, logger.FieldExpected, CmdPingResponse, logger.FieldGot, msg.Command, logger.FieldOpID, msg.OpId)
		return fmt.Errorf(errFmtPingRspCommandMismatch, msg.Command, CmdPingResponse)
	}
	return s.sendResponse(conn, msg)
}

// SendRegisterResponse validates command and sends a RegisterResponse message.
// Command validation prevents misuse of BaseMessage-only response types.
func (s *Server) SendRegisterResponse(conn net.Conn, session *Session, msg *RegisterResponse) error {
	if msg.Command != CmdRegisterResponse {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			logger.FieldSpec, specSectionRegisterRsp, logger.FieldExpected, CmdRegisterResponse, logger.FieldGot, msg.Command, logger.FieldOpID, msg.OpId)
		return fmt.Errorf(errFmtRegisterRspCommandMismatch, msg.Command, CmdRegisterResponse)
	}
	return s.sendResponse(conn, msg)
}

// SendDeregisterResponse validates command and sends a DeregisterResponse message.
// Command validation prevents misuse of BaseMessage-only response types.
func (s *Server) SendDeregisterResponse(conn net.Conn, session *Session, msg *DeregisterResponse) error {
	if msg.Command != CmdDeregisterResponse {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			logger.FieldSpec, specSectionDeregisterRsp, logger.FieldExpected, CmdDeregisterResponse, logger.FieldGot, msg.Command, logger.FieldOpID, msg.OpId)
		return fmt.Errorf(errFmtDeregisterRspCommandMismatch, msg.Command, CmdDeregisterResponse)
	}
	return s.sendResponse(conn, msg)
}

// SendULDataComplete validates command and sends a ULDataComplete message.
// Command validation prevents misuse of BaseMessage-only response types.
func (s *Server) SendULDataComplete(conn net.Conn, session *Session, msg *ULDataComplete) error {
	if msg.Command != CmdULDataComplete {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			logger.FieldSpec, specSectionULDataCmp, logger.FieldExpected, CmdULDataComplete, logger.FieldGot, msg.Command, logger.FieldOpID, msg.OpId)
		return fmt.Errorf(errFmtULDataCmpCommandMismatch, msg.Command, CmdULDataComplete)
	}
	return s.sendResponse(conn, msg)
}

// SendULDataTransmitResponse validates command and sends a ULDataTransmitResponse message.
// Command validation prevents misuse of BaseMessage-only response types.
// Note: ULDataTransmitResponse is aliased to mioty.ULDataTransmitResponse which uses CommandType.
func (s *Server) SendULDataTransmitResponse(conn net.Conn, session *Session, msg *ULDataTransmitResponse) error {
	if msg.CommandType != CmdULDataTransmitResponse {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			logger.FieldSpec, specSectionULDataTxRsp, logger.FieldExpected, CmdULDataTransmitResponse, logger.FieldGot, msg.CommandType, logger.FieldOpID, msg.OpId)
		return fmt.Errorf(errFmtULDataTxRspCommandMismatch, msg.CommandType, CmdULDataTransmitResponse)
	}
	return s.sendResponse(conn, msg)
}

// SendDLDataQueueResponse validates command and sends a DLDataQueueResponse message.
// Command validation prevents misuse of BaseMessage-only response types.
// Note: DLDataQueueResponse is aliased to mioty.DLDataQueueResponse which uses CommandType.
func (s *Server) SendDLDataQueueResponse(conn net.Conn, session *Session, msg *DLDataQueueResponse) error {
	if msg.CommandType != CmdDLDataQueueResponse {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			logger.FieldSpec, specSectionDLDataQueRsp, logger.FieldExpected, CmdDLDataQueueResponse, logger.FieldGot, msg.CommandType, logger.FieldOpID, msg.OpId)
		return fmt.Errorf(errFmtDLDataQueRspCommandMismatch, msg.CommandType, CmdDLDataQueueResponse)
	}
	return s.sendResponse(conn, msg)
}

// SendDLDataRevokeResponse validates command and sends a DLDataRevokeResponse message.
// Command validation prevents misuse of BaseMessage-only response types.
func (s *Server) SendDLDataRevokeResponse(conn net.Conn, session *Session, msg *DLDataRevokeResponse) error {
	if msg.Command != CmdDLDataRevokeResponse {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			logger.FieldSpec, specSectionDLDataRevRsp, logger.FieldExpected, CmdDLDataRevokeResponse, logger.FieldGot, msg.Command, logger.FieldOpID, msg.OpId)
		return fmt.Errorf(errFmtDLDataRevRspCommandMismatch, msg.Command, CmdDLDataRevokeResponse)
	}
	return s.sendResponse(conn, msg)
}

// SendEPStatusComplete validates command and sends an EPStatusComplete message.
func (s *Server) SendEPStatusComplete(conn net.Conn, session *Session, msg *EPStatusComplete) error {
	if msg.Command != CmdEPStatusComplete {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			logger.FieldSpec, specSectionEPStatusCmp, logger.FieldExpected, CmdEPStatusComplete, logger.FieldGot, msg.Command, logger.FieldOpID, msg.OpId)
		return fmt.Errorf(errFmtEPStatusCmpCommandMismatch, msg.Command, CmdEPStatusComplete)
	}
	return s.sendResponse(conn, msg)
}

// SendDLDataResultComplete validates command and sends a DLDataResultComplete message.
// Command validation prevents misuse of BaseMessage-only response types.
func (s *Server) SendDLDataResultComplete(conn net.Conn, session *Session, msg *DLDataResultComplete) error {
	if msg.Command != CmdDLDataResultComplete {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACICommandMismatch,
			logger.FieldSpec, specSectionDLDataResultCmp, logger.FieldExpected, CmdDLDataResultComplete, logger.FieldGot, msg.Command, logger.FieldOpID, msg.OpId)
		return fmt.Errorf(errFmtDLDataResultCmpCommandMismatch, msg.Command, CmdDLDataResultComplete)
	}
	return s.sendResponse(conn, msg)
}
