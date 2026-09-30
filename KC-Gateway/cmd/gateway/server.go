package main

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/webmux"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/rpccatalog"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c" //nolint:staticcheck // TODO: migrate to http.Server.Protocols when ecosystem support is stable
	"google.golang.org/grpc"
)

// streamingMethods names every streaming RPC the gateway proxies, so the HTTP
// server timeouts apply to unary calls only and never to streams.
var streamingMethods = rpccatalog.Streaming(rpccatalog.Proxied()...)

func isStreamingMethod(fullMethod string) bool {
	return streamingMethods[fullMethod]
}

// newMuxHandler multiplexes gRPC-web, native gRPC over HTTP/2 and CORS
// preflight onto one listener.
func newMuxHandler(cfg *config.Config, l logger.Logger, proxyServer *grpc.Server) http.Handler {
	routed := withoutHopByHopHeaders(webmux.NewHandler(proxyServer, cfg.GRPC.Web))
	streamsExempt := grpcconst.ExemptStreamsFromHTTPDeadlines(routed, isStreamingMethod, l)
	return h2c.NewHandler(streamsExempt, &http2.Server{}) //nolint:staticcheck // TODO: migrate to http.Server.Protocols
}

// withoutHopByHopHeaders strips headers the HTTP/2 upstream rejects with 400 once gRPC-web turns them into metadata.
func withoutHopByHopHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, h := range hopByHopHeaders {
			r.Header.Del(h)
		}
		next.ServeHTTP(w, r)
	})
}

// buildServer binds the listener and returns the HTTP server fronting the
// proxy.
func buildServer(cfg *config.Config, l logger.Logger, proxyServer *grpc.Server) (*http.Server, net.Listener, error) {
	listenAddr := net.JoinHostPort(cfg.GRPC.Host, strconv.Itoa(cfg.GRPC.Port))
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		l.Error(LogGatewayListenFailed, logger.FieldAddr, listenAddr, logger.FieldError, err)
		return nil, nil, err
	}
	if cfg.GRPC.Web.Enabled {
		l.Info(LogGatewayGRPCWebEnabled, logger.FieldOrigins, cfg.GRPC.Web.AllowedOrigins)
	}
	httpServer := &http.Server{
		Handler:      newMuxHandler(cfg, l, proxyServer),
		ReadTimeout:  cfg.GRPC.HTTP.ReadTimeout,
		WriteTimeout: cfg.GRPC.HTTP.WriteTimeout,
		IdleTimeout:  cfg.GRPC.HTTP.IdleTimeout,
	}
	return httpServer, listener, nil
}

// serve runs the HTTP server until it stops; an unexpected failure cancels
// the process context.
func serve(l logger.Logger, httpServer *http.Server, listener net.Listener, cancel context.CancelFunc) {
	l.Info(LogGatewayListening, logger.FieldAddr, listener.Addr().String())
	if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		l.Error(LogGatewayServerFailed, logger.FieldError, err)
		cancel()
	}
}
