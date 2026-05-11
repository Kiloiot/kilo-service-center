// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/vmihailenco/msgpack/v5"
)

// UUID16 is a 16-byte UUID that marshals as Numeric[16] per SCACI spec
//
// The MIOTY SCACI specification requires UUIDs to be encoded as arrays of 16 integers,
// not as base64 strings. This type implements custom MessagePack marshaling to ensure
// compliance with the spec.
type UUID16 [16]byte

// MarshalMsgpack encodes UUID16 as an array of 16 uint8 values per SCACI spec
func (u UUID16) MarshalMsgpack() ([]byte, error) {
	return msgpack.Marshal(u[:])
}

// UnmarshalMsgpack decodes UUID16 from an array of 16 uint8 values per SCACI spec
func (u *UUID16) UnmarshalMsgpack(dec *msgpack.Decoder) error {
	length, err := dec.DecodeArrayLen()
	if err != nil {
		return err
	}
	if length != 16 {
		return fmt.Errorf("expected 16 bytes for UUID, got %d", length)
	}
	for i := 0; i < 16; i++ {
		b, err := dec.DecodeUint8()
		if err != nil {
			return err
		}
		u[i] = b
	}
	return nil
}

// BaseMessage contains core fields present in ALL SCACI messages per §3.2
//
//revive:disable:var-naming OpId uses lowercase 'd' per MIOTY SCACI §3.2
type BaseMessage struct {
	Command string `json:"command" msgpack:"command"` // Operation mnemonic (SCACI spec field: "command")
	OpId    int64  `json:"opId" msgpack:"opId"`       // Operation ID (signed: positive for AC, negative for SC)
}

// ============================================================================
// §3.3 Connect Operation
// ============================================================================

// Connect represents the "con" message from Application Center per SCACI §3.3.1
//
//revive:disable:var-naming SnAcOpId/SnScOpId use lowercase 'd' per MIOTY SCACI §3.3.1
type Connect struct {
	BaseMessage
	Version   string      `json:"version" msgpack:"version"`                         // Protocol version "major.minor.patch"
	AcEui     uint64      `json:"acEui" msgpack:"acEui"`                             // Application Center EUI64
	Vendor    *string     `json:"vendor,omitempty" msgpack:"vendor,omitempty"`       // Optional vendor name
	Model     *string     `json:"model,omitempty" msgpack:"model,omitempty"`         // Optional model
	Name      *string     `json:"name,omitempty" msgpack:"name,omitempty"`           // Optional name
	SwVersion *string     `json:"swVersion,omitempty" msgpack:"swVersion,omitempty"` // Optional software version
	Info      interface{} `json:"info,omitempty" msgpack:"info,omitempty"`           // Optional arbitrary key-value pairs
	SnAcUUID  UUID16      `json:"snAcUuid" msgpack:"snAcUuid"`                       // AC session UUID (must match to resume)
	SnAcOpId  *int64      `json:"snAcOpId,omitempty" msgpack:"snAcOpId,omitempty"`   // Minimum required known AC opId to resume
	SnScOpId  *int64      `json:"snScOpId,omitempty" msgpack:"snScOpId,omitempty"`   // Maximum known SC opId to resume
}

// ConnectResponse represents the "conRsp" message from Service Center per SCACI §3.3.2
type ConnectResponse struct {
	BaseMessage
	Version   *string     `json:"version,omitempty" msgpack:"version,omitempty"` // Supported version (default: requested)
	ScEui     uint64      `json:"scEui" msgpack:"scEui"`                         // Service Center EUI64
	Vendor    *string     `json:"vendor,omitempty" msgpack:"vendor,omitempty"`
	Model     *string     `json:"model,omitempty" msgpack:"model,omitempty"`
	Name      *string     `json:"name,omitempty" msgpack:"name,omitempty"`
	SwVersion *string     `json:"swVersion,omitempty" msgpack:"swVersion,omitempty"`
	Info      interface{} `json:"info,omitempty" msgpack:"info,omitempty"`
	SnResume  bool        `json:"snResume" msgpack:"snResume"` // True if previous session resumed
	SnScUUID  UUID16      `json:"snScUuid" msgpack:"snScUuid"` // SC session UUID (must match to resume)
}

// ConnectComplete represents the "conCmp" message per SCACI §3.3.3
type ConnectComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// §3.4 Ping Operation
// ============================================================================

// Ping represents the "ping" message per SCACI §3.4.1
type Ping struct {
	BaseMessage
	// No additional fields beyond core
}

// PingResponse represents the "pingRsp" message per SCACI §3.4.2
type PingResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// PingComplete represents the "pingCmp" message per SCACI §3.4.3
type PingComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// §3.5 Status Operation
// ============================================================================

// Status represents the "status" message per SCACI §3.5.1
type Status struct {
	BaseMessage
	// No additional fields beyond core
}

// BaseStationStatus represents base station status info per SCACI §3.5.2
type BaseStationStatus struct {
	Eui         *uint64     `json:"eui,omitempty" msgpack:"eui,omitempty"`                 // EUI64 of Base Station
	Bidi        *bool       `json:"bidi,omitempty" msgpack:"bidi,omitempty"`               // Bidirectional capability
	Code        *int        `json:"code,omitempty" msgpack:"code,omitempty"`               // POSIX status code
	Message     *string     `json:"message,omitempty" msgpack:"message,omitempty"`         // Status message
	GeoLocation *[3]float64 `json:"geoLocation,omitempty" msgpack:"geoLocation,omitempty"` // [Lat, Lon, Alt]
	Uptime      *int64      `json:"uptime,omitempty" msgpack:"uptime,omitempty"`           // Uptime in seconds
	Vendor      *string     `json:"vendor,omitempty" msgpack:"vendor,omitempty"`
	Model       *string     `json:"model,omitempty" msgpack:"model,omitempty"`
	Name        *string     `json:"name,omitempty" msgpack:"name,omitempty"`
	Info        interface{} `json:"info,omitempty" msgpack:"info,omitempty"`
	UlLoad      *float64    `json:"ulLoad,omitempty" msgpack:"ulLoad,omitempty"` // Uplink channel load (normalized to 1.0)
	DlLoad      *float64    `json:"dlLoad,omitempty" msgpack:"dlLoad,omitempty"` // Downlink channel load (normalized to 1.0)
}

// StatusResponse represents the "statusRsp" message per SCACI §3.5.2
type StatusResponse struct {
	BaseMessage
	Code         int                 `json:"code" msgpack:"code"`                                     // POSIX error number (0 = "ok")
	Message      string              `json:"message" msgpack:"message"`                               // Status message
	Time         int64               `json:"time" msgpack:"time"`                                     // Unix UTC time (64-bit, ns resolution)
	Uptime       *int64              `json:"uptime,omitempty" msgpack:"uptime,omitempty"`             // System uptime in seconds
	BaseStations []BaseStationStatus `json:"baseStations,omitempty" msgpack:"baseStations,omitempty"` // Array of BS status
}

// StatusComplete represents the "statusCmp" message per SCACI §3.5.3
type StatusComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// §3.6 Register Operation
// ============================================================================

// Register represents the "reg" message per SCACI §3.6.1
type Register struct {
	BaseMessage
	EpEui       uint64   `json:"epEui" msgpack:"epEui"`             // End Point EUI64
	Bidi        bool     `json:"bidi" msgpack:"bidi"`               // Bidirectional End Point
	PreAttach   bool     `json:"preAttach" msgpack:"preAttach"`     // Pre-attach at Base Stations
	NwkKey      [16]byte `json:"nwkKey" msgpack:"nwkKey"`           // 16-byte network key
	ShAddr      uint16   `json:"shAddr" msgpack:"shAddr"`           // Short address
	AttachCnt   uint32   `json:"attachCnt" msgpack:"attachCnt"`     // Last known attachment counter
	PacketCnt   uint32   `json:"packetCnt" msgpack:"packetCnt"`     // Last known packet counter
	DualChan    bool     `json:"dualChan" msgpack:"dualChan"`       // Dual channel mode
	Repetition  bool     `json:"repetition" msgpack:"repetition"`   // DL repetition enabled
	WideCarrOff bool     `json:"wideCarrOff" msgpack:"wideCarrOff"` // Wide carrier offset
	LongBlkDist bool     `json:"longBlkDist" msgpack:"longBlkDist"` // Long DL interblock distance
}

// RegisterResponse represents the "regRsp" message per SCACI §3.6.2
type RegisterResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// RegisterComplete represents the "regCmp" message per SCACI §3.6.3
type RegisterComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// §3.7 Deregister Operation
// ============================================================================

// Deregister represents the "dereg" message per SCACI §3.7.1
type Deregister struct {
	BaseMessage
	EpEui uint64 `json:"epEui" msgpack:"epEui"` // End Point EUI64
}

// DeregisterResponse represents the "deregRsp" message per SCACI §3.7.2
type DeregisterResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// DeregisterComplete represents the "deregCmp" message per SCACI §3.7.3
type DeregisterComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// §3.8 UL Data Operation
// ============================================================================

// Subpackets is an alias to canonical mioty.Subpackets
// Use centralized type from KC-DB/storage/mioty/types.go
type Subpackets = mioty.Subpackets

// BaseStationReception is an alias to canonical mioty.BaseStationReception
// Use centralized type from KC-DB/storage/mioty/types.go per SCACI §3.8.1
type BaseStationReception = mioty.BaseStationReception

// ULData represents the "ulData" message per SCACI §3.8.1
type ULData struct {
	BaseMessage
	EpEui        uint64                 `json:"epEui" msgpack:"epEui"`                             // End Point EUI64
	BaseStations []BaseStationReception `json:"baseStations" msgpack:"baseStations"`               // Array of BS receptions
	PacketCnt    uint32                 `json:"packetCnt" msgpack:"packetCnt"`                     // EP packet counter
	UserData     []byte                 `json:"userData" msgpack:"userData"`                       // n-byte user data (may be empty)
	Format       *uint8                 `json:"format,omitempty" msgpack:"format,omitempty"`       // User data format ID (8-bit)
	DlOpen       bool                   `json:"dlOpen" msgpack:"dlOpen"`                           // DL window opened
	ResponseExp  bool                   `json:"responseExp" msgpack:"responseExp"`                 // EP expects response
	DlAck        bool                   `json:"dlAck" msgpack:"dlAck"`                             // EP acknowledges DL reception
	Duplicate    *bool                  `json:"duplicate,omitempty" msgpack:"duplicate,omitempty"` // Reused packet counter
}

// ULDataResponse represents the "ulDataRsp" message per SCACI §3.8.2
type ULDataResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// ULDataComplete represents the "ulDataCmp" message per SCACI §3.8.3
type ULDataComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// §3.9 UL Data Transmit Operation (Optional)
// ============================================================================

// ULDataTransmit is an alias to the canonical MIOTY type per SCACI §3.9.1
type ULDataTransmit = mioty.ULDataTransmit

// ULDataTransmitResponse is an alias to the canonical MIOTY type per SCACI §3.9.2
type ULDataTransmitResponse = mioty.ULDataTransmitResponse

// ULDataTransmitComplete is an alias to the canonical MIOTY type per SCACI §3.9.3
type ULDataTransmitComplete = mioty.ULDataTransmitComplete

// ============================================================================
// §3.10 DL Data Queue Operation
// ============================================================================

// DLDataQueue is an alias to the canonical MIOTY type per SCACI §3.10.1
type DLDataQueue = mioty.DLDataQueue

// DLDataQueueResponse is an alias to the canonical MIOTY type per SCACI §3.10.2
type DLDataQueueResponse = mioty.DLDataQueueResponse

// DLDataQueueComplete is an alias to the canonical MIOTY type per SCACI §3.10.3
type DLDataQueueComplete = mioty.DLDataQueueComplete

// ============================================================================
// §3.11 DL Data Revoke Operation
// ============================================================================

// DLDataRevoke represents the "dlDataRev" message per SCACI §3.11.1
type DLDataRevoke struct {
	BaseMessage
	EpEui     uint64 `json:"epEui" msgpack:"epEui"`         // End Point EUI64
	PacketCnt uint32 `json:"packetCnt" msgpack:"packetCnt"` // End Point packet counter of the scheduled data
}

// DLDataRevokeResponse represents the "dlDataRevRsp" message per SCACI §3.11.2
type DLDataRevokeResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// DLDataRevokeComplete represents the "dlDataRevCmp" message per SCACI §3.11.3
type DLDataRevokeComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// §3.12 DL Data Result Operation
// ============================================================================

// DLDataResult represents the "dlDataRes" message per SCACI §3.12.1
type DLDataResult struct {
	BaseMessage
	EpEui     uint64  `json:"epEui" msgpack:"epEui"`                             // End Point EUI64
	QueID     uint64  `json:"queId" msgpack:"queId"`                             // Queue ID
	Result    string  `json:"result" msgpack:"result"`                           // "sent", "expired", "invalid"
	TxTime    *int64  `json:"txTime,omitempty" msgpack:"txTime,omitempty"`       // Unix UTC center of first subpacket (ns)
	PacketCnt *uint32 `json:"packetCnt,omitempty" msgpack:"packetCnt,omitempty"` // EP packet counter (if sent)
	BsEui     *uint64 `json:"bsEui,omitempty" msgpack:"bsEui,omitempty"`         // Base Station EUI64 - only present if result=="sent" per SCACI §3.12.1
}

// DLDataResultResponse represents the "txDataResRsp" message per SCACI §3.12.2
type DLDataResultResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// DLDataResultComplete represents the "txDataResCmp" message per SCACI §3.12.3
type DLDataResultComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// §3.13 EP Status Operation
// ============================================================================

// EPStatus represents the "epStat" message per SCACI §3.13.1
type EPStatus struct {
	BaseMessage
	EpEui      uint64            `json:"epEui" msgpack:"epEui"`                               // End Point EUI64 (required)
	EpStatus   string            `json:"epStatus" msgpack:"epStatus"`                         // "attached" or "detached" (required)
	AttachCnt  *uint32           `json:"attachCnt,omitempty" msgpack:"attachCnt,omitempty"`   // EP attachment counter (OTA attach only)
	Nonce      *mioty.Numeric4   `json:"nonce,omitempty" msgpack:"nonce,omitempty"`           // 4-byte nonce (OTA attach only)
	Sign       *mioty.Numeric4   `json:"sign,omitempty" msgpack:"sign,omitempty"`             // 4-byte signature (OTA attach/detach only)
	Snr        *float64          `json:"snr,omitempty" msgpack:"snr,omitempty"`               // Reception SNR in dB (OTA attach/detach only)
	Rssi       *float64          `json:"rssi,omitempty" msgpack:"rssi,omitempty"`             // Reception RSSI in dBm (OTA attach/detach only)
	EqSnr      *float64          `json:"eqSnr,omitempty" msgpack:"eqSnr,omitempty"`           // AWGN equivalent SNR in dB (OTA attach/detach, optional)
	Subpackets *mioty.Subpackets `json:"subpackets,omitempty" msgpack:"subpackets,omitempty"` // Per-subpacket reception info (OTA attach/detach, optional)
}

// EPStatusResponse represents the "epStatRsp" message per SCACI §3.13.2
type EPStatusResponse struct {
	BaseMessage
	// No additional fields beyond core
}

// EPStatusComplete represents the "epStatCmp" message per SCACI §3.13.3
type EPStatusComplete struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// §3.14 Error Messages
// ============================================================================

// Error represents the "error" message per SCACI §3.14.1
// Wire format contains only: command, opId, code, message (per spec)
type Error struct {
	BaseMessage
	Code       int    `json:"code" msgpack:"code"`       // POSIX error number
	Message    string `json:"message" msgpack:"message"` // Error message
	ErrorToken string `json:"-" msgpack:"-"`             // Internal catalog token (not on wire per SCACI spec)
}

// ErrorAck represents the "errorAck" message per SCACI §3.14.2
type ErrorAck struct {
	BaseMessage
	// No additional fields beyond core
}

// ============================================================================
// Command Type Constants per SCACI v1.0.0
// ============================================================================

const (
	// CmdConnect is the SCACI connect command per §3.3
	CmdConnect = "con"
	// CmdConnectResponse is the connect response per §3.3
	CmdConnectResponse = "conRsp"
	// CmdConnectComplete is the connect complete per §3.3
	CmdConnectComplete = "conCmp"

	// CmdPing is the SCACI ping command per §3.4
	CmdPing = "ping"
	// CmdPingResponse is the ping response per §3.4
	CmdPingResponse = "pingRsp"
	// CmdPingComplete is the ping complete per §3.4
	CmdPingComplete = "pingCmp"

	// CmdStatus is the SCACI status command per §3.5
	CmdStatus = "status"
	// CmdStatusResponse is the status response per §3.5
	CmdStatusResponse = "statusRsp"
	// CmdStatusComplete is the status complete per §3.5
	CmdStatusComplete = "statusCmp"

	// CmdRegister is the SCACI register command per §3.6
	CmdRegister = "reg"
	// CmdRegisterResponse is the register response per §3.6
	CmdRegisterResponse = "regRsp"
	// CmdRegisterComplete is the register complete per §3.6
	CmdRegisterComplete = "regCmp"

	// CmdDeregister is the SCACI deregister command per §3.7
	CmdDeregister = "dereg"
	// CmdDeregisterResponse is the deregister response per §3.7
	CmdDeregisterResponse = "deregRsp"
	// CmdDeregisterComplete is the deregister complete per §3.7
	CmdDeregisterComplete = "deregCmp"

	// CmdULData is the SCACI uplink data command per §3.8
	CmdULData = "ulData"
	// CmdULDataResponse is the uplink data response per §3.8
	CmdULDataResponse = "ulDataRsp"
	// CmdULDataComplete is the uplink data complete per §3.8
	CmdULDataComplete = "ulDataCmp"

	// CmdULDataTransmit is the uplink data transmit command per §3.9
	CmdULDataTransmit = "ulDataTx"
	// CmdULDataTransmitResponse is the uplink data transmit response per §3.9
	CmdULDataTransmitResponse = "ulDataTxRsp"
	// CmdULDataTransmitComplete is the uplink data transmit complete per §3.9
	CmdULDataTransmitComplete = "ulDataTxCmp"

	// CmdDLDataQueue is the SCACI downlink data queue command per §3.10
	CmdDLDataQueue = "dlDataQue"
	// CmdDLDataQueueResponse is the downlink data queue response per §3.10
	CmdDLDataQueueResponse = "dlDataQueRsp"
	// CmdDLDataQueueComplete is the downlink data queue complete per §3.10
	CmdDLDataQueueComplete = "dlDataQueCmp"

	// CmdDLDataRevoke is the downlink data revoke command per §3.11
	CmdDLDataRevoke = "dlDataRev"
	// CmdDLDataRevokeResponse is the downlink data revoke response per §3.11
	CmdDLDataRevokeResponse = "dlDataRevRsp"
	// CmdDLDataRevokeComplete is the downlink data revoke complete per §3.11
	CmdDLDataRevokeComplete = "dlDataRevCmp"

	// CmdDLDataResult is the downlink data result command per §3.12.1 (shared with BSSCI §3.14)
	CmdDLDataResult = mioty.CmdDLDataResult
	// CmdDLDataResultResponse is the downlink data result response per §3.12.2 (SCACI-specific)
	CmdDLDataResultResponse = mioty.CmdDLDataResultResponseSCACI
	// CmdDLDataResultComplete is the downlink data result complete per §3.12.3 (SCACI-specific)
	CmdDLDataResultComplete = mioty.CmdDLDataResultCompleteSCACI

	// CmdEPStatus is the endpoint status command per §3.13
	CmdEPStatus = "epStat"
	// CmdEPStatusResponse is the endpoint status response per §3.13
	CmdEPStatusResponse = "epStatRsp"
	// CmdEPStatusComplete is the endpoint status complete per §3.13
	CmdEPStatusComplete = "epStatCmp"

	// CmdError is the SCACI error message per §3.14
	CmdError = "error"
	// CmdErrorAck is the error acknowledgment per §3.14
	CmdErrorAck = "errorAck"
)

// ============================================================================
// Internal Result Types for gRPC/Internal Calls
// ============================================================================

// DLDataQueueResult holds the result of a dlDataQue operation.
// Used by QueueDownlinkInternal to return results without socket I/O.
type DLDataQueueResult struct {
	QueID  uint64 // Queue ID assigned to the downlink
	BsEui  uint64 // Base Station EUI that will transmit
	OpID   int64  // Operation ID assigned
	Status string // Final status (queued, failed, etc.)
}
