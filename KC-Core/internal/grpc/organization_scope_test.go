package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	communityTestTenant = int64(7)
	ephemeralPort       = 0
)

var (
	communityTestOrg    = uuid.MustParse("6aa6b3db-ceaa-4a71-8ece-59cc2263f019")
	errCommunityOrgDown = errors.New("organization store down")
)

type fakeDefaultOrgResolver struct {
	org   uuid.UUID
	err   error
	calls int
}

func (f *fakeDefaultOrgResolver) GetDefaultOrgForTenant(_ context.Context, _ int64) (uuid.UUID, error) {
	f.calls++
	return f.org, f.err
}

type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s fakeServerStream) Context() context.Context { return s.ctx }

func TestCommunityContext_InjectsTenantAndDefaultOrganization(t *testing.T) {
	orgs := &fakeDefaultOrgResolver{org: communityTestOrg}
	community := newCommunityContext(communityTestTenant, orgs)

	var unaryTenant int64
	var unaryOrg uuid.UUID
	_, err := community.unaryInterceptor()(testutil.TestContext(), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, _ interface{}) (interface{}, error) {
		unaryTenant, _ = pkgcontext.GetTenantID(ctx)
		unaryOrg, _ = pkgcontext.RequireOrganizationID(ctx)
		return nil, nil
	})
	require.NoError(t, err)
	assert.Equal(t, communityTestTenant, unaryTenant)
	assert.Equal(t, communityTestOrg, unaryOrg)

	var streamOrg uuid.UUID
	err = community.streamInterceptor()(nil, fakeServerStream{ctx: testutil.TestContext()}, &grpc.StreamServerInfo{}, func(_ interface{}, ss grpc.ServerStream) error {
		streamOrg, _ = pkgcontext.RequireOrganizationID(ss.Context())
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, communityTestOrg, streamOrg)
	assert.Equal(t, 2, orgs.calls)
}

func TestCommunityContext_RefusesWhenTheDefaultOrganizationCannotBeResolved(t *testing.T) {
	community := newCommunityContext(communityTestTenant, &fakeDefaultOrgResolver{err: errCommunityOrgDown})
	handlerCalled := false

	_, err := community.unaryInterceptor()(testutil.TestContext(), nil, &grpc.UnaryServerInfo{}, func(context.Context, interface{}) (interface{}, error) {
		handlerCalled = true
		return nil, nil
	})
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgResolutionFailed), status.Code(err))
	assert.False(t, handlerCalled)

	err = community.streamInterceptor()(nil, fakeServerStream{ctx: testutil.TestContext()}, &grpc.StreamServerInfo{}, func(interface{}, grpc.ServerStream) error {
		handlerCalled = true
		return nil
	})
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgResolutionFailed), status.Code(err))
	assert.False(t, handlerCalled)
}

func TestNewServer_CommunityModeRequiresADefaultOrganizationResolver(t *testing.T) {
	_, err := NewServer(Config{Log: logger.NewNop(), Host: "127.0.0.1", Port: ephemeralPort, DefaultTenantID: communityTestTenant})
	require.Error(t, err)
	assert.Contains(t, err.Error(), errMsgDefaultOrgResolverRequired)
}

// The three downlink handlers that write or read organization-scoped rows
// refuse to run without an organization and otherwise hand the storage layer
// exactly the organization of the request.
func TestDownlinkHandlers_RequireTheOrganizationOfTheRequest(t *testing.T) {
	queuer := &fakeSCACIQueuer{}
	pending := &pendingMessageSvc{}
	results := &filterMessageSvc{}
	svc := testCoreService(coreFields{scaciQueuer: queuer, messageSvc: results, log: logger.NewNop()})

	withoutOrg := testutil.TestContextWithTenant(communityTestTenant)
	withOrg := testutil.TestContextWithTenantAndOrg(communityTestTenant, communityTestOrg)

	_, err := svc.SendDownlink(withoutOrg, &pb.SendDownlinkRequest{EpEui: pendingTestEpEUI, Payloads: [][]byte{{0x01}}})
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMissingOrgContext), status.Code(err))
	assert.Equal(t, 0, queuer.calls)
	_, err = svc.SendDownlink(withOrg, &pb.SendDownlinkRequest{EpEui: pendingTestEpEUI, Payloads: [][]byte{{0x01}}})
	require.NoError(t, err)
	require.NotNil(t, queuer.lastOrgID)
	assert.Equal(t, communityTestOrg, *queuer.lastOrgID)

	svc.useDownlinks(downlinkFakes{queuer: queuer, messages: pending})
	_, err = svc.UpdatePendingDownlink(withoutOrg, &pb.UpdatePendingDownlinkRequest{EpEui: pendingTestEpEUI, QueId: pendingTestQueID, Payloads: [][]byte{{0x01}}})
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMissingOrgContext), status.Code(err))
	assert.Equal(t, 0, pending.calls)
	_, err = svc.UpdatePendingDownlink(withOrg, &pb.UpdatePendingDownlinkRequest{EpEui: pendingTestEpEUI, QueId: pendingTestQueID, Payloads: [][]byte{{0x01}}})
	require.NoError(t, err)
	require.NotNil(t, pending.lastOrgID)
	assert.Equal(t, communityTestOrg, *pending.lastOrgID)

	svc.useDownlinks(downlinkFakes{queuer: queuer, messages: results})
	_, err = svc.GetDownlinkResults(withoutOrg, &pb.GetDownlinkResultsRequest{})
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMissingOrgContext), status.Code(err))
	assert.Equal(t, 0, results.calls)
	_, err = svc.GetDownlinkResults(withOrg, &pb.GetDownlinkResultsRequest{})
	require.NoError(t, err)
	require.NotNil(t, results.lastOrgID)
	assert.Equal(t, communityTestOrg, *results.lastOrgID)
}
