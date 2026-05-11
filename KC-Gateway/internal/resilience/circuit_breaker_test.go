package resilience

import (
	"errors"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	gobreaker "github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func testConfig() config.GatewayResilienceConfig {
	return config.GatewayResilienceConfig{
		DialTimeout:        5 * time.Second,
		RPCTimeout:         30 * time.Second,
		MaxRetries:         3,
		RetryBackoff:       100 * time.Millisecond,
		RetryMaxBackoff:    1 * time.Second,
		CBMaxRequests:      1,
		CBInterval:         60 * time.Second,
		CBTimeout:          200 * time.Millisecond, // Short for testing
		CBFailureThreshold: 3,
	}
}

func TestExecute_ClosedState_CallsFunction(t *testing.T) {
	b := NewUpstreamBreaker("test-core", testConfig())

	called := false
	err := b.Execute(func() error {
		called = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, called, "fn should be called when breaker is closed")
	assert.Equal(t, BreakerClosed, b.State())
}

func TestExecute_OpensAfterConsecutiveTransportFailures(t *testing.T) {
	cfg := testConfig()
	b := NewUpstreamBreaker("test-core", cfg)

	transportErr := status.Error(codes.Unavailable, "connection refused")

	for i := uint32(0); i < cfg.CBFailureThreshold; i++ {
		_ = b.Execute(func() error {
			return transportErr
		})
	}

	assert.Equal(t, BreakerOpen, b.State())
}

func TestExecute_OpenState_RejectsWithoutCallingFn(t *testing.T) {
	cfg := testConfig()
	b := NewUpstreamBreaker("test-core", cfg)

	transportErr := status.Error(codes.Unavailable, "connection refused")

	// Trip the breaker
	for i := uint32(0); i < cfg.CBFailureThreshold; i++ {
		_ = b.Execute(func() error {
			return transportErr
		})
	}
	require.Equal(t, BreakerOpen, b.State())

	// Attempt while open — fn should never be called
	called := false
	err := b.Execute(func() error {
		called = true
		return nil
	})

	assert.False(t, called, "fn must not be called when breaker is open")
	assert.True(t, errors.Is(err, gobreaker.ErrOpenState),
		"expected gobreaker.ErrOpenState, got %v", err)
}

func TestExecute_HalfOpenRecovery(t *testing.T) {
	cfg := testConfig()
	b := NewUpstreamBreaker("test-core", cfg)

	transportErr := status.Error(codes.Unavailable, "connection refused")

	// Trip the breaker
	for i := uint32(0); i < cfg.CBFailureThreshold; i++ {
		_ = b.Execute(func() error {
			return transportErr
		})
	}
	require.Equal(t, BreakerOpen, b.State())

	// Wait for open-to-half-open timeout
	time.Sleep(cfg.CBTimeout + 50*time.Millisecond)

	// Next successful request should transition to closed
	err := b.Execute(func() error {
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, BreakerClosed, b.State())
}

func TestExecute_ApplicationErrors_DoNotTrip(t *testing.T) {
	cfg := testConfig()
	b := NewUpstreamBreaker("test-core", cfg)

	appErr := status.Error(codes.NotFound, "resource not found")

	for i := 0; i < 100; i++ {
		_ = b.Execute(func() error {
			return appErr
		})
	}

	assert.Equal(t, BreakerClosed, b.State(),
		"application errors should not trip the circuit breaker")
}

func TestExecute_NilFunctionError_IsSuccess(t *testing.T) {
	b := NewUpstreamBreaker("test-core", testConfig())

	err := b.Execute(func() error {
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, BreakerClosed, b.State())
}

func TestIsSuccessful_TransportErrors(t *testing.T) {
	tests := []struct {
		name     string
		code     codes.Code
		expected bool
	}{
		{"Unavailable trips breaker", codes.Unavailable, false},
		{"Internal trips breaker", codes.Internal, false},
		{"DeadlineExceeded trips breaker", codes.DeadlineExceeded, false},
		{"NotFound does not trip", codes.NotFound, true},
		{"InvalidArgument does not trip", codes.InvalidArgument, true},
		{"Unauthenticated does not trip", codes.Unauthenticated, true},
		{"PermissionDenied does not trip", codes.PermissionDenied, true},
		{"FailedPrecondition does not trip", codes.FailedPrecondition, true},
		{"AlreadyExists does not trip", codes.AlreadyExists, true},
		{"Unimplemented does not trip", codes.Unimplemented, true},
		{"OK is successful", codes.OK, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.code == codes.OK {
				err = nil
			} else {
				err = status.Error(tt.code, "test")
			}
			assert.Equal(t, tt.expected, isSuccessful(err))
		})
	}
}

func TestIsSuccessful_NilError(t *testing.T) {
	assert.True(t, isSuccessful(nil))
}

func TestBreakerState_String(t *testing.T) {
	assert.Equal(t, "closed", BreakerClosed.String())
	assert.Equal(t, "half-open", BreakerHalfOpen.String())
	assert.Equal(t, "open", BreakerOpen.String())
	assert.Equal(t, "unknown", BreakerState(-1).String())
}
