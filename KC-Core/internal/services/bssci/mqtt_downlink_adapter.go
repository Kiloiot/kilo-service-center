package bssciservices

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-MQTT/pkg/mqtt"
	"github.com/google/uuid"
)

// DownlinkQueuer abstracts SCACI downlink queueing at the service layer; ref
// is the MQTT command's correlation ref, stored with the downlink.
type DownlinkQueuer interface {
	QueueDownlink(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue, ref string) (uint64, error)
}

// mqttDownlinkAdapter wraps DownlinkQueuer to satisfy mqtt.DownlinkEnqueuer.
type mqttDownlinkAdapter struct {
	queuer DownlinkQueuer
}

// NewMQTTDownlinkAdapter creates an adapter that satisfies mqtt.DownlinkEnqueuer.
func NewMQTTDownlinkAdapter(queuer DownlinkQueuer) (mqtt.DownlinkEnqueuer, error) {
	if queuer == nil {
		return nil, ErrNilQueuer
	}
	return &mqttDownlinkAdapter{queuer: queuer}, nil
}

// EnqueueFromMQTT queues the downlink with the command's ref and returns the
// service center queue id it was persisted under; MQTT carries no Application
// Center queue id.
func (a *mqttDownlinkAdapter) EnqueueFromMQTT(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue, ref string) (uint64, error) {
	queID, err := a.queuer.QueueDownlink(ctx, tenantID, orgID, req, ref)
	if err != nil {
		return 0, withRefusal(err)
	}
	return queID, nil
}

// withRefusal names a core refusal by its catalog token and message, so the
// MQTT publisher can tell why its downlink was not queued.
func withRefusal(err error) error {
	var queueErr *scaci.DLDataQueueError
	if errors.As(err, &queueErr) {
		def := scaci.GetErrorDefinition(queueErr.Token)
		return fmt.Errorf("%w: %w", &mqtt.DownlinkRefusal{Code: def.Token, Message: def.Message}, err)
	}
	var catalogErr *bssci.CatalogError
	if errors.As(err, &catalogErr) {
		refusal := &mqtt.DownlinkRefusal{Code: catalogErr.Token, Message: bssci.ResolveErrorMessage(catalogErr.Token)}
		return fmt.Errorf("%w: %w", refusal, err)
	}
	return err
}

// SCACIDownlinkServer is the subset of scaci.Server needed for downlink queueing.
type SCACIDownlinkServer interface {
	QueueDownlinkInternal(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue, ref string) (*scaci.DLDataQueueResult, error)
}

// scaciDownlinkQueuer wraps SCACIDownlinkServer to satisfy DownlinkQueuer.
type scaciDownlinkQueuer struct {
	server SCACIDownlinkServer
}

// NewSCACIDownlinkQueuer creates a DownlinkQueuer that delegates to scaci.Server.QueueDownlinkInternal.
func NewSCACIDownlinkQueuer(server SCACIDownlinkServer) (DownlinkQueuer, error) {
	if server == nil {
		return nil, ErrNilSCACIServer
	}
	return &scaciDownlinkQueuer{server: server}, nil
}

func (q *scaciDownlinkQueuer) QueueDownlink(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue, ref string) (uint64, error) {
	result, err := q.server.QueueDownlinkInternal(ctx, tenantID, orgID, req, ref)
	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, bssci.NewCatalogError(bssci.ErrDLQueueNilResult, bssci.POSIX_EIO)
	}
	return result.QueID, nil
}
