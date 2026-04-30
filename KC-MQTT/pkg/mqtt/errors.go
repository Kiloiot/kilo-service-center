package mqtt

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
//
// Deduplication:
//   - 9 unique messages extracted from 16 hardcoded occurrences
//   - Shared messages reused across TLS setup, connection, publish, subscribe
const (
	// ========================================================================
	// TLS & Certificate Errors (3 constants)
	// ========================================================================

	ErrReadCACertFailed     = "failed to read CA certificate"
	ErrParseCACertFailed    = "failed to parse CA certificate"
	ErrLoadClientCertFailed = "failed to load client certificate"

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
	// Command Handler Errors (7 constants)
	// ========================================================================

	// ErrCommandInvalidTopic is returned when command topic has wrong segment count
	ErrCommandInvalidTopic = "invalid command topic format"

	// ErrCommandInvalidOrgUUID is returned when organization UUID in command topic is invalid
	ErrCommandInvalidOrgUUID = "invalid organization UUID in command topic"

	// ErrCommandInvalidEUIHex is returned when endpoint EUI hex in command topic is invalid
	ErrCommandInvalidEUIHex = "invalid endpoint EUI hex in command topic"

	// ErrCommandPayloadEmpty is returned when decoded command payload is empty
	ErrCommandPayloadEmpty = "command payload is empty"

	// ErrCommandPayloadTooLarge is returned when decoded command payload exceeds MaxPayloadSize
	ErrCommandPayloadTooLarge = "command payload exceeds maximum size"

	// ErrCommandOrgResolveFailed is returned when organization lookup fails for command
	ErrCommandOrgResolveFailed = "failed to resolve organization for command"

	// ErrCommandEnqueueFailed is returned when downlink enqueue fails for command
	ErrCommandEnqueueFailed = "failed to enqueue downlink from command"

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

	// ErrCommandPayloadDecodeFailed is logged when JSON unmarshal of command payload fails
	ErrCommandPayloadDecodeFailed = "failed to decode command payload JSON"

	// ErrCommandBase64DecodeFailed is logged when base64 decoding of command data fails
	ErrCommandBase64DecodeFailed = "failed to decode base64 command data"

	// LogCommandDownlinkEnqueued is logged when a command/down message is successfully enqueued
	LogCommandDownlinkEnqueued = "MQTT command/down enqueued"

	// ========================================================================
	// Log Messages (2 constants)
	// ========================================================================

	LogMQTTResubscribeFailed = "Failed to resubscribe"
	LogMQTTConnectionLost    = "MQTT connection lost"
)
