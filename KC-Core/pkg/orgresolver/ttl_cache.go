package orgresolver

import (
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// ttlCache is a bounded map whose entries expire ttl after they were stored;
// a full cache is cleared before the next entry is stored.
type ttlCache[K comparable, V any] struct {
	mu         sync.RWMutex
	items      map[K]ttlEntry[V]
	ttl        time.Duration
	maxEntries int
	clock      clock.Clock
}

type ttlEntry[V any] struct {
	value    V
	cachedAt time.Time
}

func newTTLCache[K comparable, V any](ttl time.Duration, maxEntries int, clk clock.Clock) *ttlCache[K, V] {
	return &ttlCache[K, V]{items: make(map[K]ttlEntry[V]), ttl: ttl, maxEntries: maxEntries, clock: clk}
}

// get returns the value stored for key and its age while it is fresh.
func (c *ttlCache[K, V]) get(key K) (V, time.Duration, bool) {
	c.mu.RLock()
	entry, found := c.items[key]
	c.mu.RUnlock()

	age := c.clock.Now().Sub(entry.cachedAt)
	if !found || age >= c.ttl {
		var zero V
		return zero, 0, false
	}
	return entry.value, age, true
}

// put stores value for key and reports how many entries a full cache dropped
// first and how many entries the cache holds afterwards.
func (c *ttlCache[K, V]) put(key K, value V) (dropped, size int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.items) >= c.maxEntries {
		dropped = len(c.items)
		c.items = make(map[K]ttlEntry[V])
	}
	c.items[key] = ttlEntry[V]{value: value, cachedAt: c.clock.Now()}
	return dropped, len(c.items)
}
