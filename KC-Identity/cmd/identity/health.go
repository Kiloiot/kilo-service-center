package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// Health endpoint identity and status values.
const (
	identityServiceName = "kc-identity"
	healthStatusHealthy = "healthy"
	healthKeyHealthy    = "healthy"
	healthKeyError      = "error"
)

// healthHandler provides health check endpoints for KC-Identity.
type healthHandler struct {
	db  *sql.DB
	log logger.Logger
}

// writeBody sends a response body; the status is already written, so a
// failed write can only be logged.
func (h *healthHandler) writeBody(w http.ResponseWriter, r *http.Request, body string) {
	if _, err := w.Write([]byte(body)); err != nil {
		h.log.WarnContext(r.Context(), LogHealthResponseWriteFailed, logger.FieldPath, r.URL.Path, logger.FieldError, err)
	}
}

// writeJSON encodes a response document; like writeBody, a failure is logged.
func (h *healthHandler) writeJSON(w http.ResponseWriter, r *http.Request, doc map[string]interface{}) {
	if err := json.NewEncoder(w).Encode(doc); err != nil {
		h.log.WarnContext(r.Context(), LogHealthResponseWriteFailed, logger.FieldPath, r.URL.Path, logger.FieldError, err)
	}
}

// Health endpoint status values, canned JSON bodies and routes.
const (
	statusUnhealthy = "unhealthy"
	bodyHealthy     = `{"status":"healthy"}`
	bodyUnhealthy   = `{"status":"unhealthy"}`
	bodyAlive       = `{"status":"alive"}`

	routeHealth      = "/health"
	routeHealthReady = "/health/ready"
	routeHealthPing  = "/health/ping"
	routeHealthLive  = "/health/live"
)

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
			resp["status"] = statusUnhealthy
			resp["database"] = map[string]interface{}{healthKeyHealthy: false, healthKeyError: err.Error()}
			w.Header().Set(grpcconst.HeaderContentType, grpcconst.ContentTypeJSON)
			w.WriteHeader(http.StatusServiceUnavailable)
			h.writeJSON(w, r, resp)
			return
		}
		resp["database"] = map[string]interface{}{healthKeyHealthy: true}
	}

	w.Header().Set(grpcconst.HeaderContentType, grpcconst.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	h.writeJSON(w, r, resp)
}

// serveReady handles the /health/ready endpoint.
// Returns 503 if the database is unreachable.
func (h *healthHandler) serveReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(grpcconst.HeaderContentType, grpcconst.ContentTypeJSON)

	if h.db != nil {
		if err := h.db.PingContext(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			h.writeBody(w, r, bodyUnhealthy)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	h.writeBody(w, r, bodyHealthy)
}

// servePing handles the /health/ping endpoint (unconditional 200).
func (h *healthHandler) servePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(grpcconst.HeaderContentType, grpcconst.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	h.writeBody(w, r, bodyHealthy)
}

// serveLive handles the /health/live endpoint (unconditional 200).
func (h *healthHandler) serveLive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(grpcconst.HeaderContentType, grpcconst.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	h.writeBody(w, r, bodyAlive)
}

// startHealthServer starts the HTTP health endpoint with all standard paths.
func startHealthServer(ctx context.Context, port int, db *sql.DB) {
	l := logger.Get()
	h := &healthHandler{db: db, log: l}

	mux := http.NewServeMux()
	mux.Handle(routeHealth, h)
	mux.HandleFunc(routeHealthReady, h.serveReady)

	mux.HandleFunc(routeHealthPing, h.servePing)
	mux.HandleFunc(routeHealthLive, h.serveLive)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: healthServerReadHeaderTimeout,
		IdleTimeout:       healthServerIdleTimeout,
	}

	//nolint:gosec // G118: shutdown deliberately uses a fresh context - the parent is already done
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), healthServerShutdownTimeout) // context-root: shutdown
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			l.Warn(LogHealthServerShutdownError, logger.FieldError, err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		l.Error(LogHealthCheckServerError, logger.FieldError, err)
	}
}
