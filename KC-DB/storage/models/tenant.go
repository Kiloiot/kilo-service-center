package models

import (
	"time"
)

// Tenant represents a tenant in the system
type Tenant struct {
	ID          int64     `db:"id"`
	Name        string    `db:"name"`
	Description *string   `db:"description"` // Nullable TEXT column
	Status      string    `db:"status"`      // VARCHAR(50) NOT NULL DEFAULT 'active'
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
