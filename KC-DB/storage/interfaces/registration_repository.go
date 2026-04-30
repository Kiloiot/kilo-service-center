package interfaces

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-DB/storage/models"
)

// RegistrationResult contains the entities created during self-service registration.
type RegistrationResult struct {
	User         *models.User
	Organization *models.Organization
	TenantID     int64
}

// RegistrationParams contains the input for self-service registration.
type RegistrationParams struct {
	User        *models.User // Pre-populated with ID, email, password_hash, profile fields, is_admin=false
	CompanyName string       // Used for tenant name + organization name
	Description string       // Organization description (optional)
}

// CERegistrationParams contains input for CE self-service registration (existing tenant/org).
type CERegistrationParams struct {
	User     *models.User
	TenantID int64     // Existing CE tenant
	OrgID    uuid.UUID // Existing CE default org
}

// RegistrationRepository handles atomic account registration (user + tenant + org + membership).
type RegistrationRepository interface {
	RegisterAccount(ctx context.Context, params *RegistrationParams) (*RegistrationResult, error)
	RegisterCEAccount(ctx context.Context, params *CERegistrationParams) (*RegistrationResult, error)
}
