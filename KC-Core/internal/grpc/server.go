package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	audit "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// Server represents the gRPC server with HTTP/gRPC-web multiplexing
type Server struct {
	grpcServer   *grpc.Server
	httpServer   *http.Server
	healthServer *health.Server
	listener     net.Listener
	port         int
	enableTLS    bool
	tlsCert      string
	tlsKey       string
	log          logger.Logger
	serving      atomic.Bool
}

// Config holds the gRPC server configuration
type Config struct {
	Log                  logger.Logger
	Port                 int
	Host                 string // Bind address (empty = all interfaces)
	InternalTrustEnabled bool   // Trust gateway identity headers
	PeerSecret           string // Secret an internal peer presents before its identity headers are trusted (empty only on a loopback address)
	TLSCert              string
	TLSKey               string
	EnableTLS            bool
	Auth                 AuthConfig
	OrgResolver          OrganizationResolver // Optional organization resolution (nil = disabled)
	TenantResolver       TenantResolver       // Org UUID → tenant ID for JWT claims (nil = disabled)

	// Fail-closed org resolver per SCACI §3.10.
	// When set, enforces x-organization-id and x-user-id metadata on all calls
	// except public methods (health, reflection). Resolves org UUID → tenant ID.
	// nil = disabled (community/dev mode without strict org enforcement)
	FailClosedOrgResolver org.Resolver

	// DefaultTenantID for community single-tenant fallback (0 = disabled).
	// When FailClosedOrgResolver is nil and DefaultTenantID > 0, standalone mode
	// injects this tenant and its default organization into context for all requests.
	DefaultTenantID int64
	// DefaultOrgResolver supplies the organization every community-mode request
	// runs under (standalone and internal-trust paths); required in community mode.
	DefaultOrgResolver interceptors.DefaultOrgResolver

	// API key bearer auth lookup (nil = disabled).
	// Must implement interceptors.APIKeyAuthenticator (use NewCoreAPIKeyAdapter to wrap KC-DB repos).
	APIKeyAuth interceptors.APIKeyAuthenticator

	// RoleSource resolves the caller's roles for the method policy every non-public RPC must satisfy (required)
	RoleSource interceptors.RoleSource

	// EventWriter persists security events from interceptors (nil = no persistence)
	EventWriter audit.EventWriter
	// PlatformTenantID fallback tenant for pre-auth security events
	PlatformTenantID int64

	// AdminChecker exempts server admins from the fail-closed org-mismatch check (nil = no bypass)
	AdminChecker interceptors.AdminChecker

	// gRPC-web configuration
	GRPCWeb config.GRPCWebConfig

	// HTTP server timeouts
	HTTPConfig HTTPServerConfig

	// Reflection and health toggles
	EnableReflection bool
	EnableHealth     bool
}

// HTTPServerConfig mirrors the config.HTTPServerConfig for server initialization
type HTTPServerConfig struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// NewServer creates a new gRPC server instance with gRPC-web multiplexing
func NewServer(cfg Config) (*Server, error) {
	if cfg.Log == nil {
		return nil, errors.New(errMsgLoggerCannotBeNil)
	}
	log := cfg.Log

	// Validate TLS configuration — fail early if EnableTLS but certs missing
	if cfg.EnableTLS {
		if cfg.TLSCert == "" || cfg.TLSKey == "" {
			return nil, fmt.Errorf(errFmtTLSCertOrKeyPathMissing, cfg.TLSCert, cfg.TLSKey)
		}
	}

	// The interceptor chain is validated before the port is taken, so a
	// misconfiguration never leaves a listener behind.
	unaryInterceptors, streamInterceptors, err := buildInterceptorChain(cfg, log)
	if err != nil {
		return nil, err
	}

	// Create listener — host may be empty (all interfaces) or specific (e.g., 127.0.0.1 for internal trust)
	listenAddr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf(errFmtFailedToListen, listenAddr, err)
	}

	// Create gRPC server options (no TLS credentials - handled at HTTP layer)
	var opts []grpc.ServerOption

	// OTel stats handler for distributed tracing (before interceptor chain)
	opts = append(opts, grpc.StatsHandler(otelgrpc.NewServerHandler()))

	// Chain interceptors: auth → org resolver (if enabled) → logging
	opts = append(
		opts,
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
	)

	// Create gRPC server (no TLS credentials - handled at HTTP layer)
	grpcServer := grpc.NewServer(opts...)

	// Register health service
	var healthSvc *health.Server
	if cfg.EnableHealth {
		healthSvc = health.NewServer()
		healthpb.RegisterHealthServer(grpcServer, healthSvc)
		// Set initial serving status
		healthSvc.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
		healthSvc.SetServingStatus(pb.KiloCenterService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)
		healthSvc.SetServingStatus(pb.CoreService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)
		log.Info(LogGRPCHealthServiceRegistered)
	}

	// Enable reflection (conditional)
	if cfg.EnableReflection {
		reflection.Register(grpcServer)
		log.Info(LogGRPCReflectionEnabled)
	}

	if cfg.GRPCWeb.Enabled {
		log.Info(LogGRPCWebWrapperConfigured, logger.FieldAllowAllOrigins, cfg.GRPCWeb.AllowAllOrigins)
	}

	httpServer := createHTTPServer(grpcServer, cfg, log)

	return &Server{
		grpcServer:   grpcServer,
		httpServer:   httpServer,
		healthServer: healthSvc,
		listener:     listener,
		port:         cfg.Port,
		enableTLS:    cfg.EnableTLS,
		tlsCert:      cfg.TLSCert,
		tlsKey:       cfg.TLSKey,
		log:          log,
	}, nil
}

// Start serves until the server fails or Stop is called; a deliberate Stop returns nil.
func (s *Server) Start() error {
	s.log.Info(LogStartingGRPCServerWithGRPCWebMultiplexing, logger.FieldPort, s.port, logger.FieldTLS, s.enableTLS)

	s.httpServer.Addr = fmt.Sprintf(":%d", s.port)

	s.serving.Store(true)
	defer s.serving.Store(false)

	var err error
	if s.enableTLS && s.tlsCert != "" && s.tlsKey != "" {
		s.log.Info(LogGRPCServerUsingTLS, logger.FieldCert, s.tlsCert)
		err = s.httpServer.ServeTLS(s.listener, s.tlsCert, s.tlsKey)
	} else {
		err = s.httpServer.Serve(s.listener)
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Listening reports whether the server is serving calls.
func (s *Server) Listening() bool {
	return s.serving.Load()
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	s.log.Info(LogStoppingGRPCServer)
	s.serving.Store(false)

	// Mark health as not serving before shutdown
	if s.healthServer != nil {
		s.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
		s.healthServer.SetServingStatus(pb.KiloCenterService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_NOT_SERVING)
		s.healthServer.SetServingStatus(pb.CoreService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_NOT_SERVING)
	}

	// Graceful HTTP shutdown (includes gRPC)
	ctx, cancel := context.WithTimeout(context.Background(), s.httpServer.WriteTimeout) // context-root: shutdown
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.log.Error(LogHTTPServerShutdownError, logger.FieldError, err)
		// Force close if graceful fails
		s.grpcServer.Stop()
	}
}

// GetServer returns the underlying gRPC server for service registration
func (s *Server) GetServer() *grpc.Server {
	return s.grpcServer
}

// unaryInterceptor provides logging and error handling for unary RPCs
func unaryInterceptor(log logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		log.DebugContext(ctx, LogGRPCUnaryCall, logger.FieldMethod, info.FullMethod)

		resp, err := handler(ctx, req)
		if err != nil {
			log.ErrorContext(ctx, LogGRPCUnaryCallFailed, logger.FieldMethod, info.FullMethod, logger.FieldError, err)
		}

		return resp, err
	}
}

// communityContext stamps the community-mode identity, the default tenant and
// its default organization, onto every request of the standalone path.
type communityContext struct {
	tenantID int64
	orgs     interceptors.DefaultOrgResolver
}

func newCommunityContext(tenantID int64, orgs interceptors.DefaultOrgResolver) communityContext {
	return communityContext{tenantID: tenantID, orgs: orgs}
}

func (c communityContext) apply(ctx context.Context) (context.Context, error) {
	ctx = pkgcontext.WithTenantID(ctx, c.tenantID)
	orgID, err := c.orgs.GetDefaultOrgForTenant(ctx, c.tenantID)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgResolutionFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgResolutionFailed))
	}
	return pkgcontext.WithOrganizationID(ctx, orgID), nil
}

func (c communityContext) unaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx, err := c.apply(ctx)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (c communityContext) streamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, err := c.apply(ss.Context())
		if err != nil {
			return err
		}
		return handler(srv, &communityTenantStream{ServerStream: ss, ctx: ctx})
	}
}

// communityTenantStream wraps a ServerStream to override its context with the
// community-mode identity.
type communityTenantStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *communityTenantStream) Context() context.Context {
	return s.ctx
}

// streamInterceptor provides logging and error handling for streaming RPCs
func streamInterceptor(log logger.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		log.DebugContext(ss.Context(), LogGRPCStreamCall, logger.FieldMethod, info.FullMethod)

		err := handler(srv, ss)
		if err != nil {
			log.ErrorContext(ss.Context(), LogGRPCStreamCallFailed, logger.FieldMethod, info.FullMethod, logger.FieldError, err)
		}

		return err
	}
}
