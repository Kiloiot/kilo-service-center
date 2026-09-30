package bssciservices

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-MQTT/pkg/mqtt"
)

// MQTT payload field keys shared by the published event bodies.
const (
	mqttKeyEpEui     = "epEui"
	mqttKeyBsEui     = "bsEui"
	mqttKeyQueId     = "queId"
	mqttKeyTxTime    = "txTime"
	mqttKeyPacketCnt = "packetCnt"
	mqttKeyResult    = "result"
	mqttKeyRef       = "ref"
)

// MQTTAdapter bridges bssci.MQTTEventPublisher and the downlink result
// publisher onto mqtt.DeviceEventPublisher.PublishDeviceEvent: it converts
// uint64 EUIs to hex strings, builds the JSON payloads and delegates to KC-MQTT.
type MQTTAdapter struct {
	pub mqtt.DeviceEventPublisher
}

// NewMQTTAdapter creates the adapter over KC-MQTT; it satisfies bssci.MQTTEventPublisher.
func NewMQTTAdapter(pub mqtt.DeviceEventPublisher) *MQTTAdapter {
	return &MQTTAdapter{pub: pub}
}

// PublishUplink publishes an uplink on the organization's up topic.
func (a *MQTTAdapter) PublishUplink(ctx context.Context, orgUUID string, msg *mioty.ULDataMessage) error {
	payload, err := json.Marshal(newMQTTUplinkEvent(msg))
	if err != nil {
		return fmt.Errorf("%w: %w", errMarshalUplinkPayload, err)
	}
	return a.pub.PublishDeviceEvent(ctx, orgUUID, mioty.FormatEUI64Lower(msg.EpEui), mqtt.DeviceEventUp, payload)
}

// PublishAttach publishes an endpoint attach on the organization's topic.
func (a *MQTTAdapter) PublishAttach(ctx context.Context, orgUUID string, epEUI uint64, heardBy *uint64) error {
	epEUIHex := mioty.FormatEUI64Lower(epEUI)
	payload, err := json.Marshal(attachmentEvent(epEUIHex, mqtt.DeviceEventAttach, heardBy))
	if err != nil {
		return fmt.Errorf("%w: %w", errMarshalAttachPayload, err)
	}
	return a.pub.PublishDeviceEvent(ctx, orgUUID, epEUIHex, mqtt.DeviceEventAttach, payload)
}

// PublishDetach publishes an endpoint detach on the organization's topic.
func (a *MQTTAdapter) PublishDetach(ctx context.Context, orgUUID string, epEUI uint64, heardBy *uint64) error {
	epEUIHex := mioty.FormatEUI64Lower(epEUI)
	payload, err := json.Marshal(attachmentEvent(epEUIHex, mqtt.DeviceEventDetach, heardBy))
	if err != nil {
		return fmt.Errorf("%w: %w", errMarshalDetachPayload, err)
	}
	return a.pub.PublishDeviceEvent(ctx, orgUUID, epEUIHex, mqtt.DeviceEventDetach, payload)
}

// attachmentEvent is the body of an attach or detach event; bsEui names the
// base station that heard an over-the-air one and is absent for a decision
// taken in the service center.
func attachmentEvent(epEUIHex, event string, heardBy *uint64) map[string]interface{} {
	body := map[string]interface{}{mqttKeyEpEui: epEUIHex, "event": event}
	if heardBy != nil {
		body[mqttKeyBsEui] = mioty.FormatEUI64Lower(*heardBy)
	}
	return body
}

// PublishDownlinkResult publishes a downlink result on the organization's
// downlink_result topic, with the ref of the command that queued it.
func (a *MQTTAdapter) PublishDownlinkResult(ctx context.Context, orgUUID, ref string, result *mioty.DLDataResult) error {
	event := downlinkResultEvent(result.EpEui, result.QueId, result.Result, ref)
	if result.Result == mioty.ResultSent {
		addTransmission(event, result)
	}
	return a.publishDownlinkResultEvent(ctx, orgUUID, result.EpEui, event)
}

// PublishDownlinkAcknowledged publishes on the organization's downlink_result
// topic that the endpoint acknowledged the downlink in the uplink after
// packetCnt, with the ref of the command that queued it.
func (a *MQTTAdapter) PublishDownlinkAcknowledged(ctx context.Context, orgUUID, ref string, epEUI, queID uint64, packetCnt uint32) error {
	event := downlinkResultEvent(epEUI, queID, mqtt.DownlinkResultAcknowledged, ref)
	event[mqttKeyPacketCnt] = packetCnt
	return a.publishDownlinkResultEvent(ctx, orgUUID, epEUI, event)
}

// downlinkResultEvent is the body every downlink_result carries; ref is
// omitted for a downlink queued without one.
func downlinkResultEvent(epEUI, queID uint64, result, ref string) map[string]interface{} {
	event := map[string]interface{}{
		mqttKeyEpEui:  mioty.FormatEUI64Lower(epEUI),
		mqttKeyQueId:  queID,
		mqttKeyResult: result,
	}
	if ref != "" {
		event[mqttKeyRef] = ref
	}
	return event
}

func (a *MQTTAdapter) publishDownlinkResultEvent(ctx context.Context, orgUUID string, epEUI uint64, event map[string]interface{}) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("%w: %w", errMarshalDownlinkResultPayload, err)
	}
	return a.pub.PublishDeviceEvent(ctx, orgUUID, mioty.FormatEUI64Lower(epEUI), mqtt.DeviceEventDownlinkResult, payload)
}

// addTransmission adds where and when a sent downlink went out (BSSCI §3.14.1).
func addTransmission(event map[string]interface{}, result *mioty.DLDataResult) {
	if result.BsEui != nil {
		event[mqttKeyBsEui] = mioty.FormatEUI64Lower(*result.BsEui)
	}
	if result.TxTime != nil {
		event[mqttKeyTxTime] = *result.TxTime
	}
	if result.PacketCnt != nil {
		event[mqttKeyPacketCnt] = *result.PacketCnt
	}
}
