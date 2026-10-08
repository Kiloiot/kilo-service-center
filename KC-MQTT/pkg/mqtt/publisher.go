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

// PublishDeviceEvent publishes a device event to the canonical topic at the
// QoS of its event type.
func (p *TopicPublisher) PublishDeviceEvent(ctx context.Context, orgUUID string, epEUIHex string, eventType string, payload []byte) error {
	if orgUUID == "" {
		return fmt.Errorf("%s", ErrDeviceEventEmptyOrgUUID)
	}
	if epEUIHex == "" {
		return fmt.Errorf("%s", ErrDeviceEventEmptyEUIHex)
	}
	topic := DeviceEventTopic(p.prefix, orgUUID, epEUIHex, eventType)
	return p.client.Publish(ctx, topic, deviceEventQoS(eventType), false, payload)
}

// deviceEventQoS is the QoS a device event of eventType is published at.
func deviceEventQoS(eventType string) byte {
	switch eventType {
	case DeviceEventUp:
		return UplinkQoS
	case DeviceEventDownlinkQueued, DeviceEventDownlinkRejected, DeviceEventDownlinkResult:
		return DownlinkEventsQoS
	default:
		return EventsQoS
	}
}
