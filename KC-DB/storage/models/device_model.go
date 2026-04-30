// Package models defines the database models for KiloCenter
package models

import (
	"time"

	"github.com/google/uuid"
)

// DeviceModel represents a device model within a manufacturer's product line.
// Each model can have multiple blueprint versions for payload decoding.
// The TypeEUI field is an 8-byte MIOTY Type EUI per Application Layer Specification.
// Migration 000103 creates the device_models table with manufacturer foreign key.
type DeviceModel struct {
	ID             uuid.UUID `db:"id" json:"id"`
	ManufacturerID uuid.UUID `db:"manufacturer_id" json:"manufacturerId"`
	TenantID       int64     `db:"tenant_id" json:"tenantId"`
	Name           string    `db:"name" json:"name"`
	Code           string    `db:"code" json:"code"`                  // URL-friendly slug, unique per manufacturer
	TypeEUI        []byte    `db:"type_eui" json:"typeEui,omitempty"` // 8-byte MIOTY Type EUI (nullable)
	Description    *string   `db:"description" json:"description,omitempty"`
	DatasheetURL   *string   `db:"datasheet_url" json:"datasheetUrl,omitempty"`
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time `db:"updated_at" json:"updatedAt"`

	// Joined data (not persisted, populated on queries)
	Manufacturer   *Manufacturer `db:"-" json:"manufacturer,omitempty"`
	BlueprintCount int           `db:"blueprint_count" json:"blueprintCount,omitempty"`
}

// DeviceModelCreateParams contains the parameters for creating a new device model
type DeviceModelCreateParams struct {
	ManufacturerID uuid.UUID
	TenantID       int64
	Name           string
	Code           string
	TypeEUI        []byte // 8-byte MIOTY Type EUI (optional)
	Description    *string
	DatasheetURL   *string
}

// DeviceModelUpdateParams contains the parameters for updating an existing device model
type DeviceModelUpdateParams struct {
	Name         *string
	Code         *string
	TypeEUI      []byte // 8-byte MIOTY Type EUI (nil to clear)
	Description  *string
	DatasheetURL *string
}

// DeviceModelListParams contains the parameters for listing device models
type DeviceModelListParams struct {
	TenantID       int64
	ManufacturerID *uuid.UUID // Optional filter by manufacturer
	Limit          int
	Offset         int
	SearchTerm     string // Optional search term for name/code
}

// DeviceModelWithManufacturer represents a device model with its manufacturer details
type DeviceModelWithManufacturer struct {
	DeviceModel
	ManufacturerName string `db:"manufacturer_name" json:"manufacturerName"`
}
