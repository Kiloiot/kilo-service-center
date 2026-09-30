// KC-Gateway is the external ingress for KiloCenter.
// It terminates gRPC-web, authenticates requests, resolves organizations,
// and proxies to KC-Core via trusted internal headers.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/adapter"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/health"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/resilience"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to gateway config file")
	flag.Parse()

	cfg, err := config.LoadGateway(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger.Initialize(cfg.General.LogLevel, cfg.General.LogFormat)
	l := logger.Get()

	upstreamAddr := cfg.Gateway.Upstream()
	identityAddr := cfg.Gateway.Identity()
	healthPort := cfg.Gateway.HealthPort
	if healthPort == 0 {
		healthPort = defaultHealthPort
	}

	shutdownObservability := initObservability(cfg, l)
	defer shutdownObservability()

	l.Info(LogGatewayStarting,
		logger.FieldUpstream, upstreamAddr,
		logger.FieldGrpcPort, cfg.GRPC.Port,
		logger.FieldHealthPort, healthPort)

	resCfg := cfg.Gateway.Resilience
	coreBreaker := resilience.NewUpstreamBreaker(upstreamCore, resCfg)
	identityBreaker := resilience.NewUpstreamBreaker(upstreamIdentity, resCfg)
	breakerGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kc_gateway_circuit_breaker_state",
		Help: breakerStateHelp,
	}, []string{upstreamLabel})
	prometheus.MustRegister(breakerGauge)
	l.Info(LogGatewayBreakersInitialized,
		logger.FieldFailureThreshold, resCfg.CBFailureThreshold,
		logger.FieldTimeout, resCfg.CBTimeout)

	ups, err := dialUpstreams(l, upstreamAddr, identityAddr, resCfg)
	if err != nil {
		os.Exit(1)
	}
	defer closeUpstreams(l, ups)

	internalClient := pb.NewIdentityInternalServiceClient(ups.identityInternal)
	peerSecret := cfg.InternalAuth.PeerSecret
	orgCacheTTL := time.Duration(cfg.General.OrgCacheTTLMinutes) * time.Minute
	orgAdapter := adapter.NewIdentityRPCOrgAdapter(internalClient, peerSecret, l, orgCacheTTL, cfg.General.OrgCacheMaxEntries)
	apiKeyAdapter := adapter.NewIdentityRPCAPIKeyAdapter(internalClient, peerSecret)
	eventAdapter := adapter.NewIdentityRPCEventAdapter(internalClient, peerSecret, cfg.General.TenantID, l)

	authInterceptor, err := newAuthInterceptor(cfg, l, orgAdapter, apiKeyAdapter, eventAdapter)
	if err != nil {
		os.Exit(1)
	}
	orgUnaryInterceptor, orgStreamInterceptor, err := newOrgInterceptors(cfg, l, orgAdapter, eventAdapter)
	if err != nil {
		os.Exit(1)
	}
	rateLimiter := newRateLimiter(cfg, l)
	if rateLimiter != nil {
		defer rateLimiter.Close()
	}

	proxyServer := newProxyServer(cfg, l, ups,
		buildUnaryChain(authInterceptor.UnaryInterceptor(), orgUnaryInterceptor),
		buildStreamChain(rateLimiter, authInterceptor.StreamInterceptor(), orgStreamInterceptor,
			breakerStreamInterceptor(coreBreaker, identityBreaker, resCfg.RPCTimeout)))

	httpServer, listener, err := buildServer(cfg, l, proxyServer)
	if err != nil {
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background()) // context-root: process
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go exportBreakerState(ctx, breakerGauge, coreBreaker, identityBreaker)
	go func() {
		l.Info(LogGatewayHealthServerStarting, logger.FieldPort, healthPort)
		if err := health.StartHealthServer(ctx, l, healthPort, ups.core, ups.identity, coreBreaker, identityBreaker); err != nil && err != http.ErrServerClosed {
			l.Error(LogGatewayHealthServerFailed, logger.FieldError, err)
		}
	}()
	go serve(l, httpServer, listener, cancel)

	hostname := resolveHostname(l)
	emitStartedEvent(ctx, l, eventAdapter, cfg, hostname)

	<-sigCh
	l.Info(LogGatewayShutdownSignal)
	emitStoppedEvent(l, eventAdapter, cfg, hostname)

	shutdownServers(l, httpServer, proxyServer, shutdownTimeout)
	cancel()
	l.Info(LogGatewayStopped)
}
