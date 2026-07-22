package bssci

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNextScOpIDMonotonicAllocation verifies the durable-order allocation
// contract (BSSCI rev1 §5.2 / classic §3.2): SC operation IDs are negative,
// strictly decrementing, and a consumed ID is never rolled back regardless of
// what happens to the operation afterwards. A rollback would race concurrent
// allocations and reissue an ID already held by an in-flight operation, so a
// failed operation leaves a harmless gap instead.
func TestNextScOpIDMonotonicAllocation(t *testing.T) {
	tests := []struct {
		name        string
		initialOpId int64
		expected    []int64
	}{
		{
			name:        "FirstAllocationFromZero",
			initialOpId: 0,
			expected:    []int64{-1},
		},
		{
			name:        "SequentialAllocations",
			initialOpId: -5,
			expected:    []int64{-6, -7, -8},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{
				ProtocolSessionState: ProtocolSessionState{
					BaseStationEUI: TestEpEui01,
					LastScOpId:     tt.initialOpId,
				},
			}

			for i, want := range tt.expected {
				got := session.NextScOpID()
				assert.Equal(t, want, got, "allocation %d", i)
				assert.Equal(t, want, session.LastScOpId,
					"counter reflects the consumed ID; it is never restored")
			}
		})
	}
}

// TestNextScOpIDConcurrentAllocationNeverReuses proves the shared-counter
// invariant under concurrency: every allocated ID is unique even when many
// goroutines allocate simultaneously (run with -race). This is the property
// the removed failure-path rollback used to violate: incrementing the counter
// after a concurrent goroutine had taken the next ID reissued a live ID.
func TestNextScOpIDConcurrentAllocationNeverReuses(t *testing.T) {
	const (
		goroutines     = 16
		perGoroutine   = 250
		expectedUnique = goroutines * perGoroutine
	)

	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			BaseStationEUI: TestEpEui01,
		},
	}

	results := make([][]int64, goroutines)
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			ids := make([]int64, 0, perGoroutine)
			for i := 0; i < perGoroutine; i++ {
				ids = append(ids, session.NextScOpID())
			}
			results[g] = ids
		}(g)
	}
	wg.Wait()

	seen := make(map[int64]struct{}, expectedUnique)
	for _, ids := range results {
		for _, id := range ids {
			require.Negative(t, id, "SC operation IDs are negative")
			_, dup := seen[id]
			require.False(t, dup, "operation ID %d allocated twice", id)
			seen[id] = struct{}{}
		}
	}
	require.Len(t, seen, expectedUnique)
	assert.Equal(t, int64(-expectedUnique), session.LastScOpId,
		"counter equals the most negative allocated ID")
}
