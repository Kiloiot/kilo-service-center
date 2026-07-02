// Package grpc provides gRPC service implementations.
// Organization, Membership, and API Key RPC handlers.
package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/admin"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// validateOrgAccess validates the request org ID and confirms it belongs to the caller's tenant.
// Returns the parsed org UUID and tenant ID, or a gRPC status error.
func (s *IdentityService) validateOrgAccess(ctx context.Context, reqOrgID string) (uuid.UUID, int64, error) {
	if reqOrgID == "" {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDRequired))
	}
	orgID, err := uuid.Parse(reqOrgID)
	if err != nil {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidOrgIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidOrgIDFormat))
	}
	if s.orgSvc == nil {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}
	tenantID, err := grpcerrors.GetTenantFromContext(ctx)
	if err != nil {
		return uuid.Nil, 0, err
	}
	if _, err := s.orgSvc.GetByID(ctx, orgID, tenantID); err != nil {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgNotFound))
	}
	return orgID, tenantID, nil
}

// validateOrgAccessUnscoped validates org ID format and existence without tenant scoping.
func (s *IdentityService) validateOrgAccessUnscoped(ctx context.Context, reqOrgID string) (uuid.UUID, int64, error) {
	if reqOrgID == "" {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDRequired))
	}
	orgID, err := uuid.Parse(reqOrgID)
	if err != nil {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidOrgIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidOrgIDFormat))
	}
	if s.orgSvc == nil {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}
	org, err := s.orgSvc.GetByIDUnscoped(ctx, orgID)
	if err != nil {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgNotFound))
	}
	return orgID, org.TenantID, nil
}

// resolveOrgAccess uses server-admin unscoped access or org-admin scoped access.
func (s *IdentityService) resolveOrgAccess(ctx context.Context, reqOrgID string) (uuid.UUID, int64, error) {
	if reqOrgID == "" {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDRequired))
	}
	orgID, err := uuid.Parse(reqOrgID)
	if err != nil {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidOrgIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidOrgIDFormat))
	}

	isServerAdmin, err := s.requireServerOrOrgAdmin(ctx, orgID)
	if err != nil {
		return uuid.Nil, 0, err
	}

	if isServerAdmin {
		return s.validateOrgAccessUnscoped(ctx, reqOrgID)
	}
	return s.validateOrgAccess(ctx, reqOrgID)
}

// Organization handlers

// CreateOrganization creates a new organization.
func (s *IdentityService) CreateOrganization(ctx context.Context, req *pb.CreateOrganizationRequest) (*pb.CreateOrganizationResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	if s.orgSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.Name == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNameRequired))
	}

	// TenantID 0 makes the service provision a dedicated tenant per organization,
	// so base stations (scoped only by tenant_id) stay isolated between organizations.
	createReq := &grpcservices.OrganizationCreateRequest{
		Name:        req.Name,
		Description: req.Description,
		TenantID:    0,
	}

	org, err := s.orgSvc.Create(ctx, createReq)
	if err != nil {
		s.log.ErrorContext(ctx, "create organization failed", "name", req.Name, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateOrgFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateOrgFailed))
	}

	// Emit CRUD event
	if s.eventWriter != nil {
		orgIDStr := org.OrgID.String()
		sourceID := org.OrgID
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"orgId": orgIDStr,
			"name":  org.Name,
		})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(org.TenantID, 10),
			EventType:   models.EventTypeOrgCreated,
			Category:    models.EventCategoryAudit,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleOrgCreated,
			Description: fmt.Sprintf(models.EventDescriptionOrgCreated, org.Name),
			SourceType:  models.SourceTypeAPI,
			SourceID:    &sourceID,
			SourceName:  org.Name,
			Details:     detailsJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return &pb.CreateOrganizationResponse{
		Organization: orgModelToProto(org),
	}, nil
}

// GetOrganization returns an organization by ID.
func (s *IdentityService) GetOrganization(ctx context.Context, req *pb.GetOrganizationRequest) (*pb.GetOrganizationResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	orgID, tenantID, err := s.validateOrgAccessUnscoped(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	org, err := s.orgSvc.GetByID(ctx, orgID, tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, "get organization failed", "org_id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgNotFound))
	}

	return &pb.GetOrganizationResponse{
		Organization: orgModelToProto(org),
	}, nil
}

// UpdateOrganization updates an organization.
func (s *IdentityService) UpdateOrganization(ctx context.Context, req *pb.UpdateOrganizationRequest) (*pb.UpdateOrganizationResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	orgID, tenantID, err := s.validateOrgAccessUnscoped(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	updateReq := &grpcservices.OrganizationUpdateRequest{}
	if req.Name != "" {
		updateReq.Name = &req.Name
	}
	if req.Description != "" {
		updateReq.Description = &req.Description
	}

	org, err := s.orgSvc.Update(ctx, orgID, tenantID, updateReq)
	if err != nil {
		s.log.ErrorContext(ctx, "update organization failed", "org_id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateOrgFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateOrgFailed))
	}

	// Emit CRUD event
	if s.eventWriter != nil {
		orgIDStr := org.OrgID.String()
		sourceID := org.OrgID
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"orgId": orgIDStr,
			"name":  org.Name,
		})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(tenantID, 10),
			EventType:   models.EventTypeOrgUpdated,
			Category:    models.EventCategoryAudit,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleOrgUpdated,
			Description: fmt.Sprintf(models.EventDescriptionOrgUpdated, org.Name),
			SourceType:  models.SourceTypeAPI,
			SourceID:    &sourceID,
			SourceName:  org.Name,
			Details:     detailsJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return &pb.UpdateOrganizationResponse{
		Organization: orgModelToProto(org),
	}, nil
}

// DeleteOrganization deletes an organization.
func (s *IdentityService) DeleteOrganization(ctx context.Context, req *pb.DeleteOrganizationRequest) (*pb.DeleteOrganizationResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	orgID, tenantID, err := s.validateOrgAccessUnscoped(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if err := s.orgSvc.Delete(ctx, orgID, tenantID); err != nil {
		s.log.ErrorContext(ctx, "delete organization failed", "org_id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteOrgFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteOrgFailed))
	}

	// Emit CRUD event
	if s.eventWriter != nil {
		sourceID := orgID
		detailsJSON, _ := json.Marshal(map[string]interface{}{"orgId": req.Id})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(tenantID, 10),
			EventType:   models.EventTypeOrgDeleted,
			Category:    models.EventCategoryAudit,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleOrgDeleted,
			Description: fmt.Sprintf(models.EventDescriptionOrgDeleted, req.Id),
			SourceType:  models.SourceTypeAPI,
			SourceID:    &sourceID,
			Details:     detailsJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return &pb.DeleteOrganizationResponse{Success: true}, nil
}

// ListOrganizations returns a list of organizations.
func (s *IdentityService) ListOrganizations(ctx context.Context, req *pb.ListOrganizationsRequest) (*pb.ListOrganizationsResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	if s.orgSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	limit := grpcerrors.ClampPageSize(req.PageSize)
	offset := 0
	if req.PageToken != "" {
		if _, err := grpcerrors.ParsePaginationToken(req.PageToken, &offset); err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidPageToken))
		}
	}

	orgs, total, err := s.orgSvc.ListAll(ctx, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, "list organizations failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListOrgsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListOrgsFailed))
	}

	var pbOrgs []*pb.Organization
	for _, o := range orgs {
		pbOrgs = append(pbOrgs, orgModelToProto(o))
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
	return &pb.ListOrganizationsResponse{
		Organizations: pbOrgs,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// Membership handlers

// AddOrganizationUser adds a user to an organization.
func (s *IdentityService) AddOrganizationUser(ctx context.Context, req *pb.AddOrganizationUserRequest) (*pb.AddOrganizationUserResponse, error) {
	if s.membershipSvc == nil || s.orgSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	orgID, _, err := s.resolveOrgAccess(ctx, req.OrgId)
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
			s.log.ErrorContext(ctx, "email lookup failed", "email", req.Email, "error", err)
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
		s.log.ErrorContext(ctx, "add org user failed", "org_id", req.OrgId, "user_id", req.UserId, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAddMemberFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAddMemberFailed))
	}

	if req.IsOrgAdmin || req.IsBaseStationAdmin || req.IsEndpointAdmin {
		if err := s.membershipSvc.UpdatePermissions(ctx, orgID, userID, req.IsOrgAdmin, req.IsBaseStationAdmin, req.IsEndpointAdmin); err != nil {
			s.log.ErrorContext(ctx, "update org member permissions failed", "org_id", req.OrgId, "user_id", req.UserId, "error", err)
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateMemberFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateMemberFailed))
		}
	}

	// Fetch the newly created membership
	member, err := s.membershipSvc.GetMembership(ctx, orgID, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "get newly added member failed", "org_id", req.OrgId, "user_id", req.UserId, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMemberAddedRetrieveFail),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMemberAddedRetrieveFail))
	}

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
		s.log.ErrorContext(ctx, "get org user failed", "org_id", req.OrgId, "user_id", req.UserId, "error", err)
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
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateMaskRequired))
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

	orgID, _, err := s.resolveOrgAccess(ctx, req.OrgId)
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
			s.log.ErrorContext(ctx, "update org user role failed", "org_id", req.OrgId, "user_id", req.UserId, "error", err)
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
			s.log.ErrorContext(ctx, "get current member for permission update failed", "org_id", req.OrgId, "user_id", req.UserId, "error", err)
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
			s.log.ErrorContext(ctx, "update org member permissions failed", "org_id", req.OrgId, "user_id", req.UserId, "error", err)
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateMemberFailed),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateMemberFailed))
		}
	}

	// Fetch the updated membership
	member, err := s.membershipSvc.GetMembership(ctx, orgID, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "get updated member failed", "org_id", req.OrgId, "user_id", req.UserId, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMemberUpdatedRetrieveFail),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMemberUpdatedRetrieveFail))
	}

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

	orgID, _, err := s.resolveOrgAccess(ctx, req.OrgId)
	if err != nil {
		return nil, err
	}

	if err := s.membershipSvc.RemoveUser(ctx, orgID, targetUserID); err != nil {
		if errors.Is(err, admin.ErrCannotRemoveLastOwner) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCannotRemoveLastOwner),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCannotRemoveLastOwner))
		}
		s.log.ErrorContext(ctx, "remove org user failed", "org_id", req.OrgId, "user_id", req.UserId, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRemoveMemberFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRemoveMemberFailed))
	}

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
		s.log.ErrorContext(ctx, "list org users failed", "org_id", req.OrgId, "error", err)
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
		s.log.ErrorContext(ctx, "list user organizations failed", "user_id", req.UserId, "error", err)
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

// API Key handlers

// CreateApiKey creates a new API key.
// Tenant-isolated API key creation with ownership enforcement.
//
//revive:disable-next-line:var-naming
func (s *IdentityService) CreateApiKey(ctx context.Context, req *pb.CreateApiKeyRequest) (*pb.CreateApiKeyResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	if s.apiKeySvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	// Prefer the request org: the context org is the caller's own, not the target.
	var orgID uuid.UUID
	if req.OrganizationId != "" {
		parsed, parseErr := uuid.Parse(req.OrganizationId)
		if parseErr != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDHeaderInvalid),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDHeaderInvalid))
		}
		orgID = parsed
	} else {
		ctxOrg, ctxErr := pkgcontext.GetOrganizationID(ctx)
		if ctxErr != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDHeaderRequired),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDHeaderRequired))
		}
		orgID = ctxOrg
	}

	if req.Name == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNameRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNameRequired))
	}

	// Validate key_type using model constants.
	if req.KeyType != models.KeyTypeUser && req.KeyType != models.KeyTypeServiceAccount {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidApiKeyType),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidApiKeyType))
	}

	// User keys are tied to the authenticated user; service account keys have nil UserID.
	var userID *uuid.UUID
	if req.KeyType == models.KeyTypeUser {
		uid, err := grpcerrors.GetUserFromContext(ctx)
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMissingUserCtx),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMissingUserCtx))
		}
		userID = &uid
	}

	// Scope the key to the organization's own tenant, not the caller context: a base
	// station created with this key inherits the key's tenant, so per-org tenants keep them isolated.
	if s.orgSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}
	org, err := s.orgSvc.GetByIDUnscoped(ctx, orgID)
	if err != nil {
		s.log.ErrorContext(ctx, "resolve tenant for API key failed", "orgId", orgID.String(), "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateApiKeyFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateApiKeyFailed))
	}
	tenantID := org.TenantID

	createReq := &grpcservices.APIKeyCreateRequest{
		TenantID: tenantID,
		OrgID:    orgID,
		UserID:   userID,
		Name:     req.Name,
		KeyType:  req.KeyType,
	}

	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		createReq.ExpiresAt = &t
	}

	resp, err := s.apiKeySvc.Create(ctx, createReq)
	if err != nil {
		s.log.ErrorContext(ctx, "create API key failed", "name", req.Name, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateApiKeyFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateApiKeyFailed))
	}

	// Emit audit event for API key creation.
	if s.eventWriter != nil {
		detailsMap := map[string]interface{}{
			"keyId":     resp.APIKey.ID.String(),
			"keyPrefix": resp.APIKey.KeyPrefix,
			"keyType":   req.KeyType,
			"orgId":     orgID.String(),
		}
		var auditUserID string
		if userID != nil {
			detailsMap["userId"] = userID.String()
			auditUserID = userID.String()
		}
		detailsJSON, _ := json.Marshal(detailsMap)
		sourceID := orgID
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(tenantID, 10),
			EventType:   models.EventTypeAPIKeyCreated,
			Category:    models.EventCategoryAudit,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleAPIKeyCreated,
			Description: fmt.Sprintf(models.EventDescriptionAPIKeyCreatedFmt, req.Name, orgID.String()),
			SourceType:  models.SourceTypeAPI,
			SourceID:    &sourceID,
			SourceName:  req.Name,
			UserID:      auditUserID,
			Details:     detailsJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return &pb.CreateApiKeyResponse{
		RawKey: resp.Key,
		ApiKey: apiKeyModelToProto(resp.APIKey),
	}, nil
}

// GetApiKey returns an API key by ID.
// Tenant-isolated API key retrieval with ownership check.
//
//revive:disable-next-line:var-naming
func (s *IdentityService) GetApiKey(ctx context.Context, req *pb.GetApiKeyRequest) (*pb.GetApiKeyResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	if s.apiKeySvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	// Extract org context for tenant isolation.
	orgID, err := pkgcontext.GetOrganizationID(ctx)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDHeaderRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDHeaderRequired))
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	keyID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidAPIKeyIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidAPIKeyIDFormat))
	}

	// Use org-scoped query to enforce ownership.
	key, err := s.apiKeySvc.GetByIDAndOrg(ctx, keyID, orgID)
	if err != nil {
		s.log.ErrorContext(ctx, "get API key failed", "key_id", req.Id, "org_id", orgID, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenApiKeyNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenApiKeyNotFound))
	}

	return &pb.GetApiKeyResponse{
		ApiKey: apiKeyModelToProto(key),
	}, nil
}

// DeleteApiKey deletes an API key.
// Tenant-isolated API key deletion with ownership check.
//
//revive:disable-next-line:var-naming
func (s *IdentityService) DeleteApiKey(ctx context.Context, req *pb.DeleteApiKeyRequest) (*pb.DeleteApiKeyResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	if s.apiKeySvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	// Extract org context for tenant isolation.
	orgID, err := pkgcontext.GetOrganizationID(ctx)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDHeaderRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDHeaderRequired))
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	keyID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidAPIKeyIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidAPIKeyIDFormat))
	}

	// Prefetch key metadata before deletion for audit event.
	var keyName string
	existingKey, prefetchErr := s.apiKeySvc.GetByIDAndOrg(ctx, keyID, orgID)
	if prefetchErr == nil && existingKey != nil {
		keyName = existingKey.Name
	}

	// Use org-scoped deletion to enforce ownership.
	if err := s.apiKeySvc.DeleteByIDAndOrg(ctx, keyID, orgID); err != nil {
		s.log.ErrorContext(ctx, "delete API key failed", "key_id", req.Id, "org_id", orgID, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteApiKeyFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteApiKeyFailed))
	}

	// Emit audit event for API key deletion.
	if s.eventWriter != nil {
		tenantID, _ := grpcerrors.GetTenantFromContext(ctx)
		if keyName == "" {
			keyName = req.Id
		}
		sourceID := orgID
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"keyId": req.Id,
			"orgId": orgID.String(),
		})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(tenantID, 10),
			EventType:   models.EventTypeAPIKeyDeleted,
			Category:    models.EventCategoryAudit,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleAPIKeyDeleted,
			Description: fmt.Sprintf(models.EventDescriptionAPIKeyDeletedFmt, keyName),
			SourceType:  models.SourceTypeAPI,
			SourceID:    &sourceID,
			SourceName:  keyName,
			Details:     detailsJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return &pb.DeleteApiKeyResponse{
		Success: true,
	}, nil
}

// ListApiKeys returns a list of API keys.
//
//revive:disable-next-line:var-naming
func (s *IdentityService) ListApiKeys(ctx context.Context, req *pb.ListApiKeysRequest) (*pb.ListApiKeysResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}
	if s.apiKeySvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := grpcerrors.GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	orgID, err := pkgcontext.GetOrganizationID(ctx)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDHeaderRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDHeaderRequired))
	}

	limit := grpcerrors.ClampPageSize(req.PageSize)
	offset := 0
	if req.PageToken != "" {
		if _, err := grpcerrors.ParsePaginationToken(req.PageToken, &offset); err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidPageToken))
		}
	}

	var userID *uuid.UUID
	if req.UserId != "" {
		id, err := uuid.Parse(req.UserId)
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidUserIDFormat),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidUserIDFormat))
		}
		userID = &id
	}

	keys, total, err := s.apiKeySvc.List(ctx, tenantID, orgID, userID, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, "list API keys failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListApiKeysFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListApiKeysFailed))
	}

	var pbKeys []*pb.ApiKey
	for _, k := range keys {
		pbKeys = append(pbKeys, apiKeyModelToProto(k))
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
	return &pb.ListApiKeysResponse{
		ApiKeys:       pbKeys,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// Helper functions

func orgModelToProto(o *models.Organization) *pb.Organization {
	if o == nil {
		return nil
	}
	pbOrg := &pb.Organization{
		Id:                  o.OrgID.String(),
		TenantId:            o.TenantID,
		Name:                o.Name,
		State:               o.State,
		CanHaveBaseStations: o.CanHaveBaseStations,
		CreatedAt:           timestamppb.New(o.CreatedAt),
		UpdatedAt:           timestamppb.New(o.UpdatedAt),
	}
	if o.Description != nil {
		pbOrg.Description = *o.Description
	}
	if o.ExternalID != nil {
		pbOrg.ExternalId = *o.ExternalID
	}
	if o.MaxBaseStationCount != nil {
		if *o.MaxBaseStationCount > math.MaxInt32 {
			pbOrg.MaxBaseStationCount = math.MaxInt32
		} else if *o.MaxBaseStationCount > 0 {
			pbOrg.MaxBaseStationCount = int32(*o.MaxBaseStationCount)
		}
	}
	if o.MaxEndpointCount != nil {
		if *o.MaxEndpointCount > math.MaxInt32 {
			pbOrg.MaxEndpointCount = math.MaxInt32
		} else if *o.MaxEndpointCount > 0 {
			pbOrg.MaxEndpointCount = int32(*o.MaxEndpointCount)
		}
	}
	if o.Tags != nil {
		pbOrg.Tags = map[string]string(o.Tags)
	}
	return pbOrg
}

func apiKeyModelToProto(k *models.APIKey) *pb.ApiKey {
	if k == nil {
		return nil
	}
	pbKey := &pb.ApiKey{
		Id:        k.ID.String(),
		OrgId:     k.OrgID.String(),
		Name:      k.Name,
		KeyPrefix: k.KeyPrefix,
		KeyType:   k.KeyType,
		IsActive:  k.IsActive,
		CreatedAt: timestamppb.New(k.CreatedAt),
	}
	if k.UserID != nil {
		pbKey.UserId = k.UserID.String()
	}
	if k.ExpiresAt != nil {
		pbKey.ExpiresAt = timestamppb.New(*k.ExpiresAt)
	}
	if k.LastUsedAt != nil {
		pbKey.LastUsedAt = timestamppb.New(*k.LastUsedAt)
	}
	return pbKey
}

// orgMemberToProto converts a grpcservices.OrganizationMember to proto OrganizationUser.
// Maps all membership fields including admin flags.
func orgMemberToProto(m *grpcservices.OrganizationMember) *pb.OrganizationUser {
	if m == nil {
		return nil
	}
	return &pb.OrganizationUser{
		OrgId:              m.OrgID.String(),
		UserId:             m.UserID.String(),
		Email:              m.UserEmail,
		Role:               m.Role,
		Status:             m.Status,
		IsOrgAdmin:         m.IsOrgAdmin,
		IsBaseStationAdmin: m.IsBaseStationAdmin,
		IsEndpointAdmin:    m.IsEndpointAdmin,
		CreatedAt:          timestamppb.New(m.JoinedAt),
		UpdatedAt:          timestamppb.New(m.UpdatedAt),
	}
}

func isValidOrganizationRole(role string) bool {
	switch role {
	case models.OrganizationRoleOwner, models.OrganizationRoleAdmin, models.OrganizationRoleMember:
		return true
	default:
		return false
	}
}

func normalizeOrganizationMemberStatus(status string) (string, error) {
	switch status {
	case "":
		return "", nil
	case models.OrganizationMemberStatusFilterAll:
		return "", nil
	case models.OrganizationMemberStatusFilterInactive:
		return models.OrganizationMemberStatusFilterInactive, nil
	case models.OrganizationMemberStatusActive, models.OrganizationMemberStatusInvited, models.OrganizationMemberStatusRemoved:
		return status, nil
	default:
		return "", errors.New(grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidArgument))
	}
}
