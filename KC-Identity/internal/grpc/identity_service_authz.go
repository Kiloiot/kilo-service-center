package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// validateOrgAccess validates the request org ID and confirms it belongs to the caller's tenant.
// Returns the parsed org UUID and tenant ID, or a gRPC status error.
func (s *IdentityService) validateOrgAccess(ctx context.Context, reqOrgID string) (uuid.UUID, int64, error) {
	if reqOrgID == "" {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDRequired),
		)
	}
	orgID, err := uuid.Parse(reqOrgID)
	if err != nil {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidOrgIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidOrgIDFormat),
		)
	}
	if s.orgSvc == nil {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured),
		)
	}
	tenantID, err := grpcerrors.GetTenantFromContext(ctx)
	if err != nil {
		return uuid.Nil, 0, err
	}
	if _, err := s.orgSvc.GetByID(ctx, orgID, tenantID); err != nil {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgNotFound),
		)
	}
	return orgID, tenantID, nil
}

// validateOrgAccessUnscoped validates org ID format and existence without tenant scoping.
func (s *IdentityService) validateOrgAccessUnscoped(ctx context.Context, reqOrgID string) (uuid.UUID, int64, error) {
	if reqOrgID == "" {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDRequired),
		)
	}
	orgID, err := uuid.Parse(reqOrgID)
	if err != nil {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidOrgIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidOrgIDFormat),
		)
	}
	if s.orgSvc == nil {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured),
		)
	}
	org, err := s.orgSvc.GetByIDUnscoped(ctx, orgID)
	if err != nil {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgNotFound),
		)
	}
	return orgID, org.TenantID, nil
}

// resolveOrgAccess admits an administrator to any organization and a tenant
// manager of the organization to organizations of the caller's tenant.
func (s *IdentityService) resolveOrgAccess(ctx context.Context, reqOrgID string) (uuid.UUID, int64, error) {
	if reqOrgID == "" {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgIDRequired),
		)
	}
	orgID, err := uuid.Parse(reqOrgID)
	if err != nil {
		return uuid.Nil, 0, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidOrgIDFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidOrgIDFormat),
		)
	}
	if s.roles == nil {
		return uuid.Nil, 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}
	callerID, err := grpcerrors.GetUserFromContext(ctx)
	if err != nil {
		return uuid.Nil, 0, err
	}

	roles, err := s.roles.Resolve(ctx, callerID, orgID)
	switch {
	case err != nil:
		s.log.ErrorContext(ctx, LogRoleResolutionFailed, logger.FieldOrgIDSnake, reqOrgID, logger.FieldUserIDSnake, callerID, logger.FieldError, err)
	case roles.Admin:
		return s.validateOrgAccessUnscoped(ctx, reqOrgID)
	case roles.TenantManager:
		return s.validateOrgAccess(ctx, reqOrgID)
	}
	s.emitSecurityEvent(ctx, models.EventTypeAuthPermissionDenied, models.EventTitleAuthPermissionDenied, opNameResolveOrgAccess, detailNonManagerAttemptedMemberOp)
	return uuid.Nil, 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgAdminRequired),
		grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgAdminRequired))
}

// requireServerAdmin checks that the caller is an active server administrator.
// Fail-safe: returns ServiceNotConfigured if adminUserSvc is nil.
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

	if !authz.Effective(caller, nil).Admin {
		s.emitSecurityEvent(ctx, models.EventTypeAuthPermissionDenied, models.EventTitleAuthPermissionDenied, opNameRequireAdmin, detailNonAdminAttemptedAdminOp)
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAdminRequired))
	}

	return nil
}
