package bssciservices

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-MQTT/pkg/mqtt"
)

// mqttAdapter bridges bssci.MQTTEventPublisher → mqtt.DeviceEventPublisher.PublishDeviceEvent.
// Converts uint64 EUIs to hex strings, builds JSON payloads, and delegates to KC-MQTT.
type mqttAdapter struct {
	pub mqtt.DeviceEventPublisher
}

// NewMQTTAdapter creates an adapter that satisfies bssci.MQTTEventPublisher using KC-MQTT.
func NewMQTTAdapter(pub mqtt.DeviceEventPublisher) bssci.MQTTEventPublisher {
	return &mqttAdapter{pub: pub}
}

func (a *mqttAdapter) PublishUplink(ctx context.Context, orgUUID string, epEUI uint64, bsEUI uint64,
	rssi float64, snr float64, rxTime int64, packetCnt uint32, userData []byte) error {
	epEUIHex := fmt.Sprintf("%016x", epEUI)
	payload, err := json.Marshal(map[string]interface{}{
		"bsEui":  fmt.Sprintf("%016x", bsEUI),
		"rssi":   rssi,
		"snr":    snr,
		"rxTime": rxTime,
		"cnt":    packetCnt,
		"data":   userData,
	})
	if err != nil {
		return fmt.Errorf("mqtt adapter: failed to marshal uplink payload: %w", err)
	}
	return a.pub.PublishDeviceEvent(ctx, orgUUID, epEUIHex, mqtt.DeviceEventUp, payload)
}

func (a *mqttAdapter) PublishAttach(ctx context.Context, orgUUID string, epEUI uint64, bsEUI uint64) error {
	epEUIHex := fmt.Sprintf("%016x", epEUI)
	payload, err := json.Marshal(map[string]interface{}{
		"epEui": epEUIHex,
		"bsEui": fmt.Sprintf("%016x", bsEUI),
		"event": "attach",
	})
	if err != nil {
		return fmt.Errorf("mqtt adapter: failed to marshal attach payload: %w", err)
	}
	return a.pub.PublishDeviceEvent(ctx, orgUUID, epEUIHex, mqtt.DeviceEventAttach, payload)
}

func (a *mqttAdapter) PublishDetach(ctx context.Context, orgUUID string, epEUI uint64, bsEUI uint64) error {
	epEUIHex := fmt.Sprintf("%016x", epEUI)
	payload, err := json.Marshal(map[string]interface{}{
		"epEui": epEUIHex,
		"bsEui": fmt.Sprintf("%016x", bsEUI),
		"event": "detach",
	})
	if err != nil {
		return fmt.Errorf("mqtt adapter: failed to marshal detach payload: %w", err)
	}
	return a.pub.PublishDeviceEvent(ctx, orgUUID, epEUIHex, mqtt.DeviceEventDetach, payload)
}

func (a *mqttAdapter) PublishDownlinkResult(ctx context.Context, orgUUID string, epEUI uint64, queID uint64, result string) error {
	epEUIHex := fmt.Sprintf("%016x", epEUI)
	payload, err := json.Marshal(map[string]interface{}{
		"epEui":  epEUIHex,
		"queId":  queID,
		"result": result,
	})
	if err != nil {
		return fmt.Errorf("mqtt adapter: failed to marshal downlink result payload: %w", err)
	}
	return a.pub.PublishDeviceEvent(ctx, orgUUID, epEUIHex, mqtt.DeviceEventDownlinkResult, payload)
}
