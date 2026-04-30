package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/testutil"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
)

// mockMemberStore implements OrganizationMemberStore for testing.
type mockMemberStore struct {
	addMemberFn                   func(ctx context.Context, member *models.OrganizationMember) error
	getMemberFn                   func(ctx context.Context, orgID, userID uuid.UUID) (*models.OrganizationMemberWithEmail, error)
	updateMemberRoleFn            func(ctx context.Context, orgID, userID uuid.UUID, role string) error
	updatePermissionsFn           func(ctx context.Context, orgID, userID uuid.UUID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin bool) error
	removeMemberFn                func(ctx context.Context, orgID, userID uuid.UUID) error
	listMembersFn                 func(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*models.OrganizationMemberWithEmail, int64, error)
	countMembersByRoleFn          func(ctx context.Context, orgID uuid.UUID, role string) (int64, error)
	listUserMembershipsByTenantFn func(ctx context.Context, userID uuid.UUID, tenantID int64) ([]*models.OrganizationMembershipWithOrg, error)
}

func (m *mockMemberStore) AddMember(ctx context.Context, member *models.OrganizationMember) error {
	return m.addMemberFn(ctx, member)
}
func (m *mockMemberStore) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*models.OrganizationMemberWithEmail, error) {
	return m.getMemberFn(ctx, orgID, userID)
}
func (m *mockMemberStore) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	return m.updateMemberRoleFn(ctx, orgID, userID, role)
}
func (m *mockMemberStore) UpdateMemberPermissions(ctx context.Context, orgID, userID uuid.UUID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin bool) error {
	return m.updatePermissionsFn(ctx, orgID, userID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin)
}
func (m *mockMemberStore) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error {
	return m.removeMemberFn(ctx, orgID, userID)
}
func (m *mockMemberStore) ListMembers(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*models.OrganizationMemberWithEmail, int64, error) {
	return m.listMembersFn(ctx, orgID, status, limit, offset)
}
func (m *mockMemberStore) CountMembersByRole(ctx context.Context, orgID uuid.UUID, role string) (int64, error) {
	return m.countMembersByRoleFn(ctx, orgID, role)
}
func (m *mockMemberStore) ListUserMembershipsByTenant(ctx context.Context, userID uuid.UUID, tenantID int64) ([]*models.OrganizationMembershipWithOrg, error) {
	if m.listUserMembershipsByTenantFn != nil {
		return m.listUserMembershipsByTenantFn(ctx, userID, tenantID)
	}
	return nil, nil
}

func TestRemoveUser_LastOwnerProtection(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	memberStore := &mockMemberStore{
		getMemberFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.OrganizationMemberWithEmail, error) {
			return &models.OrganizationMemberWithEmail{
				OrgID:     orgID,
				UserID:    userID,
				Role:      models.OrganizationRoleOwner,
				Status:    models.OrganizationMemberStatusActive,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
		countMembersByRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (int64, error) {
			return 1, nil // Only one owner
		},
	}

	userStore := &mockUserStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, Email: "owner@example.com"}, nil
		},
	}

	svc := NewMembershipAdminService(memberStore, userStore, newTestLogger())

	err := svc.RemoveUser(testutil.TestContext(), orgID, userID)
	if !errors.Is(err, ErrCannotRemoveLastOwner) {
		t.Errorf("RemoveUser() error = %v, want ErrCannotRemoveLastOwner", err)
	}
}

func TestRemoveUser_NonLastOwner(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	removeCalled := false

	memberStore := &mockMemberStore{
		getMemberFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.OrganizationMemberWithEmail, error) {
			return &models.OrganizationMemberWithEmail{
				OrgID:     orgID,
				UserID:    userID,
				Role:      models.OrganizationRoleOwner,
				Status:    models.OrganizationMemberStatusActive,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
		countMembersByRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (int64, error) {
			return 2, nil // Two owners — removal is allowed
		},
		removeMemberFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
			removeCalled = true
			return nil
		},
	}

	userStore := &mockUserStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, Email: "owner@example.com"}, nil
		},
	}

	svc := NewMembershipAdminService(memberStore, userStore, newTestLogger())

	err := svc.RemoveUser(testutil.TestContext(), orgID, userID)
	if err != nil {
		t.Fatalf("RemoveUser() returned error: %v", err)
	}
	if !removeCalled {
		t.Error("RemoveMember was not called")
	}
}

func TestRemoveUser_InactiveOwnerDoesNotCount(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	// The member being removed is an inactive owner — should skip last-owner check
	memberStore := &mockMemberStore{
		getMemberFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.OrganizationMemberWithEmail, error) {
			return &models.OrganizationMemberWithEmail{
				OrgID:     orgID,
				UserID:    userID,
				Role:      models.OrganizationRoleOwner,
				Status:    "removed", // Inactive — not "active"
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
		removeMemberFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
			return nil
		},
		// CountMembersByRole should NOT be called because status != active
		countMembersByRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (int64, error) {
			t.Error("CountMembersByRole should not be called for inactive owner")
			return 0, nil
		},
	}

	userStore := &mockUserStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, Email: "old-owner@example.com"}, nil
		},
	}

	svc := NewMembershipAdminService(memberStore, userStore, newTestLogger())

	err := svc.RemoveUser(testutil.TestContext(), orgID, userID)
	if err != nil {
		t.Fatalf("RemoveUser() returned error: %v", err)
	}
}

func TestRemoveUser_NonOwnerSkipsCheck(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	removeCalled := false

	memberStore := &mockMemberStore{
		getMemberFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.OrganizationMemberWithEmail, error) {
			return &models.OrganizationMemberWithEmail{
				OrgID:     orgID,
				UserID:    userID,
				Role:      "member", // Not an owner
				Status:    models.OrganizationMemberStatusActive,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
		// CountMembersByRole should NOT be called for non-owners
		countMembersByRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (int64, error) {
			t.Error("CountMembersByRole should not be called for non-owner")
			return 0, nil
		},
		removeMemberFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
			removeCalled = true
			return nil
		},
	}

	userStore := &mockUserStore{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, Email: "member@example.com"}, nil
		},
	}

	svc := NewMembershipAdminService(memberStore, userStore, newTestLogger())

	err := svc.RemoveUser(testutil.TestContext(), orgID, userID)
	if err != nil {
		t.Fatalf("RemoveUser() returned error: %v", err)
	}
	if !removeCalled {
		t.Error("RemoveMember was not called")
	}
}

func TestRemoveUser_MemberNotFound(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	memberStore := &mockMemberStore{
		getMemberFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.OrganizationMemberWithEmail, error) {
			return nil, interfaces.ErrRecordNotFound
		},
	}

	userStore := &mockUserStore{}

	svc := NewMembershipAdminService(memberStore, userStore, newTestLogger())

	err := svc.RemoveUser(testutil.TestContext(), orgID, userID)
	if !errors.Is(err, ErrMemberNotFound) {
		t.Errorf("RemoveUser() error = %v, want ErrMemberNotFound", err)
	}
}

func TestListMembers_EmailPopulated(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	memberStore := &mockMemberStore{
		listMembersFn: func(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*models.OrganizationMemberWithEmail, int64, error) {
			return []*models.OrganizationMemberWithEmail{
				{
					OrgID:     orgID,
					UserID:    userID,
					Email:     "user@example.com",
					Role:      "member",
					Status:    models.OrganizationMemberStatusActive,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			}, 1, nil
		},
	}

	userStore := &mockUserStore{}
	svc := NewMembershipAdminService(memberStore, userStore, newTestLogger())

	members, total, err := svc.ListMembers(testutil.TestContext(), orgID, "", 20, 0)
	if err != nil {
		t.Fatalf("ListMembers() returned error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(members) != 1 {
		t.Fatalf("len(members) = %d, want 1", len(members))
	}
	if members[0].UserEmail != "user@example.com" {
		t.Errorf("UserEmail = %q, want %q", members[0].UserEmail, "user@example.com")
	}
}
