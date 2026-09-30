package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// sqlCompleteOnboarding creates the installation with a new CE ID, or
// completes an unfinished one and keeps its CE ID; an installation that has
// completed onboarding matches no row, so concurrent calls complete it once.
const sqlCompleteOnboarding = `
	INSERT INTO ce_installation (id, ce_id, company_name, onboarding_completed_at, created_at, updated_at)
	VALUES (1, gen_random_uuid(), $1, $2, $2, $2)
	ON CONFLICT (id) DO UPDATE
	SET company_name = EXCLUDED.company_name,
	    onboarding_completed_at = EXCLUDED.onboarding_completed_at,
	    updated_at = EXCLUDED.updated_at
	WHERE ce_installation.onboarding_completed_at IS NULL
	RETURNING ce_id`

// CEInstallationRepository manages the singleton CE installation record using PostgreSQL.
type CEInstallationRepository struct {
	db *sqlx.DB
}

// NewCEInstallationRepository creates a new CEInstallationRepository backed by sqlx.
func NewCEInstallationRepository(db *sqlx.DB) *CEInstallationRepository {
	return &CEInstallationRepository{db: db}
}

// Get retrieves the singleton installation record, or storage.ErrNotFound before one exists.
func (r *CEInstallationRepository) Get(ctx context.Context) (*models.CEInstallation, error) {
	var row models.CEInstallation
	err := r.db.GetContext(ctx, &row, `SELECT * FROM ce_installation WHERE id = 1`)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", errWrapCeInstallationGet, err)
	}
	return &row, nil
}

// CompleteOnboarding records companyName as completed at the given time and
// returns the installation's CE ID. It fails with
// storage.ErrInstallationOnboarded once onboarding has completed.
func (r *CEInstallationRepository) CompleteOnboarding(ctx context.Context, companyName string, at time.Time) (uuid.UUID, error) {
	var ceID uuid.UUID
	err := r.db.GetContext(ctx, &ceID, sqlCompleteOnboarding, companyName, at)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, storage.ErrInstallationOnboarded
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", errWrapCeInstallationCompleteOnboarding, err)
	}
	return ceID, nil
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
		return fmt.Errorf("%s: %w", errWrapCeInstallationUpdate, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapCeInstallationUpdate, err)
	}
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}
