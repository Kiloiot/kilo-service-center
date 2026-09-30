package delivery

import (
	"errors"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// RetryPolicy parks permanent failures and retries all others with a capped exponential backoff.
type RetryPolicy struct {
	base     time.Duration
	maxDelay time.Duration
}

// NewRetryPolicy rejects a non-positive base or a cap below the base.
func NewRetryPolicy(base, maxDelay time.Duration) (RetryPolicy, error) {
	if base <= 0 || maxDelay < base {
		return RetryPolicy{}, fmt.Errorf(errFmtRetryPolicyBounds, errInvalidRetryPolicy, base, maxDelay)
	}
	return RetryPolicy{base: base, maxDelay: maxDelay}, nil
}

// permanentFailures name a channel this process cannot deliver to or a message that is gone.
var permanentFailures = []error{errUnknownChannel, errChannelNotConfigured, storage.ErrNotFound}

// Permanent reports whether err is a failure no retry can fix.
func (p RetryPolicy) Permanent(err error) bool {
	for _, permanent := range permanentFailures {
		if errors.Is(err, permanent) {
			return true
		}
	}
	return false
}

// Delay is the wait after failed attempt n (from 1): the base doubled n-1 times, capped.
func (p RetryPolicy) Delay(attempt int) time.Duration {
	delay := p.base
	for n := 1; n < attempt; n++ {
		if delay > p.maxDelay/backoffGrowth {
			return p.maxDelay
		}
		delay *= backoffGrowth
	}
	return min(delay, p.maxDelay)
}

// Lease hides a claimed row for the longest delay, so an attempt lost with its process retries no later than a failed one.
func (p RetryPolicy) Lease() time.Duration {
	return p.maxDelay
}
