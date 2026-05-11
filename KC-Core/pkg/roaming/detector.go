// Package roaming provides endpoint roaming detection and management.
package roaming

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// ErrEndpointNotFound is returned when an endpoint cannot be found
var ErrEndpointNotFound = errors.New("endpoint not found")

// EndpointOwnershipResolver interface for resolving endpoint ownership
type EndpointOwnershipResolver interface {
	// GetEndpointOwner returns the owner tenant ID for an endpoint
	GetEndpointOwner(ctx context.Context, epEui []byte) (ownerTenantID int64, err error)

	// GetEndpointWithOwnership returns full endpoint data including ownership
	GetEndpointWithOwnership(ctx context.Context, epEui []byte, servingTenantID int64) (*models.EndPoint, error)

	// IsRoamingEnabled checks if roaming is enabled for a tenant
	IsRoamingEnabled(ctx context.Context, tenantID int64) (bool, error)

	// AreTenantsPartners checks if two tenants have a roaming agreement
	AreTenantsPartners(ctx context.Context, tenant1, tenant2 int64) (bool, error)
}

// EventRecorder interface for recording roaming events
type EventRecorder interface {
	// RecordRoamingEvent records a roaming event to the audit trail
	RecordRoamingEvent(ctx context.Context, event *models.RoamingEvent) error
}

// DetectorConfig configuration for the roaming detector
type DetectorConfig struct {
	CacheEnabled     bool
	CacheTTL         time.Duration
	CacheMaxSize     int
	EnableAuditTrail bool
	EnableMetrics    bool
}

// DefaultDetectorConfig returns default configuration
func DefaultDetectorConfig() *DetectorConfig {
	return &DetectorConfig{
		CacheEnabled:     true,
		CacheTTL:         5 * time.Minute,
		CacheMaxSize:     10000,
		EnableAuditTrail: true,
		EnableMetrics:    true,
	}
}

// ownershipCacheEntry represents a cached ownership entry
type ownershipCacheEntry struct {
	OwnerTenantID   int64
	ServingTenantID int64
	IsRoaming       bool
	CachedAt        time.Time
	LastAccessedAt  time.Time
}

// Detector implements roaming detection with caching
type Detector struct {
	config     *DetectorConfig
	resolver   EndpointOwnershipResolver
	recorder   EventRecorder
	cache      map[string]*ownershipCacheEntry // key: hex(epEui)
	cacheMutex sync.RWMutex
	metrics    *Metrics
}

// NewDetector creates a new roaming detector
func NewDetector(config *DetectorConfig, resolver EndpointOwnershipResolver, recorder EventRecorder) *Detector {
	if config == nil {
		config = DefaultDetectorConfig()
	}

	d := &Detector{
		config:   config,
		resolver: resolver,
		recorder: recorder,
		cache:    make(map[string]*ownershipCacheEntry),
	}

	if config.EnableMetrics {
		d.metrics = NewMetrics()
	}

	// Start cache cleanup goroutine
	if config.CacheEnabled && config.CacheTTL > 0 {
		go d.cacheCleanupLoop()
	}

	return d
}

// DetectRoaming determines if an endpoint is roaming
func (d *Detector) DetectRoaming(ctx context.Context, epEui []byte, servingTenantID int64) (isRoaming bool, ownerTenantID int64, err error) {
	epEuiHex := hex.EncodeToString(epEui)

	// Check cache first
	if d.config.CacheEnabled {
		if entry, found := d.getCacheEntry(epEuiHex); found {
			if d.metrics != nil {
				d.metrics.IncrementCacheHit()
			}
			return entry.IsRoaming, entry.OwnerTenantID, nil
		}
		if d.metrics != nil {
			d.metrics.IncrementCacheMiss()
		}
	}

	// Resolve ownership from database
	ownerTenantID, err = d.resolver.GetEndpointOwner(ctx, epEui)
	if err != nil {
		if errors.Is(err, ErrEndpointNotFound) {
			return false, 0, ErrEndpointNotFound
		}
		_ = ctx // Silence unused variable warning
		return false, 0, fmt.Errorf("failed to resolve ownership for endpoint %s: %w", epEuiHex, err)
	}

	// Determine if roaming
	isRoaming = ownerTenantID != servingTenantID

	// Cache the result
	if d.config.CacheEnabled {
		d.setCacheEntry(epEuiHex, &ownershipCacheEntry{
			OwnerTenantID:   ownerTenantID,
			ServingTenantID: servingTenantID,
			IsRoaming:       isRoaming,
			CachedAt:        time.Now(),
			LastAccessedAt:  time.Now(),
		})
	}

	// Record metrics
	if d.metrics != nil {
		if isRoaming {
			d.metrics.IncrementRoamingDetected()
		} else {
			d.metrics.IncrementLocalDetected()
		}
	}

	// Debug log (logger implementation would go here)
	_ = ctx // Silence unused variable warning

	return isRoaming, ownerTenantID, nil
}

// RecordAttachEvent records an endpoint attach event
func (d *Detector) RecordAttachEvent(ctx context.Context, epEui []byte, ownerTenantID, servingTenantID int64, bsEui []byte) error {
	if !d.config.EnableAuditTrail {
		return nil
	}

	event := &models.RoamingEvent{
		EventType:       models.RoamingEventAttach,
		EpEUI:           epEui,
		OwnerTenantID:   ownerTenantID,
		ServingTenantID: servingTenantID,
		ToBsEUI:         bsEui,
		CreatedAt:       time.Now(),
	}

	if ownerTenantID != servingTenantID {
		event.Reason = "Endpoint roaming to foreign network"
	} else {
		event.Reason = "Endpoint attached to home network"
	}

	return d.recorder.RecordRoamingEvent(ctx, event)
}

// RecordDetachEvent records an endpoint detach event
func (d *Detector) RecordDetachEvent(ctx context.Context, epEui []byte, ownerTenantID, servingTenantID int64, bsEui []byte) error {
	if !d.config.EnableAuditTrail {
		return nil
	}

	event := &models.RoamingEvent{
		EventType:       models.RoamingEventDetach,
		EpEUI:           epEui,
		OwnerTenantID:   ownerTenantID,
		ServingTenantID: servingTenantID,
		FromBsEUI:       bsEui,
		CreatedAt:       time.Now(),
	}

	if ownerTenantID != servingTenantID {
		event.Reason = "Endpoint detaching from foreign network"
	} else {
		event.Reason = "Endpoint detached from home network"
	}

	return d.recorder.RecordRoamingEvent(ctx, event)
}

// RecordHandoverEvent records an endpoint handover between base stations
func (d *Detector) RecordHandoverEvent(ctx context.Context, epEui []byte, ownerTenantID, servingTenantID int64, fromBsEui, toBsEui []byte) error {
	if !d.config.EnableAuditTrail {
		return nil
	}

	event := &models.RoamingEvent{
		EventType:       models.RoamingEventHandover,
		EpEUI:           epEui,
		OwnerTenantID:   ownerTenantID,
		ServingTenantID: servingTenantID,
		FromBsEUI:       fromBsEui,
		ToBsEUI:         toBsEui,
		Reason:          "Base station handover",
		CreatedAt:       time.Now(),
	}

	return d.recorder.RecordRoamingEvent(ctx, event)
}

// ValidateRoamingAllowed checks if roaming is allowed between tenants
func (d *Detector) ValidateRoamingAllowed(ctx context.Context, ownerTenantID, servingTenantID int64) error {
	// Same tenant is always allowed
	if ownerTenantID == servingTenantID {
		return nil
	}

	// Check if owner tenant has roaming enabled
	ownerRoamingEnabled, err := d.resolver.IsRoamingEnabled(ctx, ownerTenantID)
	if err != nil {
		return fmt.Errorf("failed to check roaming status for owner tenant %d: %w", ownerTenantID, err)
	}
	if !ownerRoamingEnabled {
		return fmt.Errorf("roaming not enabled for owner tenant %d", ownerTenantID)
	}

	// Check if serving tenant has roaming enabled
	servingRoamingEnabled, err := d.resolver.IsRoamingEnabled(ctx, servingTenantID)
	if err != nil {
		return fmt.Errorf("failed to check roaming status for serving tenant %d: %w", servingTenantID, err)
	}
	if !servingRoamingEnabled {
		return fmt.Errorf("roaming not enabled for serving tenant %d", servingTenantID)
	}

	// Check if tenants are partners
	arePartners, err := d.resolver.AreTenantsPartners(ctx, ownerTenantID, servingTenantID)
	if err != nil {
		return fmt.Errorf("failed to check partnership between tenants: %w", err)
	}
	if !arePartners {
		return fmt.Errorf("no roaming agreement between tenants %d and %d", ownerTenantID, servingTenantID)
	}

	return nil
}

// InvalidateCache invalidates cache entry for an endpoint
func (d *Detector) InvalidateCache(epEui []byte) {
	if !d.config.CacheEnabled {
		return
	}

	epEuiHex := hex.EncodeToString(epEui)
	d.cacheMutex.Lock()
	delete(d.cache, epEuiHex)
	d.cacheMutex.Unlock()
}

// ClearCache clears the entire cache
func (d *Detector) ClearCache() {
	if !d.config.CacheEnabled {
		return
	}

	d.cacheMutex.Lock()
	d.cache = make(map[string]*ownershipCacheEntry)
	d.cacheMutex.Unlock()
}

// GetMetrics returns roaming metrics
func (d *Detector) GetMetrics() *Metrics {
	return d.metrics
}

// getCacheEntry retrieves a cache entry
func (d *Detector) getCacheEntry(epEuiHex string) (*ownershipCacheEntry, bool) {
	d.cacheMutex.RLock()
	defer d.cacheMutex.RUnlock()

	entry, found := d.cache[epEuiHex]
	if !found {
		return nil, false
	}

	// Check if entry is expired
	if time.Since(entry.CachedAt) > d.config.CacheTTL {
		return nil, false
	}

	// Update last accessed time
	entry.LastAccessedAt = time.Now()
	return entry, true
}

// setCacheEntry sets a cache entry
func (d *Detector) setCacheEntry(epEuiHex string, entry *ownershipCacheEntry) {
	d.cacheMutex.Lock()
	defer d.cacheMutex.Unlock()

	// Enforce cache size limit (simple FIFO eviction)
	if len(d.cache) >= d.config.CacheMaxSize {
		// Remove oldest entry
		var oldestKey string
		var oldestTime time.Time
		for k, v := range d.cache {
			if oldestTime.IsZero() || v.LastAccessedAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.LastAccessedAt
			}
		}
		if oldestKey != "" {
			delete(d.cache, oldestKey)
		}
	}

	d.cache[epEuiHex] = entry
}

// cacheCleanupLoop periodically removes expired entries
func (d *Detector) cacheCleanupLoop() {
	ticker := time.NewTicker(d.config.CacheTTL / 2)
	defer ticker.Stop()

	for range ticker.C {
		d.cleanupExpiredEntries()
	}
}

// cleanupExpiredEntries removes expired cache entries
func (d *Detector) cleanupExpiredEntries() {
	d.cacheMutex.Lock()
	defer d.cacheMutex.Unlock()

	now := time.Now()
	for key, entry := range d.cache {
		if now.Sub(entry.CachedAt) > d.config.CacheTTL {
			delete(d.cache, key)
		}
	}
}
