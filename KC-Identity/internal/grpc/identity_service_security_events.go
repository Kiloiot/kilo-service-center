package grpc

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// emitSecurityEvent persists a security event when an event writer is
// configured. The request it describes proceeds either way, so a failure to
// record it is logged.
func (s *IdentityService) emitSecurityEvent(ctx context.Context, eventType, title, method, reason string) {
	if s.eventWriter == nil {
		return
	}
	tenantID := s.platformTenantID
	if tid, tErr := grpcerrors.GetTenantFromContext(ctx); tErr == nil {
		tenantID = tid
	}
	details, err := json.Marshal(map[string]interface{}{
		"method": method,
		"reason": reason,
	})
	if err != nil {
		s.log.WarnContext(ctx, LogSecurityEventWriteFailed, logger.FieldType, eventType, logger.FieldError, err)
		return
	}
	err = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(tenantID, 10),
		EventType:   eventType,
		Category:    models.EventCategorySecurity,
		Severity:    models.EventSeverityWarning,
		Title:       title,
		Description: reason,
		SourceType:  models.SourceTypeAPI,
		SourceName:  method,
		Details:     details,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		s.log.WarnContext(ctx, LogSecurityEventWriteFailed, logger.FieldType, eventType, logger.FieldError, err)
	}
}
