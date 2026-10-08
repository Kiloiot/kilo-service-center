package scaci

import (
	"net"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// handleInboundError processes error messages received from the Application Center (SCACI §3.14).
// AC sends error when it cannot process an SC-initiated operation.
// SC must acknowledge with errorAck to complete the error handshake.
func (s *Server) handleInboundError(conn net.Conn, session *Session, opId int64, payload []byte) error {
	ctx := s.sessionContext(session)

	// Decode the error message
	var errMsg Error
	if err := decodePayload(payload, &errMsg); err != nil {
		s.logger.ErrorContext(ctx, LogSCACIInvalidPayload,
			logger.FieldCommand, CmdError,
			logger.FieldOpID, opId,
			logger.FieldError, err)
		return nil // Cannot respond to malformed error
	}

	// Validate inbound error message per SCACI §3.14.1
	// AC must send non-zero code and non-empty message
	if validationErr := ValidateError(&errMsg); validationErr != "" {
		s.logger.ErrorContext(ctx, LogSCACIErrorMsgValidationFailed,
			logger.FieldCommand, CmdError,
			logger.FieldOpID, opId,
			logger.FieldErrorToken, validationErr,
			logger.FieldReceivedCode, errMsg.Code,
			logger.FieldReceivedMessage, errMsg.Message)
		// Respond with error per §3.14 - invalid error messages are protocol violations
		s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, validationErr)
		return nil
	}

	s.logger.WarnContext(ctx, LogSCACIReceivedInboundError,
		logger.FieldOpID, opId,
		logger.FieldAcEui, mioty.FormatEUI64(session.AcEui),
		logger.FieldPosixCode, errMsg.Code,
		logger.FieldMessage, errMsg.Message)

	session.UpdateLastSeen(s.clock.Now())

	// Record the inbound error via ErrorRecorder if available
	if s.errorRecorder != nil {
		if err := s.errorRecorder.RecordInboundError(ctx, session, opId, errMsg.Code, errMsg.Message); err != nil {
			s.logger.WarnContext(ctx, LogSCACIPersistInboundErrorFailed,
				logger.FieldOpID, opId,
				logger.FieldError, err)
		}
	}

	// Send errorAck to complete the error handshake per §3.14
	return s.sendErrorAck(conn, session, opId)
}

// handleErrorAck processes errorAck messages received from the Application Center (SCACI §3.14).
// AC sends errorAck after receiving an error from SC, completing the error handshake.
func (s *Server) handleErrorAck(_ net.Conn, session *Session, opId int64) error {
	ctx := s.sessionContext(session)

	s.logger.DebugContext(ctx, LogSCACIReceivedErrorAck,
		logger.FieldOpID, opId,
		logger.FieldAcEui, mioty.FormatEUI64(session.AcEui))

	session.UpdateLastSeen(s.clock.Now())

	// Complete the error handshake via ErrorRecorder if available
	if s.errorRecorder != nil {
		if err := s.errorRecorder.CompleteErrorHandshake(ctx, session, opId); err != nil {
			s.logger.WarnContext(ctx, LogSCACICompleteErrorHandshakeFailed,
				logger.FieldOpID, opId,
				logger.FieldError, err)
		}
	}

	return nil // No response per spec - errorAck completes the sequence
}
