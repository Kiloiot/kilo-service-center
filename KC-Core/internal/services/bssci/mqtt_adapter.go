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
	mqttKeyEpEui       = "epEui"
	mqttKeyBsEui       = "bsEui"
	mqttKeyQueId       = "queId"
	mqttKeyDlOpen      = "dlOpen"
	mqttKeyResponseExp = "responseExp"
	mqttKeyDlAck       = "dlAck"
	mqttKeyTxTime      = "txTime"
	mqttKeyPacketCnt   = "packetCnt"
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
	epEUIHex := mioty.FormatEUI64Lower(msg.EpEui)
	event := map[string]interface{}{
		mqttKeyBsEui:       mioty.FormatEUI64Lower(msg.BsEui),
		"rssi":             msg.RSSI,
		"snr":              msg.SNR,
		"rxTime":           msg.RxTime,
		"cnt":              msg.PacketCnt,
		"data":             msg.UserData,
		mqttKeyDlOpen:      msg.DlOpen,
		mqttKeyResponseExp: msg.ResponseExp,
		mqttKeyDlAck:       msg.DlAck,
	}
	if len(msg.DecodedPayload) > 0 {
		event["decodedPayload"] = json.RawMessage(msg.DecodedPayload)
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("%w: %w", errMarshalUplinkPayload, err)
	}
	return a.pub.PublishDeviceEvent(ctx, orgUUID, epEUIHex, mqtt.DeviceEventUp, payload)
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

// PublishDownlinkResult publishes a downlink result on the organization's downlink_result topic.
func (a *MQTTAdapter) PublishDownlinkResult(ctx context.Context, orgUUID string, result *mioty.DLDataResult) error {
	epEUIHex := mioty.FormatEUI64Lower(result.EpEui)
	event := map[string]interface{}{
		mqttKeyEpEui: epEUIHex,
		mqttKeyQueId: result.QueId,
		"result":     result.Result,
	}
	if result.Result == mioty.ResultSent {
		addTransmission(event, result)
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("%w: %w", errMarshalDownlinkResultPayload, err)
	}
	return a.pub.PublishDeviceEvent(ctx, orgUUID, epEUIHex, mqtt.DeviceEventDownlinkResult, payload)
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
