package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	"github.com/google/uuid"
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
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("%s: %w", errOpGetUser, err)
	}

	// Check if already a member
	existing, err := s.memberStore.GetMember(ctx, orgID, userID)
	if err != nil && !errors.Is(err, storage.ErrRecordNotFound) {
		return fmt.Errorf("%s: %w", errOpCheckMembership, err)
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
		s.logger.ErrorContext(ctx, LogMemberAddFailed, logger.FieldOrgIDCamel, orgID, logger.FieldUserIDCamel, userID, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpAddMember, err)
	}

	s.logger.InfoContext(ctx, LogMemberAdded, logger.FieldOrgIDCamel, orgID, logger.FieldUserIDCamel, userID, logger.FieldRole, role)
	return nil
}

// GetMembership retrieves a user's membership in an organization.
func (s *MembershipAdminService) GetMembership(ctx context.Context, orgID, userID uuid.UUID) (*grpcservices.OrganizationMember, error) {
	member, err := s.memberStore.GetMember(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrMemberNotFound
		}
		s.logger.ErrorContext(ctx, LogMembershipGetFailed, logger.FieldOrgIDCamel, orgID, logger.FieldUserIDCamel, userID, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpGetMembership, err)
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
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrMemberNotFound
		}
		return fmt.Errorf("%s: %w", errOpGetMembership, err)
	}

	if err := s.memberStore.UpdateMemberRole(ctx, orgID, userID, role); err != nil {
		s.logger.ErrorContext(ctx, LogMemberRoleUpdateFailed, logger.FieldOrgIDCamel, orgID, logger.FieldUserIDCamel, userID, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpUpdateRole, err)
	}

	s.logger.InfoContext(ctx, LogMemberRoleUpdated, logger.FieldOrgIDCamel, orgID, logger.FieldUserIDCamel, userID, logger.FieldRole, role)
	return nil
}

// UpdatePermissions updates a user's permission flags in an organization.
func (s *MembershipAdminService) UpdatePermissions(ctx context.Context, orgID, userID uuid.UUID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin bool) error {
	_, err := s.memberStore.GetMember(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrMemberNotFound
		}
		return fmt.Errorf("%s: %w", errOpGetMembership, err)
	}

	if err := s.memberStore.UpdateMemberPermissions(ctx, orgID, userID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin); err != nil {
		s.logger.ErrorContext(ctx, LogMemberPermissionsUpdateFailed, logger.FieldOrgIDCamel, orgID, logger.FieldUserIDCamel, userID, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpUpdatePermissions, err)
	}

	s.logger.InfoContext(ctx, LogMemberPermissionsUpdated, logger.FieldOrgIDCamel, orgID, logger.FieldUserIDCamel, userID)
	return nil
}

// RemoveUser removes a user from an organization.
// Returns ErrCannotRemoveLastOwner if removing the last active owner.
func (s *MembershipAdminService) RemoveUser(ctx context.Context, orgID, userID uuid.UUID) error {
	member, err := s.memberStore.GetMember(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrMemberNotFound
		}
		return fmt.Errorf("%s: %w", errOpGetMembership, err)
	}

	// Last-owner protection: prevent removing the last active owner
	if member.Role == models.OrganizationRoleOwner && member.Status == models.OrganizationMemberStatusActive {
		count, err := s.memberStore.CountMembersByRole(ctx, orgID, models.OrganizationRoleOwner)
		if err != nil {
			s.logger.ErrorContext(ctx, LogMemberOwnerCountFailed, logger.FieldOrgIDCamel, orgID, logger.FieldError, err)
			return fmt.Errorf("%s: %w", errOpCountOwners, err)
		}
		if count <= 1 {
			return ErrCannotRemoveLastOwner
		}
	}

	if err := s.memberStore.RemoveMember(ctx, orgID, userID); err != nil {
		s.logger.ErrorContext(ctx, LogMemberRemoveFailed, logger.FieldOrgIDCamel, orgID, logger.FieldUserIDCamel, userID, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpRemoveMember, err)
	}

	s.logger.InfoContext(ctx, LogMemberRemoved, logger.FieldOrgIDCamel, orgID, logger.FieldUserIDCamel, userID)
	return nil
}

// ListMembers returns paginated members of an organization.
// Email is populated from the JOIN query (no N+1 lookups).
func (s *MembershipAdminService) ListMembers(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*grpcservices.OrganizationMember, int64, error) {
	members, total, err := s.memberStore.ListMembers(ctx, orgID, status, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, LogMemberListFailed, logger.FieldOrgIDCamel, orgID, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errOpListMembers, err)
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
		s.logger.ErrorContext(ctx, LogMemberUserOrgListFailed, logger.FieldUserIDCamel, userID, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpListUserOrganizations, err)
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
