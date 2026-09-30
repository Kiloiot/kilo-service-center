// Package delivery drains the message delivery outbox: every new uplink is
// queued once per channel by the uplink store, and this worker claims due
// rows, hands the stored message to the channel, and records the result so
// a slow or unreachable consumer never loses a message.
package delivery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// Outbox hands due rows to the worker, hiding each claimed row for the lease.
type Outbox interface {
	ClaimDue(ctx context.Context, limit int, lease time.Duration) ([]models.MessageDeliveryRecord, error)
}

// Outcomes records how one delivery attempt ended.
type Outcomes interface {
	MarkDelivered(ctx context.Context, messageID uuid.UUID, channel models.DeliveryChannel) error
	Reschedule(ctx context.Context, messageID uuid.UUID, channel models.DeliveryChannel, delay time.Duration, reason string) error
	Park(ctx context.Context, messageID uuid.UUID, channel models.DeliveryChannel, reason string) error
}

// MessageReader loads the stored uplink a row points to, tenant-scoped.
type MessageReader interface {
	GetULDataMessage(ctx context.Context, id string, tenantID int64) (*mioty.ULDataMessage, error)
}

// SCACIBroadcaster fans an uplink out to the Application Centers of a tenant.
type SCACIBroadcaster interface {
	BroadcastULData(ctx context.Context, tenantID int64, data *mioty.ULDataMessage) error
}

// MQTTPublisher publishes an uplink as a device event.
type MQTTPublisher interface {
	PublishUplink(ctx context.Context, orgUUID string, msg *mioty.ULDataMessage) error
}

// EventRecorder records the system event written when a row is parked.
type EventRecorder interface {
	CreateEvent(ctx context.Context, event *models.SystemEvent) error
}

// Channels are the delivery targets; a nil channel is not configured here and its rows park.
type Channels struct {
	SCACI SCACIBroadcaster
	MQTT  MQTTPublisher
}

// Dependencies are the worker's collaborators; all but the channels are required.
type Dependencies struct {
	Outbox   Outbox
	Outcomes Outcomes
	Messages MessageReader
	Channels Channels
	Events   EventRecorder
	Clock    clock.Clock
	Logger   logger.Logger
}

// Config bounds the worker's polling and retry behaviour.
type Config struct {
	PollInterval time.Duration
	BatchSize    int
	Retry        RetryPolicy
}

// Worker is the outbox drain loop.
type Worker struct {
	deps Dependencies
	cfg  Config
}

// NewWorker wires the drain loop and rejects a missing collaborator or an unusable configuration.
func NewWorker(deps Dependencies, cfg Config) (*Worker, error) {
	if deps.Outbox == nil || deps.Outcomes == nil || deps.Messages == nil ||
		deps.Events == nil || deps.Clock == nil || deps.Logger == nil {
		return nil, errMissingDependency
	}
	if cfg.PollInterval <= 0 || cfg.BatchSize <= 0 || cfg.Retry.Lease() <= 0 {
		return nil, fmt.Errorf(errFmtInvalidConfig, errInvalidConfig, cfg.PollInterval, cfg.BatchSize, cfg.Retry.Lease())
	}
	return &Worker{deps: deps, cfg: cfg}, nil
}

// Start runs the worker in the background; the returned stop waits for the batch in flight.
func (w *Worker) Start(parent context.Context) (stop func()) {
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Run(ctx)
	}()
	return func() {
		cancel()
		<-done
	}
}

// Run drains the outbox until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()
	for {
		// A claimed batch completes so a stop never leaves a row delivered but unrecorded.
		w.DrainOnce(context.WithoutCancel(ctx))
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// DrainOnce claims one batch of due rows and delivers each of them.
func (w *Worker) DrainOnce(ctx context.Context) {
	rows, err := w.deps.Outbox.ClaimDue(ctx, w.cfg.BatchSize, w.cfg.Retry.Lease())
	if err != nil {
		w.deps.Logger.ErrorContext(ctx, LogDeliveryClaimFailed, logger.FieldError, err)
		return
	}
	for i := range rows {
		w.deliver(ctx, rows[i])
	}
}

// deliver runs one attempt for a claimed row and records the outcome.
func (w *Worker) deliver(ctx context.Context, row models.MessageDeliveryRecord) {
	err := w.attempt(ctx, row)
	switch {
	case err == nil:
		w.markDelivered(ctx, row)
	case w.cfg.Retry.Permanent(err):
		w.park(ctx, row, err)
	default:
		w.reschedule(ctx, row, err)
	}
}

// attempt hands the stored message to the row's channel.
func (w *Worker) attempt(ctx context.Context, row models.MessageDeliveryRecord) error {
	msg, err := w.deps.Messages.GetULDataMessage(ctx, row.MessageID.String(), row.OwnerTenantID)
	if err != nil {
		return fmt.Errorf("%w: %w", errLoadMessage, err)
	}
	switch row.Channel {
	case models.DeliveryChannelSCACI:
		if w.deps.Channels.SCACI == nil {
			return errChannelNotConfigured
		}
		return w.deps.Channels.SCACI.BroadcastULData(ctx, row.OwnerTenantID, msg)
	case models.DeliveryChannelMQTT:
		if w.deps.Channels.MQTT == nil || msg.OrgUUID == nil {
			return errChannelNotConfigured
		}
		return w.deps.Channels.MQTT.PublishUplink(ctx, *msg.OrgUUID, msg)
	default:
		return fmt.Errorf("%w: %s", errUnknownChannel, row.Channel)
	}
}

// markDelivered closes the row after a successful attempt.
func (w *Worker) markDelivered(ctx context.Context, row models.MessageDeliveryRecord) {
	if err := w.deps.Outcomes.MarkDelivered(ctx, row.MessageID, row.Channel); err != nil {
		w.deps.Logger.ErrorContext(ctx, LogDeliveryMarkFailed,
			logger.FieldMessageIDSnake, row.MessageID.String(), logger.FieldChannel, string(row.Channel), logger.FieldError, err)
	}
}

// reschedule keeps a transiently failed row pending for its next attempt.
func (w *Worker) reschedule(ctx context.Context, row models.MessageDeliveryRecord, cause error) {
	delay := w.cfg.Retry.Delay(row.Attempts)
	w.deps.Logger.WarnContext(ctx, LogDeliveryAttemptFailed,
		logger.FieldMessageIDSnake, row.MessageID.String(), logger.FieldChannel, string(row.Channel),
		logger.FieldAttempt, row.Attempts, logger.FieldBackoff, delay, logger.FieldError, cause)
	if err := w.deps.Outcomes.Reschedule(ctx, row.MessageID, row.Channel, delay, cause.Error()); err != nil {
		w.deps.Logger.ErrorContext(ctx, LogDeliveryRescheduleFailed,
			logger.FieldMessageIDSnake, row.MessageID.String(), logger.FieldChannel, string(row.Channel), logger.FieldError, err)
	}
}

// park takes the row out of the retry loop and leaves an operator-visible trace.
func (w *Worker) park(ctx context.Context, row models.MessageDeliveryRecord, cause error) {
	if err := w.deps.Outcomes.Park(ctx, row.MessageID, row.Channel, cause.Error()); err != nil {
		w.deps.Logger.ErrorContext(ctx, LogDeliveryParkFailed,
			logger.FieldMessageIDSnake, row.MessageID.String(), logger.FieldChannel, string(row.Channel), logger.FieldError, err)
		return
	}
	w.deps.Logger.ErrorContext(ctx, LogDeliveryParked,
		logger.FieldMessageIDSnake, row.MessageID.String(), logger.FieldChannel, string(row.Channel),
		logger.FieldAttempt, row.Attempts, logger.FieldError, cause)
	now := w.deps.Clock.Now()
	event := &models.SystemEvent{
		TenantID:    fmt.Sprintf("%d", row.OwnerTenantID),
		EventType:   eventTypeDeliveryParked,
		Category:    models.EventCategoryMessage,
		Severity:    bssci.SeverityError,
		Title:       fmt.Sprintf(eventTitleDeliveryParkedFmt, row.Channel),
		Description: fmt.Sprintf(eventDescriptionDeliveryParkedFmt, row.MessageID.String(), row.Channel, row.Attempts, cause),
		SourceType:  mioty.SourceTypeServiceCenter,
		SourceName:  row.MessageID.String(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := w.deps.Events.CreateEvent(ctx, event); err != nil && !errors.Is(err, context.Canceled) {
		w.deps.Logger.ErrorContext(ctx, LogDeliveryParkEventFailed,
			logger.FieldMessageIDSnake, row.MessageID.String(), logger.FieldError, err)
	}
}
