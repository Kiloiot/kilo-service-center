package mqtt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
)

// DownlinkEnqueuer abstracts SCACI downlink queueing for MQTT command handler.
type DownlinkEnqueuer interface {
	EnqueueFromMQTT(ctx context.Context, tenantID int64, orgID *uuid.UUID,
		epEUI uint64, payload []byte, confirmed bool) (uint64, error)
}

// TenantLookup resolves organization UUIDs to tenant IDs.
type TenantLookup interface {
	LookupTenant(ctx context.Context, orgUUID uuid.UUID) (int64, error)
}

// CommandHandler subscribes to MQTT command/down topics and enqueues downlinks via SCACI.
type CommandHandler struct {
	client   Publisher
	enqueuer DownlinkEnqueuer
	lookup   TenantLookup
	logger   logger.Logger
	prefix   string
}

// NewCommandHandler creates a handler that bridges MQTT commands to SCACI downlink queue.
func NewCommandHandler(client Publisher, enqueuer DownlinkEnqueuer, lookup TenantLookup, log logger.Logger, prefix string) *CommandHandler {
	return &CommandHandler{
		client:   client,
		enqueuer: enqueuer,
		lookup:   lookup,
		logger:   log,
		prefix:   prefix,
	}
}

// commandPayload is the JSON payload for command/down messages.
type commandPayload struct {
	Data      string `json:"data"`      // Base64-encoded payload
	Confirmed bool   `json:"confirmed"` // Request delivery confirmation
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
		h.logger.ErrorContext(ctx, ErrCommandSubscribeFailed, "topic", topic, "error", err)
		return
	}
	h.logger.InfoContext(ctx, LogCommandSubscribed, "topic", topic)

	// Block until context is cancelled
	<-ctx.Done()

	if err := h.client.Unsubscribe(ctx, topic); err != nil {
		h.logger.WarnContext(ctx, ErrCommandUnsubscribeFailed, "topic", topic, "error", err)
	}
}

func (h *CommandHandler) handleMessage(ctx context.Context, topic string, rawPayload []byte) {
	// Parse topic: {prefix}/{orgUUID}/device/{epEUIHex}/command/down
	segments := strings.Split(topic, "/")
	if len(segments) < 6 {
		h.logger.Warn(ErrCommandInvalidTopic, "topic", topic)
		return
	}

	// Extract orgUUID (segment after prefix)
	orgUUIDStr := segments[len(segments)-5]
	parsedOrgUUID, err := uuid.Parse(orgUUIDStr)
	if err != nil || parsedOrgUUID == uuid.Nil {
		h.logger.Warn(ErrCommandInvalidOrgUUID, "orgUUID", orgUUIDStr, "topic", topic)
		return
	}

	// Extract epEUIHex (segment after "device")
	epEUIHex := segments[len(segments)-3]
	epEUI, err := strconv.ParseUint(epEUIHex, 16, 64)
	if err != nil {
		h.logger.Warn(ErrCommandInvalidEUIHex, "epEUIHex", epEUIHex, "topic", topic)
		return
	}

	// Stage 1 validation: raw payload size
	if len(rawPayload) == 0 {
		h.logger.Warn(ErrCommandPayloadEmpty, "topic", topic)
		return
	}
	if len(rawPayload) > config.MaxMessageSize {
		h.logger.Warn(ErrCommandPayloadTooLarge, "topic", topic, "size", len(rawPayload))
		return
	}

	// Decode JSON payload
	var cmd commandPayload
	if err := json.Unmarshal(rawPayload, &cmd); err != nil {
		h.logger.Warn(ErrCommandPayloadDecodeFailed, "topic", topic, "error", err)
		return
	}

	// Decode base64 data
	decodedPayload, err := base64.StdEncoding.DecodeString(cmd.Data)
	if err != nil {
		h.logger.Warn(ErrCommandBase64DecodeFailed, "topic", topic, "error", err)
		return
	}

	// Stage 2 validation: decoded payload
	if len(decodedPayload) == 0 {
		h.logger.Warn(ErrCommandPayloadEmpty, "topic", topic)
		return
	}
	if len(decodedPayload) > config.MaxPayloadSize {
		h.logger.Warn(ErrCommandPayloadTooLarge, "topic", topic, "size", len(decodedPayload))
		return
	}

	// Resolve org→tenant
	tenantID, err := h.lookup.LookupTenant(ctx, parsedOrgUUID)
	if err != nil {
		h.logger.Warn(ErrCommandOrgResolveFailed,
			"orgUUID", orgUUIDStr, "error", err)
		return
	}

	// Build tenant-enriched context
	enrichedCtx := pkgcontext.WithTenantID(ctx, tenantID)
	enrichedCtx = pkgcontext.WithOrganizationID(enrichedCtx, parsedOrgUUID)

	// Enqueue downlink
	queID, err := h.enqueuer.EnqueueFromMQTT(enrichedCtx, tenantID, &parsedOrgUUID,
		epEUI, decodedPayload, cmd.Confirmed)
	if err != nil {
		h.logger.Warn(ErrCommandEnqueueFailed,
			"epEui", fmt.Sprintf("%016x", epEUI), "error", err)
		return
	}

	h.logger.Info(LogCommandDownlinkEnqueued,
		"epEui", fmt.Sprintf("%016x", epEUI),
		"queId", queID,
		"confirmed", cmd.Confirmed)
}
