// Package mioty defines the exact MIOTY BSSCI v1.0.0 data types and field structures
//
// CRITICAL: This is the SINGLE SOURCE OF TRUTH for all MIOTY protocol field definitions
//
// ALL code handling MIOTY messages MUST import and use these types:
//
//	import "github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
//
// DO NOT create duplicate type definitions elsewhere in the codebase!
//
// Field Naming Convention:
//   - Go struct fields: Capitalized for export (e.g., EpEui, PacketCnt)
//   - JSON/MessagePack tags: MIOTY spec compliant (e.g., "epEui", "packetCnt")
//   - Database tags: snake_case (e.g., "ep_eui", "packet_cnt")
//
// Based on MIOTY Base Station to Service Center Interface Specification v1.0.0
package mioty

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

// ============================================================================
// MIOTY Binary Framing (Section 4 of BSSCI spec)
// ============================================================================

// MIOTYFrameIdentifier is the 8-byte ASCII identifier for MIOTY frames
var MIOTYFrameIdentifier = [8]byte{'M', 'I', 'O', 'T', 'Y', 'B', '0', '1'} // "MIOTYB01"

// MIOTY BSSCI Protocol Version Constants
const (
	// MIOTYVersionMajor is the major version component
	MIOTYVersionMajor = 1
	// MIOTYVersionMinor is the minor version component
	MIOTYVersionMinor = 0
	// MIOTYVersionPatch is the patch version component
	MIOTYVersionPatch = 0
	// MIOTYProtocolVersion is the full BSSCI protocol version string
	MIOTYProtocolVersion = "1.0.0"
)

// Blueprint Decode Status Constants
// These mirror KC-Core/pkg/blueprint/constants.go for use in KC-DB without circular imports
const (
	// DecodeStatusFailed indicates decoding was attempted but failed
	DecodeStatusFailed = "failed"
	// DecodeStatusSkipped indicates decoding was skipped (no blueprint configured)
	DecodeStatusSkipped = "skipped"
)

// Blueprint Decode Error Constants
// These mirror KC-Core/pkg/blueprint/errors_catalog.go for use in KC-DB without circular imports
const (
	// ErrInvalidJSONPayload indicates the decoded payload was not valid JSON
	ErrInvalidJSONPayload = "blueprint.error.invalid_json_payload"
)

// Frame represents the binary frame structure that wraps all BSSCI messages
type Frame struct {
	Identifier  [8]byte // ASCII "MIOTYB01" (0x4d 49 4f 54 59 42 30 31)
	PayloadSize uint32  // Size of following JSON/MessagePack object (little-endian)
	Payload     []byte  // The actual JSON or MessagePack encoded message
}

// Serialize converts a Frame to bytes
func (f *Frame) Serialize() []byte {
	result := make([]byte, 12+len(f.Payload))
	copy(result[0:8], f.Identifier[:])
	// Note: Payload length is checked at frame construction time, not here
	// In practice, BSSCI payloads never exceed uint32 max
	binary.LittleEndian.PutUint32(result[8:12], uint32(len(f.Payload))) //nolint:gosec // payload size validated at construction
	copy(result[12:], f.Payload)
	return result
}

// ============================================================================
// Core MIOTY Types (used throughout the protocol)
// ============================================================================

// BaseMessage contains fields present in ALL BSSCI messages
type BaseMessage struct {
	CommandType string `json:"command" msgpack:"command"` // Operation mnemonic (MIOTY spec field: "command")
	OpId        int64  `json:"opId" msgpack:"opId"`       // Operation ID (signed: positive for BS, negative for SC)
}

// Type definitions per MIOTY BSSCI v1.0.0 specification
type (
	// EUI64 represents an 8-byte Extended Unique Identifier
	EUI64 uint64

	// ShortAddress is a 16-bit short address per MIOTY spec
	ShortAddress uint16

	// PacketCounter is a 32-bit packet counter
	PacketCounter uint32

	// FormatID is an 8-bit format identifier
	FormatID uint8

	// UnixNano represents Unix UTC time in nanoseconds (64-bit)
	UnixNano int64

	// SessionUUID is a 16-byte session identifier
	SessionUUID [16]byte

	// NetworkKey is a 16-byte network session key
	NetworkKey [16]byte

	// Nonce is a 4-byte nonce value
	Nonce [4]byte

	// Signature is a 4-byte signature value
	Signature [4]byte
)

// GeoLocation represents geographic coordinates [Latitude, Longitude, Altitude]
type GeoLocation [3]float64

// Subpackets represents the optional subpacket information
type Subpackets struct {
	SNR       []float64 `json:"snr" msgpack:"snr"`                         // Subpacket SNRs in dB
	RSSI      []float64 `json:"rssi" msgpack:"rssi"`                       // Subpacket RSSIs in dBm
	Frequency []int64   `json:"frequency" msgpack:"frequency"`             // Subpacket frequencies in Hz
	Phase     []float64 `json:"phase,omitempty" msgpack:"phase,omitempty"` // Subpacket phases (degrees ±180)
}

// Numeric4 represents a 4-element numeric array per MIOTY spec tables
// Serializes to [1,2,3,4] in both JSON and MessagePack (not base64)
// Used for nonce and signature fields in SCACI EP Status messages
type Numeric4 [4]uint8

// MarshalJSON implements custom JSON marshaling to numeric array
func (n Numeric4) MarshalJSON() ([]byte, error) {
	// Serialize as [1,2,3,4], not base64
	return json.Marshal([4]uint8(n))
}

// UnmarshalJSON implements custom JSON unmarshaling with length validation
func (n *Numeric4) UnmarshalJSON(data []byte) error {
	// First unmarshal to a slice to check length
	var slice []uint8
	if err := json.Unmarshal(data, &slice); err != nil {
		return fmt.Errorf("nonce/sign must be 4-element numeric array: %w", err)
	}
	if len(slice) != 4 {
		return fmt.Errorf("nonce/sign must be 4-element numeric array, got %d elements", len(slice))
	}
	// Copy to fixed array
	copy(n[:], slice)
	return nil
}

// EncodeMsgpack implements vmihailenco/msgpack/v5 custom encoding
func (n Numeric4) EncodeMsgpack(enc *msgpack.Encoder) error {
	// Encode as 4-element array
	if err := enc.EncodeArrayLen(4); err != nil {
		return err
	}
	for _, v := range n {
		if err := enc.EncodeUint8(v); err != nil {
			return err
		}
	}
	return nil
}

// DecodeMsgpack implements vmihailenco/msgpack/v5 custom decoding
func (n *Numeric4) DecodeMsgpack(dec *msgpack.Decoder) error {
	arrLen, err := dec.DecodeArrayLen()
	if err != nil {
		return err
	}
	if arrLen != 4 {
		return fmt.Errorf("nonce/sign must be 4 elements, got %d", arrLen)
	}

	for i := 0; i < 4; i++ {
		v, err := dec.DecodeUint8()
		if err != nil {
			return err
		}
		n[i] = v
	}
	return nil
}

// ============================================================================
// 5.3 Connect Operation
// ============================================================================

// MarshalJSON marshals the SessionUUID as an array of integers [1,2,3,...] instead of base64
// BSSCI §4-4.5 requires UUIDs to be marshaled as arrays of integers, not base64 strings
func (u SessionUUID) MarshalJSON() ([]byte, error) {
	arr := make([]int, 16)
	for i, b := range u {
		arr[i] = int(b)
	}
	return json.Marshal(arr)
}

// EncodeMsgpack marshals the UUID as an array of integers for MessagePack
// Uses value receiver to work with non-addressable values (struct fields passed by value)
func (u SessionUUID) EncodeMsgpack(enc *msgpack.Encoder) error {
	arr := make([]int, 16)
	for i, b := range u {
		arr[i] = int(b)
	}
	return enc.Encode(arr)
}

// Connect represents the "con" message from Base Station to Service Center
type Connect struct {
	BaseMessage
	Version     string       `json:"version" msgpack:"version"`                             // Protocol version "major.minor.patch"
	BsEui       uint64       `json:"bsEui" msgpack:"bsEui"`                                 // Base Station EUI64
	Vendor      *string      `json:"vendor,omitempty" msgpack:"vendor,omitempty"`           // Optional vendor name
	Model       *string      `json:"model,omitempty" msgpack:"model,omitempty"`             // Optional model
	Name        *string      `json:"name,omitempty" msgpack:"name,omitempty"`               // Optional name
	SwVersion   *string      `json:"swVersion,omitempty" msgpack:"swVersion,omitempty"`     // Optional software version
	Info        interface{}  `json:"info,omitempty" msgpack:"info,omitempty"`               // Optional arbitrary key-value pairs
	Bidi        bool         `json:"bidi" msgpack:"bidi"`                                   // BS is bidirectional
	GeoLocation *GeoLocation `json:"geoLocation,omitempty" msgpack:"geoLocation,omitempty"` // Optional [Lat, Lon, Alt]
	SnBsUuid    [16]byte     `json:"snBsUuid" msgpack:"snBsUuid"`                           //nolint:revive // wire protocol field name per BSSCI spec
	SnBsOpId    *int64       `json:"snBsOpId,omitempty" msgpack:"snBsOpId,omitempty"`       // Minimum required known BS opId to resume
	SnScOpId    *int64       `json:"snScOpId,omitempty" msgpack:"snScOpId,omitempty"`       // Maximum known SC opId to resume
}

// ConnectResponse represents the "conRsp" message from Service Center
type ConnectResponse struct {
	BaseMessage
	Version   string      `json:"version" msgpack:"version"`                         // Negotiated protocol version (mandatory per BSSCI §4-4.5)
	ScEui     uint64      `json:"scEui" msgpack:"scEui"`                             // Service Center EUI64
	Vendor    *string     `json:"vendor,omitempty" msgpack:"vendor,omitempty"`       // Optional vendor name
	Model     *string     `json:"model,omitempty" msgpack:"model,omitempty"`         // Optional model
	Name      *string     `json:"name,omitempty" msgpack:"name,omitempty"`           // Optional name
	SwVersion *string     `json:"swVersion,omitempty" msgpack:"swVersion,omitempty"` // Optional software version
	Info      interface{} `json:"info,omitempty" msgpack:"info,omitempty"`           // Optional arbitrary key-value pairs
	SnResume  bool        `json:"snResume" msgpack:"snResume"`                       // True if previous session resumed
	SnScUuid  SessionUUID `json:"snScUuid" msgpack:"snScUuid"`                       //nolint:revive // wire protocol field name per BSSCI spec
	SnBsOpId  *int64      `json:"snBsOpId,omitempty" msgpack:"snBsOpId,omitempty"`   // Last known BS opId when resuming
	SnScOpId  *int64      `json:"snScOpId,omitempty" msgpack:"snScOpId,omitempty"`   // Last known SC opId when resuming
}

// ConnectComplete represents the "conCmp" message
type ConnectComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// 5.4 Ping Operation
// ============================================================================

// Ping represents the "ping" message
type Ping struct {
	BaseMessage
	// No additional fields beyond core
}

// PingResponse represents the "pingRsp" message
type PingResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// PingComplete represents the "pingCmp" message
type PingComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// 5.5 Status Operation
// ============================================================================

// StatusRequest represents the "status" message
type StatusRequest struct {
	BaseMessage
	// No additional fields beyond core
}

// StatusResponse represents the "statusRsp" message
type StatusResponse struct {
	BaseMessage
	Code        int          `json:"code" msgpack:"code"`           // POSIX error number (0 = "ok")
	Message     string       `json:"message" msgpack:"message"`     // Status message
	Time        int64        `json:"time" msgpack:"time"`           // Unix UTC time (64-bit, ns resolution)
	DutyCycle   float64      `json:"dutyCycle" msgpack:"dutyCycle"` // Fraction of TX time (sliding window over 1 hour)
	GeoLocation *GeoLocation `json:"geoLocation,omitempty" msgpack:"geoLocation,omitempty"`
	Uptime      *int64       `json:"uptime,omitempty" msgpack:"uptime,omitempty"`   // Seconds
	Temp        *float64     `json:"temp,omitempty" msgpack:"temp,omitempty"`       // Temperature in °C
	CpuLoad     *float64     `json:"cpuLoad,omitempty" msgpack:"cpuLoad,omitempty"` //nolint:revive // wire protocol field name per BSSCI spec
	MemLoad     *float64     `json:"memLoad,omitempty" msgpack:"memLoad,omitempty"` // Memory load normalized to 1.0
	Config      interface{}  `json:"config,omitempty" msgpack:"config,omitempty"`   // Configuration object
}

// StatusComplete represents the "statusCmp" message
type StatusComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// 5.6 Attach Operation (OTA attach observed by BS)
// ============================================================================

// Attach represents the "att" message from Base Station
type Attach struct {
	BaseMessage
	EpEui       uint64      `json:"epEui" msgpack:"epEui"`                               // End Point EUI64
	RxTime      int64       `json:"rxTime" msgpack:"rxTime"`                             // Unix UTC center of last subpacket (ns)
	RxDuration  *int64      `json:"rxDuration,omitempty" msgpack:"rxDuration,omitempty"` // First to last subpacket center (ns)
	AttachCnt   uint32      `json:"attachCnt" msgpack:"attachCnt"`                       // End Point attachment counter
	Snr         float64     `json:"snr" msgpack:"snr"`                                   // Signal-to-noise ratio in dB
	Rssi        float64     `json:"rssi" msgpack:"rssi"`                                 // Signal strength in dBm
	EqSnr       *float64    `json:"eqSnr,omitempty" msgpack:"eqSnr,omitempty"`           // AWGN equivalent SNR in dB
	Profile     *string     `json:"profile,omitempty" msgpack:"profile,omitempty"`       // Mioty profile (e.g., "eu1")
	Subpackets  *Subpackets `json:"subpackets,omitempty" msgpack:"subpackets,omitempty"` // Subpacket details
	Nonce       [4]byte     `json:"nonce" msgpack:"nonce"`                               // 4-byte EP nonce
	Sign        [4]byte     `json:"sign" msgpack:"sign"`                                 // 4-byte EP signature
	ShAddr      *uint16     `json:"shAddr,omitempty" msgpack:"shAddr,omitempty"`         // EP short address if assigned by BS
	DualChan    bool        `json:"dualChan" msgpack:"dualChan"`                         // Dual channel mode
	Repetition  bool        `json:"repetition" msgpack:"repetition"`                     // Repetition enabled
	WideCarrOff bool        `json:"wideCarrOff" msgpack:"wideCarrOff"`                   // Wide carrier offset
	LongBlkDist bool        `json:"longBlkDist" msgpack:"longBlkDist"`                   // Long block distance
}

// AttachResponse represents the "attRsp" message
type AttachResponse struct {
	BaseMessage
	NwkSnKey [16]byte `json:"nwkSnKey" msgpack:"nwkSnKey"`                 // 16-byte EP network session key
	ShAddr   *uint16  `json:"shAddr,omitempty" msgpack:"shAddr,omitempty"` // EP short address if not assigned by BS
}

// AttachComplete represents the "attCmp" message
type AttachComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// 5.7 Detach Operation (OTA detach observed by BS)
// ============================================================================

// Detach represents the "det" message from Base Station
type Detach struct {
	BaseMessage
	EpEui      uint64      `json:"epEui" msgpack:"epEui"`                               // End Point EUI64
	RxTime     int64       `json:"rxTime" msgpack:"rxTime"`                             // Unix UTC center of last subpacket (ns)
	RxDuration *int64      `json:"rxDuration,omitempty" msgpack:"rxDuration,omitempty"` // First to last subpacket center (ns)
	PacketCnt  uint32      `json:"packetCnt" msgpack:"packetCnt"`                       // EP packet counter
	Snr        float64     `json:"snr" msgpack:"snr"`                                   // Signal-to-noise ratio in dB
	Rssi       float64     `json:"rssi" msgpack:"rssi"`                                 // Signal strength in dBm
	EqSnr      *float64    `json:"eqSnr,omitempty" msgpack:"eqSnr,omitempty"`           // AWGN equivalent SNR in dB
	Profile    *string     `json:"profile,omitempty" msgpack:"profile,omitempty"`       // Mioty profile
	Subpackets *Subpackets `json:"subpackets,omitempty" msgpack:"subpackets,omitempty"` // Subpacket details
	Sign       [4]byte     `json:"sign" msgpack:"sign"`                                 // 4-byte EP signature
}

// DetachResponse represents the "detRsp" message
type DetachResponse struct {
	BaseMessage
	Sign [4]byte `json:"sign" msgpack:"sign"` // 4-byte signature for the EP
}

// DetachComplete represents the "detCmp" message
type DetachComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// 5.8 Attach Propagate (SC → BS)
// ============================================================================

// AttachPropagate represents the "attPrp" message from Service Center
type AttachPropagate struct {
	BaseMessage
	EpEui         uint64   `json:"epEui" msgpack:"epEui"`                 // End Point EUI64
	Bidi          bool     `json:"bidi" msgpack:"bidi"`                   // EP is bidirectional
	NwkSnKey      [16]byte `json:"nwkSnKey" msgpack:"nwkSnKey"`           // 16-byte EP network session key
	ShAddr        uint16   `json:"shAddr" msgpack:"shAddr"`               // EP short address
	LastPacketCnt uint32   `json:"lastPacketCnt" msgpack:"lastPacketCnt"` // Last known EP packet counter
	DualChan      bool     `json:"dualChan" msgpack:"dualChan"`           // Dual channel mode
	Repetition    bool     `json:"repetition" msgpack:"repetition"`       // Repetition enabled
	WideCarrOff   bool     `json:"wideCarrOff" msgpack:"wideCarrOff"`     // Wide carrier offset
	LongBlkDist   bool     `json:"longBlkDist" msgpack:"longBlkDist"`     // Long block distance
}

// AttachPropagateResponse represents the "attPrpRsp" message
type AttachPropagateResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// AttachPropagateComplete represents the "attPrpCmp" message
type AttachPropagateComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// 3.9 Detach Propagate (SC → BS) - BSSCI §3.9.1
// ============================================================================

// DetachPropagate represents the "detPrp" message from Service Center
// Per BSSCI §3.9.1: only command, opId, epEui are required (shAddr is NOT in spec)
type DetachPropagate struct {
	BaseMessage
	EpEui uint64 `json:"epEui" msgpack:"epEui"` // End Point EUI64
}

// DetachPropagateResponse represents the "detPrpRsp" message
type DetachPropagateResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// DetachPropagateComplete represents the "detPrpCmp" message
type DetachPropagateComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// 5.10 UL Data (BS → SC after receiving uplink)
// ============================================================================

// ULData represents the "ulData" message from Base Station (Section 5.10.1)
type ULData struct {
	BaseMessage
	EpEui       uint64      `json:"epEui" msgpack:"epEui"`                               // End Point EUI64
	RxTime      int64       `json:"rxTime" msgpack:"rxTime"`                             // Unix UTC center of last subpacket (ns)
	RxDuration  *int64      `json:"rxDuration,omitempty" msgpack:"rxDuration,omitempty"` // First to last subpacket center (ns)
	PacketCnt   uint32      `json:"packetCnt" msgpack:"packetCnt"`                       // EP packet counter
	Snr         float64     `json:"snr" msgpack:"snr"`                                   // Signal-to-noise ratio in dB
	Rssi        float64     `json:"rssi" msgpack:"rssi"`                                 // Signal strength in dBm
	EqSnr       *float64    `json:"eqSnr,omitempty" msgpack:"eqSnr,omitempty"`           // AWGN equivalent SNR in dB
	Profile     *string     `json:"profile,omitempty" msgpack:"profile,omitempty"`       // Mioty profile (e.g., "eu1")
	Mode        *string     `json:"mode,omitempty" msgpack:"mode,omitempty"`             // Mioty mode (e.g., "ulp", "ulp-rep", "ulp-ll")
	Subpackets  *Subpackets `json:"subpackets,omitempty" msgpack:"subpackets,omitempty"` // Subpacket details
	UserData    []byte      `json:"userData" msgpack:"userData"`                         // n-byte EP user data (may be empty)
	Format      *uint8      `json:"format,omitempty" msgpack:"format,omitempty"`         // User data format ID (8-bit, default 0)
	DlOpen      bool        `json:"dlOpen" msgpack:"dlOpen"`                             // EP downlink window is opened
	ResponseExp bool        `json:"responseExp" msgpack:"responseExp"`                   // EP expects response (requires dlOpen)
	DlAck       bool        `json:"dlAck" msgpack:"dlAck"`                               // EP acknowledges DL reception (packetCnt - 1)
}

// ULDataResponse represents the "ulDataRsp" message
type ULDataResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// ULDataComplete represents the "ulDataCmp" message
type ULDataComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// 5.11 UL Data Transmit (SC → BS, optional feature)
// ============================================================================

// ULDataTransmit represents the "ulDataTx" message
type ULDataTransmit struct {
	BaseMessage
	BsEui     *uint64  `json:"bsEui,omitempty" msgpack:"bsEui,omitempty"`     // Target base station (optional, SC chooses if omitted)
	EpEui     uint64   `json:"epEui" msgpack:"epEui"`                         // End Point EUI64
	NwkSnKey  [16]byte `json:"nwkSnKey" msgpack:"nwkSnKey"`                   // 16-byte EP network session key
	ShAddr    uint16   `json:"shAddr" msgpack:"shAddr"`                       // EP short address
	PacketCnt uint32   `json:"packetCnt" msgpack:"packetCnt"`                 // EP packet counter
	Profile   *string  `json:"profile,omitempty" msgpack:"profile,omitempty"` // Mioty profile for transmission
	UserData  []byte   `json:"userData" msgpack:"userData"`                   // n-byte EP user data
	Format    *uint8   `json:"format,omitempty" msgpack:"format,omitempty"`   // User data format ID (8-bit, default 0)
}

// ULDataTransmitResponse represents the "ulDataTxRsp" message
type ULDataTransmitResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// ULDataTransmitComplete represents the "ulDataTxCmp" message
type ULDataTransmitComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// 5.12 DL Data Queue (SC → BS)
// ============================================================================

// DLDataQueue represents the "dlDataQue" message
type DLDataQueue struct {
	BaseMessage
	EpEui        uint64   `json:"epEui" msgpack:"epEui"`                                   // End Point EUI64
	QueId        uint64   `json:"queId" msgpack:"queId"`                                   // Assigned queue ID (64-bit)
	CntDepend    bool     `json:"cntDepend" msgpack:"cntDepend"`                           // userData is counter-dependent
	PacketCnt    []uint32 `json:"packetCnt,omitempty" msgpack:"packetCnt,omitempty"`       // Counters for userData (if cntDepend)
	UserData     [][]byte `json:"userData" msgpack:"userData"`                             // User data for each counter
	Format       *uint8   `json:"format,omitempty" msgpack:"format,omitempty"`             // User data format ID (8-bit)
	Prio         *float32 `json:"prio,omitempty" msgpack:"prio,omitempty"`                 // Priority (default 0)
	ResponseExp  *bool    `json:"responseExp,omitempty" msgpack:"responseExp,omitempty"`   // Request EP response
	ResponsePrio *bool    `json:"responsePrio,omitempty" msgpack:"responsePrio,omitempty"` // Request priority EP response
	DlWindReq    *bool    `json:"dlWindReq,omitempty" msgpack:"dlWindReq,omitempty"`       // Request further EP DL window
	ExpOnly      *bool    `json:"expOnly,omitempty" msgpack:"expOnly,omitempty"`           // Send only if EP expects response
	DlRxStatQry  *bool    `json:"dlRxStatQry,omitempty" msgpack:"dlRxStatQry,omitempty"`   // SCACI §3.10.1: Query DL RX status from endpoint
}

// DLDataQueueResponse represents the "dlDataQueRsp" message
type DLDataQueueResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// DLDataQueueComplete represents the "dlDataQueCmp" message
type DLDataQueueComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// 5.13 DL Data Revoke (SC → BS) - BSSCI Protocol
// ============================================================================

// DLDataRevoke represents the "dlDataRev" message per BSSCI §3.13.1
// Note: BSSCI uses queId, while SCACI uses packetCnt - different protocols!
type DLDataRevoke struct {
	BaseMessage
	EpEui uint64 `json:"epEui" msgpack:"epEui"` // End Point EUI64
	QueId uint64 `json:"queId" msgpack:"queId"` // Queue ID of the scheduled data (BSSCI)
}

// DLDataRevokeResponse represents the "dlDataRevRsp" message
type DLDataRevokeResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// DLDataRevokeComplete represents the "dlDataRevCmp" message
type DLDataRevokeComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// 5.14 DL Data Result (BS → SC after queued DL is sent/discarded)
// ============================================================================

// DLDataResult represents the "dlDataRes" message (SCACI §3.12.1)
type DLDataResult struct {
	BaseMessage
	EpEui     uint64  `json:"epEui" msgpack:"epEui"`                             // End Point EUI64
	QueId     uint64  `json:"queId" msgpack:"queId"`                             // Queue ID
	Result    string  `json:"result" msgpack:"result"`                           // "sent", "expired", or "invalid"
	TxTime    *int64  `json:"txTime,omitempty" msgpack:"txTime,omitempty"`       // Unix UTC center of first subpacket (ns) - only if sent
	PacketCnt *uint32 `json:"packetCnt,omitempty" msgpack:"packetCnt,omitempty"` // EP packet counter - only if sent
	BsEui     *uint64 `json:"bsEui,omitempty" msgpack:"bsEui,omitempty"`         // Base Station EUI64 - only present if result=="sent" per SCACI §3.12.1
}

// DLDataResultResponse represents the "txDataResRsp" message (SCACI §3.12.2)
type DLDataResultResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// DLDataResultComplete represents the "txDataResCmp" message (SCACI §3.12.3)
type DLDataResultComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// 5.15 DL RX Status (BS → SC after receiving DL RX status control segment)
// ============================================================================

// DLRxStatus represents the "dlRxStat" message
type DLRxStatus struct {
	BaseMessage
	EpEui     uint64  `json:"epEui" msgpack:"epEui"`         // End Point EUI64
	RxTime    int64   `json:"rxTime" msgpack:"rxTime"`       // Unix UTC center of last subpacket (ns)
	PacketCnt uint32  `json:"packetCnt" msgpack:"packetCnt"` // EP packet counter
	DlRxSnr   float64 `json:"dlRxSnr" msgpack:"dlRxSnr"`     // EP DL reception SNR in dB
	DlRxRssi  float64 `json:"dlRxRssi" msgpack:"dlRxRssi"`   // EP DL reception RSSI in dBm
}

// DLRxStatusResponse represents the "dlRxStatRsp" message
type DLRxStatusResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// DLRxStatusComplete represents the "dlRxStatCmp" message
type DLRxStatusComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// 5.16 DL RX Status Query (SC → BS schedules query control segment)
// ============================================================================

// DLRxStatusQuery represents the "dlRxStatQry" message
type DLRxStatusQuery struct {
	BaseMessage
	EpEui uint64 `json:"epEui" msgpack:"epEui"` // End Point EUI64
}

// DLRxStatusQueryResponse represents the "dlRxStatQryRsp" message
type DLRxStatusQueryResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// DLRxStatusQueryComplete represents the "dlRxStatQryCmp" message
type DLRxStatusQueryComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// VM Operations (Virtual MAC Mode per BSSCI §5.17-5.20)
// ============================================================================

// VMActivate represents the "vmAct" message from Service Center
type VMActivate struct {
	BaseMessage
	EpEui   uint64 `json:"epEui" msgpack:"epEui" db:"ep_eui"`
	MACType uint8  `json:"macType" msgpack:"macType" db:"mac_type"`
}

// VMDeactivate represents the "vmDeact" message from Service Center
type VMDeactivate struct {
	BaseMessage
	EpEui   uint64 `json:"epEui" msgpack:"epEui" db:"ep_eui"`
	MACType uint8  `json:"macType" msgpack:"macType" db:"mac_type"`
}

// VMStatus represents the "vmStat" message from Service Center
// Note: MACType NOT included per vm_handlers.go:930-934 actual implementation
type VMStatus struct {
	BaseMessage
	EpEui uint64 `json:"epEui" msgpack:"epEui" db:"ep_eui"`
}

// VMDLData represents the "vmDlData" message from Service Center
type VMDLData struct {
	BaseMessage
	EpEui    uint64 `json:"epEui" msgpack:"epEui" db:"ep_eui"`
	MACType  uint8  `json:"macType" msgpack:"macType" db:"mac_type"`
	UserData []byte `json:"userData,omitempty" msgpack:"userData,omitempty" db:"user_data"`
	TxTime   *int64 `json:"txTime,omitempty" msgpack:"txTime,omitempty" db:"tx_time"` // Optional: pointer type
}

// ============================================================================
// 5.17 Errors
// ============================================================================

// Error represents the "error" message
type Error struct {
	BaseMessage
	Code    int    `json:"code" msgpack:"code"`       // POSIX error number
	Message string `json:"message" msgpack:"message"` // Error message
}

// ErrorAck represents the "errorAck" message
type ErrorAck struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// Outbound Field Validation Catalog (BSSCI §2.5.3)
// ============================================================================

// OutboundFieldCatalog defines allowed fields for SC→BS messages per BSSCI §2.5.3
// Each entry lists exact field names from struct json/msgpack tags (source: types.go)
// Built from actual sendMessage call sites - verified via:
// rg -n "sendMessage\(session" kilocenter-modules/KC-Core/pkg/bssci -g'*.go'
var OutboundFieldCatalog = map[string][]string{
	// Control operations (responses we send)
	CmdConnectResponse: {"version", "scEui", "vendor", "model", "name", "swVersion", "info", "snResume", "snScUuid", "snBsOpId", "snScOpId"},
	CmdConnectComplete: {},
	CmdPing:            {}, // SC-initiated ping request
	CmdPingResponse:    {},
	CmdPingComplete:    {},

	// Status operations (SC-initiated status request per §5.8)
	CmdStatus:         {}, // SC-initiated status request
	CmdStatusComplete: {},

	// Attach operations (responses to BS-initiated attach)
	CmdAttachResponse: {"nwkSnKey", "shAddr"},
	CmdAttachComplete: {},

	// Attach propagate (SC-initiated command per §5.6)
	CmdAttachPropagate:         {"epEui", "bidi", "nwkSnKey", "shAddr", "lastPacketCnt", "dualChan", "repetition", "wideCarrOff", "longBlkDist"},
	CmdAttachPropagateResponse: {},
	CmdAttachPropagateComplete: {},

	// Detach operations (responses to BS-initiated detach)
	CmdDetachResponse: {"sign"},
	CmdDetachComplete: {},

	// Detach propagate (SC-initiated command) - BSSCI §3.9.1: only epEui required
	CmdDetachPropagate:         {"epEui"},
	CmdDetachPropagateResponse: {},
	CmdDetachPropagateComplete: {},

	// Uplink operations
	CmdULDataResponse:         {},
	CmdULDataComplete:         {},
	CmdULDataTransmit:         {"epEui", "nwkSnKey", "shAddr", "packetCnt", "profile", "userData", "format"},
	CmdULDataTransmitComplete: {},

	// Downlink operations
	CmdDLDataQueue:          {"epEui", "queId", "cntDepend", "packetCnt", "userData", "format", "prio", "responseExp", "responsePrio", "dlWindReq", "expOnly"},
	CmdDLDataQueueResponse:  {},
	CmdDLDataQueueComplete:  {},
	CmdDLDataRevoke:         {"epEui", "queId"},
	CmdDLDataRevokeResponse: {},
	CmdDLDataRevokeComplete: {},
	CmdDLDataResult:         {"queId", "code"},
	CmdDLDataResultResponse: {},
	CmdDLDataResultComplete: {},

	// DL RX Status (BS→SC status report + SC-initiated query)
	CmdDLRxStatusComplete:      {}, // BS sends dlRxStat, SC sends dlRxStatCmp
	CmdDLRxStatusQuery:         {"epEui", "startTime", "endTime", "limit", "offset"},
	CmdDLRxStatusQueryResponse: {},
	CmdDLRxStatusQueryComplete: {},

	// VM operations (SC-initiated commands)
	CmdVMActivate:           {"epEui", "macType"},
	CmdVMActivateResponse:   {},
	CmdVMActivateComplete:   {},
	CmdVMDeactivate:         {"epEui", "macType"},
	CmdVMDeactivateResponse: {},
	CmdVMDeactivateComplete: {},
	CmdVMStatus:             {"epEui"}, // NO macType per vm_handlers.go:930-934
	CmdVMStatusResponse:     {},
	CmdVMStatusComplete:     {},
	CmdVMDLData:             {"epEui", "macType", "userData", "txTime"},
	CmdVMDLDataResponse:     {},
	CmdVMDLDataComplete:     {},

	// Error handling
	CmdError:    {"code", "message"},
	CmdErrorAck: {},
}

// AllowedOutboundFields returns the whitelist for a given command
func AllowedOutboundFields(command string) ([]string, bool) {
	fields, ok := OutboundFieldCatalog[command]
	return fields, ok
}

// OutboundMandatoryFields defines required fields for SC→BS commands (non-pointer struct fields per BSSCI spec)
// Used by validateOutboundMessage to enforce mandatory field presence before transmission
var OutboundMandatoryFields = map[string][]string{
	// §5.3.2 Connect Response - version, scEui, snResume, snScUuid are mandatory
	CmdConnectResponse: {"version", "scEui", "snResume", "snScUuid"},

	// §5.5.2 Status Response - code, message, time, dutyCycle are mandatory
	CmdStatusResponse: {"code", "message", "time", "dutyCycle"},

	// §5.6.2 Attach Response - nwkSnKey, shAddr are mandatory
	CmdAttachResponse: {"nwkSnKey", "shAddr"},

	// §5.7.2 Detach Response - sign is mandatory
	CmdDetachResponse: {"sign"},

	// §5.8.1 Attach Propagate - all base configuration fields mandatory
	CmdAttachPropagate: {"epEui", "bidi", "nwkSnKey", "shAddr", "lastPacketCnt", "dualChan", "repetition", "wideCarrOff", "longBlkDist"},

	// §3.9.1 Detach Propagate - only epEui mandatory per BSSCI spec
	CmdDetachPropagate: {"epEui"},

	// §5.11.1 UL Data Transmit - epEui, nwkSnKey, shAddr, packetCnt, userData mandatory
	CmdULDataTransmit: {"epEui", "nwkSnKey", "shAddr", "packetCnt", "userData"},

	// §5.12.1 DL Data Queue - epEui, queId, cntDepend, userData mandatory
	CmdDLDataQueue: {"epEui", "queId", "cntDepend", "userData"},

	// §5.13.1 DL Data Revoke - epEui, queId mandatory
	CmdDLDataRevoke: {"epEui", "queId"},

	// §5.14.1 DL Data Result - epEui, queId, result mandatory
	CmdDLDataResult: {"epEui", "queId", "result"},

	// §5.16.1 DL RX Status Query - epEui mandatory
	CmdDLRxStatusQuery: {"epEui"},

	// VM operations - epEui, macType mandatory for activate/deactivate/dlData
	CmdVMActivate:   {"epEui", "macType"},
	CmdVMDeactivate: {"epEui", "macType"},
	CmdVMStatus:     {"epEui"},
	CmdVMDLData:     {"epEui", "macType", "userData"},

	// §5.17.1 Error - code, message mandatory
	CmdError: {"code", "message"},

	// Commands with no mandatory fields beyond command/opId (already enforced in validateOutboundMessage)
	CmdConnectComplete:         {},
	CmdPingResponse:            {},
	CmdPingComplete:            {},
	CmdStatusComplete:          {},
	CmdAttachComplete:          {},
	CmdAttachPropagateResponse: {},
	CmdAttachPropagateComplete: {},
	CmdDetachComplete:          {},
	CmdDetachPropagateResponse: {},
	CmdDetachPropagateComplete: {},
	CmdULDataResponse:          {},
	CmdULDataComplete:          {},
	CmdULDataTransmitResponse:  {},
	CmdULDataTransmitComplete:  {},
	CmdDLDataQueueResponse:     {},
	CmdDLDataQueueComplete:     {},
	CmdDLDataRevokeResponse:    {},
	CmdDLDataRevokeComplete:    {},
	CmdDLDataResultResponse:    {},
	CmdDLDataResultComplete:    {},
	CmdDLRxStatusComplete:      {},
	CmdDLRxStatusQueryResponse: {},
	CmdDLRxStatusQueryComplete: {},
	CmdVMActivateResponse:      {},
	CmdVMActivateComplete:      {},
	CmdVMDeactivateResponse:    {},
	CmdVMDeactivateComplete:    {},
	CmdVMStatusResponse:        {},
	CmdVMStatusComplete:        {},
	CmdVMDLDataResponse:        {},
	CmdVMDLDataComplete:        {},
	CmdErrorAck:                {},
}

// MandatoryOutboundFields returns the mandatory fields for a given SC→BS command
func MandatoryOutboundFields(command string) ([]string, bool) {
	fields, ok := OutboundMandatoryFields[command]
	return fields, ok
}

// BSSCI v1.0.0 command type string constants.
// These define the wire protocol command names for MIOTY Base Station Service Center Interface.
const (
	// Connection operations.
	CmdConnect         = "con"    // Direction: BStoSC - initiates connection handshake
	CmdConnectResponse = "conRsp" // Direction: SCtoBS - response for connection
	CmdConnectComplete = "conCmp" // Direction: BStoSC - completion for connection

	// Ping operations.
	CmdPing         = "ping"    // Direction: Bidirectional - initiates ping
	CmdPingResponse = "pingRsp" // Direction: BStoSC - response for ping
	CmdPingComplete = "pingCmp" // Direction: BStoSC - completion for ping

	// Status operations.
	CmdStatus         = "status"    // Direction: Bidirectional - initiates status request
	CmdStatusResponse = "statusRsp" // Direction: BStoSC - response for status
	CmdStatusComplete = "statusCmp" // Direction: BStoSC - completion for status

	// Attach operations.
	CmdAttach         = "att"    // Direction: BStoSC - initiates attach
	CmdAttachResponse = "attRsp" // Direction: SCtoBS - response for attach
	CmdAttachComplete = "attCmp" // Direction: BStoSC - completion for attach

	// Detach operations.
	CmdDetach         = "det"    // Direction: BStoSC - initiates detach
	CmdDetachResponse = "detRsp" // Direction: SCtoBS - response for detach
	CmdDetachComplete = "detCmp" // Direction: BStoSC - completion for detach

	// Attach propagate operations.
	CmdAttachPropagate         = "attPrp"    // Direction: SCtoBS - initiates attach propagation
	CmdAttachPropagateResponse = "attPrpRsp" // Direction: BStoSC - response for attach propagate
	CmdAttachPropagateComplete = "attPrpCmp" // Direction: BStoSC - completion for attach propagate

	// Detach propagate operations.
	CmdDetachPropagate         = "detPrp"    // Direction: SCtoBS - initiates detach propagation
	CmdDetachPropagateResponse = "detPrpRsp" // Direction: BStoSC - response for detach propagate
	CmdDetachPropagateComplete = "detPrpCmp" // Direction: BStoSC - completion for detach propagate

	// UL data operations.
	CmdULData         = "ulData"    // Direction: BStoSC - initiates uplink data
	CmdULDataResponse = "ulDataRsp" // Direction: SCtoBS - response for uplink data
	CmdULDataComplete = "ulDataCmp" // Direction: BStoSC - completion for uplink data

	// UL data transmit operations.
	CmdULDataTransmit         = "ulDataTx"    // Direction: SCtoBS - initiates uplink data transmit
	CmdULDataTransmitResponse = "ulDataTxRsp" // Direction: BStoSC - response for uplink data transmit
	CmdULDataTransmitComplete = "ulDataTxCmp" // Direction: BStoSC - completion for uplink data transmit

	// DL data queue operations.
	CmdDLDataQueue         = "dlDataQue"    // Direction: SCtoBS - initiates downlink data queue
	CmdDLDataQueueResponse = "dlDataQueRsp" // Direction: BStoSC - response for downlink queue
	CmdDLDataQueueComplete = "dlDataQueCmp" // Direction: BStoSC - completion for downlink queue

	// DL data revoke operations.
	CmdDLDataRevoke         = "dlDataRev"    // Direction: SCtoBS - initiates downlink data revocation
	CmdDLDataRevokeResponse = "dlDataRevRsp" // Direction: BStoSC - response for downlink revoke
	CmdDLDataRevokeComplete = "dlDataRevCmp" // Direction: BStoSC - completion for downlink revoke

	// DL data result operations (BSSCI §3.14). Note: SCACI §3.12 uses different response/complete names.
	CmdDLDataResult         = "dlDataRes"    // Direction: BStoSC - initiates downlink data result (shared: BSSCI §3.14 + SCACI §3.12.1)
	CmdDLDataResultResponse = "dlDataResRsp" // Direction: SCtoBS - response for downlink result (BSSCI §3.14 only)
	CmdDLDataResultComplete = "dlDataResCmp" // Direction: BStoSC - completion for downlink result (BSSCI §3.14 only)

	// SCACI §3.12.2-3.12.3 DL Data Result response/complete (distinct from BSSCI §3.14 names above).
	// Note: dlDataRes (§3.12.1) reuses CmdDLDataResult - same wire string for both protocols.
	CmdDLDataResultResponseSCACI = "txDataResRsp" // Direction: BStoSC - response for downlink result (SCACI §3.12.2, AC→SC)
	CmdDLDataResultCompleteSCACI = "txDataResCmp" // Direction: SCtoBS - completion for downlink result (SCACI §3.12.3, SC→AC)

	// DL RX status operations.
	CmdDLRxStatus         = "dlRxStat"    // Direction: BStoSC - initiates downlink RX status
	CmdDLRxStatusResponse = "dlRxStatRsp" // Direction: SCtoBS - response for DL RX status
	CmdDLRxStatusComplete = "dlRxStatCmp" // Direction: BStoSC - completion for DL RX status

	// DL RX status query operations.
	CmdDLRxStatusQuery         = "dlRxStatQry"    // Direction: SCtoBS - initiates downlink RX status query
	CmdDLRxStatusQueryResponse = "dlRxStatQryRsp" // Direction: BStoSC - response for DL RX status query
	CmdDLRxStatusQueryComplete = "dlRxStatQryCmp" // Direction: BStoSC - completion for DL RX status query

	// Error operations.
	CmdError    = "error"    // Direction: Bidirectional - error message
	CmdErrorAck = "errorAck" // Direction: Bidirectional - error acknowledgment

	// Variable MAC (VM) sublayer commands (BSSCI Attachment VM v0.4).
	CmdVMActivate           = "vm.activate"      // Direction: SCtoBS - initiates VM activation
	CmdVMActivateResponse   = "vm.activateRsp"   // Direction: BStoSC - response for VM activation
	CmdVMActivateComplete   = "vm.activateCmp"   // Direction: BStoSC - completion for VM activation
	CmdVMDeactivate         = "vm.deactivate"    // Direction: SCtoBS - initiates VM deactivation
	CmdVMDeactivateResponse = "vm.deactivateRsp" // Direction: BStoSC - response for VM deactivation
	CmdVMDeactivateComplete = "vm.deactivateCmp" // Direction: BStoSC - completion for VM deactivation
	CmdVMStatus             = "vm.status"        // Direction: SCtoBS - initiates VM status query
	CmdVMStatusResponse     = "vm.statusRsp"     // Direction: BStoSC - response for VM status
	CmdVMStatusComplete     = "vm.statusCmp"     // Direction: BStoSC - completion for VM status
	CmdVMDLData             = "vm.dlData"        // Direction: SCtoBS - initiates VM downlink data
	CmdVMDLDataResponse     = "vm.dlDataRsp"     // Direction: BStoSC - response for VM downlink data
	CmdVMDLDataComplete     = "vm.dlDataCmp"     // Direction: BStoSC - completion for VM downlink data
)

// ============================================================================
// MIOTY Constants
// ============================================================================

// MIOTY protocol constants for modes, profiles, results, message types, and event categories.
const (
	// ModeULP is the Ultra Low Power MIOTY transmission mode.
	ModeULP = "ulp"
	// ModeULPRep is ULP with repetition for improved reliability.
	ModeULPRep = "ulp-rep"
	// ModeULPLL is ULP Low Latency mode.
	ModeULPLL = "ulp-ll"

	// ProfileEU1 is the European regional profile 1.
	ProfileEU1 = "eu1"
	// ProfileUS1 is the US regional profile 1.
	ProfileUS1 = "us1"

	// ResultSent indicates downlink was transmitted successfully.
	ResultSent = "sent"
	// ResultExpired indicates downlink expired before transmission.
	ResultExpired = "expired"
	// ResultInvalid indicates downlink was invalid.
	ResultInvalid = "invalid"

	// MessageTypeULData is uplink data message type (DB migration 018).
	MessageTypeULData = "ul_data"
	// MessageTypeDLData is downlink data message type.
	MessageTypeDLData = "dl_data"
	// MessageTypeAttach is endpoint attach message type.
	MessageTypeAttach = "attach"
	// MessageTypeDetach is endpoint detach message type.
	MessageTypeDetach = "detach"
	// MessageTypeAttachPropagate is attach propagation message type.
	MessageTypeAttachPropagate = "attach_propagate"
	// MessageTypeDetachPropagate is detach propagation message type.
	MessageTypeDetachPropagate = "detach_propagate"
	// MessageTypeStatus is status message type.
	MessageTypeStatus = "status"
	// MessageTypeRegister is registration message type.
	MessageTypeRegister = "register"
	// MessageTypeDeregister is deregistration message type.
	MessageTypeDeregister = "deregister"

	// DirectionUplink is the uplink direction (EP to BS to SC).
	DirectionUplink = "uplink"
	// DirectionDownlink is the downlink direction (SC to BS to EP).
	DirectionDownlink = "downlink"

	// InterfaceBSSCI is the Base Station Service Center Interface.
	InterfaceBSSCI = "bssci"
	// InterfaceSCACI is the Service Center Application Center Interface.
	InterfaceSCACI = "scaci"

	// SourceTypeEndpoint indicates endpoint as event source.
	SourceTypeEndpoint = "endpoint"
	// SourceTypeBaseStation indicates base station as event source.
	SourceTypeBaseStation = "basestation"
	// SourceTypeServiceCenter indicates service center as event source.
	SourceTypeServiceCenter = "service_center"

	// CategoryEndpoint is for endpoint lifecycle and operation events.
	CategoryEndpoint = "endpoint"
	// CategoryMessage is for message processing and data events.
	CategoryMessage = "message"
	// CategoryBaseStation is for base station operation events.
	CategoryBaseStation = "basestation"
	// CategoryServiceCenter is for service center operation events.
	CategoryServiceCenter = "service_center"
	// CategorySystem is for system-level protocol events (BSSCI/SCACI).
	CategorySystem = "system"
	// CategoryUplink is for uplink-specific events.
	CategoryUplink = "uplink"
	// CategoryDownlink is for downlink-specific events.
	CategoryDownlink = "downlink"
)

// ============================================================================
// POSIX Error Codes (for status and error messages)
// ============================================================================

// POSIX error code constants for BSSCI status and error messages.
const (
	// ErrorOK indicates success (0).
	ErrorOK = 0
	// ErrorPermission indicates operation not permitted (EPERM).
	ErrorPermission = 1
	// ErrorNoEntry indicates no such file or directory (ENOENT).
	ErrorNoEntry = 2
	// ErrorIO indicates I/O error (EIO).
	ErrorIO = 5
	// ErrorNoDevice indicates no such device or address (ENXIO).
	ErrorNoDevice = 6
	// ErrorTooBig indicates argument list too long (E2BIG).
	ErrorTooBig = 7
	// ErrorNoMemory indicates out of memory (ENOMEM).
	ErrorNoMemory = 12
	// ErrorAccess indicates permission denied (EACCES).
	ErrorAccess = 13
	// ErrorBusy indicates device or resource busy (EBUSY).
	ErrorBusy = 16
	// ErrorExists indicates file exists (EEXIST).
	ErrorExists = 17
	// ErrorNotDirectory indicates not a directory (ENOTDIR).
	ErrorNotDirectory = 20
	// ErrorInvalidArgument indicates invalid argument (EINVAL).
	ErrorInvalidArgument = 22
	// ErrorNoSpace indicates no space left on device (ENOSPC).
	ErrorNoSpace = 28
	// ErrorNotSupported indicates operation not supported (EOPNOTSUPP).
	ErrorNotSupported = 95
	// ErrorTimeout indicates connection timed out (ETIMEDOUT).
	ErrorTimeout = 110
	// ErrorInProgress indicates operation now in progress (EINPROGRESS).
	ErrorInProgress = 115
)

// MaxDLUserDataBytes is the maximum downlink payload size per MIOTY radio protocol §4.3.2
const MaxDLUserDataBytes = 200

// DL RX Status field validation ranges (BSSCI §5.15.1)
const (
	DLRxSnrMinDB   = -30.0
	DLRxSnrMaxDB   = 30.0
	DLRxRssiMinDBm = -150.0
	DLRxRssiMaxDBm = 0.0
)

// ============================================================================
// Helper Functions
// ============================================================================

// EUI64FromBytes converts an 8-byte array to EUI64
func EUI64FromBytes(b [8]byte) EUI64 {
	var result uint64
	for i := 0; i < 8; i++ {
		result = (result << 8) | uint64(b[i])
	}
	return EUI64(result)
}

// ToBytes converts EUI64 to an 8-byte array
func (e EUI64) ToBytes() [8]byte {
	var result [8]byte
	for i := 7; i >= 0; i-- {
		result[i] = byte(e)
		e >>= 8
	}
	return result
}

// String returns the EUI64 as a hex string
func (e EUI64) String() string {
	bytes := e.ToBytes()
	return hex.EncodeToString(bytes[:])
}

// TimeFromUnixNano converts UnixNano to time.Time
func TimeFromUnixNano(ns UnixNano) time.Time {
	return time.Unix(0, int64(ns))
}

// ToUnixNano converts time.Time to UnixNano
func ToUnixNano(t time.Time) UnixNano {
	return UnixNano(t.UnixNano())
}

// EndpointSummary represents basic endpoint information for API responses
type EndpointSummary struct {
	EUI          string    `json:"eui"`
	MessageCount int       `json:"messageCount"`
	LastSeen     time.Time `json:"lastSeen"`
	FirstSeen    time.Time `json:"firstSeen"`
	AvgRSSI      float64   `json:"avgRssi"`
	AvgSNR       float64   `json:"avgSnr"`
}

// EndpointResponse represents a paginated endpoint response
type EndpointResponse struct {
	Endpoints  []EndpointSummary `json:"endpoints"`
	Page       int               `json:"page"`
	PageSize   int               `json:"pageSize"`
	TotalCount int               `json:"totalCount"`
	TotalPages int               `json:"totalPages"`
}

// EndpointStats represents detailed endpoint statistics
type EndpointStats struct {
	EUI                string    `json:"eui"`
	TotalMessages      int       `json:"totalMessages"`
	LastSeen           time.Time `json:"lastSeen"`
	FirstSeen          time.Time `json:"firstSeen"`
	AvgRSSI            float64   `json:"avgRssi"`
	MinRSSI            float64   `json:"minRssi"`
	MaxRSSI            float64   `json:"maxRssi"`
	AvgSNR             float64   `json:"avgSnr"`
	MinSNR             float64   `json:"minSnr"`
	MaxSNR             float64   `json:"maxSnr"`
	UniqueBaseStations int       `json:"uniqueBaseStations"`
	ActiveDays         int       `json:"activeDays"`
}

// BaseStationSummary represents basic base station information
type BaseStationSummary struct {
	EUI             string    `json:"eui"`
	Name            string    `json:"name"`
	IsOnline        bool      `json:"isOnline"`
	MessageCount    int       `json:"messageCount"`
	LastSeen        time.Time `json:"lastSeen"`
	FirstSeen       time.Time `json:"firstSeen"`
	UniqueEndpoints int       `json:"uniqueEndpoints"`
	AvgRSSI         float64   `json:"avgRssi"`
	AvgSNR          float64   `json:"avgSnr"`
}

// BaseStationResponse represents a paginated base station response
type BaseStationResponse struct {
	BaseStations []BaseStationSummary `json:"baseStations"`
	Page         int                  `json:"page"`
	PageSize     int                  `json:"pageSize"`
	TotalCount   int                  `json:"totalCount"`
	TotalPages   int                  `json:"totalPages"`
}

// BaseStationDetails represents detailed base station information from the basestations table
// This includes MIOTY status response fields per BSSCI v1.0.0 Section 3.5.2
type BaseStationDetails struct {
	ID             int64      `json:"id"`
	EUI            string     `json:"eui"`
	Name           string     `json:"name"`
	Description    *string    `json:"description,omitempty"`
	ConnectionType string     `json:"connectionType"`
	IsOnline       bool       `json:"isOnline"`
	LastSeenAt     *time.Time `json:"lastSeenAt,omitempty"`

	// MIOTY Status Response Fields from base station per BSSCI v1.0.0 Section 3.5.2
	StatusCode         int        `json:"statusCode,omitempty"`         // Status code from base station, using POSIX error numbers, 0 for "ok"
	StatusMessage      *string    `json:"statusMessage,omitempty"`      // Status message from base station
	SystemTime         *int64     `json:"systemTime,omitempty"`         // Unix UTC system time, 64 bit, ns resolution
	DutyCycle          *float64   `json:"dutyCycle,omitempty"`          // Fraction of TX time, sliding window over one hour
	UptimeSeconds      *int64     `json:"uptimeSeconds,omitempty"`      // System uptime in seconds
	TemperatureCelsius *float64   `json:"temperatureCelsius,omitempty"` // System temperature in degree Celsius
	CPULoad            *float64   `json:"cpuLoad,omitempty"`            // CPU utilization, normalized to 1.0 for all cores
	MemoryLoad         *float64   `json:"memoryLoad,omitempty"`         // Memory utilization, normalized to 1.0
	BSConfig           *string    `json:"bsConfig,omitempty"`           // Configuration object from base station (JSON string)
	LastStatusAt       *time.Time `json:"lastStatusAt,omitempty"`       // Timestamp when status response was last received

	// Hardware Info
	Vendor  *string `json:"vendor,omitempty"`
	Model   *string `json:"model,omitempty"`
	Version *string `json:"version,omitempty"`

	// Location
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	Altitude  *float64 `json:"altitude,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// BaseStationStats represents detailed base station statistics
type BaseStationStats struct {
	EUI             string    `json:"eui"`
	TotalMessages   int       `json:"totalMessages"`
	LastSeen        time.Time `json:"lastSeen"`
	FirstSeen       time.Time `json:"firstSeen"`
	AvgRSSI         float64   `json:"avgRssi"`
	MinRSSI         float64   `json:"minRssi"`
	MaxRSSI         float64   `json:"maxRssi"`
	AvgSNR          float64   `json:"avgSnr"`
	MinSNR          float64   `json:"minSnr"`
	MaxSNR          float64   `json:"maxSnr"`
	UniqueEndpoints int       `json:"uniqueEndpoints"`
	ActiveDays      int       `json:"activeDays"`
}

// BaseStationMessageStats holds aggregated message statistics for a base station
// Used by the MessageStore interface
type BaseStationMessageStats struct {
	TotalMessages     int64      `json:"totalMessages"`
	TotalEndpoints    int64      `json:"totalEndpoints"`
	MessagesToday     int64      `json:"messagesToday"`
	MessagesThisWeek  int64      `json:"messagesThisWeek"`
	MessagesThisMonth int64      `json:"messagesThisMonth"`
	AvgRSSI           float64    `json:"avgRssi"`
	AvgSNR            float64    `json:"avgSnr"`
	LastMessageAt     *time.Time `json:"lastMessageAt,omitempty"`
}

// ============================================================================
// Analytics Types (for dashboard and reporting)
// ============================================================================

// AnalyticsOverview represents high-level network analytics
type AnalyticsOverview struct {
	StartTime          time.Time        `json:"startTime"`
	EndTime            time.Time        `json:"endTime"`
	TotalMessages      int              `json:"totalMessages"`
	ActiveEndpoints    int              `json:"activeEndpoints"`
	ActiveBaseStations int              `json:"activeBaseStations"`
	AvgRSSI            float64          `json:"avgRssi"`
	AvgSNR             float64          `json:"avgSnr"`
	FirstMessage       time.Time        `json:"firstMessage"`
	LastMessage        time.Time        `json:"lastMessage"`
	HourlyActivity     []HourlyActivity `json:"hourlyActivity"`
}

// HourlyActivity represents message activity per hour
type HourlyActivity struct {
	Hour         time.Time `db:"hour" json:"hour"`
	MessageCount int       `db:"message_count" json:"messageCount"`
}

// ActivityAnalytics represents detailed activity analytics
type ActivityAnalytics struct {
	StartTime    time.Time          `json:"startTime"`
	EndTime      time.Time          `json:"endTime"`
	DailyStats   []DailyActivity    `json:"dailyStats"`
	TopEndpoints []EndpointActivity `json:"topEndpoints"`
}

// DailyActivity represents activity per day
type DailyActivity struct {
	Day                string `db:"day" json:"day"`
	MessageCount       int    `db:"message_count" json:"messageCount"`
	UniqueEndpoints    int    `db:"unique_endpoints" json:"uniqueEndpoints"`
	UniqueBaseStations int    `db:"unique_basestations" json:"uniqueBaseStations"`
}

// WeeklyActivity represents activity per ISO week
type WeeklyActivity struct {
	Week         time.Time `db:"week" json:"week"`
	MessageCount int       `db:"message_count" json:"messageCount"`
}

// MonthlyActivity represents activity per month
type MonthlyActivity struct {
	Month        time.Time `db:"month" json:"month"`
	MessageCount int       `db:"message_count" json:"messageCount"`
}

// EndpointActivity represents endpoint activity summary
// Note: EUI stored as uint64 for DB scanning, formatted to hex string by handler
type EndpointActivity struct {
	EUI          uint64    `db:"ep_eui" json:"-"`
	EUIFormatted string    `db:"-" json:"eui"`
	MessageCount int       `db:"message_count" json:"messageCount"`
	LastSeen     time.Time `db:"last_seen" json:"lastSeen"`
}

// SignalQualityAnalytics represents signal quality analytics
type SignalQualityAnalytics struct {
	StartTime     time.Time                  `json:"startTime"`
	EndTime       time.Time                  `json:"endTime"`
	Overall       SignalQualityOverall       `json:"overall"`
	ByBaseStation []BaseStationSignalQuality `json:"byBaseStation"`
}

// SignalQualityOverall represents overall signal quality metrics
type SignalQualityOverall struct {
	AvgRSSI       float64 `json:"avgRssi"`
	MinRSSI       float64 `json:"minRssi"`
	MaxRSSI       float64 `json:"maxRssi"`
	MedianRSSI    float64 `json:"medianRssi"`
	AvgSNR        float64 `json:"avgSnr"`
	MinSNR        float64 `json:"minSnr"`
	MaxSNR        float64 `json:"maxSnr"`
	MedianSNR     float64 `json:"medianSnr"`
	TotalMessages int     `json:"totalMessages"`
}

// BaseStationSignalQuality represents signal quality per base station
type BaseStationSignalQuality struct {
	EUI          uint64  `db:"bs_eui" json:"-"`
	EUIFormatted string  `db:"-" json:"eui"`
	AvgRSSI      float64 `db:"avg_rssi" json:"avgRssi"`
	AvgSNR       float64 `db:"avg_snr" json:"avgSnr"`
	MessageCount int     `db:"message_count" json:"messageCount"`
}

// ============================================================================
// BaseStationReception for SCACI UL Data (§3.8.1)
// ============================================================================

// BaseStationReception represents per-BS reception info per SCACI §3.8.1
// Used for SCACI UL data broadcasts and operation tracking
type BaseStationReception struct {
	BsEui      uint64      `json:"bsEui" msgpack:"bsEui"`
	RxTime     int64       `json:"rxTime" msgpack:"rxTime"`
	RxDuration *int64      `json:"rxDuration,omitempty" msgpack:"rxDuration,omitempty"`
	Snr        float64     `json:"snr" msgpack:"snr"`
	Rssi       float64     `json:"rssi" msgpack:"rssi"`
	EqSnr      *float64    `json:"eqSnr,omitempty" msgpack:"eqSnr,omitempty"`
	DlRxSnr    *float64    `json:"dlRxSnr,omitempty" msgpack:"dlRxSnr,omitempty"`
	DlRxRssi   *float64    `json:"dlRxRssi,omitempty" msgpack:"dlRxRssi,omitempty"`
	Profile    *string     `json:"profile,omitempty" msgpack:"profile,omitempty"`
	Mode       *string     `json:"mode,omitempty" msgpack:"mode,omitempty"`
	Subpackets *Subpackets `json:"subpackets,omitempty" msgpack:"subpackets,omitempty"`
}

// ULDataMessage is the persisted uplink record type.
// Extends the wire-level ULData fields with storage metadata (ID, tenant, org,
// timestamps, multi-BS receptions, blueprint decode results).
type ULDataMessage struct {
	// Required fields per BSSCI v1.0.0
	CommandType string  `json:"command" db:"command_type" msgpack:"command"`
	OpId        int64   `json:"opId" db:"op_id" msgpack:"opId"` // Signed per BSSCI §3.2: positive for BS, negative for SC
	EpEui       uint64  `json:"epEui" db:"ep_eui" msgpack:"epEui"`
	RxTime      int64   `json:"rxTime" db:"rx_time" msgpack:"rxTime"`
	PacketCnt   uint32  `json:"packetCnt" db:"packet_cnt" msgpack:"packetCnt"`
	SNR         float64 `json:"snr" db:"snr" msgpack:"snr"`
	RSSI        float64 `json:"rssi" db:"rssi" msgpack:"rssi"`
	UserData    []byte  `json:"userData" db:"user_data" msgpack:"userData"`
	DlOpen      bool    `json:"dlOpen" db:"dl_open" msgpack:"dlOpen"`
	ResponseExp bool    `json:"responseExp" db:"response_exp" msgpack:"responseExp"`
	DlAck       bool    `json:"dlAck" db:"dl_ack" msgpack:"dlAck"`

	// Optional fields
	RxDuration *int64      `json:"rxDuration,omitempty" db:"rx_duration" msgpack:"rxDuration,omitempty"`
	EqSnr      *float64    `json:"eqSnr,omitempty" db:"eq_snr" msgpack:"eqSnr,omitempty"`
	Profile    *string     `json:"profile,omitempty" db:"profile" msgpack:"profile,omitempty"`
	Mode       *string     `json:"mode,omitempty" db:"mode" msgpack:"mode,omitempty"`
	Format     *uint8      `json:"format,omitempty" db:"format" msgpack:"format,omitempty"`
	Subpackets *Subpackets `json:"subpackets,omitempty" db:"subpackets" msgpack:"subpackets,omitempty"`

	// Metadata for storage
	ID          string     `json:"id" db:"id"`
	BsEui       uint64     `json:"bsEui" db:"bs_eui"`
	TenantID    int64      `json:"tenantId" db:"tenant_id"`
	OrgUUID     *string    `json:"orgUuid,omitempty" db:"org_uuid"`
	ReceivedAt  time.Time  `json:"receivedAt" db:"received_at"`
	ProcessedAt *time.Time `json:"processedAt" db:"processed_at"`

	// Multi-BS context per SCACI §3.8.1 - persisted to base_stations JSONB column
	BaseStations []BaseStationReception `json:"baseStations,omitempty" msgpack:"baseStations,omitempty" db:"base_stations"`

	// Duplicate flag per SCACI §3.8.1 - persisted to duplicate BOOLEAN column
	// Pointer matches scaci.ULData.Duplicate *bool for consistent wire/broadcast shape
	// Always set to non-nil: &false on first reception, &true on duplicate
	Duplicate *bool `json:"duplicate,omitempty" msgpack:"duplicate,omitempty"`

	// Blueprint decoding metadata (Migration 000105)
	// Stores decoded payload and decoding context per MIOTY Application Layer Specification
	DecodedPayload     json.RawMessage `json:"decodedPayload,omitempty" db:"decoded_payload" msgpack:"decodedPayload,omitempty"`
	BlueprintTypeEUI   []byte          `json:"blueprintTypeEui,omitempty" db:"blueprint_type_eui" msgpack:"blueprintTypeEui,omitempty"`       // 8-byte Type EUI used for decoding
	BlueprintVersionID *uuid.UUID      `json:"blueprintVersionId,omitempty" db:"blueprint_version_id" msgpack:"blueprintVersionId,omitempty"` // UUID reference to blueprints.id
	DecodeStatus       string          `json:"decodeStatus,omitempty" db:"decode_status" msgpack:"decodeStatus,omitempty"`                    // pending, success, failed, skipped (use centralized constants)
	DecodeErrorCode    string          `json:"decodeErrorCode,omitempty" db:"decode_error_code" msgpack:"decodeErrorCode,omitempty"`          // Error token if decode_status is failed
}

// DetachMessage represents a detach message for storage in mioty_messages table
// Matches BSSCI §5.7 Detach operation with database persistence fields
type DetachMessage struct {
	// Required fields per BSSCI v1.0.0 §5.7.1
	CommandType string  `json:"command" db:"command" msgpack:"command"`
	OpId        int64   `json:"opId" db:"operation_id" msgpack:"opId"` // Signed: positive for BS
	EpEui       []byte  `json:"epEui" db:"ep_eui" msgpack:"epEui"`     // 8-byte EUI (BYTEA in DB)
	RxTime      int64   `json:"rxTime" db:"rx_time" msgpack:"rxTime"`  // Unix UTC center of last subpacket (ns)
	PacketCnt   uint32  `json:"packetCnt" db:"packet_cnt" msgpack:"packetCnt"`
	SNR         float64 `json:"snr" db:"snr" msgpack:"snr"`
	RSSI        float64 `json:"rssi" db:"rssi" msgpack:"rssi"`
	Signature   []byte  `json:"sign" db:"signature" msgpack:"sign"` // 4-byte EP signature (BYTEA in DB)

	// Optional fields per BSSCI v1.0.0 §5.7.1
	RxDuration *int64      `json:"rxDuration,omitempty" db:"rx_duration" msgpack:"rxDuration,omitempty"`
	EqSnr      *float64    `json:"eqSnr,omitempty" db:"eq_snr" msgpack:"eqSnr,omitempty"`
	Profile    *string     `json:"profile,omitempty" db:"profile" msgpack:"profile,omitempty"`
	Subpackets *Subpackets `json:"subpackets,omitempty" db:"subpackets" msgpack:"subpackets,omitempty"`

	// Metadata for mioty_messages table storage
	ID             string     `json:"id" db:"id"`                              // UUID
	BasestationEui []byte     `json:"bsEui" db:"basestation_eui"`              // 8-byte BS EUI (BYTEA in DB)
	TenantID       int64      `json:"tenantId" db:"tenant_id"`                 // Endpoint owner tenant (roaming support)
	OrgUUID        *string    `json:"orgUuid,omitempty" db:"org_uuid"`         // Endpoint owner org (roaming support)
	MessageType    string     `json:"messageType" db:"message_type"`           // "detach" enum value
	Direction      string     `json:"direction" db:"direction"`                // "uplink" enum value
	InterfaceType  string     `json:"interfaceType" db:"interface_type"`       // "bssci" enum value
	ReceivedAt     time.Time  `json:"receivedAt" db:"received_at"`             // TIMESTAMPTZ
	ProcessedAt    *time.Time `json:"processedAt,omitempty" db:"processed_at"` // TIMESTAMPTZ (nullable)
	CreatedAt      time.Time  `json:"createdAt" db:"created_at"`               // TIMESTAMPTZ
	UpdatedAt      time.Time  `json:"updatedAt" db:"updated_at"`               // TIMESTAMPTZ
}

// AttachMessage represents an attach message per BSSCI v1.0.0 §5.6-5.6.3
type AttachMessage struct {
	// Required fields per BSSCI v1.0.0 §5.6.1
	CommandType string  `json:"command" db:"command" msgpack:"command"`
	OpId        int64   `json:"opId" db:"operation_id" msgpack:"opId"` // Signed: positive for BS
	EpEui       []byte  `json:"epEui" db:"ep_eui" msgpack:"epEui"`     // 8-byte EUI (BYTEA in DB)
	RxTime      int64   `json:"rxTime" db:"rx_time" msgpack:"rxTime"`  // Unix UTC center of last subpacket (ns)
	PacketCnt   uint32  `json:"packetCnt" db:"packet_cnt" msgpack:"packetCnt"`
	SNR         float64 `json:"snr" db:"snr" msgpack:"snr"`
	RSSI        float64 `json:"rssi" db:"rssi" msgpack:"rssi"`
	Signature   []byte  `json:"sign" db:"signature" msgpack:"sign"` // 4-byte EP signature (BYTEA in DB)

	// Optional fields per BSSCI v1.0.0 §5.6.1
	RxDuration *int64      `json:"rxDuration,omitempty" db:"rx_duration" msgpack:"rxDuration,omitempty"`
	EqSnr      *float64    `json:"eqSnr,omitempty" db:"eq_snr" msgpack:"eqSnr,omitempty"`
	Profile    *string     `json:"profile,omitempty" db:"profile" msgpack:"profile,omitempty"`
	Subpackets *Subpackets `json:"subpackets,omitempty" db:"subpackets" msgpack:"subpackets,omitempty"`

	// Metadata for mioty_messages table storage
	ID             string     `json:"id" db:"id"`                              // UUID
	BasestationEui []byte     `json:"bsEui" db:"basestation_eui"`              // 8-byte BS EUI (BYTEA in DB)
	TenantID       int64      `json:"tenantId" db:"tenant_id"`                 // Tenant isolation
	MessageType    string     `json:"messageType" db:"message_type"`           // "attach" enum value
	Direction      string     `json:"direction" db:"direction"`                // "uplink" enum value
	InterfaceType  string     `json:"interfaceType" db:"interface_type"`       // "bssci" enum value
	ReceivedAt     time.Time  `json:"receivedAt" db:"received_at"`             // TIMESTAMPTZ
	ProcessedAt    *time.Time `json:"processedAt,omitempty" db:"processed_at"` // TIMESTAMPTZ (nullable)
	CreatedAt      time.Time  `json:"createdAt" db:"created_at"`               // TIMESTAMPTZ
	UpdatedAt      time.Time  `json:"updatedAt" db:"updated_at"`               // TIMESTAMPTZ
}

// AttachPropagateMessage represents an attach propagate message per BSSCI v1.0.0 §5.8-5.8.3
// Sent by Service Center to base stations to propagate endpoint attachment information
type AttachPropagateMessage struct {
	// Required fields per BSSCI v1.0.0 §5.8.1
	CommandType   string `json:"command" db:"command" msgpack:"command"`
	OpId          int64  `json:"opId" db:"operation_id" msgpack:"opId"`                      // Signed: negative for SC-initiated
	EpEui         uint64 `json:"epEui" db:"ep_eui" msgpack:"epEui"`                          // Endpoint EUI (Numeric[8])
	Bidi          bool   `json:"bidi" db:"bidi" msgpack:"bidi"`                              // Bidirectional capability
	NwkSnKey      []byte `json:"nwkSnKey" db:"nwk_sn_key" msgpack:"nwkSnKey"`                // Network session key (Numeric[16])
	ShAddr        uint16 `json:"shAddr" db:"sh_addr" msgpack:"shAddr"`                       // Short address
	LastPacketCnt uint32 `json:"lastPacketCnt" db:"last_packet_cnt" msgpack:"lastPacketCnt"` // Last packet counter

	// Optional fields per BSSCI v1.0.0 §5.8.1
	DualChan    bool `json:"dualChan" db:"dual_channel" msgpack:"dualChan"`        // Dual channel mode
	Repetition  bool `json:"repetition" db:"repetition" msgpack:"repetition"`      // DL repetition
	WideCarrOff bool `json:"wideCarrOff" db:"wide_carr_off" msgpack:"wideCarrOff"` // Wide carrier offset
	LongBlkDist bool `json:"longBlkDist" db:"long_blk_dist" msgpack:"longBlkDist"` // Long DL interblock distance

	// Metadata for messages table storage (migration 047)
	// NOTE: ID is auto-generated by repository layer (uuid.New()) before INSERT.
	// Kept for parsing/transport compatibility with existing message flows.
	ID             string     `json:"id" db:"id"`                              // UUID - auto-generated, set by repository
	BasestationEui []byte     `json:"bsEui" db:"basestation_eui"`              // 8-byte BS EUI (BYTEA in DB)
	TenantID       int64      `json:"tenantId" db:"tenant_id"`                 // Session tenant (BS owner)
	OrgUUID        *string    `json:"orgUuid,omitempty" db:"org_uuid"`         // Endpoint owner org (roaming support)
	MessageType    string     `json:"messageType" db:"message_type"`           // "attach_propagate" enum value
	Direction      string     `json:"direction" db:"direction"`                // "downlink" enum value
	InterfaceType  string     `json:"interfaceType" db:"interface_type"`       // "bssci" enum value
	ReceivedAt     time.Time  `json:"receivedAt" db:"received_at"`             // TIMESTAMPTZ
	ProcessedAt    *time.Time `json:"processedAt,omitempty" db:"processed_at"` // TIMESTAMPTZ (nullable)
	CreatedAt      time.Time  `json:"createdAt" db:"created_at"`               // TIMESTAMPTZ
	UpdatedAt      time.Time  `json:"updatedAt" db:"updated_at"`               // TIMESTAMPTZ
}

// DetachPropagateMessage represents a detach propagate message per BSSCI v1.0.0 §5.9-5.9.3
// Sent by Service Center to base stations to propagate endpoint detachment information
type DetachPropagateMessage struct {
	// Required fields per BSSCI v1.0.0 §5.9.1
	CommandType string `json:"command" db:"command" msgpack:"command"`
	OpId        int64  `json:"opId" db:"operation_id" msgpack:"opId"` // Signed: negative for SC-initiated
	EpEui       uint64 `json:"epEui" db:"ep_eui" msgpack:"epEui"`     // Endpoint EUI (Numeric[8])

	// Metadata for messages table storage (migration 047)
	// NOTE: ID is auto-generated by repository layer (uuid.New()) before INSERT.
	// Kept for parsing/transport compatibility with existing message flows.
	ID             string     `json:"id" db:"id"`                              // UUID - auto-generated, set by repository
	BasestationEui []byte     `json:"bsEui" db:"basestation_eui"`              // 8-byte BS EUI (BYTEA in DB)
	TenantID       int64      `json:"tenantId" db:"tenant_id"`                 // Session tenant (BS owner)
	OrgUUID        *string    `json:"orgUuid,omitempty" db:"org_uuid"`         // Endpoint owner org (roaming support)
	MessageType    string     `json:"messageType" db:"message_type"`           // "detach_propagate" enum value
	Direction      string     `json:"direction" db:"direction"`                // "downlink" enum value
	InterfaceType  string     `json:"interfaceType" db:"interface_type"`       // "bssci" enum value
	ReceivedAt     time.Time  `json:"receivedAt" db:"received_at"`             // TIMESTAMPTZ
	ProcessedAt    *time.Time `json:"processedAt,omitempty" db:"processed_at"` // TIMESTAMPTZ (nullable)
	CreatedAt      time.Time  `json:"createdAt" db:"created_at"`               // TIMESTAMPTZ
	UpdatedAt      time.Time  `json:"updatedAt" db:"updated_at"`               // TIMESTAMPTZ
}

// BaseStationStatusRecord represents a base station status snapshot per BSSCI v1.0.0 §3.5.2
type BaseStationStatusRecord struct {
	ID             int64           `db:"id" json:"id"`
	TenantID       int64           `db:"tenant_id" json:"tenantId"`
	BaseStationID  int64           `db:"basestation_id" json:"basestationId"`
	BasestationEUI []byte          `db:"basestation_eui" json:"bsEui"` // 8-byte BS EUI (BYTEA in DB)
	OperationID    *int64          `db:"operation_id" json:"operationId,omitempty"`
	StatusCode     int             `db:"status_code" json:"statusCode"` // INTEGER (32-bit)
	StatusMessage  string          `db:"status_message" json:"statusMessage"`
	SystemTime     int64           `db:"system_time" json:"systemTime"` // Unix UTC ns
	DutyCycle      *float64        `db:"duty_cycle" json:"dutyCycle,omitempty"`
	UptimeSeconds  *int64          `db:"uptime" json:"uptimeSeconds,omitempty"`
	Temperature    *float64        `db:"temperature" json:"temperature,omitempty"` // Celsius
	CPULoad        *float64        `db:"cpu_load" json:"cpuLoad,omitempty"`
	MemoryLoad     *float64        `db:"memory_load" json:"memoryLoad,omitempty"`
	Config         json.RawMessage `db:"config" json:"config,omitempty"` // JSONB
	Latitude       *float64        `db:"latitude" json:"latitude,omitempty"`
	Longitude      *float64        `db:"longitude" json:"longitude,omitempty"`
	Altitude       *float64        `db:"altitude" json:"altitude,omitempty"`
	ReceivedAt     time.Time       `db:"received_at" json:"receivedAt"` // TIMESTAMPTZ
}
