package resilience

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

// Test rate-limit policy values and fixtures.
const (
	testPeerPort              = 12345
	testRateLimitEnabled      = true
	testRequestsPerMin        = 60
	testBurst                 = 3
	testCleanupInterval       = 5 * time.Minute
	testSingleRequestPerMin   = 1
	testSingleBurst           = 1
	testMethodListEndpoints   = "/kilocenter.api.v1.KiloCenterService/ListEndpoints"
	testMethodRegisterAccount = "/kilocenter.api.v1.IdentityService/RegisterAccount"
	testMethodCompatRegister  = "/kilocenter.api.v1.KiloCenterService/RegisterAccount"
)

func newPeerContext(ip string) context.Context {
	return peer.NewContext(testutil.TestContext(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP(ip), Port: testPeerPort},
	})
}

func defaultConfig() config.GatewayRateLimitConfig {
	return config.GatewayRateLimitConfig{
		Enabled:         testRateLimitEnabled,
		RequestsPerMin:  testRequestsPerMin,
		Burst:           testBurst,
		CleanupInterval: testCleanupInterval,
	}
}

func passHandler(_ interface{}, _ grpc.ServerStream) error {
	return nil
}

func TestRateLimiter_AllowsNonLimitedMethods(t *testing.T) {
	rl := NewRegistrationRateLimiter(defaultConfig())
	defer rl.Close()

	interceptor := rl.StreamInterceptor()

	stream := &mockServerStream{ctx: newPeerContext("192.168.1.1")}
	info := &grpc.StreamServerInfo{FullMethod: testMethodListEndpoints}

	handlerCalled := false
	handler := func(_ interface{}, _ grpc.ServerStream) error {
		handlerCalled = true
		return nil
	}

	err := interceptor(nil, stream, info, handler)

	require.NoError(t, err)
	assert.True(t, handlerCalled, "handler should be called for non-rate-limited methods")
}

func TestRateLimiter_AllowsBurstThenRejects(t *testing.T) {
	cfg := defaultConfig()
	cfg.Burst = testBurst
	cfg.RequestsPerMin = testSingleRequestPerMin // very low rate so token refill is negligible during the test

	rl := NewRegistrationRateLimiter(cfg)
	defer rl.Close()

	interceptor := rl.StreamInterceptor()

	stream := &mockServerStream{ctx: newPeerContext("10.0.0.1")}
	info := &grpc.StreamServerInfo{FullMethod: testMethodRegisterAccount}

	// First `burst` requests should succeed
	for i := 0; i < cfg.Burst; i++ {
		err := interceptor(nil, stream, info, passHandler)
		require.NoError(t, err, "request %d within burst should be allowed", i+1)
	}

	// Next request should be rejected with ResourceExhausted
	err := interceptor(nil, stream, info, passHandler)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status")
	assert.Equal(t, codes.ResourceExhausted, st.Code())

	// Verify the second rate-limited method is also enforced
	info2 := &grpc.StreamServerInfo{FullMethod: testMethodCompatRegister}
	err = interceptor(nil, stream, info2, passHandler)
	require.Error(t, err)

	st2, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.ResourceExhausted, st2.Code())
}

func TestRateLimiter_DifferentIPsIndependent(t *testing.T) {
	cfg := defaultConfig()
	cfg.Burst = testSingleBurst
	cfg.RequestsPerMin = testSingleRequestPerMin

	rl := NewRegistrationRateLimiter(cfg)
	defer rl.Close()

	interceptor := rl.StreamInterceptor()

	info := &grpc.StreamServerInfo{FullMethod: testMethodRegisterAccount}

	streamA := &mockServerStream{ctx: newPeerContext("172.16.0.1")}
	streamB := &mockServerStream{ctx: newPeerContext("172.16.0.2")}

	// Exhaust IP A's burst
	err := interceptor(nil, streamA, info, passHandler)
	require.NoError(t, err, "first request from IP A should be allowed")

	err = interceptor(nil, streamA, info, passHandler)
	require.Error(t, err, "second request from IP A should be rejected")

	// IP B should still be allowed (independent limiter)
	err = interceptor(nil, streamB, info, passHandler)
	require.NoError(t, err, "first request from IP B should be allowed")

	err = interceptor(nil, streamB, info, passHandler)
	require.Error(t, err, "second request from IP B should be rejected")
}

func TestRateLimiter_Close(t *testing.T) {
	rl := NewRegistrationRateLimiter(defaultConfig())

	// Close should not panic
	assert.NotPanics(t, func() {
		rl.Close()
	})
}

func TestRateLimiter_DoubleClose(t *testing.T) {
	rl := NewRegistrationRateLimiter(defaultConfig())

	// Double close must not panic (sync.Once guards the channel close)
	assert.NotPanics(t, func() {
		rl.Close()
		rl.Close()
	})
}
