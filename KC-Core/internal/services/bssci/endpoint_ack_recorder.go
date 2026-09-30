package bssciservices

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// EndpointAckStore marks the transmitted downlink of one window acknowledged
// by its endpoint and returns its queue id; false when no unacknowledged one matched.
type EndpointAckStore interface {
	MarkEndpointAcknowledged(ctx context.Context, tenantID int64, epEUI uint64, windowPacketCnt int64) (int64, bool, error)
}

// DownlinkAckEvents records the endpoint's acknowledgement in the owner's events.
type DownlinkAckEvents interface {
	RecordDownlinkAcknowledged(ctx context.Context, tenantID int64, epEUI uint64, queueID int64, packetCnt uint32) error
}

// EndpointAckRecorder records the downlink an uplink's dlAck acknowledges.
type EndpointAckRecorder struct {
	store  EndpointAckStore
	events DownlinkAckEvents
	logger logger.Logger
}

// NewEndpointAckRecorder builds the recorder over the downlink queue.
func NewEndpointAckRecorder(store EndpointAckStore, events DownlinkAckEvents, log logger.Logger) (*EndpointAckRecorder, error) {
	switch {
	case store == nil:
		return nil, ErrNilEndpointAckStore
	case events == nil:
		return nil, ErrNilEndpointAckEvents
	case log == nil:
		return nil, ErrNilEndpointAckLogger
	}
	return &EndpointAckRecorder{store: store, events: events, logger: log}, nil
}

// RecordEndpointAck records that the endpoint acknowledged the downlink its
// owner transmitted in the window of packetCnt - 1 (BSSCI §3.10.1 dlAck).
func (r *EndpointAckRecorder) RecordEndpointAck(ctx context.Context, ownerTenantID int64, epEUI uint64, packetCnt uint32) error {
	// Counter 0 follows an over-the-air attach, which restarts the counter, so no earlier window exists.
	if packetCnt == 0 {
		return nil
	}
	window := packetCnt - 1
	queueID, marked, err := r.store.MarkEndpointAcknowledged(ctx, ownerTenantID, epEUI, int64(window))
	if err != nil {
		return fmt.Errorf("%w: %w", errMarkEndpointAck, err)
	}
	if !marked {
		r.logger.DebugContext(ctx, bssci.LogBSSCIEndpointAckWithoutDownlink,
			logger.FieldEpEui, epEUI, logger.FieldPacketCnt, window)
		return nil
	}
	r.logger.InfoContext(ctx, bssci.LogBSSCIDownlinkAcknowledgedByEndpoint,
		logger.FieldEpEui, epEUI, logger.FieldPacketCnt, window, logger.FieldOwnerTenantID, ownerTenantID)
	if err := r.events.RecordDownlinkAcknowledged(ctx, ownerTenantID, epEUI, queueID, window); err != nil {
		return fmt.Errorf("%w: %w", errRecordDownlinkAckEvent, err)
	}
	return nil
}
