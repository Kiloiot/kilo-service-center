package bssciservices

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-MQTT/pkg/mqtt"
	"github.com/google/uuid"
)

// DownlinkQueuer abstracts SCACI downlink queueing at the service layer; the
// MQTT command's ref and deadline are stored with the downlink.
type DownlinkQueuer interface {
	QueueDownlink(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue, command storage.DownlinkCommand) (uint64, error)
}

// CommandRefs tells whether the organization already queued a downlink for
// the endpoint under an MQTT command's ref.
type CommandRefs interface {
	CommandQueued(ctx context.Context, command storage.DownlinkCommandRef) (bool, error)
}

// mqttDownlinkAdapter wraps DownlinkQueuer and the command refs to satisfy
// mqtt.DownlinkEnqueuer.
type mqttDownlinkAdapter struct {
	queuer DownlinkQueuer
	refs   CommandRefs
}

// NewMQTTDownlinkAdapter creates an adapter that satisfies mqtt.DownlinkEnqueuer.
func NewMQTTDownlinkAdapter(queuer DownlinkQueuer, refs CommandRefs) (mqtt.DownlinkEnqueuer, error) {
	switch {
	case queuer == nil:
		return nil, ErrNilQueuer
	case refs == nil:
		return nil, ErrNilCommandRefs
	}
	return &mqttDownlinkAdapter{queuer: queuer, refs: refs}, nil
}

// CommandQueued tells whether the organization already queued a downlink for
// the endpoint under the command's ref.
func (a *mqttDownlinkAdapter) CommandQueued(ctx context.Context, command storage.DownlinkCommandRef) (bool, error) {
	return a.refs.CommandQueued(ctx, command)
}

// EnqueueFromMQTT queues the downlink with the command's ref and deadline and
// returns the service center queue id it was persisted under; MQTT carries no
// Application Center queue id.
func (a *mqttDownlinkAdapter) EnqueueFromMQTT(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue, command storage.DownlinkCommand) (uint64, error) {
	queID, err := a.queuer.QueueDownlink(ctx, tenantID, orgID, req, command)
	if err != nil {
		return 0, withRefusal(err)
	}
	return queID, nil
}

// commandOutcomes names the core tokens that answer an MQTT command in the
// command's own terms: a ref already queued is a repeat of an accepted command,
// and a deadline that passed before queueing is the command's expiry.
var commandOutcomes = map[string]error{
	scaci.ErrDownlinkCommandRefQueued: mqtt.ErrCommandAlreadyQueued,
	scaci.ErrDownlinkDeadlineElapsed:  mqtt.RefusalCommandExpired,
}

// withRefusal names a core refusal by its catalog token and message, so the
// MQTT publisher can tell why its downlink was not queued.
func withRefusal(err error) error {
	var queueErr *scaci.DLDataQueueError
	if errors.As(err, &queueErr) {
		if outcome, named := commandOutcomes[queueErr.Token]; named {
			return fmt.Errorf("%w: %w", outcome, err)
		}
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
	QueueDownlinkInternal(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue, command storage.DownlinkCommand) (*scaci.DLDataQueueResult, error)
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

func (q *scaciDownlinkQueuer) QueueDownlink(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue, command storage.DownlinkCommand) (uint64, error) {
	result, err := q.server.QueueDownlinkInternal(ctx, tenantID, orgID, req, command)
	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, bssci.NewCatalogError(bssci.ErrDLQueueNilResult, bssci.POSIX_EIO)
	}
	return result.QueID, nil
}
