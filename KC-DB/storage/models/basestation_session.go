package models

import (
	"time"

	"github.com/google/uuid"
)

// BaseStationSessionStatus represents the status of a Base Station session
type BaseStationSessionStatus string

// BaseStationSessionStatus constants for session lifecycle states.
const (
	// SessionStatusActive indicates an active session.
	SessionStatusActive BaseStationSessionStatus = "active"
	// SessionStatusResumed indicates a resumed session.
	SessionStatusResumed BaseStationSessionStatus = "resumed"
	// SessionStatusTerminated indicates a terminated session.
	SessionStatusTerminated BaseStationSessionStatus = "terminated"
)

// BaseStationSession represents a MIOTY BSSCI session per Section 3.3
// This implements session management for connection resumption and operation ID tracking
//
//revive:disable:var-naming SnBsOpId/SnScOpId use lowercase 'd' per MIOTY BSSCI §3.2
type BaseStationSession struct {
	ID            int64 `json:"id" db:"id"`
	BaseStationID int64 `json:"basestation_id" db:"basestation_id"`
	TenantID      int64 `json:"tenant_id" db:"tenant_id"`

	// MIOTY Session identifiers per BSSCI 3.3.1
	SnBsUuid [16]byte `json:"sn_bs_uuid" db:"sn_bs_uuid"` // Base Station session UUID
	SnScUuid [16]byte `json:"sn_sc_uuid" db:"sn_sc_uuid"` // Service Center session UUID

	// Operation ID tracking for session resumption per BSSCI 3.2
	SnBsOpId int64 `json:"sn_bs_op_id" db:"sn_bs_op_id"` // Last known BS operation ID
	SnScOpId int64 `json:"sn_sc_op_id" db:"sn_sc_op_id"` // Last known SC operation ID

	// Connection state
	Status       BaseStationSessionStatus `json:"status" db:"status"`
	ConnectionId *string                  `json:"connection_id,omitempty" db:"connection_id"`
	RemoteAddr   *string                  `json:"remote_addr,omitempty" db:"remote_addr"`

	// Timing
	StartedAt  time.Time  `json:"started_at" db:"started_at"`
	LastPingAt *time.Time `json:"last_ping_at,omitempty" db:"last_ping_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty" db:"ended_at"`

	// Session parameters
	CanResume       bool     `json:"can_resume" db:"can_resume"`                       // Whether session can be resumed after disconnect
	Encoding        string   `json:"encoding" db:"encoding"`                           // Message encoding: "json" or "msgpack" (BSSCI Section 1)
	ProtocolVersion *string  `json:"protocol_version,omitempty" db:"protocol_version"` // Negotiated MIOTY protocol version (BSSCI §4-4.5)
	ConnectInfo     NullJSON `json:"connect_info,omitempty" db:"connect_info"`         // BSSCI §5.3 arbitrary key-value pairs from connect message

	// Organization isolation (migration 030)
	OrganizationID *uuid.UUID `json:"organization_id,omitempty" db:"organization_id"` // Kilo Cloud org UUID

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// BaseStationSessionFilter defines filter criteria for querying Base Station sessions
type BaseStationSessionFilter struct {
	TenantID      int64
	BaseStationID *int64
	Status        []BaseStationSessionStatus
	SnBsUuid      *[16]byte
	SnScUuid      *[16]byte
	CanResume     *bool
	ActiveOnly    bool // Only return active sessions
	Since         *time.Time
	Limit         int
	Offset        int
}

// BaseStationSessionCreateRequest represents the request to create a new session
type BaseStationSessionCreateRequest struct {
	BaseStationID   int64      `json:"basestation_id" validate:"required"`
	TenantID        int64      `json:"tenant_id" validate:"required"`
	SnBsUuid        [16]byte   `json:"sn_bs_uuid" validate:"required"`
	SnScUuid        [16]byte   `json:"sn_sc_uuid" validate:"required"`
	ConnectionId    *string    `json:"connection_id,omitempty"`
	RemoteAddr      *string    `json:"remote_addr,omitempty"`
	CanResume       bool       `json:"can_resume"`
	Encoding        string     `json:"encoding" validate:"oneof=json msgpack"`                 // Message encoding (default: msgpack)
	ProtocolVersion *string    `json:"protocol_version,omitempty" validate:"omitempty,semver"` // Negotiated MIOTY protocol version (BSSCI §4-4.5)
	ConnectInfo     NullJSON   `json:"connect_info,omitempty"`                                 // BSSCI §5.3 arbitrary key-value pairs from connect message
	OrganizationID  *uuid.UUID `json:"organization_id,omitempty"`                              // Kilo Cloud org UUID
}

// BaseStationSessionUpdateRequest represents session update operations
type BaseStationSessionUpdateRequest struct {
	SnBsOpId        *int64                    `json:"sn_bs_op_id,omitempty"`
	SnScOpId        *int64                    `json:"sn_sc_op_id,omitempty"`
	Status          *BaseStationSessionStatus `json:"status,omitempty"`
	LastPingAt      *time.Time                `json:"last_ping_at,omitempty"`
	EndedAt         *time.Time                `json:"ended_at,omitempty"`
	ConnectionId    *string                   `json:"connection_id,omitempty"`
	RemoteAddr      *string                   `json:"remote_addr,omitempty"`
	Encoding        *string                   `json:"encoding,omitempty" validate:"omitempty,oneof=json msgpack"` // Message encoding
	ProtocolVersion *string                   `json:"protocol_version,omitempty" validate:"omitempty,semver"`     // Negotiated MIOTY protocol version (BSSCI §4-4.5)
	OrganizationID  *uuid.UUID                `json:"organization_id,omitempty"`                                  // Kilo Cloud org UUID
}

// IsActive returns true if the session is currently active
func (s *BaseStationSession) IsActive() bool {
	return s.Status == SessionStatusActive || s.Status == SessionStatusResumed
}

// CanResumeSession returns true if this session can be resumed after disconnection
func (s *BaseStationSession) CanResumeSession() bool {
	return s.CanResume && (s.Status == SessionStatusActive || s.Status == SessionStatusResumed)
}

// GetNextScOpId returns the next Service Center operation ID for this session
func (s *BaseStationSession) GetNextScOpId() int64 {
	// Service Center uses negative, decrementing IDs per MIOTY spec
	s.SnScOpId--
	return s.SnScOpId
}

// UpdateBsOpId updates the Base Station operation ID if the new ID is higher
func (s *BaseStationSession) UpdateBsOpId(newOpId int64) bool {
	if newOpId > s.SnBsOpId {
		s.SnBsOpId = newOpId
		return true
	}
	return false
}

// GetSessionAge returns how long this session has been active
func (s *BaseStationSession) GetSessionAge() time.Duration {
	if s.EndedAt != nil {
		return s.EndedAt.Sub(s.StartedAt)
	}
	return time.Since(s.StartedAt)
}
