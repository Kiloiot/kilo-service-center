package interceptors

import (
	"context"
	"strconv"

	audit "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"

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
	defaultOrgs      DefaultOrgResolver
	peers            PeerAuthenticator
	eventWriter      audit.EventWriter
	platformTenantID int64
}

// DefaultOrgResolver supplies the organization a community-mode request runs
// under when the gateway sends none, so handlers never see a missing one.
type DefaultOrgResolver interface {
	GetDefaultOrgForTenant(ctx context.Context, tenantID int64) (uuid.UUID, error)
}

// NewInternalTrustInterceptor creates a fail-closed internal trust interceptor.
// When communityMode is true the org header may be omitted; the default
// organization of the tenant is then resolved through WithDefaultOrgResolver.
func NewInternalTrustInterceptor(log logger.Logger, communityMode bool) *InternalTrustInterceptor {
	if log == nil {
		log = logger.Get()
	}
	return &InternalTrustInterceptor{log: log, communityMode: communityMode}
}

// WithDefaultOrgResolver sets the resolver that fills in the organization for
// community-mode requests that carry no org header.
func (it *InternalTrustInterceptor) WithDefaultOrgResolver(r DefaultOrgResolver) *InternalTrustInterceptor {
	it.defaultOrgs = r
	return it
}

// WithPeerSecret requires the peer secret before identity headers are trusted; empty, allowed only on a loopback address, disables the check.
func (it *InternalTrustInterceptor) WithPeerSecret(secret string) *InternalTrustInterceptor {
	it.peers = NewPeerAuthenticator(secret)
	return it
}

// WithEventWriter sets the security event writer for trust violation persistence.
func (it *InternalTrustInterceptor) WithEventWriter(w audit.EventWriter) *InternalTrustInterceptor {
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

		newCtx, err := it.admit(ctx, info.FullMethod)
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

		newCtx, err := it.admit(ss.Context(), info.FullMethod)
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

// admit trusts the identity headers of a request only after its peer has
// authenticated.
func (it *InternalTrustInterceptor) admit(ctx context.Context, method string) (context.Context, error) {
	if err := it.peers.Authenticate(ctx); err != nil {
		it.log.WarnContext(ctx, LogInternalTrustPeerRejected, logger.FieldMethod, method)
		it.emitSecurityEvent(ctx, method, detailInternalPeerRejected)
		return nil, err
	}
	return it.extractIdentity(ctx, method)
}

// extractIdentity reads the x-kc-internal-* headers the gateway sent into the
// request context: the tenant always, the organization per withOrganization,
// and the caller per withPrincipal.
func (it *InternalTrustInterceptor) extractIdentity(ctx context.Context, method string) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		it.emitSecurityEvent(ctx, method, detailMissingGRPCMetadata)
		return nil, status.Error(codes.Unauthenticated,
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenMissingMetadata))
	}

	tenantID, err := it.tenantFrom(ctx, md, method)
	if err != nil {
		return nil, err
	}
	ctx = pkgcontext.WithTenantID(ctx, tenantID)

	if ctx, err = it.withOrganization(ctx, md, method, tenantID); err != nil {
		return nil, err
	}
	if ctx, err = it.withPrincipal(ctx, md, method); err != nil {
		return nil, err
	}

	it.log.DebugContext(ctx, LogInternalTrustIdentityExtracted,
		logger.FieldMethod, method,
		logger.FieldTenantID, tenantID)
	return ctx, nil
}

// tenantFrom reads the required, positive tenant header.
func (it *InternalTrustInterceptor) tenantFrom(ctx context.Context, md metadata.MD, method string) (int64, error) {
	tenantHeaders := md.Get(grpcconst.MetadataKeyInternalTenantID)
	if len(tenantHeaders) == 0 {
		return 0, it.rejectHeader(ctx, method, LogInternalTrustMissingTenantHeader, detailMissingInternalTenantHeader)
	}
	tenantID, err := strconv.ParseInt(tenantHeaders[0], 10, 64)
	if err != nil || tenantID <= 0 {
		return 0, it.rejectHeader(ctx, method, LogInternalTrustInvalidTenantHeader, detailInvalidInternalTenantHeader,
			logger.FieldValue, tenantHeaders[0])
	}
	return tenantID, nil
}

// withOrganization sets the organization of the org header. The header is
// required (fail-closed) except for org-exempt methods (e.g. GetSystemStatus)
// and in community mode, where a missing one resolves to the tenant's
// default organization.
func (it *InternalTrustInterceptor) withOrganization(ctx context.Context, md metadata.MD, method string, tenantID int64) (context.Context, error) {
	orgHeaders := md.Get(grpcconst.MetadataKeyInternalOrgID)
	if len(orgHeaders) > 0 {
		orgUUID, err := uuid.Parse(orgHeaders[0])
		if err != nil {
			return nil, it.rejectHeader(ctx, method, LogInternalTrustInvalidOrgHeader, detailInvalidInternalOrgHeader,
				logger.FieldValue, orgHeaders[0])
		}
		return pkgcontext.WithOrganizationID(ctx, orgUUID), nil
	}
	switch {
	case it.communityMode:
		orgUUID, err := it.resolveDefaultOrg(ctx, method, tenantID)
		if err != nil {
			return nil, err
		}
		return pkgcontext.WithOrganizationID(ctx, orgUUID), nil
	case grpcconst.IsOrgExemptMethod(method):
		return ctx, nil
	default:
		return nil, it.rejectHeader(ctx, method, LogInternalTrustMissingOrgHeaderForNonExempt, detailMissingInternalOrgHeader)
	}
}

// withPrincipal sets the caller from the optional user or service-account
// header: at most one of them, each a UUID.
func (it *InternalTrustInterceptor) withPrincipal(ctx context.Context, md metadata.MD, method string) (context.Context, error) {
	user := firstHeader(md, grpcconst.MetadataKeyInternalUserID)
	serviceAccount := firstHeader(md, grpcconst.MetadataKeyInternalServiceAccountID)
	switch {
	case user != "" && serviceAccount != "":
		return nil, it.rejectHeader(ctx, method, LogInternalTrustConflictingPrincipalHeaders, detailConflictingInternalPrincipalHeaders)
	case user != "":
		if _, err := uuid.Parse(user); err != nil {
			return nil, it.rejectHeader(ctx, method, LogInternalTrustInvalidUserHeader, detailInvalidInternalUserHeader,
				logger.FieldValue, user)
		}
		return pkgcontext.WithUserID(ctx, user), nil
	case serviceAccount != "":
		keyID, err := uuid.Parse(serviceAccount)
		if err != nil {
			return nil, it.rejectHeader(ctx, method, LogInternalTrustInvalidServiceAccountHeader, detailInvalidInternalServiceAccountHeader,
				logger.FieldValue, serviceAccount)
		}
		return pkgcontext.WithServiceAccountID(ctx, keyID), nil
	}
	return ctx, nil
}

func firstHeader(md metadata.MD, key string) string {
	if values := md.Get(key); len(values) > 0 {
		return values[0]
	}
	return ""
}

// rejectHeader logs and records a missing or malformed identity header and
// returns the fail-closed Unauthenticated status.
func (it *InternalTrustInterceptor) rejectHeader(ctx context.Context, method, logMsg, detail string, fields ...interface{}) error {
	it.log.WarnContext(ctx, logMsg, append([]interface{}{logger.FieldMethod, method}, fields...)...)
	it.emitSecurityEvent(ctx, method, detail)
	return status.Error(codes.Unauthenticated,
		grpcconst.ResolveErrorMessage(grpcconst.ErrTokenGatewayInternalTrustInvalidHeader))
}

// emitSecurityEvent persists an internal trust violation event when an event writer is configured.
func (it *InternalTrustInterceptor) emitSecurityEvent(ctx context.Context, method, reason string) {
	recordSecurityEvent(ctx, it.eventWriter, it.platformTenantID, it.log, securityEvent{
		method: method, eventType: models.EventTypeAuthInternalTrustViolation,
		title: models.EventTitleAuthInternalTrustViolation, reason: reason,
	})
}

// resolveDefaultOrg fills the organization of a community-mode request from
// the tenant's default organization; without a resolver or on a resolver
// failure the request is refused rather than run organization-less.
func (it *InternalTrustInterceptor) resolveDefaultOrg(ctx context.Context, method string, tenantID int64) (uuid.UUID, error) {
	if it.defaultOrgs == nil {
		it.log.ErrorContext(ctx, LogInternalTrustDefaultOrgResolverMissing, logger.FieldMethod, method)
		return uuid.Nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenOrgResolverRequired),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenOrgResolverRequired))
	}
	orgUUID, err := it.defaultOrgs.GetDefaultOrgForTenant(ctx, tenantID)
	if err != nil {
		it.log.ErrorContext(ctx, LogInternalTrustDefaultOrgResolutionFailed,
			logger.FieldMethod, method, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return uuid.Nil, status.Error(grpcconst.GetGRPCCode(grpcconst.ErrTokenOrgResolutionFailed),
			grpcconst.ResolveErrorMessage(grpcconst.ErrTokenOrgResolutionFailed))
	}
	return orgUUID, nil
}
