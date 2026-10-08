// Package resilience provides upstream connection resilience for KC-Gateway.
package resilience

import (
	"fmt"

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
// Breaker state display names.
const (
	breakerNameClosed   = "closed"
	breakerNameHalfOpen = "half-open"
	breakerNameOpen     = "open"
	breakerNameUnknown  = "unknown"
)

func (s BreakerState) String() string {
	switch s {
	case BreakerClosed:
		return breakerNameClosed
	case BreakerHalfOpen:
		return breakerNameHalfOpen
	case BreakerOpen:
		return breakerNameOpen
	default:
		return breakerNameUnknown
	}
}

// UpstreamBreaker wraps a two-step gobreaker circuit breaker for an upstream
// gRPC connection; the two-step form lets a stream's outcome be recorded
// after the stream ends.
type UpstreamBreaker struct {
	cb *gobreaker.TwoStepCircuitBreaker[any]
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
		cb: gobreaker.NewTwoStepCircuitBreaker[any](settings),
	}
}

// Execute runs fn inside the circuit breaker. If the circuit is open,
// fn is never called and gobreaker.ErrOpenState is returned. A panic in fn
// counts as a failure and is re-raised.
func (b *UpstreamBreaker) Execute(fn func() error) error {
	done, err := b.cb.Allow()
	if err != nil {
		return err
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			done(fmt.Errorf(errFmtPanic, recovered))
			panic(recovered)
		}
	}()
	err = fn()
	done(err)
	return err
}

// RecordResult feeds a completed call's outcome into the breaker's counters
// without holding an execution slot for the call's duration. Used for
// long-lived streaming RPCs, whose terminal error is classified after the
// stream ends. Recording happens only while the breaker is closed: a stream
// that was admitted earlier but finishes during half-open must not consume a
// probe slot or flip the breaker - recovery is decided exclusively by the
// slot-accounted unary probes, and an open breaker rejects anyway.
func (b *UpstreamBreaker) RecordResult(err error) {
	if b.cb.State() != gobreaker.StateClosed {
		return
	}
	done, allowErr := b.cb.Allow()
	if allowErr != nil {
		// The breaker left the closed state after the check; there is nothing to record.
		return
	}
	done(err)
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
