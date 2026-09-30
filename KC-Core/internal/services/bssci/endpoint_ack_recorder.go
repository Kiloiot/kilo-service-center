package bssciservices

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// EndpointAckStore marks the transmitted downlink of one window acknowledged
// by its endpoint and, with the mark, queues the acknowledgement on the
// request's channels; it returns the downlink, or false when no
// unacknowledged one matched.
type EndpointAckStore interface {
	MarkEndpointAcknowledged(ctx context.Context, ack models.EndpointAckRequest) (*storage.DownlinkMessage, bool, error)
}

// DownlinkAckEvents records in the owner tenant's events that the endpoint
// acknowledged the downlink in the uplink after packetCnt.
type DownlinkAckEvents interface {
	RecordDownlinkAcknowledged(ctx context.Context, downlink *storage.DownlinkMessage, packetCnt uint32) error
}

// EndpointAckRecorder records the downlink an uplink's dlAck acknowledges.
// The acknowledgement reaches MQTT through the delivery outbox, which the
// mark queues it on, so whichever process stores the uplink, with or without
// an MQTT client, the process that drains the outbox publishes it.
type EndpointAckRecorder struct {
	store    EndpointAckStore
	events   DownlinkAckEvents
	channels []models.DeliveryChannel
	logger   logger.Logger
}

// NewEndpointAckRecorder builds the recorder over the downlink queue; every
// acknowledgement is queued on channels, none when the deployment publishes
// no downlink results.
func NewEndpointAckRecorder(store EndpointAckStore, events DownlinkAckEvents, channels []models.DeliveryChannel, log logger.Logger) (*EndpointAckRecorder, error) {
	switch {
	case store == nil:
		return nil, ErrNilEndpointAckStore
	case events == nil:
		return nil, ErrNilEndpointAckEvents
	case log == nil:
		return nil, ErrNilEndpointAckLogger
	}
	return &EndpointAckRecorder{store: store, events: events, channels: channels, logger: log}, nil
}

// RecordEndpointAck records that the endpoint acknowledged, in the uplink
// stored as messageID, the downlink its owner transmitted in the window of
// packetCnt - 1 (BSSCI §3.10.1 dlAck). Only the reception that marks the
// downlink queues and records the acknowledgement, so it is published once.
func (r *EndpointAckRecorder) RecordEndpointAck(ctx context.Context, ownerTenantID int64, epEUI uint64, packetCnt uint32, messageID string) error {
	// Counter 0 follows an over-the-air attach, which restarts the counter, so no earlier window exists.
	if packetCnt == 0 {
		return nil
	}
	window := packetCnt - 1
	downlink, marked, err := r.store.MarkEndpointAcknowledged(ctx, models.EndpointAckRequest{
		TenantID: ownerTenantID, EpEUI: epEUI, WindowPacketCnt: int64(window), MessageID: messageID, Channels: r.channels,
	})
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
	if err := r.events.RecordDownlinkAcknowledged(ctx, downlink, window); err != nil {
		r.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToRecordDLDataResultEvent, logger.FieldError, err)
	}
	return nil
}
