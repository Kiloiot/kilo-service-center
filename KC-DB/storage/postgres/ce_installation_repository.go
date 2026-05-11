package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/jmoiron/sqlx"
)

// CEInstallationRepository manages the singleton CE installation record using PostgreSQL.
type CEInstallationRepository struct {
	db *sqlx.DB
}

// NewCEInstallationRepository creates a new CEInstallationRepository backed by sqlx.
func NewCEInstallationRepository(db *sqlx.DB) *CEInstallationRepository {
	return &CEInstallationRepository{db: db}
}

// Get retrieves the singleton installation record. Returns nil, nil if none exists.
func (r *CEInstallationRepository) Get(ctx context.Context) (*interfaces.CEInstallation, error) {
	var row interfaces.CEInstallation
	err := r.db.GetContext(ctx, &row, `SELECT * FROM ce_installation WHERE id = 1`)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ce_installation get: %w", err)
	}
	return &row, nil
}

// Create inserts the initial singleton record.
func (r *CEInstallationRepository) Create(ctx context.Context, inst *interfaces.CEInstallation) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ce_installation (id, ce_id, company_name, onboarding_completed_at, federation_token, token_issued_at)
		VALUES (1, $1, $2, $3, $4, $5)
	`, inst.CEID, inst.CompanyName, inst.OnboardingCompletedAt, inst.FederationToken, inst.TokenIssuedAt)
	if err != nil {
		return fmt.Errorf("ce_installation create: %w", err)
	}
	return nil
}

// Update applies partial updates using a field map. Only provided keys are changed.
func (r *CEInstallationRepository) Update(ctx context.Context, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setClauses := make([]string, 0, len(updates)+1)
	args := make([]interface{}, 0, len(updates)+1)
	i := 1
	for col, val := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, i))
		args = append(args, val)
		i++
	}
	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf("UPDATE ce_installation SET %s WHERE id = 1", strings.Join(setClauses, ", "))
	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ce_installation update: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}
