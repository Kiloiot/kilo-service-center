package mqtt

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// Publisher defines the interface for MQTT publish/subscribe operations.
// Implementations must use QoS constants from constants.go (UplinkQoS, DownlinkQoS, etc.)
//
// Consumers should depend on this interface, not the concrete Client type.
// This enables dependency injection and simplifies testing.
type Publisher interface {
	// Connect establishes connection to the MQTT broker.
	// Context is used for cancellation and timeout control.
	Connect(ctx context.Context) error

	// Disconnect gracefully closes the MQTT connection.
	// Context is used for graceful shutdown timeout.
	Disconnect(ctx context.Context)

	// Publish sends a message to the specified topic.
	// qos parameter should use constants from constants.go:
	//   - QoSAtMostOnce (0)  - Fire and forget
	//   - QoSAtLeastOnce (1) - At least one delivery guaranteed
	//   - QoSExactlyOnce (2) - Exactly one delivery guaranteed
	//
	// Context is used for timeout control and tenant/org metadata propagation.
	Publish(ctx context.Context, topic string, qos byte, retained bool, payload interface{}) error

	// Subscribe registers a handler for messages on the specified topic.
	// qos parameter should use constants from constants.go.
	// Context is used for timeout control.
	Subscribe(ctx context.Context, topic string, qos byte, handler MessageHandler) error

	// Unsubscribe removes subscription from specified topics.
	// Context is used for timeout control.
	Unsubscribe(ctx context.Context, topics ...string) error

	// IsConnected returns true if client is currently connected to broker.
	IsConnected() bool
}

// MessageHandler processes incoming MQTT messages with context awareness.
// This replaces Paho's mqtt.MessageHandler signature to support tenant/org propagation.
//
// The context parameter allows:
// - Tenant ID extraction via pkgcontext.GetTenantID(ctx)
// - Organization ID extraction via pkgcontext.GetOrganizationID(ctx)
// - User ID extraction via pkgcontext.GetUserID(ctx)
// - Cancellation and timeout control
type MessageHandler func(ctx context.Context, topic string, payload []byte)

// ClientFactory creates Publisher instances for dependency injection.
// This factory pattern enables:
// - Swapping implementations (Paho, alternate MQTT libraries)
// - Mocking in unit tests
// - Configuration-driven client creation
type ClientFactory interface {
	NewClient(cfg *config.MQTTConfig, log logger.Logger) (Publisher, error)
}

// DeviceEventPublisher abstracts device event publishing to MQTT topics.
// This allows swapping MQTT with other messaging backends (e.g., Kafka, NATS).
type DeviceEventPublisher interface {
	// PublishDeviceEvent publishes a device event to the canonical topic.
	// Uses UplinkQoS for "up" events, EventsQoS for all other event types.
	// Returns error if orgUUID or epEUIHex are empty (prevents malformed topics).
	PublishDeviceEvent(ctx context.Context, orgUUID string, epEUIHex string, eventType string, payload []byte) error
}
