package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
)

const orderTestMalformedEUI = "not-an-eui"

// TestListings_RefuseANegativePageToken: a negative offset is no page of
// any listing, refused before a store is asked.
func TestListings_RefuseANegativePageToken(t *testing.T) {
	svc := newRetainedService(t, &retainedFakes{})
	const negative = "-20"

	_, err := svc.ListMessages(ownerCtx(), &pb.ListMessagesRequest{PageToken: negative})
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken), status.Code(err), "ListMessages")
	_, err = svc.ListDownlinkQueue(ownerCtx(), &pb.ListDownlinkQueueRequest{PageToken: negative})
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken), status.Code(err), "ListDownlinkQueue")
}

// TestDownlinkWrites_ReadTheEndpointBeforeTheContent: a request that names
// no endpoint is refused for that, whatever its content.
func TestDownlinkWrites_ReadTheEndpointBeforeTheContent(t *testing.T) {
	svc := newRetainedService(t, &retainedFakes{})
	twoPayloads := [][]byte{{0x01}, {0x02}}

	_, err := svc.SendDownlink(ownerCtx(), &pb.SendDownlinkRequest{EpEui: orderTestMalformedEUI, Payloads: twoPayloads})
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEUIFormat), status.Code(err), status.Convert(err).Message())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEUIFormat), status.Convert(err).Message())
}
