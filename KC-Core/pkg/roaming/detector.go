// Package roaming provides endpoint roaming detection and management.
package roaming

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// ErrEndpointNotFound is returned when an endpoint cannot be found
var ErrEndpointNotFound = errors.New("endpoint not found")

// EndpointOwnershipResolver interface for resolving endpoint ownership
type EndpointOwnershipResolver interface {
	// GetEndpointOwner returns the owner tenant ID for an endpoint
	GetEndpointOwner(ctx context.Context, epEui []byte) (ownerTenantID int64, err error)

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

// DetectorConfig configures the roaming detector from protocol.roaming.
type DetectorConfig struct {
	CacheEnabled     bool
	CacheTTL         time.Duration
	CacheMaxSize     int
	EnableAuditTrail bool
}

// Roaming event reasons recorded on attach/detach transitions.
const (
	reasonRoamingToForeign    = "Endpoint roaming to foreign network"
	reasonAttachedHome        = "Endpoint attached to home network"
	reasonDetachingForeign    = "Endpoint detaching from foreign network"
	reasonDetachedHome        = "Endpoint detached from home network"
	reasonBaseStationHandover = "Base station handover"
)

// cacheSweepsPerTTL sets the cleanup cadence relative to the cache TTL.
const cacheSweepsPerTTL = 2

// ownershipCacheEntry caches an endpoint's owner only; roaming depends on the hearing station and is decided per call.
type ownershipCacheEntry struct {
	OwnerTenantID  int64
	CachedAt       time.Time
	LastAccessedAt time.Time
}

// Detector implements roaming detection with caching
type Detector struct {
	clock      clock.Clock
	config     DetectorConfig
	resolver   EndpointOwnershipResolver
	recorder   EventRecorder
	cache      map[string]*ownershipCacheEntry // key: hex(epEui)
	cacheMutex sync.RWMutex
}

// NewDetector creates a new roaming detector and rejects a missing collaborator.
func NewDetector(config DetectorConfig, resolver EndpointOwnershipResolver, recorder EventRecorder, clk clock.Clock) (*Detector, error) {
	if resolver == nil || recorder == nil || clk == nil {
		return nil, errMissingDetectorDependency
	}

	d := &Detector{
		clock:    clk,
		config:   config,
		resolver: resolver,
		recorder: recorder,
		cache:    make(map[string]*ownershipCacheEntry),
	}

	// Start cache cleanup goroutine
	if config.CacheEnabled && config.CacheTTL > 0 {
		go d.cacheCleanupLoop()
	}

	return d, nil
}

// DetectRoaming compares the endpoint's owner with the serving tenant of this reception.
func (d *Detector) DetectRoaming(ctx context.Context, epEui []byte, servingTenantID int64) (isRoaming bool, ownerTenantID int64, err error) {
	ownerTenantID, err = d.owner(ctx, epEui)
	if err != nil {
		return false, 0, err
	}
	return ownerTenantID != servingTenantID, ownerTenantID, nil
}

// owner resolves the endpoint's owner tenant, from the cache when enabled.
func (d *Detector) owner(ctx context.Context, epEui []byte) (int64, error) {
	epEuiHex := mioty.FormatEUIBytes(epEui)
	if d.config.CacheEnabled {
		if entry, found := d.getCacheEntry(epEuiHex); found {
			return entry.OwnerTenantID, nil
		}
	}

	ownerTenantID, err := d.resolver.GetEndpointOwner(ctx, epEui)
	if err != nil {
		if errors.Is(err, ErrEndpointNotFound) {
			return 0, ErrEndpointNotFound
		}
		return 0, fmt.Errorf(errFmtResolveOwnershipForEndpoint, epEuiHex, err)
	}

	if d.config.CacheEnabled {
		now := d.clock.Now()
		d.setCacheEntry(epEuiHex, &ownershipCacheEntry{
			OwnerTenantID:  ownerTenantID,
			CachedAt:       now,
			LastAccessedAt: now,
		})
	}
	return ownerTenantID, nil
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
		CreatedAt:       d.clock.Now(),
	}

	if ownerTenantID != servingTenantID {
		event.Reason = reasonRoamingToForeign
	} else {
		event.Reason = reasonAttachedHome
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
		CreatedAt:       d.clock.Now(),
	}

	if ownerTenantID != servingTenantID {
		event.Reason = reasonDetachingForeign
	} else {
		event.Reason = reasonDetachedHome
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
		Reason:          reasonBaseStationHandover,
		CreatedAt:       d.clock.Now(),
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
		return fmt.Errorf(errFmtCheckRoamingStatusForOwnerTenant, ownerTenantID, err)
	}
	if !ownerRoamingEnabled {
		return fmt.Errorf(errFmtRoamingNotEnabledForOwnerTenant, ownerTenantID)
	}

	// Check if serving tenant has roaming enabled
	servingRoamingEnabled, err := d.resolver.IsRoamingEnabled(ctx, servingTenantID)
	if err != nil {
		return fmt.Errorf(errFmtCheckRoamingStatusForServingTenant, servingTenantID, err)
	}
	if !servingRoamingEnabled {
		return fmt.Errorf(errFmtRoamingNotEnabledForServingTenant, servingTenantID)
	}

	// Check if tenants are partners
	arePartners, err := d.resolver.AreTenantsPartners(ctx, ownerTenantID, servingTenantID)
	if err != nil {
		return fmt.Errorf(errFmtCheckPartnershipBetweenTenants, err)
	}
	if !arePartners {
		return fmt.Errorf(errFmtNoRoamingAgreement, ownerTenantID, servingTenantID)
	}

	return nil
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
	if d.clock.Now().Sub(entry.CachedAt) > d.config.CacheTTL {
		return nil, false
	}

	// Update last accessed time
	entry.LastAccessedAt = d.clock.Now()
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
	ticker := time.NewTicker(d.config.CacheTTL / cacheSweepsPerTTL)
	defer ticker.Stop()

	for range ticker.C {
		d.cleanupExpiredEntries()
	}
}

// cleanupExpiredEntries removes expired cache entries
func (d *Detector) cleanupExpiredEntries() {
	d.cacheMutex.Lock()
	defer d.cacheMutex.Unlock()

	now := d.clock.Now()
	for key, entry := range d.cache {
		if now.Sub(entry.CachedAt) > d.config.CacheTTL {
			delete(d.cache, key)
		}
	}
}
