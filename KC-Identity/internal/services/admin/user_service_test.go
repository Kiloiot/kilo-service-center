package admin

import (
	"context"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	"github.com/stretchr/testify/assert"
)

// mockUserStore implements UserAdminStore for testing.
type mockUserStore struct {
	getByIDFn     func(ctx context.Context, id uuid.UUID) (*models.User, error)
	getByEmailFn  func(ctx context.Context, email string) (*models.User, error)
	createFn      func(ctx context.Context, user *models.User) error
	updateFn      func(ctx context.Context, user *models.User) error
	deleteFn      func(ctx context.Context, id uuid.UUID) error
	listFn        func(ctx context.Context, limit, offset int) ([]*models.User, error)
	countFn       func(ctx context.Context) (int64, error)
	setPasswordFn func(ctx context.Context, userID uuid.UUID, hash string) error
}

func (m *mockUserStore) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockUserStore) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return m.getByEmailFn(ctx, email)
}
func (m *mockUserStore) Create(ctx context.Context, user *models.User) error {
	return m.createFn(ctx, user)
}
func (m *mockUserStore) Update(ctx context.Context, user *models.User) error {
	return m.updateFn(ctx, user)
}
func (m *mockUserStore) Delete(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}
func (m *mockUserStore) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	return m.listFn(ctx, limit, offset)
}
func (m *mockUserStore) Count(ctx context.Context) (int64, error) {
	return m.countFn(ctx)
}
func (m *mockUserStore) SetPasswordHash(ctx context.Context, userID uuid.UUID, hash string) error {
	return m.setPasswordFn(ctx, userID, hash)
}

func newTestLogger() logger.Logger {
	return logger.NewNop()
}

func TestCreateUser_WithManagerFields(t *testing.T) {
	var createdUser *models.User

	store := &mockUserStore{
		getByEmailFn: func(_ context.Context, _ string) (*models.User, error) {
			return nil, interfaces.ErrRecordNotFound
		},
		createFn: func(_ context.Context, user *models.User) error {
			createdUser = user
			return nil
		},
	}

	svc := NewUserAdminService(store, newTestLogger())

	req := &grpcservices.UserCreateRequest{
		Email:                "admin@example.com",
		Password:             "SecurePassword123!",
		IsAdmin:              true,
		IsActive:             true,
		IsTenantManager:      true,
		IsBaseStationManager: true,
		IsEndpointManager:    true,
		Note:                 "test note",
	}

	user, err := svc.Create(testutil.TestContext(), req)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if user == nil {
		t.Fatal("Create() returned nil user")
	}

	if createdUser == nil {
		t.Fatal("store.Create was not called")
	}

	if !createdUser.IsTenantManager {
		t.Error("IsTenantManager should be true")
	}
	if !createdUser.IsBaseStationManager {
		t.Error("IsBaseStationManager should be true")
	}
	if !createdUser.IsEndpointManager {
		t.Error("IsEndpointManager should be true")
	}
	if !createdUser.IsAdmin {
		t.Error("IsAdmin should be true")
	}
	if !createdUser.IsActive {
		t.Error("IsActive should be true")
	}
}

func TestUpdateUser_FieldMask_ToggleAdminOff(t *testing.T) {
	userID := uuid.New()
	existingUser := &models.User{
		ID:      userID,
		Email:   "admin@example.com",
		IsAdmin: true,
	}

	var updatedUser *models.User

	store := &mockUserStore{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*models.User, error) {
			if id == userID {
				// Return a copy to avoid mutation
				u := *existingUser
				return &u, nil
			}
			return nil, interfaces.ErrRecordNotFound
		},
		updateFn: func(_ context.Context, user *models.User) error {
			updatedUser = user
			return nil
		},
	}

	svc := NewUserAdminService(store, newTestLogger())

	isAdminFalse := false
	req := &grpcservices.UserUpdateRequest{
		IsAdmin: &isAdminFalse,
	}

	user, err := svc.Update(testutil.TestContext(), userID, req)
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if user.IsAdmin {
		t.Error("IsAdmin should be false after update")
	}
	if updatedUser == nil {
		t.Fatal("store.Update was not called")
	}
	if updatedUser.IsAdmin {
		t.Error("stored user IsAdmin should be false")
	}
}

func TestUpdateUser_FieldMask_ManagerFields(t *testing.T) {
	userID := uuid.New()
	existingUser := &models.User{
		ID:    userID,
		Email: "user@example.com",
	}

	var updatedUser *models.User

	store := &mockUserStore{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*models.User, error) {
			if id == userID {
				u := *existingUser
				return &u, nil
			}
			return nil, interfaces.ErrRecordNotFound
		},
		updateFn: func(_ context.Context, user *models.User) error {
			updatedUser = user
			return nil
		},
	}

	svc := NewUserAdminService(store, newTestLogger())

	trueVal := true
	req := &grpcservices.UserUpdateRequest{
		IsTenantManager:      &trueVal,
		IsBaseStationManager: &trueVal,
		IsEndpointManager:    &trueVal,
	}

	_, err := svc.Update(testutil.TestContext(), userID, req)
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if updatedUser == nil {
		t.Fatal("store.Update was not called")
	}
	if !updatedUser.IsTenantManager {
		t.Error("IsTenantManager should be true")
	}
	if !updatedUser.IsBaseStationManager {
		t.Error("IsBaseStationManager should be true")
	}
	if !updatedUser.IsEndpointManager {
		t.Error("IsEndpointManager should be true")
	}
}

func TestUpdateUser_MixedCaseEmailSelfUpdate(t *testing.T) {
	userID := uuid.New()

	store := &mockUserStore{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*models.User, error) {
			if id == userID {
				return &models.User{ID: userID, Email: "User@Example.Com", IsActive: true}, nil
			}
			return nil, interfaces.ErrRecordNotFound
		},
		getByEmailFn: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{ID: userID, Email: "user@example.com"}, nil
		},
		updateFn: func(_ context.Context, _ *models.User) error {
			return nil
		},
	}

	svc := NewUserAdminService(store, newTestLogger())

	normalized := "user@example.com"
	_, err := svc.Update(testutil.TestContext(), userID, &grpcservices.UserUpdateRequest{Email: &normalized})
	if err != nil {
		t.Fatalf("Update() returned error: %v — mixed-case self-update should not trigger duplicate", err)
	}
}

func TestUpdateUser_GetByEmailReturnsNilNil(t *testing.T) {
	userID := uuid.New()

	store := &mockUserStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, Email: "old@example.com", IsActive: true}, nil
		},
		getByEmailFn: func(_ context.Context, _ string) (*models.User, error) {
			return nil, nil // hypothetical repository returns (nil, nil)
		},
		updateFn: func(_ context.Context, _ *models.User) error {
			return nil
		},
	}

	svc := NewUserAdminService(store, newTestLogger())

	newEmail := "new@example.com"
	_, err := svc.Update(testutil.TestContext(), userID, &grpcservices.UserUpdateRequest{Email: &newEmail})
	assert.NoError(t, err) // must not panic
}

func TestUpdateUser_FieldMask_OmittedFieldsUnchanged(t *testing.T) {
	userID := uuid.New()
	existingUser := &models.User{
		ID:                   userID,
		Email:                "user@example.com",
		IsAdmin:              true,
		IsActive:             true,
		IsTenantManager:      true,
		IsBaseStationManager: false,
		IsEndpointManager:    true,
	}

	var updatedUser *models.User

	store := &mockUserStore{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*models.User, error) {
			if id == userID {
				u := *existingUser
				return &u, nil
			}
			return nil, interfaces.ErrRecordNotFound
		},
		getByEmailFn: func(_ context.Context, _ string) (*models.User, error) {
			return nil, interfaces.ErrRecordNotFound
		},
		updateFn: func(_ context.Context, user *models.User) error {
			updatedUser = user
			return nil
		},
	}

	svc := NewUserAdminService(store, newTestLogger())

	// Only update email — all bool fields should remain unchanged
	newEmail := "newemail@example.com"
	req := &grpcservices.UserUpdateRequest{
		Email: &newEmail,
	}

	_, err := svc.Update(testutil.TestContext(), userID, req)
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if updatedUser == nil {
		t.Fatal("store.Update was not called")
	}
	if updatedUser.Email != "newemail@example.com" {
		t.Errorf("Email = %q, want %q", updatedUser.Email, "newemail@example.com")
	}
	// All bools must remain as they were
	if !updatedUser.IsAdmin {
		t.Error("IsAdmin should remain true")
	}
	if !updatedUser.IsActive {
		t.Error("IsActive should remain true")
	}
	if !updatedUser.IsTenantManager {
		t.Error("IsTenantManager should remain true")
	}
	if updatedUser.IsBaseStationManager {
		t.Error("IsBaseStationManager should remain false")
	}
	if !updatedUser.IsEndpointManager {
		t.Error("IsEndpointManager should remain true")
	}
}
