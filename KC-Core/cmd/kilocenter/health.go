package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/health"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcconstants "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/version"
)

// getVersionInfo returns release manifest or exits with fatal error.
func getVersionInfo() *version.Info {
	info, err := version.Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Failed to load release manifest: %v\n", err)
		os.Exit(1)
	}

	if !info.IsProduction() && os.Getenv("KILOCENTER_ENVIRONMENT") == pkgconfig.EnvironmentProduction {
		fmt.Fprintf(os.Stderr, "WARNING: Running non-production version (%s) in production environment\n", info.Version)
	}

	return info
}

// startHealthCheck serves the health endpoints on the given port.
func startHealthCheck(log logger.Logger, port int, status, readiness *health.Service) {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           newHealthMux(log, status, readiness),
		ReadHeaderTimeout: healthServerReadHeaderTimeout,
		IdleTimeout:       healthServerIdleTimeout,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf(LogHealthCheckServerError, err)
	}
}

// newHealthMux reports every row on /health and only the rows this process
// needs to serve its API on /health/ready.
func newHealthMux(log logger.Logger, status, readiness *health.Service) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc(health.RouteHealth, status.HTTPHandler())
	mux.HandleFunc(health.RouteHealthReady, readiness.HTTPHandler())

	mux.HandleFunc(health.RouteHealthPing, func(w http.ResponseWriter, r *http.Request) {
		writeHealthBody(log, w, r, HealthResponseHealthy)
	})

	mux.HandleFunc(health.RouteHealthLive, func(w http.ResponseWriter, r *http.Request) {
		writeHealthBody(log, w, r, HealthResponseAlive)
	})

	return mux
}

// writeHealthBody answers a probe with body; a failed write means the prober
// went away, so it is only logged.
func writeHealthBody(log logger.Logger, w http.ResponseWriter, r *http.Request, body string) {
	w.Header().Set(grpcconstants.HeaderContentType, grpcconstants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if _, err := io.WriteString(w, body); err != nil {
		log.WarnContext(r.Context(), LogFailedWriteHealthResponse, logger.FieldError, err)
	}
}
