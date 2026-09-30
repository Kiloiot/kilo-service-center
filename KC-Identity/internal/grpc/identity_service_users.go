package grpc

import (
	"context"
	"fmt"
	"math"

	audit "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
)

// CreateUser creates a new user (admin only).
func (s *IdentityService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}

	if req.Email == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEmailRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEmailRequired))
	}
	if req.Password == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenPasswordRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenPasswordRequired))
	}

	createReq := &grpcservices.UserCreateRequest{
		Email:                req.Email,
		Password:             req.Password,
		IsAdmin:              req.IsAdmin,
		IsActive:             req.IsActive,
		IsTenantManager:      req.IsTenantManager,
		IsBaseStationManager: req.IsBaseStationManager,
		IsEndpointManager:    req.IsEndpointManager,
		Note:                 req.Note,
		FirstName:            req.FirstName,
		LastName:             req.LastName,
		CompanyName:          req.CompanyName,
	}

	user, err := s.adminUserSvc.Create(ctx, createReq)
	if err != nil {
		s.log.ErrorContext(ctx, LogCreateUserFailed, userFieldEmail, req.Email, logger.FieldError, err)
		return nil, passwordRefusal(err, grpcerrors.ErrTokenCreateUserFailed)
	}

	sourceID := user.ID
	s.audit.Record(ctx, audit.Event{
		TenantID:    s.platformTenantID,
		SourceID:    &sourceID,
		EventType:   models.EventTypeUserCreated,
		Title:       models.EventTitleUserCreated,
		Description: fmt.Sprintf(models.EventDescriptionUserCreated, req.Email),
		SourceName:  req.Email,
		Details:     map[string]any{auditKeyUserID: user.ID.String(), userFieldEmail: req.Email},
	})

	return &pb.CreateUserResponse{User: userToProto(user)}, nil
}

// GetUser returns a user by ID (admin only).
func (s *IdentityService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	userID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidIDFormat))
	}

	user, err := s.adminUserSvc.GetByID(ctx, userID)
	if err != nil {
		s.log.ErrorContext(ctx, LogGetUserFailed, logger.FieldUserIDSnake, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserNotFound))
	}

	return &pb.GetUserResponse{User: userToProto(user)}, nil
}

// User update field-name constants for FieldMask paths
const (
	userFieldEmail                = "email"
	userFieldNote                 = "note"
	userFieldIsAdmin              = "is_admin"
	userFieldIsActive             = "is_active"
	userFieldIsTenantManager      = "is_tenant_manager"
	userFieldIsBaseStationManager = "is_base_station_manager"
	userFieldIsEndpointManager    = "is_endpoint_manager"
)

// UpdateUser updates a user (admin only).
func (s *IdentityService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	mask := req.UpdateMask
	if mask == nil || len(mask.GetPaths()) == 0 {
		return nil, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateMaskRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateMaskRequired))
	}

	userID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidIDFormat))
	}

	// Only populate fields present in the FieldMask
	updateReq := &grpcservices.UserUpdateRequest{}
	if grpcerrors.FieldInMask(mask, userFieldEmail) {
		if req.Email == "" {
			return nil, status.Error(
				grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEmailRequired),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEmailRequired))
		}
		updateReq.Email = &req.Email
	}
	if grpcerrors.FieldInMask(mask, userFieldNote) {
		updateReq.Note = &req.Note
	}
	if grpcerrors.FieldInMask(mask, userFieldIsAdmin) {
		updateReq.IsAdmin = &req.IsAdmin
	}
	if grpcerrors.FieldInMask(mask, userFieldIsActive) {
		updateReq.IsActive = &req.IsActive
	}
	if grpcerrors.FieldInMask(mask, userFieldIsTenantManager) {
		updateReq.IsTenantManager = &req.IsTenantManager
	}
	if grpcerrors.FieldInMask(mask, userFieldIsBaseStationManager) {
		updateReq.IsBaseStationManager = &req.IsBaseStationManager
	}
	if grpcerrors.FieldInMask(mask, userFieldIsEndpointManager) {
		updateReq.IsEndpointManager = &req.IsEndpointManager
	}

	user, err := s.adminUserSvc.Update(ctx, userID, updateReq)
	if err != nil {
		s.log.ErrorContext(ctx, LogUpdateUserFailed, logger.FieldUserIDSnake, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateUserFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateUserFailed))
	}

	s.audit.Record(ctx, audit.Event{
		TenantID:    s.platformTenantID,
		EventType:   models.EventTypeUserUpdated,
		Title:       models.EventTitleUserUpdated,
		Description: fmt.Sprintf(models.EventDescriptionUserUpdated, user.Email),
		SourceName:  user.Email,
		Details:     map[string]any{auditKeyUserID: req.Id, userFieldEmail: user.Email},
	})

	return &pb.UpdateUserResponse{User: userToProto(user)}, nil
}

// DeleteUser deletes a user (admin only).
func (s *IdentityService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	userID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidIDFormat))
	}

	if err := s.adminUserSvc.Delete(ctx, userID); err != nil {
		s.log.ErrorContext(ctx, LogDeleteUserFailed, logger.FieldUserIDSnake, req.Id, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteUserFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteUserFailed))
	}

	s.audit.Record(ctx, audit.Event{
		TenantID:    s.platformTenantID,
		EventType:   models.EventTypeUserDeleted,
		Title:       models.EventTitleUserDeleted,
		Description: fmt.Sprintf(models.EventDescriptionUserDeleted, req.Id),
		Details:     map[string]any{auditKeyUserID: req.Id},
	})

	return &pb.DeleteUserResponse{Success: true}, nil
}

// ListUsers returns a list of users (admin only).
func (s *IdentityService) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
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

	users, total, err := s.adminUserSvc.List(ctx, limit, offset)
	if err != nil {
		s.log.ErrorContext(ctx, LogListUsersFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListUsersFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListUsersFailed))
	}

	var pbUsers []*pb.User
	for _, u := range users {
		pbUsers = append(pbUsers, userToProto(u))
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
	return &pb.ListUsersResponse{
		Users:         pbUsers,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}
