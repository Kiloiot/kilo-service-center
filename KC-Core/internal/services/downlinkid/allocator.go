// Package downlinkid draws the service center's own queue id for every
// downlink - the id base stations see - and retries the enqueue while the
// drawn id is already assigned to another downlink.
package downlinkid

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// Generator draws candidate queue ids: positive and at most 2^53-1, within
// the BIGINT que_id column.
type Generator interface {
	NextQueueID(ctx context.Context) (int64, error)
}

// Enqueue persists a downlink under the candidate queue id.
type Enqueue = func(ctx context.Context, queID int64) error

// Allocator runs an enqueue under generated queue ids until one is accepted,
// a failure other than a queue id collision occurs, or the attempts are
// exhausted, and returns the accepted id.
type Allocator interface {
	Allocate(ctx context.Context, enqueue Enqueue) (int64, error)
}

type allocator struct {
	ids      Generator
	attempts int
}

// New returns the allocator every downlink enqueue shares; a missing
// generator or a non-positive attempt budget is a wiring fault and is
// reported at construction rather than on the first downlink.
func New(ids Generator, attempts int) (Allocator, error) {
	if ids == nil {
		return nil, ErrNilGenerator
	}
	if attempts <= 0 {
		return nil, fmt.Errorf("%w: %d", ErrInvalidAttempts, attempts)
	}
	return allocator{ids: ids, attempts: attempts}, nil
}

func (a allocator) Allocate(ctx context.Context, enqueue Enqueue) (int64, error) {
	if enqueue == nil {
		return 0, ErrNilEnqueue
	}
	var lastErr error
	for range a.attempts {
		queID, err := a.ids.NextQueueID(ctx)
		if err != nil {
			return 0, fmt.Errorf("%w: %w", ErrEntropyUnavailable, err)
		}
		err = enqueue(ctx, queID)
		if err == nil {
			return queID, nil
		}
		if !errors.Is(err, storage.ErrDownlinkQueueIDTaken) {
			return 0, err
		}
		lastErr = err
	}
	return 0, fmt.Errorf("%w: %w", ErrAttemptsExhausted, lastErr)
}

// cryptoGenerator draws queue ids uniformly from 1 to maxQueueID with
// crypto/rand, so two service centers never collide the way UnixNano clocks did.
type cryptoGenerator struct{}

// NewCryptoGenerator returns the production Generator.
func NewCryptoGenerator() Generator {
	return cryptoGenerator{}
}

func (cryptoGenerator) NextQueueID(_ context.Context) (int64, error) {
	drawn, err := rand.Int(rand.Reader, big.NewInt(maxQueueID))
	if err != nil {
		return 0, err
	}
	return drawn.Int64() + 1, nil
}
