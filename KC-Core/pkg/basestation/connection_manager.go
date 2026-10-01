// Package basestation keeps the persisted connection state of base stations.
package basestation

import (
	"context"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// ConnectionType represents the type of connection protocol
type ConnectionType string

// Connection type constants define the protocol used for base station communication.
const (
	ConnectionTypeBSSCI ConnectionType = "bssci"
)

// ConnectionStatus represents the current connection state
type ConnectionStatus struct {
	IsOnline       bool           `json:"is_online"`
	LastSeen       time.Time      `json:"last_seen"`
	ConnectionType ConnectionType `json:"connection_type"`
	SessionID      string         `json:"session_id,omitempty"`
	// SessionStartedAt is when SessionID became the active session (its
	// completed connect handshake); zero on liveness updates.
	SessionStartedAt time.Time              `json:"session_started_at,omitempty"`
	UptimeSeconds    int64                  `json:"uptime_seconds,omitempty"`
	ConnectionDetail map[string]interface{} `json:"connection_details,omitempty"`
	LastError        string                 `json:"last_error,omitempty"`
}

// Location represents geographic coordinates
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude,omitempty"`
}

// ConnectionManager keeps the persisted connection state of base stations
// whose sessions the BSSCI server owns: liveness, online/offline transitions
// and their audit events.
type ConnectionManager struct {
	store         Store
	eventRecorder EventRecorder
}

// Column keys of the connection status updates.
const (
	fieldKeyIsOnline         = "is_online"
	fieldKeyConnectionType   = "connection_type"
	fieldKeyLastSeenAt       = "last_seen_at"
	fieldKeySessionUUID      = "session_uuid"
	fieldKeySessionStartedAt = "session_started_at"
)

// Store interface for database operations
type Store interface {
	GetBaseStation(ctx context.Context, eui [8]byte) (*BaseStation, error)
	GetBaseStationGlobal(ctx context.Context, eui [8]byte) (*BaseStation, error)
	UpdateConnectionStatus(ctx context.Context, eui [8]byte, status *ConnectionStatus) error
	// DisconnectIfCurrent marks the base station offline only while the
	// stored session identity still matches the disconnecting connection.
	// Returns false without error when a newer connection owns the row.
	DisconnectIfCurrent(ctx context.Context, eui [8]byte, connectionID string, lastSeen time.Time) (bool, error)
}

// EventRecorder records a base station event that occurred at occurredAt,
// the moment its caller observed the exchange, not when recording reaches the store.
type EventRecorder interface {
	RecordEvent(ctx context.Context, eui [8]byte, eventType string, occurredAt time.Time, data map[string]interface{}) error
}

// BaseStation represents a Base Station entity
type BaseStation struct {
	ID               int64
	TenantID         int64
	EUI              [8]byte
	Name             string
	Description      string
	ConnectionType   ConnectionType
	ServiceCenterURL string
	Vendor           string
	Model            string
	SoftwareVersion  string
	Location         *Location
	LastSeenAt       *time.Time
	Status           *ConnectionStatus
	Metadata         map[string]interface{}
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(store Store, eventRecorder EventRecorder) *ConnectionManager {
	return &ConnectionManager{
		store:         store,
		eventRecorder: eventRecorder,
	}
}

// GetBaseStationGlobal retrieves a Base Station by EUI across all tenants.
// Used during BSSCI connect handshake before tenant is resolved.
func (cm *ConnectionManager) GetBaseStationGlobal(ctx context.Context, eui [8]byte) (*BaseStation, error) {
	return cm.store.GetBaseStationGlobal(ctx, eui)
}

// UpdateLastSeen updates the last seen timestamp for a Base Station
func (cm *ConnectionManager) UpdateLastSeen(ctx context.Context, eui [8]byte) error {
	// First get the base station to determine its connection type
	bs, err := cm.store.GetBaseStation(ctx, eui)
	if err != nil {
		return fmt.Errorf(errFmtGetBaseStation, err)
	}

	status := &ConnectionStatus{
		IsOnline:       true,
		LastSeen:       time.Now(),
		ConnectionType: bs.ConnectionType,
	}

	if err := cm.store.UpdateConnectionStatus(ctx, eui, status); err != nil {
		return fmt.Errorf(errFmtUpdateLastSeen, err)
	}

	// Heartbeat events removed - they flood the system_events table with no audit value.
	// Activity is tracked via base_stations.last_seen_at, and operational events
	// (basestation_online, basestation_offline, connection_error) provide audit trail.
	return nil
}

// UpdateConnectionStatus updates the connection status for a Base Station
func (cm *ConnectionManager) UpdateConnectionStatus(ctx context.Context, eui [8]byte, status *ConnectionStatus) error {
	if err := cm.store.UpdateConnectionStatus(ctx, eui, status); err != nil {
		return fmt.Errorf(errFmtUpdateConnectionStatus, err)
	}

	eventData := map[string]interface{}{
		models.EventDetailKeyIsOnline:       status.IsOnline,
		models.EventDetailKeyConnectionType: status.ConnectionType,
		models.EventDetailKeySessionID:      status.SessionID,
	}

	eventType := models.EventTypeBaseStationOffline
	if status.IsOnline {
		eventType = models.EventTypeBaseStationOnline
	}

	return cm.eventRecorder.RecordEvent(ctx, eui, eventType, status.LastSeen, eventData)
}

// DisconnectBaseStationIfCurrent marks the base station offline only while
// the stored connection still belongs to the disconnecting session. When a
// reconnect has already replaced the connection, the newer session keeps the
// base station online and no update happens (not an error).
func (cm *ConnectionManager) DisconnectBaseStationIfCurrent(ctx context.Context, eui [8]byte, connectionID string) error {
	disconnectedAt := time.Now()
	acted, err := cm.store.DisconnectIfCurrent(ctx, eui, connectionID, disconnectedAt)
	if err != nil {
		return fmt.Errorf(errFmtDisconnectBaseStation, err)
	}
	if !acted {
		return nil
	}
	eventData := map[string]interface{}{
		models.EventDetailKeyIsOnline:  false,
		models.EventDetailKeySessionID: connectionID,
	}
	return cm.eventRecorder.RecordEvent(ctx, eui, models.EventTypeBaseStationOffline, disconnectedAt, eventData)
}
