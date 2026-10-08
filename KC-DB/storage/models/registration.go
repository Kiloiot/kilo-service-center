package models

import (
	"github.com/google/uuid"
)

// RegistrationResult contains the entities created during self-service registration.
type RegistrationResult struct {
	User         *User
	Organization *Organization
	TenantID     int64
}

// RegistrationParams contains the input for self-service registration.
type RegistrationParams struct {
	User        *User  // Pre-populated with ID, email, password_hash, profile fields, is_admin=false
	CompanyName string // Used for tenant name + organization name
	Description string // Organization description (optional)
}

// CERegistrationParams contains input for CE self-service registration (existing tenant/org).
type CERegistrationParams struct {
	User     *User
	TenantID int64     // Existing CE tenant
	OrgID    uuid.UUID // Existing CE default org
}
