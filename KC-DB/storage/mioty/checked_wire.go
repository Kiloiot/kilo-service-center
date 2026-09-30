package mioty

import "github.com/vmihailenco/msgpack/v5"

// Checked decode forms of the canonical messages an Application Center sends.
// Each lists the mandatory rows of its SCACI v1.0.0 table
// (MIOTY_SC-AC-Interface_v1.0.0.md) past the core fields command and opId,
// which the frame envelope requires before a message is routed (§3.2).

// ulDataTransmitFields is ULDataTransmit without its decoder, so the wire form
// decodes it without recursing.
type ulDataTransmitFields ULDataTransmit

// ulDataTransmitWire is the checked decode form of an ulDataTx.
type ulDataTransmitWire struct {
	EpEui                Required[RangedUint[uint64]] `json:"epEui" msgpack:"epEui"`
	NwkSnKey             Required[NetworkKey]         `json:"nwkSnKey" msgpack:"nwkSnKey"`
	ShAddr               Required[RangedUint[uint16]] `json:"shAddr" msgpack:"shAddr"`
	PacketCnt            Required[RangedUint[uint32]] `json:"packetCnt" msgpack:"packetCnt"`
	UserData             Required[UplinkUserData]     `json:"userData" msgpack:"userData"`
	BsEui                *RangedUint[uint64]          `json:"bsEui" msgpack:"bsEui"`
	Format               *RangedUint[uint8]           `json:"format" msgpack:"format"`
	ulDataTransmitFields `msgpack:",inline"`
}

// MandatoryFields is the ulDataTx table of §3.9.1 (lines 471-476) without its
// optional rows bsEui and profile.
func (w *ulDataTransmitWire) MandatoryFields() map[string]Presence {
	return map[string]Presence{
		"epEui":     &w.EpEui,
		"nwkSnKey":  &w.NwkSnKey,
		"shAddr":    &w.ShAddr,
		"packetCnt": &w.PacketCnt,
		"userData":  &w.UserData,
	}
}

// Message gives the checked values their ULDataTransmit types.
func (w *ulDataTransmitWire) Message() (ULDataTransmit, error) {
	message := ULDataTransmit(w.ulDataTransmitFields)
	message.EpEui = w.EpEui.Value.Value
	message.NwkSnKey = w.NwkSnKey.Value
	message.ShAddr = w.ShAddr.Value.Value
	message.PacketCnt = w.PacketCnt.Value.Value
	message.UserData = w.UserData.Value
	message.BsEui = w.BsEui.Pointer()
	message.Format = w.Format.Pointer()
	return message, nil
}

// DecodeMsgpack decodes an ulDataTx through its checked wire form.
func (t *ULDataTransmit) DecodeMsgpack(dec *msgpack.Decoder) error {
	return DecodeChecked[ulDataTransmitWire](dec, t)
}

// UnmarshalJSON decodes an ulDataTx through its checked wire form.
func (t *ULDataTransmit) UnmarshalJSON(data []byte) error {
	return UnmarshalChecked[ulDataTransmitWire](data, t)
}

// dlDataQueueFields is DLDataQueue without its decoder, so the wire form
// decodes it without recursing.
type dlDataQueueFields DLDataQueue

// dlDataQueueWire is the checked decode form of a dlDataQue.
type dlDataQueueWire struct {
	EpEui             Required[RangedUint[uint64]] `json:"epEui" msgpack:"epEui"`
	QueID             Required[RangedUint[uint64]] `json:"queId" msgpack:"queId"`
	CntDepend         Required[bool]               `json:"cntDepend" msgpack:"cntDepend"`
	UserData          Required[DownlinkUserData]   `json:"userData" msgpack:"userData"`
	PacketCnt         []RangedUint[uint32]         `json:"packetCnt" msgpack:"packetCnt"`
	Format            *RangedUint[uint8]           `json:"format" msgpack:"format"`
	Prio              *priority                    `json:"prio" msgpack:"prio"`
	dlDataQueueFields `msgpack:",inline"`
}

// MandatoryFields is the dlDataQue table of §3.10.1 (lines 508-512) without
// packetCnt, which the table omits when cntDepend is false.
func (w *dlDataQueueWire) MandatoryFields() map[string]Presence {
	return map[string]Presence{
		"epEui":     &w.EpEui,
		"queId":     &w.QueID,
		"cntDepend": &w.CntDepend,
		"userData":  &w.UserData,
	}
}

// Message gives the checked values their DLDataQueue types.
func (w *dlDataQueueWire) Message() (DLDataQueue, error) {
	message := DLDataQueue(w.dlDataQueueFields)
	message.EpEui = w.EpEui.Value.Value
	message.QueId = w.QueID.Value.Value
	message.CntDepend = w.CntDepend.Value
	message.UserData = w.UserData.Value
	message.Format = w.Format.Pointer()
	message.Prio = w.Prio.pointer()
	if w.PacketCnt != nil {
		message.PacketCnt = make([]uint32, len(w.PacketCnt))
		for i, counter := range w.PacketCnt {
			message.PacketCnt[i] = counter.Value
		}
	}
	return message, nil
}

// DecodeMsgpack decodes a dlDataQue through its checked wire form.
func (q *DLDataQueue) DecodeMsgpack(dec *msgpack.Decoder) error {
	return DecodeChecked[dlDataQueueWire](dec, q)
}

// UnmarshalJSON decodes a dlDataQue through its checked wire form.
func (q *DLDataQueue) UnmarshalJSON(data []byte) error {
	return UnmarshalChecked[dlDataQueueWire](data, q)
}
