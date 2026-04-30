package interceptors

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
	grpcconst "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/org"
	"github.com/kilocenter/KC-DB/storage/models"
	pkgcontext "github.com/kilocenter/pkg/context"
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
	EventWriter grpcconst.EventWriter

	// PlatformTenantID fallback tenant for pre-auth security events
	PlatformTenantID int64
}

// OrgResolverInterceptor provides fail-closed organization validation for gRPC.
type OrgResolverInterceptor struct {
	resolver         org.Resolver
	log              logger.Logger
	skipMethods      map[string]struct{}
	eventWriter      grpcconst.EventWriter
	platformTenantID int64
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

		wrappedStream := &orgResolvedServerStream{
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
		oi.log.WarnContext(ctx, "gRPC org interceptor: missing metadata",
			"method", method)
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenMissingMetadata),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenMissingMetadata))
	}

	existingTenant, tenantErr := pkgcontext.GetTenantID(ctx)
	existingOrg, orgErr := pkgcontext.GetOrganizationID(ctx)
	existingUser, userErr := pkgcontext.GetUserID(ctx)
	authHasIdentity := tenantErr == nil

	orgIDHeaders := md.Get(grpcconst.MetadataKeyOrganizationID)
	if len(orgIDHeaders) == 0 {
		oi.log.WarnContext(ctx, "gRPC org interceptor: missing x-organization-id header",
			"method", method)
		oi.emitSecurityEvent(ctx, method, models.EventTypeAuthOrgContextMissing, models.EventTitleAuthOrgContextMissing, "missing x-organization-id header")
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenOrgIDHeaderRequired),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenOrgIDHeaderRequired))
	}

	orgUUID, err := uuid.Parse(orgIDHeaders[0])
	if err != nil {
		oi.log.WarnContext(ctx, "gRPC org interceptor: invalid x-organization-id format",
			"method", method,
			"value", orgIDHeaders[0],
			"error", err)
		oi.emitSecurityEvent(ctx, method, models.EventTypeAuthOrgContextMissing, models.EventTitleAuthOrgContextMissing, "invalid x-organization-id format")
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenOrgIDHeaderInvalid),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenOrgIDHeaderInvalid))
	}

	resolvedTenantID, err := oi.resolver.LookupTenant(ctx, orgUUID)
	if err != nil {
		oi.log.ErrorContext(ctx, "gRPC org interceptor: org resolution failed",
			"method", method,
			"orgID", orgUUID.String(),
			"error", err)
		oi.emitSecurityEvent(ctx, method, models.EventTypeAuthOrgResolutionFailed, models.EventTitleAuthOrgResolutionFailed, "org UUID resolution failed")
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenOrgResolutionFailed),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenOrgResolutionFailed))
	}

	if authHasIdentity {
		if existingTenant != resolvedTenantID {
			oi.log.WarnContext(ctx, "gRPC org interceptor: tenant mismatch between auth and header",
				"method", method,
				"authTenant", existingTenant,
				"headerTenant", resolvedTenantID)
			return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenIdentityMismatch),
				grpcconst.ResolveErrorMessage(grpcconst.ErrTokenIdentityMismatch))
		}

		if orgErr == nil && existingOrg != orgUUID {
			oi.log.WarnContext(ctx, "gRPC org interceptor: org mismatch between auth and header",
				"method", method,
				"authOrg", existingOrg.String(),
				"headerOrg", orgUUID.String())
			return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenIdentityMismatch),
				grpcconst.ResolveErrorMessage(grpcconst.ErrTokenIdentityMismatch))
		}

		userIDHeaders := md.Get(grpcconst.MetadataKeyUserID)

		if userErr == nil {
			if len(userIDHeaders) == 0 {
				oi.log.WarnContext(ctx, "gRPC org interceptor: missing x-user-id header for user principal",
					"method", method,
					"orgID", orgUUID.String())
				return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenUserIDHeaderRequired),
					grpcconst.ResolveErrorMessage(grpcconst.ErrTokenUserIDHeaderRequired))
			}

			userUUID, err := uuid.Parse(userIDHeaders[0])
			if err != nil {
				oi.log.WarnContext(ctx, "gRPC org interceptor: invalid x-user-id format",
					"method", method,
					"orgID", orgUUID.String(),
					"value", userIDHeaders[0],
					"error", err)
				return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenUserIDHeaderInvalid),
					grpcconst.ResolveErrorMessage(grpcconst.ErrTokenUserIDHeaderInvalid))
			}

			if userUUID.String() != existingUser {
				oi.log.WarnContext(ctx, "gRPC org interceptor: user mismatch between auth and header",
					"method", method,
					"authUser", existingUser,
					"headerUser", userUUID.String())
				return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenIdentityMismatch),
					grpcconst.ResolveErrorMessage(grpcconst.ErrTokenIdentityMismatch))
			}
		} else {
			if len(userIDHeaders) > 0 {
				oi.log.WarnContext(ctx, "gRPC org interceptor: x-user-id header not allowed for service-account principal",
					"method", method,
					"orgID", orgUUID.String(),
					"headerUser", userIDHeaders[0])
				return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenIdentityMismatch),
					grpcconst.ResolveErrorMessage(grpcconst.ErrTokenIdentityMismatch))
			}
		}

		if orgErr != nil {
			ctx = pkgcontext.WithOrganizationID(ctx, orgUUID)
		}

		oi.log.DebugContext(ctx, "gRPC org interceptor: validated auth identity against headers",
			"method", method,
			"orgID", orgUUID.String(),
			"tenantID", existingTenant)

		return ctx, nil
	}

	// No auth identity — set all values from headers
	userIDHeaders := md.Get(grpcconst.MetadataKeyUserID)
	if len(userIDHeaders) == 0 {
		oi.log.WarnContext(ctx, "gRPC org interceptor: missing x-user-id header",
			"method", method,
			"orgID", orgUUID.String())
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenUserIDHeaderRequired),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenUserIDHeaderRequired))
	}

	userUUID, err := uuid.Parse(userIDHeaders[0])
	if err != nil {
		oi.log.WarnContext(ctx, "gRPC org interceptor: invalid x-user-id format",
			"method", method,
			"orgID", orgUUID.String(),
			"value", userIDHeaders[0],
			"error", err)
		return nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenUserIDHeaderInvalid),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenUserIDHeaderInvalid))
	}

	ctx = pkgcontext.WithOrganizationID(ctx, orgUUID)
	ctx = pkgcontext.WithUserID(ctx, userUUID.String())
	ctx = pkgcontext.WithTenantID(ctx, resolvedTenantID)

	oi.log.DebugContext(ctx, "gRPC org interceptor: resolved org context",
		"method", method,
		"orgID", orgUUID.String(),
		"userID", userUUID.String(),
		"tenantID", resolvedTenantID)

	return ctx, nil
}

// orgResolvedServerStream wraps a ServerStream with org-resolved context.
type orgResolvedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *orgResolvedServerStream) Context() context.Context {
	return s.ctx
}

// emitSecurityEvent persists a security event when an event writer is configured.
func (oi *OrgResolverInterceptor) emitSecurityEvent(ctx context.Context, method, eventType, title, reason string) {
	if oi.eventWriter == nil {
		return
	}
	tenantID := oi.platformTenantID
	if tid, err := pkgcontext.GetTenantID(ctx); err == nil {
		tenantID = tid
	}
	details, _ := json.Marshal(map[string]interface{}{
		"method": method,
		"reason": reason,
	})
	_ = oi.eventWriter.CreateEvent(ctx, &models.SystemEvent{
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
