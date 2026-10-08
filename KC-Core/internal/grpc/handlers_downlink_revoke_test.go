package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/downlinks"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	revokeRequestEUI = "0102030405060708"
	revokeQueueText  = "900"
	revokeQueueID    = uint64(900)
	revokeHolderEUI  = uint64(0x70b3d59cd00009e6)
)

func revokeHandlers(revoker *fakeDownlinkRevoker) *DownlinkHandlers {
	return testCoreService(coreFields{
		endpointSvc: &fakeEndpointSvc{},
		revoker:     revoker,
		log:         logger.NewNop(),
	}).DownlinkHandlers
}

func revokeRequest() *pb.RevokeDownlinkRequest {
	return &pb.RevokeDownlinkRequest{EpEui: revokeRequestEUI, QueueId: revokeQueueText}
}

func requireRevokeCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "expected a gRPC status, got %v", err)
	assert.Equal(t, want, st.Code(), st.Message())
}

// TestRevokeDownlink_NamesTheDownlinkOfTheRequestOwner: the revoke path is
// handed the caller's tenant, organization and endpoint with the queue id,
// so a queue id of another organization, endpoint or tenant revokes nothing
// and reads as not found (pinned at the repository).
func TestRevokeDownlink_NamesTheDownlinkOfTheRequestOwner(t *testing.T) {
	revoker := &fakeDownlinkRevoker{}
	h := revokeHandlers(revoker)

	_, err := h.RevokeDownlink(testutil.TestContextWithTenantAndOrg(1, downlinkTestOrg), revokeRequest())

	require.NoError(t, err)
	require.Len(t, revoker.refs, 1)
	ref := revoker.refs[0]
	assert.Equal(t, int64(1), ref.TenantID)
	assert.Equal(t, revokeQueueID, ref.QueID)
	require.NotNil(t, ref.OrganizationID)
	assert.Equal(t, downlinkTestOrg, *ref.OrganizationID)
	require.NotNil(t, ref.EpEUI)
	assert.Equal(t, revokeRequestEUI, mioty.FormatEUI64(*ref.EpEUI))
}

// TestRevokeDownlink_PendingDownlinkIsRevoked: a downlink no base station
// holds is revoked in the queue and the response says so.
func TestRevokeDownlink_PendingDownlinkIsRevoked(t *testing.T) {
	revoker := &fakeDownlinkRevoker{}
	h := revokeHandlers(revoker)

	resp, err := h.RevokeDownlink(testutil.TestContextWithTenantAndOrg(1, downlinkTestOrg), revokeRequest())

	require.NoError(t, err)
	assert.Equal(t, downlinks.RevokeStatusRevoked, resp.Status)
	assert.Equal(t, []uint64{revokeQueueID}, revoker.revoked)
}

// TestRevokeDownlink_HeldDownlinkRevokeIsInitiated: a downlink a base station
// holds is revoked once the station confirms, so the revoke is only initiated.
func TestRevokeDownlink_HeldDownlinkRevokeIsInitiated(t *testing.T) {
	h := revokeHandlers(&fakeDownlinkRevoker{bsEui: revokeHolderEUI})

	resp, err := h.RevokeDownlink(testutil.TestContextWithTenantAndOrg(1, downlinkTestOrg), revokeRequest())

	require.NoError(t, err)
	assert.Equal(t, downlinks.RevokeStatusInitiated, resp.Status)
}

// TestRevokeDownlink_RevokeFailuresMapToTheirStatus pins how the revoke path's
// outcomes reach the client; a queue id of another owner is the first case.
func TestRevokeDownlink_RevokeFailuresMapToTheirStatus(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"unknown, foreign or finished downlink", scheduler.ErrSchedulerQueueNotFound, codes.NotFound},
		{"holding station disconnected", scheduler.ErrSchedulerResourceMissing, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenHandshakeIncomplete)},
		{"send failure", assert.AnError, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDownlinkRevokeFailed)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := revokeHandlers(&fakeDownlinkRevoker{bsEui: revokeHolderEUI, err: tc.err})

			_, err := h.RevokeDownlink(testutil.TestContextWithTenantAndOrg(1, downlinkTestOrg), revokeRequest())

			requireRevokeCode(t, err, tc.want)
		})
	}
}

// TestRevokeDownlink_RefusesAMalformedQueueID: the queue id is a
// non-negative number; nothing else reaches the revoke path.
func TestRevokeDownlink_RefusesAMalformedQueueID(t *testing.T) {
	for queueID, token := range map[string]string{
		"":    grpcerrors.ErrTokenQueueIDRequired,
		"abc": grpcerrors.ErrTokenInvalidQueueIDFormat,
		"-1":  grpcerrors.ErrTokenQueueIDNonNegative,
	} {
		revoker := &fakeDownlinkRevoker{}
		h := revokeHandlers(revoker)

		_, err := h.RevokeDownlink(testutil.TestContextWithTenantAndOrg(1, downlinkTestOrg),
			&pb.RevokeDownlinkRequest{EpEui: revokeRequestEUI, QueueId: queueID})

		requireRevokeCode(t, err, grpcerrors.GetGRPCCode(token))
		assert.Empty(t, revoker.revoked, queueID)
	}
}
