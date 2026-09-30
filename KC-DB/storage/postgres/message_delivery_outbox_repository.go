package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// MessageDeliveryOutboxRepository hands due outbox rows to the delivery worker and
// records the result of each attempt; every time it writes or compares comes
// from its clock.
type MessageDeliveryOutboxRepository struct {
	clock clock.Clock
	db    *sqlx.DB
}

// NewMessageDeliveryOutboxRepository creates the outbox repository.
func NewMessageDeliveryOutboxRepository(db *sqlx.DB, clk clock.Clock) *MessageDeliveryOutboxRepository {
	return &MessageDeliveryOutboxRepository{clock: clk, db: db}
}

const (
	sqlInsertDeliveryRow = `
		INSERT INTO message_delivery_outbox (message_id, channel, owner_tenant_id, next_attempt_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (message_id, channel) DO NOTHING`
	sqlClaimDueDeliveries = `
		UPDATE message_delivery_outbox AS o
		SET attempts = o.attempts + 1,
		    next_attempt_at = $4::timestamptz + make_interval(secs => $2)
		FROM (
			SELECT message_id, channel
			FROM message_delivery_outbox
			WHERE status = $3 AND next_attempt_at <= $4
			ORDER BY next_attempt_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		) AS due
		WHERE o.message_id = due.message_id AND o.channel = due.channel
		RETURNING o.message_id, o.channel, o.owner_tenant_id, o.status, o.attempts,
		          o.next_attempt_at, o.last_error, o.created_at, o.delivered_at`
	sqlMarkDelivered = `
		UPDATE message_delivery_outbox
		SET status = $3, delivered_at = $4, last_error = NULL
		WHERE message_id = $1 AND channel = $2`
	sqlRescheduleDelivery = `
		UPDATE message_delivery_outbox
		SET next_attempt_at = $6::timestamptz + make_interval(secs => $3), last_error = $4
		WHERE message_id = $1 AND channel = $2 AND status = $5`
	sqlParkDelivery = `
		UPDATE message_delivery_outbox
		SET status = $3, last_error = $4
		WHERE message_id = $1 AND channel = $2`
)

// insertDeliveryRows queues one outbox row per channel inside the caller's
// transaction at queuedAt, due at dueAt.
func insertDeliveryRows(ctx context.Context, exec uplinkExecutor, messageID string, tenantID int64, channels []models.DeliveryChannel, queuedAt, dueAt time.Time) error {
	for _, channel := range channels {
		if _, err := exec.ExecContext(ctx, sqlInsertDeliveryRow, messageID, channel, tenantID, dueAt, queuedAt); err != nil {
			return fmt.Errorf("%s: %w", errWrapInsertDeliveryRow, err)
		}
	}
	return nil
}

// ClaimDue locks due pending rows, counts the attempt and hides each row from other workers for the lease.
func (r *MessageDeliveryOutboxRepository) ClaimDue(ctx context.Context, limit int, lease time.Duration) ([]models.MessageDeliveryRecord, error) {
	if limit <= 0 || lease <= 0 {
		return nil, fmt.Errorf("%s: %w", errWrapClaimDueDeliveries, storage.ErrInvalidInput)
	}
	var rows []models.MessageDeliveryRecord
	if err := r.db.SelectContext(ctx, &rows, sqlClaimDueDeliveries, limit, lease.Seconds(), models.DeliveryStatusPending, r.clock.Now()); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapClaimDueDeliveries, err)
	}
	return rows, nil
}

// MarkDelivered closes the row after a successful delivery.
func (r *MessageDeliveryOutboxRepository) MarkDelivered(ctx context.Context, messageID uuid.UUID, channel models.DeliveryChannel) error {
	result, err := r.db.ExecContext(ctx, sqlMarkDelivered, messageID, channel, models.DeliveryStatusDelivered, r.clock.Now())
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapMarkDelivered, err)
	}
	return requireDeliveryRow(result, errWrapMarkDelivered)
}

// Reschedule keeps a pending row for another attempt after delay and records the failure.
func (r *MessageDeliveryOutboxRepository) Reschedule(ctx context.Context, messageID uuid.UUID, channel models.DeliveryChannel,
	delay time.Duration, reason string,
) error {
	if delay <= 0 {
		return fmt.Errorf("%s: %w", errWrapRescheduleDelivery, storage.ErrInvalidInput)
	}
	result, err := r.db.ExecContext(ctx, sqlRescheduleDelivery, messageID, channel, delay.Seconds(), reason, models.DeliveryStatusPending, r.clock.Now())
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapRescheduleDelivery, err)
	}
	return requireDeliveryRow(result, errWrapRescheduleDelivery)
}

// Park takes the row out of the retry loop and keeps the last failure.
func (r *MessageDeliveryOutboxRepository) Park(ctx context.Context, messageID uuid.UUID, channel models.DeliveryChannel, reason string) error {
	result, err := r.db.ExecContext(ctx, sqlParkDelivery, messageID, channel, models.DeliveryStatusParked, reason)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapParkDelivery, err)
	}
	return requireDeliveryRow(result, errWrapParkDelivery)
}

func requireDeliveryRow(result interface{ RowsAffected() (int64, error) }, wrap string) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", wrap, err)
	}
	if rows == 0 {
		return fmt.Errorf("%s: %w", wrap, storage.ErrNotFound)
	}
	return nil
}
