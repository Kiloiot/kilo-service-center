package models

import (
	"time"

	"github.com/google/uuid"
)

// SCACISessionStatus represents the status of a SCACI session
type SCACISessionStatus string

const (
	// SCACIStatusActive indicates an active SCACI session
	SCACIStatusActive SCACISessionStatus = "active"

	// SCACIStatusResumed indicates a resumed SCACI session
	SCACIStatusResumed SCACISessionStatus = "resumed"

	// SCACIStatusDisconnected indicates a disconnected session (may be resumable)
	SCACIStatusDisconnected SCACISessionStatus = "disconnected"

	// SCACIStatusTerminated indicates a terminated session (not resumable)
	SCACIStatusTerminated SCACISessionStatus = "terminated"
)

// SCACISession represents a SCACI (Service Center to Application Center) session
// Implements MIOTY SCACI v1.0.0 Section 1 session management
type SCACISession struct {
	ID       int64 `db:"id" json:"id"`
	TenantID int64 `db:"tenant_id" json:"tenant_id"`

	// Application Center identification
	AcEUI [8]byte `db:"ac_eui" json:"ac_eui"` // Application Center EUI64

	// Session identifiers (SCACI §1, lines 22-27)
	SnAcUUID [16]byte `db:"sn_ac_uuid" json:"sn_ac_uuid"` // AC session UUID
	SnScUUID [16]byte `db:"sn_sc_uuid" json:"sn_sc_uuid"` // SC session UUID

	// Operation ID tracking for resumption
	LastOpIDAc int64 `db:"last_op_id_ac" json:"last_op_id_ac"` // Last AC operation ID (positive)
	LastOpIDSc int64 `db:"last_op_id_sc" json:"last_op_id_sc"` // Last SC operation ID (negative)

	// Connection state
	Status string `db:"status" json:"status"` // active, resumed, disconnected, terminated

	// TLS certificate information
	CertificateFingerprint *string `db:"certificate_fingerprint" json:"certificate_fingerprint,omitempty"`
	ClientCertSubject      *string `db:"client_cert_subject" json:"client_cert_subject,omitempty"`

	// Network information
	RemoteAddr *string `db:"remote_addr" json:"remote_addr,omitempty"`

	// TLS evidence per SCACI §1 (migration 000082)
	TLSVersion  *string `db:"tls_version" json:"tls_version,omitempty"`
	CipherSuite *string `db:"cipher_suite" json:"cipher_suite,omitempty"`

	// Protocol version per SCACI §§2.1-2.3 (migration 000083)
	// Negotiated at connect; resume attempts must match this version
	NegotiatedVersion string `db:"negotiated_version" json:"negotiated_version"`

	// Timing
	ConnectedAt    time.Time  `db:"connected_at" json:"connected_at"`
	LastHeartbeat  *time.Time `db:"last_heartbeat" json:"last_heartbeat,omitempty"`
	DisconnectedAt *time.Time `db:"disconnected_at" json:"disconnected_at,omitempty"`

	// Session control
	CanResume bool `db:"can_resume" json:"can_resume"` // Whether session can be resumed

	// Extensibility
	Metadata map[string]interface{} `db:"metadata" json:"metadata,omitempty"`

	// Organization isolation (migration 030)
	OrganizationID *uuid.UUID `db:"organization_id" json:"organization_id,omitempty"` // Kilo Cloud org UUID

	// Audit
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// SCACISessionCreateRequest represents a request to create a new SCACI session
type SCACISessionCreateRequest struct {
	TenantID               int64                  `json:"tenant_id" validate:"required"`
	AcEUI                  [8]byte                `json:"ac_eui" validate:"required"`
	SnAcUUID               [16]byte               `json:"sn_ac_uuid" validate:"required"`
	SnScUUID               [16]byte               `json:"sn_sc_uuid" validate:"required"`
	CertificateFingerprint *string                `json:"certificate_fingerprint,omitempty"`
	ClientCertSubject      *string                `json:"client_cert_subject,omitempty"`
	RemoteAddr             *string                `json:"remote_addr,omitempty"`
	TLSVersion             *string                `json:"tls_version,omitempty"`  // SCACI §1 TLS evidence
	CipherSuite            *string                `json:"cipher_suite,omitempty"` // SCACI §1 TLS evidence
	NegotiatedVersion      string                 `json:"negotiated_version"`     // SCACI §§2.1-2.3 (defaults to ProtocolVersionString if empty)
	CanResume              bool                   `json:"can_resume"`
	Metadata               map[string]interface{} `json:"metadata,omitempty"`
	OrganizationID         *uuid.UUID             `json:"organization_id,omitempty"` // Kilo Cloud org UUID
}

// SCACISessionUpdateRequest represents a request to update an existing SCACI session
type SCACISessionUpdateRequest struct {
	Status         *string                `json:"status,omitempty"`
	LastOpIDAc     *int64                 `json:"last_op_id_ac,omitempty"`
	LastOpIDSc     *int64                 `json:"last_op_id_sc,omitempty"`
	LastHeartbeat  *time.Time             `json:"last_heartbeat,omitempty"`
	DisconnectedAt *time.Time             `json:"disconnected_at,omitempty"`
	CanResume      *bool                  `json:"can_resume,omitempty"`
	TLSVersion     *string                `json:"tls_version,omitempty"`  // TLS evidence on resume per SCACI §1
	CipherSuite    *string                `json:"cipher_suite,omitempty"` // TLS evidence on resume per SCACI §1
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	OrganizationID *uuid.UUID             `json:"organization_id,omitempty"` // Kilo Cloud org UUID
}

// SCACISessionFilter represents filter criteria for querying SCACI sessions
type SCACISessionFilter struct {
	TenantID       *int64     `json:"tenant_id,omitempty"`
	OrganizationID *uuid.UUID `json:"organization_id,omitempty"` // Kilo Cloud org UUID for persistence layer filtering
	AcEUI          *[8]byte   `json:"ac_eui,omitempty"`
	Status         *string    `json:"status,omitempty"`
	CanResume      *bool      `json:"can_resume,omitempty"`
	ConnectedFrom  *time.Time `json:"connected_from,omitempty"` // Sessions connected after this time
	ConnectedTo    *time.Time `json:"connected_to,omitempty"`   // Sessions connected before this time
	Limit          int        `json:"limit,omitempty"`
	Offset         int        `json:"offset,omitempty"`
}

// SCACISessionStatistics represents aggregated session statistics
type SCACISessionStatistics struct {
	TenantID               int64   `json:"tenant_id"`
	TotalSessions          int64   `json:"total_sessions"`
	ActiveSessions         int64   `json:"active_sessions"`
	ResumedSessions        int64   `json:"resumed_sessions"`
	DisconnectedSessions   int64   `json:"disconnected_sessions"`
	TerminatedSessions     int64   `json:"terminated_sessions"`
	ResumableSessions      int64   `json:"resumable_sessions"`
	AverageSessionDuration float64 `json:"average_session_duration_hours"`
	TotalSessionTime       float64 `json:"total_session_time_hours"`
}

// SCACISessionResumptionInfo provides information for session resumption decision
// Per SCACI §1, lines 22-27: session can be resumed if UUIDs match and opIds are consistent
// Per SCACI §§2.1-2.3: resume must also use the same negotiated version
type SCACISessionResumptionInfo struct {
	CanResume            bool    `json:"can_resume"`
	SessionID            int64   `json:"session_id,omitempty"`
	LastKnownAcOpId      int64   `json:"last_known_ac_op_id,omitempty"`
	LastKnownScOpId      int64   `json:"last_known_sc_op_id,omitempty"`
	NegotiatedVersion    string  `json:"negotiated_version,omitempty"` // Protocol version from original connect (§2.1-2.3)
	SessionAgeHours      float64 `json:"session_age_hours,omitempty"`
	ReasonIfNotResumable string  `json:"reason_if_not_resumable,omitempty"`
}

// ToCreateRequest converts a SCACISession to a create request (useful for testing/seeding)
func (s *SCACISession) ToCreateRequest() *SCACISessionCreateRequest {
	return &SCACISessionCreateRequest{
		TenantID:               s.TenantID,
		AcEUI:                  s.AcEUI,
		SnAcUUID:               s.SnAcUUID,
		SnScUUID:               s.SnScUUID,
		CertificateFingerprint: s.CertificateFingerprint,
		ClientCertSubject:      s.ClientCertSubject,
		RemoteAddr:             s.RemoteAddr,
		TLSVersion:             s.TLSVersion,
		CipherSuite:            s.CipherSuite,
		NegotiatedVersion:      s.NegotiatedVersion,
		CanResume:              s.CanResume,
		Metadata:               s.Metadata,
	}
}

// IsActive returns true if the session is currently active
func (s *SCACISession) IsActive() bool {
	return s.Status == "active" || s.Status == "resumed"
}

// IsResumable returns true if the session can be resumed after disconnect
func (s *SCACISession) IsResumable() bool {
	return s.CanResume && (s.Status == "disconnected" || s.Status == "active")
}

// GetSessionDuration returns the current or total duration of the session
func (s *SCACISession) GetSessionDuration() time.Duration {
	endTime := time.Now()
	if s.DisconnectedAt != nil {
		endTime = *s.DisconnectedAt
	}
	return endTime.Sub(s.ConnectedAt)
}
