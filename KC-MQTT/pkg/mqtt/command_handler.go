package mqtt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
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
// ref is stored with the downlink and carried by its results. A refusal it
// can name is returned wrapping a *DownlinkRefusal.
type DownlinkEnqueuer interface {
	EnqueueFromMQTT(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue, ref string) (uint64, error)
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

// commandPayload is the JSON payload for command/down messages; data and
// entries are mutually exclusive.
type commandPayload struct {
	Data         *string        `json:"data"`
	Entries      []commandEntry `json:"entries"`
	Confirmed    bool           `json:"confirmed"`
	Format       *uint8         `json:"format"`
	Prio         *float32       `json:"prio"`
	ResponsePrio *bool          `json:"responsePrio"`
	DlWindReq    *bool          `json:"dlWindReq"`
	ExpOnly      *bool          `json:"expOnly"`
	DlRxStatQry  *bool          `json:"dlRxStatQry"`
	Ref          string         `json:"ref"`
}

// commandEntry is the payload for one endpoint packet counter of a
// counter-dependent downlink (BSSCI §3.12.1).
type commandEntry struct {
	PacketCnt *uint32 `json:"packetCnt"`
	Data      string  `json:"data"`
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

func (h *CommandHandler) handleMessage(ctx context.Context, topic string, rawPayload []byte) {
	target, ok := h.parseTopic(ctx, topic)
	if !ok {
		return
	}
	cmd, err := decodeCommand(rawPayload)
	if err != nil {
		h.reject(ctx, target, cmd.Ref, err)
		return
	}
	req, err := cmd.downlink(target.epEUI)
	if err != nil {
		h.reject(ctx, target, cmd.Ref, err)
		return
	}
	tenantID, err := h.lookup.LookupTenant(ctx, target.org)
	if err != nil {
		h.reject(ctx, target, cmd.Ref, fmt.Errorf("%w: %w", refusalOrgUnresolved, err))
		return
	}

	enrichedCtx := pkgcontext.WithTenantID(ctx, tenantID)
	enrichedCtx = pkgcontext.WithOrganizationID(enrichedCtx, target.org)
	queID, err := h.enqueuer.EnqueueFromMQTT(enrichedCtx, tenantID, &target.org, req, cmd.Ref)
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

// decodeCommand parses the message; a field of the wrong type, and a ref
// beyond the length the queue stores, still yield the rest of the command so
// its ref can be echoed.
func decodeCommand(rawPayload []byte) (commandPayload, error) {
	var cmd commandPayload
	switch {
	case len(rawPayload) == 0:
		return cmd, refusalEmptyPayload
	case len(rawPayload) > config.MaxMessageSize:
		return cmd, &DownlinkRefusal{Code: RejectCodeMessageTooLarge, Message: fmt.Sprintf(RejectMsgMessageTooLargeFmt, config.MaxMessageSize)}
	}
	err := json.Unmarshal(rawPayload, &cmd)
	var typeErr *json.UnmarshalTypeError
	switch {
	case err == nil && len(cmd.Ref) > storage.MaxDownlinkRefBytes:
		return cmd, refusalRefTooLong
	case err == nil:
		return cmd, nil
	case errors.As(err, &typeErr) && typeErr.Field != "":
		return cmd, fmt.Errorf("%w: %w", &DownlinkRefusal{Code: RejectCodeInvalidField, Message: fmt.Sprintf(RejectMsgInvalidFieldFmt, typeErr.Field)}, err)
	default:
		return commandPayload{}, fmt.Errorf("%w: %w", refusalInvalidJSON, err)
	}
}

// downlink maps the command onto the canonical dlDataQue request (BSSCI §3.12.1).
func (c commandPayload) downlink(epEUI uint64) (*mioty.DLDataQueue, error) {
	userData, packetCnts, err := c.payloads()
	if err != nil {
		return nil, err
	}
	responseExp := c.Confirmed
	return &mioty.DLDataQueue{
		EpEui:        epEUI,
		CntDepend:    packetCnts != nil,
		PacketCnt:    packetCnts,
		UserData:     userData,
		Format:       c.Format,
		Prio:         c.Prio,
		ResponseExp:  &responseExp,
		ResponsePrio: c.ResponsePrio,
		DlWindReq:    c.DlWindReq,
		ExpOnly:      c.ExpOnly,
		DlRxStatQry:  c.DlRxStatQry,
	}, nil
}

// payloads decodes the single data payload, or one payload per packet counter
// for counter-dependent entries; empty data is a pure acknowledgement.
func (c commandPayload) payloads() (mioty.DownlinkUserData, []uint32, error) {
	switch {
	case c.Data != nil && c.Entries != nil:
		return nil, nil, refusalDataWithEntries
	case c.Data != nil:
		payload, err := decodeDownlinkPayload(*c.Data)
		if err != nil {
			return nil, nil, err
		}
		return mioty.DownlinkUserData{payload}, nil, nil
	case c.Entries == nil:
		return nil, nil, refusalMissingData
	case len(c.Entries) == 0:
		return nil, nil, refusalEmptyEntries
	}
	userData := make(mioty.DownlinkUserData, 0, len(c.Entries))
	packetCnts := make([]uint32, 0, len(c.Entries))
	seen := make(map[uint32]bool, len(c.Entries))
	for _, entry := range c.Entries {
		if entry.PacketCnt == nil {
			return nil, nil, refusalMissingPacketCnt
		}
		if seen[*entry.PacketCnt] {
			return nil, nil, refusalDuplicatePacketCnt
		}
		seen[*entry.PacketCnt] = true
		payload, err := decodeDownlinkPayload(entry.Data)
		if err != nil {
			return nil, nil, err
		}
		userData = append(userData, payload)
		packetCnts = append(packetCnts, *entry.PacketCnt)
	}
	return userData, packetCnts, nil
}

// decodeDownlinkPayload decodes one base64 payload within the radio downlink
// limit (radio protocol §3.6.6.3).
func decodeDownlinkPayload(encoded string) ([]byte, error) {
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", refusalInvalidBase64, err)
	}
	if len(payload) > mioty.MaxDLUserDataBytes {
		return nil, refusalPayloadTooLarge
	}
	return payload, nil
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
