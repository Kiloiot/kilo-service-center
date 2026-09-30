package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/rpccatalog"
)

// WebSocket handshake a gRPC-web client with the WebSocket transport sends.
const (
	testShippedGatewayConfig   = "../../config.yaml"
	testHeaderUpgrade          = "Upgrade"
	testHeaderConnection       = "Connection"
	testHeaderWebSocketKey     = "Sec-WebSocket-Key"
	testHeaderWebSocketVersion = "Sec-WebSocket-Version"
	testHeaderWebSocketProto   = "Sec-WebSocket-Protocol"
	testUpgradeWebSocket       = "websocket"
	testConnectionUpgrade      = "Upgrade"
	testWebSocketKey           = "dGhlIHNhbXBsZSBub25jZQ=="
	testWebSocketVersion       = "13"
	testGRPCWebSocketsProtocol = "grpc-websockets"
)

// Every proxied RPC is unary or server-streaming, so gRPC-web over HTTP
// carries them all; the shipped gateway answers a gRPC-web WebSocket upgrade
// with no handler.
func TestMuxHandler_ServesNoGRPCWebOverWebSocket(t *testing.T) {
	cfg, err := config.LoadGateway(testShippedGatewayConfig)
	require.NoError(t, err)
	require.True(t, cfg.GRPC.Web.Enabled)
	require.NotEmpty(t, cfg.GRPC.Web.AllowedOrigins)

	proxied := grpc.NewServer()
	server := httptest.NewServer(newMuxHandler(cfg, logger.NewNop(), proxied))
	t.Cleanup(server.Close)

	url := server.URL + rpccatalog.FullMethod(pb.CoreService_ServiceDesc, testStreamEventsMethod)
	req, err := http.NewRequestWithContext(testutil.TestContext(), http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set(grpcconst.HeaderOrigin, cfg.GRPC.Web.AllowedOrigins[0])
	req.Header.Set(testHeaderUpgrade, testUpgradeWebSocket)
	req.Header.Set(testHeaderConnection, testConnectionUpgrade)
	req.Header.Set(testHeaderWebSocketKey, testWebSocketKey)
	req.Header.Set(testHeaderWebSocketVersion, testWebSocketVersion)
	req.Header.Set(testHeaderWebSocketProto, testGRPCWebSocketsProtocol)

	resp, err := server.Client().Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}
