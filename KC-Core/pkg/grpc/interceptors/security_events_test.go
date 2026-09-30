package interceptors

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	testHMACSecret     = "interceptor-test-secret-of-32-bytes!"
	testMalformedToken = "header.payload.signature"
	testAuthEnabled    = true
)

// TestSecurityEvents_OutliveACallerThatHungUp: a refusal is audited even when
// the caller has already gone, so hanging up cannot suppress the record.
func TestSecurityEvents_OutliveACallerThatHungUp(t *testing.T) {
	handler := func(context.Context, interface{}) (interface{}, error) {
		t.Fatal("a refused request reached the handler")
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: testNonExemptMethod}

	cases := []struct {
		name      string
		intercept func(events *recordingEventWriter) grpc.UnaryServerInterceptor
		ctx       func() context.Context
	}{
		{
			name: "auth refuses a token that fails validation",
			intercept: func(events *recordingEventWriter) grpc.UnaryServerInterceptor {
				ai, err := NewAuthInterceptor(AuthConfig{Enabled: testAuthEnabled, HMACSecret: testHMACSecret, EventWriter: events})
				if err != nil {
					t.Fatalf("NewAuthInterceptor: %v", err)
				}
				return ai.UnaryInterceptor()
			},
			ctx: func() context.Context {
				return metadata.NewIncomingContext(testutil.TestContext(),
					metadata.Pairs(grpcconst.MetadataKeyAuthorization, grpcconst.BearerPrefix+testMalformedToken))
			},
		},
		{
			name: "internal trust refuses a request without identity headers",
			intercept: func(events *recordingEventWriter) grpc.UnaryServerInterceptor {
				return NewInternalTrustInterceptor(logger.NewNop(), false).WithEventWriter(events).UnaryInterceptor()
			},
			ctx: testutil.TestContext,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events := &recordingEventWriter{}
			ctx, hangUp := context.WithCancel(tc.ctx())
			hangUp()

			if _, err := tc.intercept(events)(ctx, nil, info, handler); err == nil {
				t.Fatal("the request was admitted")
			}
			if len(events.writeErrs) != 1 {
				t.Fatalf("recorded %d events, want 1", len(events.writeErrs))
			}
			if events.writeErrs[0] != nil {
				t.Fatalf("the event was written with a finished context: %v", events.writeErrs[0])
			}
		})
	}
}
