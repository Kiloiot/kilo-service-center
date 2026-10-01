// Package webmux serves gRPC-web, native gRPC over HTTP/2 and the gRPC-web
// CORS preflight from one HTTP handler. KC-Core's own gRPC-web listener and
// KC-Gateway's ingress both build their router here.
package webmux

import (
	"net/http"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"google.golang.org/grpc"
)

// http2ProtoMajor identifies HTTP/2 requests carrying native gRPC.
const http2ProtoMajor = 2

// webServer is the part of the gRPC-web wrapper the router consumes.
type webServer interface {
	IsGrpcWebRequest(r *http.Request) bool
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// NewHandler routes a request to gRPC-web, then native gRPC over HTTP/2, then
// the CORS preflight, and answers anything else with 404. gRPC-web and its
// preflight are served only when cfg enables them.
func NewHandler(server *grpc.Server, cfg config.GRPCWebConfig) http.Handler {
	if !cfg.Enabled {
		return nativeGRPC(server, http.NotFoundHandler())
	}
	fallback := preflightOnOptions(preflight{cfg: cfg}, http.NotFoundHandler())
	return grpcWebFirst(newWrapper(server, cfg), nativeGRPC(server, fallback))
}

func grpcWebFirst(web webServer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if web.IsGrpcWebRequest(r) {
			web.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func nativeGRPC(server http.Handler, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == http2ProtoMajor && strings.HasPrefix(r.Header.Get(grpcconst.HeaderContentType), grpcconst.ContentTypeGRPC) {
			server.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func preflightOnOptions(pre http.Handler, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			pre.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
