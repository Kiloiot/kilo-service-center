package builders

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	testPeerSecret      = "peer-secret-for-tests"
	testWrongPeerSecret = "not-the-peer-secret"
	testNoPeerSecret    = ""
	testTrustedTenant   = "7"
	testPlatformTenant  = int64(7)
	testTrustedMethod   = "/kilocenter.api.v1.IdentityService/ListOrganizations"
	testInternalMethod  = "/kilocenter.api.v1.IdentityInternalService/CheckServerAdmin"
)

var testDefaultOrg = uuid.MustParse("6aa6b3db-ceaa-4a71-8ece-59cc2263f019")

type fixedDefaultOrg struct{}

func (fixedDefaultOrg) GetDefaultOrgForTenant(context.Context, int64) (uuid.UUID, error) {
	return testDefaultOrg, nil
}

type recordedEvents struct{ events []*models.SystemEvent }

func (r *recordedEvents) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	r.events = append(r.events, event)
	return nil
}

// peerCall carries the gateway's tenant header, no org header, and the given peer secret.
func peerCall(secret string) context.Context {
	md := metadata.Pairs(grpcconst.MetadataKeyInternalTenantID, testTrustedTenant)
	if secret != testNoPeerSecret {
		md.Set(grpcconst.MetadataKeyInternalPeerSecret, secret)
	}
	return metadata.NewIncomingContext(testutil.TestContext(), md)
}

func identityInterceptors(secret string, events *recordedEvents) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	peers := interceptors.NewPeerAuthenticator(secret)
	trust := trustGatewayPeers(logger.NewNop(), secret, fixedDefaultOrg{}, events, testPlatformTenant)
	return methodAwareUnary(peers, trust), methodAwareStream(peers, trust)
}

type servedCall struct {
	called bool
	org    uuid.UUID
}

func callUnary(ctx context.Context, interceptor grpc.UnaryServerInterceptor, method string) (servedCall, error) {
	var served servedCall
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: method}, func(ctx context.Context, _ interface{}) (interface{}, error) {
		served.called = true
		served.org, _ = pkgcontext.GetOrganizationID(ctx)
		return nil, nil
	})
	return served, err
}

type ctxStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *ctxStream) Context() context.Context { return s.ctx }

func callStream(ctx context.Context, interceptor grpc.StreamServerInterceptor, method string) (bool, error) {
	called := false
	err := interceptor(nil, &ctxStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: method}, func(interface{}, grpc.ServerStream) error {
		called = true
		return nil
	})
	return called, err
}

func TestIdentityInterceptors_RequireThePeerSecretWhenConfigured(t *testing.T) {
	events := &recordedEvents{}
	unary, stream := identityInterceptors(testPeerSecret, events)

	for _, method := range []string{testTrustedMethod, testInternalMethod} {
		for _, secret := range []string{testNoPeerSecret, testWrongPeerSecret} {
			served, err := callUnary(peerCall(secret), unary, method)
			assert.False(t, served.called, "%s with secret %q must be refused", method, secret)
			assert.Equal(t, codes.Unauthenticated, status.Code(err), "%s with secret %q", method, secret)

			called, err := callStream(peerCall(secret), stream, method)
			assert.False(t, called)
			assert.Equal(t, codes.Unauthenticated, status.Code(err))
		}
		served, err := callUnary(peerCall(testPeerSecret), unary, method)
		assert.NoError(t, err, method)
		assert.True(t, served.called, "%s with the peer secret is served", method)
	}
	assert.NotEmpty(t, events.events, "a refused trusted call leaves a security event")
}

func TestIdentityInterceptors_WithoutAPeerSecretKeepWorking(t *testing.T) {
	unary, stream := identityInterceptors(testNoPeerSecret, &recordedEvents{})

	for _, method := range []string{testTrustedMethod, testInternalMethod} {
		served, err := callUnary(peerCall(testNoPeerSecret), unary, method)
		assert.NoError(t, err, method)
		assert.True(t, served.called, method)

		called, err := callStream(peerCall(testNoPeerSecret), stream, method)
		assert.NoError(t, err, method)
		assert.True(t, called, method)
	}
}

func TestIdentityInterceptors_TrustedCallWithoutOrgRunsUnderTheDefaultOrganization(t *testing.T) {
	unary, _ := identityInterceptors(testPeerSecret, &recordedEvents{})

	served, err := callUnary(peerCall(testPeerSecret), unary, testTrustedMethod)
	assert.NoError(t, err)
	assert.Equal(t, testDefaultOrg, served.org)
}
