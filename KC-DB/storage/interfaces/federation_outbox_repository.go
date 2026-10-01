package interfaces

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// FederationOutboxRepository manages CE-side outbox records for the relay stream.
type FederationOutboxRepository interface {
	// Insert adds a new pending outbox record.
	Insert(ctx context.Context, record *models.FederationOutboxRecord) error

	// ListPending returns all pending and sent records for replay on reconnect.
	ListPending(ctx context.Context, limit int) ([]*models.FederationOutboxRecord, error)

	// ListPendingOnly returns only records with status 'pending' (not yet sent).
	// Used in steady-state draining to avoid re-sending already-sent records.
	ListPendingOnly(ctx context.Context, limit int) ([]*models.FederationOutboxRecord, error)

	// MarkSent transitions a record from pending to sent.
	MarkSent(ctx context.Context, relayID uuid.UUID) error

	// MarkAcked transitions a record from sent to acked.
	MarkAcked(ctx context.Context, relayID uuid.UUID) error

	// MarkRejected marks a record as permanently rejected with an error message.
	MarkRejected(ctx context.Context, relayID uuid.UUID, errMsg string) error
}
