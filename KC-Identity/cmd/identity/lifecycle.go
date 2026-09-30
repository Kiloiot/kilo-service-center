package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/observability"
)

// initObservability starts tracing and metrics and returns the shutdown hook
// for whichever of them came up.
func initObservability(cfg *pkgconfig.Config, log logger.Logger) func() {
	var shutdowns []func(context.Context) error
	tracingShutdown, err := observability.InitTracing(context.Background(), observability.TracingConfig{ // context-root: process
		Enabled:    cfg.Monitoring.TracingEnabled,
		Endpoint:   cfg.Monitoring.TracingEndpoint,
		SampleRate: cfg.Monitoring.TracingSampleRate,
	}, identityServiceName)
	if err != nil {
		log.Error(LogFailedInitTracing, logger.FieldError, err)
	} else {
		shutdowns = append(shutdowns, tracingShutdown)
	}

	metricsShutdown, err := observability.InitMetrics(context.Background(), observability.MetricsConfig{ // context-root: process
		Enabled: cfg.Monitoring.MetricsEnabled,
		Port:    cfg.Monitoring.MetricsPort,
		Path:    cfg.Monitoring.MetricsPath,
	}, identityServiceName)
	if err != nil {
		log.Error(LogFailedInitMetrics, logger.FieldError, err)
	} else {
		shutdowns = append(shutdowns, metricsShutdown)
	}
	return func() {
		for _, shutdown := range shutdowns {
			if err := shutdown(context.Background()); err != nil { // context-root: shutdown
				log.Warn(LogObservabilityShutdownFailed, logger.FieldError, err)
			}
		}
	}
}

// lifecycleEventWriter persists the service's lifecycle events.
type lifecycleEventWriter interface {
	CreateEvent(ctx context.Context, event *models.SystemEvent) error
}

// lifecycleEvents records the KC-Identity started and stopped events. The
// service runs regardless of them, so a failed write is logged.
type lifecycleEvents struct {
	writer   lifecycleEventWriter
	log      logger.Logger
	tenantID int64
	version  string
	hostname string
}

// newLifecycleEvents names this host for the events; an unresolvable hostname
// leaves the events without one.
func newLifecycleEvents(writer lifecycleEventWriter, log logger.Logger, tenantID int64, version string) *lifecycleEvents {
	hostname, err := os.Hostname()
	if err != nil {
		log.Warn(LogHostnameUnavailable, logger.FieldError, err)
	}
	return &lifecycleEvents{writer: writer, log: log, tenantID: tenantID, version: version, hostname: hostname}
}

// started records the service-started event for the gRPC port.
func (e *lifecycleEvents) started(ctx context.Context, grpcPort int) {
	listenInfo := fmt.Sprintf(listenInfoFmt, grpcPort)
	e.record(ctx, &models.SystemEvent{
		EventType:   models.EventTypeServiceStarted,
		Title:       fmt.Sprintf(models.EventTitleServiceStartedFmt, identityDisplayName, e.hostname),
		Description: fmt.Sprintf(models.EventDescriptionServiceStartedFmt, identityDisplayName, e.version, listenInfo),
	}, LogServiceStartedEvent)
}

// stopped records the service-stopped event within the shutdown event timeout.
func (e *lifecycleEvents) stopped() {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownEventTimeout) // context-root: shutdown
	defer cancel()
	e.record(ctx, &models.SystemEvent{
		EventType:   models.EventTypeServiceStopped,
		Title:       fmt.Sprintf(models.EventTitleServiceStoppedFmt, identityDisplayName, e.hostname),
		Description: fmt.Sprintf(models.EventDescriptionServiceStoppedFmt, identityDisplayName, e.version),
	}, LogServiceStoppedEvent)
}

func (e *lifecycleEvents) record(ctx context.Context, event *models.SystemEvent, recorded string) {
	event.TenantID = strconv.FormatInt(e.tenantID, 10)
	event.Category = models.EventCategorySystem
	event.Severity = models.EventSeverityInfo
	event.SourceType = models.SourceTypeSystem
	event.SourceName = identityServiceName
	event.CreatedAt = time.Now()
	event.UpdatedAt = event.CreatedAt
	if err := e.writer.CreateEvent(ctx, event); err != nil {
		e.log.WarnContext(ctx, LogLifecycleEventFailed, logger.FieldType, event.EventType, logger.FieldError, err)
		return
	}
	e.log.InfoContext(ctx, recorded)
}
