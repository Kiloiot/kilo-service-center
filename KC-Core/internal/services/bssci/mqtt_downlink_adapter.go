package bssciservices

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-MQTT/pkg/mqtt"
	"github.com/google/uuid"
)

var mqttQueueIDSeed = uint64(time.Now().UnixNano()) //nolint:gosec // G115: UnixNano is positive for current epoch

func nextMQTTQueueID() uint64 {
	return atomic.AddUint64(&mqttQueueIDSeed, 1)
}

// DownlinkQueuer abstracts SCACI downlink queueing at the service layer.
type DownlinkQueuer interface {
	QueueDownlink(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue) (uint64, error)
}

// mqttDownlinkAdapter wraps DownlinkQueuer to satisfy mqtt.DownlinkEnqueuer.
type mqttDownlinkAdapter struct {
	queuer DownlinkQueuer
}

// NewMQTTDownlinkAdapter creates an adapter that satisfies mqtt.DownlinkEnqueuer.
func NewMQTTDownlinkAdapter(queuer DownlinkQueuer) mqtt.DownlinkEnqueuer {
	return &mqttDownlinkAdapter{queuer: queuer}
}

func (a *mqttDownlinkAdapter) EnqueueFromMQTT(ctx context.Context, tenantID int64, orgID *uuid.UUID,
	epEUI uint64, payload []byte, confirmed bool) (uint64, error) {
	queID := nextMQTTQueueID()
	responseExp := confirmed

	resultQueID, err := a.queuer.QueueDownlink(ctx, tenantID, orgID, &mioty.DLDataQueue{
		EpEui:       epEUI,
		QueId:       queID,
		UserData:    [][]byte{payload},
		ResponseExp: &responseExp,
	})
	if err != nil {
		return 0, err
	}
	return resultQueID, nil
}

// SCACIDownlinkServer is the subset of scaci.Server needed for downlink queueing.
type SCACIDownlinkServer interface {
	QueueDownlinkInternal(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue) (*scaci.DLDataQueueResult, error)
}

// scaciDownlinkQueuer wraps SCACIDownlinkServer to satisfy DownlinkQueuer.
type scaciDownlinkQueuer struct {
	server SCACIDownlinkServer
}

// NewSCACIDownlinkQueuer creates a DownlinkQueuer that delegates to scaci.Server.QueueDownlinkInternal.
func NewSCACIDownlinkQueuer(server SCACIDownlinkServer) DownlinkQueuer {
	return &scaciDownlinkQueuer{server: server}
}

func (q *scaciDownlinkQueuer) QueueDownlink(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue) (uint64, error) {
	result, err := q.server.QueueDownlinkInternal(ctx, tenantID, orgID, req)
	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, bssci.NewCatalogError(bssci.ErrDLQueueNilResult, bssci.POSIX_EIO)
	}
	return result.QueID, nil
}
