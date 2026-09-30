package webmux

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
)

// headerListSeparator joins the values of a CORS list header.
const headerListSeparator = ", "

func newWrapper(server *grpc.Server, cfg config.GRPCWebConfig) *grpcweb.WrappedGrpcServer {
	return grpcweb.WrapServer(server,
		grpcweb.WithOriginFunc(func(origin string) bool { return originAllowed(cfg, origin) }),
		grpcweb.WithAllowedRequestHeaders(orDefault(cfg.AllowedHeaders, grpcconst.GRPCWebAllowedHeaders)),
	)
}

func originAllowed(cfg config.GRPCWebConfig, origin string) bool {
	return cfg.AllowAllOrigins || slices.Contains(cfg.AllowedOrigins, origin)
}

func orDefault(configured, fallback []string) []string {
	if len(configured) == 0 {
		return fallback
	}
	return configured
}

// preflight answers a gRPC-web CORS preflight from the configured policy.
type preflight struct {
	cfg config.GRPCWebConfig
}

func (p preflight) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get(grpcconst.HeaderOrigin)
	if !originAllowed(p.cfg, origin) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	header := w.Header()
	// Vary stops a cache reusing one origin's answer; a wildcard origin is invalid with credentials.
	header.Set(grpcconst.HeaderVary, grpcconst.HeaderOrigin)
	header.Set(grpcconst.HeaderAccessControlAllowOrigin, origin)
	if len(p.cfg.AllowedMethods) > 0 {
		header.Set(grpcconst.HeaderAccessControlAllowMethods, strings.Join(p.cfg.AllowedMethods, headerListSeparator))
	}
	header.Set(grpcconst.HeaderAccessControlAllowHeaders,
		strings.Join(orDefault(p.cfg.AllowedHeaders, grpcconst.GRPCWebAllowedHeaders), headerListSeparator))
	header.Set(grpcconst.HeaderAccessControlExposeHeaders,
		strings.Join(orDefault(p.cfg.ExposeHeaders, grpcconst.GRPCWebExposeHeaders), headerListSeparator))
	if p.cfg.AllowCredentials {
		header.Set(grpcconst.HeaderAccessControlAllowCredentials, grpcconst.HeaderValueTrue)
	}
	if p.cfg.MaxAge > 0 {
		header.Set(grpcconst.HeaderAccessControlMaxAge, strconv.Itoa(p.cfg.MaxAge))
	}
	w.WriteHeader(http.StatusNoContent)
}
