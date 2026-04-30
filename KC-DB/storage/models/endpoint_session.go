package models

import (
	"encoding/json"
	"time"
)

// EndPointSession represents an endpoint attachment session
type EndPointSession struct {
	ID         int64 `db:"id" json:"id"`
	EndPointID int64 `db:"endpoint_id" json:"endpoint_id"`
	TenantID   int64 `db:"tenant_id" json:"tenant_id"`

	// Session identifiers
	SessionID string `db:"session_id" json:"session_id"`
	AttachCnt int32  `db:"attach_cnt" json:"attach_cnt"`

	// Session key
	SessionKey []byte `db:"session_key" json:"-"` // Hidden in JSON for security

	// Session state
	Status string `db:"status" json:"status"` // active, expired, terminated

	// Timing
	StartedAt      time.Time  `db:"started_at" json:"started_at"`
	LastActivityAt time.Time  `db:"last_activity_at" json:"last_activity_at"`
	EndedAt        *time.Time `db:"ended_at" json:"ended_at"`

	// Session parameters
	ShAddr    *int32 `db:"sh_addr" json:"shAddr"`            // INTEGER (0-65535) per migration 087
	PacketCnt int32  `db:"last_packet_cnt" json:"packetCnt"` // Last known packet counter

	// Uplink configuration
	UplinkMode string `db:"uplink_mode" json:"uplink_mode"` // standard, low_latency, repeated

	// Downlink configuration
	DlOpen     bool `db:"dl_open" json:"dlOpen"`        // Downlink open
	ResExp     bool `db:"res_exp" json:"resExp"`        // Response expected
	DlAck      bool `db:"dl_ack" json:"dlAck"`          // Downlink acknowledgment
	Repetition bool `db:"repetition" json:"repetition"` // True if DL repetition enabled

	// Gateway association
	PrimaryBaseStationID *int64 `db:"primary_basestation_id" json:"primary_basestation_id"`

	// Statistics
	UplinkCount   int32 `db:"uplink_count" json:"uplink_count"`
	DownlinkCount int32 `db:"downlink_count" json:"downlink_count"`

	// Metadata
	Metadata  json.RawMessage `db:"metadata" json:"metadata"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt time.Time       `db:"updated_at" json:"updated_at"`
}

// EndPointSessionStats represents session statistics
type EndPointSessionStats struct {
	EndPointID         string        `json:"endpoint_id"`
	TotalSessions      int64         `json:"total_sessions"`
	ActiveSessions     int64         `json:"active_sessions"`
	AvgSessionTime     time.Duration `json:"avg_session_time"`
	AvgSessionDuration time.Duration `json:"avg_session_duration"`
	TotalUplinks       int64         `json:"total_uplinks"`
	TotalDownlinks     int64         `json:"total_downlinks"`
}
