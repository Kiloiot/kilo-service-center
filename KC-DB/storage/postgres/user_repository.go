package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// UserRepository implements the UserRepository interface for PostgreSQL
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new PostgreSQL User repository
func NewUserRepository(db *sqlx.DB) interfaces.UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	query := `
		INSERT INTO users (
			id, external_id, email, email_verified, password_hash,
			is_admin, is_active, is_tenant_manager, is_base_station_manager,
			is_endpoint_manager, note, first_name, last_name, company_name, created_at, updated_at
		) VALUES (
			:id, :external_id, :email, :email_verified, :password_hash,
			:is_admin, :is_active, :is_tenant_manager, :is_base_station_manager,
			:is_endpoint_manager, :note, :first_name, :last_name, :company_name, :created_at, :updated_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by UUID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, external_id, email, email_verified, password_hash,
			   is_admin, is_active, is_tenant_manager, is_base_station_manager,
			   is_endpoint_manager, note, first_name, last_name, company_name, created_at, updated_at
		FROM users
		WHERE id = $1`

	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user %s: %w", id, interfaces.ErrRecordNotFound)
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by email address
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, external_id, email, email_verified, password_hash,
			   is_admin, is_active, is_tenant_manager, is_base_station_manager,
			   is_endpoint_manager, note, first_name, last_name, company_name, created_at, updated_at
		FROM users
		WHERE LOWER(TRIM(email)) = LOWER(TRIM($1))`

	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user with email %s: %w", email, interfaces.ErrRecordNotFound)
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return &user, nil
}

// GetByExternalID retrieves a user by external provider ID (OIDC)
func (r *UserRepository) GetByExternalID(ctx context.Context, externalID string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, external_id, email, email_verified, password_hash,
			   is_admin, is_active, is_tenant_manager, is_base_station_manager,
			   is_endpoint_manager, note, first_name, last_name, company_name, created_at, updated_at
		FROM users
		WHERE external_id = $1`

	err := r.db.GetContext(ctx, &user, query, externalID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user with external_id %s: %w", externalID, interfaces.ErrRecordNotFound)
		}
		return nil, fmt.Errorf("get user by external_id: %w", err)
	}

	return &user, nil
}

// Update modifies an existing user
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET external_id = :external_id,
			email = :email,
			email_verified = :email_verified,
			is_admin = :is_admin,
			is_active = :is_active,
			is_tenant_manager = :is_tenant_manager,
			is_base_station_manager = :is_base_station_manager,
			is_endpoint_manager = :is_endpoint_manager,
			note = :note,
			first_name = :first_name,
			last_name = :last_name,
			company_name = :company_name,
			updated_at = NOW()
		WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user %s not found", user.ID)
	}

	return nil
}

// SetPasswordHash updates only the password hash field
func (r *UserRepository) SetPasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	query := `
		UPDATE users
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, hash)
	if err != nil {
		return fmt.Errorf("set password hash: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user %s not found", id)
	}

	return nil
}

// Delete removes a user by ID
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user %s not found", id)
	}

	return nil
}

// List returns users with pagination
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	var users []*models.User
	query := `
		SELECT id, external_id, email, email_verified, password_hash,
			   is_admin, is_active, is_tenant_manager, is_base_station_manager,
			   is_endpoint_manager, note, first_name, last_name, company_name, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	err := r.db.SelectContext(ctx, &users, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	return users, nil
}

// Count returns total user count
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM users`

	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}

	return count, nil
}
