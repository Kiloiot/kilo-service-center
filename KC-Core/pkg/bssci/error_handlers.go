package bssci

import (
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// handleError handles error messages from Base Station
func (s *Server) handleError(session *Session, msg *Message, data map[string]interface{}) error {
	code := getNumericFieldInt(data, "code", -1)
	message := getStringField(data, "message", fallbackErrorMessage)

	s.logger.ErrorContext(s.safeCtx(), LogBSSCIBaseStationReportedError,
		logger.FieldCode, code,
		logger.FieldMessage, logger.UntrustedValue(message, maxLoggedErrorMessageBytes),
		logger.FieldBaseStation, session.BaseStationEUI)

	// Handshake error routing (BSSCI §5.17): an error with opId 0 while
	// waiting for conCmp means the base station rejected the offered version
	// or connect response.
	if msg.OpId == 0 && session.ConnectState == ConnectStateAwaitingConnectComplete {
		return s.handleConnectRefusal(session, code, message)
	}

	// BSSCI §5.17: an inbound error is answered ONLY with errorAck - never with
	// an operation-specific *Cmp, and the operation type is never guessed. The
	// error and errorAck replace the normal response/completion sequence.
	if sendErr := s.sendErrorAck(session, msg.OpId); sendErr != nil {
		return sendErr
	}
	s.finalizeErroredOperation(session, msg.OpId, code, message)
	return nil
}

// handleConnectRefusal acknowledges the refused connect response, which
// completes the failed connect operation, retires a refused resume and
// closes the connection.
func (s *Server) handleConnectRefusal(session *Session, code int, message string) error {
	ackErr := s.sendErrorAck(session, 0)
	session.ConnectState = ConnectStateTerminal
	s.retireRefusedResume(s.sessionContext(session), session, code, message)
	return errors.Join(fmt.Errorf(errFmtStationRejectedConnect, code, logger.UntrustedValue(message, maxLoggedErrorMessageBytes)), ackErr)
}

// sendErrorAck answers an inbound error frame (BSSCI §5.17).
func (s *Server) sendErrorAck(session *Session, opID int64) error {
	ack := mioty.ErrorAck{BaseMessage: mioty.BaseMessage{CommandType: mioty.CmdErrorAck, OpId: opID}}
	if err := s.sendMessage(session, ack); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorAck, logger.FieldError, err)
		return err
	}
	return nil
}

// finalizeErroredOperation applies typed compensation only for an operation
// this service center is actually tracking (an errored operation is finalized
// without updating domain state). An unmatched error is acknowledged and
// audited but touches no unrelated pending state.
func (s *Server) finalizeErroredOperation(session *Session, opID int64, code int, message string) {
	if s.statusSvc == nil {
		return
	}
	pendingOp, lookupErr := s.statusSvc.GetPendingOperation(session, opID)
	if lookupErr != nil || pendingOp == nil {
		return
	}
	s.logger.InfoContext(s.safeCtx(), LogBSSCIErrorFinalizedOperation,
		logger.FieldOpID, opID,
		logger.FieldOperationType, pendingOp.OperationType)
	if compensate, ok := s.commands.onError(pendingOp.OperationType); ok {
		compensate(s, s.sessionContext(session), session, opID, code, message)
	}
	if err := s.pendingOps.remove(s.sessionContext(session), session, opID); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperationFromDatabase,
			logger.FieldError, err,
			logger.FieldOpID, opID)
	}
}

// handleErrorAck handles the errorAck message (BSSCI §3.17.2)
// This acknowledges error reception from the base station and allows operation recovery
func (s *Server) handleErrorAck(session *Session, msg *Message, _ map[string]interface{}) error {
	s.logger.InfoContext(s.safeCtx(), LogBSSCIReceivedErrorAckFromBaseStation,
		logger.FieldOpID, msg.OpId,
		logger.FieldBaseStationEUI, session.BaseStationEUI)

	// Handshake errorAck routing (BSSCI §5.17): the acknowledgement completes
	// the failed connect operation; the connection closes
	if msg.OpId == 0 && session.ConnectState == ConnectStateAwaitingConnectErrorAck {
		session.ConnectState = ConnectStateTerminal
		return errConnectOperationFailedAndWasAcknowledged
	}

	// An errorAck for the connect operation outside the awaiting state is a
	// protocol-ordering violation
	if msg.OpId == 0 && session.ConnectState != ConnectStateComplete {
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errInvalidHandshakeState)
	}

	// An errorAck is only meaningful when this service center actually sent an
	// error frame for that operation on this connection (BSSCI rev1 §5.17 /
	// classic §3.17). Consuming the recorded expectation prevents a spurious
	// or forged errorAck from finalizing an unrelated in-flight operation.
	disposition, awaited := session.consumePendingErrorAck(msg.OpId)
	if !awaited {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIUnsolicitedErrorAck,
			logger.FieldOpID, msg.OpId,
			logger.FieldBaseStationEUI, session.BaseStationEUI)
		return nil
	}

	// Only an error that replaced a pending SC operation's normal sequence may
	// finalize that operation; ack-only errors touch no pending state.
	if disposition == errorAckFinalizePendingOperation && msg.OpId < 0 {
		if err := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperationAfterErrorAck,
				logger.FieldError, err,
				logger.FieldOpID, msg.OpId)
			// Note: No manual fallback - trust StatusService single-writer pattern
		}
	}

	// An errorAck that completes an error sent during the connect handshake
	// finishes the failed exchange: the state machine goes Terminal and the
	// connection closes (BSSCI rev1 §5.17 / classic §3.17)
	if session.ConnectState == ConnectStateAwaitingConnectErrorAck {
		session.ConnectState = ConnectStateTerminal
		return errConnectStageErrorAcknowledged
	}

	return nil
}
