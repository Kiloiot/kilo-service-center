package mqtt

import (
	"fmt"
	"time"
)

// Topic format templates for MIOTY MQTT message publishing
const (
	// topicDeviceEventFormat defines canonical device event topics
	// Format: prefix/orgUUID/device/epEUIHex/event/eventType
	topicDeviceEventFormat = "%s/%s/device/%s/event/%s"

	// BrokerURLTCPFormat and BrokerURLSSLFormat render the broker address for
	// plain and TLS connections.
	BrokerURLTCPFormat = "tcp://%s:%d"
	BrokerURLSSLFormat = "ssl://%s:%d"

	// Command/down topic segment offsets counted from the end of the topic:
	// .../orgUUID/device/epEUIHex/command/down
	CommandTopicOrgUUIDOffset = 5
	CommandTopicEpEUIOffset   = 3

	// EnvironmentProduction gates the TLS-verification safety check.
	EnvironmentProduction = "production"

	// TopicDeviceCommandDownFormat defines device command/down topics for inbound downlinks
	// Format: prefix/orgUUID/device/epEUIHex/command/down
	TopicDeviceCommandDownFormat = "%s/%s/device/%s/command/down"
)

// MQTT QoS (Quality of Service) levels per MQTT 3.1.1 specification
const (
	// QoSAtMostOnce - Fire and forget delivery (QoS 0)
	QoSAtMostOnce = 0

	// QoSAtLeastOnce - Acknowledged delivery (QoS 1)
	QoSAtLeastOnce = 1

	// QoSExactlyOnce - Assured delivery (QoS 2)
	QoSExactlyOnce = 2
)

// Application-specific QoS assignments for MIOTY message types
const (
	// UplinkQoS for critical uplink data delivery
	UplinkQoS = QoSAtLeastOnce

	// DownlinkQoS for critical downlink control messages
	DownlinkQoS = QoSAtLeastOnce

	// EventsQoS for non-critical system events
	EventsQoS = QoSAtMostOnce
)

// Connection Timeouts

// DefaultPingTimeout is the MQTT keep-alive ping response deadline.
const DefaultPingTimeout = 10 * time.Second

// DefaultConnectTimeout is the maximum time to wait for broker connection.
const DefaultConnectTimeout = 30 * time.Second

// DefaultDisconnectQuiesce is the time (in milliseconds) to wait for pending
// messages to be sent before forcibly closing the connection.
const DefaultDisconnectQuiesce = 250 // milliseconds

// Operation Timeouts

// DefaultPublishTimeout is the maximum time to wait for publish acknowledgment.
const DefaultPublishTimeout = 5 * time.Second

// DefaultSubscribeTimeout is the maximum time to wait for subscription confirmation.
const DefaultSubscribeTimeout = 5 * time.Second

// DefaultUnsubscribeTimeout is the maximum time to wait for unsubscription confirmation.
const DefaultUnsubscribeTimeout = 5 * time.Second

// Device event type constants for canonical topic contract
const (
	DeviceEventUp               = "up"
	DeviceEventAttach           = "attach"
	DeviceEventDetach           = "detach"
	DeviceEventDownlinkResult   = "downlink_result"
	DeviceEventDownlinkQueued   = "downlink_queued"
	DeviceEventDownlinkRejected = "downlink_rejected"
)

// topicDeviceCommandDownWildcard is TopicDeviceCommandDownFormat with the org and endpoint segments wildcarded.
var topicDeviceCommandDownWildcard = fmt.Sprintf(TopicDeviceCommandDownFormat, "%s", "+", "+")

// Topic builder helpers

// DeviceEventTopic constructs a canonical device event topic using hex EUI string segment
// Format: prefix/orgUUID/device/epEUIHex/event/eventType
func DeviceEventTopic(prefix, orgID, epEUIHex, eventType string) string {
	return fmt.Sprintf(topicDeviceEventFormat, prefix, orgID, epEUIHex, eventType)
}

// DeviceCommandDownWildcardTopic constructs a wildcard subscription for all command/down topics
// Format: prefix/+/device/+/command/down
func DeviceCommandDownWildcardTopic(prefix string) string {
	return fmt.Sprintf(topicDeviceCommandDownWildcard, prefix)
}
