package scaci

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// markOperationFailed records opId as failed with its error details. The
// operation already failed on the wire, so a record that cannot be written is
// logged rather than returned.
func (s *Server) markOperationFailed(session *Session, opId int64, details map[string]interface{}) {
	if session.ID <= 0 || s.operationRepo == nil {
		return
	}
	ctx, cancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
	defer cancel()
	if err := s.operationRepo.UpdateOperationState(ctx, session.ID, opId, models.OperationStateFailed, details); err != nil {
		s.logger.WarnContext(ctx, LogSCACIUpdateOperationStateFailed,
			logger.FieldSessionID, session.ID, logger.FieldOpID, opId,
			logger.FieldState, models.OperationStateFailed, logger.FieldError, err)
	}
}
