package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

type failingStream struct {
	grpc.ServerStream
}

func (failingStream) Context() context.Context { return testutil.TestContext() }

// A request the service refuses is the caller's mistake and is logged as a
// warning; only a failure of the service itself is logged as an error.
func TestFailedCall_IsAnErrorOnlyWhenTheServiceFailed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		err   error
		level string
	}{
		{"invalid argument", status.Error(codes.InvalidArgument, "bad"), "warn"},
		{"not found", status.Error(codes.NotFound, "gone"), "warn"},
		{"already exists", status.Error(codes.AlreadyExists, "dup"), "warn"},
		{"permission denied", status.Error(codes.PermissionDenied, "no"), "warn"},
		{"unauthenticated", status.Error(codes.Unauthenticated, "who"), "warn"},
		{"failed precondition", status.Error(codes.FailedPrecondition, "state"), "warn"},
		{"out of range", status.Error(codes.OutOfRange, "range"), "warn"},
		{"aborted", status.Error(codes.Aborted, "conflict"), "warn"},
		{"canceled", status.Error(codes.Canceled, "gone away"), "warn"},
		{"internal", status.Error(codes.Internal, "boom"), "error"},
		{"unavailable", status.Error(codes.Unavailable, "down"), "error"},
		{"deadline exceeded", status.Error(codes.DeadlineExceeded, "slow"), "error"},
		{"plain error", errors.New("boom"), "error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			unaryLog := newCapturingLogger()
			_, err := unaryInterceptor(unaryLog)(testutil.TestContext(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Unary"},
				func(context.Context, interface{}) (interface{}, error) { return nil, tc.err })
			require.ErrorIs(t, err, tc.err)
			assert.Equal(t, tc.level, levelOf(t, unaryLog, LogGRPCUnaryCallFailed))

			streamLog := newCapturingLogger()
			err = streamInterceptor(streamLog)(nil, failingStream{}, &grpc.StreamServerInfo{FullMethod: "/svc/Stream"},
				func(interface{}, grpc.ServerStream) error { return tc.err })
			require.ErrorIs(t, err, tc.err)
			assert.Equal(t, tc.level, levelOf(t, streamLog, LogGRPCStreamCallFailed))
		})
	}
}

func levelOf(t *testing.T, log *capturingLogger, message string) string {
	t.Helper()
	for _, entry := range log.getEntries() {
		if entry.message == message {
			return entry.level
		}
	}
	t.Fatalf("no %q entry logged", message)
	return ""
}
