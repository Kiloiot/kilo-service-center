package builders

import (
	"context"
	"fmt"
	"net"
	"strings"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const identityInternalServicePrefix = "/kilocenter.api.v1.IdentityInternalService/"

// RegisterAndServe registers gRPC services and starts the gRPC server.
// Returns the grpc.Server for graceful shutdown.
func RegisterAndServe(
	cfg *pkgconfig.Config,
	infra *Infrastructure,
	identity *IdentityResult,
	cancel context.CancelFunc,
) *grpc.Server {
	log := logger.Get()

	// Build method-aware interceptor chain
	unaryInterceptor := buildUnaryInterceptor(cfg, infra)
	streamInterceptor := buildStreamInterceptor(cfg, infra)

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(unaryInterceptor),
		grpc.StreamInterceptor(streamInterceptor),
	)

	// Register services
	pb.RegisterIdentityServiceServer(grpcServer, identity.IdentityService)
	pb.RegisterIdentityInternalServiceServer(grpcServer, identity.IdentityInternalService)
	pb.RegisterKiloCenterServiceServer(grpcServer, identity.CompatService)
	log.Info(LogGRPCServicesRegistered)

	// Health service
	healthSvc := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSvc)
	healthSvc.SetServingStatus(pb.IdentityService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)
	healthSvc.SetServingStatus(pb.IdentityInternalService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)

	// Reflection for dev
	if cfg.GRPC.EnableReflection {
		reflection.Register(grpcServer)
		log.Info(LogGRPCReflectionEnabled)
	}

	// Start listener
	addr := fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port)
	go func() {
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			log.Error(LogGRPCListenFailed, logger.FieldAddress, addr, logger.FieldError, err)
			cancel()
			return
		}
		log.Info(LogGRPCServerListening, logger.FieldAddress, addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Error(LogGRPCServerFailed, logger.FieldError, err)
			cancel()
		}
	}()

	return grpcServer
}

// buildUnaryInterceptor creates the method-aware unary interceptor.
func buildUnaryInterceptor(cfg *pkgconfig.Config, infra *Infrastructure) grpc.UnaryServerInterceptor {
	return methodAwareUnary(interceptors.NewPeerAuthenticator(cfg.InternalAuth.PeerSecret), newTrustInterceptor(cfg, infra))
}

// buildStreamInterceptor creates the method-aware stream interceptor.
func buildStreamInterceptor(cfg *pkgconfig.Config, infra *Infrastructure) grpc.StreamServerInterceptor {
	return methodAwareStream(interceptors.NewPeerAuthenticator(cfg.InternalAuth.PeerSecret), newTrustInterceptor(cfg, infra))
}

func newTrustInterceptor(cfg *pkgconfig.Config, infra *Infrastructure) *interceptors.InternalTrustInterceptor {
	return trustGatewayPeers(infra.Log, cfg.InternalAuth.PeerSecret, infra.OrgResolverSvc, infra.Repos.SystemEvents, infra.TenantID)
}

// trustGatewayPeers runs a call without an org header under the tenant's default org; the gateway enforces orgs.
func trustGatewayPeers(log logger.Logger, peerSecret string, defaultOrgs interceptors.DefaultOrgResolver, events audit.EventWriter, platformTenantID int64) *interceptors.InternalTrustInterceptor {
	return interceptors.NewInternalTrustInterceptor(log, true).
		WithPeerSecret(peerSecret).
		WithDefaultOrgResolver(defaultOrgs).
		WithEventWriter(events).
		WithPlatformTenantID(platformTenantID)
}

// methodAwareUnary admits IdentityInternalService calls on the peer secret alone.
func methodAwareUnary(peers interceptors.PeerAuthenticator, trust *interceptors.InternalTrustInterceptor) grpc.UnaryServerInterceptor {
	trusted := trust.UnaryInterceptor()
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !strings.HasPrefix(info.FullMethod, identityInternalServicePrefix) {
			return trusted(ctx, req, info, handler)
		}
		if err := peers.Authenticate(ctx); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// methodAwareStream is methodAwareUnary for streaming calls.
func methodAwareStream(peers interceptors.PeerAuthenticator, trust *interceptors.InternalTrustInterceptor) grpc.StreamServerInterceptor {
	trusted := trust.StreamInterceptor()
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !strings.HasPrefix(info.FullMethod, identityInternalServicePrefix) {
			return trusted(srv, ss, info, handler)
		}
		if err := peers.Authenticate(ss.Context()); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}
