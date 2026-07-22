package main

import (
	"context"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/resilience"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// fakeServerStream carries a controllable context through the interceptor.
type fakeServerStream struct {
	ctx context.Context
}

func (f *fakeServerStream) Context() context.Context     { return f.ctx }
func (f *fakeServerStream) SetHeader(metadata.MD) error  { return nil }
func (f *fakeServerStream) SendHeader(metadata.MD) error { return nil }
func (f *fakeServerStream) SetTrailer(metadata.MD)       {}
func (f *fakeServerStream) SendMsg(_ interface{}) error  { return nil }
func (f *fakeServerStream) RecvMsg(_ interface{}) error  { return nil }

func newTestBreakers(threshold uint32) (*resilience.UpstreamBreaker, *resilience.UpstreamBreaker) {
	cfg := config.GatewayResilienceConfig{
		CBMaxRequests:      3,
		CBInterval:         time.Minute,
		CBTimeout:          time.Minute,
		CBFailureThreshold: threshold,
	}
	return resilience.NewUpstreamBreaker("core", cfg), resilience.NewUpstreamBreaker("identity", cfg)
}

func streamInfo(method string) *grpc.StreamServerInfo {
	return &grpc.StreamServerInfo{FullMethod: method}
}

// proxyTeardownErr is the shape the transparent proxy returns when the client
// side of a stream goes away.
func proxyTeardownErr() error {
	return status.Error(codes.Internal, "failed proxying s2c: context canceled")
}

// TestBreakerStream_ClientTeardownNeverTrips: the browser closing realtime
// streams (canceled client context + Internal from the proxy) must not open
// the breaker no matter how often it happens.
func TestBreakerStream_ClientTeardownNeverTrips(t *testing.T) {
	core, identity := newTestBreakers(3)
	interceptor := breakerStreamInterceptor(core, identity, 0)

	canceled, cancel := context.WithCancel(testutil.TestContext())
	cancel()
	handler := func(_ interface{}, _ grpc.ServerStream) error { return proxyTeardownErr() }

	for i := 0; i < 10; i++ {
		err := interceptor(nil, &fakeServerStream{ctx: canceled}, streamInfo("/kilocenter.api.v1.KiloCenterService/StreamMessages"), handler)
		require.Error(t, err, "the handler error is passed through")
	}

	assert.Equal(t, resilience.BreakerClosed, core.State(),
		"client-initiated stream teardown must never count as an upstream failure")
}

// TestBreakerStream_UpstreamFailureStillTrips: a stream that dies while the
// client is still connected is a real upstream failure and opens the breaker
// at the threshold.
func TestBreakerStream_UpstreamFailureStillTrips(t *testing.T) {
	core, identity := newTestBreakers(3)
	interceptor := breakerStreamInterceptor(core, identity, 0)

	handler := func(_ interface{}, _ grpc.ServerStream) error {
		return status.Error(codes.Unavailable, "upstream gone")
	}

	for i := 0; i < 3; i++ {
		_ = interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo("/kilocenter.api.v1.KiloCenterService/StreamMessages"), handler)
	}

	assert.Equal(t, resilience.BreakerOpen, core.State(),
		"genuine upstream stream failures must still trip the breaker")
}

// TestBreakerStream_OpenBreakerRejectsStreams: streams respect an open breaker
// without invoking the handler.
func TestBreakerStream_OpenBreakerRejectsStreams(t *testing.T) {
	core, identity := newTestBreakers(1)
	interceptor := breakerStreamInterceptor(core, identity, 0)

	failing := func(_ interface{}, _ grpc.ServerStream) error {
		return status.Error(codes.Unavailable, "upstream gone")
	}
	_ = interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo("/x/Stream"), failing)
	require.Equal(t, resilience.BreakerOpen, core.State())

	handlerCalled := false
	err := interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo("/x/Stream"), func(_ interface{}, _ grpc.ServerStream) error {
		handlerCalled = true
		return nil
	})

	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.Contains(t, err.Error(), "circuit breaker is open")
	assert.False(t, handlerCalled, "an open breaker must reject before the handler runs")
}

// TestBreakerStream_HalfOpenRejectsStreams: a half-open breaker admits only
// slot-accounted unary probes; streams are rejected with the recovering
// message until the probes close the breaker again.
func TestBreakerStream_HalfOpenRejectsStreams(t *testing.T) {
	cfg := config.GatewayResilienceConfig{
		CBMaxRequests:      3,
		CBInterval:         time.Minute,
		CBTimeout:          50 * time.Millisecond,
		CBFailureThreshold: 1,
	}
	core := resilience.NewUpstreamBreaker("core", cfg)
	identity := resilience.NewUpstreamBreaker("identity", cfg)
	interceptor := breakerStreamInterceptor(core, identity, 0)

	fail := func(_ interface{}, _ grpc.ServerStream) error {
		return status.Error(codes.Unavailable, "upstream gone")
	}
	_ = interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo("/x/Stream"), fail)
	require.Equal(t, resilience.BreakerOpen, core.State())

	// After CBTimeout the breaker transitions to half-open on next inspection
	time.Sleep(60 * time.Millisecond)
	require.Equal(t, resilience.BreakerHalfOpen, core.State())

	handlerCalled := false
	err := interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo("/x/Stream"), func(_ interface{}, _ grpc.ServerStream) error {
		handlerCalled = true
		return nil
	})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.Contains(t, err.Error(), "circuit breaker is recovering")
	assert.False(t, handlerCalled, "a half-open breaker must not admit streams")

	// Successful unary probes close the breaker (CBMaxRequests consecutive
	// successes), after which streams flow again.
	unary := "/kilocenter.api.v1.KiloCenterService/ListEndPoints"
	require.True(t, unaryMethods[unary], "fixture must use a real unary method")
	ok := func(_ interface{}, _ grpc.ServerStream) error { return nil }
	for i := 0; i < int(cfg.CBMaxRequests); i++ {
		require.NoError(t, interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo(unary), ok))
	}
	require.Equal(t, resilience.BreakerClosed, core.State())

	require.NoError(t, interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo("/x/Stream"), ok),
		"streams must be admitted again once the probes closed the breaker")
}

// TestBreakerStream_OldStreamCannotAlterHalfOpenRecovery: a stream admitted
// while the breaker was closed but finishing during half-open must not
// consume a probe slot or flip the breaker - recovery belongs exclusively to
// the slot-accounted unary probes.
func TestBreakerStream_OldStreamCannotAlterHalfOpenRecovery(t *testing.T) {
	cfg := config.GatewayResilienceConfig{
		CBMaxRequests:      2,
		CBInterval:         time.Minute,
		CBTimeout:          50 * time.Millisecond,
		CBFailureThreshold: 1,
	}
	core := resilience.NewUpstreamBreaker("core", cfg)
	identity := resilience.NewUpstreamBreaker("identity", cfg)
	interceptor := breakerStreamInterceptor(core, identity, 0)

	// Old stream admitted while closed; its handler blocks until released.
	// The handler signals admission so the breaker is guaranteed to trip
	// only after the old stream is already running.
	release := make(chan error)
	admitted := make(chan struct{})
	oldStreamDone := make(chan error, 1)
	go func() {
		oldStreamDone <- interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo("/x/Stream"),
			func(_ interface{}, _ grpc.ServerStream) error {
				close(admitted)
				return <-release
			})
	}()
	<-admitted

	// Trip the breaker through another stream, then advance to half-open.
	_ = interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo("/x/Stream"),
		func(_ interface{}, _ grpc.ServerStream) error {
			return status.Error(codes.Unavailable, "upstream gone")
		})
	require.Equal(t, resilience.BreakerOpen, core.State())
	time.Sleep(60 * time.Millisecond)
	require.Equal(t, resilience.BreakerHalfOpen, core.State())

	// The old stream finishes - successfully and then (via a second run)
	// with an error; neither may move the breaker out of half-open.
	release <- nil
	require.NoError(t, <-oldStreamDone)
	assert.Equal(t, resilience.BreakerHalfOpen, core.State(),
		"an old stream's success must not close a half-open breaker")

	core.RecordResult(status.Error(codes.Unavailable, "late stream failure"))
	assert.Equal(t, resilience.BreakerHalfOpen, core.State(),
		"an old stream's failure must not reopen a half-open breaker")

	// The unary probes alone complete recovery.
	unary := "/kilocenter.api.v1.KiloCenterService/ListEndPoints"
	require.True(t, unaryMethods[unary])
	ok := func(_ interface{}, _ grpc.ServerStream) error { return nil }
	for i := 0; i < int(cfg.CBMaxRequests); i++ {
		require.NoError(t, interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo(unary), ok))
	}
	assert.Equal(t, resilience.BreakerClosed, core.State(),
		"recovery is decided by the unary probes")
}

// TestBreakerStream_SuccessfulStreamRecordsSuccess: clean stream completion
// resets the consecutive-failure count.
func TestBreakerStream_SuccessfulStreamRecordsSuccess(t *testing.T) {
	core, identity := newTestBreakers(3)
	interceptor := breakerStreamInterceptor(core, identity, 0)

	fail := func(_ interface{}, _ grpc.ServerStream) error {
		return status.Error(codes.Unavailable, "blip")
	}
	ok := func(_ interface{}, _ grpc.ServerStream) error { return nil }

	_ = interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo("/x/Stream"), fail)
	_ = interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo("/x/Stream"), fail)
	require.NoError(t, interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo("/x/Stream"), ok))
	_ = interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo("/x/Stream"), fail)
	_ = interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo("/x/Stream"), fail)

	assert.Equal(t, resilience.BreakerClosed, core.State(),
		"a successful stream between failures resets the consecutive count")
}

// TestBreakerStream_UnaryPathCountsFailures: unary methods keep the original
// in-breaker execution (failures count, open breaker rejects with the mapped
// message).
func TestBreakerStream_UnaryPathCountsFailures(t *testing.T) {
	core, identity := newTestBreakers(2)
	interceptor := breakerStreamInterceptor(core, identity, time.Second)

	unary := "/kilocenter.api.v1.KiloCenterService/ListEndPoints"
	require.True(t, unaryMethods[unary], "fixture must use a real unary method")

	fail := func(_ interface{}, _ grpc.ServerStream) error {
		return status.Error(codes.Internal, "boom")
	}
	_ = interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo(unary), fail)
	_ = interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo(unary), fail)
	require.Equal(t, resilience.BreakerOpen, core.State())

	err := interceptor(nil, &fakeServerStream{ctx: testutil.TestContext()}, streamInfo(unary), fail)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
}
