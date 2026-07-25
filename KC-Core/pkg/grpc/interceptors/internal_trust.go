package interceptors

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// InternalTrustInterceptor reads gateway-injected identity headers and populates context.
// Fail-closed: rejects requests missing the internal tenant header.
// When communityMode is true, org header is optional for all methods (single-tenant).
type InternalTrustInterceptor struct {
	log              logger.Logger
	communityMode    bool
	eventWriter      grpcconst.EventWriter
	platformTenantID int64
}

// NewInternalTrustInterceptor creates a fail-closed internal trust interceptor.
// When communityMode is true, org header is treated as optional for all methods.
func NewInternalTrustInterceptor(log logger.Logger, communityMode bool) *InternalTrustInterceptor {
	if log == nil {
		log = logger.Get()
	}
	return &InternalTrustInterceptor{log: log, communityMode: communityMode}
}

// WithEventWriter sets the security event writer for trust violation persistence.
func (it *InternalTrustInterceptor) WithEventWriter(w grpcconst.EventWriter) *InternalTrustInterceptor {
	it.eventWriter = w
	return it
}

// WithPlatformTenantID sets the fallback tenant for pre-auth events.
func (it *InternalTrustInterceptor) WithPlatformTenantID(id int64) *InternalTrustInterceptor {
	it.platformTenantID = id
	return it
}

// UnaryInterceptor returns the unary server interceptor for internal trust validation.
func (it *InternalTrustInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Public methods bypass trust validation
		if grpcconst.IsPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		newCtx, err := it.extractIdentity(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}
		return handler(newCtx, req)
	}
}

// StreamInterceptor returns the stream server interceptor for internal trust validation.
func (it *InternalTrustInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if grpcconst.IsPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}

		newCtx, err := it.extractIdentity(ss.Context(), info.FullMethod)
		if err != nil {
			return err
		}

		wrappedStream := &trustedServerStream{
			ServerStream: ss,
			ctx:          newCtx,
		}
		return handler(srv, wrappedStream)
	}
}

// extractIdentity reads x-kc-internal-* headers from gateway and populates context.
func (it *InternalTrustInterceptor) extractIdentity(ctx context.Context, method string) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		it.emitSecurityEvent(ctx, method, "missing gRPC metadata")
		return nil, status.Error(codes.Unauthenticated,
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenMissingMetadata))
	}

	// Tenant ID is always required
	tenantHeaders := md.Get(grpcconst.MetadataKeyInternalTenantID)
	if len(tenantHeaders) == 0 {
		it.log.WarnContext(ctx, "internal trust: missing tenant header",
			"method", method)
		it.emitSecurityEvent(ctx, method, "missing internal tenant header")
		return nil, status.Error(codes.Unauthenticated,
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenGatewayInternalTrustInvalidHeader))
	}

	tenantID, err := strconv.ParseInt(tenantHeaders[0], 10, 64)
	if err != nil || tenantID <= 0 {
		it.log.WarnContext(ctx, "internal trust: invalid tenant header",
			"method", method, "value", tenantHeaders[0])
		it.emitSecurityEvent(ctx, method, "invalid internal tenant header")
		return nil, status.Error(codes.Unauthenticated,
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenGatewayInternalTrustInvalidHeader))
	}
	ctx = pkgcontext.WithTenantID(ctx, tenantID)

	// Org ID — required for non-exempt methods, optional for exempt methods.
	// In community mode, org header is optional for all methods.
	orgHeaders := md.Get(grpcconst.MetadataKeyInternalOrgID)
	if it.communityMode || grpcconst.IsOrgExemptMethod(method) {
		// Org header optional for exempt methods (e.g., GetSystemStatus)
		if len(orgHeaders) > 0 {
			orgUUID, err := uuid.Parse(orgHeaders[0])
			if err != nil {
				it.log.WarnContext(ctx, "internal trust: invalid org header",
					"method", method, "value", orgHeaders[0])
				it.emitSecurityEvent(ctx, method, "invalid internal org header")
				return nil, status.Error(codes.Unauthenticated,
					grpcconst.ResolveErrorMessage(grpcconst.ErrTokenGatewayInternalTrustInvalidHeader))
			}
			ctx = pkgcontext.WithOrganizationID(ctx, orgUUID)
		}
	} else {
		// Org header REQUIRED for all non-exempt methods (fail-closed)
		if len(orgHeaders) == 0 {
			it.log.WarnContext(ctx, "internal trust: missing org header for non-exempt method",
				"method", method)
			it.emitSecurityEvent(ctx, method, "missing internal org header")
			return nil, status.Error(codes.Unauthenticated,
				grpcconst.ResolveErrorMessage(grpcconst.ErrTokenGatewayInternalTrustInvalidHeader))
		}
		orgUUID, err := uuid.Parse(orgHeaders[0])
		if err != nil {
			it.log.WarnContext(ctx, "internal trust: invalid org header",
				"method", method, "value", orgHeaders[0])
			it.emitSecurityEvent(ctx, method, "invalid internal org header")
			return nil, status.Error(codes.Unauthenticated,
				grpcconst.ResolveErrorMessage(grpcconst.ErrTokenGatewayInternalTrustInvalidHeader))
		}
		ctx = pkgcontext.WithOrganizationID(ctx, orgUUID)
	}

	// User ID — optional (service accounts may not have one)
	userHeaders := md.Get(grpcconst.MetadataKeyInternalUserID)
	if len(userHeaders) > 0 && userHeaders[0] != "" {
		if _, err := uuid.Parse(userHeaders[0]); err != nil {
			it.log.WarnContext(ctx, "internal trust: invalid user header",
				"method", method, "value", userHeaders[0])
			it.emitSecurityEvent(ctx, method, "invalid internal user header")
			return nil, status.Error(codes.Unauthenticated,
				grpcconst.ResolveErrorMessage(grpcconst.ErrTokenGatewayInternalTrustInvalidHeader))
		}
		ctx = pkgcontext.WithUserID(ctx, userHeaders[0])
	}

	it.log.DebugContext(ctx, "internal trust: identity extracted",
		"method", method,
		"tenantID", tenantID)

	return ctx, nil
}

type trustedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *trustedServerStream) Context() context.Context {
	return s.ctx
}

// emitSecurityEvent persists an internal trust violation event when an event writer is configured.
func (it *InternalTrustInterceptor) emitSecurityEvent(ctx context.Context, method, reason string) {
	if it.eventWriter == nil {
		return
	}
	tenantID := it.platformTenantID
	if tid, err := pkgcontext.GetTenantID(ctx); err == nil {
		tenantID = tid
	}
	details, _ := json.Marshal(map[string]interface{}{
		auditKeyMethod: method,
		auditKeyReason: reason,
	})
	_ = it.eventWriter.CreateEvent(ctx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(tenantID, 10),
		EventType:   models.EventTypeAuthInternalTrustViolation,
		Category:    models.EventCategorySecurity,
		Severity:    models.EventSeverityWarning,
		Title:       models.EventTitleAuthInternalTrustViolation,
		Description: reason,
		SourceType:  models.SourceTypeAPI,
		SourceName:  method,
		Details:     details,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
}
