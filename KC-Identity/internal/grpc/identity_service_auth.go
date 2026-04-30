// Package grpc provides gRPC service implementations.
// Auth RPC handlers for user authentication, management, and authorization.
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

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/kilocenter/KC-Identity/internal/services/auth"
	"github.com/kilocenter/KC-Identity/internal/services/grpcservices"
)

// Login authenticates a user and returns tokens with user profile in a single round-trip.
func (s *IdentityService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if s.authSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	// Validate request
	if req.Email == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEmailRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEmailRequired))
	}
	if req.Password == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenPasswordRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenPasswordRequired))
	}

	result, err := s.authSvc.Login(ctx, req.Email, req.Password)
	if err != nil {
		s.log.ErrorContext(ctx, "login failed", "email", req.Email, "error", err)
		s.emitSecurityEvent(ctx, models.EventTypeAuthLoginFailed, models.EventTitleAuthLoginFailed, "Login", "invalid credentials for "+req.Email)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidCredentials),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidCredentials))
	}

	// Convert profile memberships to proto
	var memberships []*pb.UserMembership
	for _, m := range result.Profile.Memberships {
		memberships = append(memberships, &pb.UserMembership{
			OrgId:              m.OrgID.String(),
			OrgName:            m.OrgName,
			Role:               m.Role,
			IsOrgAdmin:         m.IsOrgAdmin,
			IsBaseStationAdmin: m.IsBaseStationAdmin,
			IsEndpointAdmin:    m.IsEndpointAdmin,
		})
	}

	resp := &pb.LoginResponse{
		Tokens: &pb.AuthTokens{
			AccessToken:      result.Tokens.AccessToken,
			RefreshToken:     result.Tokens.RefreshToken,
			AccessExpiresIn:  result.Tokens.AccessExpiresIn,
			RefreshExpiresIn: result.Tokens.RefreshExpiresIn,
		},
		User: &pb.UserProfile{
			Id:          result.Profile.ID.String(),
			Email:       result.Profile.Email,
			IsAdmin:     result.Profile.IsAdmin,
			HasPassword: result.Profile.HasPassword,
			FirstName:   result.Profile.FirstName,
			LastName:    result.Profile.LastName,
			Memberships: memberships,
		},
	}
	if result.Profile.DefaultOrgID != nil {
		resp.User.DefaultOrgId = result.Profile.DefaultOrgID.String()
	}
	return resp, nil
}

// RefreshTokens refreshes the access token using a refresh token.
func (s *IdentityService) RefreshTokens(ctx context.Context, req *pb.RefreshTokensRequest) (*pb.RefreshTokensResponse, error) {
	if s.authSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.RefreshToken == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRefreshTokenRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRefreshTokenRequired))
	}

	tokens, err := s.authSvc.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		s.log.ErrorContext(ctx, "token refresh failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidRefreshToken),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidRefreshToken))
	}

	return &pb.RefreshTokensResponse{
		Tokens: &pb.AuthTokens{
			AccessToken:      tokens.AccessToken,
			RefreshToken:     tokens.RefreshToken,
			AccessExpiresIn:  tokens.AccessExpiresIn,
			RefreshExpiresIn: tokens.RefreshExpiresIn,
		},
	}, nil
}

// GetProfile returns the current user's profile.
func (s *IdentityService) GetProfile(ctx context.Context, _ *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	if s.authSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	userID, err := grpcerrors.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	profile, err := s.authSvc.GetProfile(ctx, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "get profile failed", "user_id", userID, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetProfileFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetProfileFailed))
	}

	// Convert memberships to proto
	var memberships []*pb.UserMembership
	for _, m := range profile.Memberships {
		memberships = append(memberships, &pb.UserMembership{
			OrgId:              m.OrgID.String(),
			OrgName:            m.OrgName,
			Role:               m.Role,
			IsOrgAdmin:         m.IsOrgAdmin,
			IsBaseStationAdmin: m.IsBaseStationAdmin,
			IsEndpointAdmin:    m.IsEndpointAdmin,
		})
	}

	resp := &pb.GetProfileResponse{
		User: &pb.UserProfile{
			Id:          profile.ID.String(),
			Email:       profile.Email,
			IsAdmin:     profile.IsAdmin,
			HasPassword: profile.HasPassword,
			FirstName:   profile.FirstName,
			LastName:    profile.LastName,
			Memberships: memberships,
		},
	}
	if profile.DefaultOrgID != nil {
		resp.User.DefaultOrgId = profile.DefaultOrgID.String()
	}
	return resp, nil
}

// GetAuthSettings returns authentication settings (public endpoint).
func (s *IdentityService) GetAuthSettings(ctx context.Context, _ *pb.GetAuthSettingsRequest) (*pb.GetAuthSettingsResponse, error) {
	if s.authSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	settings, err := s.authSvc.GetAuthSettings(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "get auth settings failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetAuthSettingsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetAuthSettingsFailed))
	}

	resp := &pb.GetAuthSettingsResponse{
		Settings: &pb.AuthSettings{
			Enabled:             settings.Enabled,
			LocalLoginEnabled:   settings.LocalLoginEnabled,
			LoginUrl:            settings.LoginURL,
			LoginLabel:          settings.LoginLabel,
			LoginRedirect:       settings.LoginRedirect,
			LogoutUrl:           settings.LogoutURL,
			RefreshTokenEnabled: settings.RefreshTokenEnabled,
			RegistrationEnabled: s.registrationSvc != nil,
		},
	}
	if settings.OIDCEnabled {
		resp.Settings.Oidc = &pb.ProviderSettings{
			Enabled:  settings.OIDCEnabled,
			LoginUrl: settings.OIDCProviderURL,
		}
	}
	return resp, nil
}

// Logout invalidates the user's tokens.
func (s *IdentityService) Logout(ctx context.Context, _ *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	if s.authSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	userID, err := grpcerrors.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.authSvc.Logout(ctx, userID); err != nil {
		s.log.ErrorContext(ctx, "logout failed", "user_id", userID, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenLogoutFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLogoutFailed))
	}

	return &pb.LogoutResponse{Success: true}, nil
}

// ChangePassword changes the current user's password.
func (s *IdentityService) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	if s.authSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	userID, err := grpcerrors.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.CurrentPassword == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCurrentPasswordRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCurrentPasswordRequired))
	}
	if req.NewPassword == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNewPasswordRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNewPasswordRequired))
	}

	if err := s.authSvc.ChangePassword(ctx, userID, req.CurrentPassword, req.NewPassword); err != nil {
		s.log.ErrorContext(ctx, "change password failed", "user_id", userID, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenChangePasswordFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenChangePasswordFailed))
	}

	return &pb.ChangePasswordResponse{Success: true}, nil
}

// Admin user handlers

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
		s.log.ErrorContext(ctx, "create user failed", "email", req.Email, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateUserFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateUserFailed))
	}

	// Emit event when tenant context is available
	tenantID, tenantErr := grpcerrors.GetTenantFromContext(ctx)
	if tenantErr == nil && s.eventWriter != nil {
		userIDStr := user.ID.String()
		sourceID := user.ID
		detailsJSON, _ := json.Marshal(map[string]interface{}{"email": req.Email})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(tenantID, 10),
			EventType:   models.EventTypeUserCreated,
			Category:    models.EventCategoryAudit,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleUserCreated,
			Description: fmt.Sprintf(models.EventDescriptionUserCreated, req.Email),
			SourceType:  models.SourceTypeAPI,
			SourceID:    &sourceID,
			SourceName:  req.Email,
			UserID:      userIDStr,
			Details:     detailsJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

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
		s.log.ErrorContext(ctx, "get user failed", "user_id", req.Id, "error", err)
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
		s.log.ErrorContext(ctx, "update user failed", "user_id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateUserFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateUserFailed))
	}

	// Emit event when tenant context is available
	tenantID, tenantErr := grpcerrors.GetTenantFromContext(ctx)
	if tenantErr == nil && s.eventWriter != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"userId": req.Id,
			"email":  user.Email,
		})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(tenantID, 10),
			EventType:   models.EventTypeUserUpdated,
			Category:    models.EventCategoryAudit,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleUserUpdated,
			Description: fmt.Sprintf(models.EventDescriptionUserUpdated, user.Email),
			SourceType:  models.SourceTypeAPI,
			SourceName:  user.Email,
			UserID:      req.Id,
			Details:     detailsJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

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
		s.log.ErrorContext(ctx, "delete user failed", "user_id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteUserFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteUserFailed))
	}

	// Emit event when tenant context is available
	tenantID, tenantErr := grpcerrors.GetTenantFromContext(ctx)
	if tenantErr == nil && s.eventWriter != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{"userId": req.Id})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(tenantID, 10),
			EventType:   models.EventTypeUserDeleted,
			Category:    models.EventCategoryAudit,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleUserDeleted,
			Description: fmt.Sprintf(models.EventDescriptionUserDeleted, req.Id),
			SourceType:  models.SourceTypeAPI,
			UserID:      req.Id,
			Details:     detailsJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

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
		s.log.ErrorContext(ctx, "list users failed", "error", err)
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

// UpdateUserPassword updates a user's password (admin only).
func (s *IdentityService) UpdateUserPassword(ctx context.Context, req *pb.UpdateUserPasswordRequest) (*pb.UpdateUserPasswordResponse, error) {
	if err := s.requireServerAdmin(ctx); err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}
	if req.NewPassword == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNewPasswordRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNewPasswordRequired))
	}

	userID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidIDFormat))
	}

	if err := s.adminUserSvc.UpdatePassword(ctx, userID, req.NewPassword); err != nil {
		s.log.ErrorContext(ctx, "update user password failed", "user_id", req.Id, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdatePasswordFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdatePasswordFailed))
	}

	return &pb.UpdateUserPasswordResponse{Success: true}, nil
}

// userToProto converts a User model to proto.
func userToProto(u *models.User) *pb.User {
	if u == nil {
		return nil
	}
	pbUser := &pb.User{
		Id:                   u.ID.String(),
		Email:                u.Email,
		IsAdmin:              u.IsAdmin,
		IsActive:             u.IsActive,
		IsTenantManager:      u.IsTenantManager,
		IsBaseStationManager: u.IsBaseStationManager,
		IsEndpointManager:    u.IsEndpointManager,
		CreatedAt:            timestamppb.New(u.CreatedAt),
		UpdatedAt:            timestamppb.New(u.UpdatedAt),
	}
	if u.Note != nil {
		pbUser.Note = *u.Note
	}
	if u.FirstName != nil {
		pbUser.FirstName = *u.FirstName
	}
	if u.LastName != nil {
		pbUser.LastName = *u.LastName
	}
	if u.CompanyName != nil {
		pbUser.CompanyName = *u.CompanyName
	}
	return pbUser
}

// requireServerAdmin checks that the caller is a server admin user. Returns nil on success,
// or a gRPC status error otherwise. Fail-safe: returns Internal if adminUserSvc is nil.
func (s *IdentityService) requireServerAdmin(ctx context.Context) error {
	if s.adminUserSvc == nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	callerID, err := grpcerrors.GetUserFromContext(ctx)
	if err != nil {
		return err
	}

	caller, err := s.adminUserSvc.GetByID(ctx, callerID)
	if err != nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserNotFound))
	}

	if !caller.IsAdmin {
		s.emitSecurityEvent(ctx, models.EventTypeAuthPermissionDenied, models.EventTitleAuthPermissionDenied, "requireServerAdmin", "non-admin user attempted admin operation")
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAdminRequired))
	}

	return nil
}

// requireOrgAdmin checks that the caller is an org admin for the specified organization.
func (s *IdentityService) requireOrgAdmin(ctx context.Context, orgID uuid.UUID) error {
	if s.membershipSvc == nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	callerID, err := grpcerrors.GetUserFromContext(ctx)
	if err != nil {
		return err
	}

	member, err := s.membershipSvc.GetMembership(ctx, orgID, callerID)
	if err != nil {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgAdminRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgAdminRequired))
	}

	if !member.IsOrgAdmin {
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgAdminRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgAdminRequired))
	}

	return nil
}

// requireServerOrOrgAdmin checks if caller is server admin or org admin.
// Returns (true, nil) if server admin, (false, nil) if org admin, or error if neither.
func (s *IdentityService) requireServerOrOrgAdmin(ctx context.Context, orgID uuid.UUID) (bool, error) {
	if err := s.requireServerAdmin(ctx); err == nil {
		return true, nil
	}
	if err := s.requireOrgAdmin(ctx, orgID); err == nil {
		return false, nil
	}
	return false, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgAdminRequired),
		grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgAdminRequired))
}

// ============================================================================
// External auth handlers (OIDC/OAuth2 exchange)
// ============================================================================

// ExchangeOIDC exchanges an OIDC authorization code for tokens.
func (s *IdentityService) ExchangeOIDC(ctx context.Context, req *pb.ExchangeOIDCRequest) (*pb.LoginResponse, error) {
	if s.externalAuthSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthOIDCDisabled),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthOIDCDisabled))
	}

	if req.Code == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCodeRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCodeRequired))
	}
	if req.State == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenStateRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenStateRequired))
	}

	result, err := s.externalAuthSvc.CompleteOIDCLogin(ctx, req.Code, req.State)
	if err != nil {
		s.log.ErrorContext(ctx, "OIDC exchange failed", "error", err)
		s.emitSecurityEvent(ctx, models.EventTypeAuthOIDCExchangeFailed, models.EventTitleAuthOIDCExchangeFailed, "ExchangeOIDC", "OIDC code exchange failed")
		return nil, mapExternalAuthError(err)
	}

	resp, err := buildLoginResponse(result)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to build login response", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}
	return resp, nil
}

// ExchangeOAuth2 exchanges an OAuth2 authorization code for tokens.
func (s *IdentityService) ExchangeOAuth2(ctx context.Context, req *pb.ExchangeOAuth2Request) (*pb.LoginResponse, error) {
	if s.externalAuthSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthOAuth2Disabled),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthOAuth2Disabled))
	}

	if req.Code == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCodeRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCodeRequired))
	}
	if req.State == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenStateRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenStateRequired))
	}

	result, err := s.externalAuthSvc.CompleteOAuth2Login(ctx, req.Code, req.State)
	if err != nil {
		s.log.ErrorContext(ctx, "OAuth2 exchange failed", "error", err)
		s.emitSecurityEvent(ctx, models.EventTypeAuthOAuth2ExchangeFailed, models.EventTitleAuthOAuth2ExchangeFailed, "ExchangeOAuth2", "OAuth2 code exchange failed")
		return nil, mapExternalAuthError(err)
	}

	resp, err := buildLoginResponse(result)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to build login response", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}
	return resp, nil
}

// buildLoginResponse builds a LoginResponse from an AuthLoginResult.
// Returns error if result, tokens, or profile are nil.
func buildLoginResponse(result *grpcservices.AuthLoginResult) (*pb.LoginResponse, error) {
	if result == nil || result.Tokens == nil || result.Profile == nil {
		return nil, errors.New(grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}

	var memberships []*pb.UserMembership
	for _, m := range result.Profile.Memberships {
		memberships = append(memberships, &pb.UserMembership{
			OrgId:              m.OrgID.String(),
			OrgName:            m.OrgName,
			Role:               m.Role,
			IsOrgAdmin:         m.IsOrgAdmin,
			IsBaseStationAdmin: m.IsBaseStationAdmin,
			IsEndpointAdmin:    m.IsEndpointAdmin,
		})
	}

	resp := &pb.LoginResponse{
		Tokens: &pb.AuthTokens{
			AccessToken:      result.Tokens.AccessToken,
			RefreshToken:     result.Tokens.RefreshToken,
			AccessExpiresIn:  result.Tokens.AccessExpiresIn,
			RefreshExpiresIn: result.Tokens.RefreshExpiresIn,
		},
		User: &pb.UserProfile{
			Id:          result.Profile.ID.String(),
			Email:       result.Profile.Email,
			IsAdmin:     result.Profile.IsAdmin,
			HasPassword: result.Profile.HasPassword,
			FirstName:   result.Profile.FirstName,
			LastName:    result.Profile.LastName,
			Memberships: memberships,
		},
	}
	if result.Profile.DefaultOrgID != nil {
		resp.User.DefaultOrgId = result.Profile.DefaultOrgID.String()
	}
	return resp, nil
}

// emitSecurityEvent persists a security event when an event writer is configured.
func (s *IdentityService) emitSecurityEvent(ctx context.Context, eventType, title, method, reason string) {
	if s.eventWriter == nil {
		return
	}
	tenantID := s.platformTenantID
	if tid, tErr := grpcerrors.GetTenantFromContext(ctx); tErr == nil {
		tenantID = tid
	}
	details, _ := json.Marshal(map[string]interface{}{
		"method": method,
		"reason": reason,
	})
	_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(tenantID, 10),
		EventType:   eventType,
		Category:    models.EventCategorySecurity,
		Severity:    models.EventSeverityWarning,
		Title:       title,
		Description: reason,
		SourceType:  models.SourceTypeAPI,
		SourceName:  method,
		Details:     details,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
}

// mapExternalAuthError maps external auth service errors to gRPC status errors.
// Uses errors.Is() for robust error matching that handles wrapped errors.
func mapExternalAuthError(err error) error {
	switch {
	case errors.Is(err, auth.ErrOIDCProviderDisabled):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthOIDCDisabled),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthOIDCDisabled))
	case errors.Is(err, auth.ErrOAuth2ProviderDisabled):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthOAuth2Disabled),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthOAuth2Disabled))
	case errors.Is(err, auth.ErrStateNotFound):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthStateNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthStateNotFound))
	case errors.Is(err, auth.ErrInvalidState):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthStateInvalid),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthStateInvalid))
	case errors.Is(err, auth.ErrTokenExchangeFailed):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthTokenExchangeFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthTokenExchangeFailed))
	case errors.Is(err, auth.ErrIDTokenInvalid):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthIDTokenInvalid),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthIDTokenInvalid))
	case errors.Is(err, auth.ErrNonceMismatch):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthIDTokenInvalid),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthIDTokenInvalid))
	case errors.Is(err, auth.ErrEmailNotVerified):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthEmailNotVerified),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthEmailNotVerified))
	case errors.Is(err, auth.ErrRegistrationDisabled):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthRegistrationDisabled),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthRegistrationDisabled))
	case errors.Is(err, auth.ErrRedisUnavailable):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthRedisUnavailable),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthRedisUnavailable))
	case errors.Is(err, auth.ErrUserInfoFailed):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthUserInfoFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthUserInfoFailed))
	case errors.Is(err, auth.ErrOrgResolutionFailed):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthExternalOrgFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthExternalOrgFailed))
	case errors.Is(err, auth.ErrMembershipRequired):
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMembershipRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMembershipRequired))
	default:
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}
}
