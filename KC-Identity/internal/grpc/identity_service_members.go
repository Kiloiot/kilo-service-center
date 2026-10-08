package grpc

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/admin"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
)

// AddOrganizationUser adds a user to an organization.
func (s *IdentityService) AddOrganizationUser(ctx context.Context, req *pb.AddOrganizationUserRequest) (*pb.AddOrganizationUserResponse, error) {
	if s.membershipSvc == nil || s.orgSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	orgID, orgTenantID, err := s.resolveOrgAccess(ctx, req.OrgId)
	if err != nil {
		return nil, err
	}

	// Resolve user by ID or email (one-of validation)
	var userID uuid.UUID
	if req.UserId != "" && req.Email != "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserEmailRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserEmailRequired))
	}
	if req.UserId == "" && req.Email == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserEmailRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserEmailRequired))
	}
	if req.Email != "" {
		if s.adminUserSvc == nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
		}
		user, err := s.adminUserSvc.GetByEmail(ctx, req.Email)
		if err != nil {
			if errors.Is(err, admin.ErrUserNotFound) {
				return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserNotFound),
					grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserNotFound))
			}
			s.log.ErrorContext(ctx, LogEmailLookupFailed, logger.FieldEmail, req.Email, logger.FieldError, err)
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
		}
		userID = user.ID
	} else {
		userID, err = uuid.Parse(req.UserId)
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidUserIDFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidUserIDFormat))
		}
	}

	if req.Role == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRoleRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRoleRequired))
	}
	if !isValidOrganizationRole(req.Role) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidOrgRole),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidOrgRole))
	}

	if err := s.membershipSvc.AddUser(ctx, orgID, userID, req.Role); err != nil {
		s.log.ErrorContext(ctx, LogAddOrgUserFailed, logger.FieldOrgIDSnake, req.OrgId, logger.FieldUserIDSnake, req.UserId, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAddMemberFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAddMemberFailed))
	}

	if req.IsOrgAdmin || req.IsBaseStationAdmin || req.IsEndpointAdmin {
		if err := s.membershipSvc.UpdatePermissions(ctx, orgID, userID, req.IsOrgAdmin, req.IsBaseStationAdmin, req.IsEndpointAdmin); err != nil {
			s.log.ErrorContext(ctx, LogUpdateOrgMemberPermissionsFailed, logger.FieldOrgIDSnake, req.OrgId, logger.FieldUserIDSnake, req.UserId, logger.FieldError, err)
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateMemberFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateMemberFailed))
		}
	}

	// Fetch the newly created membership
	member, err := s.membershipSvc.GetMembership(ctx, orgID, userID)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetNewlyAddedMemberFailed, logger.FieldOrgIDSnake, req.OrgId, logger.FieldUserIDSnake, req.UserId, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMemberAddedRetrieveFail),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMemberAddedRetrieveFail))
	}

	s.audit.Record(ctx, membershipAuditEvent(models.EventTypeOrgMemberAdded, models.EventTitleOrgMemberAdded,
		fmt.Sprintf(models.EventDescriptionOrgMemberAdded, userID, orgID, req.Role),
		membershipChange{tenantID: orgTenantID, orgID: orgID, userID: userID, member: member}))
	return &pb.AddOrganizationUserResponse{
		Member: orgMemberToProto(member),
	}, nil
}

// GetOrganizationUser returns a membership.
func (s *IdentityService) GetOrganizationUser(ctx context.Context, req *pb.GetOrganizationUserRequest) (*pb.GetOrganizationUserResponse, error) {
	if s.membershipSvc == nil || s.orgSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.UserId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserIDRequired))
	}

	orgID, _, err := s.resolveOrgAccess(ctx, req.OrgId)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidUserIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidUserIDFormat))
	}

	member, err := s.membershipSvc.GetMembership(ctx, orgID, userID)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetOrgUserFailed, logger.FieldOrgIDSnake, req.OrgId, logger.FieldUserIDSnake, req.UserId, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMembershipNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMembershipNotFound))
	}

	return &pb.GetOrganizationUserResponse{
		Member: orgMemberToProto(member),
	}, nil
}

// Org user update field-name constants for FieldMask paths
const (
	orgUserFieldRole               = "role"
	orgUserFieldIsOrgAdmin         = "is_org_admin"
	orgUserFieldIsBaseStationAdmin = "is_base_station_admin"
	orgUserFieldIsEndpointAdmin    = "is_endpoint_admin"
)

// UpdateOrganizationUser updates a membership.
func (s *IdentityService) UpdateOrganizationUser(ctx context.Context, req *pb.UpdateOrganizationUserRequest) (*pb.UpdateOrganizationUserResponse, error) {
	if s.membershipSvc == nil || s.orgSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.UserId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserIDRequired))
	}

	mask := req.UpdateMask
	if mask == nil || len(mask.GetPaths()) == 0 {
		return nil, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateMaskRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateMaskRequired),
		)
	}

	// Validate role only when it is in the mask
	if grpcerrors.FieldInMask(mask, orgUserFieldRole) {
		if req.Role == "" {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRoleRequired),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRoleRequired))
		}
		if !isValidOrganizationRole(req.Role) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidOrgRole),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidOrgRole))
		}
	}

	orgID, orgTenantID, err := s.resolveOrgAccess(ctx, req.OrgId)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidUserIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidUserIDFormat))
	}

	// Update role only when in mask
	if grpcerrors.FieldInMask(mask, orgUserFieldRole) {
		if err := s.membershipSvc.UpdateRole(ctx, orgID, userID, req.Role); err != nil {
			s.log.ErrorContext(ctx, LogUpdateOrgUserRoleFailed, logger.FieldOrgIDSnake, req.OrgId, logger.FieldUserIDSnake, req.UserId, logger.FieldError, err)
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateMemberFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateMemberFailed))
		}
	}

	// Update permissions only when any permission field is in mask
	permissionsInMask := grpcerrors.FieldInMask(mask, orgUserFieldIsOrgAdmin) ||
		grpcerrors.FieldInMask(mask, orgUserFieldIsBaseStationAdmin) ||
		grpcerrors.FieldInMask(mask, orgUserFieldIsEndpointAdmin)
	if permissionsInMask {
		// Fetch current to preserve unmasked permission fields
		current, err := s.membershipSvc.GetMembership(ctx, orgID, userID)
		if err != nil {
			s.log.ErrorContext(ctx, LogGetCurrentMemberForPermissionUpdateFailed, logger.FieldOrgIDSnake, req.OrgId, logger.FieldUserIDSnake, req.UserId, logger.FieldError, err)
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMembershipNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMembershipNotFound))
		}

		isOrgAdmin := current.IsOrgAdmin
		isBSAdmin := current.IsBaseStationAdmin
		isEPAdmin := current.IsEndpointAdmin

		if grpcerrors.FieldInMask(mask, orgUserFieldIsOrgAdmin) {
			isOrgAdmin = req.IsOrgAdmin
		}
		if grpcerrors.FieldInMask(mask, orgUserFieldIsBaseStationAdmin) {
			isBSAdmin = req.IsBaseStationAdmin
		}
		if grpcerrors.FieldInMask(mask, orgUserFieldIsEndpointAdmin) {
			isEPAdmin = req.IsEndpointAdmin
		}

		if err := s.membershipSvc.UpdatePermissions(ctx, orgID, userID, isOrgAdmin, isBSAdmin, isEPAdmin); err != nil {
			s.log.ErrorContext(ctx, LogUpdateOrgMemberPermissionsFailed, logger.FieldOrgIDSnake, req.OrgId, logger.FieldUserIDSnake, req.UserId, logger.FieldError, err)
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateMemberFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateMemberFailed))
		}
	}

	// Fetch the updated membership
	member, err := s.membershipSvc.GetMembership(ctx, orgID, userID)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetUpdatedMemberFailed, logger.FieldOrgIDSnake, req.OrgId, logger.FieldUserIDSnake, req.UserId, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMemberUpdatedRetrieveFail),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMemberUpdatedRetrieveFail))
	}

	s.audit.Record(ctx, membershipAuditEvent(models.EventTypeOrgMemberUpdated, models.EventTitleOrgMemberUpdated,
		fmt.Sprintf(models.EventDescriptionOrgMemberUpdated, userID, orgID),
		membershipChange{tenantID: orgTenantID, orgID: orgID, userID: userID, member: member}))
	return &pb.UpdateOrganizationUserResponse{
		Member: orgMemberToProto(member),
	}, nil
}

// RemoveOrganizationUser removes a user from an organization.
func (s *IdentityService) RemoveOrganizationUser(ctx context.Context, req *pb.RemoveOrganizationUserRequest) (*pb.RemoveOrganizationUserResponse, error) {
	if s.membershipSvc == nil || s.orgSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.UserId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserIDRequired))
	}

	// Self-removal protection: caller cannot remove themselves
	callerID, err := grpcerrors.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	targetUserID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidUserIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidUserIDFormat))
	}
	if callerID == targetUserID {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCannotRemoveSelf),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCannotRemoveSelf))
	}

	orgID, orgTenantID, err := s.resolveOrgAccess(ctx, req.OrgId)
	if err != nil {
		return nil, err
	}

	if err := s.membershipSvc.RemoveUser(ctx, orgID, targetUserID); err != nil {
		if errors.Is(err, admin.ErrCannotRemoveLastOwner) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCannotRemoveLastOwner),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCannotRemoveLastOwner))
		}
		s.log.ErrorContext(ctx, LogRemoveOrgUserFailed, logger.FieldOrgIDSnake, req.OrgId, logger.FieldUserIDSnake, req.UserId, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRemoveMemberFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRemoveMemberFailed))
	}

	s.audit.Record(ctx, membershipAuditEvent(models.EventTypeOrgMemberRemoved, models.EventTitleOrgMemberRemoved,
		fmt.Sprintf(models.EventDescriptionOrgMemberRemoved, targetUserID, orgID),
		membershipChange{tenantID: orgTenantID, orgID: orgID, userID: targetUserID}))
	return &pb.RemoveOrganizationUserResponse{
		Success: true,
	}, nil
}

// ListOrganizationUsers returns the members of an organization.
func (s *IdentityService) ListOrganizationUsers(ctx context.Context, req *pb.ListOrganizationUsersRequest) (*pb.ListOrganizationUsersResponse, error) {
	if s.membershipSvc == nil || s.orgSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	orgID, _, err := s.resolveOrgAccess(ctx, req.OrgId)
	if err != nil {
		return nil, err
	}

	limit := grpcerrors.ClampPageSize(req.PageSize)
	offset := 0
	if req.PageToken != "" {
		if _, err := grpcerrors.ParsePaginationToken(req.PageToken, &offset); err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidPageToken))
		}
	}

	statusFilter, err := normalizeOrganizationMemberStatus(req.Status)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidArgument),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidArgument))
	}

	members, total, err := s.membershipSvc.ListMembers(ctx, orgID, statusFilter, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, LogListOrgUsersFailed, logger.FieldOrgIDSnake, req.OrgId, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListMembersFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListMembersFailed))
	}

	var pbMembers []*pb.OrganizationUser
	for _, m := range members {
		pbMembers = append(pbMembers, orgMemberToProto(m))
	}

	var nextToken string
	if offset+limit < int(total) {
		nextToken = grpcerrors.GeneratePaginationToken(offset + limit)
	}

	if total > math.MaxInt32 {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResultCountOverflow),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenResultCountOverflow))
	}

	totalCount := int32(total) //nolint:gosec // bounds checked above
	return &pb.ListOrganizationUsersResponse{
		Members:       pbMembers,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// ListUserOrganizations returns the organizations a user belongs to within the caller's tenant.
func (s *IdentityService) ListUserOrganizations(ctx context.Context, req *pb.ListUserOrganizationsRequest) (*pb.ListUserOrganizationsResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	if s.membershipSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.UserId == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserIDRequired))
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidUserIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidUserIDFormat))
	}

	tenantID, err := grpcerrors.GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	memberships, err := s.membershipSvc.ListUserOrganizations(ctx, userID, tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, LogListUserOrganizationsFailed, logger.FieldUserIDSnake, req.UserId, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListMembersFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListMembersFailed))
	}

	var pbMemberships []*pb.UserMembership
	for _, m := range memberships {
		pbMemberships = append(pbMemberships, &pb.UserMembership{
			OrgId:   m.OrgID.String(),
			OrgName: m.OrgName,
			Role:    m.Role,
			Status:  m.Status,
		})
	}

	return &pb.ListUserOrganizationsResponse{
		Memberships: pbMemberships,
	}, nil
}

// organizationRoles is the set of member roles the API accepts.
var organizationRoles = map[string]struct{}{
	models.OrganizationRoleOwner:  {},
	models.OrganizationRoleAdmin:  {},
	models.OrganizationRoleMember: {},
}

func isValidOrganizationRole(role string) bool {
	_, ok := organizationRoles[role]
	return ok
}

// memberStatusFilters maps every accepted member-status filter onto the
// repository filter it stands for; an empty value means no filter.
var memberStatusFilters = map[string]string{
	"":                                       "",
	models.OrganizationMemberStatusFilterAll: "",
	models.OrganizationMemberStatusFilterInactive: models.OrganizationMemberStatusFilterInactive,
	models.OrganizationMemberStatusActive:         models.OrganizationMemberStatusActive,
	models.OrganizationMemberStatusInvited:        models.OrganizationMemberStatusInvited,
	models.OrganizationMemberStatusRemoved:        models.OrganizationMemberStatusRemoved,
}

func normalizeOrganizationMemberStatus(status string) (string, error) {
	filter, ok := memberStatusFilters[status]
	if !ok {
		return "", errors.New(grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidArgument))
	}
	return filter, nil
}

// membershipChange names the membership an audit event describes: the
// organization, its tenant, the member, and the membership after the change
// (nil once it is gone).
type membershipChange struct {
	tenantID int64
	orgID    uuid.UUID
	userID   uuid.UUID
	member   *grpcservices.OrganizationMember
}

// membershipAuditEvent describes a membership change, filed under the tenant
// of the organization it happened in.
func membershipAuditEvent(eventType, title, description string, change membershipChange) audit.Event {
	details := map[string]any{
		auditKeyOrgID:  change.orgID.String(),
		auditKeyUserID: change.userID.String(),
	}
	if member := change.member; member != nil {
		details["role"] = member.Role
		details["isOrgAdmin"] = member.IsOrgAdmin
		details["isBaseStationAdmin"] = member.IsBaseStationAdmin
		details["isEndpointAdmin"] = member.IsEndpointAdmin
	}
	sourceID := change.orgID
	return audit.Event{
		TenantID:    change.tenantID,
		SourceID:    &sourceID,
		EventType:   eventType,
		Title:       title,
		Description: description,
		SourceName:  change.orgID.String(),
		Details:     details,
	}
}
