package grpc

import (
	"errors"
	"fmt"
	"testing"

	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"
)

var errScaciMappingTestInner = errors.New("inner failure")

func TestMapSCACIErrorToGRPC(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want string
	}{
		{name: "payload too large", err: &scaci.DLDataQueueError{Token: scaci.ErrDLPayloadTooLarge, POSIX: scaci.POSIX_EINVAL}, want: grpcerrors.ErrTokenDownlinkPayloadTooLarge},
		{name: "endpoint not found", err: &scaci.DLDataQueueError{Token: scaci.ErrEndpointNotFound, POSIX: scaci.POSIX_ENOENT}, want: grpcerrors.ErrTokenScaciEndpointNotFound},
		{name: "unidirectional endpoint", err: &scaci.DLDataQueueError{Token: scaci.ErrEndpointNotBidirectional, POSIX: scaci.POSIX_ENOTSUP}, want: grpcerrors.ErrTokenScaciEndpointNotBidirectional},
		{name: "wrapped queue error", err: fmt.Errorf("queue: %w", &scaci.DLDataQueueError{Token: scaci.ErrFailedPersistDownlink}), want: grpcerrors.ErrTokenScaciFailedPersistDownlink},
		{name: "unknown token", err: &scaci.DLDataQueueError{Token: "scaci.error.something_else"}, want: grpcerrors.ErrTokenScaciOperationFailed},
		{name: "plain error", err: errScaciMappingTestInner, want: grpcerrors.ErrTokenScaciOperationFailed},
		{name: "text mentioning a token is not a token", err: fmt.Errorf("%s: %w", scaci.ErrEndpointNotFound, errScaciMappingTestInner), want: grpcerrors.ErrTokenScaciOperationFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mapSCACIErrorToGRPC(tc.err)
			st, ok := status.FromError(got)
			require.True(t, ok)
			assert.Equal(t, grpcerrors.GetGRPCCode(tc.want), st.Code())
			assert.Equal(t, grpcerrors.ResolveErrorMessage(tc.want), st.Message())
		})
	}
	assert.NoError(t, mapSCACIErrorToGRPC(nil))
}
