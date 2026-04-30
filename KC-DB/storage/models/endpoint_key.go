package models

import (
	"encoding/json"
	"time"
)

// EndPointKey represents a cryptographic key for an endpoint
type EndPointKey struct {
	ID         string `json:"id"`
	EndPointID string `json:"endpoint_id"`
	TenantID   string `json:"tenant_id"`

	// Key type and version
	KeyType    string `json:"key_type"` // network, application, join
	KeyVersion int32  `json:"key_version"`

	// Key data (encrypted at rest in production)
	KeyValue []byte `json:"key_value"`

	// Key metadata
	KeyID     *string `json:"key_id"`    // External key identifier
	Algorithm string  `json:"algorithm"` // AES-128, AES-256

	// Validity period
	ValidFrom  time.Time  `json:"valid_from"`
	ValidUntil *time.Time `json:"valid_until"`
	IsActive   bool       `json:"is_active"`

	// Usage tracking
	UsageCount int64      `json:"usage_count"`
	LastUsedAt *time.Time `json:"last_used_at"`

	// Key rotation
	RotatedFrom    *string `json:"rotated_from"`
	RotationReason *string `json:"rotation_reason"`

	// Metadata
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
	CreatedBy *string         `json:"created_by"`
}

// EndPointKeyStats represents key usage statistics
type EndPointKeyStats struct {
	EndPointID       string           `json:"endpoint_id"`
	TotalKeys        int64            `json:"total_keys"`
	ActiveKeys       int64            `json:"active_keys"`
	InactiveKeys     int64            `json:"inactive_keys"`
	ExpiredKeys      int64            `json:"expired_keys"`
	NetworkKeys      int64            `json:"network_keys"`
	ApplicationKeys  int64            `json:"application_keys"`
	JoinKeys         int64            `json:"join_keys"`
	TotalUsage       int64            `json:"total_usage"`
	TotalUsageCount  int64            `json:"total_usage_count"`
	KeysRotatedCount int64            `json:"keys_rotated_count"`
	KeyTypeBreakdown map[string]int64 `json:"key_type_breakdown"`
}
