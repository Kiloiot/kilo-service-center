package postgres

import (
	"context"
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// messageArchiver is the archival work the scheduler runs.
type messageArchiver interface {
	ArchiveOldMessages(ctx context.Context, olderThan time.Duration) (int64, error)
	CreateMonthlyPartitions(ctx context.Context, monthsAhead int) error
	PurgeArchivedMessages(ctx context.Context, archivedBefore time.Time) (int64, error)
}

// ArchivalScheduler manages scheduled archival operations. A single loop
// goroutine runs every archival itself, so runs never overlap and Stop,
// which cancels the loop and waits for it, returns only after the run in
// flight has ended.
type ArchivalScheduler struct {
	clock    clock.Clock
	archiver messageArchiver
	logger   logger.Logger
	config   ArchivalConfig

	// lifecycle serializes Start and Stop; the loop never takes it, so Stop
	// can hold it while waiting for the loop to end.
	lifecycle sync.Mutex
	cancel    context.CancelFunc
	done      chan struct{}

	// mu guards the status the loop and GetStatus share.
	mu      sync.Mutex
	running bool
	lastRun map[string]time.Time
}

// ArchivalConfig defines configuration for archival operations
type ArchivalConfig struct {
	// Message archival settings
	MessageRetentionDays int
	MessageArchivalHour  int // Hour of day to run (0-23)

	// General settings
	ArchivalEnabled bool
	CheckInterval   time.Duration // How often to check if archival should run
}

const (
	archivalJobMessages = "messages"
)

// Default archival retention and run-hour policy (hours are local
// off-peak slots).
const (
	defaultMessageRetentionDays = 90
	defaultMessageArchivalHour  = 2
)

// monthlyPartitionsAhead creates message partitions this many months ahead.
const monthlyPartitionsAhead = 3

// purgeRetentionFactor keeps archived messages this multiple of the live
// retention before purging.
const purgeRetentionFactor = 2

// DefaultArchivalConfig returns default archival configuration
// Archival job names used for last-run bookkeeping.
func DefaultArchivalConfig() ArchivalConfig {
	return ArchivalConfig{
		MessageRetentionDays: defaultMessageRetentionDays,
		MessageArchivalHour:  defaultMessageArchivalHour,
		ArchivalEnabled:      defaultArchivalEnabled,
		CheckInterval:        defaultArchivalCheckInterval,
	}
}

// NewArchivalScheduler creates a new archival scheduler
func NewArchivalScheduler(archiver messageArchiver, logger logger.Logger, config ArchivalConfig, clk clock.Clock) *ArchivalScheduler {
	return &ArchivalScheduler{
		clock:    clk,
		archiver: archiver,
		logger:   logger,
		config:   config,
		lastRun:  make(map[string]time.Time),
	}
}

// Start begins the archival scheduler. ctx is the process lifecycle context;
// the running archival aborts when it or Stop cancels the loop.
func (s *ArchivalScheduler) Start(ctx context.Context) error {
	s.lifecycle.Lock()
	defer s.lifecycle.Unlock()

	if s.cancel != nil {
		return errTextArchivalSchedulerAlreadyRunning
	}

	if !s.config.ArchivalEnabled {
		s.logger.Info(logMsgArchivalSchedulerDisabledByConfiguration)
		return nil
	}

	loopCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})
	s.setRunning(true)
	go s.run(loopCtx, s.done)

	s.logger.Info(logMsgArchivalSchedulerStarted,
		logger.FieldCheckInterval, s.config.CheckInterval,
		logger.FieldMessageRetentionDays, s.config.MessageRetentionDays)

	return nil
}

// Stop cancels the scheduler loop and waits until it, and the archival it
// may be running, has ended.
func (s *ArchivalScheduler) Stop() error {
	s.lifecycle.Lock()
	defer s.lifecycle.Unlock()

	if s.cancel == nil {
		return nil
	}

	s.cancel()
	<-s.done
	s.cancel, s.done = nil, nil

	s.setRunning(false)
	s.logger.Info(logMsgArchivalSchedulerStopped)

	return nil
}

func (s *ArchivalScheduler) setRunning(running bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = running
}

// run is the main scheduler loop
func (s *ArchivalScheduler) run(ctx context.Context, done chan<- struct{}) {
	defer close(done)

	// Run initial check immediately
	s.checkAndRunArchival(ctx)

	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkAndRunArchival(ctx)
		}
	}
}

// checkAndRunArchival runs the message archival when it is due
func (s *ArchivalScheduler) checkAndRunArchival(ctx context.Context) {
	if s.shouldRunArchival(archivalJobMessages, s.clock.Now(), s.config.MessageArchivalHour) {
		s.runMessageArchival(ctx)
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
	s.lastRun[name] = s.clock.Now()
}

// runMessageArchival runs the message archival operation
func (s *ArchivalScheduler) runMessageArchival(ctx context.Context) {
	startTime := s.clock.Now()

	s.logger.Info(logMsgStartingScheduledMessageArchival,
		logger.FieldRetentionDays, s.config.MessageRetentionDays)

	// Archive old messages
	retention := time.Duration(s.config.MessageRetentionDays) * dayDuration
	count, err := s.archiver.ArchiveOldMessages(ctx, retention)
	if err != nil {
		s.logger.Error(logMsgMessageArchival, logger.FieldError, err)
		return
	}

	// Create monthly partitions
	if err := s.archiver.CreateMonthlyPartitions(ctx, monthlyPartitionsAhead); err != nil {
		s.logger.Error(logMsgCreateMessagePartitions, logger.FieldError, err)
	}

	// Purge very old archived messages (optional, could be configurable)
	purgeRetention := time.Duration(s.config.MessageRetentionDays*purgeRetentionFactor) * dayDuration
	purgeTime := s.clock.Now().Add(-purgeRetention)
	purgeCount, err := s.archiver.PurgeArchivedMessages(ctx, purgeTime)
	if err != nil {
		s.logger.Error(logMsgPurgeOldArchivedMessages, logger.FieldError, err)
	}

	duration := s.clock.Now().Sub(startTime)
	s.logger.Info(logMsgMessageArchivalCompleted,
		logger.FieldArchivedCount, count,
		logger.FieldPurgedCount, purgeCount,
		logger.FieldDuration, duration)

	s.markArchivalRun(archivalJobMessages)
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
	now := s.clock.Now()
	nextRuns := make(map[string]string)

	// Messages
	nextRuns["messages"] = s.calculateNextRun(now, s.config.MessageArchivalHour, s.lastRun["messages"])

	// Gateway receptions

	// Device sessions

	// Device keys

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

// dayDuration converts configured retention day counts into durations.
const dayDuration = 24 * time.Hour
