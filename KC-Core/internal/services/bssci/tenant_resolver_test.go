package bssciservices

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDownlinkQueueStore implements interfaces.DownlinkQueueStore for testing
type mockDownlinkQueueStore struct {
	tenants map[uint64]int64 // queueID → tenantID
	err     error            // Error to return
}

func newMockDownlinkQueueStore() *mockDownlinkQueueStore {
	return &mockDownlinkQueueStore{
		tenants: make(map[uint64]int64),
	}
}

func (m *mockDownlinkQueueStore) GetTenantIDByQueueID(_ context.Context, queueID uint64) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	if tid, ok := m.tenants[queueID]; ok {
		return tid, nil
	}
	return 0, fmt.Errorf("queue ID not found: %d", queueID)
}

// TestResolveTenant_HotPath tests the hot-path cache hit scenario
func TestResolveTenant_HotPath(t *testing.T) {
	// Setup: Create resolver and seed cache
	mockStore := newMockDownlinkQueueStore()
	resolver := NewTenantResolver(mockStore)
	resolver.RegisterQueueTenant(42, "100")

	// Execute: Resolve tenant from cache
	ctx := testutil.TestContext()
	tenantID, err := resolver.ResolveTenant(ctx, 42)

	// Verify: Cache hit returns cached value (no store call)
	require.NoError(t, err)
	assert.Equal(t, "100", tenantID)
}

// TestResolveTenant_ColdPath tests cache miss with successful repository lookup
func TestResolveTenant_ColdPath(t *testing.T) {
	// Setup: Create mock store with queue mapping
	mockStore := newMockDownlinkQueueStore()
	mockStore.tenants[42] = 200 // Queue 42 belongs to tenant 200
	resolver := NewTenantResolver(mockStore)

	// Execute: First call misses cache, queries store
	ctx := testutil.TestContext()
	tenantID, err := resolver.ResolveTenant(ctx, 42)

	// Verify: Repository lookup succeeds and populates cache
	require.NoError(t, err)
	assert.Equal(t, "200", tenantID)

	// Execute: Second call hits cache (no store query needed)
	// Remove from mock store to prove cache is used
	delete(mockStore.tenants, 42)
	tenantID2, err2 := resolver.ResolveTenant(ctx, 42)

	// Verify: Cache hit returns same value without store query
	require.NoError(t, err2)
	assert.Equal(t, "200", tenantID2)
}

// TestResolveTenant_DBNotFound tests cache miss with queue not in repository
func TestResolveTenant_DBNotFound(t *testing.T) {
	// Setup: Mock store with no queues
	mockStore := newMockDownlinkQueueStore()
	resolver := NewTenantResolver(mockStore)

	// Execute: Resolve unknown queue
	ctx := testutil.TestContext()
	tenantID, err := resolver.ResolveTenant(ctx, 999)

	// Verify: Returns error for not found
	require.Error(t, err)
	assert.Empty(t, tenantID)
	assert.Contains(t, err.Error(), "queue ID not found: 999")
}

// TestResolveTenant_InvalidQueueID tests validation of queueID parameter
func TestResolveTenant_InvalidQueueID(t *testing.T) {
	mockStore := newMockDownlinkQueueStore()
	resolver := NewTenantResolver(mockStore)
	ctx := testutil.TestContext()

	// Test zero queue ID
	tenantID, err := resolver.ResolveTenant(ctx, 0)
	require.Error(t, err)
	assert.Empty(t, tenantID)
	assert.Contains(t, err.Error(), "invalid queue ID: 0")

	// Test negative queue ID
	tenantID, err = resolver.ResolveTenant(ctx, -5)
	require.Error(t, err)
	assert.Empty(t, tenantID)
	assert.Contains(t, err.Error(), "invalid queue ID: -5")
}

// TestResolveTenant_NilStore tests error when store is nil and cache misses
func TestResolveTenant_NilStore(t *testing.T) {
	resolver := NewTenantResolver(nil)
	ctx := testutil.TestContext()

	// Execute: Resolve queue not in cache with nil store
	tenantID, err := resolver.ResolveTenant(ctx, 42)

	// Verify: Returns error about unavailable store
	require.Error(t, err)
	assert.Empty(t, tenantID)
	assert.Contains(t, err.Error(), "queue store not available")
}

// TestRegisterQueueTenant tests cache population
func TestRegisterQueueTenant(t *testing.T) {
	mockStore := newMockDownlinkQueueStore()
	resolver := NewTenantResolver(mockStore)
	ctx := testutil.TestContext()

	// Execute: Register queue-to-tenant mapping
	resolver.RegisterQueueTenant(42, "300")

	// Verify: Can resolve from cache (no store call needed)
	tenantID, err := resolver.ResolveTenant(ctx, 42)
	require.NoError(t, err)
	assert.Equal(t, "300", tenantID)
}

// TestUnregisterQueueTenant tests cache removal
func TestUnregisterQueueTenant(t *testing.T) {
	// Setup: Mock store with queue mapping
	mockStore := newMockDownlinkQueueStore()
	mockStore.tenants[42] = 400
	resolver := NewTenantResolver(mockStore)
	ctx := testutil.TestContext()

	// Setup: Register and verify cache hit
	resolver.RegisterQueueTenant(42, "400")
	tenantID, err := resolver.ResolveTenant(ctx, 42)
	require.NoError(t, err)
	assert.Equal(t, "400", tenantID)

	// Execute: Unregister (remove from cache)
	resolver.UnregisterQueueTenant(42)

	// Verify: Now triggers cold path (store query returns different value)
	mockStore.tenants[42] = 500 // Store has updated value
	tenantID2, err2 := resolver.ResolveTenant(ctx, 42)
	require.NoError(t, err2)
	assert.Equal(t, "500", tenantID2) // Store returns different value, proving cold path
}

// TestTenantResolver_ThreadSafety tests concurrent access to cache
func TestTenantResolver_ThreadSafety(_ *testing.T) {
	mockStore := newMockDownlinkQueueStore()
	resolver := NewTenantResolver(mockStore)
	ctx := testutil.TestContext()

	// Setup: Pre-populate cache with some entries
	for i := int64(1); i <= 10; i++ {
		resolver.RegisterQueueTenant(i, string(rune('A'+i)))
	}

	var wg sync.WaitGroup
	numGoroutines := 50

	// Execute: Concurrent Register/Resolve/Unregister operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			queueID := int64((id % 20) + 1)

			// Mix of operations
			switch id % 3 {
			case 0:
				resolver.RegisterQueueTenant(queueID, "concurrent")
			case 1:
				_, _ = resolver.ResolveTenant(ctx, queueID)
			case 2:
				resolver.UnregisterQueueTenant(queueID)
			}
		}(i)
	}

	// Verify: No data races (test with -race flag)
	wg.Wait()
}

// TestTenantResolver_Lifecycle tests complete register→resolve→unregister flow
func TestTenantResolver_Lifecycle(t *testing.T) {
	mockStore := newMockDownlinkQueueStore()
	resolver := NewTenantResolver(mockStore)
	ctx := testutil.TestContext()

	// Step 1: Register (simulates SendDLDataQueue)
	resolver.RegisterQueueTenant(42, "600")

	// Step 2: Resolve (simulates handleDLDataResult)
	tenantID, err := resolver.ResolveTenant(ctx, 42)
	require.NoError(t, err)
	assert.Equal(t, "600", tenantID)

	// Step 3: Unregister (simulates handleDLDataResultComplete)
	resolver.UnregisterQueueTenant(42)

	// Step 4: Verify cache is empty (triggers cold path)
	// Since mockStore doesn't have queue 42, it should return not found
	tenantID2, err2 := resolver.ResolveTenant(ctx, 42)
	require.Error(t, err2)
	assert.Empty(t, tenantID2)
	assert.Contains(t, err2.Error(), "queue ID not found")
}
