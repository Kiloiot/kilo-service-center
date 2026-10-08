package main

import (
	"context"
	"errors"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/adapter"
	gatewayproxy "github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/proxy"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/resilience"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/rpccatalog"
	gobreaker "github.com/sony/gobreaker/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// unaryMethods names every unary RPC the gateway proxies, so the per-RPC
// timeout applies to unary calls only and never to streams.
var unaryMethods = rpccatalog.Unary(rpccatalog.Proxied()...)

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
		// The breaker follows the call's route; a method the gateway refuses to
		// route is answered here and never counts for or against an upstream.
		breaker, err := gatewayproxy.SelectUpstream(info.FullMethod, coreBreaker, identityBreaker)
		if err != nil {
			return err
		}

		// Unary methods run inside the breaker with the per-RPC timeout:
		// they are short-lived, so occupying an execution slot is correct
		// and client cancellations already surface as codes.Canceled.
		if unaryMethods[info.FullMethod] {
			err = breaker.Execute(func() error {
				if rpcTimeout > 0 {
					ctx, cancel := context.WithTimeout(ss.Context(), rpcTimeout)
					defer cancel()
					ss = &contextServerStream{ServerStream: ss, ctx: ctx}
				}
				return handler(srv, ss)
			})

			// Map gobreaker rejection errors to gRPC Unavailable for client failover
			if errors.Is(err, gobreaker.ErrOpenState) {
				return status.Error(codes.Unavailable, statusBreakerOpen)
			}
			if errors.Is(err, gobreaker.ErrTooManyRequests) {
				return status.Error(codes.Unavailable, statusBreakerRecovering)
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
			return status.Error(codes.Unavailable, statusBreakerOpen)
		case resilience.BreakerHalfOpen:
			return status.Error(codes.Unavailable, statusBreakerRecovering)
		}

		err = handler(srv, ss)
		if err != nil && ss.Context().Err() != nil {
			// Client-initiated teardown: benign, not an upstream failure.
			return err
		}
		breaker.RecordResult(err)
		return err
	}
}

// buildUnaryChain orders the unary interceptors: auth, then the optional org
// resolver.
func buildUnaryChain(auth, org grpc.UnaryServerInterceptor) []grpc.UnaryServerInterceptor {
	chain := []grpc.UnaryServerInterceptor{auth}
	if org != nil {
		chain = append(chain, org)
	}
	return chain
}

// buildStreamChain orders the stream interceptors: optional rate limiter,
// auth, optional org resolver, then the circuit breaker.
func buildStreamChain(rateLimiter *resilience.RegistrationRateLimiter, auth, org, breaker grpc.StreamServerInterceptor) []grpc.StreamServerInterceptor {
	chain := []grpc.StreamServerInterceptor{}
	if rateLimiter != nil {
		chain = append(chain, rateLimiter.StreamInterceptor())
	}
	chain = append(chain, auth)
	if org != nil {
		chain = append(chain, org)
	}
	return append(chain, breaker)
}

// newAuthInterceptor builds the JWT/API-key authenticator with the identity
// adapters attached.
func newAuthInterceptor(cfg *config.Config, l logger.Logger, orgAdapter *adapter.IdentityRPCOrgAdapter, apiKeyAdapter *adapter.IdentityRPCAPIKeyAdapter, eventAdapter *adapter.IdentityRPCEventAdapter) (*interceptors.AuthInterceptor, error) {
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
		Logger:           l,
	})
	if err != nil {
		l.Error(LogGatewayAuthInterceptorCreateFailed, logger.FieldError, err)
		return nil, err
	}
	authInterceptor.WithAPIKeyAuthenticator(apiKeyAdapter)
	authInterceptor.WithOrganizationResolver(orgAdapter)
	authInterceptor.WithTenantResolver(orgAdapter)
	l.Info(LogGatewayAuthInterceptorConfigured, logger.FieldEnabled, cfg.Auth.Enabled)
	return authInterceptor, nil
}

// newOrgInterceptors builds the fail-closed org resolver pair, or returns nil
// interceptors in community mode where org enforcement is off.
func newOrgInterceptors(cfg *config.Config, l logger.Logger, orgAdapter *adapter.IdentityRPCOrgAdapter, eventAdapter *adapter.IdentityRPCEventAdapter) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor, error) {
	if !cfg.General.OrgEnforcementEnabled {
		l.Info(LogGatewayOrgInterceptorSkipped)
		return nil, nil, nil
	}
	orgInterceptor, err := interceptors.NewOrgResolverInterceptor(interceptors.OrgResolverInterceptorConfig{
		Resolver:         orgAdapter,
		Logger:           l,
		SkipMethods:      grpcconst.OrgExemptMethodList(),
		EventWriter:      eventAdapter,
		PlatformTenantID: cfg.General.TenantID,
		AdminChecker:     orgAdapter,
	})
	if err != nil {
		l.Error(LogGatewayOrgInterceptorCreateFailed, logger.FieldError, err)
		return nil, nil, err
	}
	l.Info(LogGatewayOrgInterceptorConfigured)
	return orgInterceptor.UnaryInterceptor(), orgInterceptor.StreamInterceptor(), nil
}

// newRateLimiter returns the per-IP registration limiter, or nil when disabled.
func newRateLimiter(cfg *config.Config, l logger.Logger) *resilience.RegistrationRateLimiter {
	if !cfg.Gateway.RateLimit.Enabled {
		return nil
	}
	l.Info(LogGatewayRateLimiterEnabled,
		logger.FieldRequestsPerMin, cfg.Gateway.RateLimit.RequestsPerMin,
		logger.FieldBurst, cfg.Gateway.RateLimit.Burst)
	return resilience.NewRegistrationRateLimiter(cfg.Gateway.RateLimit)
}
