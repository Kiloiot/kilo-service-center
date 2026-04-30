package models

import (
	"time"

	"github.com/google/uuid"
)

// API Key types (centralized constants for KC-DB boundary).
const (
	// KeyTypeUser represents a personal API key tied to a user identity.
	KeyTypeUser = "user"

	// KeyTypeServiceAccount represents an organization service account key for automation.
	// Service account keys have user_id = NULL and are scoped to the organization.
	KeyTypeServiceAccount = "service_account"
)

// APIKey represents an API key for programmatic access to KiloCenter.
//
// Key types:
//   - "user": Personal key tied to user identity (user_id NOT NULL)
//   - "service_account": Org automation key (user_id NULL, scoped to org_id)
type APIKey struct {
	ID         uuid.UUID  `db:"id"`
	TenantID   int64      `db:"tenant_id"`
	OrgID      uuid.UUID  `db:"org_id"`
	UserID     *uuid.UUID `db:"user_id"`    // Nullable for service accounts
	Name       string     `db:"name"`       // Display name for the key
	KeyHash    string     `db:"key_hash"`   // SHA-256 hash of the key
	KeyPrefix  string     `db:"key_prefix"` // First 8 chars for identification
	KeyType    string     `db:"key_type"`   // "user" or "service_account"
	ExpiresAt  *time.Time `db:"expires_at"` // Optional expiration
	LastUsedAt *time.Time `db:"last_used_at"`
	IsActive   bool       `db:"is_active"`
	CreatedAt  time.Time  `db:"created_at"`
}

// IsExpired returns true if the key has an expiration time and it has passed.
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().UTC().After(*k.ExpiresAt)
}

// IsServiceAccount returns true if this is an organization service account key.
func (k *APIKey) IsServiceAccount() bool {
	return k.KeyType == KeyTypeServiceAccount
}
