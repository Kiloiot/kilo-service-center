package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/resilience"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func testResilienceConfig() config.GatewayResilienceConfig {
	return config.GatewayResilienceConfig{
		DialTimeout:        5 * time.Second,
		RPCTimeout:         30 * time.Second,
		MaxRetries:         3,
		RetryBackoff:       100 * time.Millisecond,
		RetryMaxBackoff:    1 * time.Second,
		CBMaxRequests:      1,
		CBInterval:         60 * time.Second,
		CBTimeout:          200 * time.Millisecond,
		CBFailureThreshold: 3,
	}
}

func newIdleConn(t *testing.T) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient("passthrough:///localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestHealthHandler_AllHealthy(t *testing.T) {
	core := newIdleConn(t)
	identity := newIdleConn(t)
	cfg := testResilienceConfig()
	coreBreaker := resilience.NewUpstreamBreaker("core", cfg)
	identityBreaker := resilience.NewUpstreamBreaker("identity", cfg)

	h := NewHandler(core, identity, coreBreaker, identityBreaker)
	rr := httptest.NewRecorder()
	h.ServeHealth(rr, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "healthy", resp["status"])

	// Verify circuit breaker state in response
	cb, ok := resp["circuit_breaker"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "closed", cb["core"])
	assert.Equal(t, "closed", cb["identity"])
}

func TestHealthHandler_Degraded(t *testing.T) {
	core := newIdleConn(t)
	cfg := testResilienceConfig()
	coreBreaker := resilience.NewUpstreamBreaker("core", cfg)
	identityBreaker := resilience.NewUpstreamBreaker("identity", cfg)

	h := NewHandler(core, nil, coreBreaker, identityBreaker)
	rr := httptest.NewRecorder()
	h.ServeHealth(rr, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "degraded", resp["status"])
}

func TestHealthReady_AllHealthy(t *testing.T) {
	core := newIdleConn(t)
	identity := newIdleConn(t)
	cfg := testResilienceConfig()

	h := NewHandler(core, identity,
		resilience.NewUpstreamBreaker("core", cfg),
		resilience.NewUpstreamBreaker("identity", cfg))

	rr := httptest.NewRecorder()
	h.ServeReady(rr, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"healthy"`)
}

func TestHealthReady_BreakerOpen(t *testing.T) {
	core := newIdleConn(t)
	identity := newIdleConn(t)
	cfg := testResilienceConfig()

	coreBreaker := resilience.NewUpstreamBreaker("core", cfg)
	// Trip the core breaker
	transportErr := status.Error(codes.Unavailable, "connection refused")
	for i := uint32(0); i < cfg.CBFailureThreshold; i++ {
		_ = coreBreaker.Execute(func() error { return transportErr })
	}

	h := NewHandler(core, identity, coreBreaker, resilience.NewUpstreamBreaker("identity", cfg))
	rr := httptest.NewRecorder()
	h.ServeReady(rr, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), `"unhealthy"`)
}

func TestHealthReady_ConnUnhealthy(t *testing.T) {
	core := newIdleConn(t)
	cfg := testResilienceConfig()

	h := NewHandler(core, nil,
		resilience.NewUpstreamBreaker("core", cfg),
		resilience.NewUpstreamBreaker("identity", cfg))

	rr := httptest.NewRecorder()
	h.ServeReady(rr, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
}

func TestHealthLive(t *testing.T) {
	rr := httptest.NewRecorder()
	ServeLive(rr, httptest.NewRequest(http.MethodGet, "/health/live", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"alive"`)
}

func TestHealthPing(t *testing.T) {
	rr := httptest.NewRecorder()
	ServePing(rr, httptest.NewRequest(http.MethodGet, "/health/ping", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"healthy"`)
}
