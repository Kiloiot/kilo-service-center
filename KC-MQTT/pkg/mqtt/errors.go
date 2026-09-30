package mqtt

import (
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// Error message constants for KC-MQTT operations
//
// This file centralizes all MQTT client error messages to ensure consistency,
// enable future localization, and prevent string duplication across the MQTT package.
//
// Usage Pattern:
//
//	return nil, fmt.Errorf("%s: %w", ErrConnectionFailed, err)
//	return nil, errors.New(ErrParseCACertFailed)
//	c.logger.Error(LogMQTTResubscribeFailed, "topic", topic, "error", token.Error())
//
// Naming Convention:
//   - Err* prefix for error messages that wrap underlying errors
//   - LogMQTT* prefix for log messages (following package log convention)
//   - Use present tense for error states ("failed", "invalid")
const (
	// ========================================================================
	// TLS & Certificate Errors (3 constants)
	// ========================================================================

	ErrReadCACertFailed     = "failed to read CA certificate"
	ErrParseCACertFailed    = "failed to parse CA certificate"
	ErrLoadClientCertFailed = "failed to load client certificate"

	// ErrTLSVerifyDisabledInProduction is returned when insecure_skip_verify
	// is requested while KILOCENTER_ENV is production.
	ErrTLSVerifyDisabledInProduction = "TLS certificate verification cannot be disabled in production"

	// ========================================================================
	// MQTT Operation Errors (4 constants)
	// ========================================================================

	ErrConnectionFailed  = "mqtt connection failed"
	ErrPublishFailed     = "publish failed"
	ErrSubscribeFailed   = "subscribe failed"
	ErrUnsubscribeFailed = "unsubscribe failed"

	// ========================================================================
	// Configuration & Validation Errors (extracted from client.go:31)
	// ========================================================================

	// ErrMQTTConfigNil is returned when NewClient is called with nil config
	ErrMQTTConfigNil = "mqtt config is nil"

	// ========================================================================
	// Connection & Timeout Errors (extracted from client.go)
	// ========================================================================

	// ErrMQTTConnectionTimeout is returned when broker connection exceeds DefaultConnectTimeout
	ErrMQTTConnectionTimeout = "mqtt connection timeout"

	// ErrMQTTPublishTimeout is returned when publish acknowledgment exceeds DefaultPublishTimeout
	ErrMQTTPublishTimeout = "publish timeout"

	// ErrMQTTSubscribeTimeout is returned when subscription confirmation exceeds DefaultSubscribeTimeout
	ErrMQTTSubscribeTimeout = "subscribe timeout"

	// ErrMQTTUnsubscribeTimeout is returned when unsubscription confirmation exceeds DefaultUnsubscribeTimeout
	ErrMQTTUnsubscribeTimeout = "unsubscribe timeout"

	// ========================================================================
	// State Errors (extracted from client.go:154, 173, 197)
	// ========================================================================

	// ErrMQTTClientNotConnected is returned when attempting operations on disconnected client
	ErrMQTTClientNotConnected = "mqtt client not connected"

	// ========================================================================
	// Device Event Errors (2 constants)
	// ========================================================================

	// ErrDeviceEventEmptyOrgUUID is returned when PublishDeviceEvent is called with empty orgUUID
	ErrDeviceEventEmptyOrgUUID = "device event publish requires non-empty orgUUID"

	// ErrDeviceEventEmptyEUIHex is returned when PublishDeviceEvent is called with empty epEUIHex
	ErrDeviceEventEmptyEUIHex = "device event publish requires non-empty epEUIHex"

	// ========================================================================
	// Command Handler Errors
	// ========================================================================

	// ErrCommandInvalidTopic is returned when command topic has wrong segment count
	ErrCommandInvalidTopic = "invalid command topic format"

	// ErrCommandInvalidOrgUUID is returned when organization UUID in command topic is invalid
	ErrCommandInvalidOrgUUID = "invalid organization UUID in command topic"

	// ErrCommandInvalidEUIHex is returned when endpoint EUI hex in command topic is invalid
	ErrCommandInvalidEUIHex = "invalid endpoint EUI hex in command topic"

	// LogCommandDownlinkRejected is logged when a command/down message is refused; the
	// publisher learns the refusal on event/downlink_rejected
	LogCommandDownlinkRejected = "MQTT command/down rejected"

	// LogCommandEventPublishFailed is logged when a downlink_queued or downlink_rejected event cannot be published
	LogCommandEventPublishFailed = "failed to publish MQTT command/down outcome event"

	// ========================================================================
	// Command Handler Lifecycle Constants (7 constants)
	// ========================================================================

	// ErrCommandHandlerMissingDeps is logged when Start() detects nil collaborators
	ErrCommandHandlerMissingDeps = "command handler cannot start: missing required dependencies"

	// ErrCommandSubscribeFailed is logged when command/down topic subscription fails
	ErrCommandSubscribeFailed = "failed to subscribe to command/down topic"

	// LogCommandSubscribed is logged when command/down topic subscription succeeds
	LogCommandSubscribed = "subscribed to MQTT command/down topic"

	// ErrCommandUnsubscribeFailed is logged when command/down topic unsubscription fails
	ErrCommandUnsubscribeFailed = "failed to unsubscribe from command/down topic"

	// LogCommandDownlinkEnqueued is logged when a command/down message is successfully enqueued
	LogCommandDownlinkEnqueued = "MQTT command/down enqueued"

	// ========================================================================
	// Log Messages
	// ========================================================================

	// LogMQTTConnecting is logged when a broker connection attempt starts
	LogMQTTConnecting = "Connecting to MQTT broker"

	// LogMQTTConnected is logged when the broker connection succeeds
	LogMQTTConnected = "Successfully connected to MQTT broker"

	// LogMQTTDisconnecting is logged when the client starts a clean disconnect
	LogMQTTDisconnecting = "Disconnecting from MQTT broker"

	// LogMQTTMessagePublished is logged after a successful publish
	LogMQTTMessagePublished = "Published message"

	// LogMQTTSubscribed is logged after a successful topic subscription
	LogMQTTSubscribed = "Subscribed to topic"

	// LogMQTTUnsubscribed is logged after a successful topic unsubscription
	LogMQTTUnsubscribed = "Unsubscribed from topics"

	// LogMQTTClientConnected is logged from the Paho on-connect callback
	LogMQTTClientConnected = "MQTT client connected"

	// LogMQTTResubscribed is logged when a topic is restored after reconnect
	LogMQTTResubscribed = "Resubscribed to topic"

	// LogMQTTReconnecting is logged from the Paho reconnecting callback
	LogMQTTReconnecting = "MQTT client reconnecting"

	LogMQTTResubscribeFailed = "Failed to resubscribe"
	LogMQTTConnectionLost    = "MQTT connection lost"
)

// Downlink command refusals published on event/downlink_rejected: the code is
// the stable reason a client matches on, the message explains it.
const (
	RejectCodeEmptyPayload       = "mqtt.command.empty_payload"
	RejectMsgEmptyPayload        = "the command/down message is empty"
	RejectCodeMessageTooLarge    = "mqtt.command.message_too_large"
	RejectMsgMessageTooLargeFmt  = "the command/down message exceeds %d bytes"
	RejectCodeInvalidJSON        = "mqtt.command.invalid_json"
	RejectMsgInvalidJSON         = "the command/down message is not a JSON object"
	RejectCodeInvalidField       = "mqtt.command.invalid_field"
	RejectMsgInvalidFieldFmt     = "field %s has a value of the wrong type or range"
	RejectCodeMissingData        = "mqtt.command.missing_data"
	RejectMsgMissingData         = "the command needs data or entries"
	RejectCodeDataWithEntries    = "mqtt.command.data_with_entries"
	RejectMsgDataWithEntries     = "data and entries are mutually exclusive"
	RejectCodeEmptyEntries       = "mqtt.command.empty_entries"
	RejectMsgEmptyEntries        = "entries must hold at least one packet counter"
	RejectCodeMissingPacketCnt   = "mqtt.command.missing_packet_cnt"
	RejectMsgMissingPacketCnt    = "every entry needs a packetCnt"
	RejectCodeDuplicatePacketCnt = "mqtt.command.duplicate_packet_cnt"
	RejectMsgDuplicatePacketCnt  = "entries repeat a packetCnt"
	RejectCodeInvalidBase64      = "mqtt.command.invalid_base64"
	RejectMsgInvalidBase64       = "data is not valid base64"
	RejectCodePayloadTooLarge    = "mqtt.command.payload_too_large"
	RejectMsgPayloadTooLargeFmt  = "decoded data exceeds the %d-byte radio payload maximum"
	RejectCodeOrgUnresolved      = "mqtt.command.org_unresolved"
	RejectMsgOrgUnresolved       = "the organization in the topic could not be resolved"
	RejectCodeEnqueueFailed      = "mqtt.command.enqueue_failed"
	RejectMsgEnqueueFailed       = "the service center could not queue the downlink"
	RejectCodeRefTooLong         = "mqtt.command.ref_too_long"
	RejectMsgRefTooLongFmt       = "ref exceeds %d bytes"
	errFmtDownlinkRefusal        = "%s: %s"
)

// Refusals whose code and message never vary.
var (
	refusalEmptyPayload       = &DownlinkRefusal{Code: RejectCodeEmptyPayload, Message: RejectMsgEmptyPayload}
	refusalInvalidJSON        = &DownlinkRefusal{Code: RejectCodeInvalidJSON, Message: RejectMsgInvalidJSON}
	refusalMissingData        = &DownlinkRefusal{Code: RejectCodeMissingData, Message: RejectMsgMissingData}
	refusalDataWithEntries    = &DownlinkRefusal{Code: RejectCodeDataWithEntries, Message: RejectMsgDataWithEntries}
	refusalEmptyEntries       = &DownlinkRefusal{Code: RejectCodeEmptyEntries, Message: RejectMsgEmptyEntries}
	refusalMissingPacketCnt   = &DownlinkRefusal{Code: RejectCodeMissingPacketCnt, Message: RejectMsgMissingPacketCnt}
	refusalDuplicatePacketCnt = &DownlinkRefusal{Code: RejectCodeDuplicatePacketCnt, Message: RejectMsgDuplicatePacketCnt}
	refusalInvalidBase64      = &DownlinkRefusal{Code: RejectCodeInvalidBase64, Message: RejectMsgInvalidBase64}
	refusalPayloadTooLarge    = &DownlinkRefusal{Code: RejectCodePayloadTooLarge, Message: fmt.Sprintf(RejectMsgPayloadTooLargeFmt, mioty.MaxDLUserDataBytes)}
	refusalOrgUnresolved      = &DownlinkRefusal{Code: RejectCodeOrgUnresolved, Message: RejectMsgOrgUnresolved}
	refusalEnqueueFailed      = &DownlinkRefusal{Code: RejectCodeEnqueueFailed, Message: RejectMsgEnqueueFailed}
	refusalRefTooLong         = &DownlinkRefusal{Code: RejectCodeRefTooLong, Message: fmt.Sprintf(RejectMsgRefTooLongFmt, storage.MaxDownlinkRefBytes)}
)
