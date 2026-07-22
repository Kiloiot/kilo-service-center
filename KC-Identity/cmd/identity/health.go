package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// Health endpoint identity and status values.
const (
	identityServiceName = "kc-identity"
	healthStatusHealthy = "healthy"
	healthKeyHealthy    = "healthy"
)

// healthHandler provides health check endpoints for KC-Identity.
type healthHandler struct {
	db *sql.DB
}

// ServeHTTP handles the /health endpoint.
func (h *healthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"status":    healthStatusHealthy,
		"service":   identityServiceName,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	// Check database connectivity
	if h.db != nil {
		if err := h.db.PingContext(r.Context()); err != nil {
			resp["status"] = "unhealthy"
			resp["database"] = map[string]interface{}{healthKeyHealthy: false, "error": err.Error()}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		resp["database"] = map[string]interface{}{healthKeyHealthy: true}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// serveReady handles the /health/ready endpoint.
// Returns 503 if the database is unreachable.
func (h *healthHandler) serveReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.db != nil {
		if err := h.db.PingContext(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unhealthy"}`))
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"healthy"}`))
}

// startHealthServer starts the HTTP health endpoint with all standard paths.
func startHealthServer(ctx context.Context, port int, db *sql.DB) {
	l := logger.Get()
	h := &healthHandler{db: db}

	mux := http.NewServeMux()
	mux.Handle("/health", h)
	mux.HandleFunc("/health/ready", h.serveReady)

	mux.HandleFunc("/health/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	mux.HandleFunc("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"alive"}`))
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	//nolint:gosec // G118: shutdown deliberately uses a fresh context - the parent is already done
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		l.Error("Health check server error", "error", err)
	}
}
