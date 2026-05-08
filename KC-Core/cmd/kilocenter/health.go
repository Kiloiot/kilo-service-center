package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/kilocenter/KC-Core/internal/health"
	grpcconstants "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/pkg/version"
)

// getVersionInfo returns release manifest or exits with fatal error.
func getVersionInfo() *version.Info {
	info, err := version.Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Failed to load release manifest: %v\n", err)
		os.Exit(1)
	}

	if !info.IsProduction() && os.Getenv("KILOCENTER_ENVIRONMENT") == "production" {
		fmt.Fprintf(os.Stderr, "WARNING: Running non-production version (%s) in production environment\n", info.Version)
	}

	return info
}

// startHealthCheck starts the HTTP health check endpoint on the given port.
func startHealthCheck(port int, healthService *health.Service) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthService.HTTPHandler())

	mux.HandleFunc("/health/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(grpcconstants.HeaderContentType, grpcconstants.ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(HealthResponseHealthy))
	})

	mux.HandleFunc("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(grpcconstants.HeaderContentType, grpcconstants.ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(HealthResponseAlive))
	})

	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(grpcconstants.HeaderContentType, grpcconstants.ContentTypeJSON)
		response := healthService.CheckHealth(r.Context())
		if response.Status == health.StatusHealthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_, _ = fmt.Fprintf(w, HealthResponseFormat, response.Status)
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf(LogHealthCheckServerError, err)
	}
}
