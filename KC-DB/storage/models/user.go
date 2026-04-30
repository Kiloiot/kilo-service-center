package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents an authenticated user in the system.
// Local auth user schema with role flags for granular permissions.
type User struct {
	ID                   uuid.UUID `db:"id"`
	ExternalID           *string   `db:"external_id"` // nullable (OIDC provider ID)
	Email                string    `db:"email"`
	EmailVerified        bool      `db:"email_verified"`
	PasswordHash         *string   `db:"password_hash"` // nullable (OIDC users have no local password)
	IsAdmin              bool      `db:"is_admin"`
	IsActive             bool      `db:"is_active"`
	IsTenantManager      bool      `db:"is_tenant_manager"`       // can manage tenant config
	IsBaseStationManager bool      `db:"is_base_station_manager"` // can manage base stations
	IsEndpointManager    bool      `db:"is_endpoint_manager"`     // can manage endpoints
	Note                 *string   `db:"note"`                    // nullable
	FirstName            *string   `db:"first_name"`
	LastName             *string   `db:"last_name"`
	CompanyName          *string   `db:"company_name"`
	CreatedAt            time.Time `db:"created_at"`
	UpdatedAt            time.Time `db:"updated_at"`
}
