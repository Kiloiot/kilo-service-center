package adapters

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	eventsservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/events"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// countCacheMaxEntries bounds the cache; expired keys are purged past this size.
const countCacheMaxEntries = 1024

// CachedSystemEventStore caches CountEvents (the heavy COUNT(*)) with a short TTL
// and collapses concurrent identical counts via singleflight; GetEvents passes through.
type CachedSystemEventStore struct {
	inner   eventsservice.SystemEventStore
	sf      singleflight.Group
	ttl     time.Duration
	timeout time.Duration
	clock   clock.Clock

	mu    sync.Mutex
	items map[string]countEntry
}

// Construction faults: the cache needs the store it wraps and a clock.
var (
	ErrNilCountedStore = errors.New("count cache: event store is nil")
	ErrNilCountClock   = errors.New("count cache: clock is nil")
)

type countEntry struct {
	val int64
	exp time.Time
}

const (
	keyPartCategories     = "|cat="
	keyPartSeverity       = "|sev="
	keyPartEventTypes     = "|et="
	keyPartSince          = "|since="
	keyPartUntil          = "|until="
	keyPartBaseStationID  = "|bs="
	keyPartEndpointID     = "|ep="
	keyPartBaseStationEUI = "|bseui="
	keyPartEndpointEUI    = "|epeui="
	keyPartOpID           = "|op="
	keyPartSearch         = "|q="
)

// NewCachedSystemEventStore wraps inner; a shared count is detached from its callers, so timeout bounds it.
func NewCachedSystemEventStore(inner eventsservice.SystemEventStore, ttl, timeout time.Duration, clk clock.Clock) (*CachedSystemEventStore, error) {
	if inner == nil {
		return nil, ErrNilCountedStore
	}
	if clk == nil {
		return nil, ErrNilCountClock
	}
	return &CachedSystemEventStore{
		inner:   inner,
		ttl:     ttl,
		timeout: timeout,
		clock:   clk,
		items:   make(map[string]countEntry),
	}, nil
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

	// Detached so one caller giving up never cancels the count the others wait on.
	shared := context.WithoutCancel(ctx)
	result := c.sf.DoChan(key, func() (interface{}, error) {
		return c.count(shared, key, tenantID, filter)
	})
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case res := <-result:
		if res.Err != nil {
			return 0, res.Err
		}
		return res.Val.(int64), nil
	}
}

// count runs one shared COUNT(*) under the cache's own timeout; errors are never cached.
func (c *CachedSystemEventStore) count(ctx context.Context, key string, tenantID int64, filter *eventsservice.EventFilter) (int64, error) {
	if v, ok := c.get(key); ok {
		return v, nil
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	n, err := c.inner.CountEvents(ctx, tenantID, filter)
	if err != nil {
		return 0, err
	}
	c.set(key, n)
	return n, nil
}

func (c *CachedSystemEventStore) get(key string) (int64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok || c.clock.Now().After(e.exp) {
		return 0, false
	}
	return e.val, true
}

func (c *CachedSystemEventStore) set(key string, val int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.clock.Now()
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
	b.WriteString(keyPartCategories)
	b.WriteString(joinSorted(f.Categories))
	b.WriteString(keyPartSeverity)
	b.WriteString(joinSorted(f.Severity))
	b.WriteString(keyPartEventTypes)
	b.WriteString(joinSorted(f.EventTypes))
	b.WriteString(keyPartSince)
	b.WriteString(c.bucketTime(f.StartTime))
	b.WriteString(keyPartUntil)
	b.WriteString(c.bucketTime(f.EndTime))
	if f.BaseStationID != nil {
		b.WriteString(keyPartBaseStationID)
		b.WriteString(strconv.FormatInt(*f.BaseStationID, 10))
	}
	if f.EndpointID != nil {
		b.WriteString(keyPartEndpointID)
		b.WriteString(strconv.FormatInt(*f.EndpointID, 10))
	}
	b.WriteString(keyPartBaseStationEUI)
	b.WriteString(strings.ToLower(f.BaseStationEUI))
	b.WriteString(keyPartEndpointEUI)
	b.WriteString(strings.ToLower(f.EndpointEUI))
	if f.OpID != nil {
		b.WriteString(keyPartOpID)
		b.WriteString(strconv.FormatInt(*f.OpID, 10))
	}
	b.WriteString(keyPartSearch)
	b.WriteString(f.Search)
	return b.String()
}

func (c *CachedSystemEventStore) bucketTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	secs := int64(c.ttl / time.Second)
	if secs <= 0 {
		secs = 1
	}
	return strconv.FormatInt((t.Unix()/secs)*secs, 10)
}

func joinSorted(s []string) string {
	if len(s) == 0 {
		return ""
	}
	cp := append([]string(nil), s...)
	sort.Strings(cp)
	return strings.Join(cp, ",")
}
