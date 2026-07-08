package adapters

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	eventsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/events"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// countCacheMaxEntries bounds the cache; expired keys are purged past this size.
const countCacheMaxEntries = 1024

// CachedSystemEventStore caches CountEvents (the heavy COUNT(*)) with a short TTL
// and collapses concurrent identical counts via singleflight; GetEvents passes through.
type CachedSystemEventStore struct {
	inner eventsservice.SystemEventStore
	sf    singleflight.Group
	ttl   time.Duration
	now   func() time.Time

	mu    sync.Mutex
	items map[string]countEntry
}

type countEntry struct {
	val int64
	exp time.Time
}

// NewCachedSystemEventStore wraps inner with a count cache of the given TTL.
func NewCachedSystemEventStore(inner eventsservice.SystemEventStore, ttl time.Duration) *CachedSystemEventStore {
	return &CachedSystemEventStore{
		inner: inner,
		ttl:   ttl,
		now:   time.Now,
		items: make(map[string]countEntry),
	}
}

// GetEvents passes through — row pages are never cached.
func (c *CachedSystemEventStore) GetEvents(ctx context.Context, tenantID int64, filter *eventsservice.EventFilter, limit, offset int) ([]*models.SystemEvent, error) {
	return c.inner.GetEvents(ctx, tenantID, filter, limit, offset)
}

// CountEvents serves a cached total when fresh, else counts once (shared) and caches it.
func (c *CachedSystemEventStore) CountEvents(ctx context.Context, tenantID int64, filter *eventsservice.EventFilter) (int64, error) {
	if c.ttl <= 0 {
		return c.inner.CountEvents(ctx, tenantID, filter)
	}

	key := c.countKey(tenantID, filter)
	if v, ok := c.get(key); ok {
		return v, nil
	}

	// singleflight shares one in-flight count across identical concurrent callers.
	v, err, _ := c.sf.Do(key, func() (interface{}, error) {
		if v, ok := c.get(key); ok {
			return v, nil
		}
		n, err := c.inner.CountEvents(ctx, tenantID, filter)
		if err != nil {
			return int64(0), err // never cache errors
		}
		c.set(key, n)
		return n, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int64), nil
}

func (c *CachedSystemEventStore) get(key string) (int64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok || c.now().After(e.exp) {
		return 0, false
	}
	return e.val, true
}

func (c *CachedSystemEventStore) set(key string, val int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if len(c.items) >= countCacheMaxEntries {
		for k, e := range c.items {
			if now.After(e.exp) {
				delete(c.items, k)
			}
		}
	}
	c.items[key] = countEntry{val: val, exp: now.Add(c.ttl)}
}

// countKey mirrors CountEvents' filter (limit/offset excluded — they don't affect a
// count); time bounds are bucketed to the TTL so a moving window doesn't fragment it.
func (c *CachedSystemEventStore) countKey(tenantID int64, f *eventsservice.EventFilter) string {
	var b strings.Builder
	b.WriteString(strconv.FormatInt(tenantID, 10))
	if f == nil {
		return b.String()
	}
	b.WriteString("|cat=")
	b.WriteString(joinSorted(f.Categories))
	b.WriteString("|sev=")
	b.WriteString(joinSorted(f.Severity))
	b.WriteString("|et=")
	b.WriteString(joinSorted(f.EventTypes))
	b.WriteString("|since=")
	b.WriteString(c.bucketTime(f.StartTime))
	b.WriteString("|until=")
	b.WriteString(c.bucketTime(f.EndTime))
	if f.BaseStationID != nil {
		b.WriteString("|bs=")
		b.WriteString(strconv.FormatInt(*f.BaseStationID, 10))
	}
	if f.EndpointID != nil {
		b.WriteString("|ep=")
		b.WriteString(strconv.FormatInt(*f.EndpointID, 10))
	}
	b.WriteString("|bseui=")
	b.WriteString(strings.ToLower(f.BaseStationEUI))
	b.WriteString("|epeui=")
	b.WriteString(strings.ToLower(f.EndpointEUI))
	return b.String()
}

func (c *CachedSystemEventStore) bucketTime(t *int64) string {
	if t == nil {
		return ""
	}
	secs := int64(c.ttl / time.Second)
	if secs <= 0 {
		secs = 1
	}
	return strconv.FormatInt((*t/secs)*secs, 10)
}

func joinSorted(s []string) string {
	if len(s) == 0 {
		return ""
	}
	cp := append([]string(nil), s...)
	sort.Strings(cp)
	return strings.Join(cp, ",")
}
