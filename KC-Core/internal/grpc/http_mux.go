package grpc

import (
	"net/http"
	"sync"

	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/webmux"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"golang.org/x/net/http2"     //nolint:staticcheck // h2c cleartext HTTP/2 support
	"golang.org/x/net/http2/h2c" //nolint:staticcheck // h2c deprecation acknowledged; migration deferred
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
