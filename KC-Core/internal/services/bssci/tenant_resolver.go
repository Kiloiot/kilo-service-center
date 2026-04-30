package bssciservices

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-DB/storage/interfaces"
)

// tenantResolver implements bssci.TenantResolver interface
//
// This service consolidates queue-to-tenant mapping logic.
// It preserves the two-tier hot/cold path design:
//   - Hot path: In-memory map for recently queued downlinks
//   - Cold path: Database query via DownlinkQueueStore for cache misses
//
// Spec References:
//   - BSSCI §5.12: Queue tenant registration during dlDataQue
//   - BSSCI §5.13: Tenant resolution during dlDataRev
//   - BSSCI §5.14: Tenant resolution during dlDataRes
type tenantResolver struct {
	queueTenants   map[int64]string              // Hot path cache for queue-to-tenant mappings
	queueTenantsMu sync.RWMutex                  // Protects queueTenants map access
	queueStore     interfaces.DownlinkQueueStore // Cold path via repository interface
}

// NewTenantResolver creates a new tenant resolver service
//
// Parameters:
//   - queueStore: Repository interface for cold-path tenant lookups (can be nil for testing)
//
// Returns a TenantResolver that uses a two-tier lookup strategy:
//  1. Check in-memory cache (hot path)
//  2. Query via DownlinkQueueStore if cache miss (cold path)
//  3. Populate cache on successful repository lookup
func NewTenantResolver(queueStore interfaces.DownlinkQueueStore) bssci.TenantResolver {
	return &tenantResolver{
		queueTenants: make(map[int64]string),
		queueStore:   queueStore,
	}
}

// ResolveTenant resolves tenant ID for a queue ID (BSSCI §5.12-§5.14)
//
// Implements two-tier lookup exactly matching old Server.resolveQueueTenant (lines 1349-1374):
//  1. Check in-memory cache (hot path)
//  2. Query repository if cache miss (cold path via DownlinkQueueStore)
//  3. Populate cache on successful repository lookup
//
// This method is thread-safe and can be called concurrently from multiple handlers.
//
// Returns:
//   - tenantID string if queue found (either in cache or repository)
//   - error if queueID invalid (<=0) or not found in cache/repository
func (r *tenantResolver) ResolveTenant(ctx context.Context, queueID int64) (string, error) {
	if queueID <= 0 {
		return "", fmt.Errorf("invalid queue ID: %d", queueID)
	}

	// Hot path: Check cache
	r.queueTenantsMu.RLock()
	if tidStr, exists := r.queueTenants[queueID]; exists {
		r.queueTenantsMu.RUnlock()
		return tidStr, nil
	}
	r.queueTenantsMu.RUnlock()

	// Cold path: Repository lookup via DownlinkQueueStore interface
	if r.queueStore == nil {
		return "", fmt.Errorf("queue store not available for tenant resolution")
	}

	tenantID, err := r.queueStore.GetTenantIDByQueueID(ctx, uint64(queueID))
	if err != nil {
		return "", fmt.Errorf("cannot resolve tenant for queue %d: %w", queueID, err)
	}

	tidStr := strconv.FormatInt(tenantID, 10)

	// Populate cache for future lookups
	r.queueTenantsMu.Lock()
	r.queueTenants[queueID] = tidStr
	r.queueTenantsMu.Unlock()

	return tidStr, nil
}

// RegisterQueueTenant registers queue-to-tenant mapping (BSSCI §5.12)
//
// Called by DownlinkService.SendDLDataQueue after successful message send to populate
// the hot-path cache for fast lookup during result processing.
//
// Thread-safe for concurrent access.
func (r *tenantResolver) RegisterQueueTenant(queueID int64, tenantID string) {
	r.queueTenantsMu.Lock()
	defer r.queueTenantsMu.Unlock()
	r.queueTenants[queueID] = tenantID
}

// UnregisterQueueTenant removes queue-to-tenant mapping (BSSCI §5.14)
//
// Called by DownlinkService when queue processing completes to prevent unbounded cache growth.
//
// Thread-safe for concurrent access.
func (r *tenantResolver) UnregisterQueueTenant(queueID int64) {
	r.queueTenantsMu.Lock()
	defer r.queueTenantsMu.Unlock()
	delete(r.queueTenants, queueID)
}
