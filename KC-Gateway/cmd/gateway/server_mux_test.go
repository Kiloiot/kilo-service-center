package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	testHeaderAccessControlRequestMethod = "Access-Control-Request-Method"
	testHopByHopValue                    = "keep-alive"
)

// The shipped gateway answers a browser's CORS preflight from its gRPC-web
// configuration.
func TestMuxHandler_AnswersPreflightFromShippedConfig(t *testing.T) {
	cfg, err := config.LoadGateway(testShippedGatewayConfig)
	require.NoError(t, err)
	require.NotEmpty(t, cfg.GRPC.Web.AllowedOrigins)
	origin := cfg.GRPC.Web.AllowedOrigins[0]

	req := httptest.NewRequestWithContext(testutil.TestContext(), http.MethodOptions, "/", nil)
	req.Header.Set(grpcconst.HeaderOrigin, origin)
	req.Header.Set(testHeaderAccessControlRequestMethod, http.MethodPost)
	w := httptest.NewRecorder()
	newMuxHandler(cfg, logger.NewNop(), grpc.NewServer()).ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, origin, w.Header().Get(grpcconst.HeaderAccessControlAllowOrigin))
}

func TestWithoutHopByHopHeaders(t *testing.T) {
	var forwarded http.Header
	handler := withoutHopByHopHeaders(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		forwarded = r.Header.Clone()
	}))
	req := httptest.NewRequestWithContext(testutil.TestContext(), http.MethodPost, "/", nil)
	for _, h := range hopByHopHeaders {
		req.Header.Set(h, testHopByHopValue)
	}
	req.Header.Set(grpcconst.HeaderOrigin, testHopByHopValue)

	handler.ServeHTTP(httptest.NewRecorder(), req)

	for _, h := range hopByHopHeaders {
		require.NotContains(t, forwarded, http.CanonicalHeaderKey(h))
	}
	require.Equal(t, testHopByHopValue, forwarded.Get(grpcconst.HeaderOrigin))
}
