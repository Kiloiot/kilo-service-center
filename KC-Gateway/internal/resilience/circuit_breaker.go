// Package resilience provides upstream connection resilience for KC-Gateway.
package resilience

import (
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/sony/gobreaker/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BreakerState represents the circuit breaker state as a numeric value.
type BreakerState int

const (
	// BreakerClosed indicates the circuit is closed (healthy).
	BreakerClosed BreakerState = 0
	// BreakerHalfOpen indicates the circuit is testing recovery.
	BreakerHalfOpen BreakerState = 1
	// BreakerOpen indicates the circuit is open (upstream unhealthy).
	BreakerOpen BreakerState = 2
)

// String returns the breaker state name.
func (s BreakerState) String() string {
	switch s {
	case BreakerClosed:
		return "closed"
	case BreakerHalfOpen:
		return "half-open"
	case BreakerOpen:
		return "open"
	default:
		return "unknown"
	}
}

// UpstreamBreaker wraps a gobreaker circuit breaker for an upstream gRPC connection.
type UpstreamBreaker struct {
	cb *gobreaker.CircuitBreaker[any]
}

// NewUpstreamBreaker creates a circuit breaker for the named upstream.
func NewUpstreamBreaker(name string, cfg config.GatewayResilienceConfig) *UpstreamBreaker {
	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: cfg.CBMaxRequests,
		Interval:    cfg.CBInterval,
		Timeout:     cfg.CBTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= uint32(cfg.CBFailureThreshold)
		},
		IsSuccessful: isSuccessful,
	}

	return &UpstreamBreaker{
		cb: gobreaker.NewCircuitBreaker[any](settings),
	}
}

// Execute runs fn inside the circuit breaker. If the circuit is open,
// fn is never called and gobreaker.ErrOpenState is returned.
func (b *UpstreamBreaker) Execute(fn func() error) error {
	_, err := b.cb.Execute(func() (any, error) {
		return nil, fn()
	})
	return err
}

// State returns the current breaker state.
func (b *UpstreamBreaker) State() BreakerState {
	switch b.cb.State() {
	case gobreaker.StateClosed:
		return BreakerClosed
	case gobreaker.StateHalfOpen:
		return BreakerHalfOpen
	case gobreaker.StateOpen:
		return BreakerOpen
	default:
		return BreakerClosed
	}
}

// StateName returns the breaker state as a string.
func (b *UpstreamBreaker) StateName() string {
	return b.State().String()
}

// isSuccessful determines whether a gRPC error should be treated as a transport
// failure (trips the breaker) or an application error (does not trip).
// Only Unavailable, Internal, and DeadlineExceeded are transport failures.
func isSuccessful(err error) bool {
	if err == nil {
		return true
	}

	st, ok := status.FromError(err)
	if !ok {
		// Non-gRPC error — treat as transport failure
		return false
	}

	switch st.Code() {
	case codes.Unavailable, codes.Internal, codes.DeadlineExceeded:
		return false
	default:
		// Application-level errors (NotFound, InvalidArgument, Unauthenticated,
		// PermissionDenied, FailedPrecondition, etc.) do not trip the breaker
		return true
	}
}
