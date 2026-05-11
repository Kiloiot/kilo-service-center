package adapters

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/jmoiron/sqlx"
)

// RegistrationAdapter exposes the RegistrationRepository for KC-Identity consumption.
type RegistrationAdapter struct {
	repo *postgres.RegistrationRepository
}

// NewRegistrationAdapter creates a new adapter wrapping the Postgres registration repository.
func NewRegistrationAdapter(db *sqlx.DB) *RegistrationAdapter {
	return &RegistrationAdapter{
		repo: postgres.NewRegistrationRepository(db),
	}
}

// RegisterAccount delegates to the Postgres registration repository.
func (a *RegistrationAdapter) RegisterAccount(ctx context.Context, params *interfaces.RegistrationParams) (*interfaces.RegistrationResult, error) {
	return a.repo.RegisterAccount(ctx, params)
}

// RegisterCEAccount delegates CE registration to the Postgres registration repository.
func (a *RegistrationAdapter) RegisterCEAccount(ctx context.Context, params *interfaces.CERegistrationParams) (*interfaces.RegistrationResult, error) {
	return a.repo.RegisterCEAccount(ctx, params)
}
