package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/resilience"
	"github.com/Kiloiot/kilo-service-center/pkg/observability"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

// initObservability starts tracing and metrics and returns the shutdown hook
// for whichever of them came up.
func initObservability(cfg *config.Config, l logger.Logger) func() {
	var shutdowns []func(context.Context) error
	tracingShutdown, err := observability.InitTracing(context.Background(), observability.TracingConfig{ // context-root: process
		Enabled:    cfg.Monitoring.TracingEnabled,
		Endpoint:   cfg.Monitoring.TracingEndpoint,
		SampleRate: cfg.Monitoring.TracingSampleRate,
	}, gatewayServiceName)
	if err != nil {
		l.Error(LogGatewayTracingInitFailed, logger.FieldError, err)
	} else {
		shutdowns = append(shutdowns, tracingShutdown)
	}

	metricsShutdown, err := observability.InitMetrics(context.Background(), observability.MetricsConfig{ // context-root: process
		Enabled: cfg.Monitoring.MetricsEnabled,
		Port:    cfg.Monitoring.MetricsPort,
		Path:    cfg.Monitoring.MetricsPath,
	}, gatewayServiceName)
	if err != nil {
		l.Error(LogGatewayMetricsInitFailed, logger.FieldError, err)
	} else {
		shutdowns = append(shutdowns, metricsShutdown)
	}
	return func() {
		for _, shutdown := range shutdowns {
			if err := shutdown(context.Background()); err != nil { // context-root: shutdown
				l.Warn(LogGatewayObservabilityShutdownFailed, logger.FieldError, err)
			}
		}
	}
}

// shutdownServers stops accepting connections, gives open calls the timeout
// to finish, then closes the proxy's remaining transports. The proxy is
// served through ServeHTTP, whose transports cannot drain, so it is stopped
// rather than gracefully stopped.
func shutdownServers(l logger.Logger, httpServer *http.Server, proxyServer *grpc.Server, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout) // context-root: shutdown
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		l.Error(LogGatewayShutdownError, logger.FieldError, err)
	}
	proxyServer.Stop()
}

// exportBreakerState publishes both breakers' state as a Prometheus gauge
// until ctx ends.
func exportBreakerState(ctx context.Context, gauge *prometheus.GaugeVec, coreBreaker, identityBreaker *resilience.UpstreamBreaker) {
	ticker := time.NewTicker(breakerGaugeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			gauge.WithLabelValues(upstreamCore).Set(float64(coreBreaker.State()))
			gauge.WithLabelValues(upstreamIdentity).Set(float64(identityBreaker.State()))
		}
	}
}

// lifecycleEventWriter persists the gateway's lifecycle events.
type lifecycleEventWriter interface {
	CreateEvent(ctx context.Context, event *models.SystemEvent) error
}

// resolveHostname names this host for the lifecycle events; an unresolvable
// hostname leaves the events without one.
func resolveHostname(l logger.Logger) string {
	hostname, err := os.Hostname()
	if err != nil {
		l.Warn(LogGatewayHostnameUnavailable, logger.FieldError, err)
	}
	return hostname
}

// emitStartedEvent records the KC-Gateway started lifecycle event. The
// gateway runs without it, so a failure is logged.
func emitStartedEvent(ctx context.Context, l logger.Logger, events lifecycleEventWriter, cfg *config.Config, hostname string) {
	details, err := json.Marshal(map[string]interface{}{
		"service": gatewayServiceName,
		"host":    hostname,
		"ports":   map[string]int{"grpc-web": cfg.GRPC.Port, "health": cfg.Gateway.HealthPort},
	})
	if err != nil {
		l.WarnContext(ctx, LogGatewayLifecycleEventFailed, logger.FieldType, models.EventTypeServiceStarted, logger.FieldError, err)
		return
	}
	err = events.CreateEvent(ctx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(cfg.General.TenantID, 10),
		EventType:   models.EventTypeServiceStarted,
		Category:    models.EventCategorySystem,
		Severity:    models.EventSeverityInfo,
		Title:       fmt.Sprintf(models.EventTitleServiceStartedFmt, gatewayDisplayName, hostname),
		Description: fmt.Sprintf(models.EventDescriptionServiceStartedFmt, gatewayDisplayName, gatewayVersion, fmt.Sprintf(listenInfoFmt, cfg.GRPC.Port, cfg.Gateway.HealthPort)),
		SourceType:  models.SourceTypeSystem,
		SourceName:  gatewayServiceName,
		Details:     details,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		l.WarnContext(ctx, LogGatewayLifecycleEventFailed, logger.FieldType, models.EventTypeServiceStarted, logger.FieldError, err)
	}
}

// emitStoppedEvent records the KC-Gateway stopped lifecycle event within the
// stop-event timeout; the gateway stops regardless, so a failure is logged.
func emitStoppedEvent(l logger.Logger, events lifecycleEventWriter, cfg *config.Config, hostname string) {
	stopCtx, stopCancel := context.WithTimeout(context.Background(), stopEventTimeout) // context-root: shutdown
	defer stopCancel()
	err := events.CreateEvent(stopCtx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(cfg.General.TenantID, 10),
		EventType:   models.EventTypeServiceStopped,
		Category:    models.EventCategorySystem,
		Severity:    models.EventSeverityInfo,
		Title:       fmt.Sprintf(models.EventTitleServiceStoppedFmt, gatewayDisplayName, hostname),
		Description: fmt.Sprintf(models.EventDescriptionServiceStoppedFmt, gatewayDisplayName, gatewayVersion),
		SourceType:  models.SourceTypeSystem,
		SourceName:  gatewayServiceName,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		l.WarnContext(stopCtx, LogGatewayLifecycleEventFailed, logger.FieldType, models.EventTypeServiceStopped, logger.FieldError, err)
	}
}
