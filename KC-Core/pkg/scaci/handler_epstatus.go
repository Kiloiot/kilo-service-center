package scaci

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// handleEPStatusResponse records the acknowledgement of an SC-initiated EP
// status operation and closes the handshake with epStatCmp (SCACI §3.13.2-3).
func (s *Server) handleEPStatusResponse(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	ctx := s.sessionContext(session)
	s.logger.DebugContext(ctx, LogSCACIEPStatusResponseReceived, logger.FieldOpID, opId)

	session.UpdateLastSeen(s.clock.Now())

	if s.operationRepo != nil && session.ID > 0 {
		opCtx, opCancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
		defer opCancel()

		responseData := map[string]interface{}{"status": opStatusAck}
		if err := s.operationRepo.UpdateOperationState(opCtx, session.ID, opId,
			models.OperationStateAcknowledged, responseData); err != nil {
			s.logger.WarnContext(ctx, LogSCACIOperationStateUpdateFailed,
				logger.FieldOpID, opId, logger.FieldState, models.OperationStateAcknowledged, logger.FieldError, err)
		}
	}

	session.WriteMu.Lock()
	cmpMsg := EPStatusComplete{BaseMessage: BaseMessage{Command: CmdEPStatusComplete, OpId: opId}}
	err := s.SendEPStatusComplete(conn, session, &cmpMsg)
	session.WriteMu.Unlock()

	if err != nil {
		s.logger.ErrorContext(ctx, LogSCACISendEPStatusCompleteFailed, logger.FieldOpID, opId, logger.FieldError, err)
		if session.ID > 0 && s.operationRepo != nil {
			failCtx, failCancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
			defer failCancel()

			if updateErr := s.operationRepo.UpdateOperationState(failCtx, session.ID, opId,
				models.OperationStateFailed, map[string]interface{}{
					MetadataKeyErrorToken:  errFailedRecordOperation,
					MetadataKeyErrorDetail: fmt.Sprintf(errDetailFmtSendEPStatusCmp, err),
				}); updateErr != nil {
				s.logger.WarnContext(ctx, LogSCACIOperationStateUpdateFailed,
					logger.FieldOpID, opId, logger.FieldState, models.OperationStateFailed, logger.FieldError, updateErr)
			}
		}
		return err
	}

	s.logger.DebugContext(ctx, LogSCACIEPStatusHandshakeComplete, logger.FieldOpID, opId)

	if s.operationRepo != nil && session.ID > 0 {
		cmpCtx, cmpCancel := context.WithTimeout(ctx, dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		responseData := map[string]interface{}{
			"status":               opStatusCompleted,
			metadataKeyCompletedAt: s.clock.Now().UTC().Format(time.RFC3339),
		}
		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId,
			models.OperationStateCompleted, responseData); err != nil {
			s.logger.WarnContext(ctx, LogSCACIOperationStateUpdateFailed,
				logger.FieldOpID, opId, logger.FieldState, models.OperationStateCompleted, logger.FieldError, err)
		}
	}

	return nil
}

// rejectACIssuedEPStatusComplete answers an application-center epStatCmp: the
// Service Center initiates and completes the EP status operation (§3.13.3).
func (s *Server) rejectACIssuedEPStatusComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}
	return s.rejectACIssuedComplete(conn, session, opId, LogSCACIUnexpectedEPStatusCmp, errProtocolViolationEPStatCmp, errDetailACSentEPStatusCmp)
}
