package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/kilocenter/KC-DB/storage/interfaces"
)

// FederationOutboxRepository manages CE-side outbox records using PostgreSQL.
type FederationOutboxRepository struct {
	db *sqlx.DB
}

// NewFederationOutboxRepository creates a new FederationOutboxRepository.
func NewFederationOutboxRepository(db *sqlx.DB) *FederationOutboxRepository {
	return &FederationOutboxRepository{db: db}
}

// Insert adds a new pending outbox record.
func (r *FederationOutboxRepository) Insert(ctx context.Context, record *interfaces.FederationOutboxRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO federation_outbox (relay_id, ep_eui, bs_eui, raw_frame, received_at_ns, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, record.RelayID, record.EpEUI, record.BsEUI, record.RawFrame, record.ReceivedAtNs,
		interfaces.FederationOutboxStatusPending)
	if err != nil {
		return fmt.Errorf("federation_outbox insert: %w", err)
	}
	return nil
}

// ListPending returns pending and sent records for relay or replay on reconnect.
func (r *FederationOutboxRepository) ListPending(ctx context.Context, limit int) ([]*interfaces.FederationOutboxRecord, error) {
	var rows []*interfaces.FederationOutboxRecord
	query := `
		SELECT * FROM federation_outbox
		WHERE status IN ($1, $2)
		ORDER BY created_at ASC
	`
	args := []interface{}{interfaces.FederationOutboxStatusPending, interfaces.FederationOutboxStatusSent}
	if limit > 0 {
		query += " LIMIT $3"
		args = append(args, limit)
	}
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("federation_outbox list pending: %w", err)
	}
	return rows, nil
}

// ListPendingOnly returns only records with status 'pending', excluding already-sent records.
func (r *FederationOutboxRepository) ListPendingOnly(ctx context.Context, limit int) ([]*interfaces.FederationOutboxRecord, error) {
	var rows []*interfaces.FederationOutboxRecord
	query := `
		SELECT * FROM federation_outbox
		WHERE status = $1
		ORDER BY created_at ASC
	`
	args := []interface{}{interfaces.FederationOutboxStatusPending}
	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
	}
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("federation_outbox list pending only: %w", err)
	}
	return rows, nil
}

// MarkSent transitions a record to sent status.
func (r *FederationOutboxRepository) MarkSent(ctx context.Context, relayID uuid.UUID) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE federation_outbox SET status = $1, sent_at = $2 WHERE relay_id = $3
	`, interfaces.FederationOutboxStatusSent, now, relayID)
	return err
}

// MarkAcked transitions a record to acked status.
func (r *FederationOutboxRepository) MarkAcked(ctx context.Context, relayID uuid.UUID) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE federation_outbox SET status = $1, acked_at = $2 WHERE relay_id = $3
	`, interfaces.FederationOutboxStatusAcked, now, relayID)
	return err
}

// MarkRejected marks a record as permanently rejected.
func (r *FederationOutboxRepository) MarkRejected(ctx context.Context, relayID uuid.UUID, errMsg string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE federation_outbox SET status = $1, last_error = $2 WHERE relay_id = $3
	`, interfaces.FederationOutboxStatusRejected, errMsg, relayID)
	return err
}

// PurgeOld deletes completed records older than the cutoff.
func (r *FederationOutboxRepository) PurgeOld(ctx context.Context, olderThan time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM federation_outbox
		WHERE status IN ($1, $2) AND created_at < $3
	`, interfaces.FederationOutboxStatusAcked, interfaces.FederationOutboxStatusRejected, olderThan)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}
