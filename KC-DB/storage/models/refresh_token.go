package models

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a JWT refresh token stored for rotation and reuse detection.
type RefreshToken struct {
	ID         uuid.UUID  `db:"id"`
	UserID     uuid.UUID  `db:"user_id"`
	TokenHash  string     `db:"token_hash"` // SHA256 hex hash (never store raw tokens)
	IssuedAt   time.Time  `db:"issued_at"`
	ExpiresAt  time.Time  `db:"expires_at"`
	RevokedAt  *time.Time `db:"revoked_at"`  // Set when token is revoked
	ReplacedBy *uuid.UUID `db:"replaced_by"` // Points to successor token after rotation
	CreatedAt  time.Time  `db:"created_at"`
}

// IsExpired checks if the token has expired.
func (t *RefreshToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsRevoked checks if the token has been revoked.
func (t *RefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

// IsReplaced checks if the token has been replaced by another token (used during rotation).
func (t *RefreshToken) IsReplaced() bool {
	return t.ReplacedBy != nil
}

// IsValid checks if the token is usable (not expired, not revoked, not replaced).
func (t *RefreshToken) IsValid() bool {
	return !t.IsExpired() && !t.IsRevoked() && !t.IsReplaced()
}
