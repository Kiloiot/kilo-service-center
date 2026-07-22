// KC-Gateway is the external ingress for KiloCenter.
// It terminates gRPC-web, authenticates requests, resolves organizations,
// and proxies to KC-Core via trusted internal headers.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/adapter"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/health"
	gatewayproxy "github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/proxy"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/resilience"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/Kiloiot/kilo-service-center/pkg/observability"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	grpcproxy "github.com/mwitkow/grpc-proxy/proxy"
	"github.com/prometheus/client_golang/prometheus"
	gobreaker "github.com/sony/gobreaker/v2"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c" //nolint:staticcheck // TODO: migrate to http.Server.Protocols when ecosystem support is stable
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// gatewayServiceName identifies KC-Gateway as the source of health and system events.
const gatewayServiceName = "kc-gateway"

// unaryMethods enumerates all unary RPCs from service descriptors.
// Used to apply per-RPC timeout only to unary calls (not streams).
var unaryMethods = func() map[string]bool {
	m := make(map[string]bool)
	for _, desc := range []grpc.ServiceDesc{
		pb.CoreService_ServiceDesc,
		pb.IdentityService_ServiceDesc,
		pb.KiloCenterService_ServiceDesc,
	} {
		for _, method := range desc.Methods {
			m["/"+desc.ServiceName+"/"+method.MethodName] = true
		}
	}
	return m
}()

type contextServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *contextServerStream) Context() context.Context {
	return s.ctx
}

func breakerStreamInterceptor(
	coreBreaker, identityBreaker *resilience.UpstreamBreaker,
	rpcTimeout time.Duration,
) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		breaker := coreBreaker
		if strings.HasPrefix(info.FullMethod, "/kilocenter.api.v1.IdentityService/") ||
			gatewayproxy.IsCompatIdentityMethod(info.FullMethod) {
			breaker = identityBreaker
		}

		// Unary methods run inside the breaker with the per-RPC timeout:
		// they are short-lived, so occupying an execution slot is correct
		// and client cancellations already surface as codes.Canceled.
		if unaryMethods[info.FullMethod] {
			err := breaker.Execute(func() error {
				if rpcTimeout > 0 {
					ctx, cancel := context.WithTimeout(ss.Context(), rpcTimeout)
					defer cancel()
					ss = &contextServerStream{ServerStream: ss, ctx: ctx}
				}
				return handler(srv, ss)
			})

			// Map gobreaker rejection errors to gRPC Unavailable for client failover
			if errors.Is(err, gobreaker.ErrOpenState) {
				return status.Error(codes.Unavailable, "upstream circuit breaker is open")
			}
			if errors.Is(err, gobreaker.ErrTooManyRequests) {
				return status.Error(codes.Unavailable, "upstream circuit breaker is recovering")
			}
			return err
		}

		// Streaming RPCs (including all proxied calls via
		// UnknownServiceHandler) run outside the breaker's execution slots: a
		// long-lived stream must not hold a half-open probe slot for its
		// whole lifetime. Streams are therefore admitted only while the
		// breaker is closed - a half-open breaker probes recovery through the
		// slot-accounted unary path, and unbounded streams admitted beside
		// those probes would bypass MaxRequests entirely. The stream's
		// terminal error still feeds the counters - except when the client
		// tore the stream down (context canceled), which the transparent
		// proxy surfaces as codes.Internal "failed proxying s2c" and must not
		// count as an upstream failure.
		switch breaker.State() {
		case resilience.BreakerOpen:
			return status.Error(codes.Unavailable, "upstream circuit breaker is open")
		case resilience.BreakerHalfOpen:
			return status.Error(codes.Unavailable, "upstream circuit breaker is recovering")
		}

		err := handler(srv, ss)
		if err != nil && ss.Context().Err() != nil {
			// Client-initiated teardown: benign, not an upstream failure.
			return err
		}
		breaker.RecordResult(err)
		return err
	}
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to gateway config file")
	flag.Parse()

	// Load configuration (reuses KC-Core config loader for consistency)
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger.Initialize(cfg.General.LogLevel, cfg.General.LogFormat)
	l := logger.Get()

	// Gateway-specific config from typed struct (env overrides via viper's AutomaticEnv)
	upstreamAddr := cfg.Gateway.UpstreamAddress
	if upstreamAddr == "" {
		upstreamAddr = "localhost:50051"
	}

	healthPort := cfg.Gateway.HealthPort
	if healthPort == 0 {
		healthPort = 8087
	}

	// Initialize observability (tracing + metrics)
	tracingShutdown, err := observability.InitTracing(context.Background(), observability.TracingConfig{
		Enabled:    cfg.Monitoring.TracingEnabled,
		Endpoint:   cfg.Monitoring.TracingEndpoint,
		SampleRate: cfg.Monitoring.TracingSampleRate,
	}, gatewayServiceName)
	if err != nil {
		l.Error("Failed to init tracing", "error", err)
	} else {
		defer func() { _ = tracingShutdown(context.Background()) }()
	}

	metricsShutdown, err := observability.InitMetrics(context.Background(), observability.MetricsConfig{
		Enabled: cfg.Monitoring.MetricsEnabled,
		Port:    cfg.Monitoring.MetricsPort,
		Path:    cfg.Monitoring.MetricsPath,
	}, gatewayServiceName)
	if err != nil {
		l.Error("Failed to init metrics", "error", err)
	} else {
		defer func() { _ = metricsShutdown(context.Background()) }()
	}

	l.Info("KC-Gateway starting",
		"upstream", upstreamAddr,
		"grpcPort", cfg.GRPC.Port,
		"healthPort", healthPort)

	// Resilience configuration
	resCfg := cfg.Gateway.Resilience

	// Build retry service config for idempotent RPCs
	retryServiceConfig := resilience.BuildRetryServiceConfig(resCfg)

	// Create circuit breakers for each upstream
	coreBreaker := resilience.NewUpstreamBreaker("core", resCfg)
	identityBreaker := resilience.NewUpstreamBreaker("identity", resCfg)
	breakerGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kc_gateway_circuit_breaker_state",
		Help: "Circuit breaker state (0=closed, 1=half-open, 2=open)",
	}, []string{"upstream"})
	prometheus.MustRegister(breakerGauge)

	l.Info("Circuit breakers initialized",
		"failureThreshold", resCfg.CBFailureThreshold,
		"timeout", resCfg.CBTimeout)

	// Connect to KC-Identity for auth/org resolution via IdentityInternalService RPCs
	identityAddr := cfg.Gateway.IdentityAddress
	if identityAddr == "" {
		identityAddr = "localhost:50052"
	}

	identityConn, err := grpc.NewClient(identityAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{MinConnectTimeout: resCfg.DialTimeout}))
	if err != nil {
		l.Error("Failed to connect to KC-Identity", "address", identityAddr, "error", err)
		os.Exit(1)
	}
	defer func() { _ = identityConn.Close() }()
	l.Info("KC-Identity connection established", "address", identityAddr)

	internalClient := pb.NewIdentityInternalServiceClient(identityConn)
	peerSecret := cfg.InternalAuth.PeerSecret

	// Create unified org adapter via KC-Identity RPCs
	orgCacheTTL := time.Duration(cfg.General.OrgCacheTTLMinutes) * time.Minute
	orgCacheMax := cfg.General.OrgCacheMaxEntries
	orgAdapter := adapter.NewIdentityRPCOrgAdapter(internalClient, peerSecret, l, orgCacheTTL, orgCacheMax)

	// Create API key adapter via KC-Identity RPCs
	apiKeyAdapter := adapter.NewIdentityRPCAPIKeyAdapter(internalClient, peerSecret)

	// Create event adapter for lifecycle and security events via KC-Identity
	eventAdapter := adapter.NewIdentityRPCEventAdapter(internalClient, peerSecret, cfg.General.TenantID)

	// Create auth interceptor
	authInterceptor, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{
		Enabled:          cfg.Auth.Enabled,
		HMACSecret:       cfg.Auth.HMACSecret,
		Issuer:           cfg.Auth.Issuer,
		Audience:         cfg.Auth.Audience,
		TenantClaim:      cfg.Auth.TenantClaim,
		UserClaim:        cfg.Auth.OAuth2.UserIDClaim,
		JWKSEndpoint:     cfg.Auth.JWKSEndpoint,
		Algorithm:        cfg.Auth.Algorithm,
		EventWriter:      eventAdapter,
		PlatformTenantID: cfg.General.TenantID,
	})
	if err != nil {
		l.Error("Failed to create auth interceptor", "error", err)
		os.Exit(1)
	}
	authInterceptor.WithAPIKeyAuthenticator(apiKeyAdapter)
	authInterceptor.WithOrganizationResolver(orgAdapter)
	authInterceptor.WithTenantResolver(orgAdapter)
	l.Info("Auth interceptor configured", "enabled", cfg.Auth.Enabled)

	// Create org resolver interceptor (fail-closed for non-exempt methods).
	// Skipped in community mode (org_enforcement_enabled: false).
	var orgUnaryInterceptor grpc.UnaryServerInterceptor
	var orgStreamInterceptor grpc.StreamServerInterceptor

	if cfg.General.OrgEnforcementEnabled {
		skipMethods := make([]string, 0, len(grpcconst.OrgExemptMethods))
		for m := range grpcconst.OrgExemptMethods {
			skipMethods = append(skipMethods, m)
		}
		orgInterceptor, err := interceptors.NewOrgResolverInterceptor(interceptors.OrgResolverInterceptorConfig{
			Resolver:         orgAdapter,
			Logger:           l,
			SkipMethods:      skipMethods,
			EventWriter:      eventAdapter,
			PlatformTenantID: cfg.General.TenantID,
			AdminChecker:     orgAdapter,
		})
		if err != nil {
			l.Error("Failed to create org resolver interceptor", "error", err)
			os.Exit(1)
		}
		orgUnaryInterceptor = orgInterceptor.UnaryInterceptor()
		orgStreamInterceptor = orgInterceptor.StreamInterceptor()
		l.Info("Org resolver interceptor configured (enforcement enabled)")
	} else {
		l.Info("Org resolver interceptor skipped (community mode)")
	}

	// Establish upstream connections: KC-Core for core RPCs, KC-Identity for identity RPCs
	upstreamConn, err := grpc.NewClient(
		upstreamAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{MinConnectTimeout: resCfg.DialTimeout}),
		grpc.WithDefaultServiceConfig(retryServiceConfig),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		l.Error("Failed to create upstream connection", "error", err)
		os.Exit(1)
	}
	defer func() { _ = upstreamConn.Close() }()

	// Separate upstream connection for proxying IdentityService RPCs to KC-Identity
	identityUpstreamConn, err := grpc.NewClient(
		identityAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{MinConnectTimeout: resCfg.DialTimeout}),
		grpc.WithDefaultServiceConfig(retryServiceConfig),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		l.Error("Failed to create identity upstream connection", "error", err)
		os.Exit(1)
	}
	defer func() { _ = identityUpstreamConn.Close() }()

	l.Info("Upstream connections established",
		"core", upstreamAddr,
		"identity", identityAddr)

	// Create proxy director with method-based routing.
	// Community-mode fallback: inject default tenant only when auth did not establish
	// tenant context. Never overwrites an existing authenticated tenant.
	director := func(ctx context.Context, fullMethodName string) (context.Context, grpc.ClientConnInterface, error) {
		if !cfg.General.OrgEnforcementEnabled && cfg.General.TenantID > 0 {
			if _, err := pkgcontext.GetTenantID(ctx); err != nil {
				l.Warn("director: injecting default community tenant (auth did not set tenant)",
					"defaultTenant", cfg.General.TenantID, "method", fullMethodName)
				ctx = pkgcontext.WithTenantID(ctx, cfg.General.TenantID)
			}
		}
		conn, err := gatewayproxy.SelectUpstream(fullMethodName, upstreamConn, identityUpstreamConn)
		if err != nil {
			return ctx, nil, err
		}
		outMD := gatewayproxy.SanitizeAndInject(ctx)
		outCtx := metadata.NewOutgoingContext(ctx, outMD)
		return outCtx, conn, nil
	}

	// Per-IP rate limiter for sensitive endpoints (registration)
	var rateLimiter *resilience.RegistrationRateLimiter
	if cfg.Gateway.RateLimit.Enabled {
		rateLimiter = resilience.NewRegistrationRateLimiter(cfg.Gateway.RateLimit)
		l.Info("Registration rate limiter enabled",
			"requestsPerMin", cfg.Gateway.RateLimit.RequestsPerMin,
			"burst", cfg.Gateway.RateLimit.Burst)
	}

	// Build stream interceptor chain: rate limiter → auth → [org resolver] → circuit breaker
	streamInterceptors := []grpc.StreamServerInterceptor{}
	if rateLimiter != nil {
		streamInterceptors = append(streamInterceptors, rateLimiter.StreamInterceptor())
	}
	streamInterceptors = append(streamInterceptors, authInterceptor.StreamInterceptor())
	if orgStreamInterceptor != nil {
		streamInterceptors = append(streamInterceptors, orgStreamInterceptor)
	}
	streamInterceptors = append(streamInterceptors, breakerStreamInterceptor(coreBreaker, identityBreaker, resCfg.RPCTimeout))

	// Build unary interceptor chain: auth → [org resolver]
	unaryChain := []grpc.UnaryServerInterceptor{authInterceptor.UnaryInterceptor()}
	if orgUnaryInterceptor != nil {
		unaryChain = append(unaryChain, orgUnaryInterceptor)
	}

	// Create proxy gRPC server with interceptor chain: otel → rate limit → auth → [org resolver] → proxy
	proxyServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(unaryChain...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
		grpc.UnknownServiceHandler(grpcproxy.TransparentHandler(director)),
	)

	// Create gRPC-web wrapper
	var grpcWebWrapped *grpcweb.WrappedGrpcServer
	if cfg.GRPC.Web.Enabled {
		originFunc := func(origin string) bool {
			if cfg.GRPC.Web.AllowAllOrigins {
				return true
			}
			for _, allowed := range cfg.GRPC.Web.AllowedOrigins {
				if origin == allowed {
					return true
				}
			}
			return false
		}

		allowedHeaders := cfg.GRPC.Web.AllowedHeaders
		if len(allowedHeaders) == 0 {
			allowedHeaders = grpcconst.GRPCWebAllowedHeaders
		}

		opts := []grpcweb.Option{
			grpcweb.WithOriginFunc(originFunc),
			grpcweb.WithAllowedRequestHeaders(allowedHeaders),
		}

		if cfg.GRPC.Web.EnableWebsockets {
			opts = append(opts,
				grpcweb.WithWebsockets(true),
				grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool {
					return originFunc(req.Header.Get("Origin"))
				}),
			)
		}

		grpcWebWrapped = grpcweb.WrapServer(proxyServer, opts...)
		l.Info("gRPC-web enabled",
			"websockets", cfg.GRPC.Web.EnableWebsockets,
			"origins", cfg.GRPC.Web.AllowedOrigins)
	}

	// Create HTTP handler for multiplexing gRPC-web + native gRPC
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip hop-by-hop headers that are invalid in HTTP/2 / gRPC.
		// The gRPC-web wrapper converts HTTP/1.1 headers to gRPC metadata
		// which gets proxied to upstream via HTTP/2. If hop-by-hop headers
		// like "Connection" leak through, the upstream rejects with 400.
		for _, h := range []string{"Connection", "Keep-Alive", "Proxy-Connection", "Transfer-Encoding", "Upgrade"} {
			r.Header.Del(h)
		}

		if grpcWebWrapped != nil {
			if grpcWebWrapped.IsGrpcWebRequest(r) || grpcWebWrapped.IsGrpcWebSocketRequest(r) {
				grpcWebWrapped.ServeHTTP(w, r)
				return
			}
		}

		contentType := r.Header.Get("Content-Type")
		if r.ProtoMajor == 2 && strings.HasPrefix(contentType, "application/grpc") {
			proxyServer.ServeHTTP(w, r)
			return
		}

		if r.Method == http.MethodOptions && grpcWebWrapped != nil {
			handleCORSPreflight(w, r, cfg)
			return
		}

		http.NotFound(w, r)
	})

	h2s := &http2.Server{}
	finalHandler := h2c.NewHandler(handler, h2s) //nolint:staticcheck // TODO: migrate to http.Server.Protocols

	listenAddr := net.JoinHostPort(cfg.GRPC.Host, strconv.Itoa(cfg.GRPC.Port))
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		l.Error("Failed to listen", "addr", listenAddr, "error", err)
		os.Exit(1)
	}

	httpServer := &http.Server{
		Handler:      finalHandler,
		ReadTimeout:  cfg.GRPC.HTTP.ReadTimeout,
		WriteTimeout: cfg.GRPC.HTTP.WriteTimeout,
		IdleTimeout:  cfg.GRPC.HTTP.IdleTimeout,
	}

	// Close rate limiter cleanup goroutine on shutdown
	if rateLimiter != nil {
		defer rateLimiter.Close()
	}

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Periodically export circuit breaker state as Prometheus gauge
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				breakerGauge.WithLabelValues("core").Set(float64(coreBreaker.State()))
				breakerGauge.WithLabelValues("identity").Set(float64(identityBreaker.State()))
			}
		}
	}()

	// Start health server with circuit breaker state
	go func() {
		l.Info("Starting health server", "port", healthPort)
		if err := health.StartHealthServer(ctx, healthPort, upstreamConn, identityUpstreamConn, coreBreaker, identityBreaker); err != nil && err != http.ErrServerClosed {
			l.Error("Health server failed", "error", err)
		}
	}()

	// Start gateway
	go func() {
		l.Info("KC-Gateway listening", "addr", listenAddr)
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			l.Error("Gateway server failed", "error", err)
			cancel()
		}
	}()

	// Emit lifecycle event — KC-Gateway started
	hostname, _ := os.Hostname()
	{
		details, _ := json.Marshal(map[string]interface{}{
			"service": gatewayServiceName,
			"host":    hostname,
			"ports":   map[string]int{"grpc-web": cfg.GRPC.Port, "health": cfg.Gateway.HealthPort},
		})
		_ = eventAdapter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(cfg.General.TenantID, 10),
			EventType:   models.EventTypeServiceStarted,
			Category:    models.EventCategorySystem,
			Severity:    models.EventSeverityInfo,
			Title:       fmt.Sprintf(models.EventTitleServiceStartedFmt, "KC-Gateway", hostname),
			Description: fmt.Sprintf(models.EventDescriptionServiceStartedFmt, "KC-Gateway", "1.0.0", fmt.Sprintf("gRPC-web :%d, Health :%d", cfg.GRPC.Port, cfg.Gateway.HealthPort)),
			SourceType:  models.SourceTypeSystem,
			SourceName:  gatewayServiceName,
			Details:     details,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	// Wait for shutdown signal
	<-sigCh
	l.Info("Shutdown signal received, stopping gateway...")

	// Emit lifecycle event — KC-Gateway stopped
	{
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = eventAdapter.CreateEvent(stopCtx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(cfg.General.TenantID, 10),
			EventType:   models.EventTypeServiceStopped,
			Category:    models.EventCategorySystem,
			Severity:    models.EventSeverityInfo,
			Title:       fmt.Sprintf(models.EventTitleServiceStoppedFmt, "KC-Gateway", hostname),
			Description: fmt.Sprintf(models.EventDescriptionServiceStoppedFmt, "KC-Gateway", "1.0.0"),
			SourceType:  models.SourceTypeSystem,
			SourceName:  gatewayServiceName,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		stopCancel()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		l.Error("Gateway shutdown error", "error", err)
	}

	proxyServer.GracefulStop()
	cancel()
	l.Info("KC-Gateway stopped")
}

func handleCORSPreflight(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	origin := r.Header.Get("Origin")

	allowed := cfg.GRPC.Web.AllowAllOrigins
	if !allowed {
		for _, o := range cfg.GRPC.Web.AllowedOrigins {
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

	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Origin", origin)

	if len(cfg.GRPC.Web.AllowedMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.GRPC.Web.AllowedMethods, ", "))
	}

	allowedHeaders := cfg.GRPC.Web.AllowedHeaders
	if len(allowedHeaders) == 0 {
		allowedHeaders = grpcconst.GRPCWebAllowedHeaders
	}
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))

	exposeHeaders := cfg.GRPC.Web.ExposeHeaders
	if len(exposeHeaders) == 0 {
		exposeHeaders = grpcconst.GRPCWebExposeHeaders
	}
	w.Header().Set("Access-Control-Expose-Headers", strings.Join(exposeHeaders, ", "))

	if cfg.GRPC.Web.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if cfg.GRPC.Web.MaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.GRPC.Web.MaxAge))
	}

	w.WriteHeader(http.StatusNoContent)
}
