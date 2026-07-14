package grpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/net/http2"     //nolint:staticcheck // h2c cleartext HTTP/2 support
	"golang.org/x/net/http2/h2c" //nolint:staticcheck // h2c deprecation acknowledged; migration deferred
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server represents the gRPC server with HTTP/gRPC-web multiplexing
type Server struct {
	grpcServer     *grpc.Server
	httpServer     *http.Server
	grpcWebWrapped *grpcweb.WrappedGrpcServer
	healthServer   *health.Server
	listener       net.Listener
	port           int
	enableTLS      bool
	tlsCert        string
	tlsKey         string
	log            logger.Logger
}

// Config holds the gRPC server configuration
type Config struct {
	Port                 int
	Host                 string // Bind address (empty = all interfaces)
	InternalTrustEnabled bool   // Trust gateway identity headers
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
	// injects this tenant into context for all requests.
	DefaultTenantID int64

	// API key bearer auth lookup (nil = disabled).
	// Must implement interceptors.APIKeyAuthenticator (use NewCoreAPIKeyAdapter to wrap KC-DB repos).
	APIKeyAuth interceptors.APIKeyAuthenticator

	// Optional RBAC interceptor (nil = no role-based authorization)
	RoleResolver interceptors.RoleResolver
	// RBACPolicies maps gRPC method paths to allowed roles (nil = no RBAC)
	RBACPolicies map[string][]string
	// RBACRoleCacheTTL controls how long resolved RBAC roles are cached (0 = default 30s)
	RBACRoleCacheTTL time.Duration

	// EventWriter persists security events from interceptors (nil = no persistence)
	EventWriter grpcerrors.EventWriter
	// PlatformTenantID fallback tenant for pre-auth security events
	PlatformTenantID int64

	// AdminChecker exempts server admins from the fail-closed org-mismatch check (nil = no bypass)
	AdminChecker interceptors.AdminChecker

	// gRPC-web configuration
	GRPCWeb WebConfig

	// HTTP server timeouts
	HTTPConfig HTTPServerConfig

	// Reflection and health toggles
	EnableReflection bool
	EnableHealth     bool
}

// WebConfig mirrors the config.GRPCWebConfig for server initialization.
type WebConfig struct {
	Enabled          bool
	AllowedOrigins   []string
	AllowedHeaders   []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
	AllowAllOrigins  bool
	EnableWebsockets bool
	AllowedMethods   []string
}

// HTTPServerConfig mirrors the config.HTTPServerConfig for server initialization
type HTTPServerConfig struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// NewServer creates a new gRPC server instance with gRPC-web multiplexing
func NewServer(cfg Config) (*Server, error) {
	log := logger.Get()

	// Validate TLS configuration — fail early if EnableTLS but certs missing
	if cfg.EnableTLS {
		if cfg.TLSCert == "" || cfg.TLSKey == "" {
			return nil, fmt.Errorf("TLS enabled but certificate or key path is missing: cert=%q, key=%q", cfg.TLSCert, cfg.TLSKey)
		}
	}

	// Create listener — host may be empty (all interfaces) or specific (e.g., 127.0.0.1 for internal trust)
	listenAddr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", listenAddr, err)
	}

	// Create gRPC server options (no TLS credentials - handled at HTTP layer)
	var opts []grpc.ServerOption

	var unaryInterceptors []grpc.UnaryServerInterceptor
	var streamInterceptors []grpc.StreamServerInterceptor

	if cfg.InternalTrustEnabled {
		// Gateway mode: trust identity headers from KC-Gateway, skip auth + org resolver.
		// Community mode (no org enforcement) makes org header optional for all methods.
		communityMode := cfg.FailClosedOrgResolver == nil
		trustInterceptor := interceptors.NewInternalTrustInterceptor(log, communityMode).
			WithEventWriter(cfg.EventWriter).
			WithPlatformTenantID(cfg.PlatformTenantID)
		unaryInterceptors = append(unaryInterceptors, trustInterceptor.UnaryInterceptor())
		streamInterceptors = append(streamInterceptors, trustInterceptor.StreamInterceptor())
		if communityMode {
			log.Info("SECURITY: Running in internal-only community mode (org enforcement disabled). Trusting gateway identity headers.")
		} else {
			log.Info("SECURITY: Running in internal-only mode, trusting gateway identity headers. Ensure this port is not externally accessible.")
		}
	} else {
		// Standalone mode: full auth + org resolver (same as before)
		authInterceptor, err := NewAuthInterceptor(cfg.Auth)
		if err != nil {
			_ = listener.Close()
			return nil, fmt.Errorf("failed to create auth interceptor: %w", err)
		}

		if cfg.OrgResolver != nil {
			authInterceptor.WithOrganizationResolver(cfg.OrgResolver)
			log.Info("gRPC auth interceptor configured with organization resolution")
		}

		if cfg.TenantResolver != nil {
			authInterceptor.WithTenantResolver(cfg.TenantResolver)
			log.Info("gRPC auth interceptor configured with tenant resolution for org UUID claims")
		}

		if cfg.APIKeyAuth != nil {
			authInterceptor.WithAPIKeyAuthenticator(cfg.APIKeyAuth)
			log.Info("gRPC auth interceptor configured with API key bearer auth")
		}

		unaryInterceptors = append(unaryInterceptors, authInterceptor.UnaryInterceptor())
		streamInterceptors = append(streamInterceptors, authInterceptor.StreamInterceptor())

		// Add fail-closed org resolver if configured.
		// Skip list sourced from canonical OrgExemptMethods in pkg/grpc/public_methods.go.
		if cfg.FailClosedOrgResolver != nil {
			skipMethods := make([]string, 0, len(grpcerrors.OrgExemptMethods))
			for method := range grpcerrors.OrgExemptMethods {
				skipMethods = append(skipMethods, method)
			}
			orgInterceptor, err := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
				Resolver:         cfg.FailClosedOrgResolver,
				Logger:           log,
				SkipMethods:      skipMethods,
				EventWriter:      cfg.EventWriter,
				PlatformTenantID: cfg.PlatformTenantID,
				AdminChecker:     cfg.AdminChecker,
			})
			if err != nil {
				_ = listener.Close()
				return nil, fmt.Errorf("failed to create org resolver interceptor: %w", err)
			}
			unaryInterceptors = append(unaryInterceptors, orgInterceptor.UnaryInterceptor())
			streamInterceptors = append(streamInterceptors, orgInterceptor.StreamInterceptor())
			log.Info("gRPC server configured with fail-closed org resolver interceptor")
		} else if cfg.DefaultTenantID > 0 {
			// Community standalone mode: inject default tenant for all requests
			communityUnary := func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
				ctx = pkgcontext.WithTenantID(ctx, cfg.DefaultTenantID)
				return handler(ctx, req)
			}
			communityStream := func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
				wrapped := &communityTenantStream{ServerStream: ss, ctx: pkgcontext.WithTenantID(ss.Context(), cfg.DefaultTenantID)}
				return handler(srv, wrapped)
			}
			unaryInterceptors = append(unaryInterceptors, communityUnary)
			streamInterceptors = append(streamInterceptors, communityStream)
			log.Info("Community mode: injecting default tenant for standalone", "tenantID", cfg.DefaultTenantID)
		}
	}

	// Add RBAC authorization interceptor if policies are configured
	if len(cfg.RBACPolicies) > 0 {
		if cfg.RoleResolver == nil {
			_ = listener.Close()
			return nil, fmt.Errorf("RBAC policies configured but RoleResolver is nil")
		}
		authzInterceptor := interceptors.NewAuthorizationInterceptor(cfg.RoleResolver, cfg.RBACPolicies, log, cfg.RBACRoleCacheTTL).
			WithEventWriter(cfg.EventWriter).
			WithPlatformTenantID(cfg.PlatformTenantID)
		unaryInterceptors = append(unaryInterceptors, authzInterceptor.UnaryInterceptor())
		streamInterceptors = append(streamInterceptors, authzInterceptor.StreamInterceptor())
		log.Info("RBAC authorization interceptor enabled", "policyCount", len(cfg.RBACPolicies))
	}

	// Add logging interceptor last (both modes)
	unaryInterceptors = append(unaryInterceptors, unaryInterceptor(log))
	streamInterceptors = append(streamInterceptors, streamInterceptor(log))

	// OTel stats handler for distributed tracing (before interceptor chain)
	opts = append(opts, grpc.StatsHandler(otelgrpc.NewServerHandler()))

	// Chain interceptors: auth → org resolver (if enabled) → logging
	opts = append(opts,
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
		healthSvc.SetServingStatus("kilocenter.api.v1.KiloCenterService", healthpb.HealthCheckResponse_SERVING)
		healthSvc.SetServingStatus("kilocenter.api.v1.CoreService", healthpb.HealthCheckResponse_SERVING)
		log.Info("gRPC health service registered")
	}

	// Enable reflection (conditional)
	if cfg.EnableReflection {
		reflection.Register(grpcServer)
		log.Info("gRPC reflection enabled")
	}

	// Create gRPC-web wrapper
	var grpcWebWrapped *grpcweb.WrappedGrpcServer
	if cfg.GRPCWeb.Enabled {
		grpcWebWrapped = createGRPCWebWrapper(grpcServer, cfg.GRPCWeb, log)
		log.Info("gRPC-web wrapper configured",
			"websockets", cfg.GRPCWeb.EnableWebsockets,
			"allowAllOrigins", cfg.GRPCWeb.AllowAllOrigins)
	}

	// Create HTTP server for multiplexing
	httpServer := createHTTPServer(grpcServer, grpcWebWrapped, cfg, log)

	return &Server{
		grpcServer:     grpcServer,
		httpServer:     httpServer,
		grpcWebWrapped: grpcWebWrapped,
		healthServer:   healthSvc,
		listener:       listener,
		port:           cfg.Port,
		enableTLS:      cfg.EnableTLS,
		tlsCert:        cfg.TLSCert,
		tlsKey:         cfg.TLSKey,
		log:            log,
	}, nil
}

// createGRPCWebWrapper creates the grpc-web wrapper with CORS configuration
func createGRPCWebWrapper(grpcServer *grpc.Server, cfg WebConfig, _ logger.Logger) *grpcweb.WrappedGrpcServer {
	// Build origin check function
	originFunc := func(origin string) bool {
		if cfg.AllowAllOrigins {
			return true
		}
		for _, allowed := range cfg.AllowedOrigins {
			if origin == allowed {
				return true
			}
		}
		return false
	}

	// Use defaults from constants if not configured
	allowedHeaders := cfg.AllowedHeaders
	if len(allowedHeaders) == 0 {
		allowedHeaders = grpcerrors.GRPCWebAllowedHeaders
	}

	opts := []grpcweb.Option{
		grpcweb.WithOriginFunc(originFunc),
		grpcweb.WithAllowedRequestHeaders(allowedHeaders),
	}

	if cfg.EnableWebsockets {
		opts = append(opts,
			grpcweb.WithWebsockets(true),
			grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool {
				return originFunc(req.Header.Get("Origin"))
			}),
		)
	}

	return grpcweb.WrapServer(grpcServer, opts...)
}

// createHTTPServer creates the multiplexing HTTP server
func createHTTPServer(grpcServer *grpc.Server, grpcWebWrapped *grpcweb.WrappedGrpcServer, cfg Config, _ logger.Logger) *http.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route decision order:
		// 1. gRPC-web request (if enabled)
		// 2. gRPC-web websocket (if enabled)
		// 3. Native gRPC (HTTP/2 + application/grpc)
		// 4. CORS preflight
		// 5. 404

		if grpcWebWrapped != nil {
			if grpcWebWrapped.IsGrpcWebRequest(r) || grpcWebWrapped.IsGrpcWebSocketRequest(r) {
				grpcWebWrapped.ServeHTTP(w, r)
				return
			}
		}

		// Native gRPC: HTTP/2 with application/grpc content-type
		contentType := r.Header.Get("Content-Type")
		if r.ProtoMajor == 2 && strings.HasPrefix(contentType, "application/grpc") {
			grpcServer.ServeHTTP(w, r)
			return
		}

		// CORS preflight for gRPC-web
		if r.Method == http.MethodOptions && grpcWebWrapped != nil {
			handleCORSPreflight(w, r, cfg.GRPCWeb)
			return
		}

		// No REST - return 404
		http.NotFound(w, r)
	})

	// For non-TLS, wrap with h2c to support HTTP/2 cleartext
	var finalHandler http.Handler
	if !cfg.EnableTLS {
		h2s := &http2.Server{}
		finalHandler = h2c.NewHandler(handler, h2s) //nolint:staticcheck // h2c deprecation acknowledged; migration deferred
	} else {
		finalHandler = handler
	}

	return &http.Server{
		Handler:      finalHandler,
		ReadTimeout:  cfg.HTTPConfig.ReadTimeout,
		WriteTimeout: cfg.HTTPConfig.WriteTimeout,
		IdleTimeout:  cfg.HTTPConfig.IdleTimeout,
	}
}

// handleCORSPreflight handles CORS preflight requests for gRPC-web
func handleCORSPreflight(w http.ResponseWriter, r *http.Request, cfg WebConfig) {
	origin := r.Header.Get("Origin")

	// Check origin
	allowed := cfg.AllowAllOrigins
	if !allowed {
		for _, o := range cfg.AllowedOrigins {
			if origin == o {
				allowed = true
				break
			}
		}
	}

	if !allowed {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Set Vary header to prevent cache poisoning
	w.Header().Set("Vary", "Origin")

	// Echo the allowed origin (not wildcard when credentials are allowed)
	w.Header().Set("Access-Control-Allow-Origin", origin)

	// Use configured methods or empty slice (grpcweb handles defaults)
	if len(cfg.AllowedMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
	}

	allowedHeaders := cfg.AllowedHeaders
	if len(allowedHeaders) == 0 {
		allowedHeaders = grpcerrors.GRPCWebAllowedHeaders
	}
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))

	exposeHeaders := cfg.ExposeHeaders
	if len(exposeHeaders) == 0 {
		exposeHeaders = grpcerrors.GRPCWebExposeHeaders
	}
	w.Header().Set("Access-Control-Expose-Headers", strings.Join(exposeHeaders, ", "))

	if cfg.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if cfg.MaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.MaxAge))
	}

	w.WriteHeader(http.StatusNoContent)
}

// Start starts the gRPC server with HTTP multiplexing
func (s *Server) Start() error {
	s.log.Info("Starting gRPC server with gRPC-web multiplexing", "port", s.port, "tls", s.enableTLS)

	s.httpServer.Addr = fmt.Sprintf(":%d", s.port)

	// TLS at HTTP layer - use ServeTLS when enabled
	if s.enableTLS && s.tlsCert != "" && s.tlsKey != "" {
		s.log.Info("gRPC server using TLS", "cert", s.tlsCert)
		return s.httpServer.ServeTLS(s.listener, s.tlsCert, s.tlsKey)
	}

	// Non-TLS: h2c handler already configured for HTTP/2 cleartext
	return s.httpServer.Serve(s.listener)
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	s.log.Info("Stopping gRPC server")

	// Mark health as not serving before shutdown
	if s.healthServer != nil {
		s.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
		s.healthServer.SetServingStatus("kilocenter.api.v1.KiloCenterService", healthpb.HealthCheckResponse_NOT_SERVING)
		s.healthServer.SetServingStatus("kilocenter.api.v1.CoreService", healthpb.HealthCheckResponse_NOT_SERVING)
	}

	// Graceful HTTP shutdown (includes gRPC)
	ctx, cancel := context.WithTimeout(context.Background(), s.httpServer.WriteTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.log.Error("HTTP server shutdown error", "error", err)
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
		log.Debug("gRPC unary call", "method", info.FullMethod)

		resp, err := handler(ctx, req)

		if err != nil {
			log.Error("gRPC unary call failed", "method", info.FullMethod, "error", err)
		}

		return resp, err
	}
}

// communityTenantStream wraps a ServerStream to override context with default tenant ID.
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
		log.Debug("gRPC stream call", "method", info.FullMethod)

		err := handler(srv, ss)

		if err != nil {
			log.Error("gRPC stream call failed", "method", info.FullMethod, "error", err)
		}

		return err
	}
}
