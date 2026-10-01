package bssci

import (
	"encoding/json"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// recordVMEvent stores event with details as its payload; failureLog names
// the event when encoding or storing it fails.
func (s *Server) recordVMEvent(session *Session, details map[string]interface{}, failureLog string, event *models.SystemEvent) {
	ctx := s.sessionContext(session)
	encoded, err := json.Marshal(details)
	if err != nil {
		s.logger.ErrorContext(ctx, failureLog, logger.FieldError, err)
		return
	}
	event.Details = encoded
	if err := s.eventStore.CreateEvent(ctx, event); err != nil {
		s.logger.ErrorContext(ctx, failureLog, logger.FieldError, err)
	}
}
