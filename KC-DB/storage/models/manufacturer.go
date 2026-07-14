// Package models defines the database models for KiloCenter
package models

import (
	"time"

	"github.com/google/uuid"
)

// Manufacturer represents a device manufacturer in the KiloCenter device catalog.
// Manufacturers own DeviceModels which in turn own Blueprints for payload decoding.
// Migration 000102 creates the manufacturers table with tenant isolation.
// Migration 000107 simplifies to name + website only.
type Manufacturer struct {
	ID         uuid.UUID `db:"id" json:"id"`
	TenantID   *int64    `db:"tenant_id" json:"tenantId,omitempty"` // NULL for System-owned rows
	IsSystem   bool      `db:"is_system" json:"isSystem"`           // true = System catalog, false = tenant Custom
	Name       string    `db:"name" json:"name"`
	Website    *string   `db:"website" json:"website,omitempty"`
	IsVerified bool      `db:"is_verified" json:"isVerified"` // true if verified through GitHub registry
	CreatedAt  time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt  time.Time `db:"updated_at" json:"updatedAt"`

	// Joined data (not persisted, populated on list queries)
	ModelCount int `db:"model_count" json:"modelCount,omitempty"`
}

// ManufacturerCreateParams contains the parameters for creating a new manufacturer
type ManufacturerCreateParams struct {
	TenantID int64
	IsSystem bool // true = System row (tenant_id NULL); admin-gated at the handler
	Name     string
	Website  *string
}

// ManufacturerUpdateParams contains the parameters for updating an existing manufacturer
type ManufacturerUpdateParams struct {
	Name    *string
	Website *string
}

// ManufacturerListParams contains the parameters for listing manufacturers
type ManufacturerListParams struct {
	TenantID   int64
	IsSystem   bool // true = System catalog, false = tenant Custom
	Limit      int
	Offset     int
	SearchTerm string // Optional search term for name/code
}
