// Package interceptors provides exported gRPC interceptors.
package interceptors

import (
	"context"

	audit "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"

	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// RoleSource resolves the roles of the principal behind a request.
type RoleSource interface {
	Roles(ctx context.Context) (authz.Roles, error)
}

// MethodPolicy names the roles a method requires; ok is false for a method
// the policy does not know, which is refused.
type MethodPolicy interface {
	Requirement(fullMethod string) (requirement authz.Requirement, ok bool)
}

// AuthorizationInterceptor admits a call only when the caller's roles satisfy
// the method's requirement, and records those roles on the request context.
type AuthorizationInterceptor struct {
	roles            RoleSource
	policy           MethodPolicy
	logger           logger.Logger
	eventWriter      audit.EventWriter
	platformTenantID int64
}

// WithEventWriter sets the security event writer for permission denial persistence.
func (ai *AuthorizationInterceptor) WithEventWriter(w audit.EventWriter) *AuthorizationInterceptor {
	ai.eventWriter = w
	return ai
}

// WithPlatformTenantID sets the fallback tenant for pre-auth events.
func (ai *AuthorizationInterceptor) WithPlatformTenantID(id int64) *AuthorizationInterceptor {
	ai.platformTenantID = id
	return ai
}

// NewAuthorizationInterceptor creates the role authorization interceptor.
func NewAuthorizationInterceptor(roles RoleSource, policy MethodPolicy, log logger.Logger) *AuthorizationInterceptor {
	return &AuthorizationInterceptor{roles: roles, policy: policy, logger: log}
}

// UnaryInterceptor returns a unary server interceptor that enforces the policy.
func (ai *AuthorizationInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx, err := ai.authorize(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// StreamInterceptor enforces the policy when a stream opens and again before each message it sends.
func (ai *AuthorizationInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if grpcerrors.IsPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}
		roles, err := ai.grantedRoles(ss.Context(), info.FullMethod)
		if err != nil {
			return err
		}
		base := contextServerStream{ServerStream: ss, ctx: authz.WithRoles(ss.Context(), roles)}
		return handler(srv, newRevalidatingServerStream(base, info.FullMethod, roles, ai.grantedRoles, ai.logger))
	}
}

func (ai *AuthorizationInterceptor) authorize(ctx context.Context, method string) (context.Context, error) {
	if grpcerrors.IsPublicMethod(method) {
		return ctx, nil
	}
	roles, err := ai.grantedRoles(ctx, method)
	if err != nil {
		return nil, err
	}
	return authz.WithRoles(ctx, roles), nil
}

// grantedRoles resolves the caller's roles and refuses them unless they satisfy the method's requirement.
func (ai *AuthorizationInterceptor) grantedRoles(ctx context.Context, method string) (authz.Roles, error) {
	requirement, known := ai.policy.Requirement(method)
	if !known {
		ai.logger.WarnContext(ctx, grpcerrors.LogRBACUnknownMethod, logger.FieldMethod, method)
		ai.emitPermissionDenied(ctx, method, detailUnknownMethod)
		return authz.Roles{}, insufficientRoleError()
	}

	roles, err := ai.roles.Roles(ctx)
	if err != nil {
		ai.logger.ErrorContext(ctx, grpcerrors.LogRBACResolutionFailed, logger.FieldMethod, method, logger.FieldError, err)
		ai.emitPermissionDenied(ctx, method, detailRoleResolutionFailed)
		return authz.Roles{}, grpcerrors.ToStatusError(err)
	}

	if !requirement.GrantedTo(roles) {
		ai.logger.WarnContext(ctx, grpcerrors.LogRBACInsufficientRole, logger.FieldMethod, method)
		ai.emitPermissionDenied(ctx, method, detailInsufficientRole)
		return authz.Roles{}, insufficientRoleError()
	}
	return roles, nil
}

func insufficientRoleError() error {
	return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInsufficientRole),
		grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInsufficientRole))
}

// emitPermissionDenied persists a permission denied security event when an event writer is configured.
func (ai *AuthorizationInterceptor) emitPermissionDenied(ctx context.Context, method, reason string) {
	recordSecurityEvent(ctx, ai.eventWriter, ai.platformTenantID, ai.logger, securityEvent{
		method: method, eventType: models.EventTypeAuthPermissionDenied,
		title: models.EventTitleAuthPermissionDenied, reason: reason, userID: principalUserID(ctx),
	})
}
