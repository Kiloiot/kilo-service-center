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

// Test resilience policy values for breaker behavior.
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

	// testHalfOpenSlack pads the CBTimeout sleep so the breaker is reliably
	// past its open window before the half-open assertion.
	testHalfOpenSlack = 50 * time.Millisecond

	// testAppErrorAttempts is how many application errors are replayed to
	// prove they never trip the breaker.
	testAppErrorAttempts = 100
)

// Status texts used as test failure fixtures.
const (
	testErrConnRefused     = "connection refused"
	testErrResourceMissing = "resource not found"
	testErrGeneric         = "test"
)

func testConfig() config.GatewayResilienceConfig {
	return config.GatewayResilienceConfig{
		DialTimeout:        testDialTimeout,
		RPCTimeout:         testRPCTimeout,
		MaxRetries:         testMaxRetries,
		RetryBackoff:       testRetryBackoff,
		RetryMaxBackoff:    testRetryMaxBackoff,
		CBMaxRequests:      testCBMaxRequests,
		CBInterval:         testCBInterval,
		CBTimeout:          testCBTimeout, // Short for testing
		CBFailureThreshold: testCBFailureThreshold,
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

	transportErr := status.Error(codes.Unavailable, testErrConnRefused)

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

	transportErr := status.Error(codes.Unavailable, testErrConnRefused)

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

	transportErr := status.Error(codes.Unavailable, testErrConnRefused)

	// Trip the breaker
	for i := uint32(0); i < cfg.CBFailureThreshold; i++ {
		_ = b.Execute(func() error {
			return transportErr
		})
	}
	require.Equal(t, BreakerOpen, b.State())

	// Wait for open-to-half-open timeout
	time.Sleep(cfg.CBTimeout + testHalfOpenSlack)

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

	appErr := status.Error(codes.NotFound, testErrResourceMissing)

	for i := 0; i < testAppErrorAttempts; i++ {
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
				err = status.Error(tt.code, testErrGeneric)
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

// A panicking call counts as a failure and the panic still reaches the caller.
func TestExecute_PanicCountsAsFailureAndPropagates(t *testing.T) {
	cfg := testConfig()
	b := NewUpstreamBreaker("test-core", cfg)

	for i := uint32(0); i < cfg.CBFailureThreshold; i++ {
		assert.PanicsWithValue(t, testErrGeneric, func() {
			_ = b.Execute(func() error { panic(testErrGeneric) })
		})
	}

	assert.Equal(t, BreakerOpen, b.State())
}

// A stream result recorded while closed feeds the failure counters.
func TestRecordResult_TransportFailuresOpenTheBreaker(t *testing.T) {
	cfg := testConfig()
	b := NewUpstreamBreaker("test-core", cfg)

	for i := uint32(0); i < cfg.CBFailureThreshold; i++ {
		b.RecordResult(status.Error(codes.Unavailable, testErrConnRefused))
	}

	assert.Equal(t, BreakerOpen, b.State())
}
