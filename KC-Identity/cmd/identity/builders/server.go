package builders

import (
	"context"
	"fmt"
	"net"
	"strings"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
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
	log.Info("gRPC services registered (Identity + IdentityInternal + KiloCenterCompat)")

	// Health service
	healthSvc := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSvc)
	healthSvc.SetServingStatus("kilocenter.api.v1.IdentityService", healthpb.HealthCheckResponse_SERVING)
	healthSvc.SetServingStatus("kilocenter.api.v1.IdentityInternalService", healthpb.HealthCheckResponse_SERVING)

	// Reflection for dev
	if cfg.GRPC.EnableReflection {
		reflection.Register(grpcServer)
		log.Info("gRPC reflection enabled")
	}

	// Start listener
	addr := fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port)
	go func() {
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			log.Error("Failed to listen", "address", addr, "error", err)
			cancel()
			return
		}
		log.Info("gRPC server listening", "address", addr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("gRPC server failed", "error", err)
			cancel()
		}
	}()

	return grpcServer
}

// buildUnaryInterceptor creates a method-aware unary interceptor.
// IdentityInternalService methods use peer-secret auth.
// All other methods use InternalTrust (gateway-injected headers).
func buildUnaryInterceptor(cfg *pkgconfig.Config, infra *Infrastructure) grpc.UnaryServerInterceptor {
	// communityMode=true: KC-Identity delegates org enforcement to KC-Gateway;
	// org header is optional at this layer.
	trustInterceptor := interceptors.NewInternalTrustInterceptor(infra.Log, true).
		WithEventWriter(infra.Storage.SystemEvents()).
		WithPlatformTenantID(infra.TenantID)
	peerSecret := cfg.InternalAuth.PeerSecret

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if strings.HasPrefix(info.FullMethod, identityInternalServicePrefix) {
			newCtx, err := validatePeerSecret(ctx, peerSecret)
			if err != nil {
				return nil, err
			}
			return handler(newCtx, req)
		}
		// All other methods go through InternalTrust
		return trustInterceptor.UnaryInterceptor()(ctx, req, info, handler)
	}
}

// buildStreamInterceptor creates a method-aware stream interceptor.
func buildStreamInterceptor(cfg *pkgconfig.Config, infra *Infrastructure) grpc.StreamServerInterceptor {
	// communityMode=true: KC-Identity delegates org enforcement to KC-Gateway;
	// org header is optional at this layer.
	trustInterceptor := interceptors.NewInternalTrustInterceptor(infra.Log, true).
		WithEventWriter(infra.Storage.SystemEvents()).
		WithPlatformTenantID(infra.TenantID)
	peerSecret := cfg.InternalAuth.PeerSecret

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if strings.HasPrefix(info.FullMethod, identityInternalServicePrefix) {
			_, err := validatePeerSecret(ss.Context(), peerSecret)
			if err != nil {
				return err
			}
			return handler(srv, ss)
		}
		return trustInterceptor.StreamInterceptor()(srv, ss, info, handler)
	}
}

// validatePeerSecret checks the x-kc-internal-peer-secret header.
// Empty configured secret = dev mode (bypass auth).
func validatePeerSecret(ctx context.Context, configuredSecret string) (context.Context, error) {
	if configuredSecret == "" {
		return ctx, nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	secrets := md.Get(grpcconst.MetadataKeyInternalPeerSecret)
	if len(secrets) == 0 || secrets[0] != configuredSecret {
		return nil, status.Error(codes.Unauthenticated, "invalid peer secret")
	}

	return ctx, nil
}
