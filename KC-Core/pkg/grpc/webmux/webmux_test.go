package webmux

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// Wire values the gRPC-web transport must emit; asserted here, not exported.
const (
	testGRPCMethodPath                   = "/test.Service/Method"
	testGRPCWebHeaderValue               = "1"
	testContentTypeGRPCWebPrefix         = "application/grpc-web"
	testContentTypeGRPCWeb               = "application/grpc-web+proto"
	testContentTypePlainText             = "text/plain; charset=utf-8"
	testHeaderAccessControlRequestMethod = "Access-Control-Request-Method"
	testHeaderGRPCStatus                 = "grpc-status"
	testHeaderGRPCWeb                    = "x-grpc-web"
	testHeaderUpgrade                    = "Upgrade"
	testHeaderWebSocketProtocol          = "Sec-Websocket-Protocol"
	testUpgradeWebSocket                 = "websocket"
	testGRPCWebSocketsProtocol           = "grpc-websockets"
	testHTTP2ProtoMinor                  = 0
)

func serve(cfg config.GRPCWebConfig, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	NewHandler(grpc.NewServer(), cfg).ServeHTTP(w, req)
	return w
}

func newRequest(method, path string) *http.Request {
	return httptest.NewRequestWithContext(testutil.TestContext(), method, path, nil)
}

func grpcWebRequest() *http.Request {
	req := newRequest(http.MethodPost, testGRPCMethodPath)
	req.Header.Set(grpcconst.HeaderContentType, testContentTypeGRPCWeb)
	req.Header.Set(testHeaderGRPCWeb, testGRPCWebHeaderValue)
	req.Header.Set(grpcconst.HeaderOrigin, testOrigin)
	return req
}

func nativeGRPCRequest() *http.Request {
	req := newRequest(http.MethodPost, testGRPCMethodPath)
	req.Header.Set(grpcconst.HeaderContentType, grpcconst.ContentTypeGRPC)
	req.ProtoMajor = http2ProtoMajor
	req.ProtoMinor = testHTTP2ProtoMinor
	return req
}

func preflightRequest() *http.Request {
	req := newRequest(http.MethodOptions, testRootPath)
	req.Header.Set(grpcconst.HeaderOrigin, testOrigin)
	req.Header.Set(testHeaderAccessControlRequestMethod, http.MethodPost)
	return req
}

func TestNewHandler_Routing(t *testing.T) {
	t.Run("gRPC-web request routes to the wrapper", func(t *testing.T) {
		w := serve(testWebConfig(), grpcWebRequest())

		assert.NotEmpty(t, w.Header().Get(testHeaderGRPCStatus), "the wrapper always sets grpc-status")
		assert.True(t, strings.HasPrefix(w.Header().Get(grpcconst.HeaderContentType), testContentTypeGRPCWebPrefix))
	})

	t.Run("native gRPC request routes to the gRPC server", func(t *testing.T) {
		w := serve(testWebConfig(), nativeGRPCRequest())

		assert.True(t, strings.HasPrefix(w.Header().Get(grpcconst.HeaderContentType), grpcconst.ContentTypeGRPC))
	})

	t.Run("OPTIONS routes to the CORS preflight", func(t *testing.T) {
		w := serve(testWebConfig(), preflightRequest())

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, testOrigin, w.Header().Get(grpcconst.HeaderAccessControlAllowOrigin))
		assert.Equal(t, grpcconst.HeaderOrigin, w.Header().Get(grpcconst.HeaderVary))
	})

	t.Run("GET request returns 404", func(t *testing.T) {
		w := serve(testWebConfig(), newRequest(http.MethodGet, testRootPath))

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, testContentTypePlainText, w.Header().Get(grpcconst.HeaderContentType))
	})

	// Every RPC is unary or server-streaming, so gRPC-web is served over HTTP only.
	t.Run("gRPC-web WebSocket upgrade returns 404", func(t *testing.T) {
		req := newRequest(http.MethodGet, testGRPCMethodPath)
		req.Header.Set(grpcconst.HeaderOrigin, testOrigin)
		req.Header.Set(testHeaderUpgrade, testUpgradeWebSocket)
		req.Header.Set(testHeaderWebSocketProtocol, testGRPCWebSocketsProtocol)

		assert.Equal(t, http.StatusNotFound, serve(testWebConfig(), req).Code)
	})
}

func TestNewHandler_GRPCWebDisabled(t *testing.T) {
	disabled := testWebConfig()
	disabled.Enabled = testGRPCWebOff

	t.Run("native gRPC is still served", func(t *testing.T) {
		w := serve(disabled, nativeGRPCRequest())

		assert.True(t, strings.HasPrefix(w.Header().Get(grpcconst.HeaderContentType), grpcconst.ContentTypeGRPC))
	})

	t.Run("gRPC-web request returns 404", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, serve(disabled, grpcWebRequest()).Code)
	})

	t.Run("OPTIONS returns 404 without CORS headers", func(t *testing.T) {
		w := serve(disabled, preflightRequest())

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Empty(t, w.Header().Get(grpcconst.HeaderAccessControlAllowOrigin))
	})
}
