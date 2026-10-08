package scaci

import (
	"errors"
	"fmt"
	"net"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/vmihailenco/msgpack/v5"
)

// sendErrorWithCatalog answers opId with the catalog error per SCACI §3.14.1.
//
// POSIX code resolution: a non-zero POSIXCode in the catalog overrides
// defaultCode. contextDetail is logged only, never sent on the wire. No caller
// can act on a failed error frame, so the failure is logged with the
// application center, session, opId and token.
func (s *Server) sendErrorWithCatalog(conn net.Conn, session *Session, opId int64, defaultCode int, errorToken string, contextDetail ...string) {
	detail := ""
	if len(contextDetail) > 0 {
		detail = contextDetail[0]
	}
	posixCode, message := s.resolveOutboundError(session, opId, defaultCode, errorToken, detail)
	s.logOutboundError(session, opId, posixCode, errorToken, detail)

	sendErr := s.sendError(conn, opId, posixCode, message)
	if sendErr == nil {
		s.logger.DebugContext(s.sessionContext(session), LogSCACIErrorMessageSent,
			logger.FieldOpID, opId, logger.FieldCode, posixCode, logger.FieldMessage, message)
		return
	}
	fields := []interface{}{
		logger.FieldOpID, opId,
		logger.FieldErrorToken, errorToken,
		logger.FieldError, sendErr,
	}
	if session != nil {
		fields = append(fields, logger.FieldAcEui, mioty.FormatEUI64(session.AcEui), logger.FieldSessionID, session.ID)
	}
	s.logger.ErrorContext(s.sessionContext(session), LogSCACISendErrorFailed, fields...)
}

// resolveOutboundError returns the POSIX code and message sent for
// errorToken, persisting the error through the recorder when a session exists
// (SCACI §3.14).
func (s *Server) resolveOutboundError(session *Session, opId int64, defaultCode int, errorToken, detail string) (int, string) {
	if s.errorRecorder != nil && session != nil {
		ctx := s.sessionContext(session)
		posixCode, message, err := s.errorRecorder.RecordOutboundError(ctx, session, opId, CmdError, errorToken, defaultCode, detail)
		if err != nil {
			s.logger.WarnContext(ctx, LogSCACIRecordEventFailed, logger.FieldError, err)
		}
		return posixCode, message
	}
	def := GetErrorDefinition(errorToken)
	if def.POSIXCode != 0 {
		return def.POSIXCode, def.Message
	}
	return defaultCode, def.Message
}

// logOutboundError logs the refusal with its spec traceability; a connect
// refusal may have no session yet.
func (s *Server) logOutboundError(session *Session, opId int64, posixCode int, errorToken, detail string) {
	def := GetErrorDefinition(errorToken)
	fields := []interface{}{
		logger.FieldToken, def.Token,
		logger.FieldSpec, def.SpecSection,
		logger.FieldSeverity, def.Severity,
		logger.FieldOpID, opId,
		logger.FieldPosixCode, posixCode,
	}
	if session != nil {
		fields = append(fields, logger.FieldAcEui, mioty.FormatEUI64(session.AcEui))
		if session.TenantID > 0 {
			fields = append(fields, logger.FieldTenantID, session.TenantID)
		}
	}
	if detail != "" {
		fields = append(fields, logger.FieldContext, detail)
	}
	s.logger.WarnContext(s.sessionContext(session), LogSCACIErrorReceived, fields...)
}

// sendError writes an error frame; the error token stays internal and is not
// sent (SCACI §3.14.1).
func (s *Server) sendError(conn net.Conn, opId int64, code int, message string) error {
	payload, err := msgpack.Marshal(&Error{
		BaseMessage: BaseMessage{Command: CmdError, OpId: opId},
		Code:        code,
		Message:     message,
	})
	if err != nil {
		return fmt.Errorf(errFmtMarshalResponse, err)
	}
	if err := s.writeFrame(conn, payload); err != nil {
		return fmt.Errorf(errFmtSendFrame, err)
	}
	return nil
}

// refuseConnection answers opId with the catalog error and closes conn: the
// application center is refused before its session is established (SCACI §3.3).
func (s *Server) refuseConnection(conn net.Conn, session *Session, opId int64, errorToken string) {
	s.sendErrorWithCatalog(conn, session, opId, POSIX_EINVAL, errorToken)
	s.closeConnection(session, conn)
}

// closeConnection closes conn; a connection a failed write or a shutdown
// already closed is not a failure.
func (s *Server) closeConnection(session *Session, conn net.Conn) {
	if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		s.logger.WarnContext(s.sessionContext(session), LogSCACICloseConnectionFailed,
			logger.FieldRemote, conn.RemoteAddr().String(), logger.FieldError, err)
	}
}
