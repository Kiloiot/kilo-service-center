package mqtt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
)

// DownlinkRefusal is why a downlink command was not queued; the handler
// reports its code and message on event/downlink_rejected.
type DownlinkRefusal struct {
	Code    string
	Message string
}

func (r *DownlinkRefusal) Error() string {
	return fmt.Sprintf(errFmtDownlinkRefusal, r.Code, r.Message)
}

// DownlinkEnqueuer abstracts SCACI downlink queueing for MQTT command handler;
// the command's ref is stored with the downlink and carried by its results,
// and its deadline bounds the downlink's lifetime. A refusal it can name is
// returned wrapping a *DownlinkRefusal, and a ref already queued wraps
// ErrCommandAlreadyQueued. CommandQueued tells whether the organization
// already queued a downlink for the endpoint under the ref.
type DownlinkEnqueuer interface {
	EnqueueFromMQTT(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue, command storage.DownlinkCommand) (uint64, error)
	CommandQueued(ctx context.Context, command storage.DownlinkCommandRef) (bool, error)
}

// TenantLookup resolves organization UUIDs to tenant IDs.
type TenantLookup interface {
	LookupTenant(ctx context.Context, orgUUID uuid.UUID) (int64, error)
}

// CommandHandler subscribes to MQTT command/down topics, enqueues downlinks via
// SCACI and answers every addressable command on the organization's
// event/downlink_queued or event/downlink_rejected topic.
type CommandHandler struct {
	client   Publisher
	events   DeviceEventPublisher
	enqueuer DownlinkEnqueuer
	lookup   TenantLookup
	logger   logger.Logger
	prefix   string
}

// NewCommandHandler creates a handler that bridges MQTT commands to SCACI downlink queue.
func NewCommandHandler(client Publisher, enqueuer DownlinkEnqueuer, lookup TenantLookup, log logger.Logger, prefix string) *CommandHandler {
	return &CommandHandler{
		client:   client,
		events:   NewPublisher(client, prefix),
		enqueuer: enqueuer,
		lookup:   lookup,
		logger:   log,
		prefix:   prefix,
	}
}

// downlinkQueuedEvent is the event/downlink_queued body.
type downlinkQueuedEvent struct {
	EpEui string `json:"epEui"`
	QueID uint64 `json:"queId"`
	Ref   string `json:"ref,omitempty"`
}

// downlinkRejectedEvent is the event/downlink_rejected body.
type downlinkRejectedEvent struct {
	EpEui   string `json:"epEui"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Ref     string `json:"ref,omitempty"`
}

// commandTarget is the organization and endpoint a command/down topic addresses.
type commandTarget struct {
	org      uuid.UUID
	epEUI    uint64
	epEUIHex string
}

// Start subscribes to the command/down wildcard topic and begins processing.
func (h *CommandHandler) Start(ctx context.Context) {
	if h.client == nil || h.enqueuer == nil || h.lookup == nil || h.logger == nil {
		if h.logger != nil {
			h.logger.ErrorContext(ctx, ErrCommandHandlerMissingDeps)
		}
		return
	}

	topic := DeviceCommandDownWildcardTopic(h.prefix)
	if err := h.client.Subscribe(ctx, topic, DownlinkQoS, h.handleMessage); err != nil {
		h.logger.ErrorContext(ctx, ErrCommandSubscribeFailed, logger.FieldTopic, topic, logger.FieldError, err)
		return
	}
	h.logger.InfoContext(ctx, LogCommandSubscribed, logger.FieldTopic, topic)

	// Block until context is cancelled
	<-ctx.Done()

	if err := h.client.Unsubscribe(ctx, topic); err != nil {
		h.logger.WarnContext(ctx, ErrCommandUnsubscribeFailed, logger.FieldTopic, topic, logger.FieldError, err)
	}
}

// handleMessage answers a command/down. A ref the organization already
// queued for the endpoint is recognized as soon as the tenant is known, before
// the payload and the endpoint are checked, so a repeat of an accepted
// command is never refused, whatever changed since.
func (h *CommandHandler) handleMessage(ctx context.Context, topic string, rawPayload []byte) {
	target, ok := h.parseTopic(ctx, topic)
	if !ok {
		return
	}
	cmd, refReadable, decodeErr := decodeCommand(rawPayload)
	if decodeErr != nil && !refReadable {
		h.reject(ctx, target, cmd.Ref, decodeErr)
		return
	}
	tenantID, ok := h.admit(ctx, target, cmd.Ref, decodeErr)
	if !ok {
		return
	}
	req, err := cmd.downlink(target.epEUI)
	if err != nil {
		h.reject(ctx, target, cmd.Ref, err)
		return
	}
	command, err := cmd.command()
	if err != nil {
		h.reject(ctx, target, cmd.Ref, err)
		return
	}

	enrichedCtx := pkgcontext.WithTenantID(ctx, tenantID)
	enrichedCtx = pkgcontext.WithOrganizationID(enrichedCtx, target.org)
	queID, err := h.enqueuer.EnqueueFromMQTT(enrichedCtx, tenantID, &target.org, req, command)
	if errors.Is(err, ErrCommandAlreadyQueued) {
		h.logRepeat(ctx, target, cmd.Ref)
		return
	}
	if err != nil {
		h.reject(ctx, target, cmd.Ref, err)
		return
	}
	h.logger.InfoContext(ctx, LogCommandDownlinkEnqueued,
		logger.FieldEpEui, target.epEUIHex,
		logger.FieldQueID, queID,
		logger.FieldConfirmed, cmd.Confirmed)
	h.publish(ctx, target, DeviceEventDownlinkQueued, downlinkQueuedEvent{EpEui: target.epEUIHex, QueID: queID, Ref: cmd.Ref})
}

// parseTopic reads {prefix}/{orgUUID}/device/{epEUIHex}/command/down; a topic
// without both identities has no event topic to answer on.
func (h *CommandHandler) parseTopic(ctx context.Context, topic string) (commandTarget, bool) {
	segments := strings.Split(topic, "/")
	if len(segments) < CommandTopicOrgUUIDOffset+1 {
		h.logger.WarnContext(ctx, ErrCommandInvalidTopic, logger.FieldTopic, topic)
		return commandTarget{}, false
	}
	orgUUIDStr := segments[len(segments)-CommandTopicOrgUUIDOffset]
	org, err := uuid.Parse(orgUUIDStr)
	if err != nil || org == uuid.Nil {
		h.logger.WarnContext(ctx, ErrCommandInvalidOrgUUID, logger.FieldOrgUUID, orgUUIDStr, logger.FieldTopic, topic)
		return commandTarget{}, false
	}
	epEUIHex := segments[len(segments)-CommandTopicEpEUIOffset]
	epEUI, err := strconv.ParseUint(epEUIHex, 16, 64)
	if err != nil {
		h.logger.WarnContext(ctx, ErrCommandInvalidEUIHex, logger.FieldEpEUIHex, epEUIHex, logger.FieldTopic, topic)
		return commandTarget{}, false
	}
	return commandTarget{org: org, epEUI: epEUI, epEUIHex: mioty.FormatEUI64Lower(epEUI)}, true
}

// reject reports a refused command to its organization; an error that names
// no refusal is reported as an enqueue failure.
func (h *CommandHandler) reject(ctx context.Context, target commandTarget, ref string, err error) {
	var refusal *DownlinkRefusal
	if !errors.As(err, &refusal) {
		refusal = refusalEnqueueFailed
	}
	h.logger.WarnContext(ctx, LogCommandDownlinkRejected,
		logger.FieldEpEui, target.epEUIHex,
		logger.FieldCode, refusal.Code,
		logger.FieldError, err)
	h.publish(ctx, target, DeviceEventDownlinkRejected, downlinkRejectedEvent{
		EpEui: target.epEUIHex, Code: refusal.Code, Message: refusal.Message, Ref: ref,
	})
}

// publish sends a command outcome to the addressed organization's event topic.
func (h *CommandHandler) publish(ctx context.Context, target commandTarget, eventType string, event interface{}) {
	payload, err := json.Marshal(event)
	if err == nil {
		err = h.events.PublishDeviceEvent(ctx, target.org.String(), target.epEUIHex, eventType, payload)
	}
	if err != nil {
		h.logger.WarnContext(ctx, LogCommandEventPublishFailed,
			logger.FieldEvent, eventType, logger.FieldEpEui, target.epEUIHex, logger.FieldError, err)
	}
}
