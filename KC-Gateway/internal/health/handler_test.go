package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/resilience"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// Test resilience policy values exercised by the health endpoints.
const (
	testDialTimeout        = 5 * time.Second
	testRPCTimeout         = 30 * time.Second
	testMaxRetries         = 3
	testRetryBackoff       = 100 * time.Millisecond
	testRetryMaxBackoff    = 1 * time.Second
	testCBMaxRequests      = 1
	testCBInterval         = 60 * time.Second
	testCBTimeout          = 200 * time.Millisecond
	testCBFailureThreshold = 3
)

// testErrConnRefused is the transport-failure status text used to trip breakers.
const testErrConnRefused = "connection refused"

func testResilienceConfig() config.GatewayResilienceConfig {
	return config.GatewayResilienceConfig{
		DialTimeout:        testDialTimeout,
		RPCTimeout:         testRPCTimeout,
		MaxRetries:         testMaxRetries,
		RetryBackoff:       testRetryBackoff,
		RetryMaxBackoff:    testRetryMaxBackoff,
		CBMaxRequests:      testCBMaxRequests,
		CBInterval:         testCBInterval,
		CBTimeout:          testCBTimeout,
		CBFailureThreshold: testCBFailureThreshold,
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

	h := newHandler(core, identity, coreBreaker, identityBreaker, logger.NewNop())
	rr := httptest.NewRecorder()
	h.serveHealth(rr, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, grpcconst.ContentTypeJSON, rr.Header().Get(grpcconst.HeaderContentType))

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

	h := newHandler(core, nil, coreBreaker, identityBreaker, logger.NewNop())
	rr := httptest.NewRecorder()
	h.serveHealth(rr, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "degraded", resp["status"])
}

func TestHealthReady_AllHealthy(t *testing.T) {
	core := newIdleConn(t)
	identity := newIdleConn(t)
	cfg := testResilienceConfig()

	h := newHandler(core, identity,
		resilience.NewUpstreamBreaker("core", cfg),
		resilience.NewUpstreamBreaker("identity", cfg), logger.NewNop())

	rr := httptest.NewRecorder()
	h.serveReady(rr, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"healthy"`)
}

func TestHealthReady_BreakerOpen(t *testing.T) {
	core := newIdleConn(t)
	identity := newIdleConn(t)
	cfg := testResilienceConfig()

	coreBreaker := resilience.NewUpstreamBreaker("core", cfg)
	// Trip the core breaker
	transportErr := status.Error(codes.Unavailable, testErrConnRefused)
	for i := uint32(0); i < cfg.CBFailureThreshold; i++ {
		_ = coreBreaker.Execute(func() error { return transportErr })
	}

	h := newHandler(core, identity, coreBreaker, resilience.NewUpstreamBreaker("identity", cfg), logger.NewNop())
	rr := httptest.NewRecorder()
	h.serveReady(rr, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), `"unhealthy"`)
}

func TestHealthReady_ConnUnhealthy(t *testing.T) {
	core := newIdleConn(t)
	cfg := testResilienceConfig()

	h := newHandler(core, nil,
		resilience.NewUpstreamBreaker("core", cfg),
		resilience.NewUpstreamBreaker("identity", cfg), logger.NewNop())

	rr := httptest.NewRecorder()
	h.serveReady(rr, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
}

func TestHealthLive(t *testing.T) {
	rr := httptest.NewRecorder()
	(&handler{log: logger.NewNop()}).serveLive(rr, httptest.NewRequest(http.MethodGet, "/health/live", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"alive"`)
}

func TestHealthPing(t *testing.T) {
	rr := httptest.NewRecorder()
	(&handler{log: logger.NewNop()}).servePing(rr, httptest.NewRequest(http.MethodGet, "/health/ping", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"healthy"`)
}
