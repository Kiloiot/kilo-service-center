package grpc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// revokedResultsMessageSvc serves a downlink revoked while a base station held
// it, one revoked while still pending, and one the station reported sent.
type revokedResultsMessageSvc struct {
	fakeMessageSvc
}

func (m *revokedResultsMessageSvc) GetDownlinkResults(context.Context, int64, *uuid.UUID, storage.DownlinkResultFilter, int, int) ([]*storage.DownlinkMessage, int, error) {
	return []*storage.DownlinkMessage{
		{ID: 1, QueID: 21, Status: mioty.DLQueueStatusRevoked, BsEui: 0x70B3D59CD000FF01},
		{ID: 2, QueID: 22, Status: mioty.DLQueueStatusRevoked},
		{ID: 3, QueID: 23, Status: mioty.DLQueueStatusTransmitted, Result: mioty.ResultSent},
	}, 3, nil
}

// TestGetDownlinkResults_ReportsARevokedDownlinkAsRevoked: the BSSCI result
// column only holds what a station reported (§3.14.1), so the revocation the
// row's status records is what the API reports as the result.
func TestGetDownlinkResults_ReportsARevokedDownlinkAsRevoked(t *testing.T) {
	svc := newRetainedService(t, &retainedFakes{})
	svc.useDownlinks(downlinkFakes{messages: &revokedResultsMessageSvc{}})

	resp, err := svc.GetDownlinkResults(ownerCtx(), &pb.GetDownlinkResultsRequest{})

	require.NoError(t, err)
	require.Len(t, resp.Results, 3)
	assert.Equal(t, mioty.DLDataResultRevoked, resp.Results[0].Result, "revoked while a base station held it")
	assert.Equal(t, mioty.DLDataResultRevoked, resp.Results[1].Result, "revoked while pending")
	assert.Equal(t, mioty.ResultSent, resp.Results[2].Result, "a reported result is kept")
}

// stationRowsMessageSvc serves, to both listings, a downlink a base station
// holds and one no station holds yet.
type stationRowsMessageSvc struct {
	fakeMessageSvc
}

func stationRows() []*storage.DownlinkMessage {
	return []*storage.DownlinkMessage{
		{ID: 1, QueID: 31, Status: mioty.DLQueueStatusQueued, BsEui: 0x70B3D59CD00009E6},
		{ID: 2, QueID: 32, Status: mioty.DLQueueStatusPending},
	}
}

func (m *stationRowsMessageSvc) ListDownlinkQueue(context.Context, int64, storage.DownlinkQueueFilter, int, int) ([]*storage.DownlinkMessage, int64, error) {
	return stationRows(), 2, nil
}

func (m *stationRowsMessageSvc) GetDownlinkResults(context.Context, int64, *uuid.UUID, storage.DownlinkResultFilter, int, int) ([]*storage.DownlinkMessage, int, error) {
	return stationRows(), 2, nil
}

// TestDownlinkListings_NameTheStationOfTheRow: the queue and the results
// report the base station the row names, and no station when none holds it.
func TestDownlinkListings_NameTheStationOfTheRow(t *testing.T) {
	svc := newRetainedService(t, &retainedFakes{})
	svc.useDownlinks(downlinkFakes{messages: &stationRowsMessageSvc{}})

	queue, err := svc.ListDownlinkQueue(ownerCtx(), &pb.ListDownlinkQueueRequest{})
	require.NoError(t, err)
	results, err := svc.GetDownlinkResults(ownerCtx(), &pb.GetDownlinkResultsRequest{})
	require.NoError(t, err)

	for name, listed := range map[string][]*pb.DownlinkMessage{"queue": queue.Messages, "results": results.Results} {
		require.Len(t, listed, 2, name)
		assert.Equal(t, "70B3D59CD00009E6", listed[0].BsEui, name)
		assert.Empty(t, listed[1].BsEui, name)
	}
}
