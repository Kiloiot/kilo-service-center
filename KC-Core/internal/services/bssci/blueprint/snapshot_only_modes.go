package blueprint

import (
	"context"
	"sync"
	"time"
)

// SnapshotOnlyModeCacheTTL bounds how long a decode keeps using a tenant's switch after it changes.
const SnapshotOnlyModeCacheTTL = 60 * time.Second

// SnapshotOnlyModeSource reads a tenant's snapshot-only decoding switch from storage.
type SnapshotOnlyModeSource interface {
	GetSnapshotOnlyDecoding(ctx context.Context, tenantID int64) (bool, error)
}

type cachedMode struct {
	on     bool
	readAt time.Time
}

// CachedSnapshotOnlyModes serves the decode path the tenant switch, reading
// storage at most once per SnapshotOnlyModeCacheTTL per tenant; failed reads
// are not kept.
type CachedSnapshotOnlyModes struct {
	source  SnapshotOnlyModeSource
	now     func() time.Time
	mu      sync.Mutex
	entries map[int64]cachedMode
}

// NewCachedSnapshotOnlyModes reads the switch from source; now is the cache clock.
func NewCachedSnapshotOnlyModes(source SnapshotOnlyModeSource, now func() time.Time) *CachedSnapshotOnlyModes {
	return &CachedSnapshotOnlyModes{source: source, now: now, entries: map[int64]cachedMode{}}
}

// IsSnapshotOnly reports the tenant's switch.
func (c *CachedSnapshotOnlyModes) IsSnapshotOnly(ctx context.Context, tenantID int64) (bool, error) {
	now := c.now()
	c.mu.Lock()
	entry, ok := c.entries[tenantID]
	c.mu.Unlock()
	if ok && now.Sub(entry.readAt) < SnapshotOnlyModeCacheTTL {
		return entry.on, nil
	}
	on, err := c.source.GetSnapshotOnlyDecoding(ctx, tenantID)
	if err != nil {
		return false, err
	}
	c.mu.Lock()
	c.entries[tenantID] = cachedMode{on: on, readAt: now}
	c.mu.Unlock()
	return on, nil
}
