package grpc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Status texts injected as causes; the assertions prove they never reach
// the client.
const (
	testCauseDetail   = "internal detail about the cause"
	testNotFoundCause = "gone"
)

// TestToStatusError_TokenWinsOverStatusBearingCause pins the conversion
// order: a TokenError's deliberately chosen token decides the code and
// message even when the wrapped cause carries its own gRPC status, and the
// cause's text never reaches the client.
func TestToStatusError_TokenWinsOverStatusBearingCause(t *testing.T) {
	cause := status.Error(codes.PermissionDenied, testCauseDetail)
	err := ToStatusError(NewTokenError(ErrTokenInternalError, cause))

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, GetGRPCCode(ErrTokenInternalError), st.Code(),
		"the token's code must win over the cause's status")
	assert.Equal(t, ResolveErrorMessage(ErrTokenInternalError), st.Message())
	assert.NotContains(t, st.Message(), "internal detail",
		"the cause text must never reach the client")
}

// TestToStatusError_BareStatusPassesThrough keeps the pass-through for an
// error that carries a genuine gRPC status and no token.
func TestToStatusError_BareStatusPassesThrough(t *testing.T) {
	err := ToStatusError(fmt.Errorf("wrapped: %w", status.Error(codes.NotFound, testNotFoundCause)))

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

// TestToStatusError_UnknownBecomesGenericInternal keeps the generic fallback.
func TestToStatusError_UnknownBecomesGenericInternal(t *testing.T) {
	err := ToStatusError(fmt.Errorf("some infrastructure failure"))

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, GetGRPCCode(ErrTokenInternalError), st.Code())
	assert.NotContains(t, st.Message(), "infrastructure failure")
}

// TestToStatusError_NilStaysNil keeps the nil contract.
func TestToStatusError_NilStaysNil(t *testing.T) {
	assert.NoError(t, ToStatusError(nil))
}
