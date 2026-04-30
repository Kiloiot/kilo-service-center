package models

import (
	"encoding/json"
	"time"
)

// DeviceKeys represents cryptographic keys for a device
type DeviceKeys struct {
	ID       int64 `db:"id" json:"id"`
	DeviceID int64 `db:"device_id" json:"device_id"`
	TenantID int64 `db:"tenant_id" json:"tenant_id"`

	// Key type and version
	KeyType    string `db:"key_type" json:"key_type"`       // network, application, join, session
	KeyVersion int32  `db:"key_version" json:"key_version"` // Version number for key rotation

	// Key data
	KeyValue []byte `db:"key_value" json:"-"` // Hidden in JSON for security

	// Key metadata
	KeyID     *string `db:"key_id" json:"key_id,omitempty"` // Optional key identifier
	Algorithm string  `db:"algorithm" json:"algorithm"`     // AES-128, AES-256

	// Validity period
	ValidFrom  time.Time  `db:"valid_from" json:"valid_from"`
	ValidUntil *time.Time `db:"valid_until" json:"valid_until,omitempty"`
	IsActive   bool       `db:"is_active" json:"is_active"`

	// Usage tracking
	UsageCount int64      `db:"usage_count" json:"usage_count"`
	LastUsedAt *time.Time `db:"last_used_at" json:"last_used_at,omitempty"`

	// Key rotation
	RotatedFrom    *int64  `db:"rotated_from" json:"rotated_from,omitempty"`
	RotationReason *string `db:"rotation_reason" json:"rotation_reason,omitempty"`

	// Metadata
	Metadata  json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	CreatedBy *string         `db:"created_by" json:"created_by,omitempty"`
}

// KeyType constants
const (
	KeyTypeNetwork     = "network"
	KeyTypeApplication = "application"
	KeyTypeJoin        = "join"
	KeyTypeSession     = "session"
)

// Algorithm constants
const (
	AlgorithmAES128 = "AES-128"
	AlgorithmAES256 = "AES-256"
)
