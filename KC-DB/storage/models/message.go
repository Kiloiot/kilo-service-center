package models

import (
	"encoding/json"
	"time"
)

// Message represents an uplink message from an End Point
type Message struct {
	ID       string `json:"id"`
	DeviceID string `json:"device_id"`
	TenantID string `json:"tenant_id"`

	// Message identifiers
	MessageID string `json:"message_id"` // Unique message ID
	EPEUI     string `json:"epEui"`      // End Point EUI
	BSEUI     string `json:"bsEui"`      // Base Station EUI

	// Message content
	Payload     []byte          `json:"payload"`      // Raw payload
	PayloadHex  string          `json:"payload_hex"`  // Hex representation
	PayloadJSON json.RawMessage `json:"payload_json"` // Decoded JSON payload

	// Radio parameters
	RSSI            float64 `json:"rssi"`             // Received Signal Strength
	SNR             float64 `json:"snr"`              // Signal-to-Noise Ratio
	EqSNR           float64 `json:"eqSnr"`            // Equivalent SNR
	Frequency       int64   `json:"frequency"`        // Frequency in Hz
	SpreadingFactor int     `json:"spreading_factor"` // SF7-SF12
	Bandwidth       int64   `json:"bandwidth"`        // Bandwidth in Hz
	CodeRate        string  `json:"code_rate"`        // Code rate (4/5, 4/6, etc)

	// MIOTY-specific fields
	FrameCount  uint32 `json:"frameCnt"`    // Frame counter
	MessageType string `json:"messageType"` // uplink, downlink_ack
	CryptoMode  string `json:"cryptoMode"`  // Encryption mode
	DlOpen      bool   `json:"dlOpen"`      // Downlink open
	ResExp      bool   `json:"resExp"`      // Response expected
	DlAck       bool   `json:"dlAck"`       // Downlink acknowledgment

	// Processing metadata
	ReceivedAt     time.Time  `json:"received_at"`
	ProcessedAt    *time.Time `json:"processed_at"`
	Confirmed      bool       `json:"confirmed"`
	Retransmission bool       `json:"retransmission"`

	// Additional metadata
	Tags      []string        `json:"tags"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// MessageStats represents message statistics
type MessageStats struct {
	DeviceID          string  `json:"device_id"`
	TotalMessages     int64   `json:"total_messages"`
	UplinkMessages    int64   `json:"uplink_messages"`
	DownlinkAcks      int64   `json:"downlink_acks"`
	ConfirmedMessages int64   `json:"confirmed_messages"`
	Retransmissions   int64   `json:"retransmissions"`
	AvgRSSI           float64 `json:"avg_rssi"`
	AvgSNR            float64 `json:"avg_snr"`
	MessageRate       float64 `json:"message_rate"` // messages per hour
}
