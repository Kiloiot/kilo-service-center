// Package health provides HTTP health endpoints for KC-Gateway.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/resilience"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// statusHealthy is the health endpoint status value for a passing check.
const statusHealthy = "healthy"

// Handler provides gateway health check endpoints.
type Handler struct {
	core            *grpc.ClientConn
	identity        *grpc.ClientConn
	coreBreaker     *resilience.UpstreamBreaker
	identityBreaker *resilience.UpstreamBreaker
}

// NewHandler creates a health handler that checks both upstream connections and breaker state.
func NewHandler(core, identity *grpc.ClientConn, coreBreaker, identityBreaker *resilience.UpstreamBreaker) *Handler {
	return &Handler{
		core:            core,
		identity:        identity,
		coreBreaker:     coreBreaker,
		identityBreaker: identityBreaker,
	}
}

// checkConn returns the connectivity state and whether it is considered healthy.
func checkConn(conn *grpc.ClientConn) (string, bool) {
	if conn == nil {
		return "SHUTDOWN", false
	}
	state := conn.GetState()
	healthy := state == connectivity.Ready || state == connectivity.Idle
	return state.String(), healthy
}

// ServeHealth handles the /health endpoint with upstream and breaker state.
func (h *Handler) ServeHealth(w http.ResponseWriter, _ *http.Request) {
	resp := map[string]interface{}{
		"status":    statusHealthy,
		"service":   "kc-gateway",
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

	w.Header().Set("Content-Type", "application/json")

	if !coreOK || !identityOK {
		resp["status"] = "degraded"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// ServeReady handles the /health/ready endpoint.
// Returns 503 if any upstream is unhealthy or any breaker is open.
func (h *Handler) ServeReady(w http.ResponseWriter, _ *http.Request) {
	_, coreOK := checkConn(h.core)
	_, identityOK := checkConn(h.identity)

	coreBreakerOpen := h.coreBreaker != nil && h.coreBreaker.State() == resilience.BreakerOpen
	identityBreakerOpen := h.identityBreaker != nil && h.identityBreaker.State() == resilience.BreakerOpen

	ready := coreOK && identityOK && !coreBreakerOpen && !identityBreakerOpen

	w.Header().Set("Content-Type", "application/json")
	if ready {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"` + statusHealthy + `"}`))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"unhealthy"}`))
	}
}

// ServeLive handles the /health/live endpoint (unconditional 200).
func ServeLive(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"alive"}`))
}

// ServePing handles the /health/ping endpoint (unconditional 200).
func ServePing(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"` + statusHealthy + `"}`))
}

// StartHealthServer starts the HTTP health endpoint with all standard paths.
func StartHealthServer(ctx context.Context, port int, core, identity *grpc.ClientConn, coreBreaker, identityBreaker *resilience.UpstreamBreaker) error {
	h := NewHandler(core, identity, coreBreaker, identityBreaker)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.ServeHealth)
	mux.HandleFunc("/health/ready", h.ServeReady)
	mux.HandleFunc("/health/live", ServeLive)
	mux.HandleFunc("/health/ping", ServePing)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	//nolint:gosec // G118: shutdown deliberately uses a fresh context - the parent is already done
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	return server.ListenAndServe()
}
