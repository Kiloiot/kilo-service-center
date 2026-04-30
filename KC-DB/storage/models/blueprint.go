// Package models defines the database models for KiloCenter
package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Blueprint represents a payload decoding specification for a device model.
// Each blueprint contains a JSON specification that defines how to decode
// MIOTY payloads per the Application Layer Specification.
// Migration 000104 creates the blueprints table.
type Blueprint struct {
	ID               uuid.UUID       `db:"id" json:"id"`
	DeviceModelID    uuid.UUID       `db:"device_model_id" json:"deviceModelId"`
	TenantID         int64           `db:"tenant_id" json:"tenantId"`
	Version          string          `db:"version" json:"version"`                                 // Semantic version (e.g., "1.0.0")
	TypeEUI          []byte          `db:"type_eui" json:"typeEui"`                                // 8-byte MIOTY Type EUI (optional, resolved via device model fallback)
	SpecJSON         json.RawMessage `db:"spec_json" json:"specJson"`                              // Blueprint specification JSON
	IsDefault        bool            `db:"is_default" json:"isDefault"`                            // true if this is the default blueprint for the model
	RegistryRepo     *string         `db:"registry_repo" json:"registryRepo,omitempty"`            // GitHub repo (e.g., "mioty-alliance/device-blueprints")
	RegistryCommit   *string         `db:"registry_commit_sha" json:"registryCommitSha,omitempty"` // Git commit SHA
	RegistryVerified bool            `db:"registry_verified" json:"registryVerified"`              // true if verified from registry
	RegistryPRURL    *string         `db:"registry_pr_url" json:"registryPrUrl,omitempty"`         // GitHub PR URL for submission
	CreatedAt        time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt        time.Time       `db:"updated_at" json:"updatedAt"`

	// Joined data (not persisted, populated on queries)
	DeviceModel *DeviceModel `db:"-" json:"deviceModel,omitempty"`
}

// BlueprintCreateParams contains the parameters for creating a new blueprint
type BlueprintCreateParams struct {
	DeviceModelID uuid.UUID
	TenantID      int64
	Version       string
	TypeEUI       []byte // 8-byte MIOTY Type EUI (optional, resolved via device model fallback)
	SpecJSON      json.RawMessage
	IsDefault     bool
}

// BlueprintUpdateParams contains the parameters for updating an existing blueprint
type BlueprintUpdateParams struct {
	Version   *string
	TypeEUI   []byte // 8-byte MIOTY Type EUI
	SpecJSON  json.RawMessage
	IsDefault *bool
}

// BlueprintListParams contains the parameters for listing blueprints
type BlueprintListParams struct {
	TenantID      int64
	DeviceModelID *uuid.UUID // Optional filter by device model
	Limit         int
	Offset        int
}

// BlueprintWithModel represents a blueprint with its device model and manufacturer details
type BlueprintWithModel struct {
	Blueprint
	DeviceModelName  string    `db:"device_model_name" json:"deviceModelName"`
	DeviceModelCode  string    `db:"device_model_code" json:"deviceModelCode"`
	ManufacturerID   uuid.UUID `db:"manufacturer_id" json:"manufacturerId"`
	ManufacturerName string    `db:"manufacturer_name" json:"manufacturerName"`
}

// Note: BlueprintSpec, PayloadFormat, PayloadComponent, and DecodeResult types
// are defined in KC-Core/pkg/blueprint/types.go. Import from there to avoid duplication.
