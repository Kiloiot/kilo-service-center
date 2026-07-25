// Package interceptors provides exported gRPC interceptors.
package interceptors

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// RoleResolver resolves a user's role within an organization.
type RoleResolver interface {
	GetUserRole(ctx context.Context, orgID, userID string) (role string, active bool, err error)
}

type cachedRole struct {
	role      string
	active    bool
	expiresAt time.Time
}

// AuthorizationInterceptor enforces role-based access on gRPC methods.
type AuthorizationInterceptor struct {
	roleResolver     RoleResolver
	policies         map[string][]string // full gRPC method → allowed roles
	logger           logger.Logger
	cache            sync.Map // "orgID:userID" → cachedRole
	cacheTTL         time.Duration
	eventWriter      grpcerrors.EventWriter
	platformTenantID int64
}

// WithEventWriter sets the security event writer for permission denial persistence.
func (ai *AuthorizationInterceptor) WithEventWriter(w grpcerrors.EventWriter) *AuthorizationInterceptor {
	ai.eventWriter = w
	return ai
}

// WithPlatformTenantID sets the fallback tenant for pre-auth events.
func (ai *AuthorizationInterceptor) WithPlatformTenantID(id int64) *AuthorizationInterceptor {
	ai.platformTenantID = id
	return ai
}

// NewAuthorizationInterceptor creates a new RBAC interceptor.
// policies maps full gRPC method paths to lists of allowed roles.
// cacheTTL controls how long resolved roles are cached; zero or negative uses 30s default.
func NewAuthorizationInterceptor(resolver RoleResolver, policies map[string][]string, log logger.Logger, cacheTTL time.Duration) *AuthorizationInterceptor {
	if cacheTTL <= 0 {
		cacheTTL = time.Duration(pkgconfig.DefaultRBACRoleCacheTTLSeconds) * time.Second
	}
	return &AuthorizationInterceptor{
		roleResolver: resolver,
		policies:     policies,
		logger:       log,
		cacheTTL:     cacheTTL,
	}
}

// UnaryInterceptor returns a unary server interceptor that enforces RBAC policies.
func (ai *AuthorizationInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := ai.authorize(ctx, info.FullMethod); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// StreamInterceptor returns a stream server interceptor that enforces RBAC policies.
func (ai *AuthorizationInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := ai.authorize(ss.Context(), info.FullMethod); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

func (ai *AuthorizationInterceptor) authorize(ctx context.Context, method string) error {
	allowedRoles, ok := ai.policies[method]
	if !ok {
		return nil // method not in policy → passthrough
	}

	orgUUID, orgErr := pkgcontext.GetOrganizationID(ctx)
	userID, userErr := pkgcontext.GetUserID(ctx)

	if orgErr != nil || userErr != nil {
		ai.logger.WarnContext(ctx, grpcerrors.LogRBACMissingContext, "method", method)
		ai.emitPermissionDenied(ctx, method, "missing org or user context")
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAdminRequired))
	}

	orgID := orgUUID.String()

	role, active, err := ai.resolveRole(ctx, orgID, userID)
	if err != nil {
		ai.logger.ErrorContext(ctx, grpcerrors.LogRBACResolutionFailed, "method", method, "error", err)
		ai.emitPermissionDenied(ctx, method, "role resolution failed")
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAdminRequired))
	}

	if !active {
		ai.logger.WarnContext(ctx, grpcerrors.LogRBACInactiveMembership, "method", method, "role", role)
		ai.emitPermissionDenied(ctx, method, "inactive membership")
		return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAdminRequired))
	}

	for _, allowed := range allowedRoles {
		if role == allowed {
			return nil
		}
	}

	ai.logger.WarnContext(ctx, grpcerrors.LogRBACInsufficientRole,
		"method", method, "role", role, "required", allowedRoles)
	ai.emitPermissionDenied(ctx, method, "insufficient role")
	return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAdminRequired))
}

func (ai *AuthorizationInterceptor) resolveRole(ctx context.Context, orgID, userID string) (string, bool, error) {
	cacheKey := orgID + ":" + userID

	if cached, ok := ai.cache.Load(cacheKey); ok {
		cr := cached.(cachedRole)
		if time.Now().Before(cr.expiresAt) {
			return cr.role, cr.active, nil
		}
		ai.cache.Delete(cacheKey)
	}

	role, active, err := ai.roleResolver.GetUserRole(ctx, orgID, userID)
	if err != nil {
		return "", false, err
	}

	ai.cache.Store(cacheKey, cachedRole{
		role:      role,
		active:    active,
		expiresAt: time.Now().Add(ai.cacheTTL),
	})

	return role, active, nil
}

// emitPermissionDenied persists a permission denied security event when an event writer is configured.
func (ai *AuthorizationInterceptor) emitPermissionDenied(ctx context.Context, method, reason string) {
	if ai.eventWriter == nil {
		return
	}
	tenantID := ai.platformTenantID
	if tid, err := pkgcontext.GetTenantID(ctx); err == nil {
		tenantID = tid
	}
	details, _ := json.Marshal(map[string]interface{}{
		auditKeyMethod: method,
		auditKeyReason: reason,
	})
	_ = ai.eventWriter.CreateEvent(ctx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(tenantID, 10),
		EventType:   models.EventTypeAuthPermissionDenied,
		Category:    models.EventCategorySecurity,
		Severity:    models.EventSeverityWarning,
		Title:       models.EventTitleAuthPermissionDenied,
		Description: reason,
		SourceType:  models.SourceTypeAPI,
		SourceName:  method,
		Details:     details,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
}
