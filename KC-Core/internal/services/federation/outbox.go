package federation

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/interfaces"
)

// OutboxWriter inserts an uplink frame into the durable relay outbox.
// It is called from the BSSCI handleULData path when disposition is DispositionRelay.
type OutboxWriter struct {
	repo   interfaces.FederationOutboxRepository
	logger logger.Logger
}

// NewOutboxWriter creates an OutboxWriter backed by the given repository.
func NewOutboxWriter(repo interfaces.FederationOutboxRepository, log logger.Logger) *OutboxWriter {
	return &OutboxWriter{repo: repo, logger: log}
}

// Enqueue inserts a new pending relay record and returns the assigned relay_id.
// The caller (handleULData) must send ulDataRsp ONLY after this returns nil.
func (w *OutboxWriter) Enqueue(ctx context.Context, epEUI, bsEUI uint64, rawFrame []byte, receivedAtNs int64) (uuid.UUID, error) {
	relayID := uuid.New()
	record := &interfaces.FederationOutboxRecord{
		RelayID:      relayID,
		EpEUI:        int64(epEUI), //nolint:gosec // G115: EUI fits int64
		BsEUI:        int64(bsEUI), //nolint:gosec // G115: EUI fits int64
		RawFrame:     rawFrame,
		ReceivedAtNs: receivedAtNs,
	}
	if err := w.repo.Insert(ctx, record); err != nil {
		return uuid.Nil, fmt.Errorf("outbox enqueue: %w", err)
	}
	w.logger.DebugContext(ctx, "Uplink enqueued for federation relay",
		"relay_id", relayID, "ep_eui", epEUI)
	return relayID, nil
}
