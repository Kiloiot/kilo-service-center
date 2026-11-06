package grpc

import (
	"net/http"
	"sync"

	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/webmux"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"google.golang.org/grpc"
)

// registeredStreams reports whether a full method is a streaming RPC of the
// server. Services register before Serve, so the set is built on first use.
func registeredStreams(grpcServer *grpc.Server) func(string) bool {
	streams := sync.OnceValue(func() map[string]bool {
		return grpcerrors.StreamingMethods(grpcServer.GetServiceInfo())
	})
	return func(fullMethod string) bool { return streams()[fullMethod] }
}

// createHTTPServer creates the multiplexing HTTP server
func createHTTPServer(grpcServer *grpc.Server, cfg Config, log logger.Logger) *http.Server {
	handler := grpcerrors.ExemptStreamsFromHTTPDeadlines(webmux.NewHandler(grpcServer, cfg.GRPCWeb), registeredStreams(grpcServer), log)

	server := &http.Server{
		Handler:      handler,
		ReadTimeout:  cfg.HTTPConfig.ReadTimeout,
		WriteTimeout: cfg.HTTPConfig.WriteTimeout,
		IdleTimeout:  cfg.HTTPConfig.IdleTimeout,
	}
	if !cfg.EnableTLS {
		server.Protocols = grpcerrors.CleartextProtocols()
	}
	return server
}
