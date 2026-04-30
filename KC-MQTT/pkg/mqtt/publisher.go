package mqtt

import (
	"context"
	"fmt"
)

// TopicPublisher handles publishing MIOTY device events to MQTT topics
type TopicPublisher struct {
	client Publisher
	prefix string // Topic prefix, e.g., "mioty"
}

// NewPublisher creates a new MQTT publisher for device events.
// Returns DeviceEventPublisher interface to enable dependency injection and testing.
func NewPublisher(client Publisher, topicPrefix string) DeviceEventPublisher {
	return &TopicPublisher{
		client: client,
		prefix: topicPrefix,
	}
}

// PublishDeviceEvent publishes a device event to the canonical topic.
// Selects QoS based on event type: UplinkQoS for uplink data, EventsQoS for lifecycle events.
func (p *TopicPublisher) PublishDeviceEvent(ctx context.Context, orgUUID string, epEUIHex string, eventType string, payload []byte) error {
	if orgUUID == "" {
		return fmt.Errorf("%s", ErrDeviceEventEmptyOrgUUID)
	}
	if epEUIHex == "" {
		return fmt.Errorf("%s", ErrDeviceEventEmptyEUIHex)
	}
	topic := DeviceEventTopic(p.prefix, orgUUID, epEUIHex, eventType)
	qos := EventsQoS
	if eventType == DeviceEventUp {
		qos = UplinkQoS
	}
	return p.client.Publish(ctx, topic, byte(qos), false, payload)
}
