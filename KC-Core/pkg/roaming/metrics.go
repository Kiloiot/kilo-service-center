// Package roaming provides endpoint roaming detection metrics.
package roaming

import (
	"sync/atomic"
)

// Metrics tracks roaming detection metrics
type Metrics struct {
	// Cache metrics
	cacheHits   uint64
	cacheMisses uint64

	// Detection metrics
	roamingDetected uint64
	localDetected   uint64

	// Event metrics
	attachEvents   uint64
	detachEvents   uint64
	handoverEvents uint64

	// Error metrics
	detectionErrors uint64
	recordingErrors uint64
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{}
}

// IncrementCacheHit increments cache hit counter
func (m *Metrics) IncrementCacheHit() {
	atomic.AddUint64(&m.cacheHits, 1)
}

// IncrementCacheMiss increments cache miss counter
func (m *Metrics) IncrementCacheMiss() {
	atomic.AddUint64(&m.cacheMisses, 1)
}

// IncrementRoamingDetected increments roaming detection counter
func (m *Metrics) IncrementRoamingDetected() {
	atomic.AddUint64(&m.roamingDetected, 1)
}

// IncrementLocalDetected increments local (non-roaming) detection counter
func (m *Metrics) IncrementLocalDetected() {
	atomic.AddUint64(&m.localDetected, 1)
}

// IncrementAttachEvents increments attach event counter
func (m *Metrics) IncrementAttachEvents() {
	atomic.AddUint64(&m.attachEvents, 1)
}

// IncrementDetachEvents increments detach event counter
func (m *Metrics) IncrementDetachEvents() {
	atomic.AddUint64(&m.detachEvents, 1)
}

// IncrementHandoverEvents increments handover event counter
func (m *Metrics) IncrementHandoverEvents() {
	atomic.AddUint64(&m.handoverEvents, 1)
}

// IncrementDetectionErrors increments detection error counter
func (m *Metrics) IncrementDetectionErrors() {
	atomic.AddUint64(&m.detectionErrors, 1)
}

// IncrementRecordingErrors increments recording error counter
func (m *Metrics) IncrementRecordingErrors() {
	atomic.AddUint64(&m.recordingErrors, 1)
}

// GetCacheHits returns cache hit count
func (m *Metrics) GetCacheHits() uint64 {
	return atomic.LoadUint64(&m.cacheHits)
}

// GetCacheMisses returns cache miss count
func (m *Metrics) GetCacheMisses() uint64 {
	return atomic.LoadUint64(&m.cacheMisses)
}

// GetCacheHitRate returns cache hit rate as a percentage
func (m *Metrics) GetCacheHitRate() float64 {
	hits := atomic.LoadUint64(&m.cacheHits)
	misses := atomic.LoadUint64(&m.cacheMisses)
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

// GetRoamingDetected returns roaming detection count
func (m *Metrics) GetRoamingDetected() uint64 {
	return atomic.LoadUint64(&m.roamingDetected)
}

// GetLocalDetected returns local detection count
func (m *Metrics) GetLocalDetected() uint64 {
	return atomic.LoadUint64(&m.localDetected)
}

// GetRoamingRate returns roaming rate as a percentage
func (m *Metrics) GetRoamingRate() float64 {
	roaming := atomic.LoadUint64(&m.roamingDetected)
	local := atomic.LoadUint64(&m.localDetected)
	total := roaming + local
	if total == 0 {
		return 0
	}
	return float64(roaming) / float64(total) * 100
}

// GetAttachEvents returns attach event count
func (m *Metrics) GetAttachEvents() uint64 {
	return atomic.LoadUint64(&m.attachEvents)
}

// GetDetachEvents returns detach event count
func (m *Metrics) GetDetachEvents() uint64 {
	return atomic.LoadUint64(&m.detachEvents)
}

// GetHandoverEvents returns handover event count
func (m *Metrics) GetHandoverEvents() uint64 {
	return atomic.LoadUint64(&m.handoverEvents)
}

// GetDetectionErrors returns detection error count
func (m *Metrics) GetDetectionErrors() uint64 {
	return atomic.LoadUint64(&m.detectionErrors)
}

// GetRecordingErrors returns recording error count
func (m *Metrics) GetRecordingErrors() uint64 {
	return atomic.LoadUint64(&m.recordingErrors)
}

// Reset resets all metrics to zero
func (m *Metrics) Reset() {
	atomic.StoreUint64(&m.cacheHits, 0)
	atomic.StoreUint64(&m.cacheMisses, 0)
	atomic.StoreUint64(&m.roamingDetected, 0)
	atomic.StoreUint64(&m.localDetected, 0)
	atomic.StoreUint64(&m.attachEvents, 0)
	atomic.StoreUint64(&m.detachEvents, 0)
	atomic.StoreUint64(&m.handoverEvents, 0)
	atomic.StoreUint64(&m.detectionErrors, 0)
	atomic.StoreUint64(&m.recordingErrors, 0)
}

// MetricsSnapshot returns a snapshot of all metrics
type MetricsSnapshot struct {
	CacheHits       uint64  `json:"cacheHits"`
	CacheMisses     uint64  `json:"cacheMisses"`
	CacheHitRate    float64 `json:"cacheHitRate"`
	RoamingDetected uint64  `json:"roamingDetected"`
	LocalDetected   uint64  `json:"localDetected"`
	RoamingRate     float64 `json:"roamingRate"`
	AttachEvents    uint64  `json:"attachEvents"`
	DetachEvents    uint64  `json:"detachEvents"`
	HandoverEvents  uint64  `json:"handoverEvents"`
	DetectionErrors uint64  `json:"detectionErrors"`
	RecordingErrors uint64  `json:"recordingErrors"`
}

// GetSnapshot returns a snapshot of current metrics
func (m *Metrics) GetSnapshot() *MetricsSnapshot {
	return &MetricsSnapshot{
		CacheHits:       m.GetCacheHits(),
		CacheMisses:     m.GetCacheMisses(),
		CacheHitRate:    m.GetCacheHitRate(),
		RoamingDetected: m.GetRoamingDetected(),
		LocalDetected:   m.GetLocalDetected(),
		RoamingRate:     m.GetRoamingRate(),
		AttachEvents:    m.GetAttachEvents(),
		DetachEvents:    m.GetDetachEvents(),
		HandoverEvents:  m.GetHandoverEvents(),
		DetectionErrors: m.GetDetectionErrors(),
		RecordingErrors: m.GetRecordingErrors(),
	}
}
