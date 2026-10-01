package models

import (
	"time"

	"github.com/google/uuid"
)

// CEInstallation holds the identity and federation credentials for a Community Edition instance.
type CEInstallation struct {
	ID                    int        `db:"id"`
	CEID                  uuid.UUID  `db:"ce_id"`
	CompanyName           string     `db:"company_name"`
	OnboardingCompletedAt *time.Time `db:"onboarding_completed_at"`
	FederationToken       *string    `db:"federation_token"`
	TokenIssuedAt         *time.Time `db:"token_issued_at"`
	CreatedAt             time.Time  `db:"created_at"`
	UpdatedAt             time.Time  `db:"updated_at"`
}
