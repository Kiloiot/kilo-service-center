package bssciservices

import (
	"encoding/hex"
	"encoding/json"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// mqttUplinkEvent is the body of an MQTT event/up message: the uplink as the
// service center holds it once its reception window closed, with every
// receiving base station, and the blueprint decode outcome.
type mqttUplinkEvent struct {
	EpEui           string             `json:"epEui"`
	BsEui           string             `json:"bsEui"`
	RSSI            float64            `json:"rssi"`
	SNR             float64            `json:"snr"`
	EqSnr           *float64           `json:"eqSnr,omitempty"`
	RxTime          int64              `json:"rxTime"`
	RxDuration      *int64             `json:"rxDuration,omitempty"`
	Cnt             uint32             `json:"cnt"`
	Data            []byte             `json:"data"`
	Format          *uint8             `json:"format,omitempty"`
	Profile         *string            `json:"profile,omitempty"`
	Mode            *string            `json:"mode,omitempty"`
	Subpackets      *mioty.Subpackets  `json:"subpackets,omitempty"`
	DlOpen          bool               `json:"dlOpen"`
	ResponseExp     bool               `json:"responseExp"`
	DlAck           bool               `json:"dlAck"`
	Duplicate       *bool              `json:"duplicate,omitempty"`
	PacketCntReused bool               `json:"packetCntReused,omitempty"`
	BaseStations    []mqttUplinkRxInfo `json:"baseStations,omitempty"`
	DecodedPayload  json.RawMessage    `json:"decodedPayload,omitempty"`
	DecodeStatus    string             `json:"decodeStatus,omitempty"`
	DecodeErrorCode string             `json:"decodeErrorCode,omitempty"`
	BlueprintType   string             `json:"blueprintTypeEui,omitempty"`
}

// mqttUplinkRxInfo is one base station's reception of the uplink.
type mqttUplinkRxInfo struct {
	BsEui      string            `json:"bsEui"`
	RxTime     int64             `json:"rxTime"`
	RxDuration *int64            `json:"rxDuration,omitempty"`
	SNR        float64           `json:"snr"`
	RSSI       float64           `json:"rssi"`
	EqSnr      *float64          `json:"eqSnr,omitempty"`
	DlRxSnr    *float64          `json:"dlRxSnr,omitempty"`
	DlRxRssi   *float64          `json:"dlRxRssi,omitempty"`
	Profile    *string           `json:"profile,omitempty"`
	Mode       *string           `json:"mode,omitempty"`
	Subpackets *mioty.Subpackets `json:"subpackets,omitempty"`
}

func newMQTTUplinkEvent(msg *mioty.ULDataMessage) mqttUplinkEvent {
	event := mqttUplinkEvent{
		EpEui:           mioty.FormatEUI64Lower(msg.EpEui),
		BsEui:           mioty.FormatEUI64Lower(msg.BsEui),
		RSSI:            msg.RSSI,
		SNR:             msg.SNR,
		EqSnr:           msg.EqSnr,
		RxTime:          msg.RxTime,
		RxDuration:      msg.RxDuration,
		Cnt:             msg.PacketCnt,
		Data:            msg.UserData,
		Format:          msg.Format,
		Profile:         msg.Profile,
		Mode:            msg.Mode,
		Subpackets:      msg.Subpackets,
		DlOpen:          msg.DlOpen,
		ResponseExp:     msg.ResponseExp,
		DlAck:           msg.DlAck,
		Duplicate:       msg.Duplicate,
		PacketCntReused: msg.PacketCntReused,
		DecodeStatus:    msg.DecodeStatus,
		DecodeErrorCode: msg.DecodeErrorCode,
		BlueprintType:   hex.EncodeToString(msg.BlueprintTypeEUI),
	}
	if len(msg.DecodedPayload) > 0 {
		event.DecodedPayload = msg.DecodedPayload
	}
	for _, rx := range msg.BaseStations {
		event.BaseStations = append(event.BaseStations, mqttUplinkRxInfo{
			BsEui:      mioty.FormatEUI64Lower(rx.BsEui),
			RxTime:     rx.RxTime,
			RxDuration: rx.RxDuration,
			SNR:        rx.Snr,
			RSSI:       rx.Rssi,
			EqSnr:      rx.EqSnr,
			DlRxSnr:    rx.DlRxSnr,
			DlRxRssi:   rx.DlRxRssi,
			Profile:    rx.Profile,
			Mode:       rx.Mode,
			Subpackets: rx.Subpackets,
		})
	}
	return event
}
