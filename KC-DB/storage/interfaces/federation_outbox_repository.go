package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	// FederationOutboxStatusPending means the relay record has not been sent yet.
	FederationOutboxStatusPending = "pending"
	// FederationOutboxStatusSent means the relay record was sent but not yet acknowledged.
	FederationOutboxStatusSent = "sent"
	// FederationOutboxStatusAcked means ECE acknowledged receipt successfully.
	FederationOutboxStatusAcked = "acked"
	// FederationOutboxStatusRejected means ECE rejected the uplink (terminal; no retry).
	FederationOutboxStatusRejected = "rejected"
)

// FederationOutboxRecord represents a single uplink pending relay to ECE.
type FederationOutboxRecord struct {
	RelayID      uuid.UUID  `db:"relay_id"`
	EpEUI        int64      `db:"ep_eui"`
	BsEUI        int64      `db:"bs_eui"`
	RawFrame     []byte     `db:"raw_frame"`
	ReceivedAtNs int64      `db:"received_at_ns"`
	Status       string     `db:"status"`
	LastError    *string    `db:"last_error"`
	SentAt       *time.Time `db:"sent_at"`
	AckedAt      *time.Time `db:"acked_at"`
	CreatedAt    time.Time  `db:"created_at"`
}

// FederationOutboxRepository manages CE-side outbox records for the relay stream.
type FederationOutboxRepository interface {
	// Insert adds a new pending outbox record.
	Insert(ctx context.Context, record *FederationOutboxRecord) error

	// ListPending returns all pending and sent records for replay on reconnect.
	ListPending(ctx context.Context, limit int) ([]*FederationOutboxRecord, error)

	// ListPendingOnly returns only records with status 'pending' (not yet sent).
	// Used in steady-state draining to avoid re-sending already-sent records.
	ListPendingOnly(ctx context.Context, limit int) ([]*FederationOutboxRecord, error)

	// MarkSent transitions a record from pending to sent.
	MarkSent(ctx context.Context, relayID uuid.UUID) error

	// MarkAcked transitions a record from sent to acked.
	MarkAcked(ctx context.Context, relayID uuid.UUID) error

	// MarkRejected marks a record as permanently rejected with an error message.
	MarkRejected(ctx context.Context, relayID uuid.UUID, errMsg string) error

	// PurgeOld deletes acked and rejected records older than the given cutoff time.
	PurgeOld(ctx context.Context, olderThan time.Time) (int64, error)
}
