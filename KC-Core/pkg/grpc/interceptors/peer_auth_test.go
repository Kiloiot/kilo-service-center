package interceptors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	testPeerSecret      = "peer-secret-for-tests"
	testWrongPeerSecret = "not-the-peer-secret"
	testTrustedTenant   = "42"
	testTrustedOrg      = "00000000-0000-0000-0000-000000000001"
	testNoPeerSecret    = ""
)

// trustedCall carries valid gateway identity headers and the given peer secrets.
func trustedCall(secrets ...string) context.Context {
	md := metadata.Pairs(
		grpcconst.MetadataKeyInternalTenantID, testTrustedTenant,
		grpcconst.MetadataKeyInternalOrgID, testTrustedOrg,
	)
	for _, secret := range secrets {
		if secret != testNoPeerSecret {
			md.Append(grpcconst.MetadataKeyInternalPeerSecret, secret)
		}
	}
	return metadata.NewIncomingContext(testutil.TestContext(), md)
}

func TestPeerAuthenticator(t *testing.T) {
	configured := NewPeerAuthenticator(testPeerSecret)
	assert.NoError(t, configured.Authenticate(trustedCall(testPeerSecret)))
	for name, ctx := range map[string]context.Context{
		"missing secret":   trustedCall(testNoPeerSecret),
		"wrong secret":     trustedCall(testWrongPeerSecret),
		"secret twice":     trustedCall(testPeerSecret, testPeerSecret),
		"no metadata":      testutil.TestContext(),
		"prefix of secret": trustedCall(testPeerSecret[:len(testPeerSecret)-1]),
	} {
		assert.Equal(t, codes.Unauthenticated, status.Code(configured.Authenticate(ctx)), name)
	}

	unconfigured := NewPeerAuthenticator(testNoPeerSecret)
	assert.NoError(t, unconfigured.Authenticate(trustedCall(testNoPeerSecret)), "no configured secret keeps peers unauthenticated")
	assert.NoError(t, unconfigured.Authenticate(testutil.TestContext()))
}

func TestInternalTrust_RequiresThePeerSecretWhenConfigured(t *testing.T) {
	for _, community := range []bool{false, true} {
		interceptor := NewInternalTrustInterceptor(logger.NewNop(), community).WithPeerSecret(testPeerSecret)
		for _, secret := range []string{testNoPeerSecret, testWrongPeerSecret} {
			called := false
			_, err := interceptor.UnaryInterceptor()(trustedCall(secret), nil, &grpc.UnaryServerInfo{FullMethod: testNonExemptMethod},
				func(context.Context, interface{}) (interface{}, error) { called = true; return nil, nil })
			assert.Equal(t, codes.Unauthenticated, status.Code(err), "community=%v secret=%q", community, secret)
			assert.False(t, called, "identity headers from an unauthenticated peer are never trusted")

			streamCalled := false
			err = interceptor.StreamInterceptor()(nil, &peerStream{ctx: trustedCall(secret)}, &grpc.StreamServerInfo{FullMethod: testNonExemptMethod},
				func(interface{}, grpc.ServerStream) error { streamCalled = true; return nil })
			assert.Equal(t, codes.Unauthenticated, status.Code(err))
			assert.False(t, streamCalled)
		}

		called := false
		_, err := interceptor.UnaryInterceptor()(trustedCall(testPeerSecret), nil, &grpc.UnaryServerInfo{FullMethod: testNonExemptMethod},
			func(context.Context, interface{}) (interface{}, error) { called = true; return nil, nil })
		assert.NoError(t, err)
		assert.True(t, called, "a peer presenting the secret is trusted")
	}
}

func TestInternalTrust_WithoutAPeerSecretTrustsHeadersAsBefore(t *testing.T) {
	interceptor := NewInternalTrustInterceptor(logger.NewNop(), false)
	called := false
	_, err := interceptor.UnaryInterceptor()(trustedCall(testNoPeerSecret), nil, &grpc.UnaryServerInfo{FullMethod: testNonExemptMethod},
		func(context.Context, interface{}) (interface{}, error) { called = true; return nil, nil })
	assert.NoError(t, err)
	assert.True(t, called)
}

type peerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *peerStream) Context() context.Context { return s.ctx }
