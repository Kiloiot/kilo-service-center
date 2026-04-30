package models

import (
	"encoding/json"
	"time"
)

// GatewaySession represents a gateway connection session
type GatewaySession struct {
	ID        int64 `db:"id" json:"id"`
	GatewayID int64 `db:"gateway_id" json:"gateway_id"`
	TenantID  int64 `db:"tenant_id" json:"tenant_id"`

	// Session identifiers
	SessionID    string  `db:"session_id" json:"session_id"`
	ConnectionID *string `db:"connection_id" json:"connection_id,omitempty"`

	// Connection details
	RemoteAddr      *string `db:"remote_addr" json:"remote_addr,omitempty"`
	ProtocolVersion *string `db:"protocol_version" json:"protocol_version,omitempty"`

	// Session state
	Status string `db:"status" json:"status"` // active, disconnected, terminated

	// Timing
	StartedAt  time.Time  `db:"started_at" json:"started_at"`
	LastPingAt *time.Time `db:"last_ping_at" json:"last_ping_at,omitempty"`
	EndedAt    *time.Time `db:"ended_at" json:"ended_at,omitempty"`

	// Statistics
	MessagesReceived int64 `db:"messages_received" json:"messages_received"`
	MessagesSent     int64 `db:"messages_sent" json:"messages_sent"`
	BytesReceived    int64 `db:"bytes_received" json:"bytes_received"`
	BytesSent        int64 `db:"bytes_sent" json:"bytes_sent"`

	// Metadata
	Metadata  json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

// GatewaySessionStats represents gateway session statistics
type GatewaySessionStats struct {
	GatewayID      int64         `json:"gateway_id"`
	TotalSessions  int64         `json:"total_sessions"`
	ActiveSessions int64         `json:"active_sessions"`
	AvgSessionTime time.Duration `json:"avg_session_time"`
	TotalUplinks   int64         `json:"total_uplinks"`
	TotalDownlinks int64         `json:"total_downlinks"`
	TotalErrors    int64         `json:"total_errors"`
}
