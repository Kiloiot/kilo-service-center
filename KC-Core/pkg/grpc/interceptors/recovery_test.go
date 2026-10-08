package interceptors

import (
	"context"
	"testing"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// Synthetic method names for the interceptor info structs.
const (
	testMethodPanics       = "/test.Service/Panics"
	testMethodPanicsStream = "/test.Service/PanicsStream"
)

// recoveryTestStream satisfies grpc.ServerStream for the stream interceptor.
type recoveryTestStream struct {
	grpc.ServerStream
}

func (recoveryTestStream) Context() context.Context { return testutil.TestContext() }

// TestRecoveryInterceptor_Unary verifies a panicking unary handler yields the
// generic internal catalog error - never the panic value - and a healthy
// handler passes through untouched.
func TestRecoveryInterceptor_Unary(t *testing.T) {
	interceptor := NewRecoveryInterceptor(nil).UnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: testMethodPanics}

	resp, err := interceptor(testutil.TestContext(), nil, info,
		func(context.Context, interface{}) (interface{}, error) {
			panic("sensitive internal detail")
		})
	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcconst.GetGRPCCode(grpcconst.ErrTokenInternalError), st.Code())
	assert.Equal(t, grpcconst.ResolveErrorMessage(grpcconst.ErrTokenInternalError), st.Message(),
		"the client must receive only the generic catalog message")
	assert.NotContains(t, st.Message(), "sensitive internal detail")

	resp, err = interceptor(testutil.TestContext(), nil, info,
		func(context.Context, interface{}) (interface{}, error) { return "ok", nil })
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

// TestRecoveryInterceptor_Stream mirrors the unary contract for streams.
func TestRecoveryInterceptor_Stream(t *testing.T) {
	interceptor := NewRecoveryInterceptor(nil).StreamInterceptor()
	info := &grpc.StreamServerInfo{FullMethod: testMethodPanicsStream}

	err := interceptor(nil, recoveryTestStream{}, info,
		func(interface{}, grpc.ServerStream) error { panic("stream detail") })
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcconst.GetGRPCCode(grpcconst.ErrTokenInternalError), st.Code())
	assert.NotContains(t, st.Message(), "stream detail")

	require.NoError(t, interceptor(nil, recoveryTestStream{}, info,
		func(interface{}, grpc.ServerStream) error { return nil }))
}
