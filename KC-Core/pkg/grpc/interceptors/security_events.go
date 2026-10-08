package interceptors

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	audit "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// securityEvent is one refused request an interceptor audits.
type securityEvent struct {
	method    string
	eventType string
	title     string
	reason    string
	userID    string
}

// recordSecurityEvent persists a refusal in the request's tenant, or in the
// platform tenant before one is known. The write is detached from the request
// so a caller that hangs up on the refusal cannot suppress its record. The
// refusal stands whether or not it is recorded, so a failed write is logged.
func recordSecurityEvent(ctx context.Context, writer audit.EventWriter, platformTenantID int64, log logger.Logger, event securityEvent) {
	if writer == nil {
		return
	}
	tenantID := platformTenantID
	if tid, err := pkgcontext.GetTenantID(ctx); err == nil {
		tenantID = tid
	}
	details, err := json.Marshal(map[string]string{
		auditKeyMethod: event.method,
		auditKeyReason: event.reason,
	})
	if err != nil {
		logUnrecorded(ctx, log, event, err)
		return
	}
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), dbconfig.DefaultQueryTimeout)
	defer cancel()
	now := time.Now()
	err = writer.CreateEvent(writeCtx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(tenantID, 10),
		EventType:   event.eventType,
		Category:    models.EventCategorySecurity,
		Severity:    models.EventSeverityWarning,
		Title:       event.title,
		Description: event.reason,
		SourceType:  models.SourceTypeAPI,
		SourceName:  event.method,
		UserID:      event.userID,
		Details:     details,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		logUnrecorded(ctx, log, event, err)
	}
}

func logUnrecorded(ctx context.Context, log logger.Logger, event securityEvent, err error) {
	log.WarnContext(ctx, LogSecurityEventRecordFailed,
		logger.FieldMethod, event.method, logger.FieldEventType, event.eventType, logger.FieldError, err)
}

// principalUserID names the user a refusal is recorded against; a request
// refused before or without a user principal is recorded without one.
func principalUserID(ctx context.Context) string {
	userID, err := pkgcontext.GetUserID(ctx)
	if err != nil {
		return ""
	}
	return userID
}
