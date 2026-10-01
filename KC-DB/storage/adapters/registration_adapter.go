package adapters

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
)

// RegistrationAdapter exposes the Postgres registration repository for KC-Identity consumption.
type RegistrationAdapter struct {
	repo *postgres.RegistrationRepository
}

// NewRegistrationAdapter creates a new adapter wrapping the Postgres registration repository.
func NewRegistrationAdapter(repo *postgres.RegistrationRepository) *RegistrationAdapter {
	return &RegistrationAdapter{
		repo: repo,
	}
}

// RegisterAccount delegates to the Postgres registration repository.
func (a *RegistrationAdapter) RegisterAccount(ctx context.Context, params *models.RegistrationParams) (*models.RegistrationResult, error) {
	return a.repo.RegisterAccount(ctx, params)
}

// RegisterCEAccount delegates CE registration to the Postgres registration repository.
func (a *RegistrationAdapter) RegisterCEAccount(ctx context.Context, params *models.CERegistrationParams) (*models.RegistrationResult, error) {
	return a.repo.RegisterCEAccount(ctx, params)
}
