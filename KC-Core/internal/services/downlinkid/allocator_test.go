package downlinkid

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

const (
	testAttempts     = 5
	noAttempts       = 0
	negativeAttempts = -1
	cryptoDraws      = 256
	// javaScriptMaxSafeInteger is Number.MAX_SAFE_INTEGER, the largest integer
	// a JSON consumer in JavaScript reads exactly.
	javaScriptMaxSafeInteger = int64(1)<<53 - 1
)

var (
	errTestEntropy = errors.New("entropy down")
	errTestPersist = errors.New("persist failed")
)

type sequenceGenerator struct {
	ids   []int64
	next  int
	err   error
	calls int
}

func (g *sequenceGenerator) NextQueueID(_ context.Context) (int64, error) {
	g.calls++
	if g.err != nil {
		return 0, g.err
	}
	id := g.ids[g.next%len(g.ids)]
	g.next++
	return id, nil
}

func taken() error { return fmt.Errorf("enqueue downlink: %w", storage.ErrDownlinkQueueIDTaken) }

type recordingEnqueue struct {
	ids       []int64
	failUntil int
	failWith  error
}

func (r *recordingEnqueue) fn(_ context.Context, queID int64) error {
	r.ids = append(r.ids, queID)
	if len(r.ids) <= r.failUntil {
		return r.failWith
	}
	return nil
}

func TestAllocate_CollisionThenSuccess(t *testing.T) {
	gen := &sequenceGenerator{ids: []int64{101, 102, 103}}
	enq := &recordingEnqueue{failUntil: 1, failWith: taken()}

	got, err := mustNew(t, gen, testAttempts).Allocate(testutil.TestContext(), enq.fn)
	require.NoError(t, err)
	assert.Equal(t, int64(102), got, "the accepted id is returned")
	assert.Equal(t, []int64{101, 102}, enq.ids, "each retry draws a fresh id")
	assert.Equal(t, 2, gen.calls)
}

func TestAllocate_ExhaustsAfterTheBoundedAttempts(t *testing.T) {
	gen := &sequenceGenerator{ids: []int64{1, 2, 3, 4, 5, 6}}
	enq := &recordingEnqueue{failUntil: testAttempts + 1, failWith: taken()}

	_, err := mustNew(t, gen, testAttempts).Allocate(testutil.TestContext(), enq.fn)
	require.ErrorIs(t, err, ErrAttemptsExhausted)
	assert.ErrorIs(t, err, storage.ErrDownlinkQueueIDTaken, "the last refusal stays in the chain")
	assert.Len(t, enq.ids, testAttempts)
	assert.Equal(t, testAttempts, gen.calls)
}

func TestAllocate_EntropyFailureNeverEnqueues(t *testing.T) {
	gen := &sequenceGenerator{ids: []int64{1}, err: errTestEntropy}
	enq := &recordingEnqueue{}

	_, err := mustNew(t, gen, testAttempts).Allocate(testutil.TestContext(), enq.fn)
	require.ErrorIs(t, err, ErrEntropyUnavailable)
	require.ErrorIs(t, err, errTestEntropy)
	assert.Empty(t, enq.ids)
	assert.Equal(t, 1, gen.calls)
}

// TestAllocate_ApplicationQueueIDDuplicateIsNotRetried pins that an
// Application Center id already in flight in its organization is the caller's
// duplicate, not a service center collision a fresh id resolves.
func TestAllocate_ApplicationQueueIDDuplicateIsNotRetried(t *testing.T) {
	gen := &sequenceGenerator{ids: []int64{1, 2}}
	enq := &recordingEnqueue{failUntil: 3, failWith: fmt.Errorf("enqueue downlink: %w", storage.ErrDuplicateKey)}

	_, err := mustNew(t, gen, testAttempts).Allocate(testutil.TestContext(), enq.fn)
	require.ErrorIs(t, err, storage.ErrDuplicateKey)
	assert.Len(t, enq.ids, 1)
}

func TestAllocate_NonCollisionFailureIsNotRetried(t *testing.T) {
	gen := &sequenceGenerator{ids: []int64{1, 2}}
	enq := &recordingEnqueue{failUntil: 3, failWith: errTestPersist}

	_, err := mustNew(t, gen, testAttempts).Allocate(testutil.TestContext(), enq.fn)
	require.ErrorIs(t, err, errTestPersist)
	assert.False(t, errors.Is(err, ErrAttemptsExhausted))
	assert.Len(t, enq.ids, 1)
	assert.Equal(t, 1, gen.calls)
}

func TestCryptoGenerator_DrawsDistinctPositiveJSONSafeIDs(t *testing.T) {
	gen := NewCryptoGenerator()
	seen := make(map[int64]struct{}, cryptoDraws)
	for range cryptoDraws {
		id, err := gen.NextQueueID(testutil.TestContext())
		require.NoError(t, err)
		assert.Positive(t, id, "queue ids must be positive and fit the BIGINT que_id column")
		assert.LessOrEqual(t, id, javaScriptMaxSafeInteger, "MQTT and JSON consumers read the queue id as an exact number")
		_, dup := seen[id]
		assert.False(t, dup, "duplicate queue id drawn: %d", id)
		seen[id] = struct{}{}
	}
}

func mustNew(t *testing.T, gen Generator, attempts int) Allocator {
	t.Helper()
	allocator, err := New(gen, attempts)
	require.NoError(t, err)
	return allocator
}

func TestNew_RejectsMissingGeneratorAndAttemptBudget(t *testing.T) {
	cases := map[string]struct {
		gen      Generator
		attempts int
		want     error
	}{
		"nil generator":     {gen: nil, attempts: testAttempts, want: ErrNilGenerator},
		"zero attempts":     {gen: &sequenceGenerator{ids: []int64{1}}, attempts: noAttempts, want: ErrInvalidAttempts},
		"negative attempts": {gen: &sequenceGenerator{ids: []int64{1}}, attempts: negativeAttempts, want: ErrInvalidAttempts},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			allocator, err := New(tc.gen, tc.attempts)
			require.ErrorIs(t, err, tc.want)
			assert.Nil(t, allocator)
		})
	}
}

func TestAllocate_RejectsNilEnqueue(t *testing.T) {
	gen := &sequenceGenerator{ids: []int64{101}}
	_, err := mustNew(t, gen, testAttempts).Allocate(testutil.TestContext(), nil)
	require.ErrorIs(t, err, ErrNilEnqueue)
	assert.Zero(t, gen.calls, "no id is drawn without an enqueue to run")
}
