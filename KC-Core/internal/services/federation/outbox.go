package federation

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// OutboxAppender inserts a pending record into the durable relay outbox.
type OutboxAppender interface {
	Insert(ctx context.Context, record *models.FederationOutboxRecord) error
}

// OutboxWriter inserts an uplink frame into the durable relay outbox.
// It is called from the BSSCI handleULData path when disposition is DispositionRelay.
type OutboxWriter struct {
	repo   OutboxAppender
	logger logger.Logger
}

// NewOutboxWriter creates an OutboxWriter backed by the given repository.
func NewOutboxWriter(repo OutboxAppender, log logger.Logger) *OutboxWriter {
	return &OutboxWriter{repo: repo, logger: log}
}

// Enqueue inserts a new pending relay record and returns the assigned relay_id.
// The caller (handleULData) must send ulDataRsp ONLY after this returns nil.
func (w *OutboxWriter) Enqueue(ctx context.Context, epEUI, bsEUI uint64, rawFrame []byte, receivedAtNs int64) (uuid.UUID, error) {
	relayID := uuid.New()
	record := &models.FederationOutboxRecord{
		RelayID:      relayID,
		EpEUI:        int64(epEUI), //nolint:gosec // G115: EUI fits int64
		BsEUI:        int64(bsEUI), //nolint:gosec // G115: EUI fits int64
		RawFrame:     rawFrame,
		ReceivedAtNs: receivedAtNs,
	}
	if err := w.repo.Insert(ctx, record); err != nil {
		return uuid.Nil, fmt.Errorf("%w: %w", ErrOutboxEnqueue, err)
	}
	w.logger.DebugContext(ctx, LogUplinkEnqueued,
		logger.FieldRelayID, relayID, logger.FieldEpEuiSnake, epEUI)
	return relayID, nil
}
