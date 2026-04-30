package models

import (
	"database/sql"
	"database/sql/driver"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq/hstore"
)

// HstoreMap is a custom type that wraps map[string]string for PostgreSQL hstore scanning.
// It implements sql.Scanner and driver.Valuer for automatic conversion.
type HstoreMap map[string]string

// Scan implements sql.Scanner interface for reading hstore from database.
func (h *HstoreMap) Scan(value interface{}) error {
	if value == nil {
		*h = nil
		return nil
	}

	var hs hstore.Hstore
	if err := hs.Scan(value); err != nil {
		return err
	}

	// Convert hstore.Hstore to map[string]string
	result := make(map[string]string)
	for k, v := range hs.Map {
		if v.Valid {
			result[k] = v.String
		}
	}
	*h = result
	return nil
}

// Value implements driver.Valuer interface for writing hstore to database.
func (h HstoreMap) Value() (driver.Value, error) {
	if h == nil {
		return nil, nil
	}

	// Convert map[string]string to hstore.Hstore
	hsMap := make(map[string]sql.NullString)
	for k, v := range h {
		hsMap[k] = sql.NullString{String: v, Valid: true}
	}
	hs := hstore.Hstore{Map: hsMap}
	return hs.Value()
}

// Organization represents an organization in the Kilo Cloud multi-tenant system
// Each organization maps to a numeric tenant ID for database sharding
// Migration 029 creates one default org per existing tenant
// Migration 000099: Added description, quota fields, and tags
type Organization struct {
	OrgID               uuid.UUID `db:"org_id"`
	TenantID            int64     `db:"tenant_id"`
	Name                string    `db:"name"`
	State               string    `db:"state"`                                                          // active, suspended, archived
	ExternalID          *string   `db:"external_id" json:"external_id,omitempty"`                       // External IdP identifier for org claim mapping (nullable)
	Description         *string   `db:"description" json:"description,omitempty"`                       // nullable description
	CanHaveBaseStations bool      `db:"can_have_base_stations" json:"can_have_base_stations"`           // quota flag
	MaxBaseStationCount *int      `db:"max_base_station_count" json:"max_base_station_count,omitempty"` // NULL = unlimited
	MaxEndpointCount    *int      `db:"max_endpoint_count" json:"max_endpoint_count,omitempty"`         // NULL = unlimited
	Tags                HstoreMap `db:"tags" json:"tags,omitempty"`                                     // HSTORE key-value metadata (uses custom scanner)
	CreatedAt           time.Time `db:"created_at"`
	UpdatedAt           time.Time `db:"updated_at"`
}

// OrganizationMember represents a user's membership in an organization
// Mirrors Kilo Cloud org_membership semantics for role-based access control
// Migration 000100: Added boolean permission flags
type OrganizationMember struct {
	OrgID              uuid.UUID `db:"org_id"`
	UserID             uuid.UUID `db:"user_id"`
	Role               string    `db:"role"`                                               // owner, admin, member
	Status             string    `db:"status"`                                             // active, invited, removed
	IsOrgAdmin         bool      `db:"is_org_admin" json:"is_org_admin"`                   // can manage org settings and users
	IsBaseStationAdmin bool      `db:"is_base_station_admin" json:"is_base_station_admin"` // can manage base stations
	IsEndpointAdmin    bool      `db:"is_endpoint_admin" json:"is_endpoint_admin"`         // can manage endpoints
	CreatedAt          time.Time `db:"created_at"`
	UpdatedAt          time.Time `db:"updated_at"`
}

// OrganizationMemberWithEmail extends OrganizationMember with user email for list views.
// Used by ListOrgMembersWithEmail JOIN query.
type OrganizationMemberWithEmail struct {
	OrgID              uuid.UUID `db:"org_id" json:"org_id"`
	UserID             uuid.UUID `db:"user_id" json:"user_id"`
	Email              string    `db:"email" json:"email"` // from users table via JOIN
	Role               string    `db:"role" json:"role"`
	Status             string    `db:"status" json:"status"`
	IsOrgAdmin         bool      `db:"is_org_admin" json:"is_org_admin"`
	IsBaseStationAdmin bool      `db:"is_base_station_admin" json:"is_base_station_admin"`
	IsEndpointAdmin    bool      `db:"is_endpoint_admin" json:"is_endpoint_admin"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time `db:"updated_at" json:"updated_at"`
}
