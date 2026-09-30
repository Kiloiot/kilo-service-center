// Package health provides HTTP health endpoints for KC-Gateway.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/resilience"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// statusHealthy is the health endpoint status value for a passing check.
const statusHealthy = "healthy"

// Health endpoint status values and canned JSON bodies.
const (
	statusDegraded    = "degraded"
	stateShutdown     = "SHUTDOWN"
	bodyStatusPrefix  = `{"status":"`
	bodyStatusSuffix  = `"}`
	bodyUnhealthy     = `{"status":"unhealthy"}`
	bodyAlive         = `{"status":"alive"}`
	serviceLabelValue = "kc-gateway"
)

// Health endpoint routes.
const (
	routeHealth      = "/health"
	routeHealthReady = "/health/ready"
	routeHealthLive  = "/health/live"
	routeHealthPing  = "/health/ping"
)

// readHeaderTimeout bounds how long the health server waits for request headers.
const readHeaderTimeout = 5 * time.Second

// shutdownTimeout bounds the health server's graceful shutdown.
const shutdownTimeout = 5 * time.Second

// Health server log messages.
const (
	logHealthResponseWriteFailed = "Failed to write health response"
	logHealthServerShutdownError = "Health server shutdown error"
)

// connectionState reports an upstream connection's connectivity state; it is
// the only capability the health endpoints use from a client connection.
type connectionState interface {
	GetState() connectivity.State
}

// breakerState reports a circuit breaker's current state, numerically for the
// readiness decision and by name for the health payload.
type breakerState interface {
	State() resilience.BreakerState
	StateName() string
}

// handler provides gateway health check endpoints.
type handler struct {
	core            connectionState
	identity        connectionState
	coreBreaker     breakerState
	identityBreaker breakerState
	log             logger.Logger
}

// newHandler creates a health handler that checks both upstream connections and breaker state.
func newHandler(core, identity connectionState, coreBreaker, identityBreaker breakerState, log logger.Logger) *handler {
	return &handler{
		core:            core,
		identity:        identity,
		coreBreaker:     coreBreaker,
		identityBreaker: identityBreaker,
		log:             log,
	}
}

// writeBody sends a response body; the status is already written, so a
// failed write can only be logged.
func (h *handler) writeBody(w http.ResponseWriter, r *http.Request, body string) {
	if _, err := w.Write([]byte(body)); err != nil {
		h.log.WarnContext(r.Context(), logHealthResponseWriteFailed, logger.FieldPath, r.URL.Path, logger.FieldError, err)
	}
}

// checkConn returns the connectivity state and whether it is considered healthy.
func checkConn(conn connectionState) (string, bool) {
	if conn == nil {
		return stateShutdown, false
	}
	state := conn.GetState()
	healthy := state == connectivity.Ready || state == connectivity.Idle
	return state.String(), healthy
}

// serveHealth handles the /health endpoint with upstream and breaker state.
func (h *handler) serveHealth(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"status":    statusHealthy,
		"service":   serviceLabelValue,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	coreState, coreOK := checkConn(h.core)
	resp["upstream_core"] = map[string]interface{}{
		"state":   coreState,
		"healthy": coreOK,
	}

	identityState, identityOK := checkConn(h.identity)
	resp["upstream_identity"] = map[string]interface{}{
		"state":   identityState,
		"healthy": identityOK,
	}

	// Add circuit breaker state
	cbState := map[string]string{}
	if h.coreBreaker != nil {
		cbState["core"] = h.coreBreaker.StateName()
	}
	if h.identityBreaker != nil {
		cbState["identity"] = h.identityBreaker.StateName()
	}
	resp["circuit_breaker"] = cbState

	w.Header().Set(grpcconst.HeaderContentType, grpcconst.ContentTypeJSON)

	if !coreOK || !identityOK {
		resp["status"] = statusDegraded
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.log.WarnContext(r.Context(), logHealthResponseWriteFailed, logger.FieldPath, r.URL.Path, logger.FieldError, err)
	}
}

// serveReady handles the /health/ready endpoint.
// Returns 503 if any upstream is unhealthy or any breaker is open.
func (h *handler) serveReady(w http.ResponseWriter, r *http.Request) {
	_, coreOK := checkConn(h.core)
	_, identityOK := checkConn(h.identity)

	coreBreakerOpen := h.coreBreaker != nil && h.coreBreaker.State() == resilience.BreakerOpen
	identityBreakerOpen := h.identityBreaker != nil && h.identityBreaker.State() == resilience.BreakerOpen

	ready := coreOK && identityOK && !coreBreakerOpen && !identityBreakerOpen

	w.Header().Set(grpcconst.HeaderContentType, grpcconst.ContentTypeJSON)
	if ready {
		w.WriteHeader(http.StatusOK)
		h.writeBody(w, r, bodyStatusPrefix+statusHealthy+bodyStatusSuffix)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		h.writeBody(w, r, bodyUnhealthy)
	}
}

// serveLive handles the /health/live endpoint (unconditional 200).
func (h *handler) serveLive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(grpcconst.HeaderContentType, grpcconst.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	h.writeBody(w, r, bodyAlive)
}

// servePing handles the /health/ping endpoint (unconditional 200).
func (h *handler) servePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(grpcconst.HeaderContentType, grpcconst.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	h.writeBody(w, r, bodyStatusPrefix+statusHealthy+bodyStatusSuffix)
}

// StartHealthServer starts the HTTP health endpoint with all standard paths.
func StartHealthServer(ctx context.Context, log logger.Logger, port int, core, identity *grpc.ClientConn, coreBreaker, identityBreaker *resilience.UpstreamBreaker) error {
	// Explicit nil-to-interface conversion: a nil concrete pointer must reach
	// the handler as a nil interface so its nil checks keep working.
	var coreState, identityState connectionState
	if core != nil {
		coreState = core
	}
	if identity != nil {
		identityState = identity
	}
	var coreBreakerState, identityBreakerState breakerState
	if coreBreaker != nil {
		coreBreakerState = coreBreaker
	}
	if identityBreaker != nil {
		identityBreakerState = identityBreaker
	}
	h := newHandler(coreState, identityState, coreBreakerState, identityBreakerState, log)

	mux := http.NewServeMux()
	mux.HandleFunc(routeHealth, h.serveHealth)
	mux.HandleFunc(routeHealthReady, h.serveReady)
	mux.HandleFunc(routeHealthLive, h.serveLive)
	mux.HandleFunc(routeHealthPing, h.servePing)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	//nolint:gosec // G118: shutdown deliberately uses a fresh context - the parent is already done
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout) // context-root: shutdown
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.WarnContext(shutdownCtx, logHealthServerShutdownError, logger.FieldError, err)
		}
	}()

	return server.ListenAndServe()
}
