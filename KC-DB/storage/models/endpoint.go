// Package models defines the database models for KiloCenter
package models

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EUI represents an 8-byte Extended Unique Identifier
type EUI [8]byte

// String returns the EUI as a hex string
func (e EUI) String() string {
	return hex.EncodeToString(e[:])
}

// ToUint64 converts the EUI to a uint64 integer
func (e EUI) ToUint64() uint64 {
	var result uint64
	for i := 0; i < 8; i++ {
		result = (result << 8) | uint64(e[i])
	}
	return result
}

// MarshalText implements encoding.TextMarshaler
func (e EUI) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (e *EUI) UnmarshalText(text []byte) error {
	b, err := hex.DecodeString(string(text))
	if err != nil {
		return err
	}
	if len(b) != 8 {
		return fmt.Errorf("EUI must be 8 bytes, got %d", len(b))
	}
	copy(e[:], b)
	return nil
}

// Value implements driver.Valuer for database storage
func (e EUI) Value() (driver.Value, error) {
	return e[:], nil
}

// Scan implements sql.Scanner for database retrieval
func (e *EUI) Scan(src interface{}) error {
	switch v := src.(type) {
	case []byte:
		if len(v) != 8 {
			return fmt.Errorf("EUI must be 8 bytes, got %d", len(v))
		}
		copy(e[:], v)
		return nil
	case nil:
		return nil
	default:
		return fmt.Errorf("cannot scan %T into EUI", src)
	}
}

// EUIFromString creates an EUI from a hex string
func EUIFromString(s string) EUI {
	var eui EUI
	// Remove any hyphens or colons from the string
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, ":", "")

	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 8 {
		// Return zero EUI on error
		return eui
	}
	copy(eui[:], b)
	return eui
}

// EndPoint represents a MIOTY End Point (EP) per BSSCI v1.0.0 specification
type EndPoint struct {
	ID            int64  `db:"id" json:"id"`
	EUI           EUI    `db:"ep_eui" json:"epEui"` // End Point EUI64 per MIOTY spec
	Name          string `db:"name" json:"name"`
	Description   string `db:"description" json:"description,omitempty"`
	TenantID      int64  `db:"tenant_id" json:"tenantId"`            // Current serving tenant (for roaming)
	OwnerTenantID int64  `db:"owner_tenant_id" json:"ownerTenantId"` // Original owner tenant (roaming)

	// MIOTY Core fields per BSSCI v1.0.0
	NwkSnKey  []byte  `db:"nwk_key" json:"-"`                  // Network Session Key per BSSCI spec (hidden in JSON)
	AppKey    []byte  `db:"app_key" json:"-"`                  // Application Key (hidden in JSON)
	ShAddr    *uint16 `db:"sh_addr" json:"shAddr,omitempty"`   // Short Address (uint16, 0-65535) per SCACI §3.6.1
	Bidi      bool    `db:"bidi" json:"bidi"`                  // True if End Point is bidirectional per BSSCI Section 5.8.1
	PreAttach bool    `db:"pre_attach" json:"preAttach"`       // Pre-attach End Point at Base Stations per SCACI §3.6.1
	TypeEUI   *EUI    `db:"type_eui" json:"typeEui,omitempty"` // Type EUI for device classification

	// Endpoint Metadata (Migration 023)
	Manufacturer  *string `db:"manufacturer" json:"manufacturer,omitempty"`    // Device manufacturer (nullable)
	Model         *string `db:"model" json:"model,omitempty"`                  // Device model identifier (nullable)
	CarrierOffset int     `db:"carrier_offset" json:"carrierOffset,omitempty"` // Carrier offset in Hz (default 0)

	// Blueprint device model association (Migration 000106)
	// When set, typeEui MUST come from the model's default blueprint (TypeEUI precedence rule)
	DeviceModelID   *uuid.UUID      `db:"device_model_id" json:"deviceModelId,omitempty"`    // UUID reference to device_models.id
	CalibrationData json.RawMessage `db:"calibration_data" json:"calibrationData,omitempty"` // Per-device calibration overrides for blueprint decoding

	// Propagation Status (Migration 022)
	Propagated       bool       `db:"propagated" json:"propagated"`                // Propagation status (default false)
	PropagatedAt     *time.Time `db:"propagated_at" json:"propagatedAt,omitempty"` // Last propagation timestamp (nullable)
	PropagationCount int32      `db:"propagation_count" json:"propagationCount"`   // Number of propagations (default 0)

	// MIOTY Configuration fields per BSSCI v1.0.0 (Attach and Attach Propagate operations)
	DualChan    bool `db:"dual_chan" json:"dualChan"`        // True if End Point uses dual channel mode
	Repetition  bool `db:"repetition" json:"repetition"`     // True if End Point uses DL repetition
	WideCarrOff bool `db:"wide_carr_off" json:"wideCarrOff"` // True if End Point uses wide carrier offset
	LongBlkDist bool `db:"long_blk_dist" json:"longBlkDist"` // True if End Point uses long DL interblock distance

	// MIOTY Attach operation fields per BSSCI v1.0.0 Section 5.6.1
	AttachCnt            *uint32 `db:"attach_cnt" json:"attachCnt,omitempty"`                         // Attachment counter (uint32, 0-4294967295) per SCACI §3.6.1
	Nonce                []byte  `db:"nonce" json:"nonce,omitempty"`                                  // 4 Byte End Point nonce
	Sign                 []byte  `db:"sign" json:"sign,omitempty"`                                    // 4 Byte End Point signature
	PresharedKey         []byte  `db:"preshared_key" json:"-"`                                        // MIOTY preshared key for detach CMAC validation (hidden in JSON, optional per BSSCI §5.7)
	LastAttachRxTime     *int64  `db:"last_attach_rx_time" json:"lastAttachRxTime,omitempty"`         // Unix UTC time of last attach reception (ns)
	LastAttachRxDuration *int64  `db:"last_attach_rx_duration" json:"lastAttachRxDuration,omitempty"` // Duration of last attach reception (ns)

	// MIOTY Radio metrics fields per BSSCI v1.0.0
	LastSNR     *float64 `db:"last_snr" json:"snr,omitempty"`         // Reception signal to noise ratio (dB)
	LastRSSI    *float64 `db:"last_rssi" json:"rssi,omitempty"`       // Reception signal strength (dBm)
	LastEqSNR   *float64 `db:"last_eq_snr" json:"eqSnr,omitempty"`    // AWGN equivalent reception SNR (dB)
	LastProfile *string  `db:"last_profile" json:"profile,omitempty"` // MIOTY profile used for reception (e.g., "eu1")

	// Attach subpacket telemetry (BSSCI §5.6.1, optional)
	LastAttachSubpackets *string `db:"last_attach_subpackets" json:"lastAttachSubpackets,omitempty"`

	// MIOTY Detach telemetry fields per BSSCI v1.0.0 Section 5.7.1 (Finding 2)
	LastAttachedBsEui   *int64  `db:"last_attached_bs_eui" json:"lastAttachedBsEui,omitempty"`     // Last BS EUI that handled attach/detach propagate
	LastPropagateTime   *int64  `db:"last_propagate_time" json:"lastPropagateTime,omitempty"`      // Unix UTC nanoseconds of last propagate operation
	LastDetachTime      *int64  `db:"last_detach_time" json:"lastDetachTime,omitempty"`            // Unix UTC nanoseconds of last detach reception
	LastDetachSign      []byte  `db:"last_detach_sign" json:"lastDetachSign,omitempty"`            // 4-byte endpoint signature from detach message
	LastDetachPacketCnt *int64  `db:"last_detach_packet_cnt" json:"lastDetachPacketCnt,omitempty"` // Endpoint packet counter from detach message
	PropagateStatus     *string `db:"propagate_status" json:"propagateStatus,omitempty"`           // Propagation state: pending, completed, failed

	// MIOTY Deduplication field per BSSCI v1.0.0 Section 5.8.1
	LastPacketCnt uint32 `db:"last_packet_cnt" json:"lastPacketCnt"` // Last packet counter (uint32, 0-4294967295) per SCACI §3.6.1

	// MIOTY UL Data message fields per BSSCI v1.0.0 Section 5.10.1 (THE PAYLOAD DATA!)
	LastUserData []byte  `db:"last_user_data" json:"userData,omitempty"` // Last received user data (payload)
	LastFormatID *int32  `db:"last_format_id" json:"format,omitempty"`   // Last received user data format identifier (8 bit)
	LastMode     *string `db:"last_mode" json:"mode,omitempty"`          // Last known MIOTY mode ("ulp", "ulp-rep", "ulp-ll")

	// MIOTY UL Data reception fields per BSSCI v1.0.0 Section 5.10.1
	LastRxTime     *int64 `db:"last_rx_time" json:"rxTime,omitempty"`         // Unix UTC time of last UL Data reception (ns)
	LastRxDuration *int64 `db:"last_rx_duration" json:"rxDuration,omitempty"` // Duration of last UL Data reception (ns)
	PacketCnt      uint32 `db:"packet_cnt" json:"packetCnt"`                  // Packet counter (uint32, 0-4294967295) per SCACI §3.6.1

	// MIOTY Downlink control fields per BSSCI v1.0.0 Section 5.10.1
	LastDlOpen      bool `db:"last_dl_open" json:"dlOpen"`           // Last known downlink window state
	LastResponseExp bool `db:"last_response_exp" json:"responseExp"` // Last known response expectation state
	LastDlAck       bool `db:"last_dl_ack" json:"dlAck"`             // Last known DL acknowledgment state

	// Compatibility fields
	CryptoMode int16  `db:"crypto_mode" json:"cryptoMode"`
	EPClass    string `db:"endpoint_class" json:"epClass"` // 'Z' or 'A'

	// Status fields
	EpStatus     string     `db:"ep_status" json:"epStatus"` // Attach status: attached, detached, attaching (per migration 003)
	LastSeenAt   *time.Time `db:"last_seen_at" json:"lastSeenAt,omitempty"`
	FrameCount   int32      `db:"frame_count" json:"frameCount"`
	BatteryLevel *float32   `db:"battery_level" json:"batteryLevel,omitempty"`

	// Metadata
	Tags      map[string]string `db:"tags" json:"tags,omitempty"`
	CreatedAt time.Time         `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time         `db:"updated_at" json:"updatedAt"`
}

// DownlinkMessage represents a queued downlink message to an End Point
type DownlinkMessage struct {
	ID       int64 `db:"id" json:"id"`
	EPEUI    EUI   `db:"ep_eui" json:"epEui"` // End Point EUI (maps to ep_eui in DB)
	TenantID int64 `db:"tenant_id" json:"tenant_id"`

	// Message content
	Payload   []byte `db:"payload" json:"payload"`
	Port      int32  `db:"port" json:"port"`
	Confirmed bool   `db:"confirmed" json:"confirmed"`

	// Scheduling
	Priority   int32      `db:"priority" json:"priority"`
	ValidUntil *time.Time `db:"valid_until" json:"valid_until,omitempty"`
	EarliestAt time.Time  `db:"earliest_at" json:"earliest_at"`

	// Status tracking
	Status      string `db:"status" json:"status"` // pending, scheduled, transmitted, confirmed, failed, expired
	Attempts    int32  `db:"attempts" json:"attempts"`
	MaxAttempts int32  `db:"max_attempts" json:"max_attempts"`

	// Transmission results
	TransmittedAt  *time.Time `db:"transmitted_at" json:"transmitted_at,omitempty"`
	AcknowledgedAt *time.Time `db:"acknowledged_at" json:"acknowledged_at,omitempty"`
	GatewayEUI     *EUI       `db:"bs_eui" json:"bsEui,omitempty"`
	FailureReason  *string    `db:"failure_reason" json:"failure_reason,omitempty"`

	// Metadata
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// RoamingAgreement represents a roaming agreement with another network
type RoamingAgreement struct {
	ID               int64  `db:"id" json:"id"`
	TenantID         int64  `db:"tenant_id" json:"tenant_id"`
	PartnerNetworkID string `db:"partner_network_id" json:"partner_network_id"`
	PartnerName      string `db:"partner_name" json:"partner_name"`

	// Agreement details
	AgreementType string     `db:"agreement_type" json:"agreement_type"` // bilateral, unilateral_in, unilateral_out
	IsActive      bool       `db:"is_active" json:"is_active"`
	ValidFrom     time.Time  `db:"valid_from" json:"valid_from"`
	ValidUntil    *time.Time `db:"valid_until" json:"valid_until,omitempty"`

	// Technical details
	PartnerEndpoint string `db:"partner_endpoint" json:"partner_endpoint"`
	APIKeyHash      string `db:"api_key_hash" json:"-"` // Hidden in JSON

	// Metadata
	Metadata  map[string]interface{} `db:"metadata" json:"metadata,omitempty"`
	CreatedAt time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt time.Time              `db:"updated_at" json:"updated_at"`
}

// EndPointStats represents endpoint statistics
type EndPointStats struct {
	EndPointID     string     `json:"endpoint_id"`
	TotalUplinks   int64      `json:"total_uplinks"`
	TotalDownlinks int64      `json:"total_downlinks"`
	AvgRSSI        float64    `json:"avg_rssi"`
	AvgSNR         float64    `json:"avg_snr"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
}
