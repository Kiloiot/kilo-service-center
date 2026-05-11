package postgres

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// ArchivalScheduler manages scheduled archival operations
type ArchivalScheduler struct {
	service *ArchivalService
	logger  logger.Logger
	config  ArchivalConfig

	// Control channels
	stopCh chan struct{}
	doneCh chan struct{}
	mu     sync.Mutex

	// Status tracking
	running bool
	lastRun map[string]time.Time
}

// ArchivalConfig defines configuration for archival operations
type ArchivalConfig struct {
	// Message archival settings
	MessageRetentionDays int
	MessageArchivalHour  int // Hour of day to run (0-23)

	// Gateway reception archival settings
	GatewayReceptionRetentionDays int
	GatewayReceptionArchivalHour  int

	// Device session archival settings
	DeviceSessionRetentionDays int
	DeviceSessionArchivalHour  int

	// Device key archival settings
	DeviceKeyRetentionDays int
	DeviceKeyArchivalHour  int

	// General settings
	ArchivalEnabled bool
	CheckInterval   time.Duration // How often to check if archival should run
}

// DefaultArchivalConfig returns default archival configuration
func DefaultArchivalConfig() ArchivalConfig {
	return ArchivalConfig{
		MessageRetentionDays:          90,
		MessageArchivalHour:           2, // 2 AM
		GatewayReceptionRetentionDays: 30,
		GatewayReceptionArchivalHour:  3, // 3 AM
		DeviceSessionRetentionDays:    180,
		DeviceSessionArchivalHour:     4, // 4 AM
		DeviceKeyRetentionDays:        365,
		DeviceKeyArchivalHour:         5, // 5 AM
		ArchivalEnabled:               true,
		CheckInterval:                 15 * time.Minute,
	}
}

// NewArchivalScheduler creates a new archival scheduler
func NewArchivalScheduler(service *ArchivalService, logger logger.Logger, config ArchivalConfig) *ArchivalScheduler {
	return &ArchivalScheduler{
		service: service,
		logger:  logger,
		config:  config,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
		lastRun: make(map[string]time.Time),
	}
}

// Start begins the archival scheduler
func (s *ArchivalScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("archival scheduler already running")
	}

	if !s.config.ArchivalEnabled {
		s.logger.Info("Archival scheduler disabled by configuration")
		return nil
	}

	s.running = true
	go s.run()

	s.logger.Info("Archival scheduler started",
		"check_interval", s.config.CheckInterval,
		"message_retention_days", s.config.MessageRetentionDays,
		"gateway_reception_retention_days", s.config.GatewayReceptionRetentionDays,
		"device_session_retention_days", s.config.DeviceSessionRetentionDays,
		"device_key_retention_days", s.config.DeviceKeyRetentionDays)

	return nil
}

// Stop halts the archival scheduler
func (s *ArchivalScheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	close(s.stopCh)
	<-s.doneCh

	s.running = false
	s.logger.Info("Archival scheduler stopped")

	return nil
}

// run is the main scheduler loop
func (s *ArchivalScheduler) run() {
	defer close(s.doneCh)

	// Run initial check immediately
	s.checkAndRunArchival()

	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAndRunArchival()
		}
	}
}

// checkAndRunArchival checks if any archival operations should run
func (s *ArchivalScheduler) checkAndRunArchival() {
	now := time.Now()

	// Check messages archival
	if s.shouldRunArchival("messages", now, s.config.MessageArchivalHour) {
		go s.runMessageArchival()
	}

	// Check gateway receptions archival
	if s.shouldRunArchival("gateway_receptions", now, s.config.GatewayReceptionArchivalHour) {
		go s.runGatewayReceptionArchival()
	}

	// Check device sessions archival
	if s.shouldRunArchival("device_sessions", now, s.config.DeviceSessionArchivalHour) {
		go s.runDeviceSessionArchival()
	}

	// Check device keys archival
	if s.shouldRunArchival("device_keys", now, s.config.DeviceKeyArchivalHour) {
		go s.runDeviceKeyArchival()
	}
}

// shouldRunArchival determines if an archival operation should run
func (s *ArchivalScheduler) shouldRunArchival(name string, now time.Time, targetHour int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if we've already run today
	lastRun, exists := s.lastRun[name]
	if exists && lastRun.Day() == now.Day() && lastRun.Month() == now.Month() && lastRun.Year() == now.Year() {
		return false
	}

	// Check if we're in the target hour
	return now.Hour() == targetHour
}

// markArchivalRun marks an archival operation as completed
func (s *ArchivalScheduler) markArchivalRun(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastRun[name] = time.Now()
}

// runMessageArchival runs the message archival operation
func (s *ArchivalScheduler) runMessageArchival() {
	ctx := context.Background()
	startTime := time.Now()

	s.logger.Info("Starting scheduled message archival",
		"retention_days", s.config.MessageRetentionDays)

	// Archive old messages
	retention := time.Duration(s.config.MessageRetentionDays) * 24 * time.Hour
	count, err := s.service.ArchiveOldMessages(ctx, retention)
	if err != nil {
		s.logger.Error("Message archival failed", "error", err)
		return
	}

	// Create monthly partitions
	if err := s.service.CreateMonthlyPartitions(ctx, 3); err != nil {
		s.logger.Error("Failed to create message partitions", "error", err)
	}

	// Purge very old archived messages (optional, could be configurable)
	purgeRetention := time.Duration(s.config.MessageRetentionDays*2) * 24 * time.Hour
	purgeTime := time.Now().Add(-purgeRetention)
	purgeCount, err := s.service.PurgeArchivedMessages(ctx, purgeTime)
	if err != nil {
		s.logger.Error("Failed to purge old archived messages", "error", err)
	}

	duration := time.Since(startTime)
	s.logger.Info("Message archival completed",
		"archived_count", count,
		"purged_count", purgeCount,
		"duration", duration)

	s.markArchivalRun("messages")
}

// runGatewayReceptionArchival runs the gateway reception archival operation
func (s *ArchivalScheduler) runGatewayReceptionArchival() {
	ctx := context.Background()
	startTime := time.Now()

	s.logger.Info("Starting scheduled gateway reception archival",
		"retention_days", s.config.GatewayReceptionRetentionDays)

	retention := time.Duration(s.config.GatewayReceptionRetentionDays) * 24 * time.Hour
	count, err := s.service.ArchiveOldGatewayReceptions(ctx, retention)
	if err != nil {
		s.logger.Error("Gateway reception archival failed", "error", err)
		return
	}

	duration := time.Since(startTime)
	s.logger.Info("Gateway reception archival completed",
		"archived_count", count,
		"duration", duration)

	s.markArchivalRun("gateway_receptions")
}

// runDeviceSessionArchival runs the device session archival operation
func (s *ArchivalScheduler) runDeviceSessionArchival() {
	ctx := context.Background()
	startTime := time.Now()

	s.logger.Info("Starting scheduled device session archival",
		"retention_days", s.config.DeviceSessionRetentionDays)

	retention := time.Duration(s.config.DeviceSessionRetentionDays) * 24 * time.Hour
	count, err := s.service.ArchiveOldDeviceSessions(ctx, retention)
	if err != nil {
		s.logger.Error("Device session archival failed", "error", err)
		return
	}

	duration := time.Since(startTime)
	s.logger.Info("Device session archival completed",
		"archived_count", count,
		"duration", duration)

	s.markArchivalRun("device_sessions")
}

// runDeviceKeyArchival runs the device key archival operation
func (s *ArchivalScheduler) runDeviceKeyArchival() {
	ctx := context.Background()
	startTime := time.Now()

	s.logger.Info("Starting scheduled device key archival",
		"retention_days", s.config.DeviceKeyRetentionDays)

	retention := time.Duration(s.config.DeviceKeyRetentionDays) * 24 * time.Hour
	count, err := s.service.ArchiveOldDeviceKeys(ctx, retention)
	if err != nil {
		s.logger.Error("Device key archival failed", "error", err)
		return
	}

	duration := time.Since(startTime)
	s.logger.Info("Device key archival completed",
		"archived_count", count,
		"duration", duration)

	s.markArchivalRun("device_keys")
}

// GetStatus returns the current status of the archival scheduler
func (s *ArchivalScheduler) GetStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := map[string]interface{}{
		"running": s.running,
		"config":  s.config,
	}

	// Add last run times
	lastRuns := make(map[string]string)
	for name, t := range s.lastRun {
		lastRuns[name] = t.Format(time.RFC3339)
	}
	status["last_runs"] = lastRuns

	// Calculate next run times
	now := time.Now()
	nextRuns := make(map[string]string)

	// Messages
	nextRuns["messages"] = s.calculateNextRun(now, s.config.MessageArchivalHour, s.lastRun["messages"])

	// Gateway receptions
	nextRuns["gateway_receptions"] = s.calculateNextRun(now, s.config.GatewayReceptionArchivalHour, s.lastRun["gateway_receptions"])

	// Device sessions
	nextRuns["device_sessions"] = s.calculateNextRun(now, s.config.DeviceSessionArchivalHour, s.lastRun["device_sessions"])

	// Device keys
	nextRuns["device_keys"] = s.calculateNextRun(now, s.config.DeviceKeyArchivalHour, s.lastRun["device_keys"])

	status["next_runs"] = nextRuns

	return status
}

// calculateNextRun calculates the next run time for an archival operation
func (s *ArchivalScheduler) calculateNextRun(now time.Time, targetHour int, lastRun time.Time) string {
	// If we've already run today, next run is tomorrow
	if lastRun.Day() == now.Day() && lastRun.Month() == now.Month() && lastRun.Year() == now.Year() {
		nextRun := time.Date(now.Year(), now.Month(), now.Day()+1, targetHour, 0, 0, 0, now.Location())
		return nextRun.Format(time.RFC3339)
	}

	// If we haven't run today and it's before the target hour, run today
	if now.Hour() < targetHour {
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), targetHour, 0, 0, 0, now.Location())
		return nextRun.Format(time.RFC3339)
	}

	// Otherwise, run tomorrow
	nextRun := time.Date(now.Year(), now.Month(), now.Day()+1, targetHour, 0, 0, 0, now.Location())
	return nextRun.Format(time.RFC3339)
}
