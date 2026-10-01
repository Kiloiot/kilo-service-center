package grpc

import (
	"context"
	"errors"
	"testing"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	pkgfederation "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/federation"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"
)

const (
	testMsgAlreadyShaped = "already shaped"
)

// fakeBootstrapService returns canned results for the adapter under test.
type fakeBootstrapService struct {
	status        pkgfederation.CEStatus
	statusErr     error
	onboarding    pkgfederation.OnboardingResult
	onboardingErr error
}

func (f *fakeBootstrapService) Status(context.Context) (pkgfederation.CEStatus, error) {
	return f.status, f.statusErr
}

func (f *fakeBootstrapService) CompleteOnboarding(context.Context, string) (pkgfederation.OnboardingResult, error) {
	return f.onboarding, f.onboardingErr
}

// repoCause is a deliberately recognizable cause string that must never reach
// the client-visible status message.
const repoCause = "pq: connection refused on 10.0.0.7"

func assertCatalogStatus(t *testing.T, err error, token string) {
	t.Helper()
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "adapter must return a gRPC status error")
	assert.Equal(t, grpcerrors.GetGRPCCode(token), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(token), st.Message())
	assert.NotContains(t, st.Message(), repoCause)
	assert.NotContains(t, st.Message(), token, "catalog message must not embed the raw token")
}

func TestGetCEStatusErrorMapping(t *testing.T) {
	cases := []struct {
		name  string
		cause error
		token string
	}{
		{
			name:  "non-community edition",
			cause: pkgfederation.ErrNotCommunityEdition,
			token: grpcerrors.ErrTokenCEStatusUnavailable,
		},
		{
			name:  "repository failure",
			cause: errors.Join(pkgfederation.ErrInstallationRead, errors.New(repoCause)),
			token: grpcerrors.ErrTokenInternalError,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := NewCEBootstrapAdapter(&fakeBootstrapService{statusErr: tc.cause})
			_, err := adapter.GetCEStatus(testutil.TestContext(), &pb.GetCEStatusRequest{})
			assertCatalogStatus(t, err, tc.token)
		})
	}
}

func TestCompleteCEOnboardingErrorMapping(t *testing.T) {
	cases := []struct {
		name  string
		cause error
		token string
	}{
		{
			name:  "non-community edition",
			cause: pkgfederation.ErrNotCommunityEdition,
			token: grpcerrors.ErrTokenCEOnboardingUnavailable,
		},
		{
			name:  "missing company name",
			cause: pkgfederation.ErrCompanyNameRequired,
			token: grpcerrors.ErrTokenCompanyNameRequired,
		},
		{
			name:  "onboarding already completed",
			cause: pkgfederation.ErrOnboardingAlreadyCompleted,
			token: grpcerrors.ErrTokenCEOnboardingCompleted,
		},
		{
			name:  "repository write failure",
			cause: errors.Join(pkgfederation.ErrInstallationWrite, errors.New(repoCause)),
			token: grpcerrors.ErrTokenInternalError,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := NewCEBootstrapAdapter(&fakeBootstrapService{onboardingErr: tc.cause})
			_, err := adapter.CompleteCEOnboarding(testutil.TestContext(), &pb.CompleteCEOnboardingRequest{CompanyName: "Acme"})
			assertCatalogStatus(t, err, tc.token)
		})
	}
}

func TestBootstrapAdapterSuccessPassthrough(t *testing.T) {
	adapter := NewCEBootstrapAdapter(&fakeBootstrapService{
		status:     pkgfederation.CEStatus{CEID: "ce-1", CompanyName: "Acme", FederationConnected: true},
		onboarding: pkgfederation.OnboardingResult{CEID: "ce-1", CompanyName: "Acme"},
	})

	st, err := adapter.GetCEStatus(testutil.TestContext(), &pb.GetCEStatusRequest{})
	require.NoError(t, err)
	assert.Equal(t, "ce-1", st.GetCeId())
	assert.True(t, st.GetFederationConnected())

	ob, err := adapter.CompleteCEOnboarding(testutil.TestContext(), &pb.CompleteCEOnboardingRequest{CompanyName: "Acme"})
	require.NoError(t, err)
	assert.Equal(t, "Acme", ob.GetCompanyName())
}

func TestToStatusError(t *testing.T) {
	t.Run("nil stays nil", func(t *testing.T) {
		assert.NoError(t, grpcerrors.ToStatusError(nil))
	})
	t.Run("existing status passes through", func(t *testing.T) {
		orig := status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidArgument), testMsgAlreadyShaped)
		converted := grpcerrors.ToStatusError(orig)
		st, ok := status.FromError(converted)
		require.True(t, ok)
		assert.Equal(t, "already shaped", st.Message())
	})
	t.Run("plain error becomes generic internal", func(t *testing.T) {
		converted := grpcerrors.ToStatusError(errors.New(repoCause))
		st, ok := status.FromError(converted)
		require.True(t, ok)
		assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError), st.Code())
		assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError), st.Message())
		assert.NotContains(t, st.Message(), repoCause)
	})
}
