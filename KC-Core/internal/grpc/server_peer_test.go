package grpc

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	testServerPeerSecret = "peer-secret-for-tests"
	testServerWrongPeer  = "not-the-peer-secret"
	testServerNoPeer     = ""
	testServerTrustOn    = true
)

// statusOnlyCore answers GetSystemStatus, an authenticated but org-exempt RPC.
type statusOnlyCore struct {
	pb.UnimplementedCoreServiceServer
}

func (statusOnlyCore) GetSystemStatus(context.Context, *emptypb.Empty) (*pb.SystemStatus, error) {
	return &pb.SystemStatus{}, nil
}

func TestServer_InternalTrustRequiresTheConfiguredPeerSecret(t *testing.T) {
	srv, err := NewServer(Config{
		Log:                  logger.NewNop(),
		Host:                 testServerLoopback,
		Port:                 ephemeralPort,
		InternalTrustEnabled: testServerTrustOn,
		PeerSecret:           testServerPeerSecret,
		DefaultOrgResolver:   &fakeDefaultOrgResolver{org: communityTestOrg},
		RoleSource:           fixedRoles(authz.AllRoles),
		HTTPConfig:           HTTPServerConfig{WriteTimeout: testServerStopTimeout},
	})
	require.NoError(t, err)
	pb.RegisterCoreServiceServer(srv.GetServer(), statusOnlyCore{})
	go func() { _ = srv.Start() }()
	t.Cleanup(srv.Stop)

	conn, err := grpclib.NewClient(srv.listener.Addr().String(), grpclib.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	client := pb.NewCoreServiceClient(conn)

	call := func(secret string) error {
		md := metadata.Pairs(grpcconst.MetadataKeyInternalTenantID, strconv.FormatInt(communityTestTenant, 10))
		if secret != testServerNoPeer {
			md.Set(grpcconst.MetadataKeyInternalPeerSecret, secret)
		}
		_, err := client.GetSystemStatus(metadata.NewOutgoingContext(testutil.TestContext(), md), &emptypb.Empty{})
		return err
	}

	assert.Equal(t, codes.Unauthenticated, status.Code(call(testServerNoPeer)), "gateway headers without the peer secret are not trusted")
	assert.Equal(t, codes.Unauthenticated, status.Code(call(testServerWrongPeer)))
	assert.NoError(t, call(testServerPeerSecret), "the gateway presenting the peer secret is served")
}

// TestServer_EnforcesTheRolePolicy: the served interceptor chain refuses a
// caller whose roles do not meet the method's requirement, and a server
// without a role source is never built.
func TestServer_EnforcesTheRolePolicy(t *testing.T) {
	_, err := NewServer(Config{
		Log:                  logger.NewNop(),
		Host:                 testServerLoopback,
		Port:                 ephemeralPort,
		InternalTrustEnabled: testServerTrustOn,
		PeerSecret:           testServerPeerSecret,
		DefaultOrgResolver:   &fakeDefaultOrgResolver{org: communityTestOrg},
		HTTPConfig:           HTTPServerConfig{WriteTimeout: testServerStopTimeout},
	})
	require.Error(t, err, "a server without a role source must not start")

	srv, err := NewServer(Config{
		Log:                  logger.NewNop(),
		Host:                 testServerLoopback,
		Port:                 ephemeralPort,
		InternalTrustEnabled: testServerTrustOn,
		PeerSecret:           testServerPeerSecret,
		DefaultOrgResolver:   &fakeDefaultOrgResolver{org: communityTestOrg},
		RoleSource:           fixedRoles(authz.Roles{}),
		HTTPConfig:           HTTPServerConfig{WriteTimeout: testServerStopTimeout},
	})
	require.NoError(t, err)
	pb.RegisterCoreServiceServer(srv.GetServer(), statusOnlyCore{})
	go func() { _ = srv.Start() }()
	t.Cleanup(srv.Stop)

	conn, err := grpclib.NewClient(srv.listener.Addr().String(), grpclib.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	md := metadata.Pairs(
		grpcconst.MetadataKeyInternalTenantID, strconv.FormatInt(communityTestTenant, 10),
		grpcconst.MetadataKeyInternalPeerSecret, testServerPeerSecret,
	)
	_, err = pb.NewCoreServiceClient(conn).GetSystemStatus(metadata.NewOutgoingContext(testutil.TestContext(), md), &emptypb.Empty{})
	assert.Equal(t, codes.PermissionDenied, status.Code(err), "a caller without any role reads nothing")
}
