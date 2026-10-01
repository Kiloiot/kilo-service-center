package bssci

import "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

// pendingOperationOrWarn returns the operation opId has in flight on session.
// A completion without one still finishes, so a failed lookup is logged and
// nil is returned for the caller's metadata-less path.
func (s *Server) pendingOperationOrWarn(session *Session, opId int64) *PendingOperation {
	pendingOp, err := s.statusSvc.GetPendingOperation(session, opId)
	if err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToGetPendingOperation,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldOpID, opId,
			logger.FieldError, err)
		return nil
	}
	return pendingOp
}
