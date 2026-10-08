package main

import (
	"context"
	"errors"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	gatewayproxy "github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/proxy"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/resilience"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/rpccatalog"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// upstreams holds the internal KC-Identity connection used for auth and org
// resolution and the two connections proxied RPCs are forwarded on.
type upstreams struct {
	identityInternal *grpc.ClientConn
	core             *grpc.ClientConn
	identity         *grpc.ClientConn
}

// dialUpstreams opens the upstream connections; the proxied ones carry the
// retry service config and the OpenTelemetry client handler.
func dialUpstreams(l logger.Logger, coreAddr, identityAddr string, resCfg config.GatewayResilienceConfig) (*upstreams, error) {
	identityInternal, err := grpc.NewClient(identityAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{MinConnectTimeout: resCfg.DialTimeout}))
	if err != nil {
		l.Error(LogGatewayIdentityConnectFailed, logger.FieldAddress, identityAddr, logger.FieldError, err)
		return nil, err
	}
	l.Info(LogGatewayIdentityConnected, logger.FieldAddress, identityAddr)

	retryServiceConfig, err := resilience.BuildRetryServiceConfig(resCfg, rpccatalog.Proxied()...)
	if err != nil {
		err = errors.Join(err, identityInternal.Close())
		l.Error(LogGatewayUpstreamConnCreateFailed, logger.FieldError, err)
		return nil, err
	}
	core, err := dialProxied(coreAddr, resCfg, retryServiceConfig)
	if err != nil {
		err = errors.Join(err, identityInternal.Close())
		l.Error(LogGatewayUpstreamConnCreateFailed, logger.FieldError, err)
		return nil, err
	}
	identity, err := dialProxied(identityAddr, resCfg, retryServiceConfig)
	if err != nil {
		err = errors.Join(err, core.Close(), identityInternal.Close())
		l.Error(LogGatewayIdentityUpstreamConnCreateFailed, logger.FieldError, err)
		return nil, err
	}
	l.Info(LogGatewayUpstreamsEstablished,
		logger.FieldCore, coreAddr,
		logger.FieldIdentity, identityAddr)
	return &upstreams{identityInternal: identityInternal, core: core, identity: identity}, nil
}

func dialProxied(addr string, resCfg config.GatewayResilienceConfig, retryServiceConfig string) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{MinConnectTimeout: resCfg.DialTimeout}),
		grpc.WithDefaultServiceConfig(retryServiceConfig),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(config.GatewayUpstreamMaxResponseBytes)),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
}

// Close releases every upstream connection and reports every close failure.
func (u *upstreams) Close() error {
	return errors.Join(u.identity.Close(), u.core.Close(), u.identityInternal.Close())
}

// closeUpstreams releases the upstream connections at shutdown, when a close
// failure can only be logged.
func closeUpstreams(l logger.Logger, u *upstreams) {
	if err := u.Close(); err != nil {
		l.Warn(LogGatewayUpstreamCloseFailed, logger.FieldError, err)
	}
}

// newDirector routes each proxied call to its upstream with sanitized
// metadata. Community-mode fallback: inject the default tenant only when auth
// did not establish tenant context; never overwrite an authenticated tenant.
func newDirector(cfg *config.Config, l logger.Logger, ups *upstreams) gatewayproxy.StreamDirector {
	return func(ctx context.Context, fullMethodName string) (context.Context, grpc.ClientConnInterface, error) {
		if !cfg.General.OrgEnforcementEnabled && cfg.General.TenantID > 0 {
			if _, err := pkgcontext.GetTenantID(ctx); err != nil {
				l.Warn(LogGatewayDirectorDefaultTenantInjected,
					logger.FieldDefaultTenant, cfg.General.TenantID, logger.FieldMethod, fullMethodName)
				ctx = pkgcontext.WithTenantID(ctx, cfg.General.TenantID)
			}
		}
		conn, err := gatewayproxy.SelectUpstream(fullMethodName, ups.core, ups.identity)
		if err != nil {
			return ctx, nil, err
		}
		outMD := gatewayproxy.SanitizeAndInject(ctx, cfg.InternalAuth.PeerSecret)
		outCtx := metadata.NewOutgoingContext(ctx, outMD)
		return outCtx, conn, nil
	}
}

// newProxyServer builds the server that runs every call through the
// interceptor chains and proxies it to the upstream its method routes to.
func newProxyServer(cfg *config.Config, l logger.Logger, ups *upstreams, unary []grpc.UnaryServerInterceptor, stream []grpc.StreamServerInterceptor) *grpc.Server {
	return grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(unary...),
		grpc.ChainStreamInterceptor(stream...),
		grpc.UnknownServiceHandler(gatewayproxy.TransparentHandler(newDirector(cfg, l, ups))),
	)
}
