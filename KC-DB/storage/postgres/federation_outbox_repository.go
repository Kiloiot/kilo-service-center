package postgres

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// FederationOutboxRepository manages CE-side outbox records using PostgreSQL.
type FederationOutboxRepository struct {
	clock clock.Clock
	db    *sqlx.DB
}

// NewFederationOutboxRepository creates a new FederationOutboxRepository.
func NewFederationOutboxRepository(db *sqlx.DB, clk clock.Clock) *FederationOutboxRepository {
	return &FederationOutboxRepository{clock: clk, db: db}
}

// Insert adds a new pending outbox record.
func (r *FederationOutboxRepository) Insert(ctx context.Context, record *models.FederationOutboxRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO federation_outbox (relay_id, ep_eui, bs_eui, raw_frame, received_at_ns, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, record.RelayID, record.EpEUI, record.BsEUI, record.RawFrame, record.ReceivedAtNs,
		models.FederationOutboxStatusPending)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapFederationOutboxInsert, err)
	}
	return nil
}

// ListPending returns pending and sent records for relay or replay on reconnect.
func (r *FederationOutboxRepository) ListPending(ctx context.Context, limit int) ([]*models.FederationOutboxRecord, error) {
	var rows []*models.FederationOutboxRecord
	query := `
		SELECT * FROM federation_outbox
		WHERE status IN ($1, $2)
		ORDER BY created_at ASC
	`
	args := []interface{}{models.FederationOutboxStatusPending, models.FederationOutboxStatusSent}
	if limit > 0 {
		query += " LIMIT $3"
		args = append(args, limit)
	}
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapFederationOutboxListPending, err)
	}
	return rows, nil
}

// ListPendingOnly returns only records with status 'pending', excluding already-sent records.
func (r *FederationOutboxRepository) ListPendingOnly(ctx context.Context, limit int) ([]*models.FederationOutboxRecord, error) {
	var rows []*models.FederationOutboxRecord
	query := `
		SELECT * FROM federation_outbox
		WHERE status = $1
		ORDER BY created_at ASC
	`
	args := []interface{}{models.FederationOutboxStatusPending}
	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
	}
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapFederationOutboxListPendingOnly, err)
	}
	return rows, nil
}

// MarkSent transitions a record to sent status.
func (r *FederationOutboxRepository) MarkSent(ctx context.Context, relayID uuid.UUID) error {
	now := r.clock.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE federation_outbox SET status = $1, sent_at = $2 WHERE relay_id = $3
	`, models.FederationOutboxStatusSent, now, relayID)
	return err
}

// MarkAcked transitions a record to acked status.
func (r *FederationOutboxRepository) MarkAcked(ctx context.Context, relayID uuid.UUID) error {
	now := r.clock.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE federation_outbox SET status = $1, acked_at = $2 WHERE relay_id = $3
	`, models.FederationOutboxStatusAcked, now, relayID)
	return err
}

// MarkRejected marks a record as permanently rejected.
func (r *FederationOutboxRepository) MarkRejected(ctx context.Context, relayID uuid.UUID, errMsg string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE federation_outbox SET status = $1, last_error = $2 WHERE relay_id = $3
	`, models.FederationOutboxStatusRejected, errMsg, relayID)
	return err
}
