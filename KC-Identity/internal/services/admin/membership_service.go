package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/kilocenter/KC-Identity/internal/services/grpcservices"
)

// MembershipAdminService implements grpcservices.MembershipService.
type MembershipAdminService struct {
	memberStore OrganizationMemberStore
	userStore   UserAdminStore
	logger      logger.Logger
}

// NewMembershipAdminService creates a new membership admin service.
func NewMembershipAdminService(memberStore OrganizationMemberStore, userStore UserAdminStore, log logger.Logger) *MembershipAdminService {
	return &MembershipAdminService{
		memberStore: memberStore,
		userStore:   userStore,
		logger:      log,
	}
}

// AddUser adds a user to an organization.
func (s *MembershipAdminService) AddUser(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	// Verify user exists
	_, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("get user: %w", err)
	}

	// Check if already a member
	existing, err := s.memberStore.GetMember(ctx, orgID, userID)
	if err != nil && !errors.Is(err, interfaces.ErrRecordNotFound) {
		return fmt.Errorf("check membership: %w", err)
	}
	if existing != nil {
		return ErrMemberAlreadyExists
	}

	member := &models.OrganizationMember{
		OrgID:  orgID,
		UserID: userID,
		Role:   role,
		Status: models.OrganizationMemberStatusActive,
	}

	if err := s.memberStore.AddMember(ctx, member); err != nil {
		s.logger.ErrorContext(ctx, "failed to add member", "orgId", orgID, "userId", userID, "error", err)
		return fmt.Errorf("add member: %w", err)
	}

	s.logger.InfoContext(ctx, "member added", "orgId", orgID, "userId", userID, "role", role)
	return nil
}

// GetMembership retrieves a user's membership in an organization.
func (s *MembershipAdminService) GetMembership(ctx context.Context, orgID, userID uuid.UUID) (*grpcservices.OrganizationMember, error) {
	member, err := s.memberStore.GetMember(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrMemberNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get membership", "orgId", orgID, "userId", userID, "error", err)
		return nil, fmt.Errorf("get membership: %w", err)
	}

	return &grpcservices.OrganizationMember{
		UserID:             member.UserID,
		OrgID:              member.OrgID,
		Role:               member.Role,
		Status:             member.Status,
		IsOrgAdmin:         member.IsOrgAdmin,
		IsBaseStationAdmin: member.IsBaseStationAdmin,
		IsEndpointAdmin:    member.IsEndpointAdmin,
		JoinedAt:           member.CreatedAt,
		UpdatedAt:          member.UpdatedAt,
		UserEmail:          member.Email,
	}, nil
}

// UpdateRole updates a user's role in an organization.
func (s *MembershipAdminService) UpdateRole(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	_, err := s.memberStore.GetMember(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrMemberNotFound
		}
		return fmt.Errorf("get membership: %w", err)
	}

	if err := s.memberStore.UpdateMemberRole(ctx, orgID, userID, role); err != nil {
		s.logger.ErrorContext(ctx, "failed to update member role", "orgId", orgID, "userId", userID, "error", err)
		return fmt.Errorf("update role: %w", err)
	}

	s.logger.InfoContext(ctx, "member role updated", "orgId", orgID, "userId", userID, "role", role)
	return nil
}

// UpdatePermissions updates a user's permission flags in an organization.
func (s *MembershipAdminService) UpdatePermissions(ctx context.Context, orgID, userID uuid.UUID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin bool) error {
	_, err := s.memberStore.GetMember(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrMemberNotFound
		}
		return fmt.Errorf("get membership: %w", err)
	}

	if err := s.memberStore.UpdateMemberPermissions(ctx, orgID, userID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin); err != nil {
		s.logger.ErrorContext(ctx, "failed to update member permissions", "orgId", orgID, "userId", userID, "error", err)
		return fmt.Errorf("update permissions: %w", err)
	}

	s.logger.InfoContext(ctx, "member permissions updated", "orgId", orgID, "userId", userID)
	return nil
}

// RemoveUser removes a user from an organization.
// Returns ErrCannotRemoveLastOwner if removing the last active owner.
func (s *MembershipAdminService) RemoveUser(ctx context.Context, orgID, userID uuid.UUID) error {
	member, err := s.memberStore.GetMember(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrMemberNotFound
		}
		return fmt.Errorf("get membership: %w", err)
	}

	// Last-owner protection: prevent removing the last active owner
	if member.Role == models.OrganizationRoleOwner && member.Status == models.OrganizationMemberStatusActive {
		count, err := s.memberStore.CountMembersByRole(ctx, orgID, models.OrganizationRoleOwner)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to count owners", "orgId", orgID, "error", err)
			return fmt.Errorf("count owners: %w", err)
		}
		if count <= 1 {
			return ErrCannotRemoveLastOwner
		}
	}

	if err := s.memberStore.RemoveMember(ctx, orgID, userID); err != nil {
		s.logger.ErrorContext(ctx, "failed to remove member", "orgId", orgID, "userId", userID, "error", err)
		return fmt.Errorf("remove member: %w", err)
	}

	s.logger.InfoContext(ctx, "member removed", "orgId", orgID, "userId", userID)
	return nil
}

// ListMembers returns paginated members of an organization.
// Email is populated from the JOIN query (no N+1 lookups).
func (s *MembershipAdminService) ListMembers(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*grpcservices.OrganizationMember, int64, error) {
	members, total, err := s.memberStore.ListMembers(ctx, orgID, status, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list members", "orgId", orgID, "error", err)
		return nil, 0, fmt.Errorf("list members: %w", err)
	}

	result := make([]*grpcservices.OrganizationMember, len(members))
	for i, m := range members {
		result[i] = &grpcservices.OrganizationMember{
			UserID:             m.UserID,
			OrgID:              m.OrgID,
			Role:               m.Role,
			Status:             m.Status,
			IsOrgAdmin:         m.IsOrgAdmin,
			IsBaseStationAdmin: m.IsBaseStationAdmin,
			IsEndpointAdmin:    m.IsEndpointAdmin,
			JoinedAt:           m.CreatedAt,
			UpdatedAt:          m.UpdatedAt,
			UserEmail:          m.Email,
		}
	}

	return result, total, nil
}

// ListUserOrganizations returns organizations a user belongs to within a tenant.
func (s *MembershipAdminService) ListUserOrganizations(ctx context.Context, userID uuid.UUID, tenantID int64) ([]grpcservices.OrganizationMembership, error) {
	memberships, err := s.memberStore.ListUserMembershipsByTenant(ctx, userID, tenantID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list user organizations", "userId", userID, "error", err)
		return nil, fmt.Errorf("list user organizations: %w", err)
	}

	result := make([]grpcservices.OrganizationMembership, len(memberships))
	for i, m := range memberships {
		result[i] = grpcservices.OrganizationMembership{
			OrgID:   m.OrgID,
			OrgName: m.OrgName,
			Role:    m.Role,
			Status:  m.Status,
		}
	}
	return result, nil
}

// Ensure MembershipAdminService implements grpcservices.MembershipService
var _ grpcservices.MembershipService = (*MembershipAdminService)(nil)
