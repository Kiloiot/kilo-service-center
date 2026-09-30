package grpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// terminalOnlyResults refuses a status filter that is not terminal, as the
// results repository does.
type terminalOnlyResults struct {
	fakeMessageSvc
	calls int
}

func (r *terminalOnlyResults) GetDownlinkResults(_ context.Context, _ int64, _ *uuid.UUID, filter storage.DownlinkResultFilter, _, _ int) ([]*storage.DownlinkMessage, int, error) {
	r.calls++
	if filter.Status != "" && !mioty.DLQueueStatus(filter.Status).Terminal() {
		return nil, 0, fmt.Errorf("invalid status filter: %s", filter.Status)
	}
	return nil, 0, nil
}

// TestGetDownlinkResults_InFlightStatusFilterIsInvalidArgument: a status
// filter naming an in-flight state is the caller's mistake, never an
// internal failure, and reaches no store.
func TestGetDownlinkResults_InFlightStatusFilterIsInvalidArgument(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)
	results := &terminalOnlyResults{}
	svc.useDownlinks(downlinkFakes{messages: results})

	_, err := svc.GetDownlinkResults(ownerCtx(), &pb.GetDownlinkResultsRequest{StatusFilter: string(mioty.DLQueueStatusPending)})

	requireRevokeCode(t, err, codes.InvalidArgument)
	assert.Zero(t, results.calls, "an invalid filter reaches no store")
}

// TestGetDownlinkResults_MalformedPageTokenIsRefused: a page token that is
// not an offset is refused like every other listing's, not read as the
// first page.
func TestGetDownlinkResults_MalformedPageTokenIsRefused(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)
	results := &filterMessageSvc{}
	svc.useDownlinks(downlinkFakes{messages: results})
	for _, token := range []string{"not-an-offset", "-20"} {
		_, err := svc.GetDownlinkResults(ownerCtx(), &pb.GetDownlinkResultsRequest{PageToken: token})
		requireRevokeCode(t, err, codes.InvalidArgument)
	}
	assert.Zero(t, results.calls, "a refused page reaches no store")
}

// TestGetDownlinkResults_ResultNamesAndFinalStatesFilter: the BSSCI §3.14.1
// result names select the queue status they are stored as; a final queue
// state filters as itself.
func TestGetDownlinkResults_ResultNamesAndFinalStatesFilter(t *testing.T) {
	f := &retainedFakes{}
	svc := newRetainedService(t, f)
	results := &filterMessageSvc{}
	svc.useDownlinks(downlinkFakes{messages: results})
	for filter, stored := range map[string]mioty.DLQueueStatus{
		mioty.DLDataResultSent:               mioty.DLQueueStatusTransmitted,
		mioty.DLDataResultInvalid:            mioty.DLQueueStatusFailed,
		mioty.DLDataResultExpired:            mioty.DLQueueStatusExpired,
		mioty.DLDataResultRevoked:            mioty.DLQueueStatusRevoked,
		string(mioty.DLQueueStatusAcked):     mioty.DLQueueStatusAcked,
		string(mioty.DLQueueStatusDelivered): mioty.DLQueueStatusDelivered,
	} {
		_, err := svc.GetDownlinkResults(ownerCtx(), &pb.GetDownlinkResultsRequest{StatusFilter: filter})
		require.NoError(t, err, filter)
		assert.Equal(t, string(stored), results.results.Status, filter)
	}
}
