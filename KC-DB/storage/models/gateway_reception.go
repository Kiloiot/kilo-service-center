package models

import (
	"time"
)

// BaseStationReception represents a single basestation's reception of a message
type BaseStationReception struct {
	ID            string `json:"id"`
	MessageID     string `json:"message_id"`
	BaseStationID string `json:"basestation_id"`
	TenantID      string `json:"tenant_id"`

	// Reception metadata
	RSSI      float32  `json:"rssi"`      // Received Signal Strength Indicator
	SNR       float32  `json:"snr"`       // Signal-to-Noise Ratio
	EqSNR     *float32 `json:"eq_snr"`    // Equivalent SNR for MIOTY
	Frequency int64    `json:"frequency"` // Frequency in Hz
	Channel   *int32   `json:"channel"`   // Channel number

	// Timing information
	ReceivedAt  time.Time `json:"received_at"`
	TimeOnAirMs *int32    `json:"time_on_air_ms"`

	// Gateway location at time of reception
	BaseStationLatitude  *float64 `json:"basestation_latitude"`
	BaseStationLongitude *float64 `json:"basestation_longitude"`
	BaseStationAltitude  *float32 `json:"basestation_altitude"`

	// Additional metadata
	AntennaID     int32  `json:"antenna_id"`
	FineTimestamp *int64 `json:"fine_timestamp"` // Nanoseconds
	Context       []byte `json:"context"`        // Gateway-specific data

	CreatedAt time.Time `json:"created_at"`
}

// BaseStationReceptionStats represents aggregated reception statistics
type BaseStationReceptionStats struct {
	BaseStationID   string    `json:"basestation_id"`
	Since           time.Time `json:"since"`
	TotalReceptions int64     `json:"total_receptions"`
	UniqueMessages  int64     `json:"unique_messages"`
	AvgRSSI         float64   `json:"avg_rssi"`
	MinRSSI         float64   `json:"min_rssi"`
	MaxRSSI         float64   `json:"max_rssi"`
	AvgSNR          float64   `json:"avg_snr"`
	MinSNR          float64   `json:"min_snr"`
	MaxSNR          float64   `json:"max_snr"`
}
