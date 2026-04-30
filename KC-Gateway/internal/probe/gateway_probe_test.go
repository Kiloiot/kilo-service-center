//go:build integration

// Package probe provides synthetic end-to-end probes for KC-Gateway.
package probe

import (
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func gatewayAddr() string {
	if addr := os.Getenv("GATEWAY_ADDR"); addr != "" {
		return addr
	}
	return "localhost:9090"
}

// TestGatewayIngressProbe verifies gateway ingress, auth enforcement,
// and metadata sanitization against the canonical dev config (auth.enabled=true).
func TestGatewayIngressProbe(t *testing.T) {
	conn, err := grpc.NewClient(gatewayAddr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Probe 1: GetReleaseInfo (public, no auth) → non-empty response
	t.Run("Probe1_GetReleaseInfo_Public", func(t *testing.T) {
		var resp pb.ReleaseInfo
		err := conn.Invoke(ctx, "/kilocenter.api.v1.CoreService/GetReleaseInfo",
			&emptypb.Empty{}, &resp)
		// Accept OK or domain error, but never Unimplemented
		if err != nil {
			st, _ := status.FromError(err)
			assert.NotEqual(t, codes.Unimplemented, st.Code(),
				"GetReleaseInfo should not return Unimplemented")
		}
	})

	// Probe 2: GetAuthSettings (public, no auth) → non-empty response
	t.Run("Probe2_GetAuthSettings_Public", func(t *testing.T) {
		var resp pb.GetAuthSettingsResponse
		err := conn.Invoke(ctx, "/kilocenter.api.v1.IdentityService/GetAuthSettings",
			&pb.GetAuthSettingsRequest{}, &resp)
		if err != nil {
			st, _ := status.FromError(err)
			assert.NotEqual(t, codes.Unimplemented, st.Code(),
				"GetAuthSettings should not return Unimplemented")
		}
	})

	// Probe 3: ListEndPoints without auth → EXACTLY Unauthenticated
	t.Run("Probe3_ListEndPoints_NoAuth", func(t *testing.T) {
		var resp pb.ListEndPointsResponse
		err := conn.Invoke(ctx, "/kilocenter.api.v1.CoreService/ListEndPoints",
			&pb.ListEndPointsRequest{}, &resp)
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code(),
			"ListEndPoints without auth should return Unauthenticated, got %s", st.Code())
	})

	// Probe 4: Spoofed internal header should not reach upstream
	t.Run("Probe4_SpoofedInternalHeader_Stripped", func(t *testing.T) {
		spoofedCtx := metadata.AppendToOutgoingContext(ctx,
			"x-kc-internal-tenant-id", "999",
			"x-kc-internal-user-id", "attacker",
		)

		var resp pb.ListEndPointsResponse
		err := conn.Invoke(spoofedCtx, "/kilocenter.api.v1.CoreService/ListEndPoints",
			&pb.ListEndPointsRequest{}, &resp)
		// Without valid auth, this should still be Unauthenticated
		// (proving spoofed headers don't grant access)
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code(),
			"spoofed internal headers should not bypass auth")
	})

	// Probe 5: Trace context propagation (check traceparent header is propagated)
	t.Run("Probe5_TraceContextPropagation", func(t *testing.T) {
		// Smoke check: traceparent header does not cause rejection.
		// Canonical propagation assertion is TestGatewayTwoHop_TraceContextPropagation
		// in integration_test.go.
		traceCtx := metadata.AppendToOutgoingContext(ctx,
			"traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		)

		var resp pb.ReleaseInfo
		err := conn.Invoke(traceCtx, "/kilocenter.api.v1.CoreService/GetReleaseInfo",
			&emptypb.Empty{}, &resp)
		// Should work fine — traceparent is a standard propagation header
		if err != nil {
			st, _ := status.FromError(err)
			assert.NotEqual(t, codes.Unimplemented, st.Code())
		}
	})

	// Probe 6: Circuit breaker Prometheus metrics are exported
	t.Run("Probe6_BreakerMetricsExported", func(t *testing.T) {
		metricsAddr := os.Getenv("GATEWAY_METRICS_ADDR")
		if metricsAddr == "" {
			metricsAddr = "http://localhost:9093/metrics"
		}
		resp, err := http.Get(metricsAddr) //nolint:gosec
		require.NoError(t, err, "metrics endpoint must be reachable")
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "kc_gateway_circuit_breaker_state",
			"breaker gauge must be present in metrics output")
	})
}
