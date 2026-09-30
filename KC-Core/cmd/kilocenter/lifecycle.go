package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/cmd/kilocenter/builders"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// serviceEventWriter stores the service-stopped event.
type serviceEventWriter interface {
	CreateEvent(ctx context.Context, event *models.SystemEvent) error
}

// shutdownObservability flushes a tracing or metrics provider on exit and
// logs a provider that fails to shut down under failureLog.
func shutdownObservability(log logger.Logger, shutdown func(context.Context) error, failureLog string) {
	ctx := context.Background() // context-root: shutdown
	if err := shutdown(ctx); err != nil {
		log.ErrorContext(ctx, failureLog, logger.FieldError, err)
	}
}

// emitServiceStopped records the service-stopped event; shutdown proceeds
// either way, so a failure is logged.
func emitServiceStopped(ctx context.Context, log logger.Logger, events serviceEventWriter, tenantID int64, version string) {
	hostname := builders.ServiceHostname(ctx, log)
	if err := events.CreateEvent(ctx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(tenantID, 10),
		EventType:   models.EventTypeServiceStopped,
		Category:    models.EventCategorySystem,
		Severity:    models.EventSeverityInfo,
		Title:       fmt.Sprintf(models.EventTitleServiceStoppedFmt, builders.CoreServiceDisplayName, hostname),
		Description: fmt.Sprintf(models.EventDescriptionServiceStoppedFmt, builders.CoreServiceDisplayName, version),
		SourceType:  models.SourceTypeSystem,
		SourceName:  builders.CoreServiceSourceName,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}); err != nil {
		log.WarnContext(ctx, LogFailedEmitServiceStoppedEvent, logger.FieldError, err)
		return
	}
	log.InfoContext(ctx, LogServiceStoppedEventEmitted)
}
