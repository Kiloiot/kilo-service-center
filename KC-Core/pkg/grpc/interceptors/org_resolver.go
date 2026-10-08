package interceptors

import (
	"context"

	audit "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// OrgResolverInterceptorConfig holds configuration for the fail-closed interceptor.
type OrgResolverInterceptorConfig struct {
	// Resolver performs org UUID → tenant ID resolution (required)
	Resolver org.Resolver

	// Logger for debug/error logging (optional, uses default if nil)
	Logger logger.Logger

	// SkipMethods lists gRPC method names that bypass org enforcement.
	SkipMethods []string

	// EventWriter persists security events (optional, nil = no persistence)
	EventWriter audit.EventWriter

	// PlatformTenantID fallback tenant for pre-auth security events
	PlatformTenantID int64

	// AdminChecker checks if a user is a server admin (optional).
	// When set, server admins bypass the org mismatch check so they can
	// operate on any organization (e.g. auto-provisioning).
	AdminChecker AdminChecker
}

// AdminChecker checks whether a user is a server admin.
type AdminChecker interface {
	IsServerAdmin(ctx context.Context, userID string) (bool, error)
}

// OrgResolverInterceptor provides fail-closed organization validation for gRPC.
type OrgResolverInterceptor struct {
	resolver         org.Resolver
	log              logger.Logger
	skipMethods      map[string]struct{}
	eventWriter      audit.EventWriter
	platformTenantID int64
	adminChecker     AdminChecker
}

// NewOrgResolverInterceptor creates a fail-closed org resolver interceptor.
func NewOrgResolverInterceptor(cfg OrgResolverInterceptorConfig) (*OrgResolverInterceptor, error) {
	if cfg.Resolver == nil {
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenOrgResolverRequired),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenOrgResolverRequired))
	}

	log := cfg.Logger
	if log == nil {
		log = logger.Get()
	}

	skipMap := make(map[string]struct{}, len(cfg.SkipMethods))
	for _, method := range cfg.SkipMethods {
		skipMap[method] = struct{}{}
	}

	return &OrgResolverInterceptor{
		resolver:         cfg.Resolver,
		log:              log,
		skipMethods:      skipMap,
		eventWriter:      cfg.EventWriter,
		platformTenantID: cfg.PlatformTenantID,
		adminChecker:     cfg.AdminChecker,
	}, nil
}

// UnaryInterceptor returns the unary server interceptor for fail-closed org validation.
func (oi *OrgResolverInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if _, skip := oi.skipMethods[info.FullMethod]; skip {
			return handler(ctx, req)
		}

		newCtx, err := oi.resolveOrgContext(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}

		return handler(newCtx, req)
	}
}

// StreamInterceptor returns the stream server interceptor for fail-closed org validation.
func (oi *OrgResolverInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if _, skip := oi.skipMethods[info.FullMethod]; skip {
			return handler(srv, ss)
		}

		newCtx, err := oi.resolveOrgContext(ss.Context(), info.FullMethod)
		if err != nil {
			return err
		}

		wrappedStream := &contextServerStream{
			ServerStream: ss,
			ctx:          newCtx,
		}

		return handler(srv, wrappedStream)
	}
}

// resolveOrgContext extracts org/user from metadata, validates against any auth-established
// identity, and resolves tenant.
func (oi *OrgResolverInterceptor) resolveOrgContext(ctx context.Context, method string) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		oi.log.WarnContext(ctx, LogGrpcOrgInterceptorMissingMetadata,
			logger.FieldMethod, method)
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenMissingMetadata),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenMissingMetadata))
	}

	existingTenant, tenantErr := pkgcontext.GetTenantID(ctx)
	existingOrg, orgErr := pkgcontext.GetOrganizationID(ctx)
	existingUser, userErr := pkgcontext.GetUserID(ctx)
	authHasIdentity := tenantErr == nil

	orgIDHeaders := md.Get(grpcconst.MetadataKeyOrganizationID)
	if len(orgIDHeaders) == 0 {
		oi.log.WarnContext(ctx, LogGrpcOrgInterceptorMissingXOrganizationIDHeader,
			logger.FieldMethod, method)
		oi.emitSecurityEvent(ctx, method, models.EventTypeAuthOrgContextMissing, models.EventTitleAuthOrgContextMissing, detailMissingOrgHeader)
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenOrgIDHeaderRequired),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenOrgIDHeaderRequired))
	}

	orgUUID, err := uuid.Parse(orgIDHeaders[0])
	if err != nil {
		oi.log.WarnContext(ctx, LogGrpcOrgInterceptorInvalidXOrganizationIDFormat,
			logger.FieldMethod, method,
			logger.FieldValue, orgIDHeaders[0],
			logger.FieldError, err)
		oi.emitSecurityEvent(ctx, method, models.EventTypeAuthOrgContextMissing, models.EventTitleAuthOrgContextMissing, detailInvalidOrgHeaderFormat)
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenOrgIDHeaderInvalid),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenOrgIDHeaderInvalid))
	}

	resolvedTenantID, err := oi.resolver.LookupTenant(ctx, orgUUID)
	if err != nil {
		oi.log.ErrorContext(ctx, LogGrpcOrgInterceptorOrgResolutionFailed,
			logger.FieldMethod, method,
			logger.FieldOrgID, orgUUID.String(),
			logger.FieldError, err)
		oi.emitSecurityEvent(ctx, method, models.EventTypeAuthOrgResolutionFailed, models.EventTitleAuthOrgResolutionFailed, detailOrgUUIDResolutionFailed)
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenOrgResolutionFailed),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenOrgResolutionFailed))
	}

	if authHasIdentity {
		// Server admins can operate on any organization (cross-tenant access for provisioning)
		isAdmin := userErr == nil && oi.isServerAdmin(ctx, method, existingUser)

		if !isAdmin && existingTenant != resolvedTenantID {
			oi.log.WarnContext(ctx, LogGrpcOrgInterceptorTenantMismatchBetweenAuthAnd,
				logger.FieldMethod, method,
				logger.FieldAuthTenant, existingTenant,
				logger.FieldHeaderTenant, resolvedTenantID)
			return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenIdentityMismatch),
				grpcconst.ResolveErrorMessage(grpcconst.ErrTokenIdentityMismatch))
		}

		if !isAdmin && orgErr == nil && existingOrg != orgUUID {
			oi.log.WarnContext(ctx, LogGrpcOrgInterceptorOrgMismatchBetweenAuthAnd,
				logger.FieldMethod, method,
				logger.FieldAuthOrg, existingOrg.String(),
				logger.FieldHeaderOrg, orgUUID.String())
			return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenIdentityMismatch),
				grpcconst.ResolveErrorMessage(grpcconst.ErrTokenIdentityMismatch))
		}

		userIDHeaders := md.Get(grpcconst.MetadataKeyUserID)

		if userErr == nil {
			if len(userIDHeaders) == 0 {
				oi.log.WarnContext(ctx, LogGrpcOrgInterceptorMissingXUserIDHeaderForUserPrincipal,
					logger.FieldMethod, method,
					logger.FieldOrgID, orgUUID.String())
				return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenUserIDHeaderRequired),
					grpcconst.ResolveErrorMessage(grpcconst.ErrTokenUserIDHeaderRequired))
			}

			userUUID, err := uuid.Parse(userIDHeaders[0])
			if err != nil {
				oi.log.WarnContext(ctx, LogGrpcOrgInterceptorInvalidXUserIDFormat,
					logger.FieldMethod, method,
					logger.FieldOrgID, orgUUID.String(),
					logger.FieldValue, userIDHeaders[0],
					logger.FieldError, err)
				return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenUserIDHeaderInvalid),
					grpcconst.ResolveErrorMessage(grpcconst.ErrTokenUserIDHeaderInvalid))
			}

			if userUUID.String() != existingUser {
				oi.log.WarnContext(ctx, LogGrpcOrgInterceptorUserMismatchBetweenAuthAnd,
					logger.FieldMethod, method,
					logger.FieldAuthUser, existingUser,
					logger.FieldHeaderUser, userUUID.String())
				return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenIdentityMismatch),
					grpcconst.ResolveErrorMessage(grpcconst.ErrTokenIdentityMismatch))
			}
		} else {
			if len(userIDHeaders) > 0 {
				oi.log.WarnContext(ctx, LogGrpcOrgInterceptorXUserIDHeaderNot,
					logger.FieldMethod, method,
					logger.FieldOrgID, orgUUID.String(),
					logger.FieldHeaderUser, userIDHeaders[0])
				return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenIdentityMismatch),
					grpcconst.ResolveErrorMessage(grpcconst.ErrTokenIdentityMismatch))
			}
		}

		if orgErr != nil || isAdmin {
			// For regular users: set org only if auth didn't provide one.
			// For server admins: override with the header org to act on any organization.
			ctx = pkgcontext.WithOrganizationID(ctx, orgUUID)
		}
		if isAdmin {
			// Server admins also adopt the selected org's tenant, so tenant-scoped reads target it.
			ctx = pkgcontext.WithTenantID(ctx, resolvedTenantID)
		}

		oi.log.DebugContext(ctx, LogGrpcOrgInterceptorValidatedAuthIdentityAgainstHeaders,
			logger.FieldMethod, method,
			logger.FieldOrgID, orgUUID.String(),
			logger.FieldTenantID, resolvedTenantID)

		return ctx, nil
	}

	// No auth identity — set all values from headers
	userIDHeaders := md.Get(grpcconst.MetadataKeyUserID)
	if len(userIDHeaders) == 0 {
		oi.log.WarnContext(ctx, LogGrpcOrgInterceptorMissingXUserIDHeader,
			logger.FieldMethod, method,
			logger.FieldOrgID, orgUUID.String())
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenUserIDHeaderRequired),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenUserIDHeaderRequired))
	}

	userUUID, err := uuid.Parse(userIDHeaders[0])
	if err != nil {
		oi.log.WarnContext(ctx, LogGrpcOrgInterceptorInvalidXUserIDFormat,
			logger.FieldMethod, method,
			logger.FieldOrgID, orgUUID.String(),
			logger.FieldValue, userIDHeaders[0],
			logger.FieldError, err)
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenUserIDHeaderInvalid),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenUserIDHeaderInvalid))
	}

	ctx = pkgcontext.WithOrganizationID(ctx, orgUUID)
	ctx = pkgcontext.WithUserID(ctx, userUUID.String())
	ctx = pkgcontext.WithTenantID(ctx, resolvedTenantID)

	oi.log.DebugContext(ctx, LogGrpcOrgInterceptorResolvedOrgContext,
		logger.FieldMethod, method,
		logger.FieldOrgID, orgUUID.String(),
		logger.FieldUserID, userUUID.String(),
		logger.FieldTenantID, resolvedTenantID)

	return ctx, nil
}

// isServerAdmin reports whether userID is a server admin; a check that fails
// grants no cross-tenant access.
func (oi *OrgResolverInterceptor) isServerAdmin(ctx context.Context, method, userID string) bool {
	if oi.adminChecker == nil {
		return false
	}
	isAdmin, err := oi.adminChecker.IsServerAdmin(ctx, userID)
	if err != nil {
		oi.log.WarnContext(ctx, LogGrpcOrgInterceptorAdminCheckFailed,
			logger.FieldMethod, method, logger.FieldUserID, userID, logger.FieldError, err)
		return false
	}
	return isAdmin
}

// emitSecurityEvent persists a security event when an event writer is configured.
func (oi *OrgResolverInterceptor) emitSecurityEvent(ctx context.Context, method, eventType, title, reason string) {
	recordSecurityEvent(ctx, oi.eventWriter, oi.platformTenantID, oi.log, securityEvent{
		method: method, eventType: eventType, title: title, reason: reason,
	})
}
